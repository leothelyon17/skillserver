package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/git"
)

func TestGetRuntimeCapabilities_ReturnsGitCapabilityState(t *testing.T) {
	skillManager, err := domain.NewFileSystemManager(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}
	t.Cleanup(func() {
		_ = skillManager.Close()
	})

	server := NewServer(skillManager, skillManager, nil, nil, nil, false, nil, "")
	server.SetGitRuntimeCapabilities(GitRuntimeCapabilities{
		StoredCredentialsEnabled: true,
	})
	server.SetCatalogRuntimeCapabilities(CatalogRuntimeCapabilities{
		RulesEnabled:           true,
		RuleDirectoryAllowlist: []string{"rules", "governance"},
		RuleFilenameAllowlist:  []string{"agents.md", "rules.md"},
	})
	server.SetMCPRuntimeCapabilities(MCPRuntimeCapabilities{
		MaterializationEnabled:  true,
		AllowedDestinationRoots: []string{"/workspace", "/projects"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/capabilities", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	var payload RuntimeCapabilitiesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid runtime capability payload, got %v", err)
	}
	if !payload.Git.StoredCredentialsEnabled {
		t.Fatalf("expected stored credential capability true in response payload")
	}
	if !payload.Catalog.RulesEnabled {
		t.Fatalf("expected rules capability true in response payload")
	}
	if len(payload.Catalog.RuleDirectoryAllowlist) != 2 {
		t.Fatalf("expected 2 rule directories, got %d", len(payload.Catalog.RuleDirectoryAllowlist))
	}
	if !payload.MCP.MaterializationEnabled {
		t.Fatalf("expected materialization capability true in response payload")
	}
	if len(payload.MCP.AllowedDestinationRoots) != 2 {
		t.Fatalf("expected 2 allowed destination roots, got %d", len(payload.MCP.AllowedDestinationRoots))
	}
}

func TestGetRuntimeCapabilities_DefaultMaterializationCapabilityDisabled(t *testing.T) {
	skillManager, err := domain.NewFileSystemManager(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}
	t.Cleanup(func() {
		_ = skillManager.Close()
	})

	server := NewServer(skillManager, skillManager, nil, nil, nil, false, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/capabilities", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	var payload RuntimeCapabilitiesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid runtime capability payload, got %v", err)
	}
	if payload.MCP.MaterializationEnabled {
		t.Fatalf("expected materialization capability disabled by default")
	}
	if len(payload.MCP.AllowedDestinationRoots) != 0 {
		t.Fatalf("expected no default allowed destination roots, got %v", payload.MCP.AllowedDestinationRoots)
	}
}

func TestListGitRepos_IncludesStoredCredentialCapabilityField(t *testing.T) {
	skillDir := t.TempDir()
	skillManager, err := domain.NewFileSystemManager(skillDir, nil)
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}
	t.Cleanup(func() {
		_ = skillManager.Close()
	})

	configManager := git.NewConfigManager(skillDir)
	repoURL := "https://example.com/acme/repo-one.git"
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      git.GenerateID(repoURL),
			URL:     repoURL,
			Name:    git.ExtractRepoName(repoURL),
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	server := NewServer(skillManager, skillManager, nil, nil, configManager, false, nil, "")
	server.SetGitRuntimeCapabilities(GitRuntimeCapabilities{
		StoredCredentialsEnabled: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/git-repos", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	var repos []GitRepoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &repos); err != nil {
		t.Fatalf("expected valid git repo list payload, got %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected one git repo in payload, got %d", len(repos))
	}
	if !repos[0].StoredCredentialsEnabled {
		t.Fatalf("expected stored credential capability field to be true in git repo response")
	}
}
