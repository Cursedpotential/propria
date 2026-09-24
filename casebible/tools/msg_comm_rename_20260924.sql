-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Owner 08:59-09:00: "Chats are with AI. Messages are with people." / "Then fix it upstream." / "Fix all of it, figure out
-- what breaks, then fix that too."
--
-- 1. The 2026-09-24 tables hold only messages with people (Messenger + three SMS threads): chat_* -> msg_*.
-- 2. The 2026-09-18 tables mix messages, calls and AI chats: chat_* -> comm_* (communications), and comm_events gets a
--    stored record_kind ('message' | 'call' | 'ai_chat'; NULL = a format nobody has classified yet) with two views,
--    msg_events_20260918 (messages with people) and ai_chat_events_20260918 (AI chats).
-- 3. Every rename is recorded in raw_duck.catalog_renames_20260924 (old name -> new name), because receipts and the
--    build_script column still carry the old names.
-- Indexes and constraints follow their table's new name. Views that read these tables (timeline_*_20260918) follow by OID.
-- Run on ovh-files: docker exec -i <pg container> psql -U postgres -d casebible -v ON_ERROR_STOP=1 < msg_comm_rename_20260924.sql

BEGIN;

CREATE TABLE raw_duck.catalog_renames_20260924 (
  old_name   text PRIMARY KEY,
  new_name   text NOT NULL UNIQUE,
  kind       text NOT NULL CHECK (kind IN ('table', 'view', 'index', 'constraint', 'script')),
  reason     text NOT NULL,
  renamed_at timestamptz NOT NULL DEFAULT now(),
  script     text NOT NULL DEFAULT 'casebible/tools/msg_comm_rename_20260924.sql'
);

CREATE TEMP TABLE rename_map (old_name text, new_name text, kind text, reason text) ON COMMIT DROP;
INSERT INTO rename_map VALUES
  ('chat_conversation_registry_20260924', 'msg_conversation_registry_20260924', 'table', 'messages with people'),
  ('chat_message_norm_20260924',          'msg_norm_20260924',                  'table', 'messages with people'),
  ('chat_message_files_20260924',         'msg_files_20260924',                 'table', 'messages with people'),
  ('chat_bouts_20260924',                 'msg_bouts_20260924',                 'table', 'messages with people'),
  ('chat_bout_messages_20260924',         'msg_bout_messages_20260924',         'table', 'messages with people'),
  ('chat_bout_labels_20260924',           'msg_bout_labels_20260924',           'table', 'messages with people'),
  ('chat_bout_labels_stage_20260924',     'msg_bout_labels_stage_20260924',     'table', 'messages with people'),
  ('chat_bout_observations_cited_20260924', 'msg_bout_observations_cited_20260924', 'view', 'messages with people'),
  ('chat_events_20260918',                'comm_events_20260918',               'table', 'mixes messages, calls and AI chats'),
  ('chat_event_provenance_20260918',      'comm_event_provenance_20260918',     'table', 'mixes messages, calls and AI chats'),
  ('chat_candidates_20260918',            'comm_candidates_20260918',           'table', 'mixes messages, calls and AI chats'),
  ('chat_dir_files_20260918',             'comm_dir_files_20260918',            'table', 'mixes messages, calls and AI chats'),
  ('chat_directories_20260918',           'comm_directories_20260918',          'table', 'mixes messages, calls and AI chats');

DO $$
DECLARE m record; r record; newname text;
BEGIN
  FOR m IN SELECT * FROM rename_map WHERE kind IN ('table', 'view') LOOP
    -- Constraints first (renaming a primary key or unique constraint also renames its index).
    FOR r IN SELECT c.conname FROM pg_constraint c
             WHERE c.conrelid = format('raw_duck.%I', m.old_name)::regclass AND left(c.conname, length(m.old_name) + 1) = m.old_name || '_' LOOP
      newname := m.new_name || substr(r.conname, length(m.old_name) + 1);
      EXECUTE format('ALTER TABLE raw_duck.%I RENAME CONSTRAINT %I TO %I', m.old_name, r.conname, newname);
      INSERT INTO rename_map VALUES (r.conname, newname, 'constraint', m.reason);
    END LOOP;
    FOR r IN SELECT i.relname FROM pg_index x JOIN pg_class i ON i.oid = x.indexrelid
             WHERE x.indrelid = format('raw_duck.%I', m.old_name)::regclass AND left(i.relname, length(m.old_name) + 1) = m.old_name || '_' LOOP
      newname := m.new_name || substr(r.relname, length(m.old_name) + 1);
      EXECUTE format('ALTER INDEX raw_duck.%I RENAME TO %I', r.relname, newname);
      INSERT INTO rename_map VALUES (r.relname, newname, 'index', m.reason);
    END LOOP;
    IF m.kind = 'view' THEN
      EXECUTE format('ALTER VIEW raw_duck.%I RENAME TO %I', m.old_name, m.new_name);
    ELSIF m.kind = 'table' THEN
      EXECUTE format('ALTER TABLE raw_duck.%I RENAME TO %I', m.old_name, m.new_name);
    END IF;
  END LOOP;
