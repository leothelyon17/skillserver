package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mudler/skillserver/pkg/domain"
)

func TestExportSkill_LegacyRoute_LocalSkill_PreservesDownloadHeaders(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/skills/export/demo-skill", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/gzip" {
		t.Fatalf("expected content-type %q, got %q", "application/gzip", contentType)
	}
	if disposition := rec.Header().Get("Content-Disposition"); disposition != `attachment; filename="demo-skill.tar.gz"` {
		t.Fatalf("expected content-disposition %q, got %q", `attachment; filename="demo-skill.tar.gz"`, disposition)
	}
	if contentLength := rec.Header().Get("Content-Length"); contentLength == "" {
		t.Fatalf("expected content-length header to be populated")
	}
	if len(rec.Body.Bytes()) == 0 {
		t.Fatalf("expected non-empty archive body")
	}

	importDir := t.TempDir()
	importedSkillName, err := domain.ImportSkill(rec.Body.Bytes(), importDir)
	if err != nil {
		t.Fatalf("expected legacy export archive to be importable, got %v", err)
	}
	if importedSkillName != "demo-skill" {
		t.Fatalf("expected imported skill name %q, got %q", "demo-skill", importedSkillName)
	}
}

func TestExportSkill_LegacyRoute_RepoBackedSkillWithSlash_SupportsEncodedPath(t *testing.T) {
	t.Parallel()

	server := newExportGitBackedFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/skills/export/agents%2Fscreen-reader-testing",
		nil,
	)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	if disposition := rec.Header().Get("Content-Disposition"); disposition != `attachment; filename="agents/screen-reader-testing.tar.gz"` {
		t.Fatalf(
			"expected content-disposition %q, got %q",
			`attachment; filename="agents/screen-reader-testing.tar.gz"`,
			disposition,
		)
	}

	importDir := t.TempDir()
	importedSkillName, err := domain.ImportSkill(rec.Body.Bytes(), importDir)
	if err != nil {
		t.Fatalf("expected repo-backed export archive to be importable, got %v", err)
	}
	if importedSkillName != "screen-reader-testing" {
		t.Fatalf(
			"expected imported skill name %q, got %q",
			"screen-reader-testing",
			importedSkillName,
		)
	}
}

func TestExportSkill_LegacyRoute_MissingSkill_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/skills/export/missing-skill", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNotFound, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "catalog export item not found") {
		t.Fatalf("expected missing-item error message, got %q", rec.Body.String())
	}
}

