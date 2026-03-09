## WP-002: Catalog Reference Normalizer and Classification-State Domain Model

### Metadata

```yaml
WP_ID: WP-002
Title: Catalog Reference Normalizer and Classification-State Domain Model
Domain: backend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong match for Go backend contract shaping and domain-service implementation.
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
The codebase already has multiple normalization paths for export/materialization, but taxonomy reads and writes still expect raw `item_id` values. Classification completeness is also implied by omitted fields instead of being modeled directly.

**Scope:**
- Add shared helpers for accepting:
  - bare skill IDs
  - canonical `skill:<path>` IDs
- Reuse the normalizer across taxonomy, export, and materialization flows where skill item references are accepted.
- Add explicit classification-state fields to:
  - effective catalog items
  - metadata effective views
  - taxonomy assignment views
- Centralize completeness derivation so the logic is not duplicated across handlers.

Excluded:
- Repository pagination and usage query work (WP-003).
- REST/MCP request decoding and schema expansion (WP-005, WP-006).

**Success Criteria:**
- [x] Bare and canonical skill references resolve to the same canonical item ID.
- [x] Classification-state fields are present and deterministic for unclassified and partially classified items.
- [x] Export/materialization no longer use private one-off normalization rules for skills.

---

### Technical Requirements

**Input Contracts:**
- Canonical item-ID helpers in `pkg/domain/catalog.go`.
- Existing export/materialization normalization in `pkg/domain/catalog_export_service.go` and `pkg/domain/catalog_materialization_service.go`.
- Assignment/effective projection paths in `pkg/domain/catalog_taxonomy_assignment_service.go` and `pkg/domain/catalog_effective_service.go`.

**Output Contracts:**
- Shared normalization helpers exposed at the domain layer.
- Additive fields:
  - `has_assignment`
  - `is_fully_classified`
  - `missing_fields`

**Integration Points:**
- WP-004 consumes the normalizer and completeness model for mutation responses.
- WP-005 and WP-006 surface these domain changes over REST and MCP.

---

### Deliverables

**Code Deliverables:**
- [x] Add shared catalog item-reference normalization helpers in `pkg/domain/catalog.go`.
- [x] Refactor export/materialization code to use the shared helpers.
- [x] Extend taxonomy assignment and effective projection outputs with explicit completeness fields.
- [x] Extend metadata effective views to carry explicit completeness state.

**Test Deliverables:**
- [x] Add unit tests for legacy bare skill IDs and canonical skill IDs.
- [x] Add service tests for unclassified, partially classified, and fully classified items.
- [x] Add regression tests ensuring prompt and rule IDs remain canonical-only.

---

### Acceptance Criteria

**Functional:**
- [x] Skill-specific callers can submit either accepted ID shape without ambiguity.
- [x] Classification completeness is explicit instead of inferred from `omitempty`.
- [x] Missing-field output is stable across direct taxonomy reads and effective catalog views.

**Testing:**
- [x] Domain tests cover both compatibility and completeness semantics.
- [x] Existing export/materialization tests pass after refactor.

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
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-003 after WP-001.
- Cannot run in parallel with: WP-004 onward.

---

### Risks

**Risk 1: Normalization is applied too broadly and changes prompt/rule semantics**
- Probability: Medium
- Impact: High
- Mitigation: Restrict compatibility normalization to skill references only and keep prompt/rule paths canonical.

**Risk 2: Completeness rules become duplicated again in metadata or handler code**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep one shared derivation helper in the domain layer and ban ad hoc handler-side logic.
