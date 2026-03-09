package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/persistence"
)

func TestCatalogTaxonomyRegistryEndpoints_ServiceUnavailable_Returns503(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{
			name:   "list domains",
			method: http.MethodGet,
			target: "/api/catalog/taxonomy/domains",
		},
		{
			name:   "create subdomain",
			method: http.MethodPost,
			target: "/api/catalog/taxonomy/subdomains",
			body:   `{"subdomain_id":"subdomain-api","domain_id":"domain-platform","key":"api","name":"API"}`,
		},
		{
			name:   "delete tag",
			method: http.MethodDelete,
			target: "/api/catalog/taxonomy/tags/tag-backend",
		},
		{
			name:   "tag usage",
			method: http.MethodGet,
			target: "/api/catalog/taxonomy/tags/tag-backend/usage",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			rec := httptest.NewRecorder()
			server.echo.ServeHTTP(rec, req)

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected status %d, got %d body=%q", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCatalogTaxonomyRegistryEndpoints_DomainCRUD_MapsSuccessAndNotFound(t *testing.T) {
	t.Parallel()

	fixture := newCatalogTaxonomyFixtureServer(t)

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/taxonomy/domains",
		strings.NewReader(`{"domain_id":"domain-platform","key":" Platform Core ","name":" Platform Core ","description":" Core platform domain ","active":true}`),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusCreated, createRec.Code, createRec.Body.String())
	}

	created := decodeJSONObject(t, createRec.Body.Bytes())
	if key, _ := created["key"].(string); key != "platform-core" {
		t.Fatalf("expected normalized key %q, got %q", "platform-core", key)
	}
	if name, _ := created["name"].(string); name != "Platform Core" {
		t.Fatalf("expected trimmed name %q, got %q", "Platform Core", name)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/catalog/taxonomy/domains?key=platform-core", nil)
	listRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, listRec.Code, listRec.Body.String())
	}

	listPayload := decodeJSONArray(t, listRec.Body.Bytes())
	if len(listPayload) != 1 {
		t.Fatalf("expected one domain in filtered list, got %d payload=%q", len(listPayload), listRec.Body.String())
	}
	if domainID, _ := listPayload[0]["domain_id"].(string); domainID != "domain-platform" {
		t.Fatalf("expected domain_id %q, got %q", "domain-platform", domainID)
	}

	emptyPatchReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/taxonomy/domains/domain-platform",
		strings.NewReader(`{}`),
	)
	emptyPatchReq.Header.Set("Content-Type", "application/json")
	emptyPatchRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(emptyPatchRec, emptyPatchReq)

	if emptyPatchRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, emptyPatchRec.Code, emptyPatchRec.Body.String())
	}

	updateReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/taxonomy/domains/domain-platform",
		strings.NewReader(`{"name":"Platform Updated","active":false}`),
	)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, updateRec.Code, updateRec.Body.String())
	}

	updated := decodeJSONObject(t, updateRec.Body.Bytes())
	if name, _ := updated["name"].(string); name != "Platform Updated" {
		t.Fatalf("expected updated name %q, got %q", "Platform Updated", name)
	}
	if active, ok := updated["active"].(bool); !ok || active {
		t.Fatalf("expected active=false after update, got %v", updated["active"])
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/catalog/taxonomy/domains/domain-platform", nil)
	deleteRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNoContent, deleteRec.Code, deleteRec.Body.String())
	}

	deleteMissingReq := httptest.NewRequest(http.MethodDelete, "/api/catalog/taxonomy/domains/domain-platform", nil)
	deleteMissingRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(deleteMissingRec, deleteMissingReq)
	if deleteMissingRec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNotFound, deleteMissingRec.Code, deleteMissingRec.Body.String())
	}
}

