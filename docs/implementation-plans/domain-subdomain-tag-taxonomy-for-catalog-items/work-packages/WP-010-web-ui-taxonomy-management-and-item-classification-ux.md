## WP-010: Web UI Taxonomy Management and Item Classification UX

### Metadata

```yaml
WP_ID: WP-010
Title: Web UI Taxonomy Management and Item Classification UX
Domain: UI Layer
Priority: High
Estimated_Effort: 6 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-04
Started_Date: 2026-03-05
Completed_Date: 2026-03-05
```

---

### Description

**Context:**
ADR-005 requires global taxonomy administration and item-level taxonomy assignment in the web interface, plus visible taxonomy context in catalog cards.

**Scope:**
- Add Options entry for taxonomy manager modal with tabs for domains, subdomains, and tags.
- Extend metadata editor with taxonomy assignment fields and tag multi-select.
- Render primary/secondary domain and tag chips in catalog cards.
- Add optional taxonomy filters near catalog search controls.

Excluded:
- Backend API/MCP handler implementation.

**Success Criteria:**
- [x] Users can CRUD taxonomy objects from the UI.
- [x] Users can assign taxonomy to catalog items and persist changes.
- [x] Catalog cards and filters reflect assigned taxonomy correctly.

---

### Technical Requirements

**Input Contracts:**
- Registry APIs from WP-006.
- Item taxonomy APIs and list/search filters from WP-007.

**Output Contracts:**
- UI state/model updates in `pkg/web/ui/index.html`.
- User-facing validation and conflict handling messaging.

**Integration Points:**
- WP-011 validates UI taxonomy workflows in regression suite.

---

### Deliverables

**Code Deliverables:**
- [x] Add taxonomy manager modal, tabs, and CRUD flows.
- [x] Extend metadata modal with primary/secondary domain and subdomain selectors.
- [x] Add tag picker with multi-select assignment support.
- [x] Add card taxonomy chips and catalog taxonomy filter controls.

**Test Deliverables:**
- [x] UI integration/manual checklist for taxonomy manager and item assignment.
- [x] Regression coverage for existing catalog metadata edit behavior.

---

### Acceptance Criteria

**Functional:**
- [x] Taxonomy CRUD and assignment flows are usable end-to-end.
- [x] Subdomain selectors enforce domain scoping in UI behavior.
- [x] Item cards show taxonomy metadata without breaking existing tile actions.

**Testing:**
- [x] UI verification demonstrates filter and rendering correctness.
- [x] Existing prompt preview and skill edit flows remain stable.

---

### Dependencies

**Blocked By:**
- WP-006
- WP-007

**Blocks:**
- WP-011

**Parallel Execution:**
- Can run in parallel with: WP-009 (once API contracts are stable).
- Cannot run in parallel with: WP-006, WP-007.

---

### Risks

**Risk 1: Added UI state complexity causes regressions in existing modals**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep taxonomy state isolated and reuse existing modal lifecycle helpers.

**Risk 2: Taxonomy filter controls overload the current header layout**
- Probability: Medium
- Impact: Low
- Mitigation: Use collapsible/compact controls with responsive behavior testing.
