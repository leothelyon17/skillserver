package git_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mudler/skillserver/pkg/git"
)

func TestRedactGitAuthString(t *testing.T) {
	t.Parallel()

	raw := "clone failed for https://user:supersecret@github.com/acme/private.git token=abc123 open /var/run/secrets/git-token: permission denied"
	redacted := git.RedactGitAuthString(raw)

	if strings.Contains(redacted, "supersecret") {
		t.Fatalf("expected URL credentials to be redacted, got %q", redacted)
	}
	if strings.Contains(redacted, "abc123") {
		t.Fatalf("expected token assignment to be redacted, got %q", redacted)
	}
	if strings.Contains(redacted, "/var/run/secrets/git-token") {
		t.Fatalf("expected file path to be redacted, got %q", redacted)
	}
	if !strings.Contains(redacted, "<redacted>@") {
		t.Fatalf("expected redacted URL marker in %q", redacted)
	}
	if !strings.Contains(redacted, "<redacted-file-path>") {
		t.Fatalf("expected redacted file path marker in %q", redacted)
	}
}

func TestRedactGitAuthStringPrivateKeyBlock(t *testing.T) {
	t.Parallel()

	raw := "-----BEGIN OPENSSH PRIVATE KEY-----\nabc123\n-----END OPENSSH PRIVATE KEY-----"
	redacted := git.RedactGitAuthString(raw)
	if strings.Contains(redacted, "abc123") {
		t.Fatalf("expected private key block to be redacted, got %q", redacted)
	}
	if !strings.Contains(redacted, "<redacted-private-key>") {
		t.Fatalf("expected private key redaction marker, got %q", redacted)
	}
}

func TestRedactGitAuthError(t *testing.T) {
	t.Parallel()

	if got := git.RedactGitAuthError(nil); got != "" {
		t.Fatalf("expected empty string for nil error, got %q", got)
	}

	got := git.RedactGitAuthError(errors.New("password=hello"))
	if strings.Contains(got, "hello") {
		t.Fatalf("expected secret value to be redacted, got %q", got)
	}
}
