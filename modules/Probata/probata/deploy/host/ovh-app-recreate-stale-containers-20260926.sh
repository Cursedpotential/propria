#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-26
# Recreate the ovh-app containers that a plain `docker restart` could not start: their rootfs mount points were on the
# pre-2026-09-24 Docker store, which was deleted after the disk move ("wandered into deleted directory .../proc").
# Same compose file, same image, same volumes: `up -d --force-recreate` builds a fresh container on the current store.
# Run on ovh-app as root, from the desktop:
#   ssh -i ~/.ssh/ovh root@100.72.169.40 'bash -s' < deploy/host/ovh-app-recreate-stale-containers-20260926.sh
set -u
rec() {  # project, project dir, compose file, service
  p="$1"; w="$2"; f="$3"; s="$4"
  if [ ! -f "$f" ]; then echo "MISSING $f"; return; fi
  if docker compose -p "$p" --project-directory "$w" -f "$f" up -d --force-recreate --no-deps "$s" >/tmp/rec.log 2>&1; then
    echo "OK   $p/$s"
  else
    echo "FAIL $p/$s: $(tail -2 /tmp/rec.log | tr '\n' ' ')"
  fi
}
rec coolify-proxy /data/coolify/proxy /data/coolify/proxy/docker-compose.yml traefik
for pair in ws67wgw1qxdgxo956p2k1jvi:tool-gateway e1mshujml6bv8ldtoe8n7je0:tool-runtime \
            k272znxpa4gh6drmolut723w:contextforge z5787t1l7gl2zbrya8cxzapf:portkey \
            f29166r47gro6fjiq4d8ya92:gateway zb0hi2bi26vnndyw9eozn737:metabase-db \
            zb0hi2bi26vnndyw9eozn737:metabase oyzznioap03u34xz125l90oq:coolify-mcp \
            mn2autapl223gmgcpjqy7def:sandbox; do
  p="${pair%%:*}"; s="${pair#*:}"; d=/data/coolify/applications/$p
  f=$d/docker-compose.yaml; [ -f "$f" ] || f=$d/docker-compose.yml
  rec "$p" "$d" "$f" "$s"
done
rec homv6zeg4ay2r2puxtzakf83 /data/coolify/services/homv6zeg4ay2r2puxtzakf83 /data/coolify/services/homv6zeg4ay2r2puxtzakf83/docker-compose.yml progress-board
rec rde5k1xdda3q5fb73o8r9if5 /data/coolify/services/rde5k1xdda3q5fb73o8r9if5 /data/coolify/services/rde5k1xdda3q5fb73o8r9if5/docker-compose.yml filestash
rec portal-editor /data/probata/config/portal-editor /data/probata/config/portal-editor/compose.yml portal-editor
rec dashboards /data/dashboards /data/dashboards/compose.yml homepage
echo "running containers: $(docker ps -q | wc -l)"
echo "not compose-managed, still stopped: homepage-public, coolify-tailnet-forward (need their original docker run)"
