## WP-006: Secret-Safe Git Repo API Contracts and Handlers

### Metadata

```yaml
WP_ID: WP-006
Title: Secret-Safe Git Repo API Contracts and Handlers
Domain: API Layer
Priority: High
Estimated_Effort: 5 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-07
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Current repo handlers accept only `url` and `enabled` and echo raw URLs back to the browser. Private repo support needs additive auth fields, write-only secret submission, and stable-ID semantics while keeping legacy public-repo clients working.

**Scope:**
- Extend add/update payloads with auth descriptors and write-only stored-secret fields.
- Return masked auth summaries and redacted sync status in repo responses.
- Reuse canonical URL validation and stable `id` semantics across list/add/update/delete/toggle/sync flows.
- Surface stored-secret capability state to clients.

Excluded:
- Browser form implementation.
- README and ops docs.

**Success Criteria:**
- [ ] Legacy URL-only payloads still work.
- [ ] API responses never include secret values.
- [ ] Userinfo-bearing URLs are rejected consistently on create and update.

---

### Technical Requirements

**Input Contracts:**
- Runtime capability from WP-001.
- Repo model from WP-002.
- Stored-credential repository from WP-004.
- Syncer/status model from WP-005.

**Output Contracts:**
- Expanded request/response DTOs in `pkg/web/handlers.go`.
- Stable handler behavior keyed by repo `id`.

**Integration Points:**
- WP-007 consumes new DTOs and capability fields.
- WP-008 validates secret-free API behavior.

---

### Deliverables

**Code Deliverables:**
- [ ] Extend git repo request DTOs with `auth` metadata and write-only secret fields.
- [ ] Extend response DTOs with safe auth summary and sync-status fields.
- [ ] Add canonical URL, auth-mode/source, and stored-secret capability validation in handlers.
- [ ] Update CRUD/toggle/sync handlers to use the typed repo contract and stable IDs consistently.

**Test Deliverables:**
- [ ] API tests for add/update/list/delete/toggle/sync with public and private repo payloads.
- [ ] Tests ensuring responses never echo write-only secrets or userinfo-bearing URLs.

---

### Acceptance Criteria

**Functional:**
- [ ] `GET /api/git-repos` returns only canonical URLs and masked auth summaries.
- [ ] `POST`/`PUT` reject unsupported auth mode/source combinations with actionable errors.
- [ ] Stored-secret writes are rejected when capability is disabled.
- [ ] Sync responses include redacted status while remaining backward-compatible for existing clients.

**Testing:**
- [ ] Handler tests cover public repo compatibility, private repo validation, and stored-secret gating.
- [ ] Contract tests verify no response body contains submitted secret values.

---

### Dependencies

**Blocked By:**
- WP-001
- WP-002
- WP-004
- WP-005

**Blocks:**
- WP-007
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: None
- Cannot run in parallel with: WP-001, WP-002, WP-004, WP-005

---

### Risks

**Risk 1: Backward compatibility breaks existing UI or automation**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep `id`, `url`, `name`, and `enabled` in responses and treat new fields as additive.

**Risk 2: Handler branching becomes inconsistent across CRUD/toggle/sync endpoints**
- Probability: Medium
- Impact: Medium
- Mitigation: Centralize repo lookup, validation, and response-shaping helpers instead of duplicating logic in each route.

