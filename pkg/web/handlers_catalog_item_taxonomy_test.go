package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mudler/skillserver/pkg/domain"
)

func TestCatalogItemTaxonomyEndpoints_ServiceUnavailable_Returns503(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)
	itemID := domain.BuildSkillCatalogItemID("demo-skill")
	target := "/api/catalog/" + url.PathEscape(itemID) + "/taxonomy"

	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, target, strings.NewReader(`{"primary_domain_id":"domain-platform"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}
}

func TestCatalogItemTaxonomyEndpoints_GetAndPatch_MapsSuccessAndErrors(t *testing.T) {
	t.Parallel()

	server, _ := newCatalogMetadataFixtureServer(t)
	seedCatalogTaxonomyObjectsViaAPI(t, server)

	itemID := domain.BuildSkillCatalogItemID("demo-skill")
	target := "/api/catalog/" + url.PathEscape(itemID) + "/taxonomy"

	getReq := httptest.NewRequest(http.MethodGet, target, nil)
	getRec := httptest.NewRecorder()
	server.echo.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, getRec.Code, getRec.Body.String())
	}

	initialPayload := decodeJSONObject(t, getRec.Body.Bytes())
	if payloadItemID, _ := initialPayload["item_id"].(string); payloadItemID != itemID {
		t.Fatalf("expected item_id %q, got %q", itemID, payloadItemID)
	}
	initialTags, ok := initialPayload["tags"].([]any)
	if !ok || len(initialTags) != 0 {
		t.Fatalf("expected empty initial tags, got %+v", initialPayload["tags"])
	}

	patchBody := `{"primary_domain_id":"domain-platform","primary_subdomain_id":"subdomain-platform-api","secondary_domain_id":"domain-observability","secondary_subdomain_id":"subdomain-observability-metrics","tag_ids":["tag-backend","tag-metrics"],"updated_by":"tester"}`
	patchReq := httptest.NewRequest(http.MethodPatch, target, strings.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRec := httptest.NewRecorder()
	server.echo.ServeHTTP(patchRec, patchReq)

	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, patchRec.Code, patchRec.Body.String())
	}

	patchPayload := decodeJSONObject(t, patchRec.Body.Bytes())
	primaryDomain, ok := patchPayload["primary_domain"].(map[string]any)
	if !ok {
		t.Fatalf("expected primary_domain object, got %T", patchPayload["primary_domain"])
	}
	if id, _ := primaryDomain["id"].(string); id != "domain-platform" {
		t.Fatalf("expected primary_domain.id=domain-platform, got %q", id)
	}

	tagValues, ok := patchPayload["tags"].([]any)
	if !ok || len(tagValues) != 2 {
		t.Fatalf("expected two taxonomy tags, got %+v", patchPayload["tags"])
	}
	tagIDs := map[string]struct{}{}
	for _, rawTag := range tagValues {
		tag, ok := rawTag.(map[string]any)
		if !ok {
			t.Fatalf("expected taxonomy tag object, got %T", rawTag)
		}
		tagID, _ := tag["id"].(string)
		tagIDs[tagID] = struct{}{}
	}
	for _, expected := range []string{"tag-backend", "tag-metrics"} {
		if _, exists := tagIDs[expected]; !exists {
			t.Fatalf("expected response tags to include %q, got %+v", expected, patchPayload["tags"])
		}
	}

	invalidRelationshipReq := httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"primary_domain_id":"domain-observability","primary_subdomain_id":"subdomain-platform-api"}`),
	)
	invalidRelationshipReq.Header.Set("Content-Type", "application/json")
	invalidRelationshipRec := httptest.NewRecorder()
	server.echo.ServeHTTP(invalidRelationshipRec, invalidRelationshipReq)
	if invalidRelationshipRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, invalidRelationshipRec.Code, invalidRelationshipRec.Body.String())
	}

	missingTagReq := httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"tag_ids":["tag-missing"]}`),
	)
	missingTagReq.Header.Set("Content-Type", "application/json")
	missingTagRec := httptest.NewRecorder()
	server.echo.ServeHTTP(missingTagRec, missingTagReq)
	if missingTagRec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNotFound, missingTagRec.Code, missingTagRec.Body.String())
	}

	missingItemID := domain.BuildSkillCatalogItemID("missing-item")
	missingItemTarget := "/api/catalog/" + url.PathEscape(missingItemID) + "/taxonomy"
	missingItemReq := httptest.NewRequest(http.MethodGet, missingItemTarget, nil)
	missingItemRec := httptest.NewRecorder()
	server.echo.ServeHTTP(missingItemRec, missingItemReq)
	if missingItemRec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNotFound, missingItemRec.Code, missingItemRec.Body.String())
	}
}

func TestCatalogEndpoints_TaxonomyFilters_AreConsistentBetweenListAndSearch(t *testing.T) {
	t.Parallel()

	server, _ := newCatalogMetadataFixtureServer(t)
	seedCatalogTaxonomyObjectsViaAPI(t, server)

	localItemID := domain.BuildSkillCatalogItemID("demo-skill")
	localTarget := "/api/catalog/" + url.PathEscape(localItemID) + "/taxonomy"
	localPatchReq := httptest.NewRequest(
		http.MethodPatch,
		localTarget,
		strings.NewReader(`{"primary_domain_id":"domain-platform","primary_subdomain_id":"subdomain-platform-api","tag_ids":["tag-backend","tag-metrics"]}`),
	)
	localPatchReq.Header.Set("Content-Type", "application/json")
	localPatchRec := httptest.NewRecorder()
	server.echo.ServeHTTP(localPatchRec, localPatchReq)
	if localPatchRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, localPatchRec.Code, localPatchRec.Body.String())
	}

	gitItemID := domain.BuildSkillCatalogItemID("repo-a/git-skill")
	gitTarget := "/api/catalog/" + url.PathEscape(gitItemID) + "/taxonomy"
	gitPatchReq := httptest.NewRequest(
		http.MethodPatch,
		gitTarget,
		strings.NewReader(`{"primary_domain_id":"domain-observability","secondary_domain_id":"domain-platform","secondary_subdomain_id":"subdomain-platform-api","tag_ids":["tag-metrics"]}`),
	)
	gitPatchReq.Header.Set("Content-Type", "application/json")
	gitPatchRec := httptest.NewRecorder()
	server.echo.ServeHTTP(gitPatchRec, gitPatchReq)
	if gitPatchRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, gitPatchRec.Code, gitPatchRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	listRec := httptest.NewRecorder()
	server.echo.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	unfiltered := decodeJSONArray(t, listRec.Body.Bytes())
	if len(unfiltered) != 2 {
		t.Fatalf("expected 2 unfiltered catalog items, got %d payload=%q", len(unfiltered), listRec.Body.String())
	}

	filteredListReq := httptest.NewRequest(http.MethodGet, "/api/catalog?primary_domain_id=domain-platform", nil)
	filteredListRec := httptest.NewRecorder()
	server.echo.ServeHTTP(filteredListRec, filteredListReq)
	if filteredListRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, filteredListRec.Code, filteredListRec.Body.String())
	}
	primaryDomainItems := decodeJSONArray(t, filteredListRec.Body.Bytes())
	if len(primaryDomainItems) != 1 {
		t.Fatalf("expected one primary-domain filtered item, got %d payload=%q", len(primaryDomainItems), filteredListRec.Body.String())
	}
	if id, _ := primaryDomainItems[0]["id"].(string); id != localItemID {
		t.Fatalf("expected primary-domain filter to return %q, got %q", localItemID, id)
	}

	filteredSearchReq := httptest.NewRequest(
		http.MethodGet,
		"/api/catalog/search?q=skill&primary_domain_id=domain-platform",
		nil,
	)
	filteredSearchRec := httptest.NewRecorder()
	server.echo.ServeHTTP(filteredSearchRec, filteredSearchReq)
	if filteredSearchRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, filteredSearchRec.Code, filteredSearchRec.Body.String())
	}
	primaryDomainSearchItems := decodeJSONArray(t, filteredSearchRec.Body.Bytes())
	if len(primaryDomainSearchItems) != 1 {
		t.Fatalf(
			"expected one primary-domain search item, got %d payload=%q",
			len(primaryDomainSearchItems),
			filteredSearchRec.Body.String(),
		)
	}
	if id, _ := primaryDomainSearchItems[0]["id"].(string); id != localItemID {
		t.Fatalf("expected primary-domain search to return %q, got %q", localItemID, id)
	}

	secondaryDomainListReq := httptest.NewRequest(http.MethodGet, "/api/catalog?secondary_domain_id=domain-platform", nil)
	secondaryDomainListRec := httptest.NewRecorder()
	server.echo.ServeHTTP(secondaryDomainListRec, secondaryDomainListReq)
	if secondaryDomainListRec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d body=%q",
			http.StatusOK,
			secondaryDomainListRec.Code,
			secondaryDomainListRec.Body.String(),
		)
	}
	secondaryDomainListItems := decodeJSONArray(t, secondaryDomainListRec.Body.Bytes())
	if len(secondaryDomainListItems) != 1 {
		t.Fatalf(
			"expected one secondary-domain filtered list item, got %d payload=%q",
			len(secondaryDomainListItems),
			secondaryDomainListRec.Body.String(),
		)
	}
	if id, _ := secondaryDomainListItems[0]["id"].(string); id != gitItemID {
		t.Fatalf("expected secondary-domain list filter to return %q, got %q", gitItemID, id)
	}

	secondaryDomainSearchReq := httptest.NewRequest(
		http.MethodGet,
		"/api/catalog/search?q=skill&secondary_domain_id=domain-platform",
		nil,
	)
	secondaryDomainSearchRec := httptest.NewRecorder()
	server.echo.ServeHTTP(secondaryDomainSearchRec, secondaryDomainSearchReq)
	if secondaryDomainSearchRec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d body=%q",
			http.StatusOK,
			secondaryDomainSearchRec.Code,
			secondaryDomainSearchRec.Body.String(),
		)
	}
	secondaryDomainSearchItems := decodeJSONArray(t, secondaryDomainSearchRec.Body.Bytes())
	if len(secondaryDomainSearchItems) != 1 {
		t.Fatalf(
			"expected one secondary-domain filtered search item, got %d payload=%q",
			len(secondaryDomainSearchItems),
			secondaryDomainSearchRec.Body.String(),
		)
	}
	if id, _ := secondaryDomainSearchItems[0]["id"].(string); id != gitItemID {
		t.Fatalf("expected secondary-domain search filter to return %q, got %q", gitItemID, id)
	}

	subdomainListReq := httptest.NewRequest(http.MethodGet, "/api/catalog?subdomain_id=subdomain-platform-api", nil)
	subdomainListRec := httptest.NewRecorder()
	server.echo.ServeHTTP(subdomainListRec, subdomainListReq)
	if subdomainListRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, subdomainListRec.Code, subdomainListRec.Body.String())
	}
	subdomainListItems := decodeJSONArray(t, subdomainListRec.Body.Bytes())
	if len(subdomainListItems) != 2 {
		t.Fatalf(
			"expected two subdomain-filtered list items (primary+secondary matches), got %d payload=%q",
			len(subdomainListItems),
			subdomainListRec.Body.String(),
		)
	}

	subdomainSearchReq := httptest.NewRequest(
		http.MethodGet,
		"/api/catalog/search?q=skill&subdomain_id=subdomain-platform-api",
		nil,
	)
	subdomainSearchRec := httptest.NewRecorder()
	server.echo.ServeHTTP(subdomainSearchRec, subdomainSearchReq)
	if subdomainSearchRec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d body=%q",
			http.StatusOK,
			subdomainSearchRec.Code,
			subdomainSearchRec.Body.String(),
		)
	}
	subdomainSearchItems := decodeJSONArray(t, subdomainSearchRec.Body.Bytes())
	if len(subdomainSearchItems) != 2 {
		t.Fatalf(
			"expected two subdomain-filtered search items (primary+secondary matches), got %d payload=%q",
			len(subdomainSearchItems),
			subdomainSearchRec.Body.String(),
		)
	}

	tagAnyReq := httptest.NewRequest(http.MethodGet, "/api/catalog?tag_ids=tag-metrics", nil)
	tagAnyRec := httptest.NewRecorder()
	server.echo.ServeHTTP(tagAnyRec, tagAnyReq)
	if tagAnyRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, tagAnyRec.Code, tagAnyRec.Body.String())
	}
	tagAnyItems := decodeJSONArray(t, tagAnyRec.Body.Bytes())
	if len(tagAnyItems) != 2 {
		t.Fatalf("expected 2 tag-match-any items, got %d payload=%q", len(tagAnyItems), tagAnyRec.Body.String())
	}

	tagAnySearchReq := httptest.NewRequest(
		http.MethodGet,
		"/api/catalog/search?q=skill&tag_ids=tag-metrics",
		nil,
	)
	tagAnySearchRec := httptest.NewRecorder()
	server.echo.ServeHTTP(tagAnySearchRec, tagAnySearchReq)
	if tagAnySearchRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, tagAnySearchRec.Code, tagAnySearchRec.Body.String())
	}
	tagAnySearchItems := decodeJSONArray(t, tagAnySearchRec.Body.Bytes())
	if len(tagAnySearchItems) != 2 {
		t.Fatalf("expected 2 tag-match-any search items, got %d payload=%q", len(tagAnySearchItems), tagAnySearchRec.Body.String())
	}

	tagAllListReq := httptest.NewRequest(
		http.MethodGet,
		"/api/catalog?tag_ids=tag-backend,tag-metrics&tag_match=all",
		nil,
	)
	tagAllListRec := httptest.NewRecorder()
	server.echo.ServeHTTP(tagAllListRec, tagAllListReq)
	if tagAllListRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, tagAllListRec.Code, tagAllListRec.Body.String())
	}
	tagAllListItems := decodeJSONArray(t, tagAllListRec.Body.Bytes())
	if len(tagAllListItems) != 1 {
		t.Fatalf("expected one tag-match-all list item, got %d payload=%q", len(tagAllListItems), tagAllListRec.Body.String())
	}
	if id, _ := tagAllListItems[0]["id"].(string); id != localItemID {
		t.Fatalf("expected tag-match-all list to return %q, got %q", localItemID, id)
	}

	tagAllSearchReq := httptest.NewRequest(
		http.MethodGet,
		"/api/catalog/search?q=skill&tag_ids=tag-backend,tag-metrics&tag_match=all",
		nil,
	)
	tagAllSearchRec := httptest.NewRecorder()
	server.echo.ServeHTTP(tagAllSearchRec, tagAllSearchReq)
	if tagAllSearchRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, tagAllSearchRec.Code, tagAllSearchRec.Body.String())
	}
	tagAllSearchItems := decodeJSONArray(t, tagAllSearchRec.Body.Bytes())
	if len(tagAllSearchItems) != 1 {
		t.Fatalf("expected one tag-match-all search item, got %d payload=%q", len(tagAllSearchItems), tagAllSearchRec.Body.String())
	}
	if id, _ := tagAllSearchItems[0]["id"].(string); id != localItemID {
		t.Fatalf("expected tag-match-all search to return %q, got %q", localItemID, id)
	}

	invalidTagMatchReq := httptest.NewRequest(http.MethodGet, "/api/catalog?tag_match=invalid", nil)
	invalidTagMatchRec := httptest.NewRecorder()
	server.echo.ServeHTTP(invalidTagMatchRec, invalidTagMatchReq)
	if invalidTagMatchRec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, invalidTagMatchRec.Code, invalidTagMatchRec.Body.String())
	}
	if !strings.Contains(strings.ToLower(invalidTagMatchRec.Body.String()), "tag_match") {
		t.Fatalf("expected tag_match validation message, got %q", invalidTagMatchRec.Body.String())
	}
}

func TestCatalogEndpoints_TaxonomyFilters_RequirePersistenceRuntime(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog?primary_domain_id=domain-platform", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog/search?q=skill&tag_ids=tag-backend", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}
}

func seedCatalogTaxonomyObjectsViaAPI(t *testing.T, server *Server) {
	t.Helper()

	requests := []struct {
		target string
		body   string
	}{
		{
			target: "/api/catalog/taxonomy/domains",
			body:   `{"domain_id":"domain-platform","key":"platform","name":"Platform"}`,
		},
		{
			target: "/api/catalog/taxonomy/domains",
			body:   `{"domain_id":"domain-observability","key":"observability","name":"Observability"}`,
		},
		{
			target: "/api/catalog/taxonomy/subdomains",
			body:   `{"subdomain_id":"subdomain-platform-api","domain_id":"domain-platform","key":"api","name":"API"}`,
		},
		{
			target: "/api/catalog/taxonomy/subdomains",
			body:   `{"subdomain_id":"subdomain-observability-metrics","domain_id":"domain-observability","key":"metrics","name":"Metrics"}`,
		},
		{
			target: "/api/catalog/taxonomy/tags",
			body:   `{"tag_id":"tag-backend","key":"backend","name":"Backend"}`,
		},
		{
			target: "/api/catalog/taxonomy/tags",
			body:   `{"tag_id":"tag-metrics","key":"metrics","name":"Metrics"}`,
		},
	}

	for _, request := range requests {
		req := httptest.NewRequest(http.MethodPost, request.target, strings.NewReader(request.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.echo.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d for %q, got %d body=%q", http.StatusCreated, request.target, rec.Code, rec.Body.String())
		}
	}
}
