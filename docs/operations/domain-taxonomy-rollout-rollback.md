# Domain/Subdomain/Tag Taxonomy Rollout and Rollback Runbook

## Purpose
Deterministic rollout and rollback procedure for ADR-005 domain/subdomain/tag taxonomy classification.

## References
- ADR: [ADR-005: Domain/Subdomain/Tag Taxonomy for Catalog Items](/home/jeff/skillserver/docs/adrs/005-domain-subdomain-tag-taxonomy-for-catalog-items.md)
- Runtime/API docs: [README.md](/home/jeff/skillserver/README.md)
- Validation evidence:
  - [WP-011 taxonomy integration and regression matrix](/home/jeff/skillserver/docs/implementation-plans/domain-subdomain-tag-taxonomy-for-catalog-items/work-packages/WP-011-taxonomy-integration-and-regression-test-matrix.md)
  - [WP-011 completion summary](/home/jeff/skillserver/docs/implementation-plans/domain-subdomain-tag-taxonomy-for-catalog-items/work-packages/completion-summaries/WP-011-completion-summary.md)
  - [tests/README.md command matrix](/home/jeff/skillserver/tests/README.md)

## Runtime Controls
- `SKILLSERVER_PERSISTENCE_DATA` / `--persistence-data` (default: `false`)
- `SKILLSERVER_PERSISTENCE_DIR` / `--persistence-dir` (required when persistence is enabled)
- `SKILLSERVER_PERSISTENCE_DB_PATH` / `--persistence-db-path` (optional override)
- `SKILLSERVER_MCP_ENABLE_WRITES` / `--mcp-enable-writes` (default: `false`)

Behavior notes:
- Taxonomy REST endpoints require persistence runtime and return `503` when taxonomy services are unavailable.
- Taxonomy filters on `/api/catalog` and `/api/catalog/search` return `503` if effective metadata runtime is unavailable.
- MCP taxonomy write tools are not registered unless `SKILLSERVER_MCP_ENABLE_WRITES=true`.
- `tag_match` defaults to `any` when omitted in REST and MCP catalog filters.

## Preconditions
- Rollout owner and rollback owner are assigned.
- Candidate commit has WP-011 evidence or equivalent rerun evidence.
- Persistence storage is provisioned and writable.
- At least one catalog item exists for taxonomy assignment verification.
- Optional but recommended: `jq` installed for API payload assertions.

## Preflight Checklist
- [ ] Candidate commit passes the WP-011 command matrix:
  - `go test ./pkg/persistence -run 'TestRunMigrations_(UpgradeFromVersionOneToLatest_AppliesTaxonomySchema|CatalogTaxonomySchema_CascadeDeletesAndTagAssignmentQueriesRemainValid)' -count=1`
  - `go test ./cmd/skillserver -run 'TestCatalogPersistenceCoordinator_FullSyncAndRebuild_BackfillsLegacyLabelsIntoTaxonomyTags|TestMCPConfig_(Defaults|EnvOverrides|FlagPrecedence|InvalidEnableWritesBoolean)' -count=1`
  - `go test ./pkg/domain -run 'TestCatalogTaxonomy|TestCatalogTaxonomyLegacyLabelBackfillService|TestCatalogTaxonomyAssignmentService|TestCatalogEffectiveService_List_MergesTaxonomyReferencesAndAppliesTaxonomyFilters' -count=1`
  - `go test ./pkg/web -run 'TestCatalogTaxonomyRegistryEndpoints_|TestCatalogItemTaxonomyEndpoints_|TestCatalogEndpoints_TaxonomyFilters_' -count=1`
  - `go test ./pkg/mcp -run 'TestMCPServer_StdioRegression|TestTaxonomyWriteTools_' -count=1`
  - `npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp008-ui.spec.ts tests/playwright/wp010-ui-taxonomy.spec.ts --project=chromium`
