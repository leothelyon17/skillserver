package domain

import (
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

func TestNewCatalogEffectiveService_WithNilRepositories_ReturnsError(t *testing.T) {
	db, _ := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)

	if _, err := NewCatalogEffectiveService(nil, overlayRepo); err == nil {
		t.Fatalf("expected nil source repository error, got nil")
	}
	if _, err := NewCatalogEffectiveService(sourceRepo, nil); err == nil {
		t.Fatalf("expected nil overlay repository error, got nil")
	}
}

func TestCatalogEffectiveService_List_NilReceiver_ReturnsError(t *testing.T) {
	var service *CatalogEffectiveService
	_, err := service.List(nil, CatalogEffectiveListFilter{})
	if err == nil {
		t.Fatalf("expected nil service receiver error, got nil")
	}
}

func TestCatalogEffectiveService_List_AppliesOverlayPrecedenceAndNullFallback(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)

	localSkillID := BuildSkillCatalogItemID("local-planner")
	gitPromptID := BuildPromptCatalogItemID("repo-a/planner", "imports/prompts/system.md")
	repoName := "repo-a"
	parentSkillID := "repo-a/planner"
	resourcePath := "imports/prompts/system.md"
	syncedAt := time.Date(2026, time.March, 4, 23, 0, 0, 0, time.UTC)

	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           localSkillID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "local-planner",
		Description:      "Local planner source description",
		Content:          "Local planner source content",
		ContentHash:      buildCatalogContentHash("Local planner source content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           gitPromptID,
		Classifier:       persistence.CatalogClassifierPrompt,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoName,
		ParentSkillID:    &parentSkillID,
		ResourcePath:     &resourcePath,
		Name:             "system.md",
		Description:      "System prompt source description",
		Content:          "System prompt source content",
		ContentHash:      buildCatalogContentHash("System prompt source content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})

	if err := overlayRepo.Upsert(ctx, persistence.CatalogMetadataOverlayRow{
		ItemID:              localSkillID,
		DisplayNameOverride: stringPointer("Planner Override"),
		CustomMetadata: map[string]any{
			"owner": "platform",
		},
		Labels:    []string{"metadata", "override"},
		UpdatedAt: syncedAt,
	}); err != nil {
		t.Fatalf("expected local overlay upsert to succeed, got %v", err)
	}

	if err := overlayRepo.Upsert(ctx, persistence.CatalogMetadataOverlayRow{
		ItemID:              gitPromptID,
		DisplayNameOverride: stringPointer("   "),
		DescriptionOverride: stringPointer(""),
		UpdatedAt:           syncedAt,
	}); err != nil {
		t.Fatalf("expected git prompt overlay upsert to succeed, got %v", err)
	}

	service := newCatalogEffectiveServiceForDomainTest(t, sourceRepo, overlayRepo)

	items, err := service.List(ctx, CatalogEffectiveListFilter{})
	if err != nil {
		t.Fatalf("expected effective list query to succeed, got %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 effective items, got %d", len(items))
	}

	itemsByID := make(map[string]CatalogItem, len(items))
	for _, item := range items {
		itemsByID[item.ID] = item
	}

	localSkill := itemsByID[localSkillID]
	if localSkill.Name != "Planner Override" {
		t.Fatalf("expected local skill name override, got %q", localSkill.Name)
	}
	if localSkill.Description != "Local planner source description" {
		t.Fatalf("expected local skill description fallback, got %q", localSkill.Description)
	}
	if localSkill.CustomMetadata["owner"] != "platform" {
		t.Fatalf("expected local skill custom metadata owner platform, got %+v", localSkill.CustomMetadata)
	}
	if len(localSkill.Labels) != 2 {
		t.Fatalf("expected local skill labels to round-trip, got %+v", localSkill.Labels)
	}

	gitPrompt := itemsByID[gitPromptID]
	if gitPrompt.Name != "system.md" {
		t.Fatalf("expected git prompt name fallback to source when override empty, got %q", gitPrompt.Name)
	}
	if gitPrompt.Description != "System prompt source description" {
		t.Fatalf("expected git prompt description fallback to source, got %q", gitPrompt.Description)
	}
	if len(gitPrompt.CustomMetadata) != 0 {
		t.Fatalf("expected git prompt custom metadata default empty object, got %+v", gitPrompt.CustomMetadata)
	}
}

func TestCatalogEffectiveService_List_EnforcesMutabilityMatrixAndReadOnlyCompatibility(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)

	repoName := "repo-a"
	syncedAt := time.Date(2026, time.March, 4, 23, 30, 0, 0, time.UTC)

	rows := []persistence.CatalogSourceRow{
		{
			ItemID:           BuildSkillCatalogItemID("repo-a/git-item"),
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       &repoName,
			Name:             "git-item",
			Description:      "git source",
			Content:          "git content",
			ContentHash:      buildCatalogContentHash("git content"),
			ContentWritable:  true,
			MetadataWritable: false,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           BuildSkillCatalogItemID("local-item"),
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "local-item",
			Description:      "local source",
			Content:          "local content",
			ContentHash:      buildCatalogContentHash("local content"),
			ContentWritable:  false,
			MetadataWritable: false,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           BuildSkillCatalogItemID("file-import/item"),
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeFileImport,
			Name:             "file-import-item",
			Description:      "file import source",
			Content:          "file import content",
			ContentHash:      buildCatalogContentHash("file import content"),
			ContentWritable:  false,
			MetadataWritable: false,
			LastSyncedAt:     syncedAt,
		},
	}
	for _, row := range rows {
		mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, row)
	}

	service := newCatalogEffectiveServiceForDomainTest(t, sourceRepo, overlayRepo)

	items, err := service.List(ctx, CatalogEffectiveListFilter{})
	if err != nil {
		t.Fatalf("expected effective list query to succeed, got %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 effective items, got %d", len(items))
	}

	itemsByID := make(map[string]CatalogItem, len(items))
	for _, item := range items {
		itemsByID[item.ID] = item
		if item.MetadataWritable != true {
			t.Fatalf("expected metadata_writable=true for %q, got false", item.ID)
		}
		if item.ReadOnly != !item.ContentWritable {
			t.Fatalf("expected read_only to remain backward-compatible for %q", item.ID)
		}
	}

	if itemsByID[BuildSkillCatalogItemID("repo-a/git-item")].ContentWritable {
		t.Fatalf("expected git item content_writable=false")
	}
	if !itemsByID[BuildSkillCatalogItemID("repo-a/git-item")].ReadOnly {
		t.Fatalf("expected git item read_only=true")
	}
	if !itemsByID[BuildSkillCatalogItemID("local-item")].ContentWritable {
		t.Fatalf("expected local item content_writable=true")
	}
	if !itemsByID[BuildSkillCatalogItemID("file-import/item")].ContentWritable {
		t.Fatalf("expected file-import item content_writable=true")
	}
}

