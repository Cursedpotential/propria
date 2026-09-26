# ADR-0010: Built-in job runner, dependency-graph stages, full rebuilds

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner — via structured question), Claude (Claude Code · Fable 5)

## Context

Old pipeline = sequential pass scripts (P1→P4.5) whose ordering flip-flopped across
sessions (OQ#1: analytics-before-linking vs after). ADR-0001/0005/0007 make the database
the medium: "passes" become transformations over it. Something must run ingest jobs,
working-table rebuilds, and enrichment batches.

## Decision

1. **Orchestration = a built-in job runner inside the TraceIQ API service**: a jobs table
   (queue, status, logs, retry) + runner loop; jobs triggered via API/CLI/agent (ADR-0001
   peers). No new infrastructure.
2. **Stage order is a dependency graph, not a sequence.** Each transformation declares
   its inputs (tiers per ANALYSIS_DEPENDENCIES.md); execution order falls out. OQ#1
   dissolves — "linking vs analytics first" is answered by declared dependencies.
3. **Full rebuild on change** (new export, parser/mapper fix): working layer rebuilds
   from scratch (~minutes at low-millions rows), reconciliation counts prove consistency,
   then milestone backup (ADR-0004). Incremental processing is deferred until full
   rebuild ever becomes too slow.

## Alternatives Considered

### Dedicated orchestrator (Prefect/Dagster)
- **Pros**: real DAG tooling, UI
- **Cons**: a whole service to operate for a handful of batch jobs on one dataset
- **Why not**: right-sizing; can adopt later if job complexity grows

### n8n (already running)
- **Pros**: zero new infra, visual flows
- **Cons**: splits pipeline logic across two systems; core logic must stay in the API
- **Why not**: n8n may still *schedule/trigger* API jobs later, but is not the runner

### Incremental updates from day one
- **Pros**: faster per run
- **Cons**: change-tracking complexity + consistency edge cases the court story doesn't need
- **Why not**: full rebuild is provably consistent and fast enough at this scale

## Consequences

### Positive
- OQ#1 closed structurally; one service to operate; every run auditable in the jobs table
### Negative
- Job runner is custom code (kept minimal: table + loop + API verbs)
### Risks
- Long-running geocode batches need resumability → jobs checkpoint at batch level
  (persisted API responses per ADR-0003 make resume natural)
