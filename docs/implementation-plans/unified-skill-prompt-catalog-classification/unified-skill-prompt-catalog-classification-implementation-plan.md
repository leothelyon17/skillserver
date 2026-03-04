# Implementation Plan: ADR-003 Unified Skill/Prompt Catalog Classification

**Date Created:** 2026-03-04
**Project Owner:** @jeff
**Target Completion:** 2026-03-12
**Actual Completion:** 2026-03-04
**Status:** COMPLETED
**Source ADR:** [ADR-003: Unified Skill/Prompt Catalog Classification for Git Imports](../../adrs/003-unified-skill-prompt-catalog-classification.md)

---

## Project Overview

### Goal
Implement a unified catalog model where both skills and prompt markdown files are first-class searchable items, each with an explicit classifier (`skill` or `prompt`), while preserving existing skill CRUD behavior.

### Success Criteria
- [x] Prompt markdown files under `agent/`, `agents/`, `prompt/`, `prompts/` are indexed as top-level catalog items. ✅
- [x] `GET /api/catalog` and `GET /api/catalog/search` return mixed skill/prompt results with classifier metadata. ✅
- [x] GUI grid/search uses catalog APIs and shows a type badge (`skill` or `prompt`) on each tile. ✅
- [x] Search supports index-backed classifier filtering (no path-scan-only filtering). ✅
- [x] Existing `/api/skills` list/search CRUD behavior remains backward compatible. ✅
- [x] Regression tests cover domain, API, UI, and optional MCP parity behavior. ✅

### Scope

**In Scope:**
- Unified catalog item contract with explicit classifier.
- Prompt candidate extraction from direct and imported resources.
- Search index schema/query updates for classifier-aware filtering.
- Additive catalog API endpoints and GUI catalog adoption.
- Config-driven prompt directory allowlist for classification.
- Optional MCP parity for catalog list/search classifier filters.
- Test expansion and rollout/rollback documentation.

**Out of Scope:**
- External search/database services.
- Breaking changes to existing `/api/skills` or skill resource CRUD contracts.
- New authN/authZ model.

### Constraints
- Technical: Must stay within current filesystem + Bleve architecture.
- Compatibility: Existing `/api/skills` and read/write skill workflows remain stable.
- Security: Preserve imported-resource path boundary and read-only semantics.
- Timeline: One delivery cycle (~1 week implementation plus test/docs hardening).

---

## Requirements Analysis

### Must Have (ADR REQ-1 to REQ-4)
1. Classify prompt markdown files from allowed prompt directory segments (`agent`, `agents`, `prompt`, `prompts`) including imported virtual resources.
2. Expose prompt files as first-class catalog entries in main UI/search flow.
3. Add explicit classifier field on all catalog items.
4. Make classifier filterable/queryable in Bleve-backed search.

### Should Have (ADR REQ-5 to REQ-6)
1. Tile-level classifier badge in UI.
2. Preserve existing skill-only endpoint behavior and contracts.

### Nice to Have (ADR REQ-7)
1. MCP parity for catalog list/search with classifier filtering.

---

## Domain Mapping

### Domain Layer (`pkg/domain`)
- Add catalog item model and classifier enum.
- Build deterministic IDs for skill and prompt catalog items.
- Generate prompt catalog items from direct/imported resources with dedupe by canonical key.
- Extend search indexing/querying for classifier-aware catalog search.

### API Layer (`pkg/web`)
- Add `/api/catalog` and `/api/catalog/search` endpoints.
- Validate classifier query params and return additive catalog contracts.
- Keep existing `/api/skills` routes unchanged.

### UI Layer (`pkg/web/ui`)
- Replace skill-only list/search data source with catalog endpoints.
- Show `skill`/`prompt` badges and preserve read-only affordances.
- Handle mixed-item click behavior without breaking skill edit workflows.

### Runtime/Config Layer (`cmd/skillserver`)
- Add env/flag configuration for prompt indexing toggle and prompt directory allowlist.
- Wire runtime config into catalog builder/classification behavior.

### MCP Layer (`pkg/mcp`) - Optional/Nice-to-Have
- Add additive catalog tools (`list_catalog`, `search_catalog`) with classifier filters.
- Keep existing `list_skills`/`search_skills` tool contracts unchanged.

### Quality + Documentation
- Add unit/integration/regression tests for classifier correctness and compatibility.
- Update README and operations runbook for rollout and rollback controls.

---

## Work Package Breakdown

### Phase 1: Catalog Foundation
- [x] [WP-001: Catalog Contract and Classifier Rules](./work-packages/WP-001-catalog-contract-and-classifier-rules.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-002: Bleve Catalog Index and Classifier Filtering](./work-packages/WP-002-bleve-catalog-index-and-classifier-filtering.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-003: Manager Catalog Builder and Rebuild Integration](./work-packages/WP-003-manager-catalog-builder-and-rebuild-integration.md) ✅ COMPLETED (2026-03-04)

