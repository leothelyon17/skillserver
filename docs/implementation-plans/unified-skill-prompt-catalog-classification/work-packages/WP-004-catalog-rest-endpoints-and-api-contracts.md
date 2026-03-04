## WP-004: Catalog REST Endpoints and API Contracts

### Metadata

```yaml
WP_ID: WP-004
Title: Catalog REST Endpoints and API Contracts
Domain: API Layer
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
The web API currently exposes skill-centric endpoints only. ADR-003 requires additive catalog list/search endpoints with classifier filtering.

**Scope:**
- Add `GET /api/catalog` for unified catalog listing.
- Add `GET /api/catalog/search?q=...&classifier=...` for classifier-aware search.
- Define catalog API response DTO with classifier and prompt metadata fields.
- Keep existing `/api/skills` handlers and contracts unchanged.

Excluded:
- UI adoption of new endpoints (WP-005).
- MCP parity tools (WP-007).

**Success Criteria:**
- [ ] Catalog endpoints return mixed items with classifier metadata.
- [ ] Invalid classifier values return clear validation errors.
- [ ] Existing skill routes continue unchanged.

---

### Technical Requirements

**Input Contracts:**
- Catalog manager methods from WP-003.
- Existing Echo server routing in `pkg/web/server.go` and handlers in `pkg/web/handlers.go`.

**Output Contracts:**
- New route registrations and handlers for catalog list/search.
- Additive response struct for catalog items.

**Integration Points:**
- Consumed by WP-005 UI.
- Used by WP-008 API regression tests.

---

### Deliverables

**Code Deliverables:**
- [ ] Add catalog response DTO(s) in `pkg/web/handlers.go`.
- [ ] Implement `listCatalog` and `searchCatalog` handlers.
- [ ] Register `/api/catalog` and `/api/catalog/search` in `pkg/web/server.go`.

**Test Deliverables:**
- [ ] Add web handler tests for catalog endpoints and classifier filtering.
- [ ] Add regression tests ensuring `/api/skills` behavior remains stable.

---

### Acceptance Criteria

**Functional:**
- [ ] `GET /api/catalog` returns skill and prompt entries.
- [ ] `GET /api/catalog/search` supports optional classifier filtering.
- [ ] Skill CRUD endpoints remain unaffected.

**Testing:**
- [ ] API tests cover valid and invalid classifier inputs, empty query handling, and compatibility.

---

### Dependencies

**Blocked By:**
- WP-003

**Blocks:**
- WP-005
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-006, WP-007
- Cannot run in parallel with: WP-003

---

### Risks

**Risk 1: Catalog API shape diverges from UI needs**
- Probability: Medium
- Impact: Medium
- Mitigation: Align DTO fields with UI requirements before implementation.

**Risk 2: Route conflicts with existing wildcard/static handling**
- Probability: Low
- Impact: Medium
- Mitigation: Keep route registration under `/api` and add route integration tests.
