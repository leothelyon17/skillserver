## WP-002: Export REST Endpoint and Legacy Route Delegation

### Metadata

```yaml
WP_ID: WP-002
Title: Export REST Endpoint and Legacy Route Delegation
Domain: API Layer
Priority: High
Estimated_Effort: 3 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-08
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
The first delivery milestone is to recover the GUI skill export flow while establishing the new shared export seam.

**Scope:**
- Add `POST /api/catalog/export` as the additive export entry point.
- Re-implement `GET /api/skills/export/*` as a compatibility wrapper over the shared export service.
- Add request validation and error handling around missing IDs, malformed payloads, and export failures.

Excluded:
- Project-folder materialization endpoint (WP-008).
- UI changes beyond preserving the existing GUI export path.
- MCP tool registration.

**Success Criteria:**
- [ ] Existing GUI export no longer depends on the legacy direct archive path.
- [ ] Additive catalog export contract exists for later prompt/rule support.
- [ ] Backward compatibility for current skill downloads is preserved.

---

### Technical Requirements

**Input Contracts:**
- Shared export service from WP-001.
- Existing REST routing in `pkg/web/server.go`.

**Output Contracts:**
- New handler logic in `pkg/web/handlers.go`.
- Route wiring in `pkg/web/server.go`.
- API tests proving wrapper behavior and export endpoint validation.

**Integration Points:**
- WP-008 extends the same endpoint family for classifier-aware export/materialization behavior.
- WP-010 UI work consumes the additive export endpoint after materialization support is ready.

---

### Deliverables

**Code Deliverables:**
- [ ] Add `POST /api/catalog/export` route and handler.
- [ ] Delegate `GET /api/skills/export/*` to the shared export service instead of direct archive code.
- [ ] Return download metadata and dry-run/export planning payloads in a stable response shape.

**Test Deliverables:**
- [ ] Add REST tests for legacy wrapper success/failure cases.
- [ ] Add REST tests for malformed export payloads and missing items.
- [ ] Add integration coverage for repo-backed skill IDs with slashes.

---

### Acceptance Criteria

**Functional:**
- [ ] Existing UI-initiated skill export succeeds through the delegated route.
- [ ] `POST /api/catalog/export` validates `item_ids` and rejects empty requests.
- [ ] Legacy skill-export URL shape remains unchanged for callers.

**Testing:**
- [ ] API tests cover local skill export, repo-backed skill export, invalid payloads, and missing items.
- [ ] Regression tests confirm existing response headers/content-disposition behavior remains compatible.

---

### Dependencies

**Blocked By:**
- WP-001

**Blocks:**
- WP-008
- WP-010
- WP-011

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-001

---

### Risks

**Risk 1: Wrapper loses wildcard route compatibility for repo-backed skill IDs**
- Probability: Medium
- Impact: Medium
- Mitigation: Add explicit tests for `repoName/skillName` exports and encoded path handling.

**Risk 2: Additive endpoint shape changes again in later phases**
- Probability: Medium
- Impact: Low
- Mitigation: Keep request/response contracts narrow now and extend additively in WP-008.
