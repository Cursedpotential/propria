# Stack Decision — Probata surface (Workbench + Intake)

> _Byline: Claude Code · Opus 5 · 2026-09-27_
> Stage 1 of the app-planning-handoff pipeline. Registry versions read live from
> `registry.npmjs.org` on 2026-09-27; local versions read from every tracked
> manifest in the Propria monorepo via DuckDB, excluding `node_modules`.

## Application type

An **existing two-tier system**, not greenfield: a browser operator surface
(Probata Workbench) and a native desktop file workstation (Consignatio Intake /
Xplorer fork), both over Python and Go backends on the VPS. Stage 5 is therefore
an **audit**, not a scaffold.

## The stack is already decided

The owner settled these; this stage does not re-open them. It exists to record
the pin targets and measure how far the codebase has drifted from them.

| Layer | Choice | Current release (2026-09-27) | Workbench pins | Status |
|---|---|---|---|---|
| UI runtime | React | **19.3.0** (2026-09-09) | `19.2.3` | one minor behind |
| Build | Vite | **8.3.1** (2026-09-24) | `8.2.2` | current major |
| Language | TypeScript | **7.0.2** (2026-07-08) | `^5.9.0` | **two majors behind** |
| Routing | `@tanstack/react-router` | **1.170.39** (2026-09-23) | `^1.170.32` | current |
| Server state | `@tanstack/react-query` | **5.104.0** (2026-09-26) | `^5.60.5` | current major |
| Styling | Tailwind CSS | **4.3.3** (2026-07-16) | `^4` | current major |
| Data grid | `@glideapps/glide-data-grid` | **6.0.3** (2024-02-03) | `6.0.4-alpha24` | **alpha ahead of stable** |
| Validation | Zod | **4.6.5** (2026-09-13) | `4.4.3` (engine side) | current major |
| Desktop shell | `@tauri-apps/api` | **2.12.0** (2026-09-26) | `^2.9.1` / `2.11.1` | current major |
| Design tokens | `Propria/design-contract` | `tokens.json` → `tokens.css` | not consumed | **unused** |

## The actual problem this stage found

The stack is not in dispute. **Its application is.** Across 40 frontend apps in
five modules, measured from the manifests:

| Library | Apps | Distinct versions in use |
|---|---|---|
| typescript | 29 | **14** — `5.6.3` through `7.0.2`, including `~6.0.2` |
| react | 24 | **12** — 18.2, 18.3, 19.0, 19.1, 19.2.0/.3/.4/.7/.8 |
| vite | 21 | **13** — majors 5, 6, 7 and 8 all live |
| tailwindcss | 11 | **8** — v3 and v4 side by side |
| zod | 5 | v3 and v4 |
| glide-data-grid | 5 | 3, including the alpha |

Two routers compete: `@tanstack/react-router` in 2 apps, `react-router-dom` in 3.
**Next.js is a dependency of 5 apps across 4 modules**, which is where the
`src/app/page.tsx` naming in `workbench/web` came from — the convention leaked in
from modules that genuinely run Next, even though the Workbench itself has no
Next dependency and its own `AGENTS.md` forbids the convention.

Per-module app counts: vestigia 14, Probata 11, Consignatio 8, Legal-desktop 6,
FL-MCP 1.

There is **no shared package**. No `packages/ui`, no shared router/query/theme
setup, no single Glide wrapper. "Common stack" currently has nothing to be common
to, so every fix is made 40 times or not at all. That is the mechanism behind the
owner's recurring complaint that the same problems keep coming back.

## Integration points

Decided now, because stage 2 depends on them:

| Boundary | Mechanism | Current state |
|---|---|---|
| Workbench web → Probata API | REST over same-origin proxy | partly absent — see below |
| Workbench web → engine | REST | live |
| Intake desktop → engine | **Tauri IPC** (`invoke`) | command registered, UI does not mount it |
| Intake hosted → backend | REST `/filesystem/search` | live |
| Everything → object store | S3 API to Cloudflare R2 | **blocked**, account not entitled (error 10042) |

Two of those are the product's stated purpose and neither answers:

- `GET /api/tools` returns HTTP 200 with an **empty array**.
- `GET /api/monitored-actions/capabilities` returns **404**; no such backend route
  exists anywhere in the repository.

Recorded as critical Docstore flag `note:probata_function_access_broken_20260912`,
verified live, still active: *"This is a fail-closed scaffold, not a functional
vertical slice."* The three passing contract tests "only assert source strings…
they do not exercise a backend route or live tool."

## Time-sensitive flags

1. **Glide Data Grid is effectively unmaintained.** Latest stable `6.0.3` is from
   2024-02-03 — over nineteen months old. The Workbench runs `6.0.4-alpha24`, a
   prerelease with no stable successor. The owner's canon requires Glide for
   data-heavy tables, so this is a standing supply risk on a mandated dependency,
   not a style preference. Decide deliberately: stay pinned to the alpha, or
   choose a maintained grid and record the supersession.
2. **TypeScript 7 is the Go-port rewrite.** Moving 29 apps from 5.x to 7.x is a
   real migration, not a version bump. Pin the monorepo to one 5.x line first,
   then treat 7 as its own project.
3. **Two routers and Next.js in the same monorepo** will keep re-infecting
   conventions until the shared package exists and the odd ones out are retired
   or explicitly fenced.
4. **R2 is account-entitlement blocked**, not credential blocked. No key rotation
   repairs error 10042; it needs an R2 subscription on the matching Cloudflare
   account. Until then `object_store=false` and the Workbench stays `degraded`.

## Decision

**Keep the stack. Build the shared package. Pin every app to it.**

The Workbench is already closest to the target — React 19, Vite 8, TanStack
Router and Query, Tailwind 4, Glide — so it becomes the reference implementation
rather than another divergent app. `design-contract` is already canonical by
owner decision (2026-09-24) and simply has no consumer; giving it one is the
first concrete step toward "common stack, every module."

### Rejected alternatives

- **Re-picking the frontend stack.** Rejected: the owner decided it, the Workbench
  already runs it, and re-litigating settled choices is the pattern that produced
  40 divergent apps.
- **Upgrading everything to latest first.** Rejected: version alignment without a
  shared package regresses within a week, because nothing holds the line.
- **Leaving Next.js apps alone indefinitely.** Rejected as a permanent answer: it
  is the source of the convention drift. Acceptable short-term only if each Next
  app is explicitly fenced and named.
