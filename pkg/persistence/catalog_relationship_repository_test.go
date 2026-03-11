package persistence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	relationshipSkillAlpha = "skill:repo-a/alpha"
	relationshipSkillBeta  = "skill:repo-a/beta"
	relationshipSkillGamma = "skill:repo-a/gamma"

	relationshipPromptAlpha  = "prompt:repo-a/alpha:prompts/system-a.md"
	relationshipPromptBeta   = "prompt:repo-a/beta:prompts/system-b.md"
	relationshipPromptShared = "prompt:repo-a/shared:prompts/shared.md"

	relationshipRuleA = "rule:repo-a/shared-rules:rules/a-security.md"
	relationshipRuleB = "rule:repo-a/shared-rules:rules/b-style.md"
	relationshipRuleC = "rule:repo-a/shared-rules:rules/c-ops.md"
)

func TestNewCatalogSkillRuleRelationshipRepository_WithNilExecutor_ReturnsError(t *testing.T) {
	_, err := NewCatalogSkillRuleRelationshipRepository(nil)
	if err == nil {
		t.Fatalf("expected nil executor error, got nil")
	}
}

func TestNewCatalogSkillPromptRelationshipRepository_WithNilExecutor_ReturnsError(t *testing.T) {
	_, err := NewCatalogSkillPromptRelationshipRepository(nil)
	if err == nil {
		t.Fatalf("expected nil executor error, got nil")
	}
}

