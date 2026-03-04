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

func TestCatalogMetadataEndpoints_PatchAndGet_SupportsLocalAndGitItems(t *testing.T) {
	t.Parallel()

	server, sourceRepo := newCatalogMetadataFixtureServer(t)
	ctx := context.Background()

	localItemID := domain.BuildSkillCatalogItemID("demo-skill")
	localTarget := "/api/catalog/" + url.PathEscape(localItemID) + "/metadata"

	patchLocalRequest := `{"display_name":"Demo Skill Overlay","description":"Overlay description","labels":["catalog","local"],"custom_metadata":{"owner":"platform","priority":1}}`
	req := httptest.NewRequest(http.MethodPatch, localTarget, strings.NewReader(patchLocalRequest))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	patchPayload := decodeJSONObject(t, rec.Body.Bytes())
	effective, ok := patchPayload["effective"].(map[string]any)
	if !ok {
		t.Fatalf("expected effective metadata object, got %T", patchPayload["effective"])
	}
	if name, _ := effective["name"].(string); name != "Demo Skill Overlay" {
		t.Fatalf("expected effective name override, got %q", name)
	}
	if description, _ := effective["description"].(string); description != "Overlay description" {
		t.Fatalf("expected effective description override, got %q", description)
	}
	if contentWritable, ok := effective["content_writable"].(bool); !ok || !contentWritable {
		t.Fatalf("expected local effective content_writable=true, got %v", effective["content_writable"])
	}
	if metadataWritable, ok := effective["metadata_writable"].(bool); !ok || !metadataWritable {
		t.Fatalf("expected local effective metadata_writable=true, got %v", effective["metadata_writable"])
	}
	if readOnly, ok := effective["read_only"].(bool); !ok || readOnly {
		t.Fatalf("expected local effective read_only=false, got %v", effective["read_only"])
	}

	gitItemID := domain.BuildSkillCatalogItemID("repo-a/git-skill")
	gitTarget := "/api/catalog/" + url.PathEscape(gitItemID) + "/metadata"

	patchGitRequest := `{"display_name":"Git Skill Overlay","labels":["git-item"]}`
	req = httptest.NewRequest(http.MethodPatch, gitTarget, strings.NewReader(patchGitRequest))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	gitPatchPayload := decodeJSONObject(t, rec.Body.Bytes())
	gitEffective, ok := gitPatchPayload["effective"].(map[string]any)
	if !ok {
		t.Fatalf("expected git effective metadata object, got %T", gitPatchPayload["effective"])
	}
	if contentWritable, ok := gitEffective["content_writable"].(bool); !ok || contentWritable {
		t.Fatalf("expected git effective content_writable=false, got %v", gitEffective["content_writable"])
	}
	if metadataWritable, ok := gitEffective["metadata_writable"].(bool); !ok || !metadataWritable {
		t.Fatalf("expected git effective metadata_writable=true, got %v", gitEffective["metadata_writable"])
	}
	if readOnly, ok := gitEffective["read_only"].(bool); !ok || !readOnly {
		t.Fatalf("expected git effective read_only=true, got %v", gitEffective["read_only"])
	}

	gitSourceRow, err := sourceRepo.GetByItemID(ctx, gitItemID)
	if err != nil {
		t.Fatalf("expected git source row lookup to succeed, got %v", err)
	}
	if gitSourceRow.ContentWritable {
		t.Fatalf("expected git source content_writable to remain false after metadata patch")
	}

	req = httptest.NewRequest(http.MethodGet, gitTarget, nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	getPayload := decodeJSONObject(t, rec.Body.Bytes())
	overlay, ok := getPayload["overlay"].(map[string]any)
	if !ok {
		t.Fatalf("expected overlay metadata object, got %T", getPayload["overlay"])
	}
	if displayName, _ := overlay["display_name"].(string); displayName != "Git Skill Overlay" {
		t.Fatalf("expected overlay display_name to persist, got %q", displayName)
	}
	labels, ok := overlay["labels"].([]any)
	if !ok || len(labels) != 1 || labels[0] != "git-item" {
		t.Fatalf("expected persisted overlay labels [git-item], got %+v", overlay["labels"])
	}
}

func TestCatalogMetadataEndpoints_ListAndSearch_UseEffectiveOverlayProjection(t *testing.T) {
	t.Parallel()

	server, _ := newCatalogMetadataFixtureServer(t)
	localItemID := domain.BuildSkillCatalogItemID("demo-skill")
	localTarget := "/api/catalog/" + url.PathEscape(localItemID) + "/metadata"

	patchRequest := `{"display_name":"Local Overlay Search Name","description":"overlay search description","labels":["overlay","local-search"]}`
	req := httptest.NewRequest(http.MethodPatch, localTarget, strings.NewReader(patchRequest))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	items := decodeJSONArray(t, rec.Body.Bytes())
	localItem := findCatalogItemByID(t, items, localItemID)
	if name, _ := localItem["name"].(string); name != "Local Overlay Search Name" {
		t.Fatalf("expected list catalog name override, got %q", name)
	}
	if contentWritable, ok := localItem["content_writable"].(bool); !ok || !contentWritable {
		t.Fatalf("expected list catalog content_writable=true for local item, got %v", localItem["content_writable"])
	}
	if metadataWritable, ok := localItem["metadata_writable"].(bool); !ok || !metadataWritable {
		t.Fatalf("expected list catalog metadata_writable=true for local item, got %v", localItem["metadata_writable"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog/search?q=Local+Overlay+Search+Name", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	searchByName := decodeJSONArray(t, rec.Body.Bytes())
	if len(searchByName) != 1 {
		t.Fatalf("expected one overlay search name hit, got %d payload=%q", len(searchByName), rec.Body.String())
	}
	if id, _ := searchByName[0]["id"].(string); id != localItemID {
		t.Fatalf("expected overlay search to return local item %q, got %q", localItemID, id)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/catalog/search?q=local-search", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	searchByLabel := decodeJSONArray(t, rec.Body.Bytes())
	if len(searchByLabel) != 1 {
		t.Fatalf("expected one overlay label hit, got %d payload=%q", len(searchByLabel), rec.Body.String())
	}
	if id, _ := searchByLabel[0]["id"].(string); id != localItemID {
		t.Fatalf("expected overlay label search to return local item %q, got %q", localItemID, id)
	}
}

func TestCatalogMetadataEndpoints_ValidationAndMissingItems(t *testing.T) {
	t.Parallel()

	server, _ := newCatalogMetadataFixtureServer(t)
	localItemID := domain.BuildSkillCatalogItemID("demo-skill")
	localTarget := "/api/catalog/" + url.PathEscape(localItemID) + "/metadata"

	req := httptest.NewRequest(http.MethodPatch, localTarget, strings.NewReader(`{"unknown":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "invalid request payload") {
		t.Fatalf("expected invalid payload message, got %q", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, localTarget, strings.NewReader(`{"labels":["", "ok"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "labels cannot contain empty values") {
		t.Fatalf("expected label validation message, got %q", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, localTarget, strings.NewReader(`{"updated_by":"tester"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "at least one") {
		t.Fatalf("expected at least one field validation message, got %q", rec.Body.String())
	}

	missingItemTarget := "/api/catalog/" + url.PathEscape(domain.BuildSkillCatalogItemID("missing-item")) + "/metadata"
	req = httptest.NewRequest(http.MethodPatch, missingItemTarget, strings.NewReader(`{"display_name":"Nope"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNotFound, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, missingItemTarget, nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNotFound, rec.Code, rec.Body.String())
	}
}

func TestCatalogMetadataEndpoints_ServiceUnavailable_Returns503(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)
	itemID := domain.BuildSkillCatalogItemID("demo-skill")
	target := "/api/catalog/" + url.PathEscape(itemID) + "/metadata"

	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, target, strings.NewReader(`{"display_name":"ignored"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}
}

func newCatalogMetadataFixtureServer(t *testing.T) (*Server, *persistence.CatalogSourceRepository) {
	t.Helper()

	server := newResourceFixtureServer(t)

	dbPath := t.TempDir() + "/catalog-metadata-api.db"
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
	overlayRepo, err := persistence.NewCatalogMetadataOverlayRepository(db)
	if err != nil {
		t.Fatalf("expected overlay repository creation to succeed, got %v", err)
	}
	effectiveService, err := domain.NewCatalogEffectiveService(sourceRepo, overlayRepo)
	if err != nil {
		t.Fatalf("expected effective catalog service creation to succeed, got %v", err)
	}
	metadataService, err := domain.NewCatalogMetadataService(
		sourceRepo,
		overlayRepo,
		effectiveService,
		domain.CatalogMetadataServiceOptions{
			Now: func() time.Time {
				return time.Date(2026, time.March, 5, 2, 0, 0, 0, time.UTC)
			},
		},
	)
	if err != nil {
		t.Fatalf("expected catalog metadata service creation to succeed, got %v", err)
	}

	seedCatalogMetadataSourceRows(t, sourceRepo)
	server.SetCatalogMetadataService(metadataService)

	return server, sourceRepo
}

func seedCatalogMetadataSourceRows(t *testing.T, sourceRepo *persistence.CatalogSourceRepository) {
	t.Helper()

	ctx := context.Background()
	syncedAt := time.Date(2026, time.March, 5, 1, 30, 0, 0, time.UTC)

	repoName := "repo-a"
	rows := []persistence.CatalogSourceRow{
		{
			ItemID:           domain.BuildSkillCatalogItemID("demo-skill"),
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeLocal,
			Name:             "demo-skill",
			Description:      "fixture local skill",
			Content:          "local content",
			ContentHash:      "sha256:local",
			ContentWritable:  true,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
		{
			ItemID:           domain.BuildSkillCatalogItemID("repo-a/git-skill"),
			Classifier:       persistence.CatalogClassifierSkill,
			SourceType:       persistence.CatalogSourceTypeGit,
			SourceRepo:       &repoName,
			Name:             "git-skill",
			Description:      "fixture git skill",
			Content:          "git content",
			ContentHash:      "sha256:git",
			ContentWritable:  false,
			MetadataWritable: true,
			LastSyncedAt:     syncedAt,
		},
	}

	for _, row := range rows {
		if err := sourceRepo.Upsert(ctx, row); err != nil {
			t.Fatalf("expected source upsert for %q to succeed, got %v", row.ItemID, err)
		}
	}
}

func findCatalogItemByID(t *testing.T, items []map[string]any, itemID string) map[string]any {
	t.Helper()

	for _, item := range items {
		if value, _ := item["id"].(string); value == itemID {
			return item
		}
	}

	t.Fatalf("expected catalog item id %q, got %+v", itemID, items)
	return nil
}
