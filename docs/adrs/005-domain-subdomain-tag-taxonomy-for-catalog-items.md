# ADR-005: Domain/Subdomain/Tag Taxonomy for Catalog Items

## Metadata

| Field | Value |
|-------|-------|
| **Status** | Proposed |
| **Date** | 2026-03-04 |
| **Author(s)** | @jeff |
| **Reviewers** | TBD |
| **Work Package** | N/A |
| **Supersedes** | N/A |
| **Superseded By** | N/A |

## Summary

SkillServer already supports classifier-aware catalog items (`skill`/`prompt`) and persistent metadata overlays, but classification metadata is currently limited to free-form `labels` and `custom_metadata`. We will add first-class, persistent taxonomy objects for `domains`, `subdomains`, and `tags`, plus per-item assignment fields for `primary domain`, `secondary domain`, and tags. This enables precise MCP and API filtering, GUI taxonomy management from a global Options menu, and consistent item-card presentation of domain/tag metadata.

## Context

### Problem Statement

Operators and agents need a structured way to classify every catalog item (skill or prompt) by:

- Primary domain
- Secondary domain
- Individual tags

Current free-form labels do not provide a global controlled vocabulary, relational integrity, or consistent query semantics for agents using MCP. The system also lacks a GUI workflow for managing domain/subdomain/tag objects globally and attaching them to items.

### Current State

- Catalog items are unified across skills/prompts in [`pkg/domain/catalog.go`](/home/jeff/skillserver/pkg/domain/catalog.go) and [`pkg/domain/manager_catalog.go`](/home/jeff/skillserver/pkg/domain/manager_catalog.go).
- Persistent metadata overlays exist in SQLite (`catalog_metadata_overlays`) per ADR-004 in [`pkg/persistence/migrate.go`](/home/jeff/skillserver/pkg/persistence/migrate.go).
- Metadata edit UX exists in [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html), but fields are generic (`display_name`, `description`, `labels`, `custom_metadata`).
- MCP currently exposes classifier-aware catalog search (`list_catalog`, `search_catalog`) in [`pkg/mcp/server.go`](/home/jeff/skillserver/pkg/mcp/server.go) and [`pkg/mcp/tools.go`](/home/jeff/skillserver/pkg/mcp/tools.go), but no domain/tag object management.

### Requirements

| Requirement | Priority | Description |
|-------------|----------|-------------|
| REQ-1 | Must Have | Every catalog item can store `primary_domain`, `secondary_domain`, and zero-or-more tags. |
| REQ-2 | Must Have | Domains, subdomains, and tags are first-class persistent objects (not just free text). |
| REQ-3 | Must Have | Agents can search catalog items by domain/subdomain/tag via MCP. |
| REQ-4 | Must Have | Agents can create/update taxonomy objects and item assignments via MCP when MCP writes are enabled. |
| REQ-5 | Must Have | GUI exposes taxonomy object management from a global Options entry and lets users attach taxonomy to items. |
| REQ-6 | Should Have | Item cards show primary/secondary domain and tags directly in the catalog grid. |
| REQ-7 | Should Have | Existing `labels`/`custom_metadata` behavior remains backward-compatible during migration. |
| REQ-8 | Should Have | Taxonomy data is included in effective catalog responses and search filtering in REST + MCP. |
| REQ-9 | Nice to Have | Controlled deletion policies (prevent deleting taxonomy objects still assigned to items). |

### Constraints

- **Budget**: No new external managed database dependency.
- **Timeline**: One incremental delivery cycle aligned with current ADR implementation cadence.
- **Technical**: Build on existing SQLite persistence and effective catalog projection introduced in ADR-004.
- **Compliance**: Preserve read-only content semantics for Git-imported items while allowing metadata edits.
- **Team**: Keep APIs additive and minimize frontend disruption.

## Decision Drivers

1. **Discoverability Precision**: Agents need deterministic filtering by domain/subdomain/tag, not free-text approximations.
2. **Controlled Vocabulary**: Global taxonomy objects prevent drift and duplicate spelling variants.
3. **Persistence and Mutability Separation**: Taxonomy must persist independently of content mutability.
4. **MCP + GUI Parity**: Human users and agents need equivalent read/write classification workflows.
5. **Backward Compatibility**: Existing catalog metadata APIs and consumers should continue to function.

