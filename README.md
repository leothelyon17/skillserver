<div align="center">
  <img src="pkg/web/ui/images/logo.png" alt="SkillServer Logo" width="200">
</div>

An MCP/REST server with WebUI serving as a centralized skills database for AI Agents. It manages "Skills" (directory-based with SKILL.md files) stored in a local directory, following the [Agent Skills specification](https://agentskills.io).

## Features

- **MCP Server**: Provides tools for AI agents to list/search/read skills, query unified catalog items, and access skill resources
- **Web Interface**: Local web UI for creating, editing, and organizing skills with resource management
- **Git Synchronization**: Automatically syncs with Git repositories (skills from repos are read-only)
- **Full-Text Search**: Powered by Bleve for fast skill searching
- **Resource Management**: Dynamic discovery for scripts, references, assets, agents, prompts, and imported read-only resources
- **Agent Skills Spec Compliant**: Full support for the Agent Skills specification format

<img width="580" alt="Screenshot 2026-01-28 at 11-08-16 skillserver" src="https://github.com/user-attachments/assets/c8db8890-b888-4354-8e7e-0d2a8c37af04" />

## Catalog Contract Rollout Notes

The catalog-ID and taxonomy ergonomics rollout is tracked in
`docs/implementation-plans/catalog-id-taxonomy-and-materialization-ergonomics/`.
The sections below capture the shipped and verified contract surface for that
rollout, plus the release-readiness gate operators should use before promotion.

### ID Compatibility Matrix

| Surface | Emits | Accepts | Notes |
|---------|-------|---------|-------|
| MCP `list_skills`, `search_skills` | Canonical skill item IDs in `id` (`skill:<skill-id>`); populated `name`. | N/A | Bare skill IDs stop being emitted after rollout. |
| MCP `read_skill` and skill-resource tools | N/A | Bare `<skill-id>` and canonical `skill:<skill-id>`. | Normalized to the same parent skill; prompt/rule item IDs are rejected. |
| MCP taxonomy, export, and materialization item tools | Canonical `item_id` values. | Bare skill IDs only for skill items; prompt/rule IDs are canonical-only. | Legacy fallback is intentionally bounded to skill items. |
| REST `/api/catalog` and `/api/catalog/search` | Canonical `id` values only. | N/A | These remain the canonical catalog list/search surfaces. |
| REST `/api/catalog/:id/taxonomy` and `PATCH /api/catalog/taxonomy/batch` | Canonical `item_id` values. | Bare skill IDs or canonical `skill:<skill-id>` for skill items; prompt/rule IDs are canonical-only. | Batch results preserve canonical `item_id` values and keep the original input in `requested_item_id`. |
| REST `/api/catalog/:id/metadata` and `/api/catalog/metadata?item_id=...` | Canonical `item_id` values. | Canonical item IDs only. | Bare skill fallback is intentionally not enabled on the metadata surface. |
| REST `/api/catalog/export` and `/api/catalog/materialize` request bodies | Canonical `item_id` values in manifests/results. | Bare skill IDs or canonical `skill:<skill-id>` for skill items; prompt/rule IDs are canonical-only. | Shared export/materialization normalization is bounded to skill items for legacy compatibility. |
| Legacy `/api/skills*` routes | Existing skill-name/path semantics. | Existing skill-name/path semantics. | Explicitly outside the catalog-item ID migration. |

### Classification-State Semantics

- `has_assignment` is `true` when any taxonomy field is populated.
- `is_fully_classified` is `true` when `primary_domain` exists and the item has
  at least one tag.
- `missing_fields` is an ordered list drawn only from:
  `primary_domain`, `primary_subdomain`, `secondary_domain`,
  `secondary_subdomain`, `tags`.
- Secondary domain and subdomain gaps remain visible in `missing_fields` even
  when `is_fully_classified=true`.

### List/Search and Export Defaults

- `include_content=false` is the default for REST and MCP list/search payloads.
- Canonical list/search ordering is ascending `item_id`.
- Pagination uses `limit` and `cursor`, with response metadata
  `next_cursor` and `has_more`.
- Default `limit` is `50`; maximum `limit` is `200`.
- REST `/api/catalog` and `/api/catalog/search` keep the legacy array response
  shape when callers omit both `limit` and `cursor`; paginated REST calls
  return `{items, next_cursor, has_more}`.
- MCP `list_catalog` and `search_catalog` always return structured envelopes
  (`items` or `results`, plus pagination metadata).
- `unclassified=true` maps to `has_assignment=false`;
  `missing_primary_domain=true` and `missing_tags=true` target those specific
  missing states.
- MCP `export_catalog_items` adds `archive_root_mode=flat|materialized`, with
  `flat` as the default, and `include_archive_base64=false` by default.
- REST `POST /api/catalog/export` returns materialized archive roots in the
  manifest plus download metadata for non-dry-run responses; it never includes
  inline archive bytes.
- `POST /api/catalog/materialize` and `materialize_catalog_items` remain the
  only caller-directed write paths.

### Verified Examples

#### Canonical Skill IDs on MCP

`list_skills` and `search_skills` now emit canonical skill item IDs:

```json
{
  "skills": [
    {
      "id": "skill:demo-skill",
      "name": "demo-skill"
    }
  ]
}
```

`read_skill`, `list_skill_resources`, `read_skill_resource`, and
`get_skill_resource_info` accept either `demo-skill` or `skill:demo-skill`.
Taxonomy, export, and materialization surfaces keep that fallback only for
`skill` items.

#### REST Metadata-First Catalog Page

Paginated REST list/search calls return metadata-first item payloads plus
explicit classification state:

```bash
curl -sS "http://127.0.0.1:8080/api/catalog?limit=1"
```

```json
{
  "items": [
    {
      "id": "skill:demo-skill",
      "classifier": "skill",
      "name": "demo-skill",
      "has_assignment": false,
      "is_fully_classified": false,
      "missing_fields": [
        "primary_domain",
        "primary_subdomain",
        "secondary_domain",
        "secondary_subdomain",
        "tags"
      ]
    }
  ],
  "next_cursor": "skill:demo-skill",
  "has_more": true
}
```

If you omit both `limit` and `cursor`, REST keeps the legacy array response
shape while returning the same metadata-first item fields.

#### REST Taxonomy Patch and Batch Dry-Run

Single-item taxonomy reads and writes return the same explicit classification
state fields:

```bash
curl -sS -X PATCH "http://127.0.0.1:8080/api/catalog/skill%3Ademo-skill/taxonomy" \
  -H "Content-Type: application/json" \
  --data '{"primary_domain_id":"domain-platform","tag_ids":["tag-backend"]}'
```

```json
{
  "item_id": "skill:demo-skill",
  "has_assignment": true,
  "is_fully_classified": true,
  "missing_fields": [
    "primary_subdomain",
    "secondary_domain",
    "secondary_subdomain"
  ]
}
```

Batch mutation keeps canonical output IDs and distinguishes dry-run planning
from apply behavior:

```bash
curl -sS -X PATCH "http://127.0.0.1:8080/api/catalog/taxonomy/batch" \
  -H "Content-Type: application/json" \
  --data '{
    "dry_run": true,
    "items": [
      {
        "item_id": "skill:demo-skill",
        "primary_domain_id": "domain-platform",
        "tag_ids": ["tag-backend"]
      },
      {
        "item_id": "skill:missing-item",
        "primary_domain_id": "domain-platform"
      }
    ]
  }'
```

```json
{
  "dry_run": true,
  "items": [
    {
      "item_id": "skill:demo-skill",
      "status": "planned"
    },
    {
      "item_id": "skill:missing-item",
      "status": "not_found"
    }
  ]
}
```

For compatibility, `/api/catalog/:id/taxonomy` and batch `item_id` fields also
accept bare skill IDs such as `demo-skill`; batch results preserve that raw
input in `requested_item_id` while keeping `item_id` canonical.

#### Usage / Preflight Response

REST usage endpoints and MCP usage tools share the same summary shape:

```json
{
  "object_type": "tag",
  "object_id": "tag-backend",
  "assignment_count": 1,
  "distinct_item_count": 1,
  "preview_item_ids": ["skill:taxonomy-assigned-item"],
  "blocking_reason": "in_use"
}
```

REST endpoints:
- `GET /api/catalog/taxonomy/domains/:id/usage`
- `GET /api/catalog/taxonomy/subdomains/:id/usage`
- `GET /api/catalog/taxonomy/tags/:id/usage`

MCP tools:
- `get_taxonomy_domain_usage`
- `get_taxonomy_subdomain_usage`
- `get_taxonomy_tag_usage`

#### MCP Export Options

`export_catalog_items` defaults to flattened archive roots and omits inline
archive bytes:

```json
{
  "item_ids": ["prompt:sample-skill:imports/prompts/system.md"],
  "dry_run": true
}
```

```json
{
  "manifest": {
    "items": [
      {
        "item_id": "prompt:sample-skill:imports/prompts/system.md",
        "archive_root": "system.md"
      }
    ]
  }
}
```

Callers that want materialization-style archive roots and inline bytes must opt
in explicitly:

```json
{
  "item_ids": ["prompt:sample-skill:imports/prompts/system.md"],
  "archive_root_mode": "materialized",
  "include_archive_base64": true
}
```

```json
{
  "manifest": {
    "items": [
      {
        "archive_root": "prompts/system.md"
      }
    ]
  },
  "download": {
    "archive_base64": "<base64 omitted>"
  }
}
```

REST `POST /api/catalog/export` stays download-oriented: non-dry-run responses
return manifest plus `download.file_name`, `download.content_type`, and
`download.content_length`, but never inline bytes.

### Release Guidance

- Use the `WP-008` section in [`tests/README.md`](/home/jeff/skillserver/tests/README.md) as the go/no-go regression gate for this rollout.
- Release notes and operator caveats live in [`docs/releases/2026-03-09-catalog-id-taxonomy-and-materialization-ergonomics-release-notes.md`](/home/jeff/skillserver/docs/releases/2026-03-09-catalog-id-taxonomy-and-materialization-ergonomics-release-notes.md).
- Existing ADR-specific rollback runbooks remain the operational fallback:
  [`docs/operations/domain-taxonomy-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/domain-taxonomy-rollout-rollback.md)
  and
  [`docs/operations/rule-catalog-materialization-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/rule-catalog-materialization-rollout-rollback.md).

## Installation

### From Source

```bash
git clone https://github.com/mudler/skillserver
cd skillserver
make build
```

### Using Docker

```bash
docker pull ghcr.io/mudler/skillserver:latest
```

## Configuration

SkillServer supports both **environment variables** and **command-line flags** with this precedence order:

1. Command-line flags
2. Environment variables
3. Built-in defaults

### Environment Variables

| Variable | Alternative | Default | Description |
|----------|-------------|---------|-------------|
| `SKILLSERVER_DIR` | `SKILLS_DIR` | `./skills` | Directory to store skills |
| `SKILLSERVER_PORT` | `PORT` | `8080` | Port for the web server |
| `SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS` | (none) | `false` | Enable encrypted stored credentials for private Git repositories (requires persistence + master key) |
| `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY` | (none) | (empty) | Inline master key for stored Git credentials (mutually exclusive with `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE`) |
| `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE` | (none) | (empty) | File path containing the stored-credential master key (mutually exclusive with `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY`) |
| `SKILLSERVER_ENABLE_LOGGING` | (none) | `false` | Enable logging to stderr (default: false to avoid interfering with MCP stdio) |
| `SKILLSERVER_MCP_TRANSPORT` | (none) | `both` | MCP transport mode: `stdio`, `http`, or `both` |
| `SKILLSERVER_MCP_HTTP_PATH` | (none) | `/mcp` | Absolute HTTP route path for MCP Streamable HTTP |
| `SKILLSERVER_MCP_SESSION_TIMEOUT` | (none) | `30m` | Session timeout for MCP HTTP mode (`time.ParseDuration` format) |
| `SKILLSERVER_MCP_STATELESS` | (none) | `false` | Enable stateless MCP HTTP mode |
| `SKILLSERVER_MCP_ENABLE_WRITES` | (none) | `false` | Enable MCP taxonomy write tools (kept disabled by default) |
| `SKILLSERVER_MCP_ENABLE_MATERIALIZATION` | (none) | `false` | Enable MCP materialization write tools and REST materialization capability |
| `SKILLSERVER_MCP_ALLOWED_DESTINATION_ROOTS` | (none) | (empty) | Comma-separated absolute destination roots allowed for materialization writes (required when materialization is enabled) |
| `SKILLSERVER_MCP_ENABLE_EVENT_STORE` | (none) | `true` | Enable in-memory MCP event store for replay support |
| `SKILLSERVER_MCP_EVENT_STORE_MAX_BYTES` | (none) | `10485760` | Max bytes for MCP in-memory event store (10 MiB) |
| `SKILLSERVER_CATALOG_ENABLE_PROMPTS` | (none) | `true` | Enable prompt catalog classification/indexing in unified catalog APIs/tools |
| `SKILLSERVER_CATALOG_ENABLE_RULES` | (none) | `true` | Enable rule catalog classification/indexing in unified catalog APIs/tools |
| `SKILLSERVER_CATALOG_PROMPT_DIRS` | (none) | `agent,agents,prompt,prompts` | Comma-separated directory names used for prompt catalog detection |
| `SKILLSERVER_CATALOG_RULE_DIRS` | (none) | `rule,rules` | Comma-separated directory names used for rule catalog detection |
| `SKILLSERVER_CATALOG_RULE_FILENAMES` | (none) | `agents.md,rules.md,claude.md,gemini.md` | Comma-separated markdown filenames treated as project-root rule candidates |
| `SKILLSERVER_PERSISTENCE_DATA` | (none) | `false` | Enable SQLite-backed persistence for catalog source snapshots + metadata overlays |
| `SKILLSERVER_PERSISTENCE_DIR` | (none) | (empty) | Writable persistence directory (required when `SKILLSERVER_PERSISTENCE_DATA=true`) |
| `SKILLSERVER_PERSISTENCE_DB_PATH` | (none) | `<SKILLSERVER_PERSISTENCE_DIR>/skillserver.db` | Optional SQLite DB file path (absolute path or relative to persistence dir) |
| `SKILLSERVER_ENABLE_IMPORT_DISCOVERY` | (none) | `true` | Enable imported resource discovery and `imports/...` virtual read paths |

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | `./skills` | Directory to store skills (overrides `SKILLSERVER_DIR` or `SKILLS_DIR`) |
| `--port` | `8080` | Port for the web server (overrides `SKILLSERVER_PORT` or `PORT`) |
| `--git-repos` | (empty) | Optional comma-separated Git repository URLs to seed when no persisted repo config exists |
| `--git-enable-stored-credentials` | `false` | Enable encrypted stored credentials for private Git repositories |
| `--git-credential-master-key` | (empty) | Inline master key for stored credentials (mutually exclusive with `--git-credential-master-key-file`) |
| `--git-credential-master-key-file` | (empty) | File path containing stored-credential master key (mutually exclusive with `--git-credential-master-key`) |
| `--enable-logging` | `false` | Enable logging to stderr (overrides `SKILLSERVER_ENABLE_LOGGING`) |
| `--mcp-transport` | `both` | MCP transport mode: `stdio`, `http`, or `both` |
| `--mcp-http-path` | `/mcp` | Absolute HTTP route path for MCP Streamable HTTP |
| `--mcp-session-timeout` | `30m` | Session timeout for MCP HTTP mode |
| `--mcp-stateless` | `false` | Enable stateless MCP HTTP mode |
| `--mcp-enable-writes` | `false` | Enable MCP taxonomy write tools (kept disabled by default) |
| `--mcp-enable-materialization` | `false` | Enable MCP materialization write tools and REST materialization capability |
| `--mcp-allowed-destination-roots` | (empty) | Comma-separated absolute destination roots allowed for materialization writes (required when materialization is enabled) |
| `--mcp-enable-event-store` | `true` | Enable in-memory MCP event store |
| `--mcp-event-store-max-bytes` | `10485760` | Max bytes for in-memory MCP event store |
| `--catalog-enable-prompts` | `true` | Enable prompt catalog classification/indexing |
| `--catalog-enable-rules` | `true` | Enable rule catalog classification/indexing |
| `--catalog-prompt-dirs` | `agent,agents,prompt,prompts` | Comma-separated directory names used for prompt catalog detection |
| `--catalog-rule-dirs` | `rule,rules` | Comma-separated directory names used for rule catalog detection |
| `--catalog-rule-filenames` | `agents.md,rules.md,claude.md,gemini.md` | Comma-separated markdown filenames treated as project-root rule candidates |
| `--persistence-data` | `false` | Enable SQLite-backed persistence mode |
| `--persistence-dir` | (empty) | Writable persistence directory (required when persistence mode is enabled) |
| `--persistence-db-path` | (empty) | Optional SQLite DB file path override (absolute path or relative to persistence dir) |
| `--enable-import-discovery` | `true` | Enable imported resource discovery and `imports/...` virtual read paths |

## Usage

### Basic Usage

```bash
# Using defaults
./skillserver

# Using environment variables
export SKILLSERVER_DIR=/path/to/skills
export SKILLSERVER_PORT=9090
./skillserver

# Using command-line flags
./skillserver --dir /path/to/skills --port 9090

# Using both (flags override env vars)
export SKILLSERVER_PORT=8080
./skillserver --port 9090  # Will use 9090

# Enable logging (useful for debugging, but disabled by default to avoid interfering with MCP stdio)
./skillserver --enable-logging
# Or using environment variable
export SKILLSERVER_ENABLE_LOGGING=true
./skillserver

# Roll back to legacy direct-only discovery behavior
./skillserver --enable-import-discovery=false
# Or using environment variable
export SKILLSERVER_ENABLE_IMPORT_DISCOVERY=false
./skillserver

# Roll back unified catalog to skill-only behavior
./skillserver --catalog-enable-prompts=false
# Or using environment variable
export SKILLSERVER_CATALOG_ENABLE_PROMPTS=false
./skillserver

# Override prompt classification directories (must be single directory names)
./skillserver --catalog-prompt-dirs "agent,agents,prompts"

# Roll back rule catalog indexing (keeps skill/prompt catalog behavior)
./skillserver --catalog-enable-rules=false
# Or using environment variable
export SKILLSERVER_CATALOG_ENABLE_RULES=false
./skillserver

# Enable materialization writes with explicit allowed destination roots
./skillserver \
  --mcp-enable-materialization \
  --mcp-allowed-destination-roots "/workspace,/projects"

# Enable persistence mode (stores SQLite under mounted/local persistence dir)
mkdir -p ./data/skillserver
./skillserver --persistence-data --persistence-dir ./data/skillserver

# Optional custom DB path (relative paths resolve from --persistence-dir)
./skillserver \
  --persistence-data \
  --persistence-dir ./data/skillserver \
  --persistence-db-path state/catalog.sqlite

# Enable stored credentials mode (trusted deployments only)
# Requires persistence plus one master key source.
./skillserver \
  --persistence-data \
  --persistence-dir ./data/skillserver \
  --git-enable-stored-credentials \
  --git-credential-master-key-file ./secrets/git-master-key.txt

# Roll back to filesystem-only mode (non-destructive)
./skillserver --persistence-data=false
# Or using environment variable
export SKILLSERVER_PERSISTENCE_DATA=false
./skillserver

# Disable stored credentials while keeping env/file private-repo flows available
./skillserver --git-enable-stored-credentials=false
# Or using environment variable
export SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS=false
./skillserver
```

### Transport Mode Examples

```bash
# Default mode: both stdio + HTTP transport on /mcp
./skillserver

# Stdio only (legacy/local MCP client mode)
./skillserver --mcp-transport stdio

# HTTP only (remote MCP clients via Streamable HTTP)
./skillserver --mcp-transport http --mcp-http-path /mcp

# Both transports with custom HTTP tuning
./skillserver \
  --mcp-transport both \
  --mcp-http-path /mcp \
  --mcp-session-timeout 45m \
  --mcp-enable-writes false \
  --mcp-enable-event-store true \
  --mcp-event-store-max-bytes 2097152
```

`both` mode behavior: if stdio disconnects/exits, the HTTP transport remains active.

### With Git Synchronization

```bash
# Optional bootstrap seeding on first start
./skillserver --git-repos "https://github.com/user/repo1.git,https://github.com/user/repo2.git"

# Preferred ongoing workflow: add/edit repos in the Web UI
# (Settings persist in the repo config file and continue syncing from origin)
```

Note: there is no specific layout that the repository needs to follow. The only requirements is that in every skill you have a `SKILL.md` file, and that gets scanned automatically.

See [here](https://github.com/anthropics/skills) for an example repository.

### Docker Usage

```bash
# Using environment variables
docker run -p 8080:8080 \
  -e SKILLSERVER_DIR=/app/skills \
  -e SKILLSERVER_PORT=8080 \
  -v $(pwd)/skills:/app/skills \
  ghcr.io/mudler/skillserver:latest

# Using command-line flags
docker run -p 8080:8080 \
  -v $(pwd)/skills:/app/skills \
  ghcr.io/mudler/skillserver:latest \
  --dir /app/skills --port 8080

# Optional one-time bootstrap seeding
docker run -p 8080:8080 \
  -v $(pwd)/skills:/app/skills \
  ghcr.io/mudler/skillserver:latest \
  --dir /app/skills --port 8080 --git-repos "https://github.com/user/repo.git"
```

With MCP HTTP transport enabled:

```bash
docker run -p 8080:8080 \
  -v $(pwd)/skills:/app/skills \
  ghcr.io/mudler/skillserver:latest \
  --dir /app/skills \
  --port 8080 \
  --mcp-transport http \
  --mcp-http-path /mcp
```

With persistence mode enabled (SQLite persisted to mounted volume):

```bash
docker volume create skillserver-persistence

docker run -p 8080:8080 \
  -v $(pwd)/skills:/app/skills \
  -v skillserver-persistence:/var/lib/skillserver/persistence \
  -e SKILLSERVER_DIR=/app/skills \
  -e SKILLSERVER_PORT=8080 \
  -e SKILLSERVER_MCP_TRANSPORT=http \
  -e SKILLSERVER_PERSISTENCE_DATA=true \
  -e SKILLSERVER_PERSISTENCE_DIR=/var/lib/skillserver/persistence \
  ghcr.io/mudler/skillserver:latest
```

### Kubernetes Persistence Mode (PVC)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: skillserver
spec:
  template:
    spec:
      containers:
        - name: skillserver
          image: ghcr.io/mudler/skillserver:latest
          args: ["--dir", "/app/skills", "--port", "8080", "--mcp-transport", "http"]
          env:
            - name: SKILLSERVER_PERSISTENCE_DATA
              value: "true"
            - name: SKILLSERVER_PERSISTENCE_DIR
              value: /var/lib/skillserver/persistence
          volumeMounts:
            - name: skills
              mountPath: /app/skills
            - name: skillserver-persistence
              mountPath: /var/lib/skillserver/persistence
      volumes:
        - name: skills
          persistentVolumeClaim:
            claimName: skillserver-skills-pvc
        - name: skillserver-persistence
          persistentVolumeClaim:
            claimName: skillserver-persistence-pvc
```

## Private Git Credentials (ADR-006)

Canonical ADR: [`docs/adrs/006-private-git-repository-credential-sources.md`](/home/jeff/skillserver/docs/adrs/006-private-git-repository-credential-sources.md)

Production guidance:
- Prefer `auth.source=env` or `auth.source=file`.
- Use `auth.source=stored` only in trusted deployments where UI/API access is protected by external auth + TLS.
- Treat master-key rotation as a planned operation: stored credentials encrypted with a previous key require coordinated re-encryption/migration before old key retirement.
- Stored credentials require all of:
  - `SKILLSERVER_PERSISTENCE_DATA=true`
  - `SKILLSERVER_PERSISTENCE_DIR=<writable path>`
  - `SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS=true`
  - one master key source: `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY` or `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE`

### Git Repo API Contract (Secret-Safe)

Add/update request fields:
- `url`
- `enabled` (optional)
- `auth` (optional):
  - `mode`: `none` | `https_token` | `https_basic` | `ssh_key`
  - `source`: `none` | `env` | `file` | `stored`
  - `reference_id` (stored only, optional)
  - `username_ref`, `password_ref`, `token_ref`, `key_ref`, `known_hosts_ref` (env/file only)
- `stored_credential` (write-only, stored source only):
  - `username`, `password`, `token`, `private_key`, `passphrase`, `known_hosts`

Secret-safe response fields:
- `id`, `url`, `name`, `enabled`
- `auth_mode`, `credential_source`, `has_credentials`
- `stored_credentials_enabled`
- `last_sync_status`, `last_sync_error`

### Local Setup Examples

Public repository (no auth):

```bash
curl -sS -X POST "http://127.0.0.1:8080/api/git-repos" \
  -H "Content-Type: application/json" \
  --data '{
    "url": "https://github.com/mudler/skillserver.git"
  }'
```

Private HTTPS token via environment variable reference:

```bash
export REPO_ACME_PAT="***"

curl -sS -X POST "http://127.0.0.1:8080/api/git-repos" \
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

Private SSH key via mounted files:

```bash
curl -sS -X POST "http://127.0.0.1:8080/api/git-repos" \
  -H "Content-Type: application/json" \
  --data '{
    "url": "git@github.com:acme/private-skills.git",
    "auth": {
      "mode": "ssh_key",
      "source": "file",
      "key_ref": "/run/secrets/git/private_key",
      "known_hosts_ref": "/run/secrets/git/known_hosts"
    }
  }'
```

Stored credential mode (write-only submission):

```bash
curl -sS -X POST "http://127.0.0.1:8080/api/git-repos" \
  -H "Content-Type: application/json" \
  --data '{
    "url": "https://github.com/acme/private-skills.git",
    "auth": {
      "mode": "https_basic",
      "source": "stored",
      "reference_id": "acme/private-skills"
    },
    "stored_credential": {
      "username": "git-bot",
      "password": "***"
    }
  }'
```

### Docker: Env/File Secret Injection

```bash
docker run -p 8080:8080 \
  -v $(pwd)/skills:/app/skills \
  -v $(pwd)/secrets/git:/run/secrets/git:ro \
  -e REPO_ACME_PAT="***" \
  ghcr.io/mudler/skillserver:latest
```

Add/update repositories through the Web UI or `POST /api/git-repos`. Use `auth.source=env` with `token_ref=REPO_ACME_PAT`, or `auth.source=file` with file paths under `/run/secrets/git`.

### Kubernetes Secret Patterns (Env + File)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: skillserver
spec:
  template:
    spec:
      containers:
        - name: skillserver
          image: ghcr.io/mudler/skillserver:latest
          env:
            - name: REPO_ACME_PAT
              valueFrom:
                secretKeyRef:
                  name: private-git-credentials
                  key: token
          volumeMounts:
            - name: git-ssh
              mountPath: /var/run/secrets/git
              readOnly: true
      volumes:
        - name: git-ssh
          secret:
            secretName: private-git-ssh
```

Use API `auth.source=env` for env vars and `auth.source=file` for mounted file paths.

### Vault-Projected Env/File Patterns

SkillServer does not need direct Vault API access for ADR-006. Use projected env vars or files from Vault Agent Injector / External Secrets / CSI, then configure `auth.source=env` or `auth.source=file` in repo metadata.

### Rollback Guidance (Stored -> Env/File/Public)

1. Disable stored mode at runtime:

```bash
export SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS=false
./skillserver
```

2. Convert repos from `stored` to `env`/`file` references or public mode via `PUT /api/git-repos/:id`.
3. Re-run manual sync with `POST /api/git-repos/:id/sync`.
4. Keep encrypted rows in SQLite for recovery unless policy requires deletion.

Detailed runbook: [`docs/operations/private-git-credential-sources-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/private-git-credential-sources-rollout-rollback.md)

