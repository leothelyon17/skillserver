package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mudler/skillserver/pkg/git"
)

func TestResolveGitRepoConfigPath_DefaultsToSkillsDirectory(t *testing.T) {
	skillsDir := filepath.Join(t.TempDir(), "skills")

	resolved := resolveGitRepoConfigPath(skillsDir, PersistenceRuntimeConfig{Enabled: false})
	expected := filepath.Join(skillsDir, gitRepoConfigFileName)
	if resolved != expected {
		t.Fatalf("expected config path %q, got %q", expected, resolved)
	}
}

func TestResolveGitRepoConfigPath_UsesPersistenceDirectoryWhenEnabled(t *testing.T) {
	skillsDir := filepath.Join(t.TempDir(), "skills")
	persistenceDir := filepath.Join(t.TempDir(), "persist")

	resolved := resolveGitRepoConfigPath(skillsDir, PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     persistenceDir,
	})

	expected := filepath.Join(persistenceDir, gitRepoConfigFileName)
	if resolved != expected {
		t.Fatalf("expected config path %q, got %q", expected, resolved)
	}
}

func TestMigrateLegacyGitRepoConfigIfNeeded_NoOpWhenPersistenceDisabled(t *testing.T) {
	skillsDir := filepath.Join(t.TempDir(), "skills")
	persistenceDir := filepath.Join(t.TempDir(), "persist")

	configManager := git.NewConfigManagerWithPath(filepath.Join(persistenceDir, gitRepoConfigFileName))
	err := migrateLegacyGitRepoConfigIfNeeded(configManager, skillsDir, PersistenceRuntimeConfig{Enabled: false})
	if err != nil {
		t.Fatalf("expected no error when persistence is disabled, got %v", err)
	}

	_, statErr := os.Stat(configManager.ConfigPath())
	if !os.IsNotExist(statErr) {
		t.Fatalf("expected no migrated config file when persistence is disabled, got stat error %v", statErr)
	}
}

func TestMigrateLegacyGitRepoConfigIfNeeded_CopiesLegacyConfigToPersistencePath(t *testing.T) {
	baseDir := t.TempDir()
	skillsDir := filepath.Join(baseDir, "skills")
	persistenceDir := filepath.Join(baseDir, "persist")

	legacyManager := git.NewConfigManager(skillsDir)
	legacyRepos := []git.GitRepoConfig{
		{
			URL:     "https://github.com/acme/legacy-repo.git",
			Name:    "legacy-repo",
			Enabled: true,
		},
	}
	if err := legacyManager.SaveConfig(legacyRepos); err != nil {
		t.Fatalf("expected legacy save to succeed, got %v", err)
	}

	targetManager := git.NewConfigManagerWithPath(filepath.Join(persistenceDir, gitRepoConfigFileName))
	err := migrateLegacyGitRepoConfigIfNeeded(targetManager, skillsDir, PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     persistenceDir,
	})
	if err != nil {
		t.Fatalf("expected migration to succeed, got %v", err)
	}

	migratedRepos, err := targetManager.LoadConfig()
	if err != nil {
		t.Fatalf("expected migrated repos to load, got %v", err)
	}
	if len(migratedRepos) != 1 {
		t.Fatalf("expected one migrated repo, got %d", len(migratedRepos))
	}
	if migratedRepos[0].URL != "https://github.com/acme/legacy-repo.git" {
		t.Fatalf("expected migrated repo URL %q, got %q", "https://github.com/acme/legacy-repo.git", migratedRepos[0].URL)
	}
}

func TestMigrateLegacyGitRepoConfigIfNeeded_DoesNotOverwriteExistingPersistenceConfig(t *testing.T) {
	baseDir := t.TempDir()
	skillsDir := filepath.Join(baseDir, "skills")
	persistenceDir := filepath.Join(baseDir, "persist")

	legacyManager := git.NewConfigManager(skillsDir)
	if err := legacyManager.SaveConfig([]git.GitRepoConfig{
		{
			URL:     "https://github.com/acme/legacy-repo.git",
			Name:    "legacy-repo",
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("expected legacy save to succeed, got %v", err)
	}

	targetManager := git.NewConfigManagerWithPath(filepath.Join(persistenceDir, gitRepoConfigFileName))
	if err := targetManager.SaveConfig([]git.GitRepoConfig{
		{
			URL:     "https://github.com/acme/current-repo.git",
			Name:    "current-repo",
			Enabled: true,
		},
	}); err != nil {
		t.Fatalf("expected target save to succeed, got %v", err)
	}

	err := migrateLegacyGitRepoConfigIfNeeded(targetManager, skillsDir, PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     persistenceDir,
	})
	if err != nil {
		t.Fatalf("expected migration no-op to succeed, got %v", err)
	}

	repos, err := targetManager.LoadConfig()
	if err != nil {
		t.Fatalf("expected target repos to load, got %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected one existing repo, got %d", len(repos))
	}
	if repos[0].URL != "https://github.com/acme/current-repo.git" {
		t.Fatalf("expected existing repo URL to remain unchanged, got %q", repos[0].URL)
	}
}
