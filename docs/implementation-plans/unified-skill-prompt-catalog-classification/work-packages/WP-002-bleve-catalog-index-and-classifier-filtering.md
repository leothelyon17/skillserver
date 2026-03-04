## WP-002: Bleve Catalog Index and Classifier Filtering

### Metadata

```yaml
WP_ID: WP-002
Title: Bleve Catalog Index and Classifier Filtering
Domain: Domain Layer
Priority: High
Estimated_Effort: 5 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
`Searcher` currently indexes only skills (`name`, `content`, metadata). ADR-003 requires indexing both skills and prompts with filterable `classifier` field.

**Scope:**
- Extend Bleve indexing to support catalog documents with classifier field.
- Add query support for optional classifier filter in search execution.
- Preserve existing skill search behavior for `/api/skills/search` and MCP skill search tools.

Excluded:
- Catalog item discovery/synthesis from manager resources (WP-003).
- API endpoint changes (WP-004).

**Success Criteria:**
- [ ] Index contains classifier field for all catalog docs.
- [ ] Search can return all catalog items or classifier-filtered subsets.
- [ ] Skill-only search behavior remains backward compatible.

---

### Technical Requirements

**Input Contracts:**
- Catalog model from WP-001.
- Existing Bleve search/index code in `pkg/domain/search.go`.

**Output Contracts:**
- New searcher methods for catalog indexing/search.
- Backward-compatible wrappers for existing skill indexing/search calls.

**Integration Points:**
- WP-003 invokes catalog index rebuild entrypoint.
- WP-004 and WP-007 call classifier-filterable search methods.

---

### Deliverables

**Code Deliverables:**
- [ ] Extend `pkg/domain/search.go` with catalog indexing method(s).
- [ ] Add classifier-aware search method (`query + optional classifier`).
- [ ] Maintain compatibility path for existing `SearchSkills` calls.

**Test Deliverables:**
- [ ] Add domain search tests for classifier filters and mixed result sets.
- [ ] Add regression tests for existing skill-only behavior.

---

### Acceptance Criteria

**Functional:**
- [ ] Query without classifier returns mixed skill/prompt docs.
- [ ] Query with `classifier=skill` excludes prompts.
- [ ] Query with `classifier=prompt` excludes skills.

**Testing:**
- [ ] Search tests cover empty query guards, mixed indexing, and compatibility behavior.

---

### Dependencies

**Blocked By:**
- WP-001

**Blocks:**
- WP-003
- WP-004
- WP-007
- WP-008

**Parallel Execution:**
- Can run in parallel with: none before WP-001
- Cannot run in parallel with: WP-001

---

### Risks

**Risk 1: Query construction mismatch between skill and catalog modes**
- Probability: Medium
- Impact: Medium
- Mitigation: Keep shared query builder paths with explicit tests per mode.

**Risk 2: Index rebuild latency increase on large repos**
- Probability: Medium
- Impact: Medium
- Mitigation: Benchmark rebuild path and cap result sizes where required.
