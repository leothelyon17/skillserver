package git

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	ssh "golang.org/x/crypto/ssh"
)

// GitCredentialSourceResolver resolves a single credential reference for one source type.
type GitCredentialSourceResolver interface {
	Resolve(reference string) (string, error)
}

// GitCredentialResolverOptions configures credential-reference resolution.
type GitCredentialResolverOptions struct {
	EnvLookup    func(string) (string, bool)
	FileRead     func(string) ([]byte, error)
	EnvResolver  GitCredentialSourceResolver
	FileResolver GitCredentialSourceResolver
}

// ResolvedGitAuthCredentials stores resolved secret material for a repo auth descriptor.
type ResolvedGitAuthCredentials struct {
	Mode       string
	Source     string
	Username   string
	Password   string
	Token      string
	PrivateKey string
	Passphrase string
	KnownHosts string
}

// GitCredentialResolver resolves env/file credential references at sync time.
type GitCredentialResolver struct {
	sourceResolvers map[string]GitCredentialSourceResolver
}

// NewGitCredentialResolver creates a resolver for env/file credential sources.
func NewGitCredentialResolver(options GitCredentialResolverOptions) *GitCredentialResolver {
	envResolver := options.EnvResolver
	if envResolver == nil {
		lookupEnv := options.EnvLookup
		if lookupEnv == nil {
			lookupEnv = os.LookupEnv
		}
		envResolver = envCredentialSourceResolver{lookupEnv: lookupEnv}
	}

	fileResolver := options.FileResolver
	if fileResolver == nil {
		readFile := options.FileRead
		if readFile == nil {
			readFile = os.ReadFile
		}
		fileResolver = fileCredentialSourceResolver{readFile: readFile}
	}

	return &GitCredentialResolver{
		sourceResolvers: map[string]GitCredentialSourceResolver{
			GitRepoAuthSourceEnv:  envResolver,
			GitRepoAuthSourceFile: fileResolver,
		},
	}
}

// ValidateGitRepoAuthConfig validates auth/source/ref combinations without resolving secret values.
func ValidateGitRepoAuthConfig(auth GitRepoAuthConfig) error {
	normalized := normalizeGitRepoAuthConfig(auth)

	switch normalized.Mode {
	case GitRepoAuthModeNone:
		if normalized.Source != "" && normalized.Source != GitRepoAuthSourceNone {
			return fmt.Errorf("auth mode %q does not support source %q", normalized.Mode, normalized.Source)
		}
		return nil

	case GitRepoAuthModeHTTPSToken:
		if err := validateGitAuthSource(normalized.Mode, normalized.Source); err != nil {
			return err
		}
		if normalized.TokenRef == "" {
			return fmt.Errorf("auth mode %q requires token_ref", normalized.Mode)
		}
		return nil

	case GitRepoAuthModeHTTPSBasic:
		if err := validateGitAuthSource(normalized.Mode, normalized.Source); err != nil {
			return err
		}
		if normalized.UsernameRef == "" {
			return fmt.Errorf("auth mode %q requires username_ref", normalized.Mode)
		}
		if normalized.PasswordRef == "" {
			return fmt.Errorf("auth mode %q requires password_ref", normalized.Mode)
		}
		return nil

	case GitRepoAuthModeSSHKey:
		if err := validateGitAuthSource(normalized.Mode, normalized.Source); err != nil {
			return err
		}
		if normalized.KeyRef == "" {
			return fmt.Errorf("auth mode %q requires key_ref", normalized.Mode)
		}
		if normalized.KnownHostsRef == "" {
			return fmt.Errorf("auth mode %q requires known_hosts_ref", normalized.Mode)
		}
		return nil

	default:
		return fmt.Errorf("unsupported auth mode %q", normalized.Mode)
	}
}

func validateGitAuthSource(mode string, source string) error {
	switch source {
	case GitRepoAuthSourceEnv, GitRepoAuthSourceFile:
		return nil
	case "", GitRepoAuthSourceNone:
		return fmt.Errorf(
			"auth mode %q requires source %q or %q",
			mode,
			GitRepoAuthSourceEnv,
			GitRepoAuthSourceFile,
		)
	default:
		return fmt.Errorf("auth mode %q does not support source %q", mode, source)
	}
}