func TestCatalogTaxonomyRegistryEndpoints_SubdomainCRUD_MapsValidationAndNotFound(t *testing.T) {
	t.Parallel()

	fixture := newCatalogTaxonomyFixtureServer(t)

	createDomainReq := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/taxonomy/domains",
		strings.NewReader(`{"domain_id":"domain-platform","key":"platform","name":"Platform"}`),
	)
	createDomainReq.Header.Set("Content-Type", "application/json")
	createDomainRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(createDomainRec, createDomainReq)
	if createDomainRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusCreated, createDomainRec.Code, createDomainRec.Body.String())
	}

	invalidRelationshipReq := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/taxonomy/subdomains",
		strings.NewReader(`{"subdomain_id":"subdomain-missing","domain_id":"domain-missing","key":"api","name":"API"}`),
	)
	invalidRelationshipReq.Header.Set("Content-Type", "application/json")
	invalidRelationshipRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(invalidRelationshipRec, invalidRelationshipReq)
	if invalidRelationshipRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, invalidRelationshipRec.Code, invalidRelationshipRec.Body.String())
	}

	createSubdomainReq := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/taxonomy/subdomains",
		strings.NewReader(`{"subdomain_id":"subdomain-platform-api","domain_id":"domain-platform","key":"api","name":"API"}`),
	)
	createSubdomainReq.Header.Set("Content-Type", "application/json")
	createSubdomainRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(createSubdomainRec, createSubdomainReq)
	if createSubdomainRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusCreated, createSubdomainRec.Code, createSubdomainRec.Body.String())
	}

	listReq := httptest.NewRequest(
		http.MethodGet,
		"/api/catalog/taxonomy/subdomains?domain_id=domain-platform",
		nil,
	)
	listRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	listPayload := decodeJSONArray(t, listRec.Body.Bytes())
	if len(listPayload) != 1 {
		t.Fatalf("expected one subdomain in filtered list, got %d payload=%q", len(listPayload), listRec.Body.String())
	}

	invalidDomainPatchReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/taxonomy/subdomains/subdomain-platform-api",
		strings.NewReader(`{"domain_id":"domain-unknown"}`),
	)
	invalidDomainPatchReq.Header.Set("Content-Type", "application/json")
	invalidDomainPatchRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(invalidDomainPatchRec, invalidDomainPatchReq)
	if invalidDomainPatchRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, invalidDomainPatchRec.Code, invalidDomainPatchRec.Body.String())
	}

	updateReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/taxonomy/subdomains/subdomain-platform-api",
		strings.NewReader(`{"name":"API Gateway","active":false}`),
	)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, updateRec.Code, updateRec.Body.String())
	}

	updated := decodeJSONObject(t, updateRec.Body.Bytes())
	if name, _ := updated["name"].(string); name != "API Gateway" {
		t.Fatalf("expected updated name %q, got %q", "API Gateway", name)
	}
	if active, ok := updated["active"].(bool); !ok || active {
		t.Fatalf("expected active=false after update, got %v", updated["active"])
	}

	deleteReq := httptest.NewRequest(
		http.MethodDelete,
		"/api/catalog/taxonomy/subdomains/subdomain-platform-api",
		nil,
	)
	deleteRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNoContent, deleteRec.Code, deleteRec.Body.String())
	}

	deleteMissingReq := httptest.NewRequest(
		http.MethodDelete,
		"/api/catalog/taxonomy/subdomains/subdomain-platform-api",
		nil,
	)
	deleteMissingRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(deleteMissingRec, deleteMissingReq)
	if deleteMissingRec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNotFound, deleteMissingRec.Code, deleteMissingRec.Body.String())
	}
}

