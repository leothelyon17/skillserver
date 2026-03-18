package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mudler/skillserver/pkg/domain"
)

func TestListCatalog_ReturnsMixedCatalogItemsWithPromptMetadata(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	items := decodeJSONArray(t, rec.Body.Bytes())
	if len(items) != 2 {
		t.Fatalf("expected exactly 2 catalog items from fixture, got %d payload=%q", len(items), rec.Body.String())
	}

	skill := findCatalogItemByClassifier(t, items, "skill")
	if id, _ := skill["id"].(string); !strings.HasPrefix(id, "skill:") {
		t.Fatalf("expected skill item id to start with skill:, got %q", id)
	}
	if name, _ := skill["name"].(string); name != "demo-skill" {
		t.Fatalf("expected skill name demo-skill, got %q", name)
	}
	if _, exists := skill["content"]; exists {
		t.Fatalf("did not expect content in metadata-first default response, got %+v", skill)
	}
	if hasAssignment, ok := skill["has_assignment"].(bool); !ok || hasAssignment {
		t.Fatalf("expected has_assignment=false for fixture skill, got %v", skill["has_assignment"])
	}
	if isFullyClassified, ok := skill["is_fully_classified"].(bool); !ok || isFullyClassified {
		t.Fatalf(
			"expected is_fully_classified=false for fixture skill, got %v",
			skill["is_fully_classified"],
		)
	}
	if missingFields, ok := skill["missing_fields"].([]any); !ok || len(missingFields) != 5 {
		t.Fatalf("expected explicit missing_fields in default response, got %+v", skill["missing_fields"])
	}

	prompt := findCatalogItemByClassifier(t, items, "prompt")
	if id, _ := prompt["id"].(string); !strings.HasPrefix(id, "prompt:") {
		t.Fatalf("expected prompt item id to start with prompt:, got %q", id)
	}
	if parentSkillID, _ := prompt["parent_skill_id"].(string); parentSkillID != "demo-skill" {
		t.Fatalf("expected parent_skill_id demo-skill, got %q", parentSkillID)
	}
	if resourcePath, _ := prompt["resource_path"].(string); resourcePath != "prompts/system.md" {
		t.Fatalf("expected resource_path prompts/system.md, got %q", resourcePath)
	}
	if readOnly, ok := prompt["read_only"].(bool); !ok || readOnly {
		t.Fatalf("expected read_only=false for direct prompt resource, got %v", prompt["read_only"])
	}
	if contentWritable, ok := prompt["content_writable"].(bool); !ok || !contentWritable {
		t.Fatalf("expected content_writable=true for direct prompt resource, got %v", prompt["content_writable"])
	}
	if metadataWritable, ok := prompt["metadata_writable"].(bool); !ok || !metadataWritable {
		t.Fatalf("expected metadata_writable=true for direct prompt resource, got %v", prompt["metadata_writable"])
	}
	if _, exists := prompt["content"]; exists {
		t.Fatalf("did not expect prompt content in metadata-first default response, got %+v", prompt)
	}

	if contentWritable, ok := skill["content_writable"].(bool); !ok || !contentWritable {
		t.Fatalf("expected content_writable=true for local skill, got %v", skill["content_writable"])
	}
	if metadataWritable, ok := skill["metadata_writable"].(bool); !ok || !metadataWritable {
		t.Fatalf("expected metadata_writable=true for local skill, got %v", skill["metadata_writable"])
	}
	if readOnly, ok := skill["read_only"].(bool); !ok || readOnly {
		t.Fatalf("expected read_only=false for local skill, got %v", skill["read_only"])
	}
}

