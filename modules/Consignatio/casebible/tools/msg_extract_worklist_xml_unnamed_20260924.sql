-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Worklist of vault XML files that may be message backups under other names (owner 13:48: "I should have several XML
-- files from her phone"; only 2 were found among files named sms-*.xml). msg_export_files_20260924 classifies by name,
-- so recovered-disk names (f96544768_sms_export.xml, xml/f146876416.xml, files cut at exactly 256 MB), recycle-bin names
-- ($RLJ1ROT.xml) and Takeout copies (sms_20250218024754.xml) were never looked at. The runner identifies each file by its
-- content (sniff), skips what is not a message backup (recorded as no_reader), and salvages a file cut off mid-record
-- (elt_xml_sanitize_v2). custodian is left empty: the file's own MMS addressing says whose phone it is.
-- Scope: every vault XML over 300 KB that is not already in msg_export_files_20260924, minus obvious non-message XML
-- (Office parts, IDE/project files, app manifests, module docs).
\pset footer off
copy (
  select 'sms_backup_xml' as format_guess, v.key as vault_key, coalesce(nullif(v.sha1, ''), 'size:' || v.size || ':' || lower(regexp_replace(v.key, '^.*/', ''))) as sha1,
         v.size, '' as catalog_rel_example, '' as custodian, '' as source_device, '' as also_at
  from raw_duck.vault_objects_20260916_r4 v
  where v.key ~* '\.xml$' and v.size > 300000
    and not exists (select 1 from raw_duck.msg_export_files_20260924 f where f.vault_key = v.key)
    and v.key !~* '(\.obsidian|/word/|/xl/|docProps|\[Content_Types\]|/ppt/|workspace|\.idea|manifest|pom\.xml|/res/|PowerShell/Modules|/lib/)'
  order by v.size
) to stdout with (format csv, delimiter E'\t', header true);
