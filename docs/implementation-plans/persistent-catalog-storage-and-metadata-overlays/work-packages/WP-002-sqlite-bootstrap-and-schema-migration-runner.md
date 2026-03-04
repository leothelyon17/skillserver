## WP-002: SQLite Bootstrap and Schema Migration Runner

### Metadata

```yaml
WP_ID: WP-002
Title: SQLite Bootstrap and Schema Migration Runner
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
The system needs a durable transactional storage engine with schema versioning while remaining self-contained for Docker volume/PVC deployments.

**Scope:**
- Add SQLite connection/bootstrap layer.
- Add migration runner with schema version tracking.
- Create initial schema for source snapshots, metadata overlays, and system state.

Excluded:
- Business-level sync/reconciliation logic (WP-004).
- API endpoints and UI behavior (WP-006 and WP-008).

**Success Criteria:**
- [ ] DB initializes under configured persistence path.
- [ ] Migrations run idempotently and track version.
- [ ] Initial schema matches ADR-004 data model requirements.

---

### Technical Requirements

**Input Contracts:**
- Persistence runtime config and validated DB path from WP-001.

**Output Contracts:**
- Persistence DB bootstrap package (for example `pkg/persistence/`).
- Migration artifacts and runner API callable at startup.

**Integration Points:**
- WP-003 repository layer depends on initialized schema and DB handle.
- WP-007 startup wiring invokes migration runner during service boot.

---

### Deliverables

**Code Deliverables:**
- [ ] Add SQLite bootstrap and lifecycle helpers (for example `pkg/persistence/sqlite.go`).
- [ ] Add migration runner (for example `pkg/persistence/migrate.go`).
- [ ] Add migration for:
  - `catalog_source_items`
  - `catalog_metadata_overlays`
  - `system_state`
- [ ] Add indexes and constraints for `item_id`, classifier/source filters, and lookup paths.

**Test Deliverables:**
- [ ] Unit tests for DB open/close and migration idempotency.
- [ ] Migration progression test from empty DB to current schema version.

---

### Acceptance Criteria

**Functional:**
- [ ] DB file is created/opened at resolved path in persistence mode.
- [ ] Migration runner is repeatable and no-op on latest schema.
- [ ] Schema includes soft-delete and overlay fields required by ADR.

**Testing:**
- [ ] Migration tests validate both fresh bootstraps and repeated runs.
- [ ] Schema tests validate table existence and critical constraints.

---

### Dependencies

**Blocked By:**
- WP-001

**Blocks:**
- WP-003

**Parallel Execution:**
- Can run in parallel with: None
- Cannot run in parallel with: WP-003

---

### Risks

**Risk 1: Migration runner introduces non-deterministic schema state**
- Probability: Low
- Impact: High
- Mitigation: Single authoritative migration chain with version assertions in tests.

**Risk 2: SQLite defaults cause lock contention under test load**
- Probability: Medium
- Impact: Medium
- Mitigation: Configure and test pragmatic defaults (busy timeout, journaling mode).
