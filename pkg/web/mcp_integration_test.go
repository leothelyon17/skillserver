package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mudler/skillserver/pkg/domain"
	skillmcp "github.com/mudler/skillserver/pkg/mcp"
)

const (
	testMCPPath            = "/mcp"
	testMCPProtocolVersion = "2025-06-18"
	testMCPSessionHeader   = "Mcp-Session-Id"
	testMCPVersionHeader   = "MCP-Protocol-Version"
)

func TestMCPHTTP_InitializeSession(t *testing.T) {
	t.Parallel()

	httpServer := newMCPIntegrationServer(t, true)
	defer httpServer.Close()

	sessionID := initializeMCPSession(t, httpServer.Client(), httpServer.URL+testMCPPath)
	if sessionID == "" {
		t.Fatalf("expected non-empty session ID")
	}
}

func TestMCPHTTP_ListToolsAndCallListSkills(t *testing.T) {
	t.Parallel()

	httpServer := newMCPIntegrationServer(t, true)
	defer httpServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "integration-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcpsdk.StreamableClientTransport{Endpoint: httpServer.URL + testMCPPath}, nil)
	if err != nil {
		t.Fatalf("failed to connect MCP streamable client: %v", err)
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			t.Fatalf("failed to close MCP session: %v", closeErr)
		}
	}()

	listCtx, listCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer listCancel()

	toolsResult, err := session.ListTools(listCtx, nil)
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}
	if !toolExists(toolsResult.Tools, "list_skills") {
		t.Fatalf("expected list_skills to be present in tools list")
	}

	callResult, err := session.CallTool(listCtx, &mcpsdk.CallToolParams{
		Name:      "list_skills",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("list_skills call failed: %v", err)
	}
	if callResult.IsError {
		t.Fatalf("expected list_skills call to succeed, got isError=true")
	}

	structured, ok := callResult.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structured content map, got %T", callResult.StructuredContent)
	}

	skillsRaw, ok := structured["skills"]
	if !ok {
		t.Fatalf("expected structured content to include skills field")
	}

	skills, ok := skillsRaw.([]any)
	if !ok {
		t.Fatalf("expected skills to be an array, got %T", skillsRaw)
	}
	if len(skills) == 0 {
		t.Fatalf("expected at least one skill from test fixture")
	}
	if !skillsContainID(skills, "demo-skill") {
		t.Fatalf("expected demo-skill to be returned from list_skills")
	}
}

func TestMCPHTTP_CloseSession(t *testing.T) {
	t.Parallel()

	httpServer := newMCPIntegrationServer(t, true)
	defer httpServer.Close()

	client := httpServer.Client()
	endpoint := httpServer.URL + testMCPPath

	sessionID := initializeMCPSession(t, client, endpoint)

	closeResponse := doDelete(t, client, endpoint, sessionID)
	if closeResponse.statusCode != http.StatusNoContent {
		t.Fatalf("expected close status %d, got %d body=%q", http.StatusNoContent, closeResponse.statusCode, closeResponse.body)
	}

	postClose := doMCPPost(t, client, endpoint, initializeRequestBody(), sessionID)
	if postClose.statusCode != http.StatusNotFound {
		t.Fatalf("expected session-not-found status %d after close, got %d body=%q", http.StatusNotFound, postClose.statusCode, postClose.body)
	}
	if !strings.Contains(strings.ToLower(string(postClose.body)), "session not found") {
		t.Fatalf("expected session-not-found error body, got %q", postClose.body)
	}
}

func TestMCPHTTP_MethodMatrix(t *testing.T) {
	t.Parallel()

	httpServer := newMCPIntegrationServer(t, true)
	defer httpServer.Close()

	client := httpServer.Client()
	endpoint := httpServer.URL + testMCPPath

	getWithoutSessionRequest, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("failed to construct GET request: %v", err)
	}
	getWithoutSessionRequest.Header.Set("Accept", "text/event-stream")
	getWithoutSessionResponse := executeHTTP(t, client, getWithoutSessionRequest)
	if getWithoutSessionResponse.statusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected GET without session to return %d, got %d body=%q", http.StatusMethodNotAllowed, getWithoutSessionResponse.statusCode, getWithoutSessionResponse.body)
	}

	postInitialize := doMCPPost(t, client, endpoint, initializeRequestBody(), "")
	if postInitialize.statusCode != http.StatusOK {
		t.Fatalf("expected initialize POST to return %d, got %d body=%q", http.StatusOK, postInitialize.statusCode, postInitialize.body)
	}
	sessionID := postInitialize.header.Get(testMCPSessionHeader)
	if sessionID == "" {
		t.Fatalf("expected initialize response to include %s header", testMCPSessionHeader)
	}

	deleteWithoutSession := doDelete(t, client, endpoint, "")
	if deleteWithoutSession.statusCode != http.StatusBadRequest {
		t.Fatalf("expected DELETE without session to return %d, got %d body=%q", http.StatusBadRequest, deleteWithoutSession.statusCode, deleteWithoutSession.body)
	}

	deleteWithSession := doDelete(t, client, endpoint, sessionID)
	if deleteWithSession.statusCode != http.StatusNoContent {
		t.Fatalf("expected DELETE with active session to return %d, got %d body=%q", http.StatusNoContent, deleteWithSession.statusCode, deleteWithSession.body)
	}
}

