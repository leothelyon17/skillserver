# Implementation Plan: ADR-002 Dynamic Resource and Prompt Discovery

**Date Created:** 2026-03-02
**Project Owner:** @jeff
**Target Completion:** 2026-03-10
**Actual Completion:** 2026-03-03
**Status:** COMPLETED
**Source ADR:** [ADR-002: Dynamic Imported Resource Discovery and Prompt Support](../../adrs/002-dynamic-resource-and-prompt-discovery.md)

---

## Project Overview

### Goal
Deliver import-aware resource discovery so each skill exposes all required context files, including prompts under `agents/` and `prompts/`, while enforcing strict path boundaries and preserving MCP/REST compatibility.

### Success Criteria
- [x] `ListSkillResources` discovers direct resources in `scripts/`, `references/`, `assets/`, `agents/`, and `prompts/`.
- [x] Imported file references in `SKILL.md` are surfaced as read-only resources under a stable virtual path namespace (`imports/...`).
- [x] Path traversal and symlink escape attempts are rejected for both list and read operations.
- [x] Existing MCP and REST clients continue to function without breaking changes.
- [x] Web UI can render dynamic resource groupings including `prompts` and `imported`.
- [x] Test coverage for new behavior and regressions is in place across domain, MCP, and web packages.

### Scope

**In Scope:**
- Domain resource model extensions (`prompt` type, `origin`, `writable`, virtual-path support).
- Import parsing from skill markdown references and include tokens.
- Safe resolver bounded to skill root (local) or repo root (git skills).
- Deterministic, deduplicated resource listing and imported resource virtualization.
- Additive MCP and REST response metadata for origin and writability.
- Web UI updates for dynamic resource categories and read-only imported resources.
- Test expansion, README update, and rollout/rollback guidance.

**Out of Scope:**
- New authentication/authorization model.
- Persistent external indexing/search service.
- Full recursive import-graph traversal beyond bounded depth (can be follow-up).

### Constraints
- Technical: Must keep current `SkillManager` API and existing tool/endpoint names.
- Security: No reads outside allowed boundaries; traversal and symlink escapes must be blocked.
- Compatibility: Existing resource fields (`scripts`, `references`, `assets`) remain available.
- Timeline: One implementation cycle (about one week).

---

## Public Interface and Contract Changes

### Domain Model
- Extend `pkg/domain/resources.go`:
  - New `ResourceTypePrompt` value.
  - New resource metadata fields: `Origin`, `Writable`, and virtual path mapping support.
- Keep existing fields (`type`, `path`, `name`, `size`, `mime_type`, `readable`) intact.

### REST API (`GET /api/skills/:name/resources`)
- Preserve existing keys: `scripts`, `references`, `assets`, `readOnly`.
- Add additive keys: `prompts`, `imported`, and `groups` (dynamic grouping map).
- Resource objects include `origin` and `writable` in addition to existing metadata.

### MCP Tools
- Keep tool names unchanged:
  - `list_skill_resources`
  - `read_skill_resource`
  - `get_skill_resource_info`
- Additive metadata in resource payloads:
  - `type` now includes `prompt`.
  - New fields: `origin`, `writable`.

### Optional Rollback Control
- Add runtime flag and env var for import discovery kill-switch:
  - Flag: `--enable-import-discovery`
  - Env: `SKILLSERVER_ENABLE_IMPORT_DISCOVERY`
  - Default: `true`

---

## Work Package Breakdown

### Phase 1: Resource Foundation
- [x] [WP-001: Resource Contract Extension for Prompts and Imports](./work-packages/WP-001-resource-contract-prompts-imports.md) ✅ COMPLETED (2026-03-03)
- [x] [WP-002: Import Parser and Safe Resolver](./work-packages/WP-002-import-parser-safe-resolver.md) ✅ COMPLETED (2026-03-03)

### Phase 2: Domain Integration
- [x] [WP-003: Manager Discovery Integration and Deterministic Dedupe](./work-packages/WP-003-manager-discovery-dedupe-integration.md) ✅ COMPLETED (2026-03-03)
- [x] [WP-004: Virtual Import Read and Info Resolution](./work-packages/WP-004-virtual-import-read-info-resolution.md) ✅ COMPLETED (2026-03-03)

