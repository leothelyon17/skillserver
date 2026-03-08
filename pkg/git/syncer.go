package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

type cloneRepoFunc func(repoURL, targetDir string, auth transport.AuthMethod, progress io.Writer) error
type pullRepoFunc func(repoDir string, auth transport.AuthMethod, progress io.Writer) error
type statPathFunc func(name string) (os.FileInfo, error)

// StoredGitCredentialProvider resolves stored credentials for one repository.
type StoredGitCredentialProvider interface {
	Resolve(repo GitRepoConfig) (ResolvedGitAuthCredentials, error)
}

// RepoSyncState describes the outcome of the latest sync attempt for a repo.
type RepoSyncState string

const (
	// RepoSyncStateNever indicates no sync attempt has happened yet.
	RepoSyncStateNever RepoSyncState = "never"
	// RepoSyncStateSuccess indicates the latest sync attempt succeeded.
	RepoSyncStateSuccess RepoSyncState = "success"
	// RepoSyncStateFailed indicates the latest sync attempt failed.
	RepoSyncStateFailed RepoSyncState = "failed"
)

// RepoSyncStatus contains redacted per-repo sync status information.
type RepoSyncStatus struct {
	RepoID       string        `json:"repo_id"`
	RepoURL      string        `json:"repo_url"`
	CheckoutName string        `json:"checkout_name"`
	State        RepoSyncState `json:"state"`
	LastError    string        `json:"last_error,omitempty"`
	LastAttempt  time.Time     `json:"last_attempt,omitempty"`
	LastSuccess  time.Time     `json:"last_success,omitempty"`
}

// GitSyncer handles synchronization with Git repositories.
type GitSyncer struct {
	skillsDir string

	repos      []GitRepoConfig
	statusByID map[string]RepoSyncStatus

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	onUpdate func() error
	progress io.Writer
	logger   io.Writer

	credentialResolver       *GitCredentialResolver
	storedCredentialProvider StoredGitCredentialProvider

	cloneRepo cloneRepoFunc
	pullRepo  pullRepoFunc
	statPath  statPathFunc
	now       func() time.Time
}

// NewGitSyncer creates a new GitSyncer.
func NewGitSyncer(skillsDir string, repos []GitRepoConfig, onUpdate func() error) *GitSyncer {
	ctx, cancel := context.WithCancel(context.Background())

	syncer := &GitSyncer{
		skillsDir:                skillsDir,
		repos:                    []GitRepoConfig{},
		statusByID:               map[string]RepoSyncStatus{},
		ctx:                      ctx,
		cancel:                   cancel,
		onUpdate:                 onUpdate,
		progress:                 nil, // Default to no progress output (avoid MCP stdio interference).
		logger:                   nil,
		credentialResolver:       NewGitCredentialResolver(GitCredentialResolverOptions{}),
		storedCredentialProvider: nil,
		cloneRepo:                cloneRepository,
		pullRepo:                 pullRepository,
		statPath:                 os.Stat,
		now:                      time.Now,
	}

	normalizedRepos, err := normalizeSyncerRepoSlice(repos)
	if err != nil {
		normalizedRepos = append([]GitRepoConfig(nil), repos...)
	}
	syncer.repos = normalizedRepos
	for _, repo := range normalizedRepos {
		repoID := strings.TrimSpace(repo.ID)
		if repoID == "" {
			repoID = GenerateID(repo.URL)
		}
		syncer.statusByID[repoID] = RepoSyncStatus{
			RepoID:       repoID,
			RepoURL:      repo.URL,
			CheckoutName: ResolveRepoCheckoutName(repo),
			State:        RepoSyncStateNever,
		}
	}

	return syncer
}

// SetProgressWriter sets the writer for git progress output.
func (g *GitSyncer) SetProgressWriter(w io.Writer) {
	g.progress = w
}

// SetLogger sets the writer for log messages.
func (g *GitSyncer) SetLogger(w io.Writer) {
	g.logger = w
}

// SetCredentialResolver overrides the default env/file credential resolver.
func (g *GitSyncer) SetCredentialResolver(resolver *GitCredentialResolver) {
	if resolver == nil {
		resolver = NewGitCredentialResolver(GitCredentialResolverOptions{})
	}
	g.credentialResolver = resolver
}

