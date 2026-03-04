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
| `SKILLSERVER_ENABLE_LOGGING` | (none) | `false` | Enable logging to stderr (default: false to avoid interfering with MCP stdio) |
| `SKILLSERVER_MCP_TRANSPORT` | (none) | `both` | MCP transport mode: `stdio`, `http`, or `both` |
| `SKILLSERVER_MCP_HTTP_PATH` | (none) | `/mcp` | Absolute HTTP route path for MCP Streamable HTTP |
| `SKILLSERVER_MCP_SESSION_TIMEOUT` | (none) | `30m` | Session timeout for MCP HTTP mode (`time.ParseDuration` format) |
| `SKILLSERVER_MCP_STATELESS` | (none) | `false` | Enable stateless MCP HTTP mode |
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
| `--enable-logging` | `false` | Enable logging to stderr (overrides `SKILLSERVER_ENABLE_LOGGING`) |
| `--mcp-transport` | `both` | MCP transport mode: `stdio`, `http`, or `both` |
| `--mcp-http-path` | `/mcp` | Absolute HTTP route path for MCP Streamable HTTP |
| `--mcp-session-timeout` | `30m` | Session timeout for MCP HTTP mode |
| `--mcp-stateless` | `false` | Enable stateless MCP HTTP mode |
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

# Roll back to filesystem-only mode (non-destructive)
./skillserver --persistence-data=false
# Or using environment variable
export SKILLSERVER_PERSISTENCE_DATA=false
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

#### Catalog (ADR-003, additive)
- `GET /api/catalog` - List unified catalog items (`skill` + `prompt`) with fields `id`, `classifier`, `name`, `description`, `content`, `parent_skill_id`, `resource_path`, `custom_metadata`, `labels`, `content_writable`, `metadata_writable`, `read_only`
- `GET /api/catalog/search?q=query&classifier=skill|prompt` - Search unified catalog items with optional classifier filter
- `classifier` is case-insensitive at input and normalized to `skill` or `prompt` in responses
- Invalid classifier values return `400` (`invalid catalog classifier ...`)
- Empty or missing `q` for `/api/catalog/search` returns `400` (`query parameter 'q' is required`)
- `GET /api/catalog/:id/metadata` - Return source + overlay + effective metadata projections for one catalog item
- `PATCH /api/catalog/:id/metadata` - Update metadata overlays for one catalog item (`display_name`, `description`, `labels`, `custom_metadata`, optional `updated_by`)

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
- `list_catalog` - List unified catalog items with optional `classifier` filter (`skill` or `prompt`)
- `search_catalog` - Search unified catalog items by `query`, with optional `classifier` filter (`skill` or `prompt`)
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
