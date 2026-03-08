# Implementation Plan: ADR-007 Rule Catalog Objects and MCP Project Materialization

**Date Created:** 2026-03-08
**Project Owner:** @jeff
**Target Completion:** 2026-03-19
**Actual Completion:** 2026-03-08
**Status:** COMPLETED
**Source ADR:** [ADR-007: Rule Catalog Objects and MCP Project Materialization](../../adrs/007-rule-catalog-objects-and-mcp-project-materialization.md)

---

## Project Overview

### Goal
Extend the unified catalog so `rule` files become first-class catalog items and add one shared export/materialization service that powers GUI, REST, and MCP workflows for skills, prompts, and rules.

### Success Criteria
- [x] `rule` is a first-class catalog classifier everywhere catalog items are discovered, indexed, filtered, persisted, and returned. ✅
- [x] The broken GUI skill export path is replaced by a shared service seam without regressing current skill download behavior. ✅
- [x] REST and MCP both support batch export/materialization with dry-run planning and bounded destination-root validation. ✅
- [x] Project-rule files such as `AGENTS.md`, `RULES.md`, and `CLAUDE.md` can materialize to intended project-root targets when explicitly allowed. ✅
- [x] Imported Git content remains read-only at source while exported/materialized copies are explicit user-owned outputs. ✅
- [x] Regression coverage proves path safety, conflict-policy behavior, and backward compatibility for existing skill flows. ✅

### Scope

**In Scope:**
- Shared export/materialization domain service reused by GUI, REST, and MCP.
- First-class `rule` classifier plus configurable rule discovery and install metadata parsing.
- Persistence migration to widen classifier support to `rule`.
- Safe project-folder writes bounded by configured allowed destination roots.
- REST and MCP additive export/materialization contracts with runtime gating.
- Web UI updates for classifier-aware export/materialize actions.
- Rollout, rollback, and release guidance for staged enablement.

**Out of Scope:**
- New external package registry, blob store, or managed database.
- Source-side editing of imported Git-backed rule/prompt/skill content.
- New authentication or authorization model beyond runtime capability gates.
- Distributed multi-node coordination for concurrent materialization writes.

### Constraints
- Technical: Must build on current filesystem/Git discovery, Bleve indexing, SQLite persistence, Echo REST handlers, and MCP server architecture.
- Compatibility: Existing `/api/skills`, resource APIs, and legacy skill export behavior must remain stable during rollout.
- Safety: Materialization must reject absolute paths, traversal, and any write outside configured destination roots.
- Delivery: The first milestone should recover GUI export and establish the shared service seam before broader classifier/materialization work lands.

---

## Requirements Analysis

### Must Have (ADR REQ-1 to REQ-6)
1. Add `rule` as a first-class classifier alongside `skill` and `prompt`.
2. Make rules searchable, filterable, and taxonomized across REST and MCP catalog surfaces.
3. Provide an MCP workflow for materializing selected items into a local project folder.
4. Rebuild GUI export on the same shared service used by backend callers.
5. Enforce write safety: no absolute paths, no traversal, no writes outside allowed roots.
6. Preserve current source mutability rules for imported Git content.

### Should Have (ADR REQ-7 to REQ-9)
1. Support install metadata so rules can land at project-root targets such as `AGENTS.md`.
2. Support batching multiple catalog items per export/materialization request.
3. Preserve legacy skill export/import behavior during rollout.

### Nice to Have (ADR REQ-10)
1. Support dry-run planning so callers can inspect resolved target paths before writes occur.

---

## Public Interface and Contract Changes

### Runtime Configuration
- Extend catalog runtime config with:
  - `SKILLSERVER_CATALOG_ENABLE_RULES`
  - `SKILLSERVER_CATALOG_RULE_DIRS`
  - `SKILLSERVER_CATALOG_RULE_FILENAMES`
- Extend MCP/runtime gating with:
  - `SKILLSERVER_MCP_ENABLE_MATERIALIZATION`
  - `SKILLSERVER_MCP_ALLOWED_DESTINATION_ROOTS`
