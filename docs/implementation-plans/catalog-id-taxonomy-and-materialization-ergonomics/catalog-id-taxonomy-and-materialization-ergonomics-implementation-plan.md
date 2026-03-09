# Implementation Plan: Catalog ID, Taxonomy Classification, and Materialization Ergonomics

**Date Created:** 2026-03-09
**Project Owner:** @jeff
**Target Completion:** 2026-03-20
**Actual Completion:** 2026-03-09
**Status:** COMPLETE
**Related Plans:** [ADR-005 taxonomy rollout](../domain-subdomain-tag-taxonomy-for-catalog-items/domain-subdomain-tag-taxonomy-for-catalog-items-implementation-plan.md), [ADR-007 rule catalog and materialization rollout](../rule-catalog-objects-and-mcp-project-materialization/rule-catalog-objects-and-mcp-project-materialization-implementation-plan.md)
**Release Notes:** [Catalog ID, Taxonomy, and Materialization Ergonomics Release Notes](../../releases/2026-03-09-catalog-id-taxonomy-and-materialization-ergonomics-release-notes.md)
**Completion Report:** [Catalog ID, Taxonomy, and Materialization Ergonomics Completion Report](./catalog-id-taxonomy-and-materialization-ergonomics-completion-report.md)

---

## Project Overview

### Goal
Tighten the unified catalog contracts so catalog item IDs are normalized consistently, taxonomy classification state is explicit and queryable, taxonomy mutations support partial and batch workflows, list/search payloads are lightweight by default, export/materialization behavior is easier for agents to consume, and taxonomy usage/preflight data is surfaced directly to MCP and the web UI.

### Success Criteria
- [x] Skill, prompt, and rule callers can use a single canonical catalog-item identity model without taxonomy-specific surprises. ✅
- [x] Catalog list/search and taxonomy get responses expose explicit classification state instead of implying it through omitted fields. ✅
- [x] Clients can add/remove/clear tags and apply batch taxonomy mutations without fragile read-modify-write loops. ✅
- [x] REST and MCP list/search defaults are metadata-first, paginated, and opt into content explicitly. ✅
- [x] MCP export archives no longer default to wrapper roots such as `skills/...` unless explicitly requested. ✅
- [x] Taxonomy manager delete flows can show usage counts and impacted items before mutation attempts. ✅

### Scope

**In Scope:**
- Shared catalog item reference normalization across domain, REST, and MCP layers.
- Additive classification-state fields and list/search filters.
- Partial and batch taxonomy mutation contracts with dry-run preview.
- Metadata-first catalog list/search defaults with `include_content`, `limit`, and `cursor`.
- Export manifest/archive root ergonomics and MCP export payload reduction.
- Taxonomy usage/preflight service surfaces for domains, subdomains, and tags.
- Web UI updates for explicit classification state, usage counts, and batch taxonomy actions.
- Regression coverage and documentation updates.

**Out of Scope:**
- New authentication or authorization behavior.
- Replacing the current filesystem or SQLite persistence model.
- Reworking search indexing beyond what is needed to preserve current query semantics.
- Introducing new taxonomy object types beyond domain, subdomain, and tag.
- Non-catalog bulk metadata overlay editing outside taxonomy-specific mutations.

### Constraints
- Compatibility: existing `read_skill`, resource APIs, catalog metadata APIs, and legacy skill-focused clients must keep working during rollout.
- Technical: changes must fit the current `pkg/domain`, `pkg/persistence`, `pkg/web`, `pkg/mcp`, and Alpine UI architecture.
- Operational: MCP materialization write gating and allowed destination roots must remain intact.
- Delivery: prefer additive contracts and dual-format acceptance over breaking renames.

---

## Requirements Analysis

### Must Have
1. Standardize skill catalog identifiers so `list_skills`, taxonomy tools, export/materialization flows, and REST item routes all agree on item identity behavior.
2. Make unclassified or partially classified state explicit with stable response fields and list/search filters.
3. Add additive taxonomy mutation primitives for tag add/remove/clear and a batch patch flow with dry-run preview.
4. Make `list_catalog` and `search_catalog` metadata-first by default and paginate them deterministically.
5. Fix export/materialization ergonomics so archive roots are useful when extracted and MCP callers are not forced to parse large base64 payloads unless requested.
6. Expose taxonomy usage counts and impacted item previews before delete attempts or major edits.

### Should Have
1. Return populated display names in `list_skills` and `search_skills`.
2. Preserve content-search behavior even when response payloads omit `content`.
3. Preserve backward compatibility for bare skill IDs during the transition window.

### Nice to Have
1. Return structured conflict payloads on delete attempts so UI and agents can reuse the same preflight data.

---

## Finalized Contract Decisions

WP-001 completed on 2026-03-09, and WP-002 through WP-009 implemented and
verified the finalized contract on the same date. The sections below now
describe shipped behavior rather than a future target state.

### 1. Canonical ID Policy

Canonical catalog item IDs are fixed as:

- skill item: `skill:<canonical-skill-key>`
- prompt item: `prompt:<canonical-skill-key>:<canonical-resource-path>`
- rule item: `rule:<canonical-skill-key>:<canonical-resource-path>`

Bounded compatibility rules:

