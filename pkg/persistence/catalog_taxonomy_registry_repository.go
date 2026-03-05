package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// CatalogDomainRepository provides access to catalog_domains rows.
type CatalogDomainRepository struct {
	exec catalogQueryExecutor
}

// NewCatalogDomainRepository creates a domain repository around a DB or transaction handle.
func NewCatalogDomainRepository(exec catalogQueryExecutor) (*CatalogDomainRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("catalog domain repository query executor is required")
	}

	return &CatalogDomainRepository{exec: exec}, nil
}

// Create inserts one domain row.
func (r *CatalogDomainRepository) Create(ctx context.Context, row CatalogDomainRow) error {
	if r == nil {
		return fmt.Errorf("catalog domain repository is required")
	}

	normalized, err := validateCatalogDomainCreateRow(row)
	if err != nil {
		return err
	}

	_, err = r.exec.ExecContext(
		normalizeContext(ctx),
		`INSERT INTO catalog_domains (
			domain_id,
			key,
			name,
			description,
			active,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?);`,
		normalized.DomainID,
		normalized.Key,
		normalized.Name,
		normalized.Description,
		boolToSQLiteInteger(normalized.Active),
		formatCatalogTimestamp(normalized.CreatedAt),
		formatCatalogTimestamp(normalized.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create catalog domain row %q: %w", normalized.DomainID, err)
	}

	return nil
}

// Update updates mutable fields for one domain row by domain_id.
func (r *CatalogDomainRepository) Update(ctx context.Context, row CatalogDomainRow) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog domain repository is required")
	}

	normalized, err := validateCatalogDomainUpdateRow(row)
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`UPDATE catalog_domains
		SET key = ?,
			name = ?,
			description = ?,
			active = ?,
			updated_at = ?
		WHERE domain_id = ?;`,
		normalized.Key,
		normalized.Name,
		normalized.Description,
		boolToSQLiteInteger(normalized.Active),
		formatCatalogTimestamp(normalized.UpdatedAt),
		normalized.DomainID,
	)
	if err != nil {
		return false, fmt.Errorf("update catalog domain row %q: %w", normalized.DomainID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read update affected rows for catalog domain %q: %w", normalized.DomainID, err)
	}

	return rowsAffected > 0, nil
}

// GetByDomainID fetches one domain row by domain_id.
func (r *CatalogDomainRepository) GetByDomainID(ctx context.Context, domainID string) (CatalogDomainRow, error) {
	if r == nil {
		return CatalogDomainRow{}, fmt.Errorf("catalog domain repository is required")
	}

	normalizedDomainID, err := normalizeRequiredID(domainID, "catalog domain domain_id")
	if err != nil {
		return CatalogDomainRow{}, err
	}

	row, err := scanCatalogDomainRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				domain_id,
				key,
				name,
				description,
				active,
				created_at,
				updated_at
			FROM catalog_domains
			WHERE domain_id = ?;`,
			normalizedDomainID,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogDomainRow{}, ErrCatalogDomainNotFound
		}
		return CatalogDomainRow{}, fmt.Errorf("get catalog domain row %q: %w", normalizedDomainID, err)
	}

	return row, nil
}

// GetByKey fetches one domain row by key.
func (r *CatalogDomainRepository) GetByKey(ctx context.Context, key string) (CatalogDomainRow, error) {
	if r == nil {
		return CatalogDomainRow{}, fmt.Errorf("catalog domain repository is required")
	}

	normalizedKey, err := normalizeRequiredID(key, "catalog domain key")
	if err != nil {
		return CatalogDomainRow{}, err
	}

	row, err := scanCatalogDomainRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				domain_id,
				key,
				name,
				description,
				active,
				created_at,
				updated_at
			FROM catalog_domains
			WHERE key = ?;`,
			normalizedKey,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogDomainRow{}, ErrCatalogDomainNotFound
		}
		return CatalogDomainRow{}, fmt.Errorf("get catalog domain row by key %q: %w", normalizedKey, err)
	}

	return row, nil
}

// List returns domain rows that match the provided filter with deterministic ordering.
func (r *CatalogDomainRepository) List(ctx context.Context, filter CatalogDomainListFilter) ([]CatalogDomainRow, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog domain repository is required")
	}

	query, args, err := buildCatalogDomainListQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog domain rows: %w", err)
	}
	defer rows.Close()

	result := make([]CatalogDomainRow, 0, 16)
	for rows.Next() {
		domainRow, scanErr := scanCatalogDomainRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan catalog domain list row: %w", scanErr)
		}
		result = append(result, domainRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog domain rows: %w", err)
	}

	return result, nil
}

