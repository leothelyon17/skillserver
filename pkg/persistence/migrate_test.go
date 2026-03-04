package persistence

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLatestSchemaVersion_WithNoMigrations_ReturnsZero(t *testing.T) {
	originalMigrations := schemaMigrations
	schemaMigrations = nil
	t.Cleanup(func() {
		schemaMigrations = originalMigrations
	})

	version := LatestSchemaVersion()
	if version != 0 {
		t.Fatalf("expected latest schema version %d, got %d", 0, version)
	}
}

func TestMigrationRunner_CurrentVersion_WithNilRunner_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var runner *MigrationRunner
	_, err := runner.CurrentVersion(ctx)
	if err == nil {
		t.Fatalf("expected nil runner version query to fail, got nil")
	}
	if !strings.Contains(err.Error(), "migration runner is required") {
		t.Fatalf("expected nil runner error, got %v", err)
	}
}

func TestMigrationRunner_CurrentVersion_WithNilDatabase_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runner := NewMigrationRunner(nil)
	_, err := runner.CurrentVersion(ctx)
	if err == nil {
		t.Fatalf("expected nil database version query to fail, got nil")
	}
	if !strings.Contains(err.Error(), "sqlite database handle is required") {
		t.Fatalf("expected nil database error, got %v", err)
	}
}

func TestMigrationRunner_Run_WithNilDatabase_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runner := NewMigrationRunner(nil)
	err := runner.Run(ctx)
	if err == nil {
		t.Fatalf("expected nil database migration run to fail, got nil")
	}
	if !strings.Contains(err.Error(), "sqlite database handle is required") {
		t.Fatalf("expected nil database error, got %v", err)
	}
}

func TestMigrationRunner_Run_WithNilRunner_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var runner *MigrationRunner
	err := runner.Run(ctx)
	if err == nil {
		t.Fatalf("expected nil runner migration run to fail, got nil")
	}
	if !strings.Contains(err.Error(), "migration runner is required") {
		t.Fatalf("expected nil runner error, got %v", err)
	}
}

func TestRunMigrations_WithEmptyDatabase_AppliesCurrentSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db := openSQLiteTestDB(t, ctx)
	runner := NewMigrationRunner(db)

	preVersion, err := runner.CurrentVersion(ctx)
	if err != nil {
		t.Fatalf("expected pre-migration version query to succeed, got %v", err)
	}
	if preVersion != 0 {
		t.Fatalf("expected empty database schema version %d, got %d", 0, preVersion)
	}

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("expected migrations to succeed, got %v", err)
	}

	postVersion, err := runner.CurrentVersion(ctx)
	if err != nil {
		t.Fatalf("expected post-migration version query to succeed, got %v", err)
	}
	if postVersion != LatestSchemaVersion() {
		t.Fatalf("expected schema version %d, got %d", LatestSchemaVersion(), postVersion)
	}

	requiredTables := []string{
		"schema_migrations",
		"catalog_source_items",
		"catalog_metadata_overlays",
		"system_state",
	}
	for _, table := range requiredTables {
		exists, err := sqliteObjectExists(ctx, db, "table", table)
		if err != nil {
			t.Fatalf("expected table existence query to succeed for %q, got %v", table, err)
		}
		if !exists {
			t.Fatalf("expected table %q to exist", table)
		}
	}

	requiredIndexes := []string{
		"idx_catalog_source_classifier_deleted_at",
		"idx_catalog_source_source_filters",
		"idx_catalog_source_lookup_paths",
		"idx_catalog_source_resource_path",
	}
	for _, index := range requiredIndexes {
		exists, err := sqliteObjectExists(ctx, db, "index", index)
		if err != nil {
			t.Fatalf("expected index existence query to succeed for %q, got %v", index, err)
		}
		if !exists {
			t.Fatalf("expected index %q to exist", index)
		}
	}

	requiredColumns := []sqliteColumnExpectation{
		{table: "catalog_source_items", column: "deleted_at"},
		{table: "catalog_source_items", column: "metadata_writable"},
		{table: "catalog_metadata_overlays", column: "display_name_override"},
		{table: "catalog_metadata_overlays", column: "custom_metadata_json"},
		{table: "system_state", column: "state_key"},
	}
	for _, expectation := range requiredColumns {
		exists, err := sqliteColumnExists(ctx, db, expectation.table, expectation.column)
		if err != nil {
			t.Fatalf(
				"expected column existence query to succeed for %s.%s, got %v",
				expectation.table,
				expectation.column,
				err,
			)
		}
		if !exists {
			t.Fatalf("expected column %s.%s to exist", expectation.table, expectation.column)
		}
	}
}

