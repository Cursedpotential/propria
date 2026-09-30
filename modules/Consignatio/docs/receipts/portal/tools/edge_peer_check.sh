#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-27
# Read-only edge check, run ON ovh-app:  ssh root@100.72.169.40 'bash -s' < edge_peer_check.sh
# Shows how Traefik reaches Authentik and the Workbench and which socket peer each backend sees.
#  - Authentik: the outpost check sent from INSIDE coolify-proxy through svc:authentik's raw TCP
#    forwarder must 302 to the homepage.int callback, and Authentik must log host=homepage.int,
#    scheme=https (i.e. it trusted the peer and honoured X-Forwarded-*).
#  - Workbench: from inside coolify-proxy (Traefik's hop, peer 10.201.8.1) a request without
#    identity must be refused as "Missing or invalid Authentik identity" (trusted proxy); from the
#    host (Serve's hop, peer 100.72.169.40) as "Untrusted proxy" (no Tailscale login header).
set -u
VIP=100.66.241.25   # svc:authentik
echo "== containers"
for p in authentik-server workbench; do
  c=$(docker ps --format '{{.Names}}' | grep "^$p-" | head -1)
  echo "$c $(docker inspect "$c" --format '{{.State.StartedAt}} {{.State.Health.Status}}')"
  docker inspect "$c" --format '{{range $k,$v := .NetworkSettings.Networks}}  net {{$k}} {{$v.IPAddress}}{{"\n"}}{{end}}{{range $p,$b := .NetworkSettings.Ports}}  port {{$p}} {{json $b}}{{"\n"}}{{end}}'
  docker inspect "$c" --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -E 'TRUSTED_' | sed 's/^/  env /'
done
echo "== coolify-proxy networks"; docker inspect coolify-proxy --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}={{$v.IPAddress}} {{end}}'; echo
echo "== serve"; tailscale serve status --json | python3 -c 'import json,sys; s=json.load(sys.stdin)["Services"]; [print(k, json.dumps(s[k])) for k in ("svc:authentik","svc:workbench") if k in s]'
echo "== traefik file"; sha256sum /data/coolify/proxy/dynamic/propria-public-portal.yaml
grep -n -E '10\.201\.0\.|100\.66\.241\.25|100\.72\.169\.40:9071' /data/coolify/proxy/dynamic/propria-public-portal.yaml | sed 's/^/  /'
T0=$(date -u +%Y-%m-%dT%H:%M:%S)
echo "== authentik outpost via svc:authentik from inside coolify-proxy"
docker exec coolify-proxy sh -c "wget -S -O /dev/null -T 10 --header 'X-Forwarded-Proto: https' --header 'X-Forwarded-Host: homepage.int.mitechconsult.com' --header 'X-Forwarded-Uri: /edge-peer-check' --header 'X-Forwarded-Method: GET' --header 'X-Forwarded-For: 203.0.113.9' http://$VIP:9075/outpost.goauthentik.io/auth/traefik 2>&1" \
  | grep -m2 -E 'HTTP/|Location' | cut -c1-200
sleep 1
ak=$(docker ps --format '{{.Names}}' | grep '^authentik-server-' | head -1)
docker logs --since "$T0" "$ak" 2>&1 | grep 'edge-peer-check\|outpost.goauthentik.io/auth/traefik' | python3 -c '
import sys, json
for line in sys.stdin:
    try: d = json.loads(line)
    except Exception: continue
    sp = (d.get("spans") or [{}])[0]
    print("  authentik logged:", d.get("status"), "remote=", d.get("remote") or sp.get("remote"), "host=", d.get("host") or sp.get("host"), "scheme=", d.get("scheme") or sp.get("scheme"))
' | head -2
echo "== workbench: Traefik hop (from inside coolify-proxy) vs Serve hop (from the host)"
echo "  proxy: $(docker exec coolify-proxy sh -c 'wget -q -O - -T 8 http://100.72.169.40:9071/tools 2>&1' | head -c 160)"
echo "  host : $(curl -sS -m 8 http://100.72.169.40:9071/tools | head -c 160)"
wb=$(docker ps --format '{{.Names}}' | grep '^workbench-' | head -1)
docker logs --since "$T0" "$wb" 2>&1 | grep -E '"GET /tools' | sed 's/^/  workbench logged: /' | tail -3
