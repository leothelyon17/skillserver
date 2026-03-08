## WP-009: MCP Export and Materialization Tools

### Metadata

```yaml
WP_ID: WP-009
Title: MCP Export and Materialization Tools
Domain: MCP Layer
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
ADR-007 requires agents to request export/materialization through MCP instead of reconstructing files client-side. This is a write-capable surface and must be independently gated.

**Scope:**
- Add MCP tools for catalog export and materialization.
- Support dry-run planning so agents can inspect target paths before write execution.
- Register write-capable tools only when materialization is explicitly enabled.
- Update tool descriptions and schemas so classifier filters include `rule`.

Excluded:
- Underlying planning/write logic (WP-007).
- REST endpoint behavior (WP-008).
- UI workflows (WP-010).

**Success Criteria:**
- [ ] Read-only MCP deployments do not register materialization tools.
- [ ] Dry-run responses expose enough path information for agents to reason about writes.
- [ ] Tool errors are explicit for disallowed roots, invalid policies, and missing item IDs.

---

### Technical Requirements

**Input Contracts:**
- Runtime gating from WP-004.
- Rule-aware catalog behavior from WP-006.
- Shared materialization/export services from WP-007.

**Output Contracts:**
- MCP tool registration and schemas in `pkg/mcp/server.go`.
- Tool handlers and input/output structs in `pkg/mcp/tools.go`.
- Regression tests covering default and gated tool sets.

**Integration Points:**
- WP-011 uses these tools in regression and capability tests.
- Operators rely on WP-012 docs for rollout/rollback of the write gate.

---

### Deliverables

**Code Deliverables:**
- [ ] Add `export_catalog_items` MCP tool.
- [ ] Add `materialize_catalog_items` MCP tool with `dry_run` support, or a dedicated planning tool if that yields cleaner semantics.
- [ ] Update classifier validation/docs so `rule` is a supported value in catalog tool filters.

**Test Deliverables:**
- [ ] Extend `pkg/mcp/server_stdio_regression_test.go` for tool registration gating.
- [ ] Add tool tests for export/materialization success, dry-run, and safety failures.
- [ ] Add tests proving read-only tool sets remain unchanged when materialization is disabled.

---

### Acceptance Criteria

**Functional:**
- [ ] Materialization tools are absent unless runtime gating enables them.
- [ ] Tool responses expose resolved target paths for dry-run planning.
- [ ] Tool calls fail safely for invalid destinations and unsupported conflict policies.
- [ ] Existing catalog read tools continue to operate unchanged.

**Testing:**
- [ ] MCP tests cover registration, happy path, dry-run, invalid input, and disabled-gate behavior.
- [ ] Catalog classifier tests include `rule` in MCP filter contracts.

---

### Dependencies

**Blocked By:**
- WP-004
- WP-006
- WP-007

**Blocks:**
- WP-011
- WP-012

**Parallel Execution:**
- Can run in parallel with: WP-008
- Cannot run in parallel with: WP-007

---

### Risks

**Risk 1: Materialization tools are accidentally available in read-only deployments**
- Probability: Low
- Impact: High
- Mitigation: Gate registration at server construction time and assert the default tool set in regression tests.

**Risk 2: Tool schemas do not expose enough detail for agent dry-run decisions**
- Probability: Medium
- Impact: Medium
- Mitigation: Make dry-run responses return explicit target paths, policies, and action outcomes.
