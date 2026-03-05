package domain

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

func TestNormalizeCatalogLegacyLabelToTagKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		valid    bool
	}{
		{name: "trim and lowercase", input: "  Backend  ", expected: "backend", valid: true},
		{name: "spaces and underscore collapse", input: "Data_Science Team", expected: "data-science-team", valid: true},
		{name: "punctuation collapse", input: "Data/ML+AI", expected: "data-ml-ai", valid: true},
		{name: "drop leading separators", input: "***Platform***", expected: "platform", valid: true},
		{name: "empty after normalization invalid", input: "---", expected: "", valid: false},
		{name: "blank invalid", input: "   ", expected: "", valid: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, ok := NormalizeCatalogLegacyLabelToTagKey(tc.input)
			if ok != tc.valid {
				t.Fatalf("expected valid=%t, got %t for input %q", tc.valid, ok, tc.input)
			}
			if actual != tc.expected {
				t.Fatalf("expected normalized key %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestNewCatalogTaxonomyLegacyLabelBackfillService_WithNilDependencies_ReturnsError(t *testing.T) {
	db, _ := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	if _, err := NewCatalogTaxonomyLegacyLabelBackfillService(
		nil,
		overlayRepo,
		tagRepo,
		tagAssignmentRepo,
		CatalogTaxonomyLegacyLabelBackfillServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil source repository error, got nil")
	}
	if _, err := NewCatalogTaxonomyLegacyLabelBackfillService(
		sourceRepo,
		nil,
		tagRepo,
		tagAssignmentRepo,
		CatalogTaxonomyLegacyLabelBackfillServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil overlay repository error, got nil")
	}
	if _, err := NewCatalogTaxonomyLegacyLabelBackfillService(
		sourceRepo,
		overlayRepo,
		nil,
		tagAssignmentRepo,
		CatalogTaxonomyLegacyLabelBackfillServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil tag repository error, got nil")
	}
	if _, err := NewCatalogTaxonomyLegacyLabelBackfillService(
		sourceRepo,
		overlayRepo,
		tagRepo,
		nil,
		CatalogTaxonomyLegacyLabelBackfillServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil tag assignment repository error, got nil")
	}
}

func TestCatalogTaxonomyLegacyLabelBackfillService_BackfillFromLegacyLabels_NilReceiverReturnsError(t *testing.T) {
	var service *CatalogTaxonomyLegacyLabelBackfillService
	if _, err := service.BackfillFromLegacyLabels(nil); err == nil {
		t.Fatalf("expected nil service receiver error, got nil")
	}
}

func TestCatalogTaxonomyLegacyLabelBackfillService_BackfillFromLegacyLabels_MigratesAndIsIdempotent(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	firstBackfillAt := time.Date(2026, time.March, 5, 4, 0, 0, 0, time.UTC)
	service, err := NewCatalogTaxonomyLegacyLabelBackfillService(
		sourceRepo,
		overlayRepo,
		tagRepo,
		tagAssignmentRepo,
		CatalogTaxonomyLegacyLabelBackfillServiceOptions{
			Now: func() time.Time {
				return firstBackfillAt
			},
		},
	)
	if err != nil {
		t.Fatalf("expected backfill service creation to succeed, got %v", err)
	}

	itemAID := BuildSkillCatalogItemID("legacy-a")
	itemBID := BuildSkillCatalogItemID("legacy-b")
	itemCID := BuildSkillCatalogItemID("legacy-c")
	syncedAt := time.Date(2026, time.March, 5, 3, 30, 0, 0, time.UTC)

	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           itemAID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "legacy-a",
		Description:      "legacy a",
		Content:          "legacy a content",
		ContentHash:      buildCatalogContentHash("legacy a content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           itemBID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "legacy-b",
		Description:      "legacy b",
		Content:          "legacy b content",
		ContentHash:      buildCatalogContentHash("legacy b content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           itemCID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "legacy-c",
		Description:      "legacy c",
		Content:          "legacy c content",
		ContentHash:      buildCatalogContentHash("legacy c content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})

	if err := overlayRepo.Upsert(ctx, persistence.CatalogMetadataOverlayRow{
		ItemID:    itemAID,
		Labels:    []string{"Backend", "backend", "CLI Tools"},
		UpdatedAt: syncedAt,
	}); err != nil {
		t.Fatalf("expected itemA overlay upsert to succeed, got %v", err)
	}
	if err := overlayRepo.Upsert(ctx, persistence.CatalogMetadataOverlayRow{
		ItemID:    itemBID,
		Labels:    []string{"cli-tools", "Data/ML"},
		UpdatedAt: syncedAt,
	}); err != nil {
		t.Fatalf("expected itemB overlay upsert to succeed, got %v", err)
	}

	if err := tagRepo.Create(ctx, persistence.CatalogTagRow{
		TagID:     "tag-platform-backend",
		Key:       "backend",
		Name:      "Backend",
		Active:    true,
		CreatedAt: syncedAt,
		UpdatedAt: syncedAt,
	}); err != nil {
		t.Fatalf("expected pre-seeded backend tag create to succeed, got %v", err)
	}

	report, err := service.BackfillFromLegacyLabels(ctx)
	if err != nil {
		t.Fatalf("expected first backfill run to succeed, got %v", err)
	}
	if report.ItemsScanned != 3 {
		t.Fatalf("expected items_scanned=3, got %d", report.ItemsScanned)
	}
	if report.ItemsWithLegacyLabels != 2 {
		t.Fatalf("expected items_with_legacy_labels=2, got %d", report.ItemsWithLegacyLabels)
	}
	if report.TagsCreated != 2 {
		t.Fatalf("expected tags_created=2 (cli-tools,data-ml), got %d", report.TagsCreated)
	}
	if report.ItemAssignmentsUpdated != 2 {
		t.Fatalf("expected item_assignments_updated=2, got %d", report.ItemAssignmentsUpdated)
	}
	if len(report.NormalizationCollisions) == 0 {
		t.Fatalf("expected normalization collisions to include backend/cli-tools variants")
	}

	tags, err := tagRepo.List(ctx, persistence.CatalogTagListFilter{Keys: []string{"backend", "cli-tools", "data-ml"}})
	if err != nil {
		t.Fatalf("expected taxonomy tag list to succeed, got %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("expected 3 taxonomy tags after backfill, got %d", len(tags))
	}

	tagIDByKey := make(map[string]string, len(tags))
	for _, tag := range tags {
		tagIDByKey[tag.Key] = tag.TagID
	}
	if tagIDByKey["backend"] != "tag-platform-backend" {
		t.Fatalf("expected existing backend tag id to be preserved, got %q", tagIDByKey["backend"])
	}

	assertCatalogTagAssignmentsForItem(t, ctx, tagAssignmentRepo, itemAID, []string{
		tagIDByKey["backend"],
		tagIDByKey["cli-tools"],
	})
	assertCatalogTagAssignmentsForItem(t, ctx, tagAssignmentRepo, itemBID, []string{
		tagIDByKey["cli-tools"],
		tagIDByKey["data-ml"],
	})
	assertCatalogTagAssignmentsForItem(t, ctx, tagAssignmentRepo, itemCID, []string{})

	secondBackfillAt := time.Date(2026, time.March, 5, 5, 0, 0, 0, time.UTC)
	service.now = func() time.Time {
		return secondBackfillAt
	}

	secondReport, err := service.BackfillFromLegacyLabels(ctx)
	if err != nil {
		t.Fatalf("expected second backfill run to succeed, got %v", err)
	}
	if secondReport.TagsCreated != 0 {
		t.Fatalf("expected idempotent run to create zero tags, got %d", secondReport.TagsCreated)
	}
	if secondReport.ItemAssignmentsUpdated != 0 {
		t.Fatalf("expected idempotent run to update zero assignments, got %d", secondReport.ItemAssignmentsUpdated)
	}

	assertCatalogTagAssignmentsForItem(t, ctx, tagAssignmentRepo, itemAID, []string{
		tagIDByKey["backend"],
		tagIDByKey["cli-tools"],
	})
	assertCatalogTagAssignmentsForItem(t, ctx, tagAssignmentRepo, itemBID, []string{
		tagIDByKey["cli-tools"],
		tagIDByKey["data-ml"],
	})
}

func TestCatalogTaxonomyLegacyLabelBackfillService_BackfillFromLegacyLabels_UsesHashedTagIDOnBaseCollision(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	overlayRepo := newCatalogOverlayRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	backfillAt := time.Date(2026, time.March, 5, 6, 0, 0, 0, time.UTC)
	service, err := NewCatalogTaxonomyLegacyLabelBackfillService(
		sourceRepo,
		overlayRepo,
		tagRepo,
		tagAssignmentRepo,
		CatalogTaxonomyLegacyLabelBackfillServiceOptions{
			Now: func() time.Time {
				return backfillAt
			},
		},
	)
	if err != nil {
		t.Fatalf("expected backfill service creation to succeed, got %v", err)
	}

	itemID := BuildSkillCatalogItemID("legacy-collision")
	syncedAt := time.Date(2026, time.March, 5, 5, 30, 0, 0, time.UTC)
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "legacy-collision",
		Description:      "legacy collision",
		Content:          "legacy collision content",
		ContentHash:      buildCatalogContentHash("legacy collision content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	})
	if err := overlayRepo.Upsert(ctx, persistence.CatalogMetadataOverlayRow{
		ItemID:    itemID,
		Labels:    []string{"CLI Tools"},
		UpdatedAt: syncedAt,
	}); err != nil {
		t.Fatalf("expected overlay upsert to succeed, got %v", err)
	}

	if err := tagRepo.Create(ctx, persistence.CatalogTagRow{
		TagID:     "tag-cli-tools",
		Key:       "occupied-key",
		Name:      "Occupied",
		Active:    true,
		CreatedAt: syncedAt,
		UpdatedAt: syncedAt,
	}); err != nil {
		t.Fatalf("expected occupied base tag id fixture insert to succeed, got %v", err)
	}

	report, err := service.BackfillFromLegacyLabels(ctx)
	if err != nil {
		t.Fatalf("expected backfill with base id collision to succeed, got %v", err)
	}
	if report.TagsCreated != 1 {
		t.Fatalf("expected exactly one created tag, got %d", report.TagsCreated)
	}
	if report.ItemAssignmentsUpdated != 1 {
		t.Fatalf("expected exactly one assignment update, got %d", report.ItemAssignmentsUpdated)
	}

	createdTag, err := tagRepo.GetByKey(ctx, "cli-tools")
	if err != nil {
		t.Fatalf("expected created cli-tools tag lookup to succeed, got %v", err)
	}
	expectedTagID := buildCatalogLegacyBackfillTagID("cli-tools", true)
	if createdTag.TagID != expectedTagID {
		t.Fatalf("expected collision fallback tag_id %q, got %q", expectedTagID, createdTag.TagID)
	}
}

func assertCatalogTagAssignmentsForItem(
	t *testing.T,
	ctx context.Context,
	repo *persistence.CatalogItemTagAssignmentRepository,
	itemID string,
	expectedTagIDs []string,
) {
	t.Helper()

	rows, err := repo.ListByItemID(ctx, itemID)
	if err != nil {
		t.Fatalf("expected list tag assignments for item %q to succeed, got %v", itemID, err)
	}

	actualTagIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		actualTagIDs = append(actualTagIDs, row.TagID)
	}
	slices.Sort(actualTagIDs)

	expected := append([]string{}, expectedTagIDs...)
	slices.Sort(expected)

	if !slices.Equal(actualTagIDs, expected) {
		t.Fatalf("expected tag assignments for item %q to be %+v, got %+v", itemID, expected, actualTagIDs)
	}
}
