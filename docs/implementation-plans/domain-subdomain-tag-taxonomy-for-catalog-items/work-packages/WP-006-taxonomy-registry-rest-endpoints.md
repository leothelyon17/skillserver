## WP-006: Taxonomy Registry REST Endpoints

### Metadata

```yaml
WP_ID: WP-006
Title: Taxonomy Registry REST Endpoints
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
Operators need HTTP APIs to create, list, update, and delete taxonomy objects from UI and automation flows.

**Scope:**
- Add handlers and route wiring for domain/subdomain/tag CRUD endpoints.
- Validate request payloads and map domain/service errors to HTTP statuses.
- Return stable response shapes for UI and automation clients.

Excluded:
- Item taxonomy assignment endpoints (WP-007).
- MCP tool exposure (WP-008, WP-009).

**Success Criteria:**
- [ ] All taxonomy registry REST endpoints are reachable and validated.
- [ ] Handler error mapping is consistent across object types.
- [ ] CRUD responses are additive and backward-compatible.

---

### Technical Requirements

**Input Contracts:**
- Taxonomy service from WP-003.
- Effective projection taxonomy references from WP-004.

**Output Contracts:**
- New REST route registrations in `pkg/web/server.go`.
- New/extended handler DTOs in `pkg/web/handlers.go`.

**Integration Points:**
- WP-010 UI taxonomy manager consumes these endpoints.
- WP-011 API regression coverage validates behavior.

---

### Deliverables

**Code Deliverables:**
- [ ] Implement handlers for taxonomy domains/subdomains/tags list/create/update/delete.
- [ ] Register routes under `/api/catalog/taxonomy/*`.
- [ ] Add request validation and consistent error responses.

**Test Deliverables:**
- [ ] Handler tests for successful CRUD flows.
- [ ] Tests for validation, not-found, and conflict responses.

---

### Acceptance Criteria

**Functional:**
- [ ] REST registry endpoints expose full CRUD for taxonomy objects.
- [ ] Conflict errors surface actionable messages for in-use deletions.

**Testing:**
- [ ] API tests validate request/response contracts and status codes.
- [ ] Existing catalog metadata endpoints remain unaffected.

---

### Dependencies

**Blocked By:**
- WP-003
- WP-004

**Blocks:**
- WP-010
- WP-011

**Parallel Execution:**
- Can run in parallel with: WP-008.
- Cannot run in parallel with: WP-003, WP-004.

---

### Risks

**Risk 1: Route/DTO inconsistencies across domains/subdomains/tags**
- Probability: Medium
- Impact: Medium
- Mitigation: Use shared handler helpers and common validation mapping.

**Risk 2: Ambiguous conflict messages slow UI workflows**
- Probability: Medium
- Impact: Low
- Mitigation: Include object IDs and blocking assignment counts where available.
