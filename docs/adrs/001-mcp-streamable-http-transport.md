# ADR-001: Add Streamable HTTP Transport for MCP

## Metadata

| Field | Value |
|-------|-------|
| **Status** | Proposed |
| **Date** | 2026-03-02 |
| **Author(s)** | @jeff |
| **Reviewers** | TBD |
| **Work Package** | N/A |
| **Supersedes** | N/A |
| **Superseded By** | N/A |

## Summary

SkillServer currently exposes MCP over `stdio` only, while the web UI/API run on HTTP. For remote infrastructure deployments, agents need a network-native MCP endpoint using Streamable HTTP. We will add a first-class Streamable HTTP endpoint (`/mcp`) while preserving `stdio` support behind explicit transport configuration, enabling both local and remote MCP client patterns.

## Context

### Problem Statement

Remote agent runtimes cannot reliably use local process `stdio` transport, but SkillServer's MCP server is currently started with `StdioTransport` only. This blocks direct MCP integration for remotely hosted agents and requires custom wrappers or sidecars. We need native Streamable HTTP support so agents can connect over standard HTTP(S) in production infrastructure.

### Current State

- MCP tools are registered in [`pkg/mcp/server.go`](/home/jeff/skillserver/pkg/mcp/server.go) and executed via `Server.Run(ctx)` using `mcp.StdioTransport`.
- Application startup in [`cmd/skillserver/main.go`](/home/jeff/skillserver/cmd/skillserver/main.go) launches Echo web server in a goroutine, then blocks on MCP `stdio`.
- The HTTP server in [`pkg/web/server.go`](/home/jeff/skillserver/pkg/web/server.go) serves REST/UI routes but does not expose MCP transport endpoints.
- README documents only `stdio` MCP client configuration and warns about log interference with `stdio`.

### Requirements

| Requirement | Priority | Description |
|-------------|----------|-------------|
| REQ-1 | Must Have | Support MCP Streamable HTTP endpoint for remote agents. |
| REQ-2 | Must Have | Keep existing MCP tool behavior and protocol compatibility. |
| REQ-3 | Must Have | Support concurrent MCP sessions and graceful shutdown. |
| REQ-4 | Should Have | Preserve `stdio` support for local/dev workflows. |
| REQ-5 | Should Have | Allow session timeout and stream resumption controls. |
| REQ-6 | Nice to Have | Provide transport-level observability (session counts, errors, request duration). |

### Constraints

- **Budget**: No additional paid infrastructure components required for initial rollout.
- **Timeline**: Deliverable in one implementation cycle (roughly 1-2 weeks).
- **Technical**: Current dependency already includes `github.com/modelcontextprotocol/go-sdk v1.2.0`, which includes `NewStreamableHTTPHandler`.
- **Compliance**: Remote endpoint must run behind TLS and authenticated perimeter in production.
- **Team**: Existing codebase is Go + Echo; change should fit current structure without major framework replacement.

## Decision Drivers

1. **Remote MCP Compatibility**: The server must be reachable by remote agent platforms over HTTP(S).
2. **Backward Compatibility**: Existing `stdio` clients should not be broken by default.
3. **Operational Simplicity**: Avoid introducing additional sidecars/services where possible.
4. **Security and Control**: Keep deployment-friendly hooks for auth/TLS/session management.
5. **Implementation Risk**: Prefer an approach using existing SDK primitives with minimal custom transport logic.

## Options Considered

### Option 1: Replace `stdio` with Streamable HTTP Only

**Description**: Remove `stdio` MCP runtime path and expose only Streamable HTTP at `/mcp`.

**Implementation**:
```go
handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
    return server
}, &mcp.StreamableHTTPOptions{
    SessionTimeout: 30 * time.Minute,
})
```

**Pros**:
- Simplest runtime model for remote deployments.
- Eliminates `stdio` log interference concerns.
- Clearer docs and operational model.

**Cons**:
- Breaking change for existing `stdio` clients.
- Removes easy local piping workflows.
- Forces immediate migration for all consumers.

**Estimated Effort**: M

**Cost Implications**: Low direct cost; potential migration cost for existing users.

---

