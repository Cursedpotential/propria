#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-24
# Load one ELT proposal bundle (devbox) into the catalog staging tables (msg_extract_rows_20260924.sql, run first).
# Runs on ovh-files. Usage from the desktop:
#   ssh root@100.91.190.107 'bash -s' -- <attempt_id> <bundle_dir> < msg_extract_load_20260924.sh
# Re-loading an attempt replaces that attempt's staging rows (they are a copy of the immutable bundle).
# Gate: after loading, row counts per relation must equal manifest.json's, or the script fails.
set -euo pipefail
A="$1"
B="$2"
D=devbox-pd3xc78ahqkfswq12bpfqgy1-150427235321
PG=fgz1n7useplhk0t91uk7k1aw
# docker exec -i would swallow the rest of this script (it arrives on stdin), so every psql call that is not fed by a
# pipe reads /dev/null.
psql_q() { docker exec -i "$PG" psql -U postgres -d casebible -v ON_ERROR_STOP=1 -qAt "$@" < /dev/null; }
psql_in() { docker exec -i "$PG" psql -U postgres -d casebible -v ON_ERROR_STOP=1 -q "$@"; }
duck() { docker exec "$D" duckdb -readonly "$B/proposal.duckdb" -c "$1" < /dev/null; }

manifest=$(docker exec "$D" cat "$B/manifest.json" < /dev/null)
[ "$(echo "$manifest" | python3 -c 'import json,sys; print(json.load(sys.stdin)["attempt_id"])')" = "$A" ] \
  || { echo "manifest attempt_id is not $A"; exit 1; }

psql_q -c "delete from raw_duck.msg_extract_warnings_20260924 where attempt_id = '$A';
           delete from raw_duck.msg_extract_lineage_20260924 where attempt_id = '$A';
           delete from raw_duck.msg_extract_rows_20260924 where attempt_id = '$A';
           delete from raw_duck.msg_extract_attempts_20260924 where attempt_id = '$A';"
echo "insert into raw_duck.msg_extract_attempts_20260924 (attempt_id, bundle_dir, manifest) values (:'a', :'b', :'m'::jsonb);" \
  | psql_in -v a="$A" -v b="$B" -v m="$manifest"

duck "copy (select '$A' as attempt_id, vault_key, sha1, source_format, extractor, coalesce(block, '*') as block, record_index,
             custodian, source_device, platform, event_ts_utc, sort_ts, sort_ts_final, ts_original, ts_field, tz_status,
             event_kind, conversation_id, conversation_title, to_json(participants)::varchar as participants, sender,
             to_json(recipients)::varchar as recipients, direction, counterparty_phone, contact_name, owner_line, body,
             attachments, member_path, from_owner, katrina_thread, katrina_ref_type, katrina_conf, catrina_class,
             daughter_conf, custody_hit, housing_hit, content_key, dedup_key, dup_occurrence
           from proposed_records) to '/dev/stdout' (format csv, header true)" \
  | psql_in -c "\copy raw_duck.msg_extract_rows_20260924 (attempt_id, vault_key, sha1, source_format, extractor, block, record_index,
             custodian, source_device, platform, event_ts_utc, sort_ts, sort_ts_final, ts_original, ts_field, tz_status,
             event_kind, conversation_id, conversation_title, participants, sender, recipients, direction,
             counterparty_phone, contact_name, owner_line, body, attachments, member_path, from_owner, katrina_thread,
             katrina_ref_type, katrina_conf, catrina_class, daughter_conf, custody_hit, housing_hit, content_key,
             dedup_key, dup_occurrence) from stdin with (format csv, header true)"

duck "copy (select '$A', vault_key, sha1, source_format, extractor, custodian, source_device, also_at, source_markers,
             rows_out, row_digest, bundle_file from proposed_lineage) to '/dev/stdout' (format csv, header true)" \
  | psql_in -c "\copy raw_duck.msg_extract_lineage_20260924 from stdin with (format csv, header true)"

duck "copy (select '$A', vault_key, sha1, kind, detail from proposed_warnings) to '/dev/stdout' (format csv, header true)" \
  | psql_in -c "\copy raw_duck.msg_extract_warnings_20260924 from stdin with (format csv, header true)"

# Gate: catalog counts equal the manifest's.
for rel in rows:proposed_records lineage:proposed_lineage warnings:proposed_warnings; do
  t=${rel%%:*}; r=${rel##*:}
  want=$(echo "$manifest" | python3 -c "import json,sys; print(json.load(sys.stdin)['relations']['$r']['rows'])")
  got=$(psql_q -c "select count(*) from raw_duck.msg_extract_${t}_20260924 where attempt_id = '$A'")
  echo "$t: manifest $want, catalog $got"
  [ "$want" = "$got" ] || { echo "COUNT MISMATCH for $t"; exit 1; }
done
echo "LOADED $A"
