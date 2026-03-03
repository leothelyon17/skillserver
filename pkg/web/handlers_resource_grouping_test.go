package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mudler/skillserver/pkg/domain"
)

func TestListSkillResources_ReturnsLegacyAndAdditiveGroups(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/skills/demo-skill/resources", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	for _, key := range []string{"scripts", "references", "assets", "readOnly", "groups"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected response to include key %q", key)
		}
	}
	for _, key := range []string{"prompts", "imported"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected additive key %q when data exists", key)
		}
	}

	script := findResourceByPath(t, payload["scripts"], "scripts/hello.sh")
	if origin, _ := script["origin"].(string); origin != string(domain.ResourceOriginDirect) {
		t.Fatalf("expected script origin=%q, got %q", domain.ResourceOriginDirect, origin)
	}
	if writable, ok := script["writable"].(bool); !ok || !writable {
		t.Fatalf("expected script writable=true, got %v", script["writable"])
	}

	prompt := findResourceByPath(t, payload["prompts"], "prompts/system.md")
	if origin, _ := prompt["origin"].(string); origin != string(domain.ResourceOriginDirect) {
		t.Fatalf("expected prompt origin=%q, got %q", domain.ResourceOriginDirect, origin)
	}
	if writable, ok := prompt["writable"].(bool); !ok || !writable {
		t.Fatalf("expected prompt writable=true, got %v", prompt["writable"])
	}

	imported := findResourceByPath(t, payload["imported"], "imports/shared/context.md")
	if origin, _ := imported["origin"].(string); origin != string(domain.ResourceOriginImported) {
		t.Fatalf("expected imported origin=%q, got %q", domain.ResourceOriginImported, origin)
	}
	if writable, ok := imported["writable"].(bool); !ok || writable {
		t.Fatalf("expected imported writable=false, got %v", imported["writable"])
	}

	groups, ok := payload["groups"].(map[string]any)
	if !ok {
		t.Fatalf("expected groups object, got %T", payload["groups"])
	}
	for _, key := range []string{"scripts", "references", "assets", "prompts", "imported"} {
		if _, exists := groups[key]; !exists {
			t.Fatalf("expected groups to include key %q", key)
		}
	}
}

func TestListSkillResources_LegacyOnlySkill_PreservesLegacyShape(t *testing.T) {
	t.Parallel()

	server := newLegacyResourceFixtureServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/skills/demo-skill/resources", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	for _, key := range []string{"scripts", "references", "assets", "readOnly", "groups"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected response to include key %q", key)
		}
	}
	for _, key := range []string{"prompts", "imported"} {
		if _, ok := payload[key]; ok {
			t.Fatalf("did not expect additive key %q for legacy-only fixture", key)
		}
	}

	script := findResourceByPath(t, payload["scripts"], "scripts/hello.sh")
	if origin, _ := script["origin"].(string); origin != string(domain.ResourceOriginDirect) {
		t.Fatalf("expected script origin=%q, got %q", domain.ResourceOriginDirect, origin)
	}
	if writable, ok := script["writable"].(bool); !ok || !writable {
		t.Fatalf("expected script writable=true, got %v", script["writable"])
	}

	groups, ok := payload["groups"].(map[string]any)
	if !ok {
		t.Fatalf("expected groups object, got %T", payload["groups"])
	}
	for _, key := range []string{"scripts", "references", "assets"} {
		if _, exists := groups[key]; !exists {
			t.Fatalf("expected groups to include legacy key %q", key)
		}
	}
	for _, key := range []string{"prompts", "imported"} {
		if _, exists := groups[key]; exists {
			t.Fatalf("did not expect groups key %q for legacy-only fixture", key)
		}
	}
}

