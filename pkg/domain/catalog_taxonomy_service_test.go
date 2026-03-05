package domain

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

func TestNewCatalogTaxonomyRegistryService_WithNilRepositories_ReturnsError(t *testing.T) {
	db, _ := openCatalogSyncServiceTestDB(t)
	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	taxonomyAssignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	if _, err := NewCatalogTaxonomyRegistryService(
		nil,
		subdomainRepo,
		tagRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		CatalogTaxonomyRegistryServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil domain repository error, got nil")
	}
	if _, err := NewCatalogTaxonomyRegistryService(
		domainRepo,
		nil,
		tagRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		CatalogTaxonomyRegistryServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil subdomain repository error, got nil")
	}
	if _, err := NewCatalogTaxonomyRegistryService(
		domainRepo,
		subdomainRepo,
		nil,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		CatalogTaxonomyRegistryServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil tag repository error, got nil")
	}
	if _, err := NewCatalogTaxonomyRegistryService(
		domainRepo,
		subdomainRepo,
		tagRepo,
		nil,
		tagAssignmentRepo,
		CatalogTaxonomyRegistryServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil taxonomy assignment repository error, got nil")
	}
	if _, err := NewCatalogTaxonomyRegistryService(
		domainRepo,
		subdomainRepo,
		tagRepo,
		taxonomyAssignmentRepo,
		nil,
		CatalogTaxonomyRegistryServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil tag assignment repository error, got nil")
	}
}

func TestCatalogTaxonomyRegistryService_CreateDomain_NormalizesKeyAndMapsDuplicateConflict(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	now := time.Date(2026, time.March, 5, 11, 0, 0, 0, time.UTC)
	service := newCatalogTaxonomyRegistryServiceForDomainTest(t, db, now)

	created, err := service.CreateDomain(ctx, CatalogTaxonomyDomainCreateInput{
		DomainID:    "domain-platform",
		Key:         " Platform Core ",
		Name:        " Platform Core ",
		Description: " Core platform domain ",
	})
	if err != nil {
		t.Fatalf("expected create domain to succeed, got %v", err)
	}
	if created.Key != "platform-core" {
		t.Fatalf("expected normalized key %q, got %q", "platform-core", created.Key)
	}
	if created.Name != "Platform Core" {
		t.Fatalf("expected trimmed name %q, got %q", "Platform Core", created.Name)
	}
	if created.Description != "Core platform domain" {
		t.Fatalf("expected trimmed description %q, got %q", "Core platform domain", created.Description)
	}
	if !created.Active {
		t.Fatalf("expected created domain to default active=true")
	}

	listed, err := service.ListDomains(ctx, CatalogTaxonomyDomainListFilter{
		Key: " PLATFORM_core ",
	})
	if err != nil {
		t.Fatalf("expected list domains by normalized key to succeed, got %v", err)
	}
	if len(listed) != 1 || listed[0].DomainID != "domain-platform" {
		t.Fatalf("expected one domain with id %q, got %+v", "domain-platform", listed)
	}

	_, err = service.CreateDomain(ctx, CatalogTaxonomyDomainCreateInput{
		DomainID: "domain-platform-2",
		Key:      "platform core",
		Name:     "Platform Duplicate",
	})
	if !errors.Is(err, ErrCatalogTaxonomyConflict) {
		t.Fatalf("expected duplicate key create to map to conflict, got %v", err)
	}

	var conflictErr *CatalogTaxonomyConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected duplicate key error to include conflict details, got %v", err)
	}
	if conflictErr.ObjectType != CatalogTaxonomyObjectDomain {
		t.Fatalf("expected conflict object type %q, got %q", CatalogTaxonomyObjectDomain, conflictErr.ObjectType)
	}
	if conflictErr.Reason != CatalogTaxonomyConflictReasonDuplicateKey {
		t.Fatalf("expected duplicate key conflict reason, got %q", conflictErr.Reason)
	}
}

