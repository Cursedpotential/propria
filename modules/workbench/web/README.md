# The Platform Workbench — browser application

> _Byline: Codex · GPT-5.6-Sol · 2026-08-30._
> _Byline: Claude Code · Opus 5.5 · 2026-09-26 (Verification: `npm run smoke` is browser-free; browser journeys run on a VPS only)._

This is the browser-first operator surface for The Platform. It is a React + Vite application
served same-origin by `workbench/api` and deployed as the `knowledge-workbench` Coolify service.
It does not require Next.js, server components, server actions, or a separate JavaScript runtime in
production.

Desktop packaging is deliberately deferred. Once the browser product is complete, a Tauri host can
be added around the same client application with explicit adapters for local files, IPC, and SQLite.
The separate Case Bible desktop/sorting lane is not copied into this directory.

## Current stack

- React 19 + TypeScript
- Vite 8 for browser development and production bundling
- TanStack Router for code-split client routes
- Storybook 10 on the same Vite builder
- Tailwind CSS and the tracked Platform component primitives
- FastAPI for same-origin APIs and SPA fallback serving
- Indexed/browser state only where a feature explicitly needs it; PostgreSQL remains canonical

Glide Data Grid is the selected direction for data-heavy review tables, but table migration is not
claimed by the Vite shell release. It should be introduced one complete operational table at a time,
with its data contract and browser smoke coverage intact.

## Product boundary

The root route is the Evidence Operations Desk. The primary navigation exposes only the complete
daily path:

- `/` — live operational desk
- `/intake` — governed source selection and intake
- `/evidence/preview` — parser/message/provenance preview

Existing advanced routes remain directly addressable while they are reconciled, but they are not
advertised as finished navigation destinations. The browser must never infer a canonical write from
local state; durable API receipts remain authoritative.

## Local browser development

Use Node.js 22.13 or newer; the Vite and AI SDK dependency graph is intentionally built on the
same supported Node major used by the production image.

```powershell
npm ci
npm run dev
```

Vite listens on `127.0.0.1:5173` and proxies `/api` and `/health` to the Workbench API at
`127.0.0.1:8020`. Set `VITE_API_URL` only when a different browser API origin is intentionally
required. Classification-specific calls may use `VITE_API_BASE`; the default is `/api`.

## Verification

```powershell
npm run lint
npm run build
npm run smoke
npm run build-storybook
```

`npm run build` typechecks and writes the production bundle to `dist/`. `npm run smoke` runs the
static contract tests and launches no browser: the four browser-driven journeys in
`smoke/matter-flow.smoke.test.mjs` (Matter promotion and review, the Matter capability gate, and New
Run with a staged and a fresh file) report as skipped unless `SMOKE_BROWSER` names a browser binary,
and always on Windows. A skipped journey is not browser proof.

### Browser journeys run on a VPS only

Never set `SMOKE_BROWSER` or run the browser journeys on the owner's desktop: headless Chrome and
Edge froze it (owner ban 2026-09-24). Run them in the Probata devbox on ovh-files (the agents' Kasm
sandbox, `deploy/devbox.yaml`), from anywhere in this repository:

```bash
modules/workbench/web/smoke/run-in-devbox.sh [git-ref]
```

It packs the committed web source at the ref (default `HEAD`) with `git archive`, streams it over
SSH into the devbox, and runs `npm ci` and `npm run smoke:matter-flow` there as the non-root user
with Chrome's real binary, so Chrome keeps its own sandbox. Nothing browser-related runs locally,
and uncommitted edits are not tested. Each run unpacks into `~/browser-journeys-<sha>` in the devbox.

On another VPS, run `SMOKE_BROWSER=/path/to/chromium npm run smoke:matter-flow` from this directory
as a non-root user; there is no default browser path. `npm run smoke:matter-flow` runs
`smoke/browser-preflight.mjs` first and fails, instead of reporting four skips as a pass, on
Windows, without `SMOKE_BROWSER`, or when that path does not exist. The journeys serve the built
SPA through a same-origin fixture, drive it over the DevTools pipe, and leave each browser profile
under the repository's `to_be_deleted/` directory for owner-only cleanup.

The focused backend static-serving contract runs from the repository root:

```powershell
uv run pytest -q workbench/api/tests/test_vite_static_frontend.py
```

## Production serving

`workbench/Dockerfile` runs `npm ci`, builds `web/dist`, copies it into `/app/static`, and starts the
FastAPI service on port 8020. FastAPI routers are registered before the static mount. Extensionless
client routes fall back to `index.html`; reserved API/documentation/health paths and missing assets
remain real 404 responses.

The stable private browser address is `https://workbench.tilapia-skilift.ts.net`. A successful local
build or an old healthy container is not deployment proof; acceptance requires the exact Coolify
revision plus live root, health, and deep-link verification.

## Donor attribution

The original interface was bootstrapped from the MIT-licensed
`backblaze-b2-samples/agentic-rag-vector-starter-kit`. Its workspace, chat/RAG dashboard, B2 browser,
and Next.js runtime assumptions were removed. Shared primitive ancestry and this attribution are
retained.

The superseded Next-specific README is preserved, not deleted, at
`to_be_deleted/workbench-next-shell-20260830/workbench-web/README-next-architecture.md`.
