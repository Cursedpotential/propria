# Family Court Workbench — desktop app

> _Byline: Claude Code · Sonnet 5 · 2026-09-07. Moved out of
> `plugins/family-court-toolkit/app/` into its own tree here (owner order, same day, 15:29) — see
> the `README.moved` pointer left at the old location. Installed build:
> `C:\Users\matts\bin\family-court-workbench.exe` (+ `.cmd` dev launcher)._

A local Tauri desktop work surface for the `family-court-toolkit` Claude Code plugin: case
status, docket, timeline, memos, evidence, evals, reference material, and a Claude chat pane
(Agent SDK) that can call the plugin's own MCP tools (`survival_guide`, `court_language_review`,
and the SurrealDB `case_*` tools).

This is a **local desktop tool**, not a hosted product — see the owner's stack directive
("tanstack/storybook/glide/react", "tauri is local", "use the agent sdk and the long term token to
incorporate claude").

## What this is

- **Frontend**: Vite + React 19 + TanStack Router (file-based routes, code-split) + TanStack
  Query + TanStack Table (+ TanStack Virtual for the timeline) + Glide Data Grid (docket, exhibits)
  + Tailwind 4 (CSS-first `@theme`) + a small hand-rolled shadcn-style `components/ui/`.
- **Shell**: Tauri 2 (`src-tauri/`) — a thin Rust process that spawns the Node sidecar and hands
  its ephemeral port to the webview via one `sidecar_port` command. No filesystem/shell/network
  Tauri permissions beyond `opener` (used to open a docket row's local-file source).
- **Sidecar**: `sidecar/server.mjs` (Fastify, loopback-only) — the thing that actually needs a
  full Node runtime, because both the Agent SDK and the plugin's SurrealDB case store client
  (`mcp-app/dist/store.js`, loaded from the configured plugin root — see "Where the store lives"
  below) are Node-only. Exposes:
  - `GET /api/store/summary|search|graph|timeline|factor-map|docket|memos|status|source|reference|evidence|evals`
  - `POST /api/store/export` (`snapshot` or `platform` format)
  - `POST /api/chat` — Server-Sent Events, drives `@anthropic-ai/claude-agent-sdk`'s `query()`
  - `GET /api/auth/status`, `GET /api/health`

See `docs/ARCHITECTURE.md` for the sidecar/webview security boundary and the shared-SurrealDB-server
connection, and `docs/AUTH.md` for how the Claude Code OAuth token is resolved (and the open
question about token-use terms).

## Where the store lives

This app **never imports or edits `mcp-app/src/*`** — that stays owned by whoever maintains the
plugin's `mcp-app/src/store.ts`. All store access goes through `sidecar/lib/store-client.mjs`,
which dynamically imports the **built** module under `FAMILY_COURT_PLUGIN_ROOT` (defaulting to
`E:\AI_Workspace\plugins\plugins\family-court-toolkit`) and calls it by function name. The chat
sidecar uses the same root. `case_docket`, `case_memo`, `case_status`,
`case_source`, `case_reference`, `case_evidence_log`, `case_eval`, mode-aware `case_timeline`, and
both `case_export` formats (`snapshot`/`platform`) are all real, landed features — store.ts even
exposes compatibility alias functions (`caseMemo`, `caseStatus`, `caseSource`, `caseReference`,
`caseEvidenceLog`, `caseEvals`) matching this sidecar's exact dispatch names.

### Store connection: shared SurrealDB server

The case store is a shared SurrealDB server, not a local RocksDB file this app owns alone. The
sidecar takes `CUSTODY_CASE_DB` from an explicit environment setting or from the assignment in
`~/.secrets/family-court-toolkit.env`; missing or invalid shared-store configuration fails visibly.
Only an explicit `mem://` environment override selects an isolated test store. The sidecar never
falls back to local RocksDB. The canonical store module continues to resolve shared credentials.

## Running it

```bash
npm install
npm run build        # vite build (generates src/routeTree.gen.ts) + tsc -b typecheck
npm test             # sidecar node:test suite + vitest component tests
npm run dev          # sidecar (fixed port 4177) + vite, for browser preview
npm run tauri:dev    # real desktop shell
npm run storybook    # component gallery (dark + light, a11y addon)
```

Or just run the installed build: `C:\Users\matts\bin\family-court-workbench.cmd` (dev launcher) or
`family-court-workbench.exe` directly.

### Auth

Set `CLAUDE_CODE_OAUTH_TOKEN` (from `claude setup-token`) as an environment variable or in a
`~/.secrets/*.env` file before starting the sidecar, or have `claude login` already run
interactively on this machine. See `docs/AUTH.md`.

## Tauri build status

See the **exact recorded outcome** appended at the bottom of this file after the `npm run
tauri:build` run(s) against this tree (do not loop on Rust toolchain issues beyond a couple of
attempts, per the task's own instruction).

## App section in the plugin's own README

The plugin root `README.md` (`plugins/family-court-toolkit/README.md`) has an "App" section
(appended, not rewritten) pointing here, plus a `README.moved` pointer file left at the old
`plugins/family-court-toolkit/app/` location.
