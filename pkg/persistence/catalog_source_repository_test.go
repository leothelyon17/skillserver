package persistence

import (
	"errors"
	"testing"
	"time"
)

func TestNewCatalogSourceRepository_WithNilExecutor_ReturnsError(t *testing.T) {
	_, err := NewCatalogSourceRepository(nil)
	if err == nil {
		t.Fatalf("expected nil executor error, got nil")
	}
}

func TestCatalogSourceRepository_UpsertAndGetByItemID_RoundTripsRow(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogSourceRepositoryForTest(t, db)

	lastSyncedAt := time.Date(2026, time.March, 4, 14, 30, 0, 0, time.UTC)
	deletedAt := time.Date(2026, time.March, 4, 14, 45, 0, 0, time.UTC)
	sourceRepo := "demo-repo"
	parentSkillID := "demo-repo/planner"
	resourcePath := "prompts/system.md"

	expected := CatalogSourceRow{
		ItemID:           "prompt:demo-repo/planner:prompts/system.md",
		Classifier:       CatalogClassifierPrompt,
		SourceType:       CatalogSourceTypeGit,
		SourceRepo:       &sourceRepo,
		ParentSkillID:    &parentSkillID,
		ResourcePath:     &resourcePath,
		Name:             "system.md",
		Description:      "System prompt",
		Content:          "You are deterministic.",
		ContentHash:      "sha256:abc123",
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
		DeletedAt:        &deletedAt,
	}

	mustUpsertCatalogSourceRow(t, ctx, repo, expected)

	actual, err := repo.GetByItemID(ctx, expected.ItemID)
	if err != nil {
		t.Fatalf("expected source row lookup to succeed, got %v", err)
	}

	assertCatalogSourceRowEqual(t, expected, actual)
}

func TestCatalogSourceRepository_UpsertExistingRow_UpdatesMutableColumnsAndPreservesOverlayState(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	overlayRepo := newCatalogMetadataOverlayRepositoryForTest(t, db)

	itemID := "skill:planner"
	firstLastSyncedAt := time.Date(2026, time.March, 4, 15, 0, 0, 0, time.UTC)
	secondLastSyncedAt := time.Date(2026, time.March, 4, 16, 0, 0, 0, time.UTC)
	overlayUpdatedAt := time.Date(2026, time.March, 4, 15, 30, 0, 0, time.UTC)

	mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "planner",
		Description:      "Original description",
		Content:          "Original content",
		ContentHash:      "sha256:original",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     firstLastSyncedAt,
	})

	if err := overlayRepo.Upsert(ctx, CatalogMetadataOverlayRow{
		ItemID:              itemID,
		DisplayNameOverride: stringPointer("Planner Override"),
		CustomMetadata: map[string]any{
			"owner": "platform",
		},
		Labels:    []string{"catalog", "metadata"},
		UpdatedAt: overlayUpdatedAt,
		UpdatedBy: stringPointer("wp-003-test"),
	}); err != nil {
		t.Fatalf("expected overlay upsert to succeed, got %v", err)
	}

	mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "planner-renamed",
		Description:      "Updated description",
		Content:          "Updated content",
		ContentHash:      "sha256:updated",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     secondLastSyncedAt,
	})

	updatedSource, err := sourceRepo.GetByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected updated source row lookup to succeed, got %v", err)
	}
	if updatedSource.Name != "planner-renamed" {
		t.Fatalf("expected source name to be updated, got %q", updatedSource.Name)
	}
	if updatedSource.ContentHash != "sha256:updated" {
		t.Fatalf("expected source hash to be updated, got %q", updatedSource.ContentHash)
	}
	if !updatedSource.LastSyncedAt.Equal(secondLastSyncedAt) {
		t.Fatalf("expected source last_synced_at %s, got %s", secondLastSyncedAt, updatedSource.LastSyncedAt)
	}

	overlayRow, err := overlayRepo.GetByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected overlay row lookup to succeed, got %v", err)
	}
	if overlayRow.DisplayNameOverride == nil || *overlayRow.DisplayNameOverride != "Planner Override" {
		t.Fatalf("expected overlay display name override to remain unchanged, got %+v", overlayRow.DisplayNameOverride)
	}
	if gotOwner, ok := overlayRow.CustomMetadata["owner"].(string); !ok || gotOwner != "platform" {
		t.Fatalf("expected overlay custom metadata owner to remain unchanged, got %+v", overlayRow.CustomMetadata)
	}
	if len(overlayRow.Labels) != 2 || overlayRow.Labels[0] != "catalog" || overlayRow.Labels[1] != "metadata" {
		t.Fatalf("expected overlay labels to remain unchanged, got %+v", overlayRow.Labels)
	}
	if !overlayRow.UpdatedAt.Equal(overlayUpdatedAt) {
		t.Fatalf("expected overlay updated_at %s, got %s", overlayUpdatedAt, overlayRow.UpdatedAt)
	}
}

