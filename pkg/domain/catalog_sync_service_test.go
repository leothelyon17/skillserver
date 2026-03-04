package domain

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

func TestCatalogSyncService_SyncAll_ReconcilesCreateUpdateDeleteAndRestore(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)

	repoA := "repo-a"
	repoB := "repo-b"
	lastSyncedAt := time.Date(2026, time.March, 4, 10, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2026, time.March, 4, 10, 30, 0, 0, time.UTC)
	syncAt := time.Date(2026, time.March, 4, 12, 0, 0, 0, time.UTC)

	localPlannerID := BuildSkillCatalogItemID("local-planner")
	localStableID := BuildSkillCatalogItemID("local-stable")
	legacyPromptID := BuildPromptCatalogItemID("repo-a/planner", "imports/prompts/legacy.md")
	ghostID := BuildSkillCatalogItemID("repo-b/ghost")
	reviveID := BuildSkillCatalogItemID("repo-a/revive")

	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           localPlannerID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "local-planner",
		Description:      "old local planner",
		Content:          "old planner content",
		ContentHash:      buildCatalogContentHash("old planner content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           localStableID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "local-stable",
		Description:      "stable local skill",
		Content:          "stable local content",
		ContentHash:      buildCatalogContentHash("stable local content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           legacyPromptID,
		Classifier:       persistence.CatalogClassifierPrompt,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoA,
		ParentSkillID:    stringPointer("repo-a/planner"),
		ResourcePath:     stringPointer("imports/prompts/legacy.md"),
		Name:             "legacy.md",
		Description:      "legacy prompt",
		Content:          "legacy content",
		ContentHash:      buildCatalogContentHash("legacy content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           ghostID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoB,
		Name:             "ghost",
		Description:      "already deleted",
		Content:          "ghost",
		ContentHash:      buildCatalogContentHash("ghost"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
		DeletedAt:        &deletedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           reviveID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoA,
		Name:             "revive",
		Description:      "revive me",
		Content:          "stale content",
		ContentHash:      buildCatalogContentHash("stale content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
		DeletedAt:        &deletedAt,
	})

	overlayUpdatedAt := time.Date(2026, time.March, 4, 11, 0, 0, 0, time.UTC)
	if err := overlayRepo.Upsert(ctx, persistence.CatalogMetadataOverlayRow{
		ItemID:              localPlannerID,
		DisplayNameOverride: stringPointer("Planner Override"),
		CustomMetadata: map[string]any{
			"owner": "platform",
		},
		Labels:    []string{"catalog", "metadata"},
		UpdatedAt: overlayUpdatedAt,
	}); err != nil {
		t.Fatalf("expected overlay upsert to succeed, got %v", err)
	}

	discovered := []CatalogItem{
		{
			ID:          localPlannerID,
			Classifier:  CatalogClassifierSkill,
			Name:        "local-planner",
			Description: "new local planner",
			Content:     "new planner content",
			ReadOnly:    false,
		},
		{
			ID:          localStableID,
			Classifier:  CatalogClassifierSkill,
			Name:        "local-stable",
			Description: "stable local skill",
			Content:     "stable local content",
			ReadOnly:    false,
		},
		{
			ID:          reviveID,
			Classifier:  CatalogClassifierSkill,
			Name:        "revive",
			Description: "revived skill",
			Content:     "fresh revived content",
			ReadOnly:    true,
		},
	}

	var logBuffer bytes.Buffer
	service := newCatalogSyncServiceForDomainTest(t, sourceRepo, &logBuffer, syncAt)

	if err := service.SyncAll(discovered); err != nil {
		t.Fatalf("expected full sync to succeed, got %v", err)
	}

	localPlannerRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, localPlannerID)
	if localPlannerRow.Description != "new local planner" {
		t.Fatalf("expected local planner description to be updated, got %q", localPlannerRow.Description)
	}
	if localPlannerRow.ContentHash != buildCatalogContentHash("new planner content") {
		t.Fatalf("expected local planner content hash to be updated, got %q", localPlannerRow.ContentHash)
	}
	if !localPlannerRow.LastSyncedAt.Equal(syncAt) {
		t.Fatalf("expected local planner last_synced_at %s, got %s", syncAt, localPlannerRow.LastSyncedAt)
	}
	if localPlannerRow.DeletedAt != nil {
		t.Fatalf("expected local planner deleted_at to remain nil, got %v", localPlannerRow.DeletedAt)
	}

	localStableRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, localStableID)
	if !localStableRow.LastSyncedAt.Equal(lastSyncedAt) {
		t.Fatalf("expected unchanged local row last_synced_at %s, got %s", lastSyncedAt, localStableRow.LastSyncedAt)
	}

	legacyPromptRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, legacyPromptID)
	if legacyPromptRow.DeletedAt == nil || !legacyPromptRow.DeletedAt.Equal(syncAt) {
		t.Fatalf("expected missing repo prompt to be tombstoned at %s, got %v", syncAt, legacyPromptRow.DeletedAt)
	}

	ghostRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, ghostID)
	if ghostRow.DeletedAt == nil || !ghostRow.DeletedAt.Equal(deletedAt) {
		t.Fatalf("expected already-deleted row tombstone to remain unchanged, got %v", ghostRow.DeletedAt)
	}

	revivedRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, reviveID)
	if revivedRow.DeletedAt != nil {
		t.Fatalf("expected revived row deleted_at to be nil, got %v", revivedRow.DeletedAt)
	}
	if revivedRow.ContentHash != buildCatalogContentHash("fresh revived content") {
		t.Fatalf("expected revived row content hash to be updated, got %q", revivedRow.ContentHash)
	}

	overlayRow, err := overlayRepo.GetByItemID(ctx, localPlannerID)
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

	assertCatalogSyncLogCounts(t, logBuffer.String(), map[string]string{
		"mode":       "full",
		"repo":       "*",
		"discovered": "3",
		"existing":   "5",
		"upserted":   "2",
		"tombstoned": "1",
		"restored":   "1",
		"unchanged":  "2",
	})
}

