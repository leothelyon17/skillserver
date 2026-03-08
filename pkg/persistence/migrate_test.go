package persistence

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"strconv"
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
		"catalog_domains",
		"catalog_subdomains",
		"catalog_tags",
		"catalog_item_taxonomy_assignments",
		"catalog_item_tag_assignments",
		"git_repo_credentials",
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
		"idx_catalog_subdomains_domain_id",
		"idx_catalog_item_taxonomy_primary_domain",
		"idx_catalog_item_taxonomy_secondary_domain",
		"idx_catalog_item_taxonomy_primary_subdomain",
		"idx_catalog_item_taxonomy_secondary_subdomain",
		"idx_catalog_item_tag_assignments_tag",
		"idx_git_repo_credentials_key_metadata",
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
		{table: "catalog_domains", column: "key"},
		{table: "catalog_subdomains", column: "domain_id"},
		{table: "catalog_tags", column: "color"},
		{table: "catalog_item_taxonomy_assignments", column: "secondary_subdomain_id"},
		{table: "catalog_item_tag_assignments", column: "created_at"},
		{table: "git_repo_credentials", column: "key_id"},
		{table: "git_repo_credentials", column: "key_version"},
		{table: "git_repo_credentials", column: "ciphertext"},
		{table: "git_repo_credentials", column: "nonce"},
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
	expectedVersion := strconv.Itoa(LatestSchemaVersion())
	if systemStateVersion != expectedVersion {
		t.Fatalf("expected system_state schema version %q, got %q", expectedVersion, systemStateVersion)
	}
}

func TestRunMigrations_UpgradeFromVersionOneToLatest_AppliesPostV1Schema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db := openSQLiteTestDB(t, ctx)

	if len(schemaMigrations) < 2 {
		t.Fatalf("expected at least two schema migrations, got %d", len(schemaMigrations))
	}

	v1Runner := &MigrationRunner{
		db:         db,
		migrations: []migration{schemaMigrations[0]},
	}
	if err := v1Runner.Run(ctx); err != nil {
		t.Fatalf("expected v1 migration run to succeed, got %v", err)
	}

	v1Version, err := v1Runner.CurrentVersion(ctx)
	if err != nil {
		t.Fatalf("expected v1 schema version query to succeed, got %v", err)
	}
	if v1Version != 1 {
		t.Fatalf("expected schema version %d after v1 run, got %d", 1, v1Version)
	}

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("expected upgrade migration run to succeed, got %v", err)
	}

	upgradedVersion, err := NewMigrationRunner(db).CurrentVersion(ctx)
	if err != nil {
		t.Fatalf("expected upgraded schema version query to succeed, got %v", err)
	}
	if upgradedVersion != LatestSchemaVersion() {
		t.Fatalf("expected schema version %d after upgrade, got %d", LatestSchemaVersion(), upgradedVersion)
	}

	requiredPostV1Tables := []string{
		"catalog_domains",
		"catalog_subdomains",
		"catalog_tags",
		"catalog_item_taxonomy_assignments",
		"catalog_item_tag_assignments",
		"git_repo_credentials",
	}
	for _, table := range requiredPostV1Tables {
		exists, err := sqliteObjectExists(ctx, db, "table", table)
		if err != nil {
			t.Fatalf("expected table existence query to succeed for %q, got %v", table, err)
		}
		if !exists {
			t.Fatalf("expected post-v1 table %q to exist", table)
		}
	}
}