// SetStoredCredentialProvider configures stored-credential resolution for source=stored repos.
func (g *GitSyncer) SetStoredCredentialProvider(provider StoredGitCredentialProvider) {
	g.storedCredentialProvider = provider
}

// Start begins the Git synchronization process.
func (g *GitSyncer) Start() error {
	// Initial sync.
	if err := g.syncAll(); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}

	// Start periodic sync in background.
	go g.periodicSync()

	return nil
}

// Stop stops the Git synchronization.
func (g *GitSyncer) Stop() {
	g.cancel()
}

// GetRepos returns a copy of the current repository list.
func (g *GitSyncer) GetRepos() []GitRepoConfig {
	g.mu.RLock()
	defer g.mu.RUnlock()

	repos := make([]GitRepoConfig, len(g.repos))
	copy(repos, g.repos)
	return repos
}

// GetRepoSyncStatus returns redacted sync status for one repo ID.
func (g *GitSyncer) GetRepoSyncStatus(repoID string) (RepoSyncStatus, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	status, ok := g.statusByID[repoID]
	return status, ok
}

// GetRepoSyncStatuses returns redacted sync status for all repos keyed by repo ID.
func (g *GitSyncer) GetRepoSyncStatuses() map[string]RepoSyncStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()

	statuses := make(map[string]RepoSyncStatus, len(g.statusByID))
	for repoID, status := range g.statusByID {
		statuses[repoID] = status
	}
	return statuses
}

// AddRepo adds a new repository and syncs it.
func (g *GitSyncer) AddRepo(repo GitRepoConfig) error {
	normalizedRepo, err := normalizeSyncerRepo(repo)
	if err != nil {
		return err
	}

	g.mu.Lock()
	if repoIndexByID(g.repos, normalizedRepo.ID) >= 0 {
		g.mu.Unlock()
		return fmt.Errorf("repository already exists: %s", normalizedRepo.ID)
	}
	g.repos = append(g.repos, normalizedRepo)
	g.statusByID[normalizedRepo.ID] = RepoSyncStatus{
		RepoID:       normalizedRepo.ID,
		RepoURL:      normalizedRepo.URL,
		CheckoutName: ResolveRepoCheckoutName(normalizedRepo),
		State:        RepoSyncStateNever,
	}
	g.mu.Unlock()

	// Sync the new repo.
	if err := g.syncRepo(normalizedRepo); err != nil {
		// Remove from list if sync failed.
		g.mu.Lock()
		index := repoIndexByID(g.repos, normalizedRepo.ID)
		if index >= 0 {
			g.repos = append(g.repos[:index], g.repos[index+1:]...)
		}
		delete(g.statusByID, normalizedRepo.ID)
		g.mu.Unlock()
		return fmt.Errorf("failed to sync new repository: %w", err)
	}

	// Trigger re-indexing.
	if g.onUpdate != nil {
		if err := g.onUpdate(); err != nil {
			return fmt.Errorf("failed to trigger re-indexing: %w", err)
		}
	}

	return nil
}

