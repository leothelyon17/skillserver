package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/mudler/skillserver/pkg/domain"
	"github.com/mudler/skillserver/pkg/git"
)

func TestSyncGitRepo_UsesManualRepoSyncHookWhenConfigured(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	repoURL := "https://example.com/acme/repo-one.git"
	repoID := git.GenerateID(repoURL)
	repoName := git.ExtractRepoName(repoURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      repoID,
			URL:     repoURL,
			Name:    repoName,
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	var (
		hookCalls int
		hookRepo  git.GitRepoConfig
	)
	server.SetManualGitRepoSyncHook(func(repo git.GitRepoConfig) error {
		hookCalls++
		hookRepo = repo
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/git-repos/"+repoID+"/sync", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	if len(syncer.syncedRepoIDs()) != 1 || syncer.syncedRepoIDs()[0] != repoID {
		t.Fatalf("expected manual syncer call for repo id %q, got %+v", repoID, syncer.syncedRepoIDs())
	}
	if hookCalls != 1 {
		t.Fatalf("expected manual hook to be called once, got %d", hookCalls)
	}
	if hookRepo.ID != repoID || hookRepo.URL != repoURL || hookRepo.Name != repoName {
		t.Fatalf("unexpected hook repo payload: %+v", hookRepo)
	}
	if skillManager.rebuildIndexCallCount() != 0 {
		t.Fatalf("expected legacy rebuild fallback to be skipped when hook is configured")
	}
}

func TestSyncGitRepo_WithoutManualHook_RebuildsIndexForCompatibility(t *testing.T) {
	t.Parallel()

	configManager := git.NewConfigManager(t.TempDir())
	repoURL := "https://example.com/acme/repo-two.git"
	repoID := git.GenerateID(repoURL)
	if err := configManager.SaveConfig([]git.GitRepoConfig{
		{
			ID:      repoID,
			URL:     repoURL,
			Name:    git.ExtractRepoName(repoURL),
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("failed to seed git repo config: %v", err)
	}

	skillManager := &fakeGitSyncSkillManager{}
	syncer := &fakeGitSyncer{}
	server := NewServer(skillManager, nil, nil, syncer, configManager, false, nil, "")

	req := httptest.NewRequest(http.MethodPost, "/api/git-repos/"+repoID+"/sync", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}
	if len(syncer.syncedRepoIDs()) != 1 || syncer.syncedRepoIDs()[0] != repoID {
		t.Fatalf("expected manual syncer call for repo id %q, got %+v", repoID, syncer.syncedRepoIDs())
	}
	if skillManager.rebuildIndexCallCount() != 1 {
		t.Fatalf("expected legacy rebuild fallback to run once, got %d", skillManager.rebuildIndexCallCount())
	}
}

type fakeGitSyncSkillManager struct {
	mu              sync.Mutex
	rebuildCalls    int
	rebuildIndexErr error
}

func (m *fakeGitSyncSkillManager) ListSkills() ([]domain.Skill, error) {
	return []domain.Skill{}, nil
}

func (m *fakeGitSyncSkillManager) ReadSkill(string) (*domain.Skill, error) {
	return nil, errors.New("not implemented")
}

func (m *fakeGitSyncSkillManager) SearchSkills(string) ([]domain.Skill, error) {
	return []domain.Skill{}, nil
}

func (m *fakeGitSyncSkillManager) ListCatalogItems() ([]domain.CatalogItem, error) {
	return []domain.CatalogItem{}, nil
}

func (m *fakeGitSyncSkillManager) SearchCatalogItems(string, *domain.CatalogClassifier) ([]domain.CatalogItem, error) {
	return []domain.CatalogItem{}, nil
}

func (m *fakeGitSyncSkillManager) RebuildIndex() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rebuildCalls++
	return m.rebuildIndexErr
}

func (m *fakeGitSyncSkillManager) ListSkillResources(string) ([]domain.SkillResource, error) {
	return []domain.SkillResource{}, nil
}

func (m *fakeGitSyncSkillManager) ReadSkillResource(string, string) (*domain.ResourceContent, error) {
	return nil, errors.New("not implemented")
}

func (m *fakeGitSyncSkillManager) GetSkillResourceInfo(string, string) (*domain.SkillResource, error) {
	return nil, errors.New("not implemented")
}

func (m *fakeGitSyncSkillManager) rebuildIndexCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rebuildCalls
}

type fakeGitSyncer struct {
	mu         sync.Mutex
	repos      []git.GitRepoConfig
	statusByID map[string]git.RepoSyncStatus
	skillsDir  string
	syncErr    error
	synced     []string
}

func (s *fakeGitSyncer) GetRepos() []git.GitRepoConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]git.GitRepoConfig, len(s.repos))
	copy(result, s.repos)
	return result
}

