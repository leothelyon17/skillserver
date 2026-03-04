package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
