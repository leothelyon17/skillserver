# Release Notes: Catalog ID, Taxonomy, and Materialization Ergonomics

**Release Date:** 2026-03-09
**Implementation Plan:** [Catalog ID, Taxonomy Classification, and Materialization Ergonomics](/home/jeff/skillserver/docs/implementation-plans/catalog-id-taxonomy-and-materialization-ergonomics/catalog-id-taxonomy-and-materialization-ergonomics-implementation-plan.md)
**Related ADRs:** [ADR-005: Domain/Subdomain/Tag Taxonomy for Catalog Items](/home/jeff/skillserver/docs/adrs/005-domain-subdomain-tag-taxonomy-for-catalog-items.md), [ADR-007: Rule Catalog Objects and MCP Project Materialization](/home/jeff/skillserver/docs/adrs/007-rule-catalog-objects-and-mcp-project-materialization.md)

## Summary

This release tightens the catalog contract across REST, MCP, and the web UI so
clients can rely on one canonical item-identity model, explicit
classification-state fields, additive taxonomy mutation primitives, direct
usage/preflight reads, and lighter default catalog/export payloads.

## Shipped Contract Changes

- MCP `list_skills` and `search_skills` now emit canonical skill item IDs in
  `id` and always populate `name`.
- REST and MCP taxonomy reads now expose:
  - `has_assignment`
  - `is_fully_classified`
  - `missing_fields`
- REST and MCP list/search add effective-state filters:
  - `unclassified`
  - `missing_primary_domain`
  - `missing_tags`
- Taxonomy mutation surfaces now support:
  - additive single-item tag patching with `add_tag_ids`, `remove_tag_ids`,
    and `clear_tags`
  - batch dry-run/apply mutation via:
    - REST `PATCH /api/catalog/taxonomy/batch`
    - MCP `patch_catalog_items_taxonomy`
- Taxonomy usage/preflight reads are now first-class:
  - REST `GET /api/catalog/taxonomy/{domains|subdomains|tags}/:id/usage`
  - MCP `get_taxonomy_domain_usage`, `get_taxonomy_subdomain_usage`,
    `get_taxonomy_tag_usage`
- Catalog list/search defaults are now metadata-first:
  - `include_content=false` unless explicitly requested
  - deterministic pagination with `limit`, `cursor`, `next_cursor`, `has_more`
- MCP export ergonomics now support:
  - `archive_root_mode=flat|materialized`
  - `include_archive_base64=true|false`

## Compatibility Notes

- Bare skill ID fallback remains intentionally bounded:
  - accepted on MCP skill reads/resources
  - accepted on REST/MCP taxonomy item operations for `skill` items
  - accepted on export/materialization requests for `skill` items
- Prompt and rule item references remain canonical-only on every public
  surface.
- REST metadata endpoints remain canonical-only:
  - `/api/catalog/:id/metadata`
  - `/api/catalog/metadata?item_id=...`
- REST list/search still preserve the legacy array response shape when callers
  omit both `limit` and `cursor`; paginated REST calls and all MCP catalog
  calls use structured envelopes.

## Operator Guidance

- Treat the `WP-008` matrix in [`tests/README.md`](/home/jeff/skillserver/tests/README.md) as the release gate for this rollout.
- Ensure rollout notes are shared with MCP client owners because canonical skill
  item IDs are now emitted by `list_skills` and `search_skills`.
- Highlight the default payload-size changes during rollout:
  - catalog list/search omit `content` unless requested
  - MCP export omits inline archive bytes unless requested
  - MCP export defaults to flattened archive roots
  - REST export remains manifest/download oriented and does not inline archive
    bytes

## Verification Gate

Run the `WP-008` command matrix in [`tests/README.md`](/home/jeff/skillserver/tests/README.md) before promotion or rollback closeout.

Required focus areas:
- bare skill ID compatibility on the promised REST/MCP taxonomy surfaces
- explicit classification-state fields in metadata-first responses
- dry-run versus apply taxonomy batch behavior
- usage/preflight summary availability
- flattened archive-root behavior and export safety

## Rollback Summary

- Taxonomy fallback remains configuration-first:
  - disable MCP taxonomy writes with `SKILLSERVER_MCP_ENABLE_WRITES=false`
  - if broader taxonomy issues appear, fall back to non-persistence mode with
    `SKILLSERVER_PERSISTENCE_DATA=false`
- Materialization fallback remains configuration-first:
  - disable materialization writes with
    `SKILLSERVER_MCP_ENABLE_MATERIALIZATION=false`
  - optionally disable rule indexing with
    `SKILLSERVER_CATALOG_ENABLE_RULES=false`

Detailed runbooks:
- [Domain Taxonomy Rollout Runbook](/home/jeff/skillserver/docs/operations/domain-taxonomy-rollout-rollback.md)
- [Rule Catalog and Materialization Rollout Runbook](/home/jeff/skillserver/docs/operations/rule-catalog-materialization-rollout-rollback.md)