func TestCatalogTaxonomyRegistryService_UpdateTag_WithDuplicateKey_ReturnsConflict(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	now := time.Date(2026, time.March, 5, 12, 0, 0, 0, time.UTC)
	service := newCatalogTaxonomyRegistryServiceForDomainTest(t, db, now)

	if _, err := service.CreateTag(ctx, CatalogTaxonomyTagCreateInput{
		TagID: "tag-backend",
		Key:   "backend",
		Name:  "Backend",
	}); err != nil {
		t.Fatalf("expected backend tag create to succeed, got %v", err)
	}
	if _, err := service.CreateTag(ctx, CatalogTaxonomyTagCreateInput{
		TagID: "tag-automation",
		Key:   "automation",
		Name:  "Automation",
	}); err != nil {
		t.Fatalf("expected automation tag create to succeed, got %v", err)
	}

	_, err := service.UpdateTag(ctx, CatalogTaxonomyTagUpdateInput{
		TagID: "tag-automation",
		Key:   stringPointer(" BACKEND "),
	})
	if !errors.Is(err, ErrCatalogTaxonomyConflict) {
		t.Fatalf("expected duplicate key update to map to conflict, got %v", err)
	}

	var conflictErr *CatalogTaxonomyConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected duplicate key update to expose conflict details, got %v", err)
	}
	if conflictErr.ObjectType != CatalogTaxonomyObjectTag {
		t.Fatalf("expected conflict object type %q, got %q", CatalogTaxonomyObjectTag, conflictErr.ObjectType)
	}
	if conflictErr.Reason != CatalogTaxonomyConflictReasonDuplicateKey {
		t.Fatalf("expected duplicate key conflict reason, got %q", conflictErr.Reason)
	}
}

func TestCatalogTaxonomyRegistryService_SubdomainRelationshipValidation(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	now := time.Date(2026, time.March, 5, 13, 0, 0, 0, time.UTC)
	service := newCatalogTaxonomyRegistryServiceForDomainTest(t, db, now)

	_, err := service.CreateSubdomain(ctx, CatalogTaxonomySubdomainCreateInput{
		SubdomainID: "subdomain-missing-domain",
		DomainID:    "domain-missing",
		Key:         "api",
		Name:        "API",
	})
	if !errors.Is(err, ErrCatalogTaxonomyInvalidRelationship) {
		t.Fatalf("expected missing domain relationship validation error, got %v", err)
	}

	var relationshipErr *CatalogTaxonomyInvalidRelationshipError
	if !errors.As(err, &relationshipErr) {
		t.Fatalf("expected relationship validation error details, got %v", err)
	}
	if relationshipErr.Relationship != "domain_id" {
		t.Fatalf("expected relationship field %q, got %q", "domain_id", relationshipErr.Relationship)
	}
	if relationshipErr.ReferencedObject != "domain-missing" {
		t.Fatalf("expected missing domain reference %q, got %q", "domain-missing", relationshipErr.ReferencedObject)
	}

	if _, err := service.CreateDomain(ctx, CatalogTaxonomyDomainCreateInput{
		DomainID: "domain-platform",
		Key:      "platform",
		Name:     "Platform",
	}); err != nil {
		t.Fatalf("expected platform domain create to succeed, got %v", err)
	}
	if _, err := service.CreateSubdomain(ctx, CatalogTaxonomySubdomainCreateInput{
		SubdomainID: "subdomain-platform-api",
		DomainID:    "domain-platform",
		Key:         "api gateway",
		Name:        "API Gateway",
	}); err != nil {
		t.Fatalf("expected platform subdomain create to succeed, got %v", err)
	}

	_, err = service.UpdateSubdomain(ctx, CatalogTaxonomySubdomainUpdateInput{
		SubdomainID: "subdomain-platform-api",
		DomainID:    stringPointer("domain-unknown"),
	})
	if !errors.Is(err, ErrCatalogTaxonomyInvalidRelationship) {
		t.Fatalf("expected invalid domain reassignment error, got %v", err)
	}
}

