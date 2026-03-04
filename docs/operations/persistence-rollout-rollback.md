# Persistent Catalog Storage Rollout and Rollback Runbook

## Purpose
Deterministic rollout and rollback procedure for ADR-004 embedded SQLite persistence (`catalog_source_items` + `catalog_metadata_overlays`).

## References
- ADR: [ADR-004: Persistent Catalog Storage and Metadata Overlays](/home/jeff/skillserver/docs/adrs/004-persistent-catalog-storage-and-metadata-overlays.md)
- Runtime/API docs: [README.md](/home/jeff/skillserver/README.md)
- Validation evidence:
  - [WP-009 persistence integration and regression matrix](/home/jeff/skillserver/docs/implementation-plans/persistent-catalog-storage-and-metadata-overlays/work-packages/WP-009-persistence-integration-and-regression-test-matrix.md)
  - [WP-009 completion summary](/home/jeff/skillserver/docs/implementation-plans/persistent-catalog-storage-and-metadata-overlays/work-packages/completion-summaries/WP-009-completion-summary.md)
  - [tests/README.md command matrix](/home/jeff/skillserver/tests/README.md)

## Runtime Controls
- `SKILLSERVER_PERSISTENCE_DATA` / `--persistence-data` (default: `false`)
- `SKILLSERVER_PERSISTENCE_DIR` / `--persistence-dir` (required when persistence is enabled)
- `SKILLSERVER_PERSISTENCE_DB_PATH` / `--persistence-db-path` (optional; defaults to `<persistence-dir>/skillserver.db`)

Behavior notes:
- Persistence mode is opt-in.
- Startup fails fast when persistence is enabled and mount/path validation fails.
- Metadata overlay endpoints are available only when persistence runtime is active:
  - `GET /api/catalog/:id/metadata`
  - `PATCH /api/catalog/:id/metadata`

## Preconditions
- Rollout owner and rollback owner are assigned.
- Durable storage is provisioned (Docker named volume or Kubernetes PVC).
- Candidate build includes WP-009 test evidence.
- At least one catalog item exists for metadata persistence checks.
- Optional but recommended for full validation: at least one configured/enabled Git repository.
- Validation snippets below assume `jq` is installed.

## Pre-Deploy Validation Gates
Run these WP-009 commands on the candidate artifact before deploy:

```bash
go test ./cmd/skillserver -run 'TestCatalogPersistenceCoordinator_|TestValidatePersistenceStartupConfig_' -count=1
go test ./pkg/domain -run 'TestCatalogEffectiveService_|TestCatalogSyncService_' -count=1
go test ./pkg/web -run 'TestCatalogMetadataEndpoints_|TestSyncGitRepo_' -count=1
npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp008-ui.spec.ts --project=chromium
```

## Rollout Procedure
1. Provision or confirm durable writable storage at your target persistence path.
2. Deploy with persistence enabled and persistence directory configured.
3. Keep default DB path (`skillserver.db`) unless you need explicit pathing.
4. If using custom DB file path, ensure its parent directory already exists and is writable.

Example (containerized):

```bash
docker volume create skillserver-persistence

docker run -p 8080:8080 \
  -v $(pwd)/skills:/app/skills \
  -v skillserver-persistence:/var/lib/skillserver/persistence \
  -e SKILLSERVER_MCP_TRANSPORT=http \
  -e SKILLSERVER_PERSISTENCE_DATA=true \
  -e SKILLSERVER_PERSISTENCE_DIR=/var/lib/skillserver/persistence \
  ghcr.io/mudler/skillserver:latest
```

## Rollout Validation Checklist

### 1) Startup Checks
- [ ] Process starts without `Invalid persistence runtime configuration` errors.
- [ ] `GET /api/catalog` returns `200`.
- [ ] Catalog item payloads include mutability fields (`content_writable`, `metadata_writable`, `read_only`).

```bash
set -euo pipefail

BASE_URL="http://127.0.0.1:8080"

curl -sS "$BASE_URL/api/catalog" | jq '.' > /tmp/wp010-catalog.json
jq -e 'length >= 1' /tmp/wp010-catalog.json >/dev/null
jq -e '.[0] | has("content_writable") and has("metadata_writable") and has("read_only")' /tmp/wp010-catalog.json >/dev/null
```

