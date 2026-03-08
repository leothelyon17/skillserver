# Release Notes: ADR-007 Rule Catalog Objects and MCP Project Materialization

**Release Date:** 2026-03-08  
**ADR:** [ADR-007: Rule Catalog Objects and MCP Project Materialization](/home/jeff/skillserver/docs/adrs/007-rule-catalog-objects-and-mcp-project-materialization.md)

## Summary
This release adds first-class `rule` catalog objects and shared export/materialization flows across REST, MCP, and UI surfaces while keeping write-capable behavior explicitly capability-gated.

## Added
- Additive catalog classifier support for `rule` in unified list/search behavior.
- Additive REST export/materialization surfaces:
  - `POST /api/catalog/export`
  - `POST /api/catalog/materialize`
- Additive MCP export/materialization tools:
  - `export_catalog_items` (always registered)
  - `materialize_catalog_items` (registered only when materialization gate is enabled)
- Runtime controls for rollout gating:
  - `SKILLSERVER_CATALOG_ENABLE_RULES` / `--catalog-enable-rules`
  - `SKILLSERVER_CATALOG_RULE_DIRS` / `--catalog-rule-dirs`
  - `SKILLSERVER_CATALOG_RULE_FILENAMES` / `--catalog-rule-filenames`
  - `SKILLSERVER_MCP_ENABLE_MATERIALIZATION` / `--mcp-enable-materialization`
  - `SKILLSERVER_MCP_ALLOWED_DESTINATION_ROOTS` / `--mcp-allowed-destination-roots`

## Compatibility Statement
This change is backward-compatible and additive.
- Existing `/api/skills` and `/api/skills/search` contracts are preserved.
- Existing legacy export route (`GET /api/skills/export/*`) remains available and delegates to shared export logic.
- Existing MCP skill/resource tools remain supported.
- Materialization write behavior remains disabled by default until explicitly enabled.

## Migration and Rollback Implications
- No destructive schema rollback is required for ADR-007 rollback.
- Gate rollback order:
  1. Disable materialization writes (`SKILLSERVER_MCP_ENABLE_MATERIALIZATION=false`).
  2. Disable rule indexing if required (`SKILLSERVER_CATALOG_ENABLE_RULES=false`).
- Persistence data can remain intact for forward re-enable.

## Verification Gate
Rollout/rollback should be gated by the WP-011 command matrix in [`tests/README.md`](/home/jeff/skillserver/tests/README.md), plus evidence captured in [WP-011 completion summary](/home/jeff/skillserver/docs/implementation-plans/rule-catalog-objects-and-mcp-project-materialization/work-packages/completion-summaries/WP-011-completion-summary.md).

## Detailed Rollout Procedure
[Rule Catalog and Materialization Rollout Runbook](/home/jeff/skillserver/docs/operations/rule-catalog-materialization-rollout-rollback.md)
