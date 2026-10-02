# ADR-0011: 3-source geocoding blueprint; lean cache rows; provider-agnostic road-snapping

**Date**: 2026-07-24
**Status**: accepted
**Deciders**: Matt (owner — via structured questions), Claude (Claude Code · Fable 5)

## Context

Two geocoding designs coexist in the corpus: a simple provider-cache pair and an
elaborate 8-table 3-source design (request / per-provider results / resolution / audit /
location_key / event_geokey) with disagreement flags and manual overrides that reached
index level (idx_resolution_disagreement, idx_resolution_manual). OQ#12: owner once
rejected batch tracking on cache rows; it crept back. Snap-to-Roads was discussed but
never built; Google's Roads API is pricey.

## Decision

1. **The 3-source design is the blueprint — generalized to N providers.** Owner expects
   additional GPS/geo providers over time; the schema treats providers as an open set
   (a provider registry + per-provider result tables stamped from it), launching with
   radar / google / geodata. geocode_request → per-provider results (separate, never
   merged) → resolution step with disagreement flags + manual-override support → full
   audit. Adding a provider = registry row + result table + adapter, no schema surgery. Radar primary, Google
   Places fallback (place_id/business names only), Geodata third. Existing 50K+ Radar
   cache migrates in as evidence (prior API responses). Cache-first lookup via the
   ADR-0009 multi-precision keys.
2. **Cache rows stay lean** (OQ#12 closed per owner's original rejection): no
   batch_file/batch_timestamp on cache rows — the jobs table + API audit trail carry
   batch provenance.
3. **Road-snapping is IN scope, provider-agnostic by design**: a pluggable
   map-matching provider interface (same pattern as geocoding providers) — candidates:
   OSRM, Valhalla (open-source, free, self-hostable on owner's VPS), Mapbox map-matching
   (generous free tier); Google Roads only if ever needed. Possibly spread across
   providers/accounts for cost. Snapped coordinates land in separate columns/tables;
   raw GPS untouched (standing invariant). Forensic math runs on raw; snapping serves
   map exhibits.

## Alternatives Considered

### Simple cache pair
- **Pros**: fewer tables
- **Cons**: no structured disagreement resolution or manual override; retrofit cost later
- **Why not**: the 3-source design is what the corpus was already evolving toward and
  matches ADR-0003 evidence doctrine

### Google Roads API as the snapping provider
- **Pros**: canonical, well-documented
- **Cons**: pricey; vendor lock contradicts owner ruling
- **Why not**: free/self-hostable equivalents exist; provider interface keeps all doors open

## Consequences

### Positive
- Disagreements and manual corrections are first-class, auditable; OQ#12 closed;
  map exhibits get clean routes without forensic compromise
### Negative
- 8-table subsystem is the most complex part of the schema (accepted — it earned it)
### Risks
- Self-hosted OSRM/Valhalla needs road network data (OSM extracts) and some ops —
  evaluate effort vs Mapbox free tier at Phase C; decision is swappable by design
