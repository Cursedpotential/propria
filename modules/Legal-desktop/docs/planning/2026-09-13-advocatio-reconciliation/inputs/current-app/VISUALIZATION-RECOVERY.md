# Visualization and outside-project recovery — 2026-09-13

Read-only follow-up to the current-app audit. This inventory separates owner-retained choices, historical brainstorm candidates, donor code, and actual current application integration. No services, private corpora, databases or models were accessed. Timesketch fork implementation and original handoff packets belong to the parallel handoff lane and were not duplicated here.

## Strongest recovered choices

| Project/library | Intended purpose | Decision status | Current source presence |
|---|---|---|---|
| react-calendar-timeline | Lightweight operational/event timeline visualization | Explicitly retained alongside vis-timeline as a development option; engine chosen per concrete visualization, no forced replacement/bake-off | Probata modules/workbench/web/package.json:44 declares ^0.30.0-beta.19 under devDependencies; package-lock.json:9598 records package. No imports found in searched Workbench TypeScript/TSX; not in Advocatio current manifest/source. |
| vis-timeline + vis-data | Another retained timeline interaction model | Same retained option, not a superseded loser | Workbench package.json:52-53 declares vis-data ^8.0.5 and vis-timeline ^8.5.4 under devDependencies; lockfile root entries :45-46. No imports found in searched Workbench source; not integrated in Advocatio. |
| Evidence.dev | SQL/Markdown reports producing frozen citable outputs, separate from editable data grid | **Owner-retained standard** in D-129, 2026-09-02; re-establishing a platform-owned project is owed work, not a new proposal | Decision log records original project moved to traceIQ by owner order, commit 557294c; no current platform-owned project claimed by this audit. No Evidence.dev mount/use found in bounded current Workbench src/deploy search. TraceIQ historical content not audited. |
| Glide-class data grid | Live interactive evidence/document operation and human review | Named standard/queued integration at D-129 date; explicitly complementary to Evidence.dev | Historical D-129 says declaration only at that date. No claim about current adjacent sorter installation; separate work lane. |
| Storybook | Component workshop and shared visual/behavioral contract | Explicitly adopted; not a data/workflow authority | Workbench package.json:32-34, :46 declares Storybook React/Vite + accessibility/docs adapters. |

**Primary decision evidence:** Probata/probata/docs/awaiting-verification/WORKBENCH-STORYBOOK-AND-SURFACE-REFACTOR-2026-08-30.md:22-25 retains both timeline engines and keeps Timesketch as a distinct advanced governed projection. The same document's final “Still open” section requires selecting a lightweight engine per actual visualization and defers the Timesketch launch contract until its gates pass. Probata/probata/docs/DECISION_LOG.md:309 (D-129) explicitly retains Evidence.dev, distinguishes frozen reports from live grids, and records the tenant move and owed replacement project.

## Earlier outside-project integration inventory

The following list is documentary evidence of intent, **not six newly approved integrations**. It does not by itself establish that these are the user's exact remembered half-dozen. It provides concrete project names to reconcile with the handoff/fork lane.

| Outside project | Intended use | Source and status | Code/integration proof boundary |
|---|---|---|---|
| NeoDash | Neo4j/Graphiti entity networks, relationship timelines and incident maps | docs/planning/gui-integration-spec.md:113; candidate iframe /x/neodash/ in draft brainstorm | No NeoDash reference found in bounded current Workbench src or nonretired deploy manifests. |
| React Flow | Visualize workflow execution DAGs: custody -> parse -> store -> knowledge | gui-integration-spec.md:116; proposed in-app component G5 | No React Flow/xyflow dependency in current Workbench or Advocatio package manifest and no current imports found. |
| Surrealist | SurrealDB administration/inspection UI | gui-integration-spec.md:114; candidate embedded admin pane | Separate administration capability, not a factual-event timeline engine. No current Workbench mount found by bounded src/deploy search; no live service proof attempted. |
| Kepler.gl / Leaflet pattern | Geospatial analysis over PostGIS | gui-integration-spec.md:117, tied to visit-locations map PR #7 | Historical proposal. D-129 says visit-locations moved to traceIQ; do not copy it into Advocatio by inference. |
| Claude Code history viewer | Browse session histories; long-term native transcript viewer | gui-integration-spec.md:112; exact repository **TBC**; candidate d-kimuson/claude-code-viewer, InDate/claude-log-viewer, daaain/claude-code-log | Explicitly unresolved selection in this source. |
| CopilotKit generative UI | Charts/tables/timelines inside assistant responses | gui-integration-spec.md:115; draft platform-shell direction | Current Advocatio uses Vercel AI SDK and plain React; this draft platform plan does not establish Advocatio adoption. |
| Evidence.dev | Frozen reports | gui-integration-spec.md:111, corrected by D-129 | Stronger later retained decision described above. |