### Phase 2: Interfaces and Configuration
- [x] [WP-004: Catalog REST Endpoints and API Contracts](./work-packages/WP-004-catalog-rest-endpoints-and-api-contracts.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-005: Web UI Unified Catalog Rendering and Badges](./work-packages/WP-005-web-ui-unified-catalog-rendering-and-badges.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-006: Runtime Config for Prompt Catalog Detection](./work-packages/WP-006-runtime-config-for-prompt-catalog-detection.md) ✅ COMPLETED (2026-03-04)

### Phase 3: Parity, Validation, and Rollout
- [x] [WP-007: MCP Catalog Parity Tools (Optional)](./work-packages/WP-007-mcp-catalog-parity-tools.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-008: Integration and Regression Test Matrix](./work-packages/WP-008-integration-and-regression-test-matrix.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-009: Documentation, Rollout, and Rollback Guidance](./work-packages/WP-009-documentation-rollout-and-rollback-guidance.md) ✅ COMPLETED (2026-03-04)

---

## Dependency Graph

```text
WP-001 -> WP-002 -> WP-003 -> (WP-004 || WP-006 || WP-007)
WP-004 -> WP-005
(WP-005 || WP-006 || WP-007) -> WP-008 -> WP-009
```

### Critical Path
`WP-001 -> WP-002 -> WP-003 -> WP-004 -> WP-005 -> WP-008 -> WP-009`

### Parallel Opportunities
- After WP-003, WP-004, WP-006, and WP-007 can run in parallel.
- WP-008 starts once interface/config changes are merged.
- WP-009 starts after validation signals are green.

---

## Timeline and Effort

| Phase | Work Packages | Estimated Hours |
|-------|---------------|-----------------|
| Catalog Foundation | WP-001, WP-002, WP-003 | 14 |
| Interfaces and Configuration | WP-004, WP-005, WP-006 | 11 |
| Parity, Validation, Rollout | WP-007, WP-008, WP-009 | 12 |
| **Total** | **9 WPs** | **37** |

### Schedule Forecast
- Critical-path effort: 31 hours.
- Aggressive (parallelized): ~5 working days at 6 productive hours/day.
- Realistic (review + rework): 6-7 working days.
- Conservative with contingency buffer (x1.4): 8-9 working days.

---

## Test Strategy

### Domain Tests
- Classifier rule coverage for direct and imported paths.
- Prompt directory allowlist parsing and markdown-only enforcement.
- Stable ID generation and dedupe correctness.
- Backward-compatibility behavior for skill-only flows.

### API Tests
- `GET /api/catalog` contract validation.
- `GET /api/catalog/search?q=...&classifier=...` filter behavior.
- Invalid classifier input handling.
- Existing `/api/skills` routes unchanged.

### UI Tests
- Mixed catalog tile rendering.
- Classifier badge rendering and search result counts.
- Skill edit flow remains intact.
- Prompt item behavior remains read-only where required.

### MCP Tests (if WP-007 accepted)
- `list_catalog` and `search_catalog` outputs include classifier and prompt hits.
- Existing `list_skills`/`search_skills` tools unaffected.

---

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| False-positive prompt classification from path naming edge cases | Medium | Medium | Enforce markdown-only + normalized segment allowlist; add edge-case tests. |
| Duplicate prompt catalog documents from direct/import overlap | Medium | Low | Dedupe using stable `(skill_id, canonical_target_path)` keys and deterministic IDs. |
| Index growth affects rebuild time on large repos | Medium | Medium | Batch writes, size caps in tests, and benchmark-driven tuning. |
| UI regressions from mixed item actions | Medium | Medium | Feature-gated switch to `/api/catalog` + regression tests for skill editing workflow. |
| Client dependency on skill-only API responses | Low | Medium | Keep `/api/skills` unchanged and add catalog APIs as additive endpoints. |
| Config drift between runtime defaults and ADR expectations | Low | Medium | Add explicit env/flag defaults and startup logging of effective prompt dirs. |

---

## Assumptions and Defaults
1. Default prompt directory allowlist is `agent,agents,prompt,prompts`.
2. Prompt catalog indexing is enabled by default but can be toggled off via runtime config.
3. `SKILL.md` is always classified as `skill` and never `prompt`.
4. Existing import-discovery boundary checks remain authoritative for imported resources.
5. MCP catalog parity is implemented only if scope/time allows in-cycle (WP-007).

---

## Next Steps
1. Execute rollout smoke checks in `docs/operations/unified-catalog-rollout-rollback.md` for the target environment.
2. Attach canary output evidence to release records as part of release governance.
3. Track benchmark trend deltas for prompt-heavy repositories in future releases.
4. Triage medium-priority test-depth technical debt identified in completion summaries.

---

## Implementation Completion Summary

**Completion Date:** 2026-03-04
**Status:** ✅ COMPLETED

### Overall Metrics

| Metric | Estimated | Actual | Variance |
|--------|-----------|--------|----------|
| Total Effort | 37 hours | 17 hours | -20 hours (-54.1%) |
| Work Packages | 9 | 9 | 0 |
| Test Files Added | - | 7 new (`*_test.go`, `*.spec.ts`) | - |
| Coverage Snapshot | - | `pkg/domain` 80.9% (WP-002) | - |
| Total LOC (workspace delta estimate) | - | 5,310 lines changed | - |
| Duration | - | 1 day | - |

