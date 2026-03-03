## WP-002: Import Parser and Safe Resolver

### Metadata

```yaml
WP_ID: WP-002
Title: Import Parser and Safe Resolver
Domain: Domain Layer
Priority: High
Estimated_Effort: 5 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-02
Started_Date: 2026-03-03
Completed_Date: 2026-03-03
```

---

### Description

**Context:**
The system currently cannot resolve imported files referenced by `SKILL.md`. This package introduces bounded import parsing and safe path resolution.

**Scope:**
- Parse supported import patterns from markdown:
  - Markdown links: `[label](relative/path.md)`
  - Include tokens: `@relative/path.md`, `@/absolute/path`
- Resolve imported targets against allowed roots:
  - local skill: skill root
  - git skill: repo root
- Reject escapes and non-file targets.

Excluded:
- Integrating output into manager list/read methods (WP-003 and WP-004).

**Success Criteria:**
- [x] Parser extracts valid candidates from fixture markdown.
- [x] Resolver accepts only files inside allowed root.
- [x] Traversal and symlink-escape attempts are denied.

---

### Technical Requirements

**Input Contracts:**
- Skill path and repo/read-only context from `FileSystemManager`.
- Domain metadata introduced in WP-001.

**Output Contracts:**
- New resolver/parser package code in `pkg/domain` (for example `resource_imports.go`).
- Deterministic virtual-path generation under `imports/...`.
- Unit tests for parser and resolver edge cases.

**Integration Points:**
- WP-003 consumes parser outputs for list discovery.
- WP-004 reuses resolver for read/info path handling.

---

### Deliverables

**Code Deliverables:**
- [x] Add parser helpers for markdown links and include tokens.
- [x] Add resolver helpers with canonical-path boundary enforcement.
- [x] Add stable imported virtual path mapper.

**Test Deliverables:**
- [x] Add parser tests for valid and malformed markdown.
- [x] Add resolver tests for local-skill and git-repo boundary rules.
- [x] Add traversal and symlink-escape negative tests.

---

### Acceptance Criteria

**Functional:**
- [x] Import candidates are extracted from `SKILL.md` with deterministic ordering.
- [x] Resolved imports are always inside allowed roots.
- [x] Invalid candidates are ignored or rejected without panics.

**Testing:**
- [x] New resolver tests cover success and security failure paths.

---

### Dependencies

**Blocked By:**
- WP-001

**Blocks:**
- WP-003
- WP-004

**Parallel Execution:**
- Can run in parallel with: none (depends on base contract)
- Cannot run in parallel with: WP-003 and WP-004

---

### Risks

**Risk 1: Overly broad pattern matching introduces false positives**
- Probability: Medium
- Impact: Low
- Mitigation: Constrain accepted syntax and extension checks.

**Risk 2: Allowed-root inference for git skills is incorrect in nested layouts**
- Probability: Medium
- Impact: High
- Mitigation: Add fixture tests for nested plugin layout and repo-root mapping.