func TestCatalogTaxonomyRegistryEndpoints_TagCRUD_MapsConflictAndInUseDelete(t *testing.T) {
	t.Parallel()

	fixture := newCatalogTaxonomyFixtureServer(t)

	createBackendReq := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/taxonomy/tags",
		strings.NewReader(`{"tag_id":"tag-backend","key":"backend","name":"Backend","color":"#33aaff"}`),
	)
	createBackendReq.Header.Set("Content-Type", "application/json")
	createBackendRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(createBackendRec, createBackendReq)
	if createBackendRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusCreated, createBackendRec.Code, createBackendRec.Body.String())
	}

	createAutomationReq := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/taxonomy/tags",
		strings.NewReader(`{"tag_id":"tag-automation","key":"automation","name":"Automation"}`),
	)
	createAutomationReq.Header.Set("Content-Type", "application/json")
	createAutomationRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(createAutomationRec, createAutomationReq)
	if createAutomationRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusCreated, createAutomationRec.Code, createAutomationRec.Body.String())
	}

	duplicatePatchReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/taxonomy/tags/tag-automation",
		strings.NewReader(`{"key":"backend"}`),
	)
	duplicatePatchReq.Header.Set("Content-Type", "application/json")
	duplicatePatchRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(duplicatePatchRec, duplicatePatchReq)
	if duplicatePatchRec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusConflict, duplicatePatchRec.Code, duplicatePatchRec.Body.String())
	}

	updateReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/taxonomy/tags/tag-automation",
		strings.NewReader(`{"name":"Automation Updated","color":"#111111","active":false}`),
	)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, updateRec.Code, updateRec.Body.String())
	}

	updated := decodeJSONObject(t, updateRec.Body.Bytes())
	if name, _ := updated["name"].(string); name != "Automation Updated" {
		t.Fatalf("expected updated name %q, got %q", "Automation Updated", name)
	}
	if active, ok := updated["active"].(bool); !ok || active {
		t.Fatalf("expected active=false after update, got %v", updated["active"])
	}

	seedCatalogTaxonomyTagAssignment(t, fixture, "skill:taxonomy-assigned-item", []string{"tag-backend"})

	deleteInUseReq := httptest.NewRequest(http.MethodDelete, "/api/catalog/taxonomy/tags/tag-backend", nil)
	deleteInUseRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(deleteInUseRec, deleteInUseReq)
	if deleteInUseRec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusConflict, deleteInUseRec.Code, deleteInUseRec.Body.String())
	}
	if !strings.Contains(strings.ToLower(deleteInUseRec.Body.String()), "assigned to catalog items") {
		t.Fatalf("expected actionable in-use conflict message, got %q", deleteInUseRec.Body.String())
	}
	conflictPayload := decodeJSONObject(t, deleteInUseRec.Body.Bytes())
	conflict, ok := conflictPayload["conflict"].(map[string]any)
	if !ok {
		t.Fatalf("expected structured conflict payload, got %+v", conflictPayload)
	}
	if reason, _ := conflict["reason"].(string); reason != "in_use" {
		t.Fatalf("expected conflict reason in_use, got %+v", conflict)
	}
	referencedItemIDs, ok := conflict["referenced_item_ids"].([]any)
	if !ok || len(referencedItemIDs) != 1 {
		t.Fatalf("expected referenced_item_ids in conflict payload, got %+v", conflict["referenced_item_ids"])
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/catalog/taxonomy/tags/tag-automation", nil)
	deleteRec := httptest.NewRecorder()
	fixture.server.echo.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNoContent, deleteRec.Code, deleteRec.Body.String())
	}
}

