## WP-002: Git Repo Identity, Canonical URL Semantics, and Config Migration

### Metadata

```yaml
WP_ID: WP-002
Title: Git Repo Identity, Canonical URL Semantics, and Config Migration
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
Current repo IDs and checkout names derive from the trailing repo name in the URL, which is collision-prone and unsafe once URLs may contain credential-bearing userinfo. The repo model must become canonical, stable, and backward-compatible.

**Scope:**
- Normalize repo URLs into canonical non-secret form.
- Change `id` generation to a stable canonical-URL hash while preserving the `id` field name.
- Extend `pkg/git.GitRepoConfig` with an additive `auth` descriptor.
- Load legacy URL-only config and rewrite it safely on save.

Excluded:
- Stored-secret persistence in SQLite.
- Credential resolution and go-git auth.
- API/UI request validation beyond shared helper reuse.

**Success Criteria:**
- [ ] Userinfo-bearing URLs are rejected before persistence.
- [ ] Legacy configs load successfully and save back in the expanded schema.
- [ ] Stable `id` values do not change when credential source or secret rotation changes.

---

### Technical Requirements

**Input Contracts:**
- Existing repo config model in `pkg/git/config.go`.
- Existing startup/config bootstrap path in `cmd/skillserver/main.go`.

**Output Contracts:**
- Expanded `GitRepoConfig` with additive `auth` metadata.
- Canonical URL and stable-ID helpers in `pkg/git`.
- Backward-compatible load/save behavior for `.git-repos.json`.

**Integration Points:**
- WP-003 consumes typed auth descriptors.
- WP-005 consumes stable ID and checkout-name semantics.
- WP-006 reuses canonicalization and duplicate-detection helpers.

---

### Deliverables

**Code Deliverables:**
- [ ] Add repo URL canonicalization and userinfo rejection helpers in `pkg/git/config.go` or a dedicated companion file.
- [ ] Expand `GitRepoConfig` with `Auth` descriptor fields while retaining `id`, `url`, `name`, and `enabled`.
- [ ] Update startup bootstrap in `cmd/skillserver/main.go` to persist canonical URLs and stable IDs for env/flag-provided repos.
- [ ] Define deterministic checkout-name resolution rules that remain compatible with `pkg/domain` read-only detection.

**Test Deliverables:**
- [ ] Extend `pkg/git/config_test.go` for canonicalization, legacy config loading, stable IDs, and collision scenarios.
- [ ] Add compatibility tests for existing URL-only repo records.

---

### Acceptance Criteria

**Functional:**
- [ ] Saving config never persists raw credentials in `url`.
- [ ] Re-saving the same canonical repo config preserves the same `id`.
- [ ] Repo add/update duplicate detection compares canonical URLs rather than raw input strings.

**Testing:**
- [ ] Config tests cover HTTPS, SSH, nested paths, userinfo rejection, and legacy migration behavior.
- [ ] Compatibility tests prove old configs still load without manual migration.

---

### Dependencies

**Blocked By:**
- None

**Blocks:**
- WP-003
- WP-004
- WP-005
- WP-006
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-001
- Cannot run in parallel with: None

---

### Risks

**Risk 1: Stable-ID migration breaks existing delete/sync/toggle flows**
- Probability: Medium
- Impact: High
- Mitigation: Preserve the `id` field name and add explicit compatibility tests for CRUD flows.

**Risk 2: Checkout naming changes break git-backed read-only detection**
- Probability: Medium
- Impact: High
- Mitigation: Define checkout-name semantics explicitly and wire them into WP-005 in the same rollout sequence.