### 2) Metadata Persistence Checks
- [ ] `PATCH /api/catalog/:id/metadata` persists overlay metadata.
- [ ] `GET /api/catalog/:id/metadata` shows merged effective values.
- [ ] After process restart on same persistence mount, overlay values remain.

```bash
set -euo pipefail

BASE_URL="http://127.0.0.1:8080"
ITEM_ID=$(curl -sS "$BASE_URL/api/catalog" | jq -r '.[0].id')
ITEM_ID_ESCAPED=$(jq -rn --arg v "$ITEM_ID" '$v|@uri')
ROLL_OUT_TS=$(date -u +%Y%m%dT%H%M%SZ)
DISPLAY_NAME="WP010 Persist Check ${ROLL_OUT_TS}"

curl -sS -X PATCH "$BASE_URL/api/catalog/${ITEM_ID_ESCAPED}/metadata" \
  -H "Content-Type: application/json" \
  --data "{\"display_name\":\"${DISPLAY_NAME}\",\"labels\":[\"wp010\",\"rollout-check\"],\"updated_by\":\"wp010-runbook\"}" \
  | jq '.' > /tmp/wp010-metadata-patch.json

curl -sS "$BASE_URL/api/catalog/${ITEM_ID_ESCAPED}/metadata" | jq '.' > /tmp/wp010-metadata-get.json
jq -e --arg expected "$DISPLAY_NAME" '.effective.name == $expected' /tmp/wp010-metadata-get.json >/dev/null
jq -e '.effective.metadata_writable == true' /tmp/wp010-metadata-get.json >/dev/null

# Restart service with the same persistence mount/path, then re-run GET validation.
```

### 3) Manual Git Resync Checks (if Git repos are configured)
- [ ] Overlay metadata on a Git-backed item survives `POST /api/git-repos/:id/sync`.
- [ ] Non-target repositories are not modified by a single repo sync.

```bash
set -euo pipefail

BASE_URL="http://127.0.0.1:8080"
REPO_ID=$(curl -sS "$BASE_URL/api/git-repos" | jq -r '.[] | select(.enabled == true) | .id' | head -n 1)

if [ -n "$REPO_ID" ]; then
  GIT_ITEM_ID=$(curl -sS "$BASE_URL/api/catalog" | jq -r '.[] | select(.read_only == true) | .id' | head -n 1)
  GIT_ITEM_ID_ESCAPED=$(jq -rn --arg v "$GIT_ITEM_ID" '$v|@uri')
  GIT_DISPLAY_NAME="WP010 Git Sync Check $(date -u +%Y%m%dT%H%M%SZ)"

  curl -sS -X PATCH "$BASE_URL/api/catalog/${GIT_ITEM_ID_ESCAPED}/metadata" \
    -H "Content-Type: application/json" \
    --data "{\"display_name\":\"${GIT_DISPLAY_NAME}\",\"updated_by\":\"wp010-runbook\"}" >/dev/null

  curl -sS -X POST "$BASE_URL/api/git-repos/${REPO_ID}/sync" | jq '.' > /tmp/wp010-git-sync.json

  curl -sS "$BASE_URL/api/catalog/${GIT_ITEM_ID_ESCAPED}/metadata" | jq '.' > /tmp/wp010-git-metadata.json
  jq -e --arg expected "$GIT_DISPLAY_NAME" '.effective.name == $expected' /tmp/wp010-git-metadata.json >/dev/null
fi
```

## Backup and Recovery Guidance (SQLite)

### Backup
Recommended: stop writes (or stop service) before file-copy backup.

```bash
set -euo pipefail

PERSIST_DIR="/var/lib/skillserver/persistence"
DB_PATH="$PERSIST_DIR/skillserver.db"
BACKUP_DIR="$PERSIST_DIR/backups"
BACKUP_PATH="$BACKUP_DIR/skillserver-$(date -u +%Y%m%dT%H%M%SZ).db"

mkdir -p "$BACKUP_DIR"
cp "$DB_PATH" "$BACKUP_PATH"
sha256sum "$DB_PATH" "$BACKUP_PATH"
```

