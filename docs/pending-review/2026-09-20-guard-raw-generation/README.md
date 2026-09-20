---
tags: [probata, proffer, sql, receipt]
---
# `context.guard_raw_generation_transition()` applied to the live platform DB — 2026-09-20

> _Byline: Claude Code · Fable 5.1 · 2026-09-20. Owner go 12:55 EDT._

- Before: 3,335 chars (kept verbatim in `previous_definition_20260920.sql`, md5 of the file `e09e7fdc…`). After: the definition in `sql/bootstrap/schema_snapshot_20260907.sql`, 7,156 chars, function md5 `32133d5e17ecc847f9fdb9a81b82881a`, identical to the md5 seen inside the earlier rolled-back transaction. `context.raw_generation` had 0 rows before and after.
- Restore: run `previous_definition_20260920.sql` as `CREATE OR REPLACE`.
- **Correction to my own earlier claim:** I said the old function "will block the raw-generation stage". It would not have. Neither the snapshot nor the live DB defines any trigger that calls this function (checked: no `CREATE TRIGGER` references it; `context.raw_generation` has no user triggers). The live DB now matches the snapshot, but this function is not on any execution path. Whether a trigger is missing from the snapshot, or the function is leftover, is open.
