package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

func TestNewCatalogTaxonomyAssignmentService_WithNilRepositories_ReturnsError(t *testing.T) {
	db, _ := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	assignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)
	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)

	if _, err := NewCatalogTaxonomyAssignmentService(
		nil,
		assignmentRepo,
		tagAssignmentRepo,
		domainRepo,
		subdomainRepo,
		tagRepo,
		CatalogTaxonomyAssignmentServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil source repository error, got nil")
	}
	if _, err := NewCatalogTaxonomyAssignmentService(
		sourceRepo,
		nil,
		tagAssignmentRepo,
		domainRepo,
		subdomainRepo,
		tagRepo,
		CatalogTaxonomyAssignmentServiceOptions{},
	); err == nil {
		t.Fatalf("expected nil assignment repository error, got nil")
	}
}

func TestCatalogTaxonomyAssignmentService_Get_UnassignedItem_ReturnsEmptyAssignment(t *testing.T) {
	service, ctx, itemID := newCatalogTaxonomyAssignmentServiceForDomainTest(t)

	view, err := service.Get(ctx, itemID)
	if err != nil {
		t.Fatalf("expected unassigned item lookup to succeed, got %v", err)
	}
	if view.ItemID != itemID {
		t.Fatalf("expected item_id %q, got %q", itemID, view.ItemID)
	}
	if view.PrimaryDomain != nil || view.PrimarySubdomain != nil || view.SecondaryDomain != nil || view.SecondarySubdomain != nil {
		t.Fatalf("expected no domain/subdomain assignments, got %+v", view)
	}
	if len(view.Tags) != 0 {
		t.Fatalf("expected no tag assignments, got %+v", view.Tags)
	}
}

func TestCatalogTaxonomyAssignmentService_Patch_RejectsMismatchedDomainSubdomain(t *testing.T) {
	service, ctx, itemID := newCatalogTaxonomyAssignmentServiceForDomainTest(t)

	_, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:             itemID,
		PrimaryDomainID:    stringPointer("domain-observability"),
		PrimarySubdomainID: stringPointer("subdomain-platform-api"),
	})
	if err == nil {
		t.Fatalf("expected mismatched domain/subdomain patch to fail, got nil")
	}
	if !errors.Is(err, ErrCatalogTaxonomyInvalidRelationship) {
		t.Fatalf("expected invalid relationship error, got %v", err)
	}
}

func TestCatalogTaxonomyAssignmentService_Patch_ValidAssignmentsAndTags_RoundTrip(t *testing.T) {
	service, ctx, itemID := newCatalogTaxonomyAssignmentServiceForDomainTest(t)
	updatedAt := time.Date(2026, time.March, 5, 3, 0, 0, 0, time.UTC)

	view, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:               itemID,
		PrimarySubdomainID:   stringPointer("subdomain-platform-api"),
		SecondaryDomainID:    stringPointer("domain-observability"),
		SecondarySubdomainID: stringPointer("subdomain-observability-metrics"),
		TagIDs:               &[]string{"tag-metrics", "tag-backend"},
		UpdatedBy:            stringPointer("tester"),
		UpdatedAt:            &updatedAt,
	})
	if err != nil {
		t.Fatalf("expected valid taxonomy patch to succeed, got %v", err)
	}

	if view.PrimaryDomain == nil || view.PrimaryDomain.ID != "domain-platform" {
		t.Fatalf("expected inferred primary domain domain-platform, got %+v", view.PrimaryDomain)
	}
	if view.PrimarySubdomain == nil || view.PrimarySubdomain.ID != "subdomain-platform-api" {
		t.Fatalf("expected primary subdomain subdomain-platform-api, got %+v", view.PrimarySubdomain)
	}
	if view.SecondaryDomain == nil || view.SecondaryDomain.ID != "domain-observability" {
		t.Fatalf("expected secondary domain domain-observability, got %+v", view.SecondaryDomain)
	}
	if view.SecondarySubdomain == nil || view.SecondarySubdomain.ID != "subdomain-observability-metrics" {
		t.Fatalf("expected secondary subdomain subdomain-observability-metrics, got %+v", view.SecondarySubdomain)
	}
	if len(view.Tags) != 2 || view.Tags[0].ID != "tag-backend" || view.Tags[1].ID != "tag-metrics" {
		t.Fatalf("expected deterministic tags [tag-backend tag-metrics], got %+v", view.Tags)
	}
	if view.UpdatedAt == nil || !view.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updated_at %s, got %+v", updatedAt, view.UpdatedAt)
	}
	if view.UpdatedBy == nil || *view.UpdatedBy != "tester" {
		t.Fatalf("expected updated_by tester, got %+v", view.UpdatedBy)
	}

	roundTrip, err := service.Get(ctx, itemID)
	if err != nil {
		t.Fatalf("expected get after patch to succeed, got %v", err)
	}
	if roundTrip.PrimaryDomain == nil || roundTrip.PrimaryDomain.ID != "domain-platform" {
		t.Fatalf("expected round-trip primary domain domain-platform, got %+v", roundTrip.PrimaryDomain)
	}
	if len(roundTrip.Tags) != 2 {
		t.Fatalf("expected round-trip tags length 2, got %+v", roundTrip.Tags)
	}
}

