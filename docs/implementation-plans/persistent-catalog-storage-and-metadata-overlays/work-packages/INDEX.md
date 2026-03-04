# Work Packages: ADR-004 Persistent Catalog Storage and Metadata Overlays

## Overview
This index tracks execution packages for implementing ADR-004.

## Packages
1. [WP-001: Persistence Runtime Config and Startup Guardrails](./WP-001-persistence-runtime-config-and-startup-guardrails.md)
2. [WP-002: SQLite Bootstrap and Schema Migration Runner](./WP-002-sqlite-bootstrap-and-schema-migration-runner.md)
3. [WP-003: Catalog Source and Overlay Repository Layer](./WP-003-catalog-source-and-overlay-repository-layer.md)
4. [WP-004: Catalog Sync Engine and Reconciliation Semantics](./WP-004-catalog-sync-engine-and-reconciliation-semantics.md)
5. [WP-005: Effective Catalog Projection and Mutability Contract](./WP-005-effective-catalog-projection-and-mutability-contract.md)
6. [WP-006: Catalog Metadata API and Response Contract Extensions](./WP-006-catalog-metadata-api-and-response-contract-extensions.md)
7. [WP-007: Startup and Manual Git Resync Persistence Wiring](./WP-007-startup-and-manual-git-resync-persistence-wiring.md)
8. [WP-008: Web UI Metadata Overlay Editing and Mutability UX](./WP-008-web-ui-metadata-overlay-editing-and-mutability-ux.md)
9. [WP-009: Persistence Integration and Regression Test Matrix](./WP-009-persistence-integration-and-regression-test-matrix.md)
10. [WP-010: Operations Docs, Rollout Checklist, and Rollback Guidance](./WP-010-operations-docs-rollout-checklist-and-rollback-guidance.md)

## Dependency Order
`WP-001 -> WP-002 -> WP-003 -> (WP-004 || WP-005) -> WP-006 -> WP-008 -> WP-009 -> WP-010`

Additional branch:
`WP-001 -> WP-007`, and `WP-004 -> WP-007`, with `WP-007` required before `WP-009`.

## Completion Summaries
Add completion summaries under `./completion-summaries/` as each package is finished.
