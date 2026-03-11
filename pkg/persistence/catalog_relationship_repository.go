package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CatalogSkillRuleRelationshipRepository provides access to catalog_skill_rule_relationships rows.
type CatalogSkillRuleRelationshipRepository struct {
	exec catalogQueryExecutor
}

// NewCatalogSkillRuleRelationshipRepository creates a skill-rule relationship repository around a DB or transaction handle.
func NewCatalogSkillRuleRelationshipRepository(
	exec catalogQueryExecutor,
) (*CatalogSkillRuleRelationshipRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("catalog skill rule relationship repository query executor is required")
	}

	return &CatalogSkillRuleRelationshipRepository{exec: exec}, nil
}

// ReplaceForSkillItemID replaces one skill's complete rule set in a single transaction.
func (r *CatalogSkillRuleRelationshipRepository) ReplaceForSkillItemID(
	ctx context.Context,
	skillItemID string,
	ruleItemIDs []string,
	updatedAt time.Time,
	updatedBy *string,
) error {
	if r == nil {
		return fmt.Errorf("catalog skill rule relationship repository is required")
	}

	normalizedSkillItemID, err := normalizeRequiredID(skillItemID, "catalog skill rule relationship skill_item_id")
	if err != nil {
		return err
	}

	normalizedRuleItemIDs, err := normalizeRequiredUniqueIDList(
		ruleItemIDs,
		"catalog skill rule relationship rule_item_ids",
	)
	if err != nil {
		return err
	}

	normalizedUpdatedAt := updatedAt
	if normalizedUpdatedAt.IsZero() {
		normalizedUpdatedAt = time.Now().UTC()
	} else {
		normalizedUpdatedAt = normalizedUpdatedAt.UTC()
	}
	normalizedUpdatedBy := normalizeOptionalUpdatedBy(updatedBy)
	normalizedCtx := normalizeContext(ctx)

	err = withCatalogWriteTransaction(normalizedCtx, r.exec, func(tx catalogQueryExecutor) error {
		if len(normalizedRuleItemIDs) == 0 {
			if _, deleteErr := tx.ExecContext(
				normalizedCtx,
				`DELETE FROM catalog_skill_rule_relationships WHERE skill_item_id = ?;`,
				normalizedSkillItemID,
			); deleteErr != nil {
				return fmt.Errorf(
					"delete catalog skill rule relationship rows for skill %q: %w",
					normalizedSkillItemID,
					deleteErr,
				)
			}
			return nil
		}

		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedRuleItemIDs)), ",")
		deleteArgs := make([]any, 0, len(normalizedRuleItemIDs)+1)
		deleteArgs = append(deleteArgs, normalizedSkillItemID)
		for _, ruleItemID := range normalizedRuleItemIDs {
			deleteArgs = append(deleteArgs, ruleItemID)
		}

		if _, deleteErr := tx.ExecContext(
			normalizedCtx,
			`DELETE FROM catalog_skill_rule_relationships
			WHERE skill_item_id = ?
			AND rule_item_id NOT IN (`+inClause+`);`,
			deleteArgs...,
		); deleteErr != nil {
			return fmt.Errorf(
				"delete stale catalog skill rule relationship rows for skill %q: %w",
				normalizedSkillItemID,
				deleteErr,
			)
		}

		for _, ruleItemID := range normalizedRuleItemIDs {
			if _, insertErr := tx.ExecContext(
				normalizedCtx,
				`INSERT INTO catalog_skill_rule_relationships (
					skill_item_id,
					rule_item_id,
					created_at,
					updated_at,
					updated_by
				) VALUES (?, ?, ?, ?, ?)
				ON CONFLICT(skill_item_id, rule_item_id) DO UPDATE SET
					updated_at = excluded.updated_at,
					updated_by = excluded.updated_by;`,
				normalizedSkillItemID,
				ruleItemID,
				formatCatalogTimestamp(normalizedUpdatedAt),
				formatCatalogTimestamp(normalizedUpdatedAt),
				pointerToAny(normalizedUpdatedBy),
			); insertErr != nil {
				return fmt.Errorf(
					"insert catalog skill rule relationship row %q/%q: %w",
					normalizedSkillItemID,
					ruleItemID,
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

// List returns skill-rule relationship rows that match the provided filter with deterministic ordering.
func (r *CatalogSkillRuleRelationshipRepository) List(
	ctx context.Context,
	filter CatalogSkillRuleRelationshipListFilter,
) ([]CatalogSkillRuleRelationshipRow, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog skill rule relationship repository is required")
	}

	query, args, err := buildCatalogSkillRuleRelationshipListQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog skill rule relationship rows: %w", err)
	}
	defer rows.Close()

	result := make([]CatalogSkillRuleRelationshipRow, 0, 32)
	for rows.Next() {
		relationshipRow, scanErr := scanCatalogSkillRuleRelationshipRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan catalog skill rule relationship list row: %w", scanErr)
		}
		result = append(result, relationshipRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog skill rule relationship rows: %w", err)
	}

	return result, nil
}

// ListBySkillItemID returns skill-rule relationship rows for one skill with deterministic ordering.
func (r *CatalogSkillRuleRelationshipRepository) ListBySkillItemID(
	ctx context.Context,
	skillItemID string,
) ([]CatalogSkillRuleRelationshipRow, error) {
	return r.List(ctx, CatalogSkillRuleRelationshipListFilter{SkillItemID: skillItemID})
}

// ListByRuleItemID returns skill-rule relationship rows for one rule with deterministic ordering.
func (r *CatalogSkillRuleRelationshipRepository) ListByRuleItemID(
	ctx context.Context,
	ruleItemID string,
) ([]CatalogSkillRuleRelationshipRow, error) {
	return r.List(ctx, CatalogSkillRuleRelationshipListFilter{RuleItemID: ruleItemID})
}

// DeleteBySkillItemID deletes all skill-rule relationship rows for one skill endpoint.
func (r *CatalogSkillRuleRelationshipRepository) DeleteBySkillItemID(
	ctx context.Context,
	skillItemID string,
) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog skill rule relationship repository is required")
	}

	normalizedSkillItemID, err := normalizeRequiredID(skillItemID, "catalog skill rule relationship skill_item_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_skill_rule_relationships WHERE skill_item_id = ?;`,
		normalizedSkillItemID,
	)
	if err != nil {
		return false, fmt.Errorf(
			"delete catalog skill rule relationship rows for skill %q: %w",
			normalizedSkillItemID,
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"read delete affected rows for catalog skill rule relationships skill %q: %w",
			normalizedSkillItemID,
			err,
		)
	}

	return rowsAffected > 0, nil
}

