# Implementation Plan: ADR-006 Private Git Repository Credential Sources

**Date Created:** 2026-03-07
**Project Owner:** @jeff
**Target Completion:** 2026-03-20
**Actual Completion:** 2026-03-07
**Status:** COMPLETED
**Source ADR:** [ADR-006: Private Git Repository Credential Sources](../../adrs/006-private-git-repository-credential-sources.md)

---

## Project Overview

### Goal
Add first-class private Git repository authentication to SkillServer without leaking credentials through URLs, `.git-repos.json`, REST responses, browser state, or logs, while preserving the current public-repository workflow.

### ADR Consolidation Note
ADR-006 is now consolidated to one canonical file: [`006-private-git-repository-credential-sources.md`](../../adrs/006-private-git-repository-credential-sources.md). The shorter filename (`006-private-git-repo-credential-sources.md`) is retained only as a superseded pointer to avoid broken links.

### Success Criteria
- [x] Public repositories continue to work with URL-only configuration and no new required flags.
- [x] Private repositories support `https_token`, `https_basic`, and `ssh_key` auth modes through `env` and `file` credential sources.
- [x] Stored credentials are available only when explicitly enabled, backed by the existing SQLite persistence runtime, and protected by a master key from env or mounted file.
- [x] `.git-repos.json`, API responses, UI state, and logs never contain raw credential values or userinfo-bearing URLs.
- [x] Startup sync, periodic sync, and manual `POST /api/git-repos/:id/sync` all resolve credentials through the same code path.
- [x] Per-repo failures are visible through redacted sync status without removing existing successful checkouts.
- [x] The duplicate ADR situation is resolved in docs so there is one clear source of truth after rollout.

### Scope

**In Scope:**
- Stable repo identity and canonical URL handling for git repo configs.
- Secret-safe auth descriptor support in `.git-repos.json`.
- Env/file credential resolution plus opt-in encrypted stored credentials.
- go-git auth integration for clone and pull.
- Redacted sync status surfaced through API and UI.
- REST contract, UI workflow, and regression coverage updates.
- README and operations guidance for local, Docker, Kubernetes Secret, and Vault-projected env/file usage.

**Out of Scope:**
- Direct Vault API or Kubernetes API secret lookups.
- Built-in application authentication/authorization for the web UI.
- Multi-node shared secret synchronization beyond the existing single-binary deployment model.
- Alternate Git transports outside HTTPS and SSH.
- Browser-side retrieval of previously stored secret material.

### Constraints
- Technical: Fit the current `cmd/skillserver`, `pkg/git`, `pkg/web`, `pkg/domain`, and `pkg/persistence` architecture without replacing the Git sync loop.
- Compatibility: Keep current `id` and `url` fields in repo APIs/config for backward compatibility; change semantics rather than renaming them.
- Security: Stored-secret mode must remain disabled by default because repo-management routes are not authenticated in-process.
- Deployment: Reuse ADR-004 persistence runtime and SQLite instead of introducing a second state store.
- Operations: Secret rotation for `env` and `file` sources must not require repo re-creation.

---

## Requirements Analysis

### Must Have
1. Support authenticated clone/pull for private repositories without regressing public repositories.
2. Never persist or echo raw credentials in `.git-repos.json`, REST responses, browser state, or logs.
3. Support operator-managed secret delivery through environment variables and mounted files.
4. Support trusted GUI-based secret entry only when encrypted stored-secret mode is explicitly enabled.
5. Reject repository URLs with embedded userinfo before persistence.
6. Use one credential-resolution path for startup sync, periodic sync, and manual sync.
7. Preserve stable repo identity across credential rotation and auth-source changes.

### Should Have
1. Surface redacted per-repo auth failures through API/UI sync status.
2. Keep public URL-only repo onboarding simple.
3. Require SSH host verification material instead of silently disabling host checks.
4. Document operational prerequisites and rollback paths clearly.

---

## Public Interface and Contract Changes

### Contract Decisions Standardized by This Plan
- Keep `id` as the external repo identifier, but redefine it as a stable hash of the canonical URL instead of the current repo-name-derived string.
- Keep `url` as the persisted/API field name, but persist and return only the canonical non-secret URL.
- Standardize auth mode names on `none`, `https_token`, `https_basic`, and `ssh_key`.
- Standardize credential sources on `env`, `file`, and `stored`.
- Reuse the existing persistence SQLite database for stored credentials rather than introducing a separate database.

