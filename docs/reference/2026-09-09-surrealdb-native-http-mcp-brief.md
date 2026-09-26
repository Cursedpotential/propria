# SurrealDB server-side HTTP MCP endpoint (`/mcp` on `surreal start`) — v3.1/3.2 docs

> Byline: Claude Code (general-purpose subagent) · Sonnet 5 · 2026-09-09
> Read-only web research. All quotes below are verbatim from the raw `.mdx` source of
> `surrealdb/docs.surrealdb.com` (fetched via `raw.githubusercontent.com`, cross-checked
> against the rendered `surrealdb.com/docs/...` pages). This is the **authoritative,
> official SurrealDB documentation** for the embedded/self-hosted MCP server — distinct
> from the hosted `mcp.surrealdb.com` (SurrealDB Cloud) docs, which I also fetched and
> excluded except where explicitly noted.

Primary source pages (all live, HTTP 200 verified 2026-09-09):
- `https://surrealdb.com/docs/build/ai-agents/mcp/embedded` — **the authoritative page**, "Embedded MCP"
  (raw source: `https://raw.githubusercontent.com/surrealdb/docs.surrealdb.com/main/src/content/build/ai-agents/mcp/embedded.mdx`)
- `https://surrealdb.com/docs/reference/cli/surrealdb-cli/commands/mcp` — `surreal mcp` CLI reference
  (raw: `.../reference/cli/surrealdb-cli/commands/mcp.mdx`)
- `https://surrealdb.com/docs/reference/cli/surrealdb-cli/environment-variables` — full env-var tables, "MCP config" section
  (raw: `.../reference/cli/surrealdb-cli/environment-variables.mdx`)
- `https://surrealdb.com/docs/build/ai-agents/mcp` — hosted `mcp.surrealdb.com` overview (Cloud-only; referenced for contrast, and for the `claude mcp add --transport http` command pattern)
- `https://surrealdb.com/docs/build/ai-agents/mcp/claude` — "MCP in Claude" (also Cloud-only, but shows the exact `claude mcp add --transport http` CLI syntax)

GitHub note: `gh api search/code` for `SURREAL_MCP_ALLOWED_HOSTS`, `SURREAL_MCP_ALLOW_ALL_HOSTS`, and `"Host header is not allowed"` against `surrealdb/surrealdb` returned **zero hits** in the main crates (only the docs repo `surrealdb/docs.surrealdb.com` matched). This is very likely a GitHub code-search indexing gap on a huge repo (or the string is built via `format!()`/macro rather than a literal), not evidence the feature is undocumented-only. The docs page carries `<Since v="v3.2.1" />` tags for both env vars, which is consistent with your server reporting 403 on v3.2.4. Treat the docs as authoritative; I could not independently confirm the exact Rust source location — STILL-UNKNOWN which file in `crates/` implements it.

---

## 1. Docs page(s) describing the built-in HTTP MCP server on `surreal start`

The canonical page is **"Embedded MCP"** at `/docs/build/ai-agents/mcp/embedded` (added `<Since v="v3.1.0" />`). Verbatim:

> "SurrealDB ships a built-in [Model Context Protocol](https://modelcontextprotocol.io) server, so agents and editors can list schema, run SurrealQL, and change records through a standard set of tools. The same access control applies as on `/sql` and RPC: `DEFINE USER` permissions, table `PERMISSIONS`, and server capability flags all decide what a tool can do."
>
> "This page covers the server inside the SurrealDB binary, for databases you run yourself. For SurrealDB Cloud, the hosted [SurrealDB MCP Server](/docs/build/ai-agents/mcp) reaches the same data tools remotely..."

It explicitly distinguishes three things (confirms your framing): `surreal mcp` (stdio), `surreal start` + `/mcp` (HTTP, this doc), and the hosted `mcp.surrealdb.com` (SurrealDB Cloud, different doc page entirely).

### Transport section, verbatim:

> "### HTTP (`/mcp`)
> When you run `surreal start`, the server exposes **`POST /mcp`** on the same bind address as the REST API. Authenticate with the same headers you use elsewhere, for example `Authorization: Bearer <jwt>` or HTTP Basic.
>
> ```bash
> surreal start --user root --pass secret --bind 127.0.0.1:8000 memory
> # MCP endpoint: http://127.0.0.1:8000/mcp
> ```
>
> For a non-loopback hostname (a public FQDN, a Kubernetes service name, or a load-balancer host), the transport rejects the request with `403 Forbidden: Host header is not allowed` until you opt in. Set `SURREAL_MCP_ALLOWED_HOSTS` to your hostnames, or `SURREAL_MCP_ALLOW_ALL_HOSTS=true` behind a trusted proxy."

