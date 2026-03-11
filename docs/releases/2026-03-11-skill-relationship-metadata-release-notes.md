# Release Notes: Skill Relationship Metadata

**Release Date:** 2026-03-11
**Implementation Plan:** [ADR-008 Skill-to-Rule and Skill-to-Prompt Relationship Metadata](/home/jeff/skillserver/docs/implementation-plans/skill-rule-and-prompt-relationship-metadata/skill-rule-and-prompt-relationship-metadata-implementation-plan.md)
**Related ADRs:** [ADR-008: Skill-to-Rule and Skill-to-Prompt Relationship Metadata](/home/jeff/skillserver/docs/adrs/008-skill-rule-and-prompt-relationship-metadata.md)

## Summary

This release adds first-class relationship metadata so skills can declare one related prompt and many related rules, while prompt and rule views expose reverse-related skills across REST metadata, MCP read surfaces, and the Web UI metadata workflow.

## Shipped Contract Changes

- REST metadata reads now include an additive `relationships` object on:
  - `GET /api/catalog/:id/metadata`
  - `GET /api/catalog/metadata?item_id=...`
- REST relationship writes are now available on:
  - `PATCH /api/catalog/:id/relationships`
- MCP now exposes read-only relationship lookup through:
  - `get_catalog_item_relationships`
- Relationship payload semantics are normalized across REST and MCP:
  - `prompt` is populated only for `skill` items
  - `rules` is populated only for `skill` items
  - `skills` is populated only for reverse prompt/rule views
- Effective reads suppress deleted or missing related endpoints.
- Catalog list/search and tile rendering remain relationship-light.

## Compatibility Notes

- Relationship metadata is additive and metadata-only; it does not change source-content mutability.
- REST relationship surfaces are canonical-only.
- MCP `get_catalog_item_relationships` accepts bare skill IDs only for `skill` items; prompt and rule IDs remain canonical-only.
- Prompt and rule relationship views are read-only in v1.
- No MCP relationship write tool is registered in v1.
- No tile-level relationship rendering is included in v1.

## Operator Guidance

- Treat [WP-008 completion summary](/home/jeff/skillserver/docs/implementation-plans/skill-rule-and-prompt-relationship-metadata/work-packages/completion-summaries/WP-008-completion-summary.md) as the release gate evidence for ADR-008.
- Run the exact WP-008 command matrix before promotion and before rollback closeout.
- Keep ADR-004 persistence runtime enabled; ADR-008 adds no separate runtime flag.
- Share the v1 limits explicitly with MCP client owners and UI users:
  - GUI and REST writes are skill-owned only
  - MCP remains read-only
  - catalog tiles remain unchanged

## Verification Gate

Use this WP-008 command matrix before promotion:

```bash
go test ./pkg/persistence -run 'TestCatalogSkillRuleRelationshipRepository|TestCatalogSkillPromptRelationshipRepository|TestCatalogRelationshipRepositories' -count=1
go test ./pkg/domain -run 'TestCatalogRelationshipService' -count=1
go test ./pkg/web -run 'TestCatalogRelationshipMetadataEndpoints|TestCatalogMetadataEndpoints' -count=1
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1
npx playwright test tests/playwright/wp007-ui-relationship-metadata-editor.spec.ts --project=chromium
npx playwright test tests/playwright/wp008-ui.spec.ts --project=chromium
```

Required focus areas:
- REST and MCP relationship projection parity
- skill-only write authority
- deleted/missing endpoint suppression
- list/search compatibility
- UI metadata overlay compatibility and relationship editor behavior

## Rollback Summary

- Preferred rollback: redeploy the last pre-ADR-008 build while leaving SQLite persistence data intact.
- Do not perform destructive schema rollback for ADR-008.
- If a broader persistence-backed metadata fallback is acceptable, disable ADR-004 persistence mode with `SKILLSERVER_PERSISTENCE_DATA=false` or `--persistence-data=false`.

Detailed runbook:
- [Skill Relationship Metadata Rollout and Rollback Runbook](/home/jeff/skillserver/docs/operations/skill-relationship-metadata-rollout-rollback.md)
