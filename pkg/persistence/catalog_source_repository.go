package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type catalogQueryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// CatalogSourceRepository provides access to catalog_source_items rows.
type CatalogSourceRepository struct {
	exec catalogQueryExecutor
}

// NewCatalogSourceRepository creates a source repository around a DB or transaction handle.
func NewCatalogSourceRepository(exec catalogQueryExecutor) (*CatalogSourceRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("catalog source repository query executor is required")
	}

	return &CatalogSourceRepository{exec: exec}, nil
}

// Upsert inserts or updates one source snapshot row keyed by item_id.
func (r *CatalogSourceRepository) Upsert(ctx context.Context, row CatalogSourceRow) error {
	if r == nil {
		return fmt.Errorf("catalog source repository is required")
	}

	normalized, err := validateCatalogSourceUpsertRow(row)
	if err != nil {
		return err
	}

	_, err = r.exec.ExecContext(
		normalizeContext(ctx),
		`INSERT INTO catalog_source_items (
			item_id,
			classifier,
			source_type,
			source_repo,
			parent_skill_id,
			resource_path,
			name,
			description,
			content,
			content_hash,
			content_writable,
			metadata_writable,
			last_synced_at,
			deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(item_id) DO UPDATE SET
			classifier = excluded.classifier,
			source_type = excluded.source_type,
			source_repo = excluded.source_repo,
			parent_skill_id = excluded.parent_skill_id,
			resource_path = excluded.resource_path,
			name = excluded.name,
			description = excluded.description,
			content = excluded.content,
			content_hash = excluded.content_hash,
			content_writable = excluded.content_writable,
			metadata_writable = excluded.metadata_writable,
			last_synced_at = excluded.last_synced_at,
			deleted_at = excluded.deleted_at;`,
		normalized.ItemID,
		normalized.Classifier,
		normalized.SourceType,
		pointerToAny(normalized.SourceRepo),
		pointerToAny(normalized.ParentSkillID),
		pointerToAny(normalized.ResourcePath),
		normalized.Name,
		normalized.Description,
		normalized.Content,
		normalized.ContentHash,
		boolToSQLiteInteger(normalized.ContentWritable),
		boolToSQLiteInteger(normalized.MetadataWritable),
		formatCatalogTimestamp(normalized.LastSyncedAt),
		formatOptionalTimestamp(normalized.DeletedAt),
	)
	if err != nil {
		return fmt.Errorf("upsert catalog source row %q: %w", normalized.ItemID, err)
	}

	return nil
}

