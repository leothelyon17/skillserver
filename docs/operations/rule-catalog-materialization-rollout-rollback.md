# Rule Catalog and Materialization Rollout and Rollback Runbook

## Purpose
Deterministic phased rollout and rollback procedure for ADR-007 rule catalog objects and MCP project materialization.

## References
- ADR: [ADR-007: Rule Catalog Objects and MCP Project Materialization](/home/jeff/skillserver/docs/adrs/007-rule-catalog-objects-and-mcp-project-materialization.md)
- Runtime/API docs: [README.md](/home/jeff/skillserver/README.md)
- Runtime gate completion evidence:
  - [WP-004 completion summary](/home/jeff/skillserver/docs/implementation-plans/rule-catalog-objects-and-mcp-project-materialization/work-packages/completion-summaries/WP-004-completion-summary.md)
- Rollout safety verification evidence:
  - [WP-011 completion summary](/home/jeff/skillserver/docs/implementation-plans/rule-catalog-objects-and-mcp-project-materialization/work-packages/completion-summaries/WP-011-completion-summary.md)
  - [tests/README.md command matrix](/home/jeff/skillserver/tests/README.md)

## Runtime Controls
- `SKILLSERVER_CATALOG_ENABLE_RULES` / `--catalog-enable-rules` (default: `true`)
- `SKILLSERVER_CATALOG_RULE_DIRS` / `--catalog-rule-dirs` (default: `rule,rules`)
- `SKILLSERVER_CATALOG_RULE_FILENAMES` / `--catalog-rule-filenames` (default: `agents.md,rules.md,claude.md,gemini.md`)
- `SKILLSERVER_MCP_ENABLE_MATERIALIZATION` / `--mcp-enable-materialization` (default: `false`)
- `SKILLSERVER_MCP_ALLOWED_DESTINATION_ROOTS` / `--mcp-allowed-destination-roots` (required when materialization is enabled)

Behavior notes:
- Rule catalog indexing can be disabled without rolling back schema changes.
- Materialization writes remain disabled unless explicitly enabled.
- When materialization is enabled, allowed destination roots must be absolute and normalized.
- Rollback is gate-based (configuration only) and does not require destructive data operations.

## Preconditions
- Rollout owner and rollback owner are assigned.
- Candidate commit includes WP-011 verification evidence (or equivalent rerun evidence).
- Deploy-time destination roots are approved and absolute.
- Optional but recommended: `npx` and Playwright dependencies are available for UI gate verification.

## Verified Command Matrix (WP-011 Gate)
Use these exact commands as rollout and rollback verification gates:

```bash
go test ./pkg/domain -run 'TestCatalogExportService_|TestCatalogMaterializationService_' -count=1
go test ./pkg/persistence -run 'TestRunMigrations_UpgradeFromPreRuleSchemaToLatest_PreservesRowsAndAllowsRuleClassifier|TestCatalogSourceRepository_UpsertAndList_WithRuleClassifier_RoundTripsAndFilters|TestCatalogSourceRepository_RuleRowLifecycle_SoftDeleteAndRestorePreservesClassifierFiltering' -count=1
go test ./pkg/web -run 'TestExportSkill_LegacyRoute_|TestExportCatalog_|TestMaterializeCatalog_' -count=1
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1
npx playwright test tests/playwright/wp010-ui-export-materialization.spec.ts --project=chromium
```

## Phased Rollout Order

### Phase 1: Shared Export Seam
Goal: confirm shared export behavior while write-capable materialization remains disabled.

Runtime recommendation:
- `--mcp-enable-materialization=false`

Go/No-Go checks:
- Run the full WP-011 command matrix.
- Confirm legacy skill export compatibility gate remains green (`TestExportSkill_LegacyRoute_...`).

### Phase 2: Rule Indexing Enablement
Goal: enable and verify `rule` catalog indexing behavior.

Runtime recommendation:
- `--catalog-enable-rules=true`
- Keep explicit defaults unless there is a vetted override:
  - `--catalog-rule-dirs=rule,rules`
  - `--catalog-rule-filenames=agents.md,rules.md,claude.md,gemini.md`

Go/No-Go checks:
- Run the full WP-011 command matrix.
- Confirm persistence migration/lifecycle checks stay green for `rule` classifier flows.

### Phase 3: MCP/REST Materialization Enablement
Goal: enable write-capable materialization with bounded destination roots.

Runtime recommendation:
- `--mcp-enable-materialization=true`
- `--mcp-allowed-destination-roots=/workspace,/projects` (example; use deployment-approved roots)

Go/No-Go checks:
- Run the full WP-011 command matrix.
- Confirm explicit failures remain enforced for:
  - relative destination paths
  - outside-root destination paths
  - dry-run no-write semantics

### Phase 4: UI Enablement Validation
Goal: verify UI behavior with capability-gated write actions.

Go/No-Go checks:
- Run the full WP-011 command matrix.
- Ensure Playwright `wp010-ui-export-materialization.spec.ts` passes for:
  - dry-run-first behavior
  - capability-gated write controls
  - legacy skill export compatibility

## Rollback Triggers
Rollback immediately if any of the following occurs:
- Materialization writes fail path-safety guarantees.
- UI allows write actions when materialization gate should be off.
- Rule classifier behavior diverges from validated list/search semantics.
- Legacy skill export compatibility gate fails.

## Rollback Procedure (Ordered)
1. Disable materialization write gate.
2. Disable rule indexing gate if broader ADR-007 fallback is required.
3. Re-run WP-011 command matrix to validate fallback.

```bash
# 1) Immediate write-surface rollback
./skillserver --mcp-enable-materialization=false

# 2) Optional classifier rollback to pre-ADR-007 catalog surface
./skillserver --catalog-enable-rules=false
```

Important:
- Do not perform destructive schema rollback for this procedure.
- Keep persistence data intact for forward re-enable.

## Post-Rollback Verification Checklist
- [ ] WP-011 command matrix passes after gate rollback.
- [ ] Legacy skill export compatibility tests remain green.
- [ ] Materialization capability stays disabled in runtime behavior.
- [ ] Rule classifier can be re-enabled later without data repair.
- [ ] Incident notes include executed commands, operator, and timestamps.

## Closeout
- [ ] Record rollout/rollback decision (`go` or `no-go`) with timestamp.
- [ ] Attach gate command outputs to release records.
- [ ] Link final execution evidence in [WP-012 completion summary](/home/jeff/skillserver/docs/implementation-plans/rule-catalog-objects-and-mcp-project-materialization/work-packages/completion-summaries/WP-012-completion-summary.md).
