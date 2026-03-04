# Work Packages: Unified Skill/Prompt Catalog Classification

## Overview
This index tracks all work packages for implementing ADR-003.

## Packages
1. [WP-001: Catalog Contract and Classifier Rules](./WP-001-catalog-contract-and-classifier-rules.md)
2. [WP-002: Bleve Catalog Index and Classifier Filtering](./WP-002-bleve-catalog-index-and-classifier-filtering.md)
3. [WP-003: Manager Catalog Builder and Rebuild Integration](./WP-003-manager-catalog-builder-and-rebuild-integration.md)
4. [WP-004: Catalog REST Endpoints and API Contracts](./WP-004-catalog-rest-endpoints-and-api-contracts.md)
5. [WP-005: Web UI Unified Catalog Rendering and Badges](./WP-005-web-ui-unified-catalog-rendering-and-badges.md)
6. [WP-006: Runtime Config for Prompt Catalog Detection](./WP-006-runtime-config-for-prompt-catalog-detection.md)
7. [WP-007: MCP Catalog Parity Tools (Optional)](./WP-007-mcp-catalog-parity-tools.md)
8. [WP-008: Integration and Regression Test Matrix](./WP-008-integration-and-regression-test-matrix.md)
9. [WP-009: Documentation, Rollout, and Rollback Guidance](./WP-009-documentation-rollout-and-rollback-guidance.md)

## Dependency Order
`WP-001 -> WP-002 -> WP-003 -> (WP-004 || WP-006 || WP-007) -> WP-005 -> WP-008 -> WP-009`

## Completion Summaries
Completion summaries should be added under `./completion-summaries/` as each WP is finished.
