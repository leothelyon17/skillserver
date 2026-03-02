## WP-002: MCP Streamable HTTP Handler Support

### Metadata

```yaml
WP_ID: WP-002
Title: MCP Streamable HTTP Handler Support
Domain: MCP Transport
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-02
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
The MCP package currently runs only stdio transport. This package adds Streamable HTTP handler construction while preserving stdio behavior.

**Scope:**
- Add HTTP handler constructor on MCP server wrapper.
- Map parsed config into Streamable HTTP options.
- Support optional in-memory event store and stateless mode.

Excluded:
- Web route registration and app runtime orchestration.

**Success Criteria:**
- [ ] Handler builds correctly for all config combinations.
- [ ] Existing stdio `Run` behavior unchanged.

---

### Technical Requirements

**Input Contracts:**
- Config contract from WP-001.
- Existing server wrapper in `pkg/mcp/server.go`.

**Output Contracts:**
- New helper in `pkg/mcp/http_transport.go`.
- `Server` exposes constructor/accessor for HTTP handler.
- Tests in `pkg/mcp/http_transport_test.go`.

**Integration Points:**
- WP-003 consumes handler for route binding.
- WP-004 consumes constructor in runtime startup.

---

### Deliverables

**Code Deliverables:**
- [ ] Add `pkg/mcp/http_transport.go`.
- [ ] Extend `pkg/mcp/server.go` with HTTP handler support.
- [ ] Add `pkg/mcp/http_transport_test.go` for option mapping and event store toggles.

**Test Deliverables:**
- [ ] Unit tests for session timeout mapping.
- [ ] Unit tests for stateless mapping.
- [ ] Unit tests for event store enabled/disabled behavior.

---

### Acceptance Criteria

**Functional:**
- [ ] Streamable HTTP handler can be created from MCP server instance.
- [ ] Event store can be disabled entirely.
- [ ] Event store max-bytes is passed when enabled.
- [ ] Stdio path continues to work through existing methods.

**Testing:**
- [ ] Handler option tests cover all config permutations.

---

### Testing Strategy

**Unit Tests:**
- `TestBuildStreamableHTTPOptions_WithEventStore`
- `TestBuildStreamableHTTPOptions_WithoutEventStore`
- `TestBuildStreamableHTTPOptions_Stateless`
- `TestServer_RunStillUsesStdioTransport`

---

### Dependencies

**Blocked By:**
- WP-001

**Blocks:**
- WP-003
- WP-004
- WP-005

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-003, WP-004, WP-005

---

### Risks

**Risk 1: SDK option mismatch**
- Probability: Medium
- Impact: High
- Mitigation: Validate against go-sdk `v1.2.0` API and add compile-time tests.

**Risk 2: Silent behavior drift in stdio run path**
- Probability: Low
- Impact: High
- Mitigation: regression test on stdio run wrapper.

---

### Notes

Do not introduce request-level auth logic in this package.
