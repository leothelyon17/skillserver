## WP-004: Virtual Import Read and Info Resolution

### Metadata

```yaml
WP_ID: WP-004
Title: Virtual Import Read and Info Resolution
Domain: Domain Layer
Priority: High
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-02
Started_Date: 2026-03-03
Completed_Date: 2026-03-03
```

---

### Description

**Context:**
Imported resources need stable virtual paths for list responses and must be readable through existing APIs without exposing unsafe filesystem access.

**Scope:**
- Resolve virtual imported paths (`imports/...`) to canonical file targets.
- Apply boundary checks for read/info operations.
- Keep direct path behavior unchanged.

Excluded:
- External contract/handler updates (WP-005 and WP-006).

**Success Criteria:**
- [x] `ReadSkillResource` works for imported virtual paths.
- [x] `GetSkillResourceInfo` works for imported virtual paths.
- [x] Unsafe virtual paths are rejected.

---

### Technical Requirements

**Input Contracts:**
- Discovery and mapping outputs from WP-003.

**Output Contracts:**
- `ReadSkillResource` and `GetSkillResourceInfo` updates in `pkg/domain/manager.go`.
- Reusable path resolution helper for list/read/info consistency.

**Integration Points:**
- WP-005 MCP read/info behavior.
- WP-006 REST read/info and write guards.

---

### Deliverables

**Code Deliverables:**
- [x] Add shared path-resolution function for direct and virtual imported resources.
- [x] Update manager read/info methods to use shared resolver.

**Test Deliverables:**
- [x] Add positive and negative read/info tests for imported virtual paths.

---

### Acceptance Criteria

**Functional:**
- [x] Imported resources can be read via existing read API without changing endpoint/tool names.
- [x] Non-existent and escaped virtual paths return errors.

**Testing:**
- [x] Read/info regression tests pass for direct and imported resources.

---

### Dependencies

**Blocked By:**
- WP-003

**Blocks:**
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-005, WP-006
- Cannot run in parallel with: WP-003

---

### Risks

**Risk 1: Resolver mismatch between list and read phases**
- Probability: Medium
- Impact: Medium
- Mitigation: Centralize resolver logic and reuse across code paths.
