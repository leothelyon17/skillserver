# WP-008 Completion Summary

## Metadata

- **Work Package:** WP-008
- **Title:** Regression Matrix and Compatibility Coverage
- **Completed Date:** 2026-03-09
- **Status:** Complete
- **Estimated Effort:** 4 hours
- **Actual Effort:** 2 hours

## Deliverables Completed

- [x] Strengthened REST regression coverage in
  `pkg/web/handlers_catalog_item_taxonomy_test.go` for:
  - bare skill ID compatibility on single-item taxonomy PATCH/GET
  - canonicalized batch-taxonomy results for bare skill inputs
  - list/search parity for supported classification-state filters
- [x] Strengthened MCP stdio regression coverage in
  `pkg/mcp/server_stdio_regression_test.go` for:
  - bare skill ID compatibility on taxonomy read/write tools
  - dry-run and apply coverage for batch taxonomy mutation
  - end-to-end subdomain and tag usage/preflight tool execution
  - missing-field filter execution for metadata-first catalog tools
- [x] Extended Playwright coverage in
  `tests/playwright/wp007-ui-catalog-classification.spec.ts` for:
  - classification filter reset behavior
  - domain delete-preflight coverage
  - usage/preflight fetch failure handling
- [x] Published a dedicated WP-008 rollout matrix in `tests/README.md` with:
  - scope mapping
  - CI-friendly commands
  - rollout checklist
  - coverage notes
- [x] Marked `WP-008` complete in the work-package metadata.

## Acceptance Criteria Verification

- [x] The regression matrix maps each requested improvement area to automated
  coverage.
- [x] Compatibility behavior for bare skill IDs is proven for the supported
  REST and MCP taxonomy surfaces instead of being assumed.
- [x] Domain, repository, REST, MCP, and UI suites pass for the covered
  contracts.
- [x] UI additions stayed state-based and deterministic; no timing-sensitive
  assertions were introduced.
- [x] Export archive-root behavior remains explicitly covered by the matrix and
  the existing export regression suites.

## Test Evidence

### Commands Run

```bash
git diff --check
go test ./pkg/web -run 'TestCatalogItemTaxonomyEndpoints_BareSkillIDCompatibility_ForSingleAndBatchPatch|TestCatalogEndpoints_ClassificationStateFilters_WorkForListAndSearch' -count=1
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1
npx playwright test tests/playwright/wp007-ui-catalog-classification.spec.ts --project=chromium
go test ./pkg/domain -run 'TestCatalogTaxonomyAssignmentService_|TestCatalogMetadataService_|TestCatalogTaxonomyUsageService_|TestCatalogExportService_' -count=1
go test ./pkg/persistence -run 'TestCatalogSourceRepository_|TestCatalogItemTaxonomyAssignmentRepository_' -count=1
```

### Results

- `git diff --check`: pass
- `go test ./pkg/web -run 'TestCatalogItemTaxonomyEndpoints_BareSkillIDCompatibility_ForSingleAndBatchPatch|TestCatalogEndpoints_ClassificationStateFilters_WorkForListAndSearch' -count=1`: pass
- `go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1`: pass
- `npx playwright test tests/playwright/wp007-ui-catalog-classification.spec.ts --project=chromium`: pass
- `go test ./pkg/domain -run 'TestCatalogTaxonomyAssignmentService_|TestCatalogMetadataService_|TestCatalogTaxonomyUsageService_|TestCatalogExportService_' -count=1`: pass
- `go test ./pkg/persistence -run 'TestCatalogSourceRepository_|TestCatalogItemTaxonomyAssignmentRepository_' -count=1`: pass

## Variance from Estimates

- Completed under estimate because most domain and repository coverage was
  already in place from WP-002 through WP-007, so the remaining work stayed
  focused on transport-level compatibility, UI regression gaps, and the rollout
  matrix itself.

## Risks / Issues Encountered

- The catalog metadata REST surface still expects canonical item IDs; this WP
  intentionally did not widen that contract because the work package scope was
  regression coverage rather than new product behavior.
- Playwright runs against shared fixture state, so the new domain delete
  preflight test had to assign a matching subdomain for the temporary domain to
  avoid carrying forward an incompatible primary-subdomain selection.

## Follow-up Items

1. WP-009 should reference the new `WP-008` section in `tests/README.md` as the
   release-readiness gate for catalog ID and taxonomy ergonomics.
2. If the metadata REST surface should support bare skill IDs in the future,
   that should land as an explicit contract change with its own tests rather
   than being inferred from the taxonomy compatibility promise.