Optional online SQLite backup (if `sqlite3` is available):

```bash
sqlite3 "$DB_PATH" ".backup '$BACKUP_PATH'"
```

### Restore
1. Stop SkillServer.
2. Restore backup file to the configured DB path.
3. Start SkillServer with persistence enabled and the same mount/path configuration.
4. Validate restored metadata using `GET /api/catalog/:id/metadata`.

```bash
set -euo pipefail

BACKUP_PATH="/var/lib/skillserver/persistence/backups/<selected-backup>.db"
DB_PATH="/var/lib/skillserver/persistence/skillserver.db"

cp "$BACKUP_PATH" "$DB_PATH"
```

## Rollback Triggers
Rollback to filesystem-only mode if any of these are observed:
- Persistent startup failures due to mount/path misconfiguration.
- Migration/bootstrap errors that block server startup.
- Persistent `database is locked` errors that prevent metadata updates.
- Metadata overlay persistence/regression checks fail.

## Rollback Procedure (Filesystem-Only Mode)
Execute in this order:

1. Disable persistence mode:

```bash
# Flag-based rollback
./skillserver --persistence-data=false

# Env-based rollback
export SKILLSERVER_PERSISTENCE_DATA=false
./skillserver
```

2. Keep persistence files in place for later analysis/recovery. No destructive data operations are required.

3. Validate rollback behavior:
- [ ] `GET /api/catalog` returns `200`.
- [ ] Metadata endpoints return `503` (`catalog metadata API is unavailable`) as expected in non-persistence mode.

```bash
set -euo pipefail

BASE_URL="http://127.0.0.1:8080"
ITEM_ID=$(curl -sS "$BASE_URL/api/catalog" | jq -r '.[0].id')
ITEM_ID_ESCAPED=$(jq -rn --arg v "$ITEM_ID" '$v|@uri')

CODE=$(curl -sS -o /tmp/wp010-rollback-metadata.json -w "%{http_code}" \
  "$BASE_URL/api/catalog/${ITEM_ID_ESCAPED}/metadata")
test "$CODE" = "503"
```

## Troubleshooting

### 1) SQLite database locked
Symptoms:
- Metadata update requests fail intermittently.
- Errors contain `database is locked`.

Likely causes:
- Multiple writers against the same DB file.
- Stale process retaining file handles.

Actions:
1. Ensure only one SkillServer process writes to the configured DB path.
2. Restart SkillServer to clear stale handles.
3. Check for other processes using the DB file (`lsof <db-path>` where available).
4. Keep persistence path on local/durable filesystem with normal file locking semantics.

### 2) Persistence mount missing or not writable
Symptoms:
- Startup exits with `Invalid persistence runtime configuration`.
- Error includes `does not exist` or `not writable` for `SKILLSERVER_PERSISTENCE_DIR`.

Actions:
1. Verify mount/PVC is attached at the configured path.
2. Verify directory exists and is writable by the container/process user.
3. Confirm `SKILLSERVER_PERSISTENCE_DIR` matches the mounted path exactly.
4. Re-run startup after fixing permissions/mount.

### 3) Migration/bootstrap failure
Symptoms:
- Startup exits with errors containing `run sqlite migrations`, `apply migration`, or SQLite pragma/open failures.

Actions:
1. Enable logging (`SKILLSERVER_ENABLE_LOGGING=true`) for full startup diagnostics.
2. Verify DB parent directory exists and is writable.
3. Validate configured DB path points to a file, not a directory.
4. Restore from last known-good backup if schema/bootstrap state is unrecoverable.

## Post-Rollout / Post-Rollback Closeout
- [ ] Record timestamp + operator + commands executed.
- [ ] Attach validation artifacts (`/tmp/wp010-*.json` outputs or equivalent) to release notes.
- [ ] Link final outcome in [WP-010 completion summary](/home/jeff/skillserver/docs/implementation-plans/persistent-catalog-storage-and-metadata-overlays/work-packages/completion-summaries/WP-010-completion-summary.md).
