package main

import (
	"flag"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMCPConfig_Defaults(t *testing.T) {
	cfg, err := parseMCPConfigForTest(nil, nil)
	if err != nil {
		t.Fatalf("expected defaults to parse, got error: %v", err)
	}

	if cfg.Transport != defaultMCPTransportMode {
		t.Fatalf("expected default transport %q, got %q", defaultMCPTransportMode, cfg.Transport)
	}
	if cfg.HTTPPath != defaultMCPHTTPPath {
		t.Fatalf("expected default path %q, got %q", defaultMCPHTTPPath, cfg.HTTPPath)
	}
	if cfg.SessionTimeout != defaultMCPSessionTimeout {
		t.Fatalf("expected default session timeout %v, got %v", defaultMCPSessionTimeout, cfg.SessionTimeout)
	}
	if cfg.Stateless != defaultMCPStateless {
		t.Fatalf("expected default stateless %v, got %v", defaultMCPStateless, cfg.Stateless)
	}
	if cfg.EnableWrites != defaultMCPEnableWrites {
		t.Fatalf("expected default enable writes %v, got %v", defaultMCPEnableWrites, cfg.EnableWrites)
	}
	if cfg.EnableMaterialization != defaultMCPEnableMaterialization {
		t.Fatalf(
			"expected default enable materialization %v, got %v",
			defaultMCPEnableMaterialization,
			cfg.EnableMaterialization,
		)
	}
	if len(cfg.AllowedDestinationRoots) != 0 {
		t.Fatalf("expected no default allowed destination roots, got %v", cfg.AllowedDestinationRoots)
	}
	if cfg.EnableEventStore != defaultMCPEnableEventStore {
		t.Fatalf("expected default event store enabled %v, got %v", defaultMCPEnableEventStore, cfg.EnableEventStore)
	}
	if cfg.EventStoreMaxBytes != defaultMCPEventStoreMaxBytes {
		t.Fatalf("expected default max bytes %d, got %d", defaultMCPEventStoreMaxBytes, cfg.EventStoreMaxBytes)
	}
}

func TestMCPConfig_EnvOverrides(t *testing.T) {
	env := map[string]string{
		envMCPTransport:               "http",
		envMCPHTTPPath:                "/remote-mcp",
		envMCPSessionTimeout:          "45m",
		envMCPStateless:               "true",
		envMCPEnableWrites:            "true",
		envMCPEnableMaterialization:   "true",
		envMCPAllowedDestinationRoots: "/workspace,/projects,/workspace",
		envMCPEnableEventStore:        "false",
		envMCPEventStoreMaxBytes:      "2097152",
	}

	cfg, err := parseMCPConfigForTest(nil, env)
	if err != nil {
		t.Fatalf("expected env overrides to parse, got error: %v", err)
	}

	if cfg.Transport != MCPTransportHTTP {
		t.Fatalf("expected env transport %q, got %q", MCPTransportHTTP, cfg.Transport)
	}
	if cfg.HTTPPath != "/remote-mcp" {
		t.Fatalf("expected env path %q, got %q", "/remote-mcp", cfg.HTTPPath)
	}
	if cfg.SessionTimeout != 45*time.Minute {
		t.Fatalf("expected env timeout %v, got %v", 45*time.Minute, cfg.SessionTimeout)
	}
	if !cfg.Stateless {
		t.Fatalf("expected stateless true from env")
	}
	if !cfg.EnableWrites {
		t.Fatalf("expected enable writes true from env")
	}
	if !cfg.EnableMaterialization {
		t.Fatalf("expected enable materialization true from env")
	}
	expectedRoots := []string{"/workspace", "/projects"}
	if !reflect.DeepEqual(cfg.AllowedDestinationRoots, expectedRoots) {
		t.Fatalf("expected allowed destination roots %v, got %v", expectedRoots, cfg.AllowedDestinationRoots)
	}
	if cfg.EnableEventStore {
		t.Fatalf("expected event store false from env")
	}
	if cfg.EventStoreMaxBytes != 2097152 {
		t.Fatalf("expected env max bytes %d, got %d", 2097152, cfg.EventStoreMaxBytes)
	}
}

func TestMCPConfig_FlagPrecedence(t *testing.T) {
	env := map[string]string{
		envMCPTransport:               "http",
		envMCPHTTPPath:                "/env-mcp",
		envMCPSessionTimeout:          "1h",
		envMCPStateless:               "false",
		envMCPEnableWrites:            "false",
		envMCPEnableMaterialization:   "false",
		envMCPAllowedDestinationRoots: "/env-workspace,/env-projects",
		envMCPEnableEventStore:        "true",
		envMCPEventStoreMaxBytes:      "1024",
	}

	args := []string{
		"--mcp-transport=stdio",
		"--mcp-http-path=/flag-mcp",
		"--mcp-session-timeout=15m",
		"--mcp-stateless=true",
		"--mcp-enable-writes=true",
		"--mcp-enable-materialization=true",
		"--mcp-allowed-destination-roots=/flag-workspace,/flag-projects,/flag-workspace",
		"--mcp-enable-event-store=false",
		"--mcp-event-store-max-bytes=2048",
	}

	cfg, err := parseMCPConfigForTest(args, env)
	if err != nil {
		t.Fatalf("expected flags to override env values, got error: %v", err)
	}

	if cfg.Transport != MCPTransportStdio {
		t.Fatalf("expected flag transport %q, got %q", MCPTransportStdio, cfg.Transport)
	}
	if cfg.HTTPPath != "/flag-mcp" {
		t.Fatalf("expected flag path %q, got %q", "/flag-mcp", cfg.HTTPPath)
	}
	if cfg.SessionTimeout != 15*time.Minute {
		t.Fatalf("expected flag timeout %v, got %v", 15*time.Minute, cfg.SessionTimeout)
	}
	if !cfg.Stateless {
		t.Fatalf("expected stateless true from flag")
	}
	if !cfg.EnableWrites {
		t.Fatalf("expected enable writes true from flag")
	}
	if !cfg.EnableMaterialization {
		t.Fatalf("expected enable materialization true from flag")
	}
	expectedRoots := []string{"/flag-workspace", "/flag-projects"}
	if !reflect.DeepEqual(cfg.AllowedDestinationRoots, expectedRoots) {
		t.Fatalf("expected allowed destination roots %v, got %v", expectedRoots, cfg.AllowedDestinationRoots)
	}
	if cfg.EnableEventStore {
		t.Fatalf("expected event store false from flag")
	}
	if cfg.EventStoreMaxBytes != 2048 {
		t.Fatalf("expected flag max bytes %d, got %d", 2048, cfg.EventStoreMaxBytes)
	}
}

func TestMCPConfig_InvalidTransport(t *testing.T) {
	_, err := parseMCPConfigForTest(nil, map[string]string{
		envMCPTransport: "invalid-mode",
	})
	if err == nil {
		t.Fatalf("expected invalid transport error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid MCP transport mode") {
		t.Fatalf("expected transport validation error, got: %v", err)
	}
}

func TestMCPConfig_InvalidPath(t *testing.T) {
	_, err := parseMCPConfigForTest(nil, map[string]string{
		envMCPHTTPPath: "relative/path",
	})
	if err == nil {
		t.Fatalf("expected invalid path error, got nil")
	}
	if !strings.Contains(err.Error(), "MCP HTTP path must be absolute") {
		t.Fatalf("expected path validation error, got: %v", err)
	}
}

func TestMCPConfig_InvalidSessionTimeout(t *testing.T) {
	_, err := parseMCPConfigForTest(nil, map[string]string{
		envMCPSessionTimeout: "not-a-duration",
	})
	if err == nil {
		t.Fatalf("expected invalid session timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid MCP session timeout") {
		t.Fatalf("expected session timeout validation error, got: %v", err)
	}
}

func TestMCPConfig_InvalidEventStoreMaxBytes(t *testing.T) {
	_, err := parseMCPConfigForTest(nil, map[string]string{
		envMCPEventStoreMaxBytes: "-1",
	})
	if err == nil {
		t.Fatalf("expected invalid max bytes error, got nil")
	}
	if !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("expected max bytes validation error, got: %v", err)
	}
}

func TestMCPConfig_InvalidEnableWritesBoolean(t *testing.T) {
	_, err := parseMCPConfigForTest(nil, map[string]string{
		envMCPEnableWrites: "definitely",
	})
	if err == nil {
		t.Fatalf("expected invalid enable writes boolean error, got nil")
	}
	if !strings.Contains(err.Error(), "must be a boolean") {
		t.Fatalf("expected boolean validation error, got: %v", err)
	}
}

func TestMCPConfig_InvalidAllowedDestinationRootsRelativePath(t *testing.T) {
	_, err := parseMCPConfigForTest(nil, map[string]string{
		envMCPAllowedDestinationRoots: "relative/path",
	})
	if err == nil {
		t.Fatalf("expected invalid allowed destination roots error, got nil")
	}
	if !strings.Contains(err.Error(), "must be absolute") {
		t.Fatalf("expected absolute path validation error, got: %v", err)
	}
}

func TestMCPConfig_InvalidAllowedDestinationRootsEmptyEntry(t *testing.T) {
	_, err := parseMCPConfigForTest(nil, map[string]string{
		envMCPAllowedDestinationRoots: "/workspace, ,/projects",
	})
	if err == nil {
		t.Fatalf("expected empty entry validation error, got nil")
	}
	if !strings.Contains(err.Error(), "empty path value") {
		t.Fatalf("expected empty path validation error, got: %v", err)
	}
}

func TestMCPConfig_EnableMaterializationRequiresAllowedDestinationRoots(t *testing.T) {
	_, err := parseMCPConfigForTest(nil, map[string]string{
		envMCPEnableMaterialization: "true",
	})
	if err == nil {
		t.Fatalf("expected missing allowed destination roots error, got nil")
	}
	if !strings.Contains(err.Error(), "requires at least one allowed destination root") {
		t.Fatalf("expected materialization roots validation error, got: %v", err)
	}
}

func TestCatalogConfig_Defaults(t *testing.T) {
	cfg, err := parseCatalogConfigForTest(nil, nil)
	if err != nil {
		t.Fatalf("expected defaults to parse, got error: %v", err)
	}

	if cfg.EnablePrompts != defaultCatalogEnablePrompts {
		t.Fatalf("expected default enable prompts %v, got %v", defaultCatalogEnablePrompts, cfg.EnablePrompts)
	}
	if !reflect.DeepEqual(cfg.PromptDirectoryAllowlist, defaultCatalogPromptDirectoryAllowlist) {
		t.Fatalf("expected default prompt dirs %v, got %v", defaultCatalogPromptDirectoryAllowlist, cfg.PromptDirectoryAllowlist)
	}
	if cfg.EnableRules != defaultCatalogEnableRules {
		t.Fatalf("expected default enable rules %v, got %v", defaultCatalogEnableRules, cfg.EnableRules)
	}
	if !reflect.DeepEqual(cfg.RuleDirectoryAllowlist, defaultCatalogRuleDirectoryAllowlist) {
		t.Fatalf("expected default rule dirs %v, got %v", defaultCatalogRuleDirectoryAllowlist, cfg.RuleDirectoryAllowlist)
	}
	if !reflect.DeepEqual(cfg.RuleFilenameAllowlist, defaultCatalogRuleFilenameAllowlist) {
		t.Fatalf("expected default rule filenames %v, got %v", defaultCatalogRuleFilenameAllowlist, cfg.RuleFilenameAllowlist)
	}
}

func TestCatalogConfig_EnvOverrides(t *testing.T) {
	env := map[string]string{
		envCatalogEnablePrompts: "false",
		envCatalogPromptDirs:    " prompts , /agents/ , prompt , prompts ",
		envCatalogEnableRules:   "false",
		envCatalogRuleDirs:      " rules , /governance/ , rule , rules ",
		envCatalogRuleFilenames: " AGENTS.md , claude.md , rules.markdown , agents.md ",
	}

	cfg, err := parseCatalogConfigForTest(nil, env)
	if err != nil {
		t.Fatalf("expected env overrides to parse, got error: %v", err)
	}

	if cfg.EnablePrompts {
		t.Fatalf("expected env enable prompts false")
	}
	expectedDirs := []string{"prompts", "agents", "prompt"}
	if !reflect.DeepEqual(cfg.PromptDirectoryAllowlist, expectedDirs) {
		t.Fatalf("expected prompt dirs %v, got %v", expectedDirs, cfg.PromptDirectoryAllowlist)
	}
	if cfg.EnableRules {
		t.Fatalf("expected env enable rules false")
	}
	expectedRuleDirs := []string{"rules", "governance", "rule"}
	if !reflect.DeepEqual(cfg.RuleDirectoryAllowlist, expectedRuleDirs) {
		t.Fatalf("expected rule dirs %v, got %v", expectedRuleDirs, cfg.RuleDirectoryAllowlist)
	}
	expectedRuleFilenames := []string{"agents.md", "claude.md", "rules.markdown"}
	if !reflect.DeepEqual(cfg.RuleFilenameAllowlist, expectedRuleFilenames) {
		t.Fatalf("expected rule filenames %v, got %v", expectedRuleFilenames, cfg.RuleFilenameAllowlist)
	}
}

func TestCatalogConfig_FlagPrecedence(t *testing.T) {
	env := map[string]string{
		envCatalogEnablePrompts: "true",
		envCatalogPromptDirs:    "prompts,agents",
		envCatalogEnableRules:   "true",
		envCatalogRuleDirs:      "rules,policy",
		envCatalogRuleFilenames: "agents.md,rules.md",
	}
	args := []string{
		"--catalog-enable-prompts=false",
		"--catalog-prompt-dirs=agent,prompt",
		"--catalog-enable-rules=false",
		"--catalog-rule-dirs=governance,rule",
		"--catalog-rule-filenames=claude.md,gemini.md",
	}

	cfg, err := parseCatalogConfigForTest(args, env)
	if err != nil {
		t.Fatalf("expected flags to override env values, got error: %v", err)
	}

	if cfg.EnablePrompts {
		t.Fatalf("expected enable prompts false from flag")
	}
	expectedDirs := []string{"agent", "prompt"}
	if !reflect.DeepEqual(cfg.PromptDirectoryAllowlist, expectedDirs) {
		t.Fatalf("expected prompt dirs %v, got %v", expectedDirs, cfg.PromptDirectoryAllowlist)
	}
	if cfg.EnableRules {
		t.Fatalf("expected enable rules false from flag")
	}
	expectedRuleDirs := []string{"governance", "rule"}
	if !reflect.DeepEqual(cfg.RuleDirectoryAllowlist, expectedRuleDirs) {
		t.Fatalf("expected rule dirs %v, got %v", expectedRuleDirs, cfg.RuleDirectoryAllowlist)
	}
	expectedRuleFilenames := []string{"claude.md", "gemini.md"}
	if !reflect.DeepEqual(cfg.RuleFilenameAllowlist, expectedRuleFilenames) {
		t.Fatalf("expected rule filenames %v, got %v", expectedRuleFilenames, cfg.RuleFilenameAllowlist)
	}
}

func TestCatalogConfig_InvalidPromptDirs(t *testing.T) {
	_, err := parseCatalogConfigForTest(nil, map[string]string{
		envCatalogPromptDirs: "prompts,nested/path",
	})
	if err == nil {
		t.Fatalf("expected invalid prompt dirs error, got nil")
	}
	if !strings.Contains(err.Error(), "must be a single directory name") {
		t.Fatalf("expected actionable prompt dirs validation error, got: %v", err)
	}
}

func TestCatalogConfig_EmptyPromptDirs(t *testing.T) {
	_, err := parseCatalogConfigForTest(nil, map[string]string{
		envCatalogPromptDirs: " , ",
	})
	if err == nil {
		t.Fatalf("expected empty prompt dirs error, got nil")
	}
	if !strings.Contains(err.Error(), "must include at least one directory name") {
		t.Fatalf("expected empty prompt dirs validation error, got: %v", err)
	}
}

func TestCatalogConfig_InvalidRuleDirs(t *testing.T) {
	_, err := parseCatalogConfigForTest(nil, map[string]string{
		envCatalogRuleDirs: "rules,nested/path",
	})
	if err == nil {
		t.Fatalf("expected invalid rule dirs error, got nil")
	}
	if !strings.Contains(err.Error(), "must be a single directory name") {
		t.Fatalf("expected actionable rule dirs validation error, got: %v", err)
	}
}

func TestCatalogConfig_InvalidRuleFilenames(t *testing.T) {
	_, err := parseCatalogConfigForTest(nil, map[string]string{
		envCatalogRuleFilenames: "AGENTS.md,not-markdown.txt",
	})
	if err == nil {
		t.Fatalf("expected invalid rule filenames error, got nil")
	}
	if !strings.Contains(err.Error(), "must be a markdown filename") {
		t.Fatalf("expected actionable rule filename validation error, got: %v", err)
	}
}

func TestCatalogConfig_EmptyRuleFilenames(t *testing.T) {
	_, err := parseCatalogConfigForTest(nil, map[string]string{
		envCatalogRuleFilenames: " , ",
	})
	if err == nil {
		t.Fatalf("expected empty rule filenames error, got nil")
	}
	if !strings.Contains(err.Error(), "must include at least one markdown filename") {
		t.Fatalf("expected empty rule filename validation error, got: %v", err)
	}
}

func TestPersistenceConfig_Defaults(t *testing.T) {
	cfg, err := parsePersistenceConfigForTest(nil, nil)
	if err != nil {
		t.Fatalf("expected defaults to parse, got error: %v", err)
	}

	if cfg.Enabled {
		t.Fatalf("expected persistence disabled by default")
	}
	if cfg.Dir != "" {
		t.Fatalf("expected default persistence dir to be empty, got %q", cfg.Dir)
	}
	if cfg.DBPath != "" {
		t.Fatalf("expected default persistence DB path to be empty, got %q", cfg.DBPath)
	}
}

func TestPersistenceConfig_EnvOverrides(t *testing.T) {
	dir := t.TempDir()

	cfg, err := parsePersistenceConfigForTest(nil, map[string]string{
		envPersistenceData: "true",
		envPersistenceDir:  dir,
	})
	if err != nil {
		t.Fatalf("expected env overrides to parse, got error: %v", err)
	}

	expectedDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("failed to resolve expected abs dir: %v", err)
	}
	expectedDBPath := filepath.Join(expectedDir, defaultPersistenceDatabaseFileName)

	if !cfg.Enabled {
		t.Fatalf("expected persistence enabled from env")
	}
	if cfg.Dir != expectedDir {
		t.Fatalf("expected persistence dir %q, got %q", expectedDir, cfg.Dir)
	}
	if cfg.DBPath != expectedDBPath {
		t.Fatalf("expected persistence DB path %q, got %q", expectedDBPath, cfg.DBPath)
	}
}

func TestPersistenceConfig_EnvRelativeDBPathResolvesFromDir(t *testing.T) {
	dir := t.TempDir()

	cfg, err := parsePersistenceConfigForTest(nil, map[string]string{
		envPersistenceData:   "true",
		envPersistenceDir:    dir,
		envPersistenceDBPath: "state/skillserver.sqlite",
	})
	if err != nil {
		t.Fatalf("expected relative DB path to parse, got error: %v", err)
	}

	expectedDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("failed to resolve expected abs dir: %v", err)
	}
	expectedDBPath := filepath.Join(expectedDir, "state", "skillserver.sqlite")

	if cfg.DBPath != expectedDBPath {
		t.Fatalf("expected persistence DB path %q, got %q", expectedDBPath, cfg.DBPath)
	}
}

