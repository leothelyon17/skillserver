# Work Packages: Dynamic Resource and Prompt Discovery

## Overview
This index tracks all work packages for implementing ADR-002.

## Packages
1. [WP-001: Resource Contract Extension for Prompts and Imports](./WP-001-resource-contract-prompts-imports.md)
2. [WP-002: Import Parser and Safe Resolver](./WP-002-import-parser-safe-resolver.md)
3. [WP-003: Manager Discovery Integration and Deterministic Dedupe](./WP-003-manager-discovery-dedupe-integration.md)
4. [WP-004: Virtual Import Read and Info Resolution](./WP-004-virtual-import-read-info-resolution.md)
5. [WP-005: MCP Resource Contract Metadata Update](./WP-005-mcp-resource-contract-metadata.md)
6. [WP-006: REST Resource Grouping and Write Guards](./WP-006-rest-resource-grouping-write-guards.md)
7. [WP-007: Web UI Dynamic Resource Group Rendering](./WP-007-web-ui-dynamic-resource-groups.md)
8. [WP-008: Integration, Security, and Regression Test Matrix](./WP-008-integration-security-regression-tests.md)
9. [WP-009: Documentation and Rollout Controls](./WP-009-documentation-rollout-and-flag-guidance.md)

## Dependency Order
`WP-001 -> WP-002 -> WP-003 -> (WP-004 || WP-005 || WP-006) -> WP-007 -> WP-008 -> WP-009`

## Completion Summaries
Completion summaries should be added under `./completion-summaries/` as each WP is finished.
