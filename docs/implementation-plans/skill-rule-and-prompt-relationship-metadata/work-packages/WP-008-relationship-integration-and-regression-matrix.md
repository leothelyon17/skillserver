## WP-008: Relationship Integration and Regression Matrix

### Metadata

```yaml
WP_ID: WP-008
Title: Relationship Integration and Regression Matrix
Domain: integration
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-software-developer-system-prompt.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Best local fit for cross-surface contract validation spanning persistence, backend, MCP, and UI behavior.
Priority: High
Estimated_Effort: 5 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-11
Started_Date: 2026-03-11
Completed_Date: 2026-03-11
```

---

### Description

**Context:**
ADR-008 spans persistence, domain, REST, MCP, and UI. The release is only safe if those surfaces agree on semantics and do not regress existing metadata or catalog behavior.

**Scope:**
- Build a cross-surface regression matrix for:
  - migration + repository behavior
  - service validation and projection
  - REST metadata/read/write behavior
  - MCP read-tool behavior
  - UI load/save behavior or documented manual verification
- Verify Git-backed read-only content semantics remain unchanged.
- Verify `GET /api/catalog` and `GET /api/catalog/search` stay lean and additive.

Excluded:
- New feature implementation work outside test and verification code.

**Success Criteria:**
- [x] Relationship behavior is validated across all affected surfaces.
- [x] Existing metadata, taxonomy, and catalog discovery flows remain stable.
- [x] Test coverage catches deleted-endpoint and validation edge cases.

---

### Technical Requirements

**Input Contracts:**
- Completed work from WP-003 through WP-007.
- Existing regression harnesses in persistence, domain, web, MCP, and UI test suites.

**Output Contracts:**
- Stable regression coverage for relationship semantics and compatibility guarantees.
- Clear evidence for rollout docs in WP-009.

**Integration Points:**
- Feeds directly into WP-009 documentation and release readiness.

---

### Deliverables

**Test Deliverables:**
- [x] Extend persistence tests for repository and prune behavior if not fully covered in WP-003.
- [x] Extend domain tests for projection parity and deleted-endpoint handling.
- [x] Add REST handler tests for metadata/read-write compatibility.
- [x] Add MCP regression tests for the new read tool.
- [x] Add or update UI verification coverage, or record a manual UI checklist if automation is not feasible.

**Quality Deliverables:**
- [x] Capture an explicit matrix of scenarios and expected outcomes in the WP completion summary.

---

### Acceptance Criteria

**Functional:**
- [x] All affected surfaces agree on relationship projection semantics.
- [x] Skill-only write authority is verified end to end.
- [x] Deleted or missing related endpoints do not leak into effective relationship views.
- [x] Existing list/search and metadata overlay behaviors remain backward compatible.

**Testing:**
- [x] Automated tests pass for persistence, domain, REST, and MCP layers.
- [x] UI verification evidence exists, automated or manual.

---

### Dependencies

**Blocked By:**
- WP-003
- WP-004
- WP-005
- WP-006
- WP-007

**Blocks:**
- WP-009

**Parallel Execution:**
- Can run in parallel with: None.
- Cannot run in parallel with: Upstream implementation packages.

---

### Risks

**Risk 1: Cross-surface parity bugs are only caught after docs or rollout notes are published**
- Probability: Medium
- Impact: High
- Mitigation: Make this package a strict prerequisite for WP-009.

**Risk 2: UI verification is under-specified because the frontend is single-file and lightly automated**
- Probability: Medium
- Impact: Medium
- Mitigation: Require a manual checklist in the completion summary if automated coverage is not practical.
