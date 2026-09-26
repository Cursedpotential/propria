# TraceIQ reports — Evidence Studio setup

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Claude Code · Fable 5 · 2026-07-24_

Five report pages authored in Evidence Studio markdoc syntax (component syntax verified
against the evidence-studio MCP docs live):

| Page | What it gives you |
|---|---|
| `overview.md` | Corpus stats, monthly-coverage gap chart, event mix |
| `live-map.md` | **Date-range picker → point map of visits (sized by count, colored by overnights) + movement heatmap + dwell-stop map** |
| `daily-analysis.md` | Date range → calendar heatmap, day-by-day event table w/ Google Maps links, overnights + map |
| `places.md` | Place analytics, home-base candidates table + weighted map |
| `anomalies.md` | Impossible speeds, unrecorded stops, missing months, confidence caveats |

## Connect (one-time)

1. Open Evidence Studio → add a **PostgreSQL** source:
   host `100.119.96.29` · port `5432` · database `traceiq` · user `ai` (tailnet must be up)
2. Point Studio at this folder (`traceiq-rebuild\reports\`) or copy the pages into your
   Studio project's pages directory.
3. Queries reference schemas directly (`analysis.latest_events`, `working.waypoint`) —
   if your source config namespaces them (e.g. `traceiq.analysis...`), prepend the
   source name in the `from` clauses.

## Honest status

Pages are written to Studio's documented syntax but **not yet rendered** — I can't drive
the Studio GUI headlessly. Expect minor first-open tweaks (e.g. `size_value` scaling,
`between` filter grain). The SQL itself is tested shape — all views exist and return
data (see VALIDATION_REPORT.md counts).

Maps run on Evidence's built-in MapLibre — free, no token, no vendor lock (ADR-0011
spirit). A Mapbox tile style can be layered later purely for looks.