// RemoveRepo removes a repository from the list.
func (g *GitSyncer) RemoveRepo(repoID string) error {
	normalizedRepoID := strings.TrimSpace(repoID)
	if normalizedRepoID == "" {
		return fmt.Errorf("repository id is required")
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	index := repoIndexByID(g.repos, normalizedRepoID)
	if index < 0 {
		return fmt.Errorf("repository not found: %s", normalizedRepoID)
	}

	g.repos = append(g.repos[:index], g.repos[index+1:]...)
	delete(g.statusByID, normalizedRepoID)
	return nil
}

// GetSkillsDir returns the skills directory path.
func (g *GitSyncer) GetSkillsDir() string {
	return g.skillsDir
}

// UpdateRepos replaces the repository list and syncs all configured repos.
func (g *GitSyncer) UpdateRepos(repos []GitRepoConfig) error {
	normalizedRepos, err := normalizeSyncerRepoSlice(repos)
	if err != nil {
		return err
	}

	g.mu.Lock()
	oldRepos := make([]GitRepoConfig, len(g.repos))
	copy(oldRepos, g.repos)
	oldStatuses := make(map[string]RepoSyncStatus, len(g.statusByID))
	for repoID, status := range g.statusByID {
		oldStatuses[repoID] = status
	}

	g.repos = normalizedRepos
	g.reconcileStatusesLocked(normalizedRepos)
	g.mu.Unlock()

	if err := g.syncAll(); err != nil {
		// Restore old repos/statuses on hard sync-all errors (e.g. onUpdate failure).
		g.mu.Lock()
		g.repos = oldRepos
		g.statusByID = oldStatuses
		g.mu.Unlock()
		return err
	}

	return nil
}

func (g *GitSyncer) reconcileStatusesLocked(repos []GitRepoConfig) {
	currentIDs := make(map[string]struct{}, len(repos))
	for _, repo := range repos {
		currentIDs[repo.ID] = struct{}{}
		status, exists := g.statusByID[repo.ID]
		if !exists {
			g.statusByID[repo.ID] = RepoSyncStatus{
				RepoID:       repo.ID,
				RepoURL:      repo.URL,
				CheckoutName: ResolveRepoCheckoutName(repo),
				State:        RepoSyncStateNever,
			}
			continue
		}
		status.RepoURL = repo.URL
		status.CheckoutName = ResolveRepoCheckoutName(repo)
		g.statusByID[repo.ID] = status
	}

	for repoID := range g.statusByID {
		if _, keep := currentIDs[repoID]; !keep {
			delete(g.statusByID, repoID)
		}
	}
}

// syncAll syncs all configured repositories.
func (g *GitSyncer) syncAll() error {
	repos := g.GetRepos()
	for _, repo := range repos {
		if err := g.syncRepo(repo); err != nil {
			g.logf("Warning: failed to sync repo id=%s url=%s: %v", repo.ID, repo.URL, err)
		}
	}

	// Trigger re-indexing if callback is set.
	if g.onUpdate != nil {
		if err := g.onUpdate(); err != nil {
			return fmt.Errorf("failed to trigger re-indexing: %w", err)
		}
	}

	return nil
}

// SyncRepo manually syncs a specific repository by ID.
// This method only performs VCS synchronization. Callers are responsible for
// any follow-up catalog/index refresh behavior.
func (g *GitSyncer) SyncRepo(repoID string) error {
	normalizedRepoID := strings.TrimSpace(repoID)
	if normalizedRepoID == "" {
		return fmt.Errorf("repository id is required")
	}

	repo, found := g.getRepoByID(normalizedRepoID)
	if !found {
		return fmt.Errorf("repository not configured: %s", normalizedRepoID)
	}

	return g.syncRepo(repo)
}

func (g *GitSyncer) getRepoByID(repoID string) (GitRepoConfig, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	index := repoIndexByID(g.repos, repoID)
	if index < 0 {
		return GitRepoConfig{}, false
	}

	return g.repos[index], true
}

// syncRepo syncs a single repository.
func (g *GitSyncer) syncRepo(repo GitRepoConfig) error {
	checkoutName := ResolveRepoCheckoutName(repo)
	targetDir := filepath.Join(g.skillsDir, checkoutName)
	attemptedAt := g.now().UTC()

	authMethod, err := g.resolveAuthMethod(repo)
	if err != nil {
		syncErr := fmt.Errorf("resolve auth for repository %s: %s", repo.ID, redactSyncError(err))
		g.recordSyncFailure(repo, checkoutName, attemptedAt, syncErr)
		return syncErr
	}

	exists, err := g.repositoryDirectoryExists(targetDir)
	if err != nil {
		syncErr := fmt.Errorf("check repository directory for %s: %w", repo.ID, err)
		g.recordSyncFailure(repo, checkoutName, attemptedAt, syncErr)
		return syncErr
	}

	if !exists {
		if err := g.cloneRepo(repo.URL, targetDir, authMethod, g.progress); err != nil {
			syncErr := fmt.Errorf("clone repository %s: %s", repo.ID, redactSyncError(err))
			g.recordSyncFailure(repo, checkoutName, attemptedAt, syncErr)
			return syncErr
		}
	} else {
		if err := g.pullRepo(targetDir, authMethod, g.progress); err != nil {
			syncErr := fmt.Errorf("pull repository %s: %s", repo.ID, redactSyncError(err))
			g.recordSyncFailure(repo, checkoutName, attemptedAt, syncErr)
			return syncErr
		}
	}

	g.recordSyncSuccess(repo, checkoutName, attemptedAt)
	return nil
}

func (g *GitSyncer) resolveAuthMethod(repo GitRepoConfig) (transport.AuthMethod, error) {
	authConfig := normalizeGitRepoAuthConfig(repo.Auth)
	if authConfig.Mode == GitRepoAuthModeNone {
		return nil, nil
	}

	if authConfig.Source == GitRepoAuthSourceStored {
		if g.storedCredentialProvider == nil {
			return nil, NewRedactedGitAuthErrorf("stored credential source is not configured")
		}
		resolved, err := g.storedCredentialProvider.Resolve(repo)
		if err != nil {
			return nil, err
		}
		return BuildGitAuthMethod(resolved)
	}

	return ResolveGitAuthMethod(authConfig, g.credentialResolver)
}

func (g *GitSyncer) repositoryDirectoryExists(path string) (bool, error) {
	_, err := g.statPath(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (g *GitSyncer) recordSyncSuccess(repo GitRepoConfig, checkoutName string, attemptedAt time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.statusByID[repo.ID] = RepoSyncStatus{
		RepoID:       repo.ID,
		RepoURL:      repo.URL,
		CheckoutName: checkoutName,
		State:        RepoSyncStateSuccess,
		LastAttempt:  attemptedAt,
		LastSuccess:  attemptedAt,
		LastError:    "",
	}
}

func (g *GitSyncer) recordSyncFailure(repo GitRepoConfig, checkoutName string, attemptedAt time.Time, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	status := g.statusByID[repo.ID]
	status.RepoID = repo.ID
	status.RepoURL = repo.URL
	status.CheckoutName = checkoutName
	status.State = RepoSyncStateFailed
	status.LastAttempt = attemptedAt
	status.LastError = redactSyncError(err)
	g.statusByID[repo.ID] = status
}

// periodicSync runs periodic synchronization every 5 minutes.
func (g *GitSyncer) periodicSync() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			if err := g.syncAll(); err != nil {
				g.logf("Warning: periodic sync failed: %v", err)
			}
		}
	}
}

