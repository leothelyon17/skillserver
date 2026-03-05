package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CatalogItemTaxonomyAssignmentRepository provides access to catalog_item_taxonomy_assignments rows.
type CatalogItemTaxonomyAssignmentRepository struct {
	exec catalogQueryExecutor
}

// NewCatalogItemTaxonomyAssignmentRepository creates a taxonomy assignment repository around a DB or transaction handle.
func NewCatalogItemTaxonomyAssignmentRepository(
	exec catalogQueryExecutor,
) (*CatalogItemTaxonomyAssignmentRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("catalog item taxonomy assignment repository query executor is required")
	}

	return &CatalogItemTaxonomyAssignmentRepository{exec: exec}, nil
}

// Upsert inserts or updates one item taxonomy assignment row keyed by item_id.
func (r *CatalogItemTaxonomyAssignmentRepository) Upsert(
	ctx context.Context,
	row CatalogItemTaxonomyAssignmentRow,
) error {
	if r == nil {
		return fmt.Errorf("catalog item taxonomy assignment repository is required")
	}

	normalized, err := validateCatalogItemTaxonomyAssignmentUpsertRow(row)
	if err != nil {
		return err
	}

	_, err = r.exec.ExecContext(
		normalizeContext(ctx),
		`INSERT INTO catalog_item_taxonomy_assignments (
			item_id,
			primary_domain_id,
			primary_subdomain_id,
			secondary_domain_id,
			secondary_subdomain_id,
			updated_at,
			updated_by
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(item_id) DO UPDATE SET
			primary_domain_id = excluded.primary_domain_id,
			primary_subdomain_id = excluded.primary_subdomain_id,
			secondary_domain_id = excluded.secondary_domain_id,
			secondary_subdomain_id = excluded.secondary_subdomain_id,
			updated_at = excluded.updated_at,
			updated_by = excluded.updated_by;`,
		normalized.ItemID,
		pointerToAny(normalized.PrimaryDomainID),
		pointerToAny(normalized.PrimarySubdomainID),
		pointerToAny(normalized.SecondaryDomainID),
		pointerToAny(normalized.SecondarySubdomainID),
		formatCatalogTimestamp(normalized.UpdatedAt),
		pointerToAny(normalized.UpdatedBy),
	)
	if err != nil {
		return fmt.Errorf("upsert catalog item taxonomy assignment row %q: %w", normalized.ItemID, err)
	}

	return nil
}

// GetByItemID fetches one item taxonomy assignment row by item_id.
func (r *CatalogItemTaxonomyAssignmentRepository) GetByItemID(
	ctx context.Context,
	itemID string,
) (CatalogItemTaxonomyAssignmentRow, error) {
	if r == nil {
		return CatalogItemTaxonomyAssignmentRow{}, fmt.Errorf("catalog item taxonomy assignment repository is required")
	}

	normalizedItemID, err := normalizeRequiredID(itemID, "catalog item taxonomy assignment item_id")
	if err != nil {
		return CatalogItemTaxonomyAssignmentRow{}, err
	}

	row, err := scanCatalogItemTaxonomyAssignmentRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				item_id,
				primary_domain_id,
				primary_subdomain_id,
				secondary_domain_id,
				secondary_subdomain_id,
				updated_at,
				updated_by
			FROM catalog_item_taxonomy_assignments
			WHERE item_id = ?;`,
			normalizedItemID,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogItemTaxonomyAssignmentRow{}, ErrCatalogItemTaxonomyAssignmentNotFound
		}
		return CatalogItemTaxonomyAssignmentRow{}, fmt.Errorf(
			"get catalog item taxonomy assignment row %q: %w",
			normalizedItemID,
			err,
		)
	}

	return row, nil
}

// List returns assignment rows that match the provided filter with deterministic ordering.
func (r *CatalogItemTaxonomyAssignmentRepository) List(
	ctx context.Context,
	filter CatalogItemTaxonomyAssignmentListFilter,
) ([]CatalogItemTaxonomyAssignmentRow, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog item taxonomy assignment repository is required")
	}

	query, args, err := buildCatalogItemTaxonomyAssignmentListQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog item taxonomy assignment rows: %w", err)
	}
	defer rows.Close()

	result := make([]CatalogItemTaxonomyAssignmentRow, 0, 32)
	for rows.Next() {
		assignmentRow, scanErr := scanCatalogItemTaxonomyAssignmentRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan catalog item taxonomy assignment list row: %w", scanErr)
		}
		result = append(result, assignmentRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog item taxonomy assignment rows: %w", err)
	}

	return result, nil
}

// DeleteByItemID deletes one taxonomy assignment row by item_id.
func (r *CatalogItemTaxonomyAssignmentRepository) DeleteByItemID(
	ctx context.Context,
	itemID string,
) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog item taxonomy assignment repository is required")
	}

	normalizedItemID, err := normalizeRequiredID(itemID, "catalog item taxonomy assignment item_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_item_taxonomy_assignments WHERE item_id = ?;`,
		normalizedItemID,
	)
	if err != nil {
		return false, fmt.Errorf("delete catalog item taxonomy assignment row %q: %w", normalizedItemID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"read delete affected rows for catalog item taxonomy assignment %q: %w",
			normalizedItemID,
			err,
		)
	}

	return rowsAffected > 0, nil
}