func TestListCatalog_SupportsOptionalClassifierFiltering(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog?classifier=Prompt", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	items := decodeJSONArray(t, rec.Body.Bytes())
	if len(items) != 1 {
		t.Fatalf("expected 1 prompt catalog item, got %d payload=%q", len(items), rec.Body.String())
	}
	if classifier, _ := items[0]["classifier"].(string); classifier != "prompt" {
		t.Fatalf("expected classifier prompt, got %q", classifier)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog?classifier=skill", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	items = decodeJSONArray(t, rec.Body.Bytes())
	if len(items) != 1 {
		t.Fatalf("expected 1 skill catalog item, got %d payload=%q", len(items), rec.Body.String())
	}
	if classifier, _ := items[0]["classifier"].(string); classifier != "skill" {
		t.Fatalf("expected classifier skill, got %q", classifier)
	}
}

func TestSearchCatalog_SupportsOptionalClassifierFiltering(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/catalog/search?q=helpful&classifier=Prompt",
		nil,
	)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	items := decodeJSONArray(t, rec.Body.Bytes())
	if len(items) != 1 {
		t.Fatalf("expected 1 catalog search result, got %d payload=%q", len(items), rec.Body.String())
	}
	if classifier, _ := items[0]["classifier"].(string); classifier != "prompt" {
		t.Fatalf("expected classifier prompt, got %q", classifier)
	}
	if contentWritable, ok := items[0]["content_writable"].(bool); !ok || !contentWritable {
		t.Fatalf("expected prompt content_writable=true in search response, got %v", items[0]["content_writable"])
	}
	if metadataWritable, ok := items[0]["metadata_writable"].(bool); !ok || !metadataWritable {
		t.Fatalf("expected prompt metadata_writable=true in search response, got %v", items[0]["metadata_writable"])
	}
	if readOnly, ok := items[0]["read_only"].(bool); !ok || readOnly {
		t.Fatalf("expected prompt read_only=false in search response, got %v", items[0]["read_only"])
	}
	if _, exists := items[0]["content"]; exists {
		t.Fatalf("did not expect content in metadata-first search response, got %+v", items[0])
	}

	req = httptest.NewRequest(
		http.MethodGet,
		"/api/catalog/search?q=fixture&classifier=skill",
		nil,
	)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	items = decodeJSONArray(t, rec.Body.Bytes())
	if len(items) != 1 {
		t.Fatalf("expected 1 skill search result, got %d payload=%q", len(items), rec.Body.String())
	}
	if classifier, _ := items[0]["classifier"].(string); classifier != "skill" {
		t.Fatalf("expected classifier skill, got %q", classifier)
	}
	if contentWritable, ok := items[0]["content_writable"].(bool); !ok || !contentWritable {
		t.Fatalf("expected skill content_writable=true in search response, got %v", items[0]["content_writable"])
	}
	if metadataWritable, ok := items[0]["metadata_writable"].(bool); !ok || !metadataWritable {
		t.Fatalf("expected skill metadata_writable=true in search response, got %v", items[0]["metadata_writable"])
	}
	if readOnly, ok := items[0]["read_only"].(bool); !ok || readOnly {
		t.Fatalf("expected skill read_only=false in search response, got %v", items[0]["read_only"])
	}
	if _, exists := items[0]["content"]; exists {
		t.Fatalf("did not expect content in metadata-first search response, got %+v", items[0])
	}
}

func TestGetCatalogItem_ReturnsPromptContentByExactID(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)
	promptItemID := domain.BuildPromptCatalogItemID("demo-skill", "prompts/system.md")

	req := httptest.NewRequest(http.MethodGet, "/api/catalog/"+url.PathEscape(promptItemID), nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	item := decodeJSONObject(t, rec.Body.Bytes())
	if id, _ := item["id"].(string); id != promptItemID {
		t.Fatalf("expected prompt id %q, got %q", promptItemID, id)
	}
	if classifier, _ := item["classifier"].(string); classifier != "prompt" {
		t.Fatalf("expected classifier prompt, got %q", classifier)
	}
	if content, _ := item["content"].(string); content != "You are helpful.\n" {
		t.Fatalf("expected exact prompt content, got %q", content)
	}
}

func TestGetCatalogItem_QueryFormAcceptsBareSkillID(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog/item?item_id="+url.QueryEscape("demo-skill"), nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	item := decodeJSONObject(t, rec.Body.Bytes())
	if id, _ := item["id"].(string); id != domain.BuildSkillCatalogItemID("demo-skill") {
		t.Fatalf("expected canonical skill id %q, got %q", domain.BuildSkillCatalogItemID("demo-skill"), id)
	}
	if content, _ := item["content"].(string); !strings.Contains(content, "# Demo Skill") {
		t.Fatalf("expected skill content in exact lookup response, got %q", content)
	}
}

