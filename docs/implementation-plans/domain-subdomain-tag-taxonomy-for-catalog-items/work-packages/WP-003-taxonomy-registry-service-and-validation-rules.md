## WP-003: Taxonomy Registry Service and Validation Rules

### Metadata

```yaml
WP_ID: WP-003
Title: Taxonomy Registry Service and Validation Rules
Domain: Service Layer
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
REST/MCP layers need centralized business validation for taxonomy object lifecycle operations. Validation cannot be duplicated in handlers/tools.

**Scope:**
- Implement domain service for taxonomy object CRUD orchestration.
- Enforce rules:
  - key normalization and uniqueness semantics.
  - subdomain belongs to a valid domain.
  - delete restrictions when objects are still assigned to catalog items.
- Map persistence errors to stable domain-level error types for handler/tool adaptation.

Excluded:
- Catalog item assignment logic and effective item merge semantics (WP-004).
- Transport-layer route/tool wiring (WP-006, WP-008, WP-009).

**Success Criteria:**
- [ ] Service exposes stable create/update/delete/list contracts.
- [ ] Validation errors are deterministic and transport-agnostic.
- [ ] Delete conflicts are explicit and actionable.

---

### Technical Requirements

**Input Contracts:**
- Taxonomy repositories from WP-002.
- Existing domain service patterns in `pkg/domain/catalog_metadata_service.go`.

**Output Contracts:**
- New taxonomy service under `pkg/domain/` with typed request/response structs.
- Domain error constants for validation and conflict paths.

**Integration Points:**
- WP-006 and WP-008 consume service APIs for registry operations.
- WP-009 write tools consume same service contracts.

---

### Deliverables

**Code Deliverables:**
- [ ] Add `pkg/domain/catalog_taxonomy_service.go` with CRUD/list methods.
- [ ] Add input normalization and domain-level validation helpers.
- [ ] Add domain errors for not-found, conflict, and invalid relationships.

**Test Deliverables:**
- [ ] Unit tests for key normalization and uniqueness conflict mapping.
- [ ] Unit tests for subdomain-domain relationship validation.
- [ ] Unit tests for in-use deletion guard behavior.

---

### Acceptance Criteria

**Functional:**
- [ ] Service ensures taxonomy object integrity independent of handler/tool layer.
- [ ] Conflict and validation errors are predictable and reusable.

**Testing:**
- [ ] Service tests cover create/update/delete/list edge cases.
- [ ] Error mapping tests assert stable behavior for API/MCP layers.

---

### Dependencies

**Blocked By:**
- WP-002

**Blocks:**
- WP-006
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-005 (once WP-002 is complete).
- Cannot run in parallel with: WP-002.

---

### Risks

**Risk 1: Validation logic spread across services and handlers**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep all taxonomy business rules in this service and use thin adapters elsewhere.

**Risk 2: Domain error taxonomy too coarse for UI/MCP feedback**
- Probability: Low
- Impact: Medium
- Mitigation: Define explicit error classes and include conflict context fields.
