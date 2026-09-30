#!/bin/sh
# =============================================================================
# docstore-nightly — rebuild the Docstore from git and re-index, while nobody is working
# =============================================================================
# Byline: Claude Code · Opus 5 · 2026-09-28
#
# Owner 2026-09-28, on the "a docs push rebuilds the image" contract: "Let's set up a Cron job
# or something ... during down time or idle time."
#
# WHY NOT A PUSH WEBHOOK. Every deployment of this app back to 2026-09-19 is is_webhook=false,
# is_api=true -- push-triggered deploys were never wired. Wiring them would rebuild and restart
# the Docstore on every docs commit, which kills an in-flight index: a full run takes ~23 minutes
# and a restart loses it. A nightly rebuild picks the same changes up when no one is waiting.
#
# WHAT ONE RUN DOES
#   1. refuses to start if a sync is already running -- a rebuild would kill it
#   2. asks Coolify to deploy, which re-clones main and rebuilds the image, so new documents in
#      git arrive, and the container's start-up re-merges the /extras host-only documents
#   3. waits for the new container to answer, then asks it for a full sync
#   4. appends one line per stage to the log; it never deletes anything
#
# It does NOT push the host-only documents. Those come from a checkout, by hand, with
# tools/push-local-docstore-sources.py -- a scheduled job on the VPS cannot read the desktop.
#
# INSTALL (once, on ovh-files, as root):
#   install -m 0755 docstore-nightly.sh /data/probata/bin/docstore-nightly.sh
#   install -d -m 0700 /data/probata/secrets
#   printf 'COOLIFY_API_TOKEN=%s\n' "<token>" > /data/probata/secrets/coolify-api.env
#   chmod 0600 /data/probata/secrets/coolify-api.env
#   ( crontab -l 2>/dev/null; echo '15 8 * * * /data/probata/bin/docstore-nightly.sh' ) | crontab -
#
# 08:15 UTC is 04:15 in the owner's timezone. The host runs UTC.
# =============================================================================
set -eu

APP_UUID=ywo2qvc5catoa79zgdur5o2j
COOLIFY_API=http://100.98.98.38:8000/api/v1
SECRETS=/data/probata/secrets/coolify-api.env
LOG=/data/probata/volumes/docstore-worker/nightly.log

log() { printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >> "$LOG"; }

container() { docker ps --format '{{.Names}}' | grep '^docstore-' | head -1; }

# The token is read, never echoed. `.` would run a value containing spaces as a command, so the
# file is parsed instead.
TOKEN=$(sed -n 's/^[[:space:]]*COOLIFY_API_TOKEN[[:space:]]*=[[:space:]]*//p' "$SECRETS" | head -1)
[ -n "$TOKEN" ] || { log "ABORT no COOLIFY_API_TOKEN in $SECRETS"; exit 1; }

BEFORE=$(container)
[ -n "$BEFORE" ] || { log "ABORT no docstore container running"; exit 1; }

# 1. Never interrupt a running index.
STATE=$(docker exec "$BEFORE" python -c "
import json,os,urllib.request
r=urllib.request.Request('http://127.0.0.1:8000/runs/current',
  headers={'Authorization':'Bearer '+os.environ['DOCSTORE_API_TOKEN']})
d=json.load(urllib.request.urlopen(r,timeout=20))
print((d.get('startup_or_latest_sync') or d).get('sync',''))
" 2>/dev/null || echo unknown)
if [ "$STATE" = "running" ]; then
  log "SKIP a sync is already running; a rebuild would kill it"
  exit 0
fi
log "START previous sync state=$STATE container=$BEFORE"

# 2. Rebuild from main.
CODE=$(curl -s -o /tmp/docstore-nightly-deploy.json -w '%{http_code}' --max-time 60 \
  -H "Authorization: Bearer $TOKEN" "$COOLIFY_API/deploy?uuid=$APP_UUID&force=false")
if [ "$CODE" != "200" ]; then
  log "ABORT deploy request returned HTTP $CODE"
  exit 1
fi
log "DEPLOY requested"

# 3. Wait for a different container that answers. Health stays 'unhealthy' until a sync
#    succeeds, so the gate is "the API responds", not "docker calls it healthy".
NOW=""
i=0
while [ "$i" -lt 60 ]; do
  sleep 30
  i=$((i + 1))
  NOW=$(container)
  [ -n "$NOW" ] && [ "$NOW" != "$BEFORE" ] || continue
  docker exec "$NOW" python -c "
import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/health', timeout=10)
" >/dev/null 2>&1 && break
done
if [ -z "$NOW" ] || [ "$NOW" = "$BEFORE" ]; then
  log "ABORT no new container after 30 minutes; the old one is still serving"
  exit 1
fi
log "DEPLOYED container=$NOW"
docker logs "$NOW" 2>&1 | grep -m1 'local sources' | sed 's/^/           /' >> "$LOG" || true

# 4. Re-index.
RUN=$(docker exec "$NOW" python -c "
import json,os,urllib.request
req=urllib.request.Request('http://127.0.0.1:8000/runs',
  data=json.dumps({'scope':'full','index_kind':'docs'}).encode(),
  headers={'Content-Type':'application/json',
           'Authorization':'Bearer '+os.environ['DOCSTORE_API_TOKEN']}, method='POST')
print(json.load(urllib.request.urlopen(req,timeout=60)).get('run_id',''))
" 2>/dev/null || echo "")
[ -n "$RUN" ] || { log "ABORT sync request failed"; exit 1; }
log "SYNC started run_id=$RUN"
