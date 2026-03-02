package main

import (
	"io"
	"log"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestRuntime_BothModeStdioExitKeepsHTTP(t *testing.T) {
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

func TestRuntime_BothModeSignalShutdown(t *testing.T) {
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