- [ ] Runtime config explicitly sets (or intentionally defaults) taxonomy gates.
- [ ] MCP write-gate decision is explicit for this window:
  - Read-only MCP mode: `--mcp-enable-writes=false` (recommended default)
  - MCP write mode: `--mcp-enable-writes=true` (only when approved)
- [ ] SQLite backup exists for the deployment window.

## Rollout Procedure
1. Deploy with persistence enabled and MCP writes disabled.
2. Run post-deploy verification checks in canary.
3. Promote to full rollout only after all checks pass.
4. Optionally enable MCP write tools in a separate controlled change.

Example startup (recommended baseline):

```bash
./skillserver \
  --persistence-data=true \
  --persistence-dir ./data/skillserver \
  --mcp-enable-writes=false
```

## Post-Deploy Verification Checklist
- [ ] Taxonomy registry endpoints respond with `200` and valid JSON arrays.
- [ ] Item taxonomy assignment `GET/PATCH` works on at least one item.
- [ ] Taxonomy filters apply on both list/search catalog APIs.
- [ ] Migration/backfill evidence is available for the deployed commit.
- [ ] MCP write tools remain gated according to deployment decision.

### REST Verification Commands

```bash
set -euo pipefail

BASE_URL="http://127.0.0.1:8080"

# 1) Registry reads are available.
curl -sS "$BASE_URL/api/catalog/taxonomy/domains?active=true" | jq '.' > /tmp/wp012-domains.json
curl -sS "$BASE_URL/api/catalog/taxonomy/subdomains?active=true" | jq '.' > /tmp/wp012-subdomains.json
curl -sS "$BASE_URL/api/catalog/taxonomy/tags?active=true" | jq '.' > /tmp/wp012-tags.json

jq -e 'type == "array"' /tmp/wp012-domains.json >/dev/null
jq -e 'type == "array"' /tmp/wp012-subdomains.json >/dev/null
jq -e 'type == "array"' /tmp/wp012-tags.json >/dev/null

# 2) Pick one item and fetch taxonomy assignment view.
ITEM_ID=$(curl -sS "$BASE_URL/api/catalog" | jq -r '.[0].id')
ITEM_ID_ESCAPED=$(jq -rn --arg v "$ITEM_ID" '$v|@uri')
curl -sS "$BASE_URL/api/catalog/${ITEM_ID_ESCAPED}/taxonomy" | jq '.' > /tmp/wp012-item-taxonomy-before.json

# 3) Apply a deterministic patch (safe no-op if selectors are blank).
DOMAIN_ID=$(jq -r '.[0].domain_id // empty' /tmp/wp012-domains.json)
SUBDOMAIN_ID=$(jq -r --arg d "$DOMAIN_ID" '[.[] | select(.domain_id == $d)][0].subdomain_id // empty' /tmp/wp012-subdomains.json)
TAG_ID=$(jq -r '.[0].tag_id // empty' /tmp/wp012-tags.json)

PATCH_PAYLOAD=$(jq -n \
  --arg pd "$DOMAIN_ID" \
  --arg ps "$SUBDOMAIN_ID" \
  --arg t "$TAG_ID" \
  '{
    primary_domain_id: (if $pd == "" then null else $pd end),
    primary_subdomain_id: (if $ps == "" then null else $ps end),
    tag_ids: (if $t == "" then [] else [$t] end),
    updated_by: "wp012-runbook"
  }')

curl -sS -X PATCH "$BASE_URL/api/catalog/${ITEM_ID_ESCAPED}/taxonomy" \
  -H "Content-Type: application/json" \
  --data "$PATCH_PAYLOAD" \
  | jq '.' > /tmp/wp012-item-taxonomy-after.json

jq -e 'has("item_id") and has("tags")' /tmp/wp012-item-taxonomy-after.json >/dev/null

# 4) Verify taxonomy filters on list/search (default tag_match=any).
if [ -n "$DOMAIN_ID" ]; then
  curl -sS "$BASE_URL/api/catalog?primary_domain_id=${DOMAIN_ID}" | jq '.' > /tmp/wp012-filter-list.json
  jq -e 'type == "array"' /tmp/wp012-filter-list.json >/dev/null
fi

if [ -n "$TAG_ID" ]; then
  curl -sS "$BASE_URL/api/catalog/search?q=skill&tag_ids=${TAG_ID}" | jq '.' > /tmp/wp012-filter-search-any.json
  jq -e 'type == "array"' /tmp/wp012-filter-search-any.json >/dev/null

  curl -sS "$BASE_URL/api/catalog/search?q=skill&tag_ids=${TAG_ID}&tag_match=all" | jq '.' > /tmp/wp012-filter-search-all.json
  jq -e 'type == "array"' /tmp/wp012-filter-search-all.json >/dev/null
fi
```

