package persistence

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewCatalogDomainRepository_WithNilExecutor_ReturnsError(t *testing.T) {
	_, err := NewCatalogDomainRepository(nil)
	if err == nil {
		t.Fatalf("expected nil executor error, got nil")
	}
}

func TestNewCatalogSubdomainRepository_WithNilExecutor_ReturnsError(t *testing.T) {
	_, err := NewCatalogSubdomainRepository(nil)
	if err == nil {
		t.Fatalf("expected nil executor error, got nil")
	}
}

func TestNewCatalogTagRepository_WithNilExecutor_ReturnsError(t *testing.T) {
	_, err := NewCatalogTagRepository(nil)
	if err == nil {
		t.Fatalf("expected nil executor error, got nil")
	}
}

func TestCatalogDomainRepository_CRUDAndDeterministicListOrdering(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogDomainRepositoryForTest(t, db)

	createdAt := time.Date(2026, time.March, 4, 10, 0, 0, 0, time.UTC)
	mustCreateCatalogDomainRow(t, ctx, repo, CatalogDomainRow{
		DomainID:    "domain-platform",
		Key:         "platform",
		Name:        "Platform",
		Description: "Platform domain",
		Active:      true,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	})
	mustCreateCatalogDomainRow(t, ctx, repo, CatalogDomainRow{
		DomainID:    "domain-observability",
		Key:         "observability",
		Name:        "Observability",
		Description: "Observability domain",
		Active:      true,
		CreatedAt:   createdAt.Add(1 * time.Minute),
		UpdatedAt:   createdAt.Add(1 * time.Minute),
	})

	byID, err := repo.GetByDomainID(ctx, "domain-platform")
	if err != nil {
		t.Fatalf("expected get domain by id to succeed, got %v", err)
	}
	if byID.Key != "platform" {
		t.Fatalf("expected domain key %q, got %q", "platform", byID.Key)
	}

	byKey, err := repo.GetByKey(ctx, "observability")
	if err != nil {
		t.Fatalf("expected get domain by key to succeed, got %v", err)
	}
	if byKey.DomainID != "domain-observability" {
		t.Fatalf("expected domain_id %q, got %q", "domain-observability", byKey.DomainID)
	}

	updatedAt := createdAt.Add(2 * time.Hour)
	updated, err := repo.Update(ctx, CatalogDomainRow{
		DomainID:    "domain-platform",
		Key:         "platform-core",
		Name:        "Platform Core",
		Description: "Platform core domain",
		Active:      false,
		UpdatedAt:   updatedAt,
	})
	if err != nil {
		t.Fatalf("expected domain update to succeed, got %v", err)
	}
	if !updated {
		t.Fatalf("expected domain update to affect one row")
	}

	platformAfterUpdate, err := repo.GetByDomainID(ctx, "domain-platform")
	if err != nil {
		t.Fatalf("expected get updated domain to succeed, got %v", err)
	}
	if platformAfterUpdate.Key != "platform-core" {
		t.Fatalf("expected updated key %q, got %q", "platform-core", platformAfterUpdate.Key)
	}
	if platformAfterUpdate.Active {
		t.Fatalf("expected updated domain to be inactive")
	}
	if !platformAfterUpdate.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updated_at %s, got %s", updatedAt, platformAfterUpdate.UpdatedAt)
	}

	allDomains, err := repo.List(ctx, CatalogDomainListFilter{})
	if err != nil {
		t.Fatalf("expected domain list to succeed, got %v", err)
	}
	if len(allDomains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(allDomains))
	}
	if allDomains[0].Key != "observability" || allDomains[1].Key != "platform-core" {
		t.Fatalf("expected deterministic ordering by key, got %+v", allDomains)
	}

	activeOnly, err := repo.List(ctx, CatalogDomainListFilter{Active: boolPointer(true)})
	if err != nil {
		t.Fatalf("expected active domain list to succeed, got %v", err)
	}
	if len(activeOnly) != 1 || activeOnly[0].DomainID != "domain-observability" {
		t.Fatalf("expected one active domain, got %+v", activeOnly)
	}

	deleted, err := repo.DeleteByDomainID(ctx, "domain-observability")
	if err != nil {
		t.Fatalf("expected domain delete to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected domain delete to affect one row")
	}

	_, err = repo.GetByDomainID(ctx, "domain-observability")
	if !errors.Is(err, ErrCatalogDomainNotFound) {
		t.Fatalf("expected missing domain not found error, got %v", err)
	}
}

