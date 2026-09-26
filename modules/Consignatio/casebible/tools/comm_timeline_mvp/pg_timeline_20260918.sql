-- Byline: Claude Code · Opus 5 · 2026-09-18; renamed chat_* -> comm_* with record_kind, Claude Code · Opus 5.5 · 2026-09-24
-- Readable timeline in the catalog PG (casebible.raw_duck), so Metabase (tailnet) can show it tonight.
-- The loaded rows come from timeline_build.duckdb (events_dedup / event_provenance), piped in by load_pg.sh.
-- This is a read-copy for viewing; it is rebuilt by re-running load_pg.sh.
-- Why here as well as Surreal: surreal-intake hung twice under bulk INSERT on 2026-09-18 (see URGENT-TODO).
-- No person names in this file; the tags are computed upstream.

drop table if exists raw_duck.comm_events_20260918 cascade;
create table raw_duck.comm_events_20260918 (
  dedup_key text primary key, n_sources int, sort_ts timestamp, event_ts_utc timestamptz, tz_status text,
  ts_original text, ts_field text, source_format text, event_kind text, conversation_title text,
  conversation_id text, sender text, recipients text, participants text, direction text,
  counterparty_phone text, contact_name text, body text, attachments text,
  katrina_ref_type text, katrina_conf text, catrina_class text, daughter_conf text,
  custody_hit boolean, housing_hit boolean, vault_key text, catalog_rel text,
  -- Owner 2026-09-24: chats are with AI, messages are with people. A new format stays NULL until classified.
  record_kind text generated always as (
    case
      when source_format in ('ai_conversations_json', 'ai_chat_file') then 'ai_chat'
      when source_format in ('fb_messenger_json', 'sms_backup_xml', 'whatsapp_txt', 'google_chat_json') and event_kind = 'message' then 'message'
      when source_format = 'calls_backup_xml' and event_kind = 'call' then 'call'
    end) stored
);
drop table if exists raw_duck.comm_event_provenance_20260918;
create table raw_duck.comm_event_provenance_20260918 (
  dedup_key text, event_uid text, source_format text, extractor text, vault_key text, sha1 text,
  catalog_rel text, member_path text, record_index bigint, ts_original text, ts_field text
);