### Remote MCP (Streamable HTTP) Usage

```bash
# Start server in HTTP mode (or keep default "both")
./skillserver --mcp-transport http --mcp-http-path /mcp

ENDPOINT="http://localhost:8080/mcp"

# 1) Initialize a session
curl -i -X POST "$ENDPOINT" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "MCP-Protocol-Version: 2025-06-18" \
  --data '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"curl-client","version":"1.0.0"}}}'

# Capture the Mcp-Session-Id response header from the initialize call
SESSION_ID="<paste-session-id>"

# 2) Close the session when done
curl -i -X DELETE "$ENDPOINT" \
  -H "Mcp-Session-Id: $SESSION_ID"
```

For stateless mode (`--mcp-stateless=true`), clients do not need session lifecycle calls.

### MCP HTTP Troubleshooting

#### Session Initialization Issues

Symptoms:
- `POST /mcp` does not return `200 OK`
- No `Mcp-Session-Id` in response headers (stateful mode)

Checks:
- Confirm MCP HTTP transport is enabled: `--mcp-transport http` or `--mcp-transport both`
- Confirm path is absolute and correct: `--mcp-http-path /mcp`
- Send required initialize headers and payload:
  - `Content-Type: application/json`
  - `Accept: application/json, text/event-stream`
  - `MCP-Protocol-Version: 2025-06-18`

