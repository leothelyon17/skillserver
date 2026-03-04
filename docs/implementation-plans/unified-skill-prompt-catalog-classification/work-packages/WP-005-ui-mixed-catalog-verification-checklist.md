# WP-005 UI Mixed Catalog Verification Checklist

## Scope

Validate unified catalog rendering and behavior in `pkg/web/ui/index.html` after switching UI list/search to catalog endpoints.

## Environment

- Fixture server: `tests/playwright/run-skillserver-fixture.sh`
- Base URL: `http://127.0.0.1:18080`
- Browser: Playwright Chromium

## Checklist

- [x] Catalog grid renders mixed item types from `/api/catalog`.
- [x] Every tile renders a classifier badge (`skill` or `prompt`).
- [x] Prompt tiles open in read-only prompt view mode.
- [x] Prompt view includes parent skill guidance copy.
- [x] Search (`/api/catalog/search`) returns prompt hits by prompt content.
- [x] Search returns skill hits by skill metadata/content.
- [x] Skill edit flow remains functional from catalog tiles.
- [x] Skill create flow remains functional.
- [x] Skill delete flow remains functional.
- [x] Existing resource-tab UI regressions remain green (WP-008 spec).

## Automated Evidence

### Commands

```bash
go test ./pkg/web -v
go test ./...
npx playwright test --project=chromium
```

### Results

- `go test ./pkg/web -v`: pass
- `go test ./...`: pass
- `npx playwright test --project=chromium`: pass (6/6)

## Notes

- Skill actions (`Export`, `Delete`) are guarded to skill-classifier tiles only.
- Prompt tiles are first-class catalog items and intentionally non-editable in UI.
