package domain

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

var (
	// ErrCatalogRelationshipItemNotFound indicates that an item cannot be resolved for relationship operations.
	ErrCatalogRelationshipItemNotFound = errors.New("catalog relationship item not found")
	// ErrCatalogRelationshipReadOnlySurface indicates relationship writes were attempted on non-skill items.
	ErrCatalogRelationshipReadOnlySurface = errors.New("catalog relationship read-only surface")
	// ErrCatalogRelationshipValidation indicates invalid relationship input payloads.
	ErrCatalogRelationshipValidation = errors.New("catalog relationship validation failed")
)

type catalogRelationshipSourceRepository interface {
	GetByItemID(ctx context.Context, itemID string) (persistence.CatalogSourceRow, error)
	List(ctx context.Context, filter persistence.CatalogSourceListFilter) ([]persistence.CatalogSourceRow, error)
}

type catalogRelationshipRuleRepository interface {
	ReplaceForSkillItemID(
		ctx context.Context,
		skillItemID string,
		ruleItemIDs []string,
		updatedAt time.Time,
		updatedBy *string,
	) error
	ListBySkillItemID(ctx context.Context, skillItemID string) ([]persistence.CatalogSkillRuleRelationshipRow, error)
	ListByRuleItemID(ctx context.Context, ruleItemID string) ([]persistence.CatalogSkillRuleRelationshipRow, error)
	List(
		ctx context.Context,
		filter persistence.CatalogSkillRuleRelationshipListFilter,
	) ([]persistence.CatalogSkillRuleRelationshipRow, error)
	DeleteBySkillItemID(ctx context.Context, skillItemID string) (bool, error)
}

type catalogRelationshipPromptRepository interface {
	SetForSkillItemID(
		ctx context.Context,
		skillItemID string,
		promptItemID string,
		updatedAt time.Time,
		updatedBy *string,
	) error
	GetBySkillItemID(ctx context.Context, skillItemID string) (persistence.CatalogSkillPromptRelationshipRow, error)
	ListByPromptItemID(ctx context.Context, promptItemID string) ([]persistence.CatalogSkillPromptRelationshipRow, error)
	List(
		ctx context.Context,
		filter persistence.CatalogSkillPromptRelationshipListFilter,
	) ([]persistence.CatalogSkillPromptRelationshipRow, error)
	ClearBySkillItemID(ctx context.Context, skillItemID string) (bool, error)
	DeleteBySkillItemID(ctx context.Context, skillItemID string) (bool, error)
}

// CatalogRelationshipServiceOptions configures relationship service behavior.
type CatalogRelationshipServiceOptions struct {
	Now func() time.Time
}

// CatalogRelationshipValidationError captures one field-level relationship validation failure.
type CatalogRelationshipValidationError struct {
	Field  string
	Detail string
	Cause  error
}

func (e *CatalogRelationshipValidationError) Error() string {
	if e == nil {
		return ErrCatalogRelationshipValidation.Error()
	}

	base := strings.TrimSpace(ErrCatalogRelationshipValidation.Error())
	field := strings.TrimSpace(e.Field)
	detail := strings.TrimSpace(e.Detail)

	switch {
	case field == "" && detail == "":
		return base
	case field == "":
		return base + ": " + detail
	case detail == "":
		return base + ": " + field
	default:
		return base + ": " + field + " " + detail
	}
}

// Unwrap supports errors.Is(err, ErrCatalogRelationshipValidation).
func (e *CatalogRelationshipValidationError) Unwrap() error {
	if e == nil || e.Cause == nil {
		return ErrCatalogRelationshipValidation
	}
	return errors.Join(ErrCatalogRelationshipValidation, e.Cause)
}

// CatalogRelationshipReadOnlySurfaceError captures non-skill write attempts.
type CatalogRelationshipReadOnlySurfaceError struct {
	ItemID     string
	Classifier CatalogClassifier
}

