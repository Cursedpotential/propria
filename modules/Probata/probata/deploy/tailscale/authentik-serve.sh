#!/bin/sh
# Byline: Claude Code · Opus 5.5 · 2026-09-27.
#
# Declared Tailscale Serve configuration for svc:authentik. Run as root on the service host
# (today ovh-app, 100.72.169.40); idempotent. Service definition, registered once through the
# API (PUT /api/v2/tailnet/-/vip-services/svc:authentik): ports tcp:443 and tcp:9075, no tags
# (the OAuth client may not assign tag:docker; svc:coolify is registered the same way).
# VIP 100.66.241.25 / fd7a:115c:a1e0::a329:f11a, https://authentik.tilapia-skilift.ts.net.
# After the first apply on a new host, approve that host for the service through the API
# (POST /api/v2/tailnet/-/services/svc:authentik/device/<nodeId>/approved {"approved": true}).
#
# tcp:443  HTTPS endpoint, the owner's tailnet door to the Authentik admin UI. Serve's HTTP
#          proxy sends X-Forwarded-Proto: https and Host authentik.tilapia-skilift.ts.net, so
#          Authentik builds https URLs for that name. No login barrier beyond Authentik's own.
# tcp:9075 raw TCP forwarder, Traefik's internal hop for auth.int.mitechconsult.com and every
#          public forward-auth check. Raw TCP on purpose: Serve's HTTP proxy overwrites
#          X-Forwarded-Host and X-Forwarded-For, which breaks forward-auth.
#
# Both targets are the host's own tailnet address, so Authentik always sees the socket peer
# 100.72.169.40 (TRAEFIK_PROXY_CIDR=100.72.169.40/32 in the Authentik Coolify app). Authentik
# publishes that port only on the tailnet address (deploy/authentik.yaml). Never funnel this.
#
# Why a script and not a `tailscale serve set-config` file: on tailscale 1.102.2 the file format
# cannot express "HTTPS listener in front of a plain-HTTP backend". An http:// target makes port
# 443 plain HTTP, and an https:// target makes Serve dial the backend with TLS (measured
# 2026-09-27). get-config prints the HTTPS door as "tcp:443": "http://...", which set-config
# would then apply as plain HTTP, so do not round-trip this service through a file.
set -eu
tailscale serve clear svc:authentik
tailscale serve --service=svc:authentik --bg --https=443 http://100.72.169.40:9075
tailscale serve --service=svc:authentik --bg --tcp=9075 tcp://100.72.169.40:9075
tailscale serve advertise svc:authentik
tailscale serve status --json | python3 -c 'import json,sys; print(json.dumps(json.load(sys.stdin)["Services"]["svc:authentik"], indent=1))'
