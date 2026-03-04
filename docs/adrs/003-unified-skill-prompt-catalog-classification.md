# ADR-003: Unified Skill/Prompt Catalog Classification for Git Imports

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

SkillServer already discovers prompt-like resources inside skills, but prompt files imported from Git repositories are not first-class catalog objects in the main UI and are not indexed/searchable the same way as skills. We will introduce a unified catalog item model with an explicit classifier (`skill` vs `prompt`), detect prompt markdown files in prompt directories during Git import indexing, and expose these items in the GUI and search index. This provides consistent discovery, search filtering, and tile-level labeling without breaking existing skill CRUD flows.

## Context

### Problem Statement

Users need prompt files from imported Git repositories to appear in the main GUI and be searchable like skills. Today, prompt files are treated as resources attached to a skill, not as searchable catalog items. As a result:

- Prompt files are hard to discover unless a user opens a specific skill and browses resources.
- Search is limited to skill documents and metadata, excluding prompt markdown content as top-level hits.
- There is no item-level classifier that can be used to query the search database/index for `skill` or `prompt`.

### Current State

- Resource discovery supports prompt resource types in the domain layer (`agents/`, `prompts/`) and imported virtual resources (`imports/...`) in [`pkg/domain/manager.go`](/home/jeff/skillserver/pkg/domain/manager.go).
- Search indexing only stores skill documents (`name`, `content`, metadata) in [`pkg/domain/search.go`](/home/jeff/skillserver/pkg/domain/search.go).
- REST list/search endpoints are skill-centric in [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go):
  - `GET /api/skills`
  - `GET /api/skills/search?q=...`
- The GUI tile grid and search bar are skill-only in [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html).
- Existing ADR-002 introduced prompt/imported resource discovery but did not define first-class prompt catalog entries or a global object classifier for skill-vs-prompt queries.

### Requirements

| Requirement | Priority | Description |
|-------------|----------|-------------|
| REQ-1 | Must Have | Identify prompt files as markdown resources located under directories named `agent`, `prompt`, or `prompts` when content is imported via Git (and preserve compatibility for existing `agents` paths). |
| REQ-2 | Must Have | Expose prompt files as top-level GUI catalog tiles, not only as nested resources. |
| REQ-3 | Must Have | Add an explicit object classifier (`skill` vs `prompt`) attached to every catalog item. |
| REQ-4 | Must Have | Make the classifier queryable in the search database/index so users can filter by item type. |
| REQ-5 | Should Have | Display a small tile label in GUI: `skill` or `prompt`. |
| REQ-6 | Should Have | Preserve backward compatibility for existing skill CRUD and resource endpoints. |
| REQ-7 | Nice to Have | Provide MCP parity for catalog search/list with classifier filters. |

### Constraints

- **Budget**: No new external search/database service.
- **Timeline**: One delivery cycle (about 1 week including tests and docs).
- **Technical**: Must integrate with current file-system + Bleve index architecture.
- **Compliance**: Keep existing path-safety boundaries for imported resources.
- **Team**: Low operational overhead; implementation should remain maintainable in Go.

## Decision Drivers

1. **Discoverability**: Prompt content from imported Git repos must be visible without navigating resource tabs.
2. **Search Parity**: Prompt files should be searchable with the same UX expectations as skills.
3. **Explicit Typing**: A durable classifier is required for filterable queries and GUI labeling.
4. **Backward Compatibility**: Existing `/api/skills` and skill editing flows must remain stable.
5. **Performance**: Search/filter operations should use indexed fields, not expensive path-time scans.

## Options Considered

### Option 1: Keep Skill-Only Catalog and Infer Prompt Type at Query Time

**Description**: Continue indexing only skills. Detect prompts ad hoc by scanning resources and inferring from path names each time a user searches or opens the UI.

**Implementation**:
```text
Search flow:
  1) Query skills index
  2) For each skill hit, scan resources
  3) Infer prompt-ness from path segment names
```

**Pros**:
- Minimal schema/index changes.
- Reuses existing skill endpoints.
- Fastest initial implementation.

**Cons**:
- Prompts are not first-class search/index documents.
- Classifier is not persisted in the index database.
- Higher runtime cost and inconsistent ranking/snippets.

