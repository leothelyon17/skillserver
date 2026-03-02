## WP-001: Runtime MCP Config Contract

### Metadata

```yaml
WP_ID: WP-001
Title: Runtime MCP Config Contract
Domain: Configuration
Priority: High
Estimated_Effort: 4 hours
Status: DEFINED
Assigned_To: Unassigned
Created_Date: 2026-03-02
Started_Date: Not started
Completed_Date: Not completed
```

---

### Description

**Context:**
Runtime transport behavior must be configurable so deployments can choose stdio, HTTP, or both modes without code changes.

**Scope:**
- Add MCP transport-related config model and parser.
- Add env + flag support with precedence rules.
- Add strict validation and fail-fast errors for invalid values.

Excluded:
- Runtime orchestration logic and web route wiring (covered by later WPs).

**Success Criteria:**
- [ ] All new config fields are parsed and validated.
- [ ] Defaults resolve to ADR-selected values.
- [ ] Invalid inputs produce actionable startup errors.

---

### Technical Requirements

**Input Contracts:**
- Existing env/flag parsing in `cmd/skillserver/main.go`.
- Existing helper functions (`getEnvOrDefault`, `getEnvBool`, etc.).

**Output Contracts:**
- New config struct and parser in `cmd/skillserver/config.go`.
- Tests in `cmd/skillserver/config_test.go`.
- Clear precedence contract: flags > env > defaults.

**Integration Points:**
- WP-002 and WP-004 consume parsed config.

---

### Deliverables

**Code Deliverables:**
- [ ] Add `cmd/skillserver/config.go` for MCP runtime config parsing.
- [ ] Add transport mode enum/validator: `stdio|http|both`.
- [ ] Add parser for path, duration, booleans, and max-bytes.
- [ ] Add `cmd/skillserver/config_test.go` table tests.

**Documentation Deliverables:**
- [ ] Config defaults documented in test fixtures/comments.

---

### Acceptance Criteria

**Functional:**
- [ ] Default config resolves to:
  - transport=`both`
  - path=`/mcp`
  - session timeout=`30m`
  - stateless=`false`
  - event store enabled=`true`
  - max bytes=`10485760`
- [ ] Invalid transport mode fails fast.
- [ ] Invalid MCP path (not absolute) fails fast.
- [ ] Invalid timeout/max-bytes fails fast.

**Testing:**
- [ ] Unit tests cover defaults, env-only, flag override, and invalid inputs.

---

### Testing Strategy

**Unit Tests:**
- `TestMCPConfig_Defaults`
- `TestMCPConfig_EnvOverrides`
- `TestMCPConfig_FlagPrecedence`
- `TestMCPConfig_InvalidTransport`
- `TestMCPConfig_InvalidPath`
- `TestMCPConfig_InvalidSessionTimeout`
- `TestMCPConfig_InvalidEventStoreMaxBytes`

---

### Dependencies

**Blocked By:**
- None

**Blocks:**
- WP-002
- WP-004
- WP-007

**Parallel Execution:**
- Can run in parallel with: none (first package)
- Cannot run in parallel with: WP-002/004 (they consume this config)

---

### Risks

**Risk 1: Ambiguous precedence behavior**
- Probability: Medium
- Impact: Medium
- Mitigation: Explicit precedence tests for each field.

**Risk 2: Backward compatibility drift**
- Probability: Low
- Impact: Medium
- Mitigation: Keep old config keys untouched and additive only.

---

### Notes

This package is configuration-only and must not include runtime start/stop behavior changes.
