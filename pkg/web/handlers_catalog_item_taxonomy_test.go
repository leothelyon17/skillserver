package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/persistence"
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

func TestCatalogItemTaxonomyEndpoints_SingleItemPatchSupportsAdditiveTagMutations(t *testing.T) {
	t.Parallel()

	server, _ := newCatalogMetadataFixtureServer(t)
	seedCatalogTaxonomyObjectsViaAPI(t, server)

	itemID := domain.BuildSkillCatalogItemID("demo-skill")
	target := "/api/catalog/" + url.PathEscape(itemID) + "/taxonomy"

	req := httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"primary_domain_id":"domain-platform","tag_ids":["tag-backend"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"add_tag_ids":["tag-metrics"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	tags, ok := payload["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("expected two tags after add_tag_ids patch, got %+v", payload["tags"])
	}
	if isFullyClassified, ok := payload["is_fully_classified"].(bool); !ok || !isFullyClassified {
		t.Fatalf("expected is_fully_classified=true after additive patch, got %v", payload["is_fully_classified"])
	}

	req = httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"remove_tag_ids":["tag-backend"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload = decodeJSONObject(t, rec.Body.Bytes())
	tags, ok = payload["tags"].([]any)
	if !ok || len(tags) != 1 {
		t.Fatalf("expected one tag after remove_tag_ids patch, got %+v", payload["tags"])
	}

	req = httptest.NewRequest(
		http.MethodPatch,
		target,
		strings.NewReader(`{"clear_tags":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload = decodeJSONObject(t, rec.Body.Bytes())
	tags, ok = payload["tags"].([]any)
	if !ok || len(tags) != 0 {
		t.Fatalf("expected zero tags after clear_tags patch, got %+v", payload["tags"])
	}
}

func TestCatalogItemTaxonomyEndpoints_BatchPatchSupportsDryRunAndApply(t *testing.T) {
	t.Parallel()

	server, _ := newCatalogMetadataFixtureServer(t)
	seedCatalogTaxonomyObjectsViaAPI(t, server)

	localItemID := domain.BuildSkillCatalogItemID("demo-skill")
	gitItemID := domain.BuildSkillCatalogItemID("repo-a/git-skill")

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/taxonomy/batch",
		strings.NewReader(`{"dry_run":true,"items":[{"item_id":"`+localItemID+`","primary_domain_id":"domain-platform","tag_ids":["tag-backend"]},{"item_id":"`+gitItemID+`","secondary_domain_id":"domain-observability","add_tag_ids":["tag-metrics"]},{"item_id":"skill:missing-item","primary_domain_id":"domain-platform"}]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	if dryRun, ok := payload["dry_run"].(bool); !ok || !dryRun {
		t.Fatalf("expected dry_run=true, got %v", payload["dry_run"])
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 3 {
		t.Fatalf("expected three batch item results, got %+v", payload["items"])
	}
	if status, _ := items[0].(map[string]any)["status"].(string); status != "planned" {
		t.Fatalf("expected first dry-run status planned, got %+v", items[0])
	}
	if status, _ := items[2].(map[string]any)["status"].(string); status != "not_found" {
		t.Fatalf("expected missing-item dry-run status not_found, got %+v", items[2])
	}

	req = httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/taxonomy/batch",
		strings.NewReader(`{"items":[{"item_id":"`+localItemID+`","primary_domain_id":"domain-platform","tag_ids":["tag-backend"]},{"item_id":"`+gitItemID+`","secondary_domain_id":"domain-observability","add_tag_ids":["tag-metrics"]}]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload = decodeJSONObject(t, rec.Body.Bytes())
	items, ok = payload["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected two apply batch item results, got %+v", payload["items"])
	}
	if status, _ := items[0].(map[string]any)["status"].(string); status != "updated" {
		t.Fatalf("expected first apply status updated, got %+v", items[0])
	}
	if status, _ := items[1].(map[string]any)["status"].(string); status != "updated" {
		t.Fatalf("expected second apply status updated, got %+v", items[1])
	}
}

func TestCatalogItemTaxonomyEndpoints_BareSkillIDCompatibility_ForSingleAndBatchPatch(t *testing.T) {
	t.Parallel()

	server, _ := newCatalogMetadataFixtureServer(t)
	seedCatalogTaxonomyObjectsViaAPI(t, server)

	bareTarget := "/api/catalog/demo-skill/taxonomy"

	req := httptest.NewRequest(
		http.MethodPatch,
		bareTarget,
		strings.NewReader(`{"primary_domain_id":"domain-platform","tag_ids":["tag-backend"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	canonicalItemID := domain.BuildSkillCatalogItemID("demo-skill")
	if itemID, _ := payload["item_id"].(string); itemID != canonicalItemID {
		t.Fatalf("expected canonical item_id %q for bare skill patch, got %q", canonicalItemID, itemID)
	}
	if hasAssignment, ok := payload["has_assignment"].(bool); !ok || !hasAssignment {
		t.Fatalf("expected bare skill patch to set has_assignment=true, got %v", payload["has_assignment"])
	}

	req = httptest.NewRequest(http.MethodGet, bareTarget, nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload = decodeJSONObject(t, rec.Body.Bytes())
	if itemID, _ := payload["item_id"].(string); itemID != canonicalItemID {
		t.Fatalf("expected canonical item_id %q for bare skill get, got %q", canonicalItemID, itemID)
	}
	if tags, ok := payload["tags"].([]any); !ok || len(tags) != 1 {
		t.Fatalf("expected one persisted taxonomy tag after bare skill get, got %+v", payload["tags"])
	}

	req = httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/taxonomy/batch",
		strings.NewReader(`{"items":[{"item_id":"demo-skill","add_tag_ids":["tag-metrics"]}]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload = decodeJSONObject(t, rec.Body.Bytes())
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one batch result item, got %+v", payload["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected batch result item object, got %T", items[0])
	}
	if requestedItemID, _ := item["requested_item_id"].(string); requestedItemID != "demo-skill" {
		t.Fatalf("expected requested_item_id to preserve bare input, got %q", requestedItemID)
	}
	if itemID, _ := item["item_id"].(string); itemID != canonicalItemID {
		t.Fatalf("expected canonical item_id %q in batch result, got %q", canonicalItemID, itemID)
	}
	if status, _ := item["status"].(string); status != "updated" {
		t.Fatalf("expected batch apply status updated for bare item_id, got %+v", item)
	}
	assignment, ok := item["assignment"].(map[string]any)
	if !ok {
		t.Fatalf("expected assignment payload on batch apply result, got %+v", item["assignment"])
	}
	if tags, ok := assignment["tags"].([]any); !ok || len(tags) != 2 {
		t.Fatalf("expected additive batch patch to persist two tags, got %+v", assignment["tags"])
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

func TestCatalogEndpoints_ClassificationStateFilters_WorkForListAndSearch(t *testing.T) {
	t.Parallel()

	server, sourceRepo := newCatalogMetadataFixtureServer(t)
	seedCatalogTaxonomyObjectsViaAPI(t, server)

	if err := sourceRepo.Upsert(context.Background(), persistence.CatalogSourceRow{
		ItemID:           domain.BuildSkillCatalogItemID("plain-item"),
		Classifier:       persistence.CatalogClassifierSkill,
		SourceType:       persistence.CatalogSourceTypeLocal,
		Name:             "plain-item",
		Description:      "plain unclassified item",
		Content:          "plain content",
		ContentHash:      "sha256:plain-item",
		ContentWritable:  true,
		MetadataWritable: true,
		LastSyncedAt:     time.Date(2026, time.March, 5, 4, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("expected plain-item source upsert to succeed, got %v", err)
	}

	localItemID := domain.BuildSkillCatalogItemID("demo-skill")
	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/"+url.PathEscape(localItemID)+"/taxonomy",
		strings.NewReader(`{"primary_domain_id":"domain-platform","tag_ids":["tag-backend"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	gitItemID := domain.BuildSkillCatalogItemID("repo-a/git-skill")
	req = httptest.NewRequest(
		http.MethodPatch,
		"/api/catalog/"+url.PathEscape(gitItemID)+"/taxonomy",
		strings.NewReader(`{"secondary_domain_id":"domain-observability","tag_ids":["tag-metrics"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog?unclassified=true", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	unclassifiedItems := decodeJSONArray(t, rec.Body.Bytes())
	if len(unclassifiedItems) != 1 {
		t.Fatalf("expected one unclassified item, got %d payload=%q", len(unclassifiedItems), rec.Body.String())
	}
	if id, _ := unclassifiedItems[0]["id"].(string); id != domain.BuildSkillCatalogItemID("plain-item") {
		t.Fatalf("expected plain-item in unclassified filter, got %+v", unclassifiedItems[0])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog?missing_tags=true", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	missingTagItems := decodeJSONArray(t, rec.Body.Bytes())
	if len(missingTagItems) != 1 {
		t.Fatalf("expected one missing-tags item, got %d payload=%q", len(missingTagItems), rec.Body.String())
	}
	if id, _ := missingTagItems[0]["id"].(string); id != domain.BuildSkillCatalogItemID("plain-item") {
		t.Fatalf("expected plain-item in missing_tags filter, got %+v", missingTagItems[0])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog?missing_primary_domain=true", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	missingPrimaryListItems := decodeJSONArray(t, rec.Body.Bytes())
	if len(missingPrimaryListItems) != 2 {
		t.Fatalf(
			"expected two missing_primary_domain list items, got %d payload=%q",
			len(missingPrimaryListItems),
			rec.Body.String(),
		)
	}
	missingPrimaryListIDs := map[string]struct{}{}
	for _, item := range missingPrimaryListItems {
		id, _ := item["id"].(string)
		missingPrimaryListIDs[id] = struct{}{}
	}
	for _, expectedID := range []string{domain.BuildSkillCatalogItemID("plain-item"), gitItemID} {
		if _, exists := missingPrimaryListIDs[expectedID]; !exists {
			t.Fatalf("expected missing_primary_domain list filter to include %q, got %+v", expectedID, missingPrimaryListItems)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog/search?q=git&missing_primary_domain=true", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	missingPrimaryItems := decodeJSONArray(t, rec.Body.Bytes())
	if len(missingPrimaryItems) != 1 {
		t.Fatalf("expected one missing-primary-domain search item, got %d payload=%q", len(missingPrimaryItems), rec.Body.String())
	}
	if id, _ := missingPrimaryItems[0]["id"].(string); id != gitItemID {
		t.Fatalf("expected git item in missing_primary_domain search, got %+v", missingPrimaryItems[0])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog/search?q=plain&unclassified=true", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	unclassifiedSearchItems := decodeJSONArray(t, rec.Body.Bytes())
	if len(unclassifiedSearchItems) != 1 {
		t.Fatalf("expected one unclassified search item, got %d payload=%q", len(unclassifiedSearchItems), rec.Body.String())
	}
	if id, _ := unclassifiedSearchItems[0]["id"].(string); id != domain.BuildSkillCatalogItemID("plain-item") {
		t.Fatalf("expected plain-item in unclassified search, got %+v", unclassifiedSearchItems[0])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog/search?q=plain&missing_tags=true", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	missingTagsSearchItems := decodeJSONArray(t, rec.Body.Bytes())
	if len(missingTagsSearchItems) != 1 {
		t.Fatalf("expected one missing-tags search item, got %d payload=%q", len(missingTagsSearchItems), rec.Body.String())
	}
	if id, _ := missingTagsSearchItems[0]["id"].(string); id != domain.BuildSkillCatalogItemID("plain-item") {
		t.Fatalf("expected plain-item in missing_tags search, got %+v", missingTagsSearchItems[0])
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
