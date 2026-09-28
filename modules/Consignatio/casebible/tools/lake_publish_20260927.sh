#!/usr/bin/env bash
# lake_publish_20260927.sh - publish the Case Bible catalog (PG casebible, schema raw_duck, ovh-files)
# to B2 as Parquet under salem-data/consignatio/_system/lake/2026-09-27/, so the index lives with the data.
#
# Byline: Claude Code · Opus 5.5 · 2026-09-27 (agent for the Fable 5.1 supervising session)
# Owner 2026-09-27 00:09 EDT: "B2 is the canonical home, and that's where the index is supposed to be.
#   That's what's supposed to be cataloged. That's what's supposed to be the lakehouse."  00:14: "Finish creating the lakehouse."
# Log entry: modules/Consignatio/docs/URGENT-TODO.md (2026-09-27). Receipt: docs/receipts/lake-publish-20260927/.
#
# Runs ON ovh-files as root, one phase at a time, detached from any agent shell:
#   nohup setsid bash lake_publish_20260927.sh <phase> >> logs/<phase>.out 2>&1 < /dev/null &
# Phases (markers in $WORK/markers/<phase>.{started,done,failed}):
#   export    dry run, no B2 writes. Per table in the table list (decisions publish|ask_publish|ask_exclude):
#             PG count -> COPY via pg_duckdb (1 thread, zstd) into the PG volume -> read_parquet count ->
#             PG count again -> DESCRIBE -> size + sha256 -> move to $OUT/<table>.parquet. pg_duckdb cannot read
#             NUMERIC without precision, so it reads those columns as DOUBLE; a column whose values are all integral
#             and below 2^53 (exact in a double) is then written as bigint, any other such column stays double.
#   schema    $OUT/schema.json for the publish set: information_schema columns, keys, casts, Parquet types, views.
#   upload    rclone copy --immutable of the publish set + corrupt_missing.csv + schema.json, then rclone check
#             (SHA-1, then size-only). Add-only: --immutable refuses to change any object that already exists.
#   readback  download every uploaded object into the PG volume, sha256 each, count Parquet rows with pg_duckdb,
#             compare with the export; the downloaded copies are then moved to $WORK/readback-2026-09-27/.
#   finalize  manifest.csv + LATEST uploaded (immutable) and checked; raw_duck.lake_publish_20260927 loaded
#             from the same rows via lake_publish_20260927.sql.
#   s3probe   GET every published object over B2's S3 API (the path DuckDB/Evidence.dev readers use) and compare
#             sha256 with the catalog; the S3 key id and application key arrive as two lines on stdin.
#   status    print markers and progress (read-only).
# Table list: lake_publish_20260927.tables.txt (table_name, decision publish|exclude, status, reason).
# Only 'publish' rows are uploaded. Nothing on B2 is moved, renamed or deleted by any phase.

set -uo pipefail

DATE=2026-09-27
TAG=lake-publish-20260927
PGC=${PGC:-fgz1n7useplhk0t91uk7k1aw}                       # casebible-pg18 (Coolify data-pg-files)
WORK=${WORK:-/data/consignatio/$TAG}
OUT=$WORK/$DATE                                           # exactly the B2 layout of the dated folder
VOL=${VOL:-/var/lib/docker/volumes/postgres-data-$PGC/_data}  # host view of the container's /var/lib/postgresql
CVOL=/var/lib/postgresql                                  # container view (the only path PG can write/read)
REMOTE=${REMOTE:-b2native-full:salem-data/consignatio/_system/lake}
B2PREFIX=consignatio/_system/lake                          # object keys inside bucket salem-data
TABLES=${TABLES:-$WORK/lake_publish_20260927.tables.txt}   # tab-separated; .txt because *.tsv is git-ignored in this module
SQLFILE=${SQLFILE:-$WORK/lake_publish_20260927.sql}
MARK=$WORK/markers
LOGD=$WORK/logs
RES=$WORK/export_results.tsv
DUCK_SET="SET duckdb.threads = 1; SET duckdb.max_workers_per_postgres_scan = 0; SET duckdb.threads_for_postgres_scan = 1; SET duckdb.convert_unsupported_numeric_to_double = true;"

