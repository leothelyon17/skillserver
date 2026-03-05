package persistence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewCatalogItemTaxonomyAssignmentRepository_WithNilExecutor_ReturnsError(t *testing.T) {
	_, err := NewCatalogItemTaxonomyAssignmentRepository(nil)
	if err == nil {
		t.Fatalf("expected nil executor error, got nil")
	}
}

func TestNewCatalogItemTagAssignmentRepository_WithNilExecutor_ReturnsError(t *testing.T) {
	_, err := NewCatalogItemTagAssignmentRepository(nil)
	if err == nil {
		t.Fatalf("expected nil executor error, got nil")
	}
}

func TestCatalogItemTaxonomyAssignmentRepository_UpsertGetListAndFilterPaths(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	domainRepo := newCatalogDomainRepositoryForTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForTest(t, db)
	repo := newCatalogItemTaxonomyAssignmentRepositoryForTest(t, db)

	seedTaxonomyAssignmentFixtureSourceItems(t, ctx, sourceRepo)
	seedTaxonomyAssignmentFixtureDomainsAndSubdomains(t, ctx, domainRepo, subdomainRepo)

	firstUpdatedAt := time.Date(2026, time.March, 4, 15, 0, 0, 0, time.UTC)
	if err := repo.Upsert(ctx, CatalogItemTaxonomyAssignmentRow{
		ItemID:               "skill:item-a",
		PrimaryDomainID:      stringPointer("domain-platform"),
		PrimarySubdomainID:   stringPointer("subdomain-platform-api"),
		SecondaryDomainID:    stringPointer("domain-observability"),
		SecondarySubdomainID: stringPointer("subdomain-observability-metrics"),
		UpdatedAt:            firstUpdatedAt,
		UpdatedBy:            stringPointer("tester-a"),
	}); err != nil {
		t.Fatalf("expected taxonomy assignment upsert for item-a to succeed, got %v", err)
	}

	if err := repo.Upsert(ctx, CatalogItemTaxonomyAssignmentRow{
		ItemID:          "skill:item-b",
		PrimaryDomainID: stringPointer("domain-observability"),
		UpdatedAt:       firstUpdatedAt.Add(1 * time.Minute),
		UpdatedBy:       stringPointer("tester-b"),
	}); err != nil {
		t.Fatalf("expected taxonomy assignment upsert for item-b to succeed, got %v", err)
	}

	secondUpdatedAt := firstUpdatedAt.Add(2 * time.Hour)
	if err := repo.Upsert(ctx, CatalogItemTaxonomyAssignmentRow{
		ItemID:             "skill:item-a",
		PrimaryDomainID:    stringPointer("domain-platform"),
		PrimarySubdomainID: stringPointer("subdomain-platform-api"),
		SecondaryDomainID:  stringPointer("domain-observability"),
		UpdatedAt:          secondUpdatedAt,
		UpdatedBy:          stringPointer("tester-update"),
	}); err != nil {
		t.Fatalf("expected taxonomy assignment update for item-a to succeed, got %v", err)
	}

	itemA, err := repo.GetByItemID(ctx, "skill:item-a")
	if err != nil {
		t.Fatalf("expected get taxonomy assignment for item-a to succeed, got %v", err)
	}
	if itemA.PrimaryDomainID == nil || *itemA.PrimaryDomainID != "domain-platform" {
		t.Fatalf("expected item-a primary_domain_id to be domain-platform, got %+v", itemA.PrimaryDomainID)
	}
	if itemA.SecondarySubdomainID != nil {
		t.Fatalf("expected item-a secondary_subdomain_id to be cleared on upsert update, got %+v", itemA.SecondarySubdomainID)
	}
	if itemA.UpdatedBy == nil || *itemA.UpdatedBy != "tester-update" {
		t.Fatalf("expected item-a updated_by to round-trip, got %+v", itemA.UpdatedBy)
	}
	if !itemA.UpdatedAt.Equal(secondUpdatedAt) {
		t.Fatalf("expected item-a updated_at %s, got %s", secondUpdatedAt, itemA.UpdatedAt)
	}

	allAssignments, err := repo.List(ctx, CatalogItemTaxonomyAssignmentListFilter{})
	if err != nil {
		t.Fatalf("expected taxonomy assignment list to succeed, got %v", err)
	}
	if len(allAssignments) != 2 {
		t.Fatalf("expected 2 taxonomy assignments, got %d", len(allAssignments))
	}
	if allAssignments[0].ItemID != "skill:item-a" || allAssignments[1].ItemID != "skill:item-b" {
		t.Fatalf("expected deterministic ordering by item_id, got %+v", allAssignments)
	}

	filterDomain, err := repo.List(ctx, CatalogItemTaxonomyAssignmentListFilter{
		DomainID: stringPointer("domain-observability"),
	})
	if err != nil {
		t.Fatalf("expected taxonomy assignment list by domain to succeed, got %v", err)
	}
	if len(filterDomain) != 2 {
		t.Fatalf("expected 2 assignments matching domain filter, got %d", len(filterDomain))
	}

	filterSubdomain, err := repo.List(ctx, CatalogItemTaxonomyAssignmentListFilter{
		SubdomainID: stringPointer("subdomain-platform-api"),
	})
	if err != nil {
		t.Fatalf("expected taxonomy assignment list by subdomain to succeed, got %v", err)
	}
	if len(filterSubdomain) != 1 || filterSubdomain[0].ItemID != "skill:item-a" {
		t.Fatalf("expected one item for subdomain filter, got %+v", filterSubdomain)
	}

	filterPrimaryDomain, err := repo.List(ctx, CatalogItemTaxonomyAssignmentListFilter{
		PrimaryDomainID: stringPointer("domain-observability"),
	})
	if err != nil {
		t.Fatalf("expected taxonomy assignment list by primary domain to succeed, got %v", err)
	}
	if len(filterPrimaryDomain) != 1 || filterPrimaryDomain[0].ItemID != "skill:item-b" {
		t.Fatalf("expected one item for primary domain filter, got %+v", filterPrimaryDomain)
	}

	_, err = repo.GetByItemID(ctx, "skill:missing-item")
	if !errors.Is(err, ErrCatalogItemTaxonomyAssignmentNotFound) {
		t.Fatalf("expected missing assignment not found error, got %v", err)
	}
}

