package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	// ErrCatalogSourceNotFound indicates that a source snapshot row does not exist.
	ErrCatalogSourceNotFound = errors.New("catalog source row not found")
	// ErrCatalogMetadataOverlayNotFound indicates that an overlay row does not exist.
	ErrCatalogMetadataOverlayNotFound = errors.New("catalog metadata overlay row not found")
)

// CatalogClassifier is the persisted classifier for a catalog item.
type CatalogClassifier string

const (
	CatalogClassifierSkill  CatalogClassifier = "skill"
	CatalogClassifierPrompt CatalogClassifier = "prompt"
)

// IsValid reports whether the classifier value is supported by the schema.
func (c CatalogClassifier) IsValid() bool {
	switch c {
	case CatalogClassifierSkill, CatalogClassifierPrompt:
		return true
	default:
		return false
	}
}

// CatalogSourceType identifies the source origin for a catalog item.
type CatalogSourceType string

const (
	CatalogSourceTypeGit        CatalogSourceType = "git"
	CatalogSourceTypeLocal      CatalogSourceType = "local"
	CatalogSourceTypeFileImport CatalogSourceType = "file_import"
)

// IsValid reports whether the source type value is supported by the schema.
func (s CatalogSourceType) IsValid() bool {
	switch s {
	case CatalogSourceTypeGit, CatalogSourceTypeLocal, CatalogSourceTypeFileImport:
		return true
	default:
		return false
	}
}

// CatalogSourceRow mirrors one row in catalog_source_items.
type CatalogSourceRow struct {
	ItemID           string
	Classifier       CatalogClassifier
	SourceType       CatalogSourceType
	SourceRepo       *string
	ParentSkillID    *string
	ResourcePath     *string
	Name             string
	Description      string
	Content          string
	ContentHash      string
	ContentWritable  bool
	MetadataWritable bool
	LastSyncedAt     time.Time
	DeletedAt        *time.Time
}

// CatalogSourceListFilter constrains source row list queries.
type CatalogSourceListFilter struct {
	ItemID         string
	ItemIDs        []string
	Classifier     *CatalogClassifier
	SourceType     *CatalogSourceType
	SourceRepo     *string
	IncludeDeleted bool
}

// CatalogMetadataOverlayRow mirrors one row in catalog_metadata_overlays.
type CatalogMetadataOverlayRow struct {
	ItemID              string
	DisplayNameOverride *string
	DescriptionOverride *string
	CustomMetadata      map[string]any
	Labels              []string
	UpdatedAt           time.Time
	UpdatedBy           *string
	CustomMetadataJSON  string
	LabelsJSON          string
}

// CatalogMetadataOverlayListFilter constrains overlay list queries.
type CatalogMetadataOverlayListFilter struct {
	ItemID  string
	ItemIDs []string
}

type rowScanner interface {
	Scan(dest ...any) error
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func normalizeRequiredID(value string, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", fieldName)
	}
	return trimmed, nil
}