**Estimated Effort**: S

**Cost Implications**: Low

---

### Option 2: Unified Catalog with Explicit `skill|prompt` Classifier (Chosen)

**Description**: Build a unified catalog index containing both skills and prompt files as first-class documents. Add a required classifier field (`skill`, `prompt`) to each catalog item and expose catalog list/search APIs used by the GUI.

**Implementation**:
```text
Catalog rebuild:
  - Enumerate skills
  - Emit skill catalog item (classifier=skill)
  - Enumerate prompt candidates from skill resources
    - markdown files under: agent/, agents/, prompt/, prompts/
  - Emit prompt catalog items (classifier=prompt)
  - Index all items in Bleve with classifier field
```

**Pros**:
- Meets discoverability and search parity requirements.
- Enables direct index filtering by classifier.
- Supports GUI tile badge (`skill` / `prompt`) with stable data contract.
- Cleanly extends existing architecture without external dependencies.

**Cons**:
- Requires index schema and API additions.
- Increases indexed document count.
- Requires UI tile behavior updates for prompt items.

**Estimated Effort**: M

**Cost Implications**: Low

---

### Option 3: Separate Prompt-Specific Index and Separate GUI Section

**Description**: Create a second prompt index and separate prompt page/panel in the UI, leaving skill list/search untouched.

**Pros**:
- Limits blast radius to prompt functionality.
- Keeps existing skill flows unchanged.

**Cons**:
- Fragmented UX (not searchable in the same flow as skills).
- Duplicated indexing/search logic.
- Contradicts requirement to surface prompts same as skills.

**Estimated Effort**: M

**Cost Implications**: Low-Medium (maintenance overhead)

## Decision

### Chosen Option

**We will implement Option 2: Unified catalog with explicit `skill|prompt` classifier.**

### Rationale

Option 2 is the only option that fully satisfies all must-have requirements: first-class prompt visibility, shared search experience, and persistent type classification in the search database/index. It preserves backward compatibility by introducing additive catalog APIs while keeping skill CRUD endpoints intact. The additional complexity is acceptable and bounded within current domain/search/UI layers.

### Decision Matrix

| Criteria | Weight | Option 1 | Option 2 | Option 3 |
|----------|--------|----------|----------|----------|
| Prompt discoverability in main GUI | 5 | 2 | 5 | 3 |
| Search parity with skills | 5 | 2 | 5 | 2 |
| Classifier queryability in index | 4 | 1 | 5 | 4 |
| Backward compatibility | 4 | 5 | 4 | 5 |
| Implementation complexity/risk | 3 | 5 | 3 | 3 |
| **Weighted Total** |  | **56** | **81** | **61** |

## Consequences

### Positive

- Prompt markdown files from Git imports become visible as first-class catalog entries.
- Search can return both skill and prompt results with classifier filters.
- UI tiles can consistently show type badges (`skill`, `prompt`).
- Prompt content becomes easier to discover and reuse across repositories.

### Negative

- Index rebuild logic becomes more complex due to additional catalog item generation.
- More items in index may increase rebuild time and index size.
- UI must handle mixed item actions (editing skills vs viewing prompt resources).

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| False prompt classification from path naming edge cases | Med | Med | Restrict detection to markdown files and configurable directory segment allowlist. |
| Duplicate catalog entries for the same prompt | Med | Low | Use stable prompt IDs and dedupe by canonical `(skillID, resourcePath)` key. |
| Existing clients depend on skill-only responses | Low | Med | Keep `/api/skills` behavior unchanged; add new additive catalog endpoints. |
| Index rebuild latency grows for large repos | Med | Med | Batch indexing and document count limits; measure and tune during rollout. |

## Technical Details

### Architecture

```text
Git Sync / Local Skills
        |
        v
FileSystemManager (skills + resources)
        |
        v
Catalog Builder
  - emit classifier=skill items
  - emit classifier=prompt items
        |
        v
Bleve Index (.index)
  fields: id, classifier, name, description, content, parent_skill_id, resource_path
        |
        +--> GET /api/catalog
        +--> GET /api/catalog/search?q=...&classifier=skill|prompt
        +--> GUI tiles + search bar + badge label
```

### AWS Services Involved

