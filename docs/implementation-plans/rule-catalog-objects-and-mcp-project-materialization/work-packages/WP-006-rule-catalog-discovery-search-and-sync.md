## WP-006: Rule Catalog Discovery, Search, and Sync

### Metadata

```yaml
WP_ID: WP-006
Title: Rule Catalog Discovery, Search, and Sync
Domain: Domain Layer
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-08
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Once `rule` exists as a classifier and persistence can store it, the catalog builder, search index, and sync/effective model need to treat rules as first-class items.

**Scope:**
- Extend catalog synthesis in `pkg/domain/manager_catalog.go` to emit rule items from direct and imported resources.
- Extend search/index behavior so `classifier=rule` works everywhere classifier filters already exist.
- Ensure sync/effective logic preserves rule mutability and persistence semantics without breaking skills/prompts.

Excluded:
- REST/MCP/UI adapters that present or act on rule items.
- File-write materialization logic (WP-007).

**Success Criteria:**
- [ ] Rule items are returned from unified catalog list/search flows.
- [ ] Search index and sync logic remain stable for existing skill/prompt behavior.
- [ ] Imported rule resources obey existing repo-boundary protections.

---

### Technical Requirements

**Input Contracts:**
- Rule classifier and metadata helpers from WP-003.
- Runtime rule config from WP-004.
- Persistence classifier migration from WP-005.

**Output Contracts:**
- Updated catalog builder, searcher, and sync/effective logic in `pkg/domain/`.
- Domain tests covering rule discovery, filter behavior, and persistence sync integration.

**Integration Points:**
- WP-007 uses rule-aware catalog items for target planning.
- WP-008, WP-009, and WP-010 consume rule-aware catalog APIs.

---

### Deliverables

**Code Deliverables:**
- [ ] Update `pkg/domain/manager_catalog.go` to synthesize `rule` catalog items.
- [ ] Update `pkg/domain/search.go` and related helpers to index and query `rule`.
- [ ] Update sync/effective services so `rule` rows round-trip with correct mutability.

**Test Deliverables:**
- [ ] Add domain tests for direct and imported rule discovery.
- [ ] Add search tests for `classifier=rule`.
- [ ] Add sync/effective tests proving rule rows persist and rebuild correctly.

---

### Acceptance Criteria

**Functional:**
- [ ] Rule items appear in unified catalog list/search results when enabled.
- [ ] `listCatalog`/`searchCatalog` classifier filtering accepts `rule`.
- [ ] Existing skill/prompt discovery and ranking behavior remain unchanged.

**Testing:**
- [ ] Domain tests cover direct rule files, imported rule files, disabled-rule mode, and false-positive rejection.
- [ ] Sync/index tests cover persistence-enabled and non-persistence-enabled rebuild paths.

---

### Dependencies

**Blocked By:**
- WP-003
- WP-004
- WP-005

**Blocks:**
- WP-007
- WP-008
- WP-009
- WP-010
- WP-011
- WP-012

**Parallel Execution:**
- Can run in parallel with: none
- Cannot run in parallel with: WP-004, WP-005

---

### Risks

**Risk 1: Rule discovery duplicates existing prompt or reference items**
- Probability: Medium
- Impact: Medium
- Mitigation: Reuse canonical path and ID helpers plus classifier-specific detection rules.

**Risk 2: Search/index changes regress existing catalog behavior**
- Probability: Medium
- Impact: High
- Mitigation: Extend classifier-aware tests for skills/prompts alongside new rule cases.
