# ADR-0007: Materialized working layer (physical Tier-1 tables, not views)

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

ADR-0005 makes all structure downstream transformations over verbatim raw JSONB. Pure
views over JSONB would re-shred JSON on every query — slow and awkward on a modest
shared PG instance, and the owner works these tables constantly ("easy and fast to work
with"). Owner mandate: the basic-transformation layer is a real table, not just a view
or index.

## Decision

The Tier-1 canon materializes as **physical working tables** (events + waypoints at
minimum; exact design in T9/Phase C), built by the versioned, documented Tier-1
transformations: waypoint sequence linking + adjustments, path timestamp inference,
orphan adoption, multi-device identification/split, distances + discrepancy flags,
timezone + Eastern local time, basic derived fields/flags, display serial.

Discipline that keeps it defensible:
- The working layer is **regenerable at will** from raw + transformation code; it is
  never a source of truth and never edited by hand.
- Every row carries transformation-version stamps; rebuilds happen as a unit;
  reconciliation counts (ADR-0005) prove completeness after every rebuild.
- Rebuild is a "major operation" per ADR-0004 → ends with a verified local backup.
- Analysis views (Tier 1+ views, Tier 2/3 joins) sit on the working tables; individually
  hot analyses may also materialize, case by case.

Resulting physical architecture: **raw (evidence) → working (materialized canon) →
analysis (views, selectively materialized) → snapshots/exports (artifacts)**.

## Alternatives Considered

### Views-only over raw JSONB
- **Pros**: zero staleness risk, minimal storage
- **Cons**: JSONB shredding on every query; painful ad-hoc SQL; slow on shared instance
- **Why not**: owner works in this layer daily; ergonomics and speed are requirements

### Materialized views (PG native) instead of tables
- **Pros**: built-in refresh semantics
- **Cons**: all-or-nothing refresh, no incremental control, weaker indexing/partitioning
  control, awkward version-stamp handling
- **Why not**: plain tables + explicit rebuild transformations give the control and
  auditability ADR-0005 demands (kept as an option for individual hot analyses)

## Consequences

### Positive
- Fast, ergonomic daily querying; analysis views get simple typed columns
- Rebuild-from-raw remains the court demo (ADR-0005) — materialization changes nothing
### Negative
- Storage duplication raw↔working (acceptable; low-millions of rows)
- Staleness possible between raw ingest and rebuild → mitigated: ingest jobs trigger
  working-layer rebuild as part of the job (job not done until rebuilt + backed up)
