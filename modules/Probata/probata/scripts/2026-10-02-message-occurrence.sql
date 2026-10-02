-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Message match-up across sources (owner 2026-10-02: "both Facebook exports, deduped"; decision C, messages
-- present in more than one source match up). One working row per message; every further source that holds
-- the same message is recorded here as an occurrence of that row. Nothing is deleted; every source stays cited.
--
-- Match rule: the same platform, the same set of parties (resolved registry entity, else the registry-normalized
-- identifier, else the raw identifier), the same sender, the same sent time to the second and the same body
-- hash, from a different source version. Two phones (Matt's and Katrina's) that hold the same message resolve
-- to the same two people and so produce the same key: a cross-device match.
--
-- Applied the same way to the platform snapshot (sql/bootstrap/schema_snapshot_20260907.sql) and live.

CREATE FUNCTION working.message_match_key(p_platform text, p_parties text[], p_sender text, p_occurred timestamptz, p_body_sha256 bytea)
RETURNS text
LANGUAGE sql STABLE
SET search_path TO 'pg_catalog'
AS $$
    SELECT encode(sha256(convert_to(
        coalesce(p_platform, '') || '|' ||
        coalesce((SELECT string_agg(DISTINCT party, ',' ORDER BY party) FROM unnest(p_parties) AS party WHERE party <> ''), '') || '|' ||
        coalesce(p_sender, '') || '|' ||
        coalesce(to_char(date_trunc('second', p_occurred AT TIME ZONE 'UTC'), 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '') || '|' ||
        coalesce(encode(p_body_sha256, 'hex'), ''),
    'UTF8')), 'hex')
$$;

CREATE TABLE working.message_occurrence (
    normalized_record_id uuid NOT NULL,
    match_key text NOT NULL,
    source_version_id uuid NOT NULL,
    primary_record_id uuid NOT NULL,
    projection_kind text NOT NULL,
    perspective_person_id uuid,
    cross_device boolean DEFAULT false NOT NULL,
    deriver_version text NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT message_occurrence_pkey PRIMARY KEY (normalized_record_id),
    CONSTRAINT message_occurrence_match_key_check CHECK ((length(match_key) = 64)),
    CONSTRAINT message_occurrence_projection_kind_check CHECK ((projection_kind = ANY (ARRAY['first_party'::text, 'acquired_third_party'::text]))),
    CONSTRAINT message_occurrence_deriver_version_check CHECK ((length(deriver_version) > 0)),
    CONSTRAINT message_occurrence_primary_record_id_fkey FOREIGN KEY (primary_record_id)
        REFERENCES working.normalized_record(id) DEFERRABLE INITIALLY DEFERRED
);

COMMENT ON TABLE working.message_occurrence IS
    'One row per source occurrence of a message. normalized_record_id = primary_record_id is the source that committed the working row; any other row is a further source (another backup, export or phone) holding the same message, recorded instead of a second working row.';

CREATE INDEX message_occurrence_match_key_idx ON working.message_occurrence USING btree (match_key, recorded_at);
CREATE INDEX message_occurrence_primary_idx ON working.message_occurrence USING btree (primary_record_id);

GRANT SELECT, INSERT ON TABLE working.message_occurrence TO platform_runtime;
GRANT ALL ON TABLE working.message_occurrence TO platform_app;
GRANT EXECUTE ON FUNCTION working.message_match_key(text, text[], text, timestamptz, bytea) TO platform_runtime;
