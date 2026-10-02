# ADR-0016: Path segmentation, overnight definition, verification scope, "geodata" clarified

**Date**: 2026-07-25
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Opus 5)

## Context

Four questions were open or ambiguous going into the UI/geocoding phase. Three came from
contradictions **between the legacy corpus docs themselves**, recovered by analysis of
`traaceiq_mess` (`_unfuck/`, `_salvage/`, `Example folder/`, `.claude/memories/`); one was
the owner's own unresolved challenge recorded at the end of the 2026-07-24 session, left
mid-build as "started t004 path-time (tier-1), paused."

1. **Overnight window** — three incompatible definitions across `EXTRACTED_CONCEPTS.md`,
   `processing_directives.md` §4 ("Finalized Field Mapping", 2025-12-09), and
   `google_timeline_addendum_part2.md` §K.
2. **Path segmentation** — t004 corrects `timeline_path` times from waypoints, but Google's
   ~2h path *container* grouping was still implicitly treated as meaningful. Owner:
   "I don't necessarily believe that the two hour window has [multiple things] inside...
   I think each path is between each activity and those paths are isolated to a particular
   [device/segment]."
3. **Verification scope** — `google_timeline_addendum.md` §B proposed cross-validating only
   "forensically significant" points (within 500m of home/school/custody-exchange) as a
   cost lever. Owner ruled this inaccurate.
4. **"geodata"** — ADR-0011 lists `geodata` as the third geocoding source alongside radar
   and google, and `geo.result_geodata` exists as a per-provider result table (0 rows).

## Decision

1. **Overnight = crosses midnight, OR falls in 22:00–08:00 Eastern and lasts > 1 hour.**
   (`google_timeline_addendum_part2.md` §K wins; the other two definitions are retired.)
   Eastern is the evaluation timezone, consistent with the display standard below.
2. **One corrected path per journey; Google's ~2h container windows carry no meaning.**
   A path is bounded by the activities around it and derives its times and locations from
   its actual waypoints (t004 tier 1). The container timestamps stay verbatim in
   `raw.record` as evidence but are never used analytically.
   **Known open issue: dual-device conflation.** A single corrected path may currently
   merge waypoints from two devices. This is accepted for now and MUST be revisited
   before exploded paths are used as exhibits — see `working.event.multi_device_split`,
   `device_index`, and `working.waypoint.device_index`.
3. **No verification gating. Every location key gets a fresh provider reverse geocode**,
   regardless of cache coverage — the legacy Radar cache (49,250 rows) is prior evidence,
   not an exemption. The 500m "forensically significant" filter is rejected outright.
4. **"geodata" is a generic reference to the geo data itself, not a vendor.** ADR-0011's
   third-source slot is therefore unfilled, not implemented-as-`geodata`. Real third
   providers get registered as data per ADR-0014 (candidates: Mapbox, LocationIQ,
   Geoapify, OpenCage, BigDataCloud, HERE). `geo.result_geodata` is a misnomer and is
   retired (empty, so no data is lost).

**Viewing standard (restated, was implicit):** imperial distances (miles/feet) and US
Eastern time in every human-facing surface. Storage stays SI/UTC — meters in
`working.*`, UTC plus the pre-computed `*_eastern` columns — and conversion happens at
display only.

## Alternatives Considered

### Keep the 500m significance gate to control provider spend
- **Pros**: far fewer paid calls across 17,467 location keys
- **Cons**: owner states the premise is inaccurate; leaves most of the record
  single-sourced, and un-cross-validated points are exactly what opposing counsel probes
- **Why not**: rejected by owner; cost is controlled instead via cache-first lookup,
  r4-precision dedup, and per-provider free tiers

### Treat Google's 2h path container as the path boundary
- **Pros**: no transformation needed; matches the source's own grouping
- **Cons**: container times are ~2h-rounded (a 5-minute trip reports as 2h), so both
  duration and sequencing come out wrong
- **Why not**: demonstrably false timing; t004 already proved waypoints give real bounds

## Consequences

### Positive
- One authoritative overnight rule; path timing derived from evidence rather than Google's
  rounding; verification coverage is uniform and defensible
### Negative
- Full-corpus verification costs more API calls than a gated approach; dual-device
  conflation remains a known defect in exploded paths
### Risks
- Exploded-path exhibits are NOT safe until dual-device conflation is resolved (decision 2)
- Retiring `geo.result_geodata` leaves ADR-0011's third-source slot genuinely empty until a
  real provider is registered
