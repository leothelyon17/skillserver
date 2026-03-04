# Implementation Plan: ADR-004 Persistent Catalog Storage and Metadata Overlays

**Date Created:** 2026-03-04
**Project Owner:** @jeff
**Target Completion:** 2026-03-18
**Actual Completion:** 2026-03-04
**Status:** COMPLETED
**Source ADR:** [ADR-004: Persistent Catalog Storage and Metadata Overlays](../../adrs/004-persistent-catalog-storage-and-metadata-overlays.md)

---

## Project Overview

### Goal
Implement embedded SQLite-backed persistence for catalog source snapshots plus user-editable metadata overlays so metadata survives restarts/redeployments while preserving existing Git content read-only behavior.

### Success Criteria
- [x] Persistence mode is explicitly controlled by `SKILLSERVER_PERSISTENCE_DATA=true`. ✅
- [x] Startup fails fast with actionable errors when persistence is enabled but storage is misconfigured. ✅
- [x] Git and local catalog items are synchronized into SQLite on startup and manual Git resync. ✅
- [x] Git-backed content remains immutable while metadata remains writable and durable. ✅
- [x] Search results reflect effective (overlay-resolved) metadata. ✅
- [x] Existing non-persistence content workflows remain backward-compatible. ✅

### Scope

**In Scope:**
- Persistence runtime config and startup validation.
- SQLite schema/migrations for source snapshot + metadata overlays.
- Sync/reconciliation engine for startup and repo-specific resync.
- Effective record resolution logic and mutability flags.
- API additions for metadata overlay CRUD and additive response fields.
- UI metadata editing for all catalog item types, with content write guard preservation.
- Integration/regression tests and operations documentation.

**Out of Scope:**
- External managed databases (Postgres, cloud services).
- Replacing filesystem/Git as canonical content source.
- New authentication/authorization models.
- Multi-node active-active synchronization.

### Constraints
- Technical: Preserve existing filesystem discovery and Git sync flow semantics.
- Deployment: Must work with Docker named volumes and Kubernetes PVC mounts.
- Compatibility: Existing REST/MCP content flows remain functional in non-persistence mode.
- Operational safety: No silent metadata loss across restart/resync.
- Timeline: Deliver in incremental phases with verifiable checkpoints.

---

## Requirements Analysis

### Must Have (ADR REQ-1 to REQ-5)
1. Persistence is opt-in via `SKILLSERVER_PERSISTENCE_DATA=true`.
2. Persistence path and DB file support mounted Docker/Kubernetes storage.
3. Git-imported catalog content is synchronized at startup and manual repo sync.
4. Git-imported content stays read-only while metadata is writable/durable.
5. Local/file-imported items keep content and metadata mutability.

### Should Have (ADR REQ-6 to REQ-7)
1. Keep current REST/MCP behavior backward-compatible.
2. Search/indexing reflects overlay-resolved effective metadata.

---

## Public Interface and Contract Changes

### Environment Variables
- `SKILLSERVER_PERSISTENCE_DATA` (`true|false`, default: `false`)
- `SKILLSERVER_PERSISTENCE_DIR` (required when persistence enabled)
- `SKILLSERVER_PERSISTENCE_DB_PATH` (optional absolute or derived path)

### Additive API Behavior
- `PATCH /api/catalog/:id/metadata` to update overlay metadata for any catalog item.
- `GET /api/catalog/:id/metadata` helper to return source + overlay + effective metadata.
- `GET /api/catalog` and `GET /api/catalog/search` include additive mutability fields:
  - `content_writable`
  - `metadata_writable`
  - Backward-compatible `read_only` retained for existing clients.

### Internal Contracts
- Data model tables:
  - `catalog_source_items`
  - `catalog_metadata_overlays`
  - `system_state`
- Effective record resolution:
  - `effective_name = COALESCE(display_name_override, source.name)`
  - `effective_description = COALESCE(description_override, source.description)`
- Search rebuild source:
  - Effective view, not raw filesystem snapshot.

---

## Domain Mapping

### Infrastructure / Runtime
- Parse persistence env config, validate mount/path writability, initialize DB lifecycle.
- Wire startup and Git resync callbacks to persistence sync path when enabled.

### Data Layer
- SQLite schema and migrations.
- Source snapshot repository (`catalog_source_items`).
- Overlay repository (`catalog_metadata_overlays`) with transactional writes.