func TestCatalogItemTaxonomyAssignmentRepository_Upsert_WithInvalidForeignKey_ReturnsConstraintError(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	repo := newCatalogItemTaxonomyAssignmentRepositoryForTest(t, db)

	mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
		ItemID:           "skill:item-no-domain",
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "item-no-domain",
		Description:      "item-no-domain",
		Content:          "content",
		ContentHash:      "sha256:item-no-domain",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 16, 0, 0, 0, time.UTC),
	})

	err := repo.Upsert(ctx, CatalogItemTaxonomyAssignmentRow{
		ItemID:          "skill:item-no-domain",
		PrimaryDomainID: stringPointer("domain-missing"),
		UpdatedAt:       time.Date(2026, time.March, 4, 16, 1, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected invalid foreign key upsert to fail, got nil")
	}
	if !strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY") {
		t.Fatalf("expected invalid foreign key upsert error to mention FOREIGN KEY, got %v", err)
	}

	list, err := repo.List(ctx, CatalogItemTaxonomyAssignmentListFilter{ItemID: "skill:item-no-domain"})
	if err != nil {
		t.Fatalf("expected taxonomy assignment list after failed upsert to succeed, got %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no persisted assignment rows after failed upsert, got %+v", list)
	}
}

func TestCatalogItemTaxonomyAssignmentRepository_List_InvalidFilter_ReturnsError(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogItemTaxonomyAssignmentRepositoryForTest(t, db)

	_, err := repo.List(ctx, CatalogItemTaxonomyAssignmentListFilter{
		ItemID:  "skill:item-a",
		ItemIDs: []string{"skill:item-b"},
	})
	if err == nil {
		t.Fatalf("expected invalid assignment filter error, got nil")
	}
}

