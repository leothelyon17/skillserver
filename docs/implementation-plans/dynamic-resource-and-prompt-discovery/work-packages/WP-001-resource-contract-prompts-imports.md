## WP-001: Resource Contract Extension for Prompts and Imports

### Metadata

```yaml
WP_ID: WP-001
Title: Resource Contract Extension for Prompts and Imports
Domain: Domain Layer
Priority: High
Estimated_Effort: 4 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-02
Started_Date: 2026-03-03
Completed_Date: 2026-03-03
```

---

### Description

**Context:**
Current domain resource contracts only support `script`, `reference`, and `asset`, and cannot represent imported resource origin/writability.

**Scope:**
- Extend resource types and metadata to model prompts and imported resources.
- Update path/type helpers needed by manager, MCP, and web handlers.

Excluded:
- Import parsing and file-resolution logic (WP-002).
- Manager integration with live discovery (WP-003).

**Success Criteria:**
- [x] Domain contracts represent prompt and imported resource metadata.
- [x] Legacy type behavior remains unchanged for existing directories.
- [x] Validation rules are explicit and test-covered.

---

### Technical Requirements

**Input Contracts:**
- Existing resource model in `pkg/domain/resources.go`.

**Output Contracts:**
- Updated `ResourceType` and `SkillResource` metadata fields.
- Updated helper functions for path and type validation.
- New/updated unit tests in `pkg/domain/resources_test.go`.

**Integration Points:**
- WP-002/003 consume new type and metadata model.
- WP-005/006 expose additive fields externally.

---

### Deliverables

**Code Deliverables:**
- [x] Add `ResourceTypePrompt` in `pkg/domain/resources.go`.
- [x] Add `Origin` and `Writable` fields to `SkillResource`.
- [x] Extend type inference for `agents/` and `prompts/`.
- [x] Extend path validation helpers for virtual imported paths.

**Test Deliverables:**
- [x] Update `pkg/domain/resources_test.go` for prompt type and new validation scenarios.

---

### Acceptance Criteria

**Functional:**
- [x] `GetResourceType("agents/foo.md")` and `GetResourceType("prompts/foo.md")` resolve to `prompt`.
- [x] Legacy prefixes (`scripts/`, `references/`, `assets/`) keep existing behavior.
- [x] Invalid traversal and absolute paths are still rejected.

**Testing:**
- [x] Domain tests pass with new cases for type/validation behavior.

---

### Dependencies

**Blocked By:**
- None

**Blocks:**
- WP-002
- WP-003

**Parallel Execution:**
- Can run in parallel with: none (foundation package)
- Cannot run in parallel with: WP-002 and WP-003

---

### Risks

**Risk 1: Breaking existing path validation contracts**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep legacy validation tests and add targeted additive cases.

**Risk 2: Ambiguous type mapping for imported markdown**
- Probability: Low
- Impact: Medium
- Mitigation: Keep clear default (`reference`) unless under prompt directories.
