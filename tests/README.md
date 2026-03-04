# Test Execution Matrix

## WP-009 Persistence Integration and Regression

This matrix defines deterministic commands for validating ADR-004 persistence behavior.

### Scope

- Persistence startup sync, restart durability, and repo-scoped sync behavior.
- Effective projection and mutability contract (`content_writable`, `metadata_writable`, `read_only`).
- Metadata API regression behavior and non-persistence compatibility paths.
- UI metadata editing/regression flows under persistence mode.

### CI-Compatible Commands

```bash
go test ./cmd/skillserver -run 'TestCatalogPersistenceCoordinator_|TestValidatePersistenceStartupConfig_' -count=1
go test ./pkg/domain -run 'TestCatalogEffectiveService_|TestCatalogSyncService_' -count=1
go test ./pkg/web -run 'TestCatalogMetadataEndpoints_|TestSyncGitRepo_' -count=1
npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp008-ui.spec.ts --project=chromium
```

### Notes

- Playwright runs against `tests/playwright/run-skillserver-fixture.sh`, which starts `skillserver` with persistence mode enabled (`--persistence-data`).
- These commands are suitable for local and CI runs because they use deterministic fixtures and avoid wall-clock timing assertions.
