# Architecture — sidecar/webview boundary

> _Byline: Claude Code · Sonnet 5 · 2026-09-07_

## Why a Node sidecar at all

The webview (React) can only speak HTTP/fetch to something. Two things this app needs are
Node-only:

1. **The plugin's SurrealDB case store client** (`mcp-app/src/store.ts`, built to
   `mcp-app/dist/store.js`) — it dynamically imports `surrealdb` + `@surrealdb/node`, which ships
   native `.node` binaries. There is no WASM/browser build of this client, and there shouldn't be
   one invented for this app — the client already exists and is owned by another agent's work.
   As of 2026-09-07 the case store itself is a **shared SurrealDB server** at
   `ws://127.0.0.1:8471` (credentials resolved by store.ts from
   `~/.secrets/family-court-toolkit.env`) — this sidecar sets `CUSTODY_CASE_DB` to that URL by
   default (see `sidecar/lib/store-client.mjs`) and never opens a local `rocksdb://` file
   directly. Multiple processes (this sidecar, the plugin's own `family-court-console` MCP
   server, others) can connect to it concurrently without file-lock contention — see the
   "Open items" note below for why that matters.
2. **`@anthropic-ai/claude-agent-sdk`'s `query()`** spawns a real `claude` CLI subprocess and
   talks to it over stdio. That is fundamentally a Node/OS-process capability, not something a
   webview sandbox can do.

So: Tauri's Rust shell spawns a plain Node process (`sidecar/server.mjs`) as a child, and the
webview talks to it over `fetch()`/SSE on `127.0.0.1`. This is the same shape as "Electron app
with a backend process," just with Tauri's much smaller native shell instead of a second Chromium.

## Process topology

```
Tauri (Rust)
  └─ spawns: node sidecar/server.mjs (PORT=0 → OS picks an ephemeral port)
        │
        │  stdout: "SIDECAR_READY {"port":NNNN}"  (captured by src-tauri/src/lib.rs)
        ▼
  Rust stores the port in app state, exposes it via the `sidecar_port` Tauri command
        │
        ▼
  Webview (React) calls invoke("sidecar_port") once, caches the base URL,
  then talks to http://127.0.0.1:<port>/api/* directly via fetch()/SSE.
```

In plain browser dev (`npm run dev`, no Tauri shell), the sidecar instead binds a **fixed** dev
port (4177, see `.env.example`) because there is no Rust parent to relay an ephemeral one to a
browser tab.

## Security

- **Loopback only.** `sidecar/server.mjs` binds `127.0.0.1`, never `0.0.0.0`. CORS is restricted
  to `127.0.0.1`/`tauri://`/`https://tauri.localhost` origins.
- **No token in the webview.** `CLAUDE_CODE_OAUTH_TOKEN` (however resolved — see `AUTH.md`) is
  read and used only inside the sidecar's own Node process, to let the Agent SDK's spawned `claude`
  CLI subprocess authenticate. It is never sent to the frontend, never included in any `/api/*`
  JSON response (see `sidecar/tests/server.test.mjs`'s explicit assertion on `/api/auth/status`),
  and never printed to any log this app controls.
- **Minimal Tauri capabilities.** `src-tauri/capabilities/default.json` grants only `core:default`
  and `opener:default` (used solely to open a docket row's local-file source in the OS's default
  handler). No filesystem, shell-exec, or HTTP-plugin permissions are granted to the webview —
  every network call from the webview goes to the sidecar's loopback API, nothing else.
- **Chat tool scope.** `/api/chat` restricts the Agent SDK's `allowedTools` to `Read` plus this
  plugin's own MCP server tools (`mcp__plugin_family-court-toolkit_family-court-console__*` and,
  redundantly, an explicitly-wired `mcp__family-court-console__*` alias — see
  `sidecar/lib/chat.mjs` for why both are wired) and runs with `permissionMode: "default"` (not
  `bypassPermissions`).

## Store access boundary

`sidecar/lib/store-client.mjs` imports the **built** module from the plugin's ABSOLUTE path
(`E:\AI_Workspace\plugins\plugins\family-court-toolkit\mcp-app\dist\store.js`) — never
`mcp-app/src/store.ts`, and never edits anything under `mcp-app/src/`. This app moved out of the
plugin directory on 2026-09-07 (see `../README.md`), so it is no longer a sibling of `mcp-app/`;
the absolute path is a deliberate, documented choice for this personal, single-machine tool, not
an oversight.

## Store connection: shared SurrealDB server

As of 2026-09-07 the case store is a **SurrealDB server process** listening at
`ws://127.0.0.1:8471`, not a RocksDB file this sidecar opens exclusively. `sidecar/lib/store-client.mjs`
sets `process.env.CUSTODY_CASE_DB = "ws://127.0.0.1:8471"` at load time UNLESS the environment
already set something else (so a test run's `mem://` override always wins — see
`sidecar/tests/`). store.ts itself resolves that URL and separately resolves credentials from
`~/.secrets/family-court-toolkit.env` — this sidecar never reads or handles those credentials.
This sidecar must never fall back to opening a local `rocksdb://` file directly; if the shared
server is unreachable, the correct behavior is the store reporting `{ available: false, reason }`
(see `UnavailableNotice` in the UI), not silently switching to a private local file.

## Open items (not resolved by this scaffold)

- **Bundled sidecar for a shipped `.msi`/`.exe`.** Today `src-tauri/src/lib.rs` resolves the
  sidecar script from the dev tree, a hardcoded known install location
  (`E:\AI_Workspace\Projects\the-platform-workspace\family-court-workbench\sidecar\server.mjs`),
  or a Tauri resource directory that nothing currently populates. True `externalBin` packaging
  (a compiled, target-triple-suffixed sidecar binary Tauri bundles and launches itself) was
  investigated and set aside: `@surrealdb/node` ships a native `.node` addon that a Node Single
  Executable Application (SEA) blob cannot transparently embed, so "standalone" here means "the
  installed `.exe` still requires a system Node install" (already true of this development
  machine) rather than a fully self-contained binary. Not resolved further — this is explicitly a
  **local desktop** tool per the owner's directive, not a distributable installer for other
  machines.
- **Claude subscription token outside the official harness.** See `AUTH.md` — flagged, not
  resolved.
