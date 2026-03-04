## WP-010: Operations Docs, Rollout Checklist, and Rollback Guidance

### Metadata

```yaml
WP_ID: WP-010
Title: Operations Docs, Rollout Checklist, and Rollback Guidance
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
Persistence introduces operational requirements around storage mounts, backup/restore, and rollback behavior that must be documented before rollout.

**Scope:**
- Update user/operator documentation for persistence configuration and behavior.
- Add rollout validation checklist and rollback steps.
- Document backup and recovery guidance for SQLite persistence files.

Excluded:
- New product features beyond documentation.
- Historical migration from external databases.

**Success Criteria:**
- [x] Operators can enable/disable persistence safely in Docker and Kubernetes.
- [x] Rollback to filesystem-only mode is documented and verified.
- [x] Recovery and troubleshooting steps are actionable.

---

### Technical Requirements

**Input Contracts:**
- Final runtime and behavior from WP-001 through WP-009.

**Output Contracts:**
- README updates.
- New operations runbook (for example `docs/operations/persistence-rollout-rollback.md`).

**Integration Points:**
- References verification artifacts from WP-009.

---

### Deliverables

**Documentation Deliverables:**
- [x] Update `README.md` with persistence env vars and quick-start examples.
- [x] Add Docker volume and Kubernetes PVC configuration examples.
- [x] Add rollout validation checklist:
  - startup checks
  - metadata persistence checks
  - manual git resync checks
- [x] Add rollback plan using `SKILLSERVER_PERSISTENCE_DATA=false`.
- [x] Add troubleshooting section (DB locked, mount missing, migration failure).

**Validation Deliverables:**
- [x] Confirm docs align with tested runtime behavior and endpoint contracts.

---

### Acceptance Criteria

**Functional:**
- [x] Runbook enables a new operator to deploy persistence mode without code knowledge.
- [x] Rollback path is clear and does not require destructive data operations.
- [x] Recovery notes cover backup/restore of SQLite DB file.

**Testing:**
- [x] Documentation steps are exercised once in local Docker flow.
- [x] Documentation steps are reviewed against integration test outputs.

---

### Execution Evidence (2026-03-04)

**Documentation Artifacts:**
- Updated [README.md](/home/jeff/skillserver/README.md) with:
  - persistence env vars/flags
  - local quick-start examples
  - Docker volume + Kubernetes PVC examples
  - metadata endpoint and mutability field contract notes
- Added [docs/operations/persistence-rollout-rollback.md](/home/jeff/skillserver/docs/operations/persistence-rollout-rollback.md) with:
  - startup, metadata persistence, and manual git-resync validation checklists
  - rollback guidance (`SKILLSERVER_PERSISTENCE_DATA=false`)
  - SQLite backup/restore guidance
  - troubleshooting for DB lock, mount/path, and migration failures

**Validation Commands:**
- `go test ./cmd/skillserver -run 'TestCatalogPersistenceCoordinator_|TestValidatePersistenceStartupConfig_' -count=1` ✅
- `go test ./pkg/domain -run 'TestCatalogEffectiveService_|TestCatalogSyncService_' -count=1` ✅
- `go test ./pkg/web -run 'TestCatalogMetadataEndpoints_|TestSyncGitRepo_' -count=1` ✅
- `docker build -t skillserver:wp010-local .` ✅
- `/tmp/wp010_docker_validate_local.sh` (local Docker persistence flow) ✅
- `/tmp/wp010_rollback_validate.sh` (filesystem-only rollback endpoint behavior) ✅

---

### Dependencies

**Blocked By:**
- WP-001
- WP-009

**Blocks:**
- None

**Parallel Execution:**
- Can run in parallel with: None
- Cannot run in parallel with: N/A

---

### Risks

**Risk 1: Documentation drifts from final behavior**
- Probability: Medium
- Impact: Medium
- Mitigation: Author docs after regression suite completion and reference concrete test outputs.

**Risk 2: Incomplete rollback guidance increases downtime**
- Probability: Low
- Impact: High
- Mitigation: Keep rollback steps short, tested, and explicitly ordered.