### MCP Gate Verification
Run this against the release commit:

```bash
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression|TestTaxonomyWriteTools_' -count=1
```

Expected outcomes:
- With `--mcp-enable-writes=false`: taxonomy read tools are available, write tools are absent.
- With `--mcp-enable-writes=true`: taxonomy write tools are registered and executable.

## Migration and Backfill Validation
Use these commands to validate schema/backfill safety on the release candidate:

```bash
go test ./pkg/persistence -run 'TestRunMigrations_(UpgradeFromVersionOneToLatest_AppliesTaxonomySchema|CatalogTaxonomySchema_CascadeDeletesAndTagAssignmentQueriesRemainValid)' -count=1
go test ./cmd/skillserver -run 'TestCatalogPersistenceCoordinator_FullSyncAndRebuild_BackfillsLegacyLabelsIntoTaxonomyTags' -count=1
```

Optional runtime observation:
- Enable logging (`SKILLSERVER_ENABLE_LOGGING=true`) and confirm startup logs include taxonomy backfill completion summary with item/tag counts.

## Rollback Triggers
Rollback if any of these occur:
- Registry or assignment endpoints return unexpected `5xx` after rollout.
- Catalog taxonomy filters produce inconsistent or non-deterministic results.
- MCP write tools are exposed in an environment that should stay read-only.
- Migration/backfill checks fail for the release artifact.

## Rollback Procedure
1. Disable MCP taxonomy writes immediately:

```bash
# Flag-based rollback of MCP writes
./skillserver --mcp-enable-writes=false

# Env-based rollback of MCP writes
export SKILLSERVER_MCP_ENABLE_WRITES=false
./skillserver
```

2. If broader taxonomy/persistence issues continue, roll back to filesystem-only mode:

```bash
# Flag-based persistence rollback
./skillserver --persistence-data=false

# Env-based persistence rollback
export SKILLSERVER_PERSISTENCE_DATA=false
./skillserver
```

3. If required, restore the last known-good artifact and persistence DB backup.

## Post-Rollback Verification Checklist
- [ ] `/api/catalog` returns `200`.
- [ ] Taxonomy endpoints return `503` in non-persistence mode (expected fallback behavior).
- [ ] MCP taxonomy write tools are absent after write-gate rollback.
- [ ] Incident notes include executed commands and timestamps.

Quick rollback verification:

```bash
set -euo pipefail

BASE_URL="http://127.0.0.1:8080"
ITEM_ID=$(curl -sS "$BASE_URL/api/catalog" | jq -r '.[0].id')
ITEM_ID_ESCAPED=$(jq -rn --arg v "$ITEM_ID" '$v|@uri')

CODE=$(curl -sS -o /tmp/wp012-rollback-taxonomy.json -w "%{http_code}" \
  "$BASE_URL/api/catalog/${ITEM_ID_ESCAPED}/taxonomy")
test "$CODE" = "503"
```

## Post-Deployment Closeout
- [ ] Record rollout decision (`go`/`no-go`) with timestamp.
- [ ] Attach validation artifacts (`/tmp/wp012-*.json`) to release documentation.
- [ ] Link final outcome in [WP-012 completion summary](/home/jeff/skillserver/docs/implementation-plans/domain-subdomain-tag-taxonomy-for-catalog-items/work-packages/completion-summaries/WP-012-completion-summary.md).
