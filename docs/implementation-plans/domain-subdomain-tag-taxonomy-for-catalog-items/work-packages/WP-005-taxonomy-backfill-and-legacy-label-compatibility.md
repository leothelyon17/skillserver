## WP-005: Taxonomy Backfill and Legacy Label Compatibility

### Metadata

```yaml
WP_ID: WP-005
Title: Taxonomy Backfill and Legacy Label Compatibility
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
ADR-005 requires migration continuity: existing metadata overlay `labels` should map into taxonomy tags without breaking current consumers.

**Scope:**
- Implement idempotent label-to-tag backfill flow:
  - normalize unique tag keys from legacy labels.
  - create missing taxonomy tags.
  - create item-tag assignments from overlay labels.
- Implement compatibility rule for effective output:
  - derive `labels` from taxonomy tags when assignments exist.
  - fallback to legacy overlay labels otherwise.
- Wire backfill execution into persistence startup/migration lifecycle.

Excluded:
- Registry CRUD transport surfaces (WP-006, WP-008, WP-009).
- UI controls (WP-010).

**Success Criteria:**
- [ ] Backfill converts legacy labels without duplicates.
- [ ] Backfill is safe to re-run.
- [ ] Effective `labels` output remains backward-compatible.

---

### Technical Requirements

**Input Contracts:**
- Taxonomy repositories from WP-002.
- Effective projection + assignment service from WP-004.

**Output Contracts:**
- Backfill runner callable during startup when persistence is enabled.
- Compatibility behavior in effective metadata projection.

**Integration Points:**
- WP-011 regression matrix verifies backfill and compatibility behavior.
- WP-012 rollout docs include backfill operational checks.

---

### Deliverables

**Code Deliverables:**
- [ ] Add backfill service/runner under `pkg/domain/` or persistence runtime integration path.
- [ ] Add key normalization helper for label-to-tag migration.
- [ ] Update effective projection compatibility logic for `labels` field behavior.
- [ ] Hook backfill runner in startup persistence flow (`cmd/skillserver/persistence_catalog_runtime.go`).

**Test Deliverables:**
- [ ] Backfill unit tests for normalization and dedupe cases.
- [ ] Idempotency tests for repeated backfill runs.
- [ ] Compatibility tests for taxonomy-derived vs legacy fallback labels.

---

### Acceptance Criteria

**Functional:**
- [ ] Existing labeled items have taxonomy tags after migration.
- [ ] No duplicate taxonomy tags created from equivalent labels.
- [ ] `labels` field remains populated correctly for old and new data.

**Testing:**
- [ ] Backfill and compatibility tests pass under persistence runtime.
- [ ] No regression in metadata overlay APIs.

---

### Dependencies

**Blocked By:**
- WP-002
- WP-004

**Blocks:**
- WP-011
- WP-012

**Parallel Execution:**
- Can run in parallel with: WP-006, WP-008 (after WP-004).
- Cannot run in parallel with: WP-002, WP-004.

---

### Risks

**Risk 1: Colliding normalized keys map different labels to same tag**
- Probability: Medium
- Impact: Medium
- Mitigation: Define deterministic normalization and conflict handling; log collisions.

**Risk 2: Startup backfill introduces noticeable boot delay**
- Probability: Low
- Impact: Medium
- Mitigation: Use incremental/idempotent backfill and short-circuit when no legacy labels remain.
