## WP-003: Manager Catalog Builder and Rebuild Integration

### Metadata

```yaml
WP_ID: WP-003
Title: Manager Catalog Builder and Rebuild Integration
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
Catalog indexing requires a deterministic item builder that emits both skill and prompt items from current skill/resource discovery, including imported virtual resources.

**Scope:**
- Build manager-level catalog generation from `ListSkills` + `ListSkillResources`.
- Emit one `skill` catalog item per skill and one `prompt` item per qualifying prompt resource.
- Deduplicate prompt items with stable canonical keys.
- Rewire rebuild flow so sync-triggered reindex uses unified catalog docs.

Excluded:
- API route/controller integration (WP-004).
- UI behavior updates (WP-005).

**Success Criteria:**
- [ ] Rebuild emits deterministic catalog document set.
- [ ] Prompt catalog items include parent skill ID and resource path metadata.
- [ ] Existing git sync callback rebuild continues to work.

---

### Technical Requirements

**Input Contracts:**
- Catalog model from WP-001.
- Searcher catalog indexing from WP-002.
- Existing resource discovery in `pkg/domain/manager.go`.

**Output Contracts:**
- Manager methods for listing/searching catalog items.
- Rebuild flow that indexes catalog documents.

**Integration Points:**
- Consumed by WP-004 and WP-007 endpoint/tool handlers.
- Validated by WP-008 integration tests.

---

### Deliverables

**Code Deliverables:**
- [ ] Add manager catalog-builder helper(s) in `pkg/domain/manager.go` or adjacent file.
- [ ] Extend manager/search interfaces with catalog list/search methods (additive).
- [ ] Ensure prompt item IDs and dedupe keys are stable and canonical.

**Test Deliverables:**
- [ ] Add manager tests for mixed catalog output and dedupe behavior.
- [ ] Add git-backed import prompt coverage in manager tests.

---

### Acceptance Criteria

**Functional:**
- [ ] Prompt items are emitted for direct and imported markdown prompt resources.
- [ ] Non-prompt resources are not emitted as prompt catalog items.
- [ ] Catalog rebuild is deterministic across repeated runs.

**Testing:**
- [ ] Manager tests validate ordering, dedupe, and git import scenarios.

---

### Dependencies

**Blocked By:**
- WP-001
- WP-002

**Blocks:**
- WP-004
- WP-005
- WP-007
- WP-008

**Parallel Execution:**
- Can run in parallel with: none before WP-002
- Cannot run in parallel with: WP-001, WP-002

---

### Risks

**Risk 1: Dedupe logic removes legitimate prompt variants**
- Probability: Medium
- Impact: Medium
- Mitigation: Dedupe only by canonical target path + parent skill key.

**Risk 2: Rebuild path regresses existing skill indexing behavior**
- Probability: Low
- Impact: High
- Mitigation: Preserve skill-only compatibility tests and add migration regression tests.