- Only bare skill IDs receive backward-compatible fallback handling.
- Prompt and rule item IDs remain canonical-only on every public surface.
- No additional alias formats, classifier synonyms, or bare resource
  references are supported.

### 2. Public Surface Compatibility Matrix

| Surface | Output contract | Accepted input contract | Compatibility boundary |
|---------|-----------------|-------------------------|------------------------|
| MCP `list_skills`, `search_skills` | Emit canonical skill item IDs in `id`; populate `name` from the skill display name. | None. | Bare skill IDs stop being emitted after rollout. |
| MCP `read_skill`, `list_skill_resources`, `read_skill_resource`, `get_skill_resource_info` | N/A | Accept bare `<skill-id>` and canonical `skill:<skill-id>`; normalize both to the same parent skill key before resolution. | Prompt/rule item IDs are rejected because these are skill-scoped surfaces. |
| MCP `get_catalog_item_taxonomy`, `patch_catalog_item_taxonomy` | Return canonical `item_id` values. | Accept bare skill IDs only when the target classifier is `skill`; prompt/rule inputs must already be canonical. | Bare fallback is limited to skill items. |
| MCP `export_catalog_items`, `materialize_catalog_items` | Manifest/result payloads always report canonical `item_id` values. | Same normalization contract as taxonomy item tools. | No bare prompt/rule support; no extra write-capable export surface. |
| REST `GET /api/catalog`, `GET /api/catalog/search` | Response `id` is always canonical. | None. | These remain canonical-source list/search surfaces. |
| REST `/api/catalog/:id/taxonomy` and request-body `item_id` fields on taxonomy mutation routes | Response `item_id` values are canonical. | Accept bare skill IDs or canonical `skill:<skill-id>` for skill items; prompt/rule IDs are canonical-only. | Route decoding and batch results preserve canonical output while keeping any original bare request in `requested_item_id`. |
| REST `/api/catalog/:id/metadata` and `/api/catalog/metadata?item_id=...` | Response `item_id` values are canonical. | Canonical item IDs only. | Metadata routes intentionally do not widen bare-skill compatibility in this rollout. |
| REST `/api/catalog/export` and `/api/catalog/materialize` request-body `item_id` fields | Manifest/result payloads always report canonical `item_id` values. | Accept bare skill IDs only when the target classifier is `skill`; prompt/rule inputs must already be canonical. | Bare fallback is limited to skill items. |
| Legacy `/api/skills`, `/api/skills/search`, `/api/skills/*`, `/api/skills/export/*` | Keep current skill-name/path semantics. | Keep current bare skill route parameters. | These legacy routes are explicitly outside the catalog-item ID migration. |

### 3. Classification State Model

Additive classification-state fields are required on taxonomy reads and
effective catalog item responses:

- `has_assignment=true` when any taxonomy field is populated:
  `primary_domain`, `primary_subdomain`, `secondary_domain`,
  `secondary_subdomain`, or one or more `tags`.
- `is_fully_classified=true` when `primary_domain` is present and at least one
  tag is assigned.
- `missing_fields` is a stable, ordered list of absent fields drawn only from:
  `primary_domain`, `primary_subdomain`, `secondary_domain`,
  `secondary_subdomain`, `tags`.

Additional semantics:

- `missing_fields` order is fixed as listed above.
- Secondary domain and both subdomain references remain optional for
  `is_fully_classified`, but they still appear in `missing_fields` when absent.
- `has_assignment=false` always implies `is_fully_classified=false`.

### 4. Mutation Semantics

- `tag_ids` remains explicit full replacement when present.
- Additive tag fields are:
  - `add_tag_ids`
  - `remove_tag_ids`
  - `clear_tags`
- `tag_ids` is mutually exclusive with `add_tag_ids`, `remove_tag_ids`, and
  `clear_tags`.
- `clear_tags=true` is mutually exclusive with `add_tag_ids` and
  `remove_tag_ids`.
- `add_tag_ids` and `remove_tag_ids` may be combined only when their normalized
  sets are disjoint.
- Batch mutation request-shape validation failures reject the entire request
  before per-item execution begins.
- Per-item execution statuses are fixed as:
  - `planned`
  - `updated`
  - `unchanged`
  - `invalid`
  - `not_found`

### 5. List/Search Defaults

- `include_content=false` is the default for REST and MCP list/search payloads.
- Search continues to match content in the backend even when `content` is
  omitted from the response payload.
- Deterministic ordering is ascending canonical `item_id`.
- Pagination contract:
  - request: `limit`, `cursor`
  - response: `next_cursor`, `has_more`
  - default `limit`: `50`
  - maximum `limit`: `200`
  - `cursor` semantics: exclusive "after item_id" cursor bound to the same
    query/filter/classifier set
- Classification-state filters are additive and operate on effective state:
  - `unclassified=true` means `has_assignment=false`
  - `missing_primary_domain=true` means `primary_domain` is absent
  - `missing_tags=true` means the item has zero tags
- REST compatibility rule: legacy callers that omit `limit` and `cursor` may
  continue receiving the existing array response shape during the rollout
  window; paginated REST mode returns an object envelope with `items`,
  `next_cursor`, and `has_more`.

### 6. Export/Materialization Ergonomics

