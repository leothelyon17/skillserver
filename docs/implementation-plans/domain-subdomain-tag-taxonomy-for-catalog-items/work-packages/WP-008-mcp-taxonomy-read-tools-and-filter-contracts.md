## WP-008: MCP Taxonomy Read Tools and Filter Contracts

### Metadata

```yaml
WP_ID: WP-008
Title: MCP Taxonomy Read Tools and Filter Contracts
Domain: MCP Layer
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Agents require MCP-native visibility into taxonomy objects and assignment metadata, plus taxonomy-aware catalog filtering.

**Scope:**
- Add read tools:
  - `list_taxonomy_domains`
  - `list_taxonomy_subdomains`
  - `list_taxonomy_tags`
  - `get_catalog_item_taxonomy`
- Extend `list_catalog` and `search_catalog` tool input contracts with taxonomy filters.
- Ensure structured output includes taxonomy fields needed by agents.

Excluded:
- MCP taxonomy write tools and write-gate wiring (WP-009).

**Success Criteria:**
- [ ] New taxonomy read tools are registered and callable.
- [ ] Catalog MCP tools accept and apply taxonomy filters.
- [ ] Output contracts are deterministic and documented in tool schemas.

---

### Technical Requirements

**Input Contracts:**
- Taxonomy services from WP-003 and WP-004.
- Existing MCP server/tools architecture in `pkg/mcp/server.go` and `pkg/mcp/tools.go`.

**Output Contracts:**
- MCP tool schemas and handlers with additive contracts.
- End-to-end MCP test coverage for taxonomy read/filter behavior.

**Integration Points:**
- WP-009 builds on this contract surface for write tools.
- WP-011 includes MCP parity testing against REST behavior.

---

### Deliverables

**Code Deliverables:**
- [ ] Add taxonomy read tool schemas and handlers in `pkg/mcp/tools.go`.
- [ ] Register read tools in `pkg/mcp/server.go`.
- [ ] Extend `list_catalog` and `search_catalog` inputs/handlers for taxonomy filters.

**Test Deliverables:**
- [ ] MCP tool registration and invocation tests for new read tools.
- [ ] Filter behavior tests for taxonomy-aware list/search.
- [ ] Validation tests for invalid taxonomy filter inputs.

---

### Acceptance Criteria

**Functional:**
- [ ] MCP clients can enumerate taxonomy objects and item assignments.
- [ ] Catalog MCP tools support taxonomy selectors with clear validation errors.

**Testing:**
- [ ] MCP tests verify tool contracts, filtering semantics, and structured outputs.
- [ ] Existing non-taxonomy MCP tools remain unaffected.

---

### Dependencies

**Blocked By:**
- WP-003
- WP-004

**Blocks:**
- WP-009
- WP-011

**Parallel Execution:**
- Can run in parallel with: WP-006 and WP-007 after dependencies are complete.
- Cannot run in parallel with: WP-003, WP-004.

---

### Risks

**Risk 1: MCP filter contract drifts from REST semantics**
- Probability: Medium
- Impact: Medium
- Mitigation: Reuse shared filter translation in domain service layer where possible.

**Risk 2: Tool schema bloat makes discovery harder for agents**
- Probability: Low
- Impact: Low
- Mitigation: Keep inputs concise and aligned with REST query names.
