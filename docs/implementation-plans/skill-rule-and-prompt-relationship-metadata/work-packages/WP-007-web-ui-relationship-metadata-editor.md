## WP-007: Web UI Relationship Metadata Editor

### Metadata

```yaml
WP_ID: WP-007
Title: Web UI Relationship Metadata Editor
Domain: frontend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/web-applications-principal-developer-v2.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for interactive UI state, modal workflows, and metadata editing UX.
Priority: High
Estimated_Effort: 5 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-11
Started_Date: 2026-03-11
Completed_Date: 2026-03-11
```

---

### Description

**Context:**
ADR-008 keeps relationship management inside the existing metadata workflow. The UI must make relationship scope visible and editable without cluttering catalog tiles or confusing write authority.

**Scope:**
- Extend `pkg/web/ui/index.html` metadata modal so:
  - skill items show prompt single-select and rule multi-select editors
  - prompt and rule items show reverse-associated skills as read-only lists
  - relationship summaries show enough context to disambiguate similarly named items
- Reuse existing catalog endpoints filtered by classifier to populate candidate selectors.
- Integrate relationship save/load into the current metadata modal flow.

Excluded:
- New catalog-tile rendering.
- New standalone relationship management page.

**Success Criteria:**
- [x] Skill metadata modal loads and saves relationship state.
- [x] Prompt and rule metadata views clearly communicate read-only reverse visibility.
- [x] Main catalog tiles remain unchanged.

---

### Technical Requirements

**Input Contracts:**
- REST metadata and relationship endpoints from WP-005.
- Finalized relationship contract in `skill-rule-and-prompt-relationship-metadata-implementation-plan.md` section `WP-001 Finalized Relationship Contract (Authoritative)`.
- Existing metadata modal load/save logic in `pkg/web/ui/index.html`.

**Output Contracts:**
- Stable modal state for relationship data and candidate lists.
- Deterministic save ordering that preserves clear user error reporting.

**Integration Points:**
- Uses `GET /api/catalog/:id/metadata` and `PATCH /api/catalog/:id/relationships`.
- WP-008 validates manual or automated UI behavior against the underlying APIs.

---

### Deliverables

**Code Deliverables:**
- [x] Extend modal state, loaders, and save flow in `pkg/web/ui/index.html`.
- [x] Add prompt selector and rule multi-select controls for skills.
- [x] Add reverse-associated skill rendering for prompt/rule items.
- [x] Reuse catalog list/search calls with classifier filters for picker data.

**Test Deliverables:**
- [x] Add or update Playwright/UI verification coverage if available.
- [x] If automated UI coverage is not practical, document a manual verification checklist in the WP completion summary.

---

### Acceptance Criteria

**Functional:**
- [x] Skills can view and edit prompt/rule relationships from the metadata modal.
- [x] Prompts and rules display reverse `skills` relationships but do not expose write controls.
- [x] Save failures are surfaced cleanly without corrupting the rest of the metadata form state.
- [x] Catalog tiles remain relationship-free.

**Testing:**
- [x] UI verification covers load, save, reverse display, and unchanged tile behavior.
- [x] Existing metadata modal behaviors for labels, custom metadata, and taxonomy remain intact.

---

### Dependencies

**Blocked By:**
- WP-005

**Blocks:**
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-006.
- Cannot run in parallel with: WP-005.

---

### Risks

**Risk 1: Relationship controls make the metadata modal too noisy**
- Probability: Medium
- Impact: Medium
- Mitigation: Limit editable controls to skill items and keep prompt/rule views display-only.

**Risk 2: Candidate lists are ambiguous when names collide**
- Probability: Medium
- Impact: Medium
- Mitigation: Render `parent_skill_id` and `resource_path` alongside display names.
