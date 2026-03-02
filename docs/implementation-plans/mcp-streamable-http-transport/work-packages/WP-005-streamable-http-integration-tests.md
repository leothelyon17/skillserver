## WP-005: Streamable HTTP Integration Tests

### Metadata

```yaml
WP_ID: WP-005
Title: Streamable HTTP Integration Tests
Domain: Quality Engineering (HTTP MCP)
Priority: High
Estimated_Effort: 6 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-02
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Core acceptance depends on verifying end-to-end Streamable HTTP MCP behavior, not only unit-level option mapping.

**Scope:**
- Add end-to-end tests around `/mcp` lifecycle.
- Validate session init, tool listing/call, and close behavior.
- Validate behavior with event store enabled and disabled.

Excluded:
- Stdio compatibility regression tests (WP-006).

**Success Criteria:**
- [ ] `/mcp` lifecycle flows are exercised in tests.
- [ ] `list_skills` tool call succeeds over Streamable HTTP.

---

### Technical Requirements

**Input Contracts:**
- Runtime orchestration complete (WP-004).

**Output Contracts:**
- Integration tests in `pkg/web/mcp_integration_test.go`.

**Integration Points:**
- Uses MCP handler created from package MCP server.

---

### Deliverables

**Test Deliverables:**
- [ ] Add E2E lifecycle test: initialize -> list/call -> close.
- [ ] Add route/method matrix tests (`GET`, `POST`, `DELETE`).
- [ ] Add tests covering event-store on/off behavior.

**Documentation Deliverables:**
- [ ] Include comments documenting test assumptions/protocol headers.

---

### Acceptance Criteria

- [ ] MCP initialization over `/mcp` succeeds.
- [ ] Tool call `list_skills` succeeds via HTTP transport.
- [ ] Session close path succeeds.
- [ ] Endpoint method behavior aligns with expected protocol flow.

---

### Testing Strategy

**Integration Tests:**
- `TestMCPHTTP_InitializeSession`
- `TestMCPHTTP_ListToolsAndCallListSkills`
- `TestMCPHTTP_CloseSession`
- `TestMCPHTTP_MethodMatrix`
- `TestMCPHTTP_WithAndWithoutEventStore`

---

### Dependencies

**Blocked By:**
- WP-004

**Blocks:**
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-006, WP-007
- Cannot run in parallel with: none

---

### Risks

**Risk 1: Flaky integration tests due to async stream timing**
- Probability: Medium
- Impact: Medium
- Mitigation: deterministic timeouts, helper utilities, and robust assertions.

**Risk 2: Over-coupling tests to internal SDK implementation details**
- Probability: Low
- Impact: Medium
- Mitigation: validate externally observable behavior only.

---

### Notes

Prefer black-box assertions against HTTP responses and MCP JSON-RPC envelopes.
