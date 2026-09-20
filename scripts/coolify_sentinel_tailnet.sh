#!/usr/bin/env bash
# Byline: Claude Code · Fable 5.1 · 2026-09-20
#
# Point Coolify Sentinel on the remote servers at the control plane's TAILNET
# address. The public :8000 was closed on 2026-09-14, but ovh-app and ovh-files
# still pushed to http://74.208.130.34:8000, so every push failed and Coolify
# stopped seeing both servers (last reports 09-15 / 09-19).
#
# Runs ON ion-control (ssh debian@100.98.98.38, key ~/.ssh/ovh):
#   ssh ... 'bash -s' < scripts/coolify_sentinel_tailnet.sh
# Changes one column on two rows, then uses Coolify's own StartSentinel action
# to recreate the sentinel containers with the new endpoint. Idempotent.
set -euo pipefail
URL="http://100.98.98.38:8000"
PSQL="sudo docker exec coolify-db psql -U coolify -d coolify -At -v ON_ERROR_STOP=1"

echo "before:"; $PSQL -F ' | ' -c "select s.name, ss.sentinel_custom_url from servers s join server_settings ss on ss.server_id=s.id order by s.id"
$PSQL -c "update server_settings set sentinel_custom_url='${URL}', updated_at=now() where server_id in (select id from servers where name in ('ovh-app','ovh-files')) and sentinel_custom_url is distinct from '${URL}'"
for name in ovh-app ovh-files; do
  sudo docker exec coolify php artisan tinker --execute="\\App\\Actions\\Server\\StartSentinel::run(\\App\\Models\\Server::where('name','${name}')->firstOrFail(), true); echo 'restarted ${name}';"
done
echo "after:"; $PSQL -F ' | ' -c "select s.name, ss.sentinel_custom_url from servers s join server_settings ss on ss.server_id=s.id order by s.id"