func (s *fakeGitSyncer) AddRepo(repo git.GitRepoConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos = append(s.repos, repo)
	if s.statusByID == nil {
		s.statusByID = map[string]git.RepoSyncStatus{}
	}
	s.statusByID[repo.ID] = git.RepoSyncStatus{
		RepoID:       repo.ID,
		RepoURL:      repo.URL,
		CheckoutName: git.ResolveRepoCheckoutName(repo),
		State:        git.RepoSyncStateNever,
	}
	return nil
}

func (s *fakeGitSyncer) RemoveRepo(repoID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]git.GitRepoConfig, 0, len(s.repos))
	for _, existing := range s.repos {
		if existing.ID != repoID {
			filtered = append(filtered, existing)
		}
	}
	s.repos = filtered
	if s.statusByID != nil {
		delete(s.statusByID, repoID)
	}
	return nil
}

func (s *fakeGitSyncer) UpdateRepos(repos []git.GitRepoConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.repos = append([]git.GitRepoConfig(nil), repos...)
	statuses := make(map[string]git.RepoSyncStatus, len(repos))
	for _, repo := range repos {
		existing := git.RepoSyncStatus{
			RepoID:       repo.ID,
			RepoURL:      repo.URL,
			CheckoutName: git.ResolveRepoCheckoutName(repo),
			State:        git.RepoSyncStateNever,
		}
		if s.statusByID != nil {
			if current, ok := s.statusByID[repo.ID]; ok {
				existing = current
			}
		}
		existing.RepoID = repo.ID
		existing.RepoURL = repo.URL
		existing.CheckoutName = git.ResolveRepoCheckoutName(repo)
		if existing.State == "" {
			existing.State = git.RepoSyncStateNever
		}
		statuses[repo.ID] = existing
	}
	s.statusByID = statuses
	return nil
}

func (s *fakeGitSyncer) SyncRepo(repoID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.statusByID == nil {
		s.statusByID = map[string]git.RepoSyncStatus{}
	}

	if s.syncErr != nil {
		s.statusByID[repoID] = git.RepoSyncStatus{
			RepoID:      repoID,
			State:       git.RepoSyncStateFailed,
			LastError:   git.RedactGitAuthError(s.syncErr),
			LastAttempt: time.Now().UTC(),
		}
		return s.syncErr
	}
	s.synced = append(s.synced, repoID)
	s.statusByID[repoID] = git.RepoSyncStatus{
		RepoID:      repoID,
		State:       git.RepoSyncStateSuccess,
		LastAttempt: time.Now().UTC(),
		LastSuccess: time.Now().UTC(),
	}
	return nil
}

func (s *fakeGitSyncer) GetRepoSyncStatus(repoID string) (git.RepoSyncStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.statusByID == nil {
		return git.RepoSyncStatus{}, false
	}
	status, ok := s.statusByID[repoID]
	return status, ok
}

func (s *fakeGitSyncer) GetRepoSyncStatuses() map[string]git.RepoSyncStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]git.RepoSyncStatus, len(s.statusByID))
	for repoID, status := range s.statusByID {
		result[repoID] = status
	}
	return result
}

func (s *fakeGitSyncer) GetSkillsDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.skillsDir
}

func (s *fakeGitSyncer) syncedRepoIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.synced))
	copy(result, s.synced)
	return result
}