### Repo Config Shape
Persisted `.git-repos.json` expands from URL-only records to secret-safe records:

```json
{
  "id": "gitrepo_8f3d2c1a",
  "url": "https://github.com/acme/private-skills.git",
  "name": "private-skills",
  "enabled": true,
  "auth": {
    "mode": "https_token",
    "source": "env",
    "username_ref": "REPO_ONE_GIT_USERNAME",
    "token_ref": "REPO_ONE_GIT_TOKEN"
  }
}
```

Rules:
- `auth` is optional for public repos and defaults to `mode=none`.
- `id` and `url` remain additive/backward-compatible fields.
- Raw credentials are never written to `.git-repos.json`.
- URLs containing userinfo are rejected on create/update.

### Runtime Configuration
- `SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS=true|false` (default `false`)
- `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY`
- `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE`
- Stored-secret mode additionally requires:
  - `SKILLSERVER_PERSISTENCE_DATA=true`
  - `SKILLSERVER_PERSISTENCE_DIR`

### API Changes
- Extend `POST /api/git-repos` and `PUT /api/git-repos/:id` to accept an optional `auth` object and write-only secret payload fields for `source=stored`.
- `GET /api/git-repos` and update/sync responses return only safe fields:
  - `id`
  - `url`
  - `name`
  - `enabled`
  - `auth_mode`
  - `credential_source`
  - `has_credentials`
  - `stored_credentials_enabled`
  - `last_sync_status`
  - `last_sync_error`
- Legacy URL-only payloads remain valid and are interpreted as `auth.mode=none`.

### Internal Contracts
- `pkg/git` sync orchestration moves from `[]string` repos to `[]git.GitRepoConfig` or an equivalent typed repo model.
- `pkg/domain.FileSystemManager` read-only tracking must use deterministic checkout identifiers derived from sanitized repo metadata rather than raw URL strings.
- Stored credentials are persisted in a new SQLite table under `pkg/persistence` with encrypted blobs and key-version metadata.

---

## Domain Mapping

### Infrastructure / Runtime
- Stored-credential feature flags and master-key startup validation.
- Capability wiring so API/UI can know whether stored-secret mode is enabled.

### Data Layer
- Backward-compatible `.git-repos.json` schema evolution.
- Canonical URL normalization, stable ID generation, and config migration helpers.
- Encrypted stored-credential persistence in SQLite.

### Service Layer
- Env/file/stored credential resolution.
- go-git auth-method construction for HTTPS and SSH.
- Sync orchestration refactor so all sync entry points share one auth path.
- Redacted per-repo sync status tracking.

### API Layer
- Request validation and secret-safe response DTOs.
- URL sanitization, duplicate detection, and stable-ID-driven CRUD/sync semantics.

### UI Layer
- Auth mode/source selection.
- Env/file reference forms.
- Masked stored-secret entry and "configured" state.
- Redacted sync status display.

### Quality and Documentation
- Regression matrix for public/private compatibility, redaction, sync-path parity, and SSH verification.
- README, operations guidance, rollback flow, and ADR duplicate cleanup.

---

## Work Package Breakdown

### Phase 1: Runtime and Config Foundation
- [x] [WP-001: Private Git Runtime Flags and Startup Guardrails](./work-packages/WP-001-private-git-runtime-flags-and-startup-guardrails.md) ✅ COMPLETED (2026-03-07)
- [x] [WP-002: Git Repo Identity, Canonical URL Semantics, and Config Migration](./work-packages/WP-002-git-repo-identity-canonical-url-semantics-and-config-migration.md) ✅ COMPLETED (2026-03-07)

