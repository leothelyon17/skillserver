## WP-002: Taxonomy Persistence Repositories and Row Models

### Metadata

```yaml
WP_ID: WP-002
Title: Taxonomy Persistence Repositories and Row Models
Domain: Data Layer
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
After schema creation, ADR-005 needs typed repository abstractions for taxonomy registry and assignment persistence with validation and deterministic filtering.

**Scope:**
- Extend persistence row model definitions for domains/subdomains/tags and item assignments.
- Implement repositories for:
  - Taxonomy object CRUD and list operations.
  - Item taxonomy assignment get/upsert.
  - Item tag assignment replace/list operations.
- Support list filters needed by REST/MCP query surfaces.

Excluded:
- Business validation rules and cross-object governance (WP-003).
- Effective projection and label compatibility merges (WP-004/WP-005).

**Success Criteria:**
- [ ] Repositories expose full CRUD/list methods for taxonomy objects.
- [ ] Assignment upsert/list operations are transaction-safe.
- [ ] Repository tests cover happy path and constraint failures.

---

### Technical Requirements

**Input Contracts:**
- Taxonomy schema from WP-001.
- Existing repository style in `pkg/persistence/catalog_source_repository.go` and `pkg/persistence/catalog_overlay_repository.go`.

**Output Contracts:**
- New persistence types and repository files under `pkg/persistence/`.
- Comprehensive tests for CRUD/filter/constraint behavior.

**Integration Points:**
- WP-003 domain services consume taxonomy repositories.
- WP-004 effective projection consumes assignment and tag read paths.
- WP-005 backfill uses create/find operations for tags and assignments.

---

### Deliverables

**Code Deliverables:**
- [ ] Add taxonomy row structs and validation helpers in `pkg/persistence/catalog_row_models.go` (or taxonomy-specific row file).
- [ ] Add taxonomy registry repository implementation and interfaces.
- [ ] Add item taxonomy/tag assignment repository implementation.
- [ ] Add list filter structs supporting domain/subdomain/tag lookup paths.

**Test Deliverables:**
- [ ] Repository tests for domain/subdomain/tag CRUD.
- [ ] Tests for assignment replacement semantics and uniqueness guarantees.
- [ ] Tests for deletion restrictions and FK violations.

---

### Acceptance Criteria

**Functional:**
- [ ] All taxonomy repositories support deterministic list ordering.
- [ ] Assignment writes are idempotent and safe on repeated patch calls.
- [ ] Query helpers provide inputs needed for service/API filters.

**Testing:**
- [ ] New persistence tests cover constraint and edge-case paths.
- [ ] Existing persistence test suites stay green.

---

### Dependencies

**Blocked By:**
- WP-001

**Blocks:**
- WP-003
- WP-004
- WP-005
- WP-006
- WP-007

**Parallel Execution:**
- Can run in parallel with: None until WP-001 is complete.
- Cannot run in parallel with: WP-001.

---

### Risks

**Risk 1: Repository APIs diverge from service needs**
- Probability: Medium
- Impact: Medium
- Mitigation: Define service input/output contracts before finalizing repository signatures.

**Risk 2: Assignment replacement logic leaves stale tag links**
- Probability: Medium
- Impact: Medium
- Mitigation: Use transaction-scoped delete+insert replace semantics with rollback-safe tests.
