## WP-003: Repository Pagination and Taxonomy Usage Query Support

### Metadata

```yaml
WP_ID: WP-003
Title: Repository Pagination and Taxonomy Usage Query Support
Domain: data
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/database-architect.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong match for repository query design, pagination, and count-preview query work.
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
Catalog list/search currently reads the full matching row set in deterministic order but does not expose pagination. Taxonomy delete checks exist, but there is no cheap usage/preflight path for agents or UI callers.

**Scope:**
- Extend source-list filtering with additive pagination support.
- Add usage-count and impacted-item preview queries for:
  - domains
  - subdomains
  - tags
- Keep ordering stable on `item_id`.
- Document any index additions required by the new query shapes.

Excluded:
- Service-layer usage response assembly (WP-004).
- REST or MCP pagination contracts (WP-005, WP-006).

**Success Criteria:**
- [x] Repositories can return deterministic slices suitable for cursor pagination.
- [x] Usage queries can answer preflight questions without scanning unrelated data in handlers.
- [x] Changes remain additive to current repository contracts.

---

### Technical Requirements

**Input Contracts:**
- `pkg/persistence/catalog_source_repository.go`
- `pkg/persistence/catalog_row_models.go`
- `pkg/persistence/catalog_taxonomy_row_models.go`

**Output Contracts:**
- Additive repository filter fields for pagination.
- Query helpers returning counts and preview item IDs for taxonomy object usage.

**Integration Points:**
- WP-004 builds the usage service on top of these query helpers.
- WP-005 and WP-006 use the pagination-ready repository/service flows.

---

### Deliverables

**Code Deliverables:**
- [x] Add cursor/limit support to catalog source list filtering.
- [x] Add usage query helpers for domain, subdomain, and tag references.
- [x] Add or adjust indexes only if query plans show real need.

**Test Deliverables:**
- [x] Add repository tests for cursor pagination ordering and limits.
- [x] Add repository tests for usage counts and impacted-item preview behavior.
- [x] Add regression tests covering existing unpaginated behavior.

---

### Acceptance Criteria

**Functional:**
- [x] Pagination semantics are deterministic and based on stable item ordering.
- [x] Usage queries return the data needed for delete-preflight and manager summaries.
- [x] Existing callers can continue using repository list methods without pagination arguments.

**Testing:**
- [x] Repository coverage exists for new filters and query helpers.
- [x] Existing repository tests remain green.

---

### Dependencies

**Blocked By:**
- WP-001

**Blocks:**
- WP-004
- WP-005
- WP-006
- WP-007
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-002 after WP-001.
- Cannot run in parallel with: WP-004 onward.

---

### Risks

**Risk 1: Cursor semantics become unstable when filters vary**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep cursors tied to deterministic `item_id` ordering and document filter-bound cursor reuse rules.

**Risk 2: Usage queries introduce unnecessary schema churn**
- Probability: Low
- Impact: Medium
- Mitigation: Start with additive query helpers and only add indexes when tests or profiling justify them.
