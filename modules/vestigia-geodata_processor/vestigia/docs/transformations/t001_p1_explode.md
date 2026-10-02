# Transformation t001: P1-Explode (version 1)

> _Byline: Claude Code · Fable 5 · 2026-07-24 · Plain-language documentation per ADR-0005: what this step does, why, and a worked example. SQL: `db/transformations/t001_p1_explode_v1.sql`._

## What it does, in plain language

Google's export stores each timeline entry as a block of JSON text. This step reads every
block **exactly as Google produced it** (the block itself is never altered and remains in
the database), and copies its facts into organized table columns so they can be sorted,
filtered, and analyzed:

1. **Type**: whether the block is a visit (staying at a place), an activity (moving
   between places), or a path (a breadcrumb trail of GPS points). Google's rare
   "timelineMemory" blocks are treated as activities, and the original name is kept in
   its own column so nothing is hidden.
2. **Times**: the start and end times Google recorded, kept three ways — universal time
   (UTC), the local time *where the phone was* (using the timezone offset **Google itself
   attached to each record** — we do not guess timezones), and Eastern time for reading.
3. **Location**: the exact coordinates, untouched at full precision, plus a map point for
   distance/area calculations, plus rounded copies used only to group nearby lookups.
4. **Extras**: the Google place identifier, Google's own confidence number, Google's
   stated travel distance, and for paths, every GPS point in **Google's stated order**
   (never re-sorted by clock time, because real data contains duplicate timestamps).
5. **Flags computed here**: whether the entry touches the overnight window (a setting
   stored in a table, currently 22:00–07:00, changeable and audited — not buried in
   code), the time gap since the previous entry, and straight-line distance.

## The accounting rule (why you can trust it)

Every single raw record ends up in exactly one of two places: the events table, or an
exceptions table with a written reason. After every run, the totals are recorded and must
balance: raw records = events + exceptions, with zero unaccounted. Run of 2026-07-24:
41,963 = 40,320 + 1,643, difference **0**. The 1,643 exceptions are records outside this
step's scope (a test batch and 2 profile records), each with its reason stored.

## Worked example (real record, abridged)

Raw block (verbatim in the database):
```json
{"startTime":"2018-02-03T18:23:33.000-05:00","endTime":"2018-02-03T20:30:11.000-05:00",
 "startTimeTimezoneUtcOffsetMinutes":-300,
 "visit":{"probability":0.85,"topCandidate":{"placeId":"ChIJPXrV4aOHI4gR7TEyxVedwKQ",
          "placeLocation":{"latLng":"43.0125878°, -83.6875407°"}}}}
```
Becomes one event row: type `visit` · start (Eastern) `2018-02-03 6:23 PM` · end
`8:30 PM` · duration 2h 7m · coordinates `43.0125878, -83.6875407` (exact) · rounded key
`43.0126, -83.6875` · place `ChIJPXrV…` · confidence `0.85` · overnight flag `false`
(does not touch 22:00–07:00) · display label `1802036v`. Anyone can re-run the step and
obtain the same row from the same raw block.

## Known limits of version 1 (honest list)

- Overnight flag is the simple clock-window version; the home-base-aware version is a
  planned refinement (both will coexist; every output states which was used).
- Multi-device splitting and path-time inference (3-tier) are computed in a later step.
- Record fingerprints currently hash a normalized copy of each block; hashing the exact
  original byte run is a planned upgrade (the whole-file fingerprint already covers the
  original bytes end-to-end).
- Google's activity-type confidence is 0.0 in this export vintage (verified against raw
  blocks) — activity *type* labels should not be presented as attested; times, distances,
  and coordinates are unaffected.
