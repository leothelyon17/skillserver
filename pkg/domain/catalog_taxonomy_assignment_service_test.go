package domain

import (
	"context"
	"errors"
	"reflect"
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
	if view.HasAssignment {
		t.Fatalf("expected has_assignment=false for unassigned item")
	}
	if view.IsFullyClassified {
		t.Fatalf("expected is_fully_classified=false for unassigned item")
	}
	expectedMissingFields := []string{
		CatalogClassificationMissingPrimaryDomain,
		CatalogClassificationMissingPrimarySubdomain,
		CatalogClassificationMissingSecondaryDomain,
		CatalogClassificationMissingSecondarySubdomain,
		CatalogClassificationMissingTags,
	}
	if !reflect.DeepEqual(view.MissingFields, expectedMissingFields) {
		t.Fatalf("expected missing_fields %+v, got %+v", expectedMissingFields, view.MissingFields)
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
	if !view.HasAssignment {
		t.Fatalf("expected has_assignment=true after taxonomy patch")
	}
	if !view.IsFullyClassified {
		t.Fatalf("expected is_fully_classified=true when primary_domain and tags are present")
	}
	if len(view.MissingFields) != 0 {
		t.Fatalf("expected no missing fields for fully populated assignment, got %+v", view.MissingFields)
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
	if !roundTrip.HasAssignment || !roundTrip.IsFullyClassified {
		t.Fatalf("expected round-trip classification state to remain fully classified, got %+v", roundTrip)
	}
}

func TestCatalogTaxonomyAssignmentService_PatchBareSkillID_PrimaryDomainOnlyReturnsPartialClassificationState(t *testing.T) {
	service, ctx, _ := newCatalogTaxonomyAssignmentServiceForDomainTest(t)

	view, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:          "taxonomy-item",
		PrimaryDomainID: stringPointer("domain-platform"),
	})
	if err != nil {
		t.Fatalf("expected bare skill item patch to succeed, got %v", err)
	}

	if view.ItemID != BuildSkillCatalogItemID("taxonomy-item") {
		t.Fatalf("expected canonical item_id %q, got %q", BuildSkillCatalogItemID("taxonomy-item"), view.ItemID)
	}
	if !view.HasAssignment {
		t.Fatalf("expected has_assignment=true for partial taxonomy assignment")
	}
	if view.IsFullyClassified {
		t.Fatalf("expected is_fully_classified=false when tags are absent")
	}
	expectedMissingFields := []string{
		CatalogClassificationMissingPrimarySubdomain,
		CatalogClassificationMissingSecondaryDomain,
		CatalogClassificationMissingSecondarySubdomain,
		CatalogClassificationMissingTags,
	}
	if !reflect.DeepEqual(view.MissingFields, expectedMissingFields) {
		t.Fatalf("expected missing_fields %+v, got %+v", expectedMissingFields, view.MissingFields)
	}

	roundTrip, err := service.Get(ctx, "taxonomy-item")
	if err != nil {
		t.Fatalf("expected bare skill item get to succeed, got %v", err)
	}
	if roundTrip.ItemID != BuildSkillCatalogItemID("taxonomy-item") {
		t.Fatalf("expected canonical round-trip item_id %q, got %q", BuildSkillCatalogItemID("taxonomy-item"), roundTrip.ItemID)
	}
	if !reflect.DeepEqual(roundTrip.MissingFields, expectedMissingFields) {
		t.Fatalf("expected round-trip missing_fields %+v, got %+v", expectedMissingFields, roundTrip.MissingFields)
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

func TestCatalogTaxonomyAssignmentService_Patch_AddRemoveAndClearTags_RoundTrip(t *testing.T) {
	service, ctx, itemID := newCatalogTaxonomyAssignmentServiceForDomainTest(t)

	initial, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:    itemID,
		TagIDs:    &[]string{"tag-backend"},
		UpdatedBy: stringPointer("tester"),
	})
	if err != nil {
		t.Fatalf("expected initial tag replacement to succeed, got %v", err)
	}
	if len(initial.Tags) != 1 || initial.Tags[0].ID != "tag-backend" {
		t.Fatalf("expected initial tags [tag-backend], got %+v", initial.Tags)
	}

	added, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:    itemID,
		AddTagIDs: &[]string{"tag-metrics"},
		UpdatedBy: stringPointer("tester"),
	})
	if err != nil {
		t.Fatalf("expected additive tag patch to succeed, got %v", err)
	}
	if len(added.Tags) != 2 || added.Tags[0].ID != "tag-backend" || added.Tags[1].ID != "tag-metrics" {
		t.Fatalf("expected additive tags [tag-backend tag-metrics], got %+v", added.Tags)
	}

	removed, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:       itemID,
		RemoveTagIDs: &[]string{"tag-backend"},
	})
	if err != nil {
		t.Fatalf("expected remove tag patch to succeed, got %v", err)
	}
	if len(removed.Tags) != 1 || removed.Tags[0].ID != "tag-metrics" {
		t.Fatalf("expected remaining tags [tag-metrics], got %+v", removed.Tags)
	}

	clearTags := true
	cleared, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:    itemID,
		ClearTags: &clearTags,
	})
	if err != nil {
		t.Fatalf("expected clear tag patch to succeed, got %v", err)
	}
	if len(cleared.Tags) != 0 {
		t.Fatalf("expected cleared tags, got %+v", cleared.Tags)
	}
	if cleared.IsFullyClassified {
		t.Fatalf("expected cleared tag assignment to no longer be fully classified")
	}
}

