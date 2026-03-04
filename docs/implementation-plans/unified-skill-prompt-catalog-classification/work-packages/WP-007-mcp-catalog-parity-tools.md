## WP-007: MCP Catalog Parity Tools (Optional)

### Metadata

```yaml
WP_ID: WP-007
Title: MCP Catalog Parity Tools (Optional)
Domain: MCP Layer
Priority: Medium
Estimated_Effort: 3 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
ADR-003 lists MCP parity as a nice-to-have. Existing MCP tools are skill-centric and do not expose catalog classifier filtering.

**Scope:**
- Add additive MCP tools for unified catalog list/search.
- Support optional classifier filter (`skill` or `prompt`) for search/list operations.
- Preserve existing MCP tool behavior and contracts.

Excluded:
- Breaking changes to `list_skills` and `search_skills`.

**Success Criteria:**
- [ ] New MCP tools return classifier-aware catalog items.
- [ ] Existing MCP tools remain unchanged and passing.

---

### Technical Requirements

**Input Contracts:**
- Catalog manager/search methods from WP-003.

**Output Contracts:**
- New tool schemas/handlers in `pkg/mcp/server.go` and `pkg/mcp/tools.go`.

**Integration Points:**
- Depends on WP-003 (catalog backend).
- Validated by WP-008 MCP regression tests.

---

### Deliverables

**Code Deliverables:**
- [ ] Add `list_catalog` MCP tool.
- [ ] Add `search_catalog` MCP tool with optional classifier input.
- [ ] Return structured results with classifier, parent skill ID, resource path metadata.

**Test Deliverables:**
- [ ] Add MCP tests for new tools and compatibility tests for existing tools.

---

### Acceptance Criteria

**Functional:**
- [ ] MCP catalog tools return mixed skill/prompt sets.
- [ ] MCP classifier filter behaves as expected.
- [ ] Existing MCP skill tools remain functional.

**Testing:**
- [ ] MCP test suite covers new and legacy tool paths.

---

### Dependencies

**Blocked By:**
- WP-003

**Blocks:**
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-004, WP-006
- Cannot run in parallel with: WP-003

---

### Risks

**Risk 1: Expanded MCP surface area increases maintenance burden**
- Probability: Medium
- Impact: Low
- Mitigation: Keep tool contracts minimal and additive.

**Risk 2: Client confusion between skill-only and catalog tools**
- Probability: Low
- Impact: Medium
- Mitigation: Document clear usage boundaries and examples.
