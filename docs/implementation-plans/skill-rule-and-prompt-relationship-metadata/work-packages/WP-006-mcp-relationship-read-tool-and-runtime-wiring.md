## WP-006: MCP Relationship Read Tool and Runtime Wiring

### Metadata

```yaml
WP_ID: WP-006
Title: MCP Relationship Read Tool and Runtime Wiring
Domain: backend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for MCP tool design, structured outputs, and runtime service injection.
Priority: High
Estimated_Effort: 4 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-11
Started_Date: 2026-03-11
Completed_Date: 2026-03-11
```

---

### Description

**Context:**
Agents need deterministic read access to relationship scope through MCP, but the ADR explicitly keeps relationship writes out of MCP for the first release.

**Scope:**
- Add `get_catalog_item_relationships` as a read-only MCP tool.
- Reuse the domain relationship projection from WP-004.
- Wire the relationship service into MCP runtime setup.
- Keep `list_catalog` and `search_catalog` unchanged for relationship detail and avoid adding write tools.

Excluded:
- MCP relationship writes.
- REST or UI work.

**Success Criteria:**
- [x] The new MCP read tool is registered and callable.
- [x] Structured output matches the REST relationship projection semantics.
- [x] No new relationship write tool is exposed.

---

### Technical Requirements

**Input Contracts:**
- Relationship service from WP-004.
- Finalized relationship contract in `skill-rule-and-prompt-relationship-metadata-implementation-plan.md` section `WP-001 Finalized Relationship Contract (Authoritative)`.
- Existing MCP server and tool patterns in `pkg/mcp/server.go` and `pkg/mcp/tools.go`.

**Output Contracts:**
- Additive MCP tool schema and handler.
- Runtime service injection for relationship reads in MCP startup code.

**Integration Points:**
- WP-008 validates tool registration and end-to-end behavior.
- README and rollout docs in WP-009 document the new tool contract.

---

### Deliverables

**Code Deliverables:**
- [x] Add MCP input/output types and handler logic in `pkg/mcp/tools.go`.
- [x] Register `get_catalog_item_relationships` in `pkg/mcp/server.go`.
- [x] Wire the relationship service into server bootstrap/runtime setup in the relevant `cmd/skillserver` runtime files.
- [x] Add or extend MCP regression tests in `pkg/mcp/server_stdio_regression_test.go`.

**Test Deliverables:**
- [x] Tool registration test.
- [x] End-to-end invocation test for skill, prompt, and rule relationship reads.
- [x] Validation test for missing or invalid item IDs.
- [x] Guard test proving no relationship write tool is registered.

---

### Acceptance Criteria

**Functional:**
- [x] MCP clients can read effective relationship scope for any supported catalog item.
- [x] Tool output uses the same prompt/rules/skills semantics as REST metadata/detail reads.
- [x] Relationship writes remain unavailable through MCP.

**Testing:**
- [x] MCP regression suite covers the new tool and confirms existing tools remain unaffected.
- [x] Runtime wiring tests prove the service is available when persistence metadata runtime is configured.

---

### Dependencies

**Blocked By:**
- WP-004

**Blocks:**
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-005 after WP-004 is complete.
- Cannot run in parallel with: WP-004.

---

### Risks

**Risk 1: MCP output drifts from REST semantics**
- Probability: Medium
- Impact: High
- Mitigation: Map both surfaces from the same domain relationship view.

**Risk 2: Runtime wiring becomes conditional in a different way than metadata/taxonomy services**
- Probability: Medium
- Impact: Medium
- Mitigation: Follow the existing service-injection pattern used for metadata and taxonomy surfaces.
