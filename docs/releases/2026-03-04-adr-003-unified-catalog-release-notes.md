# Release Notes: ADR-003 Unified Skill/Prompt Catalog Classification

**Release Date:** 2026-03-04
**ADR:** [ADR-003: Unified Skill/Prompt Catalog Classification for Git Imports](/home/jeff/skillserver/docs/adrs/003-unified-skill-prompt-catalog-classification.md)

## Summary
This release adds a unified catalog model where both skills and prompt markdown files are first-class searchable items with explicit classifier metadata (`skill` or `prompt`).

## Added
- New additive REST endpoints:
  - `GET /api/catalog`
  - `GET /api/catalog/search?q=...&classifier=skill|prompt`
- New additive MCP tools:
  - `list_catalog`
  - `search_catalog`
- New runtime controls:
  - `SKILLSERVER_CATALOG_ENABLE_PROMPTS` / `--catalog-enable-prompts`
  - `SKILLSERVER_CATALOG_PROMPT_DIRS` / `--catalog-prompt-dirs`

## Compatibility Statement
This change is backward-compatible.
- Existing `/api/skills` and `/api/skills/search` contracts are preserved.
- Existing MCP skill tools (`list_skills`, `read_skill`, `search_skills`) remain supported.
- Existing resource APIs and resource MCP tools remain supported.
- Catalog endpoints/tools are additive and can be adopted incrementally by clients.

## Rollback Summary
- Prompt classification kill switch: set `SKILLSERVER_CATALOG_ENABLE_PROMPTS=false` (or `--catalog-enable-prompts=false`) and redeploy.
- For prompt directory misclassification, restore default allowlist with `SKILLSERVER_CATALOG_PROMPT_DIRS=agent,agents,prompt,prompts`.

Detailed rollout/rollback procedure: [Unified Skill/Prompt Catalog Rollout Runbook](/home/jeff/skillserver/docs/operations/unified-catalog-rollout-rollback.md)
