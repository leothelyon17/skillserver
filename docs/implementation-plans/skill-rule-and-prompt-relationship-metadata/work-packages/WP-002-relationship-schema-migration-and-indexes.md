## WP-002: Relationship Schema Migration and Indexes

### Metadata

```yaml
WP_ID: WP-002
Title: Relationship Schema Migration and Indexes
Domain: data
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/database-architect.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for SQLite schema design, foreign keys, indexes, and migration safety.
Priority: High
Estimated_Effort: 4 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-11
Started_Date: 2026-03-11
Completed_Date: 2026-03-11
```

---

### Description

**Context:**
ADR-008 requires normalized persistence so classifier validation, reverse lookup, and single-prompt-per-skill enforcement do not rely on ad hoc JSON inside metadata overlays.

**Scope:**
- Add the next SQLite migration in `pkg/persistence/migrate.go` for:
  - `catalog_skill_rule_relationships`
  - `catalog_skill_prompt_relationships`
- Add supporting indexes for reverse lookup by `rule_item_id` and `prompt_item_id`.
- Extend migration tests for schema versioning, idempotency, and foreign-key behavior.

Excluded:
- Repository query methods and row-model validation (WP-003).
- Domain-level classifier validation and pruning logic (WP-004).

**Success Criteria:**
- [x] Migration upgrades cleanly from the current latest schema to the relationship-aware schema.
- [x] Relationship tables and indexes are created idempotently.
- [x] Primary-key shape enforces one prompt per skill at the storage layer.

---

### Technical Requirements

**Input Contracts:**
- ADR-008 table definitions and index requirements.
- Finalized relationship contract in `skill-rule-and-prompt-relationship-metadata-implementation-plan.md` section `WP-001 Finalized Relationship Contract (Authoritative)`.
- Existing migration framework in `pkg/persistence/migrate.go`.

**Output Contracts:**
- Additive migration entry with deterministic ordering.
- Migration tests covering both fresh bootstrap and repeat-run idempotency.

**Integration Points:**
- WP-003 repositories rely on these tables and indexes.
- WP-004 service logic relies on storage-level uniqueness for skill->prompt rows.

---

### Deliverables

**Code Deliverables:**
- [x] Add migration DDL to `pkg/persistence/migrate.go`.
- [x] Add or update migration coverage in `pkg/persistence/migrate_test.go`.

**Test Deliverables:**
- [x] Schema bootstrap test for the new relationship tables.
- [x] Idempotency test for rerunning migrations.
- [x] Constraint test proving duplicate prompt rows for one skill are rejected by schema shape.

---

### Acceptance Criteria

**Functional:**
- [x] Latest schema version includes both relationship tables.
- [x] Reverse lookup indexes exist for prompt and rule relationships.
- [x] Migration stays additive and does not require data backfill.

**Testing:**
- [x] Migration tests pass with stable schema assertions.
- [x] Existing migration coverage continues to pass unchanged.

---

### Dependencies

**Blocked By:**
- WP-001

**Blocks:**
- WP-003
- WP-004
- WP-008

**Parallel Execution:**
- Can run in parallel with: None.
- Cannot run in parallel with: WP-003 onward.

---

### Risks

**Risk 1: Weak schema shape allows ambiguous prompt ownership**
- Probability: Medium
- Impact: High
- Mitigation: Use `skill_item_id` as the primary key on the prompt relationship table.

**Risk 2: Missing reverse indexes slows metadata/detail lookups**
- Probability: Medium
- Impact: Medium
- Mitigation: Add prompt and rule reverse indexes in the initial migration rather than as a follow-up.
