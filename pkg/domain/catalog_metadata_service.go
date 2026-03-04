package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

var (
	// ErrCatalogMetadataItemNotFound indicates that a catalog item does not exist in source snapshots.
	ErrCatalogMetadataItemNotFound = errors.New("catalog metadata item not found")
)

type catalogMetadataSourceRepository interface {
	GetByItemID(ctx context.Context, itemID string) (persistence.CatalogSourceRow, error)
}

type catalogMetadataOverlayRepository interface {
	GetByItemID(ctx context.Context, itemID string) (persistence.CatalogMetadataOverlayRow, error)
	Upsert(ctx context.Context, row persistence.CatalogMetadataOverlayRow) error
}

type catalogMetadataEffectiveService interface {
	List(ctx context.Context, filter CatalogEffectiveListFilter) ([]CatalogItem, error)
}

// CatalogMetadataServiceOptions configures catalog metadata service behavior.
type CatalogMetadataServiceOptions struct {
	Now func() time.Time
}

// CatalogMetadataPatchInput describes a partial metadata overlay mutation.
type CatalogMetadataPatchInput struct {
	ItemID              string
	DisplayNameOverride *string
	DescriptionOverride *string
	Labels              *[]string
	CustomMetadata      *map[string]any
	UpdatedBy           *string
}

// CatalogMetadataSourceView is the source snapshot state for one catalog item.
type CatalogMetadataSourceView struct {
	ItemID           string                        `json:"item_id"`
	Classifier       CatalogClassifier             `json:"classifier"`
	SourceType       persistence.CatalogSourceType `json:"source_type"`
	SourceRepo       *string                       `json:"source_repo,omitempty"`
	ParentSkillID    *string                       `json:"parent_skill_id,omitempty"`
	ResourcePath     *string                       `json:"resource_path,omitempty"`
	Name             string                        `json:"name"`
	Description      string                        `json:"description,omitempty"`
	ContentWritable  bool                          `json:"content_writable"`
	MetadataWritable bool                          `json:"metadata_writable"`
	ReadOnly         bool                          `json:"read_only"`
}

// CatalogMetadataOverlayView is the persisted overlay state for one catalog item.
type CatalogMetadataOverlayView struct {
	DisplayNameOverride *string        `json:"display_name_override,omitempty"`
	DescriptionOverride *string        `json:"description_override,omitempty"`
	CustomMetadata      map[string]any `json:"custom_metadata"`
	Labels              []string       `json:"labels"`
	UpdatedAt           *time.Time     `json:"updated_at,omitempty"`
	UpdatedBy           *string        `json:"updated_by,omitempty"`
}

// CatalogMetadataEffectiveView is the effective metadata exposed to clients.
type CatalogMetadataEffectiveView struct {
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	CustomMetadata   map[string]any `json:"custom_metadata"`
	Labels           []string       `json:"labels"`
	ContentWritable  bool           `json:"content_writable"`
	MetadataWritable bool           `json:"metadata_writable"`
	ReadOnly         bool           `json:"read_only"`
}

// CatalogMetadataView combines source, overlay, and effective metadata views.
type CatalogMetadataView struct {
	ItemID    string                       `json:"item_id"`
	Source    CatalogMetadataSourceView    `json:"source"`
	Overlay   CatalogMetadataOverlayView   `json:"overlay"`
	Effective CatalogMetadataEffectiveView `json:"effective"`
}

// CatalogMetadataService orchestrates read/write metadata overlay operations.
type CatalogMetadataService struct {
	sourceRepo  catalogMetadataSourceRepository
	overlayRepo catalogMetadataOverlayRepository
	effective   catalogMetadataEffectiveService
	now         func() time.Time
}

