## WP-001: Taxonomy Schema v2 and Indexes

### Metadata

```yaml
WP_ID: WP-001
Title: Taxonomy Schema v2 and Indexes
Domain: Data Layer
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
ADR-005 requires first-class taxonomy objects and assignment relations persisted in SQLite. The current schema only has source snapshot and metadata overlay tables.

**Scope:**
- Add migration `v2` statements in `pkg/persistence/migrate.go` for:
  - `catalog_domains`
  - `catalog_subdomains`
  - `catalog_tags`
  - `catalog_item_taxonomy_assignments`
  - `catalog_item_tag_assignments`
- Add supporting indexes for primary/secondary domain and tag lookups.
- Ensure foreign keys and delete policies enforce assignment safety (`RESTRICT` and `CASCADE` where appropriate).

Excluded:
- Repository query and mutation logic (WP-002).
- Backfill logic from legacy labels (WP-005).

**Success Criteria:**
- [ ] Migration runner upgrades cleanly from v1 to v2.
- [ ] Tables/indexes are created idempotently.
- [ ] Foreign key policies enforce required integrity constraints.

---

### Technical Requirements

**Input Contracts:**
- Existing migration framework in `pkg/persistence/migrate.go`.
- Existing v1 tables from ADR-004.

**Output Contracts:**
- Migration entries added to `schemaMigrations` with deterministic version ordering.
- Migration tests validating schema version and DDL application.

**Integration Points:**
- WP-002 repositories rely on these tables/indexes.
- WP-005 backfill relies on unique key constraints and assignment table shape.

---

### Deliverables

**Code Deliverables:**
- [ ] Add migration v2 DDL statements to `pkg/persistence/migrate.go`.
- [ ] Add indexes supporting taxonomy filters and join performance.
- [ ] Add/extend migration coverage in `pkg/persistence/migrate_test.go`.

**Test Deliverables:**
- [ ] Upgrade path test from empty DB to latest schema.
- [ ] Idempotency test re-running migrations.
- [ ] Foreign-key constraint test for key delete restrictions.

---

### Acceptance Criteria

**Functional:**
- [ ] Latest schema version increments to include taxonomy migration.
- [ ] Database contains all required taxonomy tables and indexes.
- [ ] Delete constraints prevent orphaning assignment rows.

**Testing:**
- [ ] Migration tests pass with stable schema assertions.
- [ ] No regressions in existing migration tests.

---

### Dependencies

**Blocked By:**
- None.

**Blocks:**
- WP-002
- WP-003
- WP-005

**Parallel Execution:**
- Can run in parallel with: None (first package).
- Cannot run in parallel with: WP-002 onward.

---

### Risks

**Risk 1: Incorrect FK policy causes accidental taxonomy data loss**
- Probability: Medium
- Impact: High
- Mitigation: Use explicit `ON DELETE RESTRICT` for taxonomy objects and validate with tests.

**Risk 2: Missing indexes degrade taxonomy-filter search latency**
- Probability: Medium
- Impact: Medium
- Mitigation: Add indexes during migration and verify query plans in WP-002.
