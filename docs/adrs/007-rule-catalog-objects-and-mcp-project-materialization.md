# ADR-007: Rule Catalog Objects and MCP Project Materialization

## Metadata

| Field | Value |
|-------|-------|
| **Status** | Proposed |
| **Date** | 2026-03-08 |
| **Author(s)** | @jeff |
| **Reviewers** | TBD |
| **Work Package** | N/A |
| **Supersedes** | N/A |
| **Superseded By** | N/A |

## Summary

SkillServer currently supports first-class catalog items for `skill` and `prompt`, but not for project-level rules such as `AGENTS.md`. Agents can discover items through MCP, yet they cannot ask SkillServer to materialize selected skills, prompts, or rules into a local project folder, and the existing GUI export path is a brittle skill-only flow. We will extend the unified catalog to include a third classifier, `rule`, and introduce a shared export/materialization service reused by both MCP and the GUI.

## Context

### Problem Statement

Three gaps now block the desired workflow:

- Rules are not first-class catalog objects, so agents cannot search, classify, export, or install them the same way as skills and prompts.
- MCP is read-oriented today: an agent can list or read catalog items, but not request that selected items be written into the working project directory.
- The existing GUI export function is implemented as a legacy skill-only path and is not a reliable foundation for a broader install/export workflow.

The target feature is:

1. Rules become a first-class SkillServer object type, equivalent in status to skills and prompts.
2. Agents can request selected catalog items to be downloaded/materialized into a local project folder through MCP.
3. The GUI export flow becomes functional by using the same shared export/materialization service rather than a special-case route.

### Current State

- Unified catalog classifiers are currently limited to `skill` and `prompt` in [`pkg/domain/catalog.go`](/home/jeff/skillserver/pkg/domain/catalog.go).
- Prompt catalog items are discovered from resource paths and indexed via the unified catalog builder in [`pkg/domain/manager_catalog.go`](/home/jeff/skillserver/pkg/domain/manager_catalog.go).
- Search/index behavior assumes only two classifier values in [`pkg/domain/search.go`](/home/jeff/skillserver/pkg/domain/search.go).
- Persistence constrains `catalog_source_items.classifier` to `skill` or `prompt` in [`pkg/persistence/migrate.go`](/home/jeff/skillserver/pkg/persistence/migrate.go) and [`pkg/persistence/catalog_row_models.go`](/home/jeff/skillserver/pkg/persistence/catalog_row_models.go).
- REST and MCP catalog filtering currently validate only `skill` and `prompt` in [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go) and [`pkg/mcp/tools.go`](/home/jeff/skillserver/pkg/mcp/tools.go).
- The GUI export button currently calls a skill-only endpoint from [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html), while the backend export handler is implemented as a legacy wildcard route in [`pkg/web/server.go`](/home/jeff/skillserver/pkg/web/server.go) and [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go).
- ADR-002, ADR-003, ADR-004, and ADR-005 already established the patterns for imported resource discovery, unified catalog classifiers, persistent effective catalog items, and taxonomy on catalog items.

### Requirements

| Requirement | Priority | Description |
|-------------|----------|-------------|
| REQ-1 | Must Have | Add `rule` as a first-class catalog classifier alongside `skill` and `prompt`. |
| REQ-2 | Must Have | Make rules searchable, filterable, and taxonomized through the same REST and MCP catalog surfaces as other item types. |
| REQ-3 | Must Have | Provide an MCP workflow for materializing selected skills/prompts/rules into a local project folder. |
| REQ-4 | Must Have | Make GUI export functional by reusing the same export/materialization service. |
| REQ-5 | Must Have | Enforce path safety for project-folder writes: no absolute paths, no traversal, and no writes outside configured allowed roots. |
| REQ-6 | Must Have | Preserve current source mutability rules: imported Git content remains read-only at source, while exported/materialized copies are explicit user-owned outputs. |
| REQ-7 | Should Have | Support install metadata so rule items can land at project-root targets such as `AGENTS.md`, `CLAUDE.md`, or `RULES.md`. |
| REQ-8 | Should Have | Support batching multiple catalog items in one materialization/export request. |
| REQ-9 | Should Have | Preserve legacy skill export/import behavior for backward compatibility during rollout. |
| REQ-10 | Nice to Have | Support dry-run planning so agents can inspect resolved target paths before writing files. |

