-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Worklist for the DuckDB ELT runner (comm_timeline_mvp/elt_run.py), built from the catalog, never from a bucket scan.
-- Scope (owner 09:17-09:27): Matt's side of his conversations with Katrina, priority 2023-2024 — every indexed
-- message-export file under "Messages with Katrina" (owner 09:40: all Matt and Katrina, from his devices) or whose
-- name carries one of her numbers, EXCEPT the SMS/call backups already extracted into the 09-18 discovery index
-- (none of those cover 2023-24). File types with no reader are listed too (format_guess 'no_reader:<ext>'), so the
-- run reports them as gaps instead of dropping them silently.
-- One row per distinct content (dup_group): identical bytes are the same file and are extracted once; every other
-- path holding those bytes is listed in also_at, so each extracted record cites its file and the file's copies.
-- custodian: 'Matt' only where the owner has said so (the Messages with Katrina folder). Elsewhere NULL: the
-- catalog resolves it after extraction from owner_line (the "Me" number) against raw_duck.msg_identity_20260924,
-- because a Takeout can come from any account.
-- source_device: NULL here (no device confirmed), so the runner uses 'unconfirmed:<sha1>' and merges nothing.
-- Output: TSV with a header, the runner's worklist format.
\pset footer off
copy (
  with her as (select array_agg(distinct identifier) as nums from raw_duck.msg_identity_20260924
               where person = 'Katrina' and kind = 'phone'),
  f as (
    select m.* from raw_duck.msg_export_files_20260924 m, her
    where (m.vault_key ~* '/Messages with Katrina/' or m.numbers && her.nums)
      and m.kind not in ('sms_backup_xml', 'calls_backup_xml')
  ),
  r as (
    select f.*, row_number() over (partition by dup_group
                                   order by (vault_key ~* '/Messages with Katrina/') desc, vault_key) as rn,
           bool_or(vault_key ~* '/Messages with Katrina/') over (partition by dup_group) as in_mwk
    from f
  ),
  w as (
    select
      case when p.kind = 'google_voice' and p.ext = 'html' then 'google_voice_html'
           when p.kind = 'imessage_export' and p.ext in ('html', 'htm') then 'imessage_html'
           when p.kind = 'number_named' and p.ext in ('html', 'htm') then 'google_voice_html'
           when p.kind in ('number_named', 'messages_folder', 'imessage_export') and p.ext = 'txt' then 'imessage_txt'
           when p.kind = 'messages_folder' and p.ext = 'xml' then 'sms_backup_xml'
           else 'no_reader:' || coalesce(p.ext, 'none') end as format_guess,
      p.vault_key, coalesce(nullif(p.sha1, ''), p.dup_group) as sha1, p.size_bytes as size, '' as catalog_rel_example,
      case when p.in_mwk then 'Matt' else '' end as custodian,
      '' as source_device,
      (select string_agg(o.vault_key, ' | ' order by o.vault_key) from r o
        where o.dup_group = p.dup_group and o.rn > 1) as also_at
    from r p where p.rn = 1
  )
  select * from w order by (format_guess like 'no_reader%'), size, vault_key
) to stdout with (format csv, delimiter E'\t', header true);