## Options Considered

### Option 1: Keep Free-Form Labels and Custom Metadata Only

**Description**: Store domain and tag concepts as conventional strings inside `labels` and/or `custom_metadata`.

**Implementation**:
```json
{
  "labels": ["domain:platform", "secondary:observability", "tag:python"]
}
```

**Pros**:
- Minimal schema changes.
- Fastest short-term delivery.
- Reuses current metadata modal and PATCH endpoint.

**Cons**:
- No first-class domain/subdomain/tag objects.
- No relational validation or governance.
- Fragile query semantics and duplicate taxonomy drift.

**Estimated Effort**: S

**Cost Implications**: Low

---

### Option 2: Normalized Taxonomy Registry + Item Assignments (Chosen)

**Description**: Introduce dedicated persisted objects for domains, subdomains, and tags, plus assignment tables linking them to catalog items.

**Implementation**:
```text
registry tables:
  catalog_domains
  catalog_subdomains
  catalog_tags

assignment tables:
  catalog_item_taxonomy_assignments (primary/secondary domain + optional subdomains)
  catalog_item_tag_assignments (many-to-many tags)
```

**Pros**:
- Meets all functional requirements directly.
- Enables deterministic MCP/API filters and future policy controls.
- Keeps content read-only model intact (metadata-only mutation).
- Supports GUI global management workflow (Options menu).

**Cons**:
- Requires migration, new repositories/services, and API/tool additions.
- Adds taxonomy integrity rules (delete/rename constraints).
- Introduces additional frontend modal/state complexity.

**Estimated Effort**: L

**Cost Implications**: Low

---

### Option 3: External Taxonomy Service

**Description**: Store taxonomy in a standalone service and query it from SkillServer.

**Pros**:
- Potentially reusable across multiple products.
- Independent lifecycle and scaling.

**Cons**:
- Operationally heavy for current deployment model.
- Adds network latency, credentials, and failure modes.
- Violates no-new-managed-infra preference.

**Estimated Effort**: XL

**Cost Implications**: Medium-High

## Decision

### Chosen Option

**We will implement Option 2: Normalized taxonomy registry + item assignments.**

### Rationale

Option 2 is the only approach that satisfies the requirement for domains/subdomains/tags as true persistent objects while supporting agent search and write workflows through MCP. It aligns with ADR-004’s SQLite persistence strategy and mutability model, keeps rollout incremental, and preserves compatibility by deriving legacy `labels` from the new tag assignments where needed.

### Decision Matrix

| Criteria | Weight | Option 1 | Option 2 | Option 3 |
|----------|--------|----------|----------|----------|
| Meets object-model requirement (domains/subdomains/tags) | 5 | 1 | 5 | 5 |
| MCP filter/query quality | 5 | 2 | 5 | 5 |
| MCP write support with governance | 4 | 2 | 5 | 4 |
| Deployment/operational simplicity | 4 | 5 | 4 | 1 |
| Backward compatibility | 4 | 4 | 4 | 2 |
| **Weighted Total** |  | **51** | **90** | **67** |

## Consequences

### Positive

- Agents can search catalog items by structured taxonomy (domain/subdomain/tag), improving retrieval quality.
- Taxonomy vocabulary is centrally managed and reusable across skills and prompts.
- GUI supports both global taxonomy administration and per-item assignment from existing metadata workflows.
- Future automation (recommendation/ranking/routing) can leverage stable taxonomy IDs.

### Negative

- More schema and API surface area to maintain.
- Additional UX complexity (taxonomy manager + assignment controls).
- Requires migration/backfill strategy for existing `labels`.

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Taxonomy object sprawl or duplicates | Med | Med | Enforce unique `key` constraints, add UI validation and merge guidance. |
| Breaking existing consumers of `labels` | Low | High | Keep `labels` as additive compatibility field derived from tags during transition. |
| Slow filtered search on large catalogs | Med | Med | Add DB indexes on assignment tables and filter columns; constrain query fanout. |
| Inconsistent write behavior between GUI and MCP | Med | Med | Centralize write logic in shared domain service and gate MCP writes with one runtime flag. |

## Technical Details

### Architecture

