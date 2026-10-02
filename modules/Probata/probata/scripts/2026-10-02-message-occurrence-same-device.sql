-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- Owner ruling 2026-10-02 15:40 EDT: "we don't want any duplicates unless it's a completely separate medium or
-- person or backup device. If it's a real duplicate from the exact same type of file from the exact same device,
-- then we don't need it." The match-up key therefore also names the device the source came from:
--   * an SMS Backup & Restore source: its casevault device folder (the phone's own number);
--   * any other source (a Facebook export): the platform and the perspective person (the account's owner).
-- The same message from a different phone, person or medium gets a different key and stays its own row.
-- Applied to the snapshot and live; then every existing occurrence is re-keyed under the new rule.

CREATE FUNCTION working.message_device_key(p_source_key text, p_platform text, p_perspective uuid)
RETURNS text
LANGUAGE sql IMMUTABLE
SET search_path TO 'pg_catalog'
AS $$
    SELECT CASE
        WHEN substring(p_source_key FROM '/sms-backup-restore/([0-9]+)/') IS NOT NULL
            THEN 'sms-backup-restore:' || substring(p_source_key FROM '/sms-backup-restore/([0-9]+)/')
        ELSE coalesce(p_platform, '') || ':' || coalesce(p_perspective::text, '')
    END
$$;

CREATE FUNCTION working.message_match_key(p_platform text, p_parties text[], p_sender text, p_occurred timestamptz, p_body_sha256 bytea, p_device text)
RETURNS text
LANGUAGE sql STABLE
SET search_path TO 'pg_catalog'
AS $$
    SELECT encode(sha256(convert_to(
        coalesce(p_device, '') || '|' ||
        coalesce(p_platform, '') || '|' ||
        coalesce((SELECT string_agg(DISTINCT party, ',' ORDER BY party) FROM unnest(p_parties) AS party WHERE party <> ''), '') || '|' ||
        coalesce(p_sender, '') || '|' ||
        coalesce(to_char(date_trunc('second', p_occurred AT TIME ZONE 'UTC'), 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '') || '|' ||
        coalesce(encode(p_body_sha256, 'hex'), ''),
    'UTF8')), 'hex')
$$;

GRANT EXECUTE ON FUNCTION working.message_device_key(text, text, uuid) TO platform_runtime;
GRANT EXECUTE ON FUNCTION working.message_match_key(text, text[], text, timestamptz, bytea, text) TO platform_runtime;