func TestCatalogItemTagAssignmentRepository_ReplaceForItemID_ListAndIdempotency(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	tagRepo := newCatalogTagRepositoryForTest(t, db)
	repo := newCatalogItemTagAssignmentRepositoryForTest(t, db)

	seedTagAssignmentFixtureSources(t, ctx, sourceRepo)
	seedTagAssignmentFixtureTags(t, ctx, tagRepo)

	createdAtA := time.Date(2026, time.March, 4, 17, 0, 0, 0, time.UTC)
	if err := repo.ReplaceForItemID(ctx, "skill:item-a", []string{"tag-b", "tag-a", "tag-a"}, createdAtA); err != nil {
		t.Fatalf("expected first replace for item-a to succeed, got %v", err)
	}

	rowsAfterFirstReplace, err := repo.ListByItemID(ctx, "skill:item-a")
	if err != nil {
		t.Fatalf("expected list by item after first replace to succeed, got %v", err)
	}
	if len(rowsAfterFirstReplace) != 2 {
		t.Fatalf("expected deduped tags length 2 after replace, got %d", len(rowsAfterFirstReplace))
	}
	if rowsAfterFirstReplace[0].TagID != "tag-a" || rowsAfterFirstReplace[1].TagID != "tag-b" {
		t.Fatalf("expected deterministic ordering by tag_id, got %+v", rowsAfterFirstReplace)
	}

	createdAtByTag := map[string]time.Time{}
	for _, row := range rowsAfterFirstReplace {
		createdAtByTag[row.TagID] = row.CreatedAt
	}

	createdAtB := createdAtA.Add(1 * time.Hour)
	if err := repo.ReplaceForItemID(ctx, "skill:item-a", []string{"tag-b", "tag-a"}, createdAtB); err != nil {
		t.Fatalf("expected second idempotent replace for item-a to succeed, got %v", err)
	}

	rowsAfterSecondReplace, err := repo.ListByItemID(ctx, "skill:item-a")
	if err != nil {
		t.Fatalf("expected list by item after second replace to succeed, got %v", err)
	}
	if len(rowsAfterSecondReplace) != 2 {
		t.Fatalf("expected 2 tags after idempotent replace, got %d", len(rowsAfterSecondReplace))
	}
	for _, row := range rowsAfterSecondReplace {
		if !row.CreatedAt.Equal(createdAtByTag[row.TagID]) {
			t.Fatalf("expected created_at for existing tag %q to remain stable, got %s want %s", row.TagID, row.CreatedAt, createdAtByTag[row.TagID])
		}
	}

	createdAtC := createdAtA.Add(2 * time.Hour)
	if err := repo.ReplaceForItemID(ctx, "skill:item-a", []string{"tag-a", "tag-c"}, createdAtC); err != nil {
		t.Fatalf("expected third replace for item-a to succeed, got %v", err)
	}

	rowsAfterThirdReplace, err := repo.ListByItemID(ctx, "skill:item-a")
	if err != nil {
		t.Fatalf("expected list by item after third replace to succeed, got %v", err)
	}
	if len(rowsAfterThirdReplace) != 2 {
		t.Fatalf("expected 2 tags after third replace, got %d", len(rowsAfterThirdReplace))
	}
	if rowsAfterThirdReplace[0].TagID != "tag-a" || rowsAfterThirdReplace[1].TagID != "tag-c" {
		t.Fatalf("expected tag-a and tag-c after third replace, got %+v", rowsAfterThirdReplace)
	}
	if !rowsAfterThirdReplace[0].CreatedAt.Equal(createdAtByTag["tag-a"]) {
		t.Fatalf("expected existing tag-a created_at to stay stable, got %s want %s", rowsAfterThirdReplace[0].CreatedAt, createdAtByTag["tag-a"])
	}
	if !rowsAfterThirdReplace[1].CreatedAt.Equal(createdAtC) {
		t.Fatalf("expected newly inserted tag-c created_at %s, got %s", createdAtC, rowsAfterThirdReplace[1].CreatedAt)
	}

	if err := repo.ReplaceForItemID(ctx, "skill:item-a", nil, createdAtC.Add(1*time.Hour)); err != nil {
		t.Fatalf("expected clear replace for item-a to succeed, got %v", err)
	}

	rowsAfterClear, err := repo.ListByItemID(ctx, "skill:item-a")
	if err != nil {
		t.Fatalf("expected list by item after clear to succeed, got %v", err)
	}
	if len(rowsAfterClear) != 0 {
		t.Fatalf("expected no tags after clear replace, got %+v", rowsAfterClear)
	}
}

