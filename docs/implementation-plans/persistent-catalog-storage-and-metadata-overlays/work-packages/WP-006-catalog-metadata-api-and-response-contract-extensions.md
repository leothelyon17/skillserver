## WP-006: Catalog Metadata API and Response Contract Extensions

### Metadata

```yaml
WP_ID: WP-006
Title: Catalog Metadata API and Response Contract Extensions
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
Clients need additive APIs to read and mutate overlay metadata without changing content mutability behavior.

**Scope:**
- Add `PATCH /api/catalog/:id/metadata`.
- Add `GET /api/catalog/:id/metadata`.
- Extend catalog list/search payloads with additive mutability fields.
- Validate metadata payload size/shape and reject invalid input.

Excluded:
- UI controls and flows (WP-008).
- Startup sync runtime wiring (WP-007).

**Success Criteria:**
- [ ] Metadata PATCH updates overlay rows for any catalog item.
- [ ] GET helper returns source, overlay, and effective values.
- [ ] Catalog list/search responses include mutability fields with compatibility preserved.

---

### Technical Requirements

**Input Contracts:**
- Effective projection service from WP-005.
- Synced source state from WP-004.

**Output Contracts:**
- Handler changes in `pkg/web/handlers.go` and route wiring in `pkg/web/server.go`.
- Request/response DTO definitions for metadata overlay APIs.

**Integration Points:**
- WP-008 UI consumes new metadata APIs.
- WP-009 regression suite validates contract behavior.

---

### Deliverables

**Code Deliverables:**
- [ ] Add metadata request DTOs (display name, description, labels, custom metadata).
- [ ] Implement overlay mutation handler with input validation.
- [ ] Implement metadata retrieval helper handler.
- [ ] Extend existing catalog response DTO with `content_writable` and `metadata_writable`.
- [ ] Keep legacy `read_only` field in response for compatibility.

**Test Deliverables:**
- [ ] API tests for metadata patch success/failure cases.
- [ ] API tests for unknown item IDs and validation failures.
- [ ] API tests for additive mutability fields on list/search responses.

---

### Acceptance Criteria

**Functional:**
- [ ] Metadata PATCH works for git and non-git items.
- [ ] Content mutability remains unchanged by metadata API.
- [ ] Existing catalog consumers continue to function with additive fields.

**Testing:**
- [ ] Endpoint tests cover happy path, invalid payload, and missing item.
- [ ] Contract tests verify both legacy and new fields are present and consistent.

---

### Dependencies

**Blocked By:**
- WP-004
- WP-005

**Blocks:**
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-007 (after WP-004 and WP-005 readiness)
- Cannot run in parallel with: WP-008

---

### Risks

**Risk 1: Additive API fields break strict clients**
- Probability: Low
- Impact: Medium
- Mitigation: Preserve existing fields and keep new fields additive only.

**Risk 2: Unbounded metadata payload causes abuse**
- Probability: Medium
- Impact: Medium
- Mitigation: Enforce size/type validation and reject oversized metadata documents.
