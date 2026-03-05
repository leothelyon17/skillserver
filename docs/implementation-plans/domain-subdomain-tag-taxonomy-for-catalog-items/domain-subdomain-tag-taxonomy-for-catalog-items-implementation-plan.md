# Implementation Plan: ADR-005 Domain/Subdomain/Tag Taxonomy for Catalog Items

**Date Created:** 2026-03-04
**Project Owner:** @jeff
**Target Completion:** 2026-03-21
**Actual Completion:** 2026-03-05
**Status:** COMPLETED ✅
**Source ADR:** [ADR-005: Domain/Subdomain/Tag Taxonomy for Catalog Items](../../adrs/005-domain-subdomain-tag-taxonomy-for-catalog-items.md)

---

## Project Overview

### Goal
Implement a first-class taxonomy model for catalog items (skills and prompts) with persistent domain, subdomain, and tag objects, assignment APIs, taxonomy-aware search filters, MCP parity, and UI taxonomy management.

### Success Criteria
- [x] Every catalog item supports `primary_domain`, `secondary_domain`, and zero-or-more taxonomy tags. ✅
- [x] Domains, subdomains, and tags are persisted as first-class objects with unique keys. ✅
- [x] REST and MCP list/search flows support deterministic taxonomy filtering. ✅
- [x] MCP taxonomy write tools are gated by runtime config and disabled by default. ✅
- [x] UI provides global taxonomy management and item-level taxonomy assignment. ✅
- [x] Backward compatibility is preserved for legacy `labels`/`custom_metadata` behavior. ✅

### Scope

**In Scope:**
- SQLite schema migration for taxonomy registry and assignment tables.
- Persistence repositories and row models for taxonomy CRUD + item assignments.
- Domain services for taxonomy governance, assignment validation, and effective projection merges.
- Label-to-tag backfill strategy with dual-read compatibility window.
- REST endpoints for taxonomy registry and item taxonomy assignment.
- REST list/search filter expansion for taxonomy fields.
- MCP read and write tool additions with runtime write gating.
- UI taxonomy manager modal, metadata editor taxonomy controls, and catalog card taxonomy chips.
- Integration/regression tests and rollout documentation.

**Out of Scope:**
- External database/services beyond existing SQLite persistence.
- New authN/authZ model.
- Breaking changes to existing non-taxonomy API contracts.
- Non-catalog use of taxonomy (outside skills/prompts catalog items).

### Constraints
- Technical: Build on ADR-004 persistence model and effective projection path.
- Compatibility: Preserve legacy `labels` behavior during migration window.
- Operational: Keep taxonomy writes metadata-only and maintain Git content immutability.
- Timeline: Deliver in one incremental implementation cycle.

---

## Requirements Analysis

### Must Have (ADR REQ-1 to REQ-5)
1. Persist item taxonomy assignment fields (`primary_domain`, `secondary_domain`, and tags).
2. Persist taxonomy objects (domains, subdomains, tags) as first-class entities.
3. Support agent taxonomy search and filtering in MCP catalog tools.
4. Support MCP taxonomy writes when write-gate is enabled.
5. Add GUI taxonomy object management and item assignment workflows.

### Should Have (ADR REQ-6 to REQ-8)
1. Render primary/secondary domain and tags in catalog cards.
2. Maintain backward-compatible `labels`/`custom_metadata` semantics.
3. Include taxonomy data in effective catalog responses across REST + MCP.

### Nice to Have (ADR REQ-9)
1. Enforce controlled taxonomy deletions with assignment safeguards.

---

## Public Interface and Contract Changes

### Environment Variables / Flags
- Add `SKILLSERVER_MCP_ENABLE_WRITES` (default `false`) to gate MCP taxonomy write tools.
- Optional CLI flag mirror: `--mcp-enable-writes`.
- Continue requiring persistence gate for durable taxonomy APIs:
  - `SKILLSERVER_PERSISTENCE_DATA=true`

