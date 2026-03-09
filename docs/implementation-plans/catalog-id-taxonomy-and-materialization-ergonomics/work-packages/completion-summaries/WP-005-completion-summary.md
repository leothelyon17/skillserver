# WP-005 Completion Summary

## Metadata

- **Work Package:** WP-005
- **Title:** REST Catalog and Taxonomy Contract Expansion
- **Completed Date:** 2026-03-09
- **Status:** Complete
- **Estimated Effort:** 5 hours
- **Actual Effort:** 4 hours

## Deliverables Completed

- [x] Expanded REST catalog list/search decoding in `pkg/web/handlers.go` with:
  - `include_content`
  - `limit`
  - `cursor`
  - `unclassified`
  - `missing_primary_domain`
  - `missing_tags`
- [x] Switched REST catalog list/search responses to metadata-first by default
  while preserving the legacy array shape when callers omit `limit` and
  `cursor`.
- [x] Added explicit classification-state fields to:
  - catalog list/search item DTOs
  - effective metadata DTOs returned by `GET /api/catalog/:id/metadata`
- [x] Extended single-item taxonomy patch payloads with additive tag mutation
  fields:
  - `add_tag_ids`
  - `remove_tag_ids`
  - `clear_tags`
- [x] Added a batch taxonomy mutation route in `pkg/web/server.go` and
  `pkg/web/handlers.go`:
  - `PATCH /api/catalog/taxonomy/batch`
- [x] Added delete-preflight usage routes for taxonomy objects:
  - `GET /api/catalog/taxonomy/domains/:id/usage`
  - `GET /api/catalog/taxonomy/subdomains/:id/usage`
  - `GET /api/catalog/taxonomy/tags/:id/usage`
- [x] Upgraded taxonomy conflict responses to keep the legacy `error` field
  while also returning structured conflict details when available.
- [x] Added REST coverage for:
  - metadata-first list/search responses
  - paginated envelope behavior
  - classification-state filters
  - additive single-item tag mutation
  - batch taxonomy dry-run/apply behavior
  - usage/preflight endpoints
  - structured conflict payloads

## Acceptance Criteria Verification

- [x] REST responses are metadata-first by default and paginate
  deterministically.
- [x] REST filters can target unclassified and partially classified items.
- [x] Batch and single-item taxonomy mutations are both additive and backward
  compatible.
- [x] Delete-preflight usage data is accessible without reading opaque error
  strings.

## Test Evidence

### Commands Run

```bash
gofmt -w pkg/web/server.go pkg/web/handlers.go pkg/web/handlers_catalog_test.go pkg/web/handlers_catalog_metadata_test.go pkg/web/handlers_catalog_item_taxonomy_test.go pkg/web/handlers_catalog_taxonomy_test.go
go test ./pkg/web -count=1
```

### Results

- `go test ./pkg/web -count=1`: pass

## Variance from Estimates

- Completed slightly under estimate because WP-002 through WP-004 had already
  delivered the normalization, batch mutation, and usage services needed by the
  REST layer. The remaining work stayed concentrated in `pkg/web`.

## Risks / Issues Encountered

- The rollout keeps the legacy array shape only when callers omit both `limit`
  and `cursor`. Callers opting into pagination now receive the new `{items,
  next_cursor, has_more}` envelope.
- Metadata-first list/search responses intentionally omit `content` unless
  callers opt in with `include_content=true`; search still matches content in
  the backend before response shaping.

## Follow-up Items

1. WP-006 should mirror the new REST list/search, batch taxonomy mutation, and
   usage/preflight contracts in the MCP transport.
2. WP-007 should consume the new classification-state fields, usage summaries,
   and structured conflict payloads in the taxonomy manager UI.
3. WP-008 should preserve the legacy array-shape compatibility checks alongside
   the new paginated envelope and structured conflict assertions.
