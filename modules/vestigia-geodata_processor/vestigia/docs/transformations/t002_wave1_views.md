# Transformation t002: Wave-1 Analysis Views (version 1)

> _Byline: Claude Code · Fable 5 · 2026-07-24 · Plain-language documentation per ADR-0005. SQL: `db/transformations/t002_wave1_views_v1.sql`. Views are re-runnable summaries — they store nothing and can be regenerated identically from the tables at any time._

## What these views do

All views read from a shared base (`latest_events`) which first **removes double-counting**:
her export survives as two byte-variant copies, both loaded as evidence; the base view
keeps one representative per record fingerprint (the duplicates *corroborate* each other —
see the corroboration proof: 20,161 of 20,161 fingerprints identical across the copies).

- **place_analytics** — for every place: how many visits, average/shortest/longest stay,
  first and last visit dates, how many stays touched the overnight window and what
  percentage. This is the "where does she actually live/frequent" table.
- **overnight_events** — every overnight-flagged stay, listed chronologically with
  duration and place.
- **daily_coverage / monthly_coverage** — how many entries exist per day and month; the
  gap-hunting tables (this is how the 9-month 2020 hole was found).
- **speed_anomalies** — GPS points implying travel faster than the impossible-speed
  setting (stored parameter, currently 120 mph): device glitches or data problems, listed
  for review, never silently removed.
- **dwell_stops** — moments *inside a trip* where the GPS went quiet for at least the
  dwell threshold (stored parameter, 15 min) while the position barely moved (<100m):
  candidate unrecorded stops.

## Worked example

Question: "How often did she stay overnight at place X?" → `place_analytics` row for
place `ChIJezMgkaaHI4gR…`: 823 visits, 325 overnight, first 2018-11-17, last 2022-05-28.
Every number traces back through the events table to specific verbatim raw records, each
re-derivable live.

## Version-1 limits

- Overnight counts use the simple clock-window flag (see t001 doc).
- Places are grouped by Google place-ID (and rounded location where no ID exists);
  place *names/addresses* arrive with the geocoding stage.
- Duration statistics count visits with both start and end times; entries missing either
  are excluded from averages (never from the record itself).
