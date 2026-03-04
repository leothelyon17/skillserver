package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func openMigratedSQLiteRepositoryDB(t *testing.T) (*sql.DB, context.Context) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	db := openSQLiteTestDB(t, ctx)
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("expected migrations to succeed, got %v", err)
	}

	return db, ctx
}

func newCatalogSourceRepositoryForTest(t *testing.T, db *sql.DB) *CatalogSourceRepository {
	t.Helper()

	repo, err := NewCatalogSourceRepository(db)
	if err != nil {
		t.Fatalf("expected catalog source repository creation to succeed, got %v", err)
	}

	return repo
}

func newCatalogMetadataOverlayRepositoryForTest(
	t *testing.T,
	db *sql.DB,
) *CatalogMetadataOverlayRepository {
	t.Helper()

	repo, err := NewCatalogMetadataOverlayRepository(db)
	if err != nil {
		t.Fatalf("expected catalog metadata overlay repository creation to succeed, got %v", err)
	}

	return repo
}

func stringPointer(value string) *string {
	return &value
}

func mustUpsertCatalogSourceRow(t *testing.T, ctx context.Context, repo *CatalogSourceRepository, row CatalogSourceRow) {
	t.Helper()

	if err := repo.Upsert(ctx, row); err != nil {
		t.Fatalf("expected catalog source upsert to succeed, got %v", err)
	}
}
