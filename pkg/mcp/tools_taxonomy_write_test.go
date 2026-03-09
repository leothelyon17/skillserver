package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mudler/skillserver/pkg/domain"
)

func TestTaxonomyWriteTools_UpdateDomain_RequiresUpdateFields(t *testing.T) {
	registry := &stubCatalogTaxonomyRegistryWriter{}
	_, _, err := updateTaxonomyDomain(
		context.Background(),
		nil,
		UpdateTaxonomyDomainInput{DomainID: "domain-platform"},
		registry,
	)
	if err == nil {
		t.Fatalf("expected missing update fields error, got nil")
	}
	if !strings.Contains(err.Error(), "at least one of key, name, description, or active is required") {
		t.Fatalf("expected missing update fields validation error, got %v", err)
	}
	if registry.updateDomainCalled {
		t.Fatalf("expected registry update not to be called when validation fails")
	}
}

func TestTaxonomyWriteTools_PatchCatalogItemTaxonomy_RequiresItemID(t *testing.T) {
	assignment := &stubCatalogTaxonomyAssignmentWriter{}
	_, _, err := patchCatalogItemTaxonomy(
		context.Background(),
		nil,
		PatchCatalogItemTaxonomyInput{
			PrimaryDomainID: pointerTo("domain-platform"),
		},
		assignment,
	)
	if err == nil {
		t.Fatalf("expected missing item_id validation error, got nil")
	}
	if !strings.Contains(err.Error(), "item_id is required") {
		t.Fatalf("expected missing item_id validation error, got %v", err)
	}
	if assignment.patchCalled {
		t.Fatalf("expected assignment patch not to be called when validation fails")
	}
}

func TestTaxonomyWriteTools_CreateDomain_PropagatesServiceError(t *testing.T) {
	sentinel := errors.New("service boom")
	registry := &stubCatalogTaxonomyRegistryWriter{
		createDomainErr: sentinel,
	}

	_, _, err := createTaxonomyDomain(
		context.Background(),
		nil,
		CreateTaxonomyDomainInput{
			DomainID: "domain-platform",
			Key:      "platform",
			Name:     "Platform",
		},
		registry,
	)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel error, got %v", err)
	}
	if !strings.Contains(err.Error(), "create taxonomy domain") {
		t.Fatalf("expected create taxonomy domain context in error, got %v", err)
	}
}

func TestTaxonomyWriteTools_PatchCatalogItemTaxonomy_PassesInputsToService(t *testing.T) {
	assignment := &stubCatalogTaxonomyAssignmentWriter{
		patchResult: domain.CatalogItemTaxonomyAssignment{
			ItemID: "skill:sample-skill",
			Tags:   []domain.CatalogTaxonomyReference{{ID: "tag-backend", Key: "backend", Name: "Backend"}},
		},
	}
	tagIDs := []string{"tag-backend"}
	updatedBy := "tester"

	_, output, err := patchCatalogItemTaxonomy(
		context.Background(),
		nil,
		PatchCatalogItemTaxonomyInput{
			ItemID:    "skill:sample-skill",
			TagIDs:    &tagIDs,
			UpdatedBy: &updatedBy,
		},
		assignment,
	)
	if err != nil {
		t.Fatalf("expected patch to succeed, got %v", err)
	}
	if !assignment.patchCalled {
		t.Fatalf("expected assignment patch to be called")
	}
	if assignment.lastPatchInput.ItemID != "skill:sample-skill" {
		t.Fatalf("expected item_id to be forwarded, got %q", assignment.lastPatchInput.ItemID)
	}
	if assignment.lastPatchInput.AddTagIDs != nil {
		t.Fatalf("did not expect add_tag_ids to be set in this test")
	}
	if output.ItemID != "skill:sample-skill" {
		t.Fatalf("expected output item_id %q, got %q", "skill:sample-skill", output.ItemID)
	}
	if len(output.Tags) != 1 || output.Tags[0].ID != "tag-backend" {
		t.Fatalf("expected tag-backend output tag, got %+v", output.Tags)
	}
}

func TestTaxonomyWriteTools_PatchCatalogItemsTaxonomy_PassesInputsToService(t *testing.T) {
	assignment := &stubCatalogTaxonomyAssignmentWriter{
		patchBatchResult: domain.CatalogItemTaxonomyBatchPatchResult{
			DryRun: true,
			Items: []domain.CatalogItemTaxonomyBatchPatchItemResult{
				{
					RequestedItemID: "sample-skill",
					ItemID:          "skill:sample-skill",
					Status:          domain.CatalogItemTaxonomyBatchPatchStatusPlanned,
				},
			},
		},
	}
	addTagIDs := []string{"tag-backend"}

	_, output, err := patchCatalogItemsTaxonomy(
		context.Background(),
		nil,
		PatchCatalogItemsTaxonomyInput{
			DryRun: true,
			Items: []PatchCatalogItemsTaxonomyItemInput{
				{
					ItemID:    "sample-skill",
					AddTagIDs: &addTagIDs,
				},
			},
		},
		assignment,
	)
	if err != nil {
		t.Fatalf("expected batch patch to succeed, got %v", err)
	}
	if !assignment.patchBatchCalled {
		t.Fatalf("expected assignment PatchBatch to be called")
	}
	if !assignment.lastPatchBatchRequest.DryRun {
		t.Fatalf("expected dry_run=true to be forwarded")
	}
	if len(assignment.lastPatchBatchRequest.Items) != 1 {
		t.Fatalf("expected one batch item, got %d", len(assignment.lastPatchBatchRequest.Items))
	}
	if assignment.lastPatchBatchRequest.Items[0].ItemID != "sample-skill" {
		t.Fatalf("expected raw item_id to be forwarded, got %q", assignment.lastPatchBatchRequest.Items[0].ItemID)
	}
	if assignment.lastPatchBatchRequest.Items[0].AddTagIDs == nil ||
		len(*assignment.lastPatchBatchRequest.Items[0].AddTagIDs) != 1 ||
		(*assignment.lastPatchBatchRequest.Items[0].AddTagIDs)[0] != "tag-backend" {
		t.Fatalf("expected add_tag_ids to be forwarded, got %+v", assignment.lastPatchBatchRequest.Items[0].AddTagIDs)
	}
	if !output.DryRun || len(output.Items) != 1 {
		t.Fatalf("expected dry-run batch output, got %+v", output)
	}
}

