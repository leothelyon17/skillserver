## WP-008: Rollout Validation + Rollback Runbook

### Metadata

```yaml
WP_ID: WP-008
Title: Rollout Validation + Rollback Runbook
Domain: Release Engineering
Priority: Medium
Estimated_Effort: 2 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-02
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Deployment teams need a deterministic release checklist and rollback flow for MCP transport changes.

**Scope:**
- Create rollout checklist doc.
- Define smoke tests and pass/fail gates.
- Define rollback to stdio-only configuration.

Excluded:
- Automated deployment pipeline implementation.

**Success Criteria:**
- [ ] Operators can execute rollout and rollback without additional engineering decisions.

---

### Technical Requirements

**Input Contracts:**
- Test evidence from WP-005 and WP-006.
- Documentation updates from WP-007.

**Output Contracts:**
- `docs/operations/mcp-streamable-http-rollout.md` runbook.

**Integration Points:**
- References ADR success metrics and runtime configuration.

---

### Deliverables

**Documentation Deliverables:**
- [ ] Add rollout checklist (`pre-deploy`, `canary`, `full rollout`).
- [ ] Add smoke-test commands for `/mcp` and stdio compatibility.
- [ ] Add hard rollback sequence:
  1. Set `SKILLSERVER_MCP_TRANSPORT=stdio`
  2. Redeploy
  3. Remove `/mcp` ingress routing

---

### Acceptance Criteria

- [ ] Runbook contains clear go/no-go gates.
- [ ] Runbook includes operational observability checks.
- [ ] Runbook includes full rollback criteria and steps.

---

### Testing Strategy

**Operational Validation:**
- [ ] Dry-run checklist reviewed by at least one operator.
- [ ] Smoke tests verified in staging instructions.

---

### Dependencies

**Blocked By:**
- WP-005
- WP-006
- WP-007

**Blocks:**
- Production rollout

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: all upstream WPs

---

### Risks

**Risk 1: Rollout gate criteria too vague**
- Probability: Medium
- Impact: High
- Mitigation: include explicit metrics thresholds and fail conditions.

---

### Notes

Runbook should not assume any specific cloud provider beyond standard HTTP ingress/TLS controls.
