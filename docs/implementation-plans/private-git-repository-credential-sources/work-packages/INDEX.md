# Work Packages: ADR-006 Private Git Repository Credential Sources

## Overview
This index tracks execution packages for implementing private Git repository credential sources against the canonical ADR-006 decision record.

## Packages
1. [WP-001: Private Git Runtime Flags and Startup Guardrails](./WP-001-private-git-runtime-flags-and-startup-guardrails.md)
2. [WP-002: Git Repo Identity, Canonical URL Semantics, and Config Migration](./WP-002-git-repo-identity-canonical-url-semantics-and-config-migration.md)
3. [WP-003: Env/File Credential Resolver and go-git Auth Builder](./WP-003-env-file-credential-resolver-and-go-git-auth-builder.md)
4. [WP-004: Encrypted Git Credential Store and Key Management](./WP-004-encrypted-git-credential-store-and-key-management.md)
5. [WP-005: Git Syncer Auth Integration, Checkout Semantics, and Redacted Status Reporting](./WP-005-git-syncer-auth-integration-checkout-semantics-and-redacted-status-reporting.md)
6. [WP-006: Secret-Safe Git Repo API Contracts and Handlers](./WP-006-secret-safe-git-repo-api-contracts-and-handlers.md)
7. [WP-007: Web UI Private Repo Credential Workflows and Masked Status UX](./WP-007-web-ui-private-repo-credential-workflows-and-masked-status-ux.md)
8. [WP-008: Private Repo Integration and Regression Test Matrix](./WP-008-private-repo-integration-and-regression-test-matrix.md)
9. [WP-009: Operations Docs, Rollback Guidance, and ADR Consolidation](./WP-009-operations-docs-rollback-guidance-and-adr-consolidation.md)

## Dependency Order
`WP-001 -> WP-004`
`WP-002 -> (WP-003 || WP-004) -> WP-005 -> WP-006 -> WP-007 -> WP-008 -> WP-009`

Detailed dependency notes:
- `WP-003` requires `WP-002`.
- `WP-004` requires `WP-001` and `WP-002`.
- `WP-005` requires `WP-002`, `WP-003`, and `WP-004`.
- `WP-006` requires `WP-001`, `WP-002`, `WP-004`, and `WP-005`.
- `WP-009` should not close until `WP-008` validates the shipped contract.

## Completion Summaries
Add completion summaries under `./completion-summaries/` as each package is finished.