func TestGetCatalogItem_UsesMetadataServiceForRuleExactLookup(t *testing.T) {
	t.Parallel()

	server, sourceRepo := newCatalogMetadataFixtureServer(t)
	seedCatalogRelationshipSourceRows(t, sourceRepo, "demo-skill")
	ruleItemID := domain.BuildRuleCatalogItemID("demo-skill", "rules/security.md")

	req := httptest.NewRequest(http.MethodGet, "/api/catalog/"+url.PathEscape(ruleItemID), nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	item := decodeJSONObject(t, rec.Body.Bytes())
	if id, _ := item["id"].(string); id != ruleItemID {
		t.Fatalf("expected rule id %q, got %q", ruleItemID, id)
	}
	if classifier, _ := item["classifier"].(string); classifier != "rule" {
		t.Fatalf("expected classifier rule, got %q", classifier)
	}
	if content, _ := item["content"].(string); content != "rule content" {
		t.Fatalf("expected exact rule content, got %q", content)
	}
}

func TestGetCatalogItem_MissingItemReturnsNotFound(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)
	missingItemID := domain.BuildRuleCatalogItemID("demo-skill", "rules/missing.md")

	req := httptest.NewRequest(http.MethodGet, "/api/catalog/"+url.PathEscape(missingItemID), nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNotFound, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	if errText, _ := payload["error"].(string); errText != "catalog item not found" {
		t.Fatalf("expected catalog item not found error, got %q", errText)
	}
}

func TestCatalogEndpoints_PaginationEnvelopeAndContentOptIn(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog?limit=1", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	page := decodeJSONObject(t, rec.Body.Bytes())
	items, ok := page["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one paginated item, got %+v", page["items"])
	}
	if hasMore, ok := page["has_more"].(bool); !ok || !hasMore {
		t.Fatalf("expected has_more=true for first page, got %v", page["has_more"])
	}
	nextCursor, ok := page["next_cursor"].(string)
	if !ok || strings.TrimSpace(nextCursor) == "" {
		t.Fatalf("expected next_cursor on first page, got %+v", page["next_cursor"])
	}
	firstItem, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected paginated item object, got %T", items[0])
	}
	if _, exists := firstItem["content"]; exists {
		t.Fatalf("did not expect content without include_content=true, got %+v", firstItem)
	}

	req = httptest.NewRequest(
		http.MethodGet,
		"/api/catalog?limit=1&cursor="+url.QueryEscape(nextCursor)+"&include_content=true",
		nil,
	)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	secondPage := decodeJSONObject(t, rec.Body.Bytes())
	items, ok = secondPage["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one second-page item, got %+v", secondPage["items"])
	}
	if hasMore, ok := secondPage["has_more"].(bool); !ok || hasMore {
		t.Fatalf("expected has_more=false on final page, got %v", secondPage["has_more"])
	}
	secondItem, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected second-page item object, got %T", items[0])
	}
	if _, exists := secondItem["content"]; !exists {
		t.Fatalf("expected content when include_content=true, got %+v", secondItem)
	}
}

func TestSearchCatalog_InvalidClassifier_ReturnsValidationError(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog/search?q=fixture&classifier=skills", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "invalid catalog classifier") {
		t.Fatalf("expected invalid classifier validation message, got %q", rec.Body.String())
	}
}

func TestListCatalog_InvalidClassifier_ReturnsValidationError(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalog?classifier=skills", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "invalid catalog classifier") {
		t.Fatalf("expected invalid classifier validation message, got %q", rec.Body.String())
	}
}

func TestSearchCatalog_EmptyQueryHandling_ReturnsBadRequest(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "missing q parameter",
			target: "/api/catalog/search",
		},
		{
			name:   "whitespace only query",
			target: "/api/catalog/search?q=+++",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			rec := httptest.NewRecorder()
			server.echo.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
			}
			if !strings.Contains(strings.ToLower(rec.Body.String()), "query parameter 'q' is required") {
				t.Fatalf("expected missing query validation message, got %q", rec.Body.String())
			}
		})
	}
}

