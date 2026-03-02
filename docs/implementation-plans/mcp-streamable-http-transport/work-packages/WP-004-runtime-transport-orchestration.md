## WP-004: Transport Mode Runtime Orchestration

### Metadata

```yaml
WP_ID: WP-004
Title: Transport Mode Runtime Orchestration
Domain: Runtime Orchestration
Priority: High
Estimated_Effort: 5 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-02
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Main runtime currently starts web server in goroutine and blocks on stdio MCP run. New transport modes require explicit runtime orchestration.

**Scope:**
- Refactor startup flow to support `stdio`, `http`, and `both`.
- Ensure `both` mode keeps HTTP alive if stdio exits/EOF.
- Keep graceful shutdown behavior for signal-driven termination.

Excluded:
- Detailed MCP HTTP lifecycle protocol tests (WP-005).

**Success Criteria:**
- [ ] All modes start with expected behavior.
- [ ] `both` mode stdio exit does not terminate HTTP service.
- [ ] Shutdown sequence remains graceful and deterministic.

---

### Technical Requirements

**Input Contracts:**
- Config from WP-001.
- MCP HTTP handler support from WP-002.
- Web route support from WP-003.

**Output Contracts:**
- Refactored runtime startup in `cmd/skillserver/main.go`.
- Runtime tests in `cmd/skillserver/runtime_test.go`.

**Integration Points:**
- WP-005 and WP-006 depend on orchestration behavior.
- WP-007 docs depend on final startup semantics.

---

### Deliverables

**Code Deliverables:**
- [ ] Refactor transport startup logic in `cmd/skillserver/main.go`.
- [ ] Add runtime helper(s) to isolate start/stop orchestration for testing.
- [ ] Add startup logs that print resolved mode/path/session options.
- [ ] Add lifecycle tests in `cmd/skillserver/runtime_test.go`.

**Test Deliverables:**
- [ ] Test boot behavior for each transport mode.
- [ ] Test `both` mode behavior on stdio exit.
- [ ] Test graceful shutdown path.

---

### Acceptance Criteria

- [ ] `stdio` mode: stdio MCP runs, web server still starts as expected.
- [ ] `http` mode: web server and `/mcp` run without stdio run loop.
- [ ] `both` mode: stdio and HTTP run concurrently.
- [ ] In `both`, stdio disconnect/EOF does not kill process.
- [ ] Signal shutdown stops web server and git syncer cleanly.

---

### Testing Strategy

**Unit/Runtime Tests:**
- `TestRuntime_StartModeStdio`
- `TestRuntime_StartModeHTTP`
- `TestRuntime_StartModeBoth`
- `TestRuntime_BothModeStdioExitDoesNotStopHTTP`
- `TestRuntime_GracefulShutdown`

---

### Dependencies

**Blocked By:**
- WP-001
- WP-002
- WP-003

**Blocks:**
- WP-005
- WP-006
- WP-007

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-005, WP-006, WP-007

---

### Risks

**Risk 1: Deadlock or goroutine leak in multi-transport startup**
- Probability: Medium
- Impact: High
- Mitigation: isolate lifecycle control paths and test explicit shutdown sequences.

**Risk 2: Behavior drift in legacy stdio path**
- Probability: Medium
- Impact: High
- Mitigation: regression tests and startup log verification.

---

### Notes

Maintain existing logger interference safeguards for stdio mode.
