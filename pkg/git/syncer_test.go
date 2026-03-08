package git

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

type storedCredentialProviderFunc func(repo GitRepoConfig) (ResolvedGitAuthCredentials, error)

func (f storedCredentialProviderFunc) Resolve(repo GitRepoConfig) (ResolvedGitAuthCredentials, error) {
	return f(repo)
}

func TestGitSyncerSyncRepoUsesResolvedAuthForCloneAndPull(t *testing.T) {
	t.Setenv("WP005_GIT_TOKEN", "token-value")

	repoURL := "https://github.com/acme/private-repo.git"
	repo := GitRepoConfig{
		ID:      GenerateID(repoURL),
		URL:     repoURL,
		Name:    "private-repo",
		Enabled: true,
		Auth: GitRepoAuthConfig{
			Mode:     GitRepoAuthModeHTTPSToken,
			Source:   GitRepoAuthSourceEnv,
			TokenRef: "WP005_GIT_TOKEN",
		},
	}

	syncer := NewGitSyncer(t.TempDir(), []GitRepoConfig{repo}, nil)

	var (
		cloneAuth transport.AuthMethod
		pullAuth  transport.AuthMethod
		statCalls int
	)
	fileInfo, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatalf("failed to stat temp dir: %v", err)
	}
	syncer.statPath = func(string) (os.FileInfo, error) {
		statCalls++
		if statCalls == 1 {
			return nil, os.ErrNotExist
		}
		return fileInfo, nil
	}
	syncer.cloneRepo = func(_ string, _ string, auth transport.AuthMethod, _ io.Writer) error {
		cloneAuth = auth
		return nil
	}
	syncer.pullRepo = func(_ string, auth transport.AuthMethod, _ io.Writer) error {
		pullAuth = auth
		return nil
	}

	if err := syncer.SyncRepo(repo.ID); err != nil {
		t.Fatalf("expected clone sync to succeed, got %v", err)
	}
	if err := syncer.SyncRepo(repo.ID); err != nil {
		t.Fatalf("expected pull sync to succeed, got %v", err)
	}

	assertHTTPBasicAuthPassword(t, cloneAuth, "token-value")
	assertHTTPBasicAuthPassword(t, pullAuth, "token-value")
}

func TestGitSyncerSyncAllAndManualSyncResolveCredentialsEveryAttempt(t *testing.T) {
	tokenValue := "token-initial"
	envResolver := credentialSourceResolverFunc(func(reference string) (string, error) {
		if reference != "WP005_ROTATING_TOKEN" {
			return "", errors.New("unexpected reference")
		}
		return tokenValue, nil
	})

	repoURL := "https://github.com/acme/rotating-private-repo.git"
	repo := GitRepoConfig{
		ID:      GenerateID(repoURL),
		URL:     repoURL,
		Name:    "rotating-private-repo",
		Enabled: true,
		Auth: GitRepoAuthConfig{
			Mode:     GitRepoAuthModeHTTPSToken,
			Source:   GitRepoAuthSourceEnv,
			TokenRef: "WP005_ROTATING_TOKEN",
		},
	}

	syncer := NewGitSyncer(t.TempDir(), []GitRepoConfig{repo}, nil)
	syncer.SetCredentialResolver(NewGitCredentialResolver(GitCredentialResolverOptions{
		EnvResolver: envResolver,
	}))

	fileInfo, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatalf("failed to stat temp dir: %v", err)
	}
	syncer.statPath = func(string) (os.FileInfo, error) {
		return fileInfo, nil
	}
	var pullPasswords []string
	syncer.pullRepo = func(_ string, auth transport.AuthMethod, _ io.Writer) error {
		basicAuth, ok := auth.(*githttp.BasicAuth)
		if !ok {
			t.Fatalf("expected *http.BasicAuth, got %T", auth)
		}
		pullPasswords = append(pullPasswords, basicAuth.Password)
		return nil
	}

	if err := syncer.syncAll(); err != nil {
		t.Fatalf("expected syncAll to succeed, got %v", err)
	}

	tokenValue = "token-rotated"
	if err := syncer.SyncRepo(repo.ID); err != nil {
		t.Fatalf("expected manual sync to succeed, got %v", err)
	}

	if len(pullPasswords) != 2 {
		t.Fatalf("expected two pull attempts, got %d", len(pullPasswords))
	}
	if pullPasswords[0] != "token-initial" || pullPasswords[1] != "token-rotated" {
		t.Fatalf("expected credential refresh across attempts, got %+v", pullPasswords)
	}
}

