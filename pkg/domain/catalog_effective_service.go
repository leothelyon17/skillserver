package domain

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mudler/skillserver/pkg/persistence"
)

type catalogEffectiveSourceRepository interface {
	List(ctx context.Context, filter persistence.CatalogSourceListFilter) ([]persistence.CatalogSourceRow, error)
}

type catalogEffectiveOverlayRepository interface {
	List(ctx context.Context, filter persistence.CatalogMetadataOverlayListFilter) ([]persistence.CatalogMetadataOverlayRow, error)
}

// CatalogEffectiveListFilter constrains effective projection list queries.
type CatalogEffectiveListFilter struct {
	ItemID         string
	ItemIDs        []string
	Classifier     *CatalogClassifier
	SourceType     *persistence.CatalogSourceType
	SourceRepo     *string
	IncludeDeleted bool
}

// CatalogEffectiveService projects source + overlay rows into effective catalog items.
type CatalogEffectiveService struct {
	sourceRepo  catalogEffectiveSourceRepository
	overlayRepo catalogEffectiveOverlayRepository
}

// NewCatalogEffectiveService creates an effective catalog projection service.
func NewCatalogEffectiveService(
	sourceRepo catalogEffectiveSourceRepository,
	overlayRepo catalogEffectiveOverlayRepository,
) (*CatalogEffectiveService, error) {
	if sourceRepo == nil {
		return nil, fmt.Errorf("catalog effective source repository is required")
	}
	if overlayRepo == nil {
		return nil, fmt.Errorf("catalog effective overlay repository is required")
	}

	return &CatalogEffectiveService{
		sourceRepo:  sourceRepo,
		overlayRepo: overlayRepo,
	}, nil
}

// List returns effective catalog items with deterministic ordering.
func (s *CatalogEffectiveService) List(ctx context.Context, filter CatalogEffectiveListFilter) ([]CatalogItem, error) {
	if s == nil {
		return nil, fmt.Errorf("catalog effective service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	sourceFilter, err := mapEffectiveListFilter(filter)
	if err != nil {
		return nil, err
	}

	sourceRows, err := s.sourceRepo.List(ctx, sourceFilter)
	if err != nil {
		return nil, fmt.Errorf("list catalog source rows: %w", err)
	}
	if len(sourceRows) == 0 {
		return []CatalogItem{}, nil
	}

	itemIDs := make([]string, 0, len(sourceRows))
	for _, sourceRow := range sourceRows {
		itemIDs = append(itemIDs, sourceRow.ItemID)
	}
	sort.Strings(itemIDs)

	overlays, err := s.overlayRepo.List(ctx, persistence.CatalogMetadataOverlayListFilter{ItemIDs: itemIDs})
	if err != nil {
		return nil, fmt.Errorf("list catalog metadata overlays: %w", err)
	}

	overlayByItemID := make(map[string]persistence.CatalogMetadataOverlayRow, len(overlays))
	for _, overlay := range overlays {
		overlayByItemID[overlay.ItemID] = overlay
	}

	effectiveItems := make([]CatalogItem, 0, len(sourceRows))
	for _, sourceRow := range sourceRows {
		overlay, hasOverlay := overlayByItemID[sourceRow.ItemID]
		item, mapErr := mapEffectiveCatalogItem(sourceRow, hasOverlay, overlay)
		if mapErr != nil {
			return nil, mapErr
		}
		effectiveItems = append(effectiveItems, item)
	}

	return effectiveItems, nil
}

func mapEffectiveListFilter(filter CatalogEffectiveListFilter) (persistence.CatalogSourceListFilter, error) {
	mapped := persistence.CatalogSourceListFilter{
		ItemID:         filter.ItemID,
		ItemIDs:        filter.ItemIDs,
		SourceType:     filter.SourceType,
		SourceRepo:     filter.SourceRepo,
		IncludeDeleted: filter.IncludeDeleted,
	}

	if filter.Classifier == nil {
		return mapped, nil
	}

	if !filter.Classifier.IsValid() {
		return persistence.CatalogSourceListFilter{}, fmt.Errorf("catalog classifier filter %q is invalid", *filter.Classifier)
	}

	classifier, err := mapEffectiveClassifier(*filter.Classifier)
	if err != nil {
		return persistence.CatalogSourceListFilter{}, err
	}
	mapped.Classifier = &classifier

	return mapped, nil
}

func mapEffectiveCatalogItem(
	source persistence.CatalogSourceRow,
	hasOverlay bool,
	overlay persistence.CatalogMetadataOverlayRow,
) (CatalogItem, error) {
	classifier, err := mapEffectiveDomainClassifier(source.Classifier)
	if err != nil {
		return CatalogItem{}, fmt.Errorf("map effective catalog item %q classifier: %w", source.ItemID, err)
	}

	name := strings.TrimSpace(source.Name)
	description := source.Description
	customMetadata := map[string]any{}
	labels := []string{}

	if hasOverlay {
		if override := resolveOverlayText(overlay.DisplayNameOverride); override != nil {
			name = *override
		}
		if override := resolveOverlayText(overlay.DescriptionOverride); override != nil {
			description = *override
		}
		customMetadata = copyCustomMetadata(overlay.CustomMetadata)
		labels = append(labels, overlay.Labels...)
	}

	contentWritable := source.SourceType != persistence.CatalogSourceTypeGit
	metadataWritable := true

	item := CatalogItem{
		ID:               source.ItemID,
		Classifier:       classifier,
		Name:             name,
		Description:      description,
		Content:          source.Content,
		ContentWritable:  contentWritable,
		MetadataWritable: metadataWritable,
		CustomMetadata:   customMetadata,
		Labels:           labels,
		ReadOnly:         !contentWritable,
	}

	if source.ParentSkillID != nil {
		item.ParentSkillID = *source.ParentSkillID
	}
	if source.ResourcePath != nil {
		item.ResourcePath = *source.ResourcePath
	}

	return item, nil
}

func mapEffectiveClassifier(classifier CatalogClassifier) (persistence.CatalogClassifier, error) {
	switch classifier {
	case CatalogClassifierSkill:
		return persistence.CatalogClassifierSkill, nil
	case CatalogClassifierPrompt:
		return persistence.CatalogClassifierPrompt, nil
	default:
		return "", fmt.Errorf("catalog classifier %q is invalid", classifier)
	}
}

func mapEffectiveDomainClassifier(classifier persistence.CatalogClassifier) (CatalogClassifier, error) {
	switch classifier {
	case persistence.CatalogClassifierSkill:
		return CatalogClassifierSkill, nil
	case persistence.CatalogClassifierPrompt:
		return CatalogClassifierPrompt, nil
	default:
		return "", fmt.Errorf("catalog classifier %q is invalid", classifier)
	}
}

func resolveOverlayText(override *string) *string {
	if override == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*override)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func copyCustomMetadata(customMetadata map[string]any) map[string]any {
	if len(customMetadata) == 0 {
		return map[string]any{}
	}
	copied := make(map[string]any, len(customMetadata))
	for key, value := range customMetadata {
		copied[key] = value
	}
	return copied
}
