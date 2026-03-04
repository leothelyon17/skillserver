## Work Package WP-007 Completion Summary

**Status:** ✅ Complete

**Work Package:** `WP-007-startup-and-manual-git-resync-persistence-wiring`
**Completed Date:** 2026-03-04

### Deliverables Completed

- [x] Added persistence runtime bootstrap + coordinator in `cmd/skillserver/persistence_catalog_runtime.go`:
  - SQLite bootstrap + migrations on startup in persistence mode.
  - Full sync path (`FullSyncAndRebuild`) for startup and periodic full updates.
  - Repo-scoped sync path (`RepoSyncAndRebuild`) for manual git resync.
  - Search rebuild from effective projection after sync operations.
- [x] Wired startup/runtime integration in `cmd/skillserver/main.go`:
  - Persistence runtime initialization before serving traffic.
  - Git sync callback switched to persistence-aware full sync when enabled.
  - Persistence shutdown cleanup on process exit.
  - Persistence-mode startup failure on git sync callback/start failure.
- [x] Wired manual git resync in web layer:
  - Added manual repo sync hook plumbing in `pkg/web/server.go`.
  - `POST /api/git-repos/:id/sync` now performs repo-scoped persistence sync when hook is configured.
  - Disabled-mode compatibility preserved via direct index rebuild fallback.
- [x] Updated git syncer manual behavior in `pkg/git/syncer.go`:
  - Manual `SyncRepo` now performs VCS synchronization only.
  - Post-sync index/snapshot handling is explicitly controlled by caller.
- [x] Added integration-style tests:
  - `cmd/skillserver/persistence_catalog_runtime_test.go` (startup/full sync + repo-scoped sync overlay preservation + effective index assertions).
  - `pkg/web/handlers_git_sync_test.go` (manual sync hook path + disabled-mode fallback path).

### Acceptance Criteria Verification

- [x] Startup performs migration + full sync before serving requests in persistence mode.
- [x] Manual git resync updates only affected repo source rows.
- [x] Search index rebuilds from effective data after sync operations.
- [x] Non-persistence mode manual sync behavior remains compatible.

### Test Evidence

- `go test ./cmd/skillserver ./pkg/web -run 'TestCatalogPersistenceCoordinator_|TestSyncGitRepo_' -count=1` ✅
- `go test ./cmd/skillserver ./pkg/domain ./pkg/git ./pkg/web -count=1` ✅
- `go test ./... -count=1` ✅

### Files Added

- `cmd/skillserver/persistence_catalog_runtime.go`
- `cmd/skillserver/persistence_catalog_runtime_test.go`
- `pkg/web/handlers_git_sync_test.go`
- `docs/implementation-plans/persistent-catalog-storage-and-metadata-overlays/work-packages/completion-summaries/WP-007-completion-summary.md`

### Files Updated

- `cmd/skillserver/main.go`
- `pkg/domain/manager.go`
- `pkg/git/syncer.go`
- `pkg/web/server.go`
- `pkg/web/handlers.go`

### Notes

- Repo-scoped manual sync uses repository config identity (`repo.Name`, with URL-derived fallback) to avoid broad full-sync writes.
- Effective search rebuild behavior is now centralized in the persistence coordinator.
