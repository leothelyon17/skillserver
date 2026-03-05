package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type runtimeWebServer interface {
	Start(addr string) error
	Shutdown() error
}

type runtimeMCPServer interface {
	Run(ctx context.Context) error
}

type runtimeGitSyncer interface {
	Stop()
}

type runtimeDependencies struct {
	logger        *log.Logger
	enableLogging bool
	mcpConfig     MCPRuntimeConfig
	port          string
	webServer     runtimeWebServer
	mcpServer     runtimeMCPServer
	gitSyncer     runtimeGitSyncer
	signalChan    <-chan os.Signal
}

func runRuntime(ctx context.Context, deps runtimeDependencies) error {
	switch deps.mcpConfig.Transport {
	case MCPTransportStdio, MCPTransportHTTP, MCPTransportBoth:
	default:
		return fmt.Errorf("unsupported MCP transport mode %q", deps.mcpConfig.Transport)
	}

	if deps.webServer == nil {
		return errors.New("web server is required")
	}
	if deps.port == "" {
		return errors.New("runtime port is required")
	}
	if requiresMCPStdio(deps.mcpConfig.Transport) && deps.mcpServer == nil {
		return errors.New("mcp server is required when stdio transport is enabled")
	}

	logger := deps.logger
	if logger == nil {
		logger = log.New(io.Discard, "", log.LstdFlags)
	}

	if deps.enableLogging {
		logger.Printf(
			"Resolved MCP runtime options: transport=%s http_path=%s session_timeout=%s stateless=%t writes_enabled=%t event_store_enabled=%t event_store_max_bytes=%d",
			deps.mcpConfig.Transport,
			deps.mcpConfig.HTTPPath,
			deps.mcpConfig.SessionTimeout,
			deps.mcpConfig.Stateless,
			deps.mcpConfig.EnableWrites,
			deps.mcpConfig.EnableEventStore,
			deps.mcpConfig.EventStoreMaxBytes,
		)
		logger.Printf("Starting web server on :%s", deps.port)
	}

	runtimeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	webErrCh := make(chan error, 1)
	go func() {
		webErrCh <- deps.webServer.Start(":" + deps.port)
	}()

	var stdioErrCh chan error
	if requiresMCPStdio(deps.mcpConfig.Transport) {
		stdioErrCh = make(chan error, 1)
		go func() {
			stdioErrCh <- deps.mcpServer.Run(runtimeCtx)
		}()
	}

	for {
		select {
		case <-ctx.Done():
			if deps.enableLogging {
				logger.Println("Context canceled, shutting down runtime")
			}
			return shutdownRuntime(logger, deps.enableLogging, deps.webServer, deps.gitSyncer, cancel)
		case sig := <-deps.signalChan:
			if deps.enableLogging {
				logger.Printf("Received signal %s, shutting down runtime", sig)
			}
			return shutdownRuntime(logger, deps.enableLogging, deps.webServer, deps.gitSyncer, cancel)
		case err := <-webErrCh:
			if err == nil || errors.Is(err, http.ErrServerClosed) {
				return shutdownRuntime(logger, deps.enableLogging, deps.webServer, deps.gitSyncer, cancel)
			}

			shutdownErr := shutdownRuntime(logger, deps.enableLogging, deps.webServer, deps.gitSyncer, cancel)
			if shutdownErr != nil {
				return errors.Join(fmt.Errorf("web server error: %w", err), shutdownErr)
			}
			return fmt.Errorf("web server error: %w", err)
		case err := <-stdioErrCh:
			if deps.mcpConfig.Transport == MCPTransportBoth {
				if deps.enableLogging {
					if err != nil {
						logger.Printf("MCP stdio transport exited in both mode: %v. HTTP transport remains active", err)
					} else {
						logger.Println("MCP stdio transport exited in both mode. HTTP transport remains active")
					}
				}
				stdioErrCh = nil
				continue
			}

			if deps.enableLogging && err != nil {
				logger.Printf("MCP stdio transport exited: %v", err)
			}
			return shutdownRuntime(logger, deps.enableLogging, deps.webServer, deps.gitSyncer, cancel)
		}
	}
}

func shutdownRuntime(
	logger *log.Logger,
	enableLogging bool,
	webServer runtimeWebServer,
	gitSyncer runtimeGitSyncer,
	cancel context.CancelFunc,
) error {
	cancel()

	if gitSyncer != nil {
		gitSyncer.Stop()
	}

	if err := webServer.Shutdown(); err != nil {
		return fmt.Errorf("failed to shut down web server: %w", err)
	}

	if enableLogging {
		logger.Println("Shutdown complete")
	}

	return nil
}

func requiresMCPHTTP(mode MCPTransportMode) bool {
	return mode == MCPTransportHTTP || mode == MCPTransportBoth
}

func requiresMCPStdio(mode MCPTransportMode) bool {
	return mode == MCPTransportStdio || mode == MCPTransportBoth
}