mkdir -p "$OUT" "$MARK" "$LOGD" "$WORK/describe"
ts() { date -u +%FT%TZ; }
say() { echo "$(ts) $*"; }
pg() { docker exec -i "$PGC" psql -U postgres -d casebible -X -A -t -q -v ON_ERROR_STOP=1 -F '|'; }

# table names from the list whose decision matches the regex; names are validated (they are interpolated into SQL)
list_tables() {
  awk -F'\t' -v re="$1" 'NR>1 && $2 ~ re {print $1}' "$TABLES" | while read -r t; do
    if [[ "$t" =~ ^[a-z0-9_]+$ ]]; then echo "$t"; else echo "bad table name: $t" >&2; fi
  done
}
status_of() { awk -F'\t' -v t="$1" 'NR>1 && $1==t {print $3}' "$TABLES"; }

phase_start() { rm_marker "$1"; : > "$MARK/$1.started"; say "phase $1 started"; }
rm_marker() { for s in done failed; do [ -e "$MARK/$1.$s" ] && mv "$MARK/$1.$s" "$MARK/$1.$s.prev-$(date -u +%Y%m%dT%H%M%SZ)"; done; true; }
phase_end() { if [ "$2" -eq 0 ]; then : > "$MARK/$1.done"; say "phase $1 DONE"; else echo "$2" > "$MARK/$1.failed"; say "phase $1 FAILED ($2 problems)"; fi; }

# ---------------------------------------------------------------- export (dry run: no B2 writes)
build_select() {  # sets SEL, CASTS and FROMX for table $1
  local t=$1 c isnum q integral kw=0
  local parts=() casts=()
  while IFS='|' read -r c isnum; do
    [ -z "$c" ] && continue
    q="\"${c//\"/\"\"}\""
    if [ "$isnum" = "1" ]; then
      integral=$(pg <<SQL
SELECT (coalesce(bool_and($q = trunc($q)), true) AND coalesce(max(abs($q)) < 9007199254740992, true))::int FROM raw_duck."$t";
SQL
)
      if [ "$integral" = "1" ]; then parts+=("t.$q::bigint AS $q"); casts+=("$c:numeric->bigint")
      else parts+=("t.$q::double precision AS $q"); casts+=("$c:numeric->double"); fi
    else
      parts+=("t.$q")
    fi
    grep -qxF "${c,,}" "$WORK/duckdb_keywords.txt" 2>/dev/null && kw=1
  done < <(pg <<SQL
SELECT column_name || '|' || (data_type = 'numeric' AND numeric_precision IS NULL)::int
FROM information_schema.columns WHERE table_schema = 'raw_duck' AND table_name = '$t' ORDER BY ordinal_position;
SQL
)
  # pg_duckdb re-emits column names unquoted and drops the t. qualifier when the query has one table, so a column
  # named like a DuckDB keyword (only intake_fs_ops_20260917.at) fails to parse; a one-row cross join keeps t.<name>.
  FROMX=""; [ "$kw" = 1 ] && FROMX=" CROSS JOIN (SELECT 1) AS one_row"
  local IFS=,
  SEL="${parts[*]}"
  CASTS="${casts[*]:-}"
}

