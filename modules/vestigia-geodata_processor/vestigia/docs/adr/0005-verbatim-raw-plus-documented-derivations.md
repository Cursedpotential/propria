# ADR-0005: Verbatim raw ingest + documented derivation layer (event model)

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

Old TraceIQ parsers decided at import time what a record "was" — whatever the parser
emitted was all that survived into the CSVs, so parse bugs destroyed evidence fidelity
and every schema debate (trip vs activity, column shapes) became an irreversible intake
decision. The rebuilt system must be defensible in front of a judge against expected
admissibility and methodology challenges.

## Decision

**Raw import imports; it does not interpret.** Ingest lands each source record exactly
as the JSON has it (JSONB, verbatim), plus registration facts *about the file* only:
chunk/export/subject references, byte offsets, hashes. No typing, no extraction, no
normalization at import.

**All structure is downstream transformation.** Typed tables, the canonical event model,
and every analysis are produced by versioned transformations (SQL views/materialized
tables/functions) over raw. Each transformation ships with:
- plain-language documentation of what it does and why (judge-readable),
- a worked example (specific raw input → specific output),
- a version stamp on every derived row (which transformation version produced it).
Any derived row is re-derivable from raw on demand — live, in court if needed.

**Carried-forward domain invariants** (now properties of the transformation layer,
provable by reconciliation counts):
- Never-unknown / no silent drops: every raw record maps to ≥1 derived row OR a flagged
  exception; raw count = derived + exceptions, always demonstrable.
- Waypoint ordering by sequence, never timestamp (~1% duplicate timestamps in real data).
- Orphaned paths (~30%) adopted via synthetic parents at the derivation layer.
- Raw precision never overwritten anywhere downstream.

**Type harmonization is a view concern**: e.g. rare `trip` records stay `trip` in raw
forever; the normalized event model maps them into `activity` with a `raw_type`
breadcrumb (resolves OQ#11). Misc geo-export record shapes land raw regardless and gain
normalized mappings only when needed.

**Human-readable snapshots** (owner mandate): at milestone checkpoints (post-ingest,
post-enrichment, post-analysis, pre-hearing, on-demand), generate the master
human-readable export from views (recovered ~100-col spec as template) — timestamped,
milestone-labeled, SHA-256-hashed, registered in a snapshot registry, included in
ADR-0004 local backups. Snapshots are outputs, never inputs.

**Derivation tiers** (doctrine; detail in ANALYSIS_DEPENDENCIES.md): Tier 0 evidence →
Tier 1 self-contained derivations (incl. home-base candidates from pure location
statistics) → Tier 2 external-API enrichment → Tier 3 case-knowledge enrichment →
Tier 4 cross-subject. External sources refine and label; they never gate Tier 1.

## Alternatives Considered

### Parse-time typing/normalization (old model)
- **Pros**: fewer layers; data "ready" immediately
- **Cons**: parser bugs corrupt evidence; schema debates become intake decisions;
  proven failure (silent under-extraction incident)
- **Why not**: interpretation at intake is an editorial act on evidence

### Raw + one fixed normalized schema (no versioned transformations)
- **Pros**: simpler than versioning
- **Cons**: can't prove which logic produced a row; fix-and-rederive becomes
  fix-and-pray; methodology challenges land on undocumented code
- **Why not**: judicial explainability requires versioned, documented, reproducible steps

## Consequences

### Positive
- Court demo: any exhibit row re-derived live from hashed raw JSON
- Parser/mapper fixes = re-derive + diff; evidence untouched
- Old schema debates (trip, column shapes) become cheap view changes
### Negative
- Discipline cost: every transformation needs docs + example + version (accepted; it IS
  the court-defensibility)
### Risks
- Transformation sprawl → mitigated: transformations live in the repo as migrations/
  versioned SQL with the docs beside them; ADR per significant methodology change
