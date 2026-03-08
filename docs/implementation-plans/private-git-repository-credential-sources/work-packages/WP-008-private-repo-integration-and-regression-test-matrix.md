## WP-008: Private Repo Integration and Regression Test Matrix

### Metadata

```yaml
WP_ID: WP-008
Title: Private Repo Integration and Regression Test Matrix
Domain: Quality
Priority: High
Estimated_Effort: 6 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-07
Started_Date: 2026-03-07
Completed_Date: 2026-03-07
```

---

### Description

**Context:**
This feature crosses runtime config, persistence, sync orchestration, API, and UI behavior. The final verification package must prove both security properties and backward compatibility before rollout.

**Scope:**
- Add unit, integration, and handler regression coverage not owned by earlier WPs.
- Validate public-repo compatibility plus private-repo behavior across env/file/stored sources.
- Verify redaction guarantees and sync-path parity.

Excluded:
- README and ops documentation updates.
- Post-rollout monitoring.

**Success Criteria:**
- [x] Test coverage demonstrates one credential-resolution path across startup, periodic, and manual sync.
- [x] Public URL-only repo behavior remains green.
- [x] API/UI never echo secrets.

---

### Technical Requirements

**Input Contracts:**
- Final runtime/config/service/API/UI behavior from WP-003 through WP-007.
- Existing git config and web handler test suites.

**Output Contracts:**
- Expanded automated regression suite under `pkg/git`, `pkg/web`, `cmd/skillserver`, and `pkg/persistence`.
- Manual verification notes for flows that remain difficult to automate.

**Integration Points:**
- WP-009 uses validated behavior and known limitations in rollout/rollback docs.

---

### Deliverables

**Code Deliverables:**
- [x] Add regression coverage for config migration and stable ID behavior.
- [x] Add integration tests for sync-path parity and auth failure preservation.
- [x] Add handler tests for secret-free API responses.
- [x] Add persistence tests for stored-secret encryption/decryption and missing-key behavior.

**Verification Deliverables:**
- [x] Manual verification checklist for UI masking and SSH known_hosts flows where automated coverage is limited.

---

### Acceptance Criteria

**Functional:**
- [x] Public repo add/update/delete/toggle/sync regressions are covered.
- [x] Private repo env/file/stored flows are each validated at least once.
- [x] Rotated env/file secrets are picked up on later sync attempts.
- [x] Stored-secret failures do not reveal plaintext in test logs or responses.

**Testing:**
- [x] Test runs cover `cmd/skillserver`, `pkg/git`, `pkg/persistence`, and `pkg/web`.
- [x] Any remaining manual-only cases are documented with step-by-step verification notes.

---

### Dependencies

**Blocked By:**
- WP-003
- WP-004
- WP-005
- WP-006
- WP-007

**Blocks:**
- WP-009

**Parallel Execution:**
- Can run in parallel with: None
- Cannot run in parallel with: WP-003, WP-004, WP-005, WP-006, WP-007

---

### Risks

**Risk 1: Authenticated git integration is difficult to exercise deterministically in tests**
- Probability: Medium
- Impact: Medium
- Mitigation: Combine unit coverage for auth builders with targeted integration coverage using controlled fixtures and syncer fakes where appropriate.

**Risk 2: Security regressions hide outside the happy path**
- Probability: Medium
- Impact: High
- Mitigation: Add explicit negative tests for redaction, userinfo URLs, missing secrets, and decryption failures.
