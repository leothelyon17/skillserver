package domain

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/mudler/skillserver/pkg/persistence"
)

const (
	catalogLegacyBackfillTagIDPrefix     = "tag-"
	catalogLegacyBackfillTagIDHashLength = 8
)

type catalogTaxonomyBackfillSourceRepository interface {
	List(ctx context.Context, filter persistence.CatalogSourceListFilter) ([]persistence.CatalogSourceRow, error)
}

type catalogTaxonomyBackfillOverlayRepository interface {
	List(
		ctx context.Context,
		filter persistence.CatalogMetadataOverlayListFilter,
	) ([]persistence.CatalogMetadataOverlayRow, error)
}

type catalogTaxonomyBackfillTagRepository interface {
	List(ctx context.Context, filter persistence.CatalogTagListFilter) ([]persistence.CatalogTagRow, error)
	Create(ctx context.Context, row persistence.CatalogTagRow) error
	GetByKey(ctx context.Context, key string) (persistence.CatalogTagRow, error)
}

type catalogTaxonomyBackfillTagAssignmentRepository interface {
	List(
		ctx context.Context,
		filter persistence.CatalogItemTagAssignmentListFilter,
	) ([]persistence.CatalogItemTagAssignmentRow, error)
	ReplaceForItemID(ctx context.Context, itemID string, tagIDs []string, createdAt time.Time) error
}

// CatalogTaxonomyLegacyLabelBackfillServiceOptions configures backfill behavior.
type CatalogTaxonomyLegacyLabelBackfillServiceOptions struct {
	Now func() time.Time
}

// CatalogTaxonomyLegacyLabelNormalizationCollision captures normalized key collisions.
type CatalogTaxonomyLegacyLabelNormalizationCollision struct {
	TagKey string   `json:"tag_key"`
	Labels []string `json:"labels"`
}

// CatalogTaxonomyLegacyLabelBackfillReport captures one backfill execution summary.
type CatalogTaxonomyLegacyLabelBackfillReport struct {
	ItemsScanned            int                                                `json:"items_scanned"`
	ItemsWithLegacyLabels   int                                                `json:"items_with_legacy_labels"`
	TagsCreated             int                                                `json:"tags_created"`
	ItemAssignmentsUpdated  int                                                `json:"item_assignments_updated"`
	NormalizationCollisions []CatalogTaxonomyLegacyLabelNormalizationCollision `json:"normalization_collisions"`
}

// CatalogTaxonomyLegacyLabelBackfillService migrates legacy overlay labels into taxonomy tags.
type CatalogTaxonomyLegacyLabelBackfillService struct {
	sourceRepo        catalogTaxonomyBackfillSourceRepository
	overlayRepo       catalogTaxonomyBackfillOverlayRepository
	tagRepo           catalogTaxonomyBackfillTagRepository
	tagAssignmentRepo catalogTaxonomyBackfillTagAssignmentRepository
	now               func() time.Time
}

// NewCatalogTaxonomyLegacyLabelBackfillService constructs a legacy label backfill service.
func NewCatalogTaxonomyLegacyLabelBackfillService(
	sourceRepo catalogTaxonomyBackfillSourceRepository,
	overlayRepo catalogTaxonomyBackfillOverlayRepository,
	tagRepo catalogTaxonomyBackfillTagRepository,
	tagAssignmentRepo catalogTaxonomyBackfillTagAssignmentRepository,
	options CatalogTaxonomyLegacyLabelBackfillServiceOptions,
) (*CatalogTaxonomyLegacyLabelBackfillService, error) {
	if sourceRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy backfill source repository is required")
	}
	if overlayRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy backfill overlay repository is required")
	}
	if tagRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy backfill tag repository is required")
	}
	if tagAssignmentRepo == nil {
		return nil, fmt.Errorf("catalog taxonomy backfill tag assignment repository is required")
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &CatalogTaxonomyLegacyLabelBackfillService{
		sourceRepo:        sourceRepo,
		overlayRepo:       overlayRepo,
		tagRepo:           tagRepo,
		tagAssignmentRepo: tagAssignmentRepo,
		now:               now,
	}, nil
}

