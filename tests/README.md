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

## WP-011 Taxonomy Integration and Regression

This matrix defines objective regression gates for ADR-005 taxonomy rollout readiness.

### Scope

- Migration/backfill behavior and legacy-label compatibility.
- Effective projection semantics for taxonomy references and filters.
- REST taxonomy registry CRUD + item taxonomy patch/get + list/search parity.
- MCP read/write registration matrix and taxonomy filter behavior.
- UI taxonomy manager, taxonomy assignment, and compatibility flows.

### CI-Compatible Commands

```bash
go test ./pkg/persistence -run 'TestRunMigrations_(UpgradeFromVersionOneToLatest_AppliesTaxonomySchema|CatalogTaxonomySchema_CascadeDeletesAndTagAssignmentQueriesRemainValid)' -count=1
go test ./cmd/skillserver -run 'TestCatalogPersistenceCoordinator_FullSyncAndRebuild_BackfillsLegacyLabelsIntoTaxonomyTags|TestMCPConfig_(Defaults|EnvOverrides|FlagPrecedence|InvalidEnableWritesBoolean)' -count=1
go test ./pkg/domain -run 'TestCatalogTaxonomy|TestCatalogTaxonomyLegacyLabelBackfillService|TestCatalogTaxonomyAssignmentService|TestCatalogEffectiveService_List_MergesTaxonomyReferencesAndAppliesTaxonomyFilters' -count=1
go test ./pkg/web -run 'TestCatalogTaxonomyRegistryEndpoints_|TestCatalogItemTaxonomyEndpoints_|TestCatalogEndpoints_TaxonomyFilters_' -count=1
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression|TestTaxonomyWriteTools_' -count=1
npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp008-ui.spec.ts tests/playwright/wp010-ui-taxonomy.spec.ts --project=chromium
```

### Rollout Gate Checklist

- [ ] Migration + backfill tests pass in a clean workspace.
- [ ] Effective service compatibility assertions pass (`labels` taxonomy-derived/fallback behavior).
- [ ] REST list/search taxonomy filters stay parity-consistent (`primary_domain_id`, `secondary_domain_id`, `subdomain_id`, `tag_ids`, `tag_match`).
- [ ] MCP write tools remain hidden by default and visible only when write gate is enabled.
- [ ] Playwright taxonomy and pre-taxonomy UI flows pass without regressions.

## WP-011 Integration Safety Regression Matrix (Rule Catalog Materialization)

This matrix validates ADR-007 integration safety and compatibility before rollout.

### Scope

- REST and MCP export/materialization regression coverage, including dry-run safety and path validation.
- Persistence migration + rule-row lifecycle regression checks.
- Legacy skill export/import compatibility checks.
- UI capability-gated materialization verification.

### CI-Compatible Commands

```bash
go test ./pkg/domain -run 'TestCatalogExportService_|TestCatalogMaterializationService_' -count=1
go test ./pkg/persistence -run 'TestRunMigrations_UpgradeFromPreRuleSchemaToLatest_PreservesRowsAndAllowsRuleClassifier|TestCatalogSourceRepository_UpsertAndList_WithRuleClassifier_RoundTripsAndFilters|TestCatalogSourceRepository_RuleRowLifecycle_SoftDeleteAndRestorePreservesClassifierFiltering' -count=1
go test ./pkg/web -run 'TestExportSkill_LegacyRoute_|TestExportCatalog_|TestMaterializeCatalog_' -count=1
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1
npx playwright test tests/playwright/wp010-ui-export-materialization.spec.ts --project=chromium
```

### Rollout Gate Checklist

- [ ] Dry-run materialization requests prove no filesystem writes (REST + MCP).
- [ ] Invalid destination paths and outside-root requests fail with explicit errors (REST + MCP).
- [ ] Legacy skill export archives remain import-compatible (local and repo-backed).
- [ ] Rule indexing/filtering behavior is verified for non-persistence and persistence-backed flows.
- [ ] UI write actions are capability-gated and dry-run-first before write execution.

### Coverage Mapping Notes

- Local skill scenario: `TestExportSkill_LegacyRoute_LocalSkill_PreservesDownloadHeaders`.
- Repo-backed skill scenario: `TestExportSkill_LegacyRoute_RepoBackedSkillWithSlash_SupportsEncodedPath`.
- Prompt + rule scenarios: `TestExportCatalog_DryRun_BatchMixedClassifiers_ReturnsManifest`, `TestMaterializeCatalog_DryRunBatch_ReturnsPlannedItemsWithoutWrites`, and MCP stdio regression subtests.
