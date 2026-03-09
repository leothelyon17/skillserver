# WP-004 Completion Summary

## Metadata

- **Work Package:** WP-004
- **Title:** Partial and Batch Taxonomy Mutation Services
- **Completed Date:** 2026-03-09
- **Status:** Complete
- **Estimated Effort:** 5 hours
- **Actual Effort:** 3 hours

## Deliverables Completed

- [x] Extended `CatalogItemTaxonomyAssignmentPatchInput` in
  `pkg/domain/catalog_taxonomy_assignment_service.go` with additive tag
  mutation fields:
  - `add_tag_ids`
  - `remove_tag_ids`
  - `clear_tags`
- [x] Refactored single-item taxonomy patching onto a shared plan/apply path so
  dry-run planning and write execution reuse the same mutation logic.
- [x] Added batch taxonomy mutation request/result types and deterministic item
  statuses in `pkg/domain/catalog_taxonomy_assignment_service.go`:
  - `planned`
  - `updated`
  - `unchanged`
  - `invalid`
  - `not_found`
- [x] Added reusable taxonomy usage/preflight service types and implementation
  in `pkg/domain/catalog_taxonomy_usage_service.go` with:
  - domain usage summaries
  - subdomain usage summaries
  - tag usage summaries
  - `blocking_reason=in_use` when assignments would block deletion
- [x] Added domain-service tests for additive tag mutations, batch dry-run/apply
  behavior, per-item failures, global request-shape validation, and usage
  summary responses in:
  - `pkg/domain/catalog_taxonomy_assignment_service_test.go`
  - `pkg/domain/catalog_taxonomy_usage_service_test.go`

## Acceptance Criteria Verification

- [x] Single-item taxonomy patch supports add/remove/clear without requiring the
  caller to fetch current tags first.
- [x] Batch patch supports `dry_run` and deterministic per-item result rows.
- [x] Usage/preflight summaries expose assignment counts, impacted item counts,
  preview item IDs, and delete-blocking reason data.
- [x] Batch request-shape validation rejects malformed duplicate-item requests
  before any writes occur.

## Test Evidence

### Commands Run

```bash
go test ./pkg/domain -run 'TestCatalogTaxonomyAssignmentService_|TestCatalogTaxonomyUsageService_' -count=1
go test ./pkg/domain -count=1
go test ./pkg/web ./pkg/mcp -count=1
```

### Results

- `go test ./pkg/domain -run 'TestCatalogTaxonomyAssignmentService_|TestCatalogTaxonomyUsageService_' -count=1`: pass
- `go test ./pkg/domain -count=1`: pass
- `go test ./pkg/web ./pkg/mcp -count=1`: pass

## Variance from Estimates

- Completed under the estimate because the existing WP-002 and WP-003 changes
  already provided the normalization, classification-state, and repository usage
  primitives needed for the service-layer implementation.

## Risks / Issues Encountered

- Batch mutation planning validates the entire request shape up front and then
  executes valid item plans in deterministic request order. Cross-item apply is
  not yet wrapped in a single transaction because the current domain service
  contract does not own a shared transaction boundary across both assignment
  repositories.

## Follow-up Items

1. WP-005 should expose the new additive single-item fields, batch mutation
   route, and usage/preflight summaries over REST.
2. WP-006 should register MCP batch patch and usage/preflight tools against the
   new domain service contracts.
3. WP-007 should consume `blocking_reason`, preview item IDs, and batch tag
   mutation semantics in the taxonomy manager UI.
