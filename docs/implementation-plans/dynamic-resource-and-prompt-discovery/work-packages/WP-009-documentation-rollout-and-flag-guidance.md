## WP-009: Documentation and Rollout Controls

### Metadata

```yaml
WP_ID: WP-009
Title: Documentation and Rollout Controls
Domain: Documentation
Priority: Medium
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-02
Started_Date: 2026-03-03
Completed_Date: 2026-03-03
```

---

### Description

**Context:**
ADR-002 changes user-visible resource semantics and adds operational rollback expectations. Documentation and rollout controls must be updated before enabling broadly.

**Scope:**
- Update `README.md` for prompt directories and imported discovery behavior.
- Document additive REST and MCP resource metadata.
- Add rollout notes and rollback instructions including optional import-discovery flag.

Excluded:
- New feature development.

**Success Criteria:**
- [x] User docs reflect new directory and imported resource behavior.
- [x] Rollback path is documented and actionable.
- [x] Plan status can move from PLANNING to IN_PROGRESS with clear execution order.

---

### Technical Requirements

**Input Contracts:**
- Test evidence from WP-008.

**Output Contracts:**
- README/API docs updated.
- Operational notes under `docs/operations/` if needed.
- Work package index and plan links verified.

**Integration Points:**
- Final step before implementation execution handoff.

---

### Deliverables

**Documentation Deliverables:**
- [x] Update `README.md` resource directory and endpoint/tool sections.
- [x] Add notes on imported resources (`origin`, `writable`, `imports/...`).
- [x] Add rollout and rollback guidance for import discovery behavior.

---

### Acceptance Criteria

**Functional:**
- [x] Documentation matches implemented behavior and test evidence.
- [x] Rollback instructions are concrete and executable.

**Review:**
- [x] Documentation reviewed for clarity and compatibility messaging.

---

### Dependencies

**Blocked By:**
- WP-008

**Blocks:**
- None

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-008

---

### Risks

**Risk 1: Docs lag implementation details**
- Probability: Medium
- Impact: Medium
- Mitigation: Tie updates directly to tested behavior and command evidence from WP-008.

---

### Execution Evidence (2026-03-03)

- Updated runtime rollback controls in code:
  - `cmd/skillserver/main.go`
  - `pkg/domain/manager.go`
- Updated regression coverage for rollback behavior:
  - `pkg/domain/resources_test.go`
- Updated user-facing docs:
  - `/home/jeff/skillserver/README.md`
  - `/home/jeff/skillserver/docs/operations/dynamic-resource-import-discovery-rollout.md`
- Updated plan status progression in:
  - `/home/jeff/skillserver/docs/implementation-plans/dynamic-resource-and-prompt-discovery/dynamic-resource-and-prompt-discovery-implementation-plan.md`

Validation commands:
- `go test ./pkg/domain -count=1`
- `go test ./pkg/mcp -count=1`
- `go test ./pkg/web -count=1`
- `go test ./cmd/skillserver -count=1`
- `npm run test:playwright`