### Option 2: Dual Transport with Configurable MCP Mode (Chosen)

**Description**: Add Streamable HTTP endpoint while retaining optional `stdio` mode. Introduce runtime transport config (`stdio`, `http`, `both`) and mount MCP HTTP handler at `/mcp`.

**Implementation**:
```go
// New transport mode env/flag:
// SKILLSERVER_MCP_TRANSPORT=stdio|http|both
// SKILLSERVER_MCP_HTTP_PATH=/mcp

mcpHTTP := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
    return server
}, &mcp.StreamableHTTPOptions{
    SessionTimeout: 30 * time.Minute,
    EventStore:     mcp.NewMemoryEventStore(nil),
})
```

**Pros**:
- Enables remote Streamable HTTP without breaking existing `stdio` users.
- Single binary supports local and remote deployment patterns.
- Uses upstream SDK transport implementation (lower protocol risk).

**Cons**:
- Slightly higher complexity in startup/configuration and docs.
- Need clear behavior when `both` is enabled.
- Requires careful route ordering so `/mcp` is not swallowed by UI catch-all.

**Estimated Effort**: M

**Cost Implications**: Low; no mandatory new infrastructure.

---

### Option 3: Keep SkillServer Unchanged and Deploy External MCP Bridge

**Description**: Keep current `stdio` implementation and run an additional bridge/sidecar process that translates HTTP <-> `stdio`.

**Pros**:
- Minimal changes in this repository.
- Can be trialed without touching core code.

**Cons**:
- Additional process to deploy, monitor, and secure.
- Extra latency and another failure point.
- Harder debugging due to split responsibility.

**Estimated Effort**: S in-code, M operationally

**Cost Implications**: Medium operational cost due to extra service ownership.

---

## Decision

### Chosen Option

**We will implement Option 2: Dual Transport with Configurable MCP Mode**

### Rationale

Option 2 provides the required remote Streamable HTTP capability while preserving compatibility for local `stdio` consumers. It minimizes protocol risk by relying on the SDK's built-in streamable server transport and handler, and avoids forcing users into a migration event. This option best balances delivery speed, compatibility, and long-term operability.

### Decision Matrix

| Criteria | Weight | Option 1 | Option 2 | Option 3 |
|----------|--------|----------|----------|----------|
| Remote MCP compatibility | 5 | 5 | 5 | 4 |
| Backward compatibility | 4 | 1 | 5 | 4 |
| Operational simplicity | 3 | 4 | 3 | 2 |
| Security/control posture | 3 | 4 | 4 | 2 |
| Implementation risk/effort | 2 | 4 | 3 | 2 |
| **Weighted Total** |  | **61** | **72** | **52** |

## Consequences

### Positive

- Remote agents can connect directly over MCP Streamable HTTP (`/mcp`).
- Existing `stdio` integrations can continue unchanged.
- One artifact supports both deployment topologies.
- Session timeout and event-store controls improve reliability for flaky networks.

### Negative

- Startup/config surface area increases (new env vars/flags and mode logic).
- More transport permutations to test (`stdio`, `http`, `both`).
- If misconfigured, route clashes with UI wildcard may break MCP `GET`.

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| `/mcp` GET route captured by UI wildcard | Med | High | Register MCP route before UI catch-all; add integration tests for GET/POST/DELETE. |
| Session growth from long-lived clients | Med | Med | Set sane `SessionTimeout`; expose timeout as config; monitor active sessions. |
| Transport mode confusion in deployments | Med | Med | Explicit startup log of active mode/path; README examples for each mode. |
| In-memory event retention pressure | Low | Med | Use `MemoryEventStore` max-bytes tuning; allow disabling resumption if needed. |

## Technical Details

### Architecture

```text
          Remote Agent (MCP HTTP Client)
                     |
               HTTPS /mcp
                     |
            +-------------------+
            | Ingress / LB / TLS|
            +-------------------+
                     |
            +-------------------+
            |    SkillServer    |
            |                   |
            |  Echo HTTP Server |
            |   - /api/*        |
            |   - / (UI)        |
            |   - /mcp (new)    |
            |       |           |
            |   Streamable MCP  |
            |       |           |
            |   Tool Handlers   |
            +-------------------+
                     |
              Skills directory
```