### Additive REST Endpoints
- `GET /api/catalog/taxonomy/domains`
- `POST /api/catalog/taxonomy/domains`
- `PATCH /api/catalog/taxonomy/domains/:id`
- `DELETE /api/catalog/taxonomy/domains/:id`
- `GET /api/catalog/taxonomy/subdomains`
- `POST /api/catalog/taxonomy/subdomains`
- `PATCH /api/catalog/taxonomy/subdomains/:id`
- `DELETE /api/catalog/taxonomy/subdomains/:id`
- `GET /api/catalog/taxonomy/tags`
- `POST /api/catalog/taxonomy/tags`
- `PATCH /api/catalog/taxonomy/tags/:id`
- `DELETE /api/catalog/taxonomy/tags/:id`
- `GET /api/catalog/:id/taxonomy`
- `PATCH /api/catalog/:id/taxonomy`

### Additive REST Query Filters
Apply on both `GET /api/catalog` and `GET /api/catalog/search`:
- `primary_domain_id`
- `secondary_domain_id`
- `subdomain_id` (matches primary or secondary)
- `tag_ids` (comma-separated)
- `tag_match=any|all`

### Additive MCP Tooling
Read tools:
- `list_taxonomy_domains`
- `list_taxonomy_subdomains`
- `list_taxonomy_tags`
- `get_catalog_item_taxonomy`
- Extend `list_catalog`/`search_catalog` taxonomy filters.

Write tools (only when MCP write gate enabled):
- `create_taxonomy_domain`, `update_taxonomy_domain`, `delete_taxonomy_domain`
- `create_taxonomy_subdomain`, `update_taxonomy_subdomain`, `delete_taxonomy_subdomain`
- `create_taxonomy_tag`, `update_taxonomy_tag`, `delete_taxonomy_tag`
- `patch_catalog_item_taxonomy`

### Catalog Contract Extensions
Add taxonomy references to effective catalog item contract:
- `primary_domain`
- `primary_subdomain`
- `secondary_domain`
- `secondary_subdomain`
- `tags` (`id`, `key`, `name`)

Compatibility behavior:
- Keep `labels` output.
- If taxonomy tag assignments exist, derive `labels` from taxonomy tags.
- If not, preserve legacy overlay labels.

---

## Domain Mapping

### Data Layer (`pkg/persistence`)
- Schema migration v2 and indexes.
- Taxonomy registry and assignment repositories.
- Row model validation and SQL query helpers.

### Service Layer (`pkg/domain`)
- Taxonomy registry CRUD service + validation rules.
- Assignment service and domain/subdomain consistency checks.
- Effective projection merge across source, overlay, taxonomy assignment, and tags.
- Backfill orchestration from legacy labels to taxonomy tags.

### API Layer (`pkg/web`)
- Taxonomy registry handlers/routes.
- Item taxonomy get/patch handlers.
- Taxonomy query filters on catalog list/search handlers.

### MCP Layer (`pkg/mcp`, `cmd/skillserver`)
- Taxonomy read and write tool registration.
- Runtime config/write-gate parsing and wiring.
- Taxonomy filter inputs in `list_catalog` and `search_catalog` tools.

### UI Layer (`pkg/web/ui`)
- Options-driven taxonomy manager modal with domain/subdomain/tag tabs.
- Metadata editor taxonomy fields and tag multi-select.
- Catalog card taxonomy chips and search/filter controls.

### Quality + Documentation
- Integration and regression matrix for data/service/API/MCP/UI paths.
- Rollout and rollback runbook for migration and gate enablement.

---

## Work Package Breakdown

### Phase 1: Persistence Foundation
- [x] [WP-001: Taxonomy Schema v2 and Indexes](./work-packages/WP-001-taxonomy-schema-v2-and-indexes.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-002: Taxonomy Persistence Repositories and Row Models](./work-packages/WP-002-taxonomy-persistence-repositories-and-row-models.md) ✅ COMPLETED (2026-03-04)

