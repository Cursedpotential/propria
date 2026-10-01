# HANDOFF — ContextForge federation, hosted coolify-write, SurrealDB plugin (2026-10-01)

> _Byline: Claude Code · Opus 5.5 · 2026-10-01_
STATUS: PARTIAL
BUILD_STATUS: UNKNOWN (no build or test suite covers this work; every piece below was checked with live calls instead)

## Verified-live state (do not re-derive)

| Thing | State |
|---|---|
| Hosted Coolify tools | Coolify app `coolify-mcp` (`oyzznioap03u34xz125l90oq`, ovh-app) builds from `Cursedpotential/propria-plugins`, base `/plugins/coolify-write`, compose `/compose.hosted.yaml`, watch `plugins/coolify-write/**`. Running, healthy. 42 tools at `http://100.72.169.40:8765/mcp`. |
| ContextForge `coolify-write` | Gateway `7f8f263a38d04672bd433b8b7bc460cd` (refreshed), virtual server `e0bc95b5e93148e785154c350fc72830` with 42 tools. Public route `https://mcp.mitechconsult.com/servers/e0bc95b5…/mcp`; `coolify-write-get-infrastructure-overview` returns Coolify 4.1.2 and 3 servers. |
| Clients | Claude Code plugin `coolify-write` 1.2.1 `.mcp.json` → that URL (`claude mcp list`: Connected). Codex `[mcp_servers.coolify-write]` with bearer `CF_MCP_CLIENT_TOKEN` and 22 carried-over approvals; a real Codex session listed 42 tools and called `coolify-write-list-servers`. |
| Plugin commits | `propria-plugins` 0a97ab5 (1.1.1 hosting patches, Dockerfile, compose), 3f38fa3 (sync skips `.in_use`), 3cd2370 (1.2.0 switch to ContextForge, docs), 08d61b9 (1.2.1 live Coolify version). |
| Monorepo commits | c7065eac (stale `deploy/coolify-mcp.yaml` + `deploy/docker/coolify-mcp/` removed; AGENTS.md exception), bd2e0880 (webhook cause logged). |
| SurrealDB | Plugin `surrealdb` 1.0.0 (143499f) installed in Claude Code and Codex; ContextForge server `surrealdb` `b67cbe99…` (10 tools: `docs-*`, `mem-*`). Instances on ovh-files, all 3.2.4: surreal-docs :8472, surreal-case :8471, surreal-intake :8473. |
| Push webhooks | Coolify's `/webhooks/source/github/events` answers 200 on the tailnet, no answer on the public address (closed 09-14). GitHub cannot deliver, so no push deploys any app. |

## Findings / work done

1. **One copy of the Coolify server.** The hosted copy was a stale monorepo duplicate (21 tools, 9 older tool bodies). The plugin's `scripts/server.py` lacked two hosting patches (process env over env file; `mcp.settings` host/port, stateless, JSON, DNS-rebinding off). Both merged into the plugin; the container builds from it.
2. **Docs now give the safe refresh**: `POST /gateways/<id>/tools/refresh`, then add tool ids to the virtual server. The old `MCP-PATH.md` told agents to `PUT` the gateway, which clears stored auth (Docstore outage 09-28).
3. **Why pushes never deploy** (answer to the owner's 07:38 question): GitHub cannot reach Coolify since the public ports were closed. Fleet-wide, not specific to `coolify-mcp`.
4. **`repository_project_id`** of `coolify-mcp` still names the monorepo; Coolify's API answers 422 "not allowed" to changing it. Matters only once webhooks work.

## UNRESOLVED (mandatory)

- Push deploys — blocked on owner decision below. Until then every app, `coolify-mcp` included, is deployed by hand after a push.
- `coolify-mcp` webhook link (`repository_project_id`) — API refuses; needs the Coolify UI or recreating the app on `propria-plugins`.
- Codex default model `gpt-6.1-sol` (changed outside this session) is rejected for the ChatGPT account; `codex exec` only runs with `-m gpt-5.6-sol`.
- Morph: `MORPH_API_KEY` set nowhere, so `morph-mcp` has no tools (and timed out connecting this session). Copying it from an old backup was refused by the permission classifier.
- `destreamed` plugin needs the owner to sign in via `/mcp`.
- Two switched-off ContextForge entries (gateway `ctl`, server `propria-docstore-retired-8172`) await the owner's delete.
- Rename container `coolify-mcp` → `coolify-write` offered, not answered.

## Pending owner decisions

- **Make pushes deploy again.** WHY: no push has deployed anything since 09-14. A (recommended): publish only Coolify's `/webhooks/*` through the Cloudflare tunnel; GitHub signs each call with the app secret, the rest of Coolify stays tailnet-only. B: GitHub Action joins the tailnet with the `tag:docker` auth key and calls the deploy API (more moving parts, per-repo workflow files). C: keep deploying by hand.
- **Codex model:** pick the default (gpt-5.6-sol worked yesterday).
- **surreal-intake tools:** A none (default), B open its `/mcp` allow-list and add a gateway.
- **Rename** `coolify-mcp` → `coolify-write`: yes/no.

## Next steps (work in order)

1. On the owner's webhook pick, implement it and prove it with one real push that starts a deployment by itself.
2. Repoint `coolify-mcp`'s webhook link to `propria-plugins` (UI or recreate), then push a no-op plugin change and confirm it deploys.
3. Set the Codex model when chosen; rerun the Codex check without `-m`.
4. Rename if approved (compose `container_name`, Coolify app name, docs).

## Owner working-style contract

- Structured replies: bullets, labeled blocks, white space, answer-first.
- Confirm before changes; never hard-delete (quarantine); byline every artifact; verify before claiming done.