func TestMCPHTTP_WithAndWithoutEventStore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		enableEventStore    bool
		expectedBodySnippet string
	}{
		{name: "with_event_store", enableEventStore: true, expectedBodySnippet: "failed to replay events"},
		{name: "without_event_store", enableEventStore: false, expectedBodySnippet: "stream replay unsupported"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			httpServer := newMCPIntegrationServer(t, test.enableEventStore)
			defer httpServer.Close()

			client := httpServer.Client()
			endpoint := httpServer.URL + testMCPPath

			sessionID := initializeMCPSession(t, client, endpoint)

			request, err := http.NewRequest(http.MethodGet, endpoint, nil)
			if err != nil {
				t.Fatalf("failed to construct replay GET request: %v", err)
			}
			request.Header.Set("Accept", "text/event-stream")
			request.Header.Set(testMCPSessionHeader, sessionID)
			request.Header.Set(testMCPVersionHeader, testMCPProtocolVersion)
			request.Header.Set("Last-Event-ID", "unknown_0")

			response := executeHTTP(t, client, request)
			if response.statusCode != http.StatusBadRequest {
				t.Fatalf("expected replay GET to return %d, got %d body=%q", http.StatusBadRequest, response.statusCode, response.body)
			}
			if !strings.Contains(strings.ToLower(string(response.body)), strings.ToLower(test.expectedBodySnippet)) {
				t.Fatalf("expected body to contain %q, got %q", test.expectedBodySnippet, response.body)
			}
		})
	}
}

type httpResponse struct {
	statusCode int
	header     http.Header
	body       []byte
}

func executeHTTP(t *testing.T, client *http.Client, request *http.Request) httpResponse {
	t.Helper()

	resp, err := client.Do(request)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", request.Method, request.URL.String(), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed reading response body: %v", err)
	}

	return httpResponse{
		statusCode: resp.StatusCode,
		header:     resp.Header.Clone(),
		body:       body,
	}
}

func doMCPPost(t *testing.T, client *http.Client, endpoint, payload, sessionID string) httpResponse {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to construct POST request: %v", err)
	}
	setMCPRequestHeaders(request, sessionID)

	return executeHTTP(t, client, request)
}

func doDelete(t *testing.T, client *http.Client, endpoint, sessionID string) httpResponse {
	t.Helper()

	request, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		t.Fatalf("failed to construct DELETE request: %v", err)
	}
	if sessionID != "" {
		request.Header.Set(testMCPSessionHeader, sessionID)
	}

	return executeHTTP(t, client, request)
}

func initializeMCPSession(t *testing.T, client *http.Client, endpoint string) string {
	t.Helper()

	response := doMCPPost(t, client, endpoint, initializeRequestBody(), "")
	if response.statusCode != http.StatusOK {
		t.Fatalf("expected initialize POST status %d, got %d body=%q", http.StatusOK, response.statusCode, response.body)
	}

	sessionID := response.header.Get(testMCPSessionHeader)
	if sessionID == "" {
		t.Fatalf("expected initialize response to include %s header", testMCPSessionHeader)
	}

	message := decodeJSONRPCMessage(t, response.header.Get("Content-Type"), response.body)
	if _, ok := message["result"]; !ok {
		t.Fatalf("expected initialize response to include result payload, got %v", message)
	}

	return sessionID
}

func decodeJSONRPCMessage(t *testing.T, contentType string, body []byte) map[string]any {
	t.Helper()

	mediaType := strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	switch mediaType {
	case "application/json":
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to decode JSON response: %v body=%q", err, body)
		}
		return payload
	case "text/event-stream":
		for _, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "data:") {
				continue
			}

			data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
			if data == "" {
				continue
			}

			var payload map[string]any
			if err := json.Unmarshal([]byte(data), &payload); err == nil {
				return payload
			}
		}
		t.Fatalf("failed to decode JSON-RPC message from SSE body %q", body)
	default:
		t.Fatalf("unexpected content-type %q body=%q", contentType, body)
	}

	return nil
}

func setMCPRequestHeaders(request *http.Request, sessionID string) {
	// Streamable HTTP POST requests require both application/json and text/event-stream
	// in Accept, plus content-type and protocol/session headers for stateful routing.
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set(testMCPVersionHeader, testMCPProtocolVersion)
	if sessionID != "" {
		request.Header.Set(testMCPSessionHeader, sessionID)
	}
}

func initializeRequestBody() string {
	return `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"mcp-http-integration-test","version":"1.0.0"}}}`
}

func toolExists(tools []*mcpsdk.Tool, toolName string) bool {
	for _, tool := range tools {
		if tool != nil && tool.Name == toolName {
			return true
		}
	}
	return false
}

func skillsContainID(skills []any, skillID string) bool {
	for _, entry := range skills {
		skill, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := skill["id"].(string); ok && id == skillID {
			return true
		}
	}
	return false
}

func newMCPIntegrationServer(t *testing.T, enableEventStore bool) *httptest.Server {
	t.Helper()

	skillsDir := t.TempDir()
	createTestSkill(t, skillsDir, "demo-skill")

	skillManager, err := domain.NewFileSystemManager(skillsDir, nil)
	if err != nil {
		t.Fatalf("failed to create file system manager: %v", err)
	}

	mcpServer := skillmcp.NewServer(skillManager)
	mcpHandler := mcpServer.NewStreamableHTTPHandler(skillmcp.StreamableHTTPConfig{
		SessionTimeout:     5 * time.Minute,
		Stateless:          false,
		EnableEventStore:   enableEventStore,
		EventStoreMaxBytes: 1024 * 1024,
	})

	webServer := NewServer(skillManager, skillManager, nil, nil, nil, false, mcpHandler, testMCPPath)
	return httptest.NewServer(webServer.echo)
}

func createTestSkill(t *testing.T, skillsDir, skillName string) {
	t.Helper()

	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill directory: %v", err)
	}

	content := `---
name: demo-skill
description: Integration test fixture skill
---
# Demo Skill

Fixture skill used by MCP integration tests.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create fixture SKILL.md: %v", err)
	}
}