- MCP export adds `archive_root_mode=flat|materialized`.
- Default MCP behavior is `archive_root_mode=flat`.
- `flat` removes synthetic materialization wrapper directories such as
  `skills/`, `prompts/`, and `rules/` from archive roots while preserving the
  item's natural payload root.
- `materialized` preserves the exact target paths produced by
  `materialize_catalog_items`.
- REST `POST /api/catalog/export` remains download-oriented and returns
  materialized target paths in the manifest.
- MCP export adds `include_archive_base64=false` by default; callers opt in
  only when they require the archive bytes inline.
- `materialize_catalog_items` and `POST /api/catalog/materialize` remain the
  only caller-directed write surfaces in this rollout.

### 7. Usage/Preflight API Shape

- Add explicit usage endpoints/tools per taxonomy object type:
  - domain
  - subdomain
  - tag
- Each usage response includes:
  - object ID and type
  - assignment count
  - distinct impacted item count
  - impacted item preview list (capped and sorted by canonical `item_id`)
  - `blocking_reason` for delete flows
- Initial `blocking_reason` token is `in_use` when assignments would block
  deletion; omit the field when no delete blocker exists.

---

## Domain Mapping

### Architecture
- Public contract decisions and compatibility rules.
- Shared normalization and completeness semantics.
- Archive root policy and pagination behavior.

### Backend
- Shared item-reference normalizer and completeness derivation.
- Taxonomy mutation orchestration and dry-run planning.
- Usage/preflight aggregation service.
- Export/materialization behavior coordination.

### Data
- Repository support for deterministic pagination and usage counts.
- Assignment and tag lookup helpers required by batch mutation and usage previews.
- Optional index additions only if query plans show regressions.

### Integration
- REST and MCP request/response contract expansion.
- Route/tool registration, validation, compatibility, and error encoding.

### Frontend
- Taxonomy manager usage visibility.
- Classification-state filters and item badges.
- Batch tag editing and paginated catalog navigation.

### Documentation
- README, test matrix, and rollout guidance updates.

---

## Milestones

### Milestone 1: Contract Foundation
- [x] [WP-001 Architecture Contract and Compatibility Matrix](./work-packages/WP-001-architecture-contract-and-compatibility-matrix.md) ✅ COMPLETED (2026-03-09)
- [x] [WP-002 Catalog Reference Normalizer and Classification-State Domain Model](./work-packages/WP-002-catalog-reference-normalizer-and-classification-state-domain-model.md) ✅ COMPLETED (2026-03-09)
- [x] [WP-003 Repository Pagination and Taxonomy Usage Query Support](./work-packages/WP-003-repository-pagination-and-taxonomy-usage-query-support.md) ✅ COMPLETED (2026-03-09)

### Milestone 2: Mutation and Surface Upgrades
- [x] [WP-004 Partial and Batch Taxonomy Mutation Services](./work-packages/WP-004-partial-and-batch-taxonomy-mutation-services.md) ✅ COMPLETED (2026-03-09)
- [x] [WP-005 REST Catalog and Taxonomy Contract Expansion](./work-packages/WP-005-rest-catalog-and-taxonomy-contract-expansion.md) ✅ COMPLETED (2026-03-09)
- [x] [WP-006 MCP Contract Expansion and Export Ergonomics](./work-packages/WP-006-mcp-contract-expansion-and-export-ergonomics.md) ✅ COMPLETED (2026-03-09)

### Milestone 3: UX and Validation
- [x] [WP-007 Web UI Taxonomy Manager and Catalog Classification UX](./work-packages/WP-007-web-ui-taxonomy-manager-and-catalog-classification-ux.md) ✅ COMPLETED (2026-03-09)
- [x] [WP-008 Regression Matrix and Compatibility Coverage](./work-packages/WP-008-regression-matrix-and-compatibility-coverage.md) ✅ COMPLETED (2026-03-09)

### Milestone 4: Rollout Documentation
- [x] [WP-009 Documentation, Examples, and Release Guidance](./work-packages/WP-009-documentation-examples-and-release-guidance.md) ✅ COMPLETED (2026-03-09)

---

## Work Package Breakdown

### WP-001: Architecture Contract and Compatibility Matrix

```yaml
WP_ID: WP-001
Title: Architecture Contract and Compatibility Matrix
Domain: architecture
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/implementation-planner.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for contract definition, sequencing, and compatibility planning.
Priority: High
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-09
Started_Date: 2026-03-09
Completed_Date: 2026-03-09
```

**Description:**
Define the additive contract changes before implementation starts: canonical ID policy, completeness semantics, pagination defaults, batch mutation semantics, usage/preflight response shape, and export archive root policy.

**Deliverables:**
- [x] One compatibility matrix that maps every affected public surface to its old and new ID behavior.
- [x] One completeness-state definition for `has_assignment`, `is_fully_classified`, and `missing_fields`.
- [x] One public-contract inventory covering REST, MCP, and UI-facing JSON fields.
- [x] One decision note for `archive_root_mode` and `include_archive_base64`.

**Primary File Locations:**
- `docs/implementation-plans/catalog-id-taxonomy-and-materialization-ergonomics/catalog-id-taxonomy-and-materialization-ergonomics-implementation-plan.md`
- `README.md`