func normalizeOptionalIDList(itemIDs []string) []string {
	if len(itemIDs) == 0 {
		return nil
	}

	deduped := make(map[string]struct{}, len(itemIDs))
	normalized := make([]string, 0, len(itemIDs))
	for _, raw := range itemIDs {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, exists := deduped[trimmed]; exists {
			continue
		}
		deduped[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	sort.Strings(normalized)
	return normalized
}

func validateCatalogSourceUpsertRow(row CatalogSourceRow) (CatalogSourceRow, error) {
	itemID, err := normalizeRequiredID(row.ItemID, "catalog source item_id")
	if err != nil {
		return CatalogSourceRow{}, err
	}
	row.ItemID = itemID

	if !row.Classifier.IsValid() {
		return CatalogSourceRow{}, fmt.Errorf("catalog source classifier %q is invalid", row.Classifier)
	}
	if !row.SourceType.IsValid() {
		return CatalogSourceRow{}, fmt.Errorf("catalog source source_type %q is invalid", row.SourceType)
	}

	name := strings.TrimSpace(row.Name)
	if name == "" {
		return CatalogSourceRow{}, fmt.Errorf("catalog source name is required")
	}
	row.Name = name

	if strings.TrimSpace(row.ContentHash) == "" {
		return CatalogSourceRow{}, fmt.Errorf("catalog source content_hash is required")
	}

	if row.LastSyncedAt.IsZero() {
		return CatalogSourceRow{}, fmt.Errorf("catalog source last_synced_at is required")
	}
	row.LastSyncedAt = row.LastSyncedAt.UTC()

	if row.SourceRepo != nil {
		normalized := strings.TrimSpace(*row.SourceRepo)
		if normalized == "" {
			row.SourceRepo = nil
		} else {
			row.SourceRepo = &normalized
		}
	}

	if row.ParentSkillID != nil {
		normalized := strings.TrimSpace(*row.ParentSkillID)
		if normalized == "" {
			row.ParentSkillID = nil
		} else {
			row.ParentSkillID = &normalized
		}
	}

	if row.ResourcePath != nil {
		normalized := strings.TrimSpace(*row.ResourcePath)
		if normalized == "" {
			row.ResourcePath = nil
		} else {
			row.ResourcePath = &normalized
		}
	}

	if row.DeletedAt != nil {
		timestamp := row.DeletedAt.UTC()
		row.DeletedAt = &timestamp
	}

	return row, nil
}

func validateCatalogMetadataOverlayUpsertRow(row CatalogMetadataOverlayRow) (CatalogMetadataOverlayRow, error) {
	itemID, err := normalizeRequiredID(row.ItemID, "catalog metadata overlay item_id")
	if err != nil {
		return CatalogMetadataOverlayRow{}, err
	}
	row.ItemID = itemID

	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now().UTC()
	} else {
		row.UpdatedAt = row.UpdatedAt.UTC()
	}

	if row.DisplayNameOverride != nil {
		trimmed := strings.TrimSpace(*row.DisplayNameOverride)
		row.DisplayNameOverride = &trimmed
	}
	if row.DescriptionOverride != nil {
		trimmed := strings.TrimSpace(*row.DescriptionOverride)
		row.DescriptionOverride = &trimmed
	}
	if row.UpdatedBy != nil {
		trimmed := strings.TrimSpace(*row.UpdatedBy)
		if trimmed == "" {
			row.UpdatedBy = nil
		} else {
			row.UpdatedBy = &trimmed
		}
	}

	customMetadataJSON, err := marshalCustomMetadataJSON(row.CustomMetadata)
	if err != nil {
		return CatalogMetadataOverlayRow{}, err
	}
	row.CustomMetadataJSON = customMetadataJSON

	labelsJSON, err := marshalLabelsJSON(row.Labels)
	if err != nil {
		return CatalogMetadataOverlayRow{}, err
	}
	row.LabelsJSON = labelsJSON

	return row, nil
}

func scanCatalogSourceRow(scanner rowScanner) (CatalogSourceRow, error) {
	var (
		itemID              string
		classifier          string
		sourceType          string
		sourceRepo          sql.NullString
		parentSkillID       sql.NullString
		resourcePath        sql.NullString
		name                string
		description         string
		content             string
		contentHash         string
		contentWritableRaw  int
		metadataWritableRaw int
		lastSyncedAtRaw     string
		deletedAtRaw        sql.NullString
	)

	if err := scanner.Scan(
		&itemID,
		&classifier,
		&sourceType,
		&sourceRepo,
		&parentSkillID,
		&resourcePath,
		&name,
		&description,
		&content,
		&contentHash,
		&contentWritableRaw,
		&metadataWritableRaw,
		&lastSyncedAtRaw,
		&deletedAtRaw,
	); err != nil {
		return CatalogSourceRow{}, err
	}

	parsedLastSyncedAt, err := parseCatalogTimestamp(lastSyncedAtRaw)
	if err != nil {
		return CatalogSourceRow{}, fmt.Errorf("parse catalog source last_synced_at: %w", err)
	}

	row := CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       CatalogClassifier(classifier),
		SourceType:       CatalogSourceType(sourceType),
		SourceRepo:       nullStringToPointer(sourceRepo),
		ParentSkillID:    nullStringToPointer(parentSkillID),
		ResourcePath:     nullStringToPointer(resourcePath),
		Name:             name,
		Description:      description,
		Content:          content,
		ContentHash:      contentHash,
		ContentWritable:  contentWritableRaw != 0,
		MetadataWritable: metadataWritableRaw != 0,
		LastSyncedAt:     parsedLastSyncedAt,
	}

	if deletedAtRaw.Valid {
		parsedDeletedAt, err := parseCatalogTimestamp(deletedAtRaw.String)
		if err != nil {
			return CatalogSourceRow{}, fmt.Errorf("parse catalog source deleted_at: %w", err)
		}
		row.DeletedAt = &parsedDeletedAt
	}

	return row, nil
}