### Phase 3: Interface Adaptation
- [x] [WP-005: MCP Resource Contract Metadata Update](./work-packages/WP-005-mcp-resource-contract-metadata.md) ✅ COMPLETED (2026-03-03)
- [x] [WP-006: REST Resource Grouping and Write Guards](./work-packages/WP-006-rest-resource-grouping-write-guards.md) ✅ COMPLETED (2026-03-03)
- [x] [WP-007: Web UI Dynamic Resource Group Rendering](./work-packages/WP-007-web-ui-dynamic-resource-groups.md) ✅ COMPLETED (2026-03-03)

### Phase 4: Validation and Rollout
- [x] [WP-008: Integration, Security, and Regression Test Matrix](./work-packages/WP-008-integration-security-regression-tests.md) ✅ COMPLETED (2026-03-03)
- [x] [WP-009: Documentation and Rollout Controls](./work-packages/WP-009-documentation-rollout-and-flag-guidance.md) ✅ COMPLETED (2026-03-03)

---

## Dependency Graph

```text
WP-001 -> WP-002 -> WP-003 -> (WP-004 || WP-005 || WP-006)
WP-006 -> WP-007
(WP-004 || WP-005 || WP-006 || WP-007) -> WP-008 -> WP-009
```

### Critical Path
`WP-001 -> WP-002 -> WP-003 -> WP-006 -> WP-007 -> WP-008 -> WP-009`

### Parallel Opportunities
After WP-003, WP-004, WP-005, and WP-006 can execute in parallel. WP-007 starts after WP-006; WP-008 starts once Phase 3 implementation is complete.

---

## Timeline and Effort

| Phase | Work Packages | Estimated Hours |
|-------|---------------|-----------------|
| Resource Foundation | WP-001, WP-002 | 9 |
| Domain Integration | WP-003, WP-004 | 8 |
| Interface Adaptation | WP-005, WP-006, WP-007 | 11 |
| Validation and Rollout | WP-008, WP-009 | 9 |
| **Total** | **9 WPs** | **37** |

### Schedule Forecast
- Critical-path effort: 31 hours.
- Aggressive (parallelized): about 5 working days at 6 productive hours/day.
- Realistic (review + rework buffer): 6-7 working days.
- Fits ADR timeline target of one cycle.

---

## Test Strategy

### Domain-Level
- Prompt directory discovery (`agents/`, `prompts/`).
- Imported reference extraction from markdown links and `@` includes.
- Path traversal rejection (`../`, absolute paths, symlink escapes).
- Deterministic ordering and dedupe behavior.

### API and MCP
- REST response includes legacy and additive categories.
- MCP output includes new type values and metadata fields.
- `read_skill_resource` and `get_skill_resource_info` work with virtual imported paths.

### UI
- Resource tab renders dynamic groups.
- Imported resources are visibly read-only in UI controls.
- Legacy skills with only 3 categories still render correctly.

---

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Path traversal via crafted imports | Medium | High | Canonicalize and enforce allowed-root prefix checks using resolved paths. |
| Symlink escape from allowed root | Medium | High | Resolve symlinks before boundary checks; reject escapes. |
| Client assumptions on fixed 3 buckets | Medium | Medium | Preserve legacy keys and add new fields additively. |
| Duplicate results from direct and imported scans | Medium | Low | Dedupe by canonical absolute path + stable virtual path normalization. |
| Performance degradation on large markdown graphs | Low | Medium | Bound depth/file count and parse only supported patterns. |
| Regression in resource writes for local skills | Medium | Medium | Add write-path guards and regression tests for create/update/delete. |

---

## Assumptions and Defaults
1. V1 import discovery parses `SKILL.md` plus bounded follow-up markdown imports (depth-limited).
2. Imported resources are always read-only in API/UI.
3. Git-backed skills remain read-only for all resource operations.
4. Direct resources under local skills keep existing write behavior.
5. Additive API/MCP metadata does not require client migration.

---

## Implementation Completion Summary

**Completion Date:** 2026-03-03  
**Status:** ✅ COMPLETED

### Overall Metrics

| Metric | Estimated | Actual | Variance |
|--------|-----------|--------|----------|
| Total Effort | 37 hours | 37 hours* | 0 hours (0%) |
| Work Packages | 9 | 9 | 0 |
| Test Runs (verification) | - | 5 suites (`pkg/domain`, `pkg/mcp`, `pkg/web`, `cmd/skillserver`, `playwright`) | - |
| Coverage Snapshot | - | `pkg/domain` 77.3% (latest recorded WP evidence) | - |

