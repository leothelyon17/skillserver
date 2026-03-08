package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	envGitEnableStoredCredentials = "SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS"
	envGitCredentialMasterKey     = "SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY"
	envGitCredentialMasterKeyFile = "SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE"
)

const (
	defaultGitEnableStoredCredentials = false
	defaultGitCredentialMasterKey     = ""
	defaultGitCredentialMasterKeyFile = ""
)

// GitCredentialMasterKeySource identifies how the runtime resolved the
// credential-encryption master key.
type GitCredentialMasterKeySource string

const (
	// GitCredentialMasterKeySourceNone indicates no master key source was configured.
	GitCredentialMasterKeySourceNone GitCredentialMasterKeySource = "none"
	// GitCredentialMasterKeySourceInline indicates the key was provided inline (flag/env).
	GitCredentialMasterKeySourceInline GitCredentialMasterKeySource = "inline"
	// GitCredentialMasterKeySourceFile indicates the key was loaded from a file path.
	GitCredentialMasterKeySourceFile GitCredentialMasterKeySource = "file"
)

// GitCredentialRuntimeConfig defines runtime configuration for private git stored credentials.
type GitCredentialRuntimeConfig struct {
	EnableStoredCredentials bool
	MasterKeySource         GitCredentialMasterKeySource
	MasterKey               string
	MasterKeyFile           string
}

type gitCredentialRuntimeFlagValues struct {
	enableStoredCredentials bool
	masterKey               string
	masterKeyFile           string
}

// registerGitCredentialRuntimeFlags adds private git credential runtime flags to a flag set.
func registerGitCredentialRuntimeFlags(fs *flag.FlagSet) *gitCredentialRuntimeFlagValues {
	values := &gitCredentialRuntimeFlagValues{}

	fs.BoolVar(
		&values.enableStoredCredentials,
		"git-enable-stored-credentials",
		defaultGitEnableStoredCredentials,
		"Enable encrypted stored credentials for private Git repos (env: SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS)",
	)
	fs.StringVar(
		&values.masterKey,
		"git-credential-master-key",
		defaultGitCredentialMasterKey,
		"Inline master key for encrypting stored Git credentials (env: SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY)",
	)
	fs.StringVar(
		&values.masterKeyFile,
		"git-credential-master-key-file",
		defaultGitCredentialMasterKeyFile,
		"File path containing the master key for stored Git credentials (env: SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE)",
	)

	return values
}