**Dependencies:**
- Blocked by: None
- Blocks: WP-002, WP-003, WP-004, WP-005, WP-006, WP-007, WP-008, WP-009
- Parallel Execution: None

**Acceptance Criteria:**
- [x] Every requested improvement has a concrete contract decision.
- [x] Compatibility behavior for legacy bare skill IDs is explicit.
- [x] Completeness semantics are stable enough for REST, MCP, and UI to share.
- [x] Export ergonomics decision avoids duplicate write surfaces.

### WP-002: Catalog Reference Normalizer and Classification-State Domain Model

```yaml
WP_ID: WP-002
Title: Catalog Reference Normalizer and Classification-State Domain Model
Domain: backend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong match for Go backend contract shaping and domain-service implementation.
Priority: High
Estimated_Effort: 5 hours
Status: COMPLETE
```

**Description:**
Create one shared domain normalizer for item references and add explicit classification-state derivation to effective catalog items, metadata views, and taxonomy assignment views.

**Deliverables:**
- Shared helper(s) for accepting bare skill IDs and canonical `skill:` IDs.
- Domain-model additions for `has_assignment`, `is_fully_classified`, and `missing_fields`.
- Shared completeness derivation used by effective projection and direct taxonomy reads.
- Backward-compatible helpers consumed by export/materialization code.

**Primary File Locations:**
- `pkg/domain/catalog.go`
- `pkg/domain/catalog_effective_service.go`
- `pkg/domain/catalog_metadata_service.go`
- `pkg/domain/catalog_taxonomy_assignment_service.go`
- `pkg/domain/catalog_export_service.go`
- `pkg/domain/catalog_materialization_service.go`
- `pkg/domain/catalog_effective_service_test.go`
- `pkg/domain/catalog_taxonomy_assignment_service_test.go`
- `pkg/domain/catalog_export_service_test.go`
- `pkg/domain/catalog_materialization_service_test.go`

**Dependencies:**
- Blocked by: WP-001
- Blocks: WP-004, WP-005, WP-006, WP-007, WP-008, WP-009
- Parallel Execution: Can run in parallel with WP-003 after WP-001

**Acceptance Criteria:**
- [x] Bare skill IDs and canonical skill item IDs normalize to the same canonical reference.
- [x] Prompt and rule IDs remain canonical-only and validate cleanly.
- [x] Unclassified and partially classified states are explicit in domain outputs.
- [x] Existing skill-only callers keep working without behavior regressions.

### WP-003: Repository Pagination and Taxonomy Usage Query Support

```yaml
WP_ID: WP-003
Title: Repository Pagination and Taxonomy Usage Query Support
Domain: data
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/database-architect.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong match for repository query design, pagination, and count/preview query work.
Priority: High
Estimated_Effort: 4 hours
Status: COMPLETE
```

**Description:**
Add repository support for deterministic catalog pagination and cheap taxonomy usage/preflight lookups without changing the storage model unnecessarily.

**Deliverables:**
- Additive pagination inputs for source-row listing (`limit`, `after item_id` cursor semantics).
- Usage-count and preview queries for domain, subdomain, and tag references.
- Query helpers reusable by delete-preflight flows and UI manager tables.
- Benchmarked or reasoned confirmation that no schema migration is required; if indexes are needed, document them before code lands.

**Primary File Locations:**
- `pkg/persistence/catalog_row_models.go`
- `pkg/persistence/catalog_source_repository.go`
- `pkg/persistence/catalog_taxonomy_row_models.go`
- `pkg/persistence/catalog_source_repository_test.go`
- `pkg/persistence/catalog_taxonomy_repository_test.go`

**Dependencies:**
- Blocked by: WP-001
- Blocks: WP-004, WP-005, WP-006, WP-007, WP-008
- Parallel Execution: Can run in parallel with WP-002 after WP-001

**Acceptance Criteria:**
- [x] Pagination is deterministic and based on stable item ordering.
- [x] Usage queries return counts and preview IDs without requiring full table scans in handlers.
- [x] Repository behavior remains additive and backward compatible.
- [x] New queries are covered by repository tests.

### WP-004: Partial and Batch Taxonomy Mutation Services

```yaml
WP_ID: WP-004
Title: Partial and Batch Taxonomy Mutation Services
Domain: backend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong match for mutation orchestration, validation rules, and dry-run service behavior.
Priority: High
Estimated_Effort: 5 hours
Status: COMPLETE
```

**Description:**
Extend the taxonomy assignment service so clients can add, remove, and clear tags without full replacement, and add a batch planner/executor that supports dry-run previews and per-item outcomes.

**Deliverables:**
- Extend single-item patch input with `add_tag_ids`, `remove_tag_ids`, and `clear_tags`.
- Add batch patch request/result types and dry-run preview behavior.
- Add one shared usage/preflight service for taxonomy object delete preparation.
- Preserve existing `tag_ids` full replacement semantics when explicitly supplied.

**Primary File Locations:**
- `pkg/domain/catalog_taxonomy_assignment_service.go`
- `pkg/domain/catalog_taxonomy_service.go`
- `pkg/domain/catalog_taxonomy_usage_service.go`
- `pkg/domain/catalog_taxonomy_assignment_service_test.go`
- `pkg/domain/catalog_taxonomy_service_test.go`

