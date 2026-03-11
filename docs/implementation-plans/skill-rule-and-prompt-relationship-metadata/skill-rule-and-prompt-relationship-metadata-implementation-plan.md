# Implementation Plan: ADR-008 Skill-to-Rule and Skill-to-Prompt Relationship Metadata

**Date Created:** 2026-03-11
**Project Owner:** @jeff
**Target Completion:** 2026-03-24
**Actual Completion:** 2026-03-11
**Status:** COMPLETED
**Source ADR:** [ADR-008: Skill-to-Rule and Skill-to-Prompt Relationship Metadata](../../adrs/008-skill-rule-and-prompt-relationship-metadata.md)

---

## Project Overview

### Goal
Add first-class, persisted relationship metadata so a skill can reference zero-or-more rules and zero-or-one prompt, with reverse visibility from rule and prompt detail views across GUI, REST, and MCP read surfaces.

### Success Criteria
- [x] Skills can persist `rule` relationships and a single `prompt` relationship in normalized SQLite tables.
- [x] Relationship validation enforces classifier correctness and the one-prompt-per-skill rule.
- [x] `GET /api/catalog/:id/metadata` and `GET /api/catalog/metadata?item_id=...` return normalized relationship metadata without bloating list/search payloads.
- [x] `PATCH /api/catalog/:id/relationships` supports skill-owned GUI writes while prompt/rule views remain read-only in v1.
- [x] MCP exposes the same effective relationship projection through a dedicated read tool and does not add relationship write tooling.
- [x] Regression coverage proves deleted or missing endpoints are suppressed from effective views and that Git-backed read-only content semantics remain unchanged.

### Scope

**In Scope:**
- SQLite schema changes for skill-rule and skill-prompt relationship storage.
- Persistence repositories and row-model validation for relationship reads/writes.
- Domain service for validation, effective forward/reverse projection, and invalid-endpoint pruning.
- REST metadata/detail expansion plus a skill-owned relationship PATCH surface.
- MCP read-only relationship tool and runtime wiring.
- Metadata modal updates for skill editing and prompt/rule reverse visibility.
- Cross-surface regression tests and rollout documentation.

**Out of Scope:**
- MCP relationship write tools.
- Relationship badges or chips on the main catalog tiles.
- Generic cross-item graph infrastructure beyond the two ADR-008 relationship types.
- Source-content editing changes for Git-backed prompts, rules, or skills.

### Constraints
- Technical: Must build on the current SQLite persistence model, `CatalogMetadataService`, `CatalogEffectiveService`, Echo handlers, MCP server registration, and the single-file web UI.
- Compatibility: `GET /api/catalog` and `GET /api/catalog/search` remain relationship-light discovery surfaces.
- Compliance: Relationship edits must stay metadata-only and must not weaken existing `content_writable` or Git read-only protections.
- Delivery: The implementation should fit the existing ADR/work-package cadence and remain additive to current public contracts.

---

## Requirements Analysis

### Must Have (ADR REQ-1 to REQ-6)
1. Support many-to-many relationships between skills and rules.
2. Support zero-or-one prompt relationship per skill, with prompt-to-many-skills reverse visibility.
3. Allow users to create and edit relationships manually through the GUI.
4. Show relationships in metadata/detail views for skill, prompt, and rule items.
5. Expose effective relationships through REST and MCP read surfaces.
6. Enforce classifier and cardinality validation before persistence.

### Should Have (ADR REQ-7 to REQ-8)
1. Keep relationship writes out of MCP for v1.
2. Preserve the current separation between content mutability and metadata mutability.

### Nice to Have (ADR REQ-9)
1. Keep list/search payloads lean and relationship detail loaded only on metadata/detail surfaces.

---

## Execution Agent Selection

- Architecture work uses `/home/jeff/.codex/.astra-agents/prompts/implementation-planner.md`.
- Data work uses `/home/jeff/.codex/.astra-agents/prompts/database-architect.md`.
- Backend/API/MCP/runtime work uses `/home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md`.
- Frontend modal work uses `/home/jeff/.codex/.astra-agents/prompts/web-applications-principal-developer-v2.md`.
- Cross-surface regression work uses `/home/jeff/.codex/.astra-agents/prompts/principal-software-developer-system-prompt.md`.
- Documentation-only work leaves `Execution_Agent_Prompt` blank because there is no high-confidence docs-specific local prompt match.

---