func TestSkillResourceWriteGuards_RejectImportedPaths(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	tests := []struct {
		name        string
		method      string
		target      string
		body        string
		contentType string
	}{
		{
			name:        "create",
			method:      http.MethodPost,
			target:      "/api/skills/demo-skill/resources",
			body:        `{"path":"imports/shared/new.md","content":"blocked"}`,
			contentType: "application/json",
		},
		{
			name:        "update",
			method:      http.MethodPut,
			target:      "/api/skills/demo-skill/resources/imports/shared/context.md",
			body:        "blocked update",
			contentType: "text/plain",
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			target: "/api/skills/demo-skill/resources/imports/shared/context.md",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			rec := httptest.NewRecorder()
			server.echo.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected status %d, got %d body=%q", http.StatusForbidden, rec.Code, rec.Body.String())
			}
			if !strings.Contains(strings.ToLower(rec.Body.String()), "imported") {
				t.Fatalf("expected imported guard message, got %q", rec.Body.String())
			}
		})
	}
}

func TestSkillResourceWriteGuards_AllowDirectResourceUpdate(t *testing.T) {
	t.Parallel()

	server := newResourceFixtureServer(t)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/skills/demo-skill/resources/scripts/hello.sh",
		strings.NewReader("echo updated\n"),
	)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	payload := decodeJSONObject(t, rec.Body.Bytes())
	if origin, _ := payload["origin"].(string); origin != string(domain.ResourceOriginDirect) {
		t.Fatalf("expected origin=%q, got %q", domain.ResourceOriginDirect, origin)
	}
	if writable, ok := payload["writable"].(bool); !ok || !writable {
		t.Fatalf("expected writable=true, got %v", payload["writable"])
	}
}

func newResourceFixtureServer(t *testing.T) *Server {
	t.Helper()
	return newFixtureServer(t, true)
}

func newLegacyResourceFixtureServer(t *testing.T) *Server {
	t.Helper()
	return newFixtureServer(t, false)
}

func newFixtureServer(t *testing.T, includeAdditiveResources bool) *Server {
	t.Helper()

	skillsDir := t.TempDir()
	skillDir := filepath.Join(skillsDir, "demo-skill")
	directories := []string{"scripts", "references", "assets"}
	if includeAdditiveResources {
		directories = append(directories, "prompts", "shared")
	}
	for _, dir := range directories {
		if err := os.MkdirAll(filepath.Join(skillDir, dir), 0o755); err != nil {
			t.Fatalf("failed to create fixture directory %q: %v", dir, err)
		}
	}

	skillContent := `---
name: demo-skill
description: Fixture skill
---
# Demo Skill`
	if includeAdditiveResources {
		skillContent += `

Use [Shared Context](shared/context.md).
`
	}
	skillContent += "\n"

	files := map[string][]byte{
		"SKILL.md":            []byte(skillContent),
		"scripts/hello.sh":    []byte("echo hello\n"),
		"references/guide.md": []byte("# Guide\n"),
		"assets/logo.png":     []byte{0x89, 0x50, 0x4e, 0x47},
	}
	if includeAdditiveResources {
		files["prompts/system.md"] = []byte("You are helpful.\n")
		files["shared/context.md"] = []byte("Shared context.\n")
	}
	for relPath, content := range files {
		if err := os.WriteFile(filepath.Join(skillDir, relPath), content, 0o644); err != nil {
			t.Fatalf("failed to write fixture file %q: %v", relPath, err)
		}
	}

	manager, err := domain.NewFileSystemManager(skillsDir, nil)
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}

	return NewServer(manager, manager, nil, nil, nil, false, nil, "")
}

func decodeJSONObject(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode json payload: %v body=%q", err, string(body))
	}
	return payload
}

func findResourceByPath(t *testing.T, rawGroup any, wantedPath string) map[string]any {
	t.Helper()

	entries, ok := rawGroup.([]any)
	if !ok {
		t.Fatalf("expected resource group array, got %T", rawGroup)
	}

	for _, entry := range entries {
		resource, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		path, _ := resource["path"].(string)
		if path == wantedPath {
			return resource
		}
	}

	t.Fatalf("resource with path %q not found", wantedPath)
	return nil
}
