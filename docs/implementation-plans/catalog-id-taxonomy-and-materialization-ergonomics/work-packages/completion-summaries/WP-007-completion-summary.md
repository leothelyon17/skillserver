# WP-007 Completion Summary

## Metadata

- **Work Package:** WP-007
- **Title:** Web UI Taxonomy Manager and Catalog Classification UX
- **Completed Date:** 2026-03-09
- **Status:** Complete
- **Estimated Effort:** 5 hours
- **Actual Effort:** 3 hours

## Deliverables Completed

- [x] Added explicit classification-state UX in
  `pkg/web/ui/index.html` and `pkg/web/ui/style.css`:
  - unclassified badges
  - partially-classified badges
  - gap chips for missing taxonomy fields
- [x] Added taxonomy filter controls in `pkg/web/ui/index.html` for:
  - `unclassified`
  - `missing_primary_domain`
  - `missing_tags`
- [x] Added taxonomy delete preflight UX in `pkg/web/ui/index.html` so domain,
  subdomain, and tag delete confirmations show:
  - assignment counts
  - impacted catalog item counts
  - preview item IDs
  - blocking state when references still exist
- [x] Updated metadata editing in `pkg/web/ui/index.html` to compute taxonomy
  diffs from the original modal state and use additive tag mutation
  (`add_tag_ids`, `remove_tag_ids`, `clear_tags`) instead of fragile full-tag
  replacement.
- [x] Updated list/search loading in `pkg/web/ui/index.html` to support
  metadata-first paginated envelopes and on-demand prompt/rule preview reads
  when catalog responses omit inline `content`.
- [x] Added dedicated Playwright coverage in
  `tests/playwright/wp007-ui-catalog-classification.spec.ts` for:
  - classification badges and filters
  - taxonomy delete preflight and blocked deletion
  - paginated metadata-first responses plus preview fallback reads
- [x] Updated regression support files:
  - `tests/playwright/wp010-ui-taxonomy.spec.ts`
  - `tests/README.md`

## Acceptance Criteria Verification

- [x] Unclassified and partially classified items are visible in the catalog
  grid without opening the metadata modal.
- [x] Usage counts and impacted-item previews are visible before taxonomy
  deletes, and in-use deletes are blocked in the UI.
- [x] The UI no longer relies on inline `content` in list/search payloads for
  prompt and rule previews.
- [x] Pagination is visible, deterministic, and covered by browser regression
  tests.
- [x] Existing taxonomy-manager CRUD and metadata-overlay flows remain intact.

## Test Evidence

### Commands Run

```bash
git diff --check
go test ./pkg/web -run 'TestCatalogTaxonomyRegistryEndpoints_|TestCatalogItemTaxonomyEndpoints_|TestCatalogEndpoints_TaxonomyFilters_' -count=1
npx playwright test tests/playwright/wp007-ui-catalog-classification.spec.ts --project=chromium
npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp007-ui-catalog-classification.spec.ts tests/playwright/wp008-ui.spec.ts tests/playwright/wp010-ui-taxonomy.spec.ts --project=chromium
```

### Results

- `git diff --check`: pass
- `go test ./pkg/web -run 'TestCatalogTaxonomyRegistryEndpoints_|TestCatalogItemTaxonomyEndpoints_|TestCatalogEndpoints_TaxonomyFilters_' -count=1`: pass
- `npx playwright test tests/playwright/wp007-ui-catalog-classification.spec.ts --project=chromium`: pass
- `npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp007-ui-catalog-classification.spec.ts tests/playwright/wp008-ui.spec.ts tests/playwright/wp010-ui-taxonomy.spec.ts --project=chromium`: pass

## Variance from Estimates

- Completed under the estimate because the REST and MCP contracts from WP-005
  and WP-006 were already in place, so the remaining work stayed concentrated
  in the Alpine UI and regression layer.

## Risks / Issues Encountered

- The metadata editor now intentionally skips taxonomy PATCH calls when the
  effective taxonomy selection is unchanged. Existing regression coverage had to
  be updated so it performs a real taxonomy change before waiting on that
  request.
- Taxonomy delete confirmation labels are now object-specific (`Delete Tag`,
  `Delete Domain`, `Delete Subdomain`) instead of the old generic `Confirm`,
  so taxonomy regression helpers were updated accordingly.

## Follow-up Items

1. WP-009 should document the new classification chips, filter controls,
   delete-preflight behavior, and metadata-first preview fallback.
2. Future UI work should keep the metadata modal’s taxonomy-diff behavior in
   mind so no-op saves do not reintroduce redundant taxonomy PATCH requests.
