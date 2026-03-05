package main

import (
	"flag"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/mudler/skillserver/pkg/domain"
)

const (
	envMCPTransport          = "SKILLSERVER_MCP_TRANSPORT"
	envMCPHTTPPath           = "SKILLSERVER_MCP_HTTP_PATH"
	envMCPSessionTimeout     = "SKILLSERVER_MCP_SESSION_TIMEOUT"
	envMCPStateless          = "SKILLSERVER_MCP_STATELESS"
	envMCPEnableWrites       = "SKILLSERVER_MCP_ENABLE_WRITES"
	envMCPEnableEventStore   = "SKILLSERVER_MCP_ENABLE_EVENT_STORE"
	envMCPEventStoreMaxBytes = "SKILLSERVER_MCP_EVENT_STORE_MAX_BYTES"
	envCatalogEnablePrompts  = "SKILLSERVER_CATALOG_ENABLE_PROMPTS"
	envCatalogPromptDirs     = "SKILLSERVER_CATALOG_PROMPT_DIRS"
)

const (
	defaultMCPTransportMode            = MCPTransportBoth
	defaultMCPHTTPPath                 = "/mcp"
	defaultMCPSessionTimeout           = 30 * time.Minute
	defaultMCPSessionTimeoutString     = "30m"
	defaultMCPStateless                = false
	defaultMCPEnableWrites             = false
	defaultMCPEnableEventStore         = true
	defaultMCPEventStoreMaxBytes   int = 10 * 1024 * 1024
	defaultCatalogEnablePrompts        = true
)

var (
	defaultCatalogPromptDirectoryAllowlist = domain.DefaultPromptDirectoryAllowlist()
	defaultCatalogPromptDirectoryCSV       = strings.Join(defaultCatalogPromptDirectoryAllowlist, ",")
)

// MCPTransportMode controls which MCP transports are enabled at runtime.
type MCPTransportMode string

const (
	// MCPTransportStdio enables stdio transport only.
	MCPTransportStdio MCPTransportMode = "stdio"
	// MCPTransportHTTP enables HTTP transport only.
	MCPTransportHTTP MCPTransportMode = "http"
	// MCPTransportBoth enables both stdio and HTTP transports.
	MCPTransportBoth MCPTransportMode = "both"
)

// MCPRuntimeConfig defines runtime configuration for MCP transports.
type MCPRuntimeConfig struct {
	Transport          MCPTransportMode
	HTTPPath           string
	SessionTimeout     time.Duration
	Stateless          bool
	EnableWrites       bool
	EnableEventStore   bool
	EventStoreMaxBytes int
}

type mcpRuntimeFlagValues struct {
	transport          string
	httpPath           string
	sessionTimeout     string
	stateless          bool
	enableWrites       bool
	enableEventStore   bool
	eventStoreMaxBytes int
}

// CatalogRuntimeConfig defines runtime configuration for prompt catalog behavior.
type CatalogRuntimeConfig struct {
	EnablePrompts            bool
	PromptDirectoryAllowlist []string
}

type catalogRuntimeFlagValues struct {
	enablePrompts bool
	promptDirs    string
}

// registerMCPRuntimeFlags adds MCP runtime flags to a flag set.
func registerMCPRuntimeFlags(fs *flag.FlagSet) *mcpRuntimeFlagValues {
	values := &mcpRuntimeFlagValues{}

	fs.StringVar(
		&values.transport,
		"mcp-transport",
		string(defaultMCPTransportMode),
		"MCP transport mode: stdio|http|both (env: SKILLSERVER_MCP_TRANSPORT)",
	)
	fs.StringVar(
		&values.httpPath,
		"mcp-http-path",
		defaultMCPHTTPPath,
		"MCP Streamable HTTP path (env: SKILLSERVER_MCP_HTTP_PATH)",
	)
	fs.StringVar(
		&values.sessionTimeout,
		"mcp-session-timeout",
		defaultMCPSessionTimeoutString,
		"MCP session timeout duration (env: SKILLSERVER_MCP_SESSION_TIMEOUT)",
	)
	fs.BoolVar(
		&values.stateless,
		"mcp-stateless",
		defaultMCPStateless,
		"Enable stateless MCP HTTP mode (env: SKILLSERVER_MCP_STATELESS)",
	)
	fs.BoolVar(
		&values.enableWrites,
		"mcp-enable-writes",
		defaultMCPEnableWrites,
		"Enable MCP taxonomy write tools (env: SKILLSERVER_MCP_ENABLE_WRITES)",
	)
	fs.BoolVar(
		&values.enableEventStore,
		"mcp-enable-event-store",
		defaultMCPEnableEventStore,
		"Enable in-memory MCP event store (env: SKILLSERVER_MCP_ENABLE_EVENT_STORE)",
	)
	fs.IntVar(
		&values.eventStoreMaxBytes,
		"mcp-event-store-max-bytes",
		defaultMCPEventStoreMaxBytes,
		"Max bytes for MCP in-memory event store (env: SKILLSERVER_MCP_EVENT_STORE_MAX_BYTES)",
	)

	return values
}

