#!/bin/sh
# Byline: Claude Code · Opus 5.5 · 2026-10-02
#
# Declared Tailscale Serve configuration for the Case Bible catalog (Coolify database
# casebible-pg18, uuid fgz1n7useplhk0t91uk7k1aw). Run as root on ovh-files (100.91.190.107);
# idempotent.
#
# Route (owner option A, 2026-10-02):
#   tailnet 100.91.190.107:5433  --tailscale serve tcp-->  127.0.0.1:5475  --docker-->  container :5432
# The loopback port is the database's "Ports Mappings" field in Coolify (127.0.0.1:5475:5432), so a
# Coolify restart or redeploy keeps it. Nothing binds the catalog on the tailnet address except this
# forward, and the forward never names a container IP: Docker reassigns those on recreate (the first
# 5433 forward pointed at 172.18.0.3 and would have broken on the next recreate).
#
# Callers use 100.91.190.107:5433 (or ovh-files.tilapia-skilift.ts.net:5433).
set -eu
tailscale serve --tcp=5433 off 2>/dev/null || true
tailscale serve --bg --tcp=5433 tcp://127.0.0.1:5475
tailscale serve status --json | python3 -c 'import json,sys; print(json.dumps(json.load(sys.stdin)["TCP"]["5433"]))'