### AWS Services Involved

| Service | Purpose | Configuration |
|---------|---------|---------------|
| N/A (platform-agnostic) | Can run behind any HTTP ingress/load balancer | TLS termination and auth handled at platform edge |

### Database Changes

No schema changes.

**Migration Strategy**: N/A

### API Changes

**New Endpoints**:
- `GET /mcp` - Open/resume Streamable HTTP server-to-client event stream.
- `POST /mcp` - Send MCP JSON-RPC client messages.
- `DELETE /mcp` - Close MCP session.

**Breaking Changes**:
- None intended.
- Existing REST/UI endpoints remain unchanged.

### Configuration

```yaml
# New proposed config
SKILLSERVER_MCP_TRANSPORT: "stdio"      # stdio | http | both
SKILLSERVER_MCP_HTTP_PATH: "/mcp"
SKILLSERVER_MCP_SESSION_TIMEOUT: "30m"
SKILLSERVER_MCP_STATELESS: "false"
SKILLSERVER_MCP_ENABLE_EVENT_STORE: "true"
SKILLSERVER_MCP_EVENT_STORE_MAX_BYTES: "10485760" # 10 MiB
```

## Security Considerations

### Authentication & Authorization

- Streamable HTTP endpoint should be deployed behind authenticated ingress or service mesh policy.
- If request identity is available in context, session ownership checks in SDK help reduce session hijacking risk.

### Data Protection

- TLS in transit is mandatory for remote deployment.
- No new persistent data stores are required for baseline implementation.
- Event replay data in `MemoryEventStore` is process memory only and cleared on process restart.

## Implementation Plan

### Phase 1: Transport Configuration and Wiring (2-3 days)

- [ ] Add MCP transport mode config (`stdio|http|both`) and MCP HTTP path config.
- [ ] Refactor startup flow in [`cmd/skillserver/main.go`](/home/jeff/skillserver/cmd/skillserver/main.go) to conditionally run transports.
- [ ] Add explicit startup logs for active MCP transport mode.

### Phase 2: HTTP Endpoint Integration (2-3 days)

- [ ] Add Streamable HTTP handler creation in MCP package wrapper.
- [ ] Register `/mcp` GET/POST/DELETE routes in web server before UI wildcard in [`pkg/web/server.go`](/home/jeff/skillserver/pkg/web/server.go).
- [ ] Expose session timeout and optional event-store tuning via config.

### Phase 3: Testing (2 days)

- [ ] Unit tests for config parsing and mode selection.
- [ ] Integration tests for `/mcp` lifecycle: initialize, tool call, close.
- [ ] Regression tests ensuring existing `stdio` mode still works.

### Phase 4: Documentation and Deployment (1 day)

- [ ] Update README with Streamable HTTP client examples for remote infrastructure.
- [ ] Document recommended ingress/auth/TLS deployment pattern.
- [ ] Add troubleshooting section for session IDs, accept headers, and protocol version headers.

### Rollback Plan

1. Set `SKILLSERVER_MCP_TRANSPORT=stdio`.
2. Redeploy binary without using `/mcp`.
3. Remove `/mcp` ingress routing rule.

**Rollback criteria**:
- MCP remote clients fail to initialize reliably in production.
- Session instability materially impacts tool-call reliability.
- Security controls for exposed `/mcp` endpoint are not yet in place.

## Success Metrics

- Remote MCP clients can initialize and call `list_skills` via `/mcp` with >= 99% success.
- No regressions in existing `stdio` client workflows.
- Median MCP tool-call latency over HTTP remains within acceptable operational baseline.
- Zero critical incidents from `/mcp` route conflicts or uncontrolled session growth after rollout.

## References

- [`cmd/skillserver/main.go`](/home/jeff/skillserver/cmd/skillserver/main.go)
- [`pkg/mcp/server.go`](/home/jeff/skillserver/pkg/mcp/server.go)
- [`pkg/web/server.go`](/home/jeff/skillserver/pkg/web/server.go)
- Go MCP SDK `v1.2.0`: `NewStreamableHTTPHandler`, `StreamableHTTPOptions`, `MemoryEventStore`
