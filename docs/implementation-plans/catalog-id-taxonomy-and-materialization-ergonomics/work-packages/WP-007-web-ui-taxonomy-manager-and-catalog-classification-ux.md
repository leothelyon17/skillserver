## WP-007: Web UI Taxonomy Manager and Catalog Classification UX

### Metadata

```yaml
WP_ID: WP-007
Title: Web UI Taxonomy Manager and Catalog Classification UX
Domain: frontend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-software-developer-system-prompt.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for UI state, interaction design, and end-to-end web behavior.
Priority: Medium
Estimated_Effort: 5 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-09
Started_Date: 2026-03-09
Completed_Date: 2026-03-09
```

---

### Description

**Context:**
The taxonomy manager and catalog cards already expose taxonomy data, but classification gaps are hard to spot, delete flows do not show usage counts up front, and the current list/search views assume content-heavy payloads.

**Scope:**
- Show explicit unclassified/partially classified state in the catalog grid.
- Add filter controls for:
  - unclassified items
  - missing primary domain
  - missing tags
- Display taxonomy usage/preflight data in manager delete flows.
- Update metadata editing so additive tag mutation can be used without fragile full replacement.
- Adjust list/search loading to work with paginated metadata-first responses.

Excluded:
- REST handler implementation (WP-005).
- MCP transport updates (WP-006).

**Success Criteria:**
- [x] Users can identify unclassified items without opening the metadata modal.
- [x] Delete flows show usage counts and impacted items before confirmation.
- [x] The UI remains functional when list/search responses omit `content`.

---

### Technical Requirements

**Input Contracts:**
- REST filters and explicit completeness fields from WP-005.
- Existing Alpine UI behavior in `pkg/web/ui/index.html`.

**Output Contracts:**
- Updated UI state, actions, and presentation for classification-state and usage data.
- Updated Playwright scenarios covering the new flows.

**Integration Points:**
- WP-008 validates the final UI behavior.
- WP-009 documents the user-facing behavior changes.

---

### Deliverables

**Code Deliverables:**
- [x] Add unclassified and partial-classification badges or chips in catalog cards.
- [x] Add classification-state filter controls in the catalog filter panel.
- [x] Add taxonomy usage/preflight rendering in manager delete confirmation flows.
- [x] Update metadata editor save behavior to use additive tag mutation when appropriate.
- [x] Add pagination or incremental loading behavior for catalog list/search.

**Test Deliverables:**
- [x] Update Playwright tests for taxonomy manager delete-preflight behavior.
- [x] Update Playwright tests for unclassified-item filtering and badges.
- [x] Verify content preview still works through dedicated content-read flows.

---

### Acceptance Criteria

**Functional:**
- [x] Unclassified and partially classified items are visible in the catalog grid.
- [x] Usage counts and impacted-item previews are visible before taxonomy deletes.
- [x] The UI does not rely on inline `content` in list/search payloads.
- [x] Pagination or incremental loading is usable and deterministic.

**Testing:**
- [x] Playwright coverage exercises classification-state and usage-preflight UX.
- [x] Existing taxonomy-manager CRUD flows remain intact.

---

### Dependencies

**Blocked By:**
- WP-005

**Blocks:**
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-006 once REST contracts are stable.
- Cannot run in parallel with: WP-005.

---

### Risks

**Risk 1: Additional classification chips clutter the card layout**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep state badges concise and prioritize the most actionable gaps.

**Risk 2: Pagination reduces discoverability in the catalog grid**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep default limits reasonable and make navigation clear in the UI.