// DeleteByDomainID deletes one domain row by domain_id.
func (r *CatalogDomainRepository) DeleteByDomainID(ctx context.Context, domainID string) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog domain repository is required")
	}

	normalizedDomainID, err := normalizeRequiredID(domainID, "catalog domain domain_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_domains WHERE domain_id = ?;`,
		normalizedDomainID,
	)
	if err != nil {
		return false, fmt.Errorf("delete catalog domain row %q: %w", normalizedDomainID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read delete affected rows for catalog domain %q: %w", normalizedDomainID, err)
	}

	return rowsAffected > 0, nil
}

func buildCatalogDomainListQuery(filter CatalogDomainListFilter) (string, []any, error) {
	if strings.TrimSpace(filter.DomainID) != "" && len(filter.DomainIDs) > 0 {
		return "", nil, fmt.Errorf("catalog domain filter cannot include both domain_id and domain_ids")
	}
	if strings.TrimSpace(filter.Key) != "" && len(filter.Keys) > 0 {
		return "", nil, fmt.Errorf("catalog domain filter cannot include both key and keys")
	}

	conditions := make([]string, 0, 5)
	args := make([]any, 0, 8)

	if strings.TrimSpace(filter.DomainID) != "" {
		conditions = append(conditions, "domain_id = ?")
		args = append(args, strings.TrimSpace(filter.DomainID))
	}

	normalizedDomainIDs := normalizeOptionalIDList(filter.DomainIDs)
	if len(normalizedDomainIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedDomainIDs)), ",")
		conditions = append(conditions, "domain_id IN ("+inClause+")")
		for _, domainID := range normalizedDomainIDs {
			args = append(args, domainID)
		}
	}

	if strings.TrimSpace(filter.Key) != "" {
		conditions = append(conditions, "key = ?")
		args = append(args, strings.TrimSpace(filter.Key))
	}

	normalizedKeys := normalizeOptionalIDList(filter.Keys)
	if len(normalizedKeys) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedKeys)), ",")
		conditions = append(conditions, "key IN ("+inClause+")")
		for _, key := range normalizedKeys {
			args = append(args, key)
		}
	}

	if filter.Active != nil {
		conditions = append(conditions, "active = ?")
		args = append(args, boolToSQLiteInteger(*filter.Active))
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT
		domain_id,
		key,
		name,
		description,
		active,
		created_at,
		updated_at
	FROM catalog_domains`)
	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY key ASC, domain_id ASC;")

	return queryBuilder.String(), args, nil
}

// CatalogSubdomainRepository provides access to catalog_subdomains rows.
type CatalogSubdomainRepository struct {
	exec catalogQueryExecutor
}

// NewCatalogSubdomainRepository creates a subdomain repository around a DB or transaction handle.
func NewCatalogSubdomainRepository(exec catalogQueryExecutor) (*CatalogSubdomainRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("catalog subdomain repository query executor is required")
	}

	return &CatalogSubdomainRepository{exec: exec}, nil
}

// Create inserts one subdomain row.
func (r *CatalogSubdomainRepository) Create(ctx context.Context, row CatalogSubdomainRow) error {
	if r == nil {
		return fmt.Errorf("catalog subdomain repository is required")
	}

	normalized, err := validateCatalogSubdomainCreateRow(row)
	if err != nil {
		return err
	}

	_, err = r.exec.ExecContext(
		normalizeContext(ctx),
		`INSERT INTO catalog_subdomains (
			subdomain_id,
			domain_id,
			key,
			name,
			description,
			active,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
		normalized.SubdomainID,
		normalized.DomainID,
		normalized.Key,
		normalized.Name,
		normalized.Description,
		boolToSQLiteInteger(normalized.Active),
		formatCatalogTimestamp(normalized.CreatedAt),
		formatCatalogTimestamp(normalized.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create catalog subdomain row %q: %w", normalized.SubdomainID, err)
	}

	return nil
}

// Update updates mutable fields for one subdomain row by subdomain_id.
func (r *CatalogSubdomainRepository) Update(ctx context.Context, row CatalogSubdomainRow) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog subdomain repository is required")
	}

	normalized, err := validateCatalogSubdomainUpdateRow(row)
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`UPDATE catalog_subdomains
		SET domain_id = ?,
			key = ?,
			name = ?,
			description = ?,
			active = ?,
			updated_at = ?
		WHERE subdomain_id = ?;`,
		normalized.DomainID,
		normalized.Key,
		normalized.Name,
		normalized.Description,
		boolToSQLiteInteger(normalized.Active),
		formatCatalogTimestamp(normalized.UpdatedAt),
		normalized.SubdomainID,
	)
	if err != nil {
		return false, fmt.Errorf("update catalog subdomain row %q: %w", normalized.SubdomainID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read update affected rows for catalog subdomain %q: %w", normalized.SubdomainID, err)
	}

	return rowsAffected > 0, nil
}

