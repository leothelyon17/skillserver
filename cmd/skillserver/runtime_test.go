package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

const runtimeTestWait = time.Second

func TestRuntime_StartModeStdio(t *testing.T) {
	webServer := newFakeRuntimeWebServer()
	mcpServer := newFakeRuntimeMCPServer()
	gitSyncer := newFakeRuntimeGitSyncer()
	signalChan := make(chan os.Signal, 1)

	done := runRuntimeAsync(t, runtimeDependencies{
		logger:        log.New(io.Discard, "", 0),
		enableLogging: true,
		mcpConfig:     defaultRuntimeMCPConfig(MCPTransportStdio),
		port:          "8080",
		webServer:     webServer,
		mcpServer:     mcpServer,
		gitSyncer:     gitSyncer,
		signalChan:    signalChan,
	})

	waitForEvent(t, webServer.startCalled, "web server start")
	waitForEvent(t, mcpServer.runStarted, "mcp stdio start")

	signalChan <- syscall.SIGTERM
	waitForRuntimeExit(t, done)

	if got := webServer.startAddress(); got != ":8080" {
		t.Fatalf("expected web server address %q, got %q", ":8080", got)
	}
	if got := webServer.startCallCount(); got != 1 {
		t.Fatalf("expected web server start calls %d, got %d", 1, got)
	}
	if got := mcpServer.runCallCount(); got != 1 {
		t.Fatalf("expected mcp run calls %d, got %d", 1, got)
	}
	if got := webServer.shutdownCallCount(); got != 1 {
		t.Fatalf("expected web server shutdown calls %d, got %d", 1, got)
	}
	if got := gitSyncer.stopCallCount(); got != 1 {
		t.Fatalf("expected git syncer stop calls %d, got %d", 1, got)
	}
}

func TestRuntime_StartModeHTTP(t *testing.T) {
	webServer := newFakeRuntimeWebServer()
	mcpServer := newFakeRuntimeMCPServer()
	gitSyncer := newFakeRuntimeGitSyncer()
	signalChan := make(chan os.Signal, 1)

	done := runRuntimeAsync(t, runtimeDependencies{
		logger:        log.New(io.Discard, "", 0),
		enableLogging: true,
		mcpConfig:     defaultRuntimeMCPConfig(MCPTransportHTTP),
		port:          "8080",
		webServer:     webServer,
		mcpServer:     mcpServer,
		gitSyncer:     gitSyncer,
		signalChan:    signalChan,
	})

	waitForEvent(t, webServer.startCalled, "web server start")
	assertNoEvent(t, mcpServer.runStarted, "mcp stdio start", 100*time.Millisecond)

	signalChan <- syscall.SIGTERM
	waitForRuntimeExit(t, done)

	if got := mcpServer.runCallCount(); got != 0 {
		t.Fatalf("expected mcp run calls %d in http mode, got %d", 0, got)
	}
}

func TestRuntime_StartModeBoth(t *testing.T) {
	webServer := newFakeRuntimeWebServer()
	mcpServer := newFakeRuntimeMCPServer()
	gitSyncer := newFakeRuntimeGitSyncer()
	signalChan := make(chan os.Signal, 1)

	done := runRuntimeAsync(t, runtimeDependencies{
		logger:        log.New(io.Discard, "", 0),
		enableLogging: true,
		mcpConfig:     defaultRuntimeMCPConfig(MCPTransportBoth),
		port:          "8080",
		webServer:     webServer,
		mcpServer:     mcpServer,
		gitSyncer:     gitSyncer,
		signalChan:    signalChan,
	})

	waitForEvent(t, webServer.startCalled, "web server start")
	waitForEvent(t, mcpServer.runStarted, "mcp stdio start")

	signalChan <- syscall.SIGTERM
	waitForRuntimeExit(t, done)

	if got := mcpServer.runCallCount(); got != 1 {
		t.Fatalf("expected mcp run calls %d in both mode, got %d", 1, got)
	}
}

func TestRuntime_BothModeStdioExitDoesNotStopHTTP(t *testing.T) {
	webServer := newFakeRuntimeWebServer()
	mcpServer := newFakeRuntimeMCPServer()
	gitSyncer := newFakeRuntimeGitSyncer()
	signalChan := make(chan os.Signal, 1)

	done := runRuntimeAsync(t, runtimeDependencies{
		logger:        log.New(io.Discard, "", 0),
		enableLogging: true,
		mcpConfig:     defaultRuntimeMCPConfig(MCPTransportBoth),
		port:          "8080",
		webServer:     webServer,
		mcpServer:     mcpServer,
		gitSyncer:     gitSyncer,
		signalChan:    signalChan,
	})

	waitForEvent(t, webServer.startCalled, "web server start")
	waitForEvent(t, mcpServer.runStarted, "mcp stdio start")

	mcpServer.runErr <- io.EOF
	assertNoRuntimeExit(t, done, 150*time.Millisecond)

	if got := webServer.shutdownCallCount(); got != 0 {
		t.Fatalf("expected web server to remain running after stdio exit in both mode, shutdown calls=%d", got)
	}

	signalChan <- syscall.SIGTERM
	waitForRuntimeExit(t, done)
}