The broader embed inventory also names SBV, Attu, Neo4j Browser, ContextForge Admin, Kasm, agent-ui and historical LiteLLM UI. See gui-integration-spec.md:206 and docs/handoffs/HANDOFFS.md:185. These are tool surfaces, not all visualization libraries.

**Supersession caution:** gui-integration-spec.md:3-6 labels itself a draft owner brainstorm, dated July 4. Its single-proxy/JWT and LiteLLM assumptions are stale. docs/adr/0042-portkey-replaces-litellm-model-gateway.md:5-6 and :22 establish Portkey replacing LiteLLM. DECISION_LOG.md:313 (D-133) chooses OIDC provider/client validation and rejects the earlier forward-auth/reverse-proxy architecture for internal surfaces. Do not import that old integration topology wholesale.

## Verified retained donor code

The Legal-desktop resources/build-kit/donors directory contains actual donor directories including legal-terminal-main, legal-terminal-master, LIGHT-2-main, LexRAG-main, themis-main, legal-mcp-main and other research/skill packages. Directory presence proves preservation only; original handoffs determine intended integration scope.

One additional visualization library is concretely present in donor source:

- resources/build-kit/donors/LIGHT-2-main/LIGHT-2-main/frontend/package.json:20 declares **Recharts ^3.7.0**.
- Its frontend/src/components/ConfidenceRadar.tsx:2 imports Radar, RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis and ResponsiveContainer from Recharts.
- Intended local purpose is a confidence radar display, not a timeline/relationship graph. This is **donor implementation**, not an owner-selected Advocatio dependency.

Bounded searches of donor package.json/pyproject.toml/requirements.txt (excluding dependencies/virtual environments) did not find declarations for Cytoscape, Sigma.js, React Flow/xyflow, vis-network, D3, ECharts, Plotly or Mermaid. This is not a claim those names are absent from all archived material. Timesketch's “sigma” analyzers match security rules and must not be mistaken for Sigma.js graph visualization.

## Current app versus intended integration

Advocatio's current timeline remains hand-written React markup over /v1/docket-events, with court dates and timeline sharing saved context. No retained donor package or external timeline fork is connected merely because this page exists. The Platform Workbench's two installed-declaration timeline options are a better match for the user's remembered “multiple libraries” than inventing a new tool shortlist.

Propria/resources presently contains a design subdirectory, with resources/design/CALLABILITY.md:38 listing Timeline among product surfaces. That design vocabulary does not choose an engine. No current graph library selection emerged from the bounded Propria docs/resources and Probata ADR/catalog searches.

## Search and verification record

- Listed only immediate Propria, resources, Probata modules and donor directories to locate product boundaries.
- Queried Probata docs/ADRs/decision log and Propria docs/design resources for named visualization leads.
- Verified Workbench package manifest + lockfile entries; searched its source imports.
- Verified a retained donor manifest and actual Recharts import.
- Did not inspect Timesketch internals, private data, account/session corpora, or arbitrary node_modules trees.
- Missing initial DECISIONS.md/HANDOFFS.md guesses were corrected to docs/DECISION_LOG.md and docs/handoffs/HANDOFFS.md.
- No code or service behavior was executed. Historical deployment claims remain historical.

Paths above are relative to the identified product root: Probata references resolve under E:/AI_Workspace/Projects/Propria/Probata/probata; Legal-desktop donor references resolve under E:/AI_Workspace/Projects/Propria/Legal-desktop.
