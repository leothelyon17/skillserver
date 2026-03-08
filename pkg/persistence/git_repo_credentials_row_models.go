package persistence

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrGitRepoCredentialNotFound indicates that a git credential row does not exist.
	ErrGitRepoCredentialNotFound = errors.New("git repository credential row not found")
	// ErrGitRepoCredentialDecryptFailed indicates ciphertext decryption failed.
	ErrGitRepoCredentialDecryptFailed = errors.New("failed to decrypt git repository credential")
	// ErrGitRepoCredentialKeyMismatch indicates stored key metadata does not match the active key.
	ErrGitRepoCredentialKeyMismatch = errors.New("git repository credential key metadata mismatch")
)

// GitRepoCredentialRow mirrors one row in git_repo_credentials.
type GitRepoCredentialRow struct {
	RepoID     string
	KeyID      string
	KeyVersion int
	Ciphertext []byte
	Nonce      []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// GitRepoCredentialSecretType identifies the typed secret payload stored in ciphertext.
type GitRepoCredentialSecretType string

const (
	// GitRepoCredentialSecretTypeHTTPSToken stores token-based HTTPS auth secrets.
	GitRepoCredentialSecretTypeHTTPSToken GitRepoCredentialSecretType = "https_token"
	// GitRepoCredentialSecretTypeHTTPSBasic stores username/password HTTPS auth secrets.
	GitRepoCredentialSecretTypeHTTPSBasic GitRepoCredentialSecretType = "https_basic"
	// GitRepoCredentialSecretTypeSSHKey stores SSH private key auth secrets.
	GitRepoCredentialSecretTypeSSHKey GitRepoCredentialSecretType = "ssh_key"
)

// IsValid reports whether the secret payload type is supported.
func (t GitRepoCredentialSecretType) IsValid() bool {
	switch t {
	case GitRepoCredentialSecretTypeHTTPSToken,
		GitRepoCredentialSecretTypeHTTPSBasic,
		GitRepoCredentialSecretTypeSSHKey:
		return true
	default:
		return false
	}
}

// GitRepoCredentialSecretPayload defines typed secret material for supported auth modes.
type GitRepoCredentialSecretPayload struct {
	Type       GitRepoCredentialSecretType `json:"type"`
	Username   string                      `json:"username,omitempty"`
	Password   string                      `json:"password,omitempty"`
	Token      string                      `json:"token,omitempty"`
	PrivateKey string                      `json:"private_key,omitempty"`
	Passphrase string                      `json:"passphrase,omitempty"`
	KnownHosts string                      `json:"known_hosts,omitempty"`
}

func validateGitRepoCredentialUpsertRow(row GitRepoCredentialRow) (GitRepoCredentialRow, error) {
	repoID, err := normalizeRequiredID(row.RepoID, "git repo credential repo_id")
	if err != nil {
		return GitRepoCredentialRow{}, err
	}
	row.RepoID = repoID

	keyID, err := normalizeRequiredID(row.KeyID, "git repo credential key_id")
	if err != nil {
		return GitRepoCredentialRow{}, err
	}
	row.KeyID = keyID

	if row.KeyVersion <= 0 {
		return GitRepoCredentialRow{}, fmt.Errorf("git repo credential key_version must be greater than zero")
	}

	if len(row.Ciphertext) == 0 {
		return GitRepoCredentialRow{}, fmt.Errorf("git repo credential ciphertext is required")
	}
	row.Ciphertext = copyBytes(row.Ciphertext)

	if len(row.Nonce) == 0 {
		return GitRepoCredentialRow{}, fmt.Errorf("git repo credential nonce is required")
	}
	row.Nonce = copyBytes(row.Nonce)

	nowUTC := time.Now().UTC()
	if row.CreatedAt.IsZero() {
		row.CreatedAt = nowUTC
	} else {
		row.CreatedAt = row.CreatedAt.UTC()
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = row.CreatedAt
	} else {
		row.UpdatedAt = row.UpdatedAt.UTC()
	}
	if row.UpdatedAt.Before(row.CreatedAt) {
		return GitRepoCredentialRow{}, fmt.Errorf("git repo credential updated_at cannot be earlier than created_at")
	}

	return row, nil
}

func validateGitRepoCredentialSecretPayload(
	payload GitRepoCredentialSecretPayload,
) (GitRepoCredentialSecretPayload, error) {
	if !payload.Type.IsValid() {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf(
			"git repo credential payload type %q is invalid",
			payload.Type,
		)
	}

	normalized := payload
	normalized.Username = strings.TrimSpace(payload.Username)
	normalized.Password = payload.Password
	normalized.Token = payload.Token
	normalized.PrivateKey = payload.PrivateKey
	normalized.Passphrase = payload.Passphrase
	normalized.KnownHosts = payload.KnownHosts

	switch normalized.Type {
	case GitRepoCredentialSecretTypeHTTPSToken:
		if strings.TrimSpace(normalized.Token) == "" {
			return GitRepoCredentialSecretPayload{}, fmt.Errorf("git repo credential token is required for %q", normalized.Type)
		}
		normalized.Password = ""
		normalized.PrivateKey = ""
		normalized.Passphrase = ""
		normalized.KnownHosts = ""
	case GitRepoCredentialSecretTypeHTTPSBasic:
		if strings.TrimSpace(normalized.Username) == "" {
			return GitRepoCredentialSecretPayload{}, fmt.Errorf(
				"git repo credential username is required for %q",
				normalized.Type,
			)
		}
		if strings.TrimSpace(normalized.Password) == "" {
			return GitRepoCredentialSecretPayload{}, fmt.Errorf(
				"git repo credential password is required for %q",
				normalized.Type,
			)
		}
		normalized.Token = ""
		normalized.PrivateKey = ""
		normalized.Passphrase = ""
		normalized.KnownHosts = ""
	case GitRepoCredentialSecretTypeSSHKey:
		if strings.TrimSpace(normalized.PrivateKey) == "" {
			return GitRepoCredentialSecretPayload{}, fmt.Errorf(
				"git repo credential private_key is required for %q",
				normalized.Type,
			)
		}
		if strings.TrimSpace(normalized.KnownHosts) == "" {
			return GitRepoCredentialSecretPayload{}, fmt.Errorf(
				"git repo credential known_hosts is required for %q",
				normalized.Type,
			)
		}
		normalized.Token = ""
		normalized.Password = ""
		normalized.Username = ""
	default:
		return GitRepoCredentialSecretPayload{}, fmt.Errorf(
			"git repo credential payload type %q is unsupported",
			normalized.Type,
		)
	}

	return normalized, nil
}

func marshalGitRepoCredentialSecretPayload(payload GitRepoCredentialSecretPayload) ([]byte, error) {
	normalized, err := validateGitRepoCredentialSecretPayload(payload)
	if err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal git repo credential payload: %w", err)
	}

	return encoded, nil
}

