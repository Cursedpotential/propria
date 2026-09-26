# ADR-0001: Database-centric core with API-first access (CSV pipeline retired)

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

TraceIQ's prior incarnations processed Google Timeline evidence through sequential pass
scripts communicating via intermediate CSVs. This welded business logic to file layouts,
made every AI-session handoff lossy (home_base and its views were rebuilt 3+ times),
forked backends and frontends (2 DB stacks, 4 UIs), and made "where is truth right now"
unanswerable. The data is court evidence in an active custody case: loss, overwrite, or
untraceable derivation is unacceptable.

## Decision

We build one durable, schema-as-code database as the application core. All data — raw
parsed events, saved API request/responses, and derived/analysis results — lands in it,
with content hashes on evidence and full source-traceability on derivations; nothing is
dropped. Views and indexes are built over this base at will and are always regenerable.
A single API layer fronts the database; web UI, CLI, and agents are all peers driving
that same API. File formats (CSV/Parquet/KML/GeoJSON) exist only as exports, never as
processing state.

## Alternatives Considered

### A: Keep the pass-script + CSV pipeline (status quo)
- **Pros**: familiar; matches salem_v1 scaffold; simple to inspect mid-stage
- **Cons**: proven failure — state fragmentation, session-handoff loss, logic welded to file formats
- **Why not**: it is the thing being escaped; owner explicitly rejected it

### B: Script library + ad-hoc per-task databases
- **Pros**: flexible, low ceremony
- **Cons**: reproduces today's mess (many part-filled DBs litter the corpus); no single truth
- **Why not**: fragmentation is the disease, not a feature

### C: Database-centric core, API-first (chosen)
- **Pros**: one source of truth; handoff-proof; views disposable/rebuildable; UI/CLI/agent
  parity; evidence chain lives where the data lives
- **Cons**: more upfront schema discipline; migrations required; API layer to maintain

## Consequences

### Positive
- Any session/tool/agent finds the same truth; analysis becomes cheap (a view, not a pipeline stage)
- Chain of custody is structural; frontends become thin and swappable

### Negative
- Schema design quality is now load-bearing (T9 carries the weight)
- Raw-everything storage costs disk (acceptable — it is evidence)

### Risks
- Schema churn during rebuild → mitigated by migrations-as-code + an ADR per schema change
- API layer scope creep → mitigated by starting minimal (ingest triggers + query + CRUD)

**Explicitly deferred**: database engine choice (T9), table design (T9), API shape (T13),
pipeline stage semantics as DB transformations (T8).
