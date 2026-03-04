## WP-005: Effective Catalog Projection and Mutability Contract

### Metadata

```yaml
WP_ID: WP-005
Title: Effective Catalog Projection and Mutability Contract
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
API/search layers require a single effective record view that merges source snapshot fields with overlay metadata using clear precedence and mutability semantics.

**Scope:**
- Implement effective record projection from source + overlay tables.
- Add mutability semantics:
  - `content_writable`
  - `metadata_writable`
- Extend domain catalog response model to carry additive mutability fields.

Excluded:
- HTTP endpoint additions (WP-006).
- Runtime startup/resync invocation wiring (WP-007).

**Success Criteria:**
- [ ] Effective name/description/custom metadata resolve using overlay precedence.
- [ ] Mutability fields are explicit and accurate by source type.
- [ ] Service provides stable list/search inputs for API and indexing.

---

### Technical Requirements

**Input Contracts:**
- Repositories from WP-003.
- Source row sync results from WP-004.

**Output Contracts:**
- Effective projection service (for example `pkg/domain/catalog_effective_service.go`).
- Domain model updates in `pkg/domain/catalog.go` for additive fields.

**Integration Points:**
- WP-006 uses effective projection in API responses.
- WP-007 and WP-009 use effective projection for index rebuild and regression checks.

---

### Deliverables

**Code Deliverables:**
- [ ] Add effective projection query/mapper combining source and overlay rows.
- [ ] Add deterministic ordering and filtering by classifier/source type.
- [ ] Add additive mutability fields to domain/API response model.
- [ ] Preserve legacy `read_only` semantics for backward compatibility.

**Test Deliverables:**
- [ ] Unit tests for overlay precedence rules and null override behavior.
- [ ] Mutability matrix tests for `git`, `local`, and `file_import` source types.
- [ ] Tests for stable ordering and excluded tombstoned rows.

---

### Acceptance Criteria

**Functional:**
- [ ] Overlay fields override source fields only where values are set.
- [ ] `metadata_writable=true` for all item types.
- [ ] `content_writable=false` for git-derived records and true otherwise.

**Testing:**
- [ ] Effective projection tests cover both empty and populated overlays.
- [ ] Backward-compatible `read_only` mapping is preserved and tested.

---

### Dependencies

**Blocked By:**
- WP-003

**Blocks:**
- WP-006

**Parallel Execution:**
- Can run in parallel with: WP-004 (after WP-003)
- Cannot run in parallel with: WP-006

---

### Risks

**Risk 1: Mutability fields diverge from legacy read-only behavior**
- Probability: Medium
- Impact: Medium
- Mitigation: Add explicit compatibility tests for old and new fields.

**Risk 2: Incorrect precedence merges lead to stale metadata presentation**
- Probability: Medium
- Impact: Medium
- Mitigation: Table-driven merge tests covering all override combinations.
