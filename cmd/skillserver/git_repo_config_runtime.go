package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mudler/skillserver/pkg/git"
)

const gitRepoConfigFileName = ".git-repos.json"

func resolveGitRepoConfigPath(skillsDir string, persistenceCfg PersistenceRuntimeConfig) string {
	trimmedSkillsDir := strings.TrimSpace(skillsDir)
	if persistenceCfg.Enabled {
		persistenceDir := strings.TrimSpace(persistenceCfg.Dir)
		if persistenceDir != "" {
			return filepath.Join(persistenceDir, gitRepoConfigFileName)
		}
	}
	return filepath.Join(trimmedSkillsDir, gitRepoConfigFileName)
}

func migrateLegacyGitRepoConfigIfNeeded(
	configManager *git.ConfigManager,
	skillsDir string,
	persistenceCfg PersistenceRuntimeConfig,
) error {
	if configManager == nil {
		return fmt.Errorf("config manager is required")
	}
	if !persistenceCfg.Enabled {
		return nil
	}

	targetConfigPath := filepath.Clean(strings.TrimSpace(configManager.ConfigPath()))
	if targetConfigPath == "" || targetConfigPath == "." {
		return nil
	}

	legacyConfigPath := filepath.Join(strings.TrimSpace(skillsDir), gitRepoConfigFileName)
	if filepath.Clean(legacyConfigPath) == targetConfigPath {
		return nil
	}

	if _, err := os.Stat(targetConfigPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect git repo config path %q: %w", targetConfigPath, err)
	}

	if _, err := os.Stat(legacyConfigPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to inspect legacy git repo config path %q: %w", legacyConfigPath, err)
	}

	legacyConfigManager := git.NewConfigManagerWithPath(legacyConfigPath)
	legacyRepos, err := legacyConfigManager.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load legacy git repo config %q: %w", legacyConfigPath, err)
	}
	if err := configManager.SaveConfig(legacyRepos); err != nil {
		return fmt.Errorf("failed to migrate git repo config to %q: %w", targetConfigPath, err)
	}

	return nil
}
