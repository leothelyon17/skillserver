## WP-003: Rule Classifier and Install Metadata

### Metadata

```yaml
WP_ID: WP-003
Title: Rule Classifier and Install Metadata
Domain: Domain Layer
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-08
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Rules are not yet first-class catalog objects. ADR-007 requires explicit `rule` classification plus install metadata so rules can materialize into project-root files such as `AGENTS.md`.

**Scope:**
- Add `CatalogClassifierRule` and deterministic rule catalog ID/key helpers.
- Add rule detection helpers for configured rule directories and project-rule filename allowlist.
- Parse optional `materialize.target_path` and `materialize.conflict_policy` frontmatter from file-backed catalog items.
- Validate materialize target paths as relative, traversal-free paths.

Excluded:
- Runtime flag parsing (WP-004).
- Persistence migration (WP-005).
- Catalog builder, search, and sync integration (WP-006).

**Success Criteria:**
- [ ] Rule candidates are detected only from explicit directories or filename allowlists.
- [ ] Install metadata can be parsed without changing existing prompt/skill frontmatter behavior.
- [ ] Invalid target paths are rejected before any write-capable flow uses them.

---

### Technical Requirements

**Input Contracts:**
- Existing catalog classifier helpers in `pkg/domain/catalog.go`.
- Existing prompt frontmatter parsing patterns in `pkg/domain/manager_catalog.go`.

**Output Contracts:**
- Extended classifier and helper APIs in `pkg/domain/`.
- Tests for rule detection, allowlist handling, and install metadata parsing.

**Integration Points:**
- WP-004 runtime config feeds rule directory and filename allowlists.
- WP-006 uses classifier helpers during catalog synthesis and search indexing.
- WP-007 uses materialize metadata during target-path planning.

---

### Deliverables

**Code Deliverables:**
- [ ] Extend catalog classifier model with `rule`.
- [ ] Add rule directory and filename detection helpers in `pkg/domain/`.
- [ ] Add install-metadata parsing and validation helpers for `target_path` and `conflict_policy`.

**Test Deliverables:**
- [ ] Add domain tests for rule candidate detection from direct and imported paths.
- [ ] Add tests for project-root rule files (`AGENTS.md`, `RULES.md`, `CLAUDE.md`, `GEMINI.md`).
- [ ] Add tests for invalid materialize target paths and unsupported conflict policies.

---

### Acceptance Criteria

**Functional:**
- [ ] `rule` is a valid classifier in domain helpers.
- [ ] Non-markdown or non-allowlisted markdown files do not classify as rules.
- [ ] Valid frontmatter metadata is parsed without breaking existing prompt metadata extraction.

**Testing:**
- [ ] Domain tests cover happy path, imported path handling, invalid frontmatter, and deterministic ID generation.

---

### Dependencies

**Blocked By:**
- None

**Blocks:**
- WP-004
- WP-005
- WP-006
- WP-007
- WP-008
- WP-009
- WP-010
- WP-011

**Parallel Execution:**
- Can run in parallel with: WP-001
- Cannot run in parallel with: WP-004, WP-005, WP-006

---

### Risks

**Risk 1: Detection rules classify ordinary documentation as rules**
- Probability: Medium
- Impact: Medium
- Mitigation: Restrict detection to configured directory segments and explicit filename allowlists.

**Risk 2: Install metadata parsing conflicts with existing frontmatter behavior**
- Probability: Low
- Impact: Medium
- Mitigation: Keep parsing additive and add regression cases for existing prompt metadata documents.
