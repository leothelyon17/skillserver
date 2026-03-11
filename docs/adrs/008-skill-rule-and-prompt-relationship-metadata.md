# ADR-008: Skill-to-Rule and Skill-to-Prompt Relationship Metadata

## Metadata

| Field | Value |
|-------|-------|
| **Status** | Proposed |
| **Date** | 2026-03-11 |
| **Author(s)** | @jeff |
| **Reviewers** | TBD |
| **Work Package** | N/A |
| **Supersedes** | N/A |
| **Superseded By** | N/A |

## Summary

SkillServer already supports first-class `skill`, `prompt`, and `rule` catalog items plus persistent metadata overlays, but it has no typed way to express that a skill depends on specific rules or a specific prompt. We will add normalized, persistent relationship metadata for `skill -> rule` and `skill -> prompt`, manage those relationships manually through the GUI metadata workflow, and expose the effective relationships through REST and MCP read surfaces so agents can discover the full resource scope for a selected item. MCP relationship writes are explicitly out of scope for this decision.

## Context

### Problem Statement

Users need to manually associate:

- one skill with zero or more rules,
- one rule with zero or more skills,
- one skill with zero or one prompt,
- one prompt with zero or more skills.

Today those links do not exist as first-class data. A catalog item may expose `parent_skill_id`, `resource_path`, taxonomy, labels, and `custom_metadata`, but there is no validated relationship model that lets the GUI show "this skill uses these rules and this prompt" or lets agents ask MCP/API for the related items they should inspect together. Without explicit relationships:

- users must remember dependency scope manually,
- agents cannot deterministically discover the rules and prompt associated with a skill,
- cardinality rules such as "one prompt per skill" cannot be enforced,
- storing links in ad hoc JSON would make reverse lookup and validation fragile.

### Current State

- Unified catalog items already exist for `skill`, `prompt`, and `rule` in [`pkg/domain/catalog.go`](/home/jeff/skillserver/pkg/domain/catalog.go).
- Prompt and rule items are already first-class searchable catalog objects per ADR-003 and ADR-007 in [`docs/adrs/003-unified-skill-prompt-catalog-classification.md`](/home/jeff/skillserver/docs/adrs/003-unified-skill-prompt-catalog-classification.md) and [`docs/adrs/007-rule-catalog-objects-and-mcp-project-materialization.md`](/home/jeff/skillserver/docs/adrs/007-rule-catalog-objects-and-mcp-project-materialization.md).
- Persistent metadata overlays exist in SQLite for free-form item metadata in [`pkg/persistence/migrate.go`](/home/jeff/skillserver/pkg/persistence/migrate.go) and [`pkg/domain/catalog_metadata_service.go`](/home/jeff/skillserver/pkg/domain/catalog_metadata_service.go).
- The GUI already has a metadata editing workflow in [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html), but it is limited to display name, description, taxonomy, labels, and `custom_metadata`.
- REST catalog responses already include `custom_metadata` and `labels` in [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go), but MCP catalog responses intentionally omit general metadata and only expose taxonomy-specific reads in [`pkg/mcp/server.go`](/home/jeff/skillserver/pkg/mcp/server.go) and [`pkg/mcp/tools.go`](/home/jeff/skillserver/pkg/mcp/tools.go).
- Current persistence patterns favor normalized relational structures for controlled vocabularies and assignments, as established by ADR-005.

### Requirements

| Requirement | Priority | Description |
|-------------|----------|-------------|
| REQ-1 | Must Have | Support many-to-many relationships between skills and rules. |
| REQ-2 | Must Have | Support zero-or-one prompt relationship per skill, while allowing one prompt to relate to many skills. |
| REQ-3 | Must Have | Allow users to create and edit these relationships manually through the GUI. |
| REQ-4 | Must Have | Show relationship metadata in the skill, rule, and prompt metadata views without adding relationship badges to the main catalog tiles. |
| REQ-5 | Must Have | Expose effective relationships through REST and MCP read surfaces so agents can determine related resource scope. |
| REQ-6 | Must Have | Enforce classifier and cardinality validation so invalid links cannot be persisted. |
| REQ-7 | Should Have | Keep relationship writes out of MCP for now, while leaving a clean path for future MCP write support. |
| REQ-8 | Should Have | Preserve current content mutability rules: source content may remain read-only while relationship metadata stays writable. |
| REQ-9 | Nice to Have | Keep list/search payloads lean by loading detailed relationship payloads only on metadata/detail surfaces. |

