## WP-012: Rollout, Migration, and Operations Documentation

### Metadata

```yaml
WP_ID: WP-012
Title: Rollout, Migration, and Operations Documentation
Domain: Documentation
Priority: Medium
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-04
Started_Date: 2026-03-05
Completed_Date: 2026-03-05
```

---

### Description

**Context:**
Operators need clear rollout instructions for schema/backfill changes and MCP write-gate controls before enabling taxonomy features broadly.

**Scope:**
- Add taxonomy rollout/rollback runbook under `docs/operations/`.
- Update README and release documentation for taxonomy APIs, MCP tools, and runtime gates.
- Document migration safety checks and verification commands.

Excluded:
- Additional feature implementation.

**Success Criteria:**
- [x] Runbook includes phased rollout, verification, and rollback actions.
- [x] Docs describe MCP write gate defaults and enablement process.
- [x] Migration/backfill validation steps are clear and repeatable.

---

### Technical Requirements

**Input Contracts:**
- Test evidence and checklist from WP-011.
- Final route/tool contracts from WP-006 through WP-009.

**Output Contracts:**
- Operations runbook and updated top-level docs.
- Release-note-ready summary for ADR-005 delivery.

**Integration Points:**
- Final handoff artifact for implementation execution phase.

---

### Deliverables

**Documentation Deliverables:**
- [x] Add `docs/operations/domain-taxonomy-rollout-rollback.md`.
- [x] Update `README.md` taxonomy API/MCP sections.
- [x] Add or update release notes under `docs/releases/`.
- [x] Include explicit preflight and post-deploy verification checklist.

**Validation Deliverables:**
- [x] Ensure documentation references tested behavior only.
- [x] Confirm links and commands match current repository paths.

---

### Acceptance Criteria

**Functional:**
- [x] Operations team can execute rollout and rollback using docs alone.
- [x] Runtime write-gate behavior is documented with safe defaults.

**Testing/Verification:**
- [x] Documentation steps are dry-run validated against a test environment.
- [x] No stale or broken internal links remain.

### Execution Evidence

- Added dedicated ADR-005 runbook: `docs/operations/domain-taxonomy-rollout-rollback.md`
  - Includes phased rollout procedure, migration/backfill validation, explicit preflight checklist, explicit post-deploy checklist, rollback triggers, and rollback verification commands.
- Updated runtime + contract docs in `README.md`:
  - Added `SKILLSERVER_MCP_ENABLE_WRITES` / `--mcp-enable-writes` defaults and behavior.
  - Added taxonomy REST endpoint and filter contract coverage.
  - Added MCP taxonomy read/write tool contract guidance and write-gate behavior.
  - Linked ADR-005 runbook from a dedicated rollout/rollback section.
- Added release-note-ready artifact:
  - `docs/releases/2026-03-05-adr-005-taxonomy-release-notes.md`
- Dry-run verification commands executed for doc accuracy:
  - `go test ./pkg/persistence -run 'TestRunMigrations_(UpgradeFromVersionOneToLatest_AppliesTaxonomySchema|CatalogTaxonomySchema_CascadeDeletesAndTagAssignmentQueriesRemainValid)' -count=1`
  - `go test ./cmd/skillserver -run 'TestCatalogPersistenceCoordinator_FullSyncAndRebuild_BackfillsLegacyLabelsIntoTaxonomyTags|TestMCPConfig_(Defaults|EnvOverrides|FlagPrecedence|InvalidEnableWritesBoolean)' -count=1`
  - `go test ./pkg/domain -run 'TestCatalogTaxonomy|TestCatalogTaxonomyLegacyLabelBackfillService|TestCatalogTaxonomyAssignmentService|TestCatalogEffectiveService_List_MergesTaxonomyReferencesAndAppliesTaxonomyFilters' -count=1`
  - `go test ./pkg/web -run 'TestCatalogTaxonomyRegistryEndpoints_|TestCatalogItemTaxonomyEndpoints_|TestCatalogEndpoints_TaxonomyFilters_' -count=1`
  - `go test ./pkg/mcp -run 'TestMCPServer_StdioRegression|TestTaxonomyWriteTools_' -count=1`

---

### Dependencies

**Blocked By:**
- WP-011

**Blocks:**
- None.

**Parallel Execution:**
- Can run in parallel with: Final stabilization PRs once WP-011 is mostly complete.
- Cannot run in parallel with: None.

---

### Risks

**Risk 1: Docs drift from implemented behavior during late changes**
- Probability: Medium
- Impact: Medium
- Mitigation: Update docs after final regression pass and before release cut.

**Risk 2: Missing rollback details increases incident recovery time**
- Probability: Low
- Impact: High
- Mitigation: Include explicit rollback triggers, commands, and validation checkpoints.