**Dependencies:**
- Blocked by: WP-002, WP-003
- Blocks: WP-005, WP-006, WP-007, WP-008, WP-009
- Parallel Execution: None

**Acceptance Criteria:**
- [x] Single-item taxonomy patch supports additive tag operations without read-modify-write loops.
- [x] Batch patch supports `dry_run` and deterministic per-item results.
- [x] Delete-preflight data can be produced without performing the delete.
- [x] Validation order prevents partial writes on global request-shape failures.

### WP-005: REST Catalog and Taxonomy Contract Expansion

```yaml
WP_ID: WP-005
Title: REST Catalog and Taxonomy Contract Expansion
Domain: integration
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong fit for Echo handler contracts, request decoding, and additive REST API changes.
Priority: High
Estimated_Effort: 5 hours
Status: COMPLETE
```

**Description:**
Expose the new normalization, completeness, batch mutation, usage, and pagination behavior through REST without breaking existing callers.

**Deliverables:**
- Extend `GET /api/catalog` and `GET /api/catalog/search` with:
  - `include_content`
  - `limit`
  - `cursor`
  - `unclassified`
  - `missing_primary_domain`
  - `missing_tags`
- Extend `GET /api/catalog/:id/taxonomy` and effective catalog DTOs with explicit classification-state fields.
- Extend single-item patch payloads with additive tag operations.
- Add a batch taxonomy patch endpoint with `dry_run`.
- Add usage/preflight endpoints for domain, subdomain, and tag.
- Return structured conflict details from delete flows where possible.

**Primary File Locations:**
- `pkg/web/handlers.go`
- `pkg/web/handlers_catalog_metadata_test.go`
- `pkg/web/handlers_taxonomy_test.go`
- `pkg/web/handlers_export_materialization_test.go`

**Dependencies:**
- Blocked by: WP-002, WP-003, WP-004
- Blocks: WP-007, WP-008, WP-009
- Parallel Execution: Can run in parallel with WP-006 after prerequisites complete

**Acceptance Criteria:**
- [x] REST responses are metadata-first by default and paginate deterministically.
- [x] REST filters can target unclassified and partially classified items.
- [x] Batch and single-item taxonomy mutations are both additive and backward compatible.
- [x] Delete-preflight usage data is accessible without reading opaque error strings.

### WP-006: MCP Contract Expansion and Export Ergonomics

```yaml
WP_ID: WP-006
Title: MCP Contract Expansion and Export Ergonomics
Domain: integration
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong fit for MCP tool schema changes, compatibility handling, and archive/export behavior.
Priority: High
Estimated_Effort: 5 hours
Status: COMPLETE
```

**Description:**
Bring MCP tool behavior into parity with the new REST/domain contracts and reduce agent friction for skill IDs, list/search payload size, taxonomy mutations, usage preflight, and export archives.

**Deliverables:**
- `list_skills` and `search_skills` return canonical skill IDs and populated display names.
- `read_skill` and skill-resource tools accept both bare and canonical skill IDs.
- `list_catalog` and `search_catalog` add `include_content`, `limit`, `cursor`, and classification-state filters.
- `get_catalog_item_taxonomy` returns explicit classification-state fields.
- `patch_catalog_item_taxonomy` accepts additive tag mutation inputs.
- New `patch_catalog_items_taxonomy` batch tool with `dry_run`.
- New usage read tools for domain, subdomain, and tag.
- `export_catalog_items` supports `archive_root_mode` and `include_archive_base64`, with flattened roots by default.

**Primary File Locations:**
- `pkg/mcp/tools.go`
- `pkg/mcp/tools_export_materialization.go`
- `pkg/mcp/server.go`
- `pkg/mcp/server_stdio_regression_test.go`

**Dependencies:**
- Blocked by: WP-002, WP-003, WP-004
- Blocks: WP-008, WP-009
- Parallel Execution: Can run in parallel with WP-005 after prerequisites complete

**Acceptance Criteria:**
- [x] No MCP list/search tool returns blank skill names.
- [x] Taxonomy and skill tools accept both legacy and canonical skill references where appropriate.
- [x] List/search payloads omit content unless explicitly requested.
- [x] Export archives no longer default to wrapper roots for common extraction flows.

### WP-007: Web UI Taxonomy Manager and Catalog Classification UX

```yaml
WP_ID: WP-007
Title: Web UI Taxonomy Manager and Catalog Classification UX
Domain: frontend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-software-developer-system-prompt.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for UI state, interaction design, and end-to-end web behavior.
Priority: Medium
Estimated_Effort: 5 hours
Status: COMPLETE
```

**Description:**
Update the Alpine UI so it can consume metadata-first catalog responses, show explicit classification state, expose usage/preflight information, and perform additive/batch taxonomy edits.

**Deliverables:**
- Catalog list badges for unclassified and partially classified items.
- Filter controls for unclassified items, missing primary domain, and missing tags.
- Usage counts and impacted-item preview in taxonomy manager delete flows.
- Metadata editor support for additive tag mutation and optional batch taxonomy operations.
- Pagination controls or lazy-loading behavior for lighter catalog list/search results.

