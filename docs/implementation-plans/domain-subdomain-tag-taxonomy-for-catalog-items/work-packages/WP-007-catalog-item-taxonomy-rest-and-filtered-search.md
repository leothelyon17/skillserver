## WP-007: Catalog Item Taxonomy REST and Filtered Search

### Metadata

```yaml
WP_ID: WP-007
Title: Catalog Item Taxonomy REST and Filtered Search
Domain: API Layer
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
ADR-005 requires item-level taxonomy assignment APIs and taxonomy-aware filtering on catalog list/search endpoints.

**Scope:**
- Add `GET /api/catalog/:id/taxonomy` and `PATCH /api/catalog/:id/taxonomy`.
- Extend list/search query parsing to support taxonomy filters:
  - `primary_domain_id`, `secondary_domain_id`, `subdomain_id`, `tag_ids`, `tag_match`.
- Ensure filter semantics apply consistently on list and search paths.

Excluded:
- Taxonomy object CRUD endpoints (WP-006).
- MCP contract updates (WP-008, WP-009).

**Success Criteria:**
- [ ] Item taxonomy get/patch endpoints return and mutate assignment state correctly.
- [ ] List/search taxonomy filter semantics are deterministic (`any` vs `all`).
- [ ] Existing list/search behavior remains compatible when filters are omitted.

---

### Technical Requirements

**Input Contracts:**
- Assignment and effective projection services from WP-004.
- Existing catalog handlers in `pkg/web/handlers.go`.

**Output Contracts:**
- New taxonomy assignment handler DTOs and route registrations.
- Expanded list/search filter parser and effective service invocations.

**Integration Points:**
- WP-010 UI filter controls and item editor assignment workflows.
- WP-011 validates contract and filter behavior.

---

### Deliverables

**Code Deliverables:**
- [ ] Add handlers for item taxonomy get/patch operations.
- [ ] Add taxonomy filter query parsing and validation in list/search handlers.
- [ ] Map domain/service errors to stable HTTP statuses.

**Test Deliverables:**
- [ ] API tests for item taxonomy get/patch success and failure paths.
- [ ] API tests for taxonomy filter combinations and tag match modes.
- [ ] Compatibility tests for unfiltered list/search behavior.

---

### Acceptance Criteria

**Functional:**
- [ ] `GET/PATCH /api/catalog/:id/taxonomy` contracts are stable and validated.
- [ ] Taxonomy filters operate equivalently on list and search endpoints.

**Testing:**
- [ ] API tests cover full filter matrix and error edge cases.
- [ ] Existing catalog API tests remain green.

---

### Dependencies

**Blocked By:**
- WP-004

**Blocks:**
- WP-010
- WP-011

**Parallel Execution:**
- Can run in parallel with: WP-006 (once WP-004 is complete).
- Cannot run in parallel with: WP-004.

---

### Risks

**Risk 1: Divergent semantics between list and search filters**
- Probability: Medium
- Impact: Medium
- Mitigation: Reuse shared filter parsing and effective filter mapping code paths.

**Risk 2: Tag filter parsing errors from malformed comma-separated input**
- Probability: Medium
- Impact: Low
- Mitigation: Normalize and validate IDs with explicit 400 errors.