- Expose additive runtime capabilities so the UI can hide or disable materialization affordances when unavailable.

### Catalog Item Contract
- Add `classifier=rule` to the catalog item model, search index, persistence rows, REST DTOs, and MCP responses.
- Add optional frontmatter install metadata for file-backed items:

```yaml
materialize:
  target_path: AGENTS.md
  conflict_policy: overwrite
```

- Target resolution order:
  1. `materialize.target_path`
  2. Preserved project-rule basenames at project root
  3. Classifier defaults under destination (`skills/`, `prompts/`, `rules/`)

### REST API
- `POST /api/catalog/export`
  - Accepts `item_ids`, export format/options, and optional `dry_run`
  - Returns archive payload metadata or a dry-run manifest
- `POST /api/catalog/materialize`
  - Accepts `item_ids`, `destination_dir`, `conflict_policy`, and optional `dry_run`
  - Returns resolved target paths and per-item results
- `GET /api/skills/export/*`
  - Preserved as a compatibility wrapper over the shared export service

### MCP
- Additive tools:
  - `export_catalog_items`
  - `materialize_catalog_items`
  - `plan_catalog_materialization` or equivalent `dry_run` flow
- Materialization tools register only when runtime gating explicitly enables them.

---

## Domain Mapping

### Service Layer
- Shared export service that replaces direct route-to-archive coupling.
- Shared materialization service for batching, target planning, dry-run manifests, conflict handling, and bounded writes.

### Domain Layer
- `rule` classifier contract, rule detection rules, deterministic IDs, install metadata parsing, and target-path resolution semantics.
- Catalog builder, search index, and sync/effective logic updated so rules behave like first-class catalog items.

### Data Layer
- SQLite migration and persisted classifier validation widened from `skill|prompt` to `skill|prompt|rule`.

### API Layer
- Additive export/materialization endpoints plus legacy route delegation to the shared service.

### MCP Layer
- Export/materialization tools and tool-registration capability gates.

### UI Layer
- Classifier-aware export/materialize actions, dry-run preview, and capability-sensitive controls.

### Quality + Documentation
- Cross-surface regression matrix for safety and compatibility.
- Rollout/rollback guidance, README updates, and release notes.

---

## Milestones

### Milestone 1: Shared Export Seam Recovery
- [x] [WP-001: Shared Catalog Export Service](./work-packages/WP-001-shared-catalog-export-service.md) ✅ COMPLETED (2026-03-08)
- [x] [WP-002: Export REST Endpoint and Legacy Route Delegation](./work-packages/WP-002-export-rest-route-delegation.md) ✅ COMPLETED (2026-03-08)

**Outcome:** existing GUI skill export is recovered through a shared backend service seam without waiting for rule catalog work.

### Milestone 2: Rule Catalog Foundation
- [x] [WP-003: Rule Classifier and Install Metadata](./work-packages/WP-003-rule-classifier-and-install-metadata.md) ✅ COMPLETED (2026-03-08)
- [x] [WP-004: Runtime Flags and Capability Gates](./work-packages/WP-004-runtime-flags-and-capability-gates.md) ✅ COMPLETED (2026-03-08)
- [x] [WP-005: Rule Classifier Persistence Migration](./work-packages/WP-005-rule-classifier-persistence-migration.md) ✅ COMPLETED (2026-03-08)
- [x] [WP-006: Rule Catalog Discovery, Search, and Sync](./work-packages/WP-006-rule-catalog-discovery-search-and-sync.md) ✅ COMPLETED (2026-03-08)

**Outcome:** rules become indexed, persisted, filterable catalog items with explicit runtime controls.

### Milestone 3: Materialization Surfaces
- [x] [WP-007: Materialization Planner and Safe Writes](./work-packages/WP-007-materialization-planner-and-safe-writes.md) ✅ COMPLETED (2026-03-08)
- [x] [WP-008: Catalog Materialization REST Endpoints](./work-packages/WP-008-catalog-materialization-rest-endpoints.md) ✅ COMPLETED (2026-03-08)
- [x] [WP-009: MCP Export and Materialization Tools](./work-packages/WP-009-mcp-export-materialization-tools.md) ✅ COMPLETED (2026-03-08)
- [x] [WP-010: UI Export and Materialization UX](./work-packages/WP-010-ui-export-materialization-ux.md) ✅ COMPLETED (2026-03-08)