func TestCatalogSourceRepository_List_WithDeterministicOrderingAndFilters(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogSourceRepositoryForTest(t, db)

	repoA := "repo-a"
	repoB := "repo-b"

	rows := []CatalogSourceRow{
		{
			ItemID:           "skill:alpha",
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             "alpha",
			Description:      "alpha skill",
			Content:          "alpha content",
			ContentHash:      "sha256:alpha",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 4, 11, 0, 0, 0, time.UTC),
		},
		{
			ItemID:           "prompt:alpha:prompts/system.md",
			Classifier:       CatalogClassifierPrompt,
			SourceType:       CatalogSourceTypeGit,
			SourceRepo:       &repoA,
			ParentSkillID:    stringPointer("alpha"),
			ResourcePath:     stringPointer("prompts/system.md"),
			Name:             "system.md",
			Description:      "alpha prompt",
			Content:          "alpha prompt content",
			ContentHash:      "sha256:alpha-prompt",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 4, 12, 0, 0, 0, time.UTC),
		},
		{
			ItemID:           "skill:repo-b/planner",
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeGit,
			SourceRepo:       &repoB,
			Name:             "planner",
			Description:      "repo-b skill",
			Content:          "planner content",
			ContentHash:      "sha256:planner",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 4, 13, 0, 0, 0, time.UTC),
		},
	}

	for _, row := range rows {
		mustUpsertCatalogSourceRow(t, ctx, repo, row)
	}

	deleted, err := repo.SoftDeleteByItemID(ctx, "skill:repo-b/planner", time.Date(2026, time.March, 4, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected soft-delete to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected soft-delete to affect one row")
	}

	visibleRows, err := repo.List(ctx, CatalogSourceListFilter{})
	if err != nil {
		t.Fatalf("expected source list query to succeed, got %v", err)
	}
	if len(visibleRows) != 2 {
		t.Fatalf("expected 2 visible rows, got %d", len(visibleRows))
	}
	if visibleRows[0].ItemID != "prompt:alpha:prompts/system.md" || visibleRows[1].ItemID != "skill:alpha" {
		t.Fatalf("expected deterministic ordering by item_id, got %q then %q", visibleRows[0].ItemID, visibleRows[1].ItemID)
	}

	visibleGitFromRepoA, err := repo.ListBySource(ctx, CatalogSourceTypeGit, &repoA, false)
	if err != nil {
		t.Fatalf("expected source+repo filtered list query to succeed, got %v", err)
	}
	if len(visibleGitFromRepoA) != 1 || visibleGitFromRepoA[0].ItemID != "prompt:alpha:prompts/system.md" {
		t.Fatalf("expected one visible git row for repo-a, got %+v", visibleGitFromRepoA)
	}

	allBySubset, err := repo.ListByItemIDs(ctx, []string{"skill:repo-b/planner", "skill:alpha"}, true)
	if err != nil {
		t.Fatalf("expected subset list query to succeed, got %v", err)
	}
	if len(allBySubset) != 2 {
		t.Fatalf("expected 2 rows in subset list, got %d", len(allBySubset))
	}
	if allBySubset[0].ItemID != "skill:alpha" || allBySubset[1].ItemID != "skill:repo-b/planner" {
		t.Fatalf("expected subset ordering by item_id, got %+v", allBySubset)
	}
}

