## WP-001: Architecture Contract and Compatibility Matrix

### Metadata

```yaml
WP_ID: WP-001
Title: Architecture Contract and Compatibility Matrix
Domain: architecture
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/implementation-planner.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for contract definition, sequencing, and compatibility planning.
Priority: High
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-09
Started_Date: 2026-03-09
Completed_Date: 2026-03-09
```

---

### Description

**Context:**
The current catalog surfaces have additive improvements queued, but the sharp edges are contract-level: skill IDs are inconsistent, classification completeness is implicit, list/search defaults are noisy, and export ergonomics are split across caller expectations. This package fixes the contract decisions first so downstream work does not diverge.

**Scope:**
- Define canonical ID behavior for:
  - `list_skills`
  - `search_skills`
  - `read_skill`
  - skill resource tools
  - taxonomy tools
  - export/materialization tools
  - REST catalog item routes
- Define explicit classification-state semantics for:
  - `has_assignment`
  - `is_fully_classified`
  - `missing_fields`
- Lock additive filter and pagination semantics for list/search.
- Lock export archive root semantics and MCP archive payload defaults.

Excluded:
- Implementing normalization helpers or handlers.
- Repository, UI, or test-code changes beyond documentation updates needed to record the decisions.

**Success Criteria:**
- [x] Every affected public surface has an explicit ID compatibility rule.
- [x] Completeness semantics are stable enough for REST, MCP, and UI to share.
- [x] Pagination and metadata-first defaults are defined before interface work starts.
- [x] Export ergonomics decisions avoid introducing redundant write paths.

---

### Technical Requirements

**Input Contracts:**
- Existing contracts in `pkg/mcp/tools.go`, `pkg/mcp/tools_export_materialization.go`, and `pkg/web/handlers.go`.
- Existing domain ID builders and normalizers in `pkg/domain/catalog.go`, `pkg/domain/catalog_export_service.go`, and `pkg/domain/catalog_materialization_service.go`.

**Output Contracts:**
- One approved compatibility matrix covering legacy and canonical references.
- One stable vocabulary for classification completeness and missing fields.
- One documented decision for `archive_root_mode` and `include_archive_base64`.

**Integration Points:**
- WP-002 and WP-003 depend on these decisions to avoid rework.
- WP-005 and WP-006 must implement these contracts exactly.

---

### Deliverables

**Documentation Deliverables:**
- [x] Update the implementation plan with finalized contract decisions where needed.
- [x] Add a compatibility matrix section to `README.md` or an equivalent contract note.
- [x] Record filter, pagination, and export-default decisions in implementation docs.

**Review Deliverables:**
- [x] Resolve the open choice on `list_skills.id` behavior.
- [x] Resolve the open choice on default archive-root flattening semantics.

---

### Acceptance Criteria

**Functional:**
- [x] The contract inventory covers all six requested improvement areas.
- [x] Legacy bare skill ID compatibility is explicit and bounded.
- [x] The `missing_fields` vocabulary is stable and non-ambiguous.

**Testing:**
- [x] No automated tests required for this planning package.
- [x] Follow-on packages can reference this file without reopening contract questions.

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
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: None.
- Cannot run in parallel with: Downstream packages that depend on unsettled contracts.

---

### Risks

**Risk 1: Contract decisions remain implicit and drift across layers**
- Probability: High
- Impact: High
- Mitigation: Treat this WP as a hard prerequisite and reference it from downstream package reviews.

**Risk 2: Compatibility goals become too broad and block progress**
- Probability: Medium
- Impact: Medium
- Mitigation: Limit compatibility to bare skill IDs only; keep prompt and rule IDs canonical-only.