func TestCatalogSkillRuleRelationshipRepository_ReplaceListAndDuplicateHandling(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	repo := newCatalogSkillRuleRelationshipRepositoryForTest(t, db)

	seedRelationshipFixtureSourceItems(t, ctx, sourceRepo)

	createdAtA := time.Date(2026, time.March, 11, 10, 0, 0, 0, time.UTC)
	if err := repo.ReplaceForSkillItemID(
		ctx,
		relationshipSkillAlpha,
		[]string{relationshipRuleB, relationshipRuleA},
		createdAtA,
		stringPointer("writer-a"),
	); err != nil {
		t.Fatalf("expected initial skill->rules replace to succeed, got %v", err)
	}

	rowsAfterFirstReplace, err := repo.ListBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected list by skill after first replace to succeed, got %v", err)
	}
	if len(rowsAfterFirstReplace) != 2 {
		t.Fatalf("expected 2 rule rows after first replace, got %+v", rowsAfterFirstReplace)
	}
	if rowsAfterFirstReplace[0].RuleItemID != relationshipRuleA || rowsAfterFirstReplace[1].RuleItemID != relationshipRuleB {
		t.Fatalf("expected deterministic ordering by rule_item_id, got %+v", rowsAfterFirstReplace)
	}
	for _, row := range rowsAfterFirstReplace {
		if !row.CreatedAt.Equal(createdAtA) {
			t.Fatalf("expected created_at %s, got %s", createdAtA, row.CreatedAt)
		}
		if !row.UpdatedAt.Equal(createdAtA) {
			t.Fatalf("expected updated_at %s, got %s", createdAtA, row.UpdatedAt)
		}
		if row.UpdatedBy == nil || *row.UpdatedBy != "writer-a" {
			t.Fatalf("expected updated_by writer-a, got %+v", row.UpdatedBy)
		}
	}

	createdAtB := createdAtA.Add(1 * time.Hour)
	if err := repo.ReplaceForSkillItemID(
		ctx,
		relationshipSkillAlpha,
		[]string{relationshipRuleA, relationshipRuleB},
		createdAtB,
		stringPointer("writer-b"),
	); err != nil {
		t.Fatalf("expected idempotent skill->rules replace to succeed, got %v", err)
	}

	rowsAfterSecondReplace, err := repo.ListBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected list by skill after second replace to succeed, got %v", err)
	}
	for _, row := range rowsAfterSecondReplace {
		if !row.CreatedAt.Equal(createdAtA) {
			t.Fatalf("expected stable created_at %s, got %s", createdAtA, row.CreatedAt)
		}
		if !row.UpdatedAt.Equal(createdAtB) {
			t.Fatalf("expected updated_at to advance to %s, got %s", createdAtB, row.UpdatedAt)
		}
		if row.UpdatedBy == nil || *row.UpdatedBy != "writer-b" {
			t.Fatalf("expected updated_by writer-b, got %+v", row.UpdatedBy)
		}
	}

	createdAtC := createdAtA.Add(2 * time.Hour)
	if err := repo.ReplaceForSkillItemID(
		ctx,
		relationshipSkillAlpha,
		[]string{relationshipRuleA, relationshipRuleC},
		createdAtC,
		stringPointer("writer-c"),
	); err != nil {
		t.Fatalf("expected changed skill->rules replace to succeed, got %v", err)
	}

	rowsAfterThirdReplace, err := repo.ListBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected list by skill after third replace to succeed, got %v", err)
	}
	if len(rowsAfterThirdReplace) != 2 {
		t.Fatalf("expected 2 rule rows after third replace, got %+v", rowsAfterThirdReplace)
	}
	if rowsAfterThirdReplace[0].RuleItemID != relationshipRuleA || rowsAfterThirdReplace[1].RuleItemID != relationshipRuleC {
		t.Fatalf("expected rule-a and rule-c after third replace, got %+v", rowsAfterThirdReplace)
	}
	if !rowsAfterThirdReplace[0].CreatedAt.Equal(createdAtA) {
		t.Fatalf("expected existing rule-a created_at %s, got %s", createdAtA, rowsAfterThirdReplace[0].CreatedAt)
	}
	if !rowsAfterThirdReplace[0].UpdatedAt.Equal(createdAtC) {
		t.Fatalf("expected existing rule-a updated_at %s, got %s", createdAtC, rowsAfterThirdReplace[0].UpdatedAt)
	}
	if !rowsAfterThirdReplace[1].CreatedAt.Equal(createdAtC) {
		t.Fatalf("expected new rule-c created_at %s, got %s", createdAtC, rowsAfterThirdReplace[1].CreatedAt)
	}
	if !rowsAfterThirdReplace[1].UpdatedAt.Equal(createdAtC) {
		t.Fatalf("expected new rule-c updated_at %s, got %s", createdAtC, rowsAfterThirdReplace[1].UpdatedAt)
	}

	err = repo.ReplaceForSkillItemID(
		ctx,
		relationshipSkillAlpha,
		[]string{relationshipRuleA, relationshipRuleA},
		createdAtC.Add(1*time.Minute),
		nil,
	)
	if err == nil {
		t.Fatalf("expected duplicate rule ids to fail, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate rule id error, got %v", err)
	}

	rowsAfterDuplicateFailure, err := repo.ListBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected list by skill after duplicate failure to succeed, got %v", err)
	}
	if len(rowsAfterDuplicateFailure) != 2 || rowsAfterDuplicateFailure[0].RuleItemID != relationshipRuleA || rowsAfterDuplicateFailure[1].RuleItemID != relationshipRuleC {
		t.Fatalf("expected duplicate failure to preserve prior rows, got %+v", rowsAfterDuplicateFailure)
	}

	if err := repo.ReplaceForSkillItemID(
		ctx,
		relationshipSkillAlpha,
		nil,
		createdAtC.Add(2*time.Hour),
		nil,
	); err != nil {
		t.Fatalf("expected clear skill->rules replace to succeed, got %v", err)
	}

	rowsAfterClear, err := repo.ListBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected list by skill after clear to succeed, got %v", err)
	}
	if len(rowsAfterClear) != 0 {
		t.Fatalf("expected no skill->rule rows after clear, got %+v", rowsAfterClear)
	}
}

