#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-27
# Headless-Chrome check of the Authentik edge, run inside the Probata devbox on ovh-files (never on
# the owner's desktop). Reuses the committed portal runner modules/Probata/probata/deploy/portal/
# shoot.mjs with the committed plan edge-shots.json beside this script: auth.int, workbench.int and
# homepage.int logged out (each must end on the Authentik login with zero console errors, i.e. no
# blocked mixed content) and the tailnet admin door https://authentik.tilapia-skilift.ts.net.
#   shoot_edge.sh <local-out-dir> [label]
# Output: <label>-<shot>.png and .json (title, first page text, console errors) in <local-out-dir>.
set -euo pipefail
out="${1:?usage: shoot_edge.sh <local-out-dir> [label]}"
label="${2:-edge}"
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(git -C "$here" rev-parse --show-toplevel)"
host="root@100.91.190.107"   # ovh-files on the tailnet
vps() { MSYS_NO_PATHCONV=1 ssh -i "$HOME/.ssh/ovh" -o BatchMode=yes "$host" "$@"; }
box="$(vps "docker ps --filter label=com.docker.compose.service=devbox --format '{{.Names}}' | head -n 1")"
[ -n "$box" ] || { echo "no running devbox container on $host" >&2; exit 1; }
work="/home/kasm-user/edge-shoot/$label-$(date +%Y%m%dT%H%M%S)"
stage="$(mktemp -d)"
# Committed code and plan only (HEAD), never the working tree.
git -c core.autocrlf=false -C "$repo" archive --format=tar HEAD:modules/Probata/probata/deploy/portal shoot.mjs | tar -x -C "$stage"
git -c core.autocrlf=false -C "$repo" archive --format=tar HEAD:modules/Consignatio/docs/receipts/portal/tools edge-shots.json | tar -x -C "$stage"
sed -E "s/\"name\": \"/\"name\": \"$label-/" "$stage/edge-shots.json" > "$stage/plan.json"
tar -c -C "$stage" shoot.mjs plan.json | vps "docker exec -i -u 1000 $box sh -c 'mkdir -p $work && tar -x -C $work'"
status=0
vps "docker exec -u 1000 $box node $work/shoot.mjs $work/plan.json $work/out" || status=$?
mkdir -p "$out"
vps "docker exec -u 1000 $box tar -c -C $work/out ." | tar -x --force-local -C "$out"
for f in "$out"/"$label"-*.json; do
  python3 -c 'import json,sys; m=json.load(open(sys.argv[1])); print(m["shot"]["name"], "| title:", m.get("title"), "| console errors:", len(m.get("consoleErrors", [])), m.get("consoleErrors", [])[:3])' "$f"
done
exit "$status"