func TestCatalogTaxonomyAssignmentService_Patch_RejectsAmbiguousTagMutationInputs(t *testing.T) {
	service, ctx, itemID := newCatalogTaxonomyAssignmentServiceForDomainTest(t)

	_, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:    itemID,
		TagIDs:    &[]string{"tag-backend"},
		AddTagIDs: &[]string{"tag-metrics"},
	})
	if !errors.Is(err, ErrCatalogTaxonomyValidation) {
		t.Fatalf("expected tag_ids + add_tag_ids to fail validation, got %v", err)
	}

	clearTags := true
	_, err = service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:       itemID,
		ClearTags:    &clearTags,
		RemoveTagIDs: &[]string{"tag-backend"},
	})
	if !errors.Is(err, ErrCatalogTaxonomyValidation) {
		t.Fatalf("expected clear_tags + remove_tag_ids to fail validation, got %v", err)
	}

	_, err = service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID:       itemID,
		AddTagIDs:    &[]string{"tag-backend"},
		RemoveTagIDs: &[]string{"tag-backend"},
	})
	if !errors.Is(err, ErrCatalogTaxonomyValidation) {
		t.Fatalf("expected overlapping add/remove tags to fail validation, got %v", err)
	}
}

func TestCatalogTaxonomyAssignmentService_PatchBatch_DryRunAndApply_ReturnDeterministicStatuses(t *testing.T) {
	service, ctx, itemID := newCatalogTaxonomyAssignmentServiceForDomainTest(t)
	secondItemID := seedCatalogTaxonomyAssignmentServiceTestSourceItem(t, ctx, service, "taxonomy-second-item")
	thirdItemID := seedCatalogTaxonomyAssignmentServiceTestSourceItem(t, ctx, service, "taxonomy-third-item")

	if _, err := service.Patch(ctx, CatalogItemTaxonomyAssignmentPatchInput{
		ItemID: secondItemID,
		TagIDs: &[]string{"tag-backend"},
	}); err != nil {
		t.Fatalf("expected baseline second-item patch to succeed, got %v", err)
	}

	dryRunResult, err := service.PatchBatch(ctx, CatalogItemTaxonomyBatchPatchRequest{
		DryRun: true,
		Items: []CatalogItemTaxonomyAssignmentPatchInput{
			{
				ItemID:    itemID,
				AddTagIDs: &[]string{"tag-metrics"},
			},
			{
				ItemID:    secondItemID,
				AddTagIDs: &[]string{"tag-backend"},
			},
			{
				ItemID:    "skill:missing-item",
				AddTagIDs: &[]string{"tag-backend"},
			},
			{
				ItemID:    thirdItemID,
				TagIDs:    &[]string{"tag-backend"},
				AddTagIDs: &[]string{"tag-metrics"},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected dry-run batch patch to succeed, got %v", err)
	}
	if !dryRunResult.DryRun {
		t.Fatalf("expected dry_run=true result")
	}
	if len(dryRunResult.Items) != 4 {
		t.Fatalf("expected 4 dry-run item results, got %d", len(dryRunResult.Items))
	}
	if dryRunResult.Items[0].Status != CatalogItemTaxonomyBatchPatchStatusPlanned {
		t.Fatalf("expected first dry-run item status planned, got %q", dryRunResult.Items[0].Status)
	}
	if dryRunResult.Items[1].Status != CatalogItemTaxonomyBatchPatchStatusUnchanged {
		t.Fatalf("expected second dry-run item status unchanged, got %q", dryRunResult.Items[1].Status)
	}
	if dryRunResult.Items[2].Status != CatalogItemTaxonomyBatchPatchStatusNotFound {
		t.Fatalf("expected third dry-run item status not_found, got %q", dryRunResult.Items[2].Status)
	}
	if dryRunResult.Items[3].Status != CatalogItemTaxonomyBatchPatchStatusInvalid {
		t.Fatalf("expected fourth dry-run item status invalid, got %q", dryRunResult.Items[3].Status)
	}

	afterDryRun, err := service.Get(ctx, itemID)
	if err != nil {
		t.Fatalf("expected get after dry-run to succeed, got %v", err)
	}
	if len(afterDryRun.Tags) != 0 {
		t.Fatalf("expected dry-run to avoid writes for first item, got %+v", afterDryRun.Tags)
	}

	applyResult, err := service.PatchBatch(ctx, CatalogItemTaxonomyBatchPatchRequest{
		Items: []CatalogItemTaxonomyAssignmentPatchInput{
			{
				ItemID:    itemID,
				AddTagIDs: &[]string{"tag-metrics"},
			},
			{
				ItemID:    secondItemID,
				AddTagIDs: &[]string{"tag-backend"},
			},
			{
				ItemID:    "skill:missing-item",
				AddTagIDs: &[]string{"tag-backend"},
			},
			{
				ItemID:    thirdItemID,
				TagIDs:    &[]string{"tag-backend"},
				AddTagIDs: &[]string{"tag-metrics"},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected apply batch patch to succeed, got %v", err)
	}
	if applyResult.DryRun {
		t.Fatalf("expected apply result dry_run=false")
	}
	if applyResult.Items[0].Status != CatalogItemTaxonomyBatchPatchStatusUpdated {
		t.Fatalf("expected first apply item status updated, got %q", applyResult.Items[0].Status)
	}
	if applyResult.Items[1].Status != CatalogItemTaxonomyBatchPatchStatusUnchanged {
		t.Fatalf("expected second apply item status unchanged, got %q", applyResult.Items[1].Status)
	}
	if applyResult.Items[2].Status != CatalogItemTaxonomyBatchPatchStatusNotFound {
		t.Fatalf("expected third apply item status not_found, got %q", applyResult.Items[2].Status)
	}
	if applyResult.Items[3].Status != CatalogItemTaxonomyBatchPatchStatusInvalid {
		t.Fatalf("expected fourth apply item status invalid, got %q", applyResult.Items[3].Status)
	}

	afterApply, err := service.Get(ctx, itemID)
	if err != nil {
		t.Fatalf("expected get after apply to succeed, got %v", err)
	}
	if len(afterApply.Tags) != 1 || afterApply.Tags[0].ID != "tag-metrics" {
		t.Fatalf("expected apply to persist tag-metrics for first item, got %+v", afterApply.Tags)
	}
}

func TestCatalogTaxonomyAssignmentService_PatchBatch_DuplicateCanonicalItemIDs_ReturnsGlobalValidationError(t *testing.T) {
	service, ctx, itemID := newCatalogTaxonomyAssignmentServiceForDomainTest(t)

	_, err := service.PatchBatch(ctx, CatalogItemTaxonomyBatchPatchRequest{
		Items: []CatalogItemTaxonomyAssignmentPatchInput{
			{
				ItemID:    itemID,
				AddTagIDs: &[]string{"tag-backend"},
			},
			{
				ItemID:    "taxonomy-item",
				AddTagIDs: &[]string{"tag-metrics"},
			},
		},
	})
	if !errors.Is(err, ErrCatalogTaxonomyValidation) {
		t.Fatalf("expected duplicate canonical item ids to fail validation, got %v", err)
	}

	view, getErr := service.Get(ctx, itemID)
	if getErr != nil {
		t.Fatalf("expected get after duplicate validation failure to succeed, got %v", getErr)
	}
	if len(view.Tags) != 0 {
		t.Fatalf("expected duplicate validation failure to prevent writes, got %+v", view.Tags)
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

func seedCatalogTaxonomyAssignmentServiceTestSourceItem(
	t *testing.T,
	ctx context.Context,
	service *CatalogTaxonomyAssignmentService,
	skillID string,
) string {
	t.Helper()

	itemID := BuildSkillCatalogItemID(skillID)
	sourceRepo, ok := service.sourceRepo.(*persistence.CatalogSourceRepository)
	if !ok {
		t.Fatalf("expected concrete catalog source repository, got %T", service.sourceRepo)
	}

	mustUpsertCatalogSourceRowForDomainTest(t, ctx, sourceRepo, persistence.CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             skillID,
		Description:      "taxonomy fixture item",
		Content:          "taxonomy fixture content",
		ContentHash:      buildCatalogContentHash(skillID + "-content"),
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 5, 2, 30, 0, 0, time.UTC),
	})

	return itemID
}