func TestCatalogSyncService_SyncRepo_UpdatesOnlyTargetRepoAndPreservesOverlays(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)

	repoA := "repo-a"
	repoB := "repo-b"
	lastSyncedAt := time.Date(2026, time.March, 4, 13, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2026, time.March, 4, 13, 30, 0, 0, time.UTC)
	syncAt := time.Date(2026, time.March, 4, 14, 0, 0, 0, time.UTC)

	alphaID := BuildSkillCatalogItemID("repo-a/alpha")
	stableID := BuildSkillCatalogItemID("repo-a/stable")
	legacyPromptID := BuildPromptCatalogItemID("repo-a/stable", "imports/prompts/legacy.md")
	reviveID := BuildSkillCatalogItemID("repo-a/revive")
	repoBID := BuildSkillCatalogItemID("repo-b/untouched")
	localID := BuildSkillCatalogItemID("local-untouched")

	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           alphaID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoA,
		Name:             "alpha",
		Description:      "stale alpha",
		Content:          "stale alpha content",
		ContentHash:      buildCatalogContentHash("stale alpha content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           stableID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoA,
		Name:             "stable",
		Description:      "stable repo-a row",
		Content:          "stable repo-a content",
		ContentHash:      buildCatalogContentHash("stable repo-a content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           legacyPromptID,
		Classifier:       persistence.CatalogClassifierPrompt,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoA,
		ParentSkillID:    stringPointer("repo-a/stable"),
		ResourcePath:     stringPointer("imports/prompts/legacy.md"),
		Name:             "legacy.md",
		Description:      "legacy prompt",
		Content:          "legacy prompt content",
		ContentHash:      buildCatalogContentHash("legacy prompt content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           reviveID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoA,
		Name:             "revive",
		Description:      "deleted repo-a row",
		Content:          "deleted repo-a content",
		ContentHash:      buildCatalogContentHash("deleted repo-a content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
		DeletedAt:        &deletedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           repoBID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoB,
		Name:             "untouched",
		Description:      "repo-b row",
		Content:          "repo-b content",
		ContentHash:      buildCatalogContentHash("repo-b content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           localID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "local-untouched",
		Description:      "local row",
		Content:          "local content",
		ContentHash:      buildCatalogContentHash("local content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     lastSyncedAt,
	})

	overlayUpdatedAt := time.Date(2026, time.March, 4, 13, 15, 0, 0, time.UTC)
	if err := overlayRepo.Upsert(ctx, persistence.CatalogMetadataOverlayRow{
		ItemID:              legacyPromptID,
		DisplayNameOverride: stringPointer("Legacy Prompt Override"),
		CustomMetadata: map[string]any{
			"owner": "platform",
		},
		Labels:    []string{"legacy", "repo-a"},
		UpdatedAt: overlayUpdatedAt,
	}); err != nil {
		t.Fatalf("expected overlay upsert to succeed, got %v", err)
	}

	discovered := []CatalogItem{
		{
			ID:          alphaID,
			Classifier:  CatalogClassifierSkill,
			Name:        "alpha",
			Description: "fresh alpha",
			Content:     "fresh alpha content",
			ReadOnly:    true,
		},
		{
			ID:          stableID,
			Classifier:  CatalogClassifierSkill,
			Name:        "stable",
			Description: "stable repo-a row",
			Content:     "stable repo-a content",
			ReadOnly:    true,
		},
		{
			ID:          reviveID,
			Classifier:  CatalogClassifierSkill,
			Name:        "revive",
			Description: "revived repo-a row",
			Content:     "revived repo-a content",
			ReadOnly:    true,
		},
		{
			ID:          repoBID,
			Classifier:  CatalogClassifierSkill,
			Name:        "untouched",
			Description: "repo-b row",
			Content:     "repo-b changed but should be ignored",
			ReadOnly:    true,
		},
		{
			ID:          localID,
			Classifier:  CatalogClassifierSkill,
			Name:        "local-untouched",
			Description: "local row",
			Content:     "local changed but should be ignored",
			ReadOnly:    false,
		},
	}

	var logBuffer bytes.Buffer
	service := newCatalogSyncServiceForDomainTest(t, sourceRepo, &logBuffer, syncAt)

	if err := service.SyncRepo(repoA, discovered); err != nil {
		t.Fatalf("expected repo sync to succeed, got %v", err)
	}

	alphaRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, alphaID)
	if alphaRow.ContentHash != buildCatalogContentHash("fresh alpha content") {
		t.Fatalf("expected repo-a alpha hash to be updated, got %q", alphaRow.ContentHash)
	}
	if !alphaRow.LastSyncedAt.Equal(syncAt) {
		t.Fatalf("expected repo-a alpha last_synced_at %s, got %s", syncAt, alphaRow.LastSyncedAt)
	}

	stableRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, stableID)
	if stableRow.ContentHash != buildCatalogContentHash("stable repo-a content") {
		t.Fatalf("expected repo-a stable hash to remain unchanged, got %q", stableRow.ContentHash)
	}
	if !stableRow.LastSyncedAt.Equal(lastSyncedAt) {
		t.Fatalf("expected unchanged repo-a stable last_synced_at %s, got %s", lastSyncedAt, stableRow.LastSyncedAt)
	}

	legacyPromptRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, legacyPromptID)
	if legacyPromptRow.DeletedAt == nil || !legacyPromptRow.DeletedAt.Equal(syncAt) {
		t.Fatalf("expected missing repo-a prompt to be tombstoned at %s, got %v", syncAt, legacyPromptRow.DeletedAt)
	}

	reviveRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, reviveID)
	if reviveRow.DeletedAt != nil {
		t.Fatalf("expected revived repo-a row deleted_at to be nil, got %v", reviveRow.DeletedAt)
	}
	if reviveRow.ContentHash != buildCatalogContentHash("revived repo-a content") {
		t.Fatalf("expected revived repo-a row hash to update, got %q", reviveRow.ContentHash)
	}

	repoBRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, repoBID)
	if repoBRow.ContentHash != buildCatalogContentHash("repo-b content") {
		t.Fatalf("expected repo-b row hash to remain unchanged, got %q", repoBRow.ContentHash)
	}
	if !repoBRow.LastSyncedAt.Equal(lastSyncedAt) {
		t.Fatalf("expected repo-b row last_synced_at %s, got %s", lastSyncedAt, repoBRow.LastSyncedAt)
	}

	localRow := mustGetCatalogSourceRowForDomainTest(t, ctx, sourceRepo, localID)
	if localRow.ContentHash != buildCatalogContentHash("local content") {
		t.Fatalf("expected local row hash to remain unchanged, got %q", localRow.ContentHash)
	}
	if !localRow.LastSyncedAt.Equal(lastSyncedAt) {
		t.Fatalf("expected local row last_synced_at %s, got %s", lastSyncedAt, localRow.LastSyncedAt)
	}

	overlayRow, err := overlayRepo.GetByItemID(ctx, legacyPromptID)
	if err != nil {
		t.Fatalf("expected overlay lookup to succeed, got %v", err)
	}
	if overlayRow.DisplayNameOverride == nil || *overlayRow.DisplayNameOverride != "Legacy Prompt Override" {
		t.Fatalf("expected overlay display name to remain unchanged, got %+v", overlayRow.DisplayNameOverride)
	}
	if gotOwner, ok := overlayRow.CustomMetadata["owner"].(string); !ok || gotOwner != "platform" {
		t.Fatalf("expected overlay owner metadata to remain unchanged, got %+v", overlayRow.CustomMetadata)
	}
	if len(overlayRow.Labels) != 2 || overlayRow.Labels[0] != "legacy" || overlayRow.Labels[1] != "repo-a" {
		t.Fatalf("expected overlay labels to remain unchanged, got %+v", overlayRow.Labels)
	}
	if !overlayRow.UpdatedAt.Equal(overlayUpdatedAt) {
		t.Fatalf("expected overlay updated_at %s, got %s", overlayUpdatedAt, overlayRow.UpdatedAt)
	}

	assertCatalogSyncLogCounts(t, logBuffer.String(), map[string]string{
		"mode":       "repo",
		"repo":       "repo-a",
		"discovered": "3",
		"existing":   "4",
		"upserted":   "2",
		"tombstoned": "1",
		"restored":   "1",
		"unchanged":  "1",
	})
}

