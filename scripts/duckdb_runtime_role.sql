-- Byline: Claude Code · Fable 5.1 · 2026-09-20
--
-- pg_duckdb lets only superusers run DuckDB unless `duckdb.postgres_role` names
-- a role; that GUC was empty on the live server, so every proffer ELT stage
-- (the engine connects as platform_runtime) failed with
--   "DuckDB execution is not allowed because you have not been granted the
--    duckdb.postgres_role"
-- First seen 2026-09-20 on the first real vault file to reach the ELT stage.
--
-- This script is the cluster-level half (roles are not in the schema snapshot).
-- The server half is `-c duckdb.postgres_role=platform_duckdb` in
-- deploy/docker/postgres/Dockerfile; it is postmaster-context, so it takes
-- effect only after a PostgreSQL restart. Idempotent; safe to re-run.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'platform_duckdb') THEN
        CREATE ROLE platform_duckdb NOLOGIN;
    END IF;
END
$$;

GRANT platform_duckdb TO platform_runtime;
GRANT platform_duckdb TO platform_api;

-- The extension was created while the GUC was empty, so it granted nothing to
-- this role; give it what pg_duckdb grants at CREATE EXTENSION time.
GRANT USAGE ON SCHEMA duckdb TO platform_duckdb;
GRANT ALL ON ALL TABLES IN SCHEMA duckdb TO platform_duckdb;
GRANT ALL ON ALL SEQUENCES IN SCHEMA duckdb TO platform_duckdb;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA duckdb TO platform_duckdb;