func TestCatalogEffectiveService_List_UsesDeterministicOrderingFiltersAndExcludesTombstones(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)

	repoName := "repo-a"
	parentSkillID := "repo-a/alpha"
	resourcePath := "imports/prompts/system.md"
	syncedAt := time.Date(2026, time.March, 4, 23, 45, 0, 0, time.UTC)
	deletedAt := time.Date(2026, time.March, 4, 23, 50, 0, 0, time.UTC)

	promptID := BuildPromptCatalogItemID(parentSkillID, resourcePath)
	skillAID := BuildSkillCatalogItemID("alpha")
	skillDeletedID := BuildSkillCatalogItemID("beta")

	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           promptID,
		Classifier:       persistence.CatalogClassifierPrompt,
		SourceType:       persistence.CatalogSourceTypeGit,
		SourceRepo:       &repoName,
		ParentSkillID:    &parentSkillID,
		ResourcePath:     &resourcePath,
		Name:             "system.md",
		Description:      "prompt",
		Content:          "prompt content",
		ContentHash:      buildCatalogContentHash("prompt content"),
		ContentWritable:  false,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           skillAID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "alpha",
		Description:      "alpha skill",
		Content:          "alpha content",
		ContentHash:      buildCatalogContentHash("alpha content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           skillDeletedID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "beta",
		Description:      "deleted beta",
		Content:          "beta content",
		ContentHash:      buildCatalogContentHash("beta content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
		DeletedAt:        &deletedAt,
	})

	service := newCatalogEffectiveServiceForDomainTest(t, sourceRepo, overlayRepo)

	items, err := service.List(ctx, CatalogEffectiveListFilter{})
	if err != nil {
		t.Fatalf("expected effective list query to succeed, got %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 non-deleted items, got %d", len(items))
	}
	if items[0].ID != promptID || items[1].ID != skillAID {
		t.Fatalf("expected deterministic ordering by item_id, got %q then %q", items[0].ID, items[1].ID)
	}

	skillClassifier := CatalogClassifierSkill
	skillItems, err := service.List(ctx, CatalogEffectiveListFilter{Classifier: &skillClassifier})
	if err != nil {
		t.Fatalf("expected classifier-filtered list query to succeed, got %v", err)
	}
	if len(skillItems) != 1 || skillItems[0].ID != skillAID {
		t.Fatalf("expected only visible skill row, got %+v", skillItems)
	}

	localSourceType := persistence.CatalogSourceTypeLocal
	localVisible, err := service.List(ctx, CatalogEffectiveListFilter{SourceType: &localSourceType})
	if err != nil {
		t.Fatalf("expected source-type-filtered list query to succeed, got %v", err)
	}
	if len(localVisible) != 1 || localVisible[0].ID != skillAID {
		t.Fatalf("expected only non-deleted local row, got %+v", localVisible)
	}

	localIncludingDeleted, err := service.List(ctx, CatalogEffectiveListFilter{
		SourceType:     &localSourceType,
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("expected include_deleted local list query to succeed, got %v", err)
	}
	if len(localIncludingDeleted) != 2 {
		t.Fatalf("expected deleted local row to be included, got %d", len(localIncludingDeleted))
	}
	if localIncludingDeleted[0].ID != skillAID || localIncludingDeleted[1].ID != skillDeletedID {
		t.Fatalf("expected deterministic local ordering with deleted rows, got %+v", localIncludingDeleted)
	}
}

func newCatalogEffectiveServiceForDomainTest(
	t *testing.T,
	sourceRepo *persistence.CatalogSourceRepository,
	overlayRepo *persistence.CatalogMetadataOverlayRepository,
) *CatalogEffectiveService {
	t.Helper()

	service, err := NewCatalogEffectiveService(sourceRepo, overlayRepo)
	if err != nil {
		t.Fatalf("expected effective catalog service creation to succeed, got %v", err)
	}
	return service
}
