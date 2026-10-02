-- Byline: Claude Code · Opus 5.5 · 2026-10-01
-- Lists for the three add-only copy jobs the owner approved 2026-10-01 07:01 EDT ("go"), from grab_plan_20260930:
--   restore   1,450 occurrences whose bytes exist only as a noncurrent B2 version -> server-side copy back to its key
--   r2        files whose only good copy is on R2 (corrupt replacements + R2-only files), junk excluded -> copy to B2
--   zero      vault objects that are known all-zero payloads -> server-side copy into _quarantine, then hide the original
-- Each list is printed as a block starting with '#list <name>' (tab CSV with header). Read only.
-- Log: docs/LOG.md, 2026-10-01 entry.

-- One materialized, indexed copy of the 09-20 version listing (the view parses JSON payloads on every scan).
create temp table ov as
  select file_id, object_key, size, sha1, visible, action from catalog_reconcile.object_versions;
create index on ov (file_id);
create index on ov (object_key) where visible;
create index on ov (size, sha1) where visible;
analyze ov;

-- restore: one row per distinct old version; destination = the version's own key, unless a different visible object
-- now holds that key (then a '.restored-<first 12 of file id>' sibling, so nothing is overwritten).
select '#list restore';
\copy (select distinct on (v.file_id) v.file_id, v.object_key as src_key, v.size, v.sha1, case when exists (select 1 from ov w where w.visible and w.object_key = v.object_key) then regexp_replace(v.object_key, '(\.[^./]+)?$', '.restored-' || left(v.file_id, 12) || '\1') else v.object_key end as dest_key from raw_duck.grab_plan_20260930 g join ov v on v.file_id = g.from_where where g.kind = 'restore_b2_version' and g.status = 'restore' order by v.file_id) to stdout with (format csv, delimiter E'\t', header true)

-- r2: corrupt replacements and R2-only files whose good copy is on R2. Junk = software and tool residue that is not
-- case material: interpreter environments, package caches, plugin bundles, compiled code.
select '#list r2';
\copy (select distinct split_part(from_where, ':', 1) as bucket, substr(from_where, length(split_part(from_where, ':', 1)) + 2) as path, size, string_agg(distinct kind, ',') as kinds, 'consignatio/vault/v1/_from-r2-20261001/' || split_part(from_where, ':', 1) || '/' || substr(from_where, length(split_part(from_where, ':', 1)) + 2) as dest_key from raw_duck.grab_plan_20260930 where status = 'grab_from_r2' and from_where !~* '(/|^)(site-packages|node_modules|__pycache__|flet_env|\.venv|venv|\.git|\.obsidian/plugins|_transcript_python|_system_backup)/' and from_where !~* '\.(py|pyc|pyd|pyi|js|mjs|cjs|ts|map|dll|exe|so|dylib|class|jar|whl|lib|a|o|h|c|cpp|pdb)$' group by 1, 2, 3 order by 1, 2) to stdout with (format csv, delimiter E'\t', header true)

-- zero: visible vault objects whose bytes are a known all-zero payload.
select '#list zero';
\copy (select v.file_id, v.object_key as src_key, v.size, v.sha1, 'consignatio/intake/_quarantine/zero-filled-vault-20261001/' || substr(v.object_key, length('consignatio/') + 1) as dest_key from ov v where v.visible and v.action = 'upload' and v.object_key like 'consignatio/vault/v1/%' and exists (select 1 from raw_duck.corrupt_recovery c where c.store = 'b2' and c.size = v.size and c.hash = v.sha1) order by v.object_key) to stdout with (format csv, delimiter E'\t', header true)

-- what the junk filter dropped, for the record
select '#list r2_excluded_summary';
select split_part(from_where, ':', 1), count(*), sum(size) from raw_duck.grab_plan_20260930 where status = 'grab_from_r2' and (from_where ~* '(/|^)(site-packages|node_modules|__pycache__|flet_env|\.venv|venv|\.git|\.obsidian/plugins|_transcript_python|_system_backup)/' or from_where ~* '\.(py|pyc|pyd|pyi|js|mjs|cjs|ts|map|dll|exe|so|dylib|class|jar|whl|lib|a|o|h|c|cpp|pdb)$') group by 1 order by 1;
