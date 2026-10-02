# TraceIQ Workspace — Build Brief (scaffold phase)

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Claude Code · Fable 5 · 2026-07-24 · Handoff spec for OpenCode/any builder._
> Authority: ADR-0015 (proposed) + mockup `../docs/mockups/traceiq-workspace-mockups.html`
> (Variant A = chosen). Read both FIRST. ADRs 0001-0014 in ../docs/adr/ are accepted law.

## Mission (this phase ONLY)

Scaffold the TraceIQ workspace as a Next.js/TypeScript app in THIS directory
(`traceiq-rebuild/ui/`). Static/mock-data phase: NO live DB writes, NO deployment, NO agent
wiring. Deliver a runnable `npm run dev` app that faithfully implements mockup Variant A.

## Stack (decided — do not relitigate)

- Next.js (app router) + TypeScript, pnpm or npm
- **Map**: deck.gl over MapLibre GL basemap (free, no tokens). NOT full Kepler embedded —
  Kepler.gl is a LATER pop-out window (stub the "↗ Kepler" button).
- **Chat pane**: assistant-ui (@assistant-ui/react) with a mock backend adapter (canned
  responses w/ receipt chips). Real Claude-SDK/AgentOS wiring is a later phase.
- Styling: match the mockup's token system (CSS vars, light+dark, mono-for-data). Tailwind
  ok if tokens preserved.

## Layout = mockup Variant A, exactly

Left rail (query controls + optional chat) · center deck.gl map · right results panel.
Top bar: brand, subject selector, tabs Explore/Analytics/Tables/Export/Config (only
Explore functional this phase; others = placeholder pages), Manual⇄Agent-assist pill.
Map toolbar: pins/paths/heatmap/time modes + time scrubber + "↗ Kepler" stub.
Results: sortable table, row-click highlights map pin, "↗ map"/"↗ analytics" door stubs.

## Governing constraints (from ADR-0015 — violating these = wrong build)

1. DUAL-USE: every capability operable with the chat pane collapsed/off.
2. Receipts: mock agent answers include a receipt chip citing a row id — UI treats the
   receipt as the source of truth, chat text as narration.
3. No evidence editing anywhere: only ref.* concepts (labels/tags/flags) get edit
   affordances (mock handlers this phase).

## Data (mock this phase)

`src/mock/` with ~50 events shaped like the real schema (event_id, serial_display,
event_type visit|activity|timeline_path, start/end eastern, lat/lng, place_id, tags,
overnight_simple, probability). Shape mirrors analysis.latest_events (see
../db/transformations/t002_wave1_views_v1.sql for the real columns). A `DataAdapter`
interface so the live PG/DuckDB adapter can swap in later without UI changes.

## Definition of done

- `npm run dev` renders Variant A both themes, responsive ≥900px + phone-degraded
- Filters (date range, type chips, tag chips) actually filter the mock data on map+table
- Row-click ↔ map-pin selection sync works
- Chat pane: mock exchange incl. a receipt chip that, when clicked, selects that row
- README with run instructions + honest TODO list of stubs
- COMMIT to a NEW branch `ui-scaffold` in this repo (traceiq-rebuild has no git yet —
  `git init` here if absent). NEVER push anywhere. NEVER touch E:\AI_Workspace\.git.

## Later-phase note (owner, 2026-07-24 — for the adapter design, not this phase)

Chat brain = DIRECT Claude Agent SDK loop (owner keeps direct Anthropic access) whose
toolset includes `call_agentos_agent` → AgentOS `run_agent` (REST :8000 / MCP :8000/mcp,
bearer). SDK = conductor; agno specialists = callable tools. Design the chat adapter
interface with that dual backend in mind (mock now).

## Do NOT

- Touch the live database (100.119.96.29) at all
- Modify anything outside traceiq-rebuild/ui/
- Add API keys/secrets of any kind
- Install Kepler.gl as a dependency this phase (stub only)
