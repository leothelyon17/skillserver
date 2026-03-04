# ADR-004: Persistent Catalog Storage and Metadata Overlays

## Metadata

| Field | Value |
|-------|-------|
| **Status** | Proposed |
| **Date** | 2026-03-04 |
| **Author(s)** | @jeff |
| **Reviewers** | TBD |
| **Work Package** | N/A |
| **Supersedes** | N/A |
| **Superseded By** | N/A |

## Summary

SkillServer currently depends on filesystem state (`SKILL.md`, resource files, `.git-repos.json`, Bleve index files). We need durable persistence for catalog items and user-managed metadata, including metadata changes on Git-imported read-only items, while keeping Git-imported content itself read-only. We will add an embedded SQLite persistence layer (mounted via Docker volume or Kubernetes PVC) that stores a synchronized catalog snapshot plus metadata overlays, with sync triggers on startup and manual Git resync.

## Context

### Problem Statement

Users need metadata durability across restarts and redeployments, especially for catalog items imported from Git repositories where content must remain read-only but metadata should be editable. The current model stores metadata for skills in `SKILL.md` frontmatter and does not provide a durable metadata overlay for read-only Git items. This blocks key use cases such as annotation, tagging, ownership, and classification updates for shared imported content.

### Current State

- Catalog items are discovered from filesystem content in [`pkg/domain/manager.go`](/home/jeff/skillserver/pkg/domain/manager.go) and [`pkg/domain/manager_catalog.go`](/home/jeff/skillserver/pkg/domain/manager_catalog.go).
- Search index is rebuilt from in-memory/discovered catalog data using Bleve in [`pkg/domain/search.go`](/home/jeff/skillserver/pkg/domain/search.go).
- Git repositories are synced to local filesystem paths and re-indexed on update in [`pkg/git/syncer.go`](/home/jeff/skillserver/pkg/git/syncer.go).
- Manual Git resync is available via `POST /api/git-repos/:id/sync` in [`pkg/web/handlers.go`](/home/jeff/skillserver/pkg/web/handlers.go).
- Read-only enforcement for Git-backed skill content exists in REST handlers, but metadata updates are currently tied to content updates for skills.

### Requirements

| Requirement | Priority | Description |
|-------------|----------|-------------|
| REQ-1 | Must Have | Persistence is explicitly enabled by environment variable `SKILLSERVER_PERSISTENCE_DATA=true`. |
| REQ-2 | Must Have | Persistence backend works on mounted Docker volumes and Kubernetes PVCs. |
| REQ-3 | Must Have | Git-imported skills/prompts are synchronized into the persistence store on startup and manual Git resync. |
| REQ-4 | Must Have | Git-imported content remains read-only; metadata for Git-imported catalog items is writable and durable. |
| REQ-5 | Must Have | Skills/prompts created manually or imported by file remain content-writable and metadata-writable. |
| REQ-6 | Should Have | Existing REST/MCP behavior remains backward-compatible for current content flows. |
| REQ-7 | Should Have | Search reflects effective metadata (including overlays) after sync/update. |

### Constraints

- **Budget**: No always-on external managed database requirement for default deployment.
- **Timeline**: Deliver incrementally without breaking existing file-based workflows.
- **Technical**: Must preserve existing Git sync model and read-only content protection semantics.
- **Compliance**: Persisted metadata must be auditable and recoverable; no silent loss on restart.
- **Team**: Keep operational complexity low for local/docker/k8s users.

## Decision Drivers

1. **Deterministic Durability**: Metadata must survive container restart and redeploy.
2. **Mutability Separation**: Content mutability and metadata mutability must be independently enforceable.
3. **Deployment Simplicity**: Must work equally with Docker volumes and PVC mounts.
4. **Compatibility**: Existing filesystem, import, and Git sync flows should remain valid.
5. **Operational Safety**: Startup and resync flows should self-heal database snapshots from filesystem/Git source content.

## Options Considered

### Option 1: Filesystem Sidecar Metadata Files

**Description**: Keep current model and add sidecar files (for example `.catalog-metadata.json`) near skill/prompt content.

**Pros**:
- Minimal code movement.
- No DB dependency.
- Easy manual inspection.

**Cons**:
- Hard to enforce transactional consistency and concurrent updates.
- Poor fit for rich queries/filtering at scale.
- Complex merge semantics for Git-imported read-only content plus writable metadata overlays.

**Estimated Effort**: M

**Cost Implications**: Low

---

### Option 2: Embedded SQLite Catalog Store with Overlay Metadata (Chosen)

**Description**: Add an embedded SQLite database in a persistence directory. Keep filesystem/Git as content source of truth, then synchronize discovered items into SQLite. Store user-editable metadata overlays keyed by stable catalog item ID. Expose effective catalog records by merging source snapshot + metadata overlays.

**Pros**:
- Durable single-file storage works naturally on Docker volume and PVC mounts.
- Clean separation between content mutability (`content_writable`) and metadata mutability (`metadata_writable`).
- Transactional updates and deterministic startup/resync reconciliation.
- Supports future metadata growth (tags, owner, rating, notes, custom JSON).

