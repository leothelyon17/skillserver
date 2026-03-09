## WP-006: MCP Contract Expansion and Export Ergonomics

### Metadata

```yaml
WP_ID: WP-006
Title: MCP Contract Expansion and Export Ergonomics
Domain: integration
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong fit for MCP tool schema changes, compatibility handling, and archive-export behavior.
Priority: High
Estimated_Effort: 5 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-09
Started_Date: 2026-03-09
Completed_Date: 2026-03-09
```

---

### Description

**Context:**
MCP currently has the sharpest agent-facing friction: `list_skills` returns bare IDs with blank names, taxonomy tools expect canonical IDs, catalog list/search include full content by default, and export always returns base64 archive bytes.

**Scope:**
- Update skill-focused MCP tools to use canonical skill IDs and populated display names.
- Accept both bare and canonical skill references where compatibility requires it.
- Extend catalog list/search MCP tools with:
  - `include_content`
  - `limit`
  - `cursor`
  - classification-state filters
- Extend taxonomy MCP tools with explicit completeness fields and additive tag mutation inputs.
- Add one batch taxonomy patch MCP tool with `dry_run`.
- Add usage/preflight MCP read tools.
- Add export options for:
  - `archive_root_mode`
  - `include_archive_base64`

Excluded:
- REST handlers and UI behavior.
- Materialization capability gates, which remain unchanged.

**Success Criteria:**
- [x] MCP callers no longer need to guess how to convert a skill ID into a taxonomy-safe item ID.
- [x] MCP list/search payloads are lightweight by default.
- [x] MCP export responses can skip archive bytes when callers only need planning metadata.

---

### Technical Requirements

**Input Contracts:**
- Domain normalization/completeness outputs from WP-002.
- Batch mutation and usage services from WP-004.
- Existing MCP tool registration in `pkg/mcp/server.go` and tool handlers in `pkg/mcp/tools.go`.

**Output Contracts:**
- Updated MCP tool schemas.
- Additive tool registration for batch mutation and usage reads.
- Backward-compatible handling for legacy skill-only callers.

**Integration Points:**
- WP-008 validates MCP and REST parity where required.
- WP-009 documents these MCP contract changes for agent users.

---

### Deliverables

**Code Deliverables:**
- [x] Update `list_skills` and `search_skills` IDs/names.
- [x] Update `read_skill` and skill-resource tools to accept both skill ID shapes.
- [x] Extend catalog list/search inputs and outputs for lighter defaults and pagination.
- [x] Extend taxonomy MCP tools for additive mutation and explicit completeness state.
- [x] Add batch taxonomy patch and usage/preflight tools.
- [x] Extend export MCP tool options for archive-root and archive-byte behavior.

**Test Deliverables:**
- [x] Update stdio regression tests for skill-ID compatibility and populated names.
- [x] Add MCP tests for lighter list/search defaults and pagination.
- [x] Add MCP tests for batch mutation, usage tools, and export options.

---

### Acceptance Criteria

**Functional:**
- [x] `list_skills` and `search_skills` no longer return blank names.
- [x] Skill-related MCP tools accept both bare and canonical skill IDs.
- [x] `list_catalog` and `search_catalog` omit `content` unless requested.
- [x] `export_catalog_items` can return manifest-only responses without archive bytes.

**Testing:**
- [x] MCP regression tests cover compatibility and new additive contracts.
- [x] Existing MCP materialization gating remains unchanged.

---

### Dependencies

**Blocked By:**
- WP-002
- WP-003
- WP-004

**Blocks:**
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-005 after prerequisites complete.
- Cannot run in parallel with: WP-004.

---

### Risks

**Risk 1: MCP clients depend on current `list_skills` ID shape**
- Probability: Medium
- Impact: High
- Mitigation: Keep read-side compatibility for bare skill IDs and cover it with regression tests.

**Risk 2: Export options overcomplicate the tool surface**
- Probability: Low
- Impact: Medium
- Mitigation: Keep the new options narrowly scoped and default them to the ergonomic behavior agents actually need.
