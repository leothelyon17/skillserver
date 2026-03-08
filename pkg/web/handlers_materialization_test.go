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

func TestMaterializeCatalog_DisabledCapability_ReturnsExplicitError(t *testing.T) {
	t.Parallel()

	server := newCatalogExportMaterializationFixtureServer(t)
	destinationRoot := t.TempDir()
	promptID := mustFindCatalogItemID(t, server, domain.CatalogClassifierPrompt, "prompts/system.md")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/materialize",
		strings.NewReader(
			`{"item_ids":["`+promptID+`"],"destination_dir":"`+destinationRoot+`"}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusForbidden, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "capability is disabled") {
		t.Fatalf("expected explicit capability-disabled error, got body=%q", rec.Body.String())
	}
}

func TestMaterializeCatalog_DryRunBatch_ReturnsPlannedItemsWithoutWrites(t *testing.T) {
	t.Parallel()

	server := newCatalogExportMaterializationFixtureServer(t)
	allowedRoot := t.TempDir()
	destinationDir := filepath.Join(allowedRoot, "project")
	server.SetMCPRuntimeCapabilities(MCPRuntimeCapabilities{
		MaterializationEnabled:  true,
		AllowedDestinationRoots: []string{allowedRoot},
	})

	promptID := mustFindCatalogItemID(t, server, domain.CatalogClassifierPrompt, "prompts/system.md")
	ruleID := mustFindCatalogItemID(t, server, domain.CatalogClassifierRule, "")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/materialize",
		strings.NewReader(
			`{"item_ids":["`+promptID+`","`+ruleID+`"],"destination_dir":"`+destinationDir+`","dry_run":true}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	dryRun, ok := payload["dry_run"].(bool)
	if !ok || !dryRun {
		t.Fatalf("expected dry_run=true response, got %v", payload["dry_run"])
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected two item results, got %v", payload["items"])
	}

	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			t.Fatalf("expected item result object, got %T", rawItem)
		}
		files, ok := item["files"].([]any)
		if !ok || len(files) == 0 {
			t.Fatalf("expected item files array, got %v", item["files"])
		}
		for _, rawFile := range files {
			fileResult, ok := rawFile.(map[string]any)
			if !ok {
				t.Fatalf("expected file result object, got %T", rawFile)
			}
			resolvedPath, _ := fileResult["resolved_path"].(string)
			if strings.TrimSpace(resolvedPath) == "" {
				t.Fatalf("expected resolved_path in file result, got %v", fileResult)
			}
			if _, statErr := os.Stat(resolvedPath); !os.IsNotExist(statErr) {
				t.Fatalf("expected no file write during dry-run, path=%q statErr=%v", resolvedPath, statErr)
			}
		}
	}
}

func TestMaterializeCatalog_RejectsInvalidConflictPolicy(t *testing.T) {
	t.Parallel()

	server := newCatalogExportMaterializationFixtureServer(t)
	allowedRoot := t.TempDir()
	destinationDir := filepath.Join(allowedRoot, "project")
	server.SetMCPRuntimeCapabilities(MCPRuntimeCapabilities{
		MaterializationEnabled:  true,
		AllowedDestinationRoots: []string{allowedRoot},
	})

	promptID := mustFindCatalogItemID(t, server, domain.CatalogClassifierPrompt, "prompts/system.md")
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/materialize",
		strings.NewReader(
			`{"item_ids":["`+promptID+`"],"destination_dir":"`+destinationDir+`","conflict_policy":"replace"}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "conflict policy") {
		t.Fatalf("expected conflict policy validation error, got body=%q", rec.Body.String())
	}
}

func TestMaterializeCatalog_RejectsDestinationOutsideAllowedRoots(t *testing.T) {
	t.Parallel()

	server := newCatalogExportMaterializationFixtureServer(t)
	allowedRoot := t.TempDir()
	outsideRoot := t.TempDir()
	server.SetMCPRuntimeCapabilities(MCPRuntimeCapabilities{
		MaterializationEnabled:  true,
		AllowedDestinationRoots: []string{allowedRoot},
	})

	ruleID := mustFindCatalogItemID(t, server, domain.CatalogClassifierRule, "")
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/materialize",
		strings.NewReader(
			`{"item_ids":["`+ruleID+`"],"destination_dir":"`+outsideRoot+`"}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusForbidden, rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "outside allowed roots") {
		t.Fatalf("expected allowed-root error, got body=%q", rec.Body.String())
	}
}

func TestMaterializeCatalog_RejectsRelativeDestinationPaths(t *testing.T) {
	t.Parallel()

	server := newCatalogExportMaterializationFixtureServer(t)
	allowedRoot := t.TempDir()
	server.SetMCPRuntimeCapabilities(MCPRuntimeCapabilities{
		MaterializationEnabled:  true,
		AllowedDestinationRoots: []string{allowedRoot},
	})

	ruleID := mustFindCatalogItemID(t, server, domain.CatalogClassifierRule, "")
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/catalog/materialize",
		strings.NewReader(
			`{"item_ids":["`+ruleID+`"],"destination_dir":"relative/path"}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "destination_dir must be absolute") {
		t.Fatalf("expected absolute-path validation error, got body=%q", rec.Body.String())
	}
}
