# Work Packages: ADR-005 Domain/Subdomain/Tag Taxonomy for Catalog Items

## Overview
This index tracks execution packages for implementing ADR-005.

## Packages
1. [WP-001: Taxonomy Schema v2 and Indexes](./WP-001-taxonomy-schema-v2-and-indexes.md)
2. [WP-002: Taxonomy Persistence Repositories and Row Models](./WP-002-taxonomy-persistence-repositories-and-row-models.md)
3. [WP-003: Taxonomy Registry Service and Validation Rules](./WP-003-taxonomy-registry-service-and-validation-rules.md)
4. [WP-004: Catalog Item Taxonomy Assignment and Effective Projection](./WP-004-catalog-item-taxonomy-assignment-and-effective-projection.md)
5. [WP-005: Taxonomy Backfill and Legacy Label Compatibility](./WP-005-taxonomy-backfill-and-legacy-label-compatibility.md)
6. [WP-006: Taxonomy Registry REST Endpoints](./WP-006-taxonomy-registry-rest-endpoints.md)
7. [WP-007: Catalog Item Taxonomy REST and Filtered Search](./WP-007-catalog-item-taxonomy-rest-and-filtered-search.md)
8. [WP-008: MCP Taxonomy Read Tools and Filter Contracts](./WP-008-mcp-taxonomy-read-tools-and-filter-contracts.md)
9. [WP-009: MCP Taxonomy Write Tools and Runtime Gating](./WP-009-mcp-taxonomy-write-tools-and-runtime-gating.md)
10. [WP-010: Web UI Taxonomy Management and Item Classification UX](./WP-010-web-ui-taxonomy-management-and-item-classification-ux.md)
11. [WP-011: Taxonomy Integration and Regression Test Matrix](./WP-011-taxonomy-integration-and-regression-test-matrix.md)
12. [WP-012: Rollout, Migration, and Operations Documentation](./WP-012-rollout-migration-and-operations-documentation.md)

## Dependency Order
`WP-001 -> WP-002 -> WP-003 -> WP-004`

`WP-002 -> WP-005`

`WP-003 -> (WP-006 || WP-008)`

`WP-004 -> (WP-006 || WP-007 || WP-008)`

`WP-008 -> WP-009`

`(WP-006 || WP-007) -> WP-010`

`(WP-005 || WP-007 || WP-009 || WP-010) -> WP-011 -> WP-012`

## Completion Summaries
Add completion summaries under `./completion-summaries/` as each package is finished.
