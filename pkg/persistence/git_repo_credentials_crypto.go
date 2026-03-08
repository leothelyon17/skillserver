package persistence

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const (
	defaultGitRepoCredentialKeyID          = "git-credential-master"
	defaultGitRepoCredentialKeyVersion     = 1
	gitRepoCredentialMasterKeyMinLength    = 16
	gitRepoCredentialAADSchemaVersion      = 1
	gitRepoCredentialBase64MasterKeyPrefix = "base64:"
	gitRepoCredentialHexMasterKeyPrefix    = "hex:"
)

// GitRepoCredentialCipherOptions controls key metadata used during encryption.
type GitRepoCredentialCipherOptions struct {
	KeyID      string
	KeyVersion int
}

// GitRepoCredentialCipher performs authenticated encryption/decryption for stored git credentials.
type GitRepoCredentialCipher struct {
	keyID      string
	keyVersion int
	aead       cipher.AEAD
}

// GitRepoCredentialEncryptedBlob stores encrypted payload bytes plus key metadata.
type GitRepoCredentialEncryptedBlob struct {
	KeyID      string
	KeyVersion int
	Ciphertext []byte
	Nonce      []byte
}

// NewGitRepoCredentialCipher creates an AES-256-GCM cipher from the runtime master key.
func NewGitRepoCredentialCipher(
	masterKey string,
	options GitRepoCredentialCipherOptions,
) (*GitRepoCredentialCipher, error) {
	keyMaterial, err := deriveGitRepoCredentialKeyMaterial(masterKey)
	if err != nil {
		return nil, err
	}

	keyID := strings.TrimSpace(options.KeyID)
	if keyID == "" {
		keyID = defaultGitRepoCredentialKeyID
	}

	keyVersion := options.KeyVersion
	if keyVersion == 0 {
		keyVersion = defaultGitRepoCredentialKeyVersion
	}
	if keyVersion < 0 {
		return nil, fmt.Errorf("git repo credential key version must be greater than zero")
	}

	block, err := aes.NewCipher(keyMaterial[:])
	if err != nil {
		return nil, fmt.Errorf("initialize git repo credential block cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize git repo credential AEAD cipher: %w", err)
	}

	return &GitRepoCredentialCipher{
		keyID:      keyID,
		keyVersion: keyVersion,
		aead:       aead,
	}, nil
}

// KeyID returns the active key identifier associated with the cipher.
func (c *GitRepoCredentialCipher) KeyID() string {
	if c == nil {
		return ""
	}
	return c.keyID
}

// KeyVersion returns the active key version associated with the cipher.
func (c *GitRepoCredentialCipher) KeyVersion() int {
	if c == nil {
		return 0
	}
	return c.keyVersion
}

// Encrypt serializes and encrypts one typed secret payload for the given repo.
func (c *GitRepoCredentialCipher) Encrypt(
	repoID string,
	payload GitRepoCredentialSecretPayload,
) (GitRepoCredentialEncryptedBlob, error) {
	if c == nil {
		return GitRepoCredentialEncryptedBlob{}, fmt.Errorf("git repo credential cipher is required")
	}

	normalizedRepoID, err := normalizeRequiredID(repoID, "git repo credential repo_id")
	if err != nil {
		return GitRepoCredentialEncryptedBlob{}, err
	}

	encodedPayload, err := marshalGitRepoCredentialSecretPayload(payload)
	if err != nil {
		return GitRepoCredentialEncryptedBlob{}, err
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return GitRepoCredentialEncryptedBlob{}, fmt.Errorf("generate git repo credential nonce: %w", err)
	}

	ciphertext := c.aead.Seal(
		nil,
		nonce,
		encodedPayload,
		buildGitRepoCredentialAAD(normalizedRepoID, c.keyID, c.keyVersion),
	)

	return GitRepoCredentialEncryptedBlob{
		KeyID:      c.keyID,
		KeyVersion: c.keyVersion,
		Ciphertext: ciphertext,
		Nonce:      nonce,
	}, nil
}

