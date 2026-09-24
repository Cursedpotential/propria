-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Catalog staging for DuckDB ELT extraction attempts (comm_timeline_mvp/elt_run.py). The attempt's proposal bundle
-- (proposal.duckdb + manifest.json in devbox) is the review artifact; these tables are its catalog copy, loaded by
-- msg_extract_load_20260924.sh, so every later step queries the catalog (owner 2026-09-16 "the catalog is the source of
-- truth"; 2026-09-24 07:18-07:20 "build it into the catalog and then we can extract it from the catalog into the tables").
--
-- ONE ROW PER MESSAGE PER FILE. Nothing is collapsed here. Owner 2026-09-24 09:28-09:36: a copy from a different
-- format or a different person's device is corroborating evidence, never a duplicate, and must be easy to query; a
-- true duplicate is the same device + same format + same user + same platform. So:
--   dedup_key    true-duplicate key; rows sharing it are the same message from the same device/format/user/platform
--                (e.g. incremental backups of one phone). Group by it to count copies; never delete by it.
--   content_key  the message text/thread/instant key (v1 formula, numbers normalized): a join aid.
--   corroboration across devices/formats is a separate link table (msg_corroboration_20260924), never a merge.
-- Every row cites its file: vault_key (current B2 key under salem-data/) + sha1; other paths holding the same bytes
-- are in msg_extract_lineage_20260924.also_at.
-- custodian: whose device/account the file came from (Matt | Katrina | NULL = not yet resolved).
-- owner_line: the device holder's own number on that message, where the source states it.

create table if not exists raw_duck.msg_extract_attempts_20260924 (
  attempt_id     text primary key,
  bundle_dir     text not null,           -- devbox path of the proposal bundle
  manifest       jsonb not null,          -- manifest.json as written by the runner (digests, artifact SHA-256s)
  loaded_at      timestamptz not null default now()
);

create table if not exists raw_duck.msg_extract_rows_20260924 (
  attempt_id         text not null references raw_duck.msg_extract_attempts_20260924 (attempt_id),
  vault_key          text not null,
  sha1               text not null,
  source_format      text not null,
  extractor          text not null,
  block              text not null,
  record_index       bigint not null,
  custodian          text,
  source_device      text,
  platform           text,
  event_ts_utc       timestamptz,
  sort_ts            timestamp,
  sort_ts_final      timestamp,
  ts_original        text,
  ts_field           text,
  tz_status          text,
  event_kind         text,
  conversation_id    text,
  conversation_title text,
  participants       jsonb,
  sender             text,
  recipients         jsonb,
  direction          text,
  counterparty_phone text,
  contact_name       text,
  owner_line         text,
  body               text,
  attachments        text,
  member_path        text,
  from_owner         boolean,
  katrina_thread     boolean,
  katrina_ref_type   text,
  katrina_conf       text,
  catrina_class      text,
  daughter_conf      text,
  custody_hit        boolean,
  housing_hit        boolean,
  content_key        text not null,
  dedup_key          text not null,
  dup_occurrence     int,
  primary key (attempt_id, sha1, extractor, block, record_index)
);
create index if not exists msg_extract_rows_20260924_dedup on raw_duck.msg_extract_rows_20260924 (dedup_key);
create index if not exists msg_extract_rows_20260924_cp on raw_duck.msg_extract_rows_20260924 (counterparty_phone, sort_ts_final);
create index if not exists msg_extract_rows_20260924_vk on raw_duck.msg_extract_rows_20260924 (vault_key);

create table if not exists raw_duck.msg_extract_lineage_20260924 (
  attempt_id     text not null references raw_duck.msg_extract_attempts_20260924 (attempt_id),
  vault_key      text not null,
  sha1           text not null,
  source_format  text,
  extractor      text,
  custodian      text,
  source_device  text,
  also_at        text,         -- other vault keys holding the same bytes, ' | ' separated
  source_markers bigint,       -- records counted straight from the file (NULL where the format has no marker count)
  rows_out       bigint,
  row_digest     text,
  bundle_file    text,
  primary key (attempt_id, sha1)
);

create table if not exists raw_duck.msg_extract_warnings_20260924 (
  attempt_id  text not null references raw_duck.msg_extract_attempts_20260924 (attempt_id),
  vault_key   text not null,
  sha1        text,
  kind        text not null,   -- no_reader | no_rows | count_differs | error
  detail      text
);