**Cons**:
- Adds DB lifecycle and migration responsibility.
- Requires sync/reconciliation logic and schema versioning.
- Introduces stateful behavior when persistence is enabled.

**Estimated Effort**: L

**Cost Implications**: Low

---

### Option 3: External Postgres (or Managed DB) as System of Record

**Description**: Move catalog and metadata into a networked database service.

**Pros**:
- Scales better for multi-instance and HA deployments.
- Strong centralized data management.

**Cons**:
- Significant operational burden for local/self-hosted users.
- Introduces networking, credentials, and provisioning requirements.
- Overkill for current single-instance deployment model.

**Estimated Effort**: XL

**Cost Implications**: Medium-High

## Decision

### Chosen Option

**We will implement Option 2: Embedded SQLite catalog store with metadata overlays.**

### Rationale

Option 2 is the only option that satisfies all must-have requirements without introducing managed infrastructure. It provides durable persistence through a mounted path while preserving filesystem/Git content flows. It also directly models the required rule: Git-imported content is immutable, but metadata remains writable.

### Decision Matrix

| Criteria | Weight | Option 1 | Option 2 | Option 3 |
|----------|--------|----------|----------|----------|
| Meets Git read-only + metadata writable requirement | 5 | 2 | 5 | 5 |
| Docker volume/PVC operational simplicity | 5 | 4 | 5 | 2 |
| Durability and transactional safety | 4 | 2 | 5 | 5 |
| Backward compatibility with current code paths | 4 | 4 | 4 | 2 |
| Implementation/operational complexity | 3 | 4 | 3 | 1 |
| **Weighted Total** |  | **56** | **85** | **60** |

## Consequences

### Positive

- Metadata updates become durable for all catalog items, including Git-imported skill/prompt items.
- Startup and Git resync flows continuously converge DB state to source content + overlay metadata.
- Docker and Kubernetes deployments get identical persistence behavior by mounting the same path semantics.
- Clear mutability contract becomes explicit in data model and API responses.

### Negative

- Additional persistent-state failure modes (DB locked/corrupt/migration issues).
- Schema migration discipline is now required.
- Slightly higher startup latency due to initial reconciliation.

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Persistence enabled but mount not durable/misconfigured | Med | High | Fail fast on startup when `SKILLSERVER_PERSISTENCE_DATA=true` and configured persistence path is missing/unwritable; log explicit mount guidance. |
| Item identity drift breaks metadata overlays | Med | Med | Use stable deterministic IDs from existing catalog ID builders and enforce canonical key tests. |
| Resync overwrites user metadata overlays | Low | High | Keep source snapshot and metadata overlays in separate tables; source sync updates must never overwrite overlay columns. |
| Search index divergence from DB effective state | Med | Med | Rebuild/reload search from effective DB view after each sync and metadata mutation. |

## Technical Details

### Architecture

```text
Filesystem + Git repos (existing)
        |
        | discovery + canonical IDs
        v
Catalog Sync Engine (startup, periodic git sync, manual /sync)
        |
        +--> sqlite: catalog_source_items (content snapshot, source flags)
        +--> sqlite: catalog_metadata_overlays (user metadata edits)
        |
        v
Effective Catalog View (SQL join + precedence rules)
        |
        +--> REST/MCP list/search responses
        +--> Bleve index rebuild (from effective view)
```

### Data Model

`catalog_source_items` (synchronized, mostly read-only):
- `item_id` (PK; existing deterministic catalog ID)
- `classifier` (`skill` | `prompt`)
- `source_type` (`git` | `local` | `file_import`)
- `source_repo`
- `parent_skill_id`
- `resource_path`
- `name`
- `description`
- `content`
- `content_hash`
- `content_writable` (false for Git-derived items)
- `metadata_writable` (true for all items)
- `last_synced_at`
- `deleted_at` (nullable tombstone)

`catalog_metadata_overlays` (user writable):
- `item_id` (PK/FK)
- `display_name_override` (nullable)
- `description_override` (nullable)
- `custom_metadata_json` (JSON object)
- `labels_json` (JSON array)
- `updated_at`
- `updated_by` (optional future field)

`system_state`:
- schema version
- sync cursor/checkpoints (optional future use)

### Effective Record Resolution

1. Start from `catalog_source_items`.
2. Left-join overlay by `item_id`.
3. Resolve fields with overlay precedence:
   - `effective_name = COALESCE(display_name_override, source.name)`
   - `effective_description = COALESCE(description_override, source.description)`
   - `effective_custom_metadata = overlay.custom_metadata_json OR {}`
4. Return `content_writable` and `metadata_writable` explicitly in APIs.

### Sync Lifecycle

- **Startup (when persistence enabled)**:
  1. Validate persistence path and open DB.
  2. Run schema migrations.
  3. Discover catalog from filesystem/Git as today.
  4. Upsert into `catalog_source_items`.
  5. Soft-delete missing source rows (`deleted_at` set) while preserving overlays.
  6. Rebuild search index from effective records.

