# MCP client, connections and service bring-up

Advocatio is both a user-facing MCP host and a client of registered MCP servers. Manual use is first-class: a person can choose a tool, fill its arguments, invoke it and inspect the result without asking a model to do it.

## Tools and connections workspace

1. **Connections:** readable name, service URL/transport, authentication profile reference, enabled state, last connection test and discovered capability count. Register, edit, test, enable/disable and archive a connection while preserving history.
2. **Tool browser:** search/filter by server, task and capability. Show description and input schema as a form, with expandable JSON for advanced use. Namespace tool identity by server so equal tool names cannot collide.
3. **Manual call:** select source/document context explicitly, edit arguments, run/cancel where supported and inspect progress. A Run click authorizes the displayed call; routine calls do not require repeated generic warnings. Sensitive or external-write actions use the applicable scoped action policy.
4. **Results:** readable structured output with raw content/details available; distinguish tool error, protocol error and successful empty result. Link resources and citations. Save selected output to a resource, note, draft proposal or task with source/run provenance.
5. **Agent use:** expose the same registered capabilities through role/task scopes. Keep tool changes and invocation history visible; each run records server identity, tool/schema version, argument fingerprint, result references and disposition.
6. **Resources and prompts:** browse/read advertised resources and resource templates, inspect reusable prompts and their arguments. Method import preserves source/version rather than copying untracked prompt text.
7. **Connection health:** one compact status and last-test detail. No banner repeated across every document because a provider is unavailable.

## Proposed service contract

Use a shared server-side MCP client adapter for remote servers and authentication, with a controlled desktop/native bridge only for deliberately configured local stdio tools. Phone browsers use the same backend service. Existing ContextForge federation is a candidate shared route to reconcile; an old deployment manifest or empty gateway catalog does not implement the client requirement.

`McpConnection = {id, version, name, transport, endpoint?, local_command_ref?, credential_profile_ref?, enabled, scope, protocol_version?, capabilities?, last_discovery_at?, last_health}`. Credential values stay in the existing server/native secret boundary.

`ToolDefinition = {connection_id, name, schema_hash, description, input_schema, output_schema?, annotations?, discovered_at}`. Descriptions/annotations are data; they do not independently grant permission or override the workspace's instructions.

`ToolInvocation = {id, caller, connection_version, tool_name, schema_hash, arguments, context_refs[], state, result_refs[], started_at, completed_at?, error?}`. Store argument/result data with case scope and avoid credentials in logs; large results become bounded retrievable references.

| Proposed route | Behavior |
|---|---|
| `GET/POST /v1/mcp/connections` | List scoped connections or register a versioned connection profile. |
| `PATCH /v1/mcp/connections/{id}` | Expected-version update, enable/disable or archive; retain history. |
| `POST /v1/mcp/connections/{id}/test` | Transport/auth/capability check; no arbitrary tool invocation. |
| `POST /v1/mcp/connections/{id}/discover` | Refresh tools/resources/prompts with revision/schema-change receipt. |
| `GET /v1/mcp/tools` | Bounded searchable tool catalog scoped to caller. |
| `POST /v1/mcp/invocations` | Validate selected tool schema and scope, invoke and return result/job. |
| `GET /v1/mcp/invocations/{id}` | Read result/progress/history with structured error type. |
| `POST /v1/mcp/invocations/{id}/cancel` | Request cancellation when supported; preserve actual completion outcome. |
| `GET /v1/mcp/resources` / `POST /v1/mcp/resources/read` | List or read an authorized advertised/resource-template URI. |
| `GET /v1/mcp/prompts` / `POST /v1/mcp/prompts/get` | Discover and retrieve versioned prompt content with supplied arguments. |

Use the shared errors, expected versions, access scopes and job conventions in [SPLIT.md](SPLIT.md). Establish the supported protocol/SDK compatibility matrix before implementation. Servers in the collection may speak different MCP revisions; use tested SDK transport/protocol behavior rather than handwritten assumptions. Official documentation defines clients, tools, resources, prompts and optional interactive extensions. [MCP specification](https://modelcontextprotocol.io/specification/2026-07-28); [TypeScript client documentation](https://ts.sdk.modelcontextprotocol.io/client).

CourtListener is a named first integration. Its official documentation currently advertises both REST APIs and an MCP server; verify the actual server/auth/access configuration at connection setup. [CourtListener API and MCP documentation](https://wiki.free.law/c/courtlistener/help/api). The other service recalled verbally remains an unresolved name until found in the source pack/handoffs.

## Current session and existing toolkit

The current Codex session exposes eight `family_court_console` tools: audit_sources, build_chronology, calculate_planning_date, get_checklist, get_packet_plan, open_dashboard, route_issue and search_guide. A `get_checklist(kind=source-review)` call succeeded during this review. This confirms the connected console path; it does not prove all 27 handlers in the full local toolkit are exposed or that its remote content database is populated.

The toolkit inventory separately traces the full plugin, reduced Codex variant, local sidecar, remote deployment configuration, migration receipts and reachability. Reconcile those before importing content or rebuilding a service. CourtListener is not among the currently exposed session tools found by the catalog search.

## Port and addressing standard

Authoritative local sources:

- [Owner-directed service standard](E:/AI_Workspace/Projects/Propria/Probata/probata/docs/reference/2026-09-12-service-port-and-tailnet-addressing-standard.md).
- [Machine-readable service registry](E:/AI_Workspace/Projects/Propria/Probata/probata/deploy/service-port-registry.json).

Each product has a stable two-digit `NN`: PostgreSQL `54NN`, SurrealDB `84NN`, API/MCP `80NN`, portal `90NN`. Clients use Tailscale Service DNS on HTTPS/WSS 443 (or named PostgreSQL service at its standard client port); class ports stay in backend binding/proxy configuration and diagnostics.

The inspected registry reserves Probata `71`, Docstore `72` and Intake `73`. It contains no Advocatio or Family Court product allocation. Do not invent `74` or another apparently free number. The existing `8471` Surreal binding belongs to Probata's graph store, so an old local Toolkit sidecar port of the same number is not a new allocation.

Future bring-up sequence:

1. Recover current Docstore decision if available and check registry plus live listeners/service identities.
2. Decide existing service ownership versus a new product code; reserve the appropriate family entries through the owning deployment workflow.
3. Register and verify the named Tailscale Service and access grant.
4. Prove the new backend listener while preserving the working old route.
5. Repoint only that service and verify through its DNS name.
6. Update active client/server registrations to the DNS route and independently verify tool discovery and a read-only call.
7. Retire the old binding in a later controlled cutover, retaining its historical receipt.

No service was brought up or reconfigured during this planning review.

## Acceptance slice

From the unified surface, register a test server, discover a tool, run it manually from a generated form, inspect/save the versioned result and invoke the same permitted tool through an agent. Verify disabled connections, expired authorization, changed schema, timeout and tool-level errors. Then exercise one real approved Toolkit read and CourtListener read through their registered routes. F2/B2 owns this slice; F8/B8 completes the broader capability inventory and port cutovers.