func TestCatalogSourceRepository_UpsertAndList_WithRuleClassifier_RoundTripsAndFilters(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogSourceRepositoryForTest(t, db)

	sourceRepo := "repo-a"
	lastSyncedAt := time.Date(2026, time.March, 8, 1, 0, 0, 0, time.UTC)

	rows := []CatalogSourceRow{
		{
			ItemID:           "skill:repo-a/planner",
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeGit,
			SourceRepo:       &sourceRepo,
			Name:             "planner",
			Description:      "planner skill",
			Content:          "skill content",
			ContentHash:      "sha256:skill",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           "prompt:repo-a/planner:prompts/system.md",
			Classifier:       CatalogClassifierPrompt,
			SourceType:       CatalogSourceTypeGit,
			SourceRepo:       &sourceRepo,
			ParentSkillID:    stringPointer("repo-a/planner"),
			ResourcePath:     stringPointer("prompts/system.md"),
			Name:             "system.md",
			Description:      "planner system prompt",
			Content:          "prompt content",
			ContentHash:      "sha256:prompt",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
		{
			ItemID:           "rule:repo-a/planner:rules/agents.md",
			Classifier:       CatalogClassifierRule,
			SourceType:       CatalogSourceTypeGit,
			SourceRepo:       &sourceRepo,
			ParentSkillID:    stringPointer("repo-a/planner"),
			ResourcePath:     stringPointer("rules/agents.md"),
			Name:             "agents.md",
			Description:      "planner rule file",
			Content:          "rule content",
			ContentHash:      "sha256:rule",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     lastSyncedAt,
		},
	}

	for _, row := range rows {
		mustUpsertCatalogSourceRow(t, ctx, repo, row)
	}

	ruleItem, err := repo.GetByItemID(ctx, "rule:repo-a/planner:rules/agents.md")
	if err != nil {
		t.Fatalf("expected rule source row lookup to succeed, got %v", err)
	}
	assertCatalogSourceRowEqual(t, rows[2], ruleItem)

	ruleClassifier := CatalogClassifierRule
	ruleRows, err := repo.List(ctx, CatalogSourceListFilter{Classifier: &ruleClassifier})
	if err != nil {
		t.Fatalf("expected rule classifier list query to succeed, got %v", err)
	}
	if len(ruleRows) != 1 {
		t.Fatalf("expected exactly one rule-classified row, got %d", len(ruleRows))
	}
	if ruleRows[0].ItemID != "rule:repo-a/planner:rules/agents.md" {
		t.Fatalf("expected rule-classified row item_id to match, got %q", ruleRows[0].ItemID)
	}
	if ruleRows[0].Classifier != CatalogClassifierRule {
		t.Fatalf("expected rule-classified row classifier %q, got %q", CatalogClassifierRule, ruleRows[0].Classifier)
	}
}

func TestCatalogSourceRepository_RuleRowLifecycle_SoftDeleteAndRestorePreservesClassifierFiltering(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogSourceRepositoryForTest(t, db)

	sourceRepo := "repo-a"
	parentSkillID := "repo-a/planner"
	resourcePath := "rules/agents.md"
	itemID := "rule:repo-a/planner:rules/agents.md"
	lastSyncedAt := time.Date(2026, time.March, 8, 1, 15, 0, 0, time.UTC)

	mustUpsertCatalogSourceRow(t, ctx, repo, CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       CatalogClassifierRule,
		SourceType:       CatalogSourceTypeGit,
		SourceRepo:       &sourceRepo,
		ParentSkillID:    &parentSkillID,
		ResourcePath:     &resourcePath,
		Name:             "agents.md",
		Description:      "project contributor rules",
		Content:          "follow contributor guardrails",
		ContentHash:      "sha256:rule-lifecycle",
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
	})

	ruleClassifier := CatalogClassifierRule
	initialRows, err := repo.List(ctx, CatalogSourceListFilter{Classifier: &ruleClassifier})
	if err != nil {
		t.Fatalf("expected initial rule classifier list query to succeed, got %v", err)
	}
	if len(initialRows) != 1 || initialRows[0].ItemID != itemID {
		t.Fatalf("expected one visible rule row before delete, got %+v", initialRows)
	}

	tombstoneAt := time.Date(2026, time.March, 8, 1, 30, 0, 0, time.UTC)
	deleted, err := repo.DeleteByItemID(ctx, itemID, tombstoneAt)
	if err != nil {
		t.Fatalf("expected rule row soft-delete to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected rule row soft-delete to affect one row")
	}

	visibleAfterDelete, err := repo.List(ctx, CatalogSourceListFilter{Classifier: &ruleClassifier})
	if err != nil {
		t.Fatalf("expected visible rule list query after delete to succeed, got %v", err)
	}
	if len(visibleAfterDelete) != 0 {
		t.Fatalf("expected no visible rule rows after soft-delete, got %+v", visibleAfterDelete)
	}

	includingDeleted, err := repo.List(ctx, CatalogSourceListFilter{
		Classifier:     &ruleClassifier,
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("expected include-deleted rule list query to succeed, got %v", err)
	}
	if len(includingDeleted) != 1 || includingDeleted[0].ItemID != itemID {
		t.Fatalf("expected one include-deleted rule row, got %+v", includingDeleted)
	}
	if includingDeleted[0].DeletedAt == nil || !includingDeleted[0].DeletedAt.Equal(tombstoneAt) {
		t.Fatalf("expected include-deleted row tombstone %s, got %+v", tombstoneAt, includingDeleted[0].DeletedAt)
	}

	restored, err := repo.RestoreByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected rule row restore to succeed, got %v", err)
	}
	if !restored {
		t.Fatalf("expected rule row restore to affect one row")
	}

	visibleAfterRestore, err := repo.List(ctx, CatalogSourceListFilter{Classifier: &ruleClassifier})
	if err != nil {
		t.Fatalf("expected visible rule list query after restore to succeed, got %v", err)
	}
	if len(visibleAfterRestore) != 1 || visibleAfterRestore[0].ItemID != itemID {
		t.Fatalf("expected one visible rule row after restore, got %+v", visibleAfterRestore)
	}
	if visibleAfterRestore[0].DeletedAt != nil {
		t.Fatalf("expected restored rule row deleted_at=nil, got %+v", visibleAfterRestore[0].DeletedAt)
	}
}

