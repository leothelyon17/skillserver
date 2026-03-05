## WP-009: MCP Taxonomy Write Tools and Runtime Gating

### Metadata

```yaml
WP_ID: WP-009
Title: MCP Taxonomy Write Tools and Runtime Gating
Domain: MCP Layer
Priority: High
Estimated_Effort: 4 hours
Status: COMPLETE
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: 2026-03-05
Completed_Date: 2026-03-05
```

---

### Description

**Context:**
ADR-005 requires MCP write parity but only when explicitly enabled. Default behavior must remain read-oriented.

**Scope:**
- Add MCP write tools for taxonomy object mutation and item taxonomy patch.
- Add runtime write gate (`SKILLSERVER_MCP_ENABLE_WRITES`, optional flag mirror) and wire it into MCP server construction.
- Ensure write tools are not registered when gate is disabled.

Excluded:
- REST write path implementation (WP-006, WP-007).

**Success Criteria:**
- [x] Write tools are conditionally registered based on runtime config.
- [x] Default runtime keeps write tools disabled.
- [x] Write tool behavior matches service/API validation outcomes.

---

### Technical Requirements

**Input Contracts:**
- Read tool/filter scaffolding from WP-008.
- Taxonomy services from WP-003 and WP-004.
- Runtime config parsing in `cmd/skillserver/config.go` and wiring in `cmd/skillserver/main.go`.

**Output Contracts:**
- Additive MCP write tool schemas/handlers.
- Runtime config fields and effective-config logging for write gate state.

**Integration Points:**
- WP-011 tests write tool registration matrix under enabled/disabled modes.
- WP-012 docs include operational guidance for enabling writes.

---

### Deliverables

**Code Deliverables:**
- [x] Add write gate config parsing and defaults in `cmd/skillserver/config.go` (+ tests).
- [x] Wire write gate into MCP server initialization in `cmd/skillserver/main.go` and `pkg/mcp/server.go`.
- [x] Add write tool handlers in `pkg/mcp/tools.go`.

**Test Deliverables:**
- [x] Config tests for default/explicit write-gate behavior.
- [x] MCP registration tests proving write tools are absent/present by gate state.
- [x] Handler tests for validation and error propagation.

---

### Acceptance Criteria

**Functional:**
- [x] MCP write tools are unavailable unless write gate is explicitly enabled.
- [x] Enabled write tools perform taxonomy mutation successfully through domain services.

**Testing:**
- [x] Runtime config and MCP registration tests pass for both gate states.
- [x] No regressions in existing MCP stdio/http behavior.

---

### Dependencies

**Blocked By:**
- WP-008

**Blocks:**
- WP-011
- WP-012

**Parallel Execution:**
- Can run in parallel with: WP-010 after WP-008.
- Cannot run in parallel with: WP-008.

---

### Risks

**Risk 1: Misconfigured gate unexpectedly exposes mutation tools**
- Probability: Low
- Impact: High
- Mitigation: Default false, explicit startup logging, and tests for default behavior.

**Risk 2: Write tool payload contracts diverge from REST payloads**
- Probability: Medium
- Impact: Medium
- Mitigation: Reuse shared DTO/validation shapes where possible.
