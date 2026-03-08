package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// GitRepoCredentialRepository provides encrypted credential storage keyed by git repo_id.
type GitRepoCredentialRepository struct {
	exec   catalogQueryExecutor
	cipher *GitRepoCredentialCipher
}

// NewGitRepoCredentialRepository creates a git credential repository around a DB or transaction handle.
func NewGitRepoCredentialRepository(
	exec catalogQueryExecutor,
	cipher *GitRepoCredentialCipher,
) (*GitRepoCredentialRepository, error) {
	if exec == nil {
		return nil, fmt.Errorf("git repo credential repository query executor is required")
	}
	if cipher == nil {
		return nil, fmt.Errorf("git repo credential cipher is required")
	}

	return &GitRepoCredentialRepository{
		exec:   exec,
		cipher: cipher,
	}, nil
}

// CreateCredential inserts a new encrypted credential row for one repo_id.
func (r *GitRepoCredentialRepository) CreateCredential(
	ctx context.Context,
	repoID string,
	payload GitRepoCredentialSecretPayload,
	createdAt time.Time,
) error {
	if r == nil {
		return fmt.Errorf("git repo credential repository is required")
	}

	row, err := r.encryptForWrite(repoID, payload, createdAt)
	if err != nil {
		return err
	}

	_, err = r.exec.ExecContext(
		normalizeContext(ctx),
		`INSERT INTO git_repo_credentials (
			repo_id,
			key_id,
			key_version,
			ciphertext,
			nonce,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?);`,
		row.RepoID,
		row.KeyID,
		row.KeyVersion,
		row.Ciphertext,
		row.Nonce,
		formatCatalogTimestamp(row.CreatedAt),
		formatCatalogTimestamp(row.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("create git repo credential row %q: %w", row.RepoID, err)
	}

	return nil
}

// ReplaceCredential upserts encrypted credentials for one repo_id.
func (r *GitRepoCredentialRepository) ReplaceCredential(
	ctx context.Context,
	repoID string,
	payload GitRepoCredentialSecretPayload,
	updatedAt time.Time,
) error {
	if r == nil {
		return fmt.Errorf("git repo credential repository is required")
	}

	row, err := r.encryptForWrite(repoID, payload, updatedAt)
	if err != nil {
		return err
	}

	_, err = r.exec.ExecContext(
		normalizeContext(ctx),
		`INSERT INTO git_repo_credentials (
			repo_id,
			key_id,
			key_version,
			ciphertext,
			nonce,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_id) DO UPDATE SET
			key_id = excluded.key_id,
			key_version = excluded.key_version,
			ciphertext = excluded.ciphertext,
			nonce = excluded.nonce,
			updated_at = excluded.updated_at;`,
		row.RepoID,
		row.KeyID,
		row.KeyVersion,
		row.Ciphertext,
		row.Nonce,
		formatCatalogTimestamp(row.CreatedAt),
		formatCatalogTimestamp(row.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("replace git repo credential row %q: %w", row.RepoID, err)
	}

	return nil
}

// GetCredential decrypts and returns one repo's typed secret payload.
func (r *GitRepoCredentialRepository) GetCredential(
	ctx context.Context,
	repoID string,
) (GitRepoCredentialSecretPayload, error) {
	if r == nil {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf("git repo credential repository is required")
	}

	row, err := r.GetEncryptedByRepoID(ctx, repoID)
	if err != nil {
		return GitRepoCredentialSecretPayload{}, err
	}

	payload, err := r.cipher.Decrypt(row)
	if err != nil {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf(
			"decrypt git repo credential row %q: %w",
			row.RepoID,
			err,
		)
	}

	return payload, nil
}

// GetEncryptedByRepoID fetches one encrypted credential row by repo_id.
func (r *GitRepoCredentialRepository) GetEncryptedByRepoID(
	ctx context.Context,
	repoID string,
) (GitRepoCredentialRow, error) {
	if r == nil {
		return GitRepoCredentialRow{}, fmt.Errorf("git repo credential repository is required")
	}

	normalizedRepoID, err := normalizeRequiredID(repoID, "git repo credential repo_id")
	if err != nil {
		return GitRepoCredentialRow{}, err
	}

	row, err := scanGitRepoCredentialRow(
		r.exec.QueryRowContext(
			normalizeContext(ctx),
			`SELECT
				repo_id,
				key_id,
				key_version,
				ciphertext,
				nonce,
				created_at,
				updated_at
			FROM git_repo_credentials
			WHERE repo_id = ?;`,
			normalizedRepoID,
		),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GitRepoCredentialRow{}, ErrGitRepoCredentialNotFound
		}
		return GitRepoCredentialRow{}, fmt.Errorf("get git repo credential row %q: %w", normalizedRepoID, err)
	}

	return row, nil
}

// DeleteByRepoID deletes one encrypted credential row by repo_id.
func (r *GitRepoCredentialRepository) DeleteByRepoID(ctx context.Context, repoID string) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("git repo credential repository is required")
	}

	normalizedRepoID, err := normalizeRequiredID(repoID, "git repo credential repo_id")
	if err != nil {
		return false, err
	}

	result, err := r.exec.ExecContext(
		normalizeContext(ctx),
		`DELETE FROM git_repo_credentials WHERE repo_id = ?;`,
		normalizedRepoID,
	)
	if err != nil {
		return false, fmt.Errorf("delete git repo credential row %q: %w", normalizedRepoID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read delete affected rows for git repo credential %q: %w", normalizedRepoID, err)
	}

	return rowsAffected > 0, nil
}

func (r *GitRepoCredentialRepository) encryptForWrite(
	repoID string,
	payload GitRepoCredentialSecretPayload,
	timestamp time.Time,
) (GitRepoCredentialRow, error) {
	encrypted, err := r.cipher.Encrypt(repoID, payload)
	if err != nil {
		return GitRepoCredentialRow{}, err
	}

	nowUTC := timestamp
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	} else {
		nowUTC = nowUTC.UTC()
	}

	return validateGitRepoCredentialUpsertRow(GitRepoCredentialRow{
		RepoID:     repoID,
		KeyID:      encrypted.KeyID,
		KeyVersion: encrypted.KeyVersion,
		Ciphertext: encrypted.Ciphertext,
		Nonce:      encrypted.Nonce,
		CreatedAt:  nowUTC,
		UpdatedAt:  nowUTC,
	})
}
