## WP-001: Relationship Contract and Write Authority

### Metadata

```yaml
WP_ID: WP-001
Title: Relationship Contract and Write Authority
Domain: architecture
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/implementation-planner.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for locking payload contracts, sequencing, and scope boundaries before implementation starts.
Priority: High
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Unassigned
Created_Date: 2026-03-11
Started_Date: 2026-03-11
Completed_Date: 2026-03-11
```

---

### Description

**Context:**
ADR-008 is narrow in relationship types but broad in surface area. The contract decisions must be settled first so persistence, REST, MCP, and UI work all implement the same semantics.

**Scope:**
- Define the normalized relationship response shape for skill, prompt, and rule items.
- Define skill-owned write authority, canonical item-ID expectations, and validation/error semantics.
- Define stale-row suppression and reconciliation behavior for deleted or missing related endpoints.
- Define what stays out of scope for v1, especially MCP writes and tile-level relationship rendering.

Excluded:
- Schema, repository, service, handler, tool, or UI implementation.

**Success Criteria:**
- [x] One approved relationship payload shape exists for REST and MCP reads.
- [x] Skill-owned write authority and prompt/rule reverse-read-only semantics are explicit.
- [x] Canonical ID expectations and compatibility behavior are documented before handler/tool work begins.
- [x] Deleted-endpoint suppression and reconciliation expectations are stable enough for downstream packages to implement without reopening design questions.

---

### Technical Requirements

**Input Contracts:**
- ADR-008 decision text and technical details.
- Existing metadata/taxonomy contracts in `pkg/domain/catalog_metadata_service.go`, `pkg/web/handlers.go`, `pkg/mcp/server.go`, and `README.md`.

**Output Contracts:**
- One relationship contract note captured in the implementation plan and referenced by downstream WPs.
- One stable vocabulary for:
  - `prompt`
  - `rules`
  - reverse `skills`
  - skill-owned write authority
  - deleted-endpoint suppression

**Final Contract Note (Approved):**
- Authoritative source: `docs/implementation-plans/skill-rule-and-prompt-relationship-metadata/skill-rule-and-prompt-relationship-metadata-implementation-plan.md` in section `WP-001 Finalized Relationship Contract (Authoritative)`.
- Read shape uses stable `relationships.prompt`, `relationships.rules`, and `relationships.skills` fields across REST and MCP detail surfaces.
- Write authority is skill-owned for v1; prompt/rule metadata surfaces are reverse read-only.
- Canonical ID policy is explicit: REST relationship surfaces are canonical-only, while MCP read preserves bare-skill compatibility for `skill` items only.
- Deleted/missing endpoint behavior is both lazy suppression on reads and eager stale-row reconciliation during startup/sync flows.

**Integration Points:**
- WP-002 and WP-003 depend on the final table/query shape.
- WP-004 depends on the final projection and pruning semantics.
- WP-005, WP-006, and WP-007 must implement this contract exactly.

---

### Deliverables

**Documentation Deliverables:**
- [x] Finalize the response and write payload shapes in the implementation plan.
- [x] Record canonical ID expectations for REST and MCP relationship surfaces.
- [x] Record v1 authority rules: skill writes, prompt/rule reverse read-only views.

**Review Deliverables:**
- [x] Resolve whether MCP relationship reads keep bare-skill compatibility or stay canonical-only.
- [x] Resolve whether relationship pruning happens eagerly during sync, lazily on read, or both.

---

### Acceptance Criteria

**Functional:**
- [x] Downstream work packages can reference one stable relationship contract without reopening core semantics.
- [x] No unresolved ambiguity remains around write authority, item-ID expectations, or deleted-endpoint behavior.

**Testing:**
- [x] No automated tests required for this planning package.
- [x] Follow-on packages reference this file and the implementation plan rather than redefining contracts inline.

---

### Dependencies

**Blocked By:**
- None.

**Blocks:**
- WP-002
- WP-003
- WP-004
- WP-005
- WP-006
- WP-007

**Parallel Execution:**
- Can run in parallel with: None.
- Cannot run in parallel with: Downstream implementation packages that depend on unsettled contracts.

---

### Risks

**Risk 1: Contract drift across REST, MCP, and UI**
- Probability: High
- Impact: High
- Mitigation: Treat this package as a hard prerequisite and reference it in downstream reviews.

**Risk 2: v1 scope expands into generic graph modeling**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep the contract explicitly limited to skill->rule and skill->prompt relationships.