### Constraints

- **Budget**: No new external package registry, blob store, or managed database.
- **Timeline**: Deliver incrementally; the first milestone should unblock the broken GUI export path and establish the shared service seam.
- **Technical**: Must build on the current file-system/Git discovery, Bleve index, SQLite persistence, and MCP transport architecture.
- **Compliance**: Project-folder writes must be explicitly bounded and auditable; no implicit writes outside configured destinations.
- **Team**: Keep the design maintainable for the current self-hosted single-instance deployment model.

## Decision Drivers

1. **Execution usefulness**: agents need to turn selected catalog content into files in the project they are actively editing.
2. **Catalog consistency**: rules should behave like first-class objects, not hidden conventions outside the catalog model.
3. **Single implementation path**: GUI export and MCP project materialization should share one service and one set of path rules.
4. **Safety**: local writes are higher-risk than catalog reads and must be controlled by explicit runtime configuration and path validation.
5. **Backward compatibility**: existing skill and prompt behavior must remain stable while the new rule classifier and materialization capabilities are added.

## Options Considered

### Option 1: Keep Skill/Prompt Catalog Only and Let Clients Assemble Files

**Description**: Do not add `rule` as a classifier and do not add any materialization/export service beyond the current skill archive behavior. Agents would continue reading content through existing MCP tools and then write files using client-side tooling only.

**Implementation**:
```text
Agent flow:
  1) search_catalog / read_skill / read_skill_resource
  2) client decides destination paths
  3) client writes project files itself
```

**Pros**:
- Lowest server-side implementation effort.
- No new write surface in SkillServer.
- No persistence migration required.

**Cons**:
- Rules remain outside the first-class object model.
- GUI export remains special-case and brittle.
- Every client must reinvent packaging, naming, and install-path behavior.
- No shared safety model for project-folder writes.

**Estimated Effort**: S

**Cost Implications**: Low

---

### Option 2: Extend Unified Catalog with `rule` and Add Shared Export/Materialization Service (Chosen)

**Description**: Add `rule` as a third classifier in the unified catalog, discover rule resources using configurable conventions, and introduce a shared service that can either package selected catalog items for download or materialize them directly into an allowed destination directory. GUI export and MCP materialization become thin wrappers over this same service.

**Implementation**:
```text
Catalog build:
  - emit skill items
  - emit prompt items
  - emit rule items

Shared service:
  - resolve catalog item IDs
  - compute target paths
  - validate destination root and conflict policy
  - either:
      a) produce archive/download payload
      b) write files under destination_dir

Clients:
  - GUI export -> shared service
  - MCP materialize/export tools -> shared service
```

**Pros**:
- Fully addresses the desired user workflow.
- Keeps catalog semantics consistent across skills, prompts, and rules.
- Avoids duplicate export/install logic across UI and MCP.
- Preserves the additive architecture established by earlier ADRs.

**Cons**:
- Requires changes across domain, persistence, search, REST, MCP, and UI layers.
- Introduces a new write-capable MCP path that must be carefully gated.
- Requires a schema migration because classifier values are constrained today.

**Estimated Effort**: L

**Cost Implications**: Low

---

### Option 3: Create a Separate Rule Registry and Separate Installer Subsystem

**Description**: Keep the existing unified catalog for skills/prompts only, and build a parallel rule subsystem with its own APIs, persistence, and materialization behavior.

**Pros**:
- Strong conceptual separation for project rules.
- Could evolve independently from skill/prompt behavior.

**Cons**:
- Splits search, taxonomy, and GUI behavior across multiple subsystems.
- Duplicates infrastructure and implementation patterns already solved by the unified catalog.
- Increases product and code complexity without delivering additional user value.

**Estimated Effort**: XL

**Cost Implications**: Medium

## Decision

### Chosen Option

**We will implement Option 2: extend the unified catalog with `rule` and add a shared export/materialization service.**

### Rationale

Option 2 is the only option that solves all three problems together: first-class rules, MCP-based project-folder materialization, and a reliable GUI export path. It fits the current architecture because classifiers already drive catalog indexing, filtering, persistence, and UI rendering. It also creates a clean seam for rollout: the existing export bug can be resolved by moving GUI export onto the shared service first, then expanding the classifier model and MCP tooling.

### Decision Matrix

