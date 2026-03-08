## WP-011: Integration, Safety, and Regression Matrix

### Metadata

```yaml
WP_ID: WP-011
Title: Integration, Safety, and Regression Matrix
Domain: Quality Engineering
Priority: High
Estimated_Effort: 5 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-08
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
ADR-007 changes classifier semantics, persistence, export routes, write-capable MCP tools, and UI behavior. A dedicated regression package is required before rollout because path safety and backward compatibility are both first-class requirements.

**Scope:**
- Build a cross-surface regression matrix for domain, persistence, REST, MCP, and UI behavior.
- Validate no-write guarantees for dry-run flows and invalid-path requests.
- Validate legacy compatibility for skill export and existing catalog behavior.
- Publish CI-friendly verification commands and rollout gates.

Excluded:
- Operational runbook authoring (WP-012).

**Success Criteria:**
- [ ] All new write-capable flows have explicit safety tests.
- [ ] Legacy skill export and existing skill/prompt catalog behavior remain covered.
- [ ] Rollout documentation can depend on verified evidence instead of intended behavior.

---

### Technical Requirements

**Input Contracts:**
- Persistence migration behavior from WP-005.
- REST, MCP, and UI implementations from WP-008, WP-009, and WP-010.

**Output Contracts:**
- Expanded test coverage in `pkg/domain`, `pkg/persistence`, `pkg/web`, and `pkg/mcp`.
- Verification command matrix/checklist that operators can use as rollout gates.

**Integration Points:**
- WP-012 uses this matrix as rollout go/no-go input.

---

### Deliverables

**Code Deliverables:**
- [ ] Add or extend integration tests for export/materialization REST flows.
- [ ] Add or extend MCP regression tests for gated tool registration and dry-run behavior.
- [ ] Add or extend persistence tests for migration and rule-row lifecycle behavior.
- [ ] Add verification command guidance in `tests/README.md` or equivalent.

**Test Deliverables:**
- [ ] Rule discovery/search regression coverage.
- [ ] Path-safety regression coverage.
- [ ] Legacy skill-export compatibility coverage.
- [ ] UI verification coverage for capability-gated actions.

---

### Acceptance Criteria

**Functional:**
- [ ] Dry-run requests perform no writes.
- [ ] Invalid paths and disallowed roots fail across both REST and MCP surfaces.
- [ ] Existing skill export and import workflows remain compatible.
- [ ] Rule indexing and filtering behave correctly with and without persistence enabled.

**Testing:**
- [ ] Regression commands are documented and runnable in CI/local environments.
- [ ] The matrix covers at least one local skill, one repo-backed skill, one prompt, and one rule scenario.

---

### Dependencies

**Blocked By:**
- WP-005
- WP-008
- WP-009
- WP-010

**Blocks:**
- WP-012

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-008, WP-009, WP-010

---

### Risks

**Risk 1: Safety coverage focuses on happy path and misses write-boundary failures**
- Probability: Medium
- Impact: High
- Mitigation: Make invalid destination tests first-class gates, not secondary edge cases.

**Risk 2: UI verification remains too manual to support confident rollout**
- Probability: Medium
- Impact: Medium
- Mitigation: Document a short deterministic checklist and automate stable assertions where practical.