### Phase 2: Domain Services and Compatibility
- [x] [WP-003: Taxonomy Registry Service and Validation Rules](./work-packages/WP-003-taxonomy-registry-service-and-validation-rules.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-004: Catalog Item Taxonomy Assignment and Effective Projection](./work-packages/WP-004-catalog-item-taxonomy-assignment-and-effective-projection.md) ✅ COMPLETED (2026-03-04)
- [x] [WP-005: Taxonomy Backfill and Legacy Label Compatibility](./work-packages/WP-005-taxonomy-backfill-and-legacy-label-compatibility.md) ✅ COMPLETED (2026-03-04)

### Phase 3: Interface Surfaces
- [x] [WP-006: Taxonomy Registry REST Endpoints](./work-packages/WP-006-taxonomy-registry-rest-endpoints.md) ✅ COMPLETED (2026-03-05)
- [x] [WP-007: Catalog Item Taxonomy REST and Filtered Search](./work-packages/WP-007-catalog-item-taxonomy-rest-and-filtered-search.md) ✅ COMPLETED (2026-03-05)
- [x] [WP-008: MCP Taxonomy Read Tools and Filter Contracts](./work-packages/WP-008-mcp-taxonomy-read-tools-and-filter-contracts.md) ✅ COMPLETED (2026-03-05)
- [x] [WP-009: MCP Taxonomy Write Tools and Runtime Gating](./work-packages/WP-009-mcp-taxonomy-write-tools-and-runtime-gating.md) ✅ COMPLETED (2026-03-05)
- [x] [WP-010: Web UI Taxonomy Management and Item Classification UX](./work-packages/WP-010-web-ui-taxonomy-management-and-item-classification-ux.md) ✅ COMPLETED (2026-03-05)

### Phase 4: Validation and Rollout
- [x] [WP-011: Taxonomy Integration and Regression Test Matrix](./work-packages/WP-011-taxonomy-integration-and-regression-test-matrix.md) ✅ COMPLETED (2026-03-05)
- [x] [WP-012: Rollout, Migration, and Operations Documentation](./work-packages/WP-012-rollout-migration-and-operations-documentation.md) ✅ COMPLETED (2026-03-05)

---

## Dependency Graph

```text
WP-001 -> WP-002 -> WP-003 -> WP-004
WP-002 -> WP-005
WP-003 -> (WP-006 || WP-008)
WP-004 -> (WP-006 || WP-007 || WP-008)
WP-008 -> WP-009
(WP-006 || WP-007) -> WP-010
(WP-005 || WP-007 || WP-009 || WP-010) -> WP-011 -> WP-012
```

### Critical Path
`WP-001 -> WP-002 -> WP-003 -> WP-004 -> WP-007 -> WP-010 -> WP-011 -> WP-012`

### Parallel Opportunities
- WP-005 can run in parallel with API/MCP surface work after WP-002 and WP-004.
- WP-006 and WP-008 can run in parallel after WP-003 and WP-004 are stable.
- WP-009 can begin once WP-008 contracts are stable.

---

## Timeline and Effort

| Phase | Work Packages | Estimated Hours |
|-------|---------------|-----------------|
| Persistence Foundation | WP-001, WP-002 | 9 |
| Domain Services and Compatibility | WP-003, WP-004, WP-005 | 13 |
| Interface Surfaces | WP-006, WP-007, WP-008, WP-009, WP-010 | 23 |
| Validation and Rollout | WP-011, WP-012 | 9 |
| **Total** | **12 WPs** | **54** |

### Schedule Forecast
- Critical-path effort: 38 hours.
- Aggressive (parallelized): 7 working days at 6 productive hours/day.
- Realistic (review + iteration): 8-9 working days.
- Conservative with contingency buffer (x1.3): 70 hours (~11-12 working days).

---

## Test Strategy

### Data Layer Tests
- Migration v2 idempotency and downgrade/upgrade safety.
- Repository CRUD and uniqueness/foreign-key constraints.
- Assignment and tag-join query correctness.

### Service Layer Tests
- Domain/subdomain consistency checks.
- Delete restrictions for in-use taxonomy objects.
- Effective projection merge precedence and label fallback behavior.
- Backfill idempotency and duplicate key handling.