func TestCatalogTaxonomyRegistryService_DeleteGuards_AssignedObjects_ReturnActionableConflict(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	now := time.Date(2026, time.March, 5, 14, 0, 0, 0, time.UTC)
	service := newCatalogTaxonomyRegistryServiceForDomainTest(t, db, now)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	taxonomyAssignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	if _, err := service.CreateDomain(ctx, CatalogTaxonomyDomainCreateInput{
		DomainID: "domain-platform",
		Key:      "platform",
		Name:     "Platform",
	}); err != nil {
		t.Fatalf("expected create domain to succeed, got %v", err)
	}
	if _, err := service.CreateSubdomain(ctx, CatalogTaxonomySubdomainCreateInput{
		SubdomainID: "subdomain-platform-api",
		DomainID:    "domain-platform",
		Key:         "api",
		Name:        "API",
	}); err != nil {
		t.Fatalf("expected create subdomain to succeed, got %v", err)
	}
	if _, err := service.CreateTag(ctx, CatalogTaxonomyTagCreateInput{
		TagID: "tag-backend",
		Key:   "backend",
		Name:  "Backend",
	}); err != nil {
		t.Fatalf("expected create tag to succeed, got %v", err)
	}

	itemID := "skill:taxonomy-assigned-item"
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "taxonomy-assigned-item",
		Description:      "taxonomy-assigned-item",
		Content:          "content",
		ContentHash:      "sha256:taxonomy-assigned-item",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     now,
	})

	if err := taxonomyAssignmentRepo.Upsert(ctx, persistence.CatalogItemTaxonomyAssignmentRow{
		ItemID:             itemID,
		PrimaryDomainID:    stringPointer("domain-platform"),
		PrimarySubdomainID: stringPointer("subdomain-platform-api"),
		UpdatedAt:          now,
		UpdatedBy:          stringPointer("tester"),
	}); err != nil {
		t.Fatalf("expected taxonomy assignment upsert to succeed, got %v", err)
	}
	if err := tagAssignmentRepo.ReplaceForItemID(ctx, itemID, []string{"tag-backend"}, now); err != nil {
		t.Fatalf("expected tag assignment replace to succeed, got %v", err)
	}

	assertCatalogTaxonomyInUseConflictForItem(
		t,
		service.DeleteDomain(ctx, "domain-platform"),
		CatalogTaxonomyObjectDomain,
		"domain-platform",
		itemID,
	)
	assertCatalogTaxonomyInUseConflictForItem(
		t,
		service.DeleteSubdomain(ctx, "subdomain-platform-api"),
		CatalogTaxonomyObjectSubdomain,
		"subdomain-platform-api",
		itemID,
	)
	assertCatalogTaxonomyInUseConflictForItem(
		t,
		service.DeleteTag(ctx, "tag-backend"),
		CatalogTaxonomyObjectTag,
		"tag-backend",
		itemID,
	)

	if deleted, err := taxonomyAssignmentRepo.DeleteByItemID(ctx, itemID); err != nil || !deleted {
		t.Fatalf("expected taxonomy assignment delete to succeed, deleted=%t err=%v", deleted, err)
	}
	if deleted, err := tagAssignmentRepo.DeleteByItemID(ctx, itemID); err != nil || !deleted {
		t.Fatalf("expected tag assignment delete to succeed, deleted=%t err=%v", deleted, err)
	}

	if err := service.DeleteSubdomain(ctx, "subdomain-platform-api"); err != nil {
		t.Fatalf("expected subdomain delete after unassignment to succeed, got %v", err)
	}
	if err := service.DeleteDomain(ctx, "domain-platform"); err != nil {
		t.Fatalf("expected domain delete after unassignment to succeed, got %v", err)
	}
	if err := service.DeleteTag(ctx, "tag-backend"); err != nil {
		t.Fatalf("expected tag delete after unassignment to succeed, got %v", err)
	}

	err := service.DeleteTag(ctx, "tag-backend")
	if !errors.Is(err, ErrCatalogTaxonomyTagNotFound) {
		t.Fatalf("expected second delete to map tag-not-found, got %v", err)
	}
}

func TestCatalogTaxonomyRegistryService_DeleteDomain_WithExistingSubdomains_ReturnsConflict(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	now := time.Date(2026, time.March, 5, 15, 0, 0, 0, time.UTC)
	service := newCatalogTaxonomyRegistryServiceForDomainTest(t, db, now)

	if _, err := service.CreateDomain(ctx, CatalogTaxonomyDomainCreateInput{
		DomainID: "domain-observability",
		Key:      "observability",
		Name:     "Observability",
	}); err != nil {
		t.Fatalf("expected domain create to succeed, got %v", err)
	}
	if _, err := service.CreateSubdomain(ctx, CatalogTaxonomySubdomainCreateInput{
		SubdomainID: "subdomain-observability-metrics",
		DomainID:    "domain-observability",
		Key:         "metrics",
		Name:        "Metrics",
	}); err != nil {
		t.Fatalf("expected subdomain create to succeed, got %v", err)
	}

	err := service.DeleteDomain(ctx, "domain-observability")
	if !errors.Is(err, ErrCatalogTaxonomyConflict) {
		t.Fatalf("expected delete to fail with conflict when child subdomains exist, got %v", err)
	}

	var conflictErr *CatalogTaxonomyConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected delete conflict error details, got %v", err)
	}
	if conflictErr.Reason != CatalogTaxonomyConflictReasonHasChildren {
		t.Fatalf("expected has-children conflict reason, got %q", conflictErr.Reason)
	}
	if !strings.Contains(conflictErr.Detail, "subdomain-observability-metrics") {
		t.Fatalf("expected conflict detail to mention blocking subdomain id, got %q", conflictErr.Detail)
	}
}

