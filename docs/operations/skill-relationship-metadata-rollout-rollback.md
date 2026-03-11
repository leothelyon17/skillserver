# Skill Relationship Metadata Rollout and Rollback Runbook

## Purpose
Deterministic rollout and rollback procedure for ADR-008 skill-to-prompt and skill-to-rule relationship metadata across REST, MCP, and Web UI metadata surfaces.

## References
- ADR: [ADR-008: Skill-to-Rule and Skill-to-Prompt Relationship Metadata](/home/jeff/skillserver/docs/adrs/008-skill-rule-and-prompt-relationship-metadata.md)
- Runtime and API docs: [README.md](/home/jeff/skillserver/README.md)
- Validation evidence:
  - [WP-008 completion summary](/home/jeff/skillserver/docs/implementation-plans/skill-rule-and-prompt-relationship-metadata/work-packages/completion-summaries/WP-008-completion-summary.md)
  - [WP-009 completion summary](/home/jeff/skillserver/docs/implementation-plans/skill-rule-and-prompt-relationship-metadata/work-packages/completion-summaries/WP-009-completion-summary.md)

## Runtime Controls
- ADR-008 adds no feature-specific flags or environment variables.
- Relationship metadata depends on ADR-004 persistence runtime:
  - `SKILLSERVER_PERSISTENCE_DATA` / `--persistence-data`
  - `SKILLSERVER_PERSISTENCE_DIR` / `--persistence-dir`
  - `SKILLSERVER_PERSISTENCE_DB_PATH` / `--persistence-db-path`

Behavior notes:
- Relationship metadata is additive and uses the existing persistence runtime plus startup/sync reconciliation.
- GUI and REST writes are skill-owned only in v1.
- MCP exposes read-only relationship lookup via `get_catalog_item_relationships`.
- `GET /api/catalog` and `GET /api/catalog/search` stay relationship-light.
- Catalog tiles do not render relationship badges in v1.
- Preferred rollback is deployment rollback to the last pre-ADR-008 build; no destructive schema rollback is required.

## Preconditions
- Rollout owner and rollback owner are assigned.
- Candidate build includes WP-008 verification evidence (or equivalent rerun evidence).
- ADR-004 persistence runtime is enabled and healthy in the target environment.
- `jq` is available for REST validation snippets.
- Optional but recommended: Playwright dependencies are available for UI gate verification.

## Verified Command Matrix (WP-008 Gate)
Use these exact commands as the rollout and rollback verification gate:

```bash
go test ./pkg/persistence -run 'TestCatalogSkillRuleRelationshipRepository|TestCatalogSkillPromptRelationshipRepository|TestCatalogRelationshipRepositories' -count=1
go test ./pkg/domain -run 'TestCatalogRelationshipService' -count=1
go test ./pkg/web -run 'TestCatalogRelationshipMetadataEndpoints|TestCatalogMetadataEndpoints' -count=1
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1
npx playwright test tests/playwright/wp007-ui-relationship-metadata-editor.spec.ts --project=chromium
npx playwright test tests/playwright/wp008-ui.spec.ts --project=chromium
```

## Rollout Validation Checklist
- [ ] REST metadata reads expose the additive `relationships` object for `skill`, `prompt`, and `rule` items.
- [ ] REST relationship writes remain limited to `skill` items.
- [ ] MCP `get_catalog_item_relationships` returns the same normalized relationship semantics as REST metadata reads.
- [ ] No MCP relationship write tool is registered.
- [ ] Deleted or missing related endpoints are suppressed from effective reads.
- [ ] UI metadata editing remains additive and does not change Git-backed content mutability.

## REST Validation Snippets
Use one known related `skill`/`prompt`/`rule` set from the target environment.

