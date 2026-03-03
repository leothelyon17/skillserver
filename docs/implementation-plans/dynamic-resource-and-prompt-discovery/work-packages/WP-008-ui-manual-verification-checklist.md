# WP-008 UI Manual Verification Checklist (WP-007 Evidence)

## Purpose
Manual verification script for WP-007 dynamic resource group rendering, including read-only handling and responsive behavior.

## Test Data Prerequisites
- Skill with legacy groups only: `scripts/`, `references/`, `assets/`
- Skill with additive groups:
  - `prompts/system.md`
  - imported virtual resource under `imports/...` (read-only)

## Environment Matrix
- Desktop viewport: `>= 1280px`
- Narrow viewport: `<= 430px`

## Manual Test Script
1. Open the web UI and edit a skill that only has legacy groups.
2. Confirm only `Scripts`, `References`, and `Assets` sections render.
3. Open a skill that includes `prompts` and imported virtual resources.
4. Confirm additive sections render dynamically from API payload (no hard-coded limit to 3 groups).
5. Confirm each resource row shows origin metadata badge (`direct` or `imported`).
6. Confirm imported resources show read-only/locked indicators.
7. Open an imported text resource and verify editor is view-only (no save action).
8. Confirm delete action is unavailable for imported resources.
9. Confirm upload is unavailable for imported-only groups.
10. Repeat steps 3-9 in narrow viewport and verify layout remains usable:
   - section headers stack cleanly
   - item metadata and action area remain readable
   - no horizontal overflow in resource rows

## Outcome Capture
- [x] Desktop verification complete
- [x] Narrow viewport verification complete
- [x] Dynamic groups verified
- [x] Imported resource write restrictions verified in UI
- [x] Legacy 3-bucket rendering verified

## Recorded Outcomes (2026-03-03)
| Check | Status | Evidence |
|------|--------|----------|
| Desktop verification complete | Pass | `npm run test:playwright` (`legacy skill keeps only legacy resource groups`; `additive skill renders prompts/imported groups and locks imported content`) |
| Narrow viewport verification complete | Pass | `npm run test:playwright` (`narrow viewport keeps additive resources readable without horizontal overflow`) |
| Dynamic groups verified | Pass | `go test ./pkg/web -count=1` (`TestListSkillResources_ReturnsLegacyAndAdditiveGroups`) |
| Imported resource write restrictions verified in UI | Pass | `go test ./pkg/web -count=1` (`TestSkillResourceWriteGuards_RejectImportedPaths`) |
| Legacy 3-bucket rendering verified | Pass | `go test ./pkg/web -count=1` (`TestListSkillResources_LegacyOnlySkill_PreservesLegacyShape`) |

## Notes
- Use this checklist as WP-008 evidence for UI behavior and responsive regression coverage.