func TestCatalogTaxonomyRegistryService_UpdateDomainMissingAndInvalidTagKey(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	now := time.Date(2026, time.March, 5, 16, 0, 0, 0, time.UTC)
	service := newCatalogTaxonomyRegistryServiceForDomainTest(t, db, now)

	_, err := service.UpdateDomain(ctx, CatalogTaxonomyDomainUpdateInput{
		DomainID: "domain-missing",
		Name:     stringPointer("Missing"),
	})
	if !errors.Is(err, ErrCatalogTaxonomyDomainNotFound) {
		t.Fatalf("expected update missing domain to map not-found, got %v", err)
	}

	_, err = service.CreateTag(ctx, CatalogTaxonomyTagCreateInput{
		TagID: "tag-invalid",
		Key:   "bad/key",
		Name:  "Invalid",
	})
	if !errors.Is(err, ErrCatalogTaxonomyValidation) {
		t.Fatalf("expected invalid key validation error, got %v", err)
	}
}

func newCatalogTaxonomyRegistryServiceForDomainTest(
	t *testing.T,
	db *sql.DB,
	now time.Time,
) *CatalogTaxonomyRegistryService {
	t.Helper()

	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	taxonomyAssignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	service, err := NewCatalogTaxonomyRegistryService(
		domainRepo,
		subdomainRepo,
		tagRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		CatalogTaxonomyRegistryServiceOptions{
			Now: func() time.Time {
				return now
			},
		},
	)
	if err != nil {
		t.Fatalf("expected taxonomy registry service creation to succeed, got %v", err)
	}

	return service
}

func newCatalogDomainRepositoryForDomainTest(
	t *testing.T,
	db *sql.DB,
) *persistence.CatalogDomainRepository {
	t.Helper()

	repo, err := persistence.NewCatalogDomainRepository(db)
	if err != nil {
		t.Fatalf("expected domain repository creation to succeed, got %v", err)
	}
	return repo
}

func newCatalogSubdomainRepositoryForDomainTest(
	t *testing.T,
	db *sql.DB,
) *persistence.CatalogSubdomainRepository {
	t.Helper()

	repo, err := persistence.NewCatalogSubdomainRepository(db)
	if err != nil {
		t.Fatalf("expected subdomain repository creation to succeed, got %v", err)
	}
	return repo
}

func newCatalogTagRepositoryForDomainTest(
	t *testing.T,
	db *sql.DB,
) *persistence.CatalogTagRepository {
	t.Helper()

	repo, err := persistence.NewCatalogTagRepository(db)
	if err != nil {
		t.Fatalf("expected tag repository creation to succeed, got %v", err)
	}
	return repo
}

func newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(
	t *testing.T,
	db *sql.DB,
) *persistence.CatalogItemTaxonomyAssignmentRepository {
	t.Helper()

	repo, err := persistence.NewCatalogItemTaxonomyAssignmentRepository(db)
	if err != nil {
		t.Fatalf("expected taxonomy assignment repository creation to succeed, got %v", err)
	}
	return repo
}

func newCatalogItemTagAssignmentRepositoryForDomainTest(
	t *testing.T,
	db *sql.DB,
) *persistence.CatalogItemTagAssignmentRepository {
	t.Helper()

	repo, err := persistence.NewCatalogItemTagAssignmentRepository(db)
	if err != nil {
		t.Fatalf("expected tag assignment repository creation to succeed, got %v", err)
	}
	return repo
}

func assertCatalogTaxonomyInUseConflictForItem(
	t *testing.T,
	err error,
	objectType CatalogTaxonomyObjectType,
	objectID string,
	itemID string,
) {
	t.Helper()

	if !errors.Is(err, ErrCatalogTaxonomyConflict) {
		t.Fatalf("expected in-use delete to return conflict, got %v", err)
	}

	var conflictErr *CatalogTaxonomyConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected conflict details, got %v", err)
	}
	if conflictErr.ObjectType != objectType {
		t.Fatalf("expected conflict object type %q, got %q", objectType, conflictErr.ObjectType)
	}
	if conflictErr.ObjectID != objectID {
		t.Fatalf("expected conflict object id %q, got %q", objectID, conflictErr.ObjectID)
	}
	if conflictErr.Reason != CatalogTaxonomyConflictReasonInUse {
		t.Fatalf("expected in-use conflict reason, got %q", conflictErr.Reason)
	}
	if len(conflictErr.ReferencedItemIDs) != 1 || conflictErr.ReferencedItemIDs[0] != itemID {
		t.Fatalf("expected referenced item ids [%q], got %+v", itemID, conflictErr.ReferencedItemIDs)
	}
}