## Public Interface and Contract Changes

### Persistence
- Add `catalog_skill_rule_relationships` with `(skill_item_id, rule_item_id)` primary key plus reverse lookup index on `rule_item_id`.
- Add `catalog_skill_prompt_relationships` keyed by `skill_item_id` plus reverse lookup index on `prompt_item_id`.
- Use additive migration semantics with no backfill; existing catalog items start with empty relationship sets.

### Domain Projection
- Introduce one shared relationship projection that returns:
  - For `skill`: `prompt` and `rules`
  - For `prompt`: reverse `skills`
  - For `rule`: reverse `skills`
- Suppress soft-deleted or missing endpoints from effective responses and reconcile stale rows during startup/manual sync flows.

### REST
- Extend metadata/detail responses with an additive `relationships` object.
- Add `PATCH /api/catalog/:id/relationships` for skill-owned edits:
  - `prompt_item_id`
  - `rule_item_ids`
  - optional `updated_by`
- Keep canonical `item_id` requirements aligned with existing metadata endpoints.

### MCP
- Add `get_catalog_item_relationships`.
- Reuse one normalized response shape shared with REST metadata/detail projection.
- Keep `list_catalog` and `search_catalog` unchanged for relationship detail.
- Do not register relationship write tools.

### GUI
- Skill metadata modal adds prompt single-select and rule multi-select controls.
- Prompt/rule metadata views show derived reverse-associated skills and a read-only authority note.
- Candidate selectors reuse existing catalog endpoints with classifier filters and show `parent_skill_id` and `resource_path` context for disambiguation.

---

## WP-001 Finalized Relationship Contract (Authoritative)

This section is the approved relationship contract baseline for WP-002 through
WP-007 and must be treated as normative for schema, service, REST, MCP, and UI
work.

### Relationship Read Shape (REST + MCP)

All relationship detail reads use the same additive payload shape:

```json
{
  "item_id": "skill:repo-a/base-skill",
  "relationships": {
    "prompt": {
      "id": "prompt:repo-a/base-skill:prompts/system.md",
      "classifier": "prompt",
      "name": "system",
      "parent_skill_id": "skill:repo-a/base-skill",
      "resource_path": "prompts/system.md"
    },
    "rules": [
      {
        "id": "rule:repo-b/shared-rules:rules/security.md",
        "classifier": "rule",
        "name": "security",
        "parent_skill_id": "skill:repo-b/shared-rules",
        "resource_path": "rules/security.md"
      }
    ],
    "skills": []
  }
}
```

`relationships` object semantics are fixed:
- `prompt`: object or `null`.
- `rules`: array of related `rule` items.
- `skills`: array of reverse-related `skill` items.

Classifier behavior:
- `skill` detail: `prompt` + `rules` populate forward links, `skills` is always empty.
- `prompt` detail: `skills` populates reverse links, `prompt` is `null`, `rules` is empty.
- `rule` detail: `skills` populates reverse links, `prompt` is `null`, `rules` is empty.

Relationship item fields are additive and consistent across REST and MCP:
- `id` (canonical item ID)
- `classifier` (`skill` | `prompt` | `rule`)
- `name`
- `parent_skill_id` (optional)
- `resource_path` (optional)

### Relationship Write Shape (REST)

`PATCH /api/catalog/:id/relationships` request body:

```json
{
  "prompt_item_id": "prompt:repo-a/base-skill:prompts/system.md",
  "rule_item_ids": [
    "rule:repo-b/shared-rules:rules/security.md",
    "rule:repo-b/shared-rules:rules/style.md"
  ],
  "updated_by": "gui"
}
```

Write semantics:
- Path `:id` must resolve to a `skill` item.
- `prompt_item_id`:
  - canonical prompt ID string -> set/replace current prompt link.
  - explicit `null` -> clear current prompt link.
  - omitted -> leave prompt link unchanged.
- `rule_item_ids`:
  - present -> full replacement of the skill's rule set.
  - omitted -> leave rules unchanged.
  - duplicate IDs are rejected (no silent dedupe).
- `updated_by` is optional audit metadata.

### Canonical ID and Compatibility Rules

- REST relationship surfaces are canonical-only:
  - `GET /api/catalog/:id/metadata`
  - `GET /api/catalog/metadata?item_id=...`
  - `PATCH /api/catalog/:id/relationships`
  - relationship target IDs in REST write payloads