export_one() {
  local t=$1 f="$CVOL/${TAG}__${t}.parquet" hf="$VOL/${TAG}__${t}.parquet" t0 t1 out before parq after sz sha ok
  if [ -e "$OUT/$t.parquet" ] && awk -F'\t' -v t="$t" '$1==t && $12=="1"{f=1} END{exit !f}' "$RES" 2>/dev/null; then
    say "skip $t (already exported and verified)"; return 0; fi
  t0=$(date +%s)
  build_select "$t"
  if [ -z "$SEL" ]; then say "FAIL $t: no columns (table missing?)"; return 1; fi
  out=$(pg 2>&1 <<SQL
$DUCK_SET
SELECT 'pg_before|' || count(*) FROM raw_duck."$t";
COPY (SELECT $SEL FROM raw_duck."$t" t$FROMX) TO '$f' WITH (FORMAT parquet, COMPRESSION zstd);
SELECT 'parquet|' || count(*) FROM read_parquet('$f');
SELECT 'pg_after|' || count(*) FROM raw_duck."$t";
SQL
)
  before=$(sed -n 's/^pg_before|//p' <<<"$out"); parq=$(sed -n 's/^parquet|//p' <<<"$out"); after=$(sed -n 's/^pg_after|//p' <<<"$out")
  if [ ! -s "$hf" ] || [ -z "$before" ] || [ -z "$parq" ]; then say "FAIL $t: $(tr '\n' ' ' <<<"$out" | cut -c1-400)"; return 1; fi
  pg > "$WORK/describe/$t.tsv" 2>>"$LOGD/describe.err" <<SQL
$DUCK_SET
SELECT * FROM duckdb.query(\$q\$ SELECT column_name, column_type FROM (DESCRIBE SELECT * FROM read_parquet('$f')) \$q\$);
SQL
  sz=$(stat -c %s "$hf"); sha=$(sha256sum "$hf" | cut -d' ' -f1)
  mv -n "$hf" "$OUT/$t.parquet" || { say "FAIL $t: move"; return 1; }
  t1=$(date +%s)
  ok=0; [ "$before" = "$parq" ] && [ "$parq" = "$after" ] && ok=1
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$t" "$(status_of "$t")" "$before" "$parq" "$after" "$sz" "$sha" "${CASTS:--}" \
    "$(date -u -d @"$t0" +%FT%TZ)" "$(date -u -d @"$t1" +%FT%TZ)" "$((t1 - t0))" "$ok" >> "$RES"
  say "$( [ $ok = 1 ] && echo ok || echo MISMATCH ) $t rows pg=$before parquet=$parq pg_after=$after bytes=$sz ${CASTS:+casts=$CASTS} $((t1 - t0))s"
  [ "$ok" = 1 ]
}

do_export() {
  phase_start export
  [ -s "$RES" ] || printf 'table_name\tstatus\tpg_rows_before\tparquet_rows\tpg_rows_after\tparquet_bytes\tsha256\tcasts\tstarted_at\tfinished_at\tseconds\tok\n' > "$RES"
  local fails=0 t
  pg > "$WORK/duckdb_keywords.txt" <<SQL
SELECT * FROM duckdb.query(\$q\$ SELECT lower(keyword_name) FROM duckdb_keywords() WHERE keyword_category <> 'unreserved' \$q\$);
SQL
  # smallest first, so problems surface early
  local names; names=$(list_tables '^(publish|ask_publish|ask_exclude)$' | paste -sd, - | sed "s/,/','/g")
  while read -r t; do
    [ -z "$t" ] && continue
    export_one "$t" || fails=$((fails + 1))
  done < <(pg <<SQL
SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'raw_duck' AND c.relkind = 'r' AND c.relname IN ('$names') ORDER BY pg_total_relation_size(c.oid), c.relname;
SQL
)
  # every listed table must exist
  for t in $(list_tables '^(publish|ask_publish|ask_exclude)$'); do
    awk -F'\t' -v t="$t" '$1==t{f=1} END{exit !f}' "$RES" || { say "FAIL $t: not exported (missing in PG?)"; fails=$((fails + 1)); }
  done
  phase_end export "$fails"
}

