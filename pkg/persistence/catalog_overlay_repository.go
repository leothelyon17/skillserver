package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// CatalogMetadataOverlayRepository provides access to catalog_metadata_overlays rows.
type CatalogMetadataOverlayRepository struct {
	exec catalogQueryExecutor
}

// NewCatalogMetadataOverlayRepository creates an overlay repository around a DB or transaction handle.
func NewCatalogMetadataOverlayRepository(exec catalogQueryExecutor) (*CatalogMetadataOverlayRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("catalog metadata overlay repository query executor is required")
	}

	return &CatalogMetadataOverlayRepository{exec: exec}, nil
}

// Upsert inserts or updates one metadata overlay row keyed by item_id.
func (r *CatalogMetadataOverlayRepository) Upsert(ctx context.Context, row CatalogMetadataOverlayRow) error {
	if r == nil {
		return fmt.Errorf("catalog metadata overlay repository is required")
	}

	normalized, err := validateCatalogMetadataOverlayUpsertRow(row)
	if err != nil {
		return err
	}

	_, err = r.exec.ExecContext(
		normalizeContext(ctx),
		`INSERT INTO catalog_metadata_overlays (
			item_id,
			display_name_override,
			description_override,
			custom_metadata_json,
			labels_json,
			updated_at,
			updated_by
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(item_id) DO UPDATE SET
			display_name_override = excluded.display_name_override,
			description_override = excluded.description_override,
			custom_metadata_json = excluded.custom_metadata_json,
			labels_json = excluded.labels_json,
			updated_at = excluded.updated_at,
			updated_by = excluded.updated_by;`,
		normalized.ItemID,
		pointerToAny(normalized.DisplayNameOverride),
		pointerToAny(normalized.DescriptionOverride),
		normalized.CustomMetadataJSON,
		normalized.LabelsJSON,
		formatCatalogTimestamp(normalized.UpdatedAt),
		pointerToAny(normalized.UpdatedBy),
	)
	if err != nil {
		return fmt.Errorf("upsert catalog metadata overlay row %q: %w", normalized.ItemID, err)
	}

	return nil
}

// GetByItemID fetches one metadata overlay row by item_id.
func (r *CatalogMetadataOverlayRepository) GetByItemID(
	ctx context.Context,
	itemID string,
) (CatalogMetadataOverlayRow, error) {
	if r == nil {
		return CatalogMetadataOverlayRow{}, fmt.Errorf("catalog metadata overlay repository is required")
	}

	normalizedItemID, err := normalizeRequiredID(itemID, "catalog metadata overlay item_id")
	if err != nil {
		return CatalogMetadataOverlayRow{}, err
	}

	row, err := scanCatalogMetadataOverlayRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				item_id,
				display_name_override,
				description_override,
				custom_metadata_json,
				labels_json,
				updated_at,
				updated_by
			FROM catalog_metadata_overlays
			WHERE item_id = ?;`,
			normalizedItemID,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogMetadataOverlayRow{}, ErrCatalogMetadataOverlayNotFound
		}
		return CatalogMetadataOverlayRow{}, fmt.Errorf("get catalog metadata overlay row %q: %w", normalizedItemID, err)
	}

	return row, nil
}

// List returns metadata overlay rows that match the provided filter with deterministic ordering.
func (r *CatalogMetadataOverlayRepository) List(
	ctx context.Context,
	filter CatalogMetadataOverlayListFilter,
) ([]CatalogMetadataOverlayRow, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog metadata overlay repository is required")
	}

	query, args, err := buildCatalogMetadataOverlayListQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog metadata overlay rows: %w", err)
	}
	defer rows.Close()

	result := make([]CatalogMetadataOverlayRow, 0, 16)
	for rows.Next() {
		overlayRow, scanErr := scanCatalogMetadataOverlayRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan catalog metadata overlay list row: %w", scanErr)
		}
		result = append(result, overlayRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog metadata overlay rows: %w", err)
	}

	return result, nil
}

// DeleteByItemID deletes one metadata overlay row by item_id.
func (r *CatalogMetadataOverlayRepository) DeleteByItemID(
	ctx context.Context,
	itemID string,
) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog metadata overlay repository is required")
	}

	normalizedItemID, err := normalizeRequiredID(itemID, "catalog metadata overlay item_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_metadata_overlays WHERE item_id = ?;`,
		normalizedItemID,
	)
	if err != nil {
		return false, fmt.Errorf("delete catalog metadata overlay row %q: %w", normalizedItemID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read delete affected rows for %q: %w", normalizedItemID, err)
	}

	return rowsAffected > 0, nil
}

func buildCatalogMetadataOverlayListQuery(filter CatalogMetadataOverlayListFilter) (string, []any, error) {
	if strings.TrimSpace(filter.ItemID) != "" && len(filter.ItemIDs) > 0 {
		return "", nil, fmt.Errorf("catalog metadata overlay filter cannot include both item_id and item_ids")
	}

	conditions := make([]string, 0, 2)
	args := make([]any, 0, 4)

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

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT
		item_id,
		display_name_override,
		description_override,
		custom_metadata_json,
		labels_json,
		updated_at,
		updated_by
	FROM catalog_metadata_overlays`)
	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY item_id ASC;")

	return queryBuilder.String(), args, nil
}
