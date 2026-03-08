## WP-001: Private Git Runtime Flags and Startup Guardrails

### Metadata

```yaml
WP_ID: WP-001
Title: Private Git Runtime Flags and Startup Guardrails
Domain: Infrastructure
Priority: High
Estimated_Effort: 3 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-07
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Stored credentials are intentionally higher risk than env/file secret references because the web repo-management surface is not authenticated in-process. Runtime enablement and validation must therefore be explicit, fail-safe, and discoverable by the API/UI.

**Scope:**
- Add runtime configuration parsing for stored-credential enablement and master-key sourcing.
- Require persistence mode plus master-key configuration before stored-secret mode is considered available.
- Expose runtime capability state so handlers/UI can hide stored-secret workflows when disabled.

Excluded:
- `.git-repos.json` schema evolution and repo ID migration.
- SQLite credential table creation.
- Credential resolver and syncer auth wiring.

**Success Criteria:**
- [ ] Stored-secret mode is disabled by default.
- [ ] Startup fails fast with actionable errors when stored-secret mode is enabled without persistence or a master key.
- [ ] API/UI can determine whether stored-secret workflows are allowed.

---

### Technical Requirements

**Input Contracts:**
- Existing startup/runtime flow in `cmd/skillserver/main.go`.
- Existing persistence startup guard patterns in `cmd/skillserver/persistence_runtime.go`.

**Output Contracts:**
- New runtime config parser/helper under `cmd/skillserver/` for stored-secret capability.
- Capability wiring accessible to repo handlers and UI bootstrap payloads.

**Integration Points:**
- WP-004 consumes validated key-loading/runtime capability.
- WP-006 and WP-007 depend on capability gating for stored-secret API/UI behavior.

---

### Deliverables

**Code Deliverables:**
- [ ] Add runtime config support for:
  - `SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS`
  - `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY`
  - `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE`
- [ ] Add validation enforcing persistence prerequisites when stored credentials are enabled.
- [ ] Add startup diagnostics that report capability state without printing secret material.

**Test Deliverables:**
- [ ] Unit tests for valid and invalid runtime permutations.
- [ ] Tests proving stored-secret mode remains unavailable by default.

---

### Acceptance Criteria

**Functional:**
- [ ] Public repo startup behavior is unchanged when stored-secret mode is disabled.
- [ ] Invalid stored-secret configuration fails before the server starts.
- [ ] Capability state is available to downstream API/UI wiring without exposing secrets.

**Testing:**
- [ ] Runtime parsing tests cover env, file, missing-key, and missing-persistence scenarios.
- [ ] Error messages clearly identify the missing prerequisite without exposing secret values or file contents.

---

### Dependencies

**Blocked By:**
- None

**Blocks:**
- WP-004
- WP-006
- WP-007
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-002
- Cannot run in parallel with: None

---

### Risks

**Risk 1: Misleading capability state enables unsafe UI paths**
- Probability: Medium
- Impact: High
- Mitigation: Treat capability as false unless every prerequisite validates successfully.

**Risk 2: Startup validation duplicates or drifts from persistence guardrails**
- Probability: Low
- Impact: Medium
- Mitigation: Reuse existing runtime-config and fail-fast validation patterns from ADR-004.

