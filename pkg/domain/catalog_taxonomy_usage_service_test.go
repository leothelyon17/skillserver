package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/persistence"
)

func TestNewCatalogTaxonomyUsageService_WithNilRepositories_ReturnsError(t *testing.T) {
	db, _ := openCatalogSyncServiceTestDB(t)
	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	assignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	if _, err := NewCatalogTaxonomyUsageService(
		nil,
		subdomainRepo,
		tagRepo,
		assignmentRepo,
		tagAssignmentRepo,
	); err == nil {
		t.Fatalf("expected nil domain repository error, got nil")
	}
	if _, err := NewCatalogTaxonomyUsageService(
		domainRepo,
		nil,
		tagRepo,
		assignmentRepo,
		tagAssignmentRepo,
	); err == nil {
		t.Fatalf("expected nil subdomain repository error, got nil")
	}
}

func TestCatalogTaxonomyUsageService_GetUsageSummaries_ReturnCountsPreviewAndBlockingReason(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	sourceRepo := newCatalogSourceRepositoryForDomainTest(t, db)
	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	assignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	service, err := NewCatalogTaxonomyUsageService(
		domainRepo,
		subdomainRepo,
		tagRepo,
		assignmentRepo,
		tagAssignmentRepo,
	)
	if err != nil {
		t.Fatalf("expected taxonomy usage service creation to succeed, got %v", err)
	}

	for _, itemID := range []string{"skill:item-a", "skill:item-b", "skill:item-c"} {
		mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
			ItemID:           itemID,
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             itemID,
			Description:      "taxonomy usage fixture",
			Content:          "content",
			ContentHash:      buildCatalogContentHash(itemID),
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     time.Date(2026, time.March, 9, 9, 0, 0, 0, time.UTC),
		})
	}

	for _, row := range []persistence.CatalogDomainRow{
		{DomainID: "domain-platform", Key: "platform", Name: "Platform", Active: true},
		{DomainID: "domain-observability", Key: "observability", Name: "Observability", Active: true},
	} {
		if err := domainRepo.Create(ctx, row); err != nil {
			t.Fatalf("expected create domain %q to succeed, got %v", row.DomainID, err)
		}
	}
	for _, row := range []persistence.CatalogSubdomainRow{
		{
			SubdomainID: "subdomain-platform-api",
			DomainID:    "domain-platform",
			Key:         "api",
			Name:        "API",
			Active:      true,
		},
		{
			SubdomainID: "subdomain-observability-metrics",
			DomainID:    "domain-observability",
			Key:         "metrics",
			Name:        "Metrics",
			Active:      true,
		},
	} {
		if err := subdomainRepo.Create(ctx, row); err != nil {
			t.Fatalf("expected create subdomain %q to succeed, got %v", row.SubdomainID, err)
		}
	}
	for _, row := range []persistence.CatalogTagRow{
		{TagID: "tag-backend", Key: "backend", Name: "Backend", Active: true},
		{TagID: "tag-metrics", Key: "metrics", Name: "Metrics", Active: true},
		{TagID: "tag-unused", Key: "unused", Name: "Unused", Active: true},
	} {
		if err := tagRepo.Create(ctx, row); err != nil {
			t.Fatalf("expected create tag %q to succeed, got %v", row.TagID, err)
		}
	}

	updatedAt := time.Date(2026, time.March, 9, 10, 0, 0, 0, time.UTC)
	for _, row := range []persistence.CatalogItemTaxonomyAssignmentRow{
		{
			ItemID:             "skill:item-a",
			PrimaryDomainID:    stringPointer("domain-platform"),
			PrimarySubdomainID: stringPointer("subdomain-platform-api"),
			UpdatedAt:          updatedAt,
			UpdatedBy:          stringPointer("tester-a"),
		},
		{
			ItemID:               "skill:item-b",
			SecondaryDomainID:    stringPointer("domain-platform"),
			SecondarySubdomainID: stringPointer("subdomain-platform-api"),
			UpdatedAt:            updatedAt.Add(1 * time.Minute),
			UpdatedBy:            stringPointer("tester-b"),
		},
		{
			ItemID:               "skill:item-c",
			PrimaryDomainID:      stringPointer("domain-platform"),
			PrimarySubdomainID:   stringPointer("subdomain-platform-api"),
			SecondaryDomainID:    stringPointer("domain-platform"),
			SecondarySubdomainID: stringPointer("subdomain-platform-api"),
			UpdatedAt:            updatedAt.Add(2 * time.Minute),
			UpdatedBy:            stringPointer("tester-c"),
		},
	} {
		if err := assignmentRepo.Upsert(ctx, row); err != nil {
			t.Fatalf("expected assignment upsert for %q to succeed, got %v", row.ItemID, err)
		}
	}

	if err := tagAssignmentRepo.ReplaceForItemID(ctx, "skill:item-a", []string{"tag-backend"}, updatedAt); err != nil {
		t.Fatalf("expected tag replace for item-a to succeed, got %v", err)
	}
	if err := tagAssignmentRepo.ReplaceForItemID(ctx, "skill:item-b", []string{"tag-backend", "tag-metrics"}, updatedAt); err != nil {
		t.Fatalf("expected tag replace for item-b to succeed, got %v", err)
	}

	domainUsage, err := service.GetDomainUsage(ctx, "domain-platform", 2)
	if err != nil {
		t.Fatalf("expected domain usage to succeed, got %v", err)
	}
	if domainUsage.ObjectType != CatalogTaxonomyObjectDomain || domainUsage.ObjectID != "domain-platform" {
		t.Fatalf("expected domain usage object domain/domain-platform, got %+v", domainUsage)
	}
	if domainUsage.AssignmentCount != 4 {
		t.Fatalf("expected domain assignment count 4, got %d", domainUsage.AssignmentCount)
	}
	if domainUsage.DistinctItemCount != 3 {
		t.Fatalf("expected domain distinct item count 3, got %d", domainUsage.DistinctItemCount)
	}
	expectedPreview := []string{"skill:item-a", "skill:item-b"}
	if !reflect.DeepEqual(domainUsage.PreviewItemIDs, expectedPreview) {
		t.Fatalf("expected domain usage preview %+v, got %+v", expectedPreview, domainUsage.PreviewItemIDs)
	}
	if domainUsage.BlockingReason == nil || *domainUsage.BlockingReason != CatalogTaxonomyConflictReasonInUse {
		t.Fatalf("expected domain blocking_reason=in_use, got %+v", domainUsage.BlockingReason)
	}

	subdomainUsage, err := service.GetSubdomainUsage(ctx, "subdomain-platform-api", 0)
	if err != nil {
		t.Fatalf("expected subdomain usage to succeed, got %v", err)
	}
	if subdomainUsage.AssignmentCount != 4 {
		t.Fatalf("expected subdomain assignment count 4, got %d", subdomainUsage.AssignmentCount)
	}
	if subdomainUsage.DistinctItemCount != 3 {
		t.Fatalf("expected subdomain distinct item count 3, got %d", subdomainUsage.DistinctItemCount)
	}
	if len(subdomainUsage.PreviewItemIDs) != 0 {
		t.Fatalf("expected no subdomain preview item ids when preview limit is zero, got %+v", subdomainUsage.PreviewItemIDs)
	}
	if subdomainUsage.BlockingReason == nil || *subdomainUsage.BlockingReason != CatalogTaxonomyConflictReasonInUse {
		t.Fatalf("expected subdomain blocking_reason=in_use, got %+v", subdomainUsage.BlockingReason)
	}

	unusedTagUsage, err := service.GetTagUsage(ctx, "tag-unused", 2)
	if err != nil {
		t.Fatalf("expected unused tag usage to succeed, got %v", err)
	}
	if unusedTagUsage.ObjectType != CatalogTaxonomyObjectTag || unusedTagUsage.ObjectID != "tag-unused" {
		t.Fatalf("expected tag usage object tag/tag-unused, got %+v", unusedTagUsage)
	}
	if unusedTagUsage.AssignmentCount != 0 || unusedTagUsage.DistinctItemCount != 0 {
		t.Fatalf("expected unused tag counts to be zero, got %+v", unusedTagUsage)
	}
	if len(unusedTagUsage.PreviewItemIDs) != 0 {
		t.Fatalf("expected unused tag preview item ids to be empty, got %+v", unusedTagUsage.PreviewItemIDs)
	}
	if unusedTagUsage.BlockingReason != nil {
		t.Fatalf("expected unused tag blocking_reason to be omitted, got %+v", unusedTagUsage.BlockingReason)
	}
}

func TestCatalogTaxonomyUsageService_GetUsage_MissingObjectReturnsNotFound(t *testing.T) {
	db, ctx := openCatalogSyncServiceTestDB(t)
	domainRepo := newCatalogDomainRepositoryForDomainTest(t, db)
	subdomainRepo := newCatalogSubdomainRepositoryForDomainTest(t, db)
	tagRepo := newCatalogTagRepositoryForDomainTest(t, db)
	assignmentRepo := newCatalogItemTaxonomyAssignmentRepositoryForDomainTest(t, db)
	tagAssignmentRepo := newCatalogItemTagAssignmentRepositoryForDomainTest(t, db)

	service, err := NewCatalogTaxonomyUsageService(
		domainRepo,
		subdomainRepo,
		tagRepo,
		assignmentRepo,
		tagAssignmentRepo,
	)
	if err != nil {
		t.Fatalf("expected taxonomy usage service creation to succeed, got %v", err)
	}

	_, err = service.GetTagUsage(ctx, "tag-missing", 1)
	if !errors.Is(err, ErrCatalogTaxonomyTagNotFound) {
		t.Fatalf("expected missing tag usage to map not-found, got %v", err)
	}
}
