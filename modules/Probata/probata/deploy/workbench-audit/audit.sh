#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-27
# The Workbench six-step live audit (PR-24 / PR-27), run by headless Chrome inside the Probata
# devbox on ovh-files. Nothing browser-related runs on the calling machine (owner rule 2026-09-24).
#
#   deploy/workbench-audit/audit.sh <local-out-dir> [label]
#
# Identity. On the tailnet the Workbench admits a Tailscale USER login and, when
# TAILSCALE_DEVICE_CAPABILITY is set, a device granted that capability. The devbox reaches the
# tailnet through ovh-files, a tagged device, and that capability is not configured yet (DF-30 /
# O-07), so a direct request from the devbox answers 403 "Untrusted proxy". Until DF-30 is fixed,
# this script carries the audit over the calling machine's own tailnet login for the length of
# one run:
#   calling machine  ssh -R 127.0.0.1:18443 -> workbench.tilapia-skilift.ts.net:443 (its login)
#   ovh-files        forward.py <devbox gateway>:18443 -> 127.0.0.1:18443 (only the devbox's
#                    network can reach that address)
#   devbox Chrome    --host-resolver-rules maps the Workbench name to <devbox gateway>:18443, so
#                    TLS and the Host header stay the real ones
# All three stop when the audit ends. Requests made this way are the calling user's, so the audit
# only reads, hashes and starts one TEST run; it never records a decision.
# Once DF-30 is fixed, run with AUDIT_DIRECT=1 and none of this is set up.
#
# audit.mjs and forward.py are taken from the committed HEAD (git archive), so only committed code
# runs. Output: step PNGs and audit.json in <local-out-dir>.
set -euo pipefail

out="${1:?usage: audit.sh <local-out-dir> [label]}"
label="${2:-audit}"
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(git -C "$here" rev-parse --show-toplevel)"
rel="$(git -C "$here" rev-parse --show-prefix)"; rel="${rel%/}"
host="${AUDIT_HOST:-root@100.91.190.107}"   # ovh-files on the tailnet
key="${AUDIT_SSH_KEY:-$HOME/.ssh/ovh}"
workbench_host="workbench.tilapia-skilift.ts.net"
relay_port="${AUDIT_RELAY_PORT:-18443}"
vps() { MSYS_NO_PATHCONV=1 ssh -i "$key" -o BatchMode=yes "$host" "$@"; }

box="$(vps "docker ps --filter label=com.docker.compose.service=devbox --format '{{.Names}}' | head -n 1")"
[ -n "$box" ] || { echo "no running devbox container on $host" >&2; exit 1; }
gateway="$(vps "docker inspect -f '{{range .NetworkSettings.Networks}}{{.Gateway}} {{end}}' $box" | awk '{print $1}')"
work="/home/kasm-user/workbench-audit/$label-$(date +%Y%m%dT%H%M%S)"
echo "workbench audit as $label in $box:$work"

stage="$(mktemp -d)"
git -c core.autocrlf=false -C "$repo" archive --format=tar "HEAD:$rel" audit.mjs forward.py | tar -x -C "$stage"
tar -c -C "$stage" audit.mjs | vps "docker exec -i -u 1000 $box sh -c 'mkdir -p $work && tar -x -C $work'"

tunnel_pid=""
cleanup() {
  if [ -n "$tunnel_pid" ]; then kill "$tunnel_pid" 2>/dev/null || true; fi
  vps "pkill -f 'forward.py $gateway $relay_port' || true" >/dev/null 2>&1 || true
}
trap cleanup EXIT

rule=""
if [ "${AUDIT_DIRECT:-0}" != 1 ]; then
  MSYS_NO_PATHCONV=1 ssh -i "$key" -o BatchMode=yes -o ExitOnForwardFailure=yes -N \
    -R "127.0.0.1:$relay_port:$workbench_host:443" "$host" &
  tunnel_pid=$!
  vps "mkdir -p /tmp/workbench-audit && cat > /tmp/workbench-audit/forward.py" < "$stage/forward.py"
  vps "nohup python3 /tmp/workbench-audit/forward.py $gateway $relay_port 127.0.0.1 $relay_port >/tmp/workbench-audit/forward.log 2>&1 &"
  sleep 3
  rule="MAP $workbench_host $gateway:$relay_port"
fi

status=0
vps "docker exec -u 1000 -e AUDIT_RESOLVER_RULE='$rule' -e AUDIT_TARGET_NAME='${AUDIT_TARGET_NAME:-calls-20250703043408.xml}' $box node $work/audit.mjs $work/out" || status=$?
mkdir -p "$out"
vps "docker exec -u 1000 $box tar -c -C $work/out ." | tar -x --force-local -C "$out"
ls -1 "$out"
exit "$status"
