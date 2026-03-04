## WP-007: Startup and Manual Git Resync Persistence Wiring

### Metadata

```yaml
WP_ID: WP-007
Title: Startup and Manual Git Resync Persistence Wiring
Domain: Infrastructure
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
Persistence logic must be integrated into startup and existing git sync workflows so source snapshots remain convergent with filesystem/Git state.

**Scope:**
- Wire persistence initialization and full sync into startup path when enabled.
- Wire manual `POST /api/git-repos/:id/sync` flow to repo-targeted persistence sync.
- Ensure search rebuild runs from effective catalog data after sync/update.

Excluded:
- Metadata API handlers (WP-006).
- UI behavior (WP-008).

**Success Criteria:**
- [ ] Startup performs migration + full sync before serving requests in persistence mode.
- [ ] Manual Git resync updates only affected repo source rows.
- [ ] Search index rebuilds from effective data after sync operations.

---

### Technical Requirements

**Input Contracts:**
- Runtime config from WP-001.
- Sync service from WP-004.
- Effective projection service from WP-005.

**Output Contracts:**
- Startup wiring in `cmd/skillserver/main.go`.
- Web/git callback integration in `pkg/web` and `pkg/git`.

**Integration Points:**
- WP-009 integration tests validate startup + manual sync durability behavior.
- WP-010 documentation reflects the final runtime behavior.

---

### Deliverables

**Code Deliverables:**
- [ ] Inject persistence components into runtime initialization.
- [ ] Trigger full sync on startup when persistence enabled.
- [ ] Trigger repo-scoped sync in manual git sync handler.
- [ ] Rebuild search index from effective projection after startup/resync metadata changes.

**Test Deliverables:**
- [ ] Integration tests for startup synchronization.
- [ ] Integration tests for manual git sync overlay preservation.
- [ ] Tests for disabled-mode compatibility (no persistence path required).

---

### Acceptance Criteria

**Functional:**
- [ ] Persistence-enabled startup converges DB snapshot and serves effective catalog.
- [ ] Manual repo sync does not remove overlays for unchanged items.
- [ ] Non-persistence mode behavior remains unchanged.

**Testing:**
- [ ] End-to-end sync flow tests pass with representative local + git fixtures.
- [ ] Search rebuild assertions validate effective metadata propagation.

---

### Dependencies

**Blocked By:**
- WP-001
- WP-004

**Blocks:**
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-006 (once WP-004 complete)
- Cannot run in parallel with: WP-009

---

### Risks

**Risk 1: Startup ordering bug serves stale catalog state**
- Probability: Medium
- Impact: High
- Mitigation: Enforce explicit init sequence with integration test coverage.

**Risk 2: Repo-scoped sync updates unrelated records**
- Probability: Low
- Impact: Medium
- Mitigation: Use scoped filters by repo/source and verify affected-row counts.
