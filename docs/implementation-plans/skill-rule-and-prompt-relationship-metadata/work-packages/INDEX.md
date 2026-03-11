# Work Packages: ADR-008 Skill-to-Rule and Skill-to-Prompt Relationship Metadata

## Overview
This index tracks execution packages for implementing ADR-008.

## Packages
1. [WP-001: Relationship Contract and Write Authority](./WP-001-relationship-contract-and-write-authority.md)
2. [WP-002: Relationship Schema Migration and Indexes](./WP-002-relationship-schema-migration-and-indexes.md)
3. [WP-003: Relationship Repositories and Row Models](./WP-003-relationship-repositories-and-row-models.md)
4. [WP-004: Relationship Service, Effective Projection, and Reconciliation](./WP-004-relationship-service-effective-projection-and-reconciliation.md)
5. [WP-005: REST Relationship Metadata Contracts](./WP-005-rest-relationship-metadata-contracts.md)
6. [WP-006: MCP Relationship Read Tool and Runtime Wiring](./WP-006-mcp-relationship-read-tool-and-runtime-wiring.md)
7. [WP-007: Web UI Relationship Metadata Editor](./WP-007-web-ui-relationship-metadata-editor.md)
8. [WP-008: Relationship Integration and Regression Matrix](./WP-008-relationship-integration-and-regression-matrix.md)
9. [WP-009: Rollout and Operator Documentation](./WP-009-rollout-and-operator-documentation.md)

## Dependency Order
`WP-001 -> WP-002 -> WP-003 -> WP-004`

`WP-001 -> WP-005`

`WP-004 -> (WP-005 || WP-006)`

`WP-005 -> WP-007`

`(WP-003 || WP-004 || WP-005 || WP-006 || WP-007) -> WP-008 -> WP-009`

## Completion Summaries
Add completion summaries under `./completion-summaries/` as each package is finished.
