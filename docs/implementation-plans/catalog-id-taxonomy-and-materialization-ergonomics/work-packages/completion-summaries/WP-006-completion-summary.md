# WP-006 Completion Summary

## Metadata

- **Work Package:** WP-006
- **Title:** MCP Contract Expansion and Export Ergonomics
- **Completed Date:** 2026-03-09
- **Status:** Complete
- **Estimated Effort:** 5 hours
- **Actual Effort:** 3 hours

## Deliverables Completed

- [x] Updated `list_skills` and `search_skills` in `pkg/mcp/tools.go` to emit
  canonical skill item IDs and populated display names.
- [x] Added compatibility normalization so `read_skill`,
  `list_skill_resources`, `read_skill_resource`, and `get_skill_resource_info`
  accept both bare skill IDs and canonical `skill:<id>` references.
- [x] Extended MCP catalog list/search contracts in `pkg/mcp/tools.go` with:
  - `include_content`
  - `limit`
  - `cursor`
  - `unclassified`
  - `missing_primary_domain`
  - `missing_tags`
- [x] Switched MCP catalog list/search responses to metadata-first defaults with
  explicit pagination fields:
  - `items`
  - `next_cursor`
  - `has_more`
- [x] Added explicit classification-state fields to MCP catalog and taxonomy
  item outputs:
  - `has_assignment`
  - `is_fully_classified`
  - `missing_fields`
- [x] Extended single-item taxonomy patch inputs with additive tag mutation
  fields:
  - `add_tag_ids`
  - `remove_tag_ids`
  - `clear_tags`
- [x] Added a batch taxonomy patch MCP tool in `pkg/mcp/server.go` and
  `pkg/mcp/tools.go`:
  - `patch_catalog_items_taxonomy`
- [x] Added delete-preflight taxonomy usage MCP read tools:
  - `get_taxonomy_domain_usage`
  - `get_taxonomy_subdomain_usage`
  - `get_taxonomy_tag_usage`
- [x] Extended `export_catalog_items` in
  `pkg/mcp/tools_export_materialization.go` with:
  - `archive_root_mode`
  - `include_archive_base64`
- [x] Changed MCP export defaults so non-dry-run responses return manifest plus
  download metadata, while inline archive bytes are returned only when
  explicitly requested.
- [x] Wired the persisted taxonomy usage service into MCP runtime bootstrap in
  `cmd/skillserver/persistence_catalog_runtime.go` and
  `cmd/skillserver/main.go`.
- [x] Expanded MCP regression coverage in
  `pkg/mcp/server_stdio_regression_test.go` and
  `pkg/mcp/tools_taxonomy_write_test.go` for:
  - canonical skill ID compatibility
  - metadata-first list/search behavior
  - pagination and classification filters
  - additive and batch taxonomy mutation
  - taxonomy usage/preflight reads
  - export archive-root and archive-byte options

## Acceptance Criteria Verification

- [x] MCP callers no longer need to guess how to convert skill IDs into
  taxonomy-safe item IDs.
- [x] `list_skills` and `search_skills` no longer return blank names.
- [x] Skill-related MCP tools accept both bare and canonical skill IDs.
- [x] `list_catalog` and `search_catalog` omit `content` unless requested.
- [x] MCP list/search payloads are lightweight by default.
- [x] `export_catalog_items` can return planning metadata without inline archive
  bytes.
- [x] Existing MCP materialization capability gating remains unchanged.

## Test Evidence

### Commands Run

```bash
gofmt -w pkg/mcp/tools.go pkg/mcp/server.go pkg/mcp/tools_export_materialization.go pkg/mcp/tools_taxonomy_write_test.go pkg/mcp/server_stdio_regression_test.go cmd/skillserver/persistence_catalog_runtime.go cmd/skillserver/main.go
go test ./pkg/mcp -count=1
go test ./cmd/skillserver -count=1
```

### Results

- `go test ./pkg/mcp -count=1`: pass
- `go test ./cmd/skillserver -count=1`: pass

## Variance from Estimates

- Completed under estimate because WP-002 through WP-004 had already delivered
  the catalog normalization, pagination, batch mutation, and usage services
  needed by the MCP layer. The remaining work stayed concentrated in MCP
  handlers, export shaping, runtime wiring, and regression tests.

## Risks / Issues Encountered

- `list_skills` and `search_skills` now return canonical IDs by default, so
  MCP compatibility depends on the read-side normalization remaining in place
  for existing callers that still send bare skill IDs.
- Flat export mode intentionally strips synthetic `skills/`, `prompts/`, and
  `rules/` wrappers from archive paths. Duplicate flattened paths now fail
  explicitly instead of silently colliding.

## Follow-up Items

1. WP-008 should preserve MCP compatibility coverage for canonical and bare
   skill ID inputs alongside the new pagination and taxonomy-usage contracts.
2. WP-009 should document the new MCP defaults, especially metadata-first
   catalog responses and `include_archive_base64` opt-in behavior.