func TestCatalogDomainRepository_Create_WithDuplicateKey_ReturnsConstraintError(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogDomainRepositoryForTest(t, db)

	mustCreateCatalogDomainRow(t, ctx, repo, CatalogDomainRow{
		DomainID: "domain-platform-a",
		Key:      "platform",
		Name:     "Platform A",
		Active:   true,
	})

	err := repo.Create(ctx, CatalogDomainRow{
		DomainID: "domain-platform-b",
		Key:      "platform",
		Name:     "Platform B",
		Active:   true,
	})
	if err == nil {
		t.Fatalf("expected duplicate domain key create to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Fatalf("expected duplicate key create to mention unique constraint, got %v", err)
	}
}

func TestCatalogSubdomainRepository_CRUDAndFilterPaths(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	domainRepo := newCatalogDomainRepositoryForTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForTest(t, db)

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
		SubdomainID: "subdomain-platform-tooling",
		DomainID:    "domain-platform",
		Key:         "tooling",
		Name:        "Tooling",
		Active:      true,
	})
	mustCreateCatalogSubdomainRow(t, ctx, subdomainRepo, CatalogSubdomainRow{
		SubdomainID: "subdomain-observability-api",
		DomainID:    "domain-observability",
		Key:         "api",
		Name:        "API",
		Active:      false,
	})

	err := subdomainRepo.Create(ctx, CatalogSubdomainRow{
		SubdomainID: "subdomain-duplicate",
		DomainID:    "domain-platform",
		Key:         "api",
		Name:        "Duplicate",
		Active:      true,
	})
	if err == nil {
		t.Fatalf("expected duplicate subdomain (domain_id,key) create to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Fatalf("expected duplicate subdomain create to mention unique constraint, got %v", err)
	}

	lookup, err := subdomainRepo.GetByDomainIDAndKey(ctx, "domain-platform", "tooling")
	if err != nil {
		t.Fatalf("expected get subdomain by domain+key to succeed, got %v", err)
	}
	if lookup.SubdomainID != "subdomain-platform-tooling" {
		t.Fatalf("expected subdomain_id %q, got %q", "subdomain-platform-tooling", lookup.SubdomainID)
	}

	updated, err := subdomainRepo.Update(ctx, CatalogSubdomainRow{
		SubdomainID: "subdomain-platform-api",
		DomainID:    "domain-platform",
		Key:         "api-gateway",
		Name:        "API Gateway",
		Active:      true,
		UpdatedAt:   time.Date(2026, time.March, 4, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected subdomain update to succeed, got %v", err)
	}
	if !updated {
		t.Fatalf("expected subdomain update to affect one row")
	}

	allSubdomains, err := subdomainRepo.List(ctx, CatalogSubdomainListFilter{})
	if err != nil {
		t.Fatalf("expected subdomain list to succeed, got %v", err)
	}
	if len(allSubdomains) != 3 {
		t.Fatalf("expected 3 subdomains, got %d", len(allSubdomains))
	}
	if allSubdomains[0].SubdomainID != "subdomain-observability-api" ||
		allSubdomains[1].SubdomainID != "subdomain-platform-api" ||
		allSubdomains[2].SubdomainID != "subdomain-platform-tooling" {
		t.Fatalf("expected deterministic subdomain ordering, got %+v", allSubdomains)
	}

	platformSubdomains, err := subdomainRepo.List(ctx, CatalogSubdomainListFilter{DomainID: "domain-platform"})
	if err != nil {
		t.Fatalf("expected platform subdomain list to succeed, got %v", err)
	}
	if len(platformSubdomains) != 2 {
		t.Fatalf("expected 2 platform subdomains, got %d", len(platformSubdomains))
	}

	inactiveOnly, err := subdomainRepo.List(ctx, CatalogSubdomainListFilter{Active: boolPointer(false)})
	if err != nil {
		t.Fatalf("expected inactive subdomain list to succeed, got %v", err)
	}
	if len(inactiveOnly) != 1 || inactiveOnly[0].SubdomainID != "subdomain-observability-api" {
		t.Fatalf("expected one inactive subdomain, got %+v", inactiveOnly)
	}

	deleted, err := subdomainRepo.DeleteBySubdomainID(ctx, "subdomain-observability-api")
	if err != nil {
		t.Fatalf("expected subdomain delete to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected subdomain delete to affect one row")
	}

	_, err = subdomainRepo.GetBySubdomainID(ctx, "subdomain-observability-api")
	if !errors.Is(err, ErrCatalogSubdomainNotFound) {
		t.Fatalf("expected missing subdomain not found error, got %v", err)
	}
}

func TestCatalogTagRepository_CRUDAndDeterministicListOrdering(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	repo := newCatalogTagRepositoryForTest(t, db)

	mustCreateCatalogTagRow(t, ctx, repo, CatalogTagRow{
		TagID:       "tag-backend",
		Key:         "backend",
		Name:        "Backend",
		Description: "Backend tag",
		Color:       "#0052cc",
		Active:      true,
	})
	mustCreateCatalogTagRow(t, ctx, repo, CatalogTagRow{
		TagID:       "tag-automation",
		Key:         "automation",
		Name:        "Automation",
		Description: "Automation tag",
		Color:       "#00875a",
		Active:      true,
	})

	err := repo.Create(ctx, CatalogTagRow{
		TagID:  "tag-duplicate",
		Key:    "automation",
		Name:   "Duplicate",
		Active: true,
	})
	if err == nil {
		t.Fatalf("expected duplicate tag key create to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Fatalf("expected duplicate tag create to mention unique constraint, got %v", err)
	}

	updated, err := repo.Update(ctx, CatalogTagRow{
		TagID:       "tag-backend",
		Key:         "backend-core",
		Name:        "Backend Core",
		Description: "Backend core tag",
		Color:       "#ff5630",
		Active:      false,
		UpdatedAt:   time.Date(2026, time.March, 4, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected tag update to succeed, got %v", err)
	}
	if !updated {
		t.Fatalf("expected tag update to affect one row")
	}

	allTags, err := repo.List(ctx, CatalogTagListFilter{})
	if err != nil {
		t.Fatalf("expected tag list to succeed, got %v", err)
	}
	if len(allTags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(allTags))
	}
	if allTags[0].TagID != "tag-automation" || allTags[1].TagID != "tag-backend" {
		t.Fatalf("expected deterministic tag ordering by key, got %+v", allTags)
	}

	activeOnly, err := repo.List(ctx, CatalogTagListFilter{Active: boolPointer(true)})
	if err != nil {
		t.Fatalf("expected active tag list to succeed, got %v", err)
	}
	if len(activeOnly) != 1 || activeOnly[0].TagID != "tag-automation" {
		t.Fatalf("expected one active tag, got %+v", activeOnly)
	}

	lookup, err := repo.GetByKey(ctx, "backend-core")
	if err != nil {
		t.Fatalf("expected get tag by key to succeed, got %v", err)
	}
	if lookup.TagID != "tag-backend" {
		t.Fatalf("expected tag_id %q, got %q", "tag-backend", lookup.TagID)
	}

	deleted, err := repo.DeleteByTagID(ctx, "tag-automation")
	if err != nil {
		t.Fatalf("expected tag delete to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected tag delete to affect one row")
	}

	_, err = repo.GetByTagID(ctx, "tag-automation")
	if !errors.Is(err, ErrCatalogTagNotFound) {
		t.Fatalf("expected missing tag not found error, got %v", err)
	}
}

func TestCatalogTaxonomyRegistryRepositories_DeleteRestrictions_WithAssignments_ReturnForeignKeyErrors(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	sourceRepo := newCatalogSourceRepositoryForTest(t, db)
	domainRepo := newCatalogDomainRepositoryForTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForTest(t, db)
	tagRepo := newCatalogTagRepositoryForTest(t, db)
	taxonomyAssignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForTest(t, db)

	mustUpsertCatalogSourceRow(t, ctx, sourceRepo, CatalogSourceRow{
		ItemID:           "skill:taxonomy-delete-guard",
		Classifier:       CatalogClassifierSkill,
		SourceType:       CatalogSourceTypeLocal,
		Name:             "taxonomy-delete-guard",
		Description:      "taxonomy delete guard",
		Content:          "content",
		ContentHash:      "sha256:taxonomy-delete-guard",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 4, 14, 0, 0, 0, time.UTC),
	})

	mustCreateCatalogDomainRow(t, ctx, domainRepo, CatalogDomainRow{
		DomainID: "domain-platform",
		Key:      "platform",
		Name:     "Platform",
		Active:   true,
	})
	mustCreateCatalogSubdomainRow(t, ctx, subdomainRepo, CatalogSubdomainRow{
		SubdomainID: "subdomain-platform-api",
		DomainID:    "domain-platform",
		Key:         "api",
		Name:        "API",
		Active:      true,
	})
	mustCreateCatalogTagRow(t, ctx, tagRepo, CatalogTagRow{
		TagID:  "tag-backend",
		Key:    "backend",
		Name:   "Backend",
		Active: true,
	})

	if err := taxonomyAssignmentRepo.Upsert(ctx, CatalogItemTaxonomyAssignmentRow{
		ItemID:             "skill:taxonomy-delete-guard",
		PrimaryDomainID:    stringPointer("domain-platform"),
		PrimarySubdomainID: stringPointer("subdomain-platform-api"),
		UpdatedAt:          time.Date(2026, time.March, 4, 14, 30, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("expected taxonomy assignment upsert to succeed, got %v", err)
	}

	if err := tagAssignmentRepo.ReplaceForItemID(
		ctx,
		"skill:taxonomy-delete-guard",
		[]string{"tag-backend"},
		time.Date(2026, time.March, 4, 14, 31, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("expected tag assignment replace to succeed, got %v", err)
	}

	if _, err := domainRepo.DeleteByDomainID(ctx, "domain-platform"); err == nil {
		t.Fatalf("expected assigned domain delete to fail with foreign key error, got nil")
	} else if !strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY") {
		t.Fatalf("expected assigned domain delete failure to mention FOREIGN KEY, got %v", err)
	}

	if _, err := subdomainRepo.DeleteBySubdomainID(ctx, "subdomain-platform-api"); err == nil {
		t.Fatalf("expected assigned subdomain delete to fail with foreign key error, got nil")
	} else if !strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY") {
		t.Fatalf("expected assigned subdomain delete failure to mention FOREIGN KEY, got %v", err)
	}

	if _, err := tagRepo.DeleteByTagID(ctx, "tag-backend"); err == nil {
		t.Fatalf("expected assigned tag delete to fail with foreign key error, got nil")
	} else if !strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY") {
		t.Fatalf("expected assigned tag delete failure to mention FOREIGN KEY, got %v", err)
	}

	deletedAssignments, err := taxonomyAssignmentRepo.DeleteByItemID(ctx, "skill:taxonomy-delete-guard")
	if err != nil {
		t.Fatalf("expected taxonomy assignment delete to succeed, got %v", err)
	}
	if !deletedAssignments {
		t.Fatalf("expected taxonomy assignment delete to affect one row")
	}

	deletedTags, err := tagAssignmentRepo.DeleteByItemID(ctx, "skill:taxonomy-delete-guard")
	if err != nil {
		t.Fatalf("expected tag assignment delete to succeed, got %v", err)
	}
	if !deletedTags {
		t.Fatalf("expected tag assignment delete to affect one row")
	}

	if deleted, err := subdomainRepo.DeleteBySubdomainID(ctx, "subdomain-platform-api"); err != nil || !deleted {
		t.Fatalf("expected subdomain delete after unassignment to succeed, deleted=%t err=%v", deleted, err)
	}
	if deleted, err := domainRepo.DeleteByDomainID(ctx, "domain-platform"); err != nil || !deleted {
		t.Fatalf("expected domain delete after unassignment to succeed, deleted=%t err=%v", deleted, err)
	}
	if deleted, err := tagRepo.DeleteByTagID(ctx, "tag-backend"); err != nil || !deleted {
		t.Fatalf("expected tag delete after unassignment to succeed, deleted=%t err=%v", deleted, err)
	}
}
