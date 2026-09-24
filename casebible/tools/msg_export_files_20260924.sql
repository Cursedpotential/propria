-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Index of every message-export file in the vault (owner 09:20-09:22: "All those files need to be identified,
-- extracted, deduped, classified ... entities preserved, aliases preserved. Only certain ones are going to end up
-- being canonical, but at least index them all so I know where they're all at").
-- Built from the catalog's vault listing (raw_duck.vault_objects_20260916_r4), so no bucket is re-scanned.
-- One row per file copy. Classified by path and name only; content integrity (ok / scrambled / zero-filled) is a
-- second pass (integrity column, filled later). Duplicate copies share dup_group (same sha1; size+name when no sha1).
-- Canonical choice is the owner's, recorded later in `canonical`.
-- Kinds:
--   sms_backup_xml      SMS Backup & Restore sms-*.xml          calls_backup_xml   calls-*.xml
--   imessage_export     iMessage exports (html/csv/xlsx/pdf/txt/json named imessage*, or under an imessage folder)
--   google_voice        Google Takeout Voice records (Voice/Calls/*.html etc.)
--   fb_messenger_json   Facebook/Messenger message_N.json
--   whatsapp            WhatsApp chat exports
--   number_named        any other file named for a phone number (often "<number> - <Name>.<ext>")
--   messages_folder     files under a "Messages with …" folder not caught above

create table if not exists raw_duck.msg_export_files_20260924 (
  vault_key      text primary key,
  kind           text not null,
  ext            text,
  size_bytes     bigint,
  sha1           text,
  file_name      text,
  numbers        text[],       -- 10-digit numbers found in the path
  name_in_file   text,         -- the "<Name>" part of "<number> - <Name>.<ext>" when present
  year_hint      int,          -- a 20xx year found in the path, when present
  dup_group      text,         -- copies of the same bytes share this
  copies         int,
  integrity      text,         -- filled by the integrity pass: ok | scrambled | zero_filled | unreadable
  canonical      boolean,      -- owner's choice, later
  indexed_at     timestamptz not null default now()
);

begin;
truncate raw_duck.msg_export_files_20260924;
with v as (
  select key, size, sha1, regexp_replace(key, '^.*/', '') as fname, lower(substring(key from '\.([A-Za-z0-9]+)$')) as ext
  from raw_duck.vault_objects_20260916_r4
),
c as (
  select v.*,
    case
      when fname ~* '^sms-.*\.xml$' then 'sms_backup_xml'
      when fname ~* '^calls-.*\.xml$' then 'calls_backup_xml'
      when (fname ~* 'imessage' or key ~* '/imessage[^/]*/') and key !~* '/attachments/'
           and ext in ('html', 'htm', 'csv', 'xlsx', 'pdf', 'txt', 'json') then 'imessage_export'
      when key ~* '/Voice/(Calls|Spam|Voicemail|Texts?)/' then 'google_voice'
      when key ~* '/messages/(inbox|archived_threads|filtered_threads|message_requests|e2ee_cutover)/[^/]+/message_[0-9]+\.json$' then 'fb_messenger_json'
      when fname ~* 'whatsapp' and ext in ('txt', 'zip') then 'whatsapp'
      when fname ~ '^\+?1? ?\(?[0-9]{3}\)?[ .-]?[0-9]{3}[ .-]?[0-9]{4}' and ext in ('pdf', 'txt', 'csv', 'html', 'htm', 'json', 'xlsx', 'docx') then 'number_named'
      when key ~* '/Messages with [^/]+/' and ext in ('pdf', 'txt', 'csv', 'html', 'htm', 'json', 'xlsx', 'docx', 'xml') then 'messages_folder'
    end as kind
  from v
)
insert into raw_duck.msg_export_files_20260924 (vault_key, kind, ext, size_bytes, sha1, file_name, numbers, name_in_file, year_hint, dup_group, copies)
select key, kind, ext, size, sha1, fname,
  (select array_agg(distinct m[1] || m[2] || m[3])
     from regexp_matches(key, '(?<![0-9])\+?1?[ (]*([2-9][0-9]{2})[ ).-]*([0-9]{3})[ .-]*([0-9]{4})(?![0-9])', 'g') as m),
  nullif(trim(substring(fname from '^\+?1? ?\(?[0-9]{3}\)?[ .-]?[0-9]{3}[ .-]?[0-9]{4}\s*-\s*([^.]+)')), ''),
  nullif(substring(key from '(20[12][0-9])'), '')::int,
  coalesce(nullif(sha1, ''), 'size:' || size || ':' || lower(fname)),
  count(*) over (partition by coalesce(nullif(sha1, ''), 'size:' || size || ':' || lower(fname)))
from c where kind is not null;

select kind, count(*) as files, count(distinct dup_group) as distinct_contents, pg_size_pretty(sum(size_bytes)) as size
from raw_duck.msg_export_files_20260924 group by 1 order by 2 desc;
commit;