# ---------------------------------------------------------------- schema.json
do_schema() {
  phase_start schema
  local names; names=$(list_tables '^publish$' | paste -sd, - | sed "s/,/','/g")
  pg > "$WORK/schema_columns.tsv" <<SQL
SELECT table_name, ordinal_position, column_name, data_type, udt_name, is_nullable,
       coalesce(character_maximum_length::text, ''), coalesce(numeric_precision::text, ''), coalesce(numeric_scale::text, ''),
       coalesce(replace(replace(column_default, '|', '/'), E'\n', ' '), '')
FROM information_schema.columns WHERE table_schema = 'raw_duck' AND table_name IN ('$names') ORDER BY table_name, ordinal_position;
SQL
  pg > "$WORK/schema_keys.tsv" <<SQL
SELECT tc.table_name, tc.constraint_type, tc.constraint_name, string_agg(kcu.column_name, ',' ORDER BY kcu.ordinal_position)
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu ON kcu.constraint_schema = tc.constraint_schema AND kcu.constraint_name = tc.constraint_name
WHERE tc.table_schema = 'raw_duck' AND tc.table_name IN ('$names') AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')
GROUP BY 1, 2, 3 ORDER BY 1, 2, 3;
SQL
  pg > "$WORK/schema_views.tsv" <<SQL
SELECT c.relname, replace(replace(pg_get_viewdef(c.oid, true), E'\n', ' '), '|', '/')
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'raw_duck' AND c.relkind = 'v' ORDER BY 1;
SQL
  python3 - "$WORK" "$OUT/schema.json" "$TABLES" "$DATE" <<'PY'
import csv, json, os, sys
work, dest, tables_tsv, date = sys.argv[1:5]
meta = {r["table_name"]: r for r in csv.DictReader(open(tables_tsv, encoding="utf-8"), delimiter="\t")}
res = {r["table_name"]: r for r in csv.DictReader(open(os.path.join(work, "export_results.tsv"), encoding="utf-8"), delimiter="\t")}
pub = [t for t, m in meta.items() if m["decision"] == "publish"]
cols = {}
for line in open(os.path.join(work, "schema_columns.tsv"), encoding="utf-8"):
    f = line.rstrip("\n").split("|")
    if len(f) < 10:
        continue
    t, pos, name, dt, udt, nullable, clen, prec, scale, default = f[:10]
    cols.setdefault(t, []).append({"ordinal": int(pos), "name": name, "pg_data_type": dt, "pg_udt_name": udt,
        "nullable": nullable == "YES", "char_max_length": int(clen) if clen else None,
        "numeric_precision": int(prec) if prec else None, "numeric_scale": int(scale) if scale else None,
        "pg_default": default or None})
keys = {}
for line in open(os.path.join(work, "schema_keys.tsv"), encoding="utf-8"):
    f = line.rstrip("\n").split("|")
    if len(f) == 4:
        keys.setdefault(f[0], []).append({"type": f[1], "name": f[2], "columns": f[3].split(",")})
views = []
for line in open(os.path.join(work, "schema_views.tsv"), encoding="utf-8"):
    f = line.rstrip("\n").split("|", 1)
    if len(f) == 2:
        views.append({"view": f[0], "pg_definition": f[1].strip()})
out = {"lake_date": date, "bucket": "salem-data", "prefix": f"consignatio/_system/lake/{date}/",
       "source": "PostgreSQL casebible, schema raw_duck, on ovh-files (container casebible-pg18); exported with pg_duckdb 1.1.0 (DuckDB 1.4.3), Parquet zstd",
       "generated_by": "modules/Consignatio/casebible/tools/lake_publish_20260927.sh (Claude Code · Opus 5.5 · 2026-09-27)",
       "numeric_note": "pg_duckdb cannot read PostgreSQL NUMERIC columns without precision, so it reads them as DOUBLE; a column whose values were all integral and below 2^53 (exact in a double) was written as bigint, any other such column as double. See 'casts'.",
       "tables": [], "views_not_exported": views}
for t in sorted(pub):
    r = res.get(t, {})
    desc = []
    p = os.path.join(work, "describe", f"{t}.tsv")
    if os.path.exists(p):
        for line in open(p, encoding="utf-8"):
            f = line.rstrip("\n").split("|")
            if len(f) == 2:
                desc.append({"name": f[0], "parquet_duckdb_type": f[1]})
    ptype = {d["name"]: d["parquet_duckdb_type"] for d in desc}
    cl = sorted(cols.get(t, []), key=lambda c: c["ordinal"])
    for c in cl:
        c["parquet_duckdb_type"] = ptype.get(c["name"])
    out["tables"].append({"table": t, "object": f"{t}.parquet", "status": meta[t]["status"], "reason": meta[t]["reason"],
        "rows": int(r["parquet_rows"]) if r.get("parquet_rows") else None,
        "casts": [] if r.get("casts") in (None, "", "-") else r["casts"].split(","),
        "keys": keys.get(t, []), "columns": cl})
json.dump(out, open(dest, "w", encoding="utf-8"), indent=1, ensure_ascii=False)
missing = [t for t in pub if t not in res or not cols.get(t)]
print(f"schema.json: {len(out['tables'])} tables, {sum(len(x['columns']) for x in out['tables'])} columns, {len(views)} views; missing={missing}")
sys.exit(1 if missing else 0)
PY
  phase_end schema $?
}

