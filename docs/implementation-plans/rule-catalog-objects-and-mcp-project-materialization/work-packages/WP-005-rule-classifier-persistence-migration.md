## WP-005: Rule Classifier Persistence Migration

### Metadata

```yaml
WP_ID: WP-005
Title: Rule Classifier Persistence Migration
Domain: Data Layer
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
`catalog_source_items.classifier` currently accepts only `skill` and `prompt`. Rule items cannot persist until the schema and row-model validation are widened safely.

**Scope:**
- Add a new SQLite migration that widens classifier support to `skill|prompt|rule`.
- Update persistence classifier enums and row validation logic.
- Preserve existing source rows, indexes, and migration upgrade behavior.

Excluded:
- Catalog sync logic that produces rule rows (WP-006).
- Runtime gating and rollout docs beyond migration prerequisites.

**Success Criteria:**
- [ ] Existing databases migrate cleanly without losing data.
- [ ] Persistence repositories accept `rule` classifier rows after migration.
- [ ] Migration tests cover upgrade paths from prior schema versions.

---

### Technical Requirements

**Input Contracts:**
- Current migration chain in `pkg/persistence/migrate.go`.
- Current row validation in `pkg/persistence/catalog_row_models.go`.

**Output Contracts:**
- New schema migration version.
- Updated classifier enum/validation in persistence layer.
- Migration and repository tests covering pre/post-upgrade behavior.

**Integration Points:**
- WP-006 persists discovered rule items into the source snapshot table.
- WP-011 regression matrix validates upgrade behavior in realistic sequences.

---

### Deliverables

**Code Deliverables:**
- [ ] Add migration logic in `pkg/persistence/migrate.go` to support `rule`.
- [ ] Update `pkg/persistence/catalog_row_models.go` classifier validation.
- [ ] Update any repository queries or fixtures that assume only two classifiers.

**Test Deliverables:**
- [ ] Add migration tests for upgrading existing databases with populated source rows.
- [ ] Add repository tests proving `rule` rows round-trip correctly.

---

### Acceptance Criteria

**Functional:**
- [ ] Upgrading from previous schema versions preserves existing catalog rows.
- [ ] `rule` rows can be inserted, listed, and validated successfully.
- [ ] Indexes and foreign-key relationships remain intact.

**Testing:**
- [ ] Migration tests cover forward-only upgrade path and data preservation.
- [ ] Repository tests verify `rule` classifier filtering behaves like existing classifiers.

---

### Dependencies

**Blocked By:**
- WP-003

**Blocks:**
- WP-006
- WP-011

**Parallel Execution:**
- Can run in parallel with: WP-004
- Cannot run in parallel with: WP-003

---

### Risks

**Risk 1: SQLite constraint widening requires table rebuild and risks data loss**
- Probability: Medium
- Impact: High
- Mitigation: Use a migration strategy with explicit copy/rename validation and row-count assertions in tests.

**Risk 2: Tests miss mixed-version upgrade scenarios**
- Probability: Medium
- Impact: Medium
- Mitigation: Add migration coverage starting from a realistic pre-rule schema fixture, not only empty DBs.
