## WP-006: Runtime Config for Prompt Catalog Detection

### Metadata

```yaml
WP_ID: WP-006
Title: Runtime Config for Prompt Catalog Detection
Domain: Infrastructure
Priority: Medium
Estimated_Effort: 3 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-04
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
ADR-003 proposes additive runtime controls for prompt catalog behavior. Current runtime supports import-discovery toggles but not catalog prompt allowlist controls.

**Scope:**
- Add env/flag support for prompt catalog enablement and prompt directory allowlist.
- Wire parsed config into catalog classification logic.
- Keep defaults aligned with ADR (`agent,agents,prompt,prompts`).

Excluded:
- Core classifier/search logic implementation (WP-001, WP-002).

**Success Criteria:**
- [ ] Runtime exposes additive config knobs for prompt catalog behavior.
- [ ] Defaults are safe and ADR-aligned.
- [ ] Invalid config values fail fast with clear errors.

---

### Technical Requirements

**Input Contracts:**
- Existing runtime config and flag parsing in `cmd/skillserver`.

**Output Contracts:**
- New env/flag parsing and validated settings injection into manager/search catalog builder.

**Integration Points:**
- Influences behavior of WP-001 classifier helper and WP-003 builder.
- Validated by WP-008 config regression tests.

---

### Deliverables

**Code Deliverables:**
- [ ] Add env vars + flags for prompt catalog controls in `cmd/skillserver/main.go` and related config helpers.
- [ ] Pass effective config into manager/catalog classifier components.
- [ ] Add startup log/debug output for effective prompt directory allowlist.

**Test Deliverables:**
- [ ] Add config parsing tests for defaults, overrides, invalid values.

---

### Acceptance Criteria

**Functional:**
- [ ] Prompt catalog detection can be toggled on/off at runtime.
- [ ] Prompt directory list is configurable and normalized.
- [ ] Invalid classifier config returns actionable startup error.

**Testing:**
- [ ] Runtime config tests validate default and override paths.

---

### Dependencies

**Blocked By:**
- WP-001

**Blocks:**
- WP-008
- WP-009

**Parallel Execution:**
- Can run in parallel with: WP-004, WP-007
- Cannot run in parallel with: WP-001

---

### Risks

**Risk 1: Misconfigured prompt dirs silently suppress expected prompt indexing**
- Probability: Medium
- Impact: Medium
- Mitigation: Normalize/validate config and surface effective values in logs.

**Risk 2: Runtime config complexity introduces startup regressions**
- Probability: Low
- Impact: Medium
- Mitigation: Maintain strict parse validation and tests for flag/env precedence.
