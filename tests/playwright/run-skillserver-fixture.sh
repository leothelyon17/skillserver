#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SKILLS_DIR="$ROOT_DIR/tmp/playwright-skills"

rm -rf "$SKILLS_DIR"
mkdir -p "$SKILLS_DIR"

mkdir -p "$SKILLS_DIR/legacy-skill/scripts" "$SKILLS_DIR/legacy-skill/references" "$SKILLS_DIR/legacy-skill/assets"
cat >"$SKILLS_DIR/legacy-skill/SKILL.md" <<'EOF'
---
name: legacy-skill
description: Legacy skill fixture for UI regression tests
---
# Legacy Skill

This fixture validates the legacy three-group UI rendering.
EOF
echo "echo legacy" >"$SKILLS_DIR/legacy-skill/scripts/legacy.sh"
echo "# Legacy Reference" >"$SKILLS_DIR/legacy-skill/references/guide.md"
printf '\x89PNG\r\n\x1a\n' >"$SKILLS_DIR/legacy-skill/assets/logo.png"

mkdir -p "$SKILLS_DIR/additive-skill/scripts" "$SKILLS_DIR/additive-skill/references" "$SKILLS_DIR/additive-skill/assets" "$SKILLS_DIR/additive-skill/prompts" "$SKILLS_DIR/additive-skill/shared"
cat >"$SKILLS_DIR/additive-skill/SKILL.md" <<'EOF'
---
name: additive-skill
description: Additive skill fixture for UI regression tests
---
# Additive Skill

Use [Shared Context](shared/context.md) while following [System Prompt](prompts/system.md).
EOF
echo "echo additive" >"$SKILLS_DIR/additive-skill/scripts/additive.sh"
echo "# Additive Reference" >"$SKILLS_DIR/additive-skill/references/guide.md"
printf '\x89PNG\r\n\x1a\n' >"$SKILLS_DIR/additive-skill/assets/logo.png"
echo "You are a helpful assistant." >"$SKILLS_DIR/additive-skill/prompts/system.md"
echo "Shared imported context." >"$SKILLS_DIR/additive-skill/shared/context.md"

cd "$ROOT_DIR"
exec go run ./cmd/skillserver --dir "$SKILLS_DIR" --port 18080 --mcp-transport http