### API Tests
- Taxonomy registry CRUD contracts and validation errors.
- Item taxonomy get/patch behavior and not-found handling.
- List/search taxonomy filters (`any` vs `all` tag semantics).
- Backward-compatibility assertions for `labels` and existing metadata endpoints.

### MCP Tests
- Tool registration matrix with/without MCP write gate.
- Taxonomy read/write tool contract validation.
- `list_catalog` and `search_catalog` taxonomy filter behavior parity with REST.

### UI/E2E Tests
- Options taxonomy manager CRUD flows.
- Metadata modal taxonomy assignment flow and save behavior.
- Catalog card chip rendering and taxonomy filter interactions.

---

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Legacy labels backfill creates duplicate/conflicting tags | Medium | High | Normalize tag keys, enforce unique constraints, run idempotent backfill with conflict-safe upserts. |
| Taxonomy filtering degrades query performance on large catalogs | Medium | Medium | Add indexes on assignment tables and benchmark list/search with representative datasets. |
| Inconsistent validation across service, REST, and MCP surfaces | Medium | Medium | Centralize validation in domain services and keep handlers/tools as thin adapters. |
| Accidental MCP write exposure in environments expecting read-only tools | Low | High | Default write gate to `false`, explicit runtime logging of effective config, dedicated gating tests. |
| UI complexity introduces regressions in existing metadata workflow | Medium | Medium | Reuse existing metadata modal pattern, add targeted UI tests and manual checklist in WP-011. |
| Deletion restrictions block operator workflows | Medium | Low | Return actionable conflict errors and add follow-up docs for reassignment-first deletion path. |

---

## Assumptions and Defaults
1. Persistence is enabled in target environments where taxonomy writes are required.
2. Existing deterministic catalog item IDs remain stable and continue to identify assignment rows.
3. Taxonomy key normalization uses lowercase slug semantics for uniqueness.
4. Legacy `labels` compatibility is maintained for at least one release window.
5. UI will consume additive contracts without requiring a full frontend rewrite.

---

## Next Steps
1. Share the completion report with stakeholders and archive implementation artifacts.
2. Track completion-summary template improvements so future WPs capture actual effort, coverage, and test counts explicitly.
3. Continue post-release monitoring with the WP-012 rollout checklist.

---

## Implementation Completion Summary

**Completion Date:** 2026-03-05
**Status:** ✅ COMPLETED

### Overall Metrics

| Metric | Estimated | Actual | Variance |
|--------|-----------|--------|----------|
| Total Effort | 54 hours | Not explicitly tracked in WP summaries | N/A |
| Work Packages | 12 | 12 | 0 |
| Verification Commands Logged | - | 40 | - |
| Files Touched (reported) | - | 71 | - |
| Duration | - | 2 calendar days (2026-03-04 to 2026-03-05) | - |

### Work Package Summary

| WP ID | Domain | Estimated | Actual | Status | Completed |
|-------|--------|-----------|--------|--------|-----------|
| WP-001 | Data Layer | 4h | Not tracked | ✅ | 2026-03-04 |
| WP-002 | Data Layer | 5h | Not tracked | ✅ | 2026-03-04 |
| WP-003 | Service Layer | 4h | Not tracked | ✅ | 2026-03-04 |
| WP-004 | Service Layer | 5h | Not tracked | ✅ | 2026-03-04 |
| WP-005 | Service Layer | 4h | Not tracked | ✅ | 2026-03-04 |
| WP-006 | API Layer | 4h | Not tracked | ✅ | 2026-03-05 |
| WP-007 | API Layer | 5h | Not tracked | ✅ | 2026-03-05 |
| WP-008 | MCP Layer | 4h | Not tracked | ✅ | 2026-03-05 |
| WP-009 | MCP Layer | 4h | Not tracked | ✅ | 2026-03-05 |
| WP-010 | UI Layer | 6h | Not tracked | ✅ | 2026-03-05 |
| WP-011 | Quality Engineering | 6h | Not tracked | ✅ | 2026-03-05 |
| WP-012 | Documentation | 3h | Not tracked | ✅ | 2026-03-05 |