// GetByItemID fetches one source snapshot row by item_id.
func (r *CatalogSourceRepository) GetByItemID(ctx context.Context, itemID string) (CatalogSourceRow, error) {
	if r == nil {
		return CatalogSourceRow{}, fmt.Errorf("catalog source repository is required")
	}

	normalizedItemID, err := normalizeRequiredID(itemID, "catalog source item_id")
	if err != nil {
		return CatalogSourceRow{}, err
	}

	row, err := scanCatalogSourceRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				item_id,
				classifier,
				source_type,
				source_repo,
				parent_skill_id,
				resource_path,
				name,
				description,
				content,
				content_hash,
				content_writable,
				metadata_writable,
				last_synced_at,
				deleted_at
			FROM catalog_source_items
			WHERE item_id = ?;`,
			normalizedItemID,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogSourceRow{}, ErrCatalogSourceNotFound
		}
		return CatalogSourceRow{}, fmt.Errorf("get catalog source row %q: %w", normalizedItemID, err)
	}

	return row, nil
}

// List returns source rows that match the provided filter with deterministic ordering.
func (r *CatalogSourceRepository) List(ctx context.Context, filter CatalogSourceListFilter) ([]CatalogSourceRow, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog source repository is required")
	}

	query, args, err := buildCatalogSourceListQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog source rows: %w", err)
	}
	defer rows.Close()

	result := make([]CatalogSourceRow, 0, 32)
	for rows.Next() {
		sourceRow, scanErr := scanCatalogSourceRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan catalog source list row: %w", scanErr)
		}
		result = append(result, sourceRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog source rows: %w", err)
	}

	return result, nil
}

// ListByItemIDs returns source rows constrained to a deterministic item_id subset.
func (r *CatalogSourceRepository) ListByItemIDs(
	ctx context.Context,
	itemIDs []string,
	includeDeleted bool,
) ([]CatalogSourceRow, error) {
	return r.List(ctx, CatalogSourceListFilter{ItemIDs: itemIDs, IncludeDeleted: includeDeleted})
}

// ListBySource returns source rows constrained by source type and optional source repo.
func (r *CatalogSourceRepository) ListBySource(
	ctx context.Context,
	sourceType CatalogSourceType,
	sourceRepo *string,
	includeDeleted bool,
) ([]CatalogSourceRow, error) {
	return r.List(ctx, CatalogSourceListFilter{
		SourceType:     &sourceType,
		SourceRepo:     sourceRepo,
		IncludeDeleted: includeDeleted,
	})
}

// SoftDeleteByItemID tombstones one source row by setting deleted_at.
func (r *CatalogSourceRepository) SoftDeleteByItemID(
	ctx context.Context,
	itemID string,
	deletedAt time.Time,
) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog source repository is required")
	}

	normalizedItemID, err := normalizeRequiredID(itemID, "catalog source item_id")
	if err != nil {
		return false, err
	}

	tombstoneAt := deletedAt
	if tombstoneAt.IsZero() {
		tombstoneAt = time.Now().UTC()
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`UPDATE catalog_source_items
		SET deleted_at = ?
		WHERE item_id = ?;`,
		formatCatalogTimestamp(tombstoneAt),
		normalizedItemID,
	)
	if err != nil {
		return false, fmt.Errorf("soft-delete catalog source row %q: %w", normalizedItemID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read soft-delete affected rows for %q: %w", normalizedItemID, err)
	}

	return rowsAffected > 0, nil
}

// RestoreByItemID clears deleted_at for one source row.
func (r *CatalogSourceRepository) RestoreByItemID(ctx context.Context, itemID string) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog source repository is required")
	}

	normalizedItemID, err := normalizeRequiredID(itemID, "catalog source item_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`UPDATE catalog_source_items
		SET deleted_at = NULL
		WHERE item_id = ?;`,
		normalizedItemID,
	)
	if err != nil {
		return false, fmt.Errorf("restore catalog source row %q: %w", normalizedItemID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read restore affected rows for %q: %w", normalizedItemID, err)
	}

	return rowsAffected > 0, nil
}

// DeleteByItemID soft-deletes one source row to preserve overlay history.
func (r *CatalogSourceRepository) DeleteByItemID(
	ctx context.Context,
	itemID string,
	deletedAt time.Time,
) (bool, error) {
	return r.SoftDeleteByItemID(ctx, itemID, deletedAt)
}

func buildCatalogSourceListQuery(filter CatalogSourceListFilter) (string, []any, error) {
	if strings.TrimSpace(filter.ItemID) != "" && len(filter.ItemIDs) > 0 {
		return "", nil, fmt.Errorf("catalog source filter cannot include both item_id and item_ids")
	}

	conditions := make([]string, 0, 6)
	args := make([]any, 0, 8)

	if !filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}

	if strings.TrimSpace(filter.ItemID) != "" {
		conditions = append(conditions, "item_id = ?")
		args = append(args, strings.TrimSpace(filter.ItemID))
	}

	normalizedIDs := normalizeOptionalIDList(filter.ItemIDs)
	if len(normalizedIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedIDs)), ",")
		conditions = append(conditions, "item_id IN ("+inClause+")")
		for _, itemID := range normalizedIDs {
			args = append(args, itemID)
		}
	}

	if filter.Classifier != nil {
		if !filter.Classifier.IsValid() {
			return "", nil, fmt.Errorf("catalog source filter classifier %q is invalid", *filter.Classifier)
		}
		conditions = append(conditions, "classifier = ?")
		args = append(args, *filter.Classifier)
	}

	if filter.SourceType != nil {
		if !filter.SourceType.IsValid() {
			return "", nil, fmt.Errorf("catalog source filter source_type %q is invalid", *filter.SourceType)
		}
		conditions = append(conditions, "source_type = ?")
		args = append(args, *filter.SourceType)
	}

	if filter.SourceRepo != nil {
		normalizedSourceRepo := strings.TrimSpace(*filter.SourceRepo)
		if normalizedSourceRepo == "" {
			conditions = append(conditions, "source_repo IS NULL")
		} else {
			conditions = append(conditions, "source_repo = ?")
			args = append(args, normalizedSourceRepo)
		}
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT
		item_id,
		classifier,
		source_type,
		source_repo,
		parent_skill_id,
		resource_path,
		name,
		description,
		content,
		content_hash,
		content_writable,
		metadata_writable,
		last_synced_at,
		deleted_at
	FROM catalog_source_items`)
	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY item_id ASC;")

	return queryBuilder.String(), args, nil
}

func formatOptionalTimestamp(timestamp *time.Time) any {
	if timestamp == nil {
		return nil
	}
	return formatCatalogTimestamp(timestamp.UTC())
}