func buildCatalogItemTaxonomyAssignmentListQuery(
	filter CatalogItemTaxonomyAssignmentListFilter,
) (string, []any, error) {
	if strings.TrimSpace(filter.ItemID) != "" && len(filter.ItemIDs) > 0 {
		return "", nil, fmt.Errorf("catalog item taxonomy assignment filter cannot include both item_id and item_ids")
	}

	conditions := make([]string, 0, 8)
	args := make([]any, 0, 12)

	if strings.TrimSpace(filter.ItemID) != "" {
		conditions = append(conditions, "item_id = ?")
		args = append(args, strings.TrimSpace(filter.ItemID))
	}

	normalizedItemIDs := normalizeOptionalIDList(filter.ItemIDs)
	if len(normalizedItemIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedItemIDs)), ",")
		conditions = append(conditions, "item_id IN ("+inClause+")")
		for _, itemID := range normalizedItemIDs {
			args = append(args, itemID)
		}
	}

	if normalized := normalizeOptionalForeignID(filter.PrimaryDomainID); normalized != nil {
		conditions = append(conditions, "primary_domain_id = ?")
		args = append(args, *normalized)
	}

	if normalized := normalizeOptionalForeignID(filter.SecondaryDomainID); normalized != nil {
		conditions = append(conditions, "secondary_domain_id = ?")
		args = append(args, *normalized)
	}

	if normalized := normalizeOptionalForeignID(filter.DomainID); normalized != nil {
		conditions = append(conditions, "(primary_domain_id = ? OR secondary_domain_id = ?)")
		args = append(args, *normalized, *normalized)
	}

	if normalized := normalizeOptionalForeignID(filter.PrimarySubdomainID); normalized != nil {
		conditions = append(conditions, "primary_subdomain_id = ?")
		args = append(args, *normalized)
	}

	if normalized := normalizeOptionalForeignID(filter.SecondarySubdomainID); normalized != nil {
		conditions = append(conditions, "secondary_subdomain_id = ?")
		args = append(args, *normalized)
	}

	if normalized := normalizeOptionalForeignID(filter.SubdomainID); normalized != nil {
		conditions = append(conditions, "(primary_subdomain_id = ? OR secondary_subdomain_id = ?)")
		args = append(args, *normalized, *normalized)
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT
		item_id,
		primary_domain_id,
		primary_subdomain_id,
		secondary_domain_id,
		secondary_subdomain_id,
		updated_at,
		updated_by
	FROM catalog_item_taxonomy_assignments`)
	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY item_id ASC;")

	return queryBuilder.String(), args, nil
}

// CatalogItemTagAssignmentRepository provides access to catalog_item_tag_assignments rows.
type CatalogItemTagAssignmentRepository struct {
	exec catalogQueryExecutor
}

// NewCatalogItemTagAssignmentRepository creates an item-tag assignment repository around a DB or transaction handle.
func NewCatalogItemTagAssignmentRepository(
	exec catalogQueryExecutor,
) (*CatalogItemTagAssignmentRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("catalog item tag assignment repository query executor is required")
	}

	return &CatalogItemTagAssignmentRepository{exec: exec}, nil
}

// ReplaceForItemID replaces one item's tag set using delete+insert semantics inside one transaction.
func (r *CatalogItemTagAssignmentRepository) ReplaceForItemID(
	ctx context.Context,
	itemID string,
	tagIDs []string,
	createdAt time.Time,
) error {
	if r == nil {
		return fmt.Errorf("catalog item tag assignment repository is required")
	}

	normalizedItemID, err := normalizeRequiredID(itemID, "catalog item tag assignment item_id")
	if err != nil {
		return err
	}

	normalizedTagIDs := normalizeOptionalIDList(tagIDs)
	normalizedCreatedAt := createdAt
	if normalizedCreatedAt.IsZero() {
		normalizedCreatedAt = time.Now().UTC()
	} else {
		normalizedCreatedAt = normalizedCreatedAt.UTC()
	}
	normalizedCtx := normalizeContext(ctx)

	err = withCatalogWriteTransaction(normalizedCtx, r.exec, func(tx catalogQueryExecutor) error {
		if len(normalizedTagIDs) == 0 {
			if _, deleteErr := tx.ExecContext(
				normalizedCtx,
				`DELETE FROM catalog_item_tag_assignments WHERE item_id = ?;`,
				normalizedItemID,
			); deleteErr != nil {
				return fmt.Errorf("delete catalog item tag assignments for %q: %w", normalizedItemID, deleteErr)
			}
			return nil
		}

		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedTagIDs)), ",")
		deleteArgs := make([]any, 0, len(normalizedTagIDs)+1)
		deleteArgs = append(deleteArgs, normalizedItemID)
		for _, tagID := range normalizedTagIDs {
			deleteArgs = append(deleteArgs, tagID)
		}

		if _, deleteErr := tx.ExecContext(
			normalizedCtx,
			`DELETE FROM catalog_item_tag_assignments
			WHERE item_id = ?
			AND tag_id NOT IN (`+inClause+`);`,
			deleteArgs...,
		); deleteErr != nil {
			return fmt.Errorf("delete stale catalog item tag assignments for %q: %w", normalizedItemID, deleteErr)
		}

		for _, tagID := range normalizedTagIDs {
			if _, insertErr := tx.ExecContext(
				normalizedCtx,
				`INSERT INTO catalog_item_tag_assignments (item_id, tag_id, created_at)
				VALUES (?, ?, ?)
				ON CONFLICT(item_id, tag_id) DO NOTHING;`,
				normalizedItemID,
				tagID,
				formatCatalogTimestamp(normalizedCreatedAt),
			); insertErr != nil {
				return fmt.Errorf(
					"insert catalog item tag assignment row %q/%q: %w",
					normalizedItemID,
					tagID,
					insertErr,
				)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// List returns item-tag assignment rows that match the provided filter with deterministic ordering.
func (r *CatalogItemTagAssignmentRepository) List(
	ctx context.Context,
	filter CatalogItemTagAssignmentListFilter,
) ([]CatalogItemTagAssignmentRow, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog item tag assignment repository is required")
	}

	query, args, err := buildCatalogItemTagAssignmentListQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog item tag assignment rows: %w", err)
	}
	defer rows.Close()

	result := make([]CatalogItemTagAssignmentRow, 0, 32)
	for rows.Next() {
		assignmentRow, scanErr := scanCatalogItemTagAssignmentRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan catalog item tag assignment list row: %w", scanErr)
		}
		result = append(result, assignmentRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog item tag assignment rows: %w", err)
	}

	return result, nil
}

// ListByItemID returns tag assignment rows for one item with deterministic ordering.
func (r *CatalogItemTagAssignmentRepository) ListByItemID(
	ctx context.Context,
	itemID string,
) ([]CatalogItemTagAssignmentRow, error) {
	return r.List(ctx, CatalogItemTagAssignmentListFilter{ItemID: itemID})
}

// ListItemIDsByTagIDs returns item IDs that match a tag filter.
func (r *CatalogItemTagAssignmentRepository) ListItemIDsByTagIDs(
	ctx context.Context,
	tagIDs []string,
	matchAll bool,
) ([]string, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog item tag assignment repository is required")
	}

	normalizedTagIDs := normalizeOptionalIDList(tagIDs)
	if len(normalizedTagIDs) == 0 {
		return []string{}, nil
	}

	inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedTagIDs)), ",")
	args := make([]any, 0, len(normalizedTagIDs)+1)
	for _, tagID := range normalizedTagIDs {
		args = append(args, tagID)
	}

	query := `SELECT DISTINCT item_id
	FROM catalog_item_tag_assignments
	WHERE tag_id IN (` + inClause + `)
	ORDER BY item_id ASC;`
	if matchAll {
		query = `SELECT item_id
		FROM catalog_item_tag_assignments
		WHERE tag_id IN (` + inClause + `)
		GROUP BY item_id
		HAVING COUNT(DISTINCT tag_id) = ?
		ORDER BY item_id ASC;`
		args = append(args, len(normalizedTagIDs))
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog item ids by tag filter: %w", err)
	}
	defer rows.Close()

	itemIDs := make([]string, 0, 32)
	for rows.Next() {
		var itemID string
		if err := rows.Scan(&itemID); err != nil {
			return nil, fmt.Errorf("scan catalog item id by tag filter row: %w", err)
		}
		itemIDs = append(itemIDs, itemID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog item ids by tag filter rows: %w", err)
	}

	return itemIDs, nil
}

// DeleteByItemID deletes all tag assignments for one item.
func (r *CatalogItemTagAssignmentRepository) DeleteByItemID(
	ctx context.Context,
	itemID string,
) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog item tag assignment repository is required")
	}

	normalizedItemID, err := normalizeRequiredID(itemID, "catalog item tag assignment item_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_item_tag_assignments WHERE item_id = ?;`,
		normalizedItemID,
	)
	if err != nil {
		return false, fmt.Errorf("delete catalog item tag assignment rows for %q: %w", normalizedItemID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read delete affected rows for catalog item tag assignments %q: %w", normalizedItemID, err)
	}

	return rowsAffected > 0, nil
}