// GetBySubdomainID fetches one subdomain row by subdomain_id.
func (r *CatalogSubdomainRepository) GetBySubdomainID(ctx context.Context, subdomainID string) (CatalogSubdomainRow, error) {
	if r == nil {
		return CatalogSubdomainRow{}, fmt.Errorf("catalog subdomain repository is required")
	}

	normalizedSubdomainID, err := normalizeRequiredID(subdomainID, "catalog subdomain subdomain_id")
	if err != nil {
		return CatalogSubdomainRow{}, err
	}

	row, err := scanCatalogSubdomainRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				subdomain_id,
				domain_id,
				key,
				name,
				description,
				active,
				created_at,
				updated_at
			FROM catalog_subdomains
			WHERE subdomain_id = ?;`,
			normalizedSubdomainID,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogSubdomainRow{}, ErrCatalogSubdomainNotFound
		}
		return CatalogSubdomainRow{}, fmt.Errorf("get catalog subdomain row %q: %w", normalizedSubdomainID, err)
	}

	return row, nil
}

// GetByDomainIDAndKey fetches one subdomain row by domain_id and key.
func (r *CatalogSubdomainRepository) GetByDomainIDAndKey(
	ctx context.Context,
	domainID string,
	key string,
) (CatalogSubdomainRow, error) {
	if r == nil {
		return CatalogSubdomainRow{}, fmt.Errorf("catalog subdomain repository is required")
	}

	normalizedDomainID, err := normalizeRequiredID(domainID, "catalog subdomain domain_id")
	if err != nil {
		return CatalogSubdomainRow{}, err
	}
	normalizedKey, err := normalizeRequiredID(key, "catalog subdomain key")
	if err != nil {
		return CatalogSubdomainRow{}, err
	}

	row, err := scanCatalogSubdomainRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				subdomain_id,
				domain_id,
				key,
				name,
				description,
				active,
				created_at,
				updated_at
			FROM catalog_subdomains
			WHERE domain_id = ? AND key = ?;`,
			normalizedDomainID,
			normalizedKey,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogSubdomainRow{}, ErrCatalogSubdomainNotFound
		}
		return CatalogSubdomainRow{}, fmt.Errorf(
			"get catalog subdomain row by domain %q and key %q: %w",
			normalizedDomainID,
			normalizedKey,
			err,
		)
	}

	return row, nil
}

// List returns subdomain rows that match the provided filter with deterministic ordering.
func (r *CatalogSubdomainRepository) List(
	ctx context.Context,
	filter CatalogSubdomainListFilter,
) ([]CatalogSubdomainRow, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog subdomain repository is required")
	}

	query, args, err := buildCatalogSubdomainListQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog subdomain rows: %w", err)
	}
	defer rows.Close()

	result := make([]CatalogSubdomainRow, 0, 16)
	for rows.Next() {
		subdomainRow, scanErr := scanCatalogSubdomainRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan catalog subdomain list row: %w", scanErr)
		}
		result = append(result, subdomainRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog subdomain rows: %w", err)
	}

	return result, nil
}

// DeleteBySubdomainID deletes one subdomain row by subdomain_id.
func (r *CatalogSubdomainRepository) DeleteBySubdomainID(ctx context.Context, subdomainID string) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog subdomain repository is required")
	}

	normalizedSubdomainID, err := normalizeRequiredID(subdomainID, "catalog subdomain subdomain_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_subdomains WHERE subdomain_id = ?;`,
		normalizedSubdomainID,
	)
	if err != nil {
		return false, fmt.Errorf("delete catalog subdomain row %q: %w", normalizedSubdomainID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read delete affected rows for catalog subdomain %q: %w", normalizedSubdomainID, err)
	}

	return rowsAffected > 0, nil
}

