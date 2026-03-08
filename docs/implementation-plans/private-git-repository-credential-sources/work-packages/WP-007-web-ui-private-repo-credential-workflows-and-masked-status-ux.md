## WP-007: Web UI Private Repo Credential Workflows and Masked Status UX

### Metadata

```yaml
WP_ID: WP-007
Title: Web UI Private Repo Credential Workflows and Masked Status UX
Domain: UI Layer
Priority: Medium
Estimated_Effort: 4 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-07
Started_Date: 2026-03-07
Completed_Date: 2026-03-07
```

---

### Description

**Context:**
The current UI only exposes a plain URL field. Private repo support needs a richer flow without turning the public-repo path into an overbuilt form or exposing stored-secret values on subsequent reads.

**Scope:**
- Add auth mode/source selectors to the repo add/edit UX.
- Add env/file reference fields for HTTPS and SSH auth.
- Add masked stored-secret entry when runtime capability allows it.
- Display per-repo sync status and redacted error state.

Excluded:
- Server-side handler validation.
- Documentation copy beyond inline UX text.

**Success Criteria:**
- [ ] Public repo onboarding remains simple.
- [ ] Stored-secret fields are hidden when the server does not advertise capability.
- [ ] Editing an existing repo never repopulates prior secret values into the browser.

---

### Technical Requirements

**Input Contracts:**
- Expanded repo DTOs and capability fields from WP-006.
- Existing repo management UI in `pkg/web/ui/index.html`.

**Output Contracts:**
- Updated browser state models and repo forms in `pkg/web/ui/index.html`.
- Masked status rendering and conditional controls based on capability.

**Integration Points:**
- WP-008 verifies UI/API contract alignment and secret masking behavior.

---

### Deliverables

**Code Deliverables:**
- [ ] Update repo add/edit modal state to support auth mode/source selection.
- [ ] Render env/file reference inputs by auth mode.
- [ ] Render stored-secret inputs only when enabled by server capability.
- [ ] Add sync-status badges and redacted error display in the repo list.

**Test Deliverables:**
- [ ] UI/browser verification checklist or automated coverage for public, env/file, and stored-secret flows.
- [ ] Regression checks that existing add/edit/delete/sync/toggle behavior still works.

---

### Acceptance Criteria

**Functional:**
- [ ] UI never displays previously stored secret values after save or reload.
- [ ] UI communicates "configured" or equivalent masked state for stored credentials.
- [ ] Sync status and redacted errors are understandable without leaking secrets.
- [ ] Public repos can still be created with only a URL.

**Testing:**
- [ ] Browser checks cover mode switching, source switching, validation messaging, and masked stored-secret behavior.
- [ ] Regression checks confirm legacy public repo management still works.

---

### Dependencies

**Blocked By:**
- WP-006

**Blocks:**
- WP-008

**Parallel Execution:**
- Can run in parallel with: None
- Cannot run in parallel with: WP-006

---

### Risks

**Risk 1: UI state accidentally retains secret material after submission**
- Probability: Medium
- Impact: High
- Mitigation: Reset secret input state immediately after successful save and rely on masked server summaries on reload.

**Risk 2: Added fields make the public flow confusing**
- Probability: Medium
- Impact: Medium
- Mitigation: Default to `auth.mode=none`, collapse advanced options, and show only relevant inputs.