func buildCatalogItemTagAssignmentListQuery(
	filter CatalogItemTagAssignmentListFilter,
) (string, []any, error) {
	if strings.TrimSpace(filter.ItemID) != "" && len(filter.ItemIDs) > 0 {
		return "", nil, fmt.Errorf("catalog item tag assignment filter cannot include both item_id and item_ids")
	}
	if strings.TrimSpace(filter.TagID) != "" && len(filter.TagIDs) > 0 {
		return "", nil, fmt.Errorf("catalog item tag assignment filter cannot include both tag_id and tag_ids")
	}

	conditions := make([]string, 0, 4)
	args := make([]any, 0, 10)

	if strings.TrimSpace(filter.ItemID) != "" {
		conditions = append(conditions, "item_id = ?")
		args = append(args, strings.TrimSpace(filter.ItemID))
	}

	normalizedItemIDs := normalizeOptionalIDList(filter.ItemIDs)
	if len(normalizedItemIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedItemIDs)), ",")
		conditions = append(conditions, "item_id IN ("+inClause+")")
		for _, itemID := range normalizedItemIDs {
			args = append(args, itemID)
		}
	}

	if strings.TrimSpace(filter.TagID) != "" {
		conditions = append(conditions, "tag_id = ?")
		args = append(args, strings.TrimSpace(filter.TagID))
	}

	normalizedTagIDs := normalizeOptionalIDList(filter.TagIDs)
	if len(normalizedTagIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedTagIDs)), ",")
		conditions = append(conditions, "tag_id IN ("+inClause+")")
		for _, tagID := range normalizedTagIDs {
			args = append(args, tagID)
		}
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT
		item_id,
		tag_id,
		created_at
	FROM catalog_item_tag_assignments`)
	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY item_id ASC, tag_id ASC;")

	return queryBuilder.String(), args, nil
}

type catalogTransactionBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func withCatalogWriteTransaction(
	ctx context.Context,
	exec catalogQueryExecutor,
	execute func(tx catalogQueryExecutor) error,
) error {
	if activeTx, ok := exec.(*sql.Tx); ok {
		return execute(activeTx)
	}

	beginner, ok := exec.(catalogTransactionBeginner)
	if !ok {
		return fmt.Errorf("catalog repository query executor does not support transactions")
	}

	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start catalog repository transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := execute(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit catalog repository transaction: %w", err)
	}
	committed = true

	return nil
}