### Phase 2: Credential Backends and Sync Integration
- [x] [WP-003: Env/File Credential Resolver and go-git Auth Builder](./work-packages/WP-003-env-file-credential-resolver-and-go-git-auth-builder.md) ✅ COMPLETED (2026-03-07)
- [x] [WP-004: Encrypted Git Credential Store and Key Management](./work-packages/WP-004-encrypted-git-credential-store-and-key-management.md) ✅ COMPLETED (2026-03-07)
- [x] [WP-005: Git Syncer Auth Integration, Checkout Semantics, and Redacted Status Reporting](./work-packages/WP-005-git-syncer-auth-integration-checkout-semantics-and-redacted-status-reporting.md) ✅ COMPLETED (2026-03-07)

### Phase 3: Admin Interfaces
- [x] [WP-006: Secret-Safe Git Repo API Contracts and Handlers](./work-packages/WP-006-secret-safe-git-repo-api-contracts-and-handlers.md) ✅ COMPLETED (2026-03-07)
- [x] [WP-007: Web UI Private Repo Credential Workflows and Masked Status UX](./work-packages/WP-007-web-ui-private-repo-credential-workflows-and-masked-status-ux.md) ✅ COMPLETED (2026-03-07)

### Phase 4: Verification and Rollout
- [x] [WP-008: Private Repo Integration and Regression Test Matrix](./work-packages/WP-008-private-repo-integration-and-regression-test-matrix.md) ✅ COMPLETED (2026-03-07)
- [x] [WP-009: Operations Docs, Rollback Guidance, and ADR Consolidation](./work-packages/WP-009-operations-docs-rollback-guidance-and-adr-consolidation.md) ✅ COMPLETED (2026-03-07)

---

## Dependency Graph

```text
WP-001 -> WP-004
WP-002 -> (WP-003 || WP-004)
(WP-002 && WP-003 && WP-004) -> WP-005
(WP-001 && WP-002 && WP-004 && WP-005) -> WP-006
WP-006 -> WP-007
(WP-003 && WP-004 && WP-005 && WP-006 && WP-007) -> WP-008
(WP-001 && WP-002 && WP-004 && WP-006 && WP-008) -> WP-009
```

### Critical Path
`WP-002 -> WP-004 -> WP-005 -> WP-006 -> WP-007 -> WP-008 -> WP-009`

### Parallel Opportunities
- WP-001 and WP-002 can start immediately.
- WP-003 and WP-004 can run in parallel once their prerequisites are satisfied.
- WP-009 can draft environment examples early, but should not be closed until WP-008 verifies the final behavior.

---

## Timeline and Effort

| Phase | Work Packages | Estimated Hours |
|-------|---------------|-----------------|
| Runtime and Config Foundation | WP-001, WP-002 | 8 |
| Credential Backends and Sync Integration | WP-003, WP-004, WP-005 | 15 |
| Admin Interfaces | WP-006, WP-007 | 9 |
| Verification and Rollout | WP-008, WP-009 | 9 |
| **Total** | **9 WPs** | **41** |

### Schedule Forecast
- Critical-path effort: 34 hours.
- Aggressive (parallelized): 6 working days at 6 productive hours/day.
- Realistic (implementation plus review/fix iteration): 7-8 working days.
- Conservative with contingency buffer (x1.3): 53 hours, roughly 9 working days.

---

## Test Strategy

### Unit Tests
- Canonical URL normalization and userinfo rejection.
- Stable ID generation invariance.
- Env/file/stored credential provider resolution and validation.
- go-git auth builder behavior for `none`, `https_token`, `https_basic`, and `ssh_key`.
- Encryption/decryption and key-version handling for stored credentials.
- Error and log redaction helpers.

### Integration Tests
- Startup sync, periodic sync, and manual sync use the same credential resolution path.
- Existing public repos continue to clone/pull without auth metadata.
- Private repo sync failures preserve last successful checkout and report redacted status.
- Stored-secret mode fails safely when persistence or master key is missing.
- Rotated env/file credentials are picked up on a later sync attempt.

### API Tests
- `POST`/`PUT /api/git-repos` accept auth descriptors and reject userinfo-bearing URLs.
- List/update/sync responses never echo write-only secret values.
- Stable `id` semantics remain consistent when auth source changes but canonical URL does not.
- Toggle/delete/sync flows continue to work with the expanded repo model.

### UI / Browser Verification
- Public repo add/edit flow remains minimal and usable.
- Env/file reference fields render correctly by auth mode.
- Stored-secret controls stay hidden when runtime capability is disabled.
- Editing a stored-secret repo shows masked "configured" state instead of original values.
- Sync status badges and redacted errors surface without exposing secrets.

