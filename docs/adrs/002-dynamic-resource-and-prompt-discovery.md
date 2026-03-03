# ADR-002: Dynamic Imported Resource Discovery and Prompt Support

## Metadata

| Field | Value |
|-------|-------|
| **Status** | Proposed |
| **Date** | 2026-03-02 |
| **Author(s)** | @jeff |
| **Reviewers** | TBD |
| **Work Package** | N/A |
| **Supersedes** | N/A |
| **Superseded By** | N/A |

## Summary

SkillServer currently discovers skill resources only from `scripts/`, `references/`, and `assets/` directories directly under each skill root. This misses prompt files found in `agents/` or `prompts/`, and cannot surface files referenced from imports that resolve outside a skill directory (common in git plugin repos). We will add import-aware dynamic resource discovery, first-class prompt resource support, and safe repo-boundary resolution while preserving existing API and MCP compatibility.

## Context

### Problem Statement

Users can discover skills from git repositories, but related resources are incomplete when files are not colocated under the skill root. The current model fails for:

- Prompt catalogs stored in `agents/` or `prompts/`.
- Imported markdown/file references that resolve to shared repo paths.
- Plugin-style repository layouts where skills live under subtrees and prompts live elsewhere.

This causes MCP and web users to read skill instructions that mention resources they cannot access via `list_skill_resources` or `read_skill_resource`.

### Current State

- Resource discovery is hardcoded to `scripts`, `references`, and `assets` in [`pkg/domain/manager.go`](/home/jeff/skillserver/pkg/domain/manager.go).
- Resource path validation only accepts those prefixes in [`pkg/domain/resources.go`](/home/jeff/skillserver/pkg/domain/resources.go).
- Web API groups resources into three fixed buckets in [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go).
- README documents only those three optional directories in [`README.md`](/home/jeff/skillserver/README.md).
- External example repo under test (`wshobson/agents`) contains plugin-level `agents/*.md` content outside individual skill directories (for example `plugins/agent-teams/agents/` and `plugins/agent-orchestration/agents/`).

### Requirements

| Requirement | Priority | Description |
|-------------|----------|-------------|
| REQ-1 | Must Have | Discover prompt resources under `agents/` and `prompts/` according to updated skill format. |
| REQ-2 | Must Have | Resolve and expose imported files referenced by a skill document when the targets exist. |
| REQ-3 | Must Have | Prevent path traversal and keep imported resolution inside approved boundaries (skill root for local skills, repo root for git skills). |
| REQ-4 | Must Have | Keep existing MCP and REST resource endpoints functional for existing clients. |
| REQ-5 | Should Have | Expose resource origin metadata (`direct` vs `imported`) and writability hints. |
| REQ-6 | Should Have | Keep resource listing deterministic and deduplicated. |
| REQ-7 | Nice to Have | Optionally recurse import parsing through imported markdown files with depth limits. |

### Constraints

- **Budget**: No new external service dependency.
- **Timeline**: One implementation cycle (about 1 week including tests and docs).
- **Technical**: Must integrate with current `SkillManager` resource APIs and MCP tools.
- **Compliance**: No read access outside configured skill/repo boundaries.
- **Team**: Maintainable Go implementation with low operational complexity.

## Decision Drivers

1. **Completeness of skill execution context**: agents must access referenced files, not only colocated files.
2. **Safety**: import resolution must be bounded and non-exploitable.
3. **Compatibility**: existing resource API clients must not break.
4. **Format evolution**: support the new `prompts` concept while preserving existing `agents` conventions in external repos.
5. **Operational simplicity**: avoid introducing full-blown dependency graphs or external indexers.

## Options Considered

### Option 1: Expand Static Directories Only

**Description**: Add `agents/` and `prompts/` to the existing static resource directory list.

**Implementation**:
```go
resourceDirs := []string{"scripts", "references", "assets", "agents", "prompts"}
```

**Pros**:
- Smallest code change.
- Fast to ship.
- Covers colocated prompt folders.

**Cons**:
- Does not solve imported/shared resources outside skill root.
- Still misses common git plugin layouts.
- No origin metadata.

**Estimated Effort**: S

**Cost Implications**: Low

---

### Option 2: Import-Aware Discovery with Prompt Types (Chosen)

**Description**: Keep static directory discovery and add dynamic import resolution from `SKILL.md` (and optionally imported markdown). Support prompt directories under `agents/` and `prompts/`. For git skills, allow import resolution within the owning repo root; for local skills, keep resolution within skill root.

**Implementation**:
```text
Pass 1: Discover direct resources in:
  scripts/, references/, assets/, agents/, prompts/

Pass 2: Parse import candidates from SKILL.md:
  - Markdown links: [x](relative/path.md)
  - Include tokens: @relative/path.md, @/abs/path (normalized)
Resolve each candidate against source file directory.
Accept only if target is inside allowed root and is a file.
Map to virtual resource paths under imports/ for stable API access.
```

**Pros**:
- Fixes missing imported resources.
- Supports both `agents/` and `prompts/` prompt storage patterns.
- Works with nested git plugin structures.
- Maintains endpoint/tool compatibility.

**Cons**:
- Higher implementation complexity.
- Requires careful path normalization and deduping.
- Web UI must handle additional category/metadata.

**Estimated Effort**: M

**Cost Implications**: Low

---

### Option 3: Require Explicit Resource Manifest in Frontmatter

**Description**: Add a required/optional frontmatter field listing all resource files (including prompts/imports), avoiding path parsing.

**Pros**:
- Deterministic and explicit.
- Simpler runtime resolver.

**Cons**:
- High authoring burden and drift risk.
- Breaks existing repositories unless backfilled.
- Not aligned with user expectation of automatic detection.

