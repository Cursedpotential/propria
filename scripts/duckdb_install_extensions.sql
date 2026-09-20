-- Byline: Claude Code · Fable 5.1 · 2026-09-20
--
-- Register the DuckDB extensions the proffer ELT needs so pg_duckdb autoloads
-- them at session start. Run as a SUPERUSER: the engine role cannot install or
-- load extensions (pg_duckdb disables LocalFileSystem for non-superusers).
-- Idempotent. Registered rows live in duckdb.extensions.
--
--   webbed (community) — read_xml, used by the SMS Backup & Restore template.
SELECT duckdb.install_extension('webbed', 'community');
SELECT * FROM duckdb.extensions;