### Key Achievements

- Delivered full taxonomy stack across persistence, domain services, REST, MCP, UI, and rollout documentation.
- Shipped additive contracts for taxonomy fields while preserving legacy `labels` compatibility.
- Added runtime write-gating for MCP taxonomy mutation tools with default-safe behavior (`false`).
- Validated cross-surface behavior through targeted package tests and taxonomy regression matrix coverage.
- Added rollout/rollback operator documentation and release-note-ready ADR-005 summary artifacts.

### Common Challenges Encountered

1. **Cross-surface contract synchronization** (occurred in 4 WPs)
   - Description: Maintaining matching field names and validation behavior across domain, REST, MCP, and UI.
   - Resolution pattern: Centralized rules in domain services and reused additive selector contracts.

2. **Compatibility migration from legacy labels** (occurred in 3 WPs)
   - Description: Introducing first-class taxonomy objects without breaking existing `labels` consumers.
   - Resolution pattern: Idempotent backfill plus effective projection fallback logic.

3. **Write-surface safety and operational control** (occurred in 2 WPs)
   - Description: Preventing accidental MCP mutation exposure in read-only environments.
   - Resolution pattern: Explicit runtime write gate with default `false` and gating regression tests.

### Lessons Learned

**What Went Well:**
- Domain-level validation and error taxonomy reduced duplication in REST and MCP adapters.
- Additive contract strategy kept existing catalog behavior stable while extending taxonomy capabilities.
- Work-package sequencing/dependency mapping allowed parallel execution in API/MCP/UI phases.

**What Could Be Improved:**
- WP completion summaries should consistently capture actual effort and quantitative quality metrics.
- WP definition files should be updated in-step with completion summaries to reduce status drift.
- Regression evidence should include a standard aggregate summary for easier stakeholder reporting.

**Actionable Recommendations for Future Plans:**
1. Require per-WP actual effort and coverage fields in the completion summary template.
2. Add an execution checklist item to update WP metadata status immediately when a WP closes.
3. Add a small script to auto-aggregate WP completion metrics into the implementation plan/report.

### Technical Debt Summary

| Priority | Count | Total Effort | Tickets Created |
|----------|-------|--------------|-----------------|
| High | 0 | 0h | None |
| Medium | 0 | 0h | None |
| Low | 0 | 0h | None |

**High Priority Debt Items:**
- None recorded in WP completion summaries.

### Follow-Up Items

- [ ] Add actual effort + quality metric fields to the WP completion summary template.
- [ ] Introduce a metrics aggregation helper for implementation-plan closeout.
- [ ] Run a post-release query-performance benchmark for taxonomy filters on larger catalog snapshots.

### References

**Work Package Completion Summaries:**
- [WP-001 Completion Summary](work-packages/completion-summaries/WP-001-completion-summary.md)
- [WP-002 Completion Summary](work-packages/completion-summaries/WP-002-completion-summary.md)
- [WP-003 Completion Summary](work-packages/completion-summaries/WP-003-completion-summary.md)
- [WP-004 Completion Summary](work-packages/completion-summaries/WP-004-completion-summary.md)
- [WP-005 Completion Summary](work-packages/completion-summaries/WP-005-completion-summary.md)
- [WP-006 Completion Summary](work-packages/completion-summaries/WP-006-completion-summary.md)
- [WP-007 Completion Summary](work-packages/completion-summaries/WP-007-completion-summary.md)
- [WP-008 Completion Summary](work-packages/completion-summaries/WP-008-completion-summary.md)
- [WP-009 Completion Summary](work-packages/completion-summaries/WP-009-completion-summary.md)
- [WP-010 Completion Summary](work-packages/completion-summaries/WP-010-completion-summary.md)
- [WP-011 Completion Summary](work-packages/completion-summaries/WP-011-completion-summary.md)
- [WP-012 Completion Summary](work-packages/completion-summaries/WP-012-completion-summary.md)