- **Manual Git Resync (`POST /api/git-repos/:id/sync`)**:
  1. Existing Git pull executes.
  2. Trigger catalog sync for affected repo items.
  3. Rebuild index from effective records.

- **Container Restart**:
  - Same startup flow ensures Git-imported content is re-synced, metadata overlays preserved.

### API Changes

Additive changes:
- `PATCH /api/catalog/:id/metadata` (new): update metadata overlays for any catalog item.
- `GET /api/catalog/:id/metadata` (optional helper): fetch overlay + effective metadata.
- Include `content_writable` and `metadata_writable` in catalog responses.

Compatibility:
- Existing skill/resource content write guards remain unchanged for Git-backed content.
- Existing create/import/manual flows continue to write filesystem content; sync engine persists snapshots.

### Configuration

```bash
# Enable persistence mode
SKILLSERVER_PERSISTENCE_DATA=true

# Required when persistence enabled; mount this path via Docker volume or PVC
SKILLSERVER_PERSISTENCE_DIR=/data

# SQLite file path (optional override)
SKILLSERVER_PERSISTENCE_DB_PATH=/data/skillserver.db
```

Docker example:
```bash
docker run -p 8080:8080 \
  -e SKILLSERVER_PERSISTENCE_DATA=true \
  -e SKILLSERVER_PERSISTENCE_DIR=/data \
  -v skillserver-data:/data \
  -v skillserver-skills:/app/skills \
  ghcr.io/mudler/skillserver:latest
```

Kubernetes example:
```yaml
volumeMounts:
  - name: skillserver-data
    mountPath: /data
volumes:
  - name: skillserver-data
    persistentVolumeClaim:
      claimName: skillserver-data-pvc
env:
  - name: SKILLSERVER_PERSISTENCE_DATA
    value: "true"
  - name: SKILLSERVER_PERSISTENCE_DIR
    value: /data
```

## Security Considerations

- No change to Git content read-only enforcement.
- Metadata overlay writes should validate size and key format to prevent abuse.
- SQLite file permissions should be restricted to process user.
- Backups of persistent volume become sensitive if metadata may include private annotations.

## Performance Considerations

- Startup sync adds bounded I/O proportional to catalog size.
- Runtime reads become simpler (DB-backed effective view), potentially reducing repeated filesystem parsing.
- Search rebuild remains bounded by catalog size; optimize with incremental updates in future iteration.

## Implementation Plan

### Phase 1: Persistence Foundation

- Add persistence runtime config parsing and startup validation.
- Add SQLite initialization + migration runner.
- Add persistence interfaces/repositories with tests.

### Phase 2: Catalog Sync + Overlay Model

- Implement catalog source snapshot upsert + soft-delete reconciliation.
- Add metadata overlay CRUD.
- Add effective catalog read model and API DTO extensions.

### Phase 3: Integration Wiring

- Wire startup and manual Git sync flows to persistence sync engine.
- Wire search rebuild from effective catalog view.
- Add regression tests for restart and manual resync behavior.

### Phase 4: UX + Docs

- Add UI metadata editing for all catalog item types.
- Keep Git content edit buttons disabled; metadata edit allowed.
- Update README and operations docs for Docker volume/PVC setup and recovery.

### Rollback Plan

1. Set `SKILLSERVER_PERSISTENCE_DATA=false`.
2. Restart service to return to filesystem-only mode.
3. Keep DB file for later re-enable; no content migration rollback required because filesystem remains canonical.

## Testing Strategy

- Unit tests:
  - ID stability and overlay merge precedence.
  - write-guard matrix (`content_writable`, `metadata_writable`) by source type.
  - migration/version upgrade behavior.
- Integration tests:
  - startup sync persists Git and local items.
  - manual `sync` updates Git content snapshot without dropping overlays.
  - restart preserves overlays and re-applies them to effective responses.
- End-to-end tests:
  - Git item content edit blocked, metadata edit succeeds and persists.
  - manual/file-imported item content + metadata both editable.

## Related Decisions

- [ADR-002: Dynamic Imported Resource Discovery and Prompt Support](./002-dynamic-resource-and-prompt-discovery.md)
- [ADR-003: Unified Skill/Prompt Catalog Classification for Git Imports](./003-unified-skill-prompt-catalog-classification.md)

## References

- Existing startup wiring: [`cmd/skillserver/main.go`](/home/jeff/skillserver/cmd/skillserver/main.go)
- Existing Git sync callbacks: [`pkg/git/syncer.go`](/home/jeff/skillserver/pkg/git/syncer.go)
- Existing catalog builders: [`pkg/domain/manager_catalog.go`](/home/jeff/skillserver/pkg/domain/manager_catalog.go)
- Existing index rebuild path: [`pkg/domain/search.go`](/home/jeff/skillserver/pkg/domain/search.go)
