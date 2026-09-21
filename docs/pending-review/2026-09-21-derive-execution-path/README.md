---
title: Live change — context.handler_compatibility admits execution_path 'derive'
date: 2026-09-21
status: APPLIED live 2026-09-21 00:16 EDT on owner approval ("yes go", 00:09)
domains: [probata, engine, postgres]
tags: [receipt, schema, handler-compatibility, derive, live-change]
---

# Receipt: `handler_compatibility_execution_path_check` widened

> _Byline: Claude Code · Fable 5.1 · 2026-09-21._

- **Where:** database `platform`, container `probata-db-w10gg3an43jvry4y79n6sxi1-…` on ovh-files, run as `ai`.
- **What:** `CHECK (execution_path = ANY (ARRAY['decoder','duckdb']))` -> `… ARRAY['decoder','duckdb','derive']`. Script: `applied.sql` (copy of `scripts/2026-09-20-derive-execution-path.sql` from branch `feat/derive-sms-activity`, Claude Code · Opus 5).
- **Proof order:** (1) same script with `ROLLBACK` — printed the new definition, then the live definition read back unchanged; (2) applied with `COMMIT`; (3) read back as `platform_runtime`: new definition present; 15 existing rows, 0 outside `decoder`/`duckdb` (nothing invalidated).
- **Effect on running code:** none. Nothing on main emits `derive` yet; the change only widens what is accepted.
- **Undo:** re-create the two-value CHECK (valid while no `derive` rows exist).
- **Snapshot:** `sql/bootstrap/schema_snapshot_20260907.sql` carries the same constraint on the branch, NOT yet on main — until the branch merges, a rebuild from main's snapshot would revert this. Tracked in `docs/planning/2026-09-20-TODO.md`.