func (g *GitSyncer) logf(format string, args ...any) {
	if g.logger == nil {
		return
	}
	_, _ = fmt.Fprintf(g.logger, format+"\n", args...)
}

func normalizeSyncerRepoSlice(repos []GitRepoConfig) ([]GitRepoConfig, error) {
	normalized := make([]GitRepoConfig, len(repos))
	seenIDs := make(map[string]struct{}, len(repos))

	for i, repo := range repos {
		normalizedRepo, err := normalizeSyncerRepo(repo)
		if err != nil {
			return nil, fmt.Errorf("normalize repository at index %d: %w", i, err)
		}
		if _, exists := seenIDs[normalizedRepo.ID]; exists {
			return nil, fmt.Errorf("duplicate repository id %q in sync configuration", normalizedRepo.ID)
		}
		seenIDs[normalizedRepo.ID] = struct{}{}
		normalized[i] = normalizedRepo
	}

	return normalized, nil
}

func normalizeSyncerRepo(repo GitRepoConfig) (GitRepoConfig, error) {
	normalized, err := NormalizeGitRepoConfig(repo)
	if err != nil {
		return GitRepoConfig{}, err
	}
	normalized.Name = ResolveRepoCheckoutName(normalized)
	return normalized, nil
}

func repoIndexByID(repos []GitRepoConfig, repoID string) int {
	for i, repo := range repos {
		if repo.ID == repoID {
			return i
		}
	}
	return -1
}

func redactSyncError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, transport.ErrAuthenticationRequired) {
		return "authentication required"
	}

	redacted := RedactGitAuthError(err)
	if redacted == "" {
		return "unknown synchronization error"
	}
	return redacted
}

func cloneRepository(
	repoURL, targetDir string,
	auth transport.AuthMethod,
	progress io.Writer,
) error {
	_, err := git.PlainClone(targetDir, false, &git.CloneOptions{
		URL:      repoURL,
		Auth:     auth,
		Progress: progress,
	})
	if err != nil {
		return err
	}
	return nil
}

func pullRepository(repoDir string, auth transport.AuthMethod, progress io.Writer) error {
	r, err := git.PlainOpen(repoDir)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Pull(&git.PullOptions{
		Auth:     auth,
		Progress: progress,
	})
	if err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil
		}
		return err
	}

	return nil
}
