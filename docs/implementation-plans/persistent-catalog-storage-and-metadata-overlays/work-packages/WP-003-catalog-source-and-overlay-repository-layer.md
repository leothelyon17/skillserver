## WP-003: Catalog Source and Overlay Repository Layer

### Metadata

```yaml
WP_ID: WP-003
Title: Catalog Source and Overlay Repository Layer
Domain: Data Layer
Priority: High
Estimated_Effort: 5 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
The persistence backend requires explicit data access methods for source snapshots and metadata overlays with clear transactional boundaries.

**Scope:**
- Implement data models and repository interfaces for:
  - Source snapshot upsert/read/list operations.
  - Metadata overlay upsert/read/delete operations.
- Ensure overlay writes never overwrite source data columns.

Excluded:
- Reconciliation orchestration logic (WP-004).
- Effective merge projection logic (WP-005).

**Success Criteria:**
- [ ] Source and overlay CRUD operations are implemented and tested.
- [ ] Repositories enforce strict separation between source and overlay writes.
- [ ] Data access methods support later sync and API layers.

---

### Technical Requirements

**Input Contracts:**
- Migrated SQLite schema and DB lifecycle from WP-002.

**Output Contracts:**
- Repository types/functions in `pkg/persistence` (or equivalent).
- Typed row structs and mapping helpers for source and overlay records.

**Integration Points:**
- WP-004 consumes source upsert/soft-delete methods.
- WP-005 consumes source + overlay read methods.
- WP-006 consumes overlay mutation methods via service layer.

---

### Deliverables

**Code Deliverables:**
- [ ] Add source repository (for example `pkg/persistence/catalog_source_repository.go`).
- [ ] Add overlay repository (for example `pkg/persistence/catalog_overlay_repository.go`).
- [ ] Add row model helpers for JSON fields and timestamps.
- [ ] Add deterministic query helpers for `item_id` and optional source filters.

**Test Deliverables:**
- [ ] Repository unit tests for create/upsert/read/list/delete operations.
- [ ] Tests for JSON payload round-trip (`custom_metadata_json`, `labels_json`).
- [ ] Tests ensuring overlay mutation does not alter source rows.

---

### Acceptance Criteria

**Functional:**
- [ ] Source upsert updates mutable source columns and preserves overlay table state.
- [ ] Overlay upsert supports nullable overrides and empty metadata map semantics.
- [ ] Repository methods return deterministic ordering for stable indexing.

**Testing:**
- [ ] CRUD tests pass against isolated SQLite test DB.
- [ ] Edge cases cover missing rows, null overrides, and malformed JSON rejection.

---

### Dependencies

**Blocked By:**
- WP-002

**Blocks:**
- WP-004
- WP-005

**Parallel Execution:**
- Can run in parallel with: None
- Cannot run in parallel with: WP-004 and WP-005

---

### Risks

**Risk 1: Repository API shape mismatches service-layer needs**
- Probability: Medium
- Impact: Medium
- Mitigation: Define required read/write contracts before implementation.

**Risk 2: JSON field validation inconsistencies**
- Probability: Medium
- Impact: Medium
- Mitigation: Centralize JSON marshal/unmarshal validation with explicit tests.
