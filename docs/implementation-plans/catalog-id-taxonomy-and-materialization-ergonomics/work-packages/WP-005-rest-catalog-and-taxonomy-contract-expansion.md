## WP-005: REST Catalog and Taxonomy Contract Expansion

### Metadata

```yaml
WP_ID: WP-005
Title: REST Catalog and Taxonomy Contract Expansion
Domain: integration
Execution_Agent_Prompt: /home/jeff/.codex/.astra-agents/prompts/principal-backend-developer.md
Agent_Selection_Source: local
Agent_Selection_Rationale: Strong fit for Echo handler contracts, request decoding, and additive REST API changes.
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
The REST layer is the main UI-facing surface and already exposes taxonomy CRUD, item taxonomy patch/get, and catalog list/search. It needs additive contract expansion without breaking current clients.

**Scope:**
- Extend `GET /api/catalog` and `GET /api/catalog/search` with:
  - `include_content`
  - `limit`
  - `cursor`
  - `unclassified`
  - `missing_primary_domain`
  - `missing_tags`
- Add explicit classification-state fields to catalog and taxonomy responses.
- Extend single-item taxonomy patch payloads with additive tag mutation fields.
- Add one batch taxonomy patch endpoint with `dry_run`.
- Add usage/preflight endpoints for domain, subdomain, and tag references.
- Improve conflict responses where the handler already has structured conflict data.

Excluded:
- UI consumption changes (WP-007).
- MCP transport/schema changes (WP-006).

**Success Criteria:**
- [x] REST list/search responses are metadata-first by default.
- [x] REST clients can query for unclassified and partially classified items.
- [x] Batch taxonomy mutation and usage/preflight are available through additive endpoints.

---

### Technical Requirements

**Input Contracts:**
- Domain normalization and classification-state outputs from WP-002.
- Mutation and usage services from WP-004.
- Existing Echo handlers in `pkg/web/handlers.go`.

**Output Contracts:**
- Additive REST DTO fields and query params.
- New batch mutation and usage/preflight routes with stable JSON shapes.

**Integration Points:**
- WP-007 uses these contracts directly.
- WP-008 verifies compatibility and regression coverage against these handlers.

---

### Deliverables

**Code Deliverables:**
- [x] Extend REST request decoding for new filters and pagination fields.
- [x] Add metadata-first response shaping with optional `include_content=true`.
- [x] Add batch taxonomy patch handler and route.
- [x] Add taxonomy usage/preflight handlers and routes.
- [x] Improve conflict encoding when service errors include structured usage context.

**Test Deliverables:**
- [x] Add REST tests for new filters and response fields.
- [x] Add REST tests for batch mutation dry-run and apply behavior.
- [x] Add REST tests for usage/preflight endpoints and conflict payloads.

---

### Acceptance Criteria

**Functional:**
- [x] REST list/search works with content omitted by default.
- [x] REST taxonomy responses include explicit completeness state.
- [x] Batch and single-item mutation flows are both supported.
- [x] Usage/preflight responses can be consumed without parsing error strings.

**Testing:**
- [x] Handler tests cover both compatibility and new additive contracts.
- [x] Existing REST callers remain compatible when they omit new query params and body fields.

---

### Dependencies

**Blocked By:**
- WP-002
- WP-003
- WP-004

**Blocks:**
- WP-007
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-006 after prerequisites complete.
- Cannot run in parallel with: WP-004.

---

### Risks

**Risk 1: Metadata-first defaults break callers that relied on inline content**
- Probability: Medium
- Impact: Medium
- Mitigation: Make content opt-in with `include_content=true` and keep dedicated content-read endpoints unchanged.

**Risk 2: Too many additive query params complicate validation**
- Probability: Medium
- Impact: Medium
- Mitigation: Centralize decoding and validation helpers in `pkg/web/handlers.go`.
