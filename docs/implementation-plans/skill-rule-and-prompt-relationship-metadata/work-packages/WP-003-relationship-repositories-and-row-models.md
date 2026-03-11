## WP-003: Relationship Repositories and Row Models

### Metadata

```yaml
WP_ID: WP-003
Title: Relationship Repositories and Row Models
Domain: data
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/database-architect.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for row-model validation, query filters, and deterministic repository behavior.
Priority: High
Estimated_Effort: 4 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-11
Started_Date: 2026-03-11
Completed_Date: 2026-03-11
```

---

### Description

**Context:**
The domain and transport layers need stable repository contracts for forward and reverse relationship lookups, prompt replace semantics, and stale-row cleanup without duplicating SQL in multiple packages.

**Scope:**
- Add row models and list filters for skill-rule and skill-prompt relationships.
- Add repositories around the new tables with deterministic ordering and targeted query helpers.
- Support repository operations needed by the domain service:
  - list by skill
  - list by prompt/rule
  - replace skill rule sets
  - set/clear skill prompt
  - delete/prune rows by endpoint item ID

Excluded:
- Domain classifier validation and source-item lookups (WP-004).
- REST or MCP mapping logic (WP-005/WP-006).

**Success Criteria:**
- [x] Repository contracts are sufficient for forward and reverse relationship projection.
- [x] Replace/upsert semantics are deterministic and easy for the domain service to consume.
- [x] Repository tests cover duplicate handling, ordering, and cleanup helpers.

---

### Technical Requirements

**Input Contracts:**
- Relationship schema from WP-002.
- Finalized relationship contract in `skill-rule-and-prompt-relationship-metadata-implementation-plan.md` section `WP-001 Finalized Relationship Contract (Authoritative)`.
- Existing repository patterns in `pkg/persistence/catalog_taxonomy_assignment_repository.go`.

**Output Contracts:**
- New persistence files for relationship row models and repositories.
- Stable repository errors for missing prompt relationships or empty-result list calls.

**Integration Points:**
- WP-004 consumes these repositories directly.
- WP-008 depends on repository correctness through domain-service regression tests.

---

### Deliverables

**Code Deliverables:**
- [x] Add row-model definitions in `pkg/persistence/catalog_relationship_row_models.go`.
- [x] Add repository implementations in `pkg/persistence/catalog_relationship_repository.go`.
- [x] Add repository tests in `pkg/persistence/catalog_relationship_repository_test.go`.

**Test Deliverables:**
- [x] Skill->rules replace/list tests.
- [x] Skill->prompt set/get/clear tests.
- [x] Reverse lookup tests for prompt->skills and rule->skills.
- [x] Cleanup helper tests for deleted endpoint pruning.

---

### Acceptance Criteria

**Functional:**
- [x] Repository APIs support both forward and reverse traversal.
- [x] Repository results are deterministically ordered for stable REST/MCP responses.
- [x] Cleanup helpers can remove rows for deleted skills, prompts, or rules without manual SQL duplication elsewhere.

**Testing:**
- [x] Repository unit tests pass on SQLite.
- [x] No migration or unrelated persistence tests regress.

---

### Dependencies

**Blocked By:**
- WP-002

**Blocks:**
- WP-004
- WP-008

**Parallel Execution:**
- Can run in parallel with: None.
- Cannot run in parallel with: WP-004.

---

### Risks

**Risk 1: Repository API is too thin and leaks SQL concerns upward**
- Probability: Medium
- Impact: Medium
- Mitigation: Include replace, reverse-list, and prune helpers in the repository layer from the start.

**Risk 2: Ordering differs between query paths**
- Probability: Medium
- Impact: Low
- Mitigation: Enforce explicit `ORDER BY` rules and cover them with tests.
