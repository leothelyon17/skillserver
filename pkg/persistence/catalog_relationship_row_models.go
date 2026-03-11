package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	// ErrCatalogSkillPromptRelationshipNotFound indicates that a skill->prompt relationship row does not exist.
	ErrCatalogSkillPromptRelationshipNotFound = errors.New("catalog skill prompt relationship row not found")
)

// CatalogSkillRuleRelationshipRow mirrors one row in catalog_skill_rule_relationships.
type CatalogSkillRuleRelationshipRow struct {
	SkillItemID string
	RuleItemID  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	UpdatedBy   *string
}

// CatalogSkillRuleRelationshipListFilter constrains skill-rule relationship list queries.
type CatalogSkillRuleRelationshipListFilter struct {
	SkillItemID  string
	SkillItemIDs []string
	RuleItemID   string
	RuleItemIDs  []string
}

// CatalogSkillPromptRelationshipRow mirrors one row in catalog_skill_prompt_relationships.
type CatalogSkillPromptRelationshipRow struct {
	SkillItemID  string
	PromptItemID string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	UpdatedBy    *string
}

// CatalogSkillPromptRelationshipListFilter constrains skill-prompt relationship list queries.
type CatalogSkillPromptRelationshipListFilter struct {
	SkillItemID   string
	SkillItemIDs  []string
	PromptItemID  string
	PromptItemIDs []string
}

func validateCatalogSkillRuleRelationshipUpsertRow(
	row CatalogSkillRuleRelationshipRow,
) (CatalogSkillRuleRelationshipRow, error) {
	skillItemID, err := normalizeRequiredID(row.SkillItemID, "catalog skill rule relationship skill_item_id")
	if err != nil {
		return CatalogSkillRuleRelationshipRow{}, err
	}
	row.SkillItemID = skillItemID

	ruleItemID, err := normalizeRequiredID(row.RuleItemID, "catalog skill rule relationship rule_item_id")
	if err != nil {
		return CatalogSkillRuleRelationshipRow{}, err
	}
	row.RuleItemID = ruleItemID

	row.UpdatedBy = normalizeOptionalUpdatedBy(row.UpdatedBy)

	now := time.Now().UTC()
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	} else {
		row.CreatedAt = row.CreatedAt.UTC()
	}

	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = row.CreatedAt
	} else {
		row.UpdatedAt = row.UpdatedAt.UTC()
	}

	return row, nil
}

func validateCatalogSkillPromptRelationshipUpsertRow(
	row CatalogSkillPromptRelationshipRow,
) (CatalogSkillPromptRelationshipRow, error) {
	skillItemID, err := normalizeRequiredID(row.SkillItemID, "catalog skill prompt relationship skill_item_id")
	if err != nil {
		return CatalogSkillPromptRelationshipRow{}, err
	}
	row.SkillItemID = skillItemID

	promptItemID, err := normalizeRequiredID(row.PromptItemID, "catalog skill prompt relationship prompt_item_id")
	if err != nil {
		return CatalogSkillPromptRelationshipRow{}, err
	}
	row.PromptItemID = promptItemID

	row.UpdatedBy = normalizeOptionalUpdatedBy(row.UpdatedBy)

	now := time.Now().UTC()
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	} else {
		row.CreatedAt = row.CreatedAt.UTC()
	}

	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = row.CreatedAt
	} else {
		row.UpdatedAt = row.UpdatedAt.UTC()
	}

	return row, nil
}

func scanCatalogSkillRuleRelationshipRow(scanner rowScanner) (CatalogSkillRuleRelationshipRow, error) {
	var (
		skillItemID  string
		ruleItemID   string
		createdAtRaw string
		updatedAtRaw string
		updatedByRaw sql.NullString
	)

	if err := scanner.Scan(
		&skillItemID,
		&ruleItemID,
		&createdAtRaw,
		&updatedAtRaw,
		&updatedByRaw,
	); err != nil {
		return CatalogSkillRuleRelationshipRow{}, err
	}

	createdAt, err := parseCatalogTimestamp(createdAtRaw)
	if err != nil {
		return CatalogSkillRuleRelationshipRow{}, fmt.Errorf("parse catalog skill rule relationship created_at: %w", err)
	}

	updatedAt, err := parseCatalogTimestamp(updatedAtRaw)
	if err != nil {
		return CatalogSkillRuleRelationshipRow{}, fmt.Errorf("parse catalog skill rule relationship updated_at: %w", err)
	}

	return CatalogSkillRuleRelationshipRow{
		SkillItemID: skillItemID,
		RuleItemID:  ruleItemID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		UpdatedBy:   nullStringToPointer(updatedByRaw),
	}, nil
}

func scanCatalogSkillPromptRelationshipRow(scanner rowScanner) (CatalogSkillPromptRelationshipRow, error) {
	var (
		skillItemID  string
		promptItemID string
		createdAtRaw string
		updatedAtRaw string
		updatedByRaw sql.NullString
	)

	if err := scanner.Scan(
		&skillItemID,
		&promptItemID,
		&createdAtRaw,
		&updatedAtRaw,
		&updatedByRaw,
	); err != nil {
		return CatalogSkillPromptRelationshipRow{}, err
	}

	createdAt, err := parseCatalogTimestamp(createdAtRaw)
	if err != nil {
		return CatalogSkillPromptRelationshipRow{}, fmt.Errorf("parse catalog skill prompt relationship created_at: %w", err)
	}

	updatedAt, err := parseCatalogTimestamp(updatedAtRaw)
	if err != nil {
		return CatalogSkillPromptRelationshipRow{}, fmt.Errorf("parse catalog skill prompt relationship updated_at: %w", err)
	}

	return CatalogSkillPromptRelationshipRow{
		SkillItemID:  skillItemID,
		PromptItemID: promptItemID,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		UpdatedBy:    nullStringToPointer(updatedByRaw),
	}, nil
}

func normalizeRequiredUniqueIDList(values []string, fieldName string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, fmt.Errorf("%s[%d] is required", fieldName, index)
		}
		if _, exists := seen[trimmed]; exists {
			return nil, fmt.Errorf("%s contains duplicate value %q", fieldName, trimmed)
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	sort.Strings(normalized)
	return normalized, nil
}

func normalizeOptionalUpdatedBy(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
