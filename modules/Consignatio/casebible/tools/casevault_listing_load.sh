#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-10-02
# Load one B2 listing of b2:salem-data/consignatio/casevault/ into the catalog (raw_duck.casevault_objects).
# Runs ON ovh-files. The listing is produced by (read-only, Class C list calls only):
#   rclone --config /opt/casebible/rclone.conf lsf -R --files-only --fast-list --format psh --hash SHA-1 \
#     --separator $'\t' b2:salem-data/consignatio/casevault/ > listing.tsv
# with EnvironmentFile=/data/consignatio/secrets/rclone-b2-intake.env (systemd-run). Append-only: each load
# adds one listed_at generation; raw_duck.casevault_objects_current is the newest generation per key.
# Usage: casevault_listing_load.sh <listing.tsv rel_key<TAB>size<TAB>sha1>
set -euo pipefail
LISTING="${1:?listing tsv}"
PG=fgz1n7useplhk0t91uk7k1aw
LISTED_AT="$(date -u -r "$LISTING" +%Y-%m-%dT%H:%M:%SZ)"
docker cp "$LISTING" "$PG:/tmp/casevault_listing.tsv"
docker cp "$(dirname "$0")/casevault_listing_load.sql" "$PG:/tmp/casevault_listing_load.sql"
docker exec "$PG" psql -U postgres -d casebible -At -v ON_ERROR_STOP=1 -v listed_at="$LISTED_AT" \
  -f /tmp/casevault_listing_load.sql