```bash
set -euo pipefail

BASE_URL="http://127.0.0.1:8080"
SKILL_ITEM_ID="skill:demo-skill"
PROMPT_ITEM_ID="prompt:demo-skill:prompts/system.md"
RULE_ITEM_ID="rule:demo-skill:rules/security.md"

SKILL_ITEM_ID_ESCAPED=$(printf '%s' "$SKILL_ITEM_ID" | jq -sRr @uri)
PROMPT_ITEM_ID_ESCAPED=$(printf '%s' "$PROMPT_ITEM_ID" | jq -sRr @uri)
RULE_ITEM_ID_ESCAPED=$(printf '%s' "$RULE_ITEM_ID" | jq -sRr @uri)

curl -sS "$BASE_URL/api/catalog/$SKILL_ITEM_ID_ESCAPED/metadata" | jq '.relationships'

curl -sS -X PATCH "$BASE_URL/api/catalog/$SKILL_ITEM_ID_ESCAPED/relationships" \
  -H "Content-Type: application/json" \
  --data "{\"prompt_item_id\":\"$PROMPT_ITEM_ID\",\"rule_item_ids\":[\"$RULE_ITEM_ID\"],\"updated_by\":\"ops-rollout\"}" \
  | jq '.relationships'

curl -sS "$BASE_URL/api/catalog/$PROMPT_ITEM_ID_ESCAPED/metadata" | jq '.relationships.skills'
curl -sS "$BASE_URL/api/catalog/$RULE_ITEM_ID_ESCAPED/metadata" | jq '.relationships.skills'
```

Expected outcomes:
- Skill metadata shows forward `prompt` and `rules`, with empty `skills`.
- Prompt and rule metadata show reverse `skills`, with empty forward fields.
- `PATCH /api/catalog/:id/relationships` returns `403` for prompt/rule paths, `404` for unknown items, and `400` for duplicate or classifier-invalid IDs.

## MCP Validation Expectations
- Invoke `get_catalog_item_relationships` for one known `skill`, `prompt`, and `rule` item.
- Expect the same normalized envelope used by REST metadata reads.
- Bare `<skill-id>` compatibility is allowed only when the requested item is a `skill`.
- Prompt and rule IDs remain canonical-only.
- Confirm no MCP relationship write tool is present; the regression gate above validates this objectively.

## Rollback Triggers
Rollback immediately if any of the following occurs:
- REST and MCP surfaces disagree on relationship projection semantics.
- Prompt or rule items become writable through relationship APIs.
- Deleted or missing related endpoints leak into effective relationship reads.
- UI relationship workflows regress metadata mutability or surface tile-level relationship rendering unexpectedly.
- Validation gate commands above fail in the release candidate or target environment.

## Rollback Procedure
1. Stop new relationship-edit operations through the UI or REST automation.
2. Redeploy the last pre-ADR-008 application build using the normal deployment rollback path.
3. Keep SQLite persistence data intact; do not drop relationship tables or delete relationship rows as part of rollback.
4. Re-run the WP-008 command matrix against the rolled-back build if practical.
5. If a broader persistence-backed metadata fallback is acceptable, disable ADR-004 persistence runtime instead of performing destructive data changes.

Broad fallback command:

```bash
# Flag-based fallback
./skillserver --persistence-data=false

# Env-based fallback
export SKILLSERVER_PERSISTENCE_DATA=false
./skillserver
```

Important:
- Deployment rollback is preferred for ADR-008-specific issues because disabling persistence also removes broader metadata and taxonomy features.
- Existing additive schema changes can remain in place for future re-enable.

## Post-Rollback Verification Checklist
- [ ] Relationship write paths are no longer exposed through the active deployment.
- [ ] Catalog list/search behavior is stable.
- [ ] No destructive database repair was required.
- [ ] Operators recorded the build version, timestamp, and executed verification commands.

## Closeout
- [ ] Record the rollout decision (`go` or `no-go`) with timestamp.
- [ ] Attach command outputs from the WP-008 gate to the release record.
- [ ] Link the final execution outcome in [WP-009 completion summary](/home/jeff/skillserver/docs/implementation-plans/skill-rule-and-prompt-relationship-metadata/work-packages/completion-summaries/WP-009-completion-summary.md).