**Primary File Locations:**
- `pkg/web/ui/index.html`
- `pkg/web/ui/style.css`
- `tests/playwright/wp010-ui-taxonomy.spec.ts`
- `tests/playwright/wp010-ui-export-materialization.spec.ts`

**Dependencies:**
- Blocked by: WP-005
- Blocks: WP-008, WP-009
- Parallel Execution: Can run in parallel with WP-006 once REST contracts are stable

**Acceptance Criteria:**
- [x] Taxonomy manager shows usage information before delete attempts.
- [x] UI can work without `content` embedded in catalog list/search payloads.
- [x] Explicit classification state is visible without opening the metadata modal.
- [x] Playwright coverage proves the new flows work end to end.

### WP-008: Regression Matrix and Compatibility Coverage

```yaml
WP_ID: WP-008
Title: Regression Matrix and Compatibility Coverage
Domain: integration
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong fit for cross-surface regression design and API/MCP compatibility validation.
Priority: High
Estimated_Effort: 4 hours
Status: COMPLETE
```

**Description:**
Codify the regression gates for ID normalization, completeness fields, new filters, batch mutations, lighter catalog responses, usage preflight, and export root handling.

**Deliverables:**
- Go tests for domain, REST, and MCP behavior.
- Playwright updates for taxonomy manager and export/materialization UX.
- A refreshed test execution matrix in `tests/README.md`.

**Primary File Locations:**
- `pkg/domain/*_test.go`
- `pkg/web/*_test.go`
- `pkg/mcp/server_stdio_regression_test.go`
- `tests/playwright/wp010-ui-taxonomy.spec.ts`
- `tests/playwright/wp010-ui-export-materialization.spec.ts`
- `tests/README.md`

**Dependencies:**
- Blocked by: WP-005, WP-006, WP-007
- Blocks: WP-009
- Parallel Execution: None

**Acceptance Criteria:**
- [x] Compatibility tests prove legacy bare skill IDs still work where promised.
- [x] New classification filters and completeness fields are covered in domain, REST, and MCP tests.
- [x] Batch mutation dry-run and apply behavior is covered.
- [x] Export flattening and usage-preflight flows are covered.

### WP-009: Documentation, Examples, and Release Guidance

```yaml
WP_ID: WP-009
Title: Documentation, Examples, and Release Guidance
Domain: documentation
Execution_Agent_Prompt:
Agent_Selection_Source: blank
Agent_Selection_Rationale: No highly relatable installed prompt for this narrowly scoped documentation update package.
Priority: Medium
Estimated_Effort: 3 hours
Status: COMPLETE
```

**Description:**
Update project documentation so operators and agents can discover the new ID rules, filters, mutation contracts, usage endpoints/tools, and export behavior without reading code.

**Deliverables:**
- README updates for REST and MCP contract changes.
- Example payloads for:
  - canonical skill IDs
  - classification-state fields
  - batch taxonomy patch dry-run
  - usage/preflight responses
  - export archive root modes
- Rollout notes describing additive compatibility behavior and default payload-size changes.

**Primary File Locations:**
- `README.md`
- `tests/README.md`
- `docs/implementation-plans/catalog-id-taxonomy-and-materialization-ergonomics/catalog-id-taxonomy-and-materialization-ergonomics-implementation-plan.md`

**Dependencies:**
- Blocked by: WP-005, WP-006, WP-007, WP-008
- Blocks: None
- Parallel Execution: None

**Acceptance Criteria:**
- [x] README documents every new filter, field, and tool/endpoint.
- [x] Examples use canonical IDs and show compatibility notes for bare skill IDs.
- [x] Export/materialization docs explain flattened archive roots and opt-in archive bytes.
- [x] Test matrix references the new regression gates.

---

## Dependency Graph

```text
WP-001 -> (WP-002 || WP-003)
(WP-002 || WP-003) -> WP-004
(WP-002 || WP-003 || WP-004) -> (WP-005 || WP-006)
WP-005 -> WP-007
(WP-005 || WP-006 || WP-007) -> WP-008 -> WP-009
```

### Critical Path
`WP-001 -> WP-002 -> WP-004 -> WP-005 -> WP-007 -> WP-008 -> WP-009`

### Parallel Opportunities
- WP-002 and WP-003 can run in parallel after WP-001.
- WP-005 and WP-006 can run in parallel after WP-002, WP-003, and WP-004.
- WP-007 can start as soon as WP-005 is stable; it does not need to wait for MCP work.

---

## Timeline and Effort

| Milestone | Work Packages | Estimated Hours |
|-----------|---------------|-----------------|
| Milestone 1: Contract Foundation | WP-001, WP-002, WP-003 | 12 |
| Milestone 2: Mutation and Surface Upgrades | WP-004, WP-005, WP-006 | 15 |
| Milestone 3: UX and Validation | WP-007, WP-008 | 9 |
| Milestone 4: Rollout Documentation | WP-009 | 3 |
| **Total** | **9 WPs** | **39** |

### Schedule Forecast
- Critical-path effort: 30 hours.
- Aggressive parallelized execution: 5-6 working days at 6 productive hours/day.
- Realistic execution with review and fix cycles: 7 working days.
- Conservative estimate with contingency buffer (x1.25 on critical path): 37-38 hours.

