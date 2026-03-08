package persistence

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewGitRepoCredentialCipher_WithMissingMasterKey_ReturnsError(t *testing.T) {
	_, err := NewGitRepoCredentialCipher("", GitRepoCredentialCipherOptions{})
	if err == nil {
		t.Fatalf("expected missing master key error, got nil")
	}
}

func TestNewGitRepoCredentialCipher_WithInvalidEncodedMasterKey_ReturnsError(t *testing.T) {
	_, err := NewGitRepoCredentialCipher("base64:not-valid-base64", GitRepoCredentialCipherOptions{})
	if err == nil {
		t.Fatalf("expected invalid encoded master key error, got nil")
	}
}

func TestGitRepoCredentialCipher_EncryptDecrypt_AllPayloadTypes(t *testing.T) {
	cipher := newGitRepoCredentialCipherForTest(t, "primary-master-key-value-123")

	tests := []struct {
		name    string
		payload GitRepoCredentialSecretPayload
	}{
		{
			name: "https_token",
			payload: GitRepoCredentialSecretPayload{
				Type:     GitRepoCredentialSecretTypeHTTPSToken,
				Username: "git",
				Token:    "token-secret-value",
			},
		},
		{
			name: "https_basic",
			payload: GitRepoCredentialSecretPayload{
				Type:     GitRepoCredentialSecretTypeHTTPSBasic,
				Username: "svc-user",
				Password: "svc-password",
			},
		},
		{
			name: "ssh_key",
			payload: GitRepoCredentialSecretPayload{
				Type:       GitRepoCredentialSecretTypeSSHKey,
				PrivateKey: "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----",
				Passphrase: "ssh-passphrase",
				KnownHosts: "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := cipher.Encrypt("gitrepo_test", tc.payload)
			if err != nil {
				t.Fatalf("expected encrypt to succeed, got %v", err)
			}
			if len(encrypted.Ciphertext) == 0 {
				t.Fatalf("expected ciphertext bytes, got empty slice")
			}
			if len(encrypted.Nonce) == 0 {
				t.Fatalf("expected nonce bytes, got empty slice")
			}

			decrypted, err := cipher.Decrypt(GitRepoCredentialRow{
				RepoID:     "gitrepo_test",
				KeyID:      encrypted.KeyID,
				KeyVersion: encrypted.KeyVersion,
				Ciphertext: encrypted.Ciphertext,
				Nonce:      encrypted.Nonce,
			})
			if err != nil {
				t.Fatalf("expected decrypt to succeed, got %v", err)
			}
			if decrypted != tc.payload {
				t.Fatalf("expected decrypted payload %+v, got %+v", tc.payload, decrypted)
			}
		})
	}
}