**Outcome:** REST, MCP, and UI all use the same shared planning/writing semantics for skills, prompts, and rules.

### Milestone 4: Verification and Rollout
- [x] [WP-011: Integration, Safety, and Regression Matrix](./work-packages/WP-011-integration-safety-regression-matrix.md) ✅ COMPLETED (2026-03-08)
- [x] [WP-012: Rollout, Rollback, and Release Guidance](./work-packages/WP-012-rollout-rollback-release-guidance.md) ✅ COMPLETED (2026-03-08)

**Outcome:** rollout is gated by verified safety tests and documented rollback procedures.

---

## Dependency Graph

```text
WP-001 -> WP-002

WP-003 -> (WP-004 || WP-005)
(WP-004 || WP-005) -> WP-006

(WP-001 || WP-003 || WP-004 || WP-006) -> WP-007

(WP-002 || WP-006 || WP-007) -> WP-008
(WP-004 || WP-006 || WP-007) -> WP-009

(WP-006 || WP-008) -> WP-010

(WP-005 || WP-008 || WP-009 || WP-010) -> WP-011 -> WP-012
```

### Critical Path
`WP-003 -> WP-004 -> WP-006 -> WP-007 -> WP-008 -> WP-010 -> WP-011 -> WP-012`

### Parallel Opportunities
- WP-001 and WP-003 can start immediately.
- WP-004 and WP-005 can run in parallel after WP-003.
- WP-008 and WP-009 can run in parallel once WP-007 and rule-catalog prerequisites are complete.
- WP-012 should wait for WP-011 so rollout docs reflect verified behavior rather than intended behavior.

---

## Timeline and Effort

| Phase | Work Packages | Estimated Hours |
|-------|---------------|-----------------|
| Milestone 1: Shared Export Seam Recovery | WP-001, WP-002 | 7 |
| Milestone 2: Rule Catalog Foundation | WP-003, WP-004, WP-005, WP-006 | 14 |
| Milestone 3: Materialization Surfaces | WP-007, WP-008, WP-009, WP-010 | 17 |
| Milestone 4: Verification and Rollout | WP-011, WP-012 | 8 |
| **Total** | **12 WPs** | **46** |

### Schedule Forecast
- Milestone 1: 1-2 working days and should be releasable independently.
- Critical-path effort: 32 hours.
- Aggressive parallelized execution: 6 working days at 6 productive hours/day.
- Realistic execution with review and rework: 7-8 working days.
- Conservative estimate with contingency buffer (x1.25 on critical path): 8-9 working days.

---

## Test Strategy

### Domain and Persistence
- Classifier rule coverage for direct and imported rule candidates.
- Frontmatter install-metadata parsing and invalid-path rejection.
- Migration tests proving classifier widening preserves existing rows.
- Sync/index tests proving rules remain stable across rebuilds and persistence sync cycles.

### REST API
- Export/materialize request validation and disabled-capability behavior.
- Legacy skill export route delegation and backward compatibility.
- Dry-run manifest behavior and mixed-item batch handling.

### MCP
- Tool registration gating for materialization writes.
- Export/materialize dry-run behavior and safety failures.
- Catalog list/search parity for `classifier=rule`.

### UI
- Existing skill export remains functional.
- Prompt/rule export and materialize controls appear only when supported.
- Dry-run preview and per-file outcome rendering behave correctly.

### Safety and Regression
- No writes outside allowed roots.
- No writes for absolute paths or traversal inputs.
- Conflict-policy behavior is deterministic (`error`, `overwrite`, `skip`).
- Existing skill CRUD/resource workflows remain unchanged.