func TestCatalogItemTagAssignmentRepository_ReplaceForItemID_WithInvalidTag_RollsBack(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	tagRepo := newCatalogTagRepositoryForTest(t, db)
	repo := newCatalogItemTagAssignmentRepositoryForTest(t, db)

	seedTagAssignmentFixtureSources(t, ctx, sourceRepo)
	seedTagAssignmentFixtureTags(t, ctx, tagRepo)

	baselineCreatedAt := time.Date(2026, time.March, 4, 18, 0, 0, 0, time.UTC)
	if err := repo.ReplaceForItemID(ctx, "skill:item-b", []string{"tag-a"}, baselineCreatedAt); err != nil {
		t.Fatalf("expected baseline replace for item-b to succeed, got %v", err)
	}

	err := repo.ReplaceForItemID(
		ctx,
		"skill:item-b",
		[]string{"tag-a", "tag-missing"},
		baselineCreatedAt.Add(1*time.Hour),
	)
	if err == nil {
		t.Fatalf("expected replace with missing tag to fail, got nil")
	}
	if !strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY") {
		t.Fatalf("expected replace with missing tag error to mention FOREIGN KEY, got %v", err)
	}

	rowsAfterFailure, err := repo.ListByItemID(ctx, "skill:item-b")
	if err != nil {
		t.Fatalf("expected list by item after failed replace to succeed, got %v", err)
	}
	if len(rowsAfterFailure) != 1 || rowsAfterFailure[0].TagID != "tag-a" {
		t.Fatalf("expected failed replace to rollback and keep baseline assignment, got %+v", rowsAfterFailure)
	}
	if !rowsAfterFailure[0].CreatedAt.Equal(baselineCreatedAt) {
		t.Fatalf("expected rollback to preserve original created_at %s, got %s", baselineCreatedAt, rowsAfterFailure[0].CreatedAt)
	}
}

func TestCatalogItemTagAssignmentRepository_ListItemIDsByTagIDs_MatchAnyAndAll(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	tagRepo := newCatalogTagRepositoryForTest(t, db)
	repo := newCatalogItemTagAssignmentRepositoryForTest(t, db)

	seedTagAssignmentFixtureSources(t, ctx, sourceRepo)
	seedTagAssignmentFixtureTags(t, ctx, tagRepo)

	createdAt := time.Date(2026, time.March, 4, 19, 0, 0, 0, time.UTC)
	if err := repo.ReplaceForItemID(ctx, "skill:item-a", []string{"tag-a", "tag-b"}, createdAt); err != nil {
		t.Fatalf("expected replace for item-a to succeed, got %v", err)
	}
	if err := repo.ReplaceForItemID(ctx, "skill:item-b", []string{"tag-a"}, createdAt); err != nil {
		t.Fatalf("expected replace for item-b to succeed, got %v", err)
	}
	if err := repo.ReplaceForItemID(ctx, "skill:item-c", []string{"tag-b", "tag-c"}, createdAt); err != nil {
		t.Fatalf("expected replace for item-c to succeed, got %v", err)
	}

	matchAny, err := repo.ListItemIDsByTagIDs(ctx, []string{"tag-c", "tag-a"}, false)
	if err != nil {
		t.Fatalf("expected match-any lookup to succeed, got %v", err)
	}
	if len(matchAny) != 3 || matchAny[0] != "skill:item-a" || matchAny[1] != "skill:item-b" || matchAny[2] != "skill:item-c" {
		t.Fatalf("expected deterministic match-any result set, got %+v", matchAny)
	}

	matchAllAB, err := repo.ListItemIDsByTagIDs(ctx, []string{"tag-a", "tag-b"}, true)
	if err != nil {
		t.Fatalf("expected match-all lookup for tag-a+tag-b to succeed, got %v", err)
	}
	if len(matchAllAB) != 1 || matchAllAB[0] != "skill:item-a" {
		t.Fatalf("expected only item-a for match-all tag-a+tag-b, got %+v", matchAllAB)
	}

	matchAllBC, err := repo.ListItemIDsByTagIDs(ctx, []string{"tag-b", "tag-c"}, true)
	if err != nil {
		t.Fatalf("expected match-all lookup for tag-b+tag-c to succeed, got %v", err)
	}
	if len(matchAllBC) != 1 || matchAllBC[0] != "skill:item-c" {
		t.Fatalf("expected only item-c for match-all tag-b+tag-c, got %+v", matchAllBC)
	}
}