func TestCatalogSourceRepository_SoftDeleteAndRestoreByItemID(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogSourceRepositoryForTest(t, db)

	itemID := "skill:restore-me"
	mustUpsertCatalogSourceRow(t, ctx, repo, CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "restore-me",
		Description:      "restore workflow",
		Content:          "content",
		ContentHash:      "sha256:restore",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 17, 0, 0, 0, time.UTC),
	})

	tombstoneAt := time.Date(2026, time.March, 4, 17, 30, 0, 0, time.UTC)
	deleted, err := repo.DeleteByItemID(ctx, itemID, tombstoneAt)
	if err != nil {
		t.Fatalf("expected source delete to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected source delete to affect one row")
	}

	softDeletedRow, err := repo.GetByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected deleted source row lookup to succeed, got %v", err)
	}
	if softDeletedRow.DeletedAt == nil || !softDeletedRow.DeletedAt.Equal(tombstoneAt) {
		t.Fatalf("expected deleted_at timestamp %s, got %+v", tombstoneAt, softDeletedRow.DeletedAt)
	}

	visibleRows, err := repo.List(ctx, CatalogSourceListFilter{})
	if err != nil {
		t.Fatalf("expected visible source list query to succeed, got %v", err)
	}
	if len(visibleRows) != 0 {
		t.Fatalf("expected deleted row to be excluded by default, got %+v", visibleRows)
	}

	restored, err := repo.RestoreByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected source restore to succeed, got %v", err)
	}
	if !restored {
		t.Fatalf("expected source restore to affect one row")
	}

	restoredRow, err := repo.GetByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected restored source row lookup to succeed, got %v", err)
	}
	if restoredRow.DeletedAt != nil {
		t.Fatalf("expected deleted_at to be nil after restore, got %+v", restoredRow.DeletedAt)
	}
}