// DeleteByRuleItemID deletes all skill-rule relationship rows for one rule endpoint.
func (r *CatalogSkillRuleRelationshipRepository) DeleteByRuleItemID(
	ctx context.Context,
	ruleItemID string,
) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog skill rule relationship repository is required")
	}

	normalizedRuleItemID, err := normalizeRequiredID(ruleItemID, "catalog skill rule relationship rule_item_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_skill_rule_relationships WHERE rule_item_id = ?;`,
		normalizedRuleItemID,
	)
	if err != nil {
		return false, fmt.Errorf(
			"delete catalog skill rule relationship rows for rule %q: %w",
			normalizedRuleItemID,
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"read delete affected rows for catalog skill rule relationships rule %q: %w",
			normalizedRuleItemID,
			err,
		)
	}

	return rowsAffected > 0, nil
}

func buildCatalogSkillRuleRelationshipListQuery(
	filter CatalogSkillRuleRelationshipListFilter,
) (string, []any, error) {
	if strings.TrimSpace(filter.SkillItemID) != "" && len(filter.SkillItemIDs) > 0 {
		return "", nil, fmt.Errorf("catalog skill rule relationship filter cannot include both skill_item_id and skill_item_ids")
	}
	if strings.TrimSpace(filter.RuleItemID) != "" && len(filter.RuleItemIDs) > 0 {
		return "", nil, fmt.Errorf("catalog skill rule relationship filter cannot include both rule_item_id and rule_item_ids")
	}

	conditions := make([]string, 0, 4)
	args := make([]any, 0, 10)

	if strings.TrimSpace(filter.SkillItemID) != "" {
		conditions = append(conditions, "skill_item_id = ?")
		args = append(args, strings.TrimSpace(filter.SkillItemID))
	}

	normalizedSkillItemIDs := normalizeOptionalIDList(filter.SkillItemIDs)
	if len(normalizedSkillItemIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedSkillItemIDs)), ",")
		conditions = append(conditions, "skill_item_id IN ("+inClause+")")
		for _, skillItemID := range normalizedSkillItemIDs {
			args = append(args, skillItemID)
		}
	}

	if strings.TrimSpace(filter.RuleItemID) != "" {
		conditions = append(conditions, "rule_item_id = ?")
		args = append(args, strings.TrimSpace(filter.RuleItemID))
	}

	normalizedRuleItemIDs := normalizeOptionalIDList(filter.RuleItemIDs)
	if len(normalizedRuleItemIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedRuleItemIDs)), ",")
		conditions = append(conditions, "rule_item_id IN ("+inClause+")")
		for _, ruleItemID := range normalizedRuleItemIDs {
			args = append(args, ruleItemID)
		}
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT
		skill_item_id,
		rule_item_id,
		created_at,
		updated_at,
		updated_by
	FROM catalog_skill_rule_relationships`)
	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY skill_item_id ASC, rule_item_id ASC;")

	return queryBuilder.String(), args, nil
}

// CatalogSkillPromptRelationshipRepository provides access to catalog_skill_prompt_relationships rows.
type CatalogSkillPromptRelationshipRepository struct {
	exec catalogQueryExecutor
}

// NewCatalogSkillPromptRelationshipRepository creates a skill-prompt relationship repository around a DB or transaction handle.
func NewCatalogSkillPromptRelationshipRepository(
	exec catalogQueryExecutor,
) (*CatalogSkillPromptRelationshipRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("catalog skill prompt relationship repository query executor is required")
	}

	return &CatalogSkillPromptRelationshipRepository{exec: exec}, nil
}

// Upsert inserts or updates one skill-prompt relationship row keyed by skill_item_id.
func (r *CatalogSkillPromptRelationshipRepository) Upsert(
	ctx context.Context,
	row CatalogSkillPromptRelationshipRow,
) error {
	if r == nil {
		return fmt.Errorf("catalog skill prompt relationship repository is required")
	}

	normalized, err := validateCatalogSkillPromptRelationshipUpsertRow(row)
	if err != nil {
		return err
	}

	_, err = r.exec.ExecContext(
		normalizeContext(ctx),
		`INSERT INTO catalog_skill_prompt_relationships (
			skill_item_id,
			prompt_item_id,
			created_at,
			updated_at,
			updated_by
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(skill_item_id) DO UPDATE SET
			prompt_item_id = excluded.prompt_item_id,
			updated_at = excluded.updated_at,
			updated_by = excluded.updated_by;`,
		normalized.SkillItemID,
		normalized.PromptItemID,
		formatCatalogTimestamp(normalized.CreatedAt),
		formatCatalogTimestamp(normalized.UpdatedAt),
		pointerToAny(normalized.UpdatedBy),
	)
	if err != nil {
		return fmt.Errorf(
			"upsert catalog skill prompt relationship row skill=%q prompt=%q: %w",
			normalized.SkillItemID,
			normalized.PromptItemID,
			err,
		)
	}

	return nil
}

// SetForSkillItemID sets or replaces one skill's prompt relationship.
func (r *CatalogSkillPromptRelationshipRepository) SetForSkillItemID(
	ctx context.Context,
	skillItemID string,
	promptItemID string,
	updatedAt time.Time,
	updatedBy *string,
) error {
	return r.Upsert(ctx, CatalogSkillPromptRelationshipRow{
		SkillItemID:  skillItemID,
		PromptItemID: promptItemID,
		CreatedAt:    updatedAt,
		UpdatedAt:    updatedAt,
		UpdatedBy:    updatedBy,
	})
}

// GetBySkillItemID fetches one skill-prompt relationship row by skill_item_id.
func (r *CatalogSkillPromptRelationshipRepository) GetBySkillItemID(
	ctx context.Context,
	skillItemID string,
) (CatalogSkillPromptRelationshipRow, error) {
	if r == nil {
		return CatalogSkillPromptRelationshipRow{}, fmt.Errorf("catalog skill prompt relationship repository is required")
	}

	normalizedSkillItemID, err := normalizeRequiredID(skillItemID, "catalog skill prompt relationship skill_item_id")
	if err != nil {
		return CatalogSkillPromptRelationshipRow{}, err
	}

	row, err := scanCatalogSkillPromptRelationshipRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				skill_item_id,
				prompt_item_id,
				created_at,
				updated_at,
				updated_by
			FROM catalog_skill_prompt_relationships
			WHERE skill_item_id = ?;`,
			normalizedSkillItemID,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CatalogSkillPromptRelationshipRow{}, ErrCatalogSkillPromptRelationshipNotFound
		}
		return CatalogSkillPromptRelationshipRow{}, fmt.Errorf(
			"get catalog skill prompt relationship row for skill %q: %w",
			normalizedSkillItemID,
			err,
		)
	}

	return row, nil
}

