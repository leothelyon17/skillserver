## WP-005: Git Syncer Auth Integration, Checkout Semantics, and Redacted Status Reporting

### Metadata

```yaml
WP_ID: WP-005
Title: Git Syncer Auth Integration, Checkout Semantics, and Redacted Status Reporting
Domain: Service Layer
Priority: High
Estimated_Effort: 6 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-07
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
The current syncer only understands `[]string` URLs and clones/pulls without auth. Private repo support requires a typed repo model, just-in-time credential resolution on every sync path, and safer status reporting that does not destroy working content on auth failures.

**Scope:**
- Refactor `pkg/git/syncer.go` to operate on typed repo configs rather than raw URLs.
- Apply auth resolution on startup sync, periodic sync, add/update sync, and manual sync.
- Add redacted per-repo sync status and last-error tracking.
- Define deterministic checkout-path and `pkg/domain` read-only registration behavior for the expanded repo model.

Excluded:
- REST contract changes and UI rendering.
- Stored-secret API submission flows.

**Success Criteria:**
- [ ] Public and private repos use the same sync orchestration path.
- [ ] Failed auth does not delete existing repo content.
- [ ] Manual `POST /api/git-repos/:id/sync` shares the same resolution/status path as automatic syncs.

---

### Technical Requirements

**Input Contracts:**
- Repo model from WP-002.
- Credential providers/auth builder from WP-003 and WP-004.
- Manual repo sync hook behavior already wired in `cmd/skillserver/main.go`.

**Output Contracts:**
- Refactored syncer interfaces in `pkg/git`.
- Safe sync-status model accessible to API handlers.
- Updated `pkg/domain.FileSystemManager` integration for git repo registration.

**Integration Points:**
- WP-006 reads status summaries and repo IDs from the new syncer/config contract.
- WP-008 validates startup, periodic, and manual sync parity.

---

### Deliverables

**Code Deliverables:**
- [ ] Refactor syncer repo state from `[]string` to typed repo configs.
- [ ] Wire auth into clone and pull operations for all sync entry points.
- [ ] Add redacted sync status storage keyed by repo `id`.
- [ ] Update repo add/update/delete flows to use deterministic checkout names and safe cleanup behavior.
- [ ] Update `pkg/domain` git repo registration to track repo checkouts without relying on raw URLs.

**Test Deliverables:**
- [ ] Syncer tests for public repos, auth-required repos, and retry/update flows.
- [ ] Tests ensuring failed auth leaves existing checkout intact.
- [ ] Tests proving manual sync uses the same auth path as startup/periodic sync.

---

### Acceptance Criteria

**Functional:**
- [ ] Clone and pull both use the same auth-resolution path.
- [ ] Periodic sync resolves env/file/stored credentials fresh on each attempt.
- [ ] Redacted status is available even when sync fails before catalog rebuild.
- [ ] Checkout cleanup on repo deletion uses sanitized deterministic paths.

**Testing:**
- [ ] Syncer tests cover startup sync, periodic sync, manual sync, and add/update/remove behavior.
- [ ] Regression tests prove public-repo behavior remains unchanged.

---

### Dependencies

**Blocked By:**
- WP-002
- WP-003
- WP-004

**Blocks:**
- WP-006
- WP-008

**Parallel Execution:**
- Can run in parallel with: None
- Cannot run in parallel with: WP-002, WP-003, WP-004

---

### Risks

**Risk 1: Syncer and config state diverge during update/toggle flows**
- Probability: Medium
- Impact: High
- Mitigation: Refactor handlers/syncer boundaries around one typed repo contract instead of rebuilding from URL slices ad hoc.

**Risk 2: Checkout-path changes strand existing git-backed skill directories**
- Probability: Medium
- Impact: Medium
- Mitigation: Define path compatibility rules explicitly and test migration/delete semantics.

