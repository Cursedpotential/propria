-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Tables and role for message_dedupe_workflow (engine/dedupe): the removal of same-device duplicate
-- messages committed before the match-up rule (owner 2026-10-02 15:40 EDT, removal approved 16:00;
-- 20:02 "Make sure all these things get run as Temporal activities and are traceable").
--
--   working.message_dedupe_run             one frozen plan per dedupe_id
--   working.message_dedupe_copy            the copies it removes and the row each one keeps
--   working.message_dedupe_thread_version  the thread versions the copies sat in (recomputed)
--   working.message_dedupe_receipt         one receipt per step and run (dry or live)
--
-- The plan tables have no foreign keys on purpose: they are an audit record, and a foreign key to
-- working.normalized_record would take a lock that blocks live imports while this is applied.
--
-- Every mutating statement of the workflow runs under message_dedupe_writer (NOLOGIN), which
-- platform_runtime may SET but does not inherit, so the always-on worker holds no DELETE privilege
-- outside those transactions. Apply as platform_dba (it needs CREATEROLE) when no import is committing.

\set ON_ERROR_STOP 1
BEGIN;
SET LOCAL lock_timeout = '10s';

CREATE TABLE working.message_dedupe_run (
    dedupe_id   text PRIMARY KEY CHECK (dedupe_id ~ '^[A-Za-z0-9_-]{8,96}$'),
    rule        text NOT NULL,
    workflow_id text NOT NULL,
    run_id      text NOT NULL,
    copies      bigint,
    planned_at  timestamptz NOT NULL
);

CREATE TABLE working.message_dedupe_copy (
    dedupe_id                text NOT NULL REFERENCES working.message_dedupe_run (dedupe_id),
    record_id                uuid NOT NULL,
    keeper_record_id         uuid NOT NULL,
    projection_kind          text NOT NULL CHECK (projection_kind IN ('first_party', 'acquired_third_party')),
    match_key                text NOT NULL,
    source_version_id        uuid NOT NULL,
    keeper_source_version_id uuid NOT NULL,
    PRIMARY KEY (dedupe_id, record_id),
    CHECK (record_id <> keeper_record_id)
);
CREATE INDEX message_dedupe_copy_record_idx ON working.message_dedupe_copy (record_id);

CREATE TABLE working.message_dedupe_thread_version (
    dedupe_id         text NOT NULL REFERENCES working.message_dedupe_run (dedupe_id),
    family            text NOT NULL CHECK (family IN ('first_party', 'third_party')),
    thread_version_id uuid NOT NULL,
    PRIMARY KEY (dedupe_id, family, thread_version_id)
);

CREATE TABLE working.message_dedupe_receipt (
    id           uuid PRIMARY KEY,
    dedupe_id    text NOT NULL REFERENCES working.message_dedupe_run (dedupe_id),
    step         text NOT NULL CHECK (step IN ('plan', 'repoint_occurrences', 'remove_thread_memberships',
                     'recompute_thread_versions', 'remove_first_party_messages', 'remove_third_party_messages', 'verify')),
    dry_run      boolean NOT NULL,
    status       text NOT NULL CHECK (status = 'success'),
    counts       jsonb NOT NULL,
    workflow_id  text NOT NULL,
    run_id       text NOT NULL,
    attempt      integer NOT NULL,
    started_at   timestamptz NOT NULL,
    completed_at timestamptz NOT NULL
);
-- A live step succeeds once per plan; a retried or repeated live run re-reads that receipt.
CREATE UNIQUE INDEX message_dedupe_receipt_live_once ON working.message_dedupe_receipt (dedupe_id, step) WHERE NOT dry_run;

GRANT SELECT, INSERT ON working.message_dedupe_run, working.message_dedupe_copy,
    working.message_dedupe_thread_version, working.message_dedupe_receipt TO platform_runtime;
GRANT UPDATE (copies) ON working.message_dedupe_run TO platform_runtime;

CREATE ROLE message_dedupe_writer NOLOGIN;
GRANT USAGE ON SCHEMA working TO message_dedupe_writer;
GRANT SELECT ON working.message_dedupe_copy TO message_dedupe_writer;
GRANT SELECT, UPDATE (primary_record_id) ON working.message_occurrence TO message_dedupe_writer;
GRANT SELECT, DELETE ON working.first_party_context_thread_message, working.message, working.message_participant,
    working.message_projection_route, working.third_party_message, working.third_party_message_participant
    TO message_dedupe_writer;
-- Read-only, for the plan's check that no copy is held by a row the removal does not move.
GRANT SELECT ON working.third_party_context_thread_message, working.attachment, working.content_chunk_message,
    working.record_visible_from, working.event_source_record, working.realization_event, working.realization_event_record,
    working.walk_step, working.walk_step_retrieval TO message_dedupe_writer;
GRANT USAGE ON SCHEMA context, analysis, evidence TO message_dedupe_writer;
GRANT SELECT ON context.first_party_thread_message_relative_time_anchor, context.third_party_thread_message_relative_time_anchor,
    analysis.timeline_event, analysis.knowledge_evidence_promotion, evidence.evidence_item TO message_dedupe_writer;
GRANT message_dedupe_writer TO platform_runtime WITH INHERIT FALSE, SET TRUE;

COMMIT;
