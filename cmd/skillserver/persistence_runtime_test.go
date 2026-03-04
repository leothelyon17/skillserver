package main

import (
	"errors"
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
