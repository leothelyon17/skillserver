## WP-004: Encrypted Git Credential Store and Key Management

### Metadata

```yaml
WP_ID: WP-004
Title: Encrypted Git Credential Store and Key Management
Domain: Data Layer
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
Stored credentials are optional but required to satisfy trusted GUI-based secret entry. They need to reuse the existing SQLite persistence runtime, store only ciphertext plus metadata, and keep the master key entirely outside the database.

**Scope:**
- Add SQLite schema/migration support for encrypted git repo credentials.
- Add row models and repository methods under `pkg/persistence`.
- Implement master-key loading and authenticated encryption helpers.
- Define stored-provider lookup semantics keyed by repo `id`.

Excluded:
- API/UI secret submission.
- Syncer orchestration changes.
- Env/file provider logic.

**Success Criteria:**
- [ ] Stored credentials persist only encrypted blobs plus key/version metadata.
- [ ] Stored-secret mode cannot operate without a validated master key.
- [ ] Repository methods support create, replace, read, and delete without exposing plaintext beyond process memory.

---

### Technical Requirements

**Input Contracts:**
- Runtime capability and key-loading guardrails from WP-001.
- Stable repo identity/config contract from WP-002.
- Existing SQLite bootstrap/migration patterns in `pkg/persistence`.

**Output Contracts:**
- New `git_repo_credentials` persistence schema and repository APIs.
- Encryption helpers suitable for token/basic/SSH payloads stored as typed JSON blobs.

**Integration Points:**
- WP-005 consumes stored-provider reads during sync.
- WP-006 consumes repository writes/rotations for add/update flows.
- WP-009 documents backup, key rotation, and rollback implications.

---

### Deliverables

**Code Deliverables:**
- [ ] Add SQLite migration updates for `git_repo_credentials`.
- [ ] Add row models and repository methods under `pkg/persistence/`.
- [ ] Add encryption/decryption helpers keyed from the runtime master key.
- [ ] Add key-version metadata to support future rotation workflows.

**Test Deliverables:**
- [ ] Repository tests for insert/read/update/delete.
- [ ] Encryption tests ensuring plaintext never appears in stored rows.
- [ ] Startup/runtime tests for missing or mismatched master keys.

---

### Acceptance Criteria

**Functional:**
- [ ] Stored credential payloads can represent token, basic-auth, and SSH key material without schema changes.
- [ ] Database rows contain ciphertext, nonce, and key metadata but no raw secret fields.
- [ ] Decryption failures are surfaced as redacted operational errors.

**Testing:**
- [ ] Migration tests cover new schema bootstrap on fresh and existing persistence databases.
- [ ] Repository tests verify overwrite/delete behavior for repo-bound secret records.

---

### Dependencies

**Blocked By:**
- WP-001
- WP-002

**Blocks:**
- WP-005
- WP-006
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-003
- Cannot run in parallel with: WP-001, WP-002

---

### Risks

**Risk 1: Key management semantics are underspecified and create unrecoverable failures**
- Probability: Medium
- Impact: High
- Mitigation: Version the encrypted payloads and document recovery/rotation constraints in WP-009.

**Risk 2: Stored-secret persistence couples too tightly to unrelated catalog tables**
- Probability: Low
- Impact: Medium
- Mitigation: Keep a dedicated repository/table and reuse only the SQLite bootstrap/runtime plumbing.

