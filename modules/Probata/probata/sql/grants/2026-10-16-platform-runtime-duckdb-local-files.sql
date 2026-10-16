-- Byline: Claude Code · Opus 5.5 · 2026-10-16
-- Owner decision 2026-10-16 (option A): let DuckDB structured ELT read retained
-- originals from the read-only mount /data/proffer/source-objects (deploy/data-pg.yaml).
--
-- pg_duckdb disables DuckDB's LocalFileSystem for every non-superuser that is not a
-- member of BOTH pg_read_server_files and pg_write_server_files
-- (pg_duckdb docs, settings.md, duckdb.disabled_filesystems). The proffer worker's
-- structured-ELT sessions log in as platform_runtime. The grant is made to that login
-- only -- not to the shared platform_duckdb group, which platform_api also joins.
--
-- Trade-off accepted by the owner: platform_runtime can read and write files the
-- PostgreSQL server process can reach. The retained-object mount itself is read-only.
--
-- Apply as a superuser on the platform database. Revoke with the REVOKE lines below.
GRANT pg_read_server_files TO platform_runtime;
GRANT pg_write_server_files TO platform_runtime;

-- REVOKE pg_write_server_files FROM platform_runtime;
-- REVOKE pg_read_server_files FROM platform_runtime;