### Service Layer
- Catalog sync/reconciliation orchestration:
  - startup full sync
  - repo-targeted sync on manual Git resync
  - soft-delete semantics for removed source items
- Effective overlay merge service and mutability contract enforcement.

### API Layer
- Catalog metadata overlay endpoints and validation.
- Catalog response DTO expansion with additive mutability fields.

### UI Layer
- Metadata editor UX for catalog items.
- Preserve content read-only behavior for Git-backed items.
- Allow metadata edits where `metadata_writable=true`.

### Quality + Documentation
- Regression matrix for restart/resync durability and write guards.
- README + operations runbook for Docker/Kubernetes persistence setup and rollback.

---

## Work Package Breakdown

### Phase 1: Persistence Foundation
- [x] [WP-001: Persistence Runtime Config and Startup Guardrails](./work-packages/WP-001-persistence-runtime-config-and-startup-guardrails.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-002: SQLite Bootstrap and Schema Migration Runner](./work-packages/WP-002-sqlite-bootstrap-and-schema-migration-runner.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-003: Catalog Source and Overlay Repository Layer](./work-packages/WP-003-catalog-source-and-overlay-repository-layer.md) ✅ COMPLETED (2026-03-04)

### Phase 2: Reconciliation and Effective Model
- [x] [WP-004: Catalog Sync Engine and Reconciliation Semantics](./work-packages/WP-004-catalog-sync-engine-and-reconciliation-semantics.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-005: Effective Catalog Projection and Mutability Contract](./work-packages/WP-005-effective-catalog-projection-and-mutability-contract.md) ✅ COMPLETED (2026-03-04)

### Phase 3: Interfaces and Runtime Wiring
- [x] [WP-006: Catalog Metadata API and Response Contract Extensions](./work-packages/WP-006-catalog-metadata-api-and-response-contract-extensions.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-007: Startup and Manual Git Resync Persistence Wiring](./work-packages/WP-007-startup-and-manual-git-resync-persistence-wiring.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-008: Web UI Metadata Overlay Editing and Mutability UX](./work-packages/WP-008-web-ui-metadata-overlay-editing-and-mutability-ux.md) ✅ COMPLETED (2026-03-04)

### Phase 4: Verification and Rollout
- [x] [WP-009: Persistence Integration and Regression Test Matrix](./work-packages/WP-009-persistence-integration-and-regression-test-matrix.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-010: Operations Docs, Rollout Checklist, and Rollback Guidance](./work-packages/WP-010-operations-docs-rollout-checklist-and-rollback-guidance.md) ✅ COMPLETED (2026-03-04)

---

## Dependency Graph

```text
WP-001 -> WP-002 -> WP-003 -> (WP-004 || WP-005)
WP-004 -> WP-006
WP-005 -> WP-006
WP-001 -> WP-007
WP-004 -> WP-007
WP-006 -> WP-008
(WP-006 || WP-007 || WP-008) -> WP-009 -> WP-010
```

### Critical Path
`WP-001 -> WP-002 -> WP-003 -> WP-004 -> WP-006 -> WP-008 -> WP-009 -> WP-010`

### Parallel Opportunities
- After WP-003, WP-004 and WP-005 can execute in parallel.
- WP-007 can run in parallel with WP-006 once WP-004 completes.
- WP-009 begins only after API, runtime wiring, and UI changes are merged.

---

## Timeline and Effort

| Phase | Work Packages | Estimated Hours |
|-------|---------------|-----------------|
| Persistence Foundation | WP-001, WP-002, WP-003 | 13 |
| Reconciliation and Effective Model | WP-004, WP-005 | 9 |
| Interfaces and Runtime Wiring | WP-006, WP-007, WP-008 | 12 |
| Verification and Rollout | WP-009, WP-010 | 9 |
| **Total** | **10 WPs** | **43** |

### Schedule Forecast
- Critical-path effort: 35 hours.
- Aggressive (parallelized): 7-8 working days at 6 productive hours/day.
- Realistic (code review + test iteration): 8-10 working days.
- Conservative with contingency buffer (x1.3): 56 hours (~9-10 working days).

---

## Test Strategy

### Unit Tests
- Persistence config parsing and startup guard behavior.
- Migration/version progression and schema idempotency.
- Repository CRUD and overlay merge precedence.
- Stable item identity and soft-delete reconciliation behavior.

