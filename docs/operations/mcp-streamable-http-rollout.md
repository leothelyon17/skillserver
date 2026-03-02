# MCP Streamable HTTP Rollout Runbook

## Purpose
Deterministic rollout and rollback procedure for ADR-001 MCP transport changes.

## References
- ADR: [ADR-001: Add Streamable HTTP Transport for MCP](/home/jeff/skillserver/docs/adrs/001-mcp-streamable-http-transport.md)
- Runtime options and troubleshooting: [README.md](/home/jeff/skillserver/README.md)
- Integration evidence:
  - `go test ./pkg/web -run 'TestMCPHTTP_' -count=1` (WP-005)
  - `go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1` (WP-006)
  - `go test ./cmd/skillserver -run 'TestRuntime_BothMode(StdioExitKeepsHTTP|SignalShutdown)' -count=1` (WP-006)

## Preconditions
- WP-005, WP-006, and WP-007 artifacts are merged for the release candidate commit.
- Deployment perimeter enforces TLS and authentication for the `/mcp` endpoint.
- On-call owner and rollback owner are assigned for the rollout window.

## Required Runtime Configuration Review
Confirm release uses the expected MCP config surface from ADR-001/README:
- `SKILLSERVER_MCP_TRANSPORT` (`stdio|http|both`)
- `SKILLSERVER_MCP_HTTP_PATH` (default `/mcp`)
- `SKILLSERVER_MCP_SESSION_TIMEOUT` (default `30m`)
- `SKILLSERVER_MCP_STATELESS` (default `false`)
- `SKILLSERVER_MCP_ENABLE_EVENT_STORE` (default `true`)
- `SKILLSERVER_MCP_EVENT_STORE_MAX_BYTES` (default `10485760`)

## Pre-Deploy Checklist
- [ ] Verify candidate image/binary was built from a commit where WP-005/WP-006 tests passed.
- [ ] Confirm deployment config sets explicit transport mode and HTTP path.
- [ ] Confirm `/mcp` route is registered ahead of UI catch-all route in current release.
- [ ] Confirm rollback override is prepared: `SKILLSERVER_MCP_TRANSPORT=stdio`.
- [ ] Confirm ingress change procedure is prepared to remove `/mcp` route.
- [ ] Perform dry-run walkthrough with at least one operator and capture sign-off.

## `/mcp` Smoke Test Commands
Run against canary endpoint before full rollout.

```bash
set -euo pipefail

MCP_ENDPOINT="https://<host>/mcp"
MCP_PROTOCOL_VERSION="2025-06-18"

# 0) Route sanity: /mcp is bound to MCP handler (not UI)
OPTIONS_STATUS=$(curl -sS -o /tmp/mcp-options-body.txt -w "%{http_code}" -X OPTIONS "$MCP_ENDPOINT")
echo "OPTIONS status: $OPTIONS_STATUS"
grep -qi "streamable MCP servers support GET, POST, and DELETE requests" /tmp/mcp-options-body.txt
test "$OPTIONS_STATUS" = "405"

# 1) Initialize session
INIT_HEADERS=$(mktemp)
INIT_STATUS=$(curl -sS -D "$INIT_HEADERS" -o /tmp/mcp-init-body.txt -w "%{http_code}" -X POST "$MCP_ENDPOINT" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "MCP-Protocol-Version: ${MCP_PROTOCOL_VERSION}" \
  --data '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"rollout-smoke","version":"1.0.0"}}}')
echo "Initialize status: $INIT_STATUS"
test "$INIT_STATUS" = "200"

SESSION_ID=$(awk -F': ' 'tolower($1)=="mcp-session-id" {gsub("\r","",$2); print $2}' "$INIT_HEADERS")
test -n "$SESSION_ID"

# 2) List tools
LIST_STATUS=$(curl -sS -o /tmp/mcp-tools-list.txt -w "%{http_code}" -X POST "$MCP_ENDPOINT" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "MCP-Protocol-Version: ${MCP_PROTOCOL_VERSION}" \
  -H "Mcp-Session-Id: ${SESSION_ID}" \
  --data '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
echo "tools/list status: $LIST_STATUS"
test "$LIST_STATUS" = "200"
grep -q '"list_skills"' /tmp/mcp-tools-list.txt

# 3) Call list_skills
CALL_STATUS=$(curl -sS -o /tmp/mcp-list-skills.txt -w "%{http_code}" -X POST "$MCP_ENDPOINT" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "MCP-Protocol-Version: ${MCP_PROTOCOL_VERSION}" \
  -H "Mcp-Session-Id: ${SESSION_ID}" \
  --data '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_skills","arguments":{}}}')
echo "tools/call list_skills status: $CALL_STATUS"
test "$CALL_STATUS" = "200"
grep -q '"skills"' /tmp/mcp-list-skills.txt

# 4) Close session
CLOSE_STATUS=$(curl -sS -o /tmp/mcp-close-body.txt -w "%{http_code}" -X DELETE "$MCP_ENDPOINT" \
  -H "Mcp-Session-Id: ${SESSION_ID}")
echo "DELETE status: $CLOSE_STATUS"
test "$CLOSE_STATUS" = "204"
```