func TestPersistenceConfig_FlagPrecedence(t *testing.T) {
	envDir := t.TempDir()
	flagDir := t.TempDir()

	args := []string{
		"--persistence-data=true",
		"--persistence-dir=" + flagDir,
		"--persistence-db-path=runtime.sqlite",
	}
	env := map[string]string{
		envPersistenceData:   "false",
		envPersistenceDir:    envDir,
		envPersistenceDBPath: "/tmp/env.sqlite",
	}

	cfg, err := parsePersistenceConfigForTest(args, env)
	if err != nil {
		t.Fatalf("expected flags to override env values, got error: %v", err)
	}

	expectedDir, err := filepath.Abs(flagDir)
	if err != nil {
		t.Fatalf("failed to resolve expected abs dir: %v", err)
	}
	expectedDBPath := filepath.Join(expectedDir, "runtime.sqlite")

	if !cfg.Enabled {
		t.Fatalf("expected persistence enabled from flag")
	}
	if cfg.Dir != expectedDir {
		t.Fatalf("expected persistence dir %q, got %q", expectedDir, cfg.Dir)
	}
	if cfg.DBPath != expectedDBPath {
		t.Fatalf("expected persistence DB path %q, got %q", expectedDBPath, cfg.DBPath)
	}
}

func TestPersistenceConfig_InvalidBoolean(t *testing.T) {
	_, err := parsePersistenceConfigForTest(nil, map[string]string{
		envPersistenceData: "definitely",
	})
	if err == nil {
		t.Fatalf("expected invalid boolean error, got nil")
	}
	if !strings.Contains(err.Error(), "must be a boolean") {
		t.Fatalf("expected boolean validation error, got: %v", err)
	}
}

