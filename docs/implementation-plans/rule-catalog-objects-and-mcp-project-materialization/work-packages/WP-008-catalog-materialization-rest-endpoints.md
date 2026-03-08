## WP-008: Catalog Materialization REST Endpoints

### Metadata

```yaml
WP_ID: WP-008
Title: Catalog Materialization REST Endpoints
Domain: API Layer
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
REST needs the final additive contracts for classifier-aware export and project-folder materialization, while preserving the earlier compatibility wrapper introduced in WP-002.

**Scope:**
- Finalize `POST /api/catalog/export` for skills, prompts, and rules.
- Add `POST /api/catalog/materialize` with `destination_dir`, `conflict_policy`, and `dry_run`.
- Return capability-aware errors when materialization is disabled.
- Preserve `GET /api/skills/export/*` as a compatibility wrapper over the shared service.

Excluded:
- MCP tool registration and schemas (WP-009).
- UI interaction changes (WP-010).

**Success Criteria:**
- [ ] Export and materialize endpoints delegate all planning/writing to the shared services.
- [ ] Request validation rejects invalid item lists, invalid roots, and unsupported conflict policies.
- [ ] Existing skill export URL behavior remains unchanged for clients.

---

### Technical Requirements

**Input Contracts:**
- Export route/service work from WP-002.
- Rule-aware catalog items from WP-006.
- Materialization service from WP-007.

**Output Contracts:**
- REST DTOs and handlers in `pkg/web/handlers.go`.
- Route wiring in `pkg/web/server.go`.
- Runtime capability exposure for materialization state.

**Integration Points:**
- WP-010 UI consumes these endpoints directly.
- WP-011 regression matrix validates disabled-capability and safety behavior.

---

### Deliverables

**Code Deliverables:**
- [ ] Complete classifier-aware `POST /api/catalog/export` behavior in `pkg/web/handlers.go`.
- [ ] Add `POST /api/catalog/materialize` handler and route wiring.
- [ ] Extend runtime capability responses if the UI needs additive fields for materialization visibility.
- [ ] Keep the legacy skill export route delegated to the shared export service.

**Test Deliverables:**
- [ ] Add REST tests for batch export and batch materialization.
- [ ] Add REST tests for `dry_run=true` and disabled-capability behavior.
- [ ] Add REST tests for path-safety and conflict-policy validation failures.

---

### Acceptance Criteria

**Functional:**
- [ ] Export endpoint supports mixed classifier requests once items are discoverable.
- [ ] Materialization endpoint returns planned/resolved targets and per-item outcomes.
- [ ] Disabled materialization state is surfaced as an explicit capability error, not a silent no-op.
- [ ] Legacy skill export route remains operational.

**Testing:**
- [ ] API tests cover success, validation errors, disabled gate, and dry-run flows.
- [ ] Tests verify no file writes occur during REST dry-run requests.

---

### Dependencies

**Blocked By:**
- WP-002
- WP-006
- WP-007

**Blocks:**
- WP-010
- WP-011
- WP-012

**Parallel Execution:**
- Can run in parallel with: WP-009
- Cannot run in parallel with: WP-007

---

### Risks

**Risk 1: REST response shape becomes difficult for UI and CLI callers to consume**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep per-item results explicit and stable; use dry-run output as the canonical manifest shape.

**Risk 2: Wrapper and additive endpoints diverge on filename/target semantics**
- Probability: Medium
- Impact: Medium
- Mitigation: Delegate both to the same shared services and avoid adapter-specific path logic.