// registerCatalogRuntimeFlags adds catalog runtime flags to a flag set.
func registerCatalogRuntimeFlags(fs *flag.FlagSet) *catalogRuntimeFlagValues {
	values := &catalogRuntimeFlagValues{}

	fs.BoolVar(
		&values.enablePrompts,
		"catalog-enable-prompts",
		defaultCatalogEnablePrompts,
		"Enable prompt catalog classification/indexing (env: SKILLSERVER_CATALOG_ENABLE_PROMPTS)",
	)
	fs.StringVar(
		&values.promptDirs,
		"catalog-prompt-dirs",
		defaultCatalogPromptDirectoryCSV,
		"Comma-separated prompt directory names used for prompt catalog detection (env: SKILLSERVER_CATALOG_PROMPT_DIRS)",
	)

	return values
}

// parseMCPRuntimeConfig resolves and validates MCP runtime config with precedence:
// flags > environment variables > defaults.
func parseMCPRuntimeConfig(
	fs *flag.FlagSet,
	flagValues *mcpRuntimeFlagValues,
	lookupEnv func(string) (string, bool),
) (MCPRuntimeConfig, error) {
	if fs == nil {
		return MCPRuntimeConfig{}, fmt.Errorf("flag set is required")
	}
	if flagValues == nil {
		return MCPRuntimeConfig{}, fmt.Errorf("flag values are required")
	}

	transportRaw, transportSource := resolveStringConfigValue(
		fs,
		"mcp-transport",
		flagValues.transport,
		envMCPTransport,
		string(defaultMCPTransportMode),
		lookupEnv,
	)
	transport, err := parseMCPTransportMode(transportRaw)
	if err != nil {
		return MCPRuntimeConfig{}, fmt.Errorf("%s: %w", transportSource, err)
	}

	pathRaw, pathSource := resolveStringConfigValue(
		fs,
		"mcp-http-path",
		flagValues.httpPath,
		envMCPHTTPPath,
		defaultMCPHTTPPath,
		lookupEnv,
	)
	httpPath, err := parseMCPHTTPPath(pathRaw)
	if err != nil {
		return MCPRuntimeConfig{}, fmt.Errorf("%s: %w", pathSource, err)
	}

	timeoutRaw, timeoutSource := resolveStringConfigValue(
		fs,
		"mcp-session-timeout",
		flagValues.sessionTimeout,
		envMCPSessionTimeout,
		defaultMCPSessionTimeoutString,
		lookupEnv,
	)
	sessionTimeout, err := parseMCPSessionTimeout(timeoutRaw)
	if err != nil {
		return MCPRuntimeConfig{}, fmt.Errorf("%s: %w", timeoutSource, err)
	}

	stateless, err := resolveBoolConfigValue(
		fs,
		"mcp-stateless",
		flagValues.stateless,
		envMCPStateless,
		defaultMCPStateless,
		lookupEnv,
	)
	if err != nil {
		return MCPRuntimeConfig{}, err
	}

	enableWrites, err := resolveBoolConfigValue(
		fs,
		"mcp-enable-writes",
		flagValues.enableWrites,
		envMCPEnableWrites,
		defaultMCPEnableWrites,
		lookupEnv,
	)
	if err != nil {
		return MCPRuntimeConfig{}, err
	}

	enableEventStore, err := resolveBoolConfigValue(
		fs,
		"mcp-enable-event-store",
		flagValues.enableEventStore,
		envMCPEnableEventStore,
		defaultMCPEnableEventStore,
		lookupEnv,
	)
	if err != nil {
		return MCPRuntimeConfig{}, err
	}

	eventStoreMaxBytes, err := resolveIntConfigValue(
		fs,
		"mcp-event-store-max-bytes",
		flagValues.eventStoreMaxBytes,
		envMCPEventStoreMaxBytes,
		defaultMCPEventStoreMaxBytes,
		lookupEnv,
	)
	if err != nil {
		return MCPRuntimeConfig{}, err
	}

	return MCPRuntimeConfig{
		Transport:          transport,
		HTTPPath:           httpPath,
		SessionTimeout:     sessionTimeout,
		Stateless:          stateless,
		EnableWrites:       enableWrites,
		EnableEventStore:   enableEventStore,
		EventStoreMaxBytes: eventStoreMaxBytes,
	}, nil
}

// parseCatalogRuntimeConfig resolves and validates prompt catalog runtime config with precedence:
// flags > environment variables > defaults.
func parseCatalogRuntimeConfig(
	fs *flag.FlagSet,
	flagValues *catalogRuntimeFlagValues,
	lookupEnv func(string) (string, bool),
) (CatalogRuntimeConfig, error) {
	if fs == nil {
		return CatalogRuntimeConfig{}, fmt.Errorf("flag set is required")
	}
	if flagValues == nil {
		return CatalogRuntimeConfig{}, fmt.Errorf("flag values are required")
	}

	enablePrompts, err := resolveBoolConfigValue(
		fs,
		"catalog-enable-prompts",
		flagValues.enablePrompts,
		envCatalogEnablePrompts,
		defaultCatalogEnablePrompts,
		lookupEnv,
	)
	if err != nil {
		return CatalogRuntimeConfig{}, err
	}

	promptDirsRaw, promptDirsSource := resolveStringConfigValue(
		fs,
		"catalog-prompt-dirs",
		flagValues.promptDirs,
		envCatalogPromptDirs,
		defaultCatalogPromptDirectoryCSV,
		lookupEnv,
	)
	promptDirs, err := parseCatalogPromptDirectoryAllowlist(promptDirsRaw)
	if err != nil {
		return CatalogRuntimeConfig{}, fmt.Errorf("%s: %w", promptDirsSource, err)
	}

	return CatalogRuntimeConfig{
		EnablePrompts:            enablePrompts,
		PromptDirectoryAllowlist: promptDirs,
	}, nil
}