// ResolveAuthCredentials resolves non-secret auth metadata into secret material
// using env/file providers.
func (r *GitCredentialResolver) ResolveAuthCredentials(auth GitRepoAuthConfig) (ResolvedGitAuthCredentials, error) {
	if r == nil {
		r = NewGitCredentialResolver(GitCredentialResolverOptions{})
	}

	normalized := normalizeGitRepoAuthConfig(auth)
	if err := ValidateGitRepoAuthConfig(normalized); err != nil {
		return ResolvedGitAuthCredentials{}, NewRedactedGitAuthErrorf("%v", err)
	}

	resolved := ResolvedGitAuthCredentials{
		Mode:   normalized.Mode,
		Source: normalized.Source,
	}

	switch normalized.Mode {
	case GitRepoAuthModeNone:
		return resolved, nil

	case GitRepoAuthModeHTTPSToken:
		token, err := r.resolveRequiredRef(normalized.Source, normalized.TokenRef, "token_ref")
		if err != nil {
			return ResolvedGitAuthCredentials{}, err
		}
		username, err := r.resolveOptionalRef(normalized.Source, normalized.UsernameRef, "username_ref")
		if err != nil {
			return ResolvedGitAuthCredentials{}, err
		}
		resolved.Token = token
		resolved.Username = username

	case GitRepoAuthModeHTTPSBasic:
		username, err := r.resolveRequiredRef(normalized.Source, normalized.UsernameRef, "username_ref")
		if err != nil {
			return ResolvedGitAuthCredentials{}, err
		}
		password, err := r.resolveRequiredRef(normalized.Source, normalized.PasswordRef, "password_ref")
		if err != nil {
			return ResolvedGitAuthCredentials{}, err
		}
		resolved.Username = username
		resolved.Password = password

	case GitRepoAuthModeSSHKey:
		username, err := r.resolveOptionalRef(normalized.Source, normalized.UsernameRef, "username_ref")
		if err != nil {
			return ResolvedGitAuthCredentials{}, err
		}
		privateKey, err := r.resolveRequiredRef(normalized.Source, normalized.KeyRef, "key_ref")
		if err != nil {
			return ResolvedGitAuthCredentials{}, err
		}
		knownHosts, err := r.resolveRequiredRef(normalized.Source, normalized.KnownHostsRef, "known_hosts_ref")
		if err != nil {
			return ResolvedGitAuthCredentials{}, err
		}
		passphrase, err := r.resolveOptionalRef(normalized.Source, normalized.PasswordRef, "password_ref")
		if err != nil {
			return ResolvedGitAuthCredentials{}, err
		}

		resolved.Username = username
		resolved.PrivateKey = privateKey
		resolved.KnownHosts = knownHosts
		resolved.Passphrase = passphrase
	}

	return resolved, nil
}

func (r *GitCredentialResolver) resolveRequiredRef(source string, reference string, fieldName string) (string, error) {
	value, err := r.resolveOptionalRef(source, reference, fieldName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", NewRedactedGitAuthErrorf("%s must not resolve to an empty value", fieldName)
	}
	return value, nil
}

func (r *GitCredentialResolver) resolveOptionalRef(source string, reference string, fieldName string) (string, error) {
	ref := strings.TrimSpace(reference)
	if ref == "" {
		return "", nil
	}

	resolver, err := r.sourceResolverFor(source)
	if err != nil {
		return "", err
	}

	value, err := resolver.Resolve(ref)
	if err != nil {
		return "", NewRedactedGitAuthErrorf(
			"failed to resolve %s from %s source: %v",
			fieldName,
			source,
			err,
		)
	}

	if strings.TrimSpace(value) == "" {
		return "", NewRedactedGitAuthErrorf(
			"%s resolved from %s source is empty",
			fieldName,
			source,
		)
	}

	return value, nil
}

func (r *GitCredentialResolver) sourceResolverFor(source string) (GitCredentialSourceResolver, error) {
	resolver, ok := r.sourceResolvers[source]
	if !ok || resolver == nil {
		return nil, NewRedactedGitAuthErrorf("unsupported credential source %q", source)
	}
	return resolver, nil
}

// ResolveGitAuthMethod resolves credentials and builds a go-git transport auth method.
func ResolveGitAuthMethod(auth GitRepoAuthConfig, resolver *GitCredentialResolver) (transport.AuthMethod, error) {
	if resolver == nil {
		resolver = NewGitCredentialResolver(GitCredentialResolverOptions{})
	}

	credentials, err := resolver.ResolveAuthCredentials(auth)
	if err != nil {
		return nil, err
	}

	return BuildGitAuthMethod(credentials)
}

// BuildGitAuthMethod translates resolved credentials into go-git auth methods.
func BuildGitAuthMethod(credentials ResolvedGitAuthCredentials) (transport.AuthMethod, error) {
	mode := strings.TrimSpace(strings.ToLower(credentials.Mode))

	switch mode {
	case "", GitRepoAuthModeNone:
		return nil, nil

	case GitRepoAuthModeHTTPSToken:
		token := strings.TrimSpace(credentials.Token)
		if token == "" {
			return nil, NewRedactedGitAuthErrorf("auth mode %q requires a resolved token", mode)
		}

		username := strings.TrimSpace(credentials.Username)
		if username == "" {
			username = defaultSSHRepoUser
		}

		return &githttp.BasicAuth{
			Username: username,
			Password: token,
		}, nil

	case GitRepoAuthModeHTTPSBasic:
		username := strings.TrimSpace(credentials.Username)
		password := strings.TrimSpace(credentials.Password)
		if username == "" {
			return nil, NewRedactedGitAuthErrorf("auth mode %q requires a resolved username", mode)
		}
		if password == "" {
			return nil, NewRedactedGitAuthErrorf("auth mode %q requires a resolved password", mode)
		}

		return &githttp.BasicAuth{
			Username: username,
			Password: password,
		}, nil

	case GitRepoAuthModeSSHKey:
		return buildSSHAuthMethod(credentials)
	}

	return nil, NewRedactedGitAuthErrorf("unsupported auth mode %q", mode)
}

