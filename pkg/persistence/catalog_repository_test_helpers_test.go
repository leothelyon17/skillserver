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

func newCatalogDomainRepositoryForTest(t *testing.T, db *sql.DB) *CatalogDomainRepository {
	t.Helper()

	repo, err := NewCatalogDomainRepository(db)
	if err != nil {
		t.Fatalf("expected catalog domain repository creation to succeed, got %v", err)
	}

	return repo
}

func newCatalogSubdomainRepositoryForTest(t *testing.T, db *sql.DB) *CatalogSubdomainRepository {
	t.Helper()

	repo, err := NewCatalogSubdomainRepository(db)
	if err != nil {
		t.Fatalf("expected catalog subdomain repository creation to succeed, got %v", err)
	}

	return repo
}

func newCatalogTagRepositoryForTest(t *testing.T, db *sql.DB) *CatalogTagRepository {
	t.Helper()

	repo, err := NewCatalogTagRepository(db)
	if err != nil {
		t.Fatalf("expected catalog tag repository creation to succeed, got %v", err)
	}

	return repo
}

func newCatalogItemTaxonomyAssignmentRepositoryForTest(
	t *testing.T,
	db *sql.DB,
) *CatalogItemTaxonomyAssignmentRepository {
	t.Helper()

	repo, err := NewCatalogItemTaxonomyAssignmentRepository(db)
	if err != nil {
		t.Fatalf("expected catalog item taxonomy assignment repository creation to succeed, got %v", err)
	}

	return repo
}

func newCatalogItemTagAssignmentRepositoryForTest(
	t *testing.T,
	db *sql.DB,
) *CatalogItemTagAssignmentRepository {
	t.Helper()

	repo, err := NewCatalogItemTagAssignmentRepository(db)
	if err != nil {
		t.Fatalf("expected catalog item tag assignment repository creation to succeed, got %v", err)
	}

	return repo
}

func newGitRepoCredentialCipherForTest(t *testing.T, masterKey string) *GitRepoCredentialCipher {
	t.Helper()

	cipher, err := NewGitRepoCredentialCipher(masterKey, GitRepoCredentialCipherOptions{})
	if err != nil {
		t.Fatalf("expected git repo credential cipher creation to succeed, got %v", err)
	}

	return cipher
}

func newGitRepoCredentialRepositoryForTest(
	t *testing.T,
	db *sql.DB,
	cipher *GitRepoCredentialCipher,
) *GitRepoCredentialRepository {
	t.Helper()

	repo, err := NewGitRepoCredentialRepository(db, cipher)
	if err != nil {
		t.Fatalf("expected git repo credential repository creation to succeed, got %v", err)
	}

	return repo
}

func stringPointer(value string) *string {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func mustUpsertCatalogSourceRow(t *testing.T, ctx context.Context, repo *CatalogSourceRepository, row CatalogSourceRow) {
	t.Helper()

	if err := repo.Upsert(ctx, row); err != nil {
		t.Fatalf("expected catalog source upsert to succeed, got %v", err)
	}
}

func mustCreateCatalogDomainRow(t *testing.T, ctx context.Context, repo *CatalogDomainRepository, row CatalogDomainRow) {
	t.Helper()

	if err := repo.Create(ctx, row); err != nil {
		t.Fatalf("expected catalog domain create to succeed, got %v", err)
	}
}

func mustCreateCatalogSubdomainRow(
	t *testing.T,
	ctx context.Context,
	repo *CatalogSubdomainRepository,
	row CatalogSubdomainRow,
) {
	t.Helper()

	if err := repo.Create(ctx, row); err != nil {
		t.Fatalf("expected catalog subdomain create to succeed, got %v", err)
	}
}

func mustCreateCatalogTagRow(t *testing.T, ctx context.Context, repo *CatalogTagRepository, row CatalogTagRow) {
	t.Helper()

	if err := repo.Create(ctx, row); err != nil {
		t.Fatalf("expected catalog tag create to succeed, got %v", err)
	}
}