func parseCatalogPromptDirectoryAllowlist(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	normalized := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		original := strings.TrimSpace(part)
		if original == "" {
			continue
		}

		value := strings.ToLower(strings.Trim(original, "/"))
		if value == "" {
			return nil, fmt.Errorf("catalog prompt directories contain an empty directory value")
		}
		if value == "." || value == ".." {
			return nil, fmt.Errorf("catalog prompt directory %q is not allowed", original)
		}
		if strings.ContainsAny(value, `/\`) {
			return nil, fmt.Errorf("catalog prompt directory %q must be a single directory name", original)
		}

		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("catalog prompt directories must include at least one directory name")
	}

	return normalized, nil
}

func parseMCPTransportMode(raw string) (MCPTransportMode, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch MCPTransportMode(value) {
	case MCPTransportStdio, MCPTransportHTTP, MCPTransportBoth:
		return MCPTransportMode(value), nil
	default:
		return "", fmt.Errorf("invalid MCP transport mode %q (allowed: stdio|http|both)", raw)
	}
}

func parseMCPHTTPPath(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("MCP HTTP path cannot be empty")
	}
	if !path.IsAbs(value) {
		return "", fmt.Errorf("MCP HTTP path must be absolute (start with '/'), got %q", raw)
	}

	cleaned := path.Clean(value)
	if !path.IsAbs(cleaned) {
		return "", fmt.Errorf("MCP HTTP path must resolve to an absolute path, got %q", raw)
	}
	return cleaned, nil
}

func parseMCPSessionTimeout(raw string) (time.Duration, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("MCP session timeout cannot be empty")
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid MCP session timeout %q: %w", raw, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("MCP session timeout must be greater than zero, got %q", raw)
	}

	return duration, nil
}

func resolveStringConfigValue(
	fs *flag.FlagSet,
	flagName string,
	flagValue string,
	envKey string,
	defaultValue string,
	lookupEnv func(string) (string, bool),
) (string, string) {
	if isFlagSet(fs, flagName) {
		return strings.TrimSpace(flagValue), fmt.Sprintf("flag --%s", flagName)
	}

	if envValue, ok := lookupNonEmptyEnv(lookupEnv, envKey); ok {
		return envValue, fmt.Sprintf("env %s", envKey)
	}

	return defaultValue, "default"
}

func resolveBoolConfigValue(
	fs *flag.FlagSet,
	flagName string,
	flagValue bool,
	envKey string,
	defaultValue bool,
	lookupEnv func(string) (string, bool),
) (bool, error) {
	if isFlagSet(fs, flagName) {
		return flagValue, nil
	}

	if envValue, ok := lookupNonEmptyEnv(lookupEnv, envKey); ok {
		parsed, err := strconv.ParseBool(envValue)
		if err != nil {
			return false, fmt.Errorf("env %s must be a boolean (true|false), got %q", envKey, envValue)
		}
		return parsed, nil
	}

	return defaultValue, nil
}

func resolveIntConfigValue(
	fs *flag.FlagSet,
	flagName string,
	flagValue int,
	envKey string,
	defaultValue int,
	lookupEnv func(string) (string, bool),
) (int, error) {
	var (
		value  int
		source string
	)

	switch {
	case isFlagSet(fs, flagName):
		value = flagValue
		source = fmt.Sprintf("flag --%s", flagName)
	case hasNonEmptyEnv(lookupEnv, envKey):
		envValue, _ := lookupNonEmptyEnv(lookupEnv, envKey)
		parsed, err := strconv.Atoi(envValue)
		if err != nil {
			return 0, fmt.Errorf("env %s must be an integer, got %q", envKey, envValue)
		}
		value = parsed
		source = fmt.Sprintf("env %s", envKey)
	default:
		value = defaultValue
		source = "default"
	}

	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero for MCP event store max bytes, got %d", source, value)
	}
	return value, nil
}

func hasNonEmptyEnv(lookupEnv func(string) (string, bool), key string) bool {
	_, ok := lookupNonEmptyEnv(lookupEnv, key)
	return ok
}

func lookupNonEmptyEnv(lookupEnv func(string) (string, bool), key string) (string, bool) {
	if lookupEnv == nil {
		return "", false
	}

	value, ok := lookupEnv(key)
	if !ok {
		return "", false
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}

func isFlagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