func buildCatalogSubdomainListQuery(filter CatalogSubdomainListFilter) (string, []any, error) {
	if strings.TrimSpace(filter.SubdomainID) != "" && len(filter.SubdomainIDs) > 0 {
		return "", nil, fmt.Errorf("catalog subdomain filter cannot include both subdomain_id and subdomain_ids")
	}
	if strings.TrimSpace(filter.DomainID) != "" && len(filter.DomainIDs) > 0 {
		return "", nil, fmt.Errorf("catalog subdomain filter cannot include both domain_id and domain_ids")
	}
	if strings.TrimSpace(filter.Key) != "" && len(filter.Keys) > 0 {
		return "", nil, fmt.Errorf("catalog subdomain filter cannot include both key and keys")
	}

	conditions := make([]string, 0, 6)
	args := make([]any, 0, 10)

	if strings.TrimSpace(filter.SubdomainID) != "" {
		conditions = append(conditions, "subdomain_id = ?")
		args = append(args, strings.TrimSpace(filter.SubdomainID))
	}

	normalizedSubdomainIDs := normalizeOptionalIDList(filter.SubdomainIDs)
	if len(normalizedSubdomainIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedSubdomainIDs)), ",")
		conditions = append(conditions, "subdomain_id IN ("+inClause+")")
		for _, subdomainID := range normalizedSubdomainIDs {
			args = append(args, subdomainID)
		}
	}

	if strings.TrimSpace(filter.DomainID) != "" {
		conditions = append(conditions, "domain_id = ?")
		args = append(args, strings.TrimSpace(filter.DomainID))
	}

	normalizedDomainIDs := normalizeOptionalIDList(filter.DomainIDs)
	if len(normalizedDomainIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedDomainIDs)), ",")
		conditions = append(conditions, "domain_id IN ("+inClause+")")
		for _, domainID := range normalizedDomainIDs {
			args = append(args, domainID)
		}
	}

	if strings.TrimSpace(filter.Key) != "" {
		conditions = append(conditions, "key = ?")
		args = append(args, strings.TrimSpace(filter.Key))
	}

	normalizedKeys := normalizeOptionalIDList(filter.Keys)
	if len(normalizedKeys) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedKeys)), ",")
		conditions = append(conditions, "key IN ("+inClause+")")
		for _, key := range normalizedKeys {
			args = append(args, key)
		}
	}

	if filter.Active != nil {
		conditions = append(conditions, "active = ?")
		args = append(args, boolToSQLiteInteger(*filter.Active))
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT
		subdomain_id,
		domain_id,
		key,
		name,
		description,
		active,
		created_at,
		updated_at
	FROM catalog_subdomains`)
	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY domain_id ASC, key ASC, subdomain_id ASC;")

	return queryBuilder.String(), args, nil
}

// CatalogTagRepository provides access to catalog_tags rows.
type CatalogTagRepository struct {
	exec catalogQueryExecutor
}

// NewCatalogTagRepository creates a tag repository around a DB or transaction handle.
func NewCatalogTagRepository(exec catalogQueryExecutor) (*CatalogTagRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("catalog tag repository query executor is required")
	}

	return &CatalogTagRepository{exec: exec}, nil
}

// Create inserts one tag row.
func (r *CatalogTagRepository) Create(ctx context.Context, row CatalogTagRow) error {
	if r == nil {
		return fmt.Errorf("catalog tag repository is required")
	}

	normalized, err := validateCatalogTagCreateRow(row)
	if err != nil {
		return err
	}

	_, err = r.exec.ExecContext(
		normalizeContext(ctx),
		`INSERT INTO catalog_tags (
			tag_id,
			key,
			name,
			description,
			color,
			active,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
		normalized.TagID,
		normalized.Key,
		normalized.Name,
		normalized.Description,
		normalized.Color,
		boolToSQLiteInteger(normalized.Active),
		formatCatalogTimestamp(normalized.CreatedAt),
		formatCatalogTimestamp(normalized.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create catalog tag row %q: %w", normalized.TagID, err)
	}

	return nil
}

// Update updates mutable fields for one tag row by tag_id.
func (r *CatalogTagRepository) Update(ctx context.Context, row CatalogTagRow) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog tag repository is required")
	}

	normalized, err := validateCatalogTagUpdateRow(row)
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`UPDATE catalog_tags
		SET key = ?,
			name = ?,
			description = ?,
			color = ?,
			active = ?,
			updated_at = ?
		WHERE tag_id = ?;`,
		normalized.Key,
		normalized.Name,
		normalized.Description,
		normalized.Color,
		boolToSQLiteInteger(normalized.Active),
		formatCatalogTimestamp(normalized.UpdatedAt),
		normalized.TagID,
	)
	if err != nil {
		return false, fmt.Errorf("update catalog tag row %q: %w", normalized.TagID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read update affected rows for catalog tag %q: %w", normalized.TagID, err)
	}

	return rowsAffected > 0, nil
}

