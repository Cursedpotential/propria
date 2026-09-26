# TraceIQ Workspace UI Scaffold

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


This directory contains the **TraceIQ workspace UI scaffold** — a front-end prototype for the dual-use (manual + agent-native) analysis workspace described in `BUILD_BRIEF.md` and `docs/adr/0015-dual-use-agent-native-workspace.md`.

It implements **Variant A** from `docs/mockups/traceiq-workspace-mockups.html`: a left rail for query controls and chat, a central deck.gl-over-MapLibre map, and a right-hand results table.

## Stack

- Next.js (App Router) + TypeScript
- Tailwind CSS v4 with custom light/dark tokens
- deck.gl over MapLibre GL via `react-map-gl` / `@vis.gl/react-maplibre`
- Mock data shaped like `analysis.latest_events`
- Local mock chat with receipt chips (no live LLM in this scaffold)

## Getting started

```bash
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

## Available scripts

| Script | Purpose |
|--------|---------|
| `npm run dev` | Start the Next.js dev server |
| `npm run build` | Static export to `dist/` |
| `npm run lint` | Run ESLint |

## Project layout (inside `ui/`)

```
src/
  app/              Next.js pages + global styles
  components/
    shell/          AppTopBar, QueryPanel
    map/            MapView, MapControls, MapLegend, TimeScrubber
    table/          ResultsTable
    chat/           ChatPane
  lib/              WorkspaceContext, useTheme
  mock/             generateMockEvents + DataAdapter stub
  types/            TraceEvent, FilterState, DataAdapter, etc.
```

## Current scope

- Variant A shell layout with light/dark theme toggle
- Working filters (query, date, type, tags, overnight, probability)
- Map modes: Pins, Paths, Heatmap, Time-of-day
- Results table with sort, row selection, and map fly-to sync
- Agent chat stub with deterministic receipt chips
- Placeholder pages for Analytics / Tables / Export / Config

## TODO / stubs for future phases

- [ ] Wire real `analysis.latest_events` data from the backend
- [ ] Replace mock `DataAdapter` with HTTP adapter + caching
- [ ] Integrate real assistant-ui backend or LLM with receipt grounding
- [ ] Add Kepler.gl pop-out / deep-link
- [ ] Implement Analytics, Tables, Export, and Config modules
- [ ] Add row-level evidence drawer and deterministic receipts
- [ ] Keyboard shortcuts and accessibility pass
- [ ] Persist filters/theme in URL / localStorage
- [ ] Mobile/responsive refinement

## Notes

- Do not push this branch (`ui-scaffold`) to the remote repository.
- Keep all UI work inside the `ui/` directory.