**Estimated Effort**: M (plus migration)

**Cost Implications**: Medium migration cost

## Decision

### Chosen Option

**We will implement Option 2: Import-aware discovery with prompt support.**

### Rationale

Option 2 is the only approach that satisfies both core gaps: missing imported resources and prompt discovery across real-world repo layouts. It preserves backward compatibility, requires no schema migration, and can be implemented safely with bounded path resolution and explicit metadata.

### Decision Matrix

| Criteria | Weight | Option 1 | Option 2 | Option 3 |
|----------|--------|----------|----------|----------|
| Completeness (imports + prompts) | 5 | 2 | 5 | 4 |
| Backward compatibility | 4 | 5 | 5 | 2 |
| Security control | 4 | 4 | 4 | 5 |
| Implementation risk | 3 | 5 | 3 | 3 |
| Operational simplicity | 2 | 5 | 4 | 3 |
| **Weighted Total** |  | **59** | **71** | **58** |

## Consequences

### Positive

- Skills can expose all referenced context files needed for execution.
- Prompt files under either `agents/` or `prompts/` become first-class resources.
- Git plugin repos with shared resource trees are supported without custom repo adapters.
- MCP consumers can retrieve previously inaccessible but referenced files.

### Negative

- Resource model and handlers become more complex.
- Additional metadata fields and categories must be reflected in the web UI.
- Parsing imports introduces edge cases (malformed paths, circular markdown references if recursion is enabled).

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Path traversal through crafted imports | Med | High | Canonicalize paths and enforce allowed root prefix checks before read. |
| Client assumptions on fixed 3 resource buckets | Med | Med | Keep legacy fields; add new fields in additive way (`prompts`, `imported`, `groups`). |
| Duplicate resources from direct + imported passes | Med | Low | Dedupe by canonical absolute path + stable virtual path mapping. |
| Large markdown import graphs impact performance | Low | Med | Cap recursion depth and file count; cache parse results per request. |

## Technical Details

### Architecture

```text
SkillResolver
  ├─ direct dir scan (scripts/references/assets/agents/prompts)
  ├─ SKILL.md import parser
  │    ├─ markdown links
  │    └─ include tokens (@path)
  ├─ safe path resolver (local skill root OR git repo root)
  └─ resource normalizer (type/origin/virtual path/writable)
```

### Domain Model Updates

- Extend `ResourceType` with:
  - `prompt`
- Extend `SkillResource` with:
  - `Origin` (`direct` | `imported`)
  - `Writable` (boolean)
  - `VirtualPath` (stable path returned in APIs; imported files use `imports/...`)

### Discovery Rules

1. Direct directories (if present):
   - `scripts/`, `references/`, `assets/`, `agents/`, `prompts/`
2. Import targets:
   - Parse `SKILL.md` for relative file references.
   - Resolve against current file directory.
   - Accept only file targets in allowed root boundary.
3. Type inference:
   - `agents/` or `prompts/` markdown -> `prompt`
   - `scripts/` -> `script`
   - `references/` -> `reference`
   - `assets/` -> `asset`
   - External imported markdown -> `reference` unless under prompt directories.

### API & MCP Compatibility

- Keep existing endpoints and tool names unchanged.
- REST `GET /api/skills/:name/resources`:
  - Preserve `scripts`, `references`, `assets`.
  - Add `prompts` and `imported`.
  - Add optional `groups` map for future-proof rendering.
- MCP `list_skill_resources` remains array-based; include new `type` and path/origin metadata.

### Write Operations

- Direct resources under writable local skills remain editable.
- Imported resources are read-only (`Writable=false`) even for local skills when virtualized from outside skill root.
- Git-backed skills remain fully read-only as today.

### Skill Format Update

Add prompt discovery support to skill format:

- `prompts/` (optional): Prompt markdown files.
- `agents/` (optional): Backward-compatible prompt directory name used by existing plugin ecosystems.

## Security Considerations

### Authentication & Authorization

No auth model change in this ADR; existing server-level controls remain.

### Data Protection

- Import resolution must never read outside allowed root.
- Deny symlink escape paths by resolving real paths before boundary checks.
- Keep binary-size limits and MIME handling unchanged for MCP reads.

### Audit & Compliance

- Log rejected import paths at debug level for troubleshooting.
- Add tests for traversal patterns (`../`, symlink escapes, absolute paths).

## Implementation Plan

### Phase 1: Domain Extensions

- Add new resource types/metadata in domain model.
- Expand validation to include prompt directories and virtual import paths.

### Phase 2: Resolver

- Build import parser and safe resolver in `pkg/domain`.
- Integrate into `ListSkillResources` and read/info lookups.

### Phase 3: API/MCP/UI Adaptation

- Return additive categories and metadata in REST handlers.
- Ensure MCP output includes new resource type values.
- Update web UI resource tab to render dynamic categories.

### Phase 4: Tests and Docs

- Add unit tests for prompt discovery and import resolution.
- Add git-repo fixture tests with shared `agents/` resources.
- Update README skill format and API docs.

### Rollback Plan

If regressions occur:
1. Disable import parsing via feature flag (default on after validation).
2. Fall back to direct directory scanning with prompt directories only.
3. Re-enable import parsing after fixes.

## References

### Internal

- [`pkg/domain/manager.go`](/home/jeff/skillserver/pkg/domain/manager.go)
- [`pkg/domain/resources.go`](/home/jeff/skillserver/pkg/domain/resources.go)
- [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go)
- [`README.md`](/home/jeff/skillserver/README.md)

### External

- `wshobson/agents` plugin layout examples:
  - `plugins/agent-teams/agents/`
  - `plugins/agent-orchestration/agents/`
  - `plugins/agent-teams/skills/*/SKILL.md`
