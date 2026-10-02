# ADR-0015: Dual-use agent-native workspace (unified surface)

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


**Date**: 2026-07-24
**Status**: proposed
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

TraceIQ's data/analysis core is live (raw→working→analysis on data-pg, provider layer,
ADRs 0001-0014). The owner works it today across three tools (NocoDB for tables, Evidence
for reports, CLIs for flags/providers) and wants ONE unified surface. Governing requirement,
stated explicitly: **dual-use** — manual and agent-assisted, with manual as a first-class
equal, not a fallback.

## Decision

Build (future, owner-gated) a single web workspace over the TraceIQ API with these
non-negotiable properties:

1. **Dual-use is the governing constraint.** Every capability — query, view, data entry,
   export — is fully reachable by hand. The AI agent is an *optional co-pilot* for analysis
   and retrieval, never a required path. A lawyer or the owner must be able to operate the
   entire surface with the agent turned off. A visible **Manual ⇄ Agent-assist** posture.
2. **Agent answers with receipts; it never becomes the evidence.** Any agent retrieval or
   analysis resolves to deterministic `working`/`analysis` rows that are re-derivable from
   raw (ADR-0005). The agent narrates and navigates; the exhibit is always the underlying
   verifiable row/snapshot. This makes the convenience layer and court-defensibility the
   same mechanism.
3. **First capability = "ask-and-it-shows" retrieval** (owner's pick). "Was she at X on
   the night of June 15?" → agent runs a deterministic query, drops the map pin, opens the
   exact rows. This is the cheapest layer (MCP tools over the live DB; no knowledge-graph
   or legal-agent dependency) and doubles as the court-safety proof.
4. **Hybrid brain (later layers).** Claude Agent SDK as conductor + agno specialist agents
   (a timeline agent with traceiq MCP tools; the existing legal agents lc-michigan-law /
   MCL-722.23 factor-mapper as the Analysis→Legal bridge). Derived facts (home bases,
   patterns, labels) ingested into the Graphiti temporal knowledge graph as entities that
   POINT BACK to deterministic rows — with a "prove it landed" node-count gate (owner's
   prior silent-empty-graph scar). Milvus/vectors secondary.

## Layout (owner-described)

- **Zone A — Controls + Chat**: manual query controls always visible (date/time range,
  subject, event-type, place/tag filters); optional agent chat adjacent/below.
- **Zone B — Map**: pins / heatmap / paths for the current query; click → detail.
- **Zone C — Data / Report**: results table (sortable; `ref.*` rows editable inline) or the
  rendered analysis/report for the current selection.
- **Export tab**: snapshot / CSV / KML / GeoJSON / court-exhibit generation.
- **Config tab**: tunable parameters (`ref.parameter`), providers, `ref.known_place`.

## Direction update (2026-07-24, owner)

- **Chosen base layout: Variant A (left rail)** — sidebar (controls + optional chat) · center
  map · right data. Densest console; deep-dive pop-outs relieve the density.
- **Map stack: deck.gl embedded + Kepler.gl pop-out, fed by DuckDB.**
  - In-app Zone B map = **deck.gl** (Kepler's WebGL engine, lightweight, no Redux) — fast
    pins/paths/heatmaps for the current query.
  - **Full Kepler.gl as a pop-out** (second window/tab) for exploratory deep-dives: time
    animation of movement, hexbin/heatmap aggregation, path arcs. Heavy React app, so it
    lives in its own window, not embedded in the console.
  - DuckDB (already in-stack, pg_duckdb + local DuckDB) is the shared geospatial feed
    (Arrow/GeoJSON) for both deck.gl and Kepler.
- **Evidence stays the reporting/analytics surface**, embedded for inline reports and
  **popped out** to its own tab for day/week/month analytics deep-dives.
- **Pop-out / second-window pattern (owner-requested):** any path, visit, or period gives
  two doors — **"↗ Explore on map"** (Kepler deep-dive) and **"↗ Analytics"** (Evidence
  report). Deep dives never leave the deterministic core; they read the same `working`/
  `analysis` rows.
- **Table editing surfaces (owner-requested):** the console handles inline `ref.*` edits
  for flow, but two dedicated data tools ride alongside as tabs/pop-outs —
  **Mathesar** as the friendly spreadsheet-style editor/browser for `ref.*` + `analysis.*`,
  and **pgAdmin** as the power tool for granular/raw SQL, index work, admin. Boundary holds:
  both open only the editable/read-appropriate schemas; neither is used to hand-edit
  `raw.*`/`working.*` evidence (ADR-0005/0007). All three (console, Mathesar, pgAdmin) point
  at the same `traceiq` DB, so it's one dataset seen through fit-for-purpose lenses.
- Supersedes the earlier "MapLibre" note in the Layout section above.

## Build order (each a standalone win)

1. Ask-and-it-shows retrieval (tools + chat + map + manual controls) — the "ship it" demo.
2. Graphiti ingestion of derived facts → cross-session memory + two-subject/exchange-day.
3. Legal-team handoff → automatic MCL-722.23 factor mapping with IRAC + citations.
The UI ships dual-use from day one regardless of how deep the agent layer goes.

## Alternatives Considered

- **Two-pane portal (Evidence + Mathesar behind one nav)**: fastest, but a visible seam and
  no agent story. Good interim, not the destination.
- **Evidence-shell + embedded editors (iframe Mathesar)**: mostly config, but Evidence is
  read-only by design — always bending a reporting tool to host editing; no native agent.
- **Off-the-shelf only, no custom UI**: fails the unified + dual-use + agent goals.

## Consequences

**Positive**: one surface; manual and AI users both first-class; agent is additive not
load-bearing; court-safety structural (receipts always attached).
**Negative**: real front-end build (deferred); a second agent stack to run when the hybrid
brain lands.
**Risks**: agent scope creep past "cite the deterministic row" → mitigated by the receipts
rule as a hard architectural boundary; Graphiti ingestion silently empty → node-count gate;
PII in knowledge graph → private-infra only, derived-facts-only, never raw evidence.

## Addendum — Evidence & Kepler drive mechanisms (2026-07-24, owner confirmed)

- **Evidence reports are driven from the console via URL parameters**, opened in a pop-out:
  `https://<evidence-host>/<report>?date_range=A..B&place=<id>`. Clicking a day/place/period
  builds that URL. "Live" = rebuild-on-ingest (Evidence regenerates when the working layer
  rebuilds — the deliberate milestone event of ADR-0010), which gives court-clean
  "current-as-of-last-rebuild" snapshot semantics; Evidence server-mode against live DB is
  the alternative. The `↗ analytics` door on results/detail targets this.
- **Custom Kepler visualizations are requested from the console (or the agent) and rendered
  in a pop-out**: the UI/agent builds the current query's GeoJSON (from DuckDB/PG) + a Kepler
  config (layers/filters/color-by/time-playback) and hands it to the Kepler pop-out via
  `addDataToMap`/postMessage or a config URL. Manual: "↗ Kepler" opens with the current
  selection pre-staged. Agent: "heatmap of her 2022 overnights" → generated config → pop-out.
  Kepler stays a launched-pre-configured pop-out, never embedded (it's heavy). The `↗ Kepler`
  button targets this.

## Status note

Proposed only. Companion static mockups in `docs/mockups/` (variants A/B/C) for owner review.
No app build, deployment, or agent wiring is committed by this ADR.
