-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Seed working.message_occurrence for every message committed before the match-up rule existed: each committed
-- working row becomes the primary occurrence of itself, keyed exactly as commit_first_party_messages keys new
-- messages (working.message_match_key over platform, parties, sender, time to the second, body hash).
-- Idempotent (ON CONFLICT DO NOTHING). Writes nothing else and deletes nothing.
-- Afterwards, the report at the end lists the messages already committed twice from different sources:
-- the duplicates the owner's collapse plan covers (quarantine, never delete).

BEGIN;
SET LOCAL search_path TO pg_catalog, public;

WITH first_party AS (
    SELECT m.id, m.platform, m.ts_utc AS occurred_at, m.content_sha256, r.source_version_id, r.derived_at,
           nullif(split_part(r.conversation_id, '/', 5), '')::uuid AS perspective,
           array_agg(coalesce(p.entity_id::text, 'norm:' || nullif(registry.norm_identifier(p.participant_raw), ''),
                              'raw:' || lower(btrim(p.participant_raw)))) AS parties,
           max(coalesce(p.entity_id::text, 'norm:' || nullif(registry.norm_identifier(p.participant_raw), ''),
                        'raw:' || lower(btrim(p.participant_raw)))) FILTER (WHERE p.role = 'from') AS sender
    FROM working.message m
    JOIN working.normalized_record r ON r.id = m.derived_from_record_id
    LEFT JOIN working.message_participant p ON p.message_id = m.id
    WHERE r.source = 'proffer' AND r.source_version_id IS NOT NULL
    GROUP BY m.id, m.platform, m.ts_utc, m.content_sha256, r.source_version_id, r.derived_at, r.conversation_id
)
INSERT INTO working.message_occurrence
    (normalized_record_id, match_key, source_version_id, primary_record_id, projection_kind, perspective_person_id,
     cross_device, deriver_version, recorded_at)
SELECT id, working.message_match_key(platform, parties, sender, occurred_at, content_sha256), source_version_id, id,
       'first_party', perspective, false, 'engine:message_occurrence_backfill@1.0.0', coalesce(derived_at, now())
FROM first_party
ON CONFLICT (normalized_record_id) DO NOTHING;

WITH third_party AS (
    SELECT m.id, m.platform, m.occurred_at, m.content_sha256, r.source_version_id, r.derived_at,
           nullif(split_part(r.conversation_id, '/', 5), '')::uuid AS perspective,
           array_agg(coalesce(p.entity_id::text, 'norm:' || nullif(registry.norm_identifier(p.participant_raw), ''),
                              'raw:' || lower(btrim(p.participant_raw)))) AS parties,
           max(coalesce(p.entity_id::text, 'norm:' || nullif(registry.norm_identifier(p.participant_raw), ''),
                        'raw:' || lower(btrim(p.participant_raw)))) FILTER (WHERE p.role = 'from') AS sender
    FROM working.third_party_message m
    JOIN working.normalized_record r ON r.id = m.normalized_record_id
    LEFT JOIN working.third_party_message_participant p ON p.message_id = m.id
    WHERE r.source = 'proffer' AND r.source_version_id IS NOT NULL
    GROUP BY m.id, m.platform, m.occurred_at, m.content_sha256, r.source_version_id, r.derived_at, r.conversation_id
)
INSERT INTO working.message_occurrence
    (normalized_record_id, match_key, source_version_id, primary_record_id, projection_kind, perspective_person_id,
     cross_device, deriver_version, recorded_at)
SELECT id, working.message_match_key(platform, parties, sender, occurred_at, content_sha256), source_version_id, id,
       'acquired_third_party', perspective, false, 'engine:message_occurrence_backfill@1.0.0', coalesce(derived_at, now())
FROM third_party
ON CONFLICT (normalized_record_id) DO NOTHING;

COMMIT;

-- Report: messages committed more than once from different sources (one row per message, its copies).
SELECT projection_kind,
       count(*) AS messages_with_copies,
       sum(copies - 1) AS extra_rows,
       count(*) FILTER (WHERE perspectives > 1) AS cross_device_messages
FROM (
    SELECT match_key, projection_kind, count(*) AS copies, count(DISTINCT source_version_id) AS sources,
           count(DISTINCT perspective_person_id) AS perspectives
    FROM working.message_occurrence
    WHERE normalized_record_id = primary_record_id
    GROUP BY match_key, projection_kind
    HAVING count(DISTINCT source_version_id) > 1
) duplicates
GROUP BY projection_kind;
