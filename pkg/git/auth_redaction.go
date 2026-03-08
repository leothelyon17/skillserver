package git

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	gitAuthURLUserInfoPattern = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.\-]*://)([^@/\s]+)@`)
	gitAuthSecretValuePattern = regexp.MustCompile(`(?i)\b(token|password|passphrase|secret)\s*[:=]\s*([^\s,;]+)`)
	gitAuthPrivateKeyPattern  = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	gitAuthFilePathPattern    = regexp.MustCompile(`(?i)\b(open|read|stat|lstat|remove|chmod|chown)\s+(/[^:\s]+)`)
)

// RedactGitAuthString removes sensitive auth material from an arbitrary string.
// It is safe to use in API responses and logs.
func RedactGitAuthString(input string) string {
	redacted := gitAuthPrivateKeyPattern.ReplaceAllString(input, "<redacted-private-key>")
	redacted = gitAuthURLUserInfoPattern.ReplaceAllString(redacted, `${1}<redacted>@`)
	redacted = gitAuthSecretValuePattern.ReplaceAllString(redacted, `${1}=<redacted>`)
	redacted = gitAuthFilePathPattern.ReplaceAllString(redacted, `${1} <redacted-file-path>`)
	return redacted
}

// RedactGitAuthError returns a secret-safe message for an auth-related error.
func RedactGitAuthError(err error) string {
	if err == nil {
		return ""
	}
	return RedactGitAuthString(err.Error())
}

// NewRedactedGitAuthErrorf creates an error with a sanitized message.
func NewRedactedGitAuthErrorf(format string, args ...any) error {
	return errors.New(RedactGitAuthString(fmt.Sprintf(format, args...)))
}
