## WP-005: Web UI Unified Catalog Rendering and Badges

### Metadata

```yaml
WP_ID: WP-005
Title: Web UI Unified Catalog Rendering and Badges
Domain: UI Layer
Priority: High
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
The current UI fetches only `/api/skills` and shows skill-only tiles. ADR-003 requires top-level prompt visibility and tile-level classifier labels.

**Scope:**
- Switch list/search data flow to catalog endpoints.
- Add tile badge rendering for `skill` and `prompt` classifiers.
- Preserve existing behavior for editing skills and viewing read-only items.
- Ensure empty/loading/search states still behave correctly.

Excluded:
- Back-end API implementation (WP-004).
- MCP enhancements (WP-007).

**Success Criteria:**
- [ ] Mixed catalog tiles render with classifier badge.
- [ ] Search returns mixed and filtered results correctly.
- [ ] Skill editing flow remains stable.

---

### Technical Requirements

**Input Contracts:**
- Catalog API contracts from WP-004.
- Existing Alpine.js state and rendering logic in `pkg/web/ui/index.html`.

**Output Contracts:**
- UI state updated for catalog item shape.
- Badge and mixed-item rendering behavior in tile grid.

**Integration Points:**
- Depends on catalog endpoints (WP-004).
- Validated by WP-008 UI checks.

---

### Deliverables

**Code Deliverables:**
- [ ] Update load/search functions in `pkg/web/ui/index.html` to call `/api/catalog` endpoints.
- [ ] Add classifier badge markup/styles for tile cards.
- [ ] Add mixed-item click handling guardrails (prompt view vs skill edit).

**Test Deliverables:**
- [ ] Add or update UI integration/manual checklist for mixed catalog rendering.
- [ ] Add regression assertions for skill edit/create/delete flow.

---

### Acceptance Criteria

**Functional:**
- [ ] UI displays prompts as first-class tiles.
- [ ] Every tile shows a `skill` or `prompt` badge.
- [ ] No regression in existing skill management actions.

**Testing:**
- [ ] UI verification demonstrates search and tile behavior for mixed datasets.

---

### Dependencies

**Blocked By:**
- WP-004

**Blocks:**
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-006, WP-007 (after WP-004 starts)
- Cannot run in parallel with: WP-004

---

### Risks

**Risk 1: Prompt item action flow confuses users expecting editable skills**
- Probability: Medium
- Impact: Medium
- Mitigation: Add explicit classifier badge and read-only prompt interaction copy.

**Risk 2: UI state coupling causes regressions in existing editor tabs**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep editor state paths skill-specific and guard with classifier checks.