func unmarshalGitRepoCredentialSecretPayload(raw []byte) (GitRepoCredentialSecretPayload, error) {
	if len(raw) == 0 {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf("git repo credential payload is empty")
	}

	var decoded GitRepoCredentialSecretPayload
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf("unmarshal git repo credential payload: %w", err)
	}

	normalized, err := validateGitRepoCredentialSecretPayload(decoded)
	if err != nil {
		return GitRepoCredentialSecretPayload{}, err
	}

	return normalized, nil
}

func scanGitRepoCredentialRow(scanner rowScanner) (GitRepoCredentialRow, error) {
	var (
		repoID       string
		keyID        string
		keyVersion   int
		ciphertext   []byte
		nonce        []byte
		createdAtRaw string
		updatedAtRaw string
	)

	if err := scanner.Scan(
		&repoID,
		&keyID,
		&keyVersion,
		&ciphertext,
		&nonce,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return GitRepoCredentialRow{}, err
	}

	createdAt, err := parseCatalogTimestamp(createdAtRaw)
	if err != nil {
		return GitRepoCredentialRow{}, fmt.Errorf("parse git repo credential created_at: %w", err)
	}
	updatedAt, err := parseCatalogTimestamp(updatedAtRaw)
	if err != nil {
		return GitRepoCredentialRow{}, fmt.Errorf("parse git repo credential updated_at: %w", err)
	}

	return GitRepoCredentialRow{
		RepoID:     repoID,
		KeyID:      keyID,
		KeyVersion: keyVersion,
		Ciphertext: copyBytes(ciphertext),
		Nonce:      copyBytes(nonce),
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}, nil
}

func copyBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}
	copied := make([]byte, len(value))
	copy(copied, value)
	return copied
}