Expected result: all checks succeed with no HTML payloads returned from `/mcp`.

## Stdio Compatibility Smoke Commands
Run on the candidate commit before production rollout:

```bash
go test ./pkg/mcp -run 'TestMCPServer_StdioRegression' -count=1
go test ./cmd/skillserver -run 'TestRuntime_BothMode(StdioExitKeepsHTTP|SignalShutdown)' -count=1
```

If rollout environment does not have Go toolchain/source checkout, use CI evidence for the exact release commit and treat missing evidence as no-go.

## Canary Rollout Checklist
- [ ] Deploy release to canary slice.
- [ ] Execute `/mcp` smoke command sequence successfully.
- [ ] Execute stdio compatibility smoke checks (or verify equivalent CI evidence).
- [ ] Monitor canary for at least 30 minutes.
- [ ] Capture metrics and compare with gates below.

## Go/No-Go Gates (Canary -> Full Rollout)
All gates must pass:
- `/mcp` smoke success rate >= 99% across canary validation window.
- `/mcp` HTTP 5xx rate <= 1%.
- Route conflict incidents = 0 (no HTML/UI payload observed on `/mcp`).
- Stdio regression checks pass for the release commit.
- No critical reliability incidents (process crash loops, failed shutdown behavior).
- Process memory trend remains stable (no sustained >20% growth vs canary baseline over 30 minutes).

Any gate failure is immediate no-go and triggers rollback.

## Operational Observability Checks
Track during canary and first 2-4 hours after full rollout:
- MCP request success/error rates for `POST /mcp` and `DELETE /mcp`.
- Session lifecycle anomalies (`session not found`, repeated re-initialize patterns).
- Process RSS memory trend (event-store pressure indicator).
- Container/process restart count.
- Ingress/authz logs for unexpected unauthenticated `/mcp` requests.

## Full Rollout Checklist
- [ ] Canary gates passed and explicitly approved by rollout owner.
- [ ] Deploy to full production scope.
- [ ] Re-run `/mcp` smoke commands after full deployment.
- [ ] Monitor elevated alerting for 2-4 hours.
- [ ] Record final rollout decision and evidence links.

## Rollback Criteria
Rollback immediately if any of these occur:
- `/mcp` initialize/list/call/close flow cannot sustain >=99% success.
- `/mcp` route conflict appears (UI/HTML payload or incorrect handler behavior).
- Stdio compatibility regresses for release commit.
- Memory/stability degradation threatens service reliability.
- Required perimeter security controls for `/mcp` are missing or ineffective.

## Hard Rollback Sequence
Execute in this exact order:
1. Set `SKILLSERVER_MCP_TRANSPORT=stdio`.
2. Redeploy application.
3. Remove `/mcp` ingress routing.

Then verify:
- [ ] `DELETE/POST/GET /mcp` are no longer externally routed.
- [ ] Stdio compatibility smoke checks pass.
- [ ] Incident ticket/run log includes trigger, timeline, and evidence.

## Post-Rollout / Post-Rollback Closeout
- [ ] Confirm ADR-001 success metrics and implementation-plan success criteria are met (or document which failed).
- [ ] Attach smoke command output and observability snapshots to release record.
- [ ] Update work-package completion summary with final results and follow-ups.