// List returns skill-prompt relationship rows that match the provided filter with deterministic ordering.
func (r *CatalogSkillPromptRelationshipRepository) List(
	ctx context.Context,
	filter CatalogSkillPromptRelationshipListFilter,
) ([]CatalogSkillPromptRelationshipRow, error) {
	if r == nil {
		return nil, fmt.Errorf("catalog skill prompt relationship repository is required")
	}

	query, args, err := buildCatalogSkillPromptRelationshipListQuery(filter)
	if err != nil {
		return nil, err
	}

	rows, err := r.exec.QueryContext(normalizeContext(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list catalog skill prompt relationship rows: %w", err)
	}
	defer rows.Close()

	result := make([]CatalogSkillPromptRelationshipRow, 0, 32)
	for rows.Next() {
		relationshipRow, scanErr := scanCatalogSkillPromptRelationshipRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan catalog skill prompt relationship list row: %w", scanErr)
		}
		result = append(result, relationshipRow)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog skill prompt relationship rows: %w", err)
	}

	return result, nil
}

// ListBySkillItemID returns skill-prompt relationship rows for one skill with deterministic ordering.
func (r *CatalogSkillPromptRelationshipRepository) ListBySkillItemID(
	ctx context.Context,
	skillItemID string,
) ([]CatalogSkillPromptRelationshipRow, error) {
	return r.List(ctx, CatalogSkillPromptRelationshipListFilter{SkillItemID: skillItemID})
}

