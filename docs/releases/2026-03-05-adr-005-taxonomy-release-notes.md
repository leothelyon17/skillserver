# Release Notes: ADR-005 Domain/Subdomain/Tag Taxonomy

**Release Date:** 2026-03-05
**ADR:** [ADR-005: Domain/Subdomain/Tag Taxonomy for Catalog Items](/home/jeff/skillserver/docs/adrs/005-domain-subdomain-tag-taxonomy-for-catalog-items.md)

## Summary
This release adds first-class taxonomy classification for unified catalog items (`skill` and `prompt`) with persistent domain, subdomain, and tag objects, additive REST/MCP contracts, and MCP write-gating defaults.

## Added
- Additive REST taxonomy endpoints:
  - `GET/PATCH /api/catalog/:id/taxonomy`
  - `GET/POST/PATCH/DELETE /api/catalog/taxonomy/domains`
  - `GET/POST/PATCH/DELETE /api/catalog/taxonomy/subdomains`
  - `GET/POST/PATCH/DELETE /api/catalog/taxonomy/tags`
- Additive taxonomy filters on `GET /api/catalog` and `GET /api/catalog/search`:
  - `primary_domain_id`, `secondary_domain_id`, `subdomain_id`, `tag_ids`, `tag_match=any|all`
- Additive MCP taxonomy read tools (always available):
  - `list_taxonomy_domains`
  - `list_taxonomy_subdomains`
  - `list_taxonomy_tags`
  - `get_catalog_item_taxonomy`
- Additive MCP taxonomy write tools (gated):
  - `create_taxonomy_domain`, `update_taxonomy_domain`, `delete_taxonomy_domain`
  - `create_taxonomy_subdomain`, `update_taxonomy_subdomain`, `delete_taxonomy_subdomain`
  - `create_taxonomy_tag`, `update_taxonomy_tag`, `delete_taxonomy_tag`
  - `patch_catalog_item_taxonomy`
- New runtime controls:
  - `SKILLSERVER_MCP_ENABLE_WRITES` / `--mcp-enable-writes` (default `false`)

## Compatibility Statement
This change is backward-compatible and additive.
- Existing `/api/skills` and `/api/skills/search` contracts are preserved.
- Existing MCP skill/resource tools remain supported.
- Existing metadata overlay APIs remain supported.
- MCP taxonomy write tools stay disabled unless explicitly enabled.

## Rollback Summary
- Fast safety rollback: disable MCP taxonomy write tools with `SKILLSERVER_MCP_ENABLE_WRITES=false` (or `--mcp-enable-writes=false`) and redeploy.
- Broader fallback (if required): disable persistence mode with `SKILLSERVER_PERSISTENCE_DATA=false` (or `--persistence-data=false`) and redeploy.

Detailed rollout/rollback procedure: [Domain Taxonomy Rollout Runbook](/home/jeff/skillserver/docs/operations/domain-taxonomy-rollout-rollback.md)