# ---------------------------------------------------------------- upload (add-only)
upload_list() {
  : > "$WORK/upload_list.txt"
  local t
  for t in $(list_tables '^publish$'); do echo "$t.parquet" >> "$WORK/upload_list.txt"; done
  echo corrupt_missing.csv >> "$WORK/upload_list.txt"
  echo schema.json >> "$WORK/upload_list.txt"
}

do_upload() {
  phase_start upload
  local fails=0 f
  upload_list
  while read -r f; do [ -s "$OUT/$f" ] || { say "FAIL missing local $f"; fails=$((fails + 1)); }; done < "$WORK/upload_list.txt"
  [ "$fails" -eq 0 ] || { phase_end upload "$fails"; return; }
  say "uploading $(wc -l < "$WORK/upload_list.txt") objects, $(cd "$OUT" && du -cb $(cat "$WORK/upload_list.txt") | tail -1 | cut -f1) bytes -> $REMOTE/$DATE"
  rclone copy "$OUT" "$REMOTE/$DATE" --files-from "$WORK/upload_list.txt" --immutable --transfers 2 --checkers 4 \
    --stats 60s --stats-one-line -v --log-file "$LOGD/rclone-upload.log" || fails=$((fails + 1))
  rclone check "$OUT" "$REMOTE/$DATE" --files-from "$WORK/upload_list.txt" --one-way \
    --combined "$WORK/check_hash_combined.txt" --log-file "$LOGD/rclone-check-hash.log" || fails=$((fails + 1))
  rclone check "$OUT" "$REMOTE/$DATE" --files-from "$WORK/upload_list.txt" --one-way --size-only \
    --combined "$WORK/check_size_combined.txt" --log-file "$LOGD/rclone-check-size.log" || fails=$((fails + 1))
  grep -E 'differences found|matching files|hashes could not be checked' "$LOGD/rclone-check-hash.log" "$LOGD/rclone-check-size.log" | sed 's/^/  /'
  [ "$fails" -eq 0 ] && date -u +%FT%TZ > "$WORK/published_at.txt"
  phase_end upload "$fails"
}

# ---------------------------------------------------------------- readback (B2 bytes -> sha256 + Parquet row count)
do_readback() {
  phase_start readback
  local fails=0 f t sha want rows_pg rows_rb RB="$VOL/${TAG}-readback" CRB="$CVOL/${TAG}-readback"
  [ -e "$WORK/readback-$DATE" ] && { say "FAIL readback dir exists: $WORK/readback-$DATE (move it aside first)"; phase_end readback 1; return; }
  rclone copy "$REMOTE/$DATE" "$RB" --files-from "$WORK/upload_list.txt" --transfers 2 -v --log-file "$LOGD/rclone-readback.log" \
    || fails=$((fails + 1))
  printf 'object\tsha256_local\tsha256_b2\tsha_ok\trows_pg\trows_b2_parquet\trows_ok\n' > "$WORK/readback_results.tsv"
  while read -r f; do
    want=$(sha256sum "$OUT/$f" | cut -d' ' -f1)
    sha=$( [ -s "$RB/$f" ] && sha256sum "$RB/$f" | cut -d' ' -f1 || echo missing)
    rows_pg=-; rows_rb=-
    if [[ "$f" == *.parquet ]]; then
      t=${f%.parquet}
      rows_pg=$(awk -F'\t' -v t="$t" '$1==t{print $3}' "$RES")
      rows_rb=$(pg 2>&1 <<SQL | sed -n 's/^rows|//p'
$DUCK_SET
SELECT 'rows|' || count(*) FROM read_parquet('$CRB/$f');
SQL
)
    fi
    local sok=0 rok=0
    [ "$sha" = "$want" ] && sok=1
    [ "$rows_pg" = "$rows_rb" ] && rok=1
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$f" "$want" "$sha" "$sok" "$rows_pg" "$rows_rb" "$rok" >> "$WORK/readback_results.tsv"
    [ "$sok" = 1 ] && [ "$rok" = 1 ] || { say "MISMATCH $f sha_ok=$sok rows pg=$rows_pg b2=$rows_rb"; fails=$((fails + 1)); }
  done < "$WORK/upload_list.txt"
  mv -n "$RB" "$WORK/readback-$DATE" || fails=$((fails + 1))
  say "readback: $(awk -F'\t' 'NR>1 && $4==1 && $7==1' "$WORK/readback_results.tsv" | wc -l) of $(wc -l < "$WORK/upload_list.txt") objects match (sha256 and rows)"
  phase_end readback "$fails"
}

