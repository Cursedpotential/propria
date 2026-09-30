#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-27
# Replace one Traefik dynamic file on ovh-app from a staged copy, safely. Run ON ovh-app:
#   ssh root@100.72.169.40 'bash -s' -- <file.yaml> <expected-current-sha256> <expected-new-sha256> < apply_dynamic_file.sh
# The staged copy must already be at /data/coolify/proxy/dynamic/<file.yaml>.staged-edge (an
# extension Traefik's file provider ignores). Aborts unless the live file and the staged copy match
# the expected hashes; backs the live file up beside itself; swaps with one rename; shows any
# Traefik error logged in the next few seconds.
set -euo pipefail
file="$1"; want_cur="$2"; want_new="$3"
cd /data/coolify/proxy/dynamic
cur=$(sha256sum "$file" | cut -d' ' -f1)
[ "$cur" = "$want_cur" ] || { echo "live $file is $cur, expected $want_cur: abort"; exit 3; }
new=$(sha256sum "$file.staged-edge" | cut -d' ' -f1)
[ "$new" = "$want_new" ] || { echo "staged copy is $new, expected $want_new: abort"; exit 4; }
ts=$(date -u +%Y%m%dT%H%M%SZ)
cp -p "$file" "$file.bak-$ts-pre-edge"
mv "$file.staged-edge" "$file"
echo "applied $file at $ts (backup $file.bak-$ts-pre-edge)"
sleep 4
docker logs --since "$(date -u -d '-10 sec' +%Y-%m-%dT%H:%M:%S)" coolify-proxy 2>&1 | grep -E 'ERR|error' | grep -v -i 'acme' | tail -5 || true