// ListByPromptItemID returns skill-prompt relationship rows for one prompt with deterministic ordering.
func (r *CatalogSkillPromptRelationshipRepository) ListByPromptItemID(
	ctx context.Context,
	promptItemID string,
) ([]CatalogSkillPromptRelationshipRow, error) {
	return r.List(ctx, CatalogSkillPromptRelationshipListFilter{PromptItemID: promptItemID})
}

// ClearBySkillItemID clears one skill's prompt relationship row.
func (r *CatalogSkillPromptRelationshipRepository) ClearBySkillItemID(
	ctx context.Context,
	skillItemID string,
) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog skill prompt relationship repository is required")
	}

	normalizedSkillItemID, err := normalizeRequiredID(skillItemID, "catalog skill prompt relationship skill_item_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_skill_prompt_relationships WHERE skill_item_id = ?;`,
		normalizedSkillItemID,
	)
	if err != nil {
		return false, fmt.Errorf(
			"delete catalog skill prompt relationship row for skill %q: %w",
			normalizedSkillItemID,
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"read delete affected rows for catalog skill prompt relationship skill %q: %w",
			normalizedSkillItemID,
			err,
		)
	}

	return rowsAffected > 0, nil
}

// DeleteBySkillItemID deletes one skill->prompt relationship row by skill endpoint item ID.
func (r *CatalogSkillPromptRelationshipRepository) DeleteBySkillItemID(
	ctx context.Context,
	skillItemID string,
) (bool, error) {
	return r.ClearBySkillItemID(ctx, skillItemID)
}

