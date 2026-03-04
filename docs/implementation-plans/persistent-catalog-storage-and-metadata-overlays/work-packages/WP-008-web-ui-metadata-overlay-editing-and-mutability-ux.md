## WP-008: Web UI Metadata Overlay Editing and Mutability UX

### Metadata

```yaml
WP_ID: WP-008
Title: Web UI Metadata Overlay Editing and Mutability UX
Domain: UI Layer
Priority: Medium
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Operators need to edit metadata on all catalog items, including Git-backed items, without unlocking content editing on immutable sources.

**Scope:**
- Add metadata edit controls for catalog items in UI.
- Use new metadata API endpoints for read/update operations.
- Respect mutability flags for content vs metadata actions.

Excluded:
- Backend API implementation (WP-006).
- Runtime synchronization internals (WP-007).

**Success Criteria:**
- [ ] Metadata can be edited and saved for Git and non-Git catalog items.
- [ ] Content edit/delete affordances remain disabled for non-content-writable items.
- [ ] UI reflects persisted metadata after reload/search.

---

### Technical Requirements

**Input Contracts:**
- Additive API contract and mutability fields from WP-006.

**Output Contracts:**
- UI updates in `pkg/web/ui/index.html` (and related assets if needed).
- Frontend API helpers for metadata fetch and patch calls.

**Integration Points:**
- WP-009 validates UI behavior with automated coverage.
- WP-010 documents UI behavior changes for operators.

---

### Deliverables

**Code Deliverables:**
- [ ] Add metadata edit panel/modal bound to catalog item selection.
- [ ] Add client methods for:
  - loading effective metadata
  - patching overlay metadata
- [ ] Gate actions using `content_writable` and `metadata_writable`.
- [ ] Preserve existing read-only notices and content editing guards.

**Test Deliverables:**
- [ ] UI/component tests for mutability gating.
- [ ] End-to-end test for metadata edit and persistence across reload.

---

### Acceptance Criteria

**Functional:**
- [ ] Git-backed catalog item allows metadata edits but blocks content edits.
- [ ] Local/file-import item allows both content and metadata edits.
- [ ] Metadata edits appear in list/search cards where applicable.

**Testing:**
- [ ] UI tests cover successful save, validation errors, and disabled actions.
- [ ] E2E test confirms metadata survives page reload and search flows.

---

### Dependencies

**Blocked By:**
- WP-006

**Blocks:**
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-007
- Cannot run in parallel with: WP-009

---

### Risks

**Risk 1: UI action gating mismatch with backend contract**
- Probability: Medium
- Impact: Medium
- Mitigation: Use API-provided mutability fields as single source of truth.

**Risk 2: Prompt/skill mixed item rendering regressions**
- Probability: Medium
- Impact: Medium
- Mitigation: Add targeted UI regression cases for both classifiers.