---

## Test Strategy

### Domain
- Normalize bare and canonical skill references to the same canonical item ID.
- Verify completeness derivation for unclassified, partially classified, and fully classified items.
- Verify additive tag mutation and batch dry-run semantics.
- Verify export manifest/archive roots for `flat` vs `materialized`.

### Persistence
- Verify paginated source listing is deterministic.
- Verify usage-count and preview queries for domains, subdomains, and tags.
- Verify no unintended regressions in current source-row filters.

### REST
- Verify new filters on `GET /api/catalog` and `GET /api/catalog/search`.
- Verify single-item and batch taxonomy mutation payloads.
- Verify usage/preflight endpoints and conflict payloads.

### MCP
- Verify `list_skills` and `search_skills` names/IDs.
- Verify dual-format skill ID acceptance in skill/taxonomy/export tools.
- Verify metadata-first list/search behavior and pagination.
- Verify usage tools and batch patch tool registration/behavior.

### UI
- Verify taxonomy manager usage counts and delete warnings.
- Verify catalog classification badges and filters.
- Verify metadata editor and batch-tag workflows.
- Verify content preview still works through dedicated reads when list/search omit content.

### Suggested Command Matrix

```bash
go test ./pkg/domain -run 'TestCatalogTaxonomyAssignmentService_|TestCatalogEffectiveService_|TestCatalogExportService_|TestCatalogMaterializationService_' -count=1
go test ./pkg/persistence -run 'TestCatalogSourceRepository_|TestCatalogTaxonomy' -count=1
go test ./pkg/web -run 'TestCatalog.*Taxonomy|TestCatalog.*Metadata|TestExportCatalog_|TestMaterializeCatalog_' -count=1
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression|TestTaxonomyWriteTools_' -count=1
npx playwright test tests/playwright/wp010-ui-taxonomy.spec.ts tests/playwright/wp010-ui-export-materialization.spec.ts --project=chromium
```

---

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Changing `list_skills.id` breaks legacy callers that pass IDs directly to `read_skill` | Medium | High | Accept both bare and canonical skill IDs everywhere a skill reference is read; add explicit compatibility tests. |
| Completeness semantics become ambiguous and force follow-up contract churn | Medium | Medium | Lock the `missing_fields` vocabulary and `is_fully_classified` rule in WP-001 before code changes land. |
| Batch mutation semantics produce partial-write surprises | Medium | High | Separate request-shape validation from per-item execution, require `dry_run`, and add deterministic per-item statuses. |
| Metadata-first list/search responses regress UI flows that currently rely on inline content | Medium | Medium | Keep content-search behavior in the backend, add `include_content=true`, and continue using dedicated read endpoints for previews. |
| Usage queries become too expensive on large catalogs | Low | Medium | Keep queries targeted by object ID, cap preview item lists, and add indexes only if repository tests or profiling justify them. |
| Archive-root flattening surprises existing consumers that expect wrapper folders | Low | Medium | Preserve current behavior behind `archive_root_mode=materialized` and document the default change clearly. |

---

## Implementation Completion Summary

**Completion Date:** 2026-03-09
**Status:** COMPLETE
**Work Packages:** 9/9 complete

### Overall Metrics

| Metric | Estimated | Actual | Variance |
|--------|-----------|--------|----------|
| Total Effort | 39h | 22.5h | -16.5h (-42.3%) |
| Work Packages | 9 | 9 | 0 |
| Completion Date | 2026-03-20 target | 2026-03-09 actual | 11 days early |

### Milestone Summary

| Milestone | Estimated | Actual | Variance |
|-----------|-----------|--------|----------|
| Milestone 1: Contract Foundation | 12h | 5.5h | -6.5h (-54.2%) |
| Milestone 2: Mutation and Surface Upgrades | 15h | 10h | -5h (-33.3%) |
| Milestone 3: UX and Validation | 9h | 5h | -4h (-44.4%) |
| Milestone 4: Rollout Documentation | 3h | 2h | -1h (-33.3%) |

### Work Package Summary

| WP ID | Domain | Estimated | Actual | Status | Completed |
|-------|--------|-----------|--------|--------|-----------|
| WP-001 | Architecture | 3h | 1.5h | COMPLETE | 2026-03-09 |
| WP-002 | Backend | 5h | 2h | COMPLETE | 2026-03-09 |
| WP-003 | Data | 4h | 2h | COMPLETE | 2026-03-09 |
| WP-004 | Backend | 5h | 3h | COMPLETE | 2026-03-09 |
| WP-005 | Integration | 5h | 4h | COMPLETE | 2026-03-09 |
| WP-006 | Integration | 5h | 3h | COMPLETE | 2026-03-09 |
| WP-007 | Frontend | 5h | 3h | COMPLETE | 2026-03-09 |
| WP-008 | Integration | 4h | 2h | COMPLETE | 2026-03-09 |
| WP-009 | Documentation | 3h | 2h | COMPLETE | 2026-03-09 |

### Key Achievements

