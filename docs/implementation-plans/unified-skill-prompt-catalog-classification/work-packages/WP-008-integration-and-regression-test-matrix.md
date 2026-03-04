## WP-008: Integration and Regression Test Matrix

### Metadata

```yaml
WP_ID: WP-008
Title: Integration and Regression Test Matrix
Domain: Quality Engineering
Priority: High
Estimated_Effort: 6 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-04
Started_Date: 2026-03-04
Completed_Date: 2026-03-04
```

---

### Description

**Context:**
Catalog unification touches domain indexing, manager rebuild, API contracts, UI behavior, and optional MCP tools. Regression safety is mandatory.

**Scope:**
- Add domain integration tests for prompt catalog generation and classifier filtering.
- Add API contract tests for `/api/catalog` list/search.
- Add UI verification for mixed catalog rendering and badge display.
- Add MCP regression tests when WP-007 is in scope.
- Add basic rebuild performance check for prompt-heavy repos.

Excluded:
- New production monitoring dashboards (covered in docs/ops follow-up).

**Success Criteria:**
- [x] Test matrix covers all must-have ADR requirements.
- [x] No regressions in `/api/skills`, existing MCP skill tools, or edit workflows.

---

### Technical Requirements

**Input Contracts:**
- Completed implementation from WP-001..WP-006 (and WP-007 if selected).

**Output Contracts:**
- Expanded automated test suites under `pkg/domain`, `pkg/web`, and `pkg/mcp`.
- UI verification checklist artifact under work-packages.

**Integration Points:**
- Gates WP-009 rollout documentation and release sign-off.

---

### Deliverables

**Code Deliverables:**
- [x] Add/extend domain tests for catalog generation, classifier filtering, and dedupe.
- [x] Add/extend web tests for catalog endpoint behavior.
- [x] Add/extend MCP tests for catalog parity tools (if enabled).

**Validation Deliverables:**
- [x] Add UI verification checklist for mixed tile rendering.
- [x] Capture performance measurements for index rebuild on representative fixture.

---

### Acceptance Criteria

**Functional:**
- [x] Must-have ADR requirements are directly validated by tests.
- [x] Skill-only flows remain non-regressed.

**Testing:**
- [x] Relevant package test suites pass.
- [x] Catalog behavior is deterministic and classifier-accurate in test fixtures.

---

### ADR Requirement Coverage Matrix

| ADR-003 Requirement | Validation |
|---|---|
| REQ-1: Prompt path identification and backward-compatible `agents` support | `pkg/domain/catalog_test.go`, `pkg/domain/manager_catalog_test.go`, `pkg/domain/resource_imports_test.go` |
| REQ-2: Prompt files exposed as top-level GUI catalog tiles | `tests/playwright/wp005-ui-catalog.spec.ts` (`renders mixed catalog badges and opens prompts as read-only`) |
| REQ-3: Explicit `skill`/`prompt` classifier on every catalog item | `pkg/domain/catalog_test.go`, `pkg/web/handlers_catalog_test.go`, `pkg/mcp/server_stdio_regression_test.go` |
| REQ-4: Classifier filter queryable in index/search | `pkg/domain/search_test.go`, `pkg/web/handlers_catalog_test.go`, `pkg/mcp/server_stdio_regression_test.go` |
| REQ-6: Existing skill CRUD/resource behavior remains backward compatible | `pkg/web/handlers_catalog_test.go`, `tests/playwright/wp005-ui-catalog.spec.ts`, `tests/playwright/wp008-ui.spec.ts`, MCP legacy tool assertions in `pkg/mcp/server_stdio_regression_test.go` |
| REQ-7 (optional): MCP catalog parity tools | `pkg/mcp/server_stdio_regression_test.go` (`list_catalog`, `search_catalog`) |

---

### Execution Evidence (2026-03-04)

**Domain Integration + Determinism**
- `pkg/domain/catalog_test.go`
- `pkg/domain/search_test.go`
- `pkg/domain/manager_catalog_test.go`

**API Contract Regression**
- `pkg/web/handlers_catalog_test.go`
- Covers `/api/catalog`, `/api/catalog/search`, invalid classifier, and `/api/skills` compatibility.

**MCP Regression**
- `pkg/mcp/server_stdio_regression_test.go`
- Covers legacy + catalog tool registration, classifier filtering, and invalid-input error behavior.

**UI Verification**
- Automated scripts:
  - `tests/playwright/wp005-ui-catalog.spec.ts` (mixed catalog tiles, badges, CRUD regression)
  - `tests/playwright/wp008-ui.spec.ts` (resource-tab regression and responsive checks)
- Checklist artifact:
  - `WP-008-ui-mixed-catalog-verification-checklist.md`

**Prompt-Heavy Rebuild Performance Check**
- Added benchmark:
  - `pkg/domain/manager_catalog_benchmark_test.go`
- Command:
  - `go test ./pkg/domain -run '^$' -bench BenchmarkFileSystemManager_RebuildIndex_PromptHeavyRepository -benchmem -count=1`
- Measurements:
  - `1 2814546579 ns/op`
  - `681417888 B/op`
  - `2419166 allocs/op`

**Validation Command Outcomes**
- `go test ./pkg/domain -count=1` -> `ok`
- `go test ./pkg/web -count=1` -> `ok`
- `go test ./pkg/mcp -count=1` -> `ok`
- `npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp008-ui.spec.ts --project=chromium` -> `6 passed`

---

### Dependencies

**Blocked By:**
- WP-004
- WP-005
- WP-006
- WP-007 (if in scope)

**Blocks:**
- WP-009

**Parallel Execution:**
- Can run in parallel with: none (validation gate)
- Cannot run in parallel with: incomplete upstream implementation WPs

---

### Risks

**Risk 1: Coverage gaps miss edge cases in imported prompt classification**
- Probability: Medium
- Impact: High
- Mitigation: Include edge fixtures for nested imports, aliases, and extension mismatches.

**Risk 2: Flaky UI assertions slow delivery**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep deterministic fixture data and prioritize stable assertions.
