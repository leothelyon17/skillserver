## WP-008: Integration, Security, and Regression Test Matrix

### Metadata

```yaml
WP_ID: WP-008
Title: Integration, Security, and Regression Test Matrix
Domain: Quality Engineering
Priority: High
Estimated_Effort: 6 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-02
Started_Date: 2026-03-03
Completed_Date: 2026-03-03
```

---

### Description

**Context:**
ADR-002 introduces path resolution and contract changes across domain, MCP, and REST layers. A dedicated regression/security pass is required before rollout.

**Scope:**
- Add domain fixtures for git-plugin repo layouts and shared `agents/`/`prompts/` directories.
- Validate traversal and symlink-escape protections.
- Validate MCP and REST compatibility under old and new response expectations.

Excluded:
- Final documentation and release notes (WP-009).

**Success Criteria:**
- [x] Security and compatibility tests pass in CI/local runs.
- [x] Existing behavior for legacy skills is validated.
- [x] New behavior for prompt/import discovery is validated.

---

### Technical Requirements

**Input Contracts:**
- Completed domain and interface implementation from WP-004/005/006/007.

**Output Contracts:**
- Expanded tests in `pkg/domain`, `pkg/mcp`, and `pkg/web`.
- Test evidence command list and expected outcomes.

**Integration Points:**
- Required gate before WP-009 documentation finalization.

---

### Deliverables

**Test Deliverables:**
- [x] Domain tests for imported discovery, boundary checks, and dedupe.
- [x] MCP tests for additive metadata schema behavior.
- [x] REST handler tests for grouped payload + write guards.
- [x] UI manual verification checklist and outcomes.

**Validation Commands (minimum):**
- [x] `go test ./pkg/domain -count=1`
- [x] `go test ./pkg/mcp -count=1`
- [x] `go test ./pkg/web -count=1`

---

### Acceptance Criteria

**Functional:**
- [x] Security checks block traversal and symlink escapes.
- [x] Legacy skills with only direct resources still pass all tests.

**Testing:**
- [x] All new and regression tests pass.
- [x] No flaky tests introduced in touched packages.

---

### Dependencies

**Blocked By:**
- WP-004
- WP-005
- WP-006
- WP-007

**Blocks:**
- WP-009

**Parallel Execution:**
- Can run in parallel with: none (validation gate)
- Cannot run in parallel with: WP-009

---

### Risks

**Risk 1: Insufficient fixture realism misses plugin-layout edge cases**
- Probability: Medium
- Impact: High
- Mitigation: Include fixtures mirroring ADR external references (`plugins/*/agents/`).

**Risk 2: Test matrix growth increases execution time**
- Probability: Low
- Impact: Medium
- Mitigation: Keep fixture sizes small and use targeted package tests.

---

### Execution Evidence (2026-03-03)

**Domain Regression Coverage**
- `pkg/domain/resource_imports_test.go`
  - Traversal rejection (`../`), invalid candidate rejection, and symlink-escape rejection.
  - Import resolution bounded by skill root (local) and repo root (git-backed).
- `pkg/domain/resources_test.go`
  - Deterministic dedupe between direct/imported paths.
  - Git plugin fixture validation (`plugins/.../skills/...`) with shared `agents/` and repo-level `prompts/` imports.
  - Imported prompt classification for nested plugin virtual paths under `imports/plugins/.../agents/...`.

**MCP Regression Coverage**
- `pkg/mcp/server_stdio_regression_test.go`
  - Confirms legacy stdio tool set remains available.
  - Verifies additive metadata (`origin`, `writable`) without removing legacy fields.
  - Verifies imported prompt info lookups and missing-resource behavior (`exists=false`).

**REST Regression Coverage**
- `pkg/web/handlers_resource_grouping_test.go`
  - Confirms legacy-only skill payload remains backward-compatible (`scripts`, `references`, `assets`, `groups`).
  - Confirms additive groups (`prompts`, `imported`) and per-resource metadata are returned when present.
  - Confirms imported write guards reject create/update/delete operations.

**UI Manual Verification**
- Checklist and outcomes are tracked in:
  - `docs/implementation-plans/dynamic-resource-and-prompt-discovery/work-packages/WP-008-ui-manual-verification-checklist.md`

**Validation Command Outcomes**
- `go test ./pkg/domain -count=1` → `ok`
- `go test ./pkg/mcp -count=1` → `ok`
- `go test ./pkg/web -count=1` → `ok`
- `npm run test:playwright` → `3 passed`