func TestCatalogSyncService_SyncRepo_EmptyRepoNameReturnsError(t *testing.T) {
	db, _ := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)

	service := newCatalogSyncServiceForDomainTest(t, sourceRepo, nil, time.Date(2026, time.March, 4, 15, 0, 0, 0, time.UTC))
	err := service.SyncRepo("   ", []CatalogItem{})
	if err == nil {
		t.Fatalf("expected empty repo name to return an error, got nil")
	}
	if !strings.Contains(err.Error(), "repository name is required") {
		t.Fatalf("expected empty repo name error message, got %v", err)
	}
}

func TestCatalogSyncService_NilReceiver_ReturnsError(t *testing.T) {
	var service *CatalogSyncService

	if err := service.SyncAll(nil); err == nil || !strings.Contains(err.Error(), "service is required") {
		t.Fatalf("expected nil service error from SyncAll, got %v", err)
	}

	if err := service.SyncRepo("repo-a", nil); err == nil || !strings.Contains(err.Error(), "service is required") {
		t.Fatalf("expected nil service error from SyncRepo, got %v", err)
	}
}

func TestNewCatalogSyncService_WithNilRepository_ReturnsError(t *testing.T) {
	_, err := NewCatalogSyncService(nil, CatalogSyncServiceOptions{})
	if err == nil {
		t.Fatalf("expected nil repository error, got nil")
	}
	if !strings.Contains(err.Error(), "source repository is required") {
		t.Fatalf("expected nil repository error message, got %v", err)
	}
}

