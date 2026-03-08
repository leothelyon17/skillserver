package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitCredentialConfig_DefaultsStoredCredentialsDisabled(t *testing.T) {
	cfg, err := parseGitCredentialConfigForTest(nil, nil, nil)
	if err != nil {
		t.Fatalf("expected defaults to parse, got error: %v", err)
	}

	if cfg.EnableStoredCredentials {
		t.Fatalf("expected stored credentials to be disabled by default")
	}
	if cfg.MasterKeySource != GitCredentialMasterKeySourceNone {
		t.Fatalf("expected default master key source %q, got %q", GitCredentialMasterKeySourceNone, cfg.MasterKeySource)
	}
	if cfg.MasterKey != "" {
		t.Fatalf("expected default master key to be empty")
	}
	if cfg.MasterKeyFile != "" {
		t.Fatalf("expected default master key file to be empty")
	}
}

func TestGitCredentialConfig_EnvInlineMasterKey(t *testing.T) {
	cfg, err := parseGitCredentialConfigForTest(nil, map[string]string{
		envGitEnableStoredCredentials: "true",
		envGitCredentialMasterKey:     "  super-secret-master-key  ",
	}, nil)
	if err != nil {
		t.Fatalf("expected inline master key env config to parse, got error: %v", err)
	}

	if !cfg.EnableStoredCredentials {
		t.Fatalf("expected stored credentials enabled from env")
	}
	if cfg.MasterKeySource != GitCredentialMasterKeySourceInline {
		t.Fatalf("expected master key source %q, got %q", GitCredentialMasterKeySourceInline, cfg.MasterKeySource)
	}
	if cfg.MasterKey != "super-secret-master-key" {
		t.Fatalf("expected trimmed inline master key value, got %q", cfg.MasterKey)
	}
	if cfg.MasterKeyFile != "" {
		t.Fatalf("expected empty master key file for inline key source")
	}
}

func TestGitCredentialConfig_EnvMasterKeyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "git-master.key")
	if err := os.WriteFile(keyPath, []byte("  file-backed-master-key  \n"), 0o600); err != nil {
		t.Fatalf("failed to write key fixture: %v", err)
	}

	cfg, err := parseGitCredentialConfigForTest(nil, map[string]string{
		envGitEnableStoredCredentials: "true",
		envGitCredentialMasterKeyFile: keyPath,
	}, nil)
	if err != nil {
		t.Fatalf("expected file master key env config to parse, got error: %v", err)
	}

	expectedPath, err := filepath.Abs(keyPath)
	if err != nil {
		t.Fatalf("failed resolving expected key path: %v", err)
	}
	if !cfg.EnableStoredCredentials {
		t.Fatalf("expected stored credentials enabled from env")
	}
	if cfg.MasterKeySource != GitCredentialMasterKeySourceFile {
		t.Fatalf("expected master key source %q, got %q", GitCredentialMasterKeySourceFile, cfg.MasterKeySource)
	}
	if cfg.MasterKey != "file-backed-master-key" {
		t.Fatalf("expected trimmed file-backed key, got %q", cfg.MasterKey)
	}
	if cfg.MasterKeyFile != expectedPath {
		t.Fatalf("expected resolved key file path %q, got %q", expectedPath, cfg.MasterKeyFile)
	}
}

func TestGitCredentialConfig_EnabledWithoutMasterKeyFails(t *testing.T) {
	_, err := parseGitCredentialConfigForTest(nil, map[string]string{
		envGitEnableStoredCredentials: "true",
	}, nil)
	if err == nil {
		t.Fatalf("expected missing master key error, got nil")
	}
	if !strings.Contains(err.Error(), envGitCredentialMasterKey) || !strings.Contains(err.Error(), envGitCredentialMasterKeyFile) {
		t.Fatalf("expected missing key error to reference required key env vars, got %v", err)
	}
}

func TestGitCredentialConfig_BothMasterKeySourcesFails(t *testing.T) {
	_, err := parseGitCredentialConfigForTest(nil, map[string]string{
		envGitEnableStoredCredentials: "true",
		envGitCredentialMasterKey:     "inline-value",
		envGitCredentialMasterKeyFile: "/tmp/master.key",
	}, nil)
	if err == nil {
		t.Fatalf("expected mutually exclusive key source error, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually exclusive key source error, got %v", err)
	}
}

func TestValidateGitCredentialStartupConfig_MissingPersistenceFails(t *testing.T) {
	err := validateGitCredentialStartupConfig(
		GitCredentialRuntimeConfig{
			EnableStoredCredentials: true,
			MasterKeySource:         GitCredentialMasterKeySourceInline,
			MasterKey:               "runtime-master-key",
		},
		PersistenceRuntimeConfig{
			Enabled: false,
		},
	)
	if err == nil {
		t.Fatalf("expected missing persistence startup error, got nil")
	}
	if !strings.Contains(err.Error(), envPersistenceData) {
		t.Fatalf("expected missing persistence error to reference %s, got %v", envPersistenceData, err)
	}
}

func TestValidateGitCredentialStartupConfig_ValidConfiguration(t *testing.T) {
	err := validateGitCredentialStartupConfig(
		GitCredentialRuntimeConfig{
			EnableStoredCredentials: true,
			MasterKeySource:         GitCredentialMasterKeySourceInline,
			MasterKey:               "runtime-master-key",
		},
		PersistenceRuntimeConfig{
			Enabled: true,
			Dir:     t.TempDir(),
			DBPath:  filepath.Join(t.TempDir(), "skillserver.db"),
		},
	)
	if err != nil {
		t.Fatalf("expected valid git credential startup config, got %v", err)
	}
}

func TestGitStoredCredentialCapabilityEnabled(t *testing.T) {
	cfg := GitCredentialRuntimeConfig{
		EnableStoredCredentials: true,
		MasterKeySource:         GitCredentialMasterKeySourceInline,
		MasterKey:               "runtime-master-key",
	}
	persistenceCfg := PersistenceRuntimeConfig{Enabled: true}

	if !gitStoredCredentialCapabilityEnabled(cfg, persistenceCfg) {
		t.Fatalf("expected capability to be enabled when all prerequisites are present")
	}

	if gitStoredCredentialCapabilityEnabled(cfg, PersistenceRuntimeConfig{Enabled: false}) {
		t.Fatalf("expected capability disabled when persistence is off")
	}
}

func parseGitCredentialConfigForTest(
	args []string,
	env map[string]string,
	readFile func(string) ([]byte, error),
) (GitCredentialRuntimeConfig, error) {
	fs := flag.NewFlagSet("git-credential-config-test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	flags := registerGitCredentialRuntimeFlags(fs)
	if err := fs.Parse(args); err != nil {
		return GitCredentialRuntimeConfig{}, err
	}

	return parseGitCredentialRuntimeConfig(fs, flags, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}, readFile)
}