func TestCatalogItemTagAssignmentRepository_List_InvalidFilter_ReturnsError(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogItemTagAssignmentRepositoryForTest(t, db)

	_, err := repo.List(ctx, CatalogItemTagAssignmentListFilter{
		ItemID:  "skill:item-a",
		ItemIDs: []string{"skill:item-b"},
	})
	if err == nil {
		t.Fatalf("expected invalid item filter error, got nil")
	}

	_, err = repo.List(ctx, CatalogItemTagAssignmentListFilter{
		TagID:  "tag-a",
		TagIDs: []string{"tag-b"},
	})
	if err == nil {
		t.Fatalf("expected invalid tag filter error, got nil")
	}
}

func seedTaxonomyAssignmentFixtureSourceItems(
	t *testing.T,
	ctx context.Context,
	sourceRepo *CatalogSourceRepository,
) {
	t.Helper()

	for _, itemID := range []string{"skill:item-a", "skill:item-b", "skill:item-c"} {
		mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
			ItemID:           itemID,
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             strings.TrimPrefix(itemID, "skill:"),
			Description:      "taxonomy fixture",
			Content:          "content",
			ContentHash:      "sha256:" + itemID,
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 4, 14, 0, 0, 0, time.UTC),
		})
	}
}

func seedTaxonomyAssignmentFixtureDomainsAndSubdomains(
	t *testing.T,
	ctx context.Context,
	domainRepo *CatalogDomainRepository,
	subdomainRepo *CatalogSubdomainRepository,
) {
	t.Helper()

	mustCreateCatalogDomainRow(t, ctx, domainRepo, CatalogDomainRow{
		DomainID: "domain-platform",
		Key:      "platform",
		Name:     "Platform",
		Active:   true,
	})
	mustCreateCatalogDomainRow(t, ctx, domainRepo, CatalogDomainRow{
		DomainID: "domain-observability",
		Key:      "observability",
		Name:     "Observability",
		Active:   true,
	})

	mustCreateCatalogSubdomainRow(t, ctx, subdomainRepo, CatalogSubdomainRow{
		SubdomainID: "subdomain-platform-api",
		DomainID:    "domain-platform",
		Key:         "api",
		Name:        "API",
		Active:      true,
	})
	mustCreateCatalogSubdomainRow(t, ctx, subdomainRepo, CatalogSubdomainRow{
		SubdomainID: "subdomain-observability-metrics",
		DomainID:    "domain-observability",
		Key:         "metrics",
		Name:        "Metrics",
		Active:      true,
	})
}

func seedTagAssignmentFixtureSources(
	t *testing.T,
	ctx context.Context,
	sourceRepo *CatalogSourceRepository,
) {
	t.Helper()

	for _, itemID := range []string{"skill:item-a", "skill:item-b", "skill:item-c"} {
		mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
			ItemID:           itemID,
			Classifier:       CatalogClassifierSkill,
			SourceType:       CatalogSourceTypeLocal,
			Name:             strings.TrimPrefix(itemID, "skill:"),
			Description:      "tag assignment fixture",
			Content:          "content",
			ContentHash:      "sha256:" + itemID,
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 4, 14, 0, 0, 0, time.UTC),
		})
	}
}

func seedTagAssignmentFixtureTags(t *testing.T, ctx context.Context, tagRepo *CatalogTagRepository) {
	t.Helper()

	for _, tag := range []CatalogTagRow{
		{TagID: "tag-a", Key: "a", Name: "A", Active: true},
		{TagID: "tag-b", Key: "b", Name: "B", Active: true},
		{TagID: "tag-c", Key: "c", Name: "C", Active: true},
	} {
		mustCreateCatalogTagRow(t, ctx, tagRepo, tag)
	}
}