| Criteria | Weight | Option 1 | Option 2 | Option 3 |
|----------|--------|----------|----------|----------|
| Supports first-class `rule` objects | 5 | 1 | 5 | 5 |
| Supports MCP project-folder materialization | 5 | 2 | 5 | 4 |
| Reuses one implementation for GUI + MCP | 4 | 1 | 5 | 2 |
| Backward compatibility | 4 | 5 | 4 | 2 |
| Operational simplicity | 3 | 5 | 3 | 1 |
| **Weighted Total** |  | **50** | **86** | **57** |

## Consequences

### Positive

- Rules become searchable, classifiable, exportable, and installable like other catalog items.
- Agents can request project-local files directly instead of reconstructing them client-side.
- GUI export becomes more robust because it uses the same path planning and packaging rules as MCP.
- Existing persistence and taxonomy features remain reusable because the design extends the existing classifier model rather than replacing it.

### Negative

- Classifier expansion affects many layers and requires a broad regression matrix.
- Materialization introduces a new class of safety and conflict-handling concerns.
- The UX must distinguish between reading content, exporting archives, and materializing files into a project directory.

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Rule detection is too broad and wrongly classifies generic markdown files | Med | Med | Restrict classification to configurable `rule`/`rules` directories plus an explicit filename allowlist for project-rule files. |
| Project materialization writes outside intended roots | Low | High | Require configured allowed destination roots, reject absolute paths and traversal, and support dry-run planning. |
| Schema migration breaks persistence because classifier checks allow only `skill` and `prompt` | Med | High | Add an explicit migration that widens classifier constraints before enabling rule indexing. |
| GUI and MCP drift on filename/path semantics | Med | Med | Centralize target-path planning and packaging in one shared service with thin adapters for both callers. |

## Technical Details

### Architecture

```text
Catalog Discovery
  - skills
  - prompts
  - rules
        |
        v
Unified Catalog Builder
  classifier = skill | prompt | rule
        |
        +--> Bleve index
        +--> SQLite source snapshot + overlays + taxonomy
        |
        v
Export / Materialization Service
  - resolve catalog item IDs
  - compute target paths
  - validate write roots
  - package archive OR write files
        |
        +--> GUI export
        +--> REST export/materialize endpoints
        +--> MCP export/materialize tools
```

### Object Model

Extend the classifier model in [`pkg/domain/catalog.go`](/home/jeff/skillserver/pkg/domain/catalog.go) and [`pkg/persistence/catalog_row_models.go`](/home/jeff/skillserver/pkg/persistence/catalog_row_models.go):

- `skill`: directory-backed object with `SKILL.md`
- `prompt`: file-backed catalog item derived from prompt resource discovery
- `rule`: file-backed catalog item derived from rule discovery

`rule` should be represented as a first-class catalog item like `prompt`, using existing fields:

```json
{
  "id": "rule:repo/skill-name:rules/project.md",
  "classifier": "rule",
  "name": "project-guidelines",
  "parent_skill_id": "repo/skill-name",
  "resource_path": "rules/project.md",
  "content_writable": false,
  "metadata_writable": true,
  "read_only": true
}
```

### Rule Discovery

Rule discovery should extend the same resource/discovery model introduced for prompts:

1. Default rule directories:
   - `rule`
   - `rules`
2. Default project-rule filename allowlist:
   - `AGENTS.md`
   - `RULES.md`
   - `CLAUDE.md`
   - `GEMINI.md`
3. Rule candidates must be readable text/markdown resources.
4. Imported `imports/...` rule resources remain subject to the same repo-boundary safety rules established in ADR-002.

This keeps rule detection explicit and avoids classifying arbitrary markdown content as a rule.

### Install Metadata

File-backed catalog items should support optional install metadata in frontmatter:

```yaml
---
name: project-guidelines
description: Rules for contributors working in this repo
materialize:
  target_path: AGENTS.md
  conflict_policy: overwrite
---
```

Target path resolution order:

1. `materialize.target_path`, if present and valid.
2. Preserve known project-rule basenames at project root for rules such as `AGENTS.md`.
3. Otherwise use classifier-based defaults under the requested destination:
   - `skills/<skill-name>/...`
   - `prompts/<filename>.md`
   - `rules/<filename>.md`

### Search and Persistence Changes