func TestMapCatalogItemToSourceRow_DerivesPromptFieldsFromCatalogID(t *testing.T) {
	syncedAt := time.Date(2026, time.March, 4, 16, 0, 0, 0, time.UTC)
	item := CatalogItem{
		ID:            BuildPromptCatalogItemID("repo-a/planner", "imports/prompts/system.md"),
		Classifier:    "",
		Name:          "",
		Description:   "  prompt description  ",
		Content:       "prompt content",
		ParentSkillID: "",
		ResourcePath:  "imports/prompts/system.md",
		ReadOnly:      true,
	}

	row, err := mapCatalogItemToSourceRow(item, syncedAt)
	if err != nil {
		t.Fatalf("expected prompt row mapping to succeed, got %v", err)
	}

	if row.Classifier != persistence.CatalogClassifierPrompt {
		t.Fatalf("expected prompt classifier, got %q", row.Classifier)
	}
	if row.SourceType != persistence.CatalogSourceTypeGit {
		t.Fatalf("expected git source type, got %q", row.SourceType)
	}
	if row.SourceRepo == nil || *row.SourceRepo != "repo-a" {
		t.Fatalf("expected source repo repo-a, got %+v", row.SourceRepo)
	}
	if row.ParentSkillID == nil || *row.ParentSkillID != "repo-a/planner" {
		t.Fatalf("expected parent skill id repo-a/planner, got %+v", row.ParentSkillID)
	}
	if row.Name != "system.md" {
		t.Fatalf("expected derived prompt name system.md, got %q", row.Name)
	}
	if row.Description != "prompt description" {
		t.Fatalf("expected trimmed description, got %q", row.Description)
	}
}