---

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Stable-ID migration breaks existing repo CRUD, delete, or read-only detection flows | Medium | High | Keep `id` field name stable, add migration tests, and update `pkg/domain` checkout tracking in the same sync-integration phase. |
| Secrets leak through logs, API responses, or browser state | Medium | High | Centralize redaction, add API contract tests, and treat all stored-secret fields as write-only. |
| Stored-secret mode is enabled without a persistence database or master key | Medium | High | Fail fast at startup, expose capability flags explicitly, and keep stored-secret mode off by default. |
| SSH verification is bypassed or unusably strict | Medium | High | Require `known_hosts` material for SSH, validate inputs clearly, and add focused tests for success/failure cases. |
| Auth failures remove working catalog content or repo checkouts | Low | High | Preserve existing checkout on sync failure and record redacted status separately from destructive cleanup. |
| Duplicate ADR files keep drifting after implementation | High | Medium | Make ADR consolidation a deliverable of WP-009 and document the canonical file explicitly. |

---

## Assumptions and Defaults
1. The longer ADR file, [`006-private-git-repository-credential-sources.md`](../../adrs/006-private-git-repository-credential-sources.md), is the canonical design input for implementation.
2. Stored credentials will reuse the existing ADR-004 persistence runtime and SQLite bootstrap path.
3. Public repositories remain the default URL-only path with `auth.mode=none`.
4. Env/file sources are the preferred production path; direct Vault/Kubernetes APIs remain out of scope.
5. No in-process authentication layer is introduced as part of this feature.

---

## Work Package Documents
- [Work Package Index](./work-packages/INDEX.md)
- [WP-001](./work-packages/WP-001-private-git-runtime-flags-and-startup-guardrails.md)
- [WP-002](./work-packages/WP-002-git-repo-identity-canonical-url-semantics-and-config-migration.md)
- [WP-003](./work-packages/WP-003-env-file-credential-resolver-and-go-git-auth-builder.md)
- [WP-004](./work-packages/WP-004-encrypted-git-credential-store-and-key-management.md)
- [WP-005](./work-packages/WP-005-git-syncer-auth-integration-checkout-semantics-and-redacted-status-reporting.md)
- [WP-006](./work-packages/WP-006-secret-safe-git-repo-api-contracts-and-handlers.md)
- [WP-007](./work-packages/WP-007-web-ui-private-repo-credential-workflows-and-masked-status-ux.md)
- [WP-008](./work-packages/WP-008-private-repo-integration-and-regression-test-matrix.md)
- [WP-009](./work-packages/WP-009-operations-docs-rollback-guidance-and-adr-consolidation.md)

---

## Implementation Completion Summary

**Completion Date:** 2026-03-07  
**Status:** ✅ COMPLETED

### Overall Metrics

| Metric | Estimated | Actual | Variance |
|--------|-----------|--------|----------|
| Total Effort | 41h | 36.5-41h | -4.5h to 0h (-11% to 0%) |
| Work Packages | 9 | 9 | 0 |
| Tests Added | Not explicitly tracked in WP summaries | Not explicitly tracked in WP summaries | N/A |
| Average Coverage | Not explicitly tracked in WP summaries | Not explicitly tracked in WP summaries | N/A |
| Total LOC | Not explicitly tracked in WP summaries | Not explicitly tracked in WP summaries | N/A |
| Duration | 1-2 weeks target window | 1 day (all completion summaries dated 2026-03-07) | Ahead of plan |

### Work Package Summary

| WP ID | Domain | Estimated | Actual | Status | Completed |
|-------|--------|-----------|--------|--------|-----------|
| WP-001 | Runtime / Infrastructure | 3h | 2.5-3h | ✅ | 2026-03-07 |
| WP-002 | Data / Config Migration | 5h | 4.5-5h | ✅ | 2026-03-07 |
| WP-003 | Service / Auth Resolver | 4h | 3.5-4h | ✅ | 2026-03-07 |
| WP-004 | Persistence / Encryption | 5h | 4.5-5h | ✅ | 2026-03-07 |
| WP-005 | Sync Service Integration | 6h | 5-6h | ✅ | 2026-03-07 |
| WP-006 | API Contracts / Handlers | 5h | 5h | ✅ | 2026-03-07 |
| WP-007 | Web UI | 4h | 4h | ✅ | 2026-03-07 |
| WP-008 | Verification / Regression | 6h | 5-6h | ✅ | 2026-03-07 |
| WP-009 | Documentation / Operations | 3h | 2.5-3h | ✅ | 2026-03-07 |

