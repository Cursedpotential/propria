# TraceIQ Workspace — Build Brief PHASE 2 (live data, editing, validation, map fix)

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> Handoff for OpenCode. Read ui/BUILD_BRIEF.md + docs/adr/0015 first. Branch: ui-scaffold.
> Commit incrementally, NEVER push, work in ui/ (UI) and ops/ (validation script) only,
> NEVER touch E:\AI_Workspace\.git or anything outside traceiq-rebuild/.

## State handed to you (already done, committed at 179496f)

- Live data path wired: `ui/.env.local` (TRACEIQ_DSN=postgresql://ai:ai@100.119.96.29:5432/traceiq,
  gitignored), `src/lib/db.ts` (pg Pool), `src/app/api/events/route.ts` (queries
  analysis.latest_events + ref.known_place tags), `src/lib/pgAdapter.ts` (client adapter),
  `getAdapter()` now returns pgAdapter unless NEXT_PUBLIC_DATA_MODE=mock.
- `next.config.ts` switched from static-export to SERVER app (required for /api routes) +
  serverExternalPackages:['pg']. **The /api/events endpoint returns HTTP 200 with real data
  (verified).** So live read-path WORKS at the API level.

## TASK 1 — FIX THE MAP (blocking; it crashes on render)

The map component `src/components/map/MapView.tsx` crashes. Two errors seen so far:
- With `react-map-gl/maplibre`: `cloneTransform ... Cannot read properties of undefined
  (reading 'clone')` — react-map-gl 7.1.7 is incompatible with maplibre-gl 6.0.0.
- After switching import to `@vis.gl/react-maplibre`: `Maplibre._updateSize ... Cannot read
  properties of undefined (reading 'width')` at MapView.tsx ~line 119 (the <Map> element,
  `mapLib={maplibregl}`).
Installed versions: react-map-gl 7.1.7, maplibre-gl 6.0.0, @vis.gl/react-maplibre (present),
deck.gl (present). ROOT CAUSE is a maplibre/react-wrapper version mismatch.
FIX IT PROPERLY: pick ONE compatible, current combo (e.g. @vis.gl/react-maplibre with a
maplibre-gl version it officially supports — check its peerDeps; pin maplibre-gl accordingly),
remove the unused wrapper, ensure the DeckGL overlay still composes with the basemap, and
ensure the map CONTAINER has explicit width/height before the map inits (the 'width'
undefined suggests a zero-size container at mount — guard with a sized parent or a mounted
check). VERIFY the map actually renders (npm run dev, load http://localhost:3000, no console
error, pins visible).

## TASK 2 — VERIFY & POLISH LIVE DATA

Confirm the UI renders her real corpus (analysis.latest_events ~7,236 visits / 6,379
activities / 6,545 paths; 760 places; 5,189 overnights). Wire filters (date range, event
type, tags, overnight) to actually hit /api/events params. Handle loading + empty + error
states honestly (no fake data). Keep the mock as NEXT_PUBLIC_DATA_MODE=mock fallback.

## TASK 3 — EDITING SURFACE (write-path, owner-requested)

Add real editing of ref.* (NEVER raw/working — evidence is read-only). Minimum: a write API
`src/app/api/known-place/route.ts` (POST) that upserts ref.known_place (place_id or lat_r4/
lng_r4 key, label, tags[], is_concerning, is_watch, severity, note) — mirror ops/flag.py
semantics. Wire a small inline editor in the results/detail panel (label a place, toggle
concerning/watch, add tags). Audit-friendly. Also leave the Tables tab as the future
Mathesar/pgAdmin embed target (stub/link ok).

## TASK 4 — INGEST VALIDATION SCRIPT (ops/, owner-requested)

Create `ops/validate_ingest.py` (Python, psycopg, reads DSN like the other ops scripts).
Goal: PROVE the whole source JSON ingested correctly, chunk by chunk, and SHOW it readably.
For each row in raw.chunk: re-read its source archive file (raw.chunk.archive_path under
E:\TraceIQ_Evidence\), STREAM-parse it (ijson — files are ~14MB, handle in chunks/streaming,
don't assume small), count records per top-level section (semanticSegments/timelineObjects/
rawSignals/userLocationProfile), and compare to raw.record counts for that chunk (grouped by
google_type). Verify file SHA-256 matches raw.chunk.file_sha256 (chain of custody). Also show
the working-layer reconciliation (working.build: raw_count = derived_count + exception_count,
unaccounted must be 0). Output a clear per-chunk table: source file, source count, DB count,
MATCH ✓/✗, hash ✓/✗, by-type breakdown, and a final PASS/FAIL summary. Read-only against the
DB. Print to stdout AND write ops/INGEST_VALIDATION_REPORT.md.

## Definition of done

- Map renders with real pins, no console errors, both themes.
- Filters drive real /api/events; UI shows her real numbers.
- Can label a place + flag concerning/watch from the UI (writes ref.known_place, re-reads).
- `python ops/validate_ingest.py` runs and shows per-chunk ingest proof (PASS on current data).
- `npm run build` passes. README updated. All committed to ui-scaffold. NO push.
