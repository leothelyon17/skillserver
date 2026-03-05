## WP-004: Catalog Item Taxonomy Assignment and Effective Projection

### Metadata

```yaml
WP_ID: WP-004
Title: Catalog Item Taxonomy Assignment and Effective Projection
Domain: Service Layer
Priority: High
Estimated_Effort: 5 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
ADR-005 requires taxonomy assignment state to be merged into effective catalog items and exposed uniformly to search, REST, MCP, and UI.

**Scope:**
- Implement item taxonomy assignment service with validation:
  - subdomain-domain consistency.
  - assignment existence checks.
- Extend effective projection in `pkg/domain/catalog_effective_service.go` to merge taxonomy data.
- Extend `pkg/domain/catalog.go` catalog item model with taxonomy reference fields.
- Add filter support in effective list APIs for taxonomy selectors.

Excluded:
- Legacy label backfill execution (WP-005).
- API and MCP route/tool wiring (WP-006 to WP-009).

**Success Criteria:**
- [ ] Effective catalog items include taxonomy references and tags.
- [ ] Assignment service validates relationship consistency before writes.
- [ ] Effective list supports taxonomy filtering inputs.

---

### Technical Requirements

**Input Contracts:**
- Taxonomy repositories from WP-002.
- Taxonomy object validation service from WP-003.

**Output Contracts:**
- Assignment service API used by REST and MCP patch flows.
- Updated catalog item DTO used throughout domain/API/MCP stack.

**Integration Points:**
- WP-005 consumes effective projection updates for label compatibility.
- WP-007 and WP-008 rely on taxonomy filter and assignment read/write contracts.

---

### Deliverables

**Code Deliverables:**
- [ ] Add assignment service under `pkg/domain/`.
- [ ] Extend `CatalogItem` with taxonomy reference fields and tags.
- [ ] Extend effective projection merges to include taxonomy joins.
- [ ] Add taxonomy filter options to effective list/filter contracts.

**Test Deliverables:**
- [ ] Service tests for valid/invalid assignment combinations.
- [ ] Effective projection tests for taxonomy field merge correctness.
- [ ] Filter tests for primary/secondary/subdomain/tag query semantics.

---

### Acceptance Criteria

**Functional:**
- [ ] Effective item payload includes taxonomy fields for assigned items.
- [ ] Invalid assignment patch attempts are rejected with explicit errors.
- [ ] Taxonomy filters return deterministic subsets.

**Testing:**
- [ ] Domain tests cover merge precedence and filter behavior.
- [ ] Existing non-taxonomy effective projection behavior remains intact.

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
- WP-010

**Parallel Execution:**
- Can run in parallel with: None until dependencies are complete.
- Cannot run in parallel with: WP-002, WP-003.

---

### Risks

**Risk 1: Join-heavy effective projection increases list/search latency**
- Probability: Medium
- Impact: Medium
- Mitigation: Use indexed joins and keep projection query shape bounded.

**Risk 2: Assignment validation misses cross-field mismatch edge cases**
- Probability: Medium
- Impact: High
- Mitigation: Add explicit matrix tests for domain/subdomain combinations.
