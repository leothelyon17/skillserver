# Work Packages: MCP Streamable HTTP Transport

## Overview
This index tracks all work packages for implementing ADR-001.

## Packages
1. [WP-001: Runtime MCP Config Contract](./WP-001-config-runtime-mcp-transport.md)
2. [WP-002: MCP Streamable HTTP Handler Support](./WP-002-mcp-streamable-http-handler.md)
3. [WP-003: Web Route Integration for `/mcp`](./WP-003-web-route-integration-mcp.md)
4. [WP-004: Transport Mode Runtime Orchestration](./WP-004-runtime-transport-orchestration.md)
5. [WP-005: Streamable HTTP Integration Tests](./WP-005-streamable-http-integration-tests.md)
6. [WP-006: Stdio Regression + Mixed-Mode Resilience Tests](./WP-006-stdio-regression-mixed-mode-tests.md)
7. [WP-007: Documentation Update (User + Operator)](./WP-007-documentation-updates.md)
8. [WP-008: Rollout Validation + Rollback Runbook](./WP-008-rollout-validation-rollback-runbook.md)

## Dependency Order
`WP-001 -> WP-002 -> WP-003 -> WP-004 -> (WP-005 || WP-006 || WP-007) -> WP-008`

## Completion Summaries
Completion summaries should be added under `./completion-summaries/` as each WP is finished.
