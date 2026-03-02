package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mudler/skillserver/pkg/domain"
)

func newTestWebServer(t *testing.T, mcpHandler http.Handler, mcpPath string) *Server {
	t.Helper()

	skillManager, err := domain.NewFileSystemManager(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}

	return NewServer(skillManager, skillManager, nil, nil, nil, false, mcpHandler, mcpPath)
}

func TestWebServer_MCPRoutePrecedence(t *testing.T) {
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Route-Handler", "mcp")
		w.WriteHeader(http.StatusNoContent)
	})

	server := newTestWebServer(t, mcpHandler, "/mcp")

	expectedMethods := map[string]bool{
		http.MethodGet:     false,
		http.MethodPost:    false,
		http.MethodDelete:  false,
		http.MethodOptions: false,
	}
	for _, route := range server.echo.Router().Routes() {
		if route.Path == "/mcp" {
			if _, ok := expectedMethods[route.Method]; ok {
				expectedMethods[route.Method] = true
			}
		}
	}

	for method, found := range expectedMethods {
		if !found {
			t.Fatalf("expected %s route for /mcp to be registered", method)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected MCP handler status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if got := rec.Header().Get("X-Route-Handler"); got != "mcp" {
		t.Fatalf("expected MCP handler header %q, got %q", "mcp", got)
	}
}

func TestWebServer_UIRootStillServed(t *testing.T) {
	server := newTestWebServer(t, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected UI root status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "<!doctype html>") {
		t.Fatalf("expected UI root to serve HTML document")
	}
}

func TestWebServer_APIRoutesUnaffected(t *testing.T) {
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Route-Handler", "mcp")
		w.WriteHeader(http.StatusNoContent)
	})
	server := newTestWebServer(t, mcpHandler, "/mcp")

	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected API route status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected JSON response from API route, got content-type %q", rec.Header().Get("Content-Type"))
	}
}

func TestWebServer_NoMCPRouteWhenHandlerNil(t *testing.T) {
	server := newTestWebServer(t, nil, "")

	for _, route := range server.echo.Router().Routes() {
		if route.Path == "/mcp" {
			t.Fatalf("did not expect /mcp route registration when MCP handler is nil")
		}
	}
}
