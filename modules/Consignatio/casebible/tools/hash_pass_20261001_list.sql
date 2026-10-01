-- Byline: Claude Code · Opus 5.5 · 2026-10-01
-- Key list for hash_pass_20261001.py: every visible vault object whose bytes the catalog cannot yet tie to a source by
-- an independent hash. (1) B2 stores no SHA-1 for it; (2) it is the name+size or same-size target of a source file that
-- has no usable hash (grab_plan_20260930, kind no_hash_file). Read only; prints object_key<TAB>size.
\copy (select 'object_key' as object_key, 'size' as size union all select object_key, size::text from (select distinct v.object_key, v.size from catalog_reconcile.object_versions v where v.visible and v.action = 'upload' and v.object_key like 'consignatio/vault/v1/%' and (coalesce(v.sha1, '') in ('', 'none') or v.object_key in (select from_where from raw_duck.grab_plan_20260930 where kind = 'no_hash_file' and status = 'present_by_name_size') or v.size in (select size from raw_duck.grab_plan_20260930 where kind = 'no_hash_file' and status = 'grab_from_source'))) k) to stdout with (format text)