### Work Package Summary

| WP ID | Domain | Estimated | Actual | Status | Completed |
|-------|--------|-----------|--------|--------|-----------|
| WP-001 | Domain Layer | 4h | 2.0h | ✅ | 2026-03-04 |
| WP-002 | Domain Layer | 5h | 2.5h | ✅ | 2026-03-04 |
| WP-003 | Domain Layer | 5h | 2.5h | ✅ | 2026-03-04 |
| WP-004 | API Layer | 4h | 1.5h | ✅ | 2026-03-04 |
| WP-005 | UI Layer | 4h | 2.0h | ✅ | 2026-03-04 |
| WP-006 | Infrastructure | 3h | 1.5h | ✅ | 2026-03-04 |
| WP-007 | MCP Layer | 3h | 1.5h | ✅ | 2026-03-04 |
| WP-008 | Quality Engineering | 6h | 2.0h | ✅ | 2026-03-04 |
| WP-009 | Documentation | 3h | 1.5h | ✅ | 2026-03-04 |

### Key Achievements

- Delivered additive unified catalog behavior across domain, API, UI, runtime config, and MCP interfaces without breaking `/api/skills` compatibility.
- Landed classifier-aware indexing/search and deterministic catalog IDs for mixed skill/prompt catalogs.
- Completed rollout artifacts: runbook, release notes, verification checklists, and per-WP completion summaries.

### Common Challenges Encountered

1. **Test depth and coverage baseline variance** (occurred in 2 WPs)
   - Description: Existing legacy code paths and error branches were not uniformly deep-covered.
   - Resolution pattern: Focused new tests on ADR-critical paths plus explicit follow-up notes for deeper legacy coverage.

2. **UI regression assertion flakiness** (occurred in 1 WP)
   - Description: Toast-notification checks were brittle in Playwright flows.
   - Resolution pattern: Shifted assertions to network success and stable UI state transitions.

3. **Prompt-heavy rebuild resource profile visibility** (occurred in 1 WP)
   - Description: Benchmarking surfaced meaningful memory/alloc footprints for large mixed catalogs.
   - Resolution pattern: Captured benchmark baseline and documented future trend tracking.

### Lessons Learned

**What Went Well:**
- Sequencing WP-001 to WP-003 first reduced downstream risk for API/UI/MCP integration.
- Additive contract strategy (`/api/catalog`, `list_catalog`, `search_catalog`) preserved backward compatibility cleanly.
- Verification artifacts (WP-005/WP-008 checklists + benchmark) made rollout confidence explicit.

**What Could Be Improved:**
- Normalize completion summary structure with explicit lessons/debt sections across all WPs.
- Add deeper error-path tests in the same cycle rather than deferring.
- Capture per-WP quantitative test counts in a consistent format.

**Actionable Recommendations for Future Plans:**
1. Include mandatory quantitative metrics fields in completion-summary templates.
2. Require one benchmark/trend artifact for index-affecting work by default.
3. Attach rollout canary outputs directly to release records at completion time.

### Technical Debt Summary

| Priority | Count | Total Effort | Tickets Created |
|----------|-------|--------------|-----------------|
| High | 0 | 0h | none |
| Medium | 2 | ~4h | none |
| Low | 1 | ~2h | none |

**High Priority Debt Items:**
- None identified in WP completion summaries.

### Follow-Up Items

- [ ] Add deeper legacy-domain/error-branch coverage for non-ADR critical paths.
- [ ] Expand prompt-heavy benchmark fixture size over time and track regression deltas.
- [ ] Attach canary rollout evidence to release governance artifacts.
- [ ] Reconcile completion summary template consistency across future implementation plans.

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

---

## Work Package Documents
- [Work Package Index](./work-packages/INDEX.md)
- [WP-001](./work-packages/WP-001-catalog-contract-and-classifier-rules.md)
- [WP-002](./work-packages/WP-002-bleve-catalog-index-and-classifier-filtering.md)
- [WP-003](./work-packages/WP-003-manager-catalog-builder-and-rebuild-integration.md)
- [WP-004](./work-packages/WP-004-catalog-rest-endpoints-and-api-contracts.md)
- [WP-005](./work-packages/WP-005-web-ui-unified-catalog-rendering-and-badges.md)
- [WP-006](./work-packages/WP-006-runtime-config-for-prompt-catalog-detection.md)
- [WP-007](./work-packages/WP-007-mcp-catalog-parity-tools.md)
- [WP-008](./work-packages/WP-008-integration-and-regression-test-matrix.md)
- [WP-009](./work-packages/WP-009-documentation-rollout-and-rollback-guidance.md)

## Rollout and Release Artifacts (WP-009)
- [Unified Catalog Rollout Runbook](../../operations/unified-catalog-rollout-rollback.md)
- [ADR-003 Release Notes](../../releases/2026-03-04-adr-003-unified-catalog-release-notes.md)
- [WP-009 Completion Summary](./work-packages/completion-summaries/WP-009-completion-summary.md)
- [Implementation Completion Report](./unified-skill-prompt-catalog-classification-completion-report.md)
