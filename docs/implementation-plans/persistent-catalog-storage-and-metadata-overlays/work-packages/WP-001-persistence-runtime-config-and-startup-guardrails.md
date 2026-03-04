## WP-001: Persistence Runtime Config and Startup Guardrails

### Metadata

```yaml
WP_ID: WP-001
Title: Persistence Runtime Config and Startup Guardrails
Domain: Infrastructure
Priority: High
Estimated_Effort: 3 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Persistence is opt-in. The runtime must fail safely and deterministically when enabled without a usable durable mount path.

**Scope:**
- Add persistence runtime configuration parsing.
- Validate `SKILLSERVER_PERSISTENCE_DIR` and optional DB path when enabled.
- Enforce startup fail-fast with actionable errors.

Excluded:
- SQLite schema creation and migrations (WP-002).
- Repository/query implementation (WP-003 onward).

**Success Criteria:**
- [ ] Runtime config resolves deterministic values with clear precedence.
- [ ] Startup fails fast when persistence is enabled and path is invalid/unwritable.
- [ ] Startup behavior is unchanged when persistence is disabled.

---

### Technical Requirements

**Input Contracts:**
- Existing startup/runtime flow in `cmd/skillserver/main.go`.

**Output Contracts:**
- New persistence runtime config parser and validator (for example `cmd/skillserver/persistence_runtime.go`).
- Structured startup diagnostics for persistence mode.

**Integration Points:**
- WP-002 consumes validated DB path.
- WP-007 consumes persistence-enabled runtime switch for wiring.

---

### Deliverables

**Code Deliverables:**
- [ ] Add runtime config type and parser for:
  - `SKILLSERVER_PERSISTENCE_DATA`
  - `SKILLSERVER_PERSISTENCE_DIR`
  - `SKILLSERVER_PERSISTENCE_DB_PATH`
- [ ] Add startup validation helpers for directory existence/writability.
- [ ] Add startup logging for resolved persistence mode and DB location.

**Test Deliverables:**
- [ ] Unit tests for valid/invalid config permutations.
- [ ] Unit tests for fail-fast behavior and disabled-mode passthrough.

---

### Acceptance Criteria

**Functional:**
- [ ] Persistence remains disabled unless explicitly enabled.
- [ ] Missing/unwritable persistence directory produces startup error before server start.
- [ ] Valid persistence config reaches runtime wiring without regressions.

**Testing:**
- [ ] Config parsing tests cover defaults, env set, invalid booleans, and invalid paths.
- [ ] Startup guard tests verify deterministic error messages.

---

### Dependencies

**Blocked By:**
- None

**Blocks:**
- WP-002
- WP-007
- WP-010

**Parallel Execution:**
- Can run in parallel with: None (foundational runtime package)
- Cannot run in parallel with: WP-002

---

### Risks

**Risk 1: Ambiguous runtime precedence causes operator confusion**
- Probability: Medium
- Impact: Medium
- Mitigation: Explicit config precedence tests and startup log of effective values.

**Risk 2: Partial validation allows non-durable paths**
- Probability: Low
- Impact: High
- Mitigation: Enforce writability checks and clear mount guidance in error text.
