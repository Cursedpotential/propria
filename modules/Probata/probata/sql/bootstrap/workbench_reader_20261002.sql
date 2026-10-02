-- Byline: Claude Code · Sonnet · 2026-10-02
-- The Workbench's own read-only login on the platform database.
--
-- Why: the mobile Imported view (Workbench /m) reads what has been imported -- sources, threads,
-- messages, calls -- straight from context.* and working.*. Owner option A (2026-10-02): every
-- client gets its own login, ai is the admin login only. This role is SELECT-only on the exact
-- tables the Imported view reads, read-only by default, with a statement timeout.
--
-- Apply:  psql -U ai -d platform -v pw="$(cat <the generated password file>)" -f this-file
-- The password is never in this file. On the ovh-app host it lives in
-- /data/probata/secrets/workbench/pg-reader (mode 0400), bound into the Workbench container as
-- /run/secrets/workbench-pg-reader (deploy/workbench.yaml).
--
-- Additive and idempotent: a re-run only refreshes the password and re-grants.

SELECT NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'workbench_reader') AS need_create \gset
\if :need_create
CREATE ROLE workbench_reader LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION CONNECTION LIMIT 12;
\endif
ALTER ROLE workbench_reader WITH PASSWORD :'pw';
ALTER ROLE workbench_reader SET default_transaction_read_only = on;
ALTER ROLE workbench_reader SET statement_timeout = '8s';
ALTER ROLE workbench_reader SET lock_timeout = '1s';
ALTER ROLE workbench_reader SET idle_in_transaction_session_timeout = '10s';

GRANT CONNECT ON DATABASE platform TO workbench_reader;
GRANT USAGE ON SCHEMA context, working, registry TO workbench_reader;

GRANT SELECT ON TABLE
    context.source,
    context.source_version,
    context.raw_record_identity,
    context.normalized_generation,
    context.normalized_record_identity,
    context.proffer_preview_snapshot,
    context.proffer_preview_decision,
    working.first_party_context_thread,
    working.first_party_context_thread_version,
    working.first_party_context_thread_source,
    working.first_party_context_thread_message,
    working.third_party_context_thread,
    working.third_party_context_thread_message,
    working.message,
    working.message_participant,
    working.third_party_message,
    working.third_party_message_participant,
    working.message_projection_route,
    working.call_log,
    registry.vw_case_identifier
TO workbench_reader;