func TestCatalogSourceRepository_List_WithCursorPaginationAndLimit(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogSourceRepositoryForTest(t, db)

	repoA := "repo-a"
	sourceRows := []CatalogSourceRow{
		{
			ItemID:           "skill:alpha",
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             "alpha",
			Description:      "alpha skill",
			Content:          "alpha content",
			ContentHash:      "sha256:alpha",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 9, 9, 0, 0, 0, time.UTC),
		},
		{
			ItemID:           "skill:beta",
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             "beta",
			Description:      "beta skill",
			Content:          "beta content",
			ContentHash:      "sha256:beta",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 9, 9, 5, 0, 0, time.UTC),
		},
		{
			ItemID:           "skill:delta",
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeGit,
			SourceRepo:       &repoA,
			Name:             "delta",
			Description:      "delta skill",
			Content:          "delta content",
			ContentHash:      "sha256:delta",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 9, 9, 10, 0, 0, time.UTC),
		},
		{
			ItemID:           "skill:gamma",
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeGit,
			SourceRepo:       &repoA,
			Name:             "gamma",
			Description:      "gamma skill",
			Content:          "gamma content",
			ContentHash:      "sha256:gamma",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 9, 9, 15, 0, 0, time.UTC),
		},
		{
			ItemID:           "skill:omega",
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             "omega",
			Description:      "omega skill",
			Content:          "omega content",
			ContentHash:      "sha256:omega",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 9, 9, 20, 0, 0, time.UTC),
		},
	}

	for _, row := range sourceRows {
		mustUpsertCatalogSourceRow(t, ctx, repo, row)
	}

	unpaginatedRows, err := repo.List(ctx, CatalogSourceListFilter{})
	if err != nil {
		t.Fatalf("expected unpaginated source list query to succeed, got %v", err)
	}
	if len(unpaginatedRows) != 5 {
		t.Fatalf("expected 5 unpaginated rows, got %d", len(unpaginatedRows))
	}
	assertStringSliceEqual(
		t,
		catalogSourceRowItemIDs(unpaginatedRows),
		[]string{"skill:alpha", "skill:beta", "skill:delta", "skill:gamma", "skill:omega"},
		"unpaginated item_ids",
	)

	firstPage, err := repo.List(ctx, CatalogSourceListFilter{Limit: 2})
	if err != nil {
		t.Fatalf("expected first paginated source list query to succeed, got %v", err)
	}
	assertStringSliceEqual(
		t,
		catalogSourceRowItemIDs(firstPage),
		[]string{"skill:alpha", "skill:beta"},
		"first page item_ids",
	)

	secondPage, err := repo.List(ctx, CatalogSourceListFilter{Cursor: "skill:beta", Limit: 2})
	if err != nil {
		t.Fatalf("expected second paginated source list query to succeed, got %v", err)
	}
	assertStringSliceEqual(
		t,
		catalogSourceRowItemIDs(secondPage),
		[]string{"skill:delta", "skill:gamma"},
		"second page item_ids",
	)

	gitSourceType := CatalogSourceTypeGit
	filteredFirstPage, err := repo.List(ctx, CatalogSourceListFilter{
		SourceType: &gitSourceType,
		SourceRepo: &repoA,
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("expected filtered paginated source list query to succeed, got %v", err)
	}
	assertStringSliceEqual(
		t,
		catalogSourceRowItemIDs(filteredFirstPage),
		[]string{"skill:delta"},
		"filtered first page item_ids",
	)

	filteredSecondPage, err := repo.List(ctx, CatalogSourceListFilter{
		SourceType: &gitSourceType,
		SourceRepo: &repoA,
		Cursor:     "skill:delta",
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("expected filtered second paginated source list query to succeed, got %v", err)
	}
	assertStringSliceEqual(
		t,
		catalogSourceRowItemIDs(filteredSecondPage),
		[]string{"skill:gamma"},
		"filtered second page item_ids",
	)
}

func TestCatalogSourceRepository_GetByItemID_MissingRow_ReturnsNotFound(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogSourceRepositoryForTest(t, db)

	_, err := repo.GetByItemID(ctx, "skill:does-not-exist")
	if !errors.Is(err, ErrCatalogSourceNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestCatalogSourceRepository_List_InvalidFilter_ReturnsError(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogSourceRepositoryForTest(t, db)

	_, err := repo.List(ctx, CatalogSourceListFilter{
		ItemID:  "skill:a",
		ItemIDs: []string{"skill:b"},
	})
	if err == nil {
		t.Fatalf("expected invalid filter error, got nil")
	}
}

func TestCatalogSourceRepository_List_WithNegativeLimit_ReturnsError(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogSourceRepositoryForTest(t, db)

	_, err := repo.List(ctx, CatalogSourceListFilter{Limit: -1})
	if err == nil {
		t.Fatalf("expected negative limit error, got nil")
	}
}

func assertCatalogSourceRowEqual(t *testing.T, expected, actual CatalogSourceRow) {
	t.Helper()

	if expected.ItemID != actual.ItemID {
		t.Fatalf("expected item_id %q, got %q", expected.ItemID, actual.ItemID)
	}
	if expected.Classifier != actual.Classifier {
		t.Fatalf("expected classifier %q, got %q", expected.Classifier, actual.Classifier)
	}
	if expected.SourceType != actual.SourceType {
		t.Fatalf("expected source_type %q, got %q", expected.SourceType, actual.SourceType)
	}
	assertOptionalStringEqual(t, expected.SourceRepo, actual.SourceRepo, "source_repo")
	assertOptionalStringEqual(t, expected.ParentSkillID, actual.ParentSkillID, "parent_skill_id")
	assertOptionalStringEqual(t, expected.ResourcePath, actual.ResourcePath, "resource_path")
	if expected.Name != actual.Name {
		t.Fatalf("expected name %q, got %q", expected.Name, actual.Name)
	}
	if expected.Description != actual.Description {
		t.Fatalf("expected description %q, got %q", expected.Description, actual.Description)
	}
	if expected.Content != actual.Content {
		t.Fatalf("expected content %q, got %q", expected.Content, actual.Content)
	}
	if expected.ContentHash != actual.ContentHash {
		t.Fatalf("expected content_hash %q, got %q", expected.ContentHash, actual.ContentHash)
	}
	if expected.ContentWritable != actual.ContentWritable {
		t.Fatalf("expected content_writable %t, got %t", expected.ContentWritable, actual.ContentWritable)
	}
	if expected.MetadataWritable != actual.MetadataWritable {
		t.Fatalf("expected metadata_writable %t, got %t", expected.MetadataWritable, actual.MetadataWritable)
	}
	if !expected.LastSyncedAt.Equal(actual.LastSyncedAt) {
		t.Fatalf("expected last_synced_at %s, got %s", expected.LastSyncedAt, actual.LastSyncedAt)
	}
	if expected.DeletedAt == nil && actual.DeletedAt != nil {
		t.Fatalf("expected deleted_at nil, got %s", *actual.DeletedAt)
	}
	if expected.DeletedAt != nil && (actual.DeletedAt == nil || !expected.DeletedAt.Equal(*actual.DeletedAt)) {
		t.Fatalf("expected deleted_at %v, got %v", expected.DeletedAt, actual.DeletedAt)
	}
}

func assertOptionalStringEqual(t *testing.T, expected, actual *string, fieldName string) {
	t.Helper()

	switch {
	case expected == nil && actual == nil:
		return
	case expected == nil && actual != nil:
		t.Fatalf("expected %s nil, got %q", fieldName, *actual)
	case expected != nil && actual == nil:
		t.Fatalf("expected %s %q, got nil", fieldName, *expected)
	case *expected != *actual:
		t.Fatalf("expected %s %q, got %q", fieldName, *expected, *actual)
	}
}

func catalogSourceRowItemIDs(rows []CatalogSourceRow) []string {
	itemIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		itemIDs = append(itemIDs, row.ItemID)
	}

	return itemIDs
}
