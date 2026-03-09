## WP-004: Partial and Batch Taxonomy Mutation Services

### Metadata

```yaml
WP_ID: WP-004
Title: Partial and Batch Taxonomy Mutation Services
Domain: backend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong match for mutation orchestration, validation rules, and dry-run service behavior.
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
Taxonomy mutation today is effectively full replacement for tags and single-item only. That forces clients into fragile read-modify-write loops and makes agent-side batch edits hard to plan safely.

**Scope:**
- Extend single-item patch input to support:
  - `add_tag_ids`
  - `remove_tag_ids`
  - `clear_tags`
- Add a batch patch service with:
  - per-item validation
  - dry-run planning
  - deterministic result status
- Add a usage/preflight aggregation service for taxonomy objects.
- Preserve explicit `tag_ids` full replacement behavior when present.

Excluded:
- REST endpoint decoding and transport wiring (WP-005).
- MCP tool schema and registration work (WP-006).

**Success Criteria:**
- [x] Clients can express additive tag changes without full replacement.
- [x] Batch patching can preview changes before writes occur.
- [x] Usage/preflight data is available through a cheap service call.

---

### Technical Requirements

**Input Contracts:**
- Shared skill ID normalization and completeness helpers from WP-002.
- Repository pagination and usage query support from WP-003.
- Existing assignment service in `pkg/domain/catalog_taxonomy_assignment_service.go`.

**Output Contracts:**
- Additive patch input fields for single-item mutation.
- Batch mutation request/result types with dry-run support.
- Usage/preflight service types that can be reused by REST and MCP transports.

**Integration Points:**
- WP-005 and WP-006 surface these services through REST and MCP.
- WP-007 consumes usage/preflight data in the taxonomy manager UI.

---

### Deliverables

**Code Deliverables:**
- [x] Extend `CatalogItemTaxonomyAssignmentPatchInput` with additive tag mutation fields.
- [x] Add batch taxonomy mutation service types and implementation.
- [x] Add taxonomy usage/preflight service and response types.
- [x] Ensure result objects include explicit classification-state fields after mutation.

**Test Deliverables:**
- [x] Add tests for additive tag mutation combinations.
- [x] Add tests for dry-run vs apply behavior in batch mutation.
- [x] Add tests for per-item failures and global validation failures.
- [x] Add tests for usage/preflight summaries.

---

### Acceptance Criteria

**Functional:**
- [x] Single-item mutation supports add/remove/clear without requiring current tags from the caller.
- [x] Batch mutation returns deterministic per-item statuses and preview rows.
- [x] Usage/preflight summaries identify counts and impacted item IDs.

**Testing:**
- [x] Service-level tests cover additive mutation edge cases.
- [x] Validation prevents partial writes on malformed batch requests.

---

### Dependencies

**Blocked By:**
- WP-002
- WP-003

**Blocks:**
- WP-005
- WP-006
- WP-007
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: None.
- Cannot run in parallel with: WP-005 and WP-006 because they depend on the finalized mutation contract.

---

### Risks

**Risk 1: Mixed mutation fields create ambiguous precedence**
- Probability: Medium
- Impact: High
- Mitigation: Define precedence explicitly in service code and document it in request contracts.

**Risk 2: Batch mutation dry-run diverges from apply behavior**
- Probability: Medium
- Impact: High
- Mitigation: Reuse the same planning path for dry-run and apply, with writes gated at the final step only.