### Constraints

- **Budget**: No new external database or graph service.
- **Timeline**: Should fit the current incremental ADR/work-package delivery style.
- **Technical**: Must build on the current unified catalog, SQLite persistence, REST API, GUI metadata modal, and MCP tool architecture.
- **Compliance**: Relationship edits must not weaken existing read-only protections for Git-imported content.
- **Team**: Prefer low-ambiguity data structures over generic abstractions that increase validation and UX complexity.

## Decision Drivers

1. **Relationship Integrity**: The model must enforce valid item types and the one-prompt-per-skill rule.
2. **Agent Discoverability**: Agents need deterministic read access to relationship scope through MCP/API, not conventions hidden in free-form JSON.
3. **UI Simplicity**: The feature should fit the existing metadata workflow and avoid cluttering the main catalog tiles.
4. **Architectural Consistency**: The implementation should align with the normalized persistence patterns already used for taxonomy and effective catalog projection.
5. **Incremental Delivery**: We should enable GUI-managed relationships now without committing to MCP write semantics yet.

## Options Considered

### Option 1: Store Relationships in `custom_metadata`

**Description**: Represent skill-to-rule and skill-to-prompt associations as conventional keys inside `catalog_metadata_overlays.custom_metadata`.

**Implementation**:
```json
{
  "relationships": {
    "prompt_item_id": "prompt:repo-a/base-skill:prompts/system.md",
    "rule_item_ids": [
      "rule:repo-a/base-skill:rules/security.md",
      "rule:repo-b/shared-guidelines:AGENTS.md"
    ]
  }
}
```

**Pros**:
- Smallest schema change.
- Could reuse the existing metadata PATCH flow quickly.
- No new repository/service layer required initially.

**Cons**:
- No relational integrity or classifier validation at the data layer.
- Reverse lookup for prompt/rule -> skills is expensive and ad hoc.
- Cannot reliably enforce the single-prompt-per-skill rule without custom parsing everywhere.
- MCP/API clients would need undocumented JSON parsing conventions to understand relationships.

**Estimated Effort**: S

**Cost Implications**: Low

---

### Option 2: Normalized Relationship Tables with Effective Relationship Projection (Chosen)

**Description**: Add dedicated persistence tables and a domain service for skill-to-rule and skill-to-prompt associations, then surface a normalized relationship view in GUI metadata, REST metadata, and MCP read tools.

**Implementation**:
```text
Persistence:
  catalog_skill_rule_relationships
  catalog_skill_prompt_relationships

Domain:
  CatalogRelationshipService
    - validate classifier pairs
    - enforce one prompt per skill
    - resolve reverse associations
    - hide/prune deleted endpoints

Presentation:
  GUI metadata modal
  REST metadata/detail responses
  MCP read-only relationship tool
```

**Pros**:
- Enforces the requested cardinality and classifier rules cleanly.
- Supports efficient reverse lookup for rule/prompt metadata views.
- Produces a stable, documented API/MCP contract for agents.
- Aligns with ADR-004/ADR-005 patterns for normalized persisted metadata.

**Cons**:
- Requires migration, repositories, service layer, REST additions, MCP additions, and GUI work.
- Introduces another effective metadata projection to maintain.
- Needs careful handling for soft-deleted catalog items referenced by relationships.

**Estimated Effort**: M

**Cost Implications**: Low

---

### Option 3: Generic Catalog Relationship Graph

**Description**: Introduce one generic `catalog_item_relationships` table with `source_item_id`, `target_item_id`, and `relationship_type`, then model `skill_uses_rule` and `skill_uses_prompt` as typed edges.

**Pros**:
- Most extensible if many future relationship types are expected.
- One general mechanism for all cross-item links.
- Reverse traversal is naturally supported.

**Cons**:
- More abstract than current requirements need.
- Cardinality enforcement becomes more subtle, especially for the single-prompt-per-skill rule.
- Increases product and UI ambiguity because not every future edge type should behave the same way.

