# Private Git Credential Sources Rollout and Rollback Runbook

## Purpose
Deterministic rollout and rollback procedure for ADR-006 private Git repository credential sources (`none`, `https_token`, `https_basic`, `ssh_key`) with secret sources (`env`, `file`, `stored`).

## References
- ADR: [ADR-006: Private Git Repository Credential Sources](/home/jeff/skillserver/docs/adrs/006-private-git-repository-credential-sources.md)
- Runtime/API docs: [README.md](/home/jeff/skillserver/README.md)
- Validation evidence:
  - [WP-008 private repo integration and regression test matrix](/home/jeff/skillserver/docs/implementation-plans/private-git-repository-credential-sources/work-packages/WP-008-private-repo-integration-and-regression-test-matrix.md)
  - [WP-008 completion summary](/home/jeff/skillserver/docs/implementation-plans/private-git-repository-credential-sources/work-packages/completion-summaries/WP-008-completion-summary.md)

## Runtime Controls
- `SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS` / `--git-enable-stored-credentials` (default: `false`)
- `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY` / `--git-credential-master-key`
- `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE` / `--git-credential-master-key-file`
- `SKILLSERVER_PERSISTENCE_DATA` / `--persistence-data` (required for `stored`)
- `SKILLSERVER_PERSISTENCE_DIR` / `--persistence-dir` (required for `stored`)

Behavior notes:
- Production default should be `auth.source=env` or `auth.source=file`.
- `auth.source=stored` is optional and disabled by default.
- If stored mode is enabled, startup fails fast without persistence runtime and a master key.

## Master Key Rotation Caveat
- Stored credentials are encrypted with key metadata (`key_id`, `key_version`) and cannot be decrypted with a mismatched key.
- Rotate keys using a staged approach:
  1. Bring service up with old key and capture current repo credential inventory.
  2. Re-write stored credentials using the new key (or migrate repos to env/file first).
  3. Verify sync success before retiring old key material.

## Preconditions
- Rollout owner and rollback owner are assigned.
- Candidate artifact includes WP-008 evidence.
- API/UI access is fronted by trusted authentication and TLS before enabling stored credentials.
- `jq` is available for validation snippets below.

## Pre-Deploy Validation Gates
Run these WP-008 verification commands on the candidate artifact:

```bash
go test ./cmd/skillserver -count=1
go test ./pkg/git -count=1
go test ./pkg/persistence -count=1
go test ./pkg/web -count=1
```

## Supported Deployment Patterns

### Local
- Public repositories: `auth.mode=none`.
- Private repositories: use `env` or `file` source first.
- Stored mode only for trusted local/admin scenarios.

### Docker
- Inject token/username via `-e` environment variables (`auth.source=env`).
- Mount SSH key and `known_hosts` files via read-only volumes (`auth.source=file`).

### Kubernetes Secret
- Secret -> env var mapping for token/basic flows.
- Secret -> mounted files for SSH flows.

### Vault-Projected Values
- Vault Agent Injector, External Secrets, or CSI may project env vars/files.
- Configure SkillServer repos the same way as any env/file source.
- No direct Vault API integration is required for ADR-006.

## Rollout Procedure
1. Deploy using public or env/file-backed repo configs.
2. Verify secret-safe contract responses from `GET /api/git-repos`.
3. Optionally enable stored mode only after runtime prerequisites and security boundary checks pass.
4. Add/update stored-backed repos only after step 3 passes.

Optional stored-mode startup example:

```bash
./skillserver \
  --persistence-data=true \
  --persistence-dir ./data/skillserver \
  --git-enable-stored-credentials \
  --git-credential-master-key-file ./secrets/git-master-key.txt
```

## Rollout Validation Checklist
- [ ] `GET /api/runtime/capabilities` returns `git.stored_credentials_enabled` as expected.
- [ ] `GET /api/git-repos` responses include secret-safe fields only:
  - `auth_mode`
  - `credential_source`
  - `has_credentials`
  - `stored_credentials_enabled`
  - `last_sync_status`
  - `last_sync_error`
- [ ] `POST`/`PUT` reject unsupported auth mode/source combinations with actionable errors.
- [ ] Userinfo-bearing URLs are rejected.
- [ ] Manual sync (`POST /api/git-repos/:id/sync`) reuses configured auth source and returns redacted errors.

Validation snippet:

```bash
set -euo pipefail

BASE_URL="http://127.0.0.1:8080"

curl -sS "$BASE_URL/api/runtime/capabilities" | jq '.' > /tmp/wp009-runtime-capabilities.json
jq -e '.git | has("stored_credentials_enabled")' /tmp/wp009-runtime-capabilities.json >/dev/null

curl -sS "$BASE_URL/api/git-repos" | jq '.' > /tmp/wp009-git-repos.json
jq -e 'type == "array"' /tmp/wp009-git-repos.json >/dev/null
jq -e 'all(.[]; has("auth_mode") and has("credential_source") and has("has_credentials") and has("stored_credentials_enabled") and has("last_sync_status"))' /tmp/wp009-git-repos.json >/dev/null
```

## Rollback Triggers
Rollback if any of the following are observed:
- Stored mode startup validation failures in target environment.
- Secret material appears in API responses, logs, or UI state.
- Private sync failures that cannot be recovered by credential rotation.
- Stored mode was enabled without a trusted auth/TLS boundary.

## Rollback Procedure

### 1) Disable stored-credential mode immediately

```bash
# Flag-based rollback
./skillserver --git-enable-stored-credentials=false

# Env-based rollback
export SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS=false
./skillserver
```

### 2) Move affected repositories to env/file or public mode
Use `PUT /api/git-repos/:id` with canonical URL + safe auth metadata.

Example: stored -> env (`https_token`)

```bash
curl -sS -X PUT "http://127.0.0.1:8080/api/git-repos/<repo-id>" \
  -H "Content-Type: application/json" \
  --data '{
    "url": "https://github.com/acme/private-skills.git",
    "auth": {
      "mode": "https_token",
      "source": "env",
      "token_ref": "REPO_ACME_PAT"
    }
  }'
```

Example: private -> public fallback

```bash
curl -sS -X PUT "http://127.0.0.1:8080/api/git-repos/<repo-id>" \
  -H "Content-Type: application/json" \
  --data '{
    "url": "https://github.com/acme/public-skills.git",
    "auth": {
      "mode": "none",
      "source": "none"
    }
  }'
```

### 3) Re-sync and verify

```bash
curl -sS -X POST "http://127.0.0.1:8080/api/git-repos/<repo-id>/sync" | jq '.'
```

### 4) Preserve encrypted rows for recovery
Keep `git_repo_credentials` rows unless policy explicitly requires deletion.

## Post-Rollback Verification
- [ ] `GET /api/runtime/capabilities` reflects `stored_credentials_enabled=false`.
- [ ] Public and env/file-backed repos continue syncing.
- [ ] `GET /api/git-repos` remains secret-free.

## Post-Change Closeout
- [ ] Record timestamp + operator + commands executed.
- [ ] Attach validation outputs (`/tmp/wp009-*.json`) to release notes.
- [ ] Link the final outcome in [WP-009 completion summary](/home/jeff/skillserver/docs/implementation-plans/private-git-repository-credential-sources/work-packages/completion-summaries/WP-009-completion-summary.md).