// parseGitCredentialRuntimeConfig resolves and validates private git credential
// runtime config with precedence: flags > environment variables > defaults.
func parseGitCredentialRuntimeConfig(
	fs *flag.FlagSet,
	flagValues *gitCredentialRuntimeFlagValues,
	lookupEnv func(string) (string, bool),
	readFile func(string) ([]byte, error),
) (GitCredentialRuntimeConfig, error) {
	if fs == nil {
		return GitCredentialRuntimeConfig{}, fmt.Errorf("flag set is required")
	}
	if flagValues == nil {
		return GitCredentialRuntimeConfig{}, fmt.Errorf("flag values are required")
	}
	if readFile == nil {
		readFile = os.ReadFile
	}

	enabled, err := resolveBoolConfigValue(
		fs,
		"git-enable-stored-credentials",
		flagValues.enableStoredCredentials,
		envGitEnableStoredCredentials,
		defaultGitEnableStoredCredentials,
		lookupEnv,
	)
	if err != nil {
		return GitCredentialRuntimeConfig{}, err
	}

	if !enabled {
		return GitCredentialRuntimeConfig{
			EnableStoredCredentials: false,
			MasterKeySource:         GitCredentialMasterKeySourceNone,
		}, nil
	}

	if isFlagSet(fs, "git-credential-master-key") && strings.TrimSpace(flagValues.masterKey) == "" {
		return GitCredentialRuntimeConfig{}, fmt.Errorf("flag --git-credential-master-key cannot be empty when provided")
	}
	if isFlagSet(fs, "git-credential-master-key-file") && strings.TrimSpace(flagValues.masterKeyFile) == "" {
		return GitCredentialRuntimeConfig{}, fmt.Errorf("flag --git-credential-master-key-file cannot be empty when provided")
	}

	masterKeyRaw, masterKeySource := resolveStringConfigValue(
		fs,
		"git-credential-master-key",
		flagValues.masterKey,
		envGitCredentialMasterKey,
		defaultGitCredentialMasterKey,
		lookupEnv,
	)
	masterKeyFileRaw, masterKeyFileSource := resolveStringConfigValue(
		fs,
		"git-credential-master-key-file",
		flagValues.masterKeyFile,
		envGitCredentialMasterKeyFile,
		defaultGitCredentialMasterKeyFile,
		lookupEnv,
	)

	hasInlineKey := strings.TrimSpace(masterKeyRaw) != ""
	hasKeyFile := strings.TrimSpace(masterKeyFileRaw) != ""

	if hasInlineKey && hasKeyFile {
		return GitCredentialRuntimeConfig{}, fmt.Errorf(
			"%s and %s are mutually exclusive; configure exactly one of %s or %s",
			masterKeySource,
			masterKeyFileSource,
			envGitCredentialMasterKey,
			envGitCredentialMasterKeyFile,
		)
	}

	if !hasInlineKey && !hasKeyFile {
		return GitCredentialRuntimeConfig{}, fmt.Errorf(
			"%s=true requires a master key via %s or %s",
			envGitEnableStoredCredentials,
			envGitCredentialMasterKey,
			envGitCredentialMasterKeyFile,
		)
	}

	if hasInlineKey {
		return GitCredentialRuntimeConfig{
			EnableStoredCredentials: true,
			MasterKeySource:         GitCredentialMasterKeySourceInline,
			MasterKey:               strings.TrimSpace(masterKeyRaw),
		}, nil
	}

	masterKeyFile, err := parseGitCredentialMasterKeyFilePath(masterKeyFileRaw)
	if err != nil {
		return GitCredentialRuntimeConfig{}, fmt.Errorf("%s: %w", masterKeyFileSource, err)
	}
	masterKey, err := loadGitCredentialMasterKeyFromFile(masterKeyFile, readFile)
	if err != nil {
		return GitCredentialRuntimeConfig{}, fmt.Errorf("%s: %w", masterKeyFileSource, err)
	}

	return GitCredentialRuntimeConfig{
		EnableStoredCredentials: true,
		MasterKeySource:         GitCredentialMasterKeySourceFile,
		MasterKey:               masterKey,
		MasterKeyFile:           masterKeyFile,
	}, nil
}

// validateGitCredentialStartupConfig performs startup guardrails for private git
// stored-credential mode.
func validateGitCredentialStartupConfig(
	gitCredentialCfg GitCredentialRuntimeConfig,
	persistenceCfg PersistenceRuntimeConfig,
) error {
	if !gitCredentialCfg.EnableStoredCredentials {
		return nil
	}

	if !persistenceCfg.Enabled {
		return fmt.Errorf(
			"%s=true requires %s=true",
			envGitEnableStoredCredentials,
			envPersistenceData,
		)
	}

	if strings.TrimSpace(gitCredentialCfg.MasterKey) == "" || gitCredentialCfg.MasterKeySource == GitCredentialMasterKeySourceNone {
		return fmt.Errorf(
			"%s=true requires a resolved master key via %s or %s",
			envGitEnableStoredCredentials,
			envGitCredentialMasterKey,
			envGitCredentialMasterKeyFile,
		)
	}

	return nil
}

func parseGitCredentialMasterKeyFilePath(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("master key file path cannot be empty")
	}

	cleaned := filepath.Clean(value)
	resolved, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("failed to resolve master key file path %q: %w", raw, err)
	}

	return resolved, nil
}

func loadGitCredentialMasterKeyFromFile(
	path string,
	readFile func(string) ([]byte, error),
) (string, error) {
	content, err := readFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read master key file %q: %w", path, err)
	}

	value := strings.TrimSpace(string(content))
	if value == "" {
		return "", fmt.Errorf("master key file %q is empty", path)
	}

	return value, nil
}

func gitStoredCredentialCapabilityEnabled(
	gitCredentialCfg GitCredentialRuntimeConfig,
	persistenceCfg PersistenceRuntimeConfig,
) bool {
	return gitCredentialCfg.EnableStoredCredentials &&
		persistenceCfg.Enabled &&
		gitCredentialCfg.MasterKeySource != GitCredentialMasterKeySourceNone &&
		strings.TrimSpace(gitCredentialCfg.MasterKey) != ""
}