func pointerTo[T any](value T) *T {
	return &value
}

type stubCatalogTaxonomyRegistryWriter struct {
	createDomainErr    error
	updateDomainCalled bool
}

func (s *stubCatalogTaxonomyRegistryWriter) CreateDomain(
	ctx context.Context,
	input domain.CatalogTaxonomyDomainCreateInput,
) (domain.CatalogTaxonomyDomain, error) {
	if s.createDomainErr != nil {
		return domain.CatalogTaxonomyDomain{}, s.createDomainErr
	}
	return domain.CatalogTaxonomyDomain{DomainID: input.DomainID, Key: input.Key, Name: input.Name}, nil
}

func (s *stubCatalogTaxonomyRegistryWriter) UpdateDomain(
	ctx context.Context,
	input domain.CatalogTaxonomyDomainUpdateInput,
) (domain.CatalogTaxonomyDomain, error) {
	s.updateDomainCalled = true
	return domain.CatalogTaxonomyDomain{DomainID: input.DomainID}, nil
}

func (s *stubCatalogTaxonomyRegistryWriter) DeleteDomain(ctx context.Context, domainID string) error {
	return nil
}

func (s *stubCatalogTaxonomyRegistryWriter) CreateSubdomain(
	ctx context.Context,
	input domain.CatalogTaxonomySubdomainCreateInput,
) (domain.CatalogTaxonomySubdomain, error) {
	return domain.CatalogTaxonomySubdomain{SubdomainID: input.SubdomainID}, nil
}

func (s *stubCatalogTaxonomyRegistryWriter) UpdateSubdomain(
	ctx context.Context,
	input domain.CatalogTaxonomySubdomainUpdateInput,
) (domain.CatalogTaxonomySubdomain, error) {
	return domain.CatalogTaxonomySubdomain{SubdomainID: input.SubdomainID}, nil
}

func (s *stubCatalogTaxonomyRegistryWriter) DeleteSubdomain(ctx context.Context, subdomainID string) error {
	return nil
}

func (s *stubCatalogTaxonomyRegistryWriter) CreateTag(
	ctx context.Context,
	input domain.CatalogTaxonomyTagCreateInput,
) (domain.CatalogTaxonomyTag, error) {
	return domain.CatalogTaxonomyTag{TagID: input.TagID}, nil
}

func (s *stubCatalogTaxonomyRegistryWriter) UpdateTag(
	ctx context.Context,
	input domain.CatalogTaxonomyTagUpdateInput,
) (domain.CatalogTaxonomyTag, error) {
	return domain.CatalogTaxonomyTag{TagID: input.TagID}, nil
}

func (s *stubCatalogTaxonomyRegistryWriter) DeleteTag(ctx context.Context, tagID string) error {
	return nil
}

type stubCatalogTaxonomyAssignmentWriter struct {
	patchResult           domain.CatalogItemTaxonomyAssignment
	patchErr              error
	patchCalled           bool
	lastPatchInput        domain.CatalogItemTaxonomyAssignmentPatchInput
	patchBatchResult      domain.CatalogItemTaxonomyBatchPatchResult
	patchBatchErr         error
	patchBatchCalled      bool
	lastPatchBatchRequest domain.CatalogItemTaxonomyBatchPatchRequest
}

func (s *stubCatalogTaxonomyAssignmentWriter) Patch(
	ctx context.Context,
	input domain.CatalogItemTaxonomyAssignmentPatchInput,
) (domain.CatalogItemTaxonomyAssignment, error) {
	s.patchCalled = true
	s.lastPatchInput = input
	if s.patchErr != nil {
		return domain.CatalogItemTaxonomyAssignment{}, s.patchErr
	}
	return s.patchResult, nil
}

func (s *stubCatalogTaxonomyAssignmentWriter) PatchBatch(
	ctx context.Context,
	request domain.CatalogItemTaxonomyBatchPatchRequest,
) (domain.CatalogItemTaxonomyBatchPatchResult, error) {
	s.patchBatchCalled = true
	s.lastPatchBatchRequest = request
	if s.patchBatchErr != nil {
		return domain.CatalogItemTaxonomyBatchPatchResult{}, s.patchBatchErr
	}
	return s.patchBatchResult, nil
}