# ---------------------------------------------------------------- finalize: manifest.csv, LATEST, catalog table
do_finalize() {
  phase_start finalize
  local fails=0 f t rows bytes sha pub status m="$OUT/manifest.csv"
  [ -e "$MARK/readback.done" ] || { say "FAIL readback not done"; phase_end finalize 1; return; }
  [ -e "$m" ] && { say "FAIL $m exists (write-once)"; phase_end finalize 1; return; }
  pub=$(cat "$WORK/published_at.txt")
  echo 'table_name,rows,parquet_bytes,b2_key,sha256,published_at,status,object_kind' > "$m"
  while read -r f; do
    bytes=$(stat -c %s "$OUT/$f"); sha=$(sha256sum "$OUT/$f" | cut -d' ' -f1)
    case "$f" in
      *.parquet) t=${f%.parquet}; rows=$(awk -F'\t' -v t="$t" '$1==t{print $4}' "$RES"); status=$(status_of "$t"); kind=parquet ;;
      corrupt_missing.csv) t=corrupt_missing.csv; rows=$(( $(wc -l < "$OUT/$f") - 1 )); status=receipt_export_20260914; kind=csv ;;
      schema.json) t=schema.json; rows=$(python3 -c 'import json,sys; print(len(json.load(open(sys.argv[1]))["tables"]))' "$OUT/$f"); status=schema; kind=json ;;
    esac
    echo "$t,$rows,$bytes,$B2PREFIX/$DATE/$f,$sha,$pub,$status,$kind" >> "$m"
  done < "$WORK/upload_list.txt"
  printf '%s\n' "$DATE" > "$WORK/LATEST"
  rclone copy "$OUT" "$REMOTE/$DATE" --include manifest.csv --immutable -v --log-file "$LOGD/rclone-finalize.log" || fails=$((fails + 1))
  rclone copyto "$WORK/LATEST" "$REMOTE/LATEST" --immutable -v --log-file "$LOGD/rclone-finalize.log" || fails=$((fails + 1))
  rclone check "$OUT" "$REMOTE/$DATE" --include manifest.csv --one-way --log-file "$LOGD/rclone-finalize-check.log" || fails=$((fails + 1))
  [ "$(rclone cat "$REMOTE/LATEST")" = "$DATE" ] || { say "FAIL LATEST readback"; fails=$((fails + 1)); }
  [ "$(rclone cat "$REMOTE/$DATE/manifest.csv" | sha256sum | cut -d' ' -f1)" = "$(sha256sum "$m" | cut -d' ' -f1)" ] || { say "FAIL manifest readback"; fails=$((fails + 1)); }
  # catalog table rows = manifest.csv rows + manifest.csv itself + LATEST
  cp "$m" "$WORK/manifest_catalog.csv"
  echo "manifest.csv,$(( $(wc -l < "$m") - 1 )),$(stat -c %s "$m"),$B2PREFIX/$DATE/manifest.csv,$(sha256sum "$m" | cut -d' ' -f1),$(date -u +%FT%TZ),manifest,csv" >> "$WORK/manifest_catalog.csv"
  echo "LATEST,1,$(stat -c %s "$WORK/LATEST"),$B2PREFIX/LATEST,$(sha256sum "$WORK/LATEST" | cut -d' ' -f1),$(date -u +%FT%TZ),pointer,text" >> "$WORK/manifest_catalog.csv"
  # values are generated here from validated names, hex digests, integers and timestamps only
  if ! grep -qvE '^[A-Za-z0-9_.]+,[0-9]+,[0-9]+,[A-Za-z0-9_./-]+,[0-9a-f]{64},[0-9T:Z-]+,[a-z0-9_]+,[a-z]+$' <(tail -n +2 "$WORK/manifest_catalog.csv"); then
    { cat "$SQLFILE"
      echo "BEGIN;"
      tail -n +2 "$WORK/manifest_catalog.csv" | awk -F, -v q="'" '{printf "INSERT INTO raw_duck.lake_publish_20260927 (table_name, rows, parquet_bytes, b2_key, sha256, published_at, status, object_kind) VALUES (%s, %s, %s, %s, %s, %s, %s, %s);\n", q $1 q, $2, $3, q $4 q, q $5 q, q $6 q, q $7 q, q $8 q}'
      echo "COMMIT;"
      echo "SELECT 'catalog_rows|' || count(*) || '|' || sum(parquet_bytes) FROM raw_duck.lake_publish_20260927;"
    } > "$WORK/manifest_load.sql"
    pg < "$WORK/manifest_load.sql" | tee "$LOGD/manifest_load.out" || fails=$((fails + 1))
  else
    say "FAIL manifest_catalog.csv has a row outside the allowed shape"; fails=$((fails + 1))
  fi
  phase_end finalize "$fails"
}