| Service | Purpose | Configuration |
|---------|---------|---------------|
| N/A | Local/self-hosted service architecture | N/A |

### Database Changes

No SQL schema changes.

Bleve document schema will be extended with classifier-aware catalog fields:

```json
{
  "id": "prompt:repo/skill-name:imports/prompts/system.md",
  "classifier": "prompt",
  "name": "system.md",
  "description": "Derived from prompt content/title",
  "content": "full prompt markdown",
  "parent_skill_id": "repo/skill-name",
  "resource_path": "imports/prompts/system.md",
  "read_only": true
}
```

**Migration Strategy**: Full index rebuild using existing `RebuildIndex` flow.

### API Changes

**New Endpoints**:

- `GET /api/catalog` - List catalog items (skills and prompts).
- `GET /api/catalog/search?q={query}&classifier={skill|prompt}` - Search catalog with optional classifier filter.

**Additive Response Field**:

- `classifier` on catalog item responses with values `skill` or `prompt`.

**Backward Compatibility**:

- Existing skill endpoints remain skill-only and continue supporting CRUD:
  - `GET /api/skills`
  - `GET /api/skills/search`
  - `GET/POST/PUT/DELETE /api/skills/...`

### Prompt Detection Rules

1. Candidate must be a markdown file (`.md`, optionally `.markdown` if enabled).
2. Candidate path must include a directory segment matching prompt roots:
   - `agent`
   - `prompt`
   - `prompts`
   - `agents` (backward compatibility)
3. Applies to direct resources and imported virtual resources under `imports/...`.
4. `SKILL.md` is always classified as `skill`, never `prompt`.

### Configuration

```yaml
# Proposed additive config
SKILLSERVER_CATALOG_ENABLE_PROMPTS: "true"
SKILLSERVER_CATALOG_PROMPT_DIRS: "agent,agents,prompt,prompts"
```

## Security Considerations

### Authentication & Authorization

No auth model change. Existing server and route access controls continue to apply.

### Data Protection

- Prompt item generation must reuse existing path boundary checks for imported resources.
- No additional file read permissions beyond current resource discovery boundaries.
- Read-only semantics for Git-backed prompt files are preserved in catalog responses.

## Implementation Plan

### Phase 1: Domain and Index Model

- Add unified catalog item model with `classifier` enum (`skill`, `prompt`).
- Extend search index writer/reader to index and query classifier.
- Add deterministic prompt item IDs and deduping.

### Phase 2: Prompt Candidate Extraction

- Implement prompt-path classifier helper for `agent|prompt|prompts|agents`.
- Generate prompt catalog items from discovered skill resources (direct + imported).
- Ensure Git import sync triggers full catalog rebuild through existing callbacks.

### Phase 3: API and GUI

- Add `/api/catalog` and `/api/catalog/search` handlers.
- Update GUI tile list/search to consume catalog endpoints.
- Add small tile badge for `skill` or `prompt`.

### Phase 4: Tests and Docs

- Unit tests for classifier rules and path edge cases.
- Integration tests for Git-imported prompt visibility in catalog/search.
- UI tests verifying mixed search and badge rendering.
- README/API docs update for catalog endpoints and classifier filter.

### Rollback Plan

1. Disable prompt catalog indexing via feature flag.
2. Rebuild index with skills-only documents.
3. Revert GUI list/search source back to `/api/skills`.

## Success Metrics

- Prompt files under Git-imported `agent/prompt/prompts` paths appear in the main GUI within one sync cycle.
- Search results include both skill and prompt entries with correct classifier labels.
- Classifier filter queries return correct subsets from index-backed search.
- No regression in existing skill CRUD behavior and resource safety checks.

## References

### Internal

- [`docs/adrs/002-dynamic-resource-and-prompt-discovery.md`](/home/jeff/skillserver/docs/adrs/002-dynamic-resource-and-prompt-discovery.md)
- [`pkg/domain/manager.go`](/home/jeff/skillserver/pkg/domain/manager.go)
- [`pkg/domain/search.go`](/home/jeff/skillserver/pkg/domain/search.go)
- [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go)
- [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html)
- [`pkg/git/syncer.go`](/home/jeff/skillserver/pkg/git/syncer.go)