func TestMapCatalogItemToSourceRow_DetectsFileImportSourceType(t *testing.T) {
	syncedAt := time.Date(2026, time.March, 4, 16, 30, 0, 0, time.UTC)
	item := CatalogItem{
		ID:          BuildSkillCatalogItemID("file-import/imported-skill"),
		Classifier:  CatalogClassifierSkill,
		Name:        "",
		Description: "imported skill",
		Content:     "imported content",
		ReadOnly:    false,
	}

	row, err := mapCatalogItemToSourceRow(item, syncedAt)
	if err != nil {
		t.Fatalf("expected file-import row mapping to succeed, got %v", err)
	}
	if row.SourceType != persistence.CatalogSourceTypeFileImport {
		t.Fatalf("expected file_import source type, got %q", row.SourceType)
	}
	if row.Name != "imported-skill" {
		t.Fatalf("expected derived skill name imported-skill, got %q", row.Name)
	}
}

func TestMapCatalogClassifier_WithInvalidInput_ReturnsError(t *testing.T) {
	_, err := mapCatalogClassifier(CatalogClassifier("unknown"), "invalid-id")
	if err == nil {
		t.Fatalf("expected invalid classifier error, got nil")
	}
}

func TestMapCatalogClassifier_InfersSkillFromIDPrefix(t *testing.T) {
	classifier, err := mapCatalogClassifier(CatalogClassifier(""), BuildSkillCatalogItemID("local-skill"))
	if err != nil {
		t.Fatalf("expected skill classifier inference to succeed, got %v", err)
	}
	if classifier != persistence.CatalogClassifierSkill {
		t.Fatalf("expected inferred skill classifier, got %q", classifier)
	}
}

func TestBuildDiscoveredSourceRows_WithConflictingDuplicateRows_ReturnsError(t *testing.T) {
	syncedAt := time.Date(2026, time.March, 4, 17, 0, 0, 0, time.UTC)
	itemID := BuildSkillCatalogItemID("repo-a/conflict")
	discovered := []CatalogItem{
		{
			ID:          itemID,
			Classifier:  CatalogClassifierSkill,
			Name:        "conflict",
			Description: "first",
			Content:     "first-content",
			ReadOnly:    true,
		},
		{
			ID:          itemID,
			Classifier:  CatalogClassifierSkill,
			Name:        "conflict",
			Description: "second",
			Content:     "second-content",
			ReadOnly:    true,
		},
	}

	_, err := buildDiscoveredSourceRows(
		catalogSyncScope{mode: catalogSyncModeFull},
		discovered,
		syncedAt,
	)
	if err == nil {
		t.Fatalf("expected duplicate conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "conflicting snapshots") {
		t.Fatalf("expected conflicting snapshots error, got %v", err)
	}
}

func TestResolveCatalogSkillID_HandlesParentSkillAndCatalogPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		item     CatalogItem
		expected string
	}{
		{
			name: "uses parent skill id when present",
			item: CatalogItem{
				ParentSkillID: "./repo-a/planner/",
				ID:            BuildSkillCatalogItemID("ignored"),
			},
			expected: "repo-a/planner",
		},
		{
			name: "parses skill catalog id",
			item: CatalogItem{
				ID: BuildSkillCatalogItemID("local-skill"),
			},
			expected: "local-skill",
		},
		{
			name: "parses prompt catalog id",
			item: CatalogItem{
				ID: BuildPromptCatalogItemID("repo-b/coach", "prompts/system.md"),
			},
			expected: "repo-b/coach",
		},
		{
			name: "returns empty for unknown id",
			item: CatalogItem{
				ID: "invalid-id",
			},
			expected: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if actual := resolveCatalogSkillID(testCase.item); actual != testCase.expected {
				t.Fatalf("resolveCatalogSkillID() = %q, want %q", actual, testCase.expected)
			}
		})
	}
}

