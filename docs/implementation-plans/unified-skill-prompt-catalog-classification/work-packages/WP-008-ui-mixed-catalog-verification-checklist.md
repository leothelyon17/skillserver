# WP-008 UI Mixed Catalog Verification Checklist

## Purpose

Validate mixed catalog rendering, classifier badges, and non-regression of existing skill/resource UI flows before WP-009 rollout sign-off.

## Environment Matrix

- Fixture server: `tests/playwright/run-skillserver-fixture.sh`
- Base URL: `http://127.0.0.1:18080`
- Browser: Playwright Chromium
- Desktop viewport: `1280x900`
- Narrow viewport: `390x844`

## Verification Checklist

- [x] Mixed skill + prompt tiles render from `/api/catalog`.
- [x] Every catalog tile renders a classifier badge (`skill` or `prompt`).
- [x] Prompt tiles open read-only view with parent-skill guidance.
- [x] Catalog search returns prompt hits by prompt content.
- [x] Catalog search returns skill hits by skill metadata/content.
- [x] Skill edit flow remains functional from catalog tile.
- [x] Skill create flow remains functional.
- [x] Skill delete flow remains functional.
- [x] Legacy resource groups (`Scripts`, `References`, `Assets`) remain stable.
- [x] Additive resource groups (`Prompts`, `Imported`) render only when present.
- [x] Imported resources remain locked/read-only in UI.
- [x] Narrow viewport has no horizontal overflow in resource rows/headers.

## Automated Evidence

### Commands

```bash
npx playwright test tests/playwright/wp005-ui-catalog.spec.ts tests/playwright/wp008-ui.spec.ts --project=chromium
```

### Results

- `tests/playwright/wp005-ui-catalog.spec.ts`: pass (`3/3`)
- `tests/playwright/wp008-ui.spec.ts`: pass (`3/3`)
- Combined run: pass (`6/6`)

## Notes

- The WP-005 suite remains the primary mixed catalog tile + badge regression signal.
- The WP-008 suite extends regression coverage for resource-tab grouping, lock state, and responsive behavior.
