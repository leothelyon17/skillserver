## WP-003: Web Route Integration for `/mcp`

### Metadata

```yaml
WP_ID: WP-003
Title: Web Route Integration for /mcp
Domain: API/Web Routing
Priority: High
Estimated_Effort: 3 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-02
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Echo server currently has API routes plus UI catch-all. `/mcp` must be registered before UI wildcard and support MCP HTTP methods.

**Scope:**
- Extend web server constructor to accept optional MCP handler and path.
- Register `GET`, `POST`, `DELETE`, and `OPTIONS` for MCP path.
- Keep `/api/*` and UI behavior unchanged.

Excluded:
- Runtime selection of whether MCP routes are enabled (handled by WP-004).

**Success Criteria:**
- [ ] `/mcp` resolves correctly.
- [ ] UI catch-all does not intercept `/mcp`.

---

### Technical Requirements

**Input Contracts:**
- MCP handler constructor from WP-002.

**Output Contracts:**
- Updated `pkg/web/server.go` constructor signature and route setup.
- Route precedence tests in `pkg/web/server_mcp_routes_test.go`.

**Integration Points:**
- WP-004 runtime passes MCP handler/path when HTTP transport enabled.

---

### Deliverables

**Code Deliverables:**
- [ ] Modify `pkg/web/server.go` to support optional MCP route injection.
- [ ] Add method-specific route registration for MCP.
- [ ] Add tests in `pkg/web/server_mcp_routes_test.go`.

**Test Deliverables:**
- [ ] Validate `/mcp` route precedence over UI wildcard.
- [ ] Validate UI routes still serve static content.
- [ ] Validate `/api/*` routes unaffected.

---

### Acceptance Criteria

- [ ] `GET /mcp` handled by MCP handler when configured.
- [ ] `GET /` still serves UI.
- [ ] `GET /api/skills` still hits API handler.
- [ ] When MCP handler is nil, `/mcp` route is not registered.

---

### Testing Strategy

**Unit/Integration Tests:**
- `TestWebServer_MCPRoutePrecedence`
- `TestWebServer_UIRootStillServed`
- `TestWebServer_APIRoutesUnaffected`
- `TestWebServer_NoMCPRouteWhenHandlerNil`

---

### Dependencies

**Blocked By:**
- WP-002

**Blocks:**
- WP-004
- WP-005

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-004, WP-005

---

### Risks

**Risk 1: Route ordering regression**
- Probability: Medium
- Impact: High
- Mitigation: explicit precedence tests and route registration order comments.

---

### Notes

Use method-specific registration only; do not use broad wildcard for `/mcp`.
