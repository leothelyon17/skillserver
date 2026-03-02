## WP-006: Stdio Regression + Mixed-Mode Resilience Tests

### Metadata

```yaml
WP_ID: WP-006
Title: Stdio Regression + Mixed-Mode Resilience Tests
Domain: Quality Engineering (Stdio/Compatibility)
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
Backward compatibility is a must-have. The new transport implementation must not break stdio behavior and must preserve both-mode resiliency guarantees.

**Scope:**
- Add stdio regression tests for MCP tool behavior.
- Add mixed-mode lifecycle tests validating stdio exit does not kill HTTP.

Excluded:
- HTTP protocol correctness already covered by WP-005.

**Success Criteria:**
- [ ] Existing stdio tool path remains functional.
- [ ] Both-mode resilience is verified by tests.

---

### Technical Requirements

**Input Contracts:**
- Runtime orchestration from WP-004.

**Output Contracts:**
- `pkg/mcp/server_stdio_regression_test.go`
- `cmd/skillserver/both_mode_lifecycle_test.go`

**Integration Points:**
- Tests run alongside WP-005 suite to validate full compatibility matrix.

---

### Deliverables

**Test Deliverables:**
- [ ] Add stdio regression test coverage for MCP tool registration and invocation path.
- [ ] Add both-mode lifecycle tests for stdio EOF/disconnect handling.
- [ ] Add graceful shutdown assertion coverage under mixed mode.

---

### Acceptance Criteria

- [ ] Stdio behavior remains functional under new runtime architecture.
- [ ] `both` mode continues serving HTTP when stdio exits.
- [ ] No regressions in MCP tool registration/invocation behavior.

---

### Testing Strategy

**Tests:**
- `TestMCPServer_StdioRegression`
- `TestRuntime_BothModeStdioExitKeepsHTTP`
- `TestRuntime_BothModeSignalShutdown`

---

### Dependencies

**Blocked By:**
- WP-004

**Blocks:**
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-005, WP-007
- Cannot run in parallel with: none

---

### Risks

**Risk 1: Hard-to-simulate stdio disconnect conditions**
- Probability: Medium
- Impact: Medium
- Mitigation: inject transport lifecycle abstractions for deterministic tests.

---

### Notes

Use test doubles for lifecycle coordination where direct stdio simulation is unstable.
