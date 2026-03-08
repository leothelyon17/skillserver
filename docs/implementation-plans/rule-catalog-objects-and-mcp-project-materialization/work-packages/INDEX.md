# Work Packages: ADR-007 Rule Catalog Objects and MCP Project Materialization

## Overview
This index tracks execution packages for implementing ADR-007.

## Packages
1. [WP-001: Shared Catalog Export Service](./WP-001-shared-catalog-export-service.md)
2. [WP-002: Export REST Endpoint and Legacy Route Delegation](./WP-002-export-rest-route-delegation.md)
3. [WP-003: Rule Classifier and Install Metadata](./WP-003-rule-classifier-and-install-metadata.md)
4. [WP-004: Runtime Flags and Capability Gates](./WP-004-runtime-flags-and-capability-gates.md)
5. [WP-005: Rule Classifier Persistence Migration](./WP-005-rule-classifier-persistence-migration.md)
6. [WP-006: Rule Catalog Discovery, Search, and Sync](./WP-006-rule-catalog-discovery-search-and-sync.md)
7. [WP-007: Materialization Planner and Safe Writes](./WP-007-materialization-planner-and-safe-writes.md)
8. [WP-008: Catalog Materialization REST Endpoints](./WP-008-catalog-materialization-rest-endpoints.md)
9. [WP-009: MCP Export and Materialization Tools](./WP-009-mcp-export-materialization-tools.md)
10. [WP-010: UI Export and Materialization UX](./WP-010-ui-export-materialization-ux.md)
11. [WP-011: Integration, Safety, and Regression Matrix](./WP-011-integration-safety-regression-matrix.md)
12. [WP-012: Rollout, Rollback, and Release Guidance](./WP-012-rollout-rollback-release-guidance.md)

## Dependency Order
`WP-001 -> WP-002`

`WP-003 -> (WP-004 || WP-005) -> WP-006`

`(WP-001 || WP-003 || WP-004 || WP-006) -> WP-007`

`(WP-002 || WP-006 || WP-007) -> WP-008`

`(WP-004 || WP-006 || WP-007) -> WP-009`

`(WP-006 || WP-008) -> WP-010`

`(WP-005 || WP-008 || WP-009 || WP-010) -> WP-011 -> WP-012`

## Completion Summaries
Add completion summaries under `./completion-summaries/` as each package is finished.
