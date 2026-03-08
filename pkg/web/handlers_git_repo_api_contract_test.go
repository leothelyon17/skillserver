package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/git"
	"github.com/mudler/skillserver/pkg/persistence"
)

func TestAddGitRepo_LegacyURLOnlyPayloadRemainsSupported(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/git-repos",
		strings.NewReader(`{"url":"https://github.com/acme/public-repo.git"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response GitRepoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid git repo response, got %v", err)
	}
	if response.AuthMode != git.GitRepoAuthModeNone {
		t.Fatalf("expected auth_mode %q, got %q", git.GitRepoAuthModeNone, response.AuthMode)
	}
	if response.CredentialSource != git.GitRepoAuthSourceNone {
		t.Fatalf("expected credential_source %q, got %q", git.GitRepoAuthSourceNone, response.CredentialSource)
	}
	if response.HasCredentials {
		t.Fatalf("expected has_credentials false for legacy public payload")
	}
}

func TestGitRepoPublicLifecycle_RemainsCredentialFree(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{
		skillsDir: t.TempDir(),
	}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	addReq := httptest.NewRequest(
		http.MethodPost,
		"/api/git-repos",
		strings.NewReader(`{"url":"https://github.com/acme/public-lifecycle-repo.git"}`),
	)
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	server.echo.ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusCreated {
		t.Fatalf("expected add status %d, got %d body=%q", http.StatusCreated, addRec.Code, addRec.Body.String())
	}

	var added GitRepoResponse
	if err := json.Unmarshal(addRec.Body.Bytes(), &added); err != nil {
		t.Fatalf("expected valid add response, got %v", err)
	}
	if added.AuthMode != git.GitRepoAuthModeNone {
		t.Fatalf("expected add auth_mode %q, got %q", git.GitRepoAuthModeNone, added.AuthMode)
	}
	if added.CredentialSource != git.GitRepoAuthSourceNone {
		t.Fatalf("expected add credential_source %q, got %q", git.GitRepoAuthSourceNone, added.CredentialSource)
	}
	if added.HasCredentials {
		t.Fatalf("expected add has_credentials false for public repo")
	}

	updateReq := httptest.NewRequest(
		http.MethodPut,
		"/api/git-repos/"+added.ID,
		strings.NewReader(`{"url":"https://github.com:443/acme/public-lifecycle-repo.git","enabled":true}`),
	)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	server.echo.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update status %d, got %d body=%q", http.StatusOK, updateRec.Code, updateRec.Body.String())
	}

	var updated GitRepoResponse
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("expected valid update response, got %v", err)
	}
	if updated.ID != added.ID {
		t.Fatalf("expected canonical-equivalent update to preserve stable id %q, got %q", added.ID, updated.ID)
	}
	if updated.AuthMode != git.GitRepoAuthModeNone || updated.CredentialSource != git.GitRepoAuthSourceNone || updated.HasCredentials {
		t.Fatalf("expected update response to remain credential-free, got %+v", updated)
	}

	syncReq := httptest.NewRequest(http.MethodPost, "/api/git-repos/"+added.ID+"/sync", nil)
	syncRec := httptest.NewRecorder()
	server.echo.ServeHTTP(syncRec, syncReq)
	if syncRec.Code != http.StatusOK {
		t.Fatalf("expected sync status %d, got %d body=%q", http.StatusOK, syncRec.Code, syncRec.Body.String())
	}

	var synced GitRepoResponse
	if err := json.Unmarshal(syncRec.Body.Bytes(), &synced); err != nil {
		t.Fatalf("expected valid sync response, got %v", err)
	}
	if synced.AuthMode != git.GitRepoAuthModeNone || synced.CredentialSource != git.GitRepoAuthSourceNone || synced.HasCredentials {
		t.Fatalf("expected sync response to remain credential-free, got %+v", synced)
	}

	toggleReq := httptest.NewRequest(http.MethodPost, "/api/git-repos/"+added.ID+"/toggle", nil)
	toggleRec := httptest.NewRecorder()
	server.echo.ServeHTTP(toggleRec, toggleReq)
	if toggleRec.Code != http.StatusOK {
		t.Fatalf("expected toggle status %d, got %d body=%q", http.StatusOK, toggleRec.Code, toggleRec.Body.String())
	}

	var toggled GitRepoResponse
	if err := json.Unmarshal(toggleRec.Body.Bytes(), &toggled); err != nil {
		t.Fatalf("expected valid toggle response, got %v", err)
	}
	if toggled.Enabled {
		t.Fatalf("expected repo to be disabled after toggle")
	}
	if toggled.AuthMode != git.GitRepoAuthModeNone || toggled.CredentialSource != git.GitRepoAuthSourceNone || toggled.HasCredentials {
		t.Fatalf("expected toggle response to remain credential-free, got %+v", toggled)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/git-repos/"+added.ID, nil)
	deleteRec := httptest.NewRecorder()
	server.echo.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected delete status %d, got %d body=%q", http.StatusNoContent, deleteRec.Code, deleteRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/git-repos", nil)
	listRec := httptest.NewRecorder()
	server.echo.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list status %d, got %d body=%q", http.StatusOK, listRec.Code, listRec.Body.String())
	}

	var listed []GitRepoResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("expected valid list response, got %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected repo list to be empty after delete, got %d entries", len(listed))
	}
}

func TestAddGitRepo_RejectsStoredCredentialWritesWhenCapabilityDisabled(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/git-repos",
		strings.NewReader(`{
			"url":"https://github.com/acme/private-repo.git",
			"auth":{"mode":"https_token","source":"stored"},
			"stored_credential":{"token":"super-secret-token"}
		}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "stored credentials are disabled") {
		t.Fatalf("expected stored-capability validation error, got body=%q", rec.Body.String())
	}
}

