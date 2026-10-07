-- Reject the historical unreceipted drive-loading entry point.
-- Inputs: legacy psql invocation. Outputs: explicit failure and maintained loader route.
-- Effects: none; never creates tables, inserts generations or replaces current views.
-- Choose drive_inventory_load_20261006.py after receipt validation and parent lake-finalization release.
-- Replaces the reviewed MAIN Oct 2 loader by Claude Code / Opus 5.5; source remains in MAIN/history.
-- Byline: Codex / GPT-6 / 2026-10-06.
\set ON_ERROR_STOP 1
DO $guard$ BEGIN
 RAISE EXCEPTION 'Unreceipted legacy drive loading is disabled. Use drive_inventory_load_20261006.py validate/load/readback with the frozen listing receipt and production release. No catalog changes were made.';
END $guard$;
