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
| `SKILLSERVER_GIT_REPOS` | `GIT_REPOS` | (empty) | Comma-separated Git repository URLs |
| `SKILLSERVER_GIT_ENABLE_STORED_CREDENTIALS` | (none) | `false` | Enable encrypted stored credentials for private Git repositories (requires persistence + master key) |
| `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY` | (none) | (empty) | Inline master key for stored Git credentials (mutually exclusive with `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE`) |
| `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY_FILE` | (none) | (empty) | File path containing the stored-credential master key (mutually exclusive with `SKILLSERVER_GIT_CREDENTIAL_MASTER_KEY`) |
| `SKILLSERVER_ENABLE_LOGGING` | (none) | `false` | Enable logging to stderr (default: false to avoid interfering with MCP stdio) |
| `SKILLSERVER_MCP_TRANSPORT` | (none) | `both` | MCP transport mode: `stdio`, `http`, or `both` |
| `SKILLSERVER_MCP_HTTP_PATH` | (none) | `/mcp` | Absolute HTTP route path for MCP Streamable HTTP |
| `SKILLSERVER_MCP_SESSION_TIMEOUT` | (none) | `30m` | Session timeout for MCP HTTP mode (`time.ParseDuration` format) |
| `SKILLSERVER_MCP_STATELESS` | (none) | `false` | Enable stateless MCP HTTP mode |
| `SKILLSERVER_MCP_ENABLE_WRITES` | (none) | `false` | Enable MCP taxonomy write tools (kept disabled by default) |
| `SKILLSERVER_MCP_ENABLE_EVENT_STORE` | (none) | `true` | Enable in-memory MCP event store for replay support |
| `SKILLSERVER_MCP_EVENT_STORE_MAX_BYTES` | (none) | `10485760` | Max bytes for MCP in-memory event store (10 MiB) |
| `SKILLSERVER_CATALOG_ENABLE_PROMPTS` | (none) | `true` | Enable prompt catalog classification/indexing in unified catalog APIs/tools |
| `SKILLSERVER_CATALOG_PROMPT_DIRS` | (none) | `agent,agents,prompt,prompts` | Comma-separated directory names used for prompt catalog detection |
| `SKILLSERVER_PERSISTENCE_DATA` | (none) | `false` | Enable SQLite-backed persistence for catalog source snapshots + metadata overlays |
| `SKILLSERVER_PERSISTENCE_DIR` | (none) | (empty) | Writable persistence directory (required when `SKILLSERVER_PERSISTENCE_DATA=true`) |
| `SKILLSERVER_PERSISTENCE_DB_PATH` | (none) | `<SKILLSERVER_PERSISTENCE_DIR>/skillserver.db` | Optional SQLite DB file path (absolute path or relative to persistence dir) |
| `SKILLSERVER_ENABLE_IMPORT_DISCOVERY` | (none) | `true` | Enable imported resource discovery and `imports/...` virtual read paths |

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | `./skills` | Directory to store skills (overrides `SKILLSERVER_DIR` or `SKILLS_DIR`) |
| `--port` | `8080` | Port for the web server (overrides `SKILLSERVER_PORT` or `PORT`) |
| `--git-repos` | (empty) | Comma-separated list of Git repository URLs (overrides `SKILLSERVER_GIT_REPOS` or `GIT_REPOS`) |
| `--git-enable-stored-credentials` | `false` | Enable encrypted stored credentials for private Git repositories |
| `--git-credential-master-key` | (empty) | Inline master key for stored credentials (mutually exclusive with `--git-credential-master-key-file`) |
| `--git-credential-master-key-file` | (empty) | File path containing stored-credential master key (mutually exclusive with `--git-credential-master-key`) |
| `--enable-logging` | `false` | Enable logging to stderr (overrides `SKILLSERVER_ENABLE_LOGGING`) |
| `--mcp-transport` | `both` | MCP transport mode: `stdio`, `http`, or `both` |
| `--mcp-http-path` | `/mcp` | Absolute HTTP route path for MCP Streamable HTTP |
| `--mcp-session-timeout` | `30m` | Session timeout for MCP HTTP mode |
| `--mcp-stateless` | `false` | Enable stateless MCP HTTP mode |
| `--mcp-enable-writes` | `false` | Enable MCP taxonomy write tools (kept disabled by default) |
| `--mcp-enable-event-store` | `true` | Enable in-memory MCP event store |
| `--mcp-event-store-max-bytes` | `10485760` | Max bytes for in-memory MCP event store |
| `--catalog-enable-prompts` | `true` | Enable prompt catalog classification/indexing |
| `--catalog-prompt-dirs` | `agent,agents,prompt,prompts` | Comma-separated directory names used for prompt catalog detection |
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
# Using environment variable
export SKILLSERVER_GIT_REPOS="https://github.com/user/repo1.git,https://github.com/user/repo2.git"
./skillserver

