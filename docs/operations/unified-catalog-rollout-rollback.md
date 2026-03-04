# Unified Skill/Prompt Catalog Rollout Runbook

## Purpose
Deterministic rollout and rollback procedure for ADR-003 unified skill/prompt catalog classification.

## References
- ADR: [ADR-003: Unified Skill/Prompt Catalog Classification for Git Imports](/home/jeff/skillserver/docs/adrs/003-unified-skill-prompt-catalog-classification.md)
- Runtime/API docs: [README.md](/home/jeff/skillserver/README.md)
- Validation evidence:
  - [WP-008 integration/regression matrix](/home/jeff/skillserver/docs/implementation-plans/unified-skill-prompt-catalog-classification/work-packages/WP-008-integration-and-regression-test-matrix.md)
  - [WP-008 UI mixed-catalog checklist](/home/jeff/skillserver/docs/implementation-plans/unified-skill-prompt-catalog-classification/work-packages/WP-008-ui-mixed-catalog-verification-checklist.md)
- Optional MCP parity scope (if enabled):
  - [WP-007 completion summary](/home/jeff/skillserver/docs/implementation-plans/unified-skill-prompt-catalog-classification/work-packages/completion-summaries/WP-007-completion-summary.md)

## Runtime Controls
Catalog controls:
- `SKILLSERVER_CATALOG_ENABLE_PROMPTS` / `--catalog-enable-prompts` (default `true`)
- `SKILLSERVER_CATALOG_PROMPT_DIRS` / `--catalog-prompt-dirs` (default `agent,agents,prompt,prompts`)

Related controls:
- `SKILLSERVER_ENABLE_IMPORT_DISCOVERY` / `--enable-import-discovery` (default `true`)

Behavior notes:
- `catalog-enable-prompts=false` keeps `/api/catalog` available but returns skill-only catalog items.
- `catalog-prompt-dirs` must be a comma-separated list of single directory names (no nested paths).
- `/api/skills` and `/api/skills/search` remain additive-compatible and unchanged.

## Preconditions
- WP-008 validation evidence exists for the candidate commit.
- Rollout owner and rollback owner are assigned.
- At least one fixture/tenant includes both a skill and prompt catalog item.

## Pre-Deploy Checklist
- [ ] Candidate commit has passing test evidence from WP-008:
  - `go test ./pkg/domain -count=1`
  - `go test ./pkg/web -count=1`
  - `go test ./pkg/mcp -count=1`
  - `npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp008-ui.spec.ts --project=chromium`
- [ ] Runtime config explicitly sets (or intentionally defaults) catalog flags.
- [ ] Rollback override is prepared: `SKILLSERVER_CATALOG_ENABLE_PROMPTS=false`.
- [ ] Optional MCP parity decision is explicit (`use list_catalog/search_catalog` vs legacy skill-only tools).

## API Smoke Checks
Run these checks in canary before full rollout.

```bash
set -euo pipefail

BASE_URL="http://127.0.0.1:8080"

# 1) List unified catalog and verify classifier field exists.
curl -sS "$BASE_URL/api/catalog" | jq '.' > /tmp/wp009-catalog.json
jq -e 'length >= 1' /tmp/wp009-catalog.json >/dev/null
jq -e '.[0] | has("classifier")' /tmp/wp009-catalog.json >/dev/null

# 2) Verify prompt-filtered search returns only prompt classifier entries.
curl -sS "$BASE_URL/api/catalog/search?q=prompt&classifier=prompt" | jq '.' > /tmp/wp009-catalog-prompt.json
jq -e 'all(.[]; .classifier == "prompt")' /tmp/wp009-catalog-prompt.json >/dev/null

# 3) Verify skill-filtered search returns only skill classifier entries.
curl -sS "$BASE_URL/api/catalog/search?q=skill&classifier=skill" | jq '.' > /tmp/wp009-catalog-skill.json
jq -e 'all(.[]; .classifier == "skill")' /tmp/wp009-catalog-skill.json >/dev/null

# 4) Invalid classifier must fail with HTTP 400.
INVALID_CODE=$(curl -sS -o /tmp/wp009-invalid-classifier.json -w "%{http_code}" \
  "$BASE_URL/api/catalog/search?q=skill&classifier=skills")
test "$INVALID_CODE" = "400"

# 5) Backward compatibility: /api/skills still serves skill payload.
curl -sS "$BASE_URL/api/skills" | jq '.' > /tmp/wp009-skills.json
jq -e 'length >= 1' /tmp/wp009-skills.json >/dev/null
jq -e '.[0] | has("readOnly")' /tmp/wp009-skills.json >/dev/null
jq -e '.[0] | has("classifier") | not' /tmp/wp009-skills.json >/dev/null
```

## UI Smoke Checks
- Run `tests/playwright/wp005-ui-catalog.spec.ts` for mixed tiles, badges, prompt view behavior, and skill CRUD regression.
- Run `tests/playwright/wp008-ui.spec.ts` for resource-tab grouping/lock-state and responsive checks.
- Manually confirm in UI:
  - Mixed `skill` and `prompt` tiles render.
  - Prompt tiles open read-only guidance view.
  - Skill edit/create/delete behavior still works.

## Optional MCP Parity Smoke Checks (WP-007 in scope)
Use these checks when MCP catalog tools are part of your rollout contract.

```bash
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1
```

Expected MCP outcomes:
- `list_catalog` and `search_catalog` are registered.
- Optional `classifier` filtering accepts `skill` and `prompt`.
- Legacy `list_skills` / `search_skills` behavior remains stable.

## Canary Go/No-Go Gates
All gates must pass before full rollout:
- API smoke checks pass with expected classifier behavior.
- UI smoke checks pass (`wp005` + `wp008` suites).
- `/api/skills` backward-compatibility behavior is unchanged.
- Optional MCP parity checks pass if in scope.

Any gate failure is immediate no-go and triggers rollback.

## Rollback Triggers
Rollback if any of the following occur:
- Prompt catalog entries are missing/incorrect for expected prompt paths.
- Classifier filtering returns mixed or invalid results.
- UI mixed-catalog behavior regresses core skill workflows.
- MCP clients depending on legacy skill-only workflows are impacted.

## Rollback Procedure
Execute in this order:

1. Disable prompt catalog indexing/classification.

```bash
# Flag-based rollback
./skillserver --catalog-enable-prompts=false

# Env-based rollback
export SKILLSERVER_CATALOG_ENABLE_PROMPTS=false
./skillserver
```

2. If classifier drift is caused by custom prompt directories, restore defaults.

```bash
# Flag-based rollback to defaults
./skillserver --catalog-enable-prompts=true --catalog-prompt-dirs "agent,agents,prompt,prompts"

# Env-based rollback to defaults
export SKILLSERVER_CATALOG_ENABLE_PROMPTS=true
export SKILLSERVER_CATALOG_PROMPT_DIRS="agent,agents,prompt,prompts"
./skillserver
```

3. Re-run quick validation:
- `/api/catalog` returns skill-only entries when prompts are disabled.
- `/api/catalog/search?...&classifier=prompt` returns an empty array (or known-safe baseline).
- `/api/skills` and `/api/skills/search` remain functional.

4. If issues persist, roll back to previous release artifact and rerun WP-008 package checks.

## Post-Rollout / Post-Rollback Closeout
- [ ] Record decision timestamp and command evidence.
- [ ] Link rollout evidence to WP-009 completion summary.
- [ ] Capture follow-up issues for any residual contract/documentation gaps.
