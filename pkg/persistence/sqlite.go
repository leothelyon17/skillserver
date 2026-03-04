package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	sqliteDriverName          = "sqlite"
	defaultSQLiteBusyTimeout  = 5 * time.Second
	defaultSQLiteJournalMode  = "WAL"
	defaultSQLiteSyncMode     = "NORMAL"
	defaultSQLiteMaxOpenConns = 1
)

// SQLiteBootstrapConfig controls SQLite bootstrap behavior.
type SQLiteBootstrapConfig struct {
	BusyTimeout time.Duration
}

// OpenSQLite opens a SQLite database connection and applies runtime pragmas.
func OpenSQLite(ctx context.Context, dbPath string, cfg SQLiteBootstrapConfig) (*sql.DB, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedPath, err := normalizeSQLitePath(dbPath)
	if err != nil {
		return nil, err
	}
	if err := validateSQLiteParentDirectory(normalizedPath); err != nil {
		return nil, err
	}

	db, err := sql.Open(sqliteDriverName, normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database at %q: %w", normalizedPath, err)
	}

	db.SetMaxOpenConns(defaultSQLiteMaxOpenConns)
	db.SetMaxIdleConns(defaultSQLiteMaxOpenConns)
	db.SetConnMaxLifetime(0)

	if err := applySQLitePragmas(ctx, db, cfg); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database at %q: %w", normalizedPath, err)
	}

	return db, nil
}

// BootstrapSQLite opens the SQLite database and runs all schema migrations.
func BootstrapSQLite(ctx context.Context, dbPath string, cfg SQLiteBootstrapConfig) (*sql.DB, error) {
	db, err := OpenSQLite(ctx, dbPath, cfg)
	if err != nil {
		return nil, err
	}

	if err := RunMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run sqlite migrations: %w", err)
	}

	return db, nil
}

// CloseSQLite closes the sqlite handle when present.
func CloseSQLite(db *sql.DB) error {
	if db == nil {
		return nil
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("close sqlite database: %w", err)
	}

	return nil
}

func normalizeSQLitePath(dbPath string) (string, error) {
	trimmed := strings.TrimSpace(dbPath)
	if trimmed == "" {
		return "", fmt.Errorf("sqlite database path is required")
	}

	cleaned := filepath.Clean(trimmed)
	resolved, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolve sqlite database path %q: %w", dbPath, err)
	}

	return resolved, nil
}

func validateSQLiteParentDirectory(dbPath string) error {
	parent := filepath.Dir(dbPath)
	info, err := os.Stat(parent)
	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("sqlite database parent directory %q does not exist", parent)
	case err != nil:
		return fmt.Errorf("inspect sqlite database parent directory %q: %w", parent, err)
	case !info.IsDir():
		return fmt.Errorf("sqlite database parent path %q is not a directory", parent)
	default:
		return nil
	}
}

func applySQLitePragmas(ctx context.Context, db *sql.DB, cfg SQLiteBootstrapConfig) error {
	busyTimeout := cfg.BusyTimeout
	if busyTimeout <= 0 {
		busyTimeout = defaultSQLiteBusyTimeout
	}
	busyTimeoutMS := busyTimeout.Milliseconds()

	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("enable sqlite foreign_keys pragma: %w", err)
	}

	var journalMode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode = "+defaultSQLiteJournalMode+";").Scan(&journalMode); err != nil {
		return fmt.Errorf("set sqlite journal_mode pragma: %w", err)
	}
	if !strings.EqualFold(journalMode, defaultSQLiteJournalMode) {
		return fmt.Errorf("unexpected sqlite journal mode %q", journalMode)
	}

	if _, err := db.ExecContext(ctx, "PRAGMA synchronous = "+defaultSQLiteSyncMode+";"); err != nil {
		return fmt.Errorf("set sqlite synchronous pragma: %w", err)
	}

	if _, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d;", busyTimeoutMS)); err != nil {
		return fmt.Errorf("set sqlite busy_timeout pragma: %w", err)
	}

	return nil
}