func TestCatalogTaxonomyUsageEndpoints_ReturnStructuredPreflight(t *testing.T) {
	t.Parallel()

	fixture := newCatalogTaxonomyFixtureServer(t)

	requests := []struct {
		target string
		body   string
	}{
		{
			target: "/api/catalog/taxonomy/domains",
			body:   `{"domain_id":"domain-platform","key":"platform","name":"Platform"}`,
		},
		{
			target: "/api/catalog/taxonomy/subdomains",
			body:   `{"subdomain_id":"subdomain-platform-api","domain_id":"domain-platform","key":"api","name":"API"}`,
		},
		{
			target: "/api/catalog/taxonomy/tags",
			body:   `{"tag_id":"tag-backend","key":"backend","name":"Backend"}`,
		},
	}
	for _, request := range requests {
		req := httptest.NewRequest(http.MethodPost, request.target, strings.NewReader(request.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		fixture.server.echo.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d for %q, got %d body=%q", http.StatusCreated, request.target, rec.Code, rec.Body.String())
		}
	}

	itemID := "skill:taxonomy-assigned-item"
	seedCatalogTaxonomyTagAssignment(t, fixture, itemID, []string{"tag-backend"})

	primaryDomainID := "domain-platform"
	primarySubdomainID := "subdomain-platform-api"
	if err := fixture.assignmentRepo.Upsert(context.Background(), persistence.CatalogItemTaxonomyAssignmentRow{
		ItemID:             itemID,
		PrimaryDomainID:    &primaryDomainID,
		PrimarySubdomainID: &primarySubdomainID,
		UpdatedAt:          time.Date(2026, time.March, 5, 8, 5, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("expected taxonomy assignment seed to succeed, got %v", err)
	}

	for _, target := range []string{
		"/api/catalog/taxonomy/domains/domain-platform/usage?preview_limit=1",
		"/api/catalog/taxonomy/subdomains/subdomain-platform-api/usage?preview_limit=1",
		"/api/catalog/taxonomy/tags/tag-backend/usage?preview_limit=1",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		fixture.server.echo.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d for %q, got %d body=%q", http.StatusOK, target, rec.Code, rec.Body.String())
		}

		payload := decodeJSONObject(t, rec.Body.Bytes())
		if assignmentCount, ok := payload["assignment_count"].(float64); !ok || assignmentCount != 1 {
			t.Fatalf("expected assignment_count=1 for %q, got %+v", target, payload)
		}
		if distinctItemCount, ok := payload["distinct_item_count"].(float64); !ok || distinctItemCount != 1 {
			t.Fatalf("expected distinct_item_count=1 for %q, got %+v", target, payload)
		}
		if blockingReason, _ := payload["blocking_reason"].(string); blockingReason != "in_use" {
			t.Fatalf("expected blocking_reason=in_use for %q, got %+v", target, payload)
		}
		previewItemIDs, ok := payload["preview_item_ids"].([]any)
		if !ok || len(previewItemIDs) != 1 {
			t.Fatalf("expected preview_item_ids for %q, got %+v", target, payload["preview_item_ids"])
		}
	}
}

type catalogTaxonomyFixture struct {
	server            *Server
	sourceRepo        *persistence.CatalogSourceRepository
	assignmentRepo    *persistence.CatalogItemTaxonomyAssignmentRepository
	tagAssignmentRepo *persistence.CatalogItemTagAssignmentRepository
}

func newCatalogTaxonomyFixtureServer(t *testing.T) catalogTaxonomyFixture {
	t.Helper()

	server := newResourceFixtureServer(t)

	dbPath := t.TempDir() + "/catalog-taxonomy-api.db"
	db, err := persistence.BootstrapSQLite(context.Background(), dbPath, persistence.SQLiteBootstrapConfig{})
	if err != nil {
		t.Fatalf("expected SQLite bootstrap to succeed, got %v", err)
	}
	t.Cleanup(func() {
		if closeErr := persistence.CloseSQLite(db); closeErr != nil {
			t.Fatalf("expected SQLite close to succeed, got %v", closeErr)
		}
	})

	sourceRepo, err := persistence.NewCatalogSourceRepository(db)
	if err != nil {
		t.Fatalf("expected source repository creation to succeed, got %v", err)
	}
	taxonomyAssignmentRepo, err := persistence.NewCatalogItemTaxonomyAssignmentRepository(db)
	if err != nil {
		t.Fatalf("expected taxonomy assignment repository creation to succeed, got %v", err)
	}
	tagAssignmentRepo, err := persistence.NewCatalogItemTagAssignmentRepository(db)
	if err != nil {
		t.Fatalf("expected tag assignment repository creation to succeed, got %v", err)
	}
	domainRepo, err := persistence.NewCatalogDomainRepository(db)
	if err != nil {
		t.Fatalf("expected domain repository creation to succeed, got %v", err)
	}
	subdomainRepo, err := persistence.NewCatalogSubdomainRepository(db)
	if err != nil {
		t.Fatalf("expected subdomain repository creation to succeed, got %v", err)
	}
	tagRepo, err := persistence.NewCatalogTagRepository(db)
	if err != nil {
		t.Fatalf("expected tag repository creation to succeed, got %v", err)
	}

	taxonomyRegistry, err := domain.NewCatalogTaxonomyRegistryService(
		domainRepo,
		subdomainRepo,
		tagRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
		domain.CatalogTaxonomyRegistryServiceOptions{},
	)
	if err != nil {
		t.Fatalf("expected taxonomy registry service creation to succeed, got %v", err)
	}
	taxonomyUsageService, err := domain.NewCatalogTaxonomyUsageService(
		domainRepo,
		subdomainRepo,
		tagRepo,
		taxonomyAssignmentRepo,
		tagAssignmentRepo,
	)
	if err != nil {
		t.Fatalf("expected taxonomy usage service creation to succeed, got %v", err)
	}

	server.SetCatalogTaxonomyRegistryService(taxonomyRegistry)
	server.SetCatalogTaxonomyUsageService(taxonomyUsageService)

	return catalogTaxonomyFixture{
		server:            server,
		sourceRepo:        sourceRepo,
		assignmentRepo:    taxonomyAssignmentRepo,
		tagAssignmentRepo: tagAssignmentRepo,
	}
}

func seedCatalogTaxonomyTagAssignment(
	t *testing.T,
	fixture catalogTaxonomyFixture,
	itemID string,
	tagIDs []string,
) {
	t.Helper()

	syncedAt := time.Date(2026, time.March, 5, 8, 0, 0, 0, time.UTC)
	if err := fixture.sourceRepo.Upsert(context.Background(), persistence.CatalogSourceRow{
		ItemID:           itemID,
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "taxonomy-fixture-item",
		Description:      "taxonomy fixture item",
		Content:          "fixture content",
		ContentHash:      "sha256:taxonomy-fixture-item",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     syncedAt,
	}); err != nil {
		t.Fatalf("expected taxonomy source upsert to succeed, got %v", err)
	}

	if err := fixture.tagAssignmentRepo.ReplaceForItemID(context.Background(), itemID, tagIDs, syncedAt); err != nil {
		t.Fatalf("expected tag assignment seed to succeed, got %v", err)
	}
}