func (e *CatalogRelationshipReadOnlySurfaceError) Error() string {
	if e == nil {
		return ErrCatalogRelationshipReadOnlySurface.Error()
	}
	if strings.TrimSpace(e.ItemID) == "" {
		return ErrCatalogRelationshipReadOnlySurface.Error()
	}
	return fmt.Sprintf(
		"%s: item_id=%q classifier=%q",
		ErrCatalogRelationshipReadOnlySurface.Error(),
		e.ItemID,
		e.Classifier,
	)
}

// Unwrap supports errors.Is(err, ErrCatalogRelationshipReadOnlySurface).
func (e *CatalogRelationshipReadOnlySurfaceError) Unwrap() error {
	return ErrCatalogRelationshipReadOnlySurface
}

// CatalogRelationshipItem is one normalized related catalog item descriptor.
type CatalogRelationshipItem struct {
	ID            string            `json:"id"`
	Classifier    CatalogClassifier `json:"classifier"`
	Name          string            `json:"name"`
	ParentSkillID *string           `json:"parent_skill_id,omitempty"`
	ResourcePath  *string           `json:"resource_path,omitempty"`
}

// CatalogRelationshipSet captures one normalized prompt/rules/skills relationship envelope.
type CatalogRelationshipSet struct {
	Prompt *CatalogRelationshipItem  `json:"prompt"`
	Rules  []CatalogRelationshipItem `json:"rules"`
	Skills []CatalogRelationshipItem `json:"skills"`
}

// CatalogRelationshipView is the top-level read payload for one catalog item relationship projection.
type CatalogRelationshipView struct {
	ItemID        string                 `json:"item_id"`
	Relationships CatalogRelationshipSet `json:"relationships"`
}

// CatalogRelationshipPatchInput describes one partial relationship mutation for a skill item.
type CatalogRelationshipPatchInput struct {
	ItemID          string
	PromptItemID    *string
	PromptItemIDSet bool
	RuleItemIDs     *[]string
	UpdatedBy       *string
	UpdatedAt       *time.Time
}

// CatalogRelationshipReconcileReport summarizes stale-row cleanup outcomes.
type CatalogRelationshipReconcileReport struct {
	SkillRuleRowsScanned   int `json:"skill_rule_rows_scanned"`
	SkillRuleRowsPruned    int `json:"skill_rule_rows_pruned"`
	SkillPromptRowsScanned int `json:"skill_prompt_rows_scanned"`
	SkillPromptRowsPruned  int `json:"skill_prompt_rows_pruned"`
}

// CatalogRelationshipService provides authoritative relationship validation, projection, and reconciliation flows.
type CatalogRelationshipService struct {
	sourceRepo catalogRelationshipSourceRepository
	ruleRepo   catalogRelationshipRuleRepository
	promptRepo catalogRelationshipPromptRepository
	now        func() time.Time
}