- MCP `get_catalog_item_relationships` accepts canonical IDs for all classifiers.
- MCP compatibility fallback is intentionally bounded:
  - bare `<skill-id>` is accepted only when the requested item is a `skill`.
  - bare IDs are rejected for `prompt` and `rule` item requests.

### Authority and Validation Rules (v1)

- Write authority is skill-owned only.
- Prompt and rule relationship views are reverse-derived and read-only.
- `skill -> prompt` must target classifier `prompt`.
- `skill -> rules` must target classifier `rule`.
- Non-skill relationship writes are rejected on the write surface.

### Deleted/Missing Endpoint Behavior

- Lazy suppression on reads: related items missing from source/effective catalog
  are omitted from payloads.
- Eager reconciliation on runtime sync/startup: stale rows referencing deleted or
  missing endpoints are pruned.
- Both behaviors are required; lazy suppression is the safety net and eager
  reconciliation keeps stored rows clean over time.

### Error Semantics

- `invalid_argument`: malformed IDs, classifier mismatch, duplicate rule IDs, or
  non-canonical IDs where canonical-only policy applies.
- `not_found`: source skill or referenced prompt/rule does not exist in source
  catalog state.
- `read_only_surface`: write attempted on non-skill relationship authority.
- `internal`: repository/runtime failures.

---

## Domain Mapping

### Architecture
- Lock relationship payload shapes, write authority, canonical item-ID expectations, and stale-row handling rules before code work starts.

### Data
- Add schema migration, row models, validation helpers, and repositories for forward/reverse relationship persistence.

### Backend
- Add relationship domain service, metadata/detail projection wiring, REST handlers, MCP tool handlers, and runtime reconciliation hooks.

### Frontend
- Extend the metadata modal save/load workflow for relationship editing and reverse-display states without changing catalog tiles.

### Integration
- Add regression coverage across persistence, domain, REST, MCP, and UI-facing flows.

### Documentation
- Update README/API references, rollout guidance, and release-note artifacts for the additive relationship contract.

---

## Work Package Breakdown

### Phase 1: Contract and Persistence Foundation
- [x] [WP-001: Relationship Contract and Write Authority](./work-packages/WP-001-relationship-contract-and-write-authority.md) (Completed 2026-03-11)
- [x] [WP-002: Relationship Schema Migration and Indexes](./work-packages/WP-002-relationship-schema-migration-and-indexes.md) (Completed 2026-03-11)
- [x] [WP-003: Relationship Repositories and Row Models](./work-packages/WP-003-relationship-repositories-and-row-models.md) (Completed 2026-03-11)

### Phase 2: Domain and Surface Contracts
- [x] [WP-004: Relationship Service, Effective Projection, and Reconciliation](./work-packages/WP-004-relationship-service-effective-projection-and-reconciliation.md) (Completed 2026-03-11)
- [x] [WP-005: REST Relationship Metadata Contracts](./work-packages/WP-005-rest-relationship-metadata-contracts.md) (Completed 2026-03-11)
- [x] [WP-006: MCP Relationship Read Tool and Runtime Wiring](./work-packages/WP-006-mcp-relationship-read-tool-and-runtime-wiring.md) (Completed 2026-03-11)

### Phase 3: UX and Validation
- [x] [WP-007: Web UI Relationship Metadata Editor](./work-packages/WP-007-web-ui-relationship-metadata-editor.md) (Completed 2026-03-11)
- [x] [WP-008: Relationship Integration and Regression Matrix](./work-packages/WP-008-relationship-integration-and-regression-matrix.md) (Completed 2026-03-11)

### Phase 4: Rollout
- [x] [WP-009: Rollout and Operator Documentation](./work-packages/WP-009-rollout-and-operator-documentation.md) (Completed 2026-03-11)

---

## Dependency Graph

```text
WP-001 -> WP-002 -> WP-003 -> WP-004

WP-001 -> WP-005
WP-004 -> (WP-005 || WP-006)
WP-005 -> WP-007

(WP-003 || WP-004 || WP-005 || WP-006 || WP-007) -> WP-008 -> WP-009
```

### Critical Path
`WP-001 -> WP-002 -> WP-003 -> WP-004 -> WP-005 -> WP-007 -> WP-008 -> WP-009`

