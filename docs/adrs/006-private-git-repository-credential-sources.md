# ADR-006: Private Git Repository Credential Sources

## Metadata

| Field | Value |
|-------|-------|
| **Status** | Proposed |
| **Date** | 2026-03-07 |
| **Author(s)** | @jeff |
| **Reviewers** | TBD |
| **Work Package** | N/A |
| **Supersedes** | N/A |
| **Superseded By** | N/A |

## Summary

SkillServer can currently sync only public Git repositories or repositories that happen to authenticate through out-of-band host configuration. To support private Git-backed skill content safely, we will add a Git authentication descriptor model that stores only non-sensitive repository metadata plus a credential source reference, then resolve secrets at sync time from environment variables, mounted files, or an optional encrypted in-app secret store. Kubernetes and Vault-backed deployments will integrate through env/file secret injection rather than direct SkillServer dependencies on cluster or Vault APIs.

## Context

### Problem Statement

Users want to import private Git repositories containing skills and prompt content into SkillServer. The current implementation does not support first-class authentication for clone/pull operations, the GUI only accepts a repository URL, and repository configuration is persisted as plain JSON under the skills directory. Adding credentials by embedding them in URLs or saving them directly in `.git-repos.json` would leak secrets through disk, logs, API responses, and copy/paste workflows.

### Current State

- Git repositories are configured from `SKILLSERVER_GIT_REPOS`, `GIT_REPOS`, or `--git-repos`, then persisted into `.git-repos.json` if no config exists in [`cmd/skillserver/main.go`](/home/jeff/skillserver/cmd/skillserver/main.go).
- Persisted Git repo config currently contains only `id`, `url`, `name`, and `enabled` and is written as world-readable JSON (`0644`) in [`pkg/git/config.go`](/home/jeff/skillserver/pkg/git/config.go).
- Clone and pull operations use `go-git` without any auth method configured in [`pkg/git/syncer.go`](/home/jeff/skillserver/pkg/git/syncer.go). Authentication failures are surfaced only as generic sync errors.
- The REST API and Web UI accept only a repository URL for add/update flows in [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go) and [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html).
- Git repo management routes are exposed by the web server today without built-in route-level auth in [`pkg/web/server.go`](/home/jeff/skillserver/pkg/web/server.go).
- Repository identity is currently derived from URL/path name extraction, which is acceptable for public URLs but fragile for credential-bearing URLs and collision-prone across similarly named repos in [`pkg/git/config.go`](/home/jeff/skillserver/pkg/git/config.go).

### Requirements

| Requirement | Priority | Description |
|-------------|----------|-------------|
| REQ-1 | Must Have | Support authenticated clone/pull for private repositories without regressing public repository flows. |
| REQ-2 | Must Have | Never persist or echo raw credentials in `.git-repos.json`, repository URLs, REST responses, or logs. |
| REQ-3 | Must Have | Support operator-managed credentials through env vars and mounted files suitable for Docker, Kubernetes Secrets, Vault Agent, CSI, or similar secret injection patterns. |
| REQ-4 | Must Have | Support secure self-serve GUI input for credentials when the deployment explicitly enables encrypted secret storage. |
| REQ-5 | Must Have | Use the same credential resolution path for startup sync, periodic sync, and manual `POST /api/git-repos/:id/sync`. |
| REQ-6 | Should Have | Support common private Git auth modes: HTTPS token/basic auth and SSH private key auth. |
| REQ-7 | Should Have | Allow credential rotation without requiring repository re-creation or credential-bearing URL changes. |
| REQ-8 | Should Have | Preserve stable repo identity independent of credential source or secret rotation. |

### Constraints

- **Budget**: Avoid mandatory managed secret-manager or database dependencies for the base deployment.
- **Timeline**: Deliver incrementally without rewriting the existing git sync architecture.
- **Technical**: Preserve backward compatibility for current public repo config and sync behavior.
- **Compliance**: Secrets must not be written to plain config files, returned from APIs, or logged.
- **Security**: Current SkillServer web routes do not provide built-in auth; any GUI-based secret entry must assume trusted deployment, TLS, and upstream access control.
- **Deployment**: Must work for local, Docker, and Kubernetes deployments without requiring direct Kubernetes API or Vault API integration in the first iteration.

## Decision Drivers

1. **Secret Exposure Minimization**: Raw credentials must stay out of URLs, config files, logs, and list APIs.
2. **Deployment Flexibility**: The feature must work for local development, Docker, and Kubernetes/Vault-backed deployments.
3. **Usability**: The solution should support both operator-managed secret injection and self-serve GUI onboarding.
4. **Architectural Compatibility**: Changes should fit the current config-manager plus git-syncer design rather than replacing it wholesale.
5. **Extensibility**: The design should allow additional auth methods or secret providers later without another repo-config migration.

## Options Considered