- Unified catalog-item ID normalization and explicit classification-state semantics now align across domain, REST, MCP, and UI surfaces.
- Taxonomy workflows now support additive and batch mutation semantics, plus usage/preflight visibility ahead of destructive actions.
- Metadata-first list/search defaults, flatter export ergonomics, and rollout documentation shipped with matching regression coverage across Go and Playwright suites.

### Common Challenges Encountered

1. **Compatibility boundary management** (WP-001, WP-005, WP-006, WP-008, WP-009)
   - Bare-skill compatibility, legacy array responses, and canonical-only exceptions had to stay explicit across REST, MCP, docs, and tests.
   - Resolution pattern: lock the contract early, preserve only the bounded compatibility surfaces, and prove them with targeted regression coverage.
2. **Metadata-first rollout side effects** (WP-002, WP-005, WP-006, WP-007, WP-009)
   - Lighter list/search payloads changed assumptions in adapter fakes, UI preview flows, documentation, and export behavior.
   - Resolution pattern: keep content-search behavior in the backend, add opt-in `include_content`, and update preview/test flows to use dedicated reads when inline content is absent.
3. **Deterministic mutation and pagination semantics** (WP-003, WP-004, WP-008)
   - Cursor validity, dry-run validation order, and cross-item taxonomy apply semantics required stricter boundaries than the pre-rollout code exposed.
   - Resolution pattern: bind cursors to filter sets, validate the entire batch request shape up front, and document the remaining shared-transaction gap explicitly.

### Lessons Learned

**What Went Well:**
- Contract-first sequencing kept downstream work packages focused and consistently under estimate.
- Shared normalization, pagination, and usage helpers prevented repeated implementation across service, transport, and UI layers.
- Cross-surface regression coverage made compatibility decisions reviewable instead of implicit.

**What Could Be Improved:**
- Work-package definition docs should be updated as part of each package closeout; WP-005 and WP-007 were still marked `DEFINED` when plan closeout began.
- Completion summaries should capture aggregate metrics such as tests added, files changed, and technical-debt tickets consistently.
- Shared Playwright fixture assumptions and no-op interaction paths should be documented earlier to reduce regression churn.

**Actionable Recommendations for Future Plans:**
1. Add a closeout checklist that synchronizes work-package metadata, completion summaries, and implementation-plan acceptance criteria in the same change.
2. Extend the completion-summary template with required aggregate metrics and ticket-reference fields.
3. Keep compatibility boundaries and legacy exceptions documented in the contract-foundation package before transport and UI implementation starts.

### Verification Summary

- Domain: `go test ./pkg/domain`
- Persistence: `go test ./pkg/persistence -count=1`
- REST: `go test ./pkg/web -count=1`
- MCP: `go test ./pkg/mcp -count=1`
- Runtime wiring: `go test ./cmd/skillserver -count=1`
- UI regression: `npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp007-ui-catalog-classification.spec.ts tests/playwright/wp008-ui.spec.ts tests/playwright/wp010-ui-taxonomy.spec.ts --project=chromium`

The completion summaries do not record aggregate test counts, line-count deltas, or coverage percentages consistently enough to report those totals accurately here.

### Technical Debt and Follow-Up

- No dedicated technical-debt tickets were recorded in the work-package summaries.
- Remaining known debt: cross-item taxonomy batch apply is still not wrapped in a single shared transaction across assignment repositories.
- Release and follow-up items:
  - Use the `WP-008` matrix in `tests/README.md` as the release go/no-go gate.
  - Share the release notes with operators and MCP client maintainers before promotion.
  - Treat any future REST metadata bare-skill compatibility widening as a separate additive contract change.

### References

- [WP-001 Completion Summary](./work-packages/completion-summaries/WP-001-completion-summary.md)
- [WP-002 Completion Summary](./work-packages/completion-summaries/WP-002-completion-summary.md)
- [WP-003 Completion Summary](./work-packages/completion-summaries/WP-003-completion-summary.md)
- [WP-004 Completion Summary](./work-packages/completion-summaries/WP-004-completion-summary.md)
- [WP-005 Completion Summary](./work-packages/completion-summaries/WP-005-completion-summary.md)
- [WP-006 Completion Summary](./work-packages/completion-summaries/WP-006-completion-summary.md)
- [WP-007 Completion Summary](./work-packages/completion-summaries/WP-007-completion-summary.md)
- [WP-008 Completion Summary](./work-packages/completion-summaries/WP-008-completion-summary.md)
- [WP-009 Completion Summary](./work-packages/completion-summaries/WP-009-completion-summary.md)
- [Catalog ID, Taxonomy, and Materialization Ergonomics Completion Report](./catalog-id-taxonomy-and-materialization-ergonomics-completion-report.md)

---

## Next Steps

1. Use the `WP-008` section in [`tests/README.md`](/home/jeff/skillserver/tests/README.md) as the release gate for rollout and rollback decisions.
2. Share [`docs/releases/2026-03-09-catalog-id-taxonomy-and-materialization-ergonomics-release-notes.md`](/home/jeff/skillserver/docs/releases/2026-03-09-catalog-id-taxonomy-and-materialization-ergonomics-release-notes.md) with operators and MCP client maintainers before promotion.
3. Track any future widening of REST metadata bare-skill compatibility as a separate additive contract change instead of inferring it from taxonomy compatibility.