func TestGitSyncerSyncAllAndManualSyncResolveFileCredentialsEveryAttempt(t *testing.T) {
	tempDir := t.TempDir()
	usernamePath := filepath.Join(tempDir, "username")
	passwordPath := filepath.Join(tempDir, "password")

	if err := os.WriteFile(usernamePath, []byte("ci-user"), 0o600); err != nil {
		t.Fatalf("failed to write username fixture: %v", err)
	}
	if err := os.WriteFile(passwordPath, []byte("password-initial"), 0o600); err != nil {
		t.Fatalf("failed to write password fixture: %v", err)
	}

	repoURL := "https://github.com/acme/file-backed-private-repo.git"
	repo := GitRepoConfig{
		ID:      GenerateID(repoURL),
		URL:     repoURL,
		Name:    "file-backed-private-repo",
		Enabled: true,
		Auth: GitRepoAuthConfig{
			Mode:        GitRepoAuthModeHTTPSBasic,
			Source:      GitRepoAuthSourceFile,
			UsernameRef: usernamePath,
			PasswordRef: passwordPath,
		},
	}

	syncer := NewGitSyncer(tempDir, []GitRepoConfig{repo}, nil)

	fileInfo, err := os.Stat(tempDir)
	if err != nil {
		t.Fatalf("failed to stat temp dir: %v", err)
	}
	syncer.statPath = func(string) (os.FileInfo, error) {
		return fileInfo, nil
	}

	type credentials struct {
		username string
		password string
	}
	var pullCredentials []credentials
	syncer.pullRepo = func(_ string, auth transport.AuthMethod, _ io.Writer) error {
		basicAuth, ok := auth.(*githttp.BasicAuth)
		if !ok {
			t.Fatalf("expected *http.BasicAuth, got %T", auth)
		}
		pullCredentials = append(pullCredentials, credentials{
			username: basicAuth.Username,
			password: basicAuth.Password,
		})
		return nil
	}

	if err := syncer.syncAll(); err != nil {
		t.Fatalf("expected syncAll to succeed, got %v", err)
	}

	if err := os.WriteFile(passwordPath, []byte("password-rotated"), 0o600); err != nil {
		t.Fatalf("failed to rotate password fixture: %v", err)
	}
	if err := os.WriteFile(usernamePath, []byte("ci-user-rotated"), 0o600); err != nil {
		t.Fatalf("failed to rotate username fixture: %v", err)
	}

	if err := syncer.SyncRepo(repo.ID); err != nil {
		t.Fatalf("expected manual sync to succeed after file credential rotation, got %v", err)
	}

	if len(pullCredentials) != 2 {
		t.Fatalf("expected two pull attempts, got %d", len(pullCredentials))
	}
	if pullCredentials[0].username != "ci-user" || pullCredentials[0].password != "password-initial" {
		t.Fatalf("expected initial file credentials to be used, got %+v", pullCredentials[0])
	}
	if pullCredentials[1].username != "ci-user-rotated" || pullCredentials[1].password != "password-rotated" {
		t.Fatalf("expected rotated file credentials to be used, got %+v", pullCredentials[1])
	}
}

func TestGitSyncerAuthFailureKeepsCheckoutAndStoresRedactedStatus(t *testing.T) {
	t.Setenv("WP005_AUTH_TOKEN", "supersecret-token")

	skillsDir := t.TempDir()
	repoURL := "https://github.com/acme/failing-private-repo.git"
	repo := GitRepoConfig{
		ID:      GenerateID(repoURL),
		URL:     repoURL,
		Name:    "failing-private-repo",
		Enabled: true,
		Auth: GitRepoAuthConfig{
			Mode:     GitRepoAuthModeHTTPSToken,
			Source:   GitRepoAuthSourceEnv,
			TokenRef: "WP005_AUTH_TOKEN",
		},
	}
	checkoutDir := filepath.Join(skillsDir, ResolveRepoCheckoutName(repo))
	if err := os.MkdirAll(checkoutDir, 0o755); err != nil {
		t.Fatalf("failed to create checkout dir: %v", err)
	}

	syncer := NewGitSyncer(skillsDir, []GitRepoConfig{repo}, nil)
	fileInfo, err := os.Stat(checkoutDir)
	if err != nil {
		t.Fatalf("failed to stat checkout dir: %v", err)
	}
	syncer.statPath = func(path string) (os.FileInfo, error) {
		if path == checkoutDir {
			return fileInfo, nil
		}
		return nil, os.ErrNotExist
	}
	syncer.pullRepo = func(_ string, _ transport.AuthMethod, _ io.Writer) error {
		return errors.New("password=supersecret-token")
	}

	err = syncer.SyncRepo(repo.ID)
	if err == nil {
		t.Fatalf("expected manual sync to fail")
	}
	if strings.Contains(err.Error(), "supersecret-token") {
		t.Fatalf("expected returned error to be redacted, got %q", err.Error())
	}

	if _, statErr := os.Stat(checkoutDir); statErr != nil {
		t.Fatalf("expected checkout directory to remain after sync failure, got %v", statErr)
	}

	status, ok := syncer.GetRepoSyncStatus(repo.ID)
	if !ok {
		t.Fatalf("expected sync status for repo %s", repo.ID)
	}
	if status.State != RepoSyncStateFailed {
		t.Fatalf("expected failed status, got %s", status.State)
	}
	if status.LastError == "" {
		t.Fatalf("expected redacted last error to be captured")
	}
	if strings.Contains(status.LastError, "supersecret-token") {
		t.Fatalf("expected status error to be redacted, got %q", status.LastError)
	}
}