#### Header / Protocol Mismatch

Symptoms:
- `405 Method Not Allowed` on `GET /mcp` without a session
- `400 Bad Request` when replaying streams
- `404 session not found` after posting with stale/invalid `Mcp-Session-Id`

Remediation:
- Initialize first with `POST` and keep the returned `Mcp-Session-Id` for subsequent stateful requests
- Keep `MCP-Protocol-Version` consistent per session
- If using replay (`Last-Event-ID`), ensure event store is enabled (`--mcp-enable-event-store=true`)

#### Route Conflict Symptoms

Symptoms:
- MCP client receives HTML instead of JSON/SSE
- MCP requests hit UI/API handlers instead of MCP handler

Remediation:
- Use a dedicated MCP path such as `/mcp` (default)
- Avoid reusing broad UI/API paths such as `/` or `/api/*`
- Verify route with:
  - `curl -i -X OPTIONS http://localhost:8080/mcp`
  - Expected: handler responds on MCP route methods (`GET`, `POST`, `DELETE`, `OPTIONS`)

#### Quick Rollback to Stdio Mode

Use stdio-only mode to immediately disable MCP HTTP exposure:

```bash
# Flag-based rollback
./skillserver --mcp-transport stdio

# Environment-based rollback
export SKILLSERVER_MCP_TRANSPORT=stdio
./skillserver
```