### Parallel Opportunities
- WP-005 and WP-006 can run in parallel once WP-004 stabilizes the shared relationship projection.
- WP-007 can start as soon as WP-005 lands; it does not need MCP work to finish.
- WP-009 should wait for WP-008 so rollout notes reflect tested behavior rather than intended behavior.

---

## Timeline and Effort

| Phase | Work Packages | Estimated Hours |
|-------|---------------|-----------------|
| Contract and Persistence Foundation | WP-001, WP-002, WP-003 | 11 |
| Domain and Surface Contracts | WP-004, WP-005, WP-006 | 15 |
| UX and Validation | WP-007, WP-008 | 10 |
| Rollout | WP-009 | 3 |
| **Total** | **9 WPs** | **39** |

### Schedule Forecast
- Critical-path effort: 35 hours.
- Aggressive parallelized execution: 6 working days at 6 productive hours/day.
- Realistic execution with review and iteration: 7-8 working days.
- Conservative estimate with contingency buffer (x1.25 on critical path): 8-9 working days.

---

## Test Strategy

### Data Layer Tests
- Migration idempotency and schema-version assertions for both relationship tables and indexes.
- Repository tests for forward lookup, reverse lookup, replace semantics, and deterministic ordering.
- Constraint tests for duplicate prevention and one-prompt-per-skill storage behavior.

### Backend Tests
- Classifier-pair validation for skill->rule and skill->prompt links.
- Reverse-skill projection for prompt/rule metadata views.
- Deleted or missing endpoint suppression and reconciliation/prune behavior.
- Metadata/detail projection coverage without expanding list/search payloads.

### REST Tests
- `GET /api/catalog/:id/metadata` and `GET /api/catalog/metadata` include relationship metadata.
- `PATCH /api/catalog/:id/relationships` validates item IDs, classifier rules, and skill-only write authority.
- Existing metadata overlay and catalog list/search contracts remain additive and backward compatible.

### MCP Tests
- `get_catalog_item_relationships` registers only when the relationship service is wired.
- Structured output matches REST projection semantics.
- No relationship write tools are registered.

### UI Tests
- Skill metadata modal loads prompt/rule candidates and persists relationship edits.
- Prompt/rule metadata views render reverse-associated skills as read-only.
- Catalog cards remain unchanged and free of relationship badges.

---

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| REST, MCP, and UI project different relationship semantics | Medium | High | Define one normalized domain projection in WP-001/WP-004 and reuse it everywhere. |
| Relationship rows reference soft-deleted or missing catalog items | Medium | Medium | Filter through effective catalog reads and add reconciliation pruning during sync/runtime refresh. |
| Similarly named prompt/rule candidates are hard to distinguish in the UI | Medium | Medium | Show item name plus `parent_skill_id` and `resource_path` in selectors and summaries. |
| Users expect prompt/rule-side editing in v1 | Medium | Low | Keep skill metadata as the documented write authority and show explicit read-only guidance on reverse views. |
| Relationship write validation accidentally weakens metadata/content mutability rules | Low | High | Keep relationship writes metadata-only and cover Git-backed read-only cases in regression tests. |

---

## Assumptions and Defaults
1. Relationship writes remain REST/GUI-only in v1.
2. Skill metadata is the only write-authority surface; prompt/rule views are reverse-derived and read-only.
3. REST relationship routes are canonical-only; MCP relationship reads accept bare `<skill-id>` only for skill items and remain canonical-only for prompt/rule items.
4. `GET /api/catalog` and `GET /api/catalog/search` stay relationship-light discovery endpoints.
5. Existing catalog endpoints provide enough data to drive prompt/rule picker UX without adding a new selector-only API.

---

## Implementation Completion Summary

**Completion Date:** 2026-03-11
**Status:** ✅ COMPLETED

### Overall Metrics

| Metric | Value |
|--------|-------|
| Total work packages | 9/9 completed |
| Total estimated effort | 39 hours |
| Total actual effort | 36 hours |
| Variance | -3 hours (-7.7%) |
| Duration | 1 day |
| Dedicated automated tests added | 22 new top-level tests across new persistence, domain, web, and Playwright suites, plus expanded existing MCP/runtime/metadata regressions |
| Verification rerun during close-out | `go test ./pkg/persistence`, `./pkg/domain`, `./pkg/web`, `./pkg/mcp`, and `./cmd/skillserver` targeted relationship suites all passed on 2026-03-11 |

### Work Package Summary