// Decrypt validates metadata and decrypts one stored row into a typed secret payload.
func (c *GitRepoCredentialCipher) Decrypt(
	row GitRepoCredentialRow,
) (GitRepoCredentialSecretPayload, error) {
	if c == nil {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf("git repo credential cipher is required")
	}

	normalizedRepoID, err := normalizeRequiredID(row.RepoID, "git repo credential repo_id")
	if err != nil {
		return GitRepoCredentialSecretPayload{}, err
	}

	storedKeyID := strings.TrimSpace(row.KeyID)
	if storedKeyID != c.keyID || row.KeyVersion != c.keyVersion {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf(
			"%w for repo_id %q (expected key %q version %d, got key %q version %d)",
			ErrGitRepoCredentialKeyMismatch,
			normalizedRepoID,
			c.keyID,
			c.keyVersion,
			storedKeyID,
			row.KeyVersion,
		)
	}

	if len(row.Nonce) != c.aead.NonceSize() {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf(
			"%w for repo_id %q (nonce length %d is invalid)",
			ErrGitRepoCredentialDecryptFailed,
			normalizedRepoID,
			len(row.Nonce),
		)
	}
	if len(row.Ciphertext) == 0 {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf(
			"%w for repo_id %q (ciphertext is empty)",
			ErrGitRepoCredentialDecryptFailed,
			normalizedRepoID,
		)
	}

	plaintext, err := c.aead.Open(
		nil,
		row.Nonce,
		row.Ciphertext,
		buildGitRepoCredentialAAD(normalizedRepoID, c.keyID, c.keyVersion),
	)
	if err != nil {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf(
			"%w for repo_id %q",
			ErrGitRepoCredentialDecryptFailed,
			normalizedRepoID,
		)
	}

	payload, err := unmarshalGitRepoCredentialSecretPayload(plaintext)
	if err != nil {
		return GitRepoCredentialSecretPayload{}, fmt.Errorf(
			"%w for repo_id %q",
			ErrGitRepoCredentialDecryptFailed,
			normalizedRepoID,
		)
	}

	return payload, nil
}

func buildGitRepoCredentialAAD(repoID, keyID string, keyVersion int) []byte {
	return []byte(fmt.Sprintf(
		"skillserver.git_repo_credentials.v%d|%s|%s|%d",
		gitRepoCredentialAADSchemaVersion,
		repoID,
		keyID,
		keyVersion,
	))
}

func deriveGitRepoCredentialKeyMaterial(masterKey string) ([32]byte, error) {
	var empty [32]byte

	trimmed := strings.TrimSpace(masterKey)
	if trimmed == "" {
		return empty, fmt.Errorf("git credential master key is required")
	}

	masterKeyBytes, err := parseMasterKeyBytes(trimmed)
	if err != nil {
		return empty, err
	}
	if len(masterKeyBytes) < gitRepoCredentialMasterKeyMinLength {
		return empty, fmt.Errorf(
			"git credential master key must be at least %d bytes",
			gitRepoCredentialMasterKeyMinLength,
		)
	}

	return sha256.Sum256(masterKeyBytes), nil
}

func parseMasterKeyBytes(raw string) ([]byte, error) {
	lower := strings.ToLower(raw)

	switch {
	case strings.HasPrefix(lower, gitRepoCredentialBase64MasterKeyPrefix):
		encoded := strings.TrimSpace(raw[len(gitRepoCredentialBase64MasterKeyPrefix):])
		if encoded == "" {
			return nil, fmt.Errorf("base64 master key payload is empty")
		}
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode base64 master key: %w", err)
		}
		return decoded, nil
	case strings.HasPrefix(lower, gitRepoCredentialHexMasterKeyPrefix):
		encoded := strings.TrimSpace(raw[len(gitRepoCredentialHexMasterKeyPrefix):])
		if encoded == "" {
			return nil, fmt.Errorf("hex master key payload is empty")
		}
		decoded, err := hex.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode hex master key: %w", err)
		}
		return decoded, nil
	default:
		return []byte(raw), nil
	}
}
