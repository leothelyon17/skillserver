## WP-003: Manager Discovery Integration and Deterministic Dedupe

### Metadata

```yaml
WP_ID: WP-003
Title: Manager Discovery Integration and Deterministic Dedupe
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
`ListSkillResources` currently scans three static directories only and cannot surface imported resources.

**Scope:**
- Extend direct directory scan list to include `agents/` and `prompts/`.
- Merge import parser output with direct resources.
- Deduplicate and sort resources deterministically.
- Populate `origin`, `writable`, and type metadata correctly.

Excluded:
- Read/info resolution for imported virtual paths (WP-004).

**Success Criteria:**
- [x] Resource list includes direct and imported entries when applicable.
- [x] Prompt files appear with `type=prompt`.
- [x] Duplicate files appear only once in output.

---

### Technical Requirements

**Input Contracts:**
- Import parser and safe resolver from WP-002.
- Extended domain contract from WP-001.

**Output Contracts:**
- `ListSkillResources` integration changes in `pkg/domain/manager.go`.
- Deterministic ordering and dedupe behavior.

**Integration Points:**
- WP-005 and WP-006 consume enhanced list payload.
- WP-008 validates behavior end-to-end.

---

### Deliverables

**Code Deliverables:**
- [x] Update `resourceDirs` list in manager discovery logic.
- [x] Integrate parser-based imported resource discovery into `ListSkillResources`.
- [x] Add canonical dedupe and stable sort.

**Test Deliverables:**
- [x] Extend `pkg/domain/resources_test.go` with direct+imported merge scenarios.
- [x] Add deterministic order assertions.

---

### Acceptance Criteria

**Functional:**
- [x] `ListSkillResources` returns expected resources for fixture skills using direct and imported files.
- [x] Output is deterministic across repeated calls.
- [x] Imported resources are marked read-only.

**Testing:**
- [x] Domain tests include prompt, imported, and dedupe regressions.

---

### Dependencies

**Blocked By:**
- WP-001
- WP-002

**Blocks:**
- WP-004
- WP-005
- WP-006

**Parallel Execution:**
- Can run in parallel with: none (core integration package)
- Cannot run in parallel with: WP-004, WP-005, WP-006

---

### Risks

**Risk 1: Dedupe logic hides intentionally distinct paths**
- Probability: Low
- Impact: Medium
- Mitigation: Dedupe by canonical absolute target while preserving primary virtual path strategy.

**Risk 2: Performance regression from per-file reads during listing**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep scan bounded and avoid unnecessary full-file reads where metadata suffices.
