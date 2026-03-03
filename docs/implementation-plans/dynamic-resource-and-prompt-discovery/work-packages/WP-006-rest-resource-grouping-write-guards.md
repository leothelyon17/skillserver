## WP-006: REST Resource Grouping and Write Guards

### Metadata

```yaml
WP_ID: WP-006
Title: REST Resource Grouping and Write Guards
Domain: API Layer
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
REST resource handlers currently emit three fixed buckets and do not expose imported/prompt metadata or writability hints.

**Scope:**
- Extend `listSkillResources` REST output with `prompts`, `imported`, and `groups` while preserving legacy keys.
- Include `origin` and `writable` fields in per-resource payloads.
- Add write guards in create/update/delete handlers to reject imported virtual paths.

Excluded:
- Frontend rendering changes (WP-007).

**Success Criteria:**
- [x] REST list endpoint remains backward compatible and returns additive groups.
- [x] Write operations fail cleanly for imported resources.

---

### Technical Requirements

**Input Contracts:**
- Enhanced domain metadata from WP-003/004.

**Output Contracts:**
- Handler updates in `pkg/web/handlers.go`.
- API behavior documentation updates in WP-009.

**Integration Points:**
- WP-007 consumes new response shape for UI.
- WP-008 validates REST compatibility and guard behavior.

---

### Deliverables

**Code Deliverables:**
- [x] Update response grouping logic in `listSkillResources`.
- [x] Add additive response fields (`prompts`, `imported`, `groups`, metadata fields).
- [x] Enforce write restrictions for imported virtual paths.

**Test Deliverables:**
- [x] Add REST handler tests for grouped output and write guards.

---

### Acceptance Criteria

**Functional:**
- [x] Existing keys (`scripts`, `references`, `assets`) remain present.
- [x] `prompts` and `imported` groups appear when data exists.
- [x] Imported paths cannot be updated/deleted/created via resource write handlers.

**Testing:**
- [x] REST tests cover legacy+additive payload and write rejection paths.

---

### Dependencies

**Blocked By:**
- WP-003

**Blocks:**
- WP-007
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-004, WP-005
- Cannot run in parallel with: WP-003

---

### Risks

**Risk 1: API payload drift breaks web UI assumptions**
- Probability: Medium
- Impact: Medium
- Mitigation: Preserve legacy fields and add UI fallback logic in WP-007.
