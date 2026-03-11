## WP-009: Rollout and Operator Documentation

### Metadata

```yaml
WP_ID: WP-009
Title: Rollout and Operator Documentation
Domain: documentation
Execution_Agent_Prompt:
Agent_Selection_Source: blank
Agent_Selection_Rationale: No highly relatable installed prompt for this narrowly scoped documentation and rollout package.
Priority: Medium
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-11
Started_Date: 2026-03-11
Completed_Date: 2026-03-11
```

---

### Description

**Context:**
The relationship feature is additive, but users and operators still need clear guidance on new REST/MCP surfaces, expected write authority, and rollback expectations.

**Scope:**
- Update user-facing API/tooling documentation.
- Add operator rollout and rollback guidance for the relationship metadata feature.
- Add release notes summarizing the new capability and its intentionally limited write scope.

Excluded:
- Feature implementation or new runtime flags beyond what the code introduces.

**Success Criteria:**
- [x] README documents the new REST and MCP relationship read surfaces.
- [x] Operations docs explain rollout expectations, validation, and rollback considerations.
- [x] Release notes capture user-visible behavior and known v1 limits.

---

### Technical Requirements

**Input Contracts:**
- Verified behavior from WP-008.
- Existing documentation patterns in `README.md`, `docs/operations/`, and `docs/releases/`.

**Output Contracts:**
- Updated API/tool contract documentation.
- One rollout/rollback guide for the relationship feature.
- One release-note artifact.

**Integration Points:**
- Consumes verified outputs from WP-008 and should not start earlier.

---

### Deliverables

**Documentation Deliverables:**
- [x] Update `README.md` with REST and MCP relationship surface details.
- [x] Add `docs/operations/skill-relationship-metadata-rollout-rollback.md`.
- [x] Add a date-stamped release note under `docs/releases/`.

**Review Deliverables:**
- [x] Document v1 limits clearly:
  - GUI/REST skill-owned writes only
  - MCP read-only
  - no tile-level relationship rendering

---

### Acceptance Criteria

**Functional:**
- [x] Documentation matches verified behavior from WP-008.
- [x] Rollout guidance includes validation checks and rollback considerations.
- [x] Release notes call out the additive API/MCP contracts and v1 limitations.

**Testing:**
- [x] Documentation references existing test evidence or verification steps from WP-008.

---

### Dependencies

**Blocked By:**
- WP-008

**Blocks:**
- None.

**Parallel Execution:**
- Can run in parallel with: None.
- Cannot run in parallel with: WP-008.

---

### Risks

**Risk 1: Docs describe intended rather than verified behavior**
- Probability: Medium
- Impact: Medium
- Mitigation: Do not start this package until WP-008 evidence is complete.

**Risk 2: Users assume prompt/rule-side editing exists**
- Probability: Medium
- Impact: Low
- Mitigation: Call out skill-owned write authority and MCP read-only scope prominently in docs and release notes.