func TestCatalogTaxonomyAssignmentService_Patch_MissingItemAndTag_ReturnsNotFound(t *testing.T) {
	service, ctx, _ := newCatalogTaxonomyAssignmentServiceForDomainTest(t)

	_, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:          "skill:missing-item",
		PrimaryDomainID: stringPointer("domain-platform"),
	})
	if !errors.Is(err, ErrCatalogTaxonomyAssignmentItemNotFound) {
		t.Fatalf("expected missing item not found error, got %v", err)
	}

	_, err = service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID: "skill:taxonomy-item",
		TagIDs: &[]string{"tag-missing"},
	})
	if !errors.Is(err, ErrCatalogTaxonomyTagNotFound) {
		t.Fatalf("expected missing tag not found error, got %v", err)
	}
}

func newCatalogTaxonomyAssignmentServiceForDomainTest(
	t *testing.T,
) (*CatalogTaxonomyAssignmentService, context.Context, string) {
	t.Helper()

	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	assignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)
	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)

	itemID := "skill:taxonomy-item"
	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "taxonomy-item",
		Description:      "taxonomy fixture item",
		Content:          "taxonomy fixture content",
		ContentHash:      buildCatalogContentHash("taxonomy fixture content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 5, 2, 0, 0, 0, time.UTC),
	})

	if err := domainRepo.Create(ctx, persistence.CatalogDomainRow{
		DomainID: "domain-platform",
		Key:      "platform",
		Name:     "Platform",
		Active:   true,
	}); err != nil {
		t.Fatalf("expected create domain-platform to succeed, got %v", err)
	}
	if err := domainRepo.Create(ctx, persistence.CatalogDomainRow{
		DomainID: "domain-observability",
		Key:      "observability",
		Name:     "Observability",
		Active:   true,
	}); err != nil {
		t.Fatalf("expected create domain-observability to succeed, got %v", err)
	}

	if err := subdomainRepo.Create(ctx, persistence.CatalogSubdomainRow{
		SubdomainID: "subdomain-platform-api",
		DomainID:    "domain-platform",
		Key:         "api",
		Name:        "API",
		Active:      true,
	}); err != nil {
		t.Fatalf("expected create subdomain-platform-api to succeed, got %v", err)
	}
	if err := subdomainRepo.Create(ctx, persistence.CatalogSubdomainRow{
		SubdomainID: "subdomain-observability-metrics",
		DomainID:    "domain-observability",
		Key:         "metrics",
		Name:        "Metrics",
		Active:      true,
	}); err != nil {
		t.Fatalf("expected create subdomain-observability-metrics to succeed, got %v", err)
	}

	if err := tagRepo.Create(ctx, persistence.CatalogTagRow{
		TagID:  "tag-backend",
		Key:    "backend",
		Name:   "Backend",
		Active: true,
	}); err != nil {
		t.Fatalf("expected create tag-backend to succeed, got %v", err)
	}
	if err := tagRepo.Create(ctx, persistence.CatalogTagRow{
		TagID:  "tag-metrics",
		Key:    "metrics",
		Name:   "Metrics",
		Active: true,
	}); err != nil {
		t.Fatalf("expected create tag-metrics to succeed, got %v", err)
	}

	service, err := NewCatalogTaxonomyAssignmentService(
		sourceRepo,
		assignmentRepo,
		tagAssignmentRepo,
		domainRepo,
		subdomainRepo,
		tagRepo,
		CatalogTaxonomyAssignmentServiceOptions{
			Now: func() time.Time {
				return time.Date(2026, time.March, 5, 4, 0, 0, 0, time.UTC)
			},
		},
	)
	if err != nil {
		t.Fatalf("expected taxonomy assignment service creation to succeed, got %v", err)
	}

	return service, ctx, itemID
}
