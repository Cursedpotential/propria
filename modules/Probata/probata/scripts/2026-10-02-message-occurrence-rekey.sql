-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Re-key every working.message_occurrence row under the device-aware match rule (owner ruling 2026-10-02 15:40:
-- only the same device and the same file type or platform is a duplicate). Requires
-- 2026-10-02-message-occurrence-same-device.sql. Updates match_key only; writes and deletes nothing else.
-- A further occurrence keeps its primary's key (it matched that row on the same device by construction).

\set ON_ERROR_STOP 1
BEGIN;
SET LOCAL lock_timeout = '10s';

CREATE TEMP TABLE rekey ON COMMIT DROP AS
SELECT m.id, working.message_match_key(m.platform,
         array_agg(coalesce(p.entity_id::text, 'norm:' || nullif(registry.norm_identifier(p.participant_raw), ''), 'raw:' || lower(btrim(p.participant_raw)))),
         max(coalesce(p.entity_id::text, 'norm:' || nullif(registry.norm_identifier(p.participant_raw), ''), 'raw:' || lower(btrim(p.participant_raw)))) FILTER (WHERE p.role = 'from'),
         m.ts_utc, m.content_sha256,
         working.message_device_key(src.source_key, m.platform, nullif(split_part(r.conversation_id, '/', 5), '')::uuid)) AS match_key
FROM working.message m
JOIN working.normalized_record r ON r.id = m.derived_from_record_id
JOIN context.source_version sv ON sv.id = r.source_version_id
JOIN context.source src ON src.id = sv.source_id
LEFT JOIN working.message_participant p ON p.message_id = m.id
WHERE r.source = 'proffer'
GROUP BY m.id, m.platform, m.ts_utc, m.content_sha256, src.source_key, r.conversation_id
UNION ALL
SELECT m.id, working.message_match_key(m.platform,
         array_agg(coalesce(p.entity_id::text, 'norm:' || nullif(registry.norm_identifier(p.participant_raw), ''), 'raw:' || lower(btrim(p.participant_raw)))),
         max(coalesce(p.entity_id::text, 'norm:' || nullif(registry.norm_identifier(p.participant_raw), ''), 'raw:' || lower(btrim(p.participant_raw)))) FILTER (WHERE p.role = 'from'),
         m.occurred_at, m.content_sha256,
         working.message_device_key(src.source_key, m.platform, nullif(split_part(r.conversation_id, '/', 5), '')::uuid))
FROM working.third_party_message m
JOIN working.normalized_record r ON r.id = m.normalized_record_id
JOIN context.source_version sv ON sv.id = r.source_version_id
JOIN context.source src ON src.id = sv.source_id
LEFT JOIN working.third_party_message_participant p ON p.message_id = m.id
WHERE r.source = 'proffer'
GROUP BY m.id, m.platform, m.occurred_at, m.content_sha256, src.source_key, r.conversation_id;

UPDATE working.message_occurrence o SET match_key = k.match_key
FROM rekey k WHERE o.primary_record_id = k.id AND o.match_key IS DISTINCT FROM k.match_key;

SELECT count(*) AS occurrences, count(*) FILTER (WHERE normalized_record_id <> primary_record_id) AS further_occurrences
FROM working.message_occurrence;
COMMIT;