| WP ID | Domain | Estimated | Actual | Status | Completed |
|-------|--------|-----------|--------|--------|-----------|
| WP-001 | Architecture | 3h | 3h | ✅ | 2026-03-11 |
| WP-002 | Data | 4h | 3h | ✅ | 2026-03-11 |
| WP-003 | Data | 4h | 4h | ✅ | 2026-03-11 |
| WP-004 | Backend | 6h | 6h | ✅ | 2026-03-11 |
| WP-005 | Backend | 5h | 5h | ✅ | 2026-03-11 |
| WP-006 | Backend | 4h | 4h | ✅ | 2026-03-11 |
| WP-007 | Frontend | 5h | 5h | ✅ | 2026-03-11 |
| WP-008 | Integration | 5h | 4h | ✅ | 2026-03-11 |
| WP-009 | Documentation | 3h | 2h | ✅ | 2026-03-11 |

### Key Achievements

- Delivered first-class persisted relationship metadata for `skill -> rule` and `skill -> prompt` across SQLite, domain, REST, MCP, and UI layers.
- Preserved additive contract boundaries by keeping list/search payloads relationship-light, maintaining skill-owned write authority, and leaving MCP read-only for v1.
- Landed regression coverage and rollout documentation across persistence, domain, REST, MCP, and UI-facing behavior.

### Common Challenges Encountered

1. **Cross-surface contract drift risk**
   - Resolution: WP-001 locked the authoritative contract early, and WP-004 centralized the reusable relationship projection consumed by REST and MCP.
2. **Stale or deleted relationship endpoints**
   - Resolution: effective reads suppress missing endpoints while runtime reconciliation prunes stale rows during sync/startup flows.
3. **Keeping optional runtime wiring consistent**
   - Resolution: relationship service injection followed the existing metadata/taxonomy runtime pattern and added regression tests around absent-service and registered-tool behavior.

### Lessons Learned

**What Went Well:**
- Contract-first planning prevented divergence between persistence, transport, and UI layers.
- Reusing one domain projection kept REST and MCP semantics aligned with less transport-specific logic.
- Targeted regression gates provided enough confidence to ship additive surface changes safely in one iteration.

**What Could Be Improved:**
- WP status metadata and completion summaries were not updated consistently during execution, which left WP-006 undocumented at close-out time.
- Existing completion summaries capture effort and evidence well enough to close the plan, but they do not record standardized coverage or LOC metrics.
- Close-out required reconstructing some evidence from the worktree because implementation and documentation did not always land together.

**Actionable Recommendations for Future Plans:**
1. Require each WP to update its work-package metadata and completion summary in the same change set as implementation.
2. Standardize completion summaries on a compact metrics block that always captures tests, coverage reporting approach, and change footprint.
3. Add a lightweight validation check that prevents plan finalization when any WP remains unchecked or lacks a completion summary.

### Technical Debt Summary

No technical debt items were explicitly recorded in the WP completion summaries. Deferred items such as MCP relationship writes and tile-level relationship badges remain scoped future enhancements rather than debt created by this implementation.

### Follow-Up Items

- [ ] Consider automating plan close-out checks so missing completion summaries are caught before final review.
- [ ] Consider standardizing per-WP quantitative metrics capture to reduce manual reconstruction during plan completion.

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
- [WP-001](./work-packages/WP-001-relationship-contract-and-write-authority.md)
- [WP-002](./work-packages/WP-002-relationship-schema-migration-and-indexes.md)
- [WP-003](./work-packages/WP-003-relationship-repositories-and-row-models.md)
- [WP-004](./work-packages/WP-004-relationship-service-effective-projection-and-reconciliation.md)
- [WP-005](./work-packages/WP-005-rest-relationship-metadata-contracts.md)
- [WP-006](./work-packages/WP-006-mcp-relationship-read-tool-and-runtime-wiring.md)
- [WP-007](./work-packages/WP-007-web-ui-relationship-metadata-editor.md)
- [WP-008](./work-packages/WP-008-relationship-integration-and-regression-matrix.md)
- [WP-009](./work-packages/WP-009-rollout-and-operator-documentation.md)

---

## Post-Completion Follow-Ups
1. Use the completion report for stakeholder communication and archive this plan as the authoritative delivery record for ADR-008.
2. Treat MCP relationship writes, tile-level relationship badges, and any broader graph work as separate future planning items rather than extensions of this completed plan.
3. Apply the documentation-process follow-ups above before the next multi-package implementation closes.