func TestGitRepoCredentialRepository_CreateReplaceReadDelete(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	cipher := newGitRepoCredentialCipherForTest(t, "primary-master-key-value-123")
	repo := newGitRepoCredentialRepositoryForTest(t, db, cipher)

	repoID := "gitrepo_private_skills"
	createdAt := time.Date(2026, time.March, 7, 20, 0, 0, 0, time.UTC)
	initialPayload := GitRepoCredentialSecretPayload{
		Type:     GitRepoCredentialSecretTypeHTTPSToken,
		Username: "git",
		Token:    "very-secret-token",
	}

	if err := repo.CreateCredential(ctx, repoID, initialPayload, createdAt); err != nil {
		t.Fatalf("expected create credential to succeed, got %v", err)
	}

	storedRow, err := repo.GetEncryptedByRepoID(ctx, repoID)
	if err != nil {
		t.Fatalf("expected encrypted row lookup to succeed, got %v", err)
	}
	if storedRow.KeyID == "" {
		t.Fatalf("expected key_id metadata to be set")
	}
	if storedRow.KeyVersion <= 0 {
		t.Fatalf("expected key_version metadata to be positive, got %d", storedRow.KeyVersion)
	}
	if !storedRow.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created_at %s, got %s", createdAt, storedRow.CreatedAt)
	}
	if !storedRow.UpdatedAt.Equal(createdAt) {
		t.Fatalf("expected updated_at %s, got %s", createdAt, storedRow.UpdatedAt)
	}
	if bytes.Contains(storedRow.Ciphertext, []byte(initialPayload.Token)) {
		t.Fatalf("expected ciphertext to omit plaintext token bytes")
	}

	var rawCiphertext, rawNonce []byte
	if err := db.QueryRowContext(
		ctx,
		`SELECT ciphertext, nonce FROM git_repo_credentials WHERE repo_id = ?;`,
		repoID,
	).Scan(&rawCiphertext, &rawNonce); err != nil {
		t.Fatalf("expected raw row query to succeed, got %v", err)
	}
	if bytes.Contains(rawCiphertext, []byte(initialPayload.Token)) {
		t.Fatalf("expected persisted ciphertext to omit plaintext token bytes")
	}
	if bytes.Contains(rawNonce, []byte(initialPayload.Token)) {
		t.Fatalf("expected persisted nonce to omit plaintext token bytes")
	}

	decryptedInitial, err := repo.GetCredential(ctx, repoID)
	if err != nil {
		t.Fatalf("expected credential read to succeed, got %v", err)
	}
	if decryptedInitial != initialPayload {
		t.Fatalf("expected decrypted initial payload %+v, got %+v", initialPayload, decryptedInitial)
	}

	replacedAt := createdAt.Add(2 * time.Hour)
	replacedPayload := GitRepoCredentialSecretPayload{
		Type:       GitRepoCredentialSecretTypeSSHKey,
		PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nxyz\n-----END OPENSSH PRIVATE KEY-----",
		Passphrase: "new-passphrase",
		KnownHosts: "gitlab.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI",
	}
	if err := repo.ReplaceCredential(ctx, repoID, replacedPayload, replacedAt); err != nil {
		t.Fatalf("expected replace credential to succeed, got %v", err)
	}

	replacedRow, err := repo.GetEncryptedByRepoID(ctx, repoID)
	if err != nil {
		t.Fatalf("expected replaced encrypted row lookup to succeed, got %v", err)
	}
	if !replacedRow.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created_at to stay stable after replace, got %s want %s", replacedRow.CreatedAt, createdAt)
	}
	if !replacedRow.UpdatedAt.Equal(replacedAt) {
		t.Fatalf("expected updated_at to be replaced to %s, got %s", replacedAt, replacedRow.UpdatedAt)
	}
	if bytes.Contains(replacedRow.Ciphertext, []byte(replacedPayload.PrivateKey)) {
		t.Fatalf("expected replaced ciphertext to omit plaintext private key bytes")
	}

	decryptedReplaced, err := repo.GetCredential(ctx, repoID)
	if err != nil {
		t.Fatalf("expected replaced credential read to succeed, got %v", err)
	}
	if decryptedReplaced != replacedPayload {
		t.Fatalf("expected replaced payload %+v, got %+v", replacedPayload, decryptedReplaced)
	}

	deleted, err := repo.DeleteByRepoID(ctx, repoID)
	if err != nil {
		t.Fatalf("expected delete credential to succeed, got %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete to report affected row")
	}

	deletedAgain, err := repo.DeleteByRepoID(ctx, repoID)
	if err != nil {
		t.Fatalf("expected second delete call to succeed, got %v", err)
	}
	if deletedAgain {
		t.Fatalf("expected second delete to report no affected row")
	}

	_, err = repo.GetCredential(ctx, repoID)
	if !errors.Is(err, ErrGitRepoCredentialNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestGitRepoCredentialRepository_GetCredential_WithWrongMasterKey_ReturnsDecryptFailure(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)

	encryptingCipher := newGitRepoCredentialCipherForTest(t, "master-key-one-value")
	encryptingRepo := newGitRepoCredentialRepositoryForTest(t, db, encryptingCipher)
	if err := encryptingRepo.CreateCredential(
		ctx,
		"gitrepo_wrong_key",
		GitRepoCredentialSecretPayload{
			Type:     GitRepoCredentialSecretTypeHTTPSToken,
			Username: "git",
			Token:    "top-secret-token",
		},
		time.Date(2026, time.March, 7, 22, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("expected create with first key to succeed, got %v", err)
	}

	decryptingCipher := newGitRepoCredentialCipherForTest(t, "master-key-two-value")
	decryptingRepo := newGitRepoCredentialRepositoryForTest(t, db, decryptingCipher)

	_, err := decryptingRepo.GetCredential(ctx, "gitrepo_wrong_key")
	if err == nil {
		t.Fatalf("expected decrypt with wrong key to fail, got nil")
	}
	if !errors.Is(err, ErrGitRepoCredentialDecryptFailed) {
		t.Fatalf("expected decrypt failure sentinel error, got %v", err)
	}
}

func TestGitRepoCredentialRepository_GetCredential_WithKeyMetadataMismatch_ReturnsKeyMismatch(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)

	encryptingCipher := newGitRepoCredentialCipherForTest(t, "same-master-key-value")
	encryptingRepo := newGitRepoCredentialRepositoryForTest(t, db, encryptingCipher)
	if err := encryptingRepo.CreateCredential(
		ctx,
		"gitrepo_wrong_metadata",
		GitRepoCredentialSecretPayload{
			Type:     GitRepoCredentialSecretTypeHTTPSBasic,
			Username: "svc-user",
			Password: "svc-password",
		},
		time.Date(2026, time.March, 7, 23, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("expected create with default key metadata to succeed, got %v", err)
	}

	rotatedMetadataCipher, err := NewGitRepoCredentialCipher(
		"same-master-key-value",
		GitRepoCredentialCipherOptions{
			KeyID:      "git-credential-master-rotated",
			KeyVersion: 2,
		},
	)
	if err != nil {
		t.Fatalf("expected rotated metadata cipher creation to succeed, got %v", err)
	}
	rotatedRepo := newGitRepoCredentialRepositoryForTest(t, db, rotatedMetadataCipher)

	_, err = rotatedRepo.GetCredential(ctx, "gitrepo_wrong_metadata")
	if err == nil {
		t.Fatalf("expected key metadata mismatch to fail, got nil")
	}
	if !errors.Is(err, ErrGitRepoCredentialKeyMismatch) {
		t.Fatalf("expected key mismatch sentinel error, got %v", err)
	}
}

func TestGitRepoCredentialRepository_GetCredential_NotFound(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	cipher := newGitRepoCredentialCipherForTest(t, "primary-master-key-value-123")
	repo := newGitRepoCredentialRepositoryForTest(t, db, cipher)

	_, err := repo.GetCredential(ctx, "gitrepo_missing")
	if !errors.Is(err, ErrGitRepoCredentialNotFound) {
		t.Fatalf("expected missing credential to return not found, got %v", err)
	}
}

func TestGitRepoCredentialRepository_CreateCredential_DuplicateRepoIDFails(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	cipher := newGitRepoCredentialCipherForTest(t, "primary-master-key-value-123")
	repo := newGitRepoCredentialRepositoryForTest(t, db, cipher)

	createdAt := time.Date(2026, time.March, 8, 0, 0, 0, 0, time.UTC)
	if err := repo.CreateCredential(
		ctx,
		"gitrepo_duplicate",
		GitRepoCredentialSecretPayload{
			Type:     GitRepoCredentialSecretTypeHTTPSToken,
			Username: "git",
			Token:    "first-token",
		},
		createdAt,
	); err != nil {
		t.Fatalf("expected first create call to succeed, got %v", err)
	}

	err := repo.CreateCredential(
		ctx,
		"gitrepo_duplicate",
		GitRepoCredentialSecretPayload{
			Type:     GitRepoCredentialSecretTypeHTTPSToken,
			Username: "git",
			Token:    "second-token",
		},
		createdAt.Add(1*time.Hour),
	)
	if err == nil {
		t.Fatalf("expected duplicate create to fail, got nil")
	}
}

func TestGitRepoCredentialRepository_CreateCredential_WithInvalidPayload_ReturnsError(t *testing.T) {
	db, ctx := openMigratedSQLiteRepositoryDB(t)
	cipher := newGitRepoCredentialCipherForTest(t, "primary-master-key-value-123")
	repo := newGitRepoCredentialRepositoryForTest(t, db, cipher)

	err := repo.CreateCredential(
		ctx,
		"gitrepo_invalid_payload",
		GitRepoCredentialSecretPayload{
			Type: GitRepoCredentialSecretTypeSSHKey,
			// Missing known_hosts should fail payload validation.
			PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----",
		},
		time.Now().UTC(),
	)
	if err == nil {
		t.Fatalf("expected invalid payload create to fail, got nil")
	}
}

func TestGitRepoCredentialRepository_CreateCredential_WithNilContext_UsesBackground(t *testing.T) {
	db, _ := openMigratedSQLiteRepositoryDB(t)
	cipher := newGitRepoCredentialCipherForTest(t, "primary-master-key-value-123")
	repo := newGitRepoCredentialRepositoryForTest(t, db, cipher)

	err := repo.CreateCredential(
		nil,
		"gitrepo_nil_context",
		GitRepoCredentialSecretPayload{
			Type:     GitRepoCredentialSecretTypeHTTPSToken,
			Username: "git",
			Token:    "token",
		},
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("expected create with nil context to succeed, got %v", err)
	}
}

func TestGitRepoCredentialRepository_WithCanceledContext_PropagatesError(t *testing.T) {
	db, _ := openMigratedSQLiteRepositoryDB(t)
	cipher := newGitRepoCredentialCipherForTest(t, "primary-master-key-value-123")
	repo := newGitRepoCredentialRepositoryForTest(t, db, cipher)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.CreateCredential(
		ctx,
		"gitrepo_canceled_context",
		GitRepoCredentialSecretPayload{
			Type:     GitRepoCredentialSecretTypeHTTPSToken,
			Username: "git",
			Token:    "token",
		},
		time.Now().UTC(),
	)
	if err == nil {
		t.Fatalf("expected canceled context create to fail, got nil")
	}
}
