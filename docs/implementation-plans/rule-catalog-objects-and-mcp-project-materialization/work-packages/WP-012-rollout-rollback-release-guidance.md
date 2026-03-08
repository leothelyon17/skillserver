## WP-012: Rollout, Rollback, and Release Guidance

### Metadata

```yaml
WP_ID: WP-012
Title: Rollout, Rollback, and Release Guidance
Domain: Documentation
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
Operators need a staged enablement story for shared export recovery, rule indexing, and write-capable materialization. Rollback must be short, explicit, and tied to real runtime flags.

**Scope:**
- Add a rollout/rollback runbook under `docs/operations/`.
- Update README and release-note guidance for new flags, endpoints, and MCP tools.
- Document phased enablement order and rollback steps using runtime gates.

Excluded:
- New feature implementation or additional tests beyond documenting verified behavior.

**Success Criteria:**
- [ ] Rollout can proceed in phases with explicit go/no-go checks.
- [ ] Rollback relies on configuration gates, not destructive data operations.
- [ ] User/operator docs match the verified behavior from WP-011.

---

### Technical Requirements

**Input Contracts:**
- Runtime gates from WP-004.
- Verified behavior and command matrix from WP-011.

**Output Contracts:**
- New runbook in `docs/operations/`.
- README updates and release note artifact(s) under `docs/releases/`.
- Final plan cross-links to rollout artifacts if needed.

**Integration Points:**
- Final rollout depends on validated behavior from WP-011.

---

### Deliverables

**Documentation Deliverables:**
- [ ] Add `docs/operations/rule-catalog-materialization-rollout-rollback.md`.
- [ ] Update `README.md` with rule-catalog/materialization configuration and usage notes.
- [ ] Add release-note guidance in `docs/releases/` for ADR-007 rollout.

**Operational Deliverables:**
- [ ] Phased rollout checklist covering: shared export seam, rule indexing, MCP materialization, UI enablement.
- [ ] Rollback checklist covering: disable materialization, disable rules, validate legacy export fallback.
- [ ] Post-rollback verification steps.

---

### Acceptance Criteria

**Functional:**
- [ ] Runbook documents required flags, preconditions, verification steps, and rollback order.
- [ ] README links to the operations runbook and reflects new API/MCP surfaces.
- [ ] Release guidance calls out backward-compatibility and migration implications.

**Testing/Verification:**
- [ ] Documentation references only verified commands and behaviors from WP-011.
- [ ] Rollback steps can be executed without destructive schema rollback.

---

### Dependencies

**Blocked By:**
- WP-004
- WP-011

**Blocks:**
- None

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-011

---

### Risks

**Risk 1: Rollout docs promise behavior that was not actually validated**
- Probability: Medium
- Impact: High
- Mitigation: Treat WP-011 outputs as a hard prerequisite and link to concrete verification commands.

**Risk 2: Rollback instructions omit legacy export fallback validation**
- Probability: Medium
- Impact: Medium
- Mitigation: Include explicit post-rollback checks for GUI skill export and catalog search behavior.
