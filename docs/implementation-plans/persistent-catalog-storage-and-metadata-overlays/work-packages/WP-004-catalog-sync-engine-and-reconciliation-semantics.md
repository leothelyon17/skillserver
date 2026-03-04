## WP-004: Catalog Sync Engine and Reconciliation Semantics

### Metadata

```yaml
WP_ID: WP-004
Title: Catalog Sync Engine and Reconciliation Semantics
Domain: Service Layer
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
Persistence correctness depends on deterministic synchronization from filesystem/Git discovery into source snapshots without erasing user overlays.

**Scope:**
- Implement startup full sync of discovered catalog items into `catalog_source_items`.
- Implement soft-delete reconciliation for missing source items.
- Implement targeted sync mode for a single Git repo during manual resync.

Excluded:
- Overlay mutation API logic (WP-006).
- UI metadata interactions (WP-008).

**Success Criteria:**
- [ ] Startup sync upserts all discovered source items.
- [ ] Missing source items are tombstoned (`deleted_at`) rather than hard-deleted.
- [ ] Targeted repo sync updates only affected items while preserving overlays.

---

### Technical Requirements

**Input Contracts:**
- Source and overlay repositories from WP-003.
- Existing filesystem discovery output from `FileSystemManager.ListCatalogItems()`.

**Output Contracts:**
- Sync service (for example `pkg/domain/catalog_sync_service.go`) with:
  - `SyncAll(discovered []CatalogItem) error`
  - `SyncRepo(repoName string, discovered []CatalogItem) error`

**Integration Points:**
- WP-006 and WP-005 consume synchronized source state.
- WP-007 invokes service at startup and manual git sync completion.

---

### Deliverables

**Code Deliverables:**
- [ ] Add sync orchestration service in domain layer.
- [ ] Add reconciliation logic for soft-delete and restore on reappearance.
- [ ] Add source-type/repo-aware filtering for targeted sync.
- [ ] Add operational logs with counts (upserted, tombstoned, unchanged).

**Test Deliverables:**
- [ ] Unit tests for full sync and targeted repo sync.
- [ ] Tests for tombstone behavior and reactivation of previously deleted items.
- [ ] Tests verifying overlays remain intact after any sync operation.

---

### Acceptance Criteria

**Functional:**
- [ ] Full sync converges DB source records to discovery snapshot.
- [ ] Overlay rows are unchanged by source sync operations.
- [ ] Targeted repo sync does not modify unrelated repo/local rows.

**Testing:**
- [ ] Sync tests cover create/update/delete/revive paths.
- [ ] Deterministic counts and logs are asserted in service tests.

---

### Dependencies

**Blocked By:**
- WP-003

**Blocks:**
- WP-006
- WP-007

**Parallel Execution:**
- Can run in parallel with: WP-005 (after WP-003)
- Cannot run in parallel with: WP-006 and WP-007

---

### Risks

**Risk 1: Reconciliation accidentally drops active rows**
- Probability: Medium
- Impact: High
- Mitigation: Use tombstones, not hard deletes, and assert transitions in tests.

**Risk 2: Targeted sync filter misses renamed/moved items**
- Probability: Medium
- Impact: Medium
- Mitigation: Filter by canonical item IDs and source repo metadata consistently.
