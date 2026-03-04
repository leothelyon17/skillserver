package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenSQLite_WithValidPath_CreatesDatabaseAndAppliesPragmas(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPath := filepath.Join(t.TempDir(), "skillserver.db")
	db, err := OpenSQLite(ctx, dbPath, SQLiteBootstrapConfig{})
	if err != nil {
		t.Fatalf("expected sqlite open to succeed, got %v", err)
	}
	t.Cleanup(func() {
		if closeErr := CloseSQLite(db); closeErr != nil {
			t.Fatalf("failed to close sqlite handle: %v", closeErr)
		}
	})

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected sqlite file %q to exist, got error %v", dbPath, err)
	}

	var foreignKeys int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&foreignKeys); err != nil {
		t.Fatalf("expected foreign_keys pragma query to succeed, got %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("expected foreign_keys pragma value %d, got %d", 1, foreignKeys)
	}

	var journalMode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("expected journal_mode pragma query to succeed, got %v", err)
	}
	if !strings.EqualFold(journalMode, defaultSQLiteJournalMode) {
		t.Fatalf("expected journal_mode %q, got %q", defaultSQLiteJournalMode, journalMode)
	}

	var busyTimeout int
	if err := db.QueryRowContext(ctx, "PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("expected busy_timeout pragma query to succeed, got %v", err)
	}
	if busyTimeout != int(defaultSQLiteBusyTimeout.Milliseconds()) {
		t.Fatalf(
			"expected busy_timeout %dms, got %dms",
			defaultSQLiteBusyTimeout.Milliseconds(),
			busyTimeout,
		)
	}
}

func TestOpenSQLite_WithEmptyPath_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := OpenSQLite(ctx, " ", SQLiteBootstrapConfig{})
	if err == nil {
		t.Fatalf("expected sqlite open with empty path to fail, got nil")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("expected sqlite open error to mention required path, got %v", err)
	}
}

func TestOpenSQLite_WithNilContext_Succeeds(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nil-context.db")
	db, err := OpenSQLite(nil, dbPath, SQLiteBootstrapConfig{})
	if err != nil {
		t.Fatalf("expected sqlite open with nil context to succeed, got %v", err)
	}
	t.Cleanup(func() {
		if closeErr := CloseSQLite(db); closeErr != nil {
			t.Fatalf("failed to close sqlite handle: %v", closeErr)
		}
	})
}

func TestOpenSQLite_WithMissingParentDirectory_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPath := filepath.Join(t.TempDir(), "missing", "skillserver.db")
	_, err := OpenSQLite(ctx, dbPath, SQLiteBootstrapConfig{})
	if err == nil {
		t.Fatalf("expected sqlite open with missing parent directory to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "parent directory") {
		t.Fatalf("expected sqlite open error to mention parent directory, got %v", err)
	}
}

func TestOpenSQLite_WithParentPathAsFile_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tempDir := t.TempDir()
	parentFile := filepath.Join(tempDir, "not-a-dir")
	if err := os.WriteFile(parentFile, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("failed to create parent file fixture: %v", err)
	}

	dbPath := filepath.Join(parentFile, "skillserver.db")
	_, err := OpenSQLite(ctx, dbPath, SQLiteBootstrapConfig{})
	if err == nil {
		t.Fatalf("expected sqlite open with parent file path to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "not a directory") {
		t.Fatalf("expected sqlite open error to mention non-directory parent path, got %v", err)
	}
}

func TestOpenSQLite_WithCustomBusyTimeout_AppliesConfiguredValue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPath := filepath.Join(t.TempDir(), "custom-timeout.db")
	customBusyTimeout := 2 * time.Second
	db, err := OpenSQLite(ctx, dbPath, SQLiteBootstrapConfig{
		BusyTimeout: customBusyTimeout,
	})
	if err != nil {
		t.Fatalf("expected sqlite open to succeed with custom timeout, got %v", err)
	}
	t.Cleanup(func() {
		if closeErr := CloseSQLite(db); closeErr != nil {
			t.Fatalf("failed to close sqlite handle: %v", closeErr)
		}
	})

	var busyTimeout int
	if err := db.QueryRowContext(ctx, "PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("expected busy_timeout pragma query to succeed, got %v", err)
	}
	if busyTimeout != int(customBusyTimeout.Milliseconds()) {
		t.Fatalf(
			"expected busy_timeout %dms, got %dms",
			customBusyTimeout.Milliseconds(),
			busyTimeout,
		)
	}
}

func TestCloseSQLite_WithNilHandle_ReturnsNil(t *testing.T) {
	if err := CloseSQLite(nil); err != nil {
		t.Fatalf("expected nil sqlite handle close to return nil, got %v", err)
	}
}

func TestApplySQLitePragmas_WithClosedDatabase_ReturnsError(t *testing.T) {
	db, err := sql.Open(sqliteDriverName, ":memory:")
	if err != nil {
		t.Fatalf("expected sqlite in-memory open to succeed, got %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("expected sqlite close to succeed, got %v", err)
	}

	err = applySQLitePragmas(context.Background(), db, SQLiteBootstrapConfig{})
	if err == nil {
		t.Fatalf("expected pragma application against closed DB to fail, got nil")
	}
}

func TestBootstrapSQLite_WithFreshDatabase_RunsMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPath := filepath.Join(t.TempDir(), "catalog.db")
	db, err := BootstrapSQLite(ctx, dbPath, SQLiteBootstrapConfig{})
	if err != nil {
		t.Fatalf("expected sqlite bootstrap to succeed, got %v", err)
	}
	t.Cleanup(func() {
		if closeErr := CloseSQLite(db); closeErr != nil {
			t.Fatalf("failed to close sqlite handle: %v", closeErr)
		}
	})

	runner := NewMigrationRunner(db)
	version, err := runner.CurrentVersion(ctx)
	if err != nil {
		t.Fatalf("expected schema version query to succeed, got %v", err)
	}
	if version != LatestSchemaVersion() {
		t.Fatalf("expected schema version %d, got %d", LatestSchemaVersion(), version)
	}
}

func TestBootstrapSQLite_WithInvalidMigration_ReturnsError(t *testing.T) {
	originalMigrations := schemaMigrations
	schemaMigrations = []migration{
		{
			version: 1,
			name:    "broken_schema",
			statements: []string{
				"CREATE TABLE broken_schema (",
			},
		},
	}
	t.Cleanup(func() {
		schemaMigrations = originalMigrations
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPath := filepath.Join(t.TempDir(), "broken.db")
	_, err := BootstrapSQLite(ctx, dbPath, SQLiteBootstrapConfig{})
	if err == nil {
		t.Fatalf("expected sqlite bootstrap with invalid migration to fail, got nil")
	}
	if !strings.Contains(err.Error(), "run sqlite migrations") {
		t.Fatalf("expected bootstrap error to include migration context, got %v", err)
	}
}