// BackfillFromLegacyLabels creates taxonomy tags and item-tag assignments from overlay labels.
func (s *CatalogTaxonomyLegacyLabelBackfillService) BackfillFromLegacyLabels(
	ctx context.Context,
) (CatalogTaxonomyLegacyLabelBackfillReport, error) {
	if s == nil {
		return CatalogTaxonomyLegacyLabelBackfillReport{}, fmt.Errorf("catalog taxonomy legacy label backfill service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	sourceRows, err := s.sourceRepo.List(ctx, persistence.CatalogSourceListFilter{})
	if err != nil {
		return CatalogTaxonomyLegacyLabelBackfillReport{}, fmt.Errorf("list catalog source rows for taxonomy backfill: %w", err)
	}

	report := CatalogTaxonomyLegacyLabelBackfillReport{
		ItemsScanned: len(sourceRows),
	}
	if len(sourceRows) == 0 {
		return report, nil
	}

	itemIDs := make([]string, 0, len(sourceRows))
	for _, sourceRow := range sourceRows {
		itemIDs = append(itemIDs, sourceRow.ItemID)
	}
	sort.Strings(itemIDs)

	overlayRows, err := s.overlayRepo.List(ctx, persistence.CatalogMetadataOverlayListFilter{ItemIDs: itemIDs})
	if err != nil {
		return CatalogTaxonomyLegacyLabelBackfillReport{}, fmt.Errorf("list catalog metadata overlays for taxonomy backfill: %w", err)
	}

	overlayByItemID := make(map[string]persistence.CatalogMetadataOverlayRow, len(overlayRows))
	for _, overlayRow := range overlayRows {
		overlayByItemID[overlayRow.ItemID] = overlayRow
	}

	tagKeysByItemID := make(map[string][]string, len(itemIDs))
	displayNameByKey := map[string]string{}
	rawLabelsByKey := map[string]map[string]struct{}{}
	for _, itemID := range itemIDs {
		overlayRow, exists := overlayByItemID[itemID]
		if !exists || len(overlayRow.Labels) == 0 {
			continue
		}

		itemTagKeys := normalizeCatalogLegacyLabelsToTagKeys(overlayRow.Labels)
		if len(itemTagKeys) == 0 {
			continue
		}
		tagKeysByItemID[itemID] = itemTagKeys
		report.ItemsWithLegacyLabels++

		for _, label := range overlayRow.Labels {
			trimmedLabel := strings.TrimSpace(label)
			if trimmedLabel == "" {
				continue
			}

			tagKey, ok := NormalizeCatalogLegacyLabelToTagKey(trimmedLabel)
			if !ok {
				continue
			}

			if _, exists := displayNameByKey[tagKey]; !exists {
				displayNameByKey[tagKey] = trimmedLabel
			}

			if _, exists := rawLabelsByKey[tagKey]; !exists {
				rawLabelsByKey[tagKey] = map[string]struct{}{}
			}
			rawLabelsByKey[tagKey][trimmedLabel] = struct{}{}
		}
	}

	if len(tagKeysByItemID) == 0 {
		return report, nil
	}

	allTagKeySet := map[string]struct{}{}
	for _, itemTagKeys := range tagKeysByItemID {
		for _, tagKey := range itemTagKeys {
			allTagKeySet[tagKey] = struct{}{}
		}
	}
	allTagKeys := mapKeySetToSortedSlice(allTagKeySet)

	tagRows, err := s.tagRepo.List(ctx, persistence.CatalogTagListFilter{Keys: allTagKeys})
	if err != nil {
		return CatalogTaxonomyLegacyLabelBackfillReport{}, fmt.Errorf("list taxonomy tags for legacy label backfill: %w", err)
	}

	tagByKey := make(map[string]persistence.CatalogTagRow, len(tagRows))
	for _, tagRow := range tagRows {
		tagByKey[tagRow.Key] = tagRow
	}

	backfillAt := s.now().UTC()
	for _, tagKey := range allTagKeys {
		if _, exists := tagByKey[tagKey]; exists {
			continue
		}

		tagName := strings.TrimSpace(displayNameByKey[tagKey])
		if tagName == "" {
			tagName = tagKey
		}

		tagRow, created, err := s.ensureBackfillTagByKey(ctx, tagKey, tagName, backfillAt)
		if err != nil {
			return CatalogTaxonomyLegacyLabelBackfillReport{}, err
		}
		tagByKey[tagKey] = tagRow
		if created {
			report.TagsCreated++
		}
	}

	tagAssignmentRows, err := s.tagAssignmentRepo.List(ctx, persistence.CatalogItemTagAssignmentListFilter{ItemIDs: itemIDs})
	if err != nil {
		return CatalogTaxonomyLegacyLabelBackfillReport{}, fmt.Errorf(
			"list taxonomy tag assignments for legacy label backfill: %w",
			err,
		)
	}

	existingTagIDsByItemID := make(map[string][]string, len(tagKeysByItemID))
	for _, row := range tagAssignmentRows {
		existingTagIDsByItemID[row.ItemID] = append(existingTagIDsByItemID[row.ItemID], row.TagID)
	}

	for _, itemID := range mapKeySetToSortedSlice(mapStringSliceKeys(tagKeysByItemID)) {
		itemTagKeys := tagKeysByItemID[itemID]
		desiredTagIDs := make([]string, 0, len(itemTagKeys))
		for _, tagKey := range itemTagKeys {
			tagRow, exists := tagByKey[tagKey]
			if !exists {
				continue
			}
			desiredTagIDs = append(desiredTagIDs, tagRow.TagID)
		}
		desiredTagIDs = normalizeCatalogOptionalIDList(desiredTagIDs)

		existingTagIDs := normalizeCatalogOptionalIDList(existingTagIDsByItemID[itemID])
		if slices.Equal(existingTagIDs, desiredTagIDs) {
			continue
		}

		if err := s.tagAssignmentRepo.ReplaceForItemID(ctx, itemID, desiredTagIDs, backfillAt); err != nil {
			return CatalogTaxonomyLegacyLabelBackfillReport{}, fmt.Errorf(
				"replace taxonomy tag assignments during legacy backfill for %q: %w",
				itemID,
				err,
			)
		}
		report.ItemAssignmentsUpdated++
	}

	report.NormalizationCollisions = buildCatalogLegacyLabelNormalizationCollisions(rawLabelsByKey)
	return report, nil
}

func (s *CatalogTaxonomyLegacyLabelBackfillService) ensureBackfillTagByKey(
	ctx context.Context,
	tagKey string,
	tagName string,
	createdAt time.Time,
) (persistence.CatalogTagRow, bool, error) {
	tagIDs := []string{
		buildCatalogLegacyBackfillTagID(tagKey, false),
		buildCatalogLegacyBackfillTagID(tagKey, true),
	}

	for _, tagID := range tagIDs {
		createErr := s.tagRepo.Create(ctx, persistence.CatalogTagRow{
			TagID:       tagID,
			Key:         tagKey,
			Name:        tagName,
			Description: "",
			Color:       "",
			Active:      true,
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
		})
		if createErr == nil {
			createdRow, err := s.tagRepo.GetByKey(ctx, tagKey)
			if err != nil {
				if errors.Is(err, persistence.ErrCatalogTagNotFound) {
					return persistence.CatalogTagRow{}, false, fmt.Errorf("catalog taxonomy backfill tag key %q not found after create", tagKey)
				}
				return persistence.CatalogTagRow{}, false, fmt.Errorf(
					"get created taxonomy tag by key %q for legacy backfill: %w",
					tagKey,
					err,
				)
			}
			return createdRow, true, nil
		}

		if !isCatalogTaxonomyUniqueConstraintError(createErr) {
			return persistence.CatalogTagRow{}, false, fmt.Errorf(
				"create taxonomy tag for legacy label key %q: %w",
				tagKey,
				createErr,
			)
		}

		existingRow, err := s.tagRepo.GetByKey(ctx, tagKey)
		if err == nil {
			return existingRow, false, nil
		}
		if !errors.Is(err, persistence.ErrCatalogTagNotFound) {
			return persistence.CatalogTagRow{}, false, fmt.Errorf(
				"lookup taxonomy tag by key %q after uniqueness conflict: %w",
				tagKey,
				err,
			)
		}
	}

	return persistence.CatalogTagRow{}, false, fmt.Errorf(
		"create taxonomy tag for legacy label key %q: unable to allocate deterministic tag_id",
		tagKey,
	)
}

func normalizeCatalogLegacyLabelsToTagKeys(labels []string) []string {
	tagKeySet := map[string]struct{}{}
	for _, label := range labels {
		tagKey, ok := NormalizeCatalogLegacyLabelToTagKey(label)
		if !ok {
			continue
		}
		tagKeySet[tagKey] = struct{}{}
	}
	return mapKeySetToSortedSlice(tagKeySet)
}

// NormalizeCatalogLegacyLabelToTagKey normalizes free-form label text into deterministic taxonomy tag keys.
func NormalizeCatalogLegacyLabelToTagKey(label string) (string, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(label))
	if trimmed == "" {
		return "", false
	}

	var builder strings.Builder
	lastWasSeparator := false
	for _, candidate := range trimmed {
		switch {
		case unicode.IsLetter(candidate) || unicode.IsDigit(candidate):
			builder.WriteRune(candidate)
			lastWasSeparator = false
		default:
			if builder.Len() == 0 || lastWasSeparator {
				continue
			}
			builder.WriteRune('-')
			lastWasSeparator = true
		}
	}

	normalized := strings.Trim(builder.String(), "-")
	if normalized == "" {
		return "", false
	}

	return normalized, true
}

func buildCatalogLegacyBackfillTagID(tagKey string, withHashSuffix bool) string {
	baseID := catalogLegacyBackfillTagIDPrefix + tagKey
	if !withHashSuffix {
		return baseID
	}

	digest := sha1.Sum([]byte(tagKey))
	hexDigest := hex.EncodeToString(digest[:])
	return baseID + "-" + hexDigest[:catalogLegacyBackfillTagIDHashLength]
}

func mapStringSliceKeys(values map[string][]string) map[string]struct{} {
	keys := make(map[string]struct{}, len(values))
	for key := range values {
		keys[key] = struct{}{}
	}
	return keys
}

func buildCatalogLegacyLabelNormalizationCollisions(
	rawLabelsByKey map[string]map[string]struct{},
) []CatalogTaxonomyLegacyLabelNormalizationCollision {
	if len(rawLabelsByKey) == 0 {
		return nil
	}

	collisions := make([]CatalogTaxonomyLegacyLabelNormalizationCollision, 0, len(rawLabelsByKey))
	for tagKey, rawLabels := range rawLabelsByKey {
		if len(rawLabels) <= 1 {
			continue
		}

		labels := make([]string, 0, len(rawLabels))
		for label := range rawLabels {
			labels = append(labels, label)
		}
		sort.Strings(labels)
		collisions = append(collisions, CatalogTaxonomyLegacyLabelNormalizationCollision{
			TagKey: tagKey,
			Labels: labels,
		})
	}

	sort.Slice(collisions, func(i int, j int) bool {
		return collisions[i].TagKey < collisions[j].TagKey
	})

	return collisions
}
