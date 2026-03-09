# Work Packages: Catalog ID, Taxonomy Classification, and Materialization Ergonomics

## Overview
This index tracks the execution packages for the catalog contract and UX tightening work planned on 2026-03-09.

## Packages
1. [WP-001: Architecture Contract and Compatibility Matrix](./WP-001-architecture-contract-and-compatibility-matrix.md)
2. [WP-002: Catalog Reference Normalizer and Classification-State Domain Model](./WP-002-catalog-reference-normalizer-and-classification-state-domain-model.md)
3. [WP-003: Repository Pagination and Taxonomy Usage Query Support](./WP-003-repository-pagination-and-taxonomy-usage-query-support.md)
4. [WP-004: Partial and Batch Taxonomy Mutation Services](./WP-004-partial-and-batch-taxonomy-mutation-services.md)
5. [WP-005: REST Catalog and Taxonomy Contract Expansion](./WP-005-rest-catalog-and-taxonomy-contract-expansion.md)
6. [WP-006: MCP Contract Expansion and Export Ergonomics](./WP-006-mcp-contract-expansion-and-export-ergonomics.md)
7. [WP-007: Web UI Taxonomy Manager and Catalog Classification UX](./WP-007-web-ui-taxonomy-manager-and-catalog-classification-ux.md)
8. [WP-008: Regression Matrix and Compatibility Coverage](./WP-008-regression-matrix-and-compatibility-coverage.md)
9. [WP-009: Documentation, Examples, and Release Guidance](./WP-009-documentation-examples-and-release-guidance.md)

## Dependency Order
`WP-001 -> (WP-002 || WP-003)`

`(WP-002 || WP-003) -> WP-004`

`(WP-002 || WP-003 || WP-004) -> (WP-005 || WP-006)`

`WP-005 -> WP-007`

`(WP-005 || WP-006 || WP-007) -> WP-008 -> WP-009`

## Completion Summaries
Add completion summaries under `./completion-summaries/` as each package is finished.