## Skill/Rule/Prompt Relationship Metadata (ADR-008)

Canonical ADR: [`docs/adrs/008-skill-rule-and-prompt-relationship-metadata.md`](/home/jeff/skillserver/docs/adrs/008-skill-rule-and-prompt-relationship-metadata.md)

Relationship metadata is additive:
- a `skill` can reference zero-or-one `prompt`
- a `skill` can reference zero-or-more `rule` items
- `prompt` and `rule` metadata views expose reverse-related `skills`

Behavior notes:
- Relationship edits are metadata-only and do not change `content_writable`, `metadata_writable`, or Git-backed read-only semantics.
- GUI and REST writes are skill-owned only in v1.
- MCP is read-only for relationship metadata in v1.
- `GET /api/catalog` and `GET /api/catalog/search` remain relationship-light.
- Catalog tiles intentionally do not render relationship badges or chips in v1.

### Relationship Read Surfaces

- REST `GET /api/catalog/:id/metadata`
- REST `GET /api/catalog/metadata?item_id=...`
- MCP `get_catalog_item_relationships`

All read surfaces return the same normalized envelope:

```json
{
  "item_id": "skill:demo-skill",
  "relationships": {
    "prompt": {
      "id": "prompt:demo-skill:prompts/system.md",
      "classifier": "prompt",
      "name": "system",
      "parent_skill_id": "skill:demo-skill",
      "resource_path": "prompts/system.md"
    },
    "rules": [
      {
        "id": "rule:demo-skill:rules/security.md",
        "classifier": "rule",
        "name": "security",
        "parent_skill_id": "skill:demo-skill",
        "resource_path": "rules/security.md"
      }
    ],
    "skills": []
  }
}
```