func TestRunMigrations_RepeatedExecution_IsIdempotent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db := openSQLiteTestDB(t, ctx)

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("expected first migration run to succeed, got %v", err)
	}
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("expected second migration run to succeed, got %v", err)
	}

	var migrationCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations;`).Scan(&migrationCount); err != nil {
		t.Fatalf("expected migration count query to succeed, got %v", err)
	}
	if migrationCount != len(schemaMigrations) {
		t.Fatalf("expected %d recorded migrations, got %d", len(schemaMigrations), migrationCount)
	}

	var systemStateVersion string
	if err := db.QueryRowContext(ctx, `SELECT state_value FROM system_state WHERE state_key = 'schema_version';`).Scan(&systemStateVersion); err != nil {
		t.Fatalf("expected system_state schema version query to succeed, got %v", err)
	}
	if systemStateVersion != "1" {
		t.Fatalf("expected system_state schema version %q, got %q", "1", systemStateVersion)
	}
}

func TestRunMigrations_WithNilDatabase_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := RunMigrations(ctx, nil)
	if err == nil {
		t.Fatalf("expected migration execution with nil database to fail, got nil")
	}
	if !strings.Contains(err.Error(), "sqlite database handle is required") {
		t.Fatalf("expected nil database error, got %v", err)
	}
}

func TestRunMigrations_EnforcesCriticalConstraints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db := openSQLiteTestDB(t, ctx)
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("expected migrations to succeed, got %v", err)
	}

	_, err := db.ExecContext(
		ctx,
		`INSERT INTO catalog_source_items (
			item_id,
			classifier,
			source_type,
			name,
			description,
			content,
			content_hash,
			content_writable,
			metadata_writable,
			last_synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		"invalid-classifier-item",
		"unknown",
		"git",
		"Invalid",
		"Invalid",
		"content",
		"hash",
		1,
		1,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err == nil {
		t.Fatalf("expected insert with invalid classifier to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "check constraint") {
		t.Fatalf("expected check constraint failure for classifier, got %v", err)
	}

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO catalog_metadata_overlays (item_id, custom_metadata_json, labels_json)
		VALUES (?, '{}', '[]');`,
		"missing-source-row",
	)
	if err == nil {
		t.Fatalf("expected overlay insert without source row to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("expected foreign key failure for overlay insert, got %v", err)
	}
}

func TestMigrationRunner_Run_WithBrokenStatement_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db := openSQLiteTestDB(t, ctx)
	runner := &MigrationRunner{
		db: db,
		migrations: []migration{
			{
				version: 1,
				name:    "broken_statement",
				statements: []string{
					"CREATE TABLE catalog_source_items (",
				},
			},
		},
	}

	err := runner.Run(ctx)
	if err == nil {
		t.Fatalf("expected broken migration statement to fail, got nil")
	}
	if !strings.Contains(err.Error(), "broken_statement") {
		t.Fatalf("expected migration error to include migration name, got %v", err)
	}
}

func TestMigrationRunner_CurrentVersion_WithNilContext_UsesBackgroundContext(t *testing.T) {
	db := openSQLiteTestDB(t, context.Background())
	runner := NewMigrationRunner(db)

	version, err := runner.CurrentVersion(nil)
	if err != nil {
		t.Fatalf("expected nil-context version query to succeed, got %v", err)
	}
	if version != 0 {
		t.Fatalf("expected initial schema version %d, got %d", 0, version)
	}
}

func TestMigrationRunner_Run_WithNilContext_UsesBackgroundContext(t *testing.T) {
	db := openSQLiteTestDB(t, context.Background())
	runner := NewMigrationRunner(db)

	if err := runner.Run(nil); err != nil {
		t.Fatalf("expected nil-context migration run to succeed, got %v", err)
	}
}

func TestQueryCurrentVersion_WithClosedDatabase_ReturnsError(t *testing.T) {
	db, err := sql.Open(sqliteDriverName, ":memory:")
	if err != nil {
		t.Fatalf("expected sqlite in-memory open to succeed, got %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("expected sqlite close to succeed, got %v", err)
	}

	_, err = queryCurrentVersion(context.Background(), db)
	if err == nil {
		t.Fatalf("expected current version query against closed DB to fail, got nil")
	}
}

type sqliteColumnExpectation struct {
	table  string
	column string
}

func openSQLiteTestDB(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()

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

	return db
}

func sqliteObjectExists(ctx context.Context, db *sql.DB, objectType, objectName string) (bool, error) {
	var count int
	err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		FROM sqlite_master
		WHERE type = ? AND name = ?;`,
		objectType,
		objectName,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func sqliteColumnExists(ctx context.Context, db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+tableName+`);`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, err
	}

	return false, nil
}
