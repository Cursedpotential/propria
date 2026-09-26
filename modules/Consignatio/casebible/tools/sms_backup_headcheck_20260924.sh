#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-24
# Head/tail check of every SMS backup in raw_duck.sms_backup_coverage_20260924, read from its B2 copy (owner 07:30:
# "you could do a head and tail read ... query the first and last dates"; 07:31: everything was moved to B2 and the
# catalog is the must). Read-only on B2: rclone cat --head / --tail only.
# The first bytes of an SMS Backup & Restore file declare its own record count and backup date
# (<smses count="N" backup_date="ms" ...>); the end holds the last record's date. Comparing the declared count with
# the rows the catalog parsed from that file shows whether the catalog read the whole file, which the coverage report
# depends on.
# Runs on ovh-files as root (root's rclone config has the B2 remote); writes the result into the catalog.
#   ssh root@ovh-files 'bash -s' < sms_backup_headcheck_20260924.sh
set -euo pipefail
PG=fgz1n7useplhk0t91uk7k1aw
REMOTE='b2native-full:salem-data'
# stdin comes from PSQL_IN (default /dev/null): with `bash -s` the script itself arrives on stdin, and a bare
# `docker exec -i` would swallow the rest of it.
psql_() { docker exec -i "$PG" psql -U postgres -d casebible -v ON_ERROR_STOP=1 "$@" < "${PSQL_IN:-/dev/null}"; }

psql_ -q -c "create table if not exists raw_duck.sms_backup_headcheck_20260924 (
  file text primary key, vault_key text, declared_count int, backup_date timestamptz, last_record_ts timestamptz,
  catalog_rows int, head_ok boolean, checked_at timestamptz default now());"

list=$(psql_ -At -F $'\t' -c "select c.file, min(p.vault_key), count(*) from raw_duck.sms_backup_coverage_20260924 c
  join raw_duck.comm_event_provenance_20260918 p on coalesce(p.catalog_rel, p.vault_key) = c.file group by c.file")

out=$(mktemp)
while IFS=$'\t' read -r file vk rows <&3; do
  head=$(rclone cat --head 4096 "$REMOTE/$vk" 2>/dev/null </dev/null | tr -d '\0' | head -c 4096 || true)
  count=$(printf '%s' "$head" | grep -o '<smses[^>]*count="[0-9]*"' | grep -o 'count="[0-9]*"' | grep -o '[0-9]*' | head -1 || true)
  bdate=$(printf '%s' "$head" | grep -o 'backup_date="[0-9]*"' | grep -o '[0-9]*' | head -1 || true)
  last=$(rclone cat --tail 262144 "$REMOTE/$vk" 2>/dev/null </dev/null | tr -d '\0' | grep -o ' date="[0-9]\{13\}"' | tail -1 | grep -o '[0-9]*' || true)
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$file" "$vk" "${count:-\\N}" "${bdate:-\\N}" "${last:-\\N}" "$rows" \
    "$([ -n "$head" ] && echo t || echo f)" >> "$out"
done 3<<< "$list"

psql_ -q -c "create table if not exists raw_duck.sms_backup_headcheck_stage_20260924 (file text, vault_key text,
  declared_count text, bdate text, last text, catalog_rows int, head_ok boolean);
  truncate raw_duck.sms_backup_headcheck_stage_20260924;"
PSQL_IN="$out" psql_ -q -c "\copy raw_duck.sms_backup_headcheck_stage_20260924 from stdin"
psql_ -q -c "insert into raw_duck.sms_backup_headcheck_20260924 (file, vault_key, declared_count, backup_date,
    last_record_ts, catalog_rows, head_ok)
  select file, vault_key, nullif(declared_count,'')::int, to_timestamp(nullif(bdate,'')::bigint / 1000.0),
         to_timestamp(nullif(last,'')::bigint / 1000.0), catalog_rows, head_ok
  from raw_duck.sms_backup_headcheck_stage_20260924
  on conflict (file) do update set vault_key = excluded.vault_key, declared_count = excluded.declared_count,
    backup_date = excluded.backup_date, last_record_ts = excluded.last_record_ts, catalog_rows = excluded.catalog_rows,
    head_ok = excluded.head_ok, checked_at = now();"
unlink "$out"   # the script's own temp list

psql_ -c "select c.verdict, h.declared_count, h.catalog_rows, h.declared_count - h.catalog_rows as missing_in_catalog,
  h.backup_date::date as backup_date, h.last_record_ts::date as last_record, c.last_ts::date as catalog_last, c.file
  from raw_duck.sms_backup_coverage_20260924 c left join raw_duck.sms_backup_headcheck_20260924 h using (file)
  order by c.verdict, c.last_ts desc;"
