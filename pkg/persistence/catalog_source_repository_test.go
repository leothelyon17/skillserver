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