func TestCatalogSkillRuleRelationshipRepository_ReplaceWithInvalidRule_RollsBack(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	repo := newCatalogSkillRuleRelationshipRepositoryForTest(t, db)

	seedRelationshipFixtureSourceItems(t, ctx, sourceRepo)

	baselineUpdatedAt := time.Date(2026, time.March, 11, 11, 0, 0, 0, time.UTC)
	if err := repo.ReplaceForSkillItemID(
		ctx,
		relationshipSkillBeta,
		[]string{relationshipRuleA},
		baselineUpdatedAt,
		stringPointer("baseline"),
	); err != nil {
		t.Fatalf("expected baseline skill->rules replace to succeed, got %v", err)
	}

	err := repo.ReplaceForSkillItemID(
		ctx,
		relationshipSkillBeta,
		[]string{relationshipRuleA, "rule:missing"},
		baselineUpdatedAt.Add(1*time.Hour),
		stringPointer("invalid"),
	)
	if err == nil {
		t.Fatalf("expected replace with missing rule to fail, got nil")
	}
	if !strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY") {
		t.Fatalf("expected missing rule failure to mention FOREIGN KEY, got %v", err)
	}

	rowsAfterFailure, err := repo.ListBySkillItemID(ctx, relationshipSkillBeta)
	if err != nil {
		t.Fatalf("expected list by skill after failed replace to succeed, got %v", err)
	}
	if len(rowsAfterFailure) != 1 || rowsAfterFailure[0].RuleItemID != relationshipRuleA {
		t.Fatalf("expected failed replace rollback to keep baseline rows, got %+v", rowsAfterFailure)
	}
	if !rowsAfterFailure[0].CreatedAt.Equal(baselineUpdatedAt) {
		t.Fatalf("expected rollback to preserve created_at %s, got %s", baselineUpdatedAt, rowsAfterFailure[0].CreatedAt)
	}
	if !rowsAfterFailure[0].UpdatedAt.Equal(baselineUpdatedAt) {
		t.Fatalf("expected rollback to preserve updated_at %s, got %s", baselineUpdatedAt, rowsAfterFailure[0].UpdatedAt)
	}
}