func TestPersistenceConfig_EnabledWithoutDirectory(t *testing.T) {
	_, err := parsePersistenceConfigForTest(nil, map[string]string{
		envPersistenceData: "true",
	})
	if err == nil {
		t.Fatalf("expected missing persistence directory error, got nil")
	}
	if !strings.Contains(err.Error(), "persistence directory is required") {
		t.Fatalf("expected missing directory error, got: %v", err)
	}
}

func TestPersistenceConfig_DisabledIgnoresPathValues(t *testing.T) {
	cfg, err := parsePersistenceConfigForTest(nil, map[string]string{
		envPersistenceData:   "false",
		envPersistenceDir:    "/does/not/matter",
		envPersistenceDBPath: "/also/ignored.sqlite",
	})
	if err != nil {
		t.Fatalf("expected disabled config to parse, got error: %v", err)
	}

	if cfg.Enabled {
		t.Fatalf("expected persistence disabled")
	}
	if cfg.Dir != "" || cfg.DBPath != "" {
		t.Fatalf("expected disabled config to avoid resolving dir/db path, got dir=%q db=%q", cfg.Dir, cfg.DBPath)
	}
}

func parseMCPConfigForTest(args []string, env map[string]string) (MCPRuntimeConfig, error) {
	fs := flag.NewFlagSet("mcp-config-test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	flags := registerMCPRuntimeFlags(fs)
	if err := fs.Parse(args); err != nil {
		return MCPRuntimeConfig{}, err
	}

	return parseMCPRuntimeConfig(fs, flags, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
}

func parseCatalogConfigForTest(args []string, env map[string]string) (CatalogRuntimeConfig, error) {
	fs := flag.NewFlagSet("catalog-config-test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	flags := registerCatalogRuntimeFlags(fs)
	if err := fs.Parse(args); err != nil {
		return CatalogRuntimeConfig{}, err
	}

	return parseCatalogRuntimeConfig(fs, flags, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
}

func parsePersistenceConfigForTest(args []string, env map[string]string) (PersistenceRuntimeConfig, error) {
	fs := flag.NewFlagSet("persistence-config-test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	flags := registerPersistenceRuntimeFlags(fs)
	if err := fs.Parse(args); err != nil {
		return PersistenceRuntimeConfig{}, err
	}

	return parsePersistenceRuntimeConfig(fs, flags, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
}
