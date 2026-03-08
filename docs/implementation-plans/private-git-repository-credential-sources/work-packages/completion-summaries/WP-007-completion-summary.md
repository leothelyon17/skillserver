# WP-007 Completion Summary

## Status
✅ Complete

## Deliverables Completed
- Updated git repository add/edit UI state and workflows in [`pkg/web/ui/index.html`](/home/jeff/skillserver/pkg/web/ui/index.html):
  - Auth mode selector (`none`, `https_token`, `https_basic`, `ssh_key`)
  - Credential source selector (`env`, `file`, `stored` when capability is enabled)
  - Mode/source-specific env/file reference inputs
  - Write-only stored-secret inputs for HTTPS token/basic and SSH key flows
- Added runtime capability-aware stored-secret UI gating by loading `/api/runtime/capabilities` and repo response capability hints.
- Updated add/update request payload construction in the web UI so:
  - public flow remains URL-only on create
  - private flows include structured `auth` and optional `stored_credential` payloads
  - edit flows preserve existing server-side secret material unless new values are explicitly provided
- Added per-repo UI status rendering in the repo list:
  - auth summary badge
  - credential readiness badge (`configured` / `missing` / `not required`)
  - sync status badge from `last_sync_status`
  - redacted sync error text from `last_sync_error`
- Added Playwright coverage in [`tests/playwright/wp007-ui-private-repo-credentials.spec.ts`](/home/jeff/skillserver/tests/playwright/wp007-ui-private-repo-credentials.spec.ts) for:
  - URL-only public onboarding payload
  - auth mode/source switching + env/file validation behavior
  - stored-secret capability-aware edit flow with masked values
  - regression checks for sync/toggle/delete actions

## Acceptance Criteria Check
- [x] UI never displays previously stored secret values after save or reload.
- [x] UI communicates masked configured state for stored credentials.
- [x] Sync status and redacted errors are visible without exposing credential values.
- [x] Public repos can still be created with only a URL.
- [x] Stored-secret inputs are hidden when runtime capability is not advertised.

## Test Evidence
- `npm run test:playwright -- tests/playwright/wp007-ui-private-repo-credentials.spec.ts`
- `go test ./pkg/web -count=1`

## Effort and Notes
- Estimated effort: 4 hours
- Actual effort: approximately 4 hours
- No blockers encountered.
