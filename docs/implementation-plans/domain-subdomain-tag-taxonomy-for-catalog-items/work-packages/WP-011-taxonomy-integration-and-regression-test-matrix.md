## WP-011: Taxonomy Integration and Regression Test Matrix

### Metadata

```yaml
WP_ID: WP-011
Title: Taxonomy Integration and Regression Test Matrix
Domain: Quality Engineering
Priority: High
Estimated_Effort: 6 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-04
Started_Date: 2026-03-05
Completed_Date: 2026-03-05
```

---

### Description

**Context:**
ADR-005 adds schema, service, API, MCP, and UI changes. A cross-surface regression matrix is required before rollout.

**Scope:**
- Implement integration/regression coverage for:
  - migration + backfill behaviors.
  - effective projection and filter semantics.
  - REST taxonomy CRUD and assignment APIs.
  - MCP read/write tool matrix and filter parity.
  - UI taxonomy workflows and compatibility paths.

Excluded:
- New feature logic beyond test facilitation.
- Rollout/operations docs (WP-012).

**Success Criteria:**
- [x] Must-have ADR behaviors are covered by automated tests.
- [x] Backward compatibility for labels/metadata paths is verified.
- [x] Rollout gate checklist has objective pass criteria.

---

### Technical Requirements

**Input Contracts:**
- Completed feature surfaces from WP-005 through WP-010.

**Output Contracts:**
- Test suites under `pkg/` and `tests/` aligned with existing repo patterns.
- Documented matrix/checklist for release sign-off.

**Integration Points:**
- WP-012 references this matrix as go/no-go input.

---

### Deliverables

**Code Deliverables:**
- [x] Add/extend persistence migration and backfill integration tests.
- [x] Add/extend domain service tests for assignment/filter/compatibility semantics.
- [x] Add API tests for registry CRUD + item taxonomy patch/get + list/search filters.
- [x] Add MCP tests for read/write registration and behavior by gate state.
- [x] Add UI regression checks for taxonomy manager and item chips/filters.

**Test Deliverables:**
- [x] Publish taxonomy regression matrix/checklist in test docs.
- [x] Ensure CI-friendly commands and deterministic fixtures.

---

### Acceptance Criteria

**Functional:**
- [x] Taxonomy workflows operate correctly across data/service/API/MCP/UI paths.
- [x] Compatibility behavior for legacy `labels` consumers is validated.

**Testing:**
- [x] Full taxonomy test matrix passes in clean environment.
- [x] No flaky tests introduced by taxonomy coverage.

### Execution Evidence

- `tests/README.md` now includes `WP-011 Taxonomy Integration and Regression` with CI-friendly commands and rollout gates.
- Regression assertions were extended for taxonomy filter parity:
  - REST list/search coverage now explicitly validates `secondary_domain_id`, `subdomain_id`, and default `tag_match=any`.
  - MCP list/search coverage now explicitly validates `secondary_domain_id`, `subdomain_id`, and default `tag_match=any`.
- Matrix commands executed successfully:
  - `go test ./pkg/persistence -run 'TestRunMigrations_(UpgradeFromVersionOneToLatest_AppliesTaxonomySchema|CatalogTaxonomySchema_CascadeDeletesAndTagAssignmentQueriesRemainValid)' -count=1`
  - `go test ./cmd/skillserver -run 'TestCatalogPersistenceCoordinator_FullSyncAndRebuild_BackfillsLegacyLabelsIntoTaxonomyTags|TestMCPConfig_(Defaults|EnvOverrides|FlagPrecedence|InvalidEnableWritesBoolean)' -count=1`
  - `go test ./pkg/domain -run 'TestCatalogTaxonomy|TestCatalogTaxonomyLegacyLabelBackfillService|TestCatalogTaxonomyAssignmentService|TestCatalogEffectiveService_List_MergesTaxonomyReferencesAndAppliesTaxonomyFilters' -count=1`
  - `go test ./pkg/web -run 'TestCatalogTaxonomyRegistryEndpoints_|TestCatalogItemTaxonomyEndpoints_|TestCatalogEndpoints_TaxonomyFilters_' -count=1`
  - `go test ./pkg/mcp -run 'TestMCPServer_StdioRegression|TestTaxonomyWriteTools_' -count=1`
  - `npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp008-ui.spec.ts tests/playwright/wp010-ui-taxonomy.spec.ts --project=chromium`

---

### Dependencies

**Blocked By:**
- WP-005
- WP-007
- WP-009
- WP-010

**Blocks:**
- WP-012

**Parallel Execution:**
- Can run in parallel with: Final bug fixes from prior WPs.
- Cannot run in parallel with: None strictly, but requires stable feature branches.

---

### Risks

**Risk 1: Test matrix misses one transport path (REST or MCP)**
- Probability: Medium
- Impact: High
- Mitigation: Explicitly map each ADR requirement to at least one automated test.

**Risk 2: UI regression tests become brittle**
- Probability: Medium
- Impact: Medium
- Mitigation: Prefer stable selectors and focused assertions over snapshot-heavy checks.
