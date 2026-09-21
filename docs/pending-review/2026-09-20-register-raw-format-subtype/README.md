---
tags: [probata, sql, proffer, receipt]
---

# Receipt - `context.register_raw_format_subtype(text)` - 2026-09-20

> _Byline: Claude Code · Fable 5.1 · 2026-09-20_

- **Why:** the first run ever to pass the DuckDB parse stage (`vault-e2e-thread-20260920-2045`, a derived SMS thread chunk) failed at `persist_raw_generation_activity`: `function context.guard_raw_subtype_insert() does not exist`. The register function created a trigger on every new `context.raw_<format>` table that calls that guard. The guard is one of 27 `context.*` custody functions present in the retired migrations (`sql/_stale/migrations-retired-20260907/0036_...`) and absent from both snapshots and the live database - dropped by the D-152 rebuild (owner 2026-09-07: "get the database rebuilt without it").
- **Change:** that one trigger block removed; nothing else. `raw_subtype_append_only` (-> `context.forbid_mutation()`, which exists) stays.
- **Files:** `previous_definition_20260920.sql` (live text before), `new_definition_20260920.sql` (applied); the same edit is in `sql/bootstrap/schema_snapshot_20260907.sql`.
- **Proof before apply:** inside `BEGIN ... ROLLBACK`: function replaced, `register_raw_format_subtype('ndjson')` returned `context.raw_ndjson` with 1 trigger, rollback left no table.
- **Owed at promotion time:** the 27 guards (open-generation gates, append-only and hash-manifest guards) come back together, not one at a time.
