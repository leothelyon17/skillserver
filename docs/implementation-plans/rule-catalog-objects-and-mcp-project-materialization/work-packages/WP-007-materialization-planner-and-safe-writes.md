## WP-007: Materialization Planner and Safe Writes

### Metadata

```yaml
WP_ID: WP-007
Title: Materialization Planner and Safe Writes
Domain: Service Layer
Priority: High
Estimated_Effort: 5 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-08
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
ADR-007 adds a write-capable project materialization workflow. All path planning, conflict handling, batching, dry-run behavior, and root safety need to live in one shared service so REST, MCP, and UI semantics do not drift.

**Scope:**
- Implement shared materialization planning and execution for skills, prompts, and rules.
- Resolve target paths using ADR order: frontmatter target path, project-rule basename preservation, then classifier defaults.
- Enforce conflict policies and destination-root validation for both dry-run and write modes.
- Emit structured operation results suitable for audit-friendly logs and caller manifests.

Excluded:
- REST/MCP/UI adapters (WP-008, WP-009, WP-010).
- Runtime flag parsing (WP-004).

**Success Criteria:**
- [ ] Dry-run planning returns resolved targets with no filesystem side effects.
- [ ] Real writes are rejected outside allowed roots or on invalid paths.
- [ ] Mixed batches of skills/prompts/rules produce deterministic results.

---

### Technical Requirements

**Input Contracts:**
- Shared export service patterns from WP-001.
- Rule/install metadata helpers from WP-003.
- Runtime capability and allowed-root config from WP-004.
- Rule-aware catalog items from WP-006.

**Output Contracts:**
- Shared planning/write service in `pkg/domain/`.
- Conflict-policy and result models reusable by REST and MCP adapters.
- Service tests covering write-safety and batching.

**Integration Points:**
- WP-008 delegates REST export/materialize requests to this service.
- WP-009 delegates MCP tools to this service.
- WP-010 renders dry-run and per-file results returned by adapters.

---

### Deliverables

**Code Deliverables:**
- [ ] Add a materialization service in `pkg/domain/` (for example `catalog_materialization_service.go`).
- [ ] Implement target-path resolution and conflict-policy handling.
- [ ] Reuse or extract path-boundary helpers so writes respect allowed roots and traversal rules.
- [ ] Emit structured per-item/per-file results for both dry-run and write modes.

**Test Deliverables:**
- [ ] Add service tests for `error`, `overwrite`, and `skip` conflict policies.
- [ ] Add tests for absolute-path rejection, traversal rejection, and allowed-root enforcement.
- [ ] Add tests for project-root rule targets such as `AGENTS.md`.

---

### Acceptance Criteria

**Functional:**
- [ ] Dry-run returns resolved target paths and actions without writing files.
- [ ] Materialization rejects writes outside configured roots.
- [ ] Rules with valid install metadata land at the intended project-root or relative target.
- [ ] Skills and prompts use deterministic fallback targets when no metadata override is present.

**Testing:**
- [ ] Service tests cover direct and imported items, mixed batches, and existing-file conflicts.
- [ ] Tests verify no partial writes remain on failed validation paths.

---

### Dependencies

**Blocked By:**
- WP-001
- WP-003
- WP-004
- WP-006

**Blocks:**
- WP-008
- WP-009
- WP-010
- WP-011
- WP-012

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-001, WP-006

---

### Risks

**Risk 1: Path validation misses edge cases such as symlink or cleaned-path escapes**
- Probability: Low
- Impact: High
- Mitigation: Reuse the strongest existing root-boundary helpers and add explicit escape-path tests.

**Risk 2: Conflict-policy behavior is inconsistent across batch items**
- Probability: Medium
- Impact: Medium
- Mitigation: Define per-item result contracts clearly and validate them with mixed-batch tests.