func TestCatalogSkillPromptRelationshipRepository_SetGetClearAndMissingBehavior(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	repo := newCatalogSkillPromptRelationshipRepositoryForTest(t, db)

	seedRelationshipFixtureSourceItems(t, ctx, sourceRepo)

	updatedAtA := time.Date(2026, time.March, 11, 12, 0, 0, 0, time.UTC)
	if err := repo.SetForSkillItemID(
		ctx,
		relationshipSkillAlpha,
		relationshipPromptAlpha,
		updatedAtA,
		stringPointer("setter-a"),
	); err != nil {
		t.Fatalf("expected first set skill->prompt to succeed, got %v", err)
	}

	firstPromptRow, err := repo.GetBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected get skill->prompt to succeed, got %v", err)
	}
	if firstPromptRow.PromptItemID != relationshipPromptAlpha {
		t.Fatalf("expected prompt alpha after first set, got %+v", firstPromptRow)
	}
	if !firstPromptRow.CreatedAt.Equal(updatedAtA) || !firstPromptRow.UpdatedAt.Equal(updatedAtA) {
		t.Fatalf("expected created_at/updated_at %s, got %+v", updatedAtA, firstPromptRow)
	}

	updatedAtB := updatedAtA.Add(1 * time.Hour)
	if err := repo.SetForSkillItemID(
		ctx,
		relationshipSkillAlpha,
		relationshipPromptBeta,
		updatedAtB,
		stringPointer("setter-b"),
	); err != nil {
		t.Fatalf("expected prompt replacement to succeed, got %v", err)
	}

	secondPromptRow, err := repo.GetBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected get skill->prompt after replace to succeed, got %v", err)
	}
	if secondPromptRow.PromptItemID != relationshipPromptBeta {
		t.Fatalf("expected prompt beta after replace, got %+v", secondPromptRow)
	}
	if !secondPromptRow.CreatedAt.Equal(updatedAtA) {
		t.Fatalf("expected prompt created_at to stay at %s, got %s", updatedAtA, secondPromptRow.CreatedAt)
	}
	if !secondPromptRow.UpdatedAt.Equal(updatedAtB) {
		t.Fatalf("expected prompt updated_at to advance to %s, got %s", updatedAtB, secondPromptRow.UpdatedAt)
	}
	if secondPromptRow.UpdatedBy == nil || *secondPromptRow.UpdatedBy != "setter-b" {
		t.Fatalf("expected updated_by setter-b, got %+v", secondPromptRow.UpdatedBy)
	}

	reverseRows, err := repo.ListByPromptItemID(ctx, relationshipPromptBeta)
	if err != nil {
		t.Fatalf("expected reverse prompt->skills list to succeed, got %v", err)
	}
	if len(reverseRows) != 1 || reverseRows[0].SkillItemID != relationshipSkillAlpha {
		t.Fatalf("expected one reverse row for prompt beta, got %+v", reverseRows)
	}

	deleted, err := repo.ClearBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected clear skill->prompt to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected clear skill->prompt to report deleted=true")
	}

	_, err = repo.GetBySkillItemID(ctx, relationshipSkillAlpha)
	if !errors.Is(err, ErrCatalogSkillPromptRelationshipNotFound) {
		t.Fatalf("expected missing prompt relationship error, got %v", err)
	}

	deletedAgain, err := repo.ClearBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected second clear skill->prompt to succeed, got %v", err)
	}
	if deletedAgain {
		t.Fatalf("expected second clear skill->prompt to report deleted=false")
	}

	emptyRows, err := repo.ListBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected list by skill after clear to succeed, got %v", err)
	}
	if len(emptyRows) != 0 {
		t.Fatalf("expected empty list by skill after clear, got %+v", emptyRows)
	}

	emptyReverseRows, err := repo.ListByPromptItemID(ctx, "prompt:missing")
	if err != nil {
		t.Fatalf("expected empty reverse prompt list to succeed, got %v", err)
	}
	if len(emptyReverseRows) != 0 {
		t.Fatalf("expected empty reverse prompt list, got %+v", emptyReverseRows)
	}
}

func TestCatalogSkillPromptRelationshipRepository_SetWithInvalidPrompt_RollsBack(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	repo := newCatalogSkillPromptRelationshipRepositoryForTest(t, db)

	seedRelationshipFixtureSourceItems(t, ctx, sourceRepo)

	baselineUpdatedAt := time.Date(2026, time.March, 11, 13, 0, 0, 0, time.UTC)
	if err := repo.SetForSkillItemID(
		ctx,
		relationshipSkillBeta,
		relationshipPromptAlpha,
		baselineUpdatedAt,
		stringPointer("baseline"),
	); err != nil {
		t.Fatalf("expected baseline skill->prompt set to succeed, got %v", err)
	}

	err := repo.SetForSkillItemID(
		ctx,
		relationshipSkillBeta,
		"prompt:missing",
		baselineUpdatedAt.Add(1*time.Hour),
		stringPointer("invalid"),
	)
	if err == nil {
		t.Fatalf("expected set with missing prompt to fail, got nil")
	}
	if !strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY") {
		t.Fatalf("expected missing prompt failure to mention FOREIGN KEY, got %v", err)
	}

	rowAfterFailure, err := repo.GetBySkillItemID(ctx, relationshipSkillBeta)
	if err != nil {
		t.Fatalf("expected get skill->prompt after failed set to succeed, got %v", err)
	}
	if rowAfterFailure.PromptItemID != relationshipPromptAlpha {
		t.Fatalf("expected rollback to preserve baseline prompt, got %+v", rowAfterFailure)
	}
	if !rowAfterFailure.CreatedAt.Equal(baselineUpdatedAt) {
		t.Fatalf("expected rollback to preserve created_at %s, got %s", baselineUpdatedAt, rowAfterFailure.CreatedAt)
	}
	if !rowAfterFailure.UpdatedAt.Equal(baselineUpdatedAt) {
		t.Fatalf("expected rollback to preserve updated_at %s, got %s", baselineUpdatedAt, rowAfterFailure.UpdatedAt)
	}
}

