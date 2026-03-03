# Dynamic Resource Import Discovery Rollout Runbook

## Purpose
Roll out ADR-002 dynamic prompt/imported resource discovery with a deterministic validation and rollback path.

## References
- ADR: [ADR-002: Dynamic Imported Resource Discovery and Prompt Support](/home/jeff/skillserver/docs/adrs/002-dynamic-resource-and-prompt-discovery.md)
- Work package evidence:
  - [WP-008 integration/security regression matrix](/home/jeff/skillserver/docs/implementation-plans/dynamic-resource-and-prompt-discovery/work-packages/WP-008-integration-security-regression-tests.md)
  - [WP-008 UI verification checklist](/home/jeff/skillserver/docs/implementation-plans/dynamic-resource-and-prompt-discovery/work-packages/WP-008-ui-manual-verification-checklist.md)
- Runtime docs: [README.md](/home/jeff/skillserver/README.md)

## Runtime Controls
- Flag: `--enable-import-discovery=true|false`
- Env: `SKILLSERVER_ENABLE_IMPORT_DISCOVERY=true|false`
- Default: `true`

When disabled:
- Imported discovery from `SKILL.md` is off.
- Virtual `imports/...` read/info lookups are rejected.
- Direct legacy resources remain available.

## Pre-Deploy Checklist
- [ ] Candidate commit includes WP-008 evidence and passing package tests.
- [ ] Release notes mention additive metadata (`origin`, `writable`) and dynamic groups (`prompts`, `imported`).
- [ ] Rollback owner is assigned and has permission to redeploy config changes.

## Validation Commands
Run before production rollout:

```bash
go test ./pkg/domain -count=1
go test ./pkg/mcp -count=1
go test ./pkg/web -count=1
npm run test:playwright
```

Expected:
- Domain/MCP/Web package tests: `ok`
- Playwright suite: pass (desktop + narrow viewport checks)

## Rollout Steps
1. Deploy candidate with default behavior (`enable-import-discovery=true`).
2. Run REST smoke check against at least one legacy-only skill and one additive skill.
3. Run MCP smoke check for `list_skill_resources` and `get_skill_resource_info` on an imported path.
4. Confirm UI shows dynamic groups and imported read-only locks.
5. Monitor for compatibility regressions in existing clients expecting legacy keys.

## Compatibility Smoke Expectations
- Legacy-only skills still expose `scripts`, `references`, `assets`.
- Additive skills include `prompts` and/or `imported` groups.
- Resource objects include `origin` and `writable`.
- Write operations against `imports/...` are rejected.

## Rollback Triggers
Rollback if any of the following occur:
- Client breakage from additive payload handling.
- Incorrect writability behavior for imported resources.
- Security concerns around imported path resolution.
- UI regressions that block resource usage.

## Rollback Procedure
1. Disable import discovery:
   - Flag mode: `./skillserver --enable-import-discovery=false`
   - Env mode:
     - `export SKILLSERVER_ENABLE_IMPORT_DISCOVERY=false`
     - restart/redeploy service
2. Re-run quick verification:
   - `GET /api/skills/:name/resources` on additive skill no longer includes imported-discovery results.
   - `read_skill_resource` for `imports/...` now fails with `import discovery is disabled`.
3. If issue persists, roll back to previous release artifact and repeat WP-008 package tests.

## Post-Rollback / Post-Rollout Closeout
- [ ] Record command outputs and decision timestamp.
- [ ] Link incident or rollout notes to WP-009 completion summary.
- [ ] Open follow-up issues for any residual compatibility concerns.