### Integration Tests
- Startup sync persists both local and Git-derived catalog items.
- Manual `POST /api/git-repos/:id/sync` updates source snapshot without dropping overlays.
- Restart with same persistence mount preserves metadata overlays and effective responses.
- Search rebuild/index reflects overlay-resolved fields.

### API Tests
- `PATCH /api/catalog/:id/metadata` validates payload size/shape and persists updates.
- `GET /api/catalog/:id/metadata` returns source/overlay/effective fields correctly.
- Catalog list/search include `content_writable` and `metadata_writable`.
- Existing endpoints remain backward-compatible for non-persistence mode.

### UI/E2E Tests
- Git-backed item content edit remains disabled.
- Metadata edit is enabled for Git-backed items and persists across reload.
- Local/file-imported item content and metadata both editable.

---

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Persistence enabled without durable mount | Medium | High | Fail fast at startup with explicit remediation guidance and required env diagnostics. |
| Overlay loss from incorrect reconciliation logic | Low | High | Separate source/overlay tables, transactional updates, and regression tests for manual resync + restart. |
| Canonical ID drift breaks overlay linkage | Medium | Medium | Reuse existing deterministic catalog ID builders and add invariance tests. |
| Search/index divergence from effective data | Medium | Medium | Rebuild index from effective catalog projection after each source/overlay mutation. |
| SQLite lock/contention issues under concurrent updates | Medium | Medium | Use bounded transactions, retry strategy for busy errors, and targeted concurrency tests. |
| UI regressions from mutability contract changes | Medium | Medium | Feature-level UI integration tests plus backward-compatible response fields. |

---

## Assumptions and Defaults
1. Persistence remains disabled by default unless explicitly enabled.
2. Filesystem/Git remains canonical source for content bytes.
3. Metadata overlays are bounded in size and validated at API boundary.
4. Manual git sync endpoint continues to be the explicit operator-triggered resync path.
5. Existing `read_only` field remains for compatibility while new mutability fields are additive.

---

## Work Package Documents
- [Work Package Index](./work-packages/INDEX.md)
- [WP-001](./work-packages/WP-001-persistence-runtime-config-and-startup-guardrails.md)
- [WP-002](./work-packages/WP-002-sqlite-bootstrap-and-schema-migration-runner.md)
- [WP-003](./work-packages/WP-003-catalog-source-and-overlay-repository-layer.md)
- [WP-004](./work-packages/WP-004-catalog-sync-engine-and-reconciliation-semantics.md)
- [WP-005](./work-packages/WP-005-effective-catalog-projection-and-mutability-contract.md)
- [WP-006](./work-packages/WP-006-catalog-metadata-api-and-response-contract-extensions.md)
- [WP-007](./work-packages/WP-007-startup-and-manual-git-resync-persistence-wiring.md)
- [WP-008](./work-packages/WP-008-web-ui-metadata-overlay-editing-and-mutability-ux.md)
- [WP-009](./work-packages/WP-009-persistence-integration-and-regression-test-matrix.md)
- [WP-010](./work-packages/WP-010-operations-docs-rollout-checklist-and-rollback-guidance.md)

---

## Implementation Completion Summary

**Completion Date:** 2026-03-04  
**Status:** ✅ COMPLETED

### Overall Metrics

| Metric | Estimated | Actual | Variance |
|--------|-----------|--------|----------|
| Total Effort | 43 hours | N/A (not captured consistently in WP summaries) | N/A |
| Work Packages | 10 | 10 | 0 |
| Test Files Added/Updated | - | 15 | - |
| Coverage Reports | - | 81.6% average (3 WP summaries reported `-cover`) | - |
| Duration | - | 1 calendar day (2026-03-04 to 2026-03-04) | - |

### Work Package Summary

| WP ID | Domain | Estimated | Actual | Status | Completed |
|-------|--------|-----------|--------|--------|-----------|
| WP-001 | Infrastructure | 3h | N/A | ✅ | 2026-03-04 |
| WP-002 | Data Layer | 5h | N/A | ✅ | 2026-03-04 |
| WP-003 | Data Layer | 5h | N/A | ✅ | 2026-03-04 |
| WP-004 | Service Layer | 5h | N/A | ✅ | 2026-03-04 |
| WP-005 | Service Layer | 4h | N/A | ✅ | 2026-03-04 |
| WP-006 | API Layer | 4h | N/A | ✅ | 2026-03-04 |
| WP-007 | Infrastructure | 4h | N/A | ✅ | 2026-03-04 |
| WP-008 | UI Layer | 4h | N/A | ✅ | 2026-03-04 |
| WP-009 | Quality Engineering | 6h | N/A | ✅ | 2026-03-04 |
| WP-010 | Documentation | 3h | N/A | ✅ | 2026-03-04 |