func TestDeriveCatalogItemName_FallsBackToItemID(t *testing.T) {
	name := deriveCatalogItemName(CatalogItem{
		ID: "prompt:repo-a/planner:custom/path",
	}, "")
	if name != "prompt:repo-a/planner:custom/path" {
		t.Fatalf("expected item id fallback name, got %q", name)
	}
}

func TestIsFileImportCatalogItem_RecognizesPrefixes(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{input: "file-import/my-skill", expected: true},
		{input: "file_import/my-skill", expected: true},
		{input: "imported/my-skill", expected: true},
		{input: "local-skill", expected: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.input, func(t *testing.T) {
			if actual := isFileImportCatalogItem(testCase.input); actual != testCase.expected {
				t.Fatalf("isFileImportCatalogItem(%q) = %v, want %v", testCase.input, actual, testCase.expected)
			}
		})
	}
}

func TestCatalogSourceRowsEqualForSync_WhenOnlyLastSyncedDiffers_ReturnsTrue(t *testing.T) {
	sourceRepo := "repo-a"
	rowA := persistence.CatalogSourceRow{
		ItemID:           BuildSkillCatalogItemID("repo-a/stable"),
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &sourceRepo,
		Name:             "stable",
		Description:      "stable row",
		Content:          "stable content",
		ContentHash:      buildCatalogContentHash("stable content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 18, 0, 0, 0, time.UTC),
	}
	rowB := rowA
	rowB.LastSyncedAt = time.Date(2026, time.March, 4, 19, 0, 0, 0, time.UTC)

	if !catalogSourceRowsEqualForSync(rowA, rowB) {
		t.Fatalf("expected rows to be considered unchanged when only last_synced_at differs")
	}

	rowB.Content = "updated"
	rowB.ContentHash = buildCatalogContentHash("updated")
	if catalogSourceRowsEqualForSync(rowA, rowB) {
		t.Fatalf("expected rows with different content to be considered changed")
	}
}

func TestCatalogSourceRowsEqualForSync_DetectsFieldDifferences(t *testing.T) {
	sourceRepo := "repo-a"
	parentSkillID := "repo-a/stable"
	resourcePath := "imports/prompts/system.md"
	base := persistence.CatalogSourceRow{
		ItemID:           BuildPromptCatalogItemID(parentSkillID, resourcePath),
		Classifier:       persistence.CatalogClassifierPrompt,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &sourceRepo,
		ParentSkillID:    &parentSkillID,
		ResourcePath:     &resourcePath,
		Name:             "system.md",
		Description:      "description",
		Content:          "content",
		ContentHash:      buildCatalogContentHash("content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 20, 0, 0, 0, time.UTC),
	}

	missingRepo := "repo-b"
	missingParent := "repo-b/stable"
	missingPath := "imports/prompts/other.md"
	deletedAt := time.Date(2026, time.March, 4, 21, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		mutate  func(row *persistence.CatalogSourceRow)
		expects bool
	}{
		{
			name: "classifier differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.Classifier = persistence.CatalogClassifierSkill
			},
			expects: false,
		},
		{
			name: "source type differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.SourceType = persistence.CatalogSourceTypeLocal
			},
			expects: false,
		},
		{
			name: "source repo differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.SourceRepo = &missingRepo
			},
			expects: false,
		},
		{
			name: "parent skill differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.ParentSkillID = &missingParent
			},
			expects: false,
		},
		{
			name: "resource path differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.ResourcePath = &missingPath
			},
			expects: false,
		},
		{
			name: "name differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.Name = "assistant.md"
			},
			expects: false,
		},
		{
			name: "description differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.Description = "other description"
			},
			expects: false,
		},
		{
			name: "content differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.Content = "other content"
			},
			expects: false,
		},
		{
			name: "content hash differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.ContentHash = buildCatalogContentHash("other content")
			},
			expects: false,
		},
		{
			name: "content writable differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.ContentWritable = true
			},
			expects: false,
		},
		{
			name: "metadata writable differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.MetadataWritable = false
			},
			expects: false,
		},
		{
			name: "deleted at differs",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.DeletedAt = &deletedAt
			},
			expects: false,
		},
		{
			name: "identical row matches",
			mutate: func(row *persistence.CatalogSourceRow) {
				row.LastSyncedAt = time.Date(2026, time.March, 4, 22, 0, 0, 0, time.UTC)
			},
			expects: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := base
			testCase.mutate(&candidate)

			actual := catalogSourceRowsEqualForSync(base, candidate)
			if actual != testCase.expects {
				t.Fatalf("catalogSourceRowsEqualForSync() = %v, want %v", actual, testCase.expects)
			}
		})
	}
}

