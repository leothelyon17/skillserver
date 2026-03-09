## WP-009: Documentation, Examples, and Release Guidance

### Metadata

```yaml
WP_ID: WP-009
Title: Documentation, Examples, and Release Guidance
Domain: documentation
Execution_Agent_Prompt:
Agent_Selection_Source: blank
Agent_Selection_Rationale: No highly relatable installed prompt for this narrowly scoped documentation update package.
Priority: Medium
Estimated_Effort: 3 hours
Status: COMPLETE
Assigned_To: Codex
Created_Date: 2026-03-09
Started_Date: 2026-03-09
Completed_Date: 2026-03-09
```

---

### Description

**Context:**
These changes alter how agents and UI users discover and operate on catalog items. Without updated examples and release notes, the contract improvements will remain difficult to adopt.

**Scope:**
- Update `README.md` for:
  - canonical skill IDs
  - explicit classification-state fields
  - new list/search filters
  - batch taxonomy mutation contracts
  - usage/preflight endpoints and tools
  - export archive-root and archive-byte options
- Update implementation docs and release guidance.
- Update test docs only where they reinforce release guidance from WP-008.

Excluded:
- New code behavior.
- Additional rollout automation beyond documentation.

**Success Criteria:**
- [x] Every additive contract change is documented with an example.
- [x] Compatibility notes for bare skill IDs are explicit.
- [x] Release guidance calls out the metadata-first and flattened-archive defaults.

---

### Technical Requirements

**Input Contracts:**
- Finalized and verified contracts from WP-005, WP-006, and WP-008.

**Output Contracts:**
- README and implementation-doc updates with concrete examples.
- Release guidance that operators and agent users can follow without reading code.

**Integration Points:**
- Should be the final package after regression gates are green.

---

### Deliverables

**Documentation Deliverables:**
- [x] Update `README.md` with REST and MCP examples.
- [x] Add examples for batch taxonomy dry-run and usage/preflight payloads.
- [x] Update the implementation plan and linked docs where file names or behavior changed during execution.
- [x] Summarize rollout notes and any remaining compatibility caveats.

**Review Deliverables:**
- [x] Confirm documented examples match the verified tests from WP-008.

---

### Acceptance Criteria

**Functional:**
- [x] Operators and agent users can discover the new contracts from documentation alone.
- [x] Documentation reflects verified behavior, not planned behavior.

**Testing:**
- [x] No standalone automated tests required.
- [x] Documentation examples are spot-checked against implemented routes/tools before closeout.

---

### Dependencies

**Blocked By:**
- WP-005
- WP-006
- WP-007
- WP-008

**Blocks:**
- None.

**Parallel Execution:**
- Can run in parallel with: None.
- Cannot run in parallel with: WP-008 because docs must reflect verified final behavior.

---

### Risks

**Risk 1: Docs describe the planned default rather than the shipped behavior**
- Probability: Medium
- Impact: High
- Mitigation: Treat WP-008 as a hard prerequisite and use verified examples only.

**Risk 2: Compatibility notes are easy to miss**
- Probability: Medium
- Impact: Medium
- Mitigation: Put canonical ID and legacy bare-ID acceptance notes directly next to the affected REST and MCP examples.
