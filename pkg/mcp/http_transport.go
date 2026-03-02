package mcp

import (
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StreamableHTTPConfig contains MCP Streamable HTTP settings sourced from runtime config.
type StreamableHTTPConfig struct {
	SessionTimeout     time.Duration
	Stateless          bool
	EnableEventStore   bool
	EventStoreMaxBytes int
}

// BuildStreamableHTTPOptions maps runtime config to SDK streamable HTTP options.
func BuildStreamableHTTPOptions(config StreamableHTTPConfig) *mcp.StreamableHTTPOptions {
	options := &mcp.StreamableHTTPOptions{
		Stateless:      config.Stateless,
		SessionTimeout: config.SessionTimeout,
	}

	if !config.EnableEventStore {
		return options
	}

	store := mcp.NewMemoryEventStore(nil)
	if config.EventStoreMaxBytes > 0 {
		store.SetMaxBytes(config.EventStoreMaxBytes)
	}
	options.EventStore = store

	return options
}

// NewStreamableHTTPHandler creates a Streamable HTTP handler bound to this server instance.
func (s *Server) NewStreamableHTTPHandler(config StreamableHTTPConfig) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s.mcpServer
	}, BuildStreamableHTTPOptions(config))
}

// MCPServer returns the underlying go-sdk server.
func (s *Server) MCPServer() *mcp.Server {
	return s.mcpServer
}
