#!/bin/sh
# Byline: Codex · GPT-5 · 2026-08-29 (movable private Workbench identity, originally a set-config file).
# Byline: Claude Code · Opus 5.5 · 2026-09-27 (tailnet-bound door, app caps, script form).
#
# Declared Tailscale Serve configuration for svc:workbench (VIP 100.105.91.39,
# https://workbench.tilapia-skilift.ts.net). Run as root on the tagged service host that runs
# the Workbench app (today ovh-app, 100.72.169.40); idempotent. The host is already approved
# for svc:workbench; a new host must be approved through the API
# (POST /api/v2/tailnet/-/services/svc:workbench/device/<nodeId>/approved {"approved": true}).
#
# Serve terminates HTTPS for the service and proxies to the Workbench's one tailnet-bound
# port, 100.72.169.40:9071 (deploy/workbench.yaml). Dialled from the host, its socket peer is
# the host's own tailnet address, which is the Workbench's
# WORKBENCH_TAILSCALE_SERVE_PROXY_CIDRS=100.72.169.40/32. Traefik's public hop to the same
# port arrives masqueraded as 10.201.8.1 instead, so the two doors stay distinguishable.
# --accept-app-caps forwards the tailnet grant propria.mitechconsult.com/cap/workbench.
# No Authentik on this door (owner 2026-09-26). Never funnel this service.
#
# Why a script and not a `tailscale serve set-config` file: on tailscale 1.102.2 the file
# format can express neither an HTTPS listener in front of a plain-HTTP backend nor
# accepted app capabilities (see authentik-serve.sh; measured 2026-09-27).
set -eu
tailscale serve clear svc:workbench
tailscale serve --service=svc:workbench --bg --https=443 \
  --accept-app-caps=propria.mitechconsult.com/cap/workbench http://100.72.169.40:9071
tailscale serve advertise svc:workbench
tailscale serve status --json | python3 -c 'import json,sys; print(json.dumps(json.load(sys.stdin)["Services"]["svc:workbench"], indent=1))'
