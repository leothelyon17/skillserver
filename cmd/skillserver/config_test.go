package main

import (
	"flag"
	"io"
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
	if cfg.EnableEventStore != defaultMCPEnableEventStore {
		t.Fatalf("expected default event store enabled %v, got %v", defaultMCPEnableEventStore, cfg.EnableEventStore)
	}
	if cfg.EventStoreMaxBytes != defaultMCPEventStoreMaxBytes {
		t.Fatalf("expected default max bytes %d, got %d", defaultMCPEventStoreMaxBytes, cfg.EventStoreMaxBytes)
	}
}

func TestMCPConfig_EnvOverrides(t *testing.T) {
	env := map[string]string{
		envMCPTransport:          "http",
		envMCPHTTPPath:           "/remote-mcp",
		envMCPSessionTimeout:     "45m",
		envMCPStateless:          "true",
		envMCPEnableEventStore:   "false",
		envMCPEventStoreMaxBytes: "2097152",
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
	if cfg.EnableEventStore {
		t.Fatalf("expected event store false from env")
	}
	if cfg.EventStoreMaxBytes != 2097152 {
		t.Fatalf("expected env max bytes %d, got %d", 2097152, cfg.EventStoreMaxBytes)
	}
}

func TestMCPConfig_FlagPrecedence(t *testing.T) {
	env := map[string]string{
		envMCPTransport:          "http",
		envMCPHTTPPath:           "/env-mcp",
		envMCPSessionTimeout:     "1h",
		envMCPStateless:          "false",
		envMCPEnableEventStore:   "true",
		envMCPEventStoreMaxBytes: "1024",
	}

	args := []string{
		"--mcp-transport=stdio",
		"--mcp-http-path=/flag-mcp",
		"--mcp-session-timeout=15m",
		"--mcp-stateless=true",
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