This is the **exact 403 message you observed** — the docs confirm it is expected, intentional DNS-rebinding-guard behavior, not a bug, and it is documented precisely.

Release history: `/3.1` release notes describe the feature as *"A typed, structured tool surface for AI agents, exposed as the `surreal mcp` stdio subcommand and an HTTP `/mcp` endpoint guarded by the existing authentication middleware,"* with *"Twelve typed tools, with read only and destructive hints."* The 3.1 release also introduced `SURREAL_HTTP_MAX_MCP_BODY_SIZE`, `SURREAL_MCP_QUERY_TIMEOUT_SECS`, `SURREAL_MCP_MAX_RESULT_BYTES`, `SURREAL_MCP_RUN_MAX_ARGS`, `SURREAL_MCP_PARAMS_MAX_KEYS`. The host-allowlist vars (`SURREAL_MCP_ALLOWED_HOSTS`, `SURREAL_MCP_ALLOW_ALL_HOSTS`) came later, tagged `<Since v="v3.2.1" />` — i.e. they did **not** exist in 3.1.0/3.1.x and only became configurable from 3.2.1 onward. Your v3.2.4 server has them. STILL-UNKNOWN: I could not locate a dedicated 3.2.x (non-3.2.1-specific) or 3.2.4 changelog entry beyond the env-var `Since` tags — the `/releases` index and `/3.2` page did not surface distinct MCP prose beyond what's captured in the "Embedded MCP" page itself.

---

## 2. Exact flag(s)/env vars for the Host-header allowlist (DNS-rebinding guard)

From `environment-variables.mdx`, "MCP config" table, verbatim rows:

> | `SURREAL_MCP_ALLOWED_HOSTS` *(Since v3.2.1)* | Default: **loopback only (`localhost`, `127.0.0.1`, `::1`)** | Allowed values: **Comma-separated exact hostnames** | Notes: "Hostnames accepted in the HTTP `Host` header for `/mcp` (DNS-rebinding guard). A non-empty list **replaces** the loopback default, so include `localhost` yourself if you still need it. **Entries without a port match any port.** Ignored when `SURREAL_MCP_ALLOW_ALL_HOSTS` is set." |
> | `SURREAL_MCP_ALLOW_ALL_HOSTS` *(Since v3.2.1)* | Default: `false` | Allowed values: `true` / `1` to enable | Notes: "Disables the `Host`-header allowlist (accept any `Host`). Escape hatch for a trusted proxy or load balancer. **Takes precedence over** `SURREAL_MCP_ALLOWED_HOSTS`." |

**No CLI `--allow-hosts` flag exists** for this — it is env-var-only (unlike `--allow-net`/`--deny-funcs` etc., which are CLI flags). I found no flag form in either the CLI reference or the embedded-MCP page; both docs pages give only the env vars. STILL-UNKNOWN whether a flag equivalent exists — the docs never show one, and the "Configuration" table on the embedded page (which lists every other MCP flag/env pairing) lists these two as env-var-only.

### Adding your tailnet address `100.91.190.107:8471`

Per "Entries without a port match any port," the allowed-hosts list is matched against the literal `Host` header value the client sends, which for a non-standard port includes the port (`host:port`). So to allow `http://100.91.190.107:8471/mcp`, set:

```bash
SURREAL_MCP_ALLOWED_HOSTS=100.91.190.107:8471
```

(Include `localhost` too if you still want loopback access, since a non-empty list **replaces** the default: `SURREAL_MCP_ALLOWED_HOSTS=100.91.190.107:8471,localhost`.) If you'd rather match the IP on any port, docs say an entry *without* a port matches any port, so `SURREAL_MCP_ALLOWED_HOSTS=100.91.190.107` would also work for any port on that host — but the doc's own wording implies exact/full-value matching otherwise, so prefer the explicit `ip:port` form for least privilege.

### Can it be disabled?

Yes — `SURREAL_MCP_ALLOW_ALL_HOSTS=true` (or `1`) disables the allowlist entirely and takes precedence over `SURREAL_MCP_ALLOWED_HOSTS`. Docs explicitly warn this is meant only "behind a trusted proxy" / as an "escape hatch for a trusted proxy or load balancer" — not for direct exposure. Security checklist on the same page reiterates: *"On a public hostname, set `SURREAL_MCP_ALLOWED_HOSTS`, or `SURREAL_MCP_ALLOW_ALL_HOSTS` only behind a trusted proxy. The default allowlist is loopback-only."*

---

## 3. Authentication

Verbatim, "HTTP (`/mcp`)" section:

> "Authenticate with the same headers you use elsewhere, for example `Authorization: Bearer <jwt>` or HTTP Basic."

And from the table in "When to use `surreal mcp` vs `surreal start`":

> | **Authentication** | Owner-level access on every tool call - no login step *(stdio)* | Normal SurrealDB auth (Bearer JWT, HTTP Basic, …) *(HTTP)* |

So: **both root/ns/db user Basic auth AND Bearer JWT are supported**, using the exact same auth machinery as `/sql` and RPC (`DEFINE USER` permissions, table `PERMISSIONS`, capability flags). The page's own "Connect an editor" HTTP example uses Basic:

```json
{
  "mcpServers": {
    "surrealdb": {
      "url": "http://127.0.0.1:8000/mcp",
      "headers": {
        "Authorization": "Basic <base64-encoded username:password>"
      }
    }
  }
}
```

Record access (SCOPE/record users) is **not explicitly mentioned** on this page — the docs only say "Normal SurrealDB auth (Bearer JWT, HTTP Basic, …)" (note the "…"). Given record-access users authenticate the same way as any SurrealDB user (via a JWT obtained through `signin`), it's reasonable to infer a record-access JWT would work as a Bearer token like any other JWT, but the docs do **not spell this out for MCP specifically** — STILL-UNKNOWN (docs silent on record-access users explicitly against `/mcp`).

### Namespace/database selection

Verbatim, "Selecting a namespace and database per call" (protocol `2026-07-28`, i.e. 3.3.0+ behavior — see caveat in §4 for 3.2.4):

> "Because a stateless request has no session to hold a selection, every tool except `use` takes optional `namespace` and `database` arguments. Resolution takes the first value it finds, filling each part independently:
> 1. The tool call's own `namespace` and `database` arguments
> 2. The `surreal-ns` and `surreal-db` request headers
> 3. The handshake session's `use` selection, on the revisions that still have one
> 4. The server's configured defaults"

