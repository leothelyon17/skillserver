## WP-001: Shared Catalog Export Service

### Metadata

```yaml
WP_ID: WP-001
Title: Shared Catalog Export Service
Domain: Service Layer
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-08
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
The current GUI export path depends on a legacy wildcard REST route that directly calls `domain.ExportSkill`. ADR-007 requires one shared service seam that future REST, MCP, and UI flows can all reuse.

**Scope:**
- Introduce a shared export service abstraction in `pkg/domain/` for catalog-item export requests.
- Preserve current skill archive behavior by adapting `pkg/domain/archive.go` behind the new service.
- Define export request/result types that later work packages can extend for prompts and rules.

Excluded:
- Project-folder materialization writes (WP-007).
- Rule classifier expansion (WP-003, WP-006).
- REST, MCP, and UI adapters (WP-002, WP-008, WP-009, WP-010).

**Success Criteria:**
- [ ] Export logic is no longer route-coupled.
- [ ] Existing skill archive behavior remains byte-compatible or file-compatible.
- [ ] Service contracts are reusable for later prompt/rule export flows.

---

### Technical Requirements

**Input Contracts:**
- Existing archive helper in `pkg/domain/archive.go`.
- Existing `SkillManager`/filesystem resolution behavior for skill IDs.

**Output Contracts:**
- New export service types and orchestration in `pkg/domain/`.
- Service tests proving archive compatibility and deterministic naming.

**Integration Points:**
- WP-002 delegates the legacy REST route and new export endpoint to this service.
- WP-007 reuses the same request/result model family when materialization is added.

---

### Deliverables

**Code Deliverables:**
- [ ] Add shared export service files in `pkg/domain/` (for example `catalog_export_service.go`).
- [ ] Refactor `pkg/domain/archive.go` so it becomes a helper behind the service instead of the primary public seam.
- [ ] Add request/result models for archive payload generation and dry-run manifests.

**Test Deliverables:**
- [ ] Add service tests in `pkg/domain/` covering local and git-backed skill exports.
- [ ] Verify exported archive contents remain compatible with current import expectations.

---

### Acceptance Criteria

**Functional:**
- [ ] The service can export an existing skill without using REST route logic.
- [ ] Archive file naming remains deterministic for local and repo-backed skills.
- [ ] Unsupported or missing item IDs fail with explicit errors.

**Testing:**
- [ ] Tests cover local skill export, git skill export, and missing-skill failures.
- [ ] Tests prove the refactor does not break `ImportSkill` compatibility.

---

### Dependencies

**Blocked By:**
- None

**Blocks:**
- WP-002
- WP-007

**Parallel Execution:**
- Can run in parallel with: WP-003
- Cannot run in parallel with: WP-002

---

### Risks

**Risk 1: Service refactor changes archive structure**
- Probability: Medium
- Impact: High
- Mitigation: Compare resulting tarball contents against current route behavior in tests.

**Risk 2: Export contract overfits skill-only behavior**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep request/result types classifier-agnostic even if only skill export is implemented first.