# ---------------------------------------------------------------- s3probe: read every published object over the S3 API
# The lake's readers (DuckDB httpfs, Evidence.dev) use B2's S3 endpoint, not rclone's native API. This GETs every
# object in manifest_catalog.csv over S3 with boto3 and compares sha256 with the catalog. Credentials: two lines on
# stdin (S3 key id, application key); they are never printed or written.
do_s3probe() {
  phase_start s3probe
  python3 - "$WORK" "${S3_ENDPOINT:-https://s3.us-west-004.backblazeb2.com}" "${S3_REGION:-us-west-004}" 3<&0 <<'PY'
import csv, hashlib, os, sys
import boto3
from botocore.config import Config
work, endpoint, region = sys.argv[1:4]
creds = os.fdopen(3)
kid, sec = creds.readline().strip(), creds.readline().strip()
if not kid or not sec:
    sys.exit("no credentials on stdin")
s3 = boto3.client("s3", endpoint_url=endpoint, region_name=region, aws_access_key_id=kid, aws_secret_access_key=sec,
                  config=Config(signature_version="s3v4", s3={"addressing_style": "path"},
                                request_checksum_calculation="when_required", response_checksum_validation="when_required"))
rows = list(csv.DictReader(open(os.path.join(work, "manifest_catalog.csv"), encoding="utf-8")))
ok = bad = 0
with open(os.path.join(work, "s3probe_results.tsv"), "w", encoding="utf-8") as out:
    out.write("b2_key\tbytes_catalog\tbytes_s3\tsha256_catalog\tsha256_s3\tok\n")
    for r in rows:
        try:
            body = s3.get_object(Bucket="salem-data", Key=r["b2_key"])["Body"]
            h, n = hashlib.sha256(), 0
            for chunk in iter(lambda: body.read(1 << 20), b""):
                h.update(chunk); n += len(chunk)
            got = h.hexdigest()
        except Exception as e:  # report the error class only; never the request
            got, n = "error:" + type(e).__name__ + ":" + str(getattr(e, "response", {}).get("Error", {}).get("Code", "")), -1
        good = int(got == r["sha256"] and n == int(r["parquet_bytes"]))
        ok += good; bad += 1 - good
        out.write(f"{r['b2_key']}\t{r['parquet_bytes']}\t{n}\t{r['sha256']}\t{got}\t{good}\n")
print(f"s3probe: {ok} of {len(rows)} objects read over S3 ({endpoint}) with matching sha256 and size; {bad} failed")
sys.exit(1 if bad else 0)
PY
  phase_end s3probe $?
}

do_status() {
  ls -la "$MARK" 2>/dev/null
  [ -s "$RES" ] && { echo "exported: $(($(wc -l < "$RES") - 1)) tables, $(awk -F'\t' 'NR>1{s+=$6} END{print s}' "$RES") bytes, ok=$(awk -F'\t' 'NR>1 && $12==1' "$RES" | wc -l)"; tail -3 "$RES" | cut -f1-6,11,12; }
  true
}

case "${1:-}" in
  export) do_export ;;
  schema) do_schema ;;
  upload) do_upload ;;
  readback) do_readback ;;
  finalize) do_finalize ;;
  s3probe) do_s3probe ;;
  status) do_status ;;
  *) echo "usage: $0 export|schema|upload|readback|finalize|s3probe|status" >&2; exit 2 ;;
esac