func TestGitSyncerStoredCredentialFailureIsRedactedInErrorAndStatus(t *testing.T) {
	repoURL := "https://github.com/acme/stored-failure-private-repo.git"
	repo := GitRepoConfig{
		ID:      GenerateID(repoURL),
		URL:     repoURL,
		Name:    "stored-failure-private-repo",
		Enabled: true,
		Auth: GitRepoAuthConfig{
			Mode:   GitRepoAuthModeHTTPSToken,
			Source: GitRepoAuthSourceStored,
		},
	}

	syncer := NewGitSyncer(t.TempDir(), []GitRepoConfig{repo}, nil)
	syncer.SetStoredCredentialProvider(storedCredentialProviderFunc(func(GitRepoConfig) (ResolvedGitAuthCredentials, error) {
		return ResolvedGitAuthCredentials{}, errors.New("token=stored-super-secret")
	}))

	err := syncer.SyncRepo(repo.ID)
	if err == nil {
		t.Fatalf("expected stored credential sync to fail")
	}
	if strings.Contains(err.Error(), "stored-super-secret") {
		t.Fatalf("expected sync error to redact stored secret, got %q", err.Error())
	}

	status, ok := syncer.GetRepoSyncStatus(repo.ID)
	if !ok {
		t.Fatalf("expected sync status for repo %s", repo.ID)
	}
	if status.State != RepoSyncStateFailed {
		t.Fatalf("expected failed status, got %s", status.State)
	}
	if status.LastError == "" {
		t.Fatalf("expected redacted last error to be captured")
	}
	if strings.Contains(status.LastError, "stored-super-secret") {
		t.Fatalf("expected status error to redact stored secret, got %q", status.LastError)
	}
}

func TestGitSyncerStoredCredentialProviderUsedForStoredSource(t *testing.T) {
	repoURL := "https://github.com/acme/stored-private-repo.git"
	repo := GitRepoConfig{
		ID:      GenerateID(repoURL),
		URL:     repoURL,
		Name:    "stored-private-repo",
		Enabled: true,
		Auth: GitRepoAuthConfig{
			Mode:        GitRepoAuthModeHTTPSBasic,
			Source:      GitRepoAuthSourceStored,
			ReferenceID: "gitrepo_custom_reference",
		},
	}

	syncer := NewGitSyncer(t.TempDir(), []GitRepoConfig{repo}, nil)

	var providerCalls int
	syncer.SetStoredCredentialProvider(storedCredentialProviderFunc(func(resolvedRepo GitRepoConfig) (ResolvedGitAuthCredentials, error) {
		providerCalls++
		if resolvedRepo.ID != repo.ID {
			t.Fatalf("expected repo id %s, got %s", repo.ID, resolvedRepo.ID)
		}
		return ResolvedGitAuthCredentials{
			Mode:     GitRepoAuthModeHTTPSBasic,
			Source:   GitRepoAuthSourceStored,
			Username: "alice",
			Password: "stored-password",
		}, nil
	}))

	syncer.statPath = func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	var cloneAuth transport.AuthMethod
	syncer.cloneRepo = func(_ string, _ string, auth transport.AuthMethod, _ io.Writer) error {
		cloneAuth = auth
		return nil
	}

	if err := syncer.SyncRepo(repo.ID); err != nil {
		t.Fatalf("expected stored-credential sync to succeed, got %v", err)
	}

	if providerCalls != 1 {
		t.Fatalf("expected stored credential provider to be called once, got %d", providerCalls)
	}
	assertHTTPBasicAuthPassword(t, cloneAuth, "stored-password")
}

type credentialSourceResolverFunc func(reference string) (string, error)

func (f credentialSourceResolverFunc) Resolve(reference string) (string, error) {
	return f(reference)
}

func assertHTTPBasicAuthPassword(t *testing.T, method transport.AuthMethod, password string) {
	t.Helper()

	basicAuth, ok := method.(*githttp.BasicAuth)
	if !ok {
		t.Fatalf("expected *http.BasicAuth auth method, got %T", method)
	}
	if basicAuth.Password != password {
		t.Fatalf("expected auth password %q, got %q", password, basicAuth.Password)
	}
}