func buildSSHAuthMethod(credentials ResolvedGitAuthCredentials) (transport.AuthMethod, error) {
	privateKey := strings.TrimSpace(credentials.PrivateKey)
	if privateKey == "" {
		return nil, NewRedactedGitAuthErrorf("auth mode %q requires a resolved private key", GitRepoAuthModeSSHKey)
	}

	knownHosts := strings.TrimSpace(credentials.KnownHosts)
	if knownHosts == "" {
		return nil, NewRedactedGitAuthErrorf("auth mode %q requires known_hosts verification data", GitRepoAuthModeSSHKey)
	}

	passphrase := credentials.Passphrase
	if privateKeyRequiresPassphrase(privateKey) && strings.TrimSpace(passphrase) == "" {
		return nil, NewRedactedGitAuthErrorf(
			"auth mode %q requires password_ref when the private key is encrypted",
			GitRepoAuthModeSSHKey,
		)
	}

	username := strings.TrimSpace(credentials.Username)
	if username == "" {
		username = defaultSSHRepoUser
	}

	method, err := gitssh.NewPublicKeys(username, []byte(privateKey), passphrase)
	if err != nil {
		return nil, NewRedactedGitAuthErrorf("failed to parse ssh private key: %v", err)
	}

	hostKeyCallback, err := newKnownHostsCallbackFromData(knownHosts)
	if err != nil {
		return nil, err
	}
	method.HostKeyCallback = hostKeyCallback

	return method, nil
}

func privateKeyRequiresPassphrase(privateKey string) bool {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return false
	}

	if x509.IsEncryptedPEMBlock(block) {
		return true
	}

	blockType := strings.ToUpper(strings.TrimSpace(block.Type))
	if strings.Contains(blockType, "ENCRYPTED PRIVATE KEY") {
		return true
	}

	return strings.Contains(strings.ToUpper(block.Headers["Proc-Type"]), "ENCRYPTED")
}

func newKnownHostsCallbackFromData(knownHostsData string) (ssh.HostKeyCallback, error) {
	trimmed := strings.TrimSpace(knownHostsData)
	if trimmed == "" {
		return nil, NewRedactedGitAuthErrorf("known_hosts data is required")
	}

	tempFile, err := os.CreateTemp("", "skillserver-known-hosts-*")
	if err != nil {
		return nil, NewRedactedGitAuthErrorf("failed to initialize known_hosts validation data: %v", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.WriteString(trimmed + "\n"); err != nil {
		_ = tempFile.Close()
		return nil, NewRedactedGitAuthErrorf("failed to prepare known_hosts validation data")
	}
	if err := tempFile.Close(); err != nil {
		return nil, NewRedactedGitAuthErrorf("failed to prepare known_hosts validation data")
	}

	callback, err := gitssh.NewKnownHostsCallback(tempPath)
	if err != nil {
		return nil, NewRedactedGitAuthErrorf("known_hosts data is invalid")
	}

	return callback, nil
}

type envCredentialSourceResolver struct {
	lookupEnv func(string) (string, bool)
}

func (r envCredentialSourceResolver) Resolve(reference string) (string, error) {
	if r.lookupEnv == nil {
		return "", fmt.Errorf("environment lookup is not configured")
	}

	ref := strings.TrimSpace(reference)
	if ref == "" {
		return "", fmt.Errorf("environment variable reference is empty")
	}

	value, ok := r.lookupEnv(ref)
	if !ok {
		return "", fmt.Errorf("environment variable %q is not set", ref)
	}

	resolved := strings.TrimSpace(value)
	if resolved == "" {
		return "", fmt.Errorf("environment variable %q resolved empty value", ref)
	}

	return resolved, nil
}

type fileCredentialSourceResolver struct {
	readFile func(string) ([]byte, error)
}

func (r fileCredentialSourceResolver) Resolve(reference string) (string, error) {
	if r.readFile == nil {
		return "", fmt.Errorf("file reader is not configured")
	}

	ref := strings.TrimSpace(reference)
	if ref == "" {
		return "", fmt.Errorf("file credential reference is empty")
	}

	content, err := r.readFile(ref)
	if err != nil {
		return "", fmt.Errorf("file credential reference could not be read")
	}

	resolved := strings.TrimSpace(string(content))
	if resolved == "" {
		return "", fmt.Errorf("file credential reference resolved empty value")
	}

	return resolved, nil
}