END $$;

INSERT INTO rename_map VALUES
  ('casebible/tools/chat_extract_registry_20260924.sql',   'casebible/tools/msg_extract_registry_20260924.sql',   'script', 'messages with people'),
  ('casebible/tools/chat_message_norm_20260924.sql',       'casebible/tools/msg_norm_20260924.sql',               'script', 'messages with people'),
  ('casebible/tools/chat_message_files_20260924.sql',      'casebible/tools/msg_files_20260924.sql',              'script', 'messages with people'),
  ('casebible/tools/chat_bouts_20260924.sql',              'casebible/tools/msg_bouts_20260924.sql',              'script', 'messages with people'),
  ('casebible/tools/chat_bouts_custody_party_20260924.sql', 'casebible/tools/msg_bouts_custody_party_20260924.sql', 'script', 'messages with people'),
  ('casebible/tools/chat_bout_labels_upsert_20260924.sql', 'casebible/tools/msg_bout_labels_upsert_20260924.sql', 'script', 'messages with people'),
  ('casebible/tools/chat_candidates_20260918.sql',         'casebible/tools/comm_candidates_20260918.sql',        'script', 'mixes messages, calls and AI chats'),
  ('casebible/tools/chat_directories_20260918.sql',        'casebible/tools/comm_directories_20260918.sql',       'script', 'mixes messages, calls and AI chats'),
  ('casebible/tools/chat_elt_worklist_20260918.sql',       'casebible/tools/comm_elt_worklist_20260918.sql',      'script', 'mixes messages, calls and AI chats'),
  ('casebible/tools/chat_timeline_mvp/',                   'casebible/tools/comm_timeline_mvp/',                  'script', 'mixes messages, calls and AI chats');

INSERT INTO raw_duck.catalog_renames_20260924 (old_name, new_name, kind, reason)
SELECT old_name, new_name, kind, reason FROM rename_map;

-- The mixed events table says what each row is. Formats are listed explicitly; a new format stays NULL until classified.
ALTER TABLE raw_duck.comm_events_20260918 ADD COLUMN record_kind text GENERATED ALWAYS AS (
  CASE
    WHEN source_format IN ('ai_conversations_json', 'ai_chat_file') THEN 'ai_chat'
    WHEN source_format IN ('fb_messenger_json', 'sms_backup_xml', 'whatsapp_txt', 'google_chat_json') AND event_kind = 'message' THEN 'message'
    WHEN source_format = 'calls_backup_xml' AND event_kind = 'call' THEN 'call'
  END) STORED;

CREATE VIEW raw_duck.msg_events_20260918 AS
  SELECT * FROM raw_duck.comm_events_20260918 WHERE record_kind = 'message';
COMMENT ON VIEW raw_duck.msg_events_20260918 IS 'Messages with people (SMS, Messenger, WhatsApp, Google Chat). Owner 2026-09-24: messages are with people.';

CREATE VIEW raw_duck.ai_chat_events_20260918 AS
  SELECT * FROM raw_duck.comm_events_20260918 WHERE record_kind = 'ai_chat';
COMMENT ON VIEW raw_duck.ai_chat_events_20260918 IS 'Conversations with AI. Owner 2026-09-24: chats are with AI.';

COMMENT ON TABLE raw_duck.comm_events_20260918 IS 'Communication events: messages with people, calls and AI chats; record_kind says which. Renamed from chat_events_20260918 on 2026-09-24.';

COMMIT;
