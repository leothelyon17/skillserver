package main

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/persistence"
)

func TestCatalogPersistenceCoordinator_FullSyncAndRebuild_IndexesEffectiveCatalog(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	writeSkillFixture(
		t,
		filepath.Join(skillsDir, "local-skill"),
		"local-skill",
		"Local skill source description",
		"# Local Skill\n",
	)

	manager, err := domain.NewFileSystemManager(skillsDir, nil)
	if err != nil {
		t.Fatalf("failed to initialize file system manager: %v", err)
	}

	persistenceDir := t.TempDir()
	cfg := PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     persistenceDir,
		DBPath:  filepath.Join(persistenceDir, "catalog.sqlite"),
	}
	runtime, err := bootstrapCatalogPersistenceRuntime(
		context.Background(),
		cfg,
		manager,
		log.New(io.Discard, "", 0),
	)
	if err != nil {
		t.Fatalf("failed to bootstrap persistence runtime: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := runtime.Close(); closeErr != nil {
			t.Fatalf("failed closing persistence runtime: %v", closeErr)
		}
	})

	if err := runtime.coordinator.FullSyncAndRebuild(context.Background()); err != nil {
		t.Fatalf("full sync and rebuild failed: %v", err)
	}

	runner := persistence.NewMigrationRunner(runtime.db)
	version, err := runner.CurrentVersion(context.Background())
	if err != nil {
		t.Fatalf("failed to query schema version: %v", err)
	}
	if version != persistence.LatestSchemaVersion() {
		t.Fatalf("expected schema version %d, got %d", persistence.LatestSchemaVersion(), version)
	}

	sourceRows, err := runtime.sourceRepo.List(context.Background(), persistence.CatalogSourceListFilter{})
	if err != nil {
		t.Fatalf("failed to list source rows: %v", err)
	}
	if len(sourceRows) != 1 {
		t.Fatalf("expected 1 source row after startup sync, got %d", len(sourceRows))
	}

	overrideName := "Local Skill Overlay Name"
	if err := runtime.overlayRepo.Upsert(context.Background(), persistence.CatalogMetadataOverlayRow{
		ItemID:              sourceRows[0].ItemID,
		DisplayNameOverride: &overrideName,
		UpdatedAt:           time.Date(2026, time.March, 4, 15, 30, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("failed to upsert overlay row: %v", err)
	}

	if err := runtime.coordinator.FullSyncAndRebuild(context.Background()); err != nil {
		t.Fatalf("full sync and rebuild with overlay failed: %v", err)
	}

	searchResults, err := manager.SearchCatalogItems(overrideName, nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if !catalogItemsContainID(searchResults, sourceRows[0].ItemID) {
		t.Fatalf("expected search results to include item %q after effective rebuild", sourceRows[0].ItemID)
	}
}

func TestCatalogPersistenceCoordinator_RepoSyncAndRebuild_UpdatesOnlyTargetRepoAndPreservesOverlay(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	repoOneName := "repo-one"
	repoTwoName := "repo-two"

	repoOneSkillDir := filepath.Join(skillsDir, repoOneName, "alpha-skill")
	repoTwoSkillDir := filepath.Join(skillsDir, repoTwoName, "bravo-skill")
	writeSkillFixture(
		t,
		repoOneSkillDir,
		"alpha-skill",
		"Repo one source description",
		"# Repo One Skill\n",
	)
	writeSkillFixture(
		t,
		repoTwoSkillDir,
		"bravo-skill",
		"Repo two source description",
		"# Repo Two Skill\n",
	)

	manager, err := domain.NewFileSystemManager(skillsDir, []string{repoOneName, repoTwoName})
	if err != nil {
		t.Fatalf("failed to initialize file system manager: %v", err)
	}

	persistenceDir := t.TempDir()
	cfg := PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     persistenceDir,
		DBPath:  filepath.Join(persistenceDir, "catalog.sqlite"),
	}
	runtime, err := bootstrapCatalogPersistenceRuntime(
		context.Background(),
		cfg,
		manager,
		log.New(io.Discard, "", 0),
	)
	if err != nil {
		t.Fatalf("failed to bootstrap persistence runtime: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := runtime.Close(); closeErr != nil {
			t.Fatalf("failed closing persistence runtime: %v", closeErr)
		}
	})

	if err := runtime.coordinator.FullSyncAndRebuild(context.Background()); err != nil {
		t.Fatalf("initial full sync failed: %v", err)
	}

	gitSourceType := persistence.CatalogSourceTypeGit

	repoOneRows, err := runtime.sourceRepo.List(context.Background(), persistence.CatalogSourceListFilter{
		SourceType:     &gitSourceType,
		SourceRepo:     &repoOneName,
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("failed to list repo-one rows: %v", err)
	}
	repoTwoRows, err := runtime.sourceRepo.List(context.Background(), persistence.CatalogSourceListFilter{
		SourceType:     &gitSourceType,
		SourceRepo:     &repoTwoName,
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("failed to list repo-two rows: %v", err)
	}
	if len(repoOneRows) != 1 || len(repoTwoRows) != 1 {
		t.Fatalf("expected one source row per git repo, got repo-one=%d repo-two=%d", len(repoOneRows), len(repoTwoRows))
	}

	repoOneRowBefore := repoOneRows[0]
	repoTwoRowBefore := repoTwoRows[0]

	repoTwoOverlayName := "Repo Two Overlay"
	if err := runtime.overlayRepo.Upsert(context.Background(), persistence.CatalogMetadataOverlayRow{
		ItemID:              repoTwoRowBefore.ItemID,
		DisplayNameOverride: &repoTwoOverlayName,
		UpdatedAt:           time.Date(2026, time.March, 4, 16, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("failed to upsert repo-two overlay: %v", err)
	}

	writeSkillFixture(
		t,
		repoOneSkillDir,
		"alpha-skill",
		"Repo one UPDATED description",
		"# Repo One Skill Updated\n",
	)

	if err := runtime.coordinator.RepoSyncAndRebuild(context.Background(), repoOneName); err != nil {
		t.Fatalf("repo-scoped sync failed: %v", err)
	}

	repoOneRowAfter, err := runtime.sourceRepo.GetByItemID(context.Background(), repoOneRowBefore.ItemID)
	if err != nil {
		t.Fatalf("failed to get updated repo-one row: %v", err)
	}
	if repoOneRowAfter.Description != "Repo one UPDATED description" {
		t.Fatalf("expected repo-one description to update, got %q", repoOneRowAfter.Description)
	}

	repoTwoRowAfter, err := runtime.sourceRepo.GetByItemID(context.Background(), repoTwoRowBefore.ItemID)
	if err != nil {
		t.Fatalf("failed to get repo-two row: %v", err)
	}
	if !repoTwoRowAfter.LastSyncedAt.Equal(repoTwoRowBefore.LastSyncedAt) {
		t.Fatalf(
			"expected repo-two row to remain untouched by repo-one sync; last_synced_at before=%s after=%s",
			repoTwoRowBefore.LastSyncedAt.Format(time.RFC3339Nano),
			repoTwoRowAfter.LastSyncedAt.Format(time.RFC3339Nano),
		)
	}

	repoTwoOverlay, err := runtime.overlayRepo.GetByItemID(context.Background(), repoTwoRowBefore.ItemID)
	if err != nil {
		t.Fatalf("expected repo-two overlay to be preserved, got error: %v", err)
	}
	if repoTwoOverlay.DisplayNameOverride == nil || *repoTwoOverlay.DisplayNameOverride != repoTwoOverlayName {
		t.Fatalf("expected repo-two overlay name %q, got %#v", repoTwoOverlayName, repoTwoOverlay.DisplayNameOverride)
	}

	searchResults, err := manager.SearchCatalogItems(repoTwoOverlayName, nil)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if !catalogItemsContainID(searchResults, repoTwoRowBefore.ItemID) {
		t.Fatalf("expected effective search index to include repo-two overlay item %q", repoTwoRowBefore.ItemID)
	}
}

func TestCatalogPersistenceCoordinator_FullSyncAndRebuild_PreservesOverlayAcrossRuntimeRestart(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	skillsDir := t.TempDir()
	writeSkillFixture(
		t,
		filepath.Join(skillsDir, "restart-skill"),
		"restart-skill",
		"Restart fixture source description",
		"# Restart Skill\n",
	)

	managerFirst, err := domain.NewFileSystemManager(skillsDir, nil)
	if err != nil {
		t.Fatalf("failed to initialize first file system manager: %v", err)
	}

	persistenceDir := t.TempDir()
	cfg := PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     persistenceDir,
		DBPath:  filepath.Join(persistenceDir, "catalog.sqlite"),
	}
	runtimeFirst, err := bootstrapCatalogPersistenceRuntime(
		ctx,
		cfg,
		managerFirst,
		log.New(io.Discard, "", 0),
	)
	if err != nil {
		t.Fatalf("failed to bootstrap first persistence runtime: %v", err)
	}
	t.Cleanup(func() {
		if runtimeFirst != nil {
			if closeErr := runtimeFirst.Close(); closeErr != nil {
				t.Fatalf("failed closing first persistence runtime: %v", closeErr)
			}
		}
	})

	if err := runtimeFirst.coordinator.FullSyncAndRebuild(ctx); err != nil {
		t.Fatalf("first full sync and rebuild failed: %v", err)
	}

	sourceRows, err := runtimeFirst.sourceRepo.List(ctx, persistence.CatalogSourceListFilter{})
	if err != nil {
		t.Fatalf("failed to list first-runtime source rows: %v", err)
	}
	if len(sourceRows) != 1 {
		t.Fatalf("expected 1 source row in first runtime, got %d", len(sourceRows))
	}

	itemID := sourceRows[0].ItemID
	restartOverlayName := "Restart Overlay Name"
	if err := runtimeFirst.overlayRepo.Upsert(ctx, persistence.CatalogMetadataOverlayRow{
		ItemID:              itemID,
		DisplayNameOverride: &restartOverlayName,
		UpdatedAt:           time.Date(2026, time.March, 4, 18, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("failed to upsert first-runtime overlay row: %v", err)
	}

	if err := runtimeFirst.coordinator.FullSyncAndRebuild(ctx); err != nil {
		t.Fatalf("first full sync and rebuild with overlay failed: %v", err)
	}

	searchResults, err := managerFirst.SearchCatalogItems(restartOverlayName, nil)
	if err != nil {
		t.Fatalf("first-runtime search failed: %v", err)
	}
	if !catalogItemsContainID(searchResults, itemID) {
		t.Fatalf("expected first-runtime search to include item %q", itemID)
	}

	if err := runtimeFirst.Close(); err != nil {
		t.Fatalf("failed to close first runtime before restart: %v", err)
	}
	runtimeFirst = nil

	runtimeSecond, err := bootstrapCatalogPersistenceRuntime(
		ctx,
		cfg,
		managerFirst,
		log.New(io.Discard, "", 0),
	)
	if err != nil {
		t.Fatalf("failed to bootstrap second persistence runtime: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := runtimeSecond.Close(); closeErr != nil {
			t.Fatalf("failed closing second persistence runtime: %v", closeErr)
		}
	})

	if err := runtimeSecond.coordinator.FullSyncAndRebuild(ctx); err != nil {
		t.Fatalf("second full sync and rebuild failed: %v", err)
	}

	overlayRow, err := runtimeSecond.overlayRepo.GetByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected overlay row lookup after restart to succeed, got %v", err)
	}
	if overlayRow.DisplayNameOverride == nil || *overlayRow.DisplayNameOverride != restartOverlayName {
		t.Fatalf("expected overlay display name %q after restart, got %#v", restartOverlayName, overlayRow.DisplayNameOverride)
	}

	effectiveItems, err := runtimeSecond.coordinator.effectiveService.List(ctx, domain.CatalogEffectiveListFilter{})
	if err != nil {
		t.Fatalf("failed to list effective items after restart: %v", err)
	}
	foundEffectiveItem := false
	for _, item := range effectiveItems {
		if item.ID == itemID {
			foundEffectiveItem = true
			if item.Name != restartOverlayName {
				t.Fatalf("expected effective item name %q after restart, got %q", restartOverlayName, item.Name)
			}
		}
	}
	if !foundEffectiveItem {
		t.Fatalf("expected effective item %q after restart", itemID)
	}

	restartedSearchResults, err := managerFirst.SearchCatalogItems(restartOverlayName, nil)
	if err != nil {
		t.Fatalf("second-runtime search failed: %v", err)
	}
	if !catalogItemsContainID(restartedSearchResults, itemID) {
		t.Fatalf("expected second-runtime search to include item %q after restart", itemID)
	}
}

func writeSkillFixture(t *testing.T, skillDir string, name string, description string, body string) {
	t.Helper()

	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory %q: %v", skillDir, err)
	}

	skillMarkdown := "---\n" +
		"name: " + name + "\n" +
		"description: " + description + "\n" +
		"---\n\n" +
		body

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMarkdown), 0o644); err != nil {
		t.Fatalf("failed to write skill fixture %q: %v", skillDir, err)
	}
}

func catalogItemsContainID(items []domain.CatalogItem, itemID string) bool {
	for _, item := range items {
		if item.ID == itemID {
			return true
		}
	}
	return false
}
