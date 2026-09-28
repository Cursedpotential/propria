#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-27
# Survival test for the public edge, run ON ovh-app. It recreates coolify-proxy the way Coolify's
# proxy start does and splits Coolify's two steps so the edge can be probed in between:
#   ssh root@100.72.169.40 'bash -s' -- recreate < proxy_recreate_test.sh
#       prints the networks the proxy is on, then `docker compose up -d --force-recreate --wait`
#       from /data/coolify/proxy (the compose Coolify renders). The new container is on the
#       declared `coolify` network only, which is exactly what today's 13:48 recreate produced.
#       Probe the public routes now: the edge must not need any other network.
#   ssh root@100.72.169.40 'bash -s' -- reconnect <net> [<net> ...] < proxy_recreate_test.sh
#       Coolify's next step (connect the proxy to each Coolify app network) for the networks the
#       first call printed, so docker-label routes of other apps come back. Hand-made networks
#       (propria-edge) are refused: nothing on the edge may depend on them.
set -euo pipefail
cd /data/coolify/proxy
case "${1:-}" in
  recreate)
    echo "before: $(docker inspect coolify-proxy --format '{{.State.StartedAt}}') networks: $(docker inspect coolify-proxy --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}')"
    docker compose up -d --force-recreate --wait
    echo "after:  $(docker inspect coolify-proxy --format '{{.State.StartedAt}} {{.State.Health.Status}}') networks: $(docker inspect coolify-proxy --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}={{$v.IPAddress}} {{end}}')"
    ;;
  reconnect)
    shift
    for net in "$@"; do
      case "$net" in propria-edge) echo "refusing hand-made network $net"; continue ;; esac
      if docker inspect coolify-proxy --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}' | tr ' ' '\n' | grep -qx "$net"; then
        echo "already on $net"
      else
        docker network connect "$net" coolify-proxy && echo "connected $net"
      fi
    done
    echo "now: $(docker inspect coolify-proxy --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}')"
    ;;
  *) echo "usage: recreate | reconnect <net>..." >&2; exit 2 ;;
esac
