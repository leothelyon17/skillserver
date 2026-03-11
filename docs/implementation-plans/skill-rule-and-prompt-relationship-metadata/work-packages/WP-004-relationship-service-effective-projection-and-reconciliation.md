## WP-004: Relationship Service, Effective Projection, and Reconciliation

### Metadata

```yaml
WP_ID: WP-004
Title: Relationship Service, Effective Projection, and Reconciliation
Domain: backend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for Go domain services, validation rules, projection design, and runtime reconciliation wiring.
Priority: High
Estimated_Effort: 6 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-11
Started_Date: 2026-03-11
Completed_Date: 2026-03-11
```

---

### Description

**Context:**
ADR-008 needs one shared relationship projection so REST and MCP do not drift, plus service-level validation to enforce classifier and cardinality rules above raw persistence.

**Scope:**
- Add a domain service that:
  - validates skill->rule and skill->prompt writes against source-item classifiers
  - enforces one prompt per skill
  - resolves forward and reverse relationship views
  - suppresses missing or soft-deleted related endpoints
- Extend metadata/detail domain views to carry additive relationship data.
- Add reconciliation support so stale rows can be pruned during startup or manual sync flows.

Excluded:
- REST route/DTO work (WP-005).
- MCP tool registration and schemas (WP-006).
- UI picker/rendering logic (WP-007).

**Success Criteria:**
- [x] One domain-level relationship view is reusable across REST and MCP.
- [x] Invalid classifier pairs and invalid item IDs fail before persistence.
- [x] Effective reads hide missing/deleted endpoints and keep reverse associations accurate.
- [x] Runtime reconciliation has a clear path to prune stale rows after sync cycles.

---

### Technical Requirements

**Input Contracts:**
- Finalized relationship contract in `skill-rule-and-prompt-relationship-metadata-implementation-plan.md` section `WP-001 Finalized Relationship Contract (Authoritative)`.
- Source/effective metadata patterns in `pkg/domain/catalog_metadata_service.go` and `pkg/domain/catalog_effective_service.go`.
- Relationship repositories from WP-003.
- Runtime sync coordination in `cmd/skillserver/persistence_catalog_runtime.go`.

**Output Contracts:**
- New service in `pkg/domain/catalog_relationship_service.go` with stable read/write methods.
- Additive relationship fields in metadata/detail domain views.
- Runtime reconciliation hook for stale relationship cleanup.

**Integration Points:**
- WP-005 consumes service read and patch methods.
- WP-006 consumes service read methods.
- WP-008 validates reconciliation behavior through integration tests.

---

### Deliverables

**Code Deliverables:**
- [x] Add `pkg/domain/catalog_relationship_service.go`.
- [x] Add `pkg/domain/catalog_relationship_service_test.go`.
- [x] Extend `pkg/domain/catalog_metadata_service.go` to surface additive relationship metadata.
- [x] Update runtime coordination in `cmd/skillserver/persistence_catalog_runtime.go` and related tests to support stale-row pruning or reconciliation.

**Test Deliverables:**
- [x] Classifier validation tests for skill/prompt/rule combinations.
- [x] Single-prompt-per-skill enforcement tests.
- [x] Reverse skill projection tests for prompt and rule items.
- [x] Soft-delete or missing-endpoint suppression tests.
- [x] Reconciliation/prune tests tied to sync/runtime flows.

---

### Acceptance Criteria

**Functional:**
- [x] Domain service provides one normalized relationship payload for all transports.
- [x] Service write methods reject non-skill write targets and invalid target classifiers.
- [x] Reads remain stable when related endpoints disappear from the effective catalog.

**Testing:**
- [x] Domain-service tests cover happy path, validation failures, and deleted-endpoint scenarios.
- [x] Runtime tests cover stale-row pruning or equivalent reconciliation behavior.

---

### Dependencies

**Blocked By:**
- WP-001
- WP-003

**Blocks:**
- WP-005
- WP-006
- WP-008

**Parallel Execution:**
- Can run in parallel with: None.
- Cannot run in parallel with: WP-005 and WP-006.

---

### Risks

**Risk 1: Projection logic duplicates effective catalog behavior and becomes inconsistent**
- Probability: Medium
- Impact: High
- Mitigation: Resolve endpoint visibility through shared source/effective reads instead of parallel ad hoc filtering.

**Risk 2: Reconciliation is implemented too late and stale rows leak into detail views**
- Probability: Medium
- Impact: Medium
- Mitigation: Make stale-row handling part of this package rather than a later cleanup package.
