## WP-003: Env/File Credential Resolver and go-git Auth Builder

### Metadata

```yaml
WP_ID: WP-003
Title: Env/File Credential Resolver and go-git Auth Builder
Domain: Service Layer
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-07
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Private repo support depends first on operator-managed secret references. Before sync integration changes land, SkillServer needs a typed credential resolver for env/file sources plus a safe translation layer into go-git auth methods.

**Scope:**
- Add provider resolution for `env` and `file` credential sources.
- Build go-git auth for `none`, `https_token`, `https_basic`, and `ssh_key`.
- Validate required refs per auth mode, including `known_hosts` support for SSH.
- Centralize secret-safe redaction for resolver and auth-builder errors.

Excluded:
- Stored-secret provider logic.
- Syncer orchestration changes.
- REST/UI request handling.

**Success Criteria:**
- [ ] Env/file refs resolve just-in-time from process env or mounted files.
- [ ] SSH auth requires explicit host verification material.
- [ ] Resolver/auth-builder errors are safe to show in logs and API status.

---

### Technical Requirements

**Input Contracts:**
- Expanded repo auth descriptor from WP-002.
- go-git clone/pull auth expectations in `pkg/git/syncer.go`.

**Output Contracts:**
- New credential resolver/auth builder package or files under `pkg/git/`.
- Redaction helpers reusable by syncer and API layers.

**Integration Points:**
- WP-005 consumes the resolver/auth builder on every sync path.
- WP-008 verifies resolver behavior across modes and failure cases.

---

### Deliverables

**Code Deliverables:**
- [ ] Add resolver interfaces/types for env/file lookups under `pkg/git/`.
- [ ] Add auth builder helpers for HTTPS token/basic and SSH key flows.
- [ ] Validate passphrase and `known_hosts` requirements for `ssh_key`.
- [ ] Add reusable redaction helpers for auth-related errors.

**Test Deliverables:**
- [ ] Unit tests for all supported auth modes and source combinations.
- [ ] Negative tests for missing refs, unreadable files, and unsupported auth/source pairings.

---

### Acceptance Criteria

**Functional:**
- [ ] `env` and `file` sources resolve credentials only at sync time, not at config-save time.
- [ ] Missing or malformed refs return redacted, actionable errors.
- [ ] SSH auth never falls back to insecure host-key ignore behavior.

**Testing:**
- [ ] Resolver tests cover env success/failure, file success/failure, and redacted error strings.
- [ ] Auth-builder tests cover HTTPS token, HTTPS basic, SSH key with known_hosts, and public `none` mode.

---

### Dependencies

**Blocked By:**
- WP-002

**Blocks:**
- WP-005
- WP-008

**Parallel Execution:**
- Can run in parallel with: WP-004 (once WP-002 is complete)
- Cannot run in parallel with: WP-002

---

### Risks

**Risk 1: go-git auth behavior differs subtly between clone and pull**
- Probability: Medium
- Impact: Medium
- Mitigation: Standardize auth creation in one builder used by both code paths.

**Risk 2: Redaction rules miss provider-specific leakage cases**
- Probability: Medium
- Impact: High
- Mitigation: Add explicit tests for token-like strings, userinfo URLs, and file-path error messages.