Classifier semantics:
- `skill` metadata populates forward `prompt` and `rules`; `skills` stays empty.
- `prompt` metadata populates reverse `skills`; `prompt` is `null` and `rules` is empty.
- `rule` metadata populates reverse `skills`; `prompt` is `null` and `rules` is empty.

ID compatibility:
- REST relationship surfaces are canonical-only.
- MCP `get_catalog_item_relationships` accepts bare `<skill-id>` only when the target item is a `skill`.
- `prompt` and `rule` reads require canonical item IDs on both REST and MCP surfaces.

### Relationship Write Surface

Use REST `PATCH /api/catalog/:id/relationships` to update skill-owned metadata:

```json
{
  "prompt_item_id": "prompt:demo-skill:prompts/system.md",
  "rule_item_ids": [
    "rule:demo-skill:rules/security.md",
    "rule:demo-skill:rules/style.md"
  ],
  "updated_by": "gui"
}
```

Write semantics:
- Path `:id` must resolve to a `skill` item.
- `prompt_item_id`:
  - canonical prompt ID string sets or replaces the current prompt link
  - explicit `null` clears the current prompt link
  - omission leaves the prompt link unchanged
- `rule_item_ids`:
  - present replaces the full rule set
  - omission leaves rules unchanged
  - duplicate IDs are rejected
- Prompt and rule write attempts return `403`.
- Unknown source/target items return `404`.
- Validation failures return `400`.

### Verification Evidence

Treat the following as the release-readiness evidence for ADR-008:
- [`WP-008 completion summary`](/home/jeff/skillserver/docs/implementation-plans/skill-rule-and-prompt-relationship-metadata/work-packages/completion-summaries/WP-008-completion-summary.md)
- `go test ./pkg/web -run 'TestCatalogRelationshipMetadataEndpoints|TestCatalogMetadataEndpoints' -count=1`
- `go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1`
- `npx playwright test tests/playwright/wp007-ui-relationship-metadata-editor.spec.ts --project=chromium`
- `npx playwright test tests/playwright/wp008-ui.spec.ts --project=chromium`

