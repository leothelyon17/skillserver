## WP-007: Documentation Update (User + Operator)

### Metadata

```yaml
WP_ID: WP-007
Title: Documentation Update (User + Operator)
Domain: Documentation
Priority: Medium
Estimated_Effort: 3 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-02
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Transport changes add multiple runtime options and usage patterns. Documentation must prevent deployment/config mistakes.

**Scope:**
- Update README with new env vars/flags.
- Add stdio/http/both usage examples.
- Add troubleshooting guidance for common MCP HTTP issues.

Excluded:
- Deep security hardening docs beyond perimeter guidance.

**Success Criteria:**
- [ ] README is complete for all new options and transport modes.
- [ ] Remote MCP usage examples are present and actionable.

---

### Technical Requirements

**Input Contracts:**
- Final runtime behavior from WP-004.

**Output Contracts:**
- Updated `README.md`.

**Integration Points:**
- WP-008 rollout runbook references README configuration guidance.

---

### Deliverables

**Documentation Deliverables:**
- [ ] Add all new env vars and defaults.
- [ ] Add all new flags and precedence note.
- [ ] Add transport mode examples:
  - stdio
  - http
  - both
- [ ] Add troubleshooting section for:
  - session initialization issues
  - header/protocol mismatch
  - route conflict symptoms
  - quick rollback to stdio mode

---

### Acceptance Criteria

- [ ] README contains full option matrix with defaults.
- [ ] README examples reflect actual runtime behavior.
- [ ] Troubleshooting section includes concrete diagnosis/remediation steps.

---

### Testing Strategy

**Documentation Validation:**
- [ ] Verify commands/examples are syntactically valid.
- [ ] Verify environment variable names exactly match implementation.

---

### Dependencies

**Blocked By:**
- WP-004

**Blocks:**
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-005, WP-006
- Cannot run in parallel with: none

---

### Risks

**Risk 1: Documentation drift from implemented behavior**
- Probability: Medium
- Impact: Medium
- Mitigation: update docs only after final runtime semantics are merged.

---

### Notes

Keep README backward-compatible by preserving existing stdio client sections.
