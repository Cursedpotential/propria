# Vestigia workspace UI

> _Byline: Claude Code · Opus 5.5 · 2026-10-02. Re-conformed to the code: server app over live
> data, not the static mock scaffold the earlier README described._

The dual-use (manual + agent-assisted) analysis workspace from `BUILD_BRIEF.md`,
`BUILD_BRIEF_PHASE2.md` and ADR-0015 (proposed). Layout is Variant A of
`../docs/mockups/traceiq-workspace-mockups.html`: query rail and chat on the left, map in the
centre, results table on the right.

## Stack

- Next.js 16.2 (App Router, server app; `pg` is a server external), React 19, TypeScript
- Tailwind CSS v4 with light/dark tokens
- deck.gl 9 over MapLibre GL 5 via `@vis.gl/react-maplibre`
- `pg` against the live `traceiq` database (`src/lib/db.ts`, reads `TRACEIQ_DSN`; see
  `../.env.example`)

## Scripts

| Script | Purpose |
|--------|---------|
| `npm run dev` | Next.js dev server |
| `npm run build` | Production build (server app) |
| `npm run start` | Serve the production build |
| `npm run lint` | ESLint |

The UI has only ever run as a desktop dev server. Hosting it on a VPS is plan item V-2
(`../docs/plans/2026-10-02-continuation-plan.md`).

## What works

- Live data: `/api/events` and `/api/events/bounds` with windowed loading (`src/lib/pgAdapter.ts`).
- Known-place editing: `/api/known-place`, `KnownPlaceEditor`.
- Filters, map modes (pins, paths, heatmap, time of day), table ↔ map selection sync, themes.

## Not built yet

- Analytics, Tables, Export and Config tabs (placeholders).
- Chat: canned answers from `src/mock/adapter.ts`; no model wired.
- Kepler.gl pop-out, evidence drawer with receipts, keyboard and accessibility pass, filter/theme
  persistence, phone layout.
- Authentication.

Owner direction 2026-10-02: analysis moves to SurrealDB; the UI will read analysis from there and
source records from Postgres (plan Phase 3).