Detailed rollout and rollback runbook: [`docs/operations/skill-relationship-metadata-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/skill-relationship-metadata-rollout-rollback.md)

## MCP Client Configuration

SkillServer supports MCP over stdio and Streamable HTTP. The examples below are stdio-based client configurations.

**Note:** When using SkillServer as an MCP server, logging is disabled by default to avoid interfering with the stdio protocol. Enable it only for debugging purposes.

### [Wiz](https://github.com/mudler/wiz)

Add SkillServer to your Wiz configuration file (typically `~/.config/wiz/config.yaml` or similar):

```yaml
mcp_servers:
  skillserver:
    command: docker
    args:
      - "run"
      - "-i"
      - "--rm"
      - "-v"
      - "/host/path/to/skills:/app/skills"
      - "ghcr.io/mudler/skillserver:latest"
    env:
      SKILLSERVER_DIR: "/app/skills"
      SKILLSERVER_PORT: "9090"
      # Enable logging for debugging (default: false, disabled to avoid interfering with MCP stdio)
      # SKILLSERVER_ENABLE_LOGGING: "true"
```

### [LocalAI](https://github.com/mudler/LocalAI)

Add SkillServer to your LocalAI MCP configuration (typically in your LocalAI model config file):

```yaml
mcp:
  stdio: |
    {
      "mcpServers": {
        "skillserver": {
          "command": "docker",
          "args": [
            "run", "-i", "--rm",
            "-v", "/host/path/to/skills:/app/skills",
            "-e", "SKILLSERVER_DIR=/app/skills",
            "-e", "SKILLSERVER_PORT=9090",
            "ghcr.io/mudler/skillserver:latest"
          ]
        }
      }
    }
```

With Git synchronization:

```yaml
mcp:
  stdio: |
    {
      "mcpServers": {
        "skillserver": {
          "command": "docker",
          "args": [
            "run", "-i", "--rm",
            "-v", "/host/path/to/skills:/app/skills",
            "ghcr.io/mudler/skillserver:latest",
            "--dir", "/app/skills",
            "--port", "9090",
            "--git-repos", "https://github.com/user/repo.git"
          ]
        }
      }
    }
```

### Claude Desktop

Add SkillServer to your Claude Desktop MCP configuration (typically `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS or `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "skillserver": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "/host/path/to/skills:/app/skills",
        "ghcr.io/mudler/skillserver:latest"
      ],
      "env": {
        "SKILLSERVER_DIR": "/app/skills",
        "SKILLSERVER_PORT": "9090"
      }
    }
  }
}
```

### Cline / Other MCP Clients

Most MCP clients support stdio-based servers. Configure SkillServer using Docker:

```yaml
# Generic MCP client configuration
mcp_servers:
  skillserver:
    command: docker
    args:
      - "run"
      - "-i"
      - "--rm"
      - "-v"
      - "/host/path/to/skills:/app/skills"
      - "ghcr.io/mudler/skillserver:latest"
    env:
      SKILLSERVER_DIR: "/app/skills"
      SKILLSERVER_PORT: "9090"
```

**Using the binary directly** (if you prefer not to use Docker):

```yaml
mcp_servers:
  skillserver:
    command: /path/to/skillserver
    args: []  # Optional command-line arguments
    env:      # Optional environment variables
      SKILLSERVER_DIR: "/path/to/skills"
      SKILLSERVER_PORT: "9090"
```

## Skill Format

Skills follow the [Agent Skills specification](https://agentskills.io). Each skill is a directory containing:

- **SKILL.md** (required): Markdown file with YAML frontmatter containing:
  - `name` (required): Skill name matching directory name
  - `description` (required): Description of what the skill does
  - `license` (optional): License information
  - `compatibility` (optional): Environment requirements
  - `metadata` (optional): Additional metadata
  - `allowed-tools` (optional): Pre-approved tools

- **scripts/** (optional): Executable code (Python, Bash, JavaScript, etc.)
- **references/** (optional): Additional documentation files
- **assets/** (optional): Static resources (templates, images, data files)
- **agents/** (optional): Agent prompt files
- **prompts/** (optional): Prompt files (system, assistant, etc.)

Example structure:
```
my-skill/
├── SKILL.md
├── scripts/
│   └── process.py
├── agents/
│   └── coach.md
├── prompts/
│   └── system.md
├── references/
│   └── API.md
└── assets/
    └── template.docx
```

Imported resources referenced by `SKILL.md` links/includes are exposed as virtual read-only paths under `imports/...` (for example `imports/prompts/shared.md`).

## API Endpoints

### REST API

#### Skills
- `GET /api/skills` - List all skills (local and from git repos)
- `GET /api/skills/:name` - Get skill content
- `POST /api/skills` - Create new skill
- `PUT /api/skills/:name` - Update skill (blocks read-only skills)
- `DELETE /api/skills/:name` - Delete skill (blocks read-only skills)
- `GET /api/skills/search?q=query` - Search skills

#### Git Repositories (ADR-006, additive)
- `GET /api/git-repos` - List configured repositories with secret-safe auth summary
- `POST /api/git-repos` - Add repository (`url`, optional `enabled`, optional `auth`, optional write-only `stored_credential`)
- `PUT /api/git-repos/:id` - Update repository using canonical URL + stable ID behavior
- `DELETE /api/git-repos/:id` - Delete repository config and checkout
- `POST /api/git-repos/:id/toggle` - Toggle repository enabled state
- `POST /api/git-repos/:id/sync` - Trigger manual sync for one enabled repository
- Response auth/sync fields: `auth_mode`, `credential_source`, `has_credentials`, `stored_credentials_enabled`, `last_sync_status`, `last_sync_error`

#### Runtime Capabilities (ADR-004/005/006/007 support)
- `GET /api/runtime/capabilities` - Return runtime capability gates (for example `git.stored_credentials_enabled`, `catalog.rules_enabled`, `mcp.materialization_enabled`, `mcp.allowed_destination_roots`)

#### Catalog (ADR-003 + ADR-007, additive)
- `GET /api/catalog` - List unified catalog items (`skill` + `prompt` + `rule`) with metadata-first fields including `id`, `classifier`, `name`, `description`, `parent_skill_id`, `resource_path`, `custom_metadata`, `labels`, `content_writable`, `metadata_writable`, `read_only`, `has_assignment`, `is_fully_classified`, and `missing_fields`
- `GET /api/catalog/search?q=query&classifier=skill|prompt|rule` - Search unified catalog items with optional classifier filter and the same metadata-first item fields
- REST list/search compatibility note: omitting both `limit` and `cursor` keeps the legacy array response shape; paginated calls return `{items, next_cursor, has_more}`
- `POST /api/catalog/export` - Export one or more catalog items as a `tar.gz` archive with optional dry-run planning (`item_ids`, optional `format`, optional `dry_run`); accepts bare skill IDs only for `skill` items
- `POST /api/catalog/materialize` - Plan or materialize one or more catalog items into an absolute destination directory (`item_ids`, `destination_dir`, optional `conflict_policy=error|overwrite|skip`, optional `dry_run`)
- Optional taxonomy filters for both list/search:
  - `primary_domain_id`
  - `secondary_domain_id`
  - `subdomain_id` (matches primary or secondary subdomain)
  - `tag_ids` (comma-separated IDs)
  - `tag_match=any|all` (defaults to `any`)
- Optional classification-state filters for both list/search:
  - `unclassified`
  - `missing_primary_domain`
  - `missing_tags`
- `classifier` is case-insensitive at input and normalized to `skill`, `prompt`, or `rule` in responses
- Invalid classifier values return `400` (`invalid catalog classifier ...`)
- Empty or missing `q` for `/api/catalog/search` returns `400` (`query parameter 'q' is required`)
- `POST /api/catalog/materialize` returns `403` (`catalog materialization capability is disabled`) when materialization capability is disabled.
- `GET /api/catalog/:id/metadata` - Return source + overlay + effective metadata projections for one catalog item (canonical item IDs only)
- `GET /api/catalog/metadata?item_id=...` - Query-form metadata read for one catalog item (canonical item IDs only)
- `PATCH /api/catalog/:id/metadata` - Update metadata overlays for one catalog item (`display_name`, `description`, `labels`, `custom_metadata`, optional `updated_by`)
- Metadata responses include an additive `relationships` object:
  - `prompt` - zero-or-one related prompt for `skill` items, otherwise `null`
  - `rules` - ordered related rules for `skill` items, otherwise empty
  - `skills` - reverse-related skills for `prompt` and `rule` items, otherwise empty
- `PATCH /api/catalog/:id/relationships` - Replace skill-owned relationship metadata (`prompt_item_id`, `rule_item_ids`, optional `updated_by`); REST is canonical-only and rejects prompt/rule write attempts
- Relationship detail is intentionally absent from `GET /api/catalog` and `GET /api/catalog/search`; fetch it from metadata/detail surfaces only

#### Taxonomy (ADR-005, additive; persistence mode required)
- `GET /api/catalog/:id/taxonomy` - Get taxonomy assignment metadata for one catalog item (`has_assignment`, `is_fully_classified`, `missing_fields`, bare skill IDs accepted for `skill` items)
- `PATCH /api/catalog/:id/taxonomy` - Patch taxonomy assignment metadata for one catalog item (`tag_ids`, additive `add_tag_ids`, `remove_tag_ids`, `clear_tags`, optional `updated_by`)
- `PATCH /api/catalog/taxonomy/batch` - Dry-run or apply batch taxonomy mutations (`dry_run`, `items[].item_id`, taxonomy selectors, additive tag mutation fields)
- `GET /api/catalog/taxonomy/domains` - List taxonomy domains (`domain_id`, `domain_ids`, `key`, `keys`, `active` filters)
- `POST /api/catalog/taxonomy/domains` - Create taxonomy domain
- `PATCH /api/catalog/taxonomy/domains/:id` - Update taxonomy domain
- `DELETE /api/catalog/taxonomy/domains/:id` - Delete taxonomy domain
- `GET /api/catalog/taxonomy/domains/:id/usage` - Get domain usage/preflight summary (`assignment_count`, `distinct_item_count`, `preview_item_ids`, optional `blocking_reason`)
- `GET /api/catalog/taxonomy/subdomains` - List taxonomy subdomains (`subdomain_id`, `subdomain_ids`, `domain_id`, `domain_ids`, `key`, `keys`, `active` filters)
- `POST /api/catalog/taxonomy/subdomains` - Create taxonomy subdomain
- `PATCH /api/catalog/taxonomy/subdomains/:id` - Update taxonomy subdomain
- `DELETE /api/catalog/taxonomy/subdomains/:id` - Delete taxonomy subdomain
- `GET /api/catalog/taxonomy/subdomains/:id/usage` - Get subdomain usage/preflight summary
- `GET /api/catalog/taxonomy/tags` - List taxonomy tags (`tag_id`, `tag_ids`, `key`, `keys`, `active` filters)
- `POST /api/catalog/taxonomy/tags` - Create taxonomy tag
- `PATCH /api/catalog/taxonomy/tags/:id` - Update taxonomy tag
- `DELETE /api/catalog/taxonomy/tags/:id` - Delete taxonomy tag
- `GET /api/catalog/taxonomy/tags/:id/usage` - Get tag usage/preflight summary
- Taxonomy endpoints return `503` when persistence runtime is disabled/unavailable.

#### Resources
- `GET /api/skills/:name/resources` - List resources with legacy buckets (`scripts`, `references`, `assets`) plus additive groups (`prompts`, `imported`, `groups`) when present; each resource includes `origin` and `writable`
- `GET /api/skills/:name/resources/*` - Get/download a resource file
- `POST /api/skills/:name/resources` - Upload/create a direct resource (multipart/form-data or JSON); imported `imports/...` targets are blocked
- `PUT /api/skills/:name/resources/*` - Update a direct resource file; imported `imports/...` targets are blocked
- `DELETE /api/skills/:name/resources/*` - Delete a direct resource; imported `imports/...` targets are blocked

### MCP Tools

#### Skills
- `list_skills` - List all available skills (returns canonical skill item IDs and populated names)
- `read_skill` - Read the full content of a skill by its ID (accepts bare or canonical skill IDs)
- `search_skills` - Search for skills by query string (returns canonical skill item IDs and populated names)

#### Catalog (ADR-003 + ADR-007, additive)
- `list_catalog` - List unified catalog items with optional `classifier` filter (`skill`, `prompt`, or `rule`), optional taxonomy filters (`primary_domain_id`, `secondary_domain_id`, `subdomain_id`, `tag_ids`, `tag_match`), optional classification-state filters (`unclassified`, `missing_primary_domain`, `missing_tags`), and optional `include_content`
- `search_catalog` - Search unified catalog items by `query`, with optional classifier/taxonomy/classification-state filters and optional `include_content`
- `get_catalog_item_relationships` - Return one catalog item's relationship metadata (`item_id`, `relationships.prompt`, `relationships.rules`, `relationships.skills`) using the same normalized envelope as REST metadata reads; accepts bare skill IDs only for `skill` items and requires canonical IDs for `prompt`/`rule` items
- `export_catalog_items` - Export one or more catalog items as `tar.gz` with optional dry-run planning output, optional `archive_root_mode=flat|materialized`, and optional `include_archive_base64=true`
- `materialize_catalog_items` - Materialize one or more catalog items into an allowed destination directory (registered only when materialization gate is enabled)
- Taxonomy read tools (always registered):
  - `list_taxonomy_domains`
  - `list_taxonomy_subdomains`
  - `list_taxonomy_tags`
  - `get_catalog_item_taxonomy`
  - `get_taxonomy_domain_usage`
  - `get_taxonomy_subdomain_usage`
  - `get_taxonomy_tag_usage`
- Taxonomy write tools (registered only when `--mcp-enable-writes=true` or `SKILLSERVER_MCP_ENABLE_WRITES=true`):
  - `create_taxonomy_domain`, `update_taxonomy_domain`, `delete_taxonomy_domain`
  - `create_taxonomy_subdomain`, `update_taxonomy_subdomain`, `delete_taxonomy_subdomain`
  - `create_taxonomy_tag`, `update_taxonomy_tag`, `delete_taxonomy_tag`
  - `patch_catalog_item_taxonomy`
  - `patch_catalog_items_taxonomy`
- Relationship tooling note:
  - MCP exposes relationship reads only in v1.
  - No MCP relationship write tool is registered.
- Optional migration strategy:
  - Existing clients can keep using `list_skills`/`search_skills`
  - New mixed-item clients should adopt `list_catalog`/`search_catalog` for classifier-aware behavior

#### Resources
- `list_skill_resources` - List resources in a skill, including additive prompt/imported resources; each item includes `origin` and `writable`
- `read_skill_resource` - Read the content of a resource file (UTF-8 for text, base64 for binary, max 1MB), including `imports/...` paths when import discovery is enabled
- `get_skill_resource_info` - Get metadata (`type`, `origin`, `writable`, size, mime) without reading content

## Unified Catalog Rollout and Rollback (ADR-003)

Runtime controls:
- Flag: `--catalog-enable-prompts=true|false`
- Env: `SKILLSERVER_CATALOG_ENABLE_PROMPTS=true|false`
- Flag: `--catalog-prompt-dirs=agent,agents,prompt,prompts`
- Env: `SKILLSERVER_CATALOG_PROMPT_DIRS=agent,agents,prompt,prompts`

Rollback options:
- Prompt kill-switch rollback to skill-only catalog:
  - `./skillserver --catalog-enable-prompts=false`
- Prompt directory rollback to known-safe defaults:
  - `./skillserver --catalog-prompt-dirs "agent,agents,prompt,prompts"`

Detailed rollout/rollback runbook: [`docs/operations/unified-catalog-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/unified-catalog-rollout-rollback.md)

## Persistent Catalog Rollout and Rollback (ADR-004)

Runtime controls:
- Flag: `--persistence-data=true|false`
- Env: `SKILLSERVER_PERSISTENCE_DATA=true|false`
- Flag: `--persistence-dir=/path/to/mounted/writable/dir`
- Env: `SKILLSERVER_PERSISTENCE_DIR=/path/to/mounted/writable/dir`
- Flag: `--persistence-db-path=skillserver.db` (or absolute path)
- Env: `SKILLSERVER_PERSISTENCE_DB_PATH=skillserver.db` (or absolute path)

Behavior notes:
- Persistence mode is opt-in and defaults to disabled.
- When persistence is enabled, startup fails fast if mount/path guardrails are invalid.
- Metadata overlay endpoints (`GET/PATCH /api/catalog/:id/metadata`) require persistence mode and return `503` when unavailable.
- Rollback to filesystem-only mode is non-destructive and does not require deleting the SQLite file.

Quick rollback:

```bash
# Flag-based rollback
./skillserver --persistence-data=false

# Env-based rollback
export SKILLSERVER_PERSISTENCE_DATA=false
./skillserver
```

Detailed rollout/rollback runbook: [`docs/operations/persistence-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/persistence-rollout-rollback.md)

## Domain/Subdomain/Tag Taxonomy Rollout and Rollback (ADR-005)

Runtime controls:
- Flag: `--mcp-enable-writes=true|false`
- Env: `SKILLSERVER_MCP_ENABLE_WRITES=true|false`
- Persistence controls from ADR-004 remain required for durable taxonomy APIs.

Behavior notes:
- MCP taxonomy write tools are disabled by default and require explicit enablement.
- Taxonomy REST endpoints and taxonomy-filtered catalog list/search require persistence runtime.

Quick rollback:

```bash
# Immediate MCP write-gate rollback
./skillserver --mcp-enable-writes=false

# Equivalent env override
export SKILLSERVER_MCP_ENABLE_WRITES=false
./skillserver
```

Detailed rollout/rollback runbook: [`docs/operations/domain-taxonomy-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/domain-taxonomy-rollout-rollback.md)

## Rule Catalog and Materialization Rollout and Rollback (ADR-007)

Runtime controls:
- Flag: `--catalog-enable-rules=true|false`
- Env: `SKILLSERVER_CATALOG_ENABLE_RULES=true|false`
- Flag: `--catalog-rule-dirs=rule,rules`
- Env: `SKILLSERVER_CATALOG_RULE_DIRS=rule,rules`
- Flag: `--catalog-rule-filenames=agents.md,rules.md,claude.md,gemini.md`
- Env: `SKILLSERVER_CATALOG_RULE_FILENAMES=agents.md,rules.md,claude.md,gemini.md`
- Flag: `--mcp-enable-materialization=true|false`
- Env: `SKILLSERVER_MCP_ENABLE_MATERIALIZATION=true|false`
- Flag: `--mcp-allowed-destination-roots=/workspace,/projects`
- Env: `SKILLSERVER_MCP_ALLOWED_DESTINATION_ROOTS=/workspace,/projects`

Behavior notes:
- Rule indexing is enabled by default and can be disabled without destructive schema rollback.
- Materialization writes are disabled by default and require at least one absolute allowed destination root when enabled.
- `export_catalog_items` and `POST /api/catalog/export` remain available when materialization writes are disabled.

Quick rollback:

```bash
# Immediate write-gate rollback
./skillserver --mcp-enable-materialization=false

# Optional classifier rollback to hide rule items
./skillserver --catalog-enable-rules=false
```

Detailed rollout/rollback runbook: [`docs/operations/rule-catalog-materialization-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/rule-catalog-materialization-rollout-rollback.md)

## Private Git Credential Sources Rollout and Rollback (ADR-006)

Runtime controls:
- Flag: `--git-enable-stored-credentials=true|false`
- Env: `SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS=true|false`
- Flag: `--git-credential-master-key=<key>`
- Env: `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY=<key>`
- Flag: `--git-credential-master-key-file=/path/to/key`
- Env: `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE=/path/to/key`
- Persistence controls from ADR-004 remain required when stored credentials are enabled.

Behavior notes:
- `env` and `file` credential sources are the preferred production path.
- Stored credentials are disabled by default and require persistence + master key + trusted auth/TLS boundary.
- Disabling stored mode does not disable public or env/file-backed repositories.

Quick rollback:

```bash
# Disable stored-credential mode
./skillserver --git-enable-stored-credentials=false

# Equivalent env override
export SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS=false
./skillserver
```

Detailed rollout/rollback runbook: [`docs/operations/private-git-credential-sources-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/private-git-credential-sources-rollout-rollback.md)

## Skill/Rule/Prompt Relationship Metadata Rollout and Rollback (ADR-008)

Runtime controls:
- ADR-008 adds no feature-specific flags or environment variables.
- Relationship metadata depends on ADR-004 persistence runtime:
  - `--persistence-data=true|false`
  - `--persistence-dir=/path/to/writable/dir`
  - `--persistence-db-path=skillserver.db` (or absolute path)
  - `SKILLSERVER_PERSISTENCE_DATA`
  - `SKILLSERVER_PERSISTENCE_DIR`
  - `SKILLSERVER_PERSISTENCE_DB_PATH`

Behavior notes:
- Relationship metadata is additive; list/search payloads remain relationship-light.
- GUI and REST writes stay skill-owned only.
- MCP exposes read-only relationship lookup through `get_catalog_item_relationships`.
- Preferred rollback is deployment rollback to the last pre-ADR-008 build while leaving SQLite data intact.
- Broader fallback remains available by disabling ADR-004 persistence mode, but that also removes other persistence-backed metadata/taxonomy surfaces.

Quick fallback:

```bash
# Broad metadata/taxonomy fallback
./skillserver --persistence-data=false

# Equivalent env override
export SKILLSERVER_PERSISTENCE_DATA=false
./skillserver
```

Detailed rollout/rollback runbook: [`docs/operations/skill-relationship-metadata-rollout-rollback.md`](/home/jeff/skillserver/docs/operations/skill-relationship-metadata-rollout-rollback.md)

## Dynamic Resource Discovery and Rollout Control

- Direct resources are discovered from `scripts/`, `references/`, `assets/`, `agents/`, and `prompts/`.
- Imported references found in `SKILL.md` are surfaced as virtual read-only paths under `imports/...`.
- `origin` is `direct` or `imported`.
- `writable` is `false` for imported resources and git-backed skill resources.

### Import Discovery Rollback

```bash
# Disable imported discovery and imports/... read paths
./skillserver --enable-import-discovery=false

# Equivalent env override
export SKILLSERVER_ENABLE_IMPORT_DISCOVERY=false
./skillserver
```

Detailed rollout and rollback procedure: [`docs/operations/dynamic-resource-import-discovery-rollout.md`](/home/jeff/skillserver/docs/operations/dynamic-resource-import-discovery-rollout.md)

## Web Interface

The web UI provides a user-friendly interface for managing skills:

### Skill Management
- **Create Skills**: Create new skills with proper frontmatter validation
- **Edit Skills**: Edit skill content and metadata (read-only for git repo skills)
- **Delete Skills**: Delete local skills (read-only skills cannot be deleted)
- **Search**: Full-text search across all skills

### Resource Management
- **Upload Resources**: Upload files to direct writable groups (`scripts/`, `references/`, `assets/`, `agents/`, `prompts/`)
- **View Resources**: Click text files to view/edit, binary files to download
- **Edit Resources**: Edit text-based direct resources when `writable=true`
- **Delete Resources**: Remove direct resources from skills (read-only/imported resources protected)

### Features
- **Read-Only Indicators**: Skills from git repositories are clearly marked and protected
- **Real-time Validation**: Skill name validation according to Agent Skills spec
- **Unified Catalog Tiles**: Mixed `skill`/`prompt` tiles with classifier badges and prompt read-only guidance
- **Resource Browser**: Dynamic grouped view for legacy and additive resource groups (`prompts`, `imported`)
- **Tabbed Interface**: Switch between skill content and resources

Access the web UI at `http://localhost:8080` (or your configured port).

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### UI Regression Tests (Playwright)

```bash
npm install
npx playwright install chromium
npm run test:playwright
```

### Running

```bash
make run
```

### Docker Build

```bash
make docker-build
```

## License

MIT