func TestCatalogRelationshipRepositories_ReverseLookupAndPruneHelpers(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	ruleRepo := newCatalogSkillRuleRelationshipRepositoryForTest(t, db)
	promptRepo := newCatalogSkillPromptRelationshipRepositoryForTest(t, db)

	seedRelationshipFixtureSourceItems(t, ctx, sourceRepo)

	updatedAt := time.Date(2026, time.March, 11, 14, 0, 0, 0, time.UTC)

	if err := promptRepo.SetForSkillItemID(ctx, relationshipSkillAlpha, relationshipPromptShared, updatedAt, nil); err != nil {
		t.Fatalf("expected prompt set for skill alpha to succeed, got %v", err)
	}
	if err := promptRepo.SetForSkillItemID(ctx, relationshipSkillBeta, relationshipPromptShared, updatedAt, nil); err != nil {
		t.Fatalf("expected prompt set for skill beta to succeed, got %v", err)
	}
	if err := promptRepo.SetForSkillItemID(ctx, relationshipSkillGamma, relationshipPromptAlpha, updatedAt, nil); err != nil {
		t.Fatalf("expected prompt set for skill gamma to succeed, got %v", err)
	}

	if err := ruleRepo.ReplaceForSkillItemID(ctx, relationshipSkillAlpha, []string{relationshipRuleA, relationshipRuleB}, updatedAt, nil); err != nil {
		t.Fatalf("expected rules replace for skill alpha to succeed, got %v", err)
	}
	if err := ruleRepo.ReplaceForSkillItemID(ctx, relationshipSkillBeta, []string{relationshipRuleB}, updatedAt, nil); err != nil {
		t.Fatalf("expected rules replace for skill beta to succeed, got %v", err)
	}
	if err := ruleRepo.ReplaceForSkillItemID(ctx, relationshipSkillGamma, []string{relationshipRuleA, relationshipRuleC}, updatedAt, nil); err != nil {
		t.Fatalf("expected rules replace for skill gamma to succeed, got %v", err)
	}

	reversePromptRows, err := promptRepo.ListByPromptItemID(ctx, relationshipPromptShared)
	if err != nil {
		t.Fatalf("expected reverse prompt->skills lookup to succeed, got %v", err)
	}
	if len(reversePromptRows) != 2 || reversePromptRows[0].SkillItemID != relationshipSkillAlpha || reversePromptRows[1].SkillItemID != relationshipSkillBeta {
		t.Fatalf("expected deterministic reverse prompt->skills rows, got %+v", reversePromptRows)
	}

	reverseRuleRows, err := ruleRepo.ListByRuleItemID(ctx, relationshipRuleA)
	if err != nil {
		t.Fatalf("expected reverse rule->skills lookup to succeed, got %v", err)
	}
	if len(reverseRuleRows) != 2 || reverseRuleRows[0].SkillItemID != relationshipSkillAlpha || reverseRuleRows[1].SkillItemID != relationshipSkillGamma {
		t.Fatalf("expected deterministic reverse rule->skills rows, got %+v", reverseRuleRows)
	}

	deletedPromptRows, err := promptRepo.DeleteByPromptItemID(ctx, relationshipPromptShared)
	if err != nil {
		t.Fatalf("expected prune delete by prompt endpoint to succeed, got %v", err)
	}
	if !deletedPromptRows {
		t.Fatalf("expected delete-by-prompt prune helper to report deleted=true")
	}

	postPromptPruneRows, err := promptRepo.ListByPromptItemID(ctx, relationshipPromptShared)
	if err != nil {
		t.Fatalf("expected list by prompt after prune to succeed, got %v", err)
	}
	if len(postPromptPruneRows) != 0 {
		t.Fatalf("expected no rows for pruned prompt endpoint, got %+v", postPromptPruneRows)
	}

	deletedRuleRows, err := ruleRepo.DeleteByRuleItemID(ctx, relationshipRuleB)
	if err != nil {
		t.Fatalf("expected prune delete by rule endpoint to succeed, got %v", err)
	}
	if !deletedRuleRows {
		t.Fatalf("expected delete-by-rule prune helper to report deleted=true")
	}

	postRulePruneRows, err := ruleRepo.ListByRuleItemID(ctx, relationshipRuleB)
	if err != nil {
		t.Fatalf("expected list by rule after prune to succeed, got %v", err)
	}
	if len(postRulePruneRows) != 0 {
		t.Fatalf("expected no rows for pruned rule endpoint, got %+v", postRulePruneRows)
	}

	deletedSkillRuleRows, err := ruleRepo.DeleteBySkillItemID(ctx, relationshipSkillAlpha)
	if err != nil {
		t.Fatalf("expected prune delete by skill endpoint for rules to succeed, got %v", err)
	}
	if !deletedSkillRuleRows {
		t.Fatalf("expected delete-by-skill for rules to report deleted=true")
	}

	deletedSkillPromptRows, err := promptRepo.DeleteBySkillItemID(ctx, relationshipSkillGamma)
	if err != nil {
		t.Fatalf("expected prune delete by skill endpoint for prompt to succeed, got %v", err)
	}
	if !deletedSkillPromptRows {
		t.Fatalf("expected delete-by-skill for prompt to report deleted=true")
	}

	remainingRuleRowsForSkillGamma, err := ruleRepo.ListBySkillItemID(ctx, relationshipSkillGamma)
	if err != nil {
		t.Fatalf("expected list by skill gamma to succeed, got %v", err)
	}
	if len(remainingRuleRowsForSkillGamma) != 2 {
		t.Fatalf("expected skill gamma rule rows to remain after unrelated prompt prune, got %+v", remainingRuleRowsForSkillGamma)
	}

	remainingPromptRowsForSkillBeta, err := promptRepo.ListBySkillItemID(ctx, relationshipSkillBeta)
	if err != nil {
		t.Fatalf("expected list by skill beta prompt rows to succeed, got %v", err)
	}
	if len(remainingPromptRowsForSkillBeta) != 0 {
		t.Fatalf("expected skill beta prompt rows to be pruned with shared prompt endpoint delete, got %+v", remainingPromptRowsForSkillBeta)
	}
}