### Key Achievements

- Delivered private repository authentication across env/file/stored credential sources without exposing secret values in config, API, UI, or logs.
- Preserved backwards compatibility for URL-only public repository workflows while upgrading internal identity semantics to stable canonical URL IDs.
- Unified startup, periodic, and manual sync flows under one credential-resolution/auth-construction path.
- Added encrypted stored-credential persistence with startup guardrails and capability gating.
- Completed operations and ADR consolidation documentation with rollout and rollback guidance.

### Common Challenges Encountered

1. **Cross-suite test stability noise** (occurred in 2 WPs)
   - Description: One full-suite run exposed pre-existing unrelated failures and one transient test flake.
   - Resolution pattern: Focused package-level verification and reruns were used to isolate feature-scope validation from unrelated instability.

2. **Cross-layer contract consistency** (occurred across multiple WPs)
   - Description: Keeping config, sync, API, and UI aligned on canonical URL/ID and secret-safe response fields required coordinated changes.
   - Resolution pattern: Standardized typed contracts and expanded contract tests were used to keep behavior aligned.

### Lessons Learned

The following items are inferred from completion summary content because dedicated "lessons learned" sections were not consistently present.

**What Went Well:**
- Work package boundaries stayed clean and minimized cross-domain rework.
- Secret-safety requirements were enforced through both implementation patterns and tests.
- Backward compatibility was preserved while introducing substantial internal changes.

**What Could Be Improved:**
- Completion summaries should capture quantitative metrics (test counts, coverage deltas, LOC deltas) consistently.
- A deterministic full-suite verification pass should be stabilized and tracked to reduce ambiguity in closeout evidence.
- Explicit lessons-learned sections should be included in each WP summary template output.

**Actionable Recommendations for Future Plans:**
1. Require a standardized completion-summary template with mandatory quantitative metrics and lessons fields.
2. Add a final "full-suite stabilization" gate to explicitly classify unrelated flakes vs. scope regressions.
3. Include cross-package dependency checkpoints for shared contracts (config/API/UI) during implementation, not only at the end.

### Technical Debt Summary

| Priority | Count | Total Effort | Tickets Created |
|----------|-------|--------------|-----------------|
| High | 0 | 0h | None referenced |
| Medium | 2 | ~4h | None referenced |
| Low | 1 | ~1h | None referenced |

**High Priority Debt Items:**
- None identified in WP completion summaries.

### Follow-Up Items

- [ ] Stabilize and root-cause the transient `pkg/web` temp-directory cleanup flake noted during WP-008.
- [ ] Investigate/resolve unrelated full-suite failures noted during WP-004 (`pkg/git` parallel `Setenv` panic path).
- [ ] Standardize completion summary template usage to include quantitative metrics and explicit lessons sections.

### References

**Work Package Completion Summaries:**
- [WP-001 Completion Summary](./work-packages/completion-summaries/WP-001-completion-summary.md)
- [WP-002 Completion Summary](./work-packages/completion-summaries/WP-002-completion-summary.md)
- [WP-003 Completion Summary](./work-packages/completion-summaries/WP-003-completion-summary.md)
- [WP-004 Completion Summary](./work-packages/completion-summaries/WP-004-completion-summary.md)
- [WP-005 Completion Summary](./work-packages/completion-summaries/WP-005-completion-summary.md)
- [WP-006 Completion Summary](./work-packages/completion-summaries/WP-006-completion-summary.md)
- [WP-007 Completion Summary](./work-packages/completion-summaries/WP-007-completion-summary.md)
- [WP-008 Completion Summary](./work-packages/completion-summaries/WP-008-completion-summary.md)
- [WP-009 Completion Summary](./work-packages/completion-summaries/WP-009-completion-summary.md)
