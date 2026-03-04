## WP-009: Persistence Integration and Regression Test Matrix

### Metadata

```yaml
WP_ID: WP-009
Title: Persistence Integration and Regression Test Matrix
Domain: Quality Engineering
Priority: High
Estimated_Effort: 6 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-04
Started_Date: 2026-03-04
Completed_Date: 2026-03-04
```

---

### Description

**Context:**
ADR-004 introduces stateful behavior and new mutability rules, so regression protection must cover startup, sync, API, search, and UI paths.

**Scope:**
- Implement end-to-end and integration tests for persistence-enabled flows.
- Validate non-persistence compatibility behavior.
- Validate effective metadata propagation to search and API outputs.

Excluded:
- Authoring new feature logic not required for tests.
- Operational docs (WP-010).

**Success Criteria:**
- [x] Restart durability and manual git-resync overlay preservation are verified.
- [x] Write-guard matrix is validated for each source type.
- [x] Search results include overlay-resolved metadata.

---

### Technical Requirements

**Input Contracts:**
- API and UI behavior from WP-006, WP-007, and WP-008.

**Output Contracts:**
- Integration test suites under `pkg/...` and/or `tests/...`.
- Test fixture setup for local and Git-backed sample content.

**Integration Points:**
- WP-010 documentation references this test matrix as rollout gate.

---

### Deliverables

**Code Deliverables:**
- [x] Add integration tests for startup sync and restart persistence.
- [x] Add integration tests for manual git sync without overlay loss.
- [x] Add API contract regression tests for mutability fields and metadata endpoints.
- [x] Add UI/E2E tests for metadata editing matrix.
- [x] Add compatibility tests for persistence disabled mode.

**Test Deliverables:**
- [x] Test matrix document/checklist in package comments or `tests/README`.
- [x] Automated CI-compatible commands for persistence-mode test execution.

---

### Acceptance Criteria

**Functional:**
- [x] All ADR must-have behaviors are covered by automated tests.
- [x] Search and list responses reflect overlay updates after mutation/sync.
- [x] Existing non-persistence content flows remain green.

**Testing:**
- [x] New tests pass consistently in isolated environments.
- [x] Flaky scenarios are removed or stabilized with deterministic fixtures.

---

### ADR Requirement Coverage Matrix

| ADR-004 Requirement | Validation |
|---|---|
| REQ-1: Persistence mode is opt-in (`SKILLSERVER_PERSISTENCE_DATA=true`) | `cmd/skillserver/config_test.go`, `cmd/skillserver/persistence_runtime_test.go` |
| REQ-2: Persistence directory/DB path validation for durable mounts | `cmd/skillserver/persistence_runtime_test.go` |
| REQ-3: Startup and manual git sync persist source snapshots | `cmd/skillserver/persistence_catalog_runtime_test.go`, `pkg/web/handlers_git_sync_test.go` |
| REQ-4: Git content remains read-only; metadata remains writable/durable | `pkg/domain/catalog_effective_service_test.go`, `pkg/web/handlers_catalog_metadata_test.go`, `tests/playwright/wp008-ui.spec.ts` |
| REQ-5: Local/file-import content + metadata mutability preserved | `pkg/domain/catalog_effective_service_test.go`, `pkg/web/handlers_catalog_test.go`, `pkg/web/handlers_catalog_metadata_test.go` |
| REQ-6: Backward-compatible non-persistence behavior | `pkg/web/handlers_git_sync_test.go`, `cmd/skillserver/persistence_runtime_test.go`, `pkg/web/handlers_catalog_test.go` |
| REQ-7: Search/index reflects overlay-resolved effective metadata | `cmd/skillserver/persistence_catalog_runtime_test.go`, `pkg/web/handlers_catalog_metadata_test.go` |

---

### Execution Evidence (2026-03-04)

**Runtime + Persistence Integration**
- `cmd/skillserver/persistence_catalog_runtime_test.go`
  - full-sync indexing from effective projection
  - repo-scoped manual sync preserving non-target overlays
  - restart durability on the same SQLite path (`..._PreservesOverlayAcrossRuntimeRestart`)

**Domain Mutability and Effective Projection**
- `pkg/domain/catalog_effective_service_test.go`
- `pkg/domain/catalog_sync_service_test.go`

**API Contract and Compatibility Regression**
- `pkg/web/handlers_catalog_metadata_test.go`
- `pkg/web/handlers_git_sync_test.go`
- `pkg/web/handlers_catalog_test.go`

**UI/E2E Regression**
- `tests/playwright/wp008-ui.spec.ts`
- `tests/playwright/wp005-ui-catalog.spec.ts`

**CI-Compatible Command Matrix**
- `go test ./cmd/skillserver -run 'TestCatalogPersistenceCoordinator_|TestValidatePersistenceStartupConfig_' -count=1`
- `go test ./pkg/domain -run 'TestCatalogEffectiveService_|TestCatalogSyncService_' -count=1`
- `go test ./pkg/web -run 'TestCatalogMetadataEndpoints_|TestSyncGitRepo_' -count=1`
- `npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp008-ui.spec.ts --project=chromium`

See `tests/README.md` for consolidated execution commands and scope.

---

### Dependencies

**Blocked By:**
- WP-006
- WP-007
- WP-008

**Blocks:**
- WP-010

**Parallel Execution:**
- Can run in parallel with: None
- Cannot run in parallel with: WP-010

---

### Risks

**Risk 1: Integration tests become brittle due to timing-sensitive sync flows**
- Probability: Medium
- Impact: Medium
- Mitigation: Use deterministic fixtures and explicit synchronization points.

**Risk 2: Coverage misses source-type edge cases**
- Probability: Medium
- Impact: High
- Mitigation: Explicit matrix coverage for `git`, `local`, and `file_import`.
