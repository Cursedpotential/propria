# ADR-0009: Coordinate representation per layer; multi-device detection carried + upgradeable

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner), Claude (Claude Code · Fable 5)

## Context

OQ#3: old docs fought over separate lat/lng columns vs a single "lat,lng" string — a
CSV-era argument about one representation. The raw→working architecture (ADR-0005/0007)
removes the premise. Multi-device detection mechanism needed carrying into the new
working layer; corpus index `idx_slow_points`/`idx_waypoint_speed` shows waypoint-level
speed/dwell work existed.

## Decision

**Coordinates per layer:** raw keeps whatever the JSON had, verbatim (E7 integers,
degree-strings — untouched). Working tables carry numeric `lat`/`lng` (double precision)
AND a PostGIS geometry point (native distance/geofence/cluster queries) AND the r3–r6 +
geohash8/9 cache-key columns (ADR-0002 stack). Display strings live in views/snapshots
only. Raw precision never overwritten anywhere (standing invariant).

**Multi-device:** v1 mechanism carried into the working-layer build — duplicate
timestamp + impossible-distance ⇒ split into device_0/device_1 with `multi_device_split`,
`device_index`, `split_from_segment` breadcrumbs; distance threshold is an ADR-0008
tunable (default 100m, historical). v2 (smarter lane, T11): device-track clustering
across longer windows to catch second devices without exact-timestamp collisions.

## Alternatives Considered

### Single "lat,lng" string column (one old convention)
- **Pros**: matches some legacy docs
- **Cons**: kills native numeric/spatial indexing; string-parsing in every query
- **Why not**: display concern promoted into storage; belongs in views only

### Numeric columns without PostGIS geometry
- **Pros**: one less column
- **Cons**: every spatial query recomputes geometry; loses GiST indexing
- **Why not**: geometry column is the point of choosing PostGIS (ADR-0002)

## Consequences

### Positive
- OQ#3 dissolved; spatial queries native and fast; every historical representation
  still derivable for court display
### Negative
- Coordinate data stored ~3 ways in working rows (numeric, geometry, cache keys) —
  deliberate redundancy, cheap at this scale
### Risks
- Geometry SRID/projection discipline (WGS84) must be fixed at Phase C schema time and
  documented in the transformation docs
