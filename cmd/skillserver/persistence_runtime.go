package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	envPersistenceData   = "SKILLSERVER_PERSISTENCE_DATA"
	envPersistenceDir    = "SKILLSERVER_PERSISTENCE_DIR"
	envPersistenceDBPath = "SKILLSERVER_PERSISTENCE_DB_PATH"
)

const (
	defaultPersistenceData             = false
	defaultPersistenceDir              = ""
	defaultPersistenceDBPath           = ""
	defaultPersistenceDatabaseFileName = "skillserver.db"
)

// PersistenceRuntimeConfig defines runtime configuration for durable persistence.
type PersistenceRuntimeConfig struct {
	Enabled bool
	Dir     string
	DBPath  string
}

type persistenceRuntimeFlagValues struct {
	enabled bool
	dir     string
	dbPath  string
}

var persistenceDirectoryWriteProbe = probeDirectoryWritable

// registerPersistenceRuntimeFlags adds persistence runtime flags to a flag set.
func registerPersistenceRuntimeFlags(fs *flag.FlagSet) *persistenceRuntimeFlagValues {
	values := &persistenceRuntimeFlagValues{}

	fs.BoolVar(
		&values.enabled,
		"persistence-data",
		defaultPersistenceData,
		"Enable durable persistence mode (env: SKILLSERVER_PERSISTENCE_DATA)",
	)
	fs.StringVar(
		&values.dir,
		"persistence-dir",
		defaultPersistenceDir,
		"Writable persistence directory path (env: SKILLSERVER_PERSISTENCE_DIR)",
	)
	fs.StringVar(
		&values.dbPath,
		"persistence-db-path",
		defaultPersistenceDBPath,
		"SQLite database file path override (env: SKILLSERVER_PERSISTENCE_DB_PATH)",
	)

	return values
}

// parsePersistenceRuntimeConfig resolves and validates persistence runtime config with precedence:
// flags > environment variables > defaults.
func parsePersistenceRuntimeConfig(
	fs *flag.FlagSet,
	flagValues *persistenceRuntimeFlagValues,
	lookupEnv func(string) (string, bool),
) (PersistenceRuntimeConfig, error) {
	if fs == nil {
		return PersistenceRuntimeConfig{}, fmt.Errorf("flag set is required")
	}
	if flagValues == nil {
		return PersistenceRuntimeConfig{}, fmt.Errorf("flag values are required")
	}

	enabled, err := resolveBoolConfigValue(
		fs,
		"persistence-data",
		flagValues.enabled,
		envPersistenceData,
		defaultPersistenceData,
		lookupEnv,
	)
	if err != nil {
		return PersistenceRuntimeConfig{}, err
	}

	if !enabled {
		return PersistenceRuntimeConfig{Enabled: false}, nil
	}

	dirRaw, dirSource := resolveStringConfigValue(
		fs,
		"persistence-dir",
		flagValues.dir,
		envPersistenceDir,
		defaultPersistenceDir,
		lookupEnv,
	)
	dir, err := parsePersistenceDirectory(dirRaw)
	if err != nil {
		return PersistenceRuntimeConfig{}, fmt.Errorf("%s: %w", dirSource, err)
	}

	dbPathRaw, dbPathSource := resolveStringConfigValue(
		fs,
		"persistence-db-path",
		flagValues.dbPath,
		envPersistenceDBPath,
		defaultPersistenceDBPath,
		lookupEnv,
	)
	dbPath, err := parsePersistenceDBPath(dbPathRaw, dir)
	if err != nil {
		return PersistenceRuntimeConfig{}, fmt.Errorf("%s: %w", dbPathSource, err)
	}

	return PersistenceRuntimeConfig{
		Enabled: true,
		Dir:     dir,
		DBPath:  dbPath,
	}, nil
}

func parsePersistenceDirectory(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("persistence directory is required when %s=true", envPersistenceData)
	}

	cleaned := filepath.Clean(value)
	resolved, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("failed to resolve persistence directory %q: %w", raw, err)
	}

	return resolved, nil
}

func parsePersistenceDBPath(raw string, persistenceDir string) (string, error) {
	if strings.TrimSpace(persistenceDir) == "" {
		return "", fmt.Errorf("persistence directory is required to resolve database path")
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return filepath.Join(persistenceDir, defaultPersistenceDatabaseFileName), nil
	}

	if filepath.IsAbs(value) {
		return filepath.Clean(value), nil
	}

	cleaned := filepath.Clean(value)
	if cleaned == "." {
		return "", fmt.Errorf("persistence DB path must reference a file, got %q", raw)
	}

	return filepath.Join(persistenceDir, cleaned), nil
}

// validatePersistenceStartupConfig performs startup guardrails for persistence mode.
func validatePersistenceStartupConfig(cfg PersistenceRuntimeConfig) error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.Dir == "" {
		return fmt.Errorf("%s is required when %s=true", envPersistenceDir, envPersistenceData)
	}
	if cfg.DBPath == "" {
		return fmt.Errorf("%s resolved to an empty path while %s=true", envPersistenceDBPath, envPersistenceData)
	}

	if err := validatePersistenceDirectory(cfg.Dir); err != nil {
		return err
	}
	if err := validatePersistenceDBPath(cfg.DBPath); err != nil {
		return err
	}

	return nil
}

func validatePersistenceDirectory(dir string) error {
	info, err := os.Stat(dir)
	switch {
	case os.IsNotExist(err):
		return fmt.Errorf(
			"%s=%q does not exist; mount a writable directory (Docker volume/PVC) and set %s to that mount path",
			envPersistenceDir,
			dir,
			envPersistenceDir,
		)
	case err != nil:
		return fmt.Errorf("failed to inspect %s=%q: %w", envPersistenceDir, dir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s=%q is not a directory", envPersistenceDir, dir)
	}

	if err := persistenceDirectoryWriteProbe(dir); err != nil {
		return fmt.Errorf("%s=%q is not writable: %w", envPersistenceDir, dir, err)
	}

	return nil
}

func validatePersistenceDBPath(dbPath string) error {
	dbDir := filepath.Dir(dbPath)

	info, err := os.Stat(dbDir)
	switch {
	case os.IsNotExist(err):
		return fmt.Errorf(
			"%s=%q has parent directory %q that does not exist",
			envPersistenceDBPath,
			dbPath,
			dbDir,
		)
	case err != nil:
		return fmt.Errorf("failed to inspect parent directory %q for %s=%q: %w", dbDir, envPersistenceDBPath, dbPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s=%q has parent path %q that is not a directory", envPersistenceDBPath, dbPath, dbDir)
	}

	if err := persistenceDirectoryWriteProbe(dbDir); err != nil {
		return fmt.Errorf("%s=%q has parent directory %q that is not writable: %w", envPersistenceDBPath, dbPath, dbDir, err)
	}

	fileInfo, err := os.Stat(dbPath)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return fmt.Errorf("failed to inspect %s=%q: %w", envPersistenceDBPath, dbPath, err)
	case fileInfo.IsDir():
		return fmt.Errorf("%s=%q points to a directory; expected a writable database file path", envPersistenceDBPath, dbPath)
	default:
		return nil
	}
}

func probeDirectoryWritable(dir string) error {
	file, err := os.CreateTemp(dir, ".skillserver-write-check-*")
	if err != nil {
		return err
	}

	name := file.Name()
	closeErr := file.Close()
	removeErr := os.Remove(name)
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}

	return nil
}