// NewCatalogRelationshipService creates a relationship service.
func NewCatalogRelationshipService(
	sourceRepo catalogRelationshipSourceRepository,
	ruleRepo catalogRelationshipRuleRepository,
	promptRepo catalogRelationshipPromptRepository,
	options CatalogRelationshipServiceOptions,
) (*CatalogRelationshipService, error) {
	if sourceRepo == nil {
		return nil, fmt.Errorf("catalog relationship source repository is required")
	}
	if ruleRepo == nil {
		return nil, fmt.Errorf("catalog relationship rule repository is required")
	}
	if promptRepo == nil {
		return nil, fmt.Errorf("catalog relationship prompt repository is required")
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &CatalogRelationshipService{
		sourceRepo: sourceRepo,
		ruleRepo:   ruleRepo,
		promptRepo: promptRepo,
		now:        now,
	}, nil
}

// Get resolves one item's effective relationship view.
func (s *CatalogRelationshipService) Get(ctx context.Context, itemID string) (CatalogRelationshipView, error) {
	if s == nil {
		return CatalogRelationshipView{}, fmt.Errorf("catalog relationship service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	itemRef, err := normalizeCatalogRelationshipItemReference(itemID, "item_id")
	if err != nil {
		return CatalogRelationshipView{}, err
	}

	sourceRow, err := s.getActiveSourceRow(ctx, itemRef.ItemID)
	if err != nil {
		return CatalogRelationshipView{}, err
	}
	classifier, err := mapCatalogRelationshipDomainClassifier(sourceRow.Classifier)
	if err != nil {
		return CatalogRelationshipView{}, fmt.Errorf(
			"map catalog relationship classifier for %q: %w",
			sourceRow.ItemID,
			err,
		)
	}

	relationships := newCatalogRelationshipSet()
	switch classifier {
	case CatalogClassifierSkill:
		resolved, resolveErr := s.resolveSkillRelationships(ctx, sourceRow.ItemID)
		if resolveErr != nil {
			return CatalogRelationshipView{}, resolveErr
		}
		relationships = resolved
	case CatalogClassifierPrompt:
		resolvedSkills, resolveErr := s.resolveSkillsForPrompt(ctx, sourceRow.ItemID)
		if resolveErr != nil {
			return CatalogRelationshipView{}, resolveErr
		}
		relationships.Skills = resolvedSkills
	case CatalogClassifierRule:
		resolvedSkills, resolveErr := s.resolveSkillsForRule(ctx, sourceRow.ItemID)
		if resolveErr != nil {
			return CatalogRelationshipView{}, resolveErr
		}
		relationships.Skills = resolvedSkills
	default:
		return CatalogRelationshipView{}, &CatalogRelationshipValidationError{
			Field:  "item_id",
			Detail: fmt.Sprintf("classifier %q is unsupported", classifier),
		}
	}

	return CatalogRelationshipView{
		ItemID:        sourceRow.ItemID,
		Relationships: relationships,
	}, nil
}

// Patch applies one skill-owned relationship update and returns the updated effective view.
func (s *CatalogRelationshipService) Patch(
	ctx context.Context,
	input CatalogRelationshipPatchInput,
) (CatalogRelationshipView, error) {
	if s == nil {
		return CatalogRelationshipView{}, fmt.Errorf("catalog relationship service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	itemRef, err := normalizeCatalogRelationshipItemReference(input.ItemID, "item_id")
	if err != nil {
		return CatalogRelationshipView{}, err
	}

	sourceRow, err := s.getActiveSourceRow(ctx, itemRef.ItemID)
	if err != nil {
		return CatalogRelationshipView{}, err
	}
	classifier, err := mapCatalogRelationshipDomainClassifier(sourceRow.Classifier)
	if err != nil {
		return CatalogRelationshipView{}, fmt.Errorf(
			"map relationship patch source classifier for %q: %w",
			sourceRow.ItemID,
			err,
		)
	}
	if classifier != CatalogClassifierSkill {
		return CatalogRelationshipView{}, &CatalogRelationshipReadOnlySurfaceError{
			ItemID:     sourceRow.ItemID,
			Classifier: classifier,
		}
	}

	updatedAt := s.now().UTC()
	if input.UpdatedAt != nil {
		if !input.UpdatedAt.IsZero() {
			updatedAt = input.UpdatedAt.UTC()
		}
	}
	updatedBy := normalizeCatalogRelationshipOptionalText(input.UpdatedBy)

	if input.PromptItemIDSet {
		if input.PromptItemID == nil {
			if _, clearErr := s.promptRepo.ClearBySkillItemID(ctx, sourceRow.ItemID); clearErr != nil {
				return CatalogRelationshipView{}, fmt.Errorf(
					"clear catalog prompt relationship for %q: %w",
					sourceRow.ItemID,
					clearErr,
				)
			}
		} else {
			promptRef, normalizeErr := normalizeCatalogRelationshipItemReference(
				*input.PromptItemID,
				"prompt_item_id",
			)
			if normalizeErr != nil {
				return CatalogRelationshipView{}, normalizeErr
			}
			if promptRef.Classifier != CatalogClassifierPrompt {
				return CatalogRelationshipView{}, &CatalogRelationshipValidationError{
					Field:  "prompt_item_id",
					Detail: "must reference a prompt item",
				}
			}
			if ensureErr := s.ensureActiveItemHasClassifier(
				ctx,
				promptRef.ItemID,
				CatalogClassifierPrompt,
				"prompt_item_id",
			); ensureErr != nil {
				return CatalogRelationshipView{}, ensureErr
			}

			if setErr := s.promptRepo.SetForSkillItemID(
				ctx,
				sourceRow.ItemID,
				promptRef.ItemID,
				updatedAt,
				updatedBy,
			); setErr != nil {
				return CatalogRelationshipView{}, fmt.Errorf(
					"set catalog prompt relationship for %q to %q: %w",
					sourceRow.ItemID,
					promptRef.ItemID,
					setErr,
				)
			}
		}
	}

	if input.RuleItemIDs != nil {
		ruleItemIDs, normalizeErr := normalizeCatalogRelationshipRuleItemIDs(*input.RuleItemIDs)
		if normalizeErr != nil {
			return CatalogRelationshipView{}, normalizeErr
		}
		if ensureErr := s.ensureActiveItemsHaveClassifier(
			ctx,
			ruleItemIDs,
			CatalogClassifierRule,
			"rule_item_ids",
		); ensureErr != nil {
			return CatalogRelationshipView{}, ensureErr
		}
		if replaceErr := s.ruleRepo.ReplaceForSkillItemID(
			ctx,
			sourceRow.ItemID,
			ruleItemIDs,
			updatedAt,
			updatedBy,
		); replaceErr != nil {
			return CatalogRelationshipView{}, fmt.Errorf(
				"replace catalog rule relationships for %q: %w",
				sourceRow.ItemID,
				replaceErr,
			)
		}
	}

	return s.Get(ctx, sourceRow.ItemID)
}

// Reconcile prunes stale relationship rows after sync cycles.
func (s *CatalogRelationshipService) Reconcile(ctx context.Context) (CatalogRelationshipReconcileReport, error) {
	if s == nil {
		return CatalogRelationshipReconcileReport{}, fmt.Errorf("catalog relationship service is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	activeSourceRows, err := s.sourceRepo.List(ctx, persistence.CatalogSourceListFilter{})
	if err != nil {
		return CatalogRelationshipReconcileReport{}, fmt.Errorf(
			"list active catalog source rows for relationship reconciliation: %w",
			err,
		)
	}
	activeByItemID := make(map[string]persistence.CatalogSourceRow, len(activeSourceRows))
	for _, row := range activeSourceRows {
		activeByItemID[row.ItemID] = row
	}

	report := CatalogRelationshipReconcileReport{}
	now := s.now().UTC()

	ruleRows, err := s.ruleRepo.List(ctx, persistence.CatalogSkillRuleRelationshipListFilter{})
	if err != nil {
		return report, fmt.Errorf("list skill-rule relationships for reconciliation: %w", err)
	}
	report.SkillRuleRowsScanned = len(ruleRows)

	rulesBySkill := make(map[string][]string, len(ruleRows))
	keptRulesBySkill := make(map[string][]string, len(ruleRows))
	for _, relationshipRow := range ruleRows {
		rulesBySkill[relationshipRow.SkillItemID] = append(
			rulesBySkill[relationshipRow.SkillItemID],
			relationshipRow.RuleItemID,
		)

		skillRow, hasSkill := activeByItemID[relationshipRow.SkillItemID]
		if !hasSkill || skillRow.Classifier != persistence.CatalogClassifierSkill {
			continue
		}
		ruleRow, hasRule := activeByItemID[relationshipRow.RuleItemID]
		if !hasRule || ruleRow.Classifier != persistence.CatalogClassifierRule {
			continue
		}

		keptRulesBySkill[relationshipRow.SkillItemID] = append(
			keptRulesBySkill[relationshipRow.SkillItemID],
			relationshipRow.RuleItemID,
		)
	}

	for skillItemID, existingRuleItemIDs := range rulesBySkill {
		skillRow, hasSkill := activeByItemID[skillItemID]
		if !hasSkill || skillRow.Classifier != persistence.CatalogClassifierSkill {
			if len(existingRuleItemIDs) == 0 {
				continue
			}
			if _, deleteErr := s.ruleRepo.DeleteBySkillItemID(ctx, skillItemID); deleteErr != nil {
				return report, fmt.Errorf(
					"delete stale rule relationships for missing/deleted skill %q: %w",
					skillItemID,
					deleteErr,
				)
			}
			report.SkillRuleRowsPruned += len(existingRuleItemIDs)
			continue
		}

		keptRuleItemIDs := keptRulesBySkill[skillItemID]
		if len(keptRuleItemIDs) == len(existingRuleItemIDs) {
			continue
		}
		if replaceErr := s.ruleRepo.ReplaceForSkillItemID(
			ctx,
			skillItemID,
			keptRuleItemIDs,
			now,
			nil,
		); replaceErr != nil {
			return report, fmt.Errorf("replace stale rule relationships for skill %q: %w", skillItemID, replaceErr)
		}
		report.SkillRuleRowsPruned += len(existingRuleItemIDs) - len(keptRuleItemIDs)
	}

	promptRows, err := s.promptRepo.List(ctx, persistence.CatalogSkillPromptRelationshipListFilter{})
	if err != nil {
		return report, fmt.Errorf("list skill-prompt relationships for reconciliation: %w", err)
	}
	report.SkillPromptRowsScanned = len(promptRows)

	for _, relationshipRow := range promptRows {
		skillRow, hasSkill := activeByItemID[relationshipRow.SkillItemID]
		if !hasSkill || skillRow.Classifier != persistence.CatalogClassifierSkill {
			if _, deleteErr := s.promptRepo.DeleteBySkillItemID(ctx, relationshipRow.SkillItemID); deleteErr != nil {
				return report, fmt.Errorf(
					"delete stale prompt relationship for missing/deleted skill %q: %w",
					relationshipRow.SkillItemID,
					deleteErr,
				)
			}
			report.SkillPromptRowsPruned++
			continue
		}

		promptRow, hasPrompt := activeByItemID[relationshipRow.PromptItemID]
		if !hasPrompt || promptRow.Classifier != persistence.CatalogClassifierPrompt {
			if _, deleteErr := s.promptRepo.DeleteBySkillItemID(ctx, relationshipRow.SkillItemID); deleteErr != nil {
				return report, fmt.Errorf(
					"delete stale prompt relationship for skill %q: %w",
					relationshipRow.SkillItemID,
					deleteErr,
				)
			}
			report.SkillPromptRowsPruned++
		}
	}

	return report, nil
}

func (s *CatalogRelationshipService) resolveSkillRelationships(
	ctx context.Context,
	skillItemID string,
) (CatalogRelationshipSet, error) {
	relationships := newCatalogRelationshipSet()

	promptRelationshipRow, err := s.promptRepo.GetBySkillItemID(ctx, skillItemID)
	if err != nil {
		if !errors.Is(err, persistence.ErrCatalogSkillPromptRelationshipNotFound) {
			return CatalogRelationshipSet{}, fmt.Errorf(
				"get prompt relationship for skill %q: %w",
				skillItemID,
				err,
			)
		}
	} else {
		promptRowByID, listErr := s.listActiveSourceRowsByItemIDs(ctx, []string{promptRelationshipRow.PromptItemID})
		if listErr != nil {
			return CatalogRelationshipSet{}, listErr
		}
		promptSourceRow, exists := promptRowByID[promptRelationshipRow.PromptItemID]
		if exists && promptSourceRow.Classifier == persistence.CatalogClassifierPrompt {
			promptItem, mapErr := mapCatalogRelationshipItem(promptSourceRow)
			if mapErr != nil {
				return CatalogRelationshipSet{}, mapErr
			}
			relationships.Prompt = &promptItem
		}
	}

	ruleRelationshipRows, err := s.ruleRepo.ListBySkillItemID(ctx, skillItemID)
	if err != nil {
		return CatalogRelationshipSet{}, fmt.Errorf("list rule relationships for skill %q: %w", skillItemID, err)
	}
	if len(ruleRelationshipRows) == 0 {
		return relationships, nil
	}

	ruleItemIDs := make([]string, 0, len(ruleRelationshipRows))
	for _, relationshipRow := range ruleRelationshipRows {
		ruleItemIDs = append(ruleItemIDs, relationshipRow.RuleItemID)
	}
	ruleRowsByItemID, err := s.listActiveSourceRowsByItemIDs(ctx, ruleItemIDs)
	if err != nil {
		return CatalogRelationshipSet{}, err
	}

	for _, relationshipRow := range ruleRelationshipRows {
		ruleSourceRow, exists := ruleRowsByItemID[relationshipRow.RuleItemID]
		if !exists || ruleSourceRow.Classifier != persistence.CatalogClassifierRule {
			continue
		}
		ruleItem, mapErr := mapCatalogRelationshipItem(ruleSourceRow)
		if mapErr != nil {
			return CatalogRelationshipSet{}, mapErr
		}
		relationships.Rules = append(relationships.Rules, ruleItem)
	}

	return relationships, nil
}

func (s *CatalogRelationshipService) resolveSkillsForPrompt(
	ctx context.Context,
	promptItemID string,
) ([]CatalogRelationshipItem, error) {
	promptRows, err := s.promptRepo.ListByPromptItemID(ctx, promptItemID)
	if err != nil {
		return nil, fmt.Errorf("list reverse prompt relationships for %q: %w", promptItemID, err)
	}
	if len(promptRows) == 0 {
		return []CatalogRelationshipItem{}, nil
	}

	skillItemIDs := make([]string, 0, len(promptRows))
	for _, row := range promptRows {
		skillItemIDs = append(skillItemIDs, row.SkillItemID)
	}
	skillRowsByItemID, err := s.listActiveSourceRowsByItemIDs(ctx, skillItemIDs)
	if err != nil {
		return nil, err
	}

	skills := make([]CatalogRelationshipItem, 0, len(promptRows))
	for _, row := range promptRows {
		skillSourceRow, exists := skillRowsByItemID[row.SkillItemID]
		if !exists || skillSourceRow.Classifier != persistence.CatalogClassifierSkill {
			continue
		}
		skillItem, mapErr := mapCatalogRelationshipItem(skillSourceRow)
		if mapErr != nil {
			return nil, mapErr
		}
		skills = append(skills, skillItem)
	}

	return skills, nil
}

func (s *CatalogRelationshipService) resolveSkillsForRule(
	ctx context.Context,
	ruleItemID string,
) ([]CatalogRelationshipItem, error) {
	ruleRows, err := s.ruleRepo.ListByRuleItemID(ctx, ruleItemID)
	if err != nil {
		return nil, fmt.Errorf("list reverse rule relationships for %q: %w", ruleItemID, err)
	}
	if len(ruleRows) == 0 {
		return []CatalogRelationshipItem{}, nil
	}

	skillItemIDs := make([]string, 0, len(ruleRows))
	for _, row := range ruleRows {
		skillItemIDs = append(skillItemIDs, row.SkillItemID)
	}
	skillRowsByItemID, err := s.listActiveSourceRowsByItemIDs(ctx, skillItemIDs)
	if err != nil {
		return nil, err
	}

	skills := make([]CatalogRelationshipItem, 0, len(ruleRows))
	for _, row := range ruleRows {
		skillSourceRow, exists := skillRowsByItemID[row.SkillItemID]
		if !exists || skillSourceRow.Classifier != persistence.CatalogClassifierSkill {
			continue
		}
		skillItem, mapErr := mapCatalogRelationshipItem(skillSourceRow)
		if mapErr != nil {
			return nil, mapErr
		}
		skills = append(skills, skillItem)
	}

	return skills, nil
}

func (s *CatalogRelationshipService) getActiveSourceRow(
	ctx context.Context,
	itemID string,
) (persistence.CatalogSourceRow, error) {
	sourceRow, err := s.sourceRepo.GetByItemID(ctx, itemID)
	if err != nil {
		if errors.Is(err, persistence.ErrCatalogSourceNotFound) {
			return persistence.CatalogSourceRow{}, fmt.Errorf("%w: item_id=%q", ErrCatalogRelationshipItemNotFound, itemID)
		}
		return persistence.CatalogSourceRow{}, fmt.Errorf("get source row for relationship item %q: %w", itemID, err)
	}
	if sourceRow.DeletedAt != nil {
		return persistence.CatalogSourceRow{}, fmt.Errorf("%w: item_id=%q", ErrCatalogRelationshipItemNotFound, itemID)
	}
	return sourceRow, nil
}

func (s *CatalogRelationshipService) listActiveSourceRowsByItemIDs(
	ctx context.Context,
	itemIDs []string,
) (map[string]persistence.CatalogSourceRow, error) {
	normalizedItemIDs := normalizeCatalogRelationshipOptionalIDs(itemIDs)
	if len(normalizedItemIDs) == 0 {
		return map[string]persistence.CatalogSourceRow{}, nil
	}

	rows, err := s.sourceRepo.List(ctx, persistence.CatalogSourceListFilter{ItemIDs: normalizedItemIDs})
	if err != nil {
		return nil, fmt.Errorf("list active source rows for relationship endpoints: %w", err)
	}

	byID := make(map[string]persistence.CatalogSourceRow, len(rows))
	for _, row := range rows {
		byID[row.ItemID] = row
	}

	return byID, nil
}

func (s *CatalogRelationshipService) ensureActiveItemHasClassifier(
	ctx context.Context,
	itemID string,
	expectedClassifier CatalogClassifier,
	field string,
) error {
	sourceRow, err := s.getActiveSourceRow(ctx, itemID)
	if err != nil {
		return err
	}

	classifier, err := mapCatalogRelationshipDomainClassifier(sourceRow.Classifier)
	if err != nil {
		return fmt.Errorf("map classifier for relationship item %q: %w", sourceRow.ItemID, err)
	}
	if classifier != expectedClassifier {
		return &CatalogRelationshipValidationError{
			Field:  field,
			Detail: fmt.Sprintf("must reference a %s item", expectedClassifier),
		}
	}
	return nil
}

func (s *CatalogRelationshipService) ensureActiveItemsHaveClassifier(
	ctx context.Context,
	itemIDs []string,
	expectedClassifier CatalogClassifier,
	field string,
) error {
	if len(itemIDs) == 0 {
		return nil
	}

	rowsByID, err := s.listActiveSourceRowsByItemIDs(ctx, itemIDs)
	if err != nil {
		return err
	}

	for _, itemID := range itemIDs {
		sourceRow, exists := rowsByID[itemID]
		if !exists {
			return fmt.Errorf("%w: %s=%q", ErrCatalogRelationshipItemNotFound, field, itemID)
		}

		classifier, mapErr := mapCatalogRelationshipDomainClassifier(sourceRow.Classifier)
		if mapErr != nil {
			return fmt.Errorf("map classifier for relationship item %q: %w", sourceRow.ItemID, mapErr)
		}
		if classifier != expectedClassifier {
			return &CatalogRelationshipValidationError{
				Field:  field,
				Detail: fmt.Sprintf("must reference only %s items", expectedClassifier),
			}
		}
	}

	return nil
}

func newCatalogRelationshipSet() CatalogRelationshipSet {
	return CatalogRelationshipSet{
		Rules:  []CatalogRelationshipItem{},
		Skills: []CatalogRelationshipItem{},
	}
}

func copyCatalogRelationshipSet(input CatalogRelationshipSet) CatalogRelationshipSet {
	copied := newCatalogRelationshipSet()

	if input.Prompt != nil {
		promptItem := *input.Prompt
		promptItem.ParentSkillID = copyCatalogRelationshipOptionalString(promptItem.ParentSkillID)
		promptItem.ResourcePath = copyCatalogRelationshipOptionalString(promptItem.ResourcePath)
		copied.Prompt = &promptItem
	}

	if len(input.Rules) > 0 {
		copied.Rules = make([]CatalogRelationshipItem, 0, len(input.Rules))
		for _, rule := range input.Rules {
			nextRule := rule
			nextRule.ParentSkillID = copyCatalogRelationshipOptionalString(nextRule.ParentSkillID)
			nextRule.ResourcePath = copyCatalogRelationshipOptionalString(nextRule.ResourcePath)
			copied.Rules = append(copied.Rules, nextRule)
		}
	}

	if len(input.Skills) > 0 {
		copied.Skills = make([]CatalogRelationshipItem, 0, len(input.Skills))
		for _, skill := range input.Skills {
			nextSkill := skill
			nextSkill.ParentSkillID = copyCatalogRelationshipOptionalString(nextSkill.ParentSkillID)
			nextSkill.ResourcePath = copyCatalogRelationshipOptionalString(nextSkill.ResourcePath)
			copied.Skills = append(copied.Skills, nextSkill)
		}
	}

	return copied
}

func mapCatalogRelationshipItem(sourceRow persistence.CatalogSourceRow) (CatalogRelationshipItem, error) {
	classifier, err := mapCatalogRelationshipDomainClassifier(sourceRow.Classifier)
	if err != nil {
		return CatalogRelationshipItem{}, fmt.Errorf("map relationship classifier for %q: %w", sourceRow.ItemID, err)
	}

	item := CatalogRelationshipItem{
		ID:         sourceRow.ItemID,
		Classifier: classifier,
		Name:       sourceRow.Name,
	}
	item.ParentSkillID = copyCatalogRelationshipOptionalString(sourceRow.ParentSkillID)
	item.ResourcePath = copyCatalogRelationshipOptionalString(sourceRow.ResourcePath)

	return item, nil
}

func mapCatalogRelationshipDomainClassifier(
	classifier persistence.CatalogClassifier,
) (CatalogClassifier, error) {
	switch classifier {
	case persistence.CatalogClassifierSkill:
		return CatalogClassifierSkill, nil
	case persistence.CatalogClassifierPrompt:
		return CatalogClassifierPrompt, nil
	case persistence.CatalogClassifierRule:
		return CatalogClassifierRule, nil
	default:
		return "", fmt.Errorf("catalog classifier %q is invalid", classifier)
	}
}

func normalizeCatalogRelationshipItemReference(rawItemID string, field string) (CatalogItemReference, error) {
	trimmed := strings.TrimSpace(rawItemID)
	if trimmed == "" {
		return CatalogItemReference{}, &CatalogRelationshipValidationError{
			Field:  field,
			Detail: "is required",
		}
	}

	reference, err := NormalizeCatalogItemReference(trimmed)
	if err != nil {
		return CatalogItemReference{}, &CatalogRelationshipValidationError{
			Field:  field,
			Detail: "is invalid",
			Cause:  err,
		}
	}

	return reference, nil
}

func normalizeCatalogRelationshipRuleItemIDs(rawRuleItemIDs []string) ([]string, error) {
	if len(rawRuleItemIDs) == 0 {
		return []string{}, nil
	}

	normalizedRuleItemIDs := make([]string, 0, len(rawRuleItemIDs))
	seen := make(map[string]struct{}, len(rawRuleItemIDs))
	for index, rawRuleItemID := range rawRuleItemIDs {
		fieldName := fmt.Sprintf("rule_item_ids[%d]", index)
		reference, err := normalizeCatalogRelationshipItemReference(rawRuleItemID, fieldName)
		if err != nil {
			return nil, err
		}
		if reference.Classifier != CatalogClassifierRule {
			return nil, &CatalogRelationshipValidationError{
				Field:  fieldName,
				Detail: "must reference a rule item",
			}
		}
		if _, exists := seen[reference.ItemID]; exists {
			return nil, &CatalogRelationshipValidationError{
				Field:  "rule_item_ids",
				Detail: fmt.Sprintf("contains duplicate value %q", reference.ItemID),
			}
		}
		seen[reference.ItemID] = struct{}{}
		normalizedRuleItemIDs = append(normalizedRuleItemIDs, reference.ItemID)
	}

	sort.Strings(normalizedRuleItemIDs)
	return normalizedRuleItemIDs, nil
}

func normalizeCatalogRelationshipOptionalIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, rawValue := range values {
		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

func normalizeCatalogRelationshipOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func copyCatalogRelationshipOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := strings.TrimSpace(*value)
	if copied == "" {
		return nil
	}
	return &copied
}
