## WP-004: Runtime Flags and Capability Gates

### Metadata

```yaml
WP_ID: WP-004
Title: Runtime Flags and Capability Gates
Domain: Infrastructure
Priority: High
Estimated_Effort: 3 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-08
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Rule indexing and MCP materialization are rollout-sensitive features. They need explicit runtime controls, startup validation, and capability exposure to downstream adapters.

**Scope:**
- Extend catalog runtime config with rule enablement, rule directory allowlist, and rule filename allowlist.
- Extend MCP/runtime config with materialization enablement and allowed destination roots.
- Wire validated config into startup, server options, and runtime capability responses used by the UI.

Excluded:
- Domain detection logic itself (WP-003).
- Persistence migration (WP-005).
- Export/materialization service logic (WP-001, WP-007).

**Success Criteria:**
- [ ] Invalid rule/materialization config fails fast at startup.
- [ ] Materialization tooling is independently gated from existing read-only catalog features.
- [ ] UI-visible runtime capabilities reflect actual server configuration.

---

### Technical Requirements

**Input Contracts:**
- Existing config parsing patterns in `cmd/skillserver/config.go`.
- Existing runtime capability endpoint in `pkg/web`.

**Output Contracts:**
- Extended config structs and validation helpers.
- Startup wiring in `cmd/skillserver/main.go`.
- MCP server options and UI-visible capability fields.

**Integration Points:**
- WP-006 consumes rule runtime config during catalog discovery.
- WP-007 and WP-009 consume destination-root and materialization gate configuration.
- WP-010 uses runtime capability flags to show or hide actions.

---

### Deliverables

**Code Deliverables:**
- [ ] Add rule-related catalog flags/env vars in `cmd/skillserver/config.go`.
- [ ] Add materialization gate and allowed-root flags/env vars in `cmd/skillserver/config.go`.
- [ ] Wire validated config through `cmd/skillserver/main.go`, `pkg/mcp/server.go`, and runtime capability responses.

**Test Deliverables:**
- [ ] Add config parsing/validation tests for rule dirs, rule filenames, and allowed destination roots.
- [ ] Add capability tests proving materialization remains disabled unless explicitly enabled.

---

### Acceptance Criteria

**Functional:**
- [ ] Empty or relative allowed destination roots are rejected.
- [ ] Materialization write tools remain disabled by default.
- [ ] Effective rule/materialization config is available to dependent services and adapters.

**Testing:**
- [ ] Config tests cover precedence, normalization, dedupe, and invalid values.
- [ ] Runtime capability tests cover enabled and disabled states.

---

### Dependencies

**Blocked By:**
- WP-003

**Blocks:**
- WP-006
- WP-007
- WP-008
- WP-009
- WP-010
- WP-012

**Parallel Execution:**
- Can run in parallel with: WP-005
- Cannot run in parallel with: WP-003

---

### Risks

**Risk 1: Capability flags drift between startup wiring and adapter behavior**
- Probability: Medium
- Impact: Medium
- Mitigation: Centralize config in shared structs and reuse them across server constructors.

**Risk 2: Overly permissive root parsing weakens write safety**
- Probability: Low
- Impact: High
- Mitigation: Require normalized absolute roots and reject empty, relative, or duplicate values.
