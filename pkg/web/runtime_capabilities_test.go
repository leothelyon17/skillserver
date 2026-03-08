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

	server := NewServer(skillManager, skillManager, nil, nil, nil, false, nil, "")
	server.SetGitRuntimeCapabilities(GitRuntimeCapabilities{
		StoredCredentialsEnabled: true,
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
}

func TestListGitRepos_IncludesStoredCredentialCapabilityField(t *testing.T) {
	skillDir := t.TempDir()
	skillManager, err := domain.NewFileSystemManager(skillDir, nil)
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}

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