// DeleteByPromptItemID deletes all skill->prompt relationship rows for one prompt endpoint.
func (r *CatalogSkillPromptRelationshipRepository) DeleteByPromptItemID(
	ctx context.Context,
	promptItemID string,
) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("catalog skill prompt relationship repository is required")
	}

	normalizedPromptItemID, err := normalizeRequiredID(promptItemID, "catalog skill prompt relationship prompt_item_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM catalog_skill_prompt_relationships WHERE prompt_item_id = ?;`,
		normalizedPromptItemID,
	)
	if err != nil {
		return false, fmt.Errorf(
			"delete catalog skill prompt relationship rows for prompt %q: %w",
			normalizedPromptItemID,
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf(
			"read delete affected rows for catalog skill prompt relationships prompt %q: %w",
			normalizedPromptItemID,
			err,
		)
	}

	return rowsAffected > 0, nil
}

func buildCatalogSkillPromptRelationshipListQuery(
	filter CatalogSkillPromptRelationshipListFilter,
) (string, []any, error) {
	if strings.TrimSpace(filter.SkillItemID) != "" && len(filter.SkillItemIDs) > 0 {
		return "", nil, fmt.Errorf("catalog skill prompt relationship filter cannot include both skill_item_id and skill_item_ids")
	}
	if strings.TrimSpace(filter.PromptItemID) != "" && len(filter.PromptItemIDs) > 0 {
		return "", nil, fmt.Errorf("catalog skill prompt relationship filter cannot include both prompt_item_id and prompt_item_ids")
	}

	conditions := make([]string, 0, 4)
	args := make([]any, 0, 10)

	if strings.TrimSpace(filter.SkillItemID) != "" {
		conditions = append(conditions, "skill_item_id = ?")
		args = append(args, strings.TrimSpace(filter.SkillItemID))
	}

	normalizedSkillItemIDs := normalizeOptionalIDList(filter.SkillItemIDs)
	if len(normalizedSkillItemIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedSkillItemIDs)), ",")
		conditions = append(conditions, "skill_item_id IN ("+inClause+")")
		for _, skillItemID := range normalizedSkillItemIDs {
			args = append(args, skillItemID)
		}
	}

	if strings.TrimSpace(filter.PromptItemID) != "" {
		conditions = append(conditions, "prompt_item_id = ?")
		args = append(args, strings.TrimSpace(filter.PromptItemID))
	}

	normalizedPromptItemIDs := normalizeOptionalIDList(filter.PromptItemIDs)
	if len(normalizedPromptItemIDs) > 0 {
		inClause := strings.TrimRight(strings.Repeat("?,", len(normalizedPromptItemIDs)), ",")
		conditions = append(conditions, "prompt_item_id IN ("+inClause+")")
		for _, promptItemID := range normalizedPromptItemIDs {
			args = append(args, promptItemID)
		}
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT
		skill_item_id,
		prompt_item_id,
		created_at,
		updated_at,
		updated_by
	FROM catalog_skill_prompt_relationships`)
	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY skill_item_id ASC, prompt_item_id ASC;")

	return queryBuilder.String(), args, nil
}