func TestAddGitRepo_StoredCredentialWritePersistsAndResponseIsSecretSafe(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{}
	store := newFakeGitCredentialStore()
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")
	server.SetGitRuntimeCapabilities(GitRuntimeCapabilities{
		StoredCredentialsEnabled: true,
	})
	server.SetGitCredentialStore(store)

	secretValue := "super-secret-token-value"
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/git-repos",
		strings.NewReader(`{
			"url":"https://github.com/acme/private-repo.git",
			"auth":{"mode":"https_token","source":"stored"},
			"stored_credential":{"username":"git","token":"`+secretValue+`"}
		}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusCreated, rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), secretValue) {
		t.Fatalf("expected add response to redact submitted secret value, got body=%q", rec.Body.String())
	}

	var created GitRepoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("expected valid add response payload, got %v", err)
	}
	if created.AuthMode != git.GitRepoAuthModeHTTPSToken {
		t.Fatalf("expected auth_mode %q, got %q", git.GitRepoAuthModeHTTPSToken, created.AuthMode)
	}
	if created.CredentialSource != git.GitRepoAuthSourceStored {
		t.Fatalf("expected credential_source %q, got %q", git.GitRepoAuthSourceStored, created.CredentialSource)
	}
	if !created.HasCredentials {
		t.Fatalf("expected has_credentials true for stored credential write")
	}
	if !store.hasCredential(created.ID) {
		t.Fatalf("expected stored credential row for repo id %q", created.ID)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/git-repos", nil)
	listRec := httptest.NewRecorder()
	server.echo.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list status %d, got %d body=%q", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), secretValue) {
		t.Fatalf("expected list response to remain secret-free, got body=%q", listRec.Body.String())
	}
}

func TestGitRepoList_ReturnsSafeAuthSummaryAndSyncStatus(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	publicURL := "https://github.com/acme/public-repo.git"
	privateURL := "https://github.com/acme/private-repo.git"
	publicID := git.GenerateID(publicURL)
	privateID := git.GenerateID(privateURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      publicID,
			URL:     publicURL,
			Name:    git.ResolveCheckoutName(publicURL),
			Enabled: true,
			Auth: git.GitRepoAuthConfig{
				Mode: git.GitRepoAuthModeNone,
			},
		},
		{
			ID:      privateID,
			URL:     privateURL,
			Name:    git.ResolveCheckoutName(privateURL),
			Enabled: true,
			Auth: git.GitRepoAuthConfig{
				Mode:     git.GitRepoAuthModeHTTPSToken,
				Source:   git.GitRepoAuthSourceEnv,
				TokenRef: "ACME_PRIVATE_TOKEN",
			},
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{
		statusByID: map[string]git.RepoSyncStatus{
			privateID: {
				RepoID:      privateID,
				State:       git.RepoSyncStateFailed,
				LastError:   "authentication required",
				LastAttempt: time.Now().UTC(),
			},
		},
	}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/api/git-repos", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "token_ref") {
		t.Fatalf("expected list response to exclude raw auth reference fields, got body=%q", rec.Body.String())
	}

	var repos []GitRepoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &repos); err != nil {
		t.Fatalf("expected valid list payload, got %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected two repos in list payload, got %d", len(repos))
	}

	repoByID := make(map[string]GitRepoResponse, len(repos))
	for _, repo := range repos {
		repoByID[repo.ID] = repo
	}
	privateResponse := repoByID[privateID]
	if privateResponse.AuthMode != git.GitRepoAuthModeHTTPSToken {
		t.Fatalf("expected auth_mode %q, got %q", git.GitRepoAuthModeHTTPSToken, privateResponse.AuthMode)
	}
	if privateResponse.CredentialSource != git.GitRepoAuthSourceEnv {
		t.Fatalf("expected credential_source %q, got %q", git.GitRepoAuthSourceEnv, privateResponse.CredentialSource)
	}
	if !privateResponse.HasCredentials {
		t.Fatalf("expected has_credentials true for env token refs")
	}
	if privateResponse.LastSyncStatus != string(git.RepoSyncStateFailed) {
		t.Fatalf("expected last_sync_status %q, got %q", git.RepoSyncStateFailed, privateResponse.LastSyncStatus)
	}
	if privateResponse.LastSyncError == "" {
		t.Fatalf("expected redacted last_sync_error to be present")
	}
}

func TestUpdateGitRepo_PreservesExistingAuthWhenAuthPayloadOmitted(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	repoURL := "https://github.com/acme/private-repo.git"
	repoID := git.GenerateID(repoURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      repoID,
			URL:     repoURL,
			Name:    git.ResolveCheckoutName(repoURL),
			Enabled: true,
			Auth: git.GitRepoAuthConfig{
				Mode:     git.GitRepoAuthModeHTTPSToken,
				Source:   git.GitRepoAuthSourceEnv,
				TokenRef: "PRIVATE_TOKEN_ENV",
			},
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{
		repos: []git.GitRepoConfig{
			{
				ID:      repoID,
				URL:     repoURL,
				Name:    git.ResolveCheckoutName(repoURL),
				Enabled: true,
				Auth: git.GitRepoAuthConfig{
					Mode:     git.GitRepoAuthModeHTTPSToken,
					Source:   git.GitRepoAuthSourceEnv,
					TokenRef: "PRIVATE_TOKEN_ENV",
				},
			},
		},
	}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/git-repos/"+repoID,
		strings.NewReader(`{"url":"https://github.com:443/acme/private-repo.git","enabled":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response GitRepoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid update response payload, got %v", err)
	}
	if response.AuthMode != git.GitRepoAuthModeHTTPSToken {
		t.Fatalf("expected auth_mode %q, got %q", git.GitRepoAuthModeHTTPSToken, response.AuthMode)
	}
	if response.CredentialSource != git.GitRepoAuthSourceEnv {
		t.Fatalf("expected credential_source %q, got %q", git.GitRepoAuthSourceEnv, response.CredentialSource)
	}
	if !response.HasCredentials {
		t.Fatalf("expected has_credentials true when existing auth refs remain configured")
	}
}

func TestDeleteGitRepo_DeletesImplicitStoredCredentialReference(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	repoURL := "https://github.com/acme/private-delete.git"
	repoID := git.GenerateID(repoURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      repoID,
			URL:     repoURL,
			Name:    git.ResolveCheckoutName(repoURL),
			Enabled: true,
			Auth: git.GitRepoAuthConfig{
				Mode:   git.GitRepoAuthModeHTTPSToken,
				Source: git.GitRepoAuthSourceStored,
			},
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{
		repos: []git.GitRepoConfig{
			{
				ID:      repoID,
				URL:     repoURL,
				Name:    git.ResolveCheckoutName(repoURL),
				Enabled: true,
				Auth: git.GitRepoAuthConfig{
					Mode:   git.GitRepoAuthModeHTTPSToken,
					Source: git.GitRepoAuthSourceStored,
				},
			},
		},
	}
	store := newFakeGitCredentialStore()
	store.payloadByRepoID[repoID] = persistence.GitRepoCredentialSecretPayload{
		Type:  persistence.GitRepoCredentialSecretTypeHTTPSToken,
		Token: "to-delete",
	}

	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")
	server.SetGitRuntimeCapabilities(GitRuntimeCapabilities{StoredCredentialsEnabled: true})
	server.SetGitCredentialStore(store)

	req := httptest.NewRequest(http.MethodDelete, "/api/git-repos/"+repoID, nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusNoContent, rec.Code, rec.Body.String())
	}
	if store.hasCredential(repoID) {
		t.Fatalf("expected implicit stored credential row to be deleted for repo id %q", repoID)
	}
}

func TestToggleGitRepo_ReturnsExpandedContractFields(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	repoURL := "https://github.com/acme/toggle-private.git"
	repoID := git.GenerateID(repoURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      repoID,
			URL:     repoURL,
			Name:    git.ResolveCheckoutName(repoURL),
			Enabled: true,
			Auth: git.GitRepoAuthConfig{
				Mode:     git.GitRepoAuthModeHTTPSToken,
				Source:   git.GitRepoAuthSourceEnv,
				TokenRef: "TOGGLE_PRIVATE_TOKEN",
			},
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{
		repos: []git.GitRepoConfig{
			{
				ID:      repoID,
				URL:     repoURL,
				Name:    git.ResolveCheckoutName(repoURL),
				Enabled: true,
				Auth: git.GitRepoAuthConfig{
					Mode:     git.GitRepoAuthModeHTTPSToken,
					Source:   git.GitRepoAuthSourceEnv,
					TokenRef: "TOGGLE_PRIVATE_TOKEN",
				},
			},
		},
	}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(http.MethodPost, "/api/git-repos/"+repoID+"/toggle", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response GitRepoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid toggle payload, got %v", err)
	}
	if response.Enabled {
		t.Fatalf("expected toggle to disable the repository")
	}
	if response.AuthMode != git.GitRepoAuthModeHTTPSToken {
		t.Fatalf("expected auth_mode %q, got %q", git.GitRepoAuthModeHTTPSToken, response.AuthMode)
	}
	if response.CredentialSource != git.GitRepoAuthSourceEnv {
		t.Fatalf("expected credential_source %q, got %q", git.GitRepoAuthSourceEnv, response.CredentialSource)
	}
	if !response.HasCredentials {
		t.Fatalf("expected has_credentials true for env token refs")
	}
}

func TestSyncGitRepo_ReturnsExpandedContractFields(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	repoURL := "https://github.com/acme/sync-private.git"
	repoID := git.GenerateID(repoURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      repoID,
			URL:     repoURL,
			Name:    git.ResolveCheckoutName(repoURL),
			Enabled: true,
			Auth: git.GitRepoAuthConfig{
				Mode:     git.GitRepoAuthModeHTTPSToken,
				Source:   git.GitRepoAuthSourceEnv,
				TokenRef: "SYNC_PRIVATE_TOKEN",
			},
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{
		repos: []git.GitRepoConfig{
			{
				ID:      repoID,
				URL:     repoURL,
				Name:    git.ResolveCheckoutName(repoURL),
				Enabled: true,
				Auth: git.GitRepoAuthConfig{
					Mode:     git.GitRepoAuthModeHTTPSToken,
					Source:   git.GitRepoAuthSourceEnv,
					TokenRef: "SYNC_PRIVATE_TOKEN",
				},
			},
		},
	}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(http.MethodPost, "/api/git-repos/"+repoID+"/sync", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response GitRepoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid sync payload, got %v", err)
	}
	if response.LastSyncStatus != string(git.RepoSyncStateSuccess) {
		t.Fatalf("expected last_sync_status %q, got %q", git.RepoSyncStateSuccess, response.LastSyncStatus)
	}
	if response.LastSyncError != "" {
		t.Fatalf("expected empty last_sync_error for successful sync, got %q", response.LastSyncError)
	}
	if response.AuthMode != git.GitRepoAuthModeHTTPSToken {
		t.Fatalf("expected auth_mode %q, got %q", git.GitRepoAuthModeHTTPSToken, response.AuthMode)
	}
	if response.CredentialSource != git.GitRepoAuthSourceEnv {
		t.Fatalf("expected credential_source %q, got %q", git.GitRepoAuthSourceEnv, response.CredentialSource)
	}
}

func TestSyncGitRepo_FailureResponseRemainsSecretSafe(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	repoURL := "https://github.com/acme/sync-failure-private.git"
	repoID := git.GenerateID(repoURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      repoID,
			URL:     repoURL,
			Name:    git.ResolveCheckoutName(repoURL),
			Enabled: true,
			Auth: git.GitRepoAuthConfig{
				Mode:     git.GitRepoAuthModeHTTPSToken,
				Source:   git.GitRepoAuthSourceEnv,
				TokenRef: "SYNC_FAILURE_TOKEN",
			},
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{
		repos: []git.GitRepoConfig{
			{
				ID:      repoID,
				URL:     repoURL,
				Name:    git.ResolveCheckoutName(repoURL),
				Enabled: true,
				Auth: git.GitRepoAuthConfig{
					Mode:     git.GitRepoAuthModeHTTPSToken,
					Source:   git.GitRepoAuthSourceEnv,
					TokenRef: "SYNC_FAILURE_TOKEN",
				},
			},
		},
		syncErr: errors.New("token=sync-failure-super-secret"),
	}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	syncReq := httptest.NewRequest(http.MethodPost, "/api/git-repos/"+repoID+"/sync", nil)
	syncRec := httptest.NewRecorder()
	server.echo.ServeHTTP(syncRec, syncReq)

	if syncRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected sync failure status %d, got %d body=%q", http.StatusInternalServerError, syncRec.Code, syncRec.Body.String())
	}
	if strings.Contains(syncRec.Body.String(), "sync-failure-super-secret") {
		t.Fatalf("expected sync failure response to redact secret value, got body=%q", syncRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/git-repos", nil)
	listRec := httptest.NewRecorder()
	server.echo.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list status %d, got %d body=%q", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), "sync-failure-super-secret") {
		t.Fatalf("expected list payload to redact sync failure secret, got body=%q", listRec.Body.String())
	}

	var repos []GitRepoResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &repos); err != nil {
		t.Fatalf("expected valid list payload, got %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected one repo in list payload, got %d", len(repos))
	}
	if repos[0].LastSyncStatus != string(git.RepoSyncStateFailed) {
		t.Fatalf("expected last_sync_status %q, got %q", git.RepoSyncStateFailed, repos[0].LastSyncStatus)
	}
	if repos[0].LastSyncError == "" {
		t.Fatalf("expected last_sync_error in list payload after failure")
	}
	if strings.Contains(repos[0].LastSyncError, "sync-failure-super-secret") {
		t.Fatalf("expected list last_sync_error to redact secret value, got %q", repos[0].LastSyncError)
	}
}

type fakeGitCredentialStore struct {
	mu              sync.Mutex
	payloadByRepoID map[string]persistence.GitRepoCredentialSecretPayload
}

func newFakeGitCredentialStore() *fakeGitCredentialStore {
	return &fakeGitCredentialStore{
		payloadByRepoID: map[string]persistence.GitRepoCredentialSecretPayload{},
	}
}

func (s *fakeGitCredentialStore) ReplaceCredential(
	_ context.Context,
	repoID string,
	payload persistence.GitRepoCredentialSecretPayload,
	_ time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payloadByRepoID[repoID] = payload
	return nil
}

func (s *fakeGitCredentialStore) DeleteByRepoID(_ context.Context, repoID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.payloadByRepoID[repoID]
	delete(s.payloadByRepoID, repoID)
	return exists, nil
}

func (s *fakeGitCredentialStore) GetEncryptedByRepoID(
	_ context.Context,
	repoID string,
) (persistence.GitRepoCredentialRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, exists := s.payloadByRepoID[repoID]
	if !exists {
		return persistence.GitRepoCredentialRow{}, persistence.ErrGitRepoCredentialNotFound
	}
	return persistence.GitRepoCredentialRow{
		RepoID:     repoID,
		KeyID:      "test-key",
		KeyVersion: 1,
		Ciphertext: []byte("cipher:" + payload.Type),
		Nonce:      []byte("nonce"),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

func (s *fakeGitCredentialStore) hasCredential(repoID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.payloadByRepoID[repoID]
	return ok
}