### Option 1: Inline Credentials in URL or Plain Repo Config

**Description**: Allow users to paste tokenized HTTPS URLs or raw credentials into repo config and pass those values directly to `go-git`.

**Pros**:
- Lowest implementation effort.
- Works with existing add/update flows with minimal API changes.
- Easy to explain for one-off local setups.

**Cons**:
- Secrets leak into `.git-repos.json`, shell history, screenshots, logs, and copied URLs.
- Credential rotation requires URL mutation and usually re-entry everywhere the repo is referenced.
- Breaks clean repo identity because userinfo-bearing URLs are not stable.
- Unacceptable default for shared or Kubernetes-hosted deployments.

**Estimated Effort**: S

**Cost Implications**: Low

---

### Option 2: External Secret References Only (Env/File)

**Description**: Extend repo config with auth type plus environment-variable names or mounted-file paths. SkillServer resolves credentials only from process env or filesystem paths provided by the host/container platform.

**Pros**:
- Keeps raw secrets out of SkillServer-managed persistence and APIs.
- Fits Docker, Kubernetes Secrets, Vault Agent, CSI, and External Secrets workflows well.
- No in-app secret encryption lifecycle required.
- Works even when the web UI is not trusted to receive secret material.

**Cons**:
- Poor self-serve UX for standalone/local users.
- Requires separate operator action to inject env vars/files before repo creation succeeds.
- Harder to onboard multiple repos with distinct credentials from the GUI alone.
- Rotation often depends on external restart/reload semantics.

**Estimated Effort**: M

**Cost Implications**: Low

---

### Option 3: Credential Source Abstraction with Env/File Refs and Optional Encrypted GUI Store (Chosen)

**Description**: Extend repo config with an auth descriptor and credential source metadata, but never store secret values in the repo config itself. Resolve secrets through a provider abstraction with three initial providers:

- `env`: resolve credentials from named environment variables.
- `file`: resolve credentials from mounted files.
- `stored`: resolve credentials from an encrypted SkillServer-managed secret store backed by SQLite persistence and a master key supplied by env var or mounted file.

The GUI supports env/file reference entry everywhere and can optionally accept raw credential material for one-time secure storage when encrypted secret storage is explicitly enabled. Kubernetes and Vault-backed deployments use `env` or `file` sources via standard platform secret injection; SkillServer does not call Kubernetes or Vault APIs directly in the first iteration.

**Pros**:
- Meets both self-serve GUI and operator-managed deployment requirements.
- Keeps raw secrets out of `.git-repos.json`, URLs, and normal REST responses.
- Aligns naturally with Kubernetes/Vault practices through secret injection rather than bespoke API integration.
- Creates a clean extension point for future providers or auth modes.
- Allows stable repo identity derived from canonical non-secret URL plus opaque credential handle.

**Cons**:
- More implementation work than env/file-only support.
- Introduces encryption-key lifecycle and backup concerns for the stored-secret provider.
- GUI secret entry is only safe in deployments with upstream auth/TLS and must be gated accordingly.
- Expands the integration test matrix across providers and auth modes.

**Estimated Effort**: L

**Cost Implications**: Low-Medium

## Decision

### Chosen Option

**We will implement Option 3: credential source abstraction with env/file references and an optional encrypted GUI-managed secret store.**

### Rationale

Option 3 is the only approach that satisfies all must-have requirements without forcing every deployment to adopt a managed secret service or accept insecure plaintext secret storage. It preserves the current repo-management architecture, uses Kubernetes/Vault-friendly secret injection patterns by default, and still provides a self-serve GUI path for trusted deployments that opt into persistence and encryption-key management.

We are explicitly **not** choosing direct native Kubernetes Secret or Vault API lookups in the first iteration. Env/file injection already satisfies the deployment use cases while avoiding extra network dependencies, RBAC, Vault auth flows, and sync-path latency. If direct Vault integration becomes necessary later, it can be added behind the same credential-provider interface.

### Decision Matrix

| Criteria | Weight | Option 1 | Option 2 | Option 3 |
|----------|--------|----------|----------|----------|
| Secret safety | 5 | 1 | 5 | 4 |
| GUI self-service UX | 4 | 5 | 2 | 5 |
| Kubernetes/Vault compatibility | 4 | 2 | 5 | 5 |
| Implementation/operational simplicity | 3 | 5 | 4 | 3 |
| Extensibility | 3 | 1 | 3 | 5 |
| Rotation and long-term maintainability | 3 | 2 | 4 | 5 |
| **Weighted Total** |  | **57** | **86** | **99** |

## Consequences

### Positive

- SkillServer can support private Git-backed skill imports without encouraging credential-in-URL anti-patterns.
- Kubernetes and Vault-backed deployments can reuse existing secret injection tooling through env vars or mounted files.
- Standalone or trusted internal deployments gain a workable GUI onboarding path for private repos.
- Public-repo behavior stays backward-compatible and can continue using URL-only config.
- The same credential-resolution layer can later support additional secret providers or auth modes.