func TestRunMigrations_UpgradeFromVersionTwoToLatest_AppliesGitCredentialSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db := openSQLiteTestDB(t, ctx)

	if len(schemaMigrations) < 3 {
		t.Fatalf("expected at least three schema migrations, got %d", len(schemaMigrations))
	}

	v2Runner := &MigrationRunner{
		db:         db,
		migrations: slices.Clone(schemaMigrations[:2]),
	}
	if err := v2Runner.Run(ctx); err != nil {
		t.Fatalf("expected v2 migration run to succeed, got %v", err)
	}

	v2Version, err := v2Runner.CurrentVersion(ctx)
	if err != nil {
		t.Fatalf("expected v2 schema version query to succeed, got %v", err)
	}
	if v2Version != 2 {
		t.Fatalf("expected schema version %d after v2 run, got %d", 2, v2Version)
	}

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("expected migration upgrade from v2 to latest to succeed, got %v", err)
	}

	latestVersion, err := NewMigrationRunner(db).CurrentVersion(ctx)
	if err != nil {
		t.Fatalf("expected latest schema version query to succeed, got %v", err)
	}
	if latestVersion != LatestSchemaVersion() {
		t.Fatalf("expected upgraded schema version %d, got %d", LatestSchemaVersion(), latestVersion)
	}

	credentialTableExists, err := sqliteObjectExists(ctx, db, "table", "git_repo_credentials")
	if err != nil {
		t.Fatalf("expected git_repo_credentials table existence query to succeed, got %v", err)
	}
	if !credentialTableExists {
		t.Fatalf("expected git_repo_credentials table to exist after v2 upgrade")
	}

	keyMetadataIndexExists, err := sqliteObjectExists(ctx, db, "index", "idx_git_repo_credentials_key_metadata")
	if err != nil {
		t.Fatalf("expected key metadata index existence query to succeed, got %v", err)
	}
	if !keyMetadataIndexExists {
		t.Fatalf("expected idx_git_repo_credentials_key_metadata index to exist after v2 upgrade")
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

	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err = db.ExecContext(
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
		"skill:taxonomy-item",
		"skill",
		"local",
		"taxonomy-item",
		"taxonomy item",
		"content",
		"hash-taxonomy",
		1,
		1,
		now,
	)
	if err != nil {
		t.Fatalf("expected source insert for taxonomy fixture to succeed, got %v", err)
	}

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO catalog_domains (domain_id, key, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?);`,
		"domain-1",
		"platform",
		"Platform",
		now,
		now,
		"domain-2",
		"operations",
		"Operations",
		now,
		now,
	)
	if err != nil {
		t.Fatalf("expected taxonomy domain inserts to succeed, got %v", err)
	}

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO catalog_subdomains (subdomain_id, domain_id, key, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?);`,
		"subdomain-1",
		"domain-1",
		"orchestration",
		"Orchestration",
		now,
		now,
	)
	if err != nil {
		t.Fatalf("expected taxonomy subdomain insert to succeed, got %v", err)
	}

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO catalog_tags (tag_id, key, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?);`,
		"tag-1",
		"python",
		"Python",
		now,
		now,
	)
	if err != nil {
		t.Fatalf("expected taxonomy tag insert to succeed, got %v", err)
	}

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO catalog_item_taxonomy_assignments (
			item_id,
			primary_domain_id,
			primary_subdomain_id,
			secondary_domain_id,
			updated_at
		) VALUES (?, ?, ?, ?, ?);`,
		"skill:taxonomy-item",
		"domain-1",
		"subdomain-1",
		"domain-2",
		now,
	)
	if err != nil {
		t.Fatalf("expected taxonomy assignment insert to succeed, got %v", err)
	}

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO catalog_item_tag_assignments (item_id, tag_id, created_at)
		VALUES (?, ?, ?);`,
		"skill:taxonomy-item",
		"tag-1",
		now,
	)
	if err != nil {
		t.Fatalf("expected taxonomy tag assignment insert to succeed, got %v", err)
	}

	_, err = db.ExecContext(ctx, `DELETE FROM catalog_domains WHERE domain_id = ?;`, "domain-2")
	if err == nil {
		t.Fatalf("expected delete of assigned secondary domain to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("expected foreign key failure for assigned secondary domain delete, got %v", err)
	}

	_, err = db.ExecContext(ctx, `DELETE FROM catalog_subdomains WHERE subdomain_id = ?;`, "subdomain-1")
	if err == nil {
		t.Fatalf("expected delete of assigned subdomain to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("expected foreign key failure for assigned subdomain delete, got %v", err)
	}

	_, err = db.ExecContext(ctx, `DELETE FROM catalog_tags WHERE tag_id = ?;`, "tag-1")
	if err == nil {
		t.Fatalf("expected delete of assigned tag to fail, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("expected foreign key failure for assigned tag delete, got %v", err)
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM catalog_source_items WHERE item_id = ?;`, "skill:taxonomy-item"); err != nil {
		t.Fatalf("expected source delete to succeed, got %v", err)
	}

	var taxonomyAssignmentCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_item_taxonomy_assignments WHERE item_id = ?;`, "skill:taxonomy-item").Scan(&taxonomyAssignmentCount); err != nil {
		t.Fatalf("expected taxonomy assignment count query to succeed, got %v", err)
	}
	if taxonomyAssignmentCount != 0 {
		t.Fatalf("expected taxonomy assignments to cascade-delete with source item, got %d", taxonomyAssignmentCount)
	}

	var tagAssignmentCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_item_tag_assignments WHERE item_id = ?;`, "skill:taxonomy-item").Scan(&tagAssignmentCount); err != nil {
		t.Fatalf("expected tag assignment count query to succeed, got %v", err)
	}
	if tagAssignmentCount != 0 {
		t.Fatalf("expected tag assignments to cascade-delete with source item, got %d", tagAssignmentCount)
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM catalog_subdomains WHERE subdomain_id = ?;`, "subdomain-1"); err != nil {
		t.Fatalf("expected subdomain delete after unassignment to succeed, got %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM catalog_domains WHERE domain_id = ?;`, "domain-1"); err != nil {
		t.Fatalf("expected primary domain delete after unassignment to succeed, got %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM catalog_domains WHERE domain_id = ?;`, "domain-2"); err != nil {
		t.Fatalf("expected secondary domain delete after unassignment to succeed, got %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM catalog_tags WHERE tag_id = ?;`, "tag-1"); err != nil {
		t.Fatalf("expected tag delete after unassignment to succeed, got %v", err)
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