func TestCatalogRelationshipRepositories_List_InvalidFiltersReturnError(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	ruleRepo := newCatalogSkillRuleRelationshipRepositoryForTest(t, db)
	promptRepo := newCatalogSkillPromptRelationshipRepositoryForTest(t, db)

	_, err := ruleRepo.List(ctx, CatalogSkillRuleRelationshipListFilter{
		SkillItemID:  relationshipSkillAlpha,
		SkillItemIDs: []string{relationshipSkillBeta},
	})
	if err == nil {
		t.Fatalf("expected invalid skill filter for rule relationship list, got nil")
	}

	_, err = ruleRepo.List(ctx, CatalogSkillRuleRelationshipListFilter{
		RuleItemID:  relationshipRuleA,
		RuleItemIDs: []string{relationshipRuleB},
	})
	if err == nil {
		t.Fatalf("expected invalid rule filter for rule relationship list, got nil")
	}

	_, err = promptRepo.List(ctx, CatalogSkillPromptRelationshipListFilter{
		SkillItemID:  relationshipSkillAlpha,
		SkillItemIDs: []string{relationshipSkillBeta},
	})
	if err == nil {
		t.Fatalf("expected invalid skill filter for prompt relationship list, got nil")
	}

	_, err = promptRepo.List(ctx, CatalogSkillPromptRelationshipListFilter{
		PromptItemID:  relationshipPromptAlpha,
		PromptItemIDs: []string{relationshipPromptBeta},
	})
	if err == nil {
		t.Fatalf("expected invalid prompt filter for prompt relationship list, got nil")
	}
}