func TestExportCatalog_DryRun_ReturnsManifestWithoutDownloadMetadata(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/export",
		strings.NewReader(`{"item_ids":["skill:demo-skill"],"dry_run":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	if dryRun, ok := payload["dry_run"].(bool); !ok || !dryRun {
		t.Fatalf("expected dry_run=true response, got %v", payload["dry_run"])
	}
	if format, _ := payload["format"].(string); format != "tar.gz" {
		t.Fatalf("expected format %q, got %q", "tar.gz", format)
	}
	if _, exists := payload["download"]; exists {
		t.Fatalf("did not expect download metadata for dry-run response")
	}

	manifest, ok := payload["manifest"].(map[string]any)
	if !ok {
		t.Fatalf("expected manifest object, got %T", payload["manifest"])
	}
	items, ok := manifest["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one manifest item, got %v", manifest["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected manifest item object, got %T", items[0])
	}
	if itemID, _ := item["item_id"].(string); itemID != "skill:demo-skill" {
		t.Fatalf("expected manifest item_id %q, got %q", "skill:demo-skill", itemID)
	}
}

func TestExportCatalog_NonDryRun_ReturnsDownloadMetadata(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/export",
		strings.NewReader(`{"item_ids":["demo-skill"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	if dryRun, ok := payload["dry_run"].(bool); !ok || dryRun {
		t.Fatalf("expected dry_run=false response, got %v", payload["dry_run"])
	}

	download, ok := payload["download"].(map[string]any)
	if !ok {
		t.Fatalf("expected download metadata object, got %T", payload["download"])
	}
	if fileName, _ := download["file_name"].(string); fileName != "demo-skill.tar.gz" {
		t.Fatalf("expected file_name %q, got %q", "demo-skill.tar.gz", fileName)
	}
	if contentType, _ := download["content_type"].(string); contentType != "application/gzip" {
		t.Fatalf("expected content_type %q, got %q", "application/gzip", contentType)
	}
	contentLength, ok := download["content_length"].(float64)
	if !ok || contentLength <= 0 {
		t.Fatalf("expected positive content_length, got %v", download["content_length"])
	}
	if legacyURL, _ := download["legacy_skill_export_url"].(string); legacyURL != "/api/skills/export/demo-skill" {
		t.Fatalf(
			"expected legacy_skill_export_url %q, got %q",
			"/api/skills/export/demo-skill",
			legacyURL,
		)
	}
}

func TestExportCatalog_RepoBackedSkillWithSlash_DryRunManifestUsesCanonicalIDs(t *testing.T) {
	t.Parallel()

	server := newExportGitBackedFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/export",
		strings.NewReader(`{"item_ids":["skill:agents/screen-reader-testing"],"dry_run":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	manifest, ok := payload["manifest"].(map[string]any)
	if !ok {
		t.Fatalf("expected manifest object, got %T", payload["manifest"])
	}
	items, ok := manifest["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one manifest item, got %v", manifest["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected manifest item object, got %T", items[0])
	}
	if itemID, _ := item["item_id"].(string); itemID != "skill:agents/screen-reader-testing" {
		t.Fatalf(
			"expected manifest item_id %q, got %q",
			"skill:agents/screen-reader-testing",
			itemID,
		)
	}
}

func TestExportCatalog_DryRun_BatchMixedClassifiers_ReturnsManifest(t *testing.T) {
	t.Parallel()

	server := newCatalogExportMaterializationFixtureServer(t)

	skillID := mustFindCatalogItemID(t, server, domain.CatalogClassifierSkill, "")
	promptID := mustFindCatalogItemID(t, server, domain.CatalogClassifierPrompt, "prompts/system.md")
	ruleID := mustFindCatalogItemID(t, server, domain.CatalogClassifierRule, "")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/export",
		strings.NewReader(
			`{"item_ids":["`+skillID+`","`+promptID+`","`+ruleID+`"],"dry_run":true}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	if _, exists := payload["download"]; exists {
		t.Fatalf("did not expect download metadata for dry-run response")
	}

	manifest, ok := payload["manifest"].(map[string]any)
	if !ok {
		t.Fatalf("expected manifest object, got %T", payload["manifest"])
	}
	items, ok := manifest["items"].([]any)
	if !ok || len(items) != 3 {
		t.Fatalf("expected three manifest items, got %v", manifest["items"])
	}

	itemIDs := make(map[string]struct{}, len(items))
	classifiers := make(map[string]struct{}, len(items))
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			t.Fatalf("expected manifest item object, got %T", rawItem)
		}
		itemID, _ := item["item_id"].(string)
		classifier, _ := item["classifier"].(string)
		itemIDs[itemID] = struct{}{}
		classifiers[classifier] = struct{}{}
	}

	for _, wantedID := range []string{skillID, promptID, ruleID} {
		if _, exists := itemIDs[wantedID]; !exists {
			t.Fatalf("expected manifest to include item %q, got %v", wantedID, itemIDs)
		}
	}
	for _, wantedClassifier := range []string{"skill", "prompt", "rule"} {
		if _, exists := classifiers[wantedClassifier]; !exists {
			t.Fatalf("expected manifest to include classifier %q, got %v", wantedClassifier, classifiers)
		}
	}
}

func TestExportCatalog_NonDryRun_BatchMixedClassifiers_ReturnsCatalogArchiveMetadata(t *testing.T) {
	t.Parallel()

	server := newCatalogExportMaterializationFixtureServer(t)

	promptID := mustFindCatalogItemID(t, server, domain.CatalogClassifierPrompt, "prompts/system.md")
	ruleID := mustFindCatalogItemID(t, server, domain.CatalogClassifierRule, "")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/export",
		strings.NewReader(`{"item_ids":["`+promptID+`","`+ruleID+`"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	download, ok := payload["download"].(map[string]any)
	if !ok {
		t.Fatalf("expected download metadata object, got %T", payload["download"])
	}
	if fileName, _ := download["file_name"].(string); fileName != "catalog-export.tar.gz" {
		t.Fatalf("expected file_name %q, got %q", "catalog-export.tar.gz", fileName)
	}
	if contentType, _ := download["content_type"].(string); contentType != "application/gzip" {
		t.Fatalf("expected content_type %q, got %q", "application/gzip", contentType)
	}
	contentLength, ok := download["content_length"].(float64)
	if !ok || contentLength <= 0 {
		t.Fatalf("expected positive content_length, got %v", download["content_length"])
	}
	if _, exists := download["legacy_skill_export_url"]; exists {
		t.Fatalf("did not expect legacy skill export URL for mixed batch export")
	}
}

func TestExportCatalog_RejectsMalformedPayload(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/export",
		strings.NewReader(`{"item_ids":`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid request payload") {
		t.Fatalf("expected invalid-payload message, got %q", rec.Body.String())
	}
}

func TestExportCatalog_RejectsEmptyItemIDs(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/export",
		strings.NewReader(`{"item_ids":[]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "at least one item id is required") {
		t.Fatalf("expected empty-item-id validation message, got %q", rec.Body.String())
	}
}

func TestExportCatalog_MissingItem_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/export",
		strings.NewReader(`{"item_ids":["skill:missing-skill"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNotFound, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "catalog export item not found") {
		t.Fatalf("expected missing-item error message, got %q", rec.Body.String())
	}
}

func newExportGitBackedFixtureServer(t *testing.T) *Server {
	t.Helper()

	skillsDir := t.TempDir()
	repoName := "agents"
	skillName := "screen-reader-testing"
	skillDir := filepath.Join(skillsDir, repoName, skillName)

	if err := os.MkdirAll(filepath.Join(skillDir, "references"), 0o755); err != nil {
		t.Fatalf("failed to create fixture directories: %v", err)
	}

	skillMarkdown := `---
name: screen-reader-testing
description: Fixture git-backed export skill
---
# Screen Reader Testing
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMarkdown), 0o644); err != nil {
		t.Fatalf("failed to write fixture skill: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "references", "guide.md"),
		[]byte("# Guide\n"),
		0o644,
	); err != nil {
		t.Fatalf("failed to write fixture resource: %v", err)
	}

	manager, err := domain.NewFileSystemManager(skillsDir, []string{repoName})
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}

	return NewServer(manager, manager, nil, nil, nil, false, nil, "")
}

func newCatalogExportMaterializationFixtureServer(t *testing.T) *Server {
	t.Helper()

	skillsDir := t.TempDir()
	skillDir := filepath.Join(skillsDir, "demo-skill")

	if err := os.MkdirAll(filepath.Join(skillDir, "prompts"), 0o755); err != nil {
		t.Fatalf("failed to create prompts directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(skillDir, "rules"), 0o755); err != nil {
		t.Fatalf("failed to create rules directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(skillDir, "scripts"), 0o755); err != nil {
		t.Fatalf("failed to create scripts directory: %v", err)
	}

	skillMarkdown := `---
name: demo-skill
description: Fixture skill
---
# Demo Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMarkdown), 0o644); err != nil {
		t.Fatalf("failed to write fixture SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "prompts", "system.md"), []byte("# Prompt\n"), 0o644); err != nil {
		t.Fatalf("failed to write fixture prompt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "rules", "agents.md"), []byte("# Rules\n"), 0o644); err != nil {
		t.Fatalf("failed to write fixture rule file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "scripts", "hello.sh"), []byte("echo hello\n"), 0o644); err != nil {
		t.Fatalf("failed to write fixture script: %v", err)
	}

	manager, err := domain.NewFileSystemManager(skillsDir, nil)
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}

	return NewServer(manager, manager, nil, nil, nil, false, nil, "")
}

func mustFindCatalogItemID(
	t *testing.T,
	server *Server,
	classifier domain.CatalogClassifier,
	resourcePath string,
) string {
	t.Helper()

	items, err := server.skillManager.ListCatalogItems()
	if err != nil {
		t.Fatalf("expected ListCatalogItems to succeed, got %v", err)
	}
	for _, item := range items {
		if item.Classifier != classifier {
			continue
		}
		if resourcePath != "" && item.ResourcePath != resourcePath {
			continue
		}
		return item.ID
	}

	t.Fatalf(
		"expected catalog item for classifier=%q resource_path=%q; got items=%v",
		classifier,
		resourcePath,
		items,
	)
	return ""
}
