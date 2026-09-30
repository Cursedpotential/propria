#!/usr/bin/env bash
# fct_sources_inventory_20260928.sh - list every copy of the Family Court Toolkit's primary legal sources and load
# the listings into the catalog (raw_duck.fct_sources_inventory_20260928) with fct_sources_inventory_20260928.sql.
#
# Byline: Claude Code · Opus 5.5 · 2026-09-28 (agent sources-b2-sync for the "portal cut over" session)
# Owner 2026-09-28 04:01 EDT: "Everything needs to be migrated to B2. R2 is being retired."
# Plan and receipt: modules/Consignatio/docs/receipts/2026-09-28-fct-sources-r2-to-b2-plan.md
#
# Read-only against every storage surface. Runs ON ovh-files as root. It writes only TSVs under $WORK and the
# catalog table. Inputs it cannot produce itself arrive in $WORK before it runs:
#   desktop_local.tsv     the desktop plugin copy, hashed on the desktop (path, size, md5, sha256)
#   store_reference.tsv   surreal-case reference rows under sources/primary (path, size=-1, md5='', sha256)
# Produced here:
#   r2.tsv                one R2 listing call on r2:casebible-sorted/fct-sources/primary (size + md5 ETag)
#   vps_local.tsv         the OpenCode-home plugin copy on ovh-files (the 2026-09-08 upload source), hashed here
#   b2_probe.tsv          one listing of the proposed B2 prefix (plan decision D1; expected empty before the transfer)
set -euo pipefail
export LC_ALL=C   # join needs both inputs sorted in the same collation

WORK=${WORK:-/data/consignatio/fct-sources-20260928}
PGC=${PGC:-fgz1n7useplhk0t91uk7k1aw}
R2CONF=${R2CONF:-/opt/casebible/rclone.conf}
R2SRC=r2:casebible-sorted/fct-sources/primary
VPSSRC=${VPSSRC:-/data/probata/volumes/opencode/home/.claude/local-plugins/plugins/family-court-toolkit/content/custody-guide/sources/primary}
B2DST=${B2DST:-b2native-full:salem-data/consignatio/casevault/KnowledgeBase/legal/fct-sources/primary}
mkdir -p "$WORK"
cd "$WORK"

rclone --config "$R2CONF" lsf -R --files-only --format pst --separator $'\t' "$R2SRC" > r2.path_size_time.tsv
rclone --config "$R2CONF" lsf -R --files-only --format ph --hash MD5 --separator $'\t' "$R2SRC" > r2.path_md5.tsv
join -t $'\t' <(sort r2.path_size_time.tsv) <(sort r2.path_md5.tsv) | awk -F'\t' 'BEGIN{OFS="\t"} {print $1,$2,$4,""}' > r2.tsv

( cd "$VPSSRC" && find . -type f -printf '%P\n' | sort | while IFS= read -r p; do
    printf '%s\t%s\t%s\t%s\n' "$p" "$(stat -c %s "$p")" "$(md5sum "$p" | cut -d' ' -f1)" "$(sha256sum "$p" | cut -d' ' -f1)"
  done ) > vps_local.tsv

rclone lsf -R --files-only --format ps --separator $'\t' "$B2DST" > b2_probe.tsv 2>/dev/null || : > b2_probe.tsv

for f in desktop_local.tsv store_reference.tsv r2.tsv vps_local.tsv b2_probe.tsv; do
  [ -e "$f" ] || { echo "missing $WORK/$f" >&2; exit 1; }
  echo "$f $(wc -l < "$f") rows"
done
docker cp "$WORK/." "$PGC:/tmp/fct-sources-20260928/"
docker exec -i "$PGC" psql -U postgres -d casebible -X -v ON_ERROR_STOP=1 -f - < "$WORK/fct_sources_inventory_20260928.sql"
