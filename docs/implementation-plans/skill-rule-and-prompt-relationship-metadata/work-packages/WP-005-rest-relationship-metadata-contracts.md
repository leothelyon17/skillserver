## WP-005: REST Relationship Metadata Contracts

### Metadata

```yaml
WP_ID: WP-005
Title: REST Relationship Metadata Contracts
Domain: backend
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong local match for Echo handler work, DTO design, validation, and additive API contract changes.
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
The GUI metadata workflow needs an additive REST surface for reading and writing relationships while keeping current list/search payload size and metadata overlay semantics stable.

**Scope:**
- Extend metadata/detail REST responses with additive relationship data.
- Add `PATCH /api/catalog/:id/relationships` for skill-owned edits.
- Keep list/search responses unchanged with respect to relationship detail.
- Map service validation and not-found errors into stable HTTP responses.

Excluded:
- MCP tool surface (WP-006).
- UI implementation in `pkg/web/ui/index.html` (WP-007).

**Success Criteria:**
- [x] Metadata responses include normalized relationship data.
- [x] Skill relationship PATCH validates payload shape and rejects non-skill writes.
- [x] Existing metadata overlay endpoints remain additive and backward compatible.

---

### Technical Requirements

**Input Contracts:**
- Relationship service from WP-004.
- Finalized relationship contract in `skill-rule-and-prompt-relationship-metadata-implementation-plan.md` section `WP-001 Finalized Relationship Contract (Authoritative)`.
- Existing metadata handlers and DTOs in `pkg/web/handlers.go` and route registration in `pkg/web/server.go`.

**Output Contracts:**
- New/updated REST DTOs for relationship reads and skill-owned patch requests.
- Stable error mapping for invalid item IDs, invalid classifiers, not found, and cardinality conflicts.

**Integration Points:**
- WP-007 uses these endpoints directly.
- WP-008 validates HTTP contract parity and backward compatibility.

---

### Deliverables

**Code Deliverables:**
- [x] Extend metadata response structs in `pkg/web/handlers.go` with additive `relationships` fields.
- [x] Add request decoding and handler logic for `PATCH /api/catalog/:id/relationships`.
- [x] Register the new route in `pkg/web/server.go`.
- [x] Add tests in `pkg/web/handlers_catalog_relationship_test.go` and update metadata handler tests as needed.

**Test Deliverables:**
- [x] Metadata GET tests for skill, prompt, and rule items with relationship payloads.
- [x] Relationship PATCH happy-path test for skill items.
- [x] Validation tests for invalid target classifier, duplicate IDs, unknown items, and non-skill write targets.
- [x] Compatibility test proving `GET /api/catalog` and `GET /api/catalog/search` remain relationship-light.

---

### Acceptance Criteria

**Functional:**
- [x] `GET /api/catalog/:id/metadata` and `GET /api/catalog/metadata` return additive relationship data.
- [x] `PATCH /api/catalog/:id/relationships` supports prompt replacement and rule-set replacement for skills.
- [x] Prompt and rule metadata views expose reverse `skills` data but reject write attempts on the relationship endpoint.

**Testing:**
- [x] Handler tests cover happy path, validation errors, not found, and transport-level compatibility.
- [x] Existing catalog metadata tests remain green.

---

### Dependencies

**Blocked By:**
- WP-001
- WP-004

**Blocks:**
- WP-007
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-006 after WP-004 is complete.
- Cannot run in parallel with: WP-007.

---

### Risks

**Risk 1: Relationship fields accidentally leak into list/search payloads**
- Probability: Low
- Impact: Medium
- Mitigation: Keep additive fields scoped to metadata/detail DTOs only.

**Risk 2: Error mapping differs from taxonomy/metadata patterns**
- Probability: Medium
- Impact: Medium
- Mitigation: Reuse existing handler error style and preserve additive contract semantics.
