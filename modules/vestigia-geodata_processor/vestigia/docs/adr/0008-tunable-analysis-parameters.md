# ADR-0008: Dual-clock time + tunable, audited analysis parameters

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

Old docs disagreed on the overnight window (22:00–07:00 vs –08:00) and flagged the
simple hour-check as inadequate (no true timezone handling, no multi-day spans, no
home-base awareness) — OQ#4. Similar buried constants exist elsewhere (multi-device
100m, impossible-speed 120–200mph). Hardcoded analysis constants are indefensible in
court and unfixable without code changes.

## Decision

1. **Dual-clock storage** on working rows: UTC + local-time-where-the-phone-was
   (offline tz-from-coordinates) + Eastern display time.
2. **Analysis rules are parameterized, not hardcoded.** Tunables (overnight window,
   multi-device distance, speed thresholds, geofence radii, time tolerances) live in a
   parameters table with standardized defaults (overnight default 22:00–07:00 local).
   Views/transformations on the working table read parameters; changing a parameter
   changes the analysis without touching code.
3. **A parameter tool** (API/CLI verb, later UI) lets the owner adjust tunables.
   Every change is logged (old value, new value, when, by whom). Exhibits/snapshots
   record the exact parameter values they were generated under.
4. **Definitions version and coexist**: overnight_simple(window) at Tier 1 now;
   home-base-aware overnight_v2 (asleep-where, multi-day spans) once Tier-3 labels
   exist; both queryable side by side.
5. **Sensitivity checks before exhibits**: for parameterized classifications, compute
   whether plausible parameter nudges (07:00 vs 08:00, 100m vs 150m) flip any
   conclusion; fragile conclusions get flagged before opposing counsel finds them.
6. **Owner mandate**: analyses are invited to get smarter than their old versions —
   the architecture now supports advanced variants cheaply (T11 will exploit this).

## Alternatives Considered

### Hardcoded constants (old model)
- **Pros**: simple
- **Cons**: indefensible ("why 22:00?" → "it was in the code"), unfixable without
  redeploy, caused the OQ#4 flip-flop
- **Why not**: parameters + audit trail cost little and answer the courtroom question

### Single blessed overnight definition
- **Pros**: one answer
- **Cons**: forces the old fight; loses the simple-vs-smart comparison that itself
  has evidentiary value
- **Why not**: coexisting versioned definitions are free under ADR-0005/0007

## Consequences

### Positive
- OQ#4 dissolves; parameter fights become sensitivity analyses; exhibits self-document
### Negative
- Parameters table + audit is one more moving part (small)
### Risks
- Parameter sprawl → each tunable must be documented (name, meaning, default,
  plausible range) as part of its transformation's judge-readable docs