// GetByTagID fetches one tag row by tag_id.
func (r *CatalogTagRepository) GetByTagID(ctx context.Context, tagID string) (CatalogTagRow, error) {
	if r == nil {
		return CatalogTagRow{}, fmt.Errorf("catalog tag repository is required")
	}

	normalizedTagID, err := normalizeRequiredID(tagID, "catalog tag tag_id")
	if err != nil {
		return CatalogTagRow{}, err
	}

	row, err := scanCatalogTagRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				tag_id,
				key,
				name,
				description,
				color,
				active,
				created_at,
				updated_at
			FROM catalog_tags
			WHERE tag_id = ?;`,
			normalizedTagID,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogTagRow{}, ErrCatalogTagNotFound
		}
		return CatalogTagRow{}, fmt.Errorf("get catalog tag row %q: %w", normalizedTagID, err)
	}

	return row, nil
}

// GetByKey fetches one tag row by key.
func (r *CatalogTagRepository) GetByKey(ctx context.Context, key string) (CatalogTagRow, error) {
	if r == nil {
		return CatalogTagRow{}, fmt.Errorf("catalog tag repository is required")
	}

	normalizedKey, err := normalizeRequiredID(key, "catalog tag key")
	if err != nil {
		return CatalogTagRow{}, err
	}

	row, err := scanCatalogTagRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				tag_id,
				key,
				name,
				description,
				color,
				active,
				created_at,
				updated_at
			FROM catalog_tags
			WHERE key = ?;`,
			normalizedKey,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogTagRow{}, ErrCatalogTagNotFound
		}
		return CatalogTagRow{}, fmt.Errorf("get catalog tag row by key %q: %w", normalizedKey, err)
	}

	return row, nil
}

// List returns tag rows that match the provided filter with deterministic ordering.
func (r *CatalogTagRepository) List(ctx context.Context, filter CatalogTagListFilter) ([]CatalogTagRow, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog tag repository is required")
	}

	query, args, err := buildCatalogTagListQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog tag rows: %w", err)
	}
	defer rows.Close()

	result := make([]CatalogTagRow, 0, 16)
	for rows.Next() {
		tagRow, scanErr := scanCatalogTagRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan catalog tag list row: %w", scanErr)
		}
		result = append(result, tagRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog tag rows: %w", err)
	}

	return result, nil
}

// DeleteByTagID deletes one tag row by tag_id.
func (r *CatalogTagRepository) DeleteByTagID(ctx context.Context, tagID string) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog tag repository is required")
	}

	normalizedTagID, err := normalizeRequiredID(tagID, "catalog tag tag_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_tags WHERE tag_id = ?;`,
		normalizedTagID,
	)
	if err != nil {
		return false, fmt.Errorf("delete catalog tag row %q: %w", normalizedTagID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read delete affected rows for catalog tag %q: %w", normalizedTagID, err)
	}

	return rowsAffected > 0, nil
}

func buildCatalogTagListQuery(filter CatalogTagListFilter) (string, []any, error) {
	if strings.TrimSpace(filter.TagID) != "" && len(filter.TagIDs) > 0 {
		return "", nil, fmt.Errorf("catalog tag filter cannot include both tag_id and tag_ids")
	}
	if strings.TrimSpace(filter.Key) != "" && len(filter.Keys) > 0 {
		return "", nil, fmt.Errorf("catalog tag filter cannot include both key and keys")
	}

	conditions := make([]string, 0, 5)
	args := make([]any, 0, 8)

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

	if strings.TrimSpace(filter.Key) != "" {
		conditions = append(conditions, "key = ?")
		args = append(args, strings.TrimSpace(filter.Key))
	}

	normalizedKeys := normalizeOptionalIDList(filter.Keys)
	if len(normalizedKeys) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedKeys)), ",")
		conditions = append(conditions, "key IN ("+inClause+")")
		for _, key := range normalizedKeys {
			args = append(args, key)
		}
	}

	if filter.Active != nil {
		conditions = append(conditions, "active = ?")
		args = append(args, boolToSQLiteInteger(*filter.Active))
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT
		tag_id,
		key,
		name,
		description,
		color,
		active,
		created_at,
		updated_at
	FROM catalog_tags`)
	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY key ASC, tag_id ASC;")

	return queryBuilder.String(), args, nil
}