- Extend classifier parsing and validation in:
  - [`pkg/domain/catalog.go`](/home/jeff/skillserver/pkg/domain/catalog.go)
  - [`pkg/domain/search.go`](/home/jeff/skillserver/pkg/domain/search.go)
  - [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go)
  - [`pkg/mcp/tools.go`](/home/jeff/skillserver/pkg/mcp/tools.go)
- Add a new persistence migration after the taxonomy migration in [`pkg/persistence/migrate.go`](/home/jeff/skillserver/pkg/persistence/migrate.go) so `catalog_source_items.classifier` accepts `rule`.
- No new persistence tables are required; existing source/overlay/taxonomy tables remain classifier-agnostic once the constraint is widened.

### API Changes

**New additive endpoints**:

- `POST /api/catalog/export`
  - Input: `item_ids`, `format`, optional `dry_run`
  - Output: archive payload or manifest
- `POST /api/catalog/materialize`
  - Input: `item_ids`, `destination_dir`, `conflict_policy`, optional `dry_run`
  - Output: resolved target paths and results

**Legacy compatibility**:

- Keep the current skill export flow as a compatibility wrapper during rollout.
- Re-implement the GUI export button on top of the shared export service rather than the legacy wildcard path.

### MCP Changes

Additive MCP tools:

- `export_catalog_items`
  - Package selected catalog items for download/use by the caller.
- `materialize_catalog_items`
  - Write selected catalog items into a configured project destination root.
- Optional:
  - `plan_catalog_materialization`

Suggested runtime flags:

```bash
SKILLSERVER_CATALOG_ENABLE_RULES=true
SKILLSERVER_CATALOG_RULE_DIRS=rule,rules
SKILLSERVER_CATALOG_RULE_FILENAMES=AGENTS.md,RULES.md,CLAUDE.md,GEMINI.md
SKILLSERVER_MCP_ENABLE_MATERIALIZATION=true
SKILLSERVER_MCP_ALLOWED_DESTINATION_ROOTS=/workspace,/projects
```

Materialization tools should be gated separately from read-only MCP tools.

### GUI Changes

The GUI should shift from a skill-only export action to classifier-aware export behavior:

- `skill`: export full skill package as today, but via shared service.
- `prompt`: export/materialize as single-file artifact.
- `rule`: export/materialize as single-file artifact with project-root-aware target path support.

This also addresses the current “export never worked” issue by removing dependence on the legacy special-case flow and aligning the GUI with the same path semantics used by the backend service.

### Implementation Plan

**Phase 1: Shared export seam**
- Introduce export/materialization domain service.
- Move existing GUI skill export to that service via a compatibility wrapper.
- Add route and integration regression coverage.

**Phase 2: Rule classifier foundation**
- Add `rule` classifier in domain, persistence, search, REST, and MCP.
- Add configurable rule detection and install metadata parsing.

**Phase 3: MCP materialization**
- Add guarded MCP tools for archive export and project-folder materialization.
- Enforce destination-root and conflict-policy rules.

**Phase 4: UX and documentation**
- Update the GUI to surface export/materialize actions for prompts and rules.
- Update docs, rollout guidance, and regression matrices.

### Rollback Plan

1. Disable `SKILLSERVER_MCP_ENABLE_MATERIALIZATION`.
2. Disable `SKILLSERVER_CATALOG_ENABLE_RULES`.
3. Keep legacy skill export behavior active.
4. If required, stop indexing `rule` items and preserve existing persisted rows until a formal downgrade path is executed.

## Security Considerations

### Authentication & Authorization

Materialization is a write capability and should remain independently gated from catalog read access. The new MCP write tools should not register unless explicitly enabled at runtime.

### Data Protection

- Reject absolute target paths and any path containing traversal segments.
- Restrict writes to configured allowed roots.
- Record execution metadata for materialization operations when persistence is enabled.

## References

- [ADR-002: Dynamic Imported Resource Discovery and Prompt Support](./002-dynamic-resource-and-prompt-discovery.md)
- [ADR-003: Unified Skill/Prompt Catalog Classification for Git Imports](./003-unified-skill-prompt-catalog-classification.md)
- [ADR-004: Persistent Catalog Storage and Metadata Overlays](./004-persistent-catalog-storage-and-metadata-overlays.md)
- [ADR-005: Domain/Subdomain/Tag Taxonomy for Catalog Items](./005-domain-subdomain-tag-taxonomy-for-catalog-items.md)