```text
                +-----------------------------+
                | GUI (Catalog + Options UI) |
                +-------------+---------------+
                              |
                   REST /api/catalog/*
                              |
                              v
                +-----------------------------+
                | Catalog Taxonomy Services   |
                | - taxonomy registry CRUD    |
                | - item assignment service   |
                | - effective projection      |
                +-------------+---------------+
                              |
                              v
                +-----------------------------+
                | SQLite Persistence          |
                | source + overlay + taxonomy |
                +-------------+---------------+
                              |
                              v
                +-----------------------------+
                | MCP Tools                   |
                | - list/search by taxonomy   |
                | - optional write tools      |
                +-----------------------------+
```

### Data Model

Add migration `v2` in [`pkg/persistence/migrate.go`](/home/jeff/skillserver/pkg/persistence/migrate.go):

```sql
CREATE TABLE IF NOT EXISTS catalog_domains (
  domain_id TEXT PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS catalog_subdomains (
  subdomain_id TEXT PRIMARY KEY,
  domain_id TEXT NOT NULL,
  key TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(domain_id, key),
  FOREIGN KEY (domain_id) REFERENCES catalog_domains(domain_id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS catalog_tags (
  tag_id TEXT PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  color TEXT NOT NULL DEFAULT '',
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS catalog_item_taxonomy_assignments (
  item_id TEXT PRIMARY KEY,
  primary_domain_id TEXT,
  primary_subdomain_id TEXT,
  secondary_domain_id TEXT,
  secondary_subdomain_id TEXT,
  updated_at TEXT NOT NULL,
  updated_by TEXT,
  FOREIGN KEY (item_id) REFERENCES catalog_source_items(item_id) ON DELETE CASCADE,
  FOREIGN KEY (primary_domain_id) REFERENCES catalog_domains(domain_id) ON DELETE RESTRICT,
  FOREIGN KEY (primary_subdomain_id) REFERENCES catalog_subdomains(subdomain_id) ON DELETE RESTRICT,
  FOREIGN KEY (secondary_domain_id) REFERENCES catalog_domains(domain_id) ON DELETE RESTRICT,
  FOREIGN KEY (secondary_subdomain_id) REFERENCES catalog_subdomains(subdomain_id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS catalog_item_tag_assignments (
  item_id TEXT NOT NULL,
  tag_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (item_id, tag_id),
  FOREIGN KEY (item_id) REFERENCES catalog_source_items(item_id) ON DELETE CASCADE,
  FOREIGN KEY (tag_id) REFERENCES catalog_tags(tag_id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_item_taxonomy_primary_domain
ON catalog_item_taxonomy_assignments(primary_domain_id);

CREATE INDEX IF NOT EXISTS idx_item_taxonomy_secondary_domain
ON catalog_item_taxonomy_assignments(secondary_domain_id);

CREATE INDEX IF NOT EXISTS idx_item_tag_assignments_tag
ON catalog_item_tag_assignments(tag_id);
```

### Effective Projection

Extend effective catalog projection in [`pkg/domain/catalog_effective_service.go`](/home/jeff/skillserver/pkg/domain/catalog_effective_service.go) to merge:

1. Source item (`catalog_source_items`)
2. Existing metadata overlay (`catalog_metadata_overlays`)
3. Taxonomy assignment (`catalog_item_taxonomy_assignments`)
4. Tag joins (`catalog_item_tag_assignments` + `catalog_tags`)

Output model extension in [`pkg/domain/catalog.go`](/home/jeff/skillserver/pkg/domain/catalog.go):

- `primary_domain`
- `primary_subdomain`
- `secondary_domain`
- `secondary_subdomain`
- `tags` (object refs with `id`, `key`, `name`)

Backward compatibility:

- Keep existing `labels` in API/MCP output.
- Derive `labels` from assigned tag names when tag assignments exist.
- Fallback to legacy overlay labels when no tag assignments exist.

### API Changes

Extend REST API in [`pkg/web/server.go`](/home/jeff/skillserver/pkg/web/server.go) and handlers in [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go):

New registry endpoints:
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

New item-assignment endpoints:
- `GET /api/catalog/:id/taxonomy`
- `PATCH /api/catalog/:id/taxonomy`

Catalog query filters (additive) on list/search:
- `primary_domain_id`
- `secondary_domain_id`
- `subdomain_id` (matches either primary or secondary subdomain)
- `tag_ids` (comma-separated)
- `tag_match=any|all`

