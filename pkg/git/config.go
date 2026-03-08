package git

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// GitRepoAuthModeNone is the default auth mode for public repositories.
	GitRepoAuthModeNone = "none"
	// GitRepoAuthModeHTTPSToken uses HTTPS with a token/password-style credential.
	GitRepoAuthModeHTTPSToken = "https_token"
	// GitRepoAuthModeHTTPSBasic uses HTTPS basic authentication.
	GitRepoAuthModeHTTPSBasic = "https_basic"
	// GitRepoAuthModeSSHKey uses SSH public-key authentication.
	GitRepoAuthModeSSHKey = "ssh_key"

	// GitRepoAuthSourceNone indicates no credential source (public repo mode).
	GitRepoAuthSourceNone = "none"
	// GitRepoAuthSourceEnv resolves credential references from environment variables.
	GitRepoAuthSourceEnv = "env"
	// GitRepoAuthSourceFile resolves credential references from mounted files.
	GitRepoAuthSourceFile = "file"
	// GitRepoAuthSourceStored resolves credentials from the optional encrypted store.
	GitRepoAuthSourceStored = "stored"

	repoIDPrefix           = "gitrepo_"
	repoIDHashPrefixLength = 8
	defaultSSHRepoUser     = "git"
)

var scpLikeRepoURLPattern = regexp.MustCompile(`^([^@\s/:]+)@([^:/\s]+):(.+)$`)
var checkoutNameSanitizePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// GitRepoAuthConfig represents non-secret authentication metadata for a git repository.
// Raw credential values are intentionally excluded from this model.
type GitRepoAuthConfig struct {
	Mode          string `json:"mode,omitempty"`
	Source        string `json:"source,omitempty"`
	ReferenceID   string `json:"reference_id,omitempty"`
	UsernameRef   string `json:"username_ref,omitempty"`
	PasswordRef   string `json:"password_ref,omitempty"`
	TokenRef      string `json:"token_ref,omitempty"`
	KeyRef        string `json:"key_ref,omitempty"`
	KnownHostsRef string `json:"known_hosts_ref,omitempty"`
}

// GitRepoConfig represents a git repository configuration
type GitRepoConfig struct {
	ID      string            `json:"id"`
	URL     string            `json:"url"`
	Name    string            `json:"name"`
	Enabled bool              `json:"enabled"`
	Auth    GitRepoAuthConfig `json:"auth"`
}

