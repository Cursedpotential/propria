#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-24
# Move the SMS backups that a newer original fully covers into the B2 quarantine (owner 08:20: "move them into a B2
# quarantine folder ... dry-run first with sizes. do this"). Source of truth: raw_duck.sms_backup_coverage_20260924
# (verdict covered_by_newer; each verified by sms_backup_headcheck_20260924: declared count == catalog rows).
# Destination keeps the full original vault path under the quarantine prefix used for the zero-filled hold:
#   b2:salem-data/consignatio/intake/_quarantine/superseded-sms-backups/v1/<original vault_key>
# Server-side move inside one bucket. B2 keeps the old version at the source as a hidden version (no hard delete).
# Every move is recorded in raw_duck.vault_moves_20260924 (old key, new key, size, sha1, reason, covered_by).
# Usage on ovh-files as root:  MODE=dry-run bash -s < this   |   MODE=execute bash -s < this
set -euo pipefail
MODE="${MODE:-dry-run}"
PG=fgz1n7useplhk0t91uk7k1aw
B2='b2native-full:salem-data'
QPREFIX='consignatio/intake/_quarantine/superseded-sms-backups/v1'
psql_() { docker exec -i "$PG" psql -U postgres -d casebible -v ON_ERROR_STOP=1 "$@" < "${PSQL_IN:-/dev/null}"; }

list=$(psql_ -At -F $'\t' -c "select h.vault_key, c.covered_by, h.declared_count from raw_duck.sms_backup_coverage_20260924 c
  join raw_duck.sms_backup_headcheck_20260924 h using (file)
  where c.verdict = 'covered_by_newer' and h.declared_count = h.catalog_rows order by h.vault_key")
n=$(printf '%s\n' "$list" | grep -c . || true)
echo "mode=$MODE files=$n"
[ "$n" -eq 19 ] || { echo "expected 19 covered files, got $n; stopping"; exit 1; }

total=0; out=$(mktemp)
while IFS=$'\t' read -r vk covered cnt <&3; do
  meta=$(rclone lsjson --hash --no-mimetype "$B2/$vk" </dev/null 2>/dev/null | tr -d '\n')
  size=$(printf '%s' "$meta" | grep -o '"Size":[0-9]*' | grep -o '[0-9]*' | head -1)
  sha1=$(printf '%s' "$meta" | grep -o '"sha1":"[0-9a-f]*"' | cut -d'"' -f4 | head -1)
  [ -n "$size" ] || { echo "MISSING at source: $vk"; exit 1; }
  total=$((total + size))
  printf '%12s  %s\n' "$size" "$vk"
  if [ "$MODE" = execute ]; then
    rclone moveto "$B2/$vk" "$B2/$QPREFIX/$vk" </dev/null
    dmeta=$(rclone lsjson --hash --no-mimetype "$B2/$QPREFIX/$vk" </dev/null 2>/dev/null | tr -d '\n')
    dsize=$(printf '%s' "$dmeta" | grep -o '"Size":[0-9]*' | grep -o '[0-9]*' | head -1)
    dsha1=$(printf '%s' "$dmeta" | grep -o '"sha1":"[0-9a-f]*"' | cut -d'"' -f4 | head -1)
    left=$(rclone lsjson --no-mimetype "$B2/$vk" </dev/null 2>/dev/null | grep -c '"Path"' || true)
    [ "$dsize" = "$size" ] && [ "$dsha1" = "$sha1" ] && [ "$left" = 0 ] || { echo "VERIFY FAILED: $vk"; exit 1; }
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$vk" "$QPREFIX/$vk" "$size" "$sha1" "$covered" "$cnt" >> "$out"
  fi
done 3<<< "$list"
echo "total_bytes=$total ($((total / 1048576)) MiB) -> $B2/$QPREFIX/"

if [ "$MODE" = execute ]; then
  psql_ -q -c "create table if not exists raw_duck.vault_moves_20260924 (
    old_key text primary key, new_key text not null, size_bytes bigint, sha1 text, reason text not null,
    covered_by text, record_count int, moved_at timestamptz not null default now(), script text not null);"
  psql_ -q -c "create table if not exists raw_duck.vault_moves_stage_20260924 (old_key text, new_key text, size_bytes bigint,
    sha1 text, covered_by text, record_count int); truncate raw_duck.vault_moves_stage_20260924;"
  PSQL_IN="$out" psql_ -q -c "\copy raw_duck.vault_moves_stage_20260924 from stdin"
  psql_ -q -c "insert into raw_duck.vault_moves_20260924 (old_key, new_key, size_bytes, sha1, reason, covered_by, record_count, script)
    select old_key, new_key, size_bytes, sha1, 'superseded SMS backup: every message is in a newer original backup',
           covered_by, record_count, 'Consignatio/casebible/tools/sms_backup_quarantine_20260924.sh'
    from raw_duck.vault_moves_stage_20260924 on conflict (old_key) do nothing;"
  psql_ -c "select count(*) as moved, sum(size_bytes) as bytes from raw_duck.vault_moves_20260924 where reason like 'superseded SMS backup%';"
fi
unlink "$out"