---

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Rule detection is too broad and classifies arbitrary markdown as rules | Medium | Medium | Require explicit rule directories or filename allowlist plus markdown/text validation. |
| Materialization writes outside intended project roots | Low | High | Centralize path normalization, require absolute allowed roots, and add regression tests for traversal and symlink escapes. |
| SQLite migration for classifier widening loses data or breaks upgrades | Medium | High | Add a dedicated schema migration with migration tests from prior versions and pre/post row-count assertions. |
| GUI, REST, and MCP drift on target-path semantics | Medium | Medium | Keep target planning in one shared service and make all adapters thin. |
| Conflict-policy defaults surprise users and overwrite local files | Medium | Medium | Default to `error`, require explicit override, and surface dry-run previews before writes. |
| Batch export/materialization degrades performance on larger repos | Low | Medium | Reuse batch-oriented service APIs, keep manifests bounded, and measure with regression tests for representative item counts. |

---

## Assumptions and Defaults
1. Materialization remains disabled by default until rollout guidance is followed.
2. Allowed destination roots are absolute paths and must be configured explicitly when materialization is enabled.
3. `conflict_policy=error` is the default; `overwrite` and `skip` are additive options.
4. The legacy skill export route continues to return archive-compatible downloads even after the shared service is introduced.
5. Imported Git content remains immutable at source; only exported/materialized copies become user-owned files.
6. Auditability is satisfied in the first iteration via structured logs and response manifests, not via a new persistence table.

---

## Work Package Documents
- [Work Package Index](./work-packages/INDEX.md)
- [WP-001](./work-packages/WP-001-shared-catalog-export-service.md)
- [WP-002](./work-packages/WP-002-export-rest-route-delegation.md)
- [WP-003](./work-packages/WP-003-rule-classifier-and-install-metadata.md)
- [WP-004](./work-packages/WP-004-runtime-flags-and-capability-gates.md)
- [WP-005](./work-packages/WP-005-rule-classifier-persistence-migration.md)
- [WP-006](./work-packages/WP-006-rule-catalog-discovery-search-and-sync.md)
- [WP-007](./work-packages/WP-007-materialization-planner-and-safe-writes.md)
- [WP-008](./work-packages/WP-008-catalog-materialization-rest-endpoints.md)
- [WP-009](./work-packages/WP-009-mcp-export-materialization-tools.md)
- [WP-010](./work-packages/WP-010-ui-export-materialization-ux.md)
- [WP-011](./work-packages/WP-011-integration-safety-regression-matrix.md)
- [WP-012](./work-packages/WP-012-rollout-rollback-release-guidance.md)

## Rollout Artifacts
- [ADR-007 rollout/rollback runbook](../../operations/rule-catalog-materialization-rollout-rollback.md)
- [ADR-007 release notes](../../releases/2026-03-08-adr-007-rule-catalog-materialization-release-notes.md)
- [WP-011 completion summary](./work-packages/completion-summaries/WP-011-completion-summary.md)
- [WP-012 completion summary](./work-packages/completion-summaries/WP-012-completion-summary.md)
- [Implementation completion report](./rule-catalog-objects-and-mcp-project-materialization-completion-report.md)

## Implementation Completion Summary

**Completion Date:** 2026-03-08  
**Status:** ✅ COMPLETED

### Overall Metrics

| Metric | Estimated | Actual | Variance |
|--------|-----------|--------|----------|
| Total Effort | 46 hours | N/A (not captured in WP completion summaries) | N/A |
| Work Packages | 12 | 12 | 0 |
| Test File Touches | - | 20 entries across WP file-change logs | - |
| Coverage (reported packages only) | - | 73.9% average (`pkg/persistence` 74.9%, `pkg/domain` 72.8%) | - |
| Unique Files Changed | - | 57 | - |
| Duration | 8-9 working days forecast | 1 calendar day (all WPs completed on 2026-03-08) | Ahead of forecast |

### Work Package Summary

