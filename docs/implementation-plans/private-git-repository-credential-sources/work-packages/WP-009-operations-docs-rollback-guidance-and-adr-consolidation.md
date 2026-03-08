## WP-009: Operations Docs, Rollback Guidance, and ADR Consolidation

### Metadata

```yaml
WP_ID: WP-009
Title: Operations Docs, Rollback Guidance, and ADR Consolidation
Domain: Documentation
Priority: Medium
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-07
Started_Date: 2026-03-07
Completed_Date: 2026-03-07
```

---

### Description

**Context:**
The feature changes operator workflows and currently has two ADR files describing overlapping contracts. Rollout documentation needs to explain the supported deployment patterns clearly and leave the repo with one authoritative ADR.

**Scope:**
- Update README and operations guidance for public repos, env/file private repos, and stored-secret mode.
- Document local, Docker, Kubernetes Secret, and Vault-projected env/file examples.
- Document master-key handling, rotation caveats, rollback steps, and safe UI deployment prerequisites.
- Consolidate or cross-link the duplicate ADR files so future work references one canonical decision record.

Excluded:
- New infrastructure automation.
- Direct Vault client implementation.

**Success Criteria:**
- [x] Operators have explicit setup examples for each supported credential source.
- [x] Rollback steps cover disabling stored secrets and falling back to env/file or public mode.
- [x] The duplicate ADR ambiguity is removed from the docs tree.

---

### Technical Requirements

**Input Contracts:**
- Verified runtime/API behavior from WP-001 through WP-008.
- Existing README and operations documentation structure.

**Output Contracts:**
- Updated README and at least one ops/runbook document under `docs/operations/`.
- Resolved ADR cross-linking or duplicate removal plan in `docs/adrs/`.

**Integration Points:**
- Final rollout depends on validated behavior from WP-008.

---

### Deliverables

**Documentation Deliverables:**
- [x] Update `README.md` with private repo examples and security caveats.
- [x] Add or update operations guidance covering:
  - local setup
  - Docker env/file injection
  - Kubernetes Secret env/file patterns
  - Vault Agent or CSI projected env/file patterns
  - stored-secret prerequisites and rollback
- [x] Consolidate the ADR duplication by keeping one canonical ADR-006 path and cross-referencing or removing the shorter draft.

**Verification Deliverables:**
- [x] Validate documentation examples against the final tested contract from WP-008.

---

### Acceptance Criteria

**Functional:**
- [x] Docs state clearly that env/file sources are the preferred production path.
- [x] Docs state clearly that stored-secret mode requires persistence, a master key, TLS, and an external auth boundary.
- [x] Rollback steps cover disabling stored credentials without breaking public repos.
- [x] Repo readers can identify one authoritative ADR-006 document after completion.

**Testing:**
- [x] Example config snippets match final runtime variable names and API contract.
- [x] Rollback instructions are reviewed against the implemented behavior.

---

### Dependencies

**Blocked By:**
- WP-001
- WP-002
- WP-004
- WP-006
- WP-008

**Blocks:**
- None

**Parallel Execution:**
- Can run in parallel with: Early draft work only
- Cannot run in parallel with: Final closeout before WP-008 completes

---

### Risks

**Risk 1: Docs drift from final variable names or contract details**
- Probability: Medium
- Impact: Medium
- Mitigation: Do not finalize examples until WP-008 confirms the shipped contract.

**Risk 2: Duplicate ADR cleanup loses useful context**
- Probability: Low
- Impact: Low
- Mitigation: Keep a short cross-reference note or superseded marker instead of deleting history without trace.