// ConfigManager manages git repository configurations
type ConfigManager struct {
	configPath string
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager(skillsDir string) *ConfigManager {
	return &ConfigManager{
		configPath: filepath.Join(skillsDir, ".git-repos.json"),
	}
}

// LoadConfig loads git repository configurations from the config file
func (cm *ConfigManager) LoadConfig() ([]GitRepoConfig, error) {
	// If config file doesn't exist, return empty slice
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		return []GitRepoConfig{}, nil
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var repos []GitRepoConfig
	if len(data) == 0 {
		return []GitRepoConfig{}, nil
	}

	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	normalizedRepos := make([]GitRepoConfig, len(repos))
	for i, repo := range repos {
		normalizedRepo, err := NormalizeGitRepoConfig(repo)
		if err != nil {
			return nil, fmt.Errorf("invalid git repo config at index %d: %w", i, err)
		}
		normalizedRepos[i] = normalizedRepo
	}

	return normalizedRepos, nil
}

// SaveConfig saves git repository configurations to the config file
func (cm *ConfigManager) SaveConfig(repos []GitRepoConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	normalizedRepos := make([]GitRepoConfig, len(repos))
	for i, repo := range repos {
		normalizedRepo, err := NormalizeGitRepoConfig(repo)
		if err != nil {
			return fmt.Errorf("invalid git repo config at index %d: %w", i, err)
		}
		normalizedRepos[i] = normalizedRepo
	}

	data, err := json.MarshalIndent(normalizedRepos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// NormalizeGitRepoConfig canonicalizes and backfills repo fields so legacy
// URL-only records can be re-saved in the expanded schema safely.
func NormalizeGitRepoConfig(repo GitRepoConfig) (GitRepoConfig, error) {
	canonicalURL, err := CanonicalizeRepoURL(repo.URL)
	if err != nil {
		return GitRepoConfig{}, err
	}

	normalized := repo
	normalized.URL = canonicalURL
	normalized.ID = GenerateID(canonicalURL)
	if strings.TrimSpace(normalized.Name) == "" {
		normalized.Name = ResolveCheckoutName(canonicalURL)
	}
	normalized.Auth = normalizeGitRepoAuthConfig(normalized.Auth)

	return normalized, nil
}

func normalizeGitRepoAuthConfig(auth GitRepoAuthConfig) GitRepoAuthConfig {
	normalized := auth
	normalized.Mode = strings.TrimSpace(strings.ToLower(normalized.Mode))
	if normalized.Mode == "" {
		normalized.Mode = GitRepoAuthModeNone
	}
	normalized.Source = strings.TrimSpace(strings.ToLower(normalized.Source))
	normalized.ReferenceID = strings.TrimSpace(normalized.ReferenceID)
	normalized.UsernameRef = strings.TrimSpace(normalized.UsernameRef)
	normalized.PasswordRef = strings.TrimSpace(normalized.PasswordRef)
	normalized.TokenRef = strings.TrimSpace(normalized.TokenRef)
	normalized.KeyRef = strings.TrimSpace(normalized.KeyRef)
	normalized.KnownHostsRef = strings.TrimSpace(normalized.KnownHostsRef)

	return normalized
}

// CanonicalizeRepoURL normalizes a git remote URL into a non-secret canonical form.
// HTTP(S) userinfo is rejected to prevent persisting credential-bearing URLs.
func CanonicalizeRepoURL(repoURL string) (string, error) {
	trimmed := strings.TrimSpace(repoURL)
	if trimmed == "" {
		return "", fmt.Errorf("repository URL is required")
	}

	if scpLikeRepoURLPattern.MatchString(trimmed) {
		return canonicalizeSCPLikeRepoURL(trimmed)
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid repository URL: %w", err)
	}

	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme == "" {
		return "", fmt.Errorf("repository URL scheme is required")
	}

	switch scheme {
	case "http", "https", "ssh":
	default:
		return "", fmt.Errorf("unsupported repository URL scheme %q", scheme)
	}

	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("repository URL host is required")
	}

	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("repository URL must not include query or fragment components")
	}

	if scheme == "http" || scheme == "https" {
		if parsed.User != nil {
			return "", fmt.Errorf("repository URL must not include userinfo")
		}
	} else if parsed.User != nil {
		if _, hasPassword := parsed.User.Password(); hasPassword {
			return "", fmt.Errorf("repository URL must not include password in SSH userinfo")
		}
	}

	host, err := normalizeHostPort(parsed.Host, scheme)
	if err != nil {
		return "", err
	}

	repoPath, err := normalizeRepoPath(parsed.Path)
	if err != nil {
		return "", err
	}

	if scheme == "ssh" {
		sshUser := defaultSSHRepoUser
		if parsed.User != nil {
			user := strings.TrimSpace(parsed.User.Username())
			if user != "" {
				sshUser = user
			}
		}
		return fmt.Sprintf("ssh://%s@%s/%s", sshUser, host, repoPath), nil
	}

	return fmt.Sprintf("%s://%s/%s", scheme, host, repoPath), nil
}

func canonicalizeSCPLikeRepoURL(repoURL string) (string, error) {
	matches := scpLikeRepoURLPattern.FindStringSubmatch(strings.TrimSpace(repoURL))
	if len(matches) != 4 {
		return "", fmt.Errorf("invalid SCP-like repository URL")
	}

	user := strings.TrimSpace(matches[1])
	host := strings.TrimSpace(matches[2])
	repoPath := strings.TrimSpace(matches[3])

	if user == "" || host == "" || repoPath == "" {
		return "", fmt.Errorf("invalid SCP-like repository URL")
	}

	normalizedHost, err := normalizeHostPort(host, "ssh")
	if err != nil {
		return "", err
	}

	normalizedPath, err := normalizeRepoPath(repoPath)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ssh://%s@%s/%s", user, normalizedHost, normalizedPath), nil
}

func normalizeHostPort(hostPort, scheme string) (string, error) {
	candidate := strings.TrimSpace(hostPort)
	if candidate == "" {
		return "", fmt.Errorf("repository URL host is required")
	}

	host := candidate
	port := ""

	if strings.Contains(candidate, ":") {
		parsedHost, parsedPort, err := net.SplitHostPort(candidate)
		if err == nil {
			host = parsedHost
			port = parsedPort
		}
	}

	host = strings.TrimSpace(strings.Trim(host, "[]"))
	host = strings.ToLower(host)
	if host == "" {
		return "", fmt.Errorf("repository URL host is required")
	}

	switch {
	case scheme == "http" && port == "80":
		port = ""
	case scheme == "https" && port == "443":
		port = ""
	case scheme == "ssh" && port == "22":
		port = ""
	}

	if port == "" {
		return host, nil
	}

	return net.JoinHostPort(host, port), nil
}

func normalizeRepoPath(rawPath string) (string, error) {
	pathValue := strings.TrimSpace(rawPath)
	pathValue = strings.ReplaceAll(pathValue, "\\", "/")
	pathValue = strings.Trim(pathValue, "/")

	if pathValue == "" {
		return "", fmt.Errorf("repository URL path is required")
	}

	pathValue = path.Clean(pathValue)
	pathValue = strings.Trim(pathValue, "/")
	if pathValue == "" || pathValue == "." {
		return "", fmt.Errorf("repository URL path is required")
	}
	if strings.HasPrefix(pathValue, "../") || pathValue == ".." {
		return "", fmt.Errorf("repository URL path must not traverse parent directories")
	}

	return pathValue, nil
}

// ExtractRepoName extracts a repository name from a URL.
func ExtractRepoName(repoURL string) string {
	canonicalURL, err := CanonicalizeRepoURL(repoURL)
	if err != nil {
		return ""
	}

	parsed, err := url.Parse(canonicalURL)
	if err != nil {
		return ""
	}

	repoPath := strings.TrimSpace(parsed.Path)
	repoPath = strings.Trim(repoPath, "/")
	if repoPath == "" {
		return ""
	}

	return strings.TrimSuffix(path.Base(repoPath), ".git")
}

// ResolveCheckoutName returns the deterministic local checkout directory name
// derived from canonical repository metadata.
func ResolveCheckoutName(repoURL string) string {
	repoName := strings.TrimSpace(ExtractRepoName(repoURL))
	if repoName != "" {
		return repoName
	}

	return GenerateID(repoURL)
}

// ResolveRepoCheckoutName returns the deterministic local checkout directory
// name for a typed repository configuration. It prefers a sanitized config
// name, then canonical URL-derived fallback names.
func ResolveRepoCheckoutName(repo GitRepoConfig) string {
	if sanitized := sanitizeCheckoutName(repo.Name); sanitized != "" {
		return sanitized
	}

	if sanitized := sanitizeCheckoutName(ResolveCheckoutName(repo.URL)); sanitized != "" {
		return sanitized
	}

	if sanitized := sanitizeCheckoutName(repo.ID); sanitized != "" {
		return sanitized
	}

	return GenerateID(repo.URL)
}

// GenerateID generates a stable ID derived from the canonical repository URL.
func GenerateID(repoURL string) string {
	canonicalURL, err := CanonicalizeRepoURL(repoURL)
	if err != nil {
		canonicalURL = strings.TrimSpace(repoURL)
	}

	digest := sha256.Sum256([]byte(canonicalURL))
	hashPrefix := hex.EncodeToString(digest[:])[:repoIDHashPrefixLength]
	return repoIDPrefix + hashPrefix
}

func sanitizeCheckoutName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.ReplaceAll(trimmed, "/", "-")
	trimmed = strings.ReplaceAll(trimmed, "\\", "-")
	trimmed = checkoutNameSanitizePattern.ReplaceAllString(trimmed, "-")
	trimmed = strings.Trim(trimmed, "-.")
	if trimmed == "" {
		return ""
	}
	if trimmed == "." || trimmed == ".." {
		return ""
	}

	return trimmed
}