### Key Achievements

- Delivered opt-in SQLite persistence with startup guardrails, migrations, and deterministic source/overlay repositories.
- Added full startup and manual Git resync persistence wiring with effective catalog search rebuild semantics.
- Shipped additive metadata overlay APIs plus UI metadata editing with mutability-aware UX and regression coverage.
- Completed rollout/rollback documentation for Docker and Kubernetes persistence operations.

### Common Challenges Encountered

1. **Backward compatibility while introducing new persistence contracts** (observed in WP-001, WP-005, WP-006, WP-007, WP-009, WP-010)
   - Resolution pattern: keep additive fields/endpoints and preserve legacy behavior when persistence mode is disabled.
2. **Deterministic reconciliation and indexing behavior** (observed in WP-003, WP-004, WP-005, WP-007, WP-009)
   - Resolution pattern: enforce stable ID/order semantics and rebuild search from effective projection state.
3. **Operational safety for rollout and rollback** (observed in WP-001, WP-002, WP-007, WP-010)
   - Resolution pattern: fail-fast startup validation, scoped sync controls, and explicit runbook/rollback guidance.

### Lessons Learned

**What Went Well:**
- Domain-isolated work packages enabled clear sequencing and parallel execution opportunities.
- Additive API/model contract strategy reduced regression risk for existing clients.
- Early automated test coverage per domain made integration validation straightforward in WP-009.

**What Could Be Improved:**
- Completion summaries should include standardized effort and variance tracking fields.
- Completion summaries should consistently include explicit lessons learned and technical debt sections.
- Work package metadata status fields should be updated uniformly as part of closeout.

**Actionable Recommendations for Future Plans:**
1. Add required `Estimated vs Actual` metrics fields to completion summary templates.
2. Enforce `Lessons Learned` and `Technical Debt` sections in every completion summary.
3. Add PR/commit references to each WP completion summary for traceability.

### Technical Debt Summary

| Priority | Count | Total Effort | Tickets Created |
|----------|-------|--------------|-----------------|
| High | 0 | 0h | None |
| Medium | 0 | 0h | None |
| Low | 0 | 0h | None |

**High Priority Debt Items:**
- None identified in work package completion summaries.

### Follow-Up Items

- [ ] Standardize completion summary template to require effort metrics, lessons learned, and technical debt tracking.
- [ ] Backfill `Status`, `Started_Date`, and `Completed_Date` metadata fields in WP-001 through WP-008 definition files for consistency.
- [ ] Convert runbook checks into a periodic operational validation checklist for persistence-enabled environments.

### References

**Work Package Completion Summaries:**
- [WP-001 Completion Summary](./work-packages/completion-summaries/WP-001-completion-summary.md)
- [WP-002 Completion Summary](./work-packages/completion-summaries/WP-002-completion-summary.md)
- [WP-003 Completion Summary](./work-packages/completion-summaries/WP-003-completion-summary.md)
- [WP-004 Completion Summary](./work-packages/completion-summaries/WP-004-completion-summary.md)
- [WP-005 Completion Summary](./work-packages/completion-summaries/WP-005-completion-summary.md)
- [WP-006 Completion Summary](./work-packages/completion-summaries/WP-006-completion-summary.md)
- [WP-007 Completion Summary](./work-packages/completion-summaries/WP-007-completion-summary.md)
- [WP-008 Completion Summary](./work-packages/completion-summaries/WP-008-completion-summary.md)
- [WP-009 Completion Summary](./work-packages/completion-summaries/WP-009-completion-summary.md)
- [WP-010 Completion Summary](./work-packages/completion-summaries/WP-010-completion-summary.md)

---

## Next Steps
1. Share the completion report with stakeholders and link it from the ADR/work package index.
2. Schedule follow-up process improvements for completion-summary quality and metadata hygiene.
3. Track persistence operational checks as part of release readiness for future iterations.