**Estimated Effort**: M-L

**Cost Implications**: Low

## Decision

### Chosen Option

**We will implement Option 2: normalized relationship tables with effective relationship projection.**

### Rationale

Option 2 is the smallest design that still gives us strong validation, clean reverse lookup, and an explicit read contract for agents. Option 1 is faster but would bury a first-class feature inside untyped JSON, making relationship discovery and integrity brittle. Option 3 is attractive for future extensibility, but it introduces a more generic graph model than we need today and makes the first delivery harder to reason about. Because the current requirement is narrowly scoped to two skill-owned relationship types with distinct cardinalities, dedicated tables and a focused relationship service provide the best fit.

### Decision Matrix

| Criteria | Weight | Option 1 | Option 2 | Option 3 |
|----------|--------|----------|----------|----------|
| Cardinality and type enforcement | 5 | 1 | 5 | 3 |
| Agent-readable API/MCP contract | 5 | 2 | 5 | 4 |
| Reverse lookup efficiency | 4 | 2 | 5 | 5 |
| Implementation clarity for current scope | 4 | 4 | 4 | 2 |
| Future extensibility | 2 | 2 | 4 | 5 |
| **Weighted Total** |  | **40** | **87** | **70** |

## Consequences

### Positive

- Skills, prompts, and rules can all show relationship scope in metadata without changing the tile grid.
- Agents get a deterministic way to discover associated prompt and rule resources for a skill.
- The single-prompt-per-skill rule becomes enforceable through schema and service validation.
- The design stays consistent with the repository's current normalized metadata direction instead of creating another ad hoc JSON convention.

### Negative

- The feature touches persistence, domain, REST, MCP, and frontend layers.
- Relationship payloads add another set of detail views that must stay aligned between GUI and MCP.
- Initial GUI editing is best anchored on skill metadata, which means prompt and rule metadata views are display-first in the first release.

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Relationships point to items later soft-deleted from the effective catalog | Med | Med | Filter relationships through the effective catalog view and prune invalid rows during sync/startup reconciliation. |
| MCP and REST expose different relationship semantics | Med | High | Use one shared domain projection and mirror the response shape across both surfaces. |
| Users expect tile-level relationship visibility and the grid becomes noisy | Med | Low | Keep relationships out of tile cards and load them only in metadata/detail views. |
| Editing from both sides causes conflicting mental models | Med | Med | Make skill metadata the initial write authority, and show reverse prompt/rule associations as derived metadata in v1. |

## Technical Details

### Architecture

```text
Catalog Source + Overlays + Taxonomy
                |
                v
      Catalog Relationship Service
      - validate writes
      - resolve forward and reverse links
      - enforce one prompt per skill
                |
        +-------+--------+
        |                |
        v                v
  REST metadata      MCP read tool
  + GUI metadata     + optional detail inclusion
```

### Database Changes

Add a new schema migration after the current relationship-free metadata migrations:

```sql
CREATE TABLE IF NOT EXISTS catalog_skill_rule_relationships (
  skill_item_id TEXT NOT NULL,
  rule_item_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  updated_by TEXT,
  PRIMARY KEY (skill_item_id, rule_item_id),
  FOREIGN KEY (skill_item_id) REFERENCES catalog_source_items(item_id) ON UPDATE CASCADE ON DELETE CASCADE,
  FOREIGN KEY (rule_item_id) REFERENCES catalog_source_items(item_id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_catalog_skill_rule_relationships_rule_item_id
ON catalog_skill_rule_relationships (rule_item_id);

CREATE TABLE IF NOT EXISTS catalog_skill_prompt_relationships (
  skill_item_id TEXT PRIMARY KEY,
  prompt_item_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  updated_by TEXT,
  FOREIGN KEY (skill_item_id) REFERENCES catalog_source_items(item_id) ON UPDATE CASCADE ON DELETE CASCADE,
  FOREIGN KEY (prompt_item_id) REFERENCES catalog_source_items(item_id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_catalog_skill_prompt_relationships_prompt_item_id
ON catalog_skill_prompt_relationships (prompt_item_id);
```

Validation rules in the domain layer:

- `catalog_skill_rule_relationships.skill_item_id` must reference a `skill` item.
- `catalog_skill_rule_relationships.rule_item_id` must reference a `rule` item.
- `catalog_skill_prompt_relationships.skill_item_id` must reference a `skill` item.
- `catalog_skill_prompt_relationships.prompt_item_id` must reference a `prompt` item.
- A skill may have at most one prompt relationship.
- Reverse prompt/rule relationships are derived from the stored skill-owned rows; they are not stored twice.

**Migration Strategy**: additive migration with no backfill. Existing items start with empty relationship sets.

### Effective Relationship View

Expose a normalized projection keyed by catalog item ID:

```json
{
  "item_id": "skill:repo-a/python-pro",
  "relationships": {
    "prompt": {
      "id": "prompt:repo-a/base-prompts:prompts/system.md",
      "classifier": "prompt",
      "name": "system"
    },
    "rules": [
      {
        "id": "rule:repo-b/shared-rules:rules/security.md",
        "classifier": "rule",
        "name": "security"
      }
    ],
    "skills": []
  }
}
```

Projection semantics:

- For `skill` items: return `prompt` and `rules`.
- For `prompt` items: return reverse `skills`.
- For `rule` items: return reverse `skills`.
- Suppress relationships whose endpoints are soft-deleted or missing from the effective catalog.

### API Changes

**REST Read**:

- Extend `GET /api/catalog/:id/metadata` and `GET /api/catalog/metadata?item_id=...` to include a `relationships` object.
- Keep `GET /api/catalog` and `GET /api/catalog/search` lean by default; relationship detail is fetched on metadata/detail surfaces, not for tile rendering.

**REST Write**:

- Add `PATCH /api/catalog/:id/relationships` for GUI-driven edits.
- Initial write support is skill-owned:
  - skill payload supports `prompt_item_id` and `rule_item_ids`
  - prompt/rule payloads are read-only in v1 and return reverse associations only

Example write payload:

```json
{
  "prompt_item_id": "prompt:repo-a/base-prompts:prompts/system.md",
  "rule_item_ids": [
    "rule:repo-b/shared-rules:rules/security.md",
    "rule:repo-b/shared-rules:rules/style.md"
  ],
  "updated_by": "gui"
}
```

**Breaking Changes**: None. Existing metadata and taxonomy endpoints remain additive and backward-compatible.

### MCP Changes

Add read-only relationship visibility for agents:

- New tool: `get_catalog_item_relationships`
- Input: `item_id`
- Output: normalized `relationships` view using the same domain projection as REST

`list_catalog` and `search_catalog` remain focused on discovery. Relationship expansion stays detail-oriented so MCP payloads do not become unnecessarily heavy by default.

### GUI Changes

Extend the existing metadata modal in [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html):

- Skill metadata:
  - single-select prompt picker
  - multi-select rule picker
  - summary chips/list of currently linked items
- Prompt metadata:
  - reverse-associated skills list
  - read-only note that prompt associations are managed from skill metadata in v1
- Rule metadata:
  - reverse-associated skills list
  - read-only note that rule associations are managed from skill metadata in v1

Candidate selectors should query the catalog by classifier and display item name plus enough context (`parent_skill_id`, `resource_path`) to disambiguate similarly named prompts/rules.

### Configuration

No new feature flag is required beyond existing persistence and metadata capabilities. The feature follows the same deployment model as current SQLite-backed catalog metadata.

## Security Considerations

### Authentication & Authorization

No new auth model is introduced. Relationship writes should follow the same trust boundary and authorization pattern currently used for GUI metadata mutations. MCP remains read-only for this feature.

### Data Protection

Relationships are metadata only. They do not change source content and must not bypass existing `content_writable` or Git read-only protections. Audit fields such as `updated_at` and `updated_by` should be preserved on relationship writes for operational traceability.

## References

### Internal

- [ADR-004: Persistent Catalog Storage and Metadata Overlays](./004-persistent-catalog-storage-and-metadata-overlays.md)
- [ADR-005: Domain/Subdomain/Tag Taxonomy for Catalog Items](./005-domain-subdomain-tag-taxonomy-for-catalog-items.md)
- [ADR-007: Rule Catalog Objects and MCP Project Materialization](./007-rule-catalog-objects-and-mcp-project-materialization.md)

### External

- N/A