func seedRelationshipFixtureSourceItems(
	t *testing.T,
	ctx context.Context,
	sourceRepo *CatalogSourceRepository,
) {
	t.Helper()

	lastSyncedAt := time.Date(2026, time.March, 11, 9, 0, 0, 0, time.UTC)
	rows := []CatalogSourceRow{
		{
			ItemID:           relationshipSkillAlpha,
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             "alpha",
			Description:      "relationship fixture skill alpha",
			Content:          "skill alpha content",
			ContentHash:      "sha256:relationship-skill-alpha",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           relationshipSkillBeta,
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             "beta",
			Description:      "relationship fixture skill beta",
			Content:          "skill beta content",
			ContentHash:      "sha256:relationship-skill-beta",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           relationshipSkillGamma,
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             "gamma",
			Description:      "relationship fixture skill gamma",
			Content:          "skill gamma content",
			ContentHash:      "sha256:relationship-skill-gamma",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           relationshipPromptAlpha,
			Classifier:       CatalogClassifierPrompt,
			SourceType:       CatalogSourceTypeLocal,
			ParentSkillID:    stringPointer(relationshipSkillAlpha),
			ResourcePath:     stringPointer("prompts/system-a.md"),
			Name:             "system-a",
			Description:      "relationship fixture prompt alpha",
			Content:          "prompt alpha content",
			ContentHash:      "sha256:relationship-prompt-alpha",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           relationshipPromptBeta,
			Classifier:       CatalogClassifierPrompt,
			SourceType:       CatalogSourceTypeLocal,
			ParentSkillID:    stringPointer(relationshipSkillBeta),
			ResourcePath:     stringPointer("prompts/system-b.md"),
			Name:             "system-b",
			Description:      "relationship fixture prompt beta",
			Content:          "prompt beta content",
			ContentHash:      "sha256:relationship-prompt-beta",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           relationshipPromptShared,
			Classifier:       CatalogClassifierPrompt,
			SourceType:       CatalogSourceTypeLocal,
			ParentSkillID:    stringPointer(relationshipSkillAlpha),
			ResourcePath:     stringPointer("prompts/shared.md"),
			Name:             "shared",
			Description:      "relationship fixture shared prompt",
			Content:          "shared prompt content",
			ContentHash:      "sha256:relationship-prompt-shared",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           relationshipRuleA,
			Classifier:       CatalogClassifierRule,
			SourceType:       CatalogSourceTypeLocal,
			ParentSkillID:    stringPointer("skill:repo-a/shared-rules"),
			ResourcePath:     stringPointer("rules/a-security.md"),
			Name:             "rule-a",
			Description:      "relationship fixture rule a",
			Content:          "rule a content",
			ContentHash:      "sha256:relationship-rule-a",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           relationshipRuleB,
			Classifier:       CatalogClassifierRule,
			SourceType:       CatalogSourceTypeLocal,
			ParentSkillID:    stringPointer("skill:repo-a/shared-rules"),
			ResourcePath:     stringPointer("rules/b-style.md"),
			Name:             "rule-b",
			Description:      "relationship fixture rule b",
			Content:          "rule b content",
			ContentHash:      "sha256:relationship-rule-b",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           relationshipRuleC,
			Classifier:       CatalogClassifierRule,
			SourceType:       CatalogSourceTypeLocal,
			ParentSkillID:    stringPointer("skill:repo-a/shared-rules"),
			ResourcePath:     stringPointer("rules/c-ops.md"),
			Name:             "rule-c",
			Description:      "relationship fixture rule c",
			Content:          "rule c content",
			ContentHash:      "sha256:relationship-rule-c",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
	}

	for _, row := range rows {
		mustUpsertCatalogSourceRow(t, ctx, sourceRepo, row)
	}
}