func openCatalogSyncServiceTestDB(t *testing.T) (*sql.DB, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	dbPath := filepath.Join(t.TempDir(), "catalog-sync-service.db")
	db, err := persistence.BootstrapSQLite(ctx, dbPath, persistence.SQLiteBootstrapConfig{})
	if err != nil {
		t.Fatalf("expected sqlite bootstrap to succeed, got %v", err)
	}
	t.Cleanup(func() {
		if closeErr := persistence.CloseSQLite(db); closeErr != nil {
			t.Fatalf("expected sqlite close to succeed, got %v", closeErr)
		}
	})

	return db, ctx
}

func newCatalogSyncServiceForDomainTest(
	t *testing.T,
	sourceRepo *persistence.CatalogSourceRepository,
	logBuffer *bytes.Buffer,
	syncAt time.Time,
) *CatalogSyncService {
	t.Helper()

	logger := log.New(logBuffer, "", 0)
	if logBuffer == nil {
		logger = nil
	}

	service, err := NewCatalogSyncService(sourceRepo, CatalogSyncServiceOptions{
		Logger: logger,
		Now: func() time.Time {
			return syncAt
		},
	})
	if err != nil {
		t.Fatalf("expected catalog sync service creation to succeed, got %v", err)
	}

	return service
}

func newCatalogSourceRepositoryForDomainTest(t *testing.T, db *sql.DB) *persistence.CatalogSourceRepository {
	t.Helper()

	repo, err := persistence.NewCatalogSourceRepository(db)
	if err != nil {
		t.Fatalf("expected source repository creation to succeed, got %v", err)
	}
	return repo
}

func newCatalogOverlayRepositoryForDomainTest(t *testing.T, db *sql.DB) *persistence.CatalogMetadataOverlayRepository {
	t.Helper()

	repo, err := persistence.NewCatalogMetadataOverlayRepository(db)
	if err != nil {
		t.Fatalf("expected overlay repository creation to succeed, got %v", err)
	}
	return repo
}

func mustUpsertCatalogSourceRowForDomainTest(
	t *testing.T,
	ctx context.Context,
	repo *persistence.CatalogSourceRepository,
	row persistence.CatalogSourceRow,
) {
	t.Helper()

	if err := repo.Upsert(ctx, row); err != nil {
		t.Fatalf("expected source row upsert to succeed, got %v", err)
	}
}

func mustGetCatalogSourceRowForDomainTest(
	t *testing.T,
	ctx context.Context,
	repo *persistence.CatalogSourceRepository,
	itemID string,
) persistence.CatalogSourceRow {
	t.Helper()

	row, err := repo.GetByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected source row lookup for %q to succeed, got %v", itemID, err)
	}
	return row
}

func assertCatalogSyncLogCounts(t *testing.T, logs string, expected map[string]string) {
	t.Helper()

	for key, value := range expected {
		token := key + "=" + value
		if !strings.Contains(logs, token) {
			t.Fatalf("expected sync logs to contain %q, got %q", token, logs)
		}
	}
}
