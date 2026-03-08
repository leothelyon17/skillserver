## WP-010: UI Export and Materialization UX

### Metadata

```yaml
WP_ID: WP-010
Title: UI Export and Materialization UX
Domain: UI Layer
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-08
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
The GUI currently exposes a skill-only export action. ADR-007 requires classifier-aware export/materialize behavior for skills, prompts, and rules while respecting runtime capability gates.

**Scope:**
- Update UI actions so export/materialize semantics are classifier-aware.
- Add a materialization modal or panel for destination directory, conflict policy, and dry-run preview.
- Preserve current skill editing and read-only affordances.
- Hide or disable materialization actions when runtime capabilities indicate they are unavailable.

Excluded:
- Backend endpoint implementation (WP-008).
- MCP workflows (WP-009).

**Success Criteria:**
- [ ] Skill export remains simple and stable.
- [ ] Prompt/rule items can preview and invoke materialization through the new REST contracts.
- [ ] The UI reflects capability state clearly instead of allowing failing actions blindly.

---

### Technical Requirements

**Input Contracts:**
- Rule-aware catalog items from WP-006.
- REST export/materialize endpoints and capability fields from WP-008.
- Existing UI behavior in `pkg/web/ui/index.html`.

**Output Contracts:**
- Updated UI action handlers and modal state in `pkg/web/ui/index.html`.
- Any related styling or utility updates required in `pkg/web/ui/style.css`.
- Manual verification notes or UI-focused assertions usable by WP-011.

**Integration Points:**
- Depends on WP-008 for endpoint behavior.
- WP-011 regression matrix validates the finished UX and capability gating.

---

### Deliverables

**Code Deliverables:**
- [ ] Replace skill-only UI assumptions with classifier-aware export/materialize action handling.
- [ ] Add destination/conflict-policy controls and dry-run preview rendering.
- [ ] Update capability checks so disabled write features are not surfaced as active UI controls.

**Test Deliverables:**
- [ ] Add UI/manual verification checklist items for skills, prompts, and rules.
- [ ] Add or update UI tests/assertions if the repository already covers this flow.

---

### Acceptance Criteria

**Functional:**
- [ ] Existing skill export action still works after the new UI changes.
- [ ] Prompt/rule actions expose dry-run preview before writes.
- [ ] Disabled materialization capability hides or disables write actions.
- [ ] Error toasts/messages distinguish export failures from materialization failures.

**Testing:**
- [ ] Verification covers at least one skill, one prompt, and one rule item.
- [ ] Verification confirms the UI does not attempt writes during dry-run preview.

---

### Dependencies

**Blocked By:**
- WP-006
- WP-008

**Blocks:**
- WP-011
- WP-012

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-008

---

### Risks

**Risk 1: UI grows confusing due to classifier-specific actions**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep the action model simple: export always available where supported, materialize only when capability is enabled.

**Risk 2: UI exposes write actions before backend capability checks are available**
- Probability: Medium
- Impact: Medium
- Mitigation: Bind visibility to runtime capabilities and verify disabled states explicitly.
