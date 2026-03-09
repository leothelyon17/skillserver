## WP-008: Regression Matrix and Compatibility Coverage

### Metadata

```yaml
WP_ID: WP-008
Title: Regression Matrix and Compatibility Coverage
Domain: integration
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong fit for cross-surface regression design and API-MCP compatibility validation.
Priority: High
Estimated_Effort: 4 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-09
Started_Date: 2026-03-09
Completed_Date: 2026-03-09
```

---

### Description

**Context:**
This change set is deliberately additive, but it touches domain, repository, REST, MCP, and UI contracts. The release gate needs explicit regression coverage for compatibility and payload-shape changes.

**Scope:**
- Add or extend automated tests for:
  - skill ID compatibility
  - explicit classification-state fields
  - unclassified and missing-field filters
  - batch taxonomy mutation dry-run/apply behavior
  - metadata-first list/search defaults
  - usage/preflight endpoints and tools
  - flattened export archive roots
- Publish the command matrix in `tests/README.md`.

Excluded:
- New product logic beyond test facilitation.
- End-user documentation updates beyond the test matrix.

**Success Criteria:**
- [x] Compatibility promises are enforced by automated tests.
- [x] The rollout gate includes deterministic commands for REST, MCP, and UI coverage.
- [x] Payload-size and archive-root behavior changes are explicitly tested.

---

### Technical Requirements

**Input Contracts:**
- Completed REST and MCP surfaces from WP-005 and WP-006.
- Completed UI behavior from WP-007.

**Output Contracts:**
- Expanded tests under `pkg/domain`, `pkg/persistence`, `pkg/web`, `pkg/mcp`, and `tests/playwright`.
- Updated test matrix/checklist in `tests/README.md`.

**Integration Points:**
- WP-009 uses this matrix as the release/readiness reference.

---

### Deliverables

**Code Deliverables:**
- [x] Add domain and repository tests for normalization, completeness, usage, and pagination.
- [x] Add REST tests for filters, batch mutation, and usage/preflight responses.
- [x] Add MCP tests for compatibility, lighter payloads, and export options.
- [x] Add UI regression coverage for classification-state and usage-preflight flows.

**Test Deliverables:**
- [x] Update `tests/README.md` with CI-friendly commands and rollout gates.
- [x] Ensure tests cover both legacy-compatible and new canonical request shapes.

---

### Acceptance Criteria

**Functional:**
- [x] The regression matrix maps each requested improvement area to automated coverage.
- [x] Compatibility behavior for bare skill IDs is proven instead of assumed.

**Testing:**
- [x] Domain, REST, MCP, and UI suites pass for the new contracts.
- [x] No new flaky or timing-sensitive UI assertions are introduced.

---

### Dependencies

**Blocked By:**
- WP-005
- WP-006
- WP-007

**Blocks:**
- WP-009

**Parallel Execution:**
- Can run in parallel with: Final fixups on prior packages.
- Cannot run in parallel with: None strictly, but requires feature surfaces to be stable.

---

### Risks

**Risk 1: Compatibility coverage misses one of the dual-format skill entry points**
- Probability: Medium
- Impact: High
- Mitigation: Build a matrix of every skill-reference accepting surface and add at least one automated case per surface.

**Risk 2: UI tests become brittle as pagination is introduced**
- Probability: Medium
- Impact: Medium
- Mitigation: Use focused, state-based assertions instead of count assumptions tied to one page size.