`*` Per-work-package actual-hour tracking was not captured in each completion summary; totals use plan estimates as the completion baseline.

### Work Package Summary

| WP ID | Domain | Estimated | Status | Completed |
|-------|--------|-----------|--------|-----------|
| WP-001 | Domain Layer | 4h | ✅ | 2026-03-03 |
| WP-002 | Domain Layer | 5h | ✅ | 2026-03-03 |
| WP-003 | Domain Layer | 5h | ✅ | 2026-03-03 |
| WP-004 | Domain Layer | 3h | ✅ | 2026-03-03 |
| WP-005 | MCP Layer | 3h | ✅ | 2026-03-03 |
| WP-006 | API Layer | 4h | ✅ | 2026-03-03 |
| WP-007 | Web UI | 4h | ✅ | 2026-03-03 |
| WP-008 | Quality Engineering | 6h | ✅ | 2026-03-03 |
| WP-009 | Documentation | 3h | ✅ | 2026-03-03 |

### Key Achievements

- Delivered additive prompt/import discovery while preserving legacy MCP and REST compatibility.
- Enforced safe imported-resource resolution with traversal/symlink escape protection.
- Shipped dynamic web resource grouping with read-only imported resource controls.
- Added rollout and rollback controls (`--enable-import-discovery`, `SKILLSERVER_ENABLE_IMPORT_DISCOVERY`) and operations runbook guidance.

### Common Challenges Encountered

1. **Shared test fixture contention in domain integration**
   - Description: an early WP-003 run encountered a Bleve Bolt lock when constructing managers against the same temp structure.
   - Resolution pattern: reuse manager context in tests and isolate repository configuration updates.

2. **Prompt typing across nested import virtual paths**
   - Description: plugin-style import paths required prompt classification beyond top-level directory prefixes.
   - Resolution pattern: classify `imports/...` paths as prompt when `agents/` or `prompts/` appears in normalized segments.

3. **Backward-compatible contract expansion**
   - Description: legacy clients expected fixed resource groups/fields.
   - Resolution pattern: keep legacy keys/fields intact and add `prompts`, `imported`, `groups`, `origin`, and `writable` additively.

### Lessons Learned

**What Went Well:**
- Sequencing by domain boundary reduced rework and preserved parallel execution opportunities.
- Shared resolver utilities prevented list/read/info drift.
- Regression coverage across domain/MCP/web/playwright caught edge-case behavior before rollout.

**What Could Be Improved:**
- Completion summaries should capture actual effort hours and LOC uniformly for better variance reporting.
- Completion summary format should include a consistent metrics block and follow-up debt schema.
- Work package metadata status updates should be automated as part of completion generation.

**Actionable Recommendations for Future Plans:**
1. Add mandatory per-WP metrics fields (`Actual_Effort`, `LOC_Changed`, `Tests_Added`) in completion-summary templates.
2. Add a lightweight completion linter that blocks plan-finalization when required summary fields are missing.
3. Include one standardized compatibility matrix test checklist early (not only at final validation WPs).

### Technical Debt Summary

| Priority | Count | Total Effort | Tickets Created |
|----------|-------|--------------|-----------------|
| High | 0 | 0h | None |
| Medium | 1 | 2h | None |
| Low | 1 | 2h | None |

**Tracked Debt Items:**
- Medium: add automated validation for completion-summary quality/required metrics sections.
- Low: normalize completion-summary templates across all implementation-plan folders.

### Follow-Up Items

- [ ] Add a reusable completion-summary schema validator to CI/docs tooling.
- [ ] Standardize completion-summary template fields across existing implementation plans.

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
- [WP-001](./work-packages/WP-001-resource-contract-prompts-imports.md)
- [WP-002](./work-packages/WP-002-import-parser-safe-resolver.md)
- [WP-003](./work-packages/WP-003-manager-discovery-dedupe-integration.md)
- [WP-004](./work-packages/WP-004-virtual-import-read-info-resolution.md)
- [WP-005](./work-packages/WP-005-mcp-resource-contract-metadata.md)
- [WP-006](./work-packages/WP-006-rest-resource-grouping-write-guards.md)
- [WP-007](./work-packages/WP-007-web-ui-dynamic-resource-groups.md)
- [WP-008](./work-packages/WP-008-integration-security-regression-tests.md)
- [WP-009](./work-packages/WP-009-documentation-rollout-and-flag-guidance.md)