### MCP Changes

Extend MCP tooling in [`pkg/mcp/server.go`](/home/jeff/skillserver/pkg/mcp/server.go) and [`pkg/mcp/tools.go`](/home/jeff/skillserver/pkg/mcp/tools.go):

Read tools (always enabled):
- `list_taxonomy_domains`
- `list_taxonomy_subdomains`
- `list_taxonomy_tags`
- `get_catalog_item_taxonomy`
- `list_catalog` / `search_catalog` with taxonomy filter inputs

Write tools (enabled only when `SKILLSERVER_MCP_ENABLE_WRITES=true`):
- `create_taxonomy_domain`, `update_taxonomy_domain`, `delete_taxonomy_domain`
- `create_taxonomy_subdomain`, `update_taxonomy_subdomain`, `delete_taxonomy_subdomain`
- `create_taxonomy_tag`, `update_taxonomy_tag`, `delete_taxonomy_tag`
- `patch_catalog_item_taxonomy`

### GUI Changes

Update [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html):

- Add header `Options` button opening a taxonomy manager modal.
- Taxonomy manager modal tabs:
  - Domains
  - Subdomains
  - Tags
- Add per-item taxonomy fields to metadata editor:
  - Primary domain selector
  - Primary subdomain selector
  - Secondary domain selector
  - Secondary subdomain selector
  - Tag multi-select picker
- Render taxonomy chips in item cards:
  - Primary domain chip
  - Secondary domain chip
  - Existing tags row (from taxonomy tags)
- Add optional domain/tag filters near search controls.

### Migration Strategy

1. Apply schema migration v2.
2. Backfill tags from legacy overlay labels:
   - For each unique label, create `catalog_tags` row (`key` derived from normalized label).
   - Create `catalog_item_tag_assignments` for existing label-item relations.
3. Keep dual-read compatibility (`labels` + tags) for one release window.
4. Shift UI default to taxonomy-based tags while preserving old payload compatibility.

### Configuration

```bash
# Existing persistence gate remains required for durable taxonomy APIs.
SKILLSERVER_PERSISTENCE_DATA=true

# New MCP write gate for taxonomy/object mutation tools.
SKILLSERVER_MCP_ENABLE_WRITES=false
```

## Security Considerations

### Authentication & Authorization

- MCP taxonomy write tools are not registered unless `SKILLSERVER_MCP_ENABLE_WRITES=true`.
- REST taxonomy writes should follow the same trust boundary as existing metadata write endpoints.

### Data Protection

- Taxonomy writes validate key/name lengths and character sets.
- Delete operations are `RESTRICT` when objects are assigned, preventing accidental data loss.
- `updated_by`/`updated_at` fields remain part of assignment and overlay mutation audit trails.

### Input Validation

- Prevent circular or inconsistent assignments (for example, subdomain not belonging to selected domain).
- Enforce uniqueness on object keys.
- Normalize case and whitespace for deterministic matching.

## Rollout and Testing

### Rollout

1. Ship migration + read-path support.
2. Enable GUI taxonomy display and Options modal.
3. Enable taxonomy assignment writes (REST).
4. Enable MCP write tools only where operationally allowed.

### Test Coverage

- Persistence repository tests for new tables and constraints.
- Domain service tests for assignment validation and effective projection merge behavior.
- API handler tests for CRUD, filters, and backward compatibility.
- MCP tool tests for read/write tool registration and filter semantics.
- Playwright coverage for Options modal workflows and item-card rendering.

## References

### Internal

- [ADR-003: Unified Skill/Prompt Catalog Classification](./003-unified-skill-prompt-catalog-classification.md)
- [ADR-004: Persistent Catalog Storage and Metadata Overlays](./004-persistent-catalog-storage-and-metadata-overlays.md)
- [`pkg/domain/catalog.go`](/home/jeff/skillserver/pkg/domain/catalog.go)
- [`pkg/domain/catalog_effective_service.go`](/home/jeff/skillserver/pkg/domain/catalog_effective_service.go)
- [`pkg/persistence/migrate.go`](/home/jeff/skillserver/pkg/persistence/migrate.go)
- [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go)
- [`pkg/mcp/tools.go`](/home/jeff/skillserver/pkg/mcp/tools.go)
- [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html)

### External

- N/A
