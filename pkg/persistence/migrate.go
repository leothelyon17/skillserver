package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"
)

type migration struct {
	version    int
	name       string
	statements []string
}

// MigrationRunner executes a linear chain of schema migrations.
type MigrationRunner struct {
	db         *sql.DB
	migrations []migration
}

var schemaMigrations = []migration{
	{
		version: 1,
		name:    "initial_catalog_persistence_schema",
		statements: []string{
			`CREATE TABLE IF NOT EXISTS catalog_source_items (
				item_id TEXT PRIMARY KEY,
				classifier TEXT NOT NULL CHECK (classifier IN ('skill', 'prompt')),
				source_type TEXT NOT NULL CHECK (source_type IN ('git', 'local', 'file_import')),
				source_repo TEXT,
				parent_skill_id TEXT,
				resource_path TEXT,
				name TEXT NOT NULL,
				description TEXT NOT NULL DEFAULT '',
				content TEXT NOT NULL DEFAULT '',
				content_hash TEXT NOT NULL,
				content_writable INTEGER NOT NULL CHECK (content_writable IN (0, 1)),
				metadata_writable INTEGER NOT NULL DEFAULT 1 CHECK (metadata_writable IN (0, 1)),
				last_synced_at TEXT NOT NULL,
				deleted_at TEXT
			);`,
			`CREATE INDEX IF NOT EXISTS idx_catalog_source_classifier_deleted_at
			ON catalog_source_items (classifier, deleted_at);`,
			`CREATE INDEX IF NOT EXISTS idx_catalog_source_source_filters
			ON catalog_source_items (source_type, source_repo, classifier, deleted_at);`,
			`CREATE INDEX IF NOT EXISTS idx_catalog_source_lookup_paths
			ON catalog_source_items (parent_skill_id, resource_path);`,
			`CREATE INDEX IF NOT EXISTS idx_catalog_source_resource_path
			ON catalog_source_items (resource_path);`,
			`CREATE TABLE IF NOT EXISTS catalog_metadata_overlays (
				item_id TEXT PRIMARY KEY,
				display_name_override TEXT,
				description_override TEXT,
				custom_metadata_json TEXT NOT NULL DEFAULT '{}',
				labels_json TEXT NOT NULL DEFAULT '[]',
				updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
				updated_by TEXT,
				FOREIGN KEY (item_id) REFERENCES catalog_source_items(item_id) ON UPDATE CASCADE ON DELETE CASCADE
			);`,
			`CREATE TABLE IF NOT EXISTS system_state (
				state_key TEXT PRIMARY KEY,
				state_value TEXT NOT NULL,
				updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			);`,
		},
	},
}

// NewMigrationRunner creates a migration runner for the provided sqlite handle.
func NewMigrationRunner(db *sql.DB) *MigrationRunner {
	copied := slices.Clone(schemaMigrations)
	return &MigrationRunner{
		db:         db,
		migrations: copied,
	}
}

// LatestSchemaVersion returns the highest available migration version.
func LatestSchemaVersion() int {
	if len(schemaMigrations) == 0 {
		return 0
	}
	return schemaMigrations[len(schemaMigrations)-1].version
}

// RunMigrations runs all pending migrations inside a single transaction.
func RunMigrations(ctx context.Context, db *sql.DB) error {
	runner := NewMigrationRunner(db)
	return runner.Run(ctx)
}

// CurrentVersion returns the latest applied schema migration version.
func (r *MigrationRunner) CurrentVersion(ctx context.Context) (int, error) {
	if r == nil {
		return 0, fmt.Errorf("migration runner is required")
	}
	if r.db == nil {
		return 0, fmt.Errorf("sqlite database handle is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := r.db.ExecContext(ctx, createSchemaMigrationsTableSQL); err != nil {
		return 0, fmt.Errorf("ensure schema migrations table: %w", err)
	}

	version, err := queryCurrentVersion(ctx, r.db)
	if err != nil {
		return 0, err
	}

	return version, nil
}

// Run applies any pending migrations and records the applied schema version.
func (r *MigrationRunner) Run(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("migration runner is required")
	}
	if r.db == nil {
		return fmt.Errorf("sqlite database handle is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start sqlite migration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, createSchemaMigrationsTableSQL); err != nil {
		return fmt.Errorf("ensure schema migrations table: %w", err)
	}

	appliedVersions, err := queryAppliedVersions(ctx, tx)
	if err != nil {
		return err
	}

	for _, nextMigration := range r.migrations {
		if _, alreadyApplied := appliedVersions[nextMigration.version]; alreadyApplied {
			continue
		}

		if err := applyMigrationStatements(ctx, tx, nextMigration); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, insertSchemaMigrationSQL, nextMigration.version, nextMigration.name, timestampNowUTC()); err != nil {
			return fmt.Errorf("record migration version %d: %w", nextMigration.version, err)
		}
		if _, err := tx.ExecContext(ctx, upsertSystemSchemaVersionSQL, strconv.Itoa(nextMigration.version), timestampNowUTC()); err != nil {
			return fmt.Errorf("update system_state schema version %d: %w", nextMigration.version, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sqlite migration transaction: %w", err)
	}

	return nil
}

func applyMigrationStatements(ctx context.Context, tx *sql.Tx, nextMigration migration) error {
	for statementIndex, statement := range nextMigration.statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf(
				"apply migration v%d (%s), statement %d: %w",
				nextMigration.version,
				nextMigration.name,
				statementIndex+1,
				err,
			)
		}
	}
	return nil
}

func queryAppliedVersions(ctx context.Context, tx *sql.Tx) (map[int]struct{}, error) {
	rows, err := tx.QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY version;`)
	if err != nil {
		return nil, fmt.Errorf("query applied sqlite migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]struct{})
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied sqlite migration version: %w", err)
		}
		applied[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied sqlite migration versions: %w", err)
	}

	return applied, nil
}

func queryCurrentVersion(ctx context.Context, db *sql.DB) (int, error) {
	var currentVersion sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations;`).Scan(&currentVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("query current sqlite schema version: %w", err)
	}
	if !currentVersion.Valid {
		return 0, nil
	}

	return int(currentVersion.Int64), nil
}

func timestampNowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

const (
	createSchemaMigrationsTableSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	);`

	insertSchemaMigrationSQL = `INSERT INTO schema_migrations (version, name, applied_at)
	VALUES (?, ?, ?);`

	upsertSystemSchemaVersionSQL = `INSERT INTO system_state (state_key, state_value, updated_at)
	VALUES ('schema_version', ?, ?)
	ON CONFLICT(state_key) DO UPDATE SET
		state_value = excluded.state_value,
		updated_at = excluded.updated_at;`
)