### Negative

- The product now owns an encryption-key lifecycle for GUI-managed secrets.
- Stored-secret support depends on persistence mode and additional configuration.
- GUI secret entry cannot be treated as safe on internet-exposed instances unless an external auth/TLS boundary already exists.
- Repo-config validation and sync error handling become more complex.

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Stored secrets become undecryptable after master-key loss or mismatch | Med | High | Require explicit master-key configuration, fail fast on startup, and document backup/rotation procedures. |
| Secrets or secret references leak through API responses or logs | Med | High | Make secret fields write-only, redact auth errors/logs, and return only masked credential summaries from list APIs. |
| Repo identity changes or collisions break read-only mapping and sync continuity | Med | Med | Derive repo IDs from canonical non-secret URL hash rather than repo name or credential-bearing URL. |
| Rotated env/file secrets are not picked up consistently | Med | Med | Resolve credentials fresh on each sync attempt and document restart/resync expectations by provider type. |
| GUI credential entry is enabled on an untrusted deployment | Low | High | Gate stored-secret mode behind explicit feature/config checks and document requirement for upstream auth and TLS. |

## Technical Details

### Architecture

```text
CLI / Env / Web UI
        |
        +--> repo config (.git-repos.json)
        |      - canonical_url
        |      - repo_id
        |      - enabled
        |      - auth descriptor
        |      - credential source metadata only
        |
        +--> optional encrypted secret store (SQLite, when source=stored)
        |
        v
Git Credential Resolver
   - env provider
   - file provider
   - stored provider
        |
        v
AuthMethod Builder
        |
        v
go-git clone/pull (startup, periodic sync, manual sync)
        |
        v
catalog rebuild / persistence sync (existing flow)
```

### Data Model

Extend `GitRepoConfig` with non-secret auth metadata only:

- `repo_id`: stable hash of canonical URL (new primary identifier)
- `canonical_url`: normalized URL with userinfo removed
- `display_name`: human-readable repo name
- `enabled`
- `auth.mode`: `none` | `https_token` | `https_basic` | `ssh_key`
- `auth.source`: `env` | `file` | `stored`
- `auth.reference_id`: opaque handle or logical reference name
- `auth.username_ref` / `auth.password_ref` / `auth.token_ref` / `auth.key_ref` / `auth.known_hosts_ref`: provider-specific non-secret references

Add an encrypted secret table when `auth.source=stored` and persistence is enabled:

- `git_repo_credentials.repo_id`
- `git_repo_credentials.key_id`
- `git_repo_credentials.ciphertext`
- `git_repo_credentials.nonce`
- `git_repo_credentials.created_at`
- `git_repo_credentials.updated_at`

Secret payloads are stored as encrypted JSON blobs so the provider can support token, username/password, or SSH key material without further schema changes.

### Credential Resolution Rules

1. Normalize the configured repository URL into a canonical non-secret form.
2. Derive `repo_id` from the canonical URL hash.
3. Load the repo auth descriptor from config.
4. Resolve credential material just-in-time from `env`, `file`, or `stored`.
5. Build the appropriate `go-git` auth method for clone/pull.
6. Redact all auth material before returning errors or logs.

Credential resolution happens on every sync attempt so env/file rotations can take effect without rewriting repo config.

### Supported Auth Modes

- `none`: public repository flow; current behavior.
- `https_token`: token or PAT supplied by provider, optional username defaulting to `git` when provider does not specify one.
- `https_basic`: username and password resolved from provider.
- `ssh_key`: private key, optional passphrase, and optional `known_hosts` material resolved from provider.

### API Changes

Additive API changes:

- Extend `POST /api/git-repos` and `PUT /api/git-repos/:id` to accept an `auth` object.
- Treat secret-bearing request fields as write-only. Responses must never echo secret values.
- `GET /api/git-repos` returns only masked auth summaries such as `auth_mode`, `credential_source`, and `has_credentials`.
- Preserve backward compatibility for legacy URL-only payloads by treating them as `auth.mode=none`.

If GUI-managed secrets are enabled, secret submission may be handled by the same add/update endpoint or a dedicated credential subresource, but returned DTOs must remain secret-free either way.

### Configuration

Existing public-repo config remains valid:

```bash
SKILLSERVER_GIT_REPOS="https://github.com/org/public-skills.git"
```

New stored-secret mode requires persistence plus an encryption key:

```bash
SKILLSERVER_PERSISTENCE_DATA=true
SKILLSERVER_PERSISTENCE_DIR=/var/lib/skillserver/persistence
SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE=/var/run/secrets/skillserver/git-master-key
```

Kubernetes/Vault-style deployment uses env/file references instead of in-app Vault access:

```yaml
env:
  - name: REPO_ONE_GIT_TOKEN
    valueFrom:
      secretKeyRef:
        name: skillserver-git
        key: repo-one-token
volumeMounts:
  - name: vault-secrets
    mountPath: /vault/secrets
```

Example repo config shape after migration:

```json
{
  "repo_id": "gitrepo_8f3d2c1a",
  "canonical_url": "https://github.com/acme/private-skills.git",
  "name": "private-skills",
  "enabled": true,
  "auth": {
    "mode": "https_token",
    "source": "env",
    "token_ref": "REPO_ONE_GIT_TOKEN",
    "username_ref": "REPO_ONE_GIT_USERNAME"
  }
}
```

## Security Considerations

- Reject URLs containing embedded userinfo for add/update flows.
- Never write raw secret material to `.git-repos.json`.
- Never return raw secret material from REST APIs.
- Redact credentials and provider-resolved values from logs and error strings.
- Require `SKILLSERVER_PERSISTENCE_DATA=true` plus `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY` or `_FILE` before enabling `auth.source=stored`.
- Hide or disable GUI secret-entry controls unless stored-secret mode is enabled.
- Document that GUI secret entry assumes the SkillServer UI is fronted by trusted auth and TLS; otherwise only env/file reference entry should be allowed.

## Performance Considerations

- Env/file/stored providers add negligible overhead relative to clone/pull operations.
- Avoiding direct Vault/Kubernetes API lookups keeps sync latency deterministic and prevents secret-manager outages from becoming a hard dependency beyond normal secret injection.
- Just-in-time resolution per sync attempt slightly increases control-plane work but simplifies rotation semantics.

## Implementation Plan

### Phase 1: Repo Identity and Config Schema

- Add canonical URL normalization and stable hashed repo IDs.
- Extend `GitRepoConfig` with auth descriptor fields and migration logic for legacy config.
- Validate that secret-bearing URLs are rejected.

### Phase 2: Credential Resolver and Sync Integration

- Introduce credential-provider interfaces for `env`, `file`, and `stored`.
- Add auth-method construction for HTTPS token/basic and SSH key flows.
- Wire credential resolution into clone, pull, startup sync, periodic sync, and manual sync paths.
- Add log/error redaction.

### Phase 3: Encrypted Stored Secrets and Web UI

- Add encrypted credential storage in SQLite when persistence mode is enabled.
- Add master-key startup validation and key-rotation guidance.
- Extend the Web UI and REST handlers with auth-mode/source selection and write-only secret submission.
- Return masked auth summaries in list/detail responses.

### Phase 4: Docs and Operations

- Update README with private-repo examples for local, Docker, Kubernetes Secret, and Vault Agent/CSI patterns.
- Add rotation and rollback runbooks.
- Document safe deployment prerequisites for GUI-managed secrets.

### Rollback Plan

1. Disable stored-secret mode by removing the master-key config and hiding GUI secret entry.
2. Convert affected repos to env/file-backed references or public-repo mode.
3. Keep repo sync behavior for public repositories unchanged.
4. Retain encrypted secret rows for later recovery if rollback is operational rather than destructive.

## Testing Strategy

- Unit tests:
  - canonical URL normalization and repo ID stability
  - auth descriptor validation
  - env/file/stored provider resolution
  - redaction of error/log output
- Integration tests:
  - authenticated clone/pull for HTTPS token and SSH key flows
  - periodic/manual sync reuses the same credential resolution behavior
  - stored-secret provider fails safely without persistence or master key
  - rotated env/file credentials are picked up on subsequent sync attempts
- End-to-end tests:
  - GUI add/update flow with env/file references
  - GUI add/update flow with stored secrets when enabled
  - `GET /api/git-repos` never returns secret values
  - public repos remain backward-compatible

## Related Decisions

- [ADR-002: Dynamic Imported Resource Discovery and Prompt Support](./002-dynamic-resource-and-prompt-discovery.md)
- [ADR-003: Unified Skill/Prompt Catalog Classification for Git Imports](./003-unified-skill-prompt-catalog-classification.md)
- [ADR-004: Persistent Catalog Storage and Metadata Overlays](./004-persistent-catalog-storage-and-metadata-overlays.md)

## References

- Existing startup/config wiring: [`cmd/skillserver/main.go`](/home/jeff/skillserver/cmd/skillserver/main.go)
- Existing git repo config persistence: [`pkg/git/config.go`](/home/jeff/skillserver/pkg/git/config.go)
- Existing git sync implementation: [`pkg/git/syncer.go`](/home/jeff/skillserver/pkg/git/syncer.go)
- Existing git repo REST handlers: [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go)
- Existing Web UI repo modal: [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html)
- Current user-facing git repo documentation: [`README.md`](/home/jeff/skillserver/README.md)
