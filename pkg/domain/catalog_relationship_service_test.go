package domain

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

func TestCatalogRelationshipService_Get_ReturnsForwardRelationshipsForSkill(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	ruleRepo := newCatalogSkillRuleRelationshipRepositoryForDomainTest(t, db)
	promptRepo := newCatalogSkillPromptRelationshipRepositoryForDomainTest(t, db)
	service := newCatalogRelationshipServiceForDomainTest(t, sourceRepo, ruleRepo, promptRepo)

	skillItemID := BuildSkillCatalogItemID("repo-a/base-skill")
	promptItemID := BuildPromptCatalogItemID("repo-a/base-skill", "prompts/system.md")
	ruleAItemID := BuildRuleCatalogItemID("repo-b/shared-rules", "rules/security.md")
	ruleBItemID := BuildRuleCatalogItemID("repo-b/shared-rules", "rules/style.md")
	syncedAt := time.Date(2026, time.March, 7, 10, 30, 0, 0, time.UTC)

	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           skillItemID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "base-skill",
		Description:      "base skill",
		Content:          "skill content",
		ContentHash:      buildCatalogContentHash("skill content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           promptItemID,
		Classifier:       persistence.CatalogClassifierPrompt,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       stringPointer("repo-a"),
		ParentSkillID:    stringPointer("repo-a/base-skill"),
		ResourcePath:     stringPointer("prompts/system.md"),
		Name:             "system.md",
		Description:      "system prompt",
		Content:          "prompt content",
		ContentHash:      buildCatalogContentHash("prompt content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           ruleAItemID,
		Classifier:       persistence.CatalogClassifierRule,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       stringPointer("repo-b"),
		ParentSkillID:    stringPointer("repo-b/shared-rules"),
		ResourcePath:     stringPointer("rules/security.md"),
		Name:             "security.md",
		Description:      "security rule",
		Content:          "security content",
		ContentHash:      buildCatalogContentHash("security content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           ruleBItemID,
		Classifier:       persistence.CatalogClassifierRule,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       stringPointer("repo-b"),
		ParentSkillID:    stringPointer("repo-b/shared-rules"),
		ResourcePath:     stringPointer("rules/style.md"),
		Name:             "style.md",
		Description:      "style rule",
		Content:          "style content",
		ContentHash:      buildCatalogContentHash("style content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})

	mustSetCatalogPromptRelationshipForDomainTest(t, ctx, promptRepo, skillItemID, promptItemID, syncedAt)
	mustReplaceCatalogRuleRelationshipsForDomainTest(
		t,
		ctx,
		ruleRepo,
		skillItemID,
		[]string{ruleBItemID, ruleAItemID},
		syncedAt,
	)

	view, err := service.Get(ctx, skillItemID)
	if err != nil {
		t.Fatalf("expected relationship get to succeed, got %v", err)
	}

	if view.ItemID != skillItemID {
		t.Fatalf("expected relationship view item %q, got %q", skillItemID, view.ItemID)
	}
	if view.Relationships.Prompt == nil {
		t.Fatalf("expected prompt relationship projection to be populated")
	}
	if view.Relationships.Prompt.ID != promptItemID {
		t.Fatalf("expected prompt relationship id %q, got %q", promptItemID, view.Relationships.Prompt.ID)
	}
	if view.Relationships.Prompt.Classifier != CatalogClassifierPrompt {
		t.Fatalf(
			"expected prompt relationship classifier %q, got %q",
			CatalogClassifierPrompt,
			view.Relationships.Prompt.Classifier,
		)
	}

	expectedRuleIDs := []string{ruleAItemID, ruleBItemID}
	actualRuleIDs := make([]string, 0, len(view.Relationships.Rules))
	for _, rule := range view.Relationships.Rules {
		actualRuleIDs = append(actualRuleIDs, rule.ID)
	}
	if !reflect.DeepEqual(actualRuleIDs, expectedRuleIDs) {
		t.Fatalf("expected forward rule ids %+v, got %+v", expectedRuleIDs, actualRuleIDs)
	}
	if len(view.Relationships.Skills) != 0 {
		t.Fatalf("expected skill forward relationship to expose no reverse skills, got %+v", view.Relationships.Skills)
	}
}

func TestCatalogRelationshipService_Get_ReturnsReverseSkillsForPromptAndRule(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	ruleRepo := newCatalogSkillRuleRelationshipRepositoryForDomainTest(t, db)
	promptRepo := newCatalogSkillPromptRelationshipRepositoryForDomainTest(t, db)
	service := newCatalogRelationshipServiceForDomainTest(t, sourceRepo, ruleRepo, promptRepo)

	skillAlphaItemID := BuildSkillCatalogItemID("repo-a/alpha")
	skillBetaItemID := BuildSkillCatalogItemID("repo-a/beta")
	promptSharedItemID := BuildPromptCatalogItemID("repo-a/shared", "prompts/system.md")
	ruleSharedItemID := BuildRuleCatalogItemID("repo-a/shared", "rules/security.md")
	syncedAt := time.Date(2026, time.March, 7, 11, 0, 0, 0, time.UTC)

	for _, row := range []persistence.CatalogSourceRow{
		{
			ItemID:           skillAlphaItemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "alpha",
			Description:      "alpha skill",
			Content:          "alpha content",
			ContentHash:      buildCatalogContentHash("alpha content"),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           skillBetaItemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "beta",
			Description:      "beta skill",
			Content:          "beta content",
			ContentHash:      buildCatalogContentHash("beta content"),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           promptSharedItemID,
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/shared"),
			ResourcePath:     stringPointer("prompts/system.md"),
			Name:             "system.md",
			Description:      "shared prompt",
			Content:          "prompt content",
			ContentHash:      buildCatalogContentHash("prompt content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           ruleSharedItemID,
			Classifier:       persistence.CatalogClassifierRule,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/shared"),
			ResourcePath:     stringPointer("rules/security.md"),
			Name:             "security.md",
			Description:      "shared rule",
			Content:          "rule content",
			ContentHash:      buildCatalogContentHash("rule content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
	} {
		mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, row)
	}

	mustSetCatalogPromptRelationshipForDomainTest(t, ctx, promptRepo, skillAlphaItemID, promptSharedItemID, syncedAt)
	mustSetCatalogPromptRelationshipForDomainTest(t, ctx, promptRepo, skillBetaItemID, promptSharedItemID, syncedAt)
	mustReplaceCatalogRuleRelationshipsForDomainTest(
		t,
		ctx,
		ruleRepo,
		skillAlphaItemID,
		[]string{ruleSharedItemID},
		syncedAt,
	)
	mustReplaceCatalogRuleRelationshipsForDomainTest(
		t,
		ctx,
		ruleRepo,
		skillBetaItemID,
		[]string{ruleSharedItemID},
		syncedAt,
	)

	promptView, err := service.Get(ctx, promptSharedItemID)
	if err != nil {
		t.Fatalf("expected prompt relationship get to succeed, got %v", err)
	}
	if promptView.Relationships.Prompt != nil {
		t.Fatalf("expected prompt reverse view prompt relationship to be nil, got %+v", promptView.Relationships.Prompt)
	}
	if len(promptView.Relationships.Rules) != 0 {
		t.Fatalf("expected prompt reverse view rules to be empty, got %+v", promptView.Relationships.Rules)
	}
	expectedSkillIDs := []string{skillAlphaItemID, skillBetaItemID}
	actualPromptSkillIDs := make([]string, 0, len(promptView.Relationships.Skills))
	for _, skill := range promptView.Relationships.Skills {
		actualPromptSkillIDs = append(actualPromptSkillIDs, skill.ID)
	}
	if !reflect.DeepEqual(actualPromptSkillIDs, expectedSkillIDs) {
		t.Fatalf("expected prompt reverse skills %+v, got %+v", expectedSkillIDs, actualPromptSkillIDs)
	}

	ruleView, err := service.Get(ctx, ruleSharedItemID)
	if err != nil {
		t.Fatalf("expected rule relationship get to succeed, got %v", err)
	}
	if ruleView.Relationships.Prompt != nil {
		t.Fatalf("expected rule reverse view prompt relationship to be nil, got %+v", ruleView.Relationships.Prompt)
	}
	if len(ruleView.Relationships.Rules) != 0 {
		t.Fatalf("expected rule reverse view rules to be empty, got %+v", ruleView.Relationships.Rules)
	}
	actualRuleSkillIDs := make([]string, 0, len(ruleView.Relationships.Skills))
	for _, skill := range ruleView.Relationships.Skills {
		actualRuleSkillIDs = append(actualRuleSkillIDs, skill.ID)
	}
	if !reflect.DeepEqual(actualRuleSkillIDs, expectedSkillIDs) {
		t.Fatalf("expected rule reverse skills %+v, got %+v", expectedSkillIDs, actualRuleSkillIDs)
	}
}

func TestCatalogRelationshipService_Get_SuppressesSoftDeletedEndpoints(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	ruleRepo := newCatalogSkillRuleRelationshipRepositoryForDomainTest(t, db)
	promptRepo := newCatalogSkillPromptRelationshipRepositoryForDomainTest(t, db)
	service := newCatalogRelationshipServiceForDomainTest(t, sourceRepo, ruleRepo, promptRepo)

	skillItemID := BuildSkillCatalogItemID("repo-a/dependency-skill")
	promptItemID := BuildPromptCatalogItemID("repo-a/dependency-skill", "prompts/system.md")
	ruleItemID := BuildRuleCatalogItemID("repo-a/dependency-skill", "rules/security.md")
	syncedAt := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)

	for _, row := range []persistence.CatalogSourceRow{
		{
			ItemID:           skillItemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "dependency-skill",
			Description:      "dependency skill",
			Content:          "skill content",
			ContentHash:      buildCatalogContentHash("skill content"),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           promptItemID,
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/dependency-skill"),
			ResourcePath:     stringPointer("prompts/system.md"),
			Name:             "system.md",
			Description:      "dependency prompt",
			Content:          "prompt content",
			ContentHash:      buildCatalogContentHash("prompt content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           ruleItemID,
			Classifier:       persistence.CatalogClassifierRule,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/dependency-skill"),
			ResourcePath:     stringPointer("rules/security.md"),
			Name:             "security.md",
			Description:      "dependency rule",
			Content:          "rule content",
			ContentHash:      buildCatalogContentHash("rule content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
	} {
		mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, row)
	}

	mustSetCatalogPromptRelationshipForDomainTest(t, ctx, promptRepo, skillItemID, promptItemID, syncedAt)
	mustReplaceCatalogRuleRelationshipsForDomainTest(t, ctx, ruleRepo, skillItemID, []string{ruleItemID}, syncedAt)

	mustSoftDeleteCatalogSourceRowForDomainTest(t, ctx, sourceRepo, promptItemID)
	mustSoftDeleteCatalogSourceRowForDomainTest(t, ctx, sourceRepo, ruleItemID)

	view, err := service.Get(ctx, skillItemID)
	if err != nil {
		t.Fatalf("expected relationship get for skill to succeed, got %v", err)
	}
	if view.Relationships.Prompt != nil {
		t.Fatalf("expected deleted prompt endpoint to be suppressed, got %+v", view.Relationships.Prompt)
	}
	if len(view.Relationships.Rules) != 0 {
		t.Fatalf("expected deleted rule endpoint to be suppressed, got %+v", view.Relationships.Rules)
	}

	_, err = service.Get(ctx, promptItemID)
	if !errors.Is(err, ErrCatalogRelationshipItemNotFound) {
		t.Fatalf("expected deleted prompt view to return not found, got %v", err)
	}
}

func TestCatalogRelationshipService_Patch_ValidatesClassifierAndWriteAuthority(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	ruleRepo := newCatalogSkillRuleRelationshipRepositoryForDomainTest(t, db)
	promptRepo := newCatalogSkillPromptRelationshipRepositoryForDomainTest(t, db)
	service := newCatalogRelationshipServiceForDomainTest(t, sourceRepo, ruleRepo, promptRepo)

	skillItemID := BuildSkillCatalogItemID("repo-a/patch-target")
	promptItemID := BuildPromptCatalogItemID("repo-a/patch-target", "prompts/system.md")
	ruleItemID := BuildRuleCatalogItemID("repo-a/patch-target", "rules/security.md")
	syncedAt := time.Date(2026, time.March, 7, 12, 30, 0, 0, time.UTC)

	for _, row := range []persistence.CatalogSourceRow{
		{
			ItemID:           skillItemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "patch-target",
			Description:      "patch target skill",
			Content:          "skill content",
			ContentHash:      buildCatalogContentHash("skill content"),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           promptItemID,
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/patch-target"),
			ResourcePath:     stringPointer("prompts/system.md"),
			Name:             "system.md",
			Description:      "patch prompt",
			Content:          "prompt content",
			ContentHash:      buildCatalogContentHash("prompt content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           ruleItemID,
			Classifier:       persistence.CatalogClassifierRule,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/patch-target"),
			ResourcePath:     stringPointer("rules/security.md"),
			Name:             "security.md",
			Description:      "patch rule",
			Content:          "rule content",
			ContentHash:      buildCatalogContentHash("rule content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
	} {
		mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, row)
	}

	_, err := service.Patch(ctx, CatalogRelationshipPatchInput{
		ItemID:          promptItemID,
		PromptItemIDSet: true,
		PromptItemID:    &promptItemID,
	})
	if !errors.Is(err, ErrCatalogRelationshipReadOnlySurface) {
		t.Fatalf("expected non-skill patch to fail with read-only-surface, got %v", err)
	}

	_, err = service.Patch(ctx, CatalogRelationshipPatchInput{
		ItemID:          skillItemID,
		PromptItemIDSet: true,
		PromptItemID:    &ruleItemID,
	})
	if !errors.Is(err, ErrCatalogRelationshipValidation) {
		t.Fatalf("expected prompt classifier mismatch to fail validation, got %v", err)
	}

	_, err = service.Patch(ctx, CatalogRelationshipPatchInput{
		ItemID:      skillItemID,
		RuleItemIDs: &[]string{promptItemID},
	})
	if !errors.Is(err, ErrCatalogRelationshipValidation) {
		t.Fatalf("expected rule classifier mismatch to fail validation, got %v", err)
	}

	_, err = service.Patch(ctx, CatalogRelationshipPatchInput{
		ItemID:      skillItemID,
		RuleItemIDs: &[]string{ruleItemID, ruleItemID},
	})
	if !errors.Is(err, ErrCatalogRelationshipValidation) {
		t.Fatalf("expected duplicate rule ids to fail validation, got %v", err)
	}
}

func TestCatalogRelationshipService_Patch_EnforcesSinglePromptPerSkill(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	ruleRepo := newCatalogSkillRuleRelationshipRepositoryForDomainTest(t, db)
	promptRepo := newCatalogSkillPromptRelationshipRepositoryForDomainTest(t, db)
	service := newCatalogRelationshipServiceForDomainTest(t, sourceRepo, ruleRepo, promptRepo)

	skillItemID := BuildSkillCatalogItemID("repo-a/prompt-enforcement")
	promptAItemID := BuildPromptCatalogItemID("repo-a/prompt-enforcement", "prompts/system-a.md")
	promptBItemID := BuildPromptCatalogItemID("repo-a/prompt-enforcement", "prompts/system-b.md")
	ruleItemID := BuildRuleCatalogItemID("repo-a/prompt-enforcement", "rules/security.md")
	syncedAt := time.Date(2026, time.March, 7, 13, 0, 0, 0, time.UTC)

	for _, row := range []persistence.CatalogSourceRow{
		{
			ItemID:           skillItemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "prompt-enforcement",
			Description:      "prompt enforcement skill",
			Content:          "skill content",
			ContentHash:      buildCatalogContentHash("skill content"),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           promptAItemID,
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/prompt-enforcement"),
			ResourcePath:     stringPointer("prompts/system-a.md"),
			Name:             "system-a.md",
			Description:      "first prompt",
			Content:          "prompt a content",
			ContentHash:      buildCatalogContentHash("prompt a content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           promptBItemID,
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/prompt-enforcement"),
			ResourcePath:     stringPointer("prompts/system-b.md"),
			Name:             "system-b.md",
			Description:      "second prompt",
			Content:          "prompt b content",
			ContentHash:      buildCatalogContentHash("prompt b content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           ruleItemID,
			Classifier:       persistence.CatalogClassifierRule,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/prompt-enforcement"),
			ResourcePath:     stringPointer("rules/security.md"),
			Name:             "security.md",
			Description:      "security rule",
			Content:          "rule content",
			ContentHash:      buildCatalogContentHash("rule content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
	} {
		mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, row)
	}

	_, err := service.Patch(ctx, CatalogRelationshipPatchInput{
		ItemID:          skillItemID,
		PromptItemIDSet: true,
		PromptItemID:    &promptAItemID,
		RuleItemIDs:     &[]string{ruleItemID},
	})
	if err != nil {
		t.Fatalf("expected first relationship patch to succeed, got %v", err)
	}

	view, err := service.Patch(ctx, CatalogRelationshipPatchInput{
		ItemID:          skillItemID,
		PromptItemIDSet: true,
		PromptItemID:    &promptBItemID,
	})
	if err != nil {
		t.Fatalf("expected prompt replacement patch to succeed, got %v", err)
	}

	promptRows, err := promptRepo.ListBySkillItemID(ctx, skillItemID)
	if err != nil {
		t.Fatalf("expected prompt relationship list by skill to succeed, got %v", err)
	}
	if len(promptRows) != 1 {
		t.Fatalf("expected one prompt relationship row per skill, got %d", len(promptRows))
	}
	if promptRows[0].PromptItemID != promptBItemID {
		t.Fatalf("expected prompt relationship to be replaced with %q, got %q", promptBItemID, promptRows[0].PromptItemID)
	}

	if view.Relationships.Prompt == nil || view.Relationships.Prompt.ID != promptBItemID {
		t.Fatalf("expected prompt projection to resolve %q, got %+v", promptBItemID, view.Relationships.Prompt)
	}
	if len(view.Relationships.Rules) != 1 || view.Relationships.Rules[0].ID != ruleItemID {
		t.Fatalf("expected existing rule relationship %q to remain intact, got %+v", ruleItemID, view.Relationships.Rules)
	}
}

func TestCatalogRelationshipService_Reconcile_PrunesStaleRows(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	ruleRepo := newCatalogSkillRuleRelationshipRepositoryForDomainTest(t, db)
	promptRepo := newCatalogSkillPromptRelationshipRepositoryForDomainTest(t, db)
	service := newCatalogRelationshipServiceForDomainTest(t, sourceRepo, ruleRepo, promptRepo)

	skillItemID := BuildSkillCatalogItemID("repo-a/reconcile-skill")
	softDeletedSkillItemID := BuildSkillCatalogItemID("repo-a/reconcile-soft-deleted-skill")
	nonRuleTargetItemID := BuildSkillCatalogItemID("repo-a/reconcile-non-rule-target")
	ruleItemID := BuildRuleCatalogItemID("repo-a/reconcile-skill", "rules/keep.md")
	staleRuleItemID := BuildRuleCatalogItemID("repo-a/reconcile-skill", "rules/stale.md")
	promptItemID := BuildPromptCatalogItemID("repo-a/reconcile-skill", "prompts/keep.md")
	stalePromptItemID := BuildPromptCatalogItemID("repo-a/reconcile-skill", "prompts/stale.md")
	syncedAt := time.Date(2026, time.March, 7, 13, 30, 0, 0, time.UTC)

	for _, row := range []persistence.CatalogSourceRow{
		{
			ItemID:           skillItemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "reconcile-skill",
			Description:      "active skill",
			Content:          "skill content",
			ContentHash:      buildCatalogContentHash("skill content"),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           softDeletedSkillItemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "reconcile-soft-deleted-skill",
			Description:      "soft deleted skill",
			Content:          "soft deleted skill content",
			ContentHash:      buildCatalogContentHash("soft deleted skill content"),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           nonRuleTargetItemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "reconcile-non-rule-target",
			Description:      "non-rule target",
			Content:          "non-rule target content",
			ContentHash:      buildCatalogContentHash("non-rule target content"),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           ruleItemID,
			Classifier:       persistence.CatalogClassifierRule,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/reconcile-skill"),
			ResourcePath:     stringPointer("rules/keep.md"),
			Name:             "keep.md",
			Description:      "keep rule",
			Content:          "keep rule content",
			ContentHash:      buildCatalogContentHash("keep rule content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           staleRuleItemID,
			Classifier:       persistence.CatalogClassifierRule,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/reconcile-skill"),
			ResourcePath:     stringPointer("rules/stale.md"),
			Name:             "stale.md",
			Description:      "stale rule",
			Content:          "stale rule content",
			ContentHash:      buildCatalogContentHash("stale rule content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           promptItemID,
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/reconcile-skill"),
			ResourcePath:     stringPointer("prompts/keep.md"),
			Name:             "keep.md",
			Description:      "keep prompt",
			Content:          "keep prompt content",
			ContentHash:      buildCatalogContentHash("keep prompt content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           stalePromptItemID,
			Classifier:       persistence.CatalogClassifierPrompt,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       stringPointer("repo-a"),
			ParentSkillID:    stringPointer("repo-a/reconcile-skill"),
			ResourcePath:     stringPointer("prompts/stale.md"),
			Name:             "stale.md",
			Description:      "stale prompt",
			Content:          "stale prompt content",
			ContentHash:      buildCatalogContentHash("stale prompt content"),
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
	} {
		mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, row)
	}

	mustReplaceCatalogRuleRelationshipsForDomainTest(
		t,
		ctx,
		ruleRepo,
		skillItemID,
		[]string{ruleItemID, staleRuleItemID, nonRuleTargetItemID},
		syncedAt,
	)
	mustSetCatalogPromptRelationshipForDomainTest(t, ctx, promptRepo, skillItemID, stalePromptItemID, syncedAt)
	mustSetCatalogPromptRelationshipForDomainTest(
		t,
		ctx,
		promptRepo,
		softDeletedSkillItemID,
		promptItemID,
		syncedAt,
	)

	mustSoftDeleteCatalogSourceRowForDomainTest(t, ctx, sourceRepo, softDeletedSkillItemID)
	mustSoftDeleteCatalogSourceRowForDomainTest(t, ctx, sourceRepo, staleRuleItemID)
	mustSoftDeleteCatalogSourceRowForDomainTest(t, ctx, sourceRepo, stalePromptItemID)

	report, err := service.Reconcile(ctx)
	if err != nil {
		t.Fatalf("expected relationship reconciliation to succeed, got %v", err)
	}

	if report.SkillRuleRowsScanned != 3 {
		t.Fatalf("expected 3 scanned skill-rule rows, got %d", report.SkillRuleRowsScanned)
	}
	if report.SkillRuleRowsPruned != 2 {
		t.Fatalf("expected 2 pruned skill-rule rows, got %d", report.SkillRuleRowsPruned)
	}
	if report.SkillPromptRowsScanned != 2 {
		t.Fatalf("expected 2 scanned skill-prompt rows, got %d", report.SkillPromptRowsScanned)
	}
	if report.SkillPromptRowsPruned != 2 {
		t.Fatalf("expected 2 pruned skill-prompt rows, got %d", report.SkillPromptRowsPruned)
	}

	remainingRuleRows, err := ruleRepo.ListBySkillItemID(ctx, skillItemID)
	if err != nil {
		t.Fatalf("expected remaining rule relationship list to succeed, got %v", err)
	}
	if len(remainingRuleRows) != 1 || remainingRuleRows[0].RuleItemID != ruleItemID {
		t.Fatalf("expected one remaining keep rule relationship %q, got %+v", ruleItemID, remainingRuleRows)
	}

	remainingPromptRows, err := promptRepo.ListBySkillItemID(ctx, skillItemID)
	if err != nil {
		t.Fatalf("expected remaining prompt relationship list for active skill to succeed, got %v", err)
	}
	if len(remainingPromptRows) != 0 {
		t.Fatalf("expected stale prompt relationship rows to be pruned for active skill, got %+v", remainingPromptRows)
	}

	softDeletedSkillPromptRows, err := promptRepo.ListBySkillItemID(ctx, softDeletedSkillItemID)
	if err != nil {
		t.Fatalf("expected remaining prompt relationship list for soft-deleted skill to succeed, got %v", err)
	}
	if len(softDeletedSkillPromptRows) != 0 {
		t.Fatalf(
			"expected prompt relationship rows for soft-deleted skill to be pruned, got %+v",
			softDeletedSkillPromptRows,
		)
	}
}

func newCatalogRelationshipServiceForDomainTest(
	t *testing.T,
	sourceRepo *persistence.CatalogSourceRepository,
	ruleRepo *persistence.CatalogSkillRuleRelationshipRepository,
	promptRepo *persistence.CatalogSkillPromptRelationshipRepository,
) *CatalogRelationshipService {
	t.Helper()

	service, err := NewCatalogRelationshipService(
		sourceRepo,
		ruleRepo,
		promptRepo,
		CatalogRelationshipServiceOptions{},
	)
	if err != nil {
		t.Fatalf("expected relationship service creation to succeed, got %v", err)
	}
	return service
}

func newCatalogSkillRuleRelationshipRepositoryForDomainTest(
	t *testing.T,
	db *sql.DB,
) *persistence.CatalogSkillRuleRelationshipRepository {
	t.Helper()

	repo, err := persistence.NewCatalogSkillRuleRelationshipRepository(db)
	if err != nil {
		t.Fatalf("expected rule relationship repository creation to succeed, got %v", err)
	}
	return repo
}

func newCatalogSkillPromptRelationshipRepositoryForDomainTest(
	t *testing.T,
	db *sql.DB,
) *persistence.CatalogSkillPromptRelationshipRepository {
	t.Helper()

	repo, err := persistence.NewCatalogSkillPromptRelationshipRepository(db)
	if err != nil {
		t.Fatalf("expected prompt relationship repository creation to succeed, got %v", err)
	}
	return repo
}

func mustSetCatalogPromptRelationshipForDomainTest(
	t *testing.T,
	ctx context.Context,
	repo *persistence.CatalogSkillPromptRelationshipRepository,
	skillItemID string,
	promptItemID string,
	updatedAt time.Time,
) {
	t.Helper()

	if err := repo.SetForSkillItemID(ctx, skillItemID, promptItemID, updatedAt, nil); err != nil {
		t.Fatalf("expected prompt relationship set to succeed, got %v", err)
	}
}

func mustReplaceCatalogRuleRelationshipsForDomainTest(
	t *testing.T,
	ctx context.Context,
	repo *persistence.CatalogSkillRuleRelationshipRepository,
	skillItemID string,
	ruleItemIDs []string,
	updatedAt time.Time,
) {
	t.Helper()

	if err := repo.ReplaceForSkillItemID(ctx, skillItemID, ruleItemIDs, updatedAt, nil); err != nil {
		t.Fatalf("expected rule relationship replace to succeed, got %v", err)
	}
}

func mustSoftDeleteCatalogSourceRowForDomainTest(
	t *testing.T,
	ctx context.Context,
	repo *persistence.CatalogSourceRepository,
	itemID string,
) {
	t.Helper()

	if _, err := repo.SoftDeleteByItemID(ctx, itemID, time.Date(2026, time.March, 7, 14, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("expected source soft delete for %q to succeed, got %v", itemID, err)
	}
}