| WP ID | Domain | Estimated | Actual | Status | Completed |
|-------|--------|-----------|--------|--------|-----------|
| WP-001 | Service Layer | 4h | N/A | ✅ | 2026-03-08 |
| WP-002 | API Layer | 3h | N/A | ✅ | 2026-03-08 |
| WP-003 | Domain Layer | 4h | N/A | ✅ | 2026-03-08 |
| WP-004 | Infrastructure | 3h | N/A | ✅ | 2026-03-08 |
| WP-005 | Data Layer | 3h | N/A | ✅ | 2026-03-08 |
| WP-006 | Domain Layer | 4h | N/A | ✅ | 2026-03-08 |
| WP-007 | Service Layer | 5h | N/A | ✅ | 2026-03-08 |
| WP-008 | API Layer | 4h | N/A | ✅ | 2026-03-08 |
| WP-009 | MCP Layer | 4h | N/A | ✅ | 2026-03-08 |
| WP-010 | UI Layer | 4h | N/A | ✅ | 2026-03-08 |
| WP-011 | Quality Engineering | 5h | N/A | ✅ | 2026-03-08 |
| WP-012 | Documentation | 3h | N/A | ✅ | 2026-03-08 |

### Key Achievements

- Restored and generalized catalog export via a shared service seam used by legacy and additive API flows.
- Added first-class `rule` classifier behavior across domain contracts, discovery/indexing, persistence migration, REST, MCP, and UI.
- Delivered capability-gated materialization planning and writes with destination-root safety checks across REST and MCP.
- Added regression coverage plus rollout/rollback and release documentation tied to ADR-007.

### Common Challenges Encountered

1. **Layered rollout sequencing across domains** (recurring)
   - Description: Work had to be deliberately staged so adapters (REST/MCP/UI) depended on stabilized shared domain services.
   - Resolution pattern: Kept service seams additive and used compatibility wrappers to preserve existing behavior.

2. **Safety hardening for file materialization** (recurring)
   - Description: Materialization introduced path traversal/outside-root risk and conflict-policy behavior risk.
   - Resolution pattern: Centralized destination validation, defaulted to safe conflict handling, and enforced regression tests for unsafe inputs.

3. **Backward compatibility while expanding classifier scope** (recurring)
   - Description: `rule` support needed to land without regressing existing skill/prompt APIs and UI workflows.
   - Resolution pattern: Added classifier behavior incrementally with targeted tests and capability gating.

### Lessons Learned

**What Went Well:**
- Shared domain services reduced adapter drift across REST, MCP, and UI surfaces.
- Runtime capability flags enabled safe incremental rollout for write-capable features.
- Dedicated verification and rollout work packages improved release readiness evidence.

**What Could Be Improved:**
- Completion summaries should record actual effort per WP to enable variance analysis.
- Completion summaries should include standardized lessons-learned and technical-debt sections.
- Cross-WP metrics (tests added, LOC deltas) should be captured in machine-readable form.

**Actionable Recommendations for Future Plans:**
1. Require numeric `Actual_Effort` and `Time_Spent` fields in every completion summary.
2. Standardize completion-summary sections for challenges, lessons learned, and technical debt.
3. Add an automated aggregator script to compute per-plan metrics from WP metadata and git diff stats.

### Technical Debt Summary

| Priority | Count | Total Effort | Tickets Created |
|----------|-------|--------------|-----------------|
| High | 0 documented | N/A | N/A |
| Medium | 0 documented | N/A | N/A |
| Low | 0 documented | N/A | N/A |

**High Priority Debt Items:**
- None explicitly recorded in WP completion summaries.

### Follow-Up Items

- [ ] Add a completion-summary template guardrail/check that enforces actual effort and lessons sections.
- [ ] Add a reporting helper to aggregate test-count and LOC metrics automatically at plan-close time.
- [ ] Consider capturing PR/commit references per WP for traceability in future implementation plans.

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
- [WP-011 Completion Summary](./work-packages/completion-summaries/WP-011-completion-summary.md)
- [WP-012 Completion Summary](./work-packages/completion-summaries/WP-012-completion-summary.md)

---

## Next Steps
1. Share the completion report with stakeholders and archive this plan as delivered.
2. Track the follow-up reporting/template improvements in a separate maintenance ticket.
3. Use the ADR-007 rollout runbook as the operational source of truth for staged enablement.
