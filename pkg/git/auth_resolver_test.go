package git_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	ssh "golang.org/x/crypto/ssh"

	"github.com/mudler/skillserver/pkg/git"
)

func TestValidateGitRepoAuthConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		auth    git.GitRepoAuthConfig
		wantErr string
	}{
		{
			name: "none mode without source is valid",
			auth: git.GitRepoAuthConfig{
				Mode: git.GitRepoAuthModeNone,
			},
		},
		{
			name: "none mode with env source is invalid",
			auth: git.GitRepoAuthConfig{
				Mode:   git.GitRepoAuthModeNone,
				Source: git.GitRepoAuthSourceEnv,
			},
			wantErr: "does not support source",
		},
		{
			name: "https token requires token ref",
			auth: git.GitRepoAuthConfig{
				Mode:   git.GitRepoAuthModeHTTPSToken,
				Source: git.GitRepoAuthSourceEnv,
			},
			wantErr: "requires token_ref",
		},
		{
			name: "https token with env source and token ref is valid",
			auth: git.GitRepoAuthConfig{
				Mode:     git.GitRepoAuthModeHTTPSToken,
				Source:   git.GitRepoAuthSourceEnv,
				TokenRef: "TOKEN_ENV",
			},
		},
		{
			name: "https token requires supported source",
			auth: git.GitRepoAuthConfig{
				Mode:     git.GitRepoAuthModeHTTPSToken,
				TokenRef: "TOKEN_ENV",
			},
			wantErr: "requires source",
		},
		{
			name: "https basic requires username and password refs",
			auth: git.GitRepoAuthConfig{
				Mode:        git.GitRepoAuthModeHTTPSBasic,
				Source:      git.GitRepoAuthSourceFile,
				UsernameRef: "/tmp/user",
				PasswordRef: "/tmp/pass",
			},
		},
		{
			name: "ssh key requires known hosts ref",
			auth: git.GitRepoAuthConfig{
				Mode:   git.GitRepoAuthModeSSHKey,
				Source: git.GitRepoAuthSourceEnv,
				KeyRef: "PRIVATE_KEY",
			},
			wantErr: "requires known_hosts_ref",
		},
		{
			name: "stored source unsupported in wp003 resolver",
			auth: git.GitRepoAuthConfig{
				Mode:     git.GitRepoAuthModeHTTPSToken,
				Source:   git.GitRepoAuthSourceStored,
				TokenRef: "TOKEN",
			},
			wantErr: "does not support source",
		},
		{
			name: "unknown auth mode rejected",
			auth: git.GitRepoAuthConfig{
				Mode: "custom_mode",
			},
			wantErr: "unsupported auth mode",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := git.ValidateGitRepoAuthConfig(tc.auth)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestGitCredentialResolverResolveAuthCredentials(t *testing.T) {
	t.Run("env source resolves token mode references", func(t *testing.T) {
		t.Setenv("TEST_GIT_TOKEN", "  token-value  ")
		t.Setenv("TEST_GIT_USER", "repo-user")

		resolver := git.NewGitCredentialResolver(git.GitCredentialResolverOptions{})
		resolved, err := resolver.ResolveAuthCredentials(git.GitRepoAuthConfig{
			Mode:        git.GitRepoAuthModeHTTPSToken,
			Source:      git.GitRepoAuthSourceEnv,
			TokenRef:    "TEST_GIT_TOKEN",
			UsernameRef: "TEST_GIT_USER",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if resolved.Mode != git.GitRepoAuthModeHTTPSToken {
			t.Fatalf("expected mode %q, got %q", git.GitRepoAuthModeHTTPSToken, resolved.Mode)
		}
		if resolved.Token != "token-value" {
			t.Fatalf("expected token to be trimmed, got %q", resolved.Token)
		}
		if resolved.Username != "repo-user" {
			t.Fatalf("expected username repo-user, got %q", resolved.Username)
		}
	})

	t.Run("file source resolves basic auth references", func(t *testing.T) {
		tempDir := t.TempDir()
		usernamePath := filepath.Join(tempDir, "username")
		passwordPath := filepath.Join(tempDir, "password")
		if err := os.WriteFile(usernamePath, []byte("  alice \n"), 0600); err != nil {
			t.Fatalf("failed to write username file: %v", err)
		}
		if err := os.WriteFile(passwordPath, []byte(" secret-value\n"), 0600); err != nil {
			t.Fatalf("failed to write password file: %v", err)
		}

		resolver := git.NewGitCredentialResolver(git.GitCredentialResolverOptions{})
		resolved, err := resolver.ResolveAuthCredentials(git.GitRepoAuthConfig{
			Mode:        git.GitRepoAuthModeHTTPSBasic,
			Source:      git.GitRepoAuthSourceFile,
			UsernameRef: usernamePath,
			PasswordRef: passwordPath,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if resolved.Username != "alice" {
			t.Fatalf("expected username alice, got %q", resolved.Username)
		}
		if resolved.Password != "secret-value" {
			t.Fatalf("expected password secret-value, got %q", resolved.Password)
		}
	})

	t.Run("missing env reference is actionable and redacted", func(t *testing.T) {
		resolver := git.NewGitCredentialResolver(git.GitCredentialResolverOptions{})
		_, err := resolver.ResolveAuthCredentials(git.GitRepoAuthConfig{
			Mode:     git.GitRepoAuthModeHTTPSToken,
			Source:   git.GitRepoAuthSourceEnv,
			TokenRef: "MISSING_GIT_TOKEN",
		})
		if err == nil {
			t.Fatal("expected error for missing env reference")
		}
		if !strings.Contains(err.Error(), "MISSING_GIT_TOKEN") {
			t.Fatalf("expected env reference name in error, got %q", err.Error())
		}
	})

	t.Run("unreadable file reference does not leak file path", func(t *testing.T) {
		tempDir := t.TempDir()
		secretPath := filepath.Join(tempDir, "private", "token.txt")

		resolver := git.NewGitCredentialResolver(git.GitCredentialResolverOptions{})
		_, err := resolver.ResolveAuthCredentials(git.GitRepoAuthConfig{
			Mode:     git.GitRepoAuthModeHTTPSToken,
			Source:   git.GitRepoAuthSourceFile,
			TokenRef: secretPath,
		})
		if err == nil {
			t.Fatal("expected error for unreadable file")
		}
		if !strings.Contains(err.Error(), "file credential reference could not be read") {
			t.Fatalf("expected unreadable file message, got %q", err.Error())
		}
		if strings.Contains(err.Error(), secretPath) {
			t.Fatalf("error leaked file path %q: %q", secretPath, err.Error())
		}
	})

	t.Run("none mode resolves without source lookups", func(t *testing.T) {
		resolver := git.NewGitCredentialResolver(git.GitCredentialResolverOptions{})
		resolved, err := resolver.ResolveAuthCredentials(git.GitRepoAuthConfig{
			Mode: git.GitRepoAuthModeNone,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resolved.Mode != git.GitRepoAuthModeNone {
			t.Fatalf("expected mode none, got %q", resolved.Mode)
		}
	})

	t.Run("token mode allows optional username ref to be omitted", func(t *testing.T) {
		t.Setenv("TOKEN_ONLY", "token-only-value")

		resolver := git.NewGitCredentialResolver(git.GitCredentialResolverOptions{})
		resolved, err := resolver.ResolveAuthCredentials(git.GitRepoAuthConfig{
			Mode:     git.GitRepoAuthModeHTTPSToken,
			Source:   git.GitRepoAuthSourceEnv,
			TokenRef: "TOKEN_ONLY",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resolved.Username != "" {
			t.Fatalf("expected empty username when username_ref is omitted, got %q", resolved.Username)
		}
		if resolved.Token != "token-only-value" {
			t.Fatalf("expected token-only-value, got %q", resolved.Token)
		}
	})

	t.Run("ssh key mode resolves file refs including passphrase and known_hosts", func(t *testing.T) {
		tempDir := t.TempDir()
		usernamePath := filepath.Join(tempDir, "ssh-user")
		keyPath := filepath.Join(tempDir, "ssh-key")
		passphrasePath := filepath.Join(tempDir, "ssh-passphrase")
		knownHostsPath := filepath.Join(tempDir, "known-hosts")
		privateKey := generateRSAPrivateKeyPEM(t)
		knownHostsLine, _ := generateKnownHostsLine(t, "example.com")

		if err := os.WriteFile(usernamePath, []byte("git"), 0600); err != nil {
			t.Fatalf("failed to write username ref: %v", err)
		}
		if err := os.WriteFile(keyPath, []byte(privateKey), 0600); err != nil {
			t.Fatalf("failed to write key ref: %v", err)
		}
		if err := os.WriteFile(passphrasePath, []byte("passphrase"), 0600); err != nil {
			t.Fatalf("failed to write passphrase ref: %v", err)
		}
		if err := os.WriteFile(knownHostsPath, []byte(knownHostsLine), 0600); err != nil {
			t.Fatalf("failed to write known_hosts ref: %v", err)
		}

		resolver := git.NewGitCredentialResolver(git.GitCredentialResolverOptions{})
		resolved, err := resolver.ResolveAuthCredentials(git.GitRepoAuthConfig{
			Mode:          git.GitRepoAuthModeSSHKey,
			Source:        git.GitRepoAuthSourceFile,
			UsernameRef:   usernamePath,
			KeyRef:        keyPath,
			PasswordRef:   passphrasePath,
			KnownHostsRef: knownHostsPath,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resolved.Username != "git" {
			t.Fatalf("expected username git, got %q", resolved.Username)
		}
		if resolved.PrivateKey == "" || resolved.KnownHosts == "" {
			t.Fatalf("expected key and known_hosts to resolve, got %+v", resolved)
		}
		if resolved.Passphrase != "passphrase" {
			t.Fatalf("expected passphrase from password_ref, got %q", resolved.Passphrase)
		}
	})

	t.Run("configured empty env value fails resolution", func(t *testing.T) {
		t.Setenv("EMPTY_GIT_TOKEN", "   ")

		resolver := git.NewGitCredentialResolver(git.GitCredentialResolverOptions{})
		_, err := resolver.ResolveAuthCredentials(git.GitRepoAuthConfig{
			Mode:     git.GitRepoAuthModeHTTPSToken,
			Source:   git.GitRepoAuthSourceEnv,
			TokenRef: "EMPTY_GIT_TOKEN",
		})
		if err == nil {
			t.Fatal("expected empty env value to fail resolution")
		}
		if !strings.Contains(err.Error(), "resolved empty value") {
			t.Fatalf("expected empty value error, got %q", err.Error())
		}
	})

	t.Run("nil resolver fallback path is supported", func(t *testing.T) {
		t.Setenv("NIL_RESOLVER_TOKEN", "token-value")
		var resolver *git.GitCredentialResolver

		resolved, err := resolver.ResolveAuthCredentials(git.GitRepoAuthConfig{
			Mode:     git.GitRepoAuthModeHTTPSToken,
			Source:   git.GitRepoAuthSourceEnv,
			TokenRef: "NIL_RESOLVER_TOKEN",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resolved.Token != "token-value" {
			t.Fatalf("expected token-value, got %q", resolved.Token)
		}
	})
}

func TestBuildGitAuthMethod(t *testing.T) {
	t.Parallel()

	t.Run("none mode returns nil auth method", func(t *testing.T) {
		method, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode: git.GitRepoAuthModeNone,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if method != nil {
			t.Fatalf("expected nil auth method, got %T", method)
		}
	})

	t.Run("https token mode returns basic auth with default username", func(t *testing.T) {
		method, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode:  git.GitRepoAuthModeHTTPSToken,
			Token: "token-123",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		basicAuth, ok := method.(*githttp.BasicAuth)
		if !ok {
			t.Fatalf("expected *http.BasicAuth, got %T", method)
		}
		if basicAuth.Username != "git" {
			t.Fatalf("expected default username git, got %q", basicAuth.Username)
		}
		if basicAuth.Password != "token-123" {
			t.Fatalf("expected token password token-123, got %q", basicAuth.Password)
		}
	})

	t.Run("https basic mode returns basic auth", func(t *testing.T) {
		method, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode:     git.GitRepoAuthModeHTTPSBasic,
			Username: "alice",
			Password: "password-123",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		basicAuth, ok := method.(*githttp.BasicAuth)
		if !ok {
			t.Fatalf("expected *http.BasicAuth, got %T", method)
		}
		if basicAuth.Username != "alice" || basicAuth.Password != "password-123" {
			t.Fatalf("unexpected basic auth payload: %+v", basicAuth)
		}
	})

	t.Run("ssh key mode enforces known hosts and returns public keys auth", func(t *testing.T) {
		clientPrivateKey := generateRSAPrivateKeyPEM(t)
		knownHostsLine, hostPublicKey := generateKnownHostsLine(t, "example.com")

		method, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode:       git.GitRepoAuthModeSSHKey,
			Username:   "git",
			PrivateKey: clientPrivateKey,
			KnownHosts: knownHostsLine,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		sshAuth, ok := method.(*gitssh.PublicKeys)
		if !ok {
			t.Fatalf("expected *ssh.PublicKeys, got %T", method)
		}
		if sshAuth.HostKeyCallback == nil {
			t.Fatal("expected HostKeyCallback to be configured")
		}

		remoteAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}
		if err := sshAuth.HostKeyCallback("example.com:22", remoteAddr, hostPublicKey); err != nil {
			t.Fatalf("expected known_hosts callback to accept known host key, got %v", err)
		}
	})

	t.Run("ssh key mode rejects empty known hosts", func(t *testing.T) {
		_, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode:       git.GitRepoAuthModeSSHKey,
			PrivateKey: generateRSAPrivateKeyPEM(t),
			KnownHosts: "   ",
		})
		if err == nil {
			t.Fatal("expected error for missing known_hosts data")
		}
		if !strings.Contains(err.Error(), "known_hosts") {
			t.Fatalf("expected known_hosts validation error, got %q", err.Error())
		}
	})

	t.Run("ssh key mode validates encrypted key passphrase requirement", func(t *testing.T) {
		encryptedKey := generateEncryptedRSAPrivateKeyPEM(t, "correct-horse")
		knownHostsLine, _ := generateKnownHostsLine(t, "example.com")

		_, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode:       git.GitRepoAuthModeSSHKey,
			PrivateKey: encryptedKey,
			KnownHosts: knownHostsLine,
		})
		if err == nil {
			t.Fatal("expected error when passphrase is missing")
		}
		if !strings.Contains(err.Error(), "requires password_ref") {
			t.Fatalf("expected missing passphrase validation error, got %q", err.Error())
		}

		_, err = git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode:       git.GitRepoAuthModeSSHKey,
			PrivateKey: encryptedKey,
			Passphrase: "correct-horse",
			KnownHosts: knownHostsLine,
		})
		if err != nil {
			t.Fatalf("expected encrypted key with passphrase to work, got %v", err)
		}
	})

	t.Run("ssh key mode rejects malformed known hosts data", func(t *testing.T) {
		_, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode:       git.GitRepoAuthModeSSHKey,
			PrivateKey: generateRSAPrivateKeyPEM(t),
			KnownHosts: "not-a-valid-known-hosts-line",
		})
		if err == nil {
			t.Fatal("expected malformed known_hosts data to fail")
		}
		if !strings.Contains(err.Error(), "known_hosts data is invalid") {
			t.Fatalf("expected known_hosts parse error, got %q", err.Error())
		}
	})

	t.Run("unsupported auth mode fails", func(t *testing.T) {
		_, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode: "custom-mode",
		})
		if err == nil {
			t.Fatal("expected unsupported mode error")
		}
		if !strings.Contains(err.Error(), "unsupported auth mode") {
			t.Fatalf("expected unsupported mode error, got %q", err.Error())
		}
	})

	t.Run("https token mode rejects empty token", func(t *testing.T) {
		_, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode:  git.GitRepoAuthModeHTTPSToken,
			Token: " ",
		})
		if err == nil {
			t.Fatal("expected empty token to fail")
		}
		if !strings.Contains(err.Error(), "requires a resolved token") {
			t.Fatalf("expected resolved token error, got %q", err.Error())
		}
	})

	t.Run("https basic mode rejects empty username", func(t *testing.T) {
		_, err := git.BuildGitAuthMethod(git.ResolvedGitAuthCredentials{
			Mode:     git.GitRepoAuthModeHTTPSBasic,
			Username: " ",
			Password: "password",
		})
		if err == nil {
			t.Fatal("expected empty username to fail")
		}
		if !strings.Contains(err.Error(), "requires a resolved username") {
			t.Fatalf("expected resolved username error, got %q", err.Error())
		}
	})
}

func TestResolveGitAuthMethod(t *testing.T) {
	t.Run("resolves token mode into basic auth", func(t *testing.T) {
		t.Setenv("REPO_TOKEN", "token-from-env")

		resolver := git.NewGitCredentialResolver(git.GitCredentialResolverOptions{})
		method, err := git.ResolveGitAuthMethod(git.GitRepoAuthConfig{
			Mode:     git.GitRepoAuthModeHTTPSToken,
			Source:   git.GitRepoAuthSourceEnv,
			TokenRef: "REPO_TOKEN",
		}, resolver)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		basicAuth, ok := method.(*githttp.BasicAuth)
		if !ok {
			t.Fatalf("expected *http.BasicAuth, got %T", method)
		}
		if basicAuth.Password != "token-from-env" {
			t.Fatalf("expected token password token-from-env, got %q", basicAuth.Password)
		}
	})

	t.Run("nil resolver uses default resolver", func(t *testing.T) {
		t.Setenv("REPO_TOKEN_DEFAULT", "default-token")

		method, err := git.ResolveGitAuthMethod(git.GitRepoAuthConfig{
			Mode:     git.GitRepoAuthModeHTTPSToken,
			Source:   git.GitRepoAuthSourceEnv,
			TokenRef: "REPO_TOKEN_DEFAULT",
		}, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		basicAuth, ok := method.(*githttp.BasicAuth)
		if !ok {
			t.Fatalf("expected *http.BasicAuth, got %T", method)
		}
		if basicAuth.Password != "default-token" {
			t.Fatalf("expected default-token, got %q", basicAuth.Password)
		}
	})
}

func generateRSAPrivateKeyPEM(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate RSA private key: %v", err)
	}

	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return string(pem.EncodeToMemory(block))
}

func generateEncryptedRSAPrivateKeyPEM(t *testing.T, passphrase string) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate RSA private key: %v", err)
	}

	encryptedBlock, err := x509.EncryptPEMBlock(
		rand.Reader,
		"RSA PRIVATE KEY",
		x509.MarshalPKCS1PrivateKey(key),
		[]byte(passphrase),
		x509.PEMCipherAES256,
	)
	if err != nil {
		t.Fatalf("failed to encrypt private key: %v", err)
	}

	return string(pem.EncodeToMemory(encryptedBlock))
}

func generateKnownHostsLine(t *testing.T, host string) (string, ssh.PublicKey) {
	t.Helper()

	hostKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate host key: %v", err)
	}

	publicKey, err := ssh.NewPublicKey(&hostKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal host public key: %v", err)
	}

	line := fmt.Sprintf("%s %s", host, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey))))
	return line, publicKey
}
