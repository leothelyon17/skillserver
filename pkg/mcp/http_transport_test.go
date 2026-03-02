package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestBuildStreamableHTTPOptions_WithEventStore(t *testing.T) {
	cfg := StreamableHTTPConfig{
		SessionTimeout:     45 * time.Minute,
		Stateless:          false,
		EnableEventStore:   true,
		EventStoreMaxBytes: 2 * 1024 * 1024,
	}

	opts := BuildStreamableHTTPOptions(cfg)

	if opts.SessionTimeout != cfg.SessionTimeout {
		t.Fatalf("expected session timeout %v, got %v", cfg.SessionTimeout, opts.SessionTimeout)
	}
	if opts.Stateless {
		t.Fatalf("expected stateless false, got true")
	}
	if opts.EventStore == nil {
		t.Fatalf("expected event store to be configured")
	}

	eventStore, ok := opts.EventStore.(*mcpsdk.MemoryEventStore)
	if !ok {
		t.Fatalf("expected memory event store, got %T", opts.EventStore)
	}
	if got := eventStore.MaxBytes(); got != cfg.EventStoreMaxBytes {
		t.Fatalf("expected event store max bytes %d, got %d", cfg.EventStoreMaxBytes, got)
	}
}

func TestBuildStreamableHTTPOptions_WithoutEventStore(t *testing.T) {
	cfg := StreamableHTTPConfig{
		SessionTimeout:     30 * time.Minute,
		Stateless:          false,
		EnableEventStore:   false,
		EventStoreMaxBytes: 4 * 1024 * 1024,
	}

	opts := BuildStreamableHTTPOptions(cfg)

	if opts.EventStore != nil {
		t.Fatalf("expected no event store when disabled, got %T", opts.EventStore)
	}
	if opts.SessionTimeout != cfg.SessionTimeout {
		t.Fatalf("expected session timeout %v, got %v", cfg.SessionTimeout, opts.SessionTimeout)
	}
	if opts.Stateless {
		t.Fatalf("expected stateless false, got true")
	}
}

func TestBuildStreamableHTTPOptions_Stateless(t *testing.T) {
	cfg := StreamableHTTPConfig{
		SessionTimeout:     15 * time.Minute,
		Stateless:          true,
		EnableEventStore:   false,
		EventStoreMaxBytes: 1024,
	}

	opts := BuildStreamableHTTPOptions(cfg)

	if !opts.Stateless {
		t.Fatalf("expected stateless true, got false")
	}
	if opts.SessionTimeout != cfg.SessionTimeout {
		t.Fatalf("expected session timeout %v, got %v", cfg.SessionTimeout, opts.SessionTimeout)
	}
}

func TestServer_NewStreamableHTTPHandler_ConfigPermutations(t *testing.T) {
	server := NewServer(nil)

	configs := []StreamableHTTPConfig{
		{
			SessionTimeout:     30 * time.Minute,
			Stateless:          false,
			EnableEventStore:   true,
			EventStoreMaxBytes: 10 * 1024 * 1024,
		},
		{
			SessionTimeout:     30 * time.Minute,
			Stateless:          false,
			EnableEventStore:   false,
			EventStoreMaxBytes: 10 * 1024 * 1024,
		},
		{
			SessionTimeout:     30 * time.Minute,
			Stateless:          true,
			EnableEventStore:   false,
			EventStoreMaxBytes: 10 * 1024 * 1024,
		},
	}

	for _, cfg := range configs {
		handler := server.NewStreamableHTTPHandler(cfg)
		if handler == nil {
			t.Fatalf("expected non-nil handler for config %+v", cfg)
		}
	}
}

func TestServer_RunStillUsesStdioTransport(t *testing.T) {
	server := &Server{}

	var capturedTransport mcpsdk.Transport
	expectedErr := errors.New("run sentinel")
	server.runWithTransport = func(_ context.Context, transport mcpsdk.Transport) error {
		capturedTransport = transport
		return expectedErr
	}

	err := server.Run(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected run error %v, got %v", expectedErr, err)
	}

	if capturedTransport == nil {
		t.Fatalf("expected transport to be captured")
	}
	if _, ok := capturedTransport.(*mcpsdk.StdioTransport); !ok {
		t.Fatalf("expected *mcp.StdioTransport, got %T", capturedTransport)
	}
}
