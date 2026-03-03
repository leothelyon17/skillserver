## WP-007: Web UI Dynamic Resource Group Rendering

### Metadata

```yaml
WP_ID: WP-007
Title: Web UI Dynamic Resource Group Rendering
Domain: Web UI
Priority: Medium
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
UI currently hard-codes three resource sections (`scripts`, `references`, `assets`). It must display additive categories and imported-read-only hints.

**Scope:**
- Render resource groups dynamically from API payload.
- Keep backward compatibility for legacy response format.
- Surface read-only badges/controls for imported resources.

Excluded:
- Backend API contract changes (WP-006).

**Success Criteria:**
- [x] UI displays prompts/imported groups when present.
- [x] Imported resources cannot be modified through UI actions.
- [x] Legacy 3-bucket responses still render correctly.

---

### Technical Requirements

**Input Contracts:**
- REST response updates from WP-006.

**Output Contracts:**
- UI updates in `pkg/web/ui/index.html` (and style updates if needed).

**Integration Points:**
- Uses existing resource view/edit/delete API calls with writability guard behavior.

---

### Deliverables

**Code Deliverables:**
- [x] Replace hard-coded resource sections with dynamic group loop.
- [x] Display resource origin metadata (`direct`/`imported`) in list items.
- [x] Disable edit/delete actions for non-writable resources.

**Test Deliverables:**
- [x] Add UI behavior checklist/manual test script in WP-008 evidence.

---

### Acceptance Criteria

**Functional:**
- [x] Dynamic groups render with graceful empty states.
- [x] Read-only imported resources are clearly indicated.
- [x] Existing script/reference/asset UX remains intact.

**Testing:**
- [x] Manual verification covers desktop and narrow viewport rendering.

---

### Dependencies

**Blocked By:**
- WP-006

**Blocks:**
- WP-008

**Parallel Execution:**
- Can run in parallel with: none (depends on API payload shape)
- Cannot run in parallel with: WP-006

---

### Risks

**Risk 1: Dynamic rendering introduces UX regressions in resource tab**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep fallback rendering path and validate against current fixtures.