// NewCatalogMetadataService creates a metadata overlay service.
func NewCatalogMetadataService(
	sourceRepo catalogMetadataSourceRepository,
	overlayRepo catalogMetadataOverlayRepository,
	effective catalogMetadataEffectiveService,
	options CatalogMetadataServiceOptions,
) (*CatalogMetadataService, error) {
	if sourceRepo == nil {
		return nil, fmt.Errorf("catalog metadata source repository is required")
	}
	if overlayRepo == nil {
		return nil, fmt.Errorf("catalog metadata overlay repository is required")
	}
	if effective == nil {
		return nil, fmt.Errorf("catalog metadata effective service is required")
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &CatalogMetadataService{
		sourceRepo:  sourceRepo,
		overlayRepo: overlayRepo,
		effective:   effective,
		now:         now,
	}, nil
}

// Get returns source + overlay + effective metadata for one catalog item ID.
func (s *CatalogMetadataService) Get(ctx context.Context, itemID string) (CatalogMetadataView, error) {
	if s == nil {
		return CatalogMetadataView{}, fmt.Errorf("catalog metadata service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedItemID := strings.TrimSpace(itemID)
	if normalizedItemID == "" {
		return CatalogMetadataView{}, fmt.Errorf("catalog metadata item id is required")
	}

	sourceRow, err := s.sourceRepo.GetByItemID(ctx, normalizedItemID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogSourceNotFound) {
			return CatalogMetadataView{}, ErrCatalogMetadataItemNotFound
		}
		return CatalogMetadataView{}, fmt.Errorf("get catalog source item %q: %w", normalizedItemID, err)
	}
	if sourceRow.DeletedAt != nil {
		return CatalogMetadataView{}, ErrCatalogMetadataItemNotFound
	}

	overlayRow, hasOverlay, err := s.getOverlayRow(ctx, normalizedItemID)
	if err != nil {
		return CatalogMetadataView{}, err
	}

	effectiveItems, err := s.effective.List(ctx, CatalogEffectiveListFilter{ItemID: normalizedItemID})
	if err != nil {
		return CatalogMetadataView{}, fmt.Errorf("list effective catalog item %q: %w", normalizedItemID, err)
	}
	if len(effectiveItems) == 0 {
		return CatalogMetadataView{}, ErrCatalogMetadataItemNotFound
	}

	effectiveItem := effectiveItems[0]
	return mapCatalogMetadataView(sourceRow, effectiveItem, overlayRow, hasOverlay)
}

// Patch updates the metadata overlay for one catalog item ID and returns the effective view.
func (s *CatalogMetadataService) Patch(ctx context.Context, input CatalogMetadataPatchInput) (CatalogMetadataView, error) {
	if s == nil {
		return CatalogMetadataView{}, fmt.Errorf("catalog metadata service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedItemID := strings.TrimSpace(input.ItemID)
	if normalizedItemID == "" {
		return CatalogMetadataView{}, fmt.Errorf("catalog metadata item id is required")
	}

	sourceRow, err := s.sourceRepo.GetByItemID(ctx, normalizedItemID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogSourceNotFound) {
			return CatalogMetadataView{}, ErrCatalogMetadataItemNotFound
		}
		return CatalogMetadataView{}, fmt.Errorf("get catalog source item %q: %w", normalizedItemID, err)
	}
	if sourceRow.DeletedAt != nil {
		return CatalogMetadataView{}, ErrCatalogMetadataItemNotFound
	}

	overlayRow, _, err := s.getOverlayRow(ctx, normalizedItemID)
	if err != nil {
		return CatalogMetadataView{}, err
	}

	mergedOverlay := persistence.CatalogMetadataOverlayRow{
		ItemID:              normalizedItemID,
		DisplayNameOverride: overlayRow.DisplayNameOverride,
		DescriptionOverride: overlayRow.DescriptionOverride,
		CustomMetadata:      copyCatalogMetadataMap(overlayRow.CustomMetadata),
		Labels:              append([]string{}, overlayRow.Labels...),
		UpdatedAt:           s.now().UTC(),
		UpdatedBy:           normalizeCatalogMetadataOptionalText(input.UpdatedBy),
	}

	if input.DisplayNameOverride != nil {
		displayName := strings.TrimSpace(*input.DisplayNameOverride)
		mergedOverlay.DisplayNameOverride = &displayName
	}
	if input.DescriptionOverride != nil {
		description := strings.TrimSpace(*input.DescriptionOverride)
		mergedOverlay.DescriptionOverride = &description
	}
	if input.CustomMetadata != nil {
		mergedOverlay.CustomMetadata = copyCatalogMetadataMap(*input.CustomMetadata)
	}
	if input.Labels != nil {
		mergedOverlay.Labels = append([]string{}, (*input.Labels)...)
	}

	if err := s.overlayRepo.Upsert(ctx, mergedOverlay); err != nil {
		return CatalogMetadataView{}, fmt.Errorf("upsert catalog metadata overlay %q: %w", normalizedItemID, err)
	}

	return s.Get(ctx, normalizedItemID)
}

// List returns effective catalog items through the same effective projection dependency.
func (s *CatalogMetadataService) List(
	ctx context.Context,
	filter CatalogEffectiveListFilter,
) ([]CatalogItem, error) {
	if s == nil {
		return nil, fmt.Errorf("catalog metadata service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	items, err := s.effective.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list effective catalog items: %w", err)
	}

	return items, nil
}

func (s *CatalogMetadataService) getOverlayRow(
	ctx context.Context,
	itemID string,
) (persistence.CatalogMetadataOverlayRow, bool, error) {
	overlayRow, err := s.overlayRepo.GetByItemID(ctx, itemID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogMetadataOverlayNotFound) {
			return persistence.CatalogMetadataOverlayRow{
				ItemID:         itemID,
				CustomMetadata: map[string]any{},
				Labels:         []string{},
			}, false, nil
		}
		return persistence.CatalogMetadataOverlayRow{}, false, fmt.Errorf("get catalog metadata overlay %q: %w", itemID, err)
	}

	if overlayRow.CustomMetadata == nil {
		overlayRow.CustomMetadata = map[string]any{}
	}
	if overlayRow.Labels == nil {
		overlayRow.Labels = []string{}
	}

	return overlayRow, true, nil
}

func mapCatalogMetadataView(
	sourceRow persistence.CatalogSourceRow,
	effectiveItem CatalogItem,
	overlayRow persistence.CatalogMetadataOverlayRow,
	hasOverlay bool,
) (CatalogMetadataView, error) {
	classifier, err := mapCatalogMetadataClassifier(sourceRow.Classifier)
	if err != nil {
		return CatalogMetadataView{}, fmt.Errorf("map source classifier for %q: %w", sourceRow.ItemID, err)
	}

	source := CatalogMetadataSourceView{
		ItemID:           sourceRow.ItemID,
		Classifier:       classifier,
		SourceType:       sourceRow.SourceType,
		SourceRepo:       sourceRow.SourceRepo,
		ParentSkillID:    sourceRow.ParentSkillID,
		ResourcePath:     sourceRow.ResourcePath,
		Name:             sourceRow.Name,
		Description:      sourceRow.Description,
		ContentWritable:  sourceRow.ContentWritable,
		MetadataWritable: sourceRow.MetadataWritable,
		ReadOnly:         !sourceRow.ContentWritable,
	}

	overlay := CatalogMetadataOverlayView{
		DisplayNameOverride: overlayRow.DisplayNameOverride,
		DescriptionOverride: overlayRow.DescriptionOverride,
		CustomMetadata:      copyCatalogMetadataMap(overlayRow.CustomMetadata),
		Labels:              append([]string{}, overlayRow.Labels...),
		UpdatedBy:           overlayRow.UpdatedBy,
	}
	if hasOverlay {
		updatedAt := overlayRow.UpdatedAt.UTC()
		overlay.UpdatedAt = &updatedAt
	}
	if overlay.CustomMetadata == nil {
		overlay.CustomMetadata = map[string]any{}
	}
	if overlay.Labels == nil {
		overlay.Labels = []string{}
	}

	effective := CatalogMetadataEffectiveView{
		Name:             effectiveItem.Name,
		Description:      effectiveItem.Description,
		CustomMetadata:   copyCatalogMetadataMap(effectiveItem.CustomMetadata),
		Labels:           append([]string{}, effectiveItem.Labels...),
		ContentWritable:  effectiveItem.ContentWritable,
		MetadataWritable: effectiveItem.MetadataWritable,
		ReadOnly:         effectiveItem.ReadOnly,
	}
	if effective.CustomMetadata == nil {
		effective.CustomMetadata = map[string]any{}
	}
	if effective.Labels == nil {
		effective.Labels = []string{}
	}

	return CatalogMetadataView{
		ItemID:    sourceRow.ItemID,
		Source:    source,
		Overlay:   overlay,
		Effective: effective,
	}, nil
}

func mapCatalogMetadataClassifier(classifier persistence.CatalogClassifier) (CatalogClassifier, error) {
	switch classifier {
	case persistence.CatalogClassifierSkill:
		return CatalogClassifierSkill, nil
	case persistence.CatalogClassifierPrompt:
		return CatalogClassifierPrompt, nil
	default:
		return "", fmt.Errorf("catalog classifier %q is invalid", classifier)
	}
}

func copyCatalogMetadataMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}

	copied := make(map[string]any, len(input))
	for key, value := range input {
		copied[key] = value
	}
	return copied
}

func normalizeCatalogMetadataOptionalText(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
