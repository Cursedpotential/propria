# ADR-0012: Analysis first wave (overnight/home-base + anomalies); home_base durability

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner — via structured question), Claude (Claude Code · Fable 5)

## Context

~36 recovered views span place analytics, overnight, home-base, schedule verification,
deception, problematic locations, anomaly dashboards. home_base was historically rebuilt
3+ times and lost in AI-session handoffs (OQ#8). Owner mandate: analyses get smarter
than their old versions (ADR-0008 §6).

## Decision

**Wave 1** (Tier-1, runs on her export with zero external input):
- Overnight + home-base lane: place analytics (visit counts, avg/min/max duration,
  first/last visit, overnight counts + percentage per place), home-base candidate
  clustering, overnight_simple + (later) home-base-aware overnight_v2.
- Anomaly suite: impossible travel, bouncy/boomerang trips, excessive detours,
  micro-trips, gap analysis, multi-device v1, and the NEW intra-path waypoint-gap
  unrecorded-stop detection + anomalous-point cross-comparison (owner-specified v2 lane;
  corpus evidence idx_slow_points/idx_waypoint_speed).

**Wave 2** (needs owner Tier-3 reference data): schedule verification / deception
detection (custody order times, claims ledger), problematic locations/contacts.

**home_base durability (OQ#8 closed structurally):** home_base and all reference tables
are migration-managed tables in the durable PG core (ADR-0001/0002); views depending on
them live in versioned transformation code in the repo. The historical loss mode
(CSV/session handoffs) no longer exists.

Immature-but-good recovered views (heatmap, daily timeline, hourly patterns, location
frequency, radar match quality) queue behind the waves as smarter-lane candidates.

## Consequences

### Positive
- Wave 1 delivers court-relevant output (where she slept, travel anomalies) immediately
  post-ingest; Wave 2's external-input needs are explicit (ANALYSIS_DEPENDENCIES.md)
### Negative
- Schedule/deception exhibits wait on owner homework (claims ledger, order times)
### Risks
- Smarter-v2 scope creep → v1 parity first (old view logic reproduced), v2 additions
  as separate versioned definitions