# Using command-line flag
./skillserver --git-repos "https://github.com/user/repo1.git,https://github.com/user/repo2.git"
```

Note: there is no specific layout that the repository needs to follow. The only requirements is that in every skill you have a `SKILL.md` file, and that gets scanned automatically.

See [here](https://github.com/anthropics/skills) for an example repository.

### Docker Usage

```bash
# Using environment variables
docker run -p 8080:8080 \
  -e SKILLSERVER_DIR=/app/skills \
  -e SKILLSERVER_PORT=8080 \
  -e SKILLSERVER_GIT_REPOS="https://github.com/user/repo.git" \
  -v $(pwd)/skills:/app/skills \
  ghcr.io/mudler/skillserver:latest

# Using command-line flags
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
  -e SKILLSERVER_GIT_REPOS="git@github.com:acme/private-skills.git" \
  -e REPO_ACME_PAT="***" \
  ghcr.io/mudler/skillserver:latest
```

Use API `auth.source=env` with `token_ref=REPO_ACME_PAT`, or `auth.source=file` with file paths under `/run/secrets/git`.

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
      # Optional: Git repositories to sync
      # SKILLSERVER_GIT_REPOS: "https://github.com/user/repo.git"
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
          "env": {
            "SKILLSERVER_DIR": "/app/skills",
            "SKILLSERVER_PORT": "9090",
            "SKILLSERVER_GIT_REPOS": "https://github.com/user/repo.git"
          },
          "args": [
            "run", "-i", "--rm",
            "-v", "/host/path/to/skills:/app/skills",
            "-e", "SKILLSERVER_DIR",
            "-e", "SKILLSERVER_PORT",
            "-e", "SKILLSERVER_GIT_REPOS",
            "ghcr.io/mudler/skillserver:latest"
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
        "SKILLSERVER_PORT": "9090",
        "SKILLSERVER_GIT_REPOS": "https://github.com/user/repo.git"
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

#### Runtime Capabilities (ADR-006 support)
- `GET /api/runtime/capabilities` - Return runtime capability gates (for example `git.stored_credentials_enabled`)

#### Catalog (ADR-003, additive)
- `GET /api/catalog` - List unified catalog items (`skill` + `prompt`) with fields `id`, `classifier`, `name`, `description`, `content`, `parent_skill_id`, `resource_path`, `custom_metadata`, `labels`, `content_writable`, `metadata_writable`, `read_only`
- `GET /api/catalog/search?q=query&classifier=skill|prompt` - Search unified catalog items with optional classifier filter
- Optional taxonomy filters for both list/search:
  - `primary_domain_id`
  - `secondary_domain_id`
  - `subdomain_id` (matches primary or secondary subdomain)
  - `tag_ids` (comma-separated IDs)
  - `tag_match=any|all` (defaults to `any`)
- `classifier` is case-insensitive at input and normalized to `skill` or `prompt` in responses
- Invalid classifier values return `400` (`invalid catalog classifier ...`)
- Empty or missing `q` for `/api/catalog/search` returns `400` (`query parameter 'q' is required`)
- `GET /api/catalog/:id/metadata` - Return source + overlay + effective metadata projections for one catalog item
- `PATCH /api/catalog/:id/metadata` - Update metadata overlays for one catalog item (`display_name`, `description`, `labels`, `custom_metadata`, optional `updated_by`)

#### Taxonomy (ADR-005, additive; persistence mode required)
- `GET /api/catalog/:id/taxonomy` - Get taxonomy assignment metadata for one catalog item
- `PATCH /api/catalog/:id/taxonomy` - Patch taxonomy assignment metadata for one catalog item
- `GET /api/catalog/taxonomy/domains` - List taxonomy domains (`domain_id`, `domain_ids`, `key`, `keys`, `active` filters)
- `POST /api/catalog/taxonomy/domains` - Create taxonomy domain
- `PATCH /api/catalog/taxonomy/domains/:id` - Update taxonomy domain
- `DELETE /api/catalog/taxonomy/domains/:id` - Delete taxonomy domain
- `GET /api/catalog/taxonomy/subdomains` - List taxonomy subdomains (`subdomain_id`, `subdomain_ids`, `domain_id`, `domain_ids`, `key`, `keys`, `active` filters)
- `POST /api/catalog/taxonomy/subdomains` - Create taxonomy subdomain
- `PATCH /api/catalog/taxonomy/subdomains/:id` - Update taxonomy subdomain
- `DELETE /api/catalog/taxonomy/subdomains/:id` - Delete taxonomy subdomain
- `GET /api/catalog/taxonomy/tags` - List taxonomy tags (`tag_id`, `tag_ids`, `key`, `keys`, `active` filters)
- `POST /api/catalog/taxonomy/tags` - Create taxonomy tag
- `PATCH /api/catalog/taxonomy/tags/:id` - Update taxonomy tag
- `DELETE /api/catalog/taxonomy/tags/:id` - Delete taxonomy tag
- Taxonomy endpoints return `503` when persistence runtime is disabled/unavailable.

#### Resources
- `GET /api/skills/:name/resources` - List resources with legacy buckets (`scripts`, `references`, `assets`) plus additive groups (`prompts`, `imported`, `groups`) when present; each resource includes `origin` and `writable`
- `GET /api/skills/:name/resources/*` - Get/download a resource file
- `POST /api/skills/:name/resources` - Upload/create a direct resource (multipart/form-data or JSON); imported `imports/...` targets are blocked
- `PUT /api/skills/:name/resources/*` - Update a direct resource file; imported `imports/...` targets are blocked
- `DELETE /api/skills/:name/resources/*` - Delete a direct resource; imported `imports/...` targets are blocked

### MCP Tools

#### Skills
- `list_skills` - List all available skills (returns skill IDs for use with read_skill)
- `read_skill` - Read the full content of a skill by its ID
- `search_skills` - Search for skills by query string

#### Catalog (ADR-003, additive)
- `list_catalog` - List unified catalog items with optional `classifier` filter (`skill` or `prompt`) and optional taxonomy filters (`primary_domain_id`, `secondary_domain_id`, `subdomain_id`, `tag_ids`, `tag_match`)
- `search_catalog` - Search unified catalog items by `query`, with optional `classifier` + taxonomy filters
- Taxonomy read tools (always registered):
  - `list_taxonomy_domains`
  - `list_taxonomy_subdomains`
  - `list_taxonomy_tags`
  - `get_catalog_item_taxonomy`
- Taxonomy write tools (registered only when `--mcp-enable-writes=true` or `SKILLSERVER_MCP_ENABLE_WRITES=true`):
  - `create_taxonomy_domain`, `update_taxonomy_domain`, `delete_taxonomy_domain`
  - `create_taxonomy_subdomain`, `update_taxonomy_subdomain`, `delete_taxonomy_subdomain`
  - `create_taxonomy_tag`, `update_taxonomy_tag`, `delete_taxonomy_tag`
  - `patch_catalog_item_taxonomy`
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
