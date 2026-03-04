package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

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
	if len(syncer.syncedURLs()) != 1 || syncer.syncedURLs()[0] != repoURL {
		t.Fatalf("expected manual syncer call for %q, got %+v", repoURL, syncer.syncedURLs())
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
	if len(syncer.syncedURLs()) != 1 || syncer.syncedURLs()[0] != repoURL {
		t.Fatalf("expected manual syncer call for %q, got %+v", repoURL, syncer.syncedURLs())
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
	mu        sync.Mutex
	repos     []string
	skillsDir string
	syncErr   error
	synced    []string
}

func (s *fakeGitSyncer) GetRepos() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.repos))
	copy(result, s.repos)
	return result
}

func (s *fakeGitSyncer) AddRepo(repoURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos = append(s.repos, repoURL)
	return nil
}

func (s *fakeGitSyncer) RemoveRepo(repoURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]string, 0, len(s.repos))
	for _, existing := range s.repos {
		if existing != repoURL {
			filtered = append(filtered, existing)
		}
	}
	s.repos = filtered
	return nil
}

func (s *fakeGitSyncer) UpdateRepos(repos []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.repos = append([]string(nil), repos...)
	return nil
}

func (s *fakeGitSyncer) SyncRepo(repoURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.syncErr != nil {
		return s.syncErr
	}
	s.synced = append(s.synced, repoURL)
	return nil
}

func (s *fakeGitSyncer) GetSkillsDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.skillsDir
}

func (s *fakeGitSyncer) syncedURLs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.synced))
	copy(result, s.synced)
	return result
}