So: **query params — no; it's tool-call arguments, or `surreal-ns`/`surreal-db` HTTP headers, or the MCP `use` tool/session selection, or server defaults** — in that precedence order. On v3.2.4 (pre-3.3.0, still handshake-based per the docs — see §4), the session-level `use` tool selection is the primary mechanism, with headers/args support depending on when each was added; the docs don't give a version-gated breakdown of exactly which of these four resolution steps existed pre-3.3.0 beyond noting "on the revisions that still have one" for the handshake session. STILL-UNKNOWN: exact behavior differences on 3.2.4 specifically for header vs. tool-arg precedence (the doc's per-call resolution list appears to be written from the current/3.3.0 vantage point).

---

## 4. Tool list, transport, protocol version, capability flags

### Transport
**Streamable HTTP** (`POST /mcp`), not SSE — the docs never mention SSE for the embedded server; all examples use `POST /mcp` streamable-HTTP semantics via the `url`+`headers` MCP client config shape.

### Published tools (`tools/list`), verbatim table:

> | Tool | Purpose |
> | `query` | Run SurrealQL and return serialised results |
> | `gql` | Run an ISO GQL query (on by default from 3.3.0; **on 3.2.x needs `--allow-experimental gql`**) |
> | `graphql` | Run a GraphQL query against the configured schema |
> | `select`, `create`, `insert`, `upsert`, `update`, `delete`, `relate` | Data manipulation helpers |
> | `run` | Call a database function with typed arguments |
> | `list` | List namespaces, databases, tables, indexes, users, … |
> | `use` | Select the namespace and database context |
> | `info` | Schema or engine information for a scope |

> "Legacy names such as `list_tables`, `use_database`, and `version` are no longer published. Use `list` and `use` instead."

This directly answers your capability-flag question for one tool: **`gql` requires `--allow-experimental gql` on 3.2.x** (your 3.2.4 falls in this bracket) — without that flag, `gql` calls should fail even though it's still listed. No other tool in the list is gated by a capability flag per the docs; general capability flags (`--allow-net`, `--deny-funcs`, `--allow-origin`) affect what *operations inside* tool calls (e.g. `run` calling an `http::*` function) can do, not whether the tools themselves are published.

### MCP protocol version

> "## Protocol versions <Since v="v3.3.0" />
> The server states which MCP specification revisions it implements rather than inheriting them from the SDK it links against, and it advertises **`2026-07-28`**. An unknown version in a request degrades to that revision.
> `2026-07-28` removes the `initialize` handshake and protocol sessions: a request carries its own protocol version, identity, and capabilities... Earlier handshake-based revisions are served on the same endpoint, so clients built against them keep working unchanged."

**Important caveat for your v3.2.4 server**: this whole "Protocol versions" section, the `2026-07-28` advertised version, and the stateless per-call ns/db resolution model are tagged `<Since v="v3.3.0" />` — i.e. **this is 3.3.0+ behavior, not yet applicable to your 3.2.4 install**. On 3.2.4, the server is on an earlier handshake-based MCP protocol revision (the docs don't name which pre-3.3.0 revision number 3.2.4 speaks — STILL-UNKNOWN exact protocol-version string for 3.2.x specifically), and session state (via the `mcp-session-id` header, acting as a bearer token, idle-timeout 5 minutes by default per the WARNING callout) is still in play rather than the fully stateless 3.3.0 model.

### Session header (applies to 3.2.4)

> "Run `/mcp` behind TLS in production. The session header acts like a bearer token for the life of the session: anyone who holds it can repeat tool calls as the same user until the session expires, five minutes after the last request by default."

### Server identity

> "The server also identifies itself to clients as `surrealdb`. Before 3.3.0 it reported the name of the MCP SDK crate it was built with." — so on 3.2.4, `serverInfo.name` in the MCP `initialize` response is likely still the underlying SDK crate name, not `surrealdb`. STILL-UNKNOWN exact pre-3.3.0 name string; docs don't give it.

### Configuration table (full, verbatim), for completeness:

| Variable | Default | Effect |
|---|---|---|
| `SURREAL_MCP_QUERY_TIMEOUT_SECS` | 60 | Outer timeout on each tool execution (`0` disables) |
| `SURREAL_MCP_MAX_RESULT_BYTES` | 256 KiB | Cap on serialised tool output (`0` disables) |
| `SURREAL_MCP_RUN_MAX_ARGS` | 64 | Maximum arguments to `run` |
| `SURREAL_MCP_PARAMS_MAX_KEYS` | 256 | Maximum top-level keys in parameter objects |
| `SURREAL_MCP_PARAMS_MAX_QL_BYTES` | 4 KiB | Maximum byte length of a `$ql` string inside a `*_data` payload |
| `SURREAL_MCP_SCHEMA_RESOURCE_MAX_TABLES` | 200 | Cap on tables enriched in the database schema resource |
| `SURREAL_MCP_ALLOWED_HOSTS` (v3.2.1+) | loopback only | Exact `Host` values accepted for HTTP `/mcp` |
| `SURREAL_MCP_ALLOW_ALL_HOSTS` (v3.2.1+) | `false` | Accept any `Host` (trusted-proxy escape hatch) |
| `SURREAL_HTTP_MAX_MCP_BODY_SIZE` | 4 MiB | Maximum HTTP body size for `/mcp` |

Also: `surrealdb.mcp.*` observability counters/histograms exist from 3.1.0 (`/docs/manage/observability/metrics`), and audit records go to the `surrealdb::mcp::audit` tracing target (tool name, subject, ns, db, outcome — never query text/row payloads) — useful if you want to confirm auth/host behavior via logs rather than guessing.

---

## 5. Exact client configs

### Claude Code `.mcp.json` (HTTP transport) — directly from docs

The **embedded-MCP page's own example** (this is the doc's literal example for a self-hosted `/mcp`, using Basic auth, which is exactly your scenario):

```json
{
  "mcpServers": {
    "surrealdb": {
      "url": "http://127.0.0.1:8000/mcp",
      "headers": {
        "Authorization": "Basic <base64-encoded username:password>"
      }
    }
  }
}
```

Adapted for your tailnet address `100.91.190.107:8471`:

```json
{
  "mcpServers": {
    "surrealdb": {
      "url": "http://100.91.190.107:8471/mcp",
      "headers": {
        "Authorization": "Basic <base64-encoded username:password>"
      }
    }
  }
}
```

The docs also give the equivalent `claude` CLI incantation (from "MCP in Claude," shown against the hosted server but the `--transport http ... --header` syntax is generic and applies identically to a self-hosted URL):

```bash
claude mcp add --transport http surrealdb http://100.91.190.107:8471/mcp \
  --header "Authorization: Basic <base64-encoded username:password>"
```

`--scope project` writes the entry into a committed `.mcp.json` per docs (*"To connect it for one project rather than everywhere, add `--scope project`. The entry is written to `.mcp.json` in the repository..."*).

Remember: `100.91.190.107:8471` must be in `SURREAL_MCP_ALLOWED_HOSTS` on the server first, or you'll get the same 403 regardless of a correct client config.

### Codex `config.toml`

**STILL-UNKNOWN per SurrealDB docs** — none of the fetched SurrealDB pages (`mcp/embedded.mdx`, `mcp/claude.mdx`, `mcp/cursor` link, `mcp.mdx`, CLI reference, environment-variables) mention Codex, OpenAI Codex CLI, or `config.toml` anywhere. I did not find a SurrealDB-authored Codex example. (Not included here as a guess per your instruction to avoid guessing — if you want, I can separately look up Codex's own generic HTTP-MCP `config.toml` schema from Codex's own docs rather than SurrealDB's, since SurrealDB simply doesn't cover it.)

---

## 6. `$ql` sentinel: typed values and `NONE` through `run` / `query` / `*_data` (added 2026-09-09 07:22 EDT, verified live)

Every JSON argument the MCP tools accept goes through one converter (`surrealdb/mcp/src/tools/mod.rs`, `json_to_surreal_value`): JSON `null` → SurrealQL `NULL`, numbers/strings/arrays/objects keep their types, and a single-key object `{"$ql": "<surrealql value>"}` is replaced by the parsed value. The `run` tool's schema, verbatim: "Optional argument list. Values are bound with their native types -- numbers stay numbers, objects stay objects. Embed typed SurrealDB values via `{"$ql": "<expr>"}` -- e.g. pass a record id as `{"$ql": "person:alice"}` or a decimal as `{"$ql": "9.99dec"}`." The `query` tool's `parameters` and the `create`/`insert`/`update`/`upsert` `*_data` payloads honour the same sentinel.

Source comment on `QL_SENTINEL`: "This is the canonical way for MCP clients to express types JSON cannot represent natively (decimal, datetime, duration, record id, uuid, bytes, geometry literals, ...) without resorting to the raw `query` tool." The body must be a non-empty string, at most `SURREAL_MCP_PARAMS_MAX_QL_BYTES` (4 KiB), the only key in its object, and it is parsed as one *value* (`syn::value_legacy_strand`), so a statement inside it is rejected.

Consequences for our functions (all `option<T>` parameters, plain, as documented):

| Need | Send |
|---|---|
| skip a middle optional argument | `{"$ql": "NONE"}` |
| skip trailing optionals | omit them; a shorter `args` array is accepted |
| record id | `{"$ql": "document:abc"}` |
| datetime | `{"$ql": "d'2026-09-09T00:00:00Z'"}` |
| JSON `null` | do not: it is `NULL`, and `option<T>` rejects it with `Expected none | … but found NULL` |

Verified 2026-09-09 07:22 EDT on the local store: `run type::is_none [{"$ql": "NONE"}]` → `true`; `[null]` → `false`; `query "RETURN [type::is_none($x), $x = NONE, type::of($x)]"` with `parameters {"x": {"$ql": "NONE"}}` → `[true, true, "none"]`.

## Summary of what's confirmed vs. still-unknown

**Confirmed, docs-verbatim:**
- `/mcp` (POST, streamable HTTP) is real, documented, on `surreal start`, same bind as REST API.
- The exact 403 `Host header is not allowed` message is documented, expected behavior (DNS-rebinding guard), default = loopback only.
- Fix: `SURREAL_MCP_ALLOWED_HOSTS=100.91.190.107:8471` (comma-separated, port-specific unless omitted); or `SURREAL_MCP_ALLOW_ALL_HOSTS=true` to disable (trusted-proxy only). Both env-var-only, v3.2.1+, no CLI flag.
- Auth: Bearer JWT or HTTP Basic, same middleware as `/sql`/RPC.
- Tools: `query`, `gql` (needs `--allow-experimental gql` on 3.2.x), `graphql`, `select`, `create`, `insert`, `upsert`, `update`, `delete`, `relate`, `run`, `list`, `use`, `info`.
- Claude Code `.mcp.json` shape confirmed verbatim from docs (Basic auth via `headers.Authorization`).

**STILL-UNKNOWN (docs silent):**
- Exact Rust source file implementing the allowlist (GitHub code search on `surrealdb/surrealdb` found nothing — likely an indexing gap, not absence).
- Whether record-access (SCOPE) JWTs are explicitly supported for `/mcp` (docs say "Bearer JWT, HTTP Basic, …" without detail).
- Exact pre-3.3.0 (i.e., your 3.2.4) MCP protocol-version string and server-name string — the docs' "Protocol versions" section is explicitly `<Since v="v3.3.0" />` and doesn't backfill what 3.2.x reports.
- Exact per-call ns/db header/arg precedence behavior specifically on 3.2.4 (the 4-step resolution list appears written for 3.3.0+; "on the revisions that still have one" implies 3.2.4 still uses session-based `use`).
- Any Codex `config.toml` example — SurrealDB docs never mention Codex.
