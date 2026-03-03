## WP-005: MCP Resource Contract Metadata Update

### Metadata

```yaml
WP_ID: WP-005
Title: MCP Resource Contract Metadata Update
Domain: MCP Layer
Priority: Medium
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-02
Started_Date: 2026-03-03
Completed_Date: 2026-03-03
```

---

### Description

**Context:**
MCP tool responses currently assume a fixed resource taxonomy and do not expose origin/writability metadata.

**Scope:**
- Update MCP response structs to include additive metadata.
- Update tool descriptions to include prompt/import support.
- Preserve existing tool names and invocation flow.

Excluded:
- Domain discovery logic (WP-003/004).

**Success Criteria:**
- [x] MCP list/read/info tools remain backward compatible.
- [x] MCP list responses include `type=prompt` and additive metadata fields.

---

### Technical Requirements

**Input Contracts:**
- `SkillResource` metadata from WP-003/004.

**Output Contracts:**
- Updates in `pkg/mcp/tools.go` and `pkg/mcp/server.go` descriptions.

**Integration Points:**
- Consumed by external MCP clients.
- Verified by WP-008 contract tests.

---

### Deliverables

**Code Deliverables:**
- [x] Extend `SkillResourceInfo` and info-output structs with additive metadata.
- [x] Update MCP tool descriptions to mention prompt/imported resources.

**Test Deliverables:**
- [x] Add or extend MCP tests to assert tool output schema and compatibility.

---

### Acceptance Criteria

**Functional:**
- [x] Existing MCP clients can still parse old fields unchanged.
- [x] New fields are present when manager provides metadata.

**Testing:**
- [x] MCP regression tests remain green; new schema assertions added.

---

### Dependencies

**Blocked By:**
- WP-003

**Blocks:**
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-004, WP-006
- Cannot run in parallel with: WP-003

---

### Risks

**Risk 1: Client-side strict decoders reject additive fields**
- Probability: Low
- Impact: Medium
- Mitigation: Keep old fields and structure unchanged; add contract note in docs.
