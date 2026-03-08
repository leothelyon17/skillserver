package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mudler/skillserver/pkg/git"
)

func TestAddGitRepo_UsesCanonicalURLForDuplicateDetection(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	existingURL := "https://github.com/acme/repo-one.git"
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      git.GenerateID(existingURL),
			URL:     existingURL,
			Name:    git.ResolveCheckoutName(existingURL),
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/git-repos",
		strings.NewReader(`{"url":"https://github.com:443/acme/repo-one.git"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "repository already exists") {
		t.Fatalf("expected duplicate repository error, got body=%q", rec.Body.String())
	}
	if len(syncer.GetRepos()) != 0 {
		t.Fatalf("expected duplicate add to be rejected before syncer mutation, got repos=%v", syncer.GetRepos())
	}
}

func TestAddGitRepo_RejectsUserInfoBearingURL(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/git-repos",
		strings.NewReader(`{"url":"https://token@github.com/acme/private.git"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "must not include userinfo") {
		t.Fatalf("expected userinfo validation error, got body=%q", rec.Body.String())
	}
}

func TestUpdateGitRepo_UsesCanonicalURLForDuplicateDetection(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	repoOneURL := "https://github.com/acme/repo-one.git"
	repoTwoURL := "https://github.com/acme/repo-two.git"
	repoOneID := git.GenerateID(repoOneURL)
	repoTwoID := git.GenerateID(repoTwoURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      repoOneID,
			URL:     repoOneURL,
			Name:    git.ResolveCheckoutName(repoOneURL),
			Enabled: true,
		},
		{
			ID:      repoTwoID,
			URL:     repoTwoURL,
			Name:    git.ResolveCheckoutName(repoTwoURL),
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{
		repos: []git.GitRepoConfig{
			{ID: repoOneID, URL: repoOneURL, Name: git.ResolveCheckoutName(repoOneURL), Enabled: true},
			{ID: repoTwoID, URL: repoTwoURL, Name: git.ResolveCheckoutName(repoTwoURL), Enabled: true},
		},
	}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/git-repos/"+repoTwoID,
		strings.NewReader(`{"url":"https://github.com:443/acme/repo-one.git","enabled":true}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "repository already exists") {
		t.Fatalf("expected duplicate repository error, got body=%q", rec.Body.String())
	}
}

func TestUpdateGitRepo_RejectsUserInfoBearingURL(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	repoURL := "https://github.com/acme/repo-one.git"
	repoID := git.GenerateID(repoURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      repoID,
			URL:     repoURL,
			Name:    git.ResolveCheckoutName(repoURL),
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{
		repos: []git.GitRepoConfig{
			{ID: repoID, URL: repoURL, Name: git.ResolveCheckoutName(repoURL), Enabled: true},
		},
	}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/git-repos/"+repoID,
		strings.NewReader(`{"url":"https://token@github.com/acme/repo-one.git","enabled":true}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "must not include userinfo") {
		t.Fatalf("expected userinfo validation error, got body=%q", rec.Body.String())
	}
}