func scanCatalogMetadataOverlayRow(scanner rowScanner) (CatalogMetadataOverlayRow, error) {
	var (
		itemID              string
		displayNameOverride sql.NullString
		descriptionOverride sql.NullString
		customMetadataJSON  string
		labelsJSON          string
		updatedAtRaw        string
		updatedBy           sql.NullString
	)

	if err := scanner.Scan(
		&itemID,
		&displayNameOverride,
		&descriptionOverride,
		&customMetadataJSON,
		&labelsJSON,
		&updatedAtRaw,
		&updatedBy,
	); err != nil {
		return CatalogMetadataOverlayRow{}, err
	}

	customMetadata, err := unmarshalCustomMetadataJSON(customMetadataJSON)
	if err != nil {
		return CatalogMetadataOverlayRow{}, fmt.Errorf("parse catalog metadata overlay custom_metadata_json: %w", err)
	}
	labels, err := unmarshalLabelsJSON(labelsJSON)
	if err != nil {
		return CatalogMetadataOverlayRow{}, fmt.Errorf("parse catalog metadata overlay labels_json: %w", err)
	}

	updatedAt, err := parseCatalogTimestamp(updatedAtRaw)
	if err != nil {
		return CatalogMetadataOverlayRow{}, fmt.Errorf("parse catalog metadata overlay updated_at: %w", err)
	}

	return CatalogMetadataOverlayRow{
		ItemID:              itemID,
		DisplayNameOverride: nullStringToPointer(displayNameOverride),
		DescriptionOverride: nullStringToPointer(descriptionOverride),
		CustomMetadata:      customMetadata,
		Labels:              labels,
		UpdatedAt:           updatedAt,
		UpdatedBy:           nullStringToPointer(updatedBy),
		CustomMetadataJSON:  customMetadataJSON,
		LabelsJSON:          labelsJSON,
	}, nil
}

func marshalCustomMetadataJSON(customMetadata map[string]any) (string, error) {
	if customMetadata == nil {
		customMetadata = map[string]any{}
	}
	encoded, err := json.Marshal(customMetadata)
	if err != nil {
		return "", fmt.Errorf("marshal custom metadata: %w", err)
	}
	return string(encoded), nil
}

func unmarshalCustomMetadataJSON(raw string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "{}"
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid JSON object: %w", err)
	}

	asMap, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid JSON object: expected object")
	}
	if asMap == nil {
		return map[string]any{}, nil
	}

	return asMap, nil
}

func marshalLabelsJSON(labels []string) (string, error) {
	if labels == nil {
		labels = []string{}
	}
	encoded, err := json.Marshal(labels)
	if err != nil {
		return "", fmt.Errorf("marshal labels: %w", err)
	}
	return string(encoded), nil
}

func unmarshalLabelsJSON(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "[]"
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %w", err)
	}

	asSlice, ok := decoded.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid JSON array: expected array")
	}

	labels := make([]string, 0, len(asSlice))
	for index, entry := range asSlice {
		value, ok := entry.(string)
		if !ok {
			return nil, fmt.Errorf("invalid JSON array: labels[%d] must be a string", index)
		}
		labels = append(labels, value)
	}

	return labels, nil
}

func formatCatalogTimestamp(timestamp time.Time) string {
	return timestamp.UTC().Format(time.RFC3339Nano)
}

func parseCatalogTimestamp(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported timestamp format %q", raw)
}

func nullStringToPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	copyValue := value.String
	return &copyValue
}

func pointerToAny(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func boolToSQLiteInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}