func TestRuntime_GracefulShutdown(t *testing.T) {
	webServer := newFakeRuntimeWebServer()
	mcpServer := newFakeRuntimeMCPServer()
	gitSyncer := newFakeRuntimeGitSyncer()
	signalChan := make(chan os.Signal, 1)

	done := runRuntimeAsync(t, runtimeDependencies{
		logger:        log.New(io.Discard, "", 0),
		enableLogging: true,
		mcpConfig:     defaultRuntimeMCPConfig(MCPTransportBoth),
		port:          "8080",
		webServer:     webServer,
		mcpServer:     mcpServer,
		gitSyncer:     gitSyncer,
		signalChan:    signalChan,
	})

	waitForEvent(t, webServer.startCalled, "web server start")
	waitForEvent(t, mcpServer.runStarted, "mcp stdio start")

	signalChan <- syscall.SIGTERM
	waitForRuntimeExit(t, done)
	waitForEvent(t, webServer.shutdownCalled, "web server shutdown")
	waitForEvent(t, mcpServer.ctxCanceled, "mcp context cancellation")
	waitForEvent(t, gitSyncer.stopped, "git syncer stop")

	if got := webServer.shutdownCallCount(); got != 1 {
		t.Fatalf("expected web server shutdown calls %d, got %d", 1, got)
	}
	if got := gitSyncer.stopCallCount(); got != 1 {
		t.Fatalf("expected git syncer stop calls %d, got %d", 1, got)
	}
}

func defaultRuntimeMCPConfig(mode MCPTransportMode) MCPRuntimeConfig {
	return MCPRuntimeConfig{
		Transport:          mode,
		HTTPPath:           defaultMCPHTTPPath,
		SessionTimeout:     defaultMCPSessionTimeout,
		Stateless:          defaultMCPStateless,
		EnableWrites:       defaultMCPEnableWrites,
		EnableEventStore:   defaultMCPEnableEventStore,
		EventStoreMaxBytes: defaultMCPEventStoreMaxBytes,
	}
}

func runRuntimeAsync(t *testing.T, deps runtimeDependencies) <-chan error {
	t.Helper()

	done := make(chan error, 1)
	go func() {
		done <- runRuntime(context.Background(), deps)
	}()
	return done
}

func waitForRuntimeExit(t *testing.T, done <-chan error) {
	t.Helper()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected runtime to exit without error, got %v", err)
		}
	case <-time.After(runtimeTestWait):
		t.Fatalf("timed out waiting for runtime exit")
	}
}

func assertNoRuntimeExit(t *testing.T, done <-chan error, wait time.Duration) {
	t.Helper()

	select {
	case err := <-done:
		t.Fatalf("expected runtime to still be running, but exited with %v", err)
	case <-time.After(wait):
	}
}

func waitForEvent(t *testing.T, event <-chan struct{}, name string) {
	t.Helper()

	select {
	case <-event:
	case <-time.After(runtimeTestWait):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func assertNoEvent(t *testing.T, event <-chan struct{}, name string, wait time.Duration) {
	t.Helper()

	select {
	case <-event:
		t.Fatalf("expected no %s event", name)
	case <-time.After(wait):
	}
}

type fakeRuntimeWebServer struct {
	mu             sync.Mutex
	startCalls     int
	shutdownCalls  int
	addr           string
	startErr       chan error
	startCalled    chan struct{}
	shutdownCalled chan struct{}
}

func newFakeRuntimeWebServer() *fakeRuntimeWebServer {
	return &fakeRuntimeWebServer{
		startErr:       make(chan error, 1),
		startCalled:    make(chan struct{}, 1),
		shutdownCalled: make(chan struct{}, 1),
	}
}

func (f *fakeRuntimeWebServer) Start(addr string) error {
	f.mu.Lock()
	f.startCalls++
	f.addr = addr
	f.mu.Unlock()

	notify(f.startCalled)
	return <-f.startErr
}

func (f *fakeRuntimeWebServer) Shutdown() error {
	f.mu.Lock()
	f.shutdownCalls++
	f.mu.Unlock()

	notify(f.shutdownCalled)

	select {
	case f.startErr <- http.ErrServerClosed:
	default:
	}

	return nil
}

func (f *fakeRuntimeWebServer) startCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.startCalls
}

func (f *fakeRuntimeWebServer) shutdownCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.shutdownCalls
}

func (f *fakeRuntimeWebServer) startAddress() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.addr
}

type fakeRuntimeMCPServer struct {
	mu          sync.Mutex
	runCalls    int
	runErr      chan error
	runStarted  chan struct{}
	ctxCanceled chan struct{}
}

func newFakeRuntimeMCPServer() *fakeRuntimeMCPServer {
	return &fakeRuntimeMCPServer{
		runErr:      make(chan error, 1),
		runStarted:  make(chan struct{}, 1),
		ctxCanceled: make(chan struct{}, 1),
	}
}

func (f *fakeRuntimeMCPServer) Run(ctx context.Context) error {
	f.mu.Lock()
	f.runCalls++
	f.mu.Unlock()

	notify(f.runStarted)

	select {
	case err := <-f.runErr:
		return err
	case <-ctx.Done():
		notify(f.ctxCanceled)
		return nil
	}
}

func (f *fakeRuntimeMCPServer) runCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.runCalls
}

type fakeRuntimeGitSyncer struct {
	mu        sync.Mutex
	stopCalls int
	stopped   chan struct{}
}

func newFakeRuntimeGitSyncer() *fakeRuntimeGitSyncer {
	return &fakeRuntimeGitSyncer{
		stopped: make(chan struct{}, 1),
	}
}

func (f *fakeRuntimeGitSyncer) Stop() {
	f.mu.Lock()
	f.stopCalls++
	f.mu.Unlock()
	notify(f.stopped)
}

func (f *fakeRuntimeGitSyncer) stopCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopCalls
}

func notify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
