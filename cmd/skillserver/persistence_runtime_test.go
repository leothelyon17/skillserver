package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePersistenceStartupConfig_DisabledPassthrough(t *testing.T) {
	err := validatePersistenceStartupConfig(PersistenceRuntimeConfig{
		Enabled: false,
		Dir:     "/missing/path",
		DBPath:  "/missing/path/skillserver.db",
	})
	if err != nil {
		t.Fatalf("expected disabled persistence mode to bypass startup validation, got %v", err)
	}
}

func TestValidatePersistenceStartupConfig_MissingDirectory(t *testing.T) {
	err := validatePersistenceStartupConfig(PersistenceRuntimeConfig{
		Enabled: true,
		DBPath:  "/tmp/skillserver.db",
	})
	if err == nil {
		t.Fatalf("expected missing directory error, got nil")
	}
	if !strings.Contains(err.Error(), envPersistenceDir) {
		t.Fatalf("expected missing directory error to reference %s, got %v", envPersistenceDir, err)
	}
}

func TestValidatePersistenceStartupConfig_DirectoryDoesNotExist(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "missing")

	err := validatePersistenceStartupConfig(PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     missingDir,
		DBPath:  filepath.Join(missingDir, "skillserver.db"),
	})
	if err == nil {
		t.Fatalf("expected missing directory error, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing directory error message, got %v", err)
	}
}

func TestValidatePersistenceStartupConfig_DirectoryNotWritable(t *testing.T) {
	dir := t.TempDir()
	oldProbe := persistenceDirectoryWriteProbe
	persistenceDirectoryWriteProbe = func(string) error {
		return errors.New("permission denied")
	}
	defer func() {
		persistenceDirectoryWriteProbe = oldProbe
	}()

	err := validatePersistenceStartupConfig(PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     dir,
		DBPath:  filepath.Join(dir, "skillserver.db"),
	})
	if err == nil {
		t.Fatalf("expected not-writable directory error, got nil")
	}
	if !strings.Contains(err.Error(), "not writable") {
		t.Fatalf("expected not-writable error message, got %v", err)
	}
}

func TestValidatePersistenceStartupConfig_DBPathParentDirectoryMissing(t *testing.T) {
	dir := t.TempDir()
	err := validatePersistenceStartupConfig(PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     dir,
		DBPath:  filepath.Join(dir, "nested", "skillserver.db"),
	})
	if err == nil {
		t.Fatalf("expected DB parent directory error, got nil")
	}
	if !strings.Contains(err.Error(), "parent directory") {
		t.Fatalf("expected parent directory error message, got %v", err)
	}
}

func TestValidatePersistenceStartupConfig_DBPathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "catalog")
	if err := os.Mkdir(dbPath, 0o755); err != nil {
		t.Fatalf("failed to create directory-backed DB path fixture: %v", err)
	}

	err := validatePersistenceStartupConfig(PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     dir,
		DBPath:  dbPath,
	})
	if err == nil {
		t.Fatalf("expected DB path directory error, got nil")
	}
	if !strings.Contains(err.Error(), "points to a directory") {
		t.Fatalf("expected DB path directory error message, got %v", err)
	}
}

func TestValidatePersistenceStartupConfig_ValidConfiguration(t *testing.T) {
	dir := t.TempDir()
	err := validatePersistenceStartupConfig(PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     dir,
		DBPath:  filepath.Join(dir, "skillserver.db"),
	})
	if err != nil {
		t.Fatalf("expected valid persistence configuration to pass startup guardrails, got %v", err)
	}
}

func TestShouldAutoEnablePersistenceRuntime_DefaultDisabledConfig_ReturnsTrue(t *testing.T) {
	fs := flag.NewFlagSet("auto-persistence-default", flag.ContinueOnError)
	flagValues := registerPersistenceRuntimeFlags(fs)
	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	cfg, err := parsePersistenceRuntimeConfig(fs, flagValues, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	if !shouldAutoEnablePersistenceRuntime(cfg, fs, func(string) (string, bool) {
		return "", false
	}) {
		t.Fatalf("expected default disabled persistence config to auto-enable at runtime")
	}
}

func TestShouldAutoEnablePersistenceRuntime_ExplicitFalseFlag_ReturnsFalse(t *testing.T) {
	fs := flag.NewFlagSet("auto-persistence-flag-false", flag.ContinueOnError)
	flagValues := registerPersistenceRuntimeFlags(fs)
	if err := fs.Parse([]string{"--persistence-data=false"}); err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	cfg, err := parsePersistenceRuntimeConfig(fs, flagValues, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	if shouldAutoEnablePersistenceRuntime(cfg, fs, func(string) (string, bool) {
		return "", false
	}) {
		t.Fatalf("expected explicit --persistence-data=false to disable auto-enable fallback")
	}
}

func TestShouldAutoEnablePersistenceRuntime_ExplicitFalseEnv_ReturnsFalse(t *testing.T) {
	fs := flag.NewFlagSet("auto-persistence-env-false", flag.ContinueOnError)
	flagValues := registerPersistenceRuntimeFlags(fs)
	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	cfg, err := parsePersistenceRuntimeConfig(fs, flagValues, func(key string) (string, bool) {
		if key == envPersistenceData {
			return "false", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("expected parse to succeed, got %v", err)
	}

	if shouldAutoEnablePersistenceRuntime(cfg, fs, func(key string) (string, bool) {
		if key == envPersistenceData {
			return "false", true
		}
		return "", false
	}) {
		t.Fatalf("expected explicit %s=false to disable auto-enable fallback", envPersistenceData)
	}
}

func TestResolveImplicitPersistenceRuntimeConfig_UsesSkillsDirStateSubdirectory(t *testing.T) {
	skillsDir := t.TempDir()

	cfg, err := resolveImplicitPersistenceRuntimeConfig(skillsDir)
	if err != nil {
		t.Fatalf("expected implicit persistence config resolution to succeed, got %v", err)
	}

	expectedDir := filepath.Join(skillsDir, defaultImplicitPersistenceDirName)
	if cfg.Dir != expectedDir {
		t.Fatalf("expected implicit persistence dir %q, got %q", expectedDir, cfg.Dir)
	}
	if cfg.DBPath != filepath.Join(expectedDir, defaultPersistenceDatabaseFileName) {
		t.Fatalf(
			"expected implicit persistence db path %q, got %q",
			filepath.Join(expectedDir, defaultPersistenceDatabaseFileName),
			cfg.DBPath,
		)
	}

	info, statErr := os.Stat(cfg.Dir)
	if statErr != nil {
		t.Fatalf("expected implicit persistence dir to exist, got %v", statErr)
	}
	if !info.IsDir() {
		t.Fatalf("expected implicit persistence path %q to be a directory", cfg.Dir)
	}
}
