## WP-001: Catalog Contract and Classifier Rules

### Metadata

```yaml
WP_ID: WP-001
Title: Catalog Contract and Classifier Rules
Domain: Domain Layer
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
The codebase currently models only skill entities for search and listing. ADR-003 requires a unified catalog model with an explicit classifier so prompts become first-class objects.

**Scope:**
- Define catalog domain model (`CatalogItem`) with explicit classifier enum (`skill`, `prompt`).
- Define classifier helper rules for markdown prompt detection (`agent|agents|prompt|prompts`).
- Add deterministic ID and canonical key helpers for prompt and skill catalog items.

Excluded:
- Bleve query/index implementation details (WP-002).
- Manager catalog generation flow (WP-003).

**Success Criteria:**
- [ ] Catalog contracts are defined and compile cleanly.
- [ ] Classifier helper correctly identifies prompt candidates from path + extension.
- [ ] Stable IDs are deterministic across rebuilds.

---

### Technical Requirements

**Input Contracts:**
- Existing `Skill`, `SkillResource`, and resource origin/type models in `pkg/domain/`.

**Output Contracts:**
- New catalog model and classifier helpers in `pkg/domain`.
- Unit tests for classifier and ID stability.

**Integration Points:**
- WP-002 uses catalog contract for indexing.
- WP-003 uses helper functions during catalog item synthesis.

---

### Deliverables

**Code Deliverables:**
- [ ] Add `CatalogClassifier` and `CatalogItem` types (new file in `pkg/domain`, e.g. `catalog.go`).
- [ ] Add classifier helper for prompt detection using configurable directory allowlist.
- [ ] Add deterministic ID builders for skill and prompt catalog items.

**Test Deliverables:**
- [ ] Add/extend tests in `pkg/domain` for classifier rules and stable IDs.

---

### Acceptance Criteria

**Functional:**
- [ ] `SKILL.md` is always `skill` classifier.
- [ ] Markdown files in allowed prompt directories classify as `prompt`.
- [ ] Non-markdown files are not classified as prompt catalog items.

**Testing:**
- [ ] Domain tests validate happy path and edge cases (nested segments, imported paths, extension mismatch).

---

### Dependencies

**Blocked By:**
- None

**Blocks:**
- WP-002
- WP-003
- WP-006

**Parallel Execution:**
- Can run in parallel with: none (foundation package)
- Cannot run in parallel with: WP-002, WP-003

---

### Risks

**Risk 1: Overly broad classifier rules create false prompt positives**
- Probability: Medium
- Impact: Medium
- Mitigation: Require markdown extension + bounded segment match.

**Risk 2: Unstable ID generation creates duplicate index docs**
- Probability: Medium
- Impact: Low
- Mitigation: Use canonicalized path keys and deterministic ID format.
