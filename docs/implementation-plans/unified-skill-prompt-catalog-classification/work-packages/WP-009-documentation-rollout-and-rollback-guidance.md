## WP-009: Documentation, Rollout, and Rollback Guidance

### Metadata

```yaml
WP_ID: WP-009
Title: Documentation, Rollout, and Rollback Guidance
Domain: Documentation
Priority: Medium
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-04
Started_Date: 2026-03-04
Completed_Date: 2026-03-04
```

---

### Description

**Context:**
ADR-003 introduces additive catalog APIs, classifier semantics, and runtime knobs. Operators and users need clear rollout and rollback instructions.

**Scope:**
- Update README/API docs for catalog endpoints and classifier behavior.
- Add operations runbook for rollout checks and rollback path.
- Document optional MCP catalog parity usage if WP-007 is included.

Excluded:
- New feature development.

**Success Criteria:**
- [x] Docs reflect final API contracts and runtime config.
- [x] Rollback path is explicit and executable.
- [x] Operational checks verify prompt catalog visibility and search filtering.

---

### Technical Requirements

**Input Contracts:**
- Final behavior from WP-001..WP-008.

**Output Contracts:**
- Updated `README.md` and/or docs pages for catalog endpoints.
- New operations document under `docs/operations/` for ADR-003 rollout.

**Integration Points:**
- Final release readiness package for stakeholders.

---

### Deliverables

**Documentation Deliverables:**
- [x] Update API usage docs for `/api/catalog` and `/api/catalog/search`.
- [x] Document classifier values, semantics, and filtering examples.
- [x] Add rollout/rollback runbook with validation checklist.

**Operational Deliverables:**
- [x] Include kill-switch/config rollback steps.
- [x] Include post-deploy smoke checks for catalog/search/UI behavior.

---

### Acceptance Criteria

**Functional:**
- [x] Operator can roll forward and roll back without ambiguity.
- [x] User-facing docs accurately describe catalog behavior.

**Quality:**
- [x] Documentation reviewed and linked from implementation plan.
- [x] Release notes include backward-compatibility statement.

---

### Dependencies

**Blocked By:**
- WP-008

**Blocks:**
- None

**Parallel Execution:**
- Can run in parallel with: minor prep while WP-008 executes
- Cannot run in parallel with: final acceptance before WP-008 sign-off

---

### Risks

**Risk 1: Docs drift from final implementation details**
- Probability: Medium
- Impact: Medium
- Mitigation: Draft only after WP-008 validation and link to tested examples.

**Risk 2: Rollback steps omit config/index rebuild dependencies**
- Probability: Low
- Impact: High
- Mitigation: Validate rollback in a staging-like environment before release.

---

### Execution Outcome

**Artifacts Delivered:**
- `README.md` updated with:
  - catalog runtime config (`SKILLSERVER_CATALOG_ENABLE_PROMPTS`, `SKILLSERVER_CATALOG_PROMPT_DIRS`)
  - additive REST catalog endpoint contract and classifier semantics
  - additive MCP catalog tool guidance and migration notes
  - ADR-003 rollout/rollback section linking runbook
- New operations runbook:
  - `docs/operations/unified-catalog-rollout-rollback.md`
- New release notes document:
  - `docs/releases/2026-03-04-adr-003-unified-catalog-release-notes.md`
- Implementation plan linkage updates:
  - `docs/implementation-plans/unified-skill-prompt-catalog-classification/unified-skill-prompt-catalog-classification-implementation-plan.md`