func TestCatalogEndpoints_KeepSkillsRoutesStable(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	skills := decodeJSONArray(t, rec.Body.Bytes())
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill from fixture, got %d payload=%q", len(skills), rec.Body.String())
	}
	if _, exists := skills[0]["classifier"]; exists {
		t.Fatalf("did not expect classifier field on /api/skills response, got payload=%q", rec.Body.String())
	}
	if id, _ := skills[0]["id"].(string); id != "demo-skill" {
		t.Fatalf("expected id demo-skill on /api/skills response, got %q", id)
	}
	if sourcePath, _ := skills[0]["sourcePath"].(string); sourcePath != "demo-skill" {
		t.Fatalf("expected sourcePath demo-skill on /api/skills response, got %q", sourcePath)
	}
	if _, exists := skills[0]["sourceRepo"]; exists {
		t.Fatalf("did not expect sourceRepo for local /api/skills response, got payload=%q", rec.Body.String())
	}
	if readOnly, ok := skills[0]["readOnly"].(bool); !ok || readOnly {
		t.Fatalf("expected readOnly=false for local skill, got %v", skills[0]["readOnly"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/skills/search?q=fixture", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	searchResults := decodeJSONArray(t, rec.Body.Bytes())
	if len(searchResults) != 1 {
		t.Fatalf("expected 1 /api/skills search result, got %d payload=%q", len(searchResults), rec.Body.String())
	}
	if name, _ := searchResults[0]["name"].(string); name != "demo-skill" {
		t.Fatalf("expected /api/skills search result demo-skill, got %q", name)
	}
	if id, _ := searchResults[0]["id"].(string); id != "demo-skill" {
		t.Fatalf("expected /api/skills search result id demo-skill, got %q", id)
	}
	if sourcePath, _ := searchResults[0]["sourcePath"].(string); sourcePath != "demo-skill" {
		t.Fatalf("expected /api/skills search result sourcePath demo-skill, got %q", sourcePath)
	}
}

func TestSearchCatalog_GitBackedItemsExposeSourceIdentity(t *testing.T) {
	t.Parallel()

	server := newGitBackedIdentityFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/catalog/search?q=architecture",
		nil,
	)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	items := decodeJSONArray(t, rec.Body.Bytes())
	if len(items) < 2 {
		t.Fatalf("expected at least 2 catalog search items, got %d payload=%q", len(items), rec.Body.String())
	}

	skill := findCatalogItemByClassifier(t, items, "skill")
	if sourceRepo, _ := skill["source_repo"].(string); sourceRepo != "agents" {
		t.Fatalf("expected git skill source_repo agents, got %q", sourceRepo)
	}
	if sourcePath, _ := skill["source_path"].(string); sourcePath != "plugins/documentation-generation/skills/architecture-decision-records" {
		t.Fatalf("expected git skill source_path to identify nested repo path, got %q", sourcePath)
	}

	prompt := findCatalogItemByResourcePath(t, items, "prompts/architecture-decision-record-template.md")
	if sourceRepo, _ := prompt["source_repo"].(string); sourceRepo != "agents" {
		t.Fatalf("expected git prompt source_repo agents, got %q", sourceRepo)
	}
	if sourcePath, _ := prompt["source_path"].(string); sourcePath != "plugins/documentation-generation/skills/architecture-decision-records" {
		t.Fatalf("expected git prompt source_path to identify owning skill path, got %q", sourcePath)
	}
	if parentSkillID, _ := prompt["parent_skill_id"].(string); parentSkillID != "agents/architecture-decision-records" {
		t.Fatalf("expected git prompt parent_skill_id agents/architecture-decision-records, got %q", parentSkillID)
	}
}

func decodeJSONArray(t *testing.T, body []byte) []map[string]any {
	t.Helper()

	var payload []map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode json array payload: %v body=%q", err, string(body))
	}
	return payload
}

func findCatalogItemByClassifier(t *testing.T, items []map[string]any, classifier string) map[string]any {
	t.Helper()

	for _, item := range items {
		if value, _ := item["classifier"].(string); value == classifier {
			return item
		}
	}

	t.Fatalf("expected catalog item with classifier %q, got %+v", classifier, items)
	return nil
}

func findCatalogItemByResourcePath(t *testing.T, items []map[string]any, resourcePath string) map[string]any {
	t.Helper()

	for _, item := range items {
		if value, _ := item["resource_path"].(string); value == resourcePath {
			return item
		}
	}

	t.Fatalf("expected catalog item with resource_path %q, got %+v", resourcePath, items)
	return nil
}
