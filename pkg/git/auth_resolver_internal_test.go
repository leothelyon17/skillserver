package git

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"testing"

	ssh "golang.org/x/crypto/ssh"
)

func TestNewKnownHostsCallbackFromDataRejectsEmptyData(t *testing.T) {
	t.Parallel()

	_, err := newKnownHostsCallbackFromData("   ")
	if err == nil {
		t.Fatal("expected error for empty known_hosts data")
	}
	if !strings.Contains(err.Error(), "known_hosts data is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewKnownHostsCallbackFromDataBuildsCallback(t *testing.T) {
	t.Parallel()

	knownHostsLine, hostKey := generateKnownHostsFixture(t, "internal.example.com")
	callback, err := newKnownHostsCallbackFromData(knownHostsLine)
	if err != nil {
		t.Fatalf("expected callback, got error %v", err)
	}
	if callback == nil {
		t.Fatal("expected callback to be non-nil")
	}

	remoteAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}
	if err := callback("internal.example.com:22", remoteAddr, hostKey); err != nil {
		t.Fatalf("expected known_hosts callback to accept host key, got %v", err)
	}
}

func TestPrivateKeyRequiresPassphrase(t *testing.T) {
	t.Parallel()

	if privateKeyRequiresPassphrase("not-a-pem") {
		t.Fatal("expected invalid PEM to not be marked as encrypted")
	}

	plain := generatePlainPrivateKeyPEM(t)
	if privateKeyRequiresPassphrase(plain) {
		t.Fatal("expected plain key to not require passphrase")
	}

	encrypted := generateEncryptedPrivateKeyPEM(t, "passphrase")
	if !privateKeyRequiresPassphrase(encrypted) {
		t.Fatal("expected encrypted key to require passphrase")
	}
}

func TestSourceResolverForRejectsNilResolverEntry(t *testing.T) {
	t.Parallel()

	resolver := &GitCredentialResolver{
		sourceResolvers: map[string]GitCredentialSourceResolver{
			GitRepoAuthSourceEnv: nil,
		},
	}

	_, err := resolver.sourceResolverFor(GitRepoAuthSourceEnv)
	if err == nil {
		t.Fatal("expected error for nil resolver entry")
	}
}

func TestEnvCredentialSourceResolverValidation(t *testing.T) {
	t.Parallel()

	r := envCredentialSourceResolver{}
	if _, err := r.Resolve("TOKEN"); err == nil {
		t.Fatal("expected nil lookupEnv to fail")
	}

	r = envCredentialSourceResolver{
		lookupEnv: func(string) (string, bool) { return "", false },
	}
	if _, err := r.Resolve("   "); err == nil {
		t.Fatal("expected empty env reference to fail")
	}
}

func TestFileCredentialSourceResolverValidation(t *testing.T) {
	t.Parallel()

	r := fileCredentialSourceResolver{}
	if _, err := r.Resolve("/tmp/secret"); err == nil {
		t.Fatal("expected nil file reader to fail")
	}

	r = fileCredentialSourceResolver{
		readFile: func(string) ([]byte, error) { return nil, nil },
	}
	if _, err := r.Resolve("   "); err == nil {
		t.Fatal("expected empty file reference to fail")
	}
}

func generateKnownHostsFixture(t *testing.T, host string) (string, ssh.PublicKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate host key: %v", err)
	}
	publicKey, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal host key: %v", err)
	}

	line := fmt.Sprintf("%s %s", host, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey))))
	return line, publicKey
}

func generatePlainPrivateKeyPEM(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return string(pem.EncodeToMemory(block))
}

func generateEncryptedPrivateKeyPEM(t *testing.T, passphrase string) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
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
