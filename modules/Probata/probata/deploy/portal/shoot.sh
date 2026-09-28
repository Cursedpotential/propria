#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-26
# Screenshots of the Propria Homepage portal, taken by headless Chrome inside the Probata devbox on
# ovh-files. Nothing browser-related runs on the calling machine (owner rule 2026-09-24: no Chrome,
# Edge, headless browser or Playwright on the owner's desktop).
#
#   deploy/portal/shoot.sh live    <local-out-dir> [label]   # the deployed portal
#   deploy/portal/shoot.sh preview <local-out-dir> [label]   # this checkout's deploy/portal config
#
# live    : tailnet portal at 1600x1000 (viewport + full page), scrolled part-way (the button column
#           should stay in view), at 1366x768, with the quick-launch search open, and at phone
#           width; the public portal logged out (the Authentik login is the expected result); and
#           the public instance's own content through its tailnet-bound port 100.72.169.40:3012.
# preview : renders this checkout's shared/ + tailnet/ + public/ config with a throwaway Homepage
#           v1.13.2 process (the same image digest the Dockerfile pins, unpacked from ghcr.io into
#           ~/portal-work) listening on 127.0.0.1 inside the devbox, then stops it. /progress is
#           re-pointed at the progress board the way tailscale serve and Traefik do in production.
# shoot.mjs is taken from the committed HEAD (git archive), so only committed screenshot code runs.
# Output: <label>-*.png, <label>-*-full.png and <label>-*.json (layout numbers) in <local-out-dir>.
set -euo pipefail

if [ "${1:-}" = "--in-devbox" ]; then
  # ---------------------------------------------------------------- runs inside the devbox
  mode="$2"; label="$3"; work="$4"
  cd "$work"
  plan="$work/plan.json"
  board="http://100.72.169.40:3020"
  rewrite() {  # origin -> JSON rewrite rules for /progress on that origin (with and without port)
    local origin="$1" bare
    bare="$(printf '%s' "$origin" | sed -E 's#(https?://[^:/]+):[0-9]+#\1#')"
    printf '[{"prefix":"%s/progress","to":"%s"},{"prefix":"%s/progress","to":"%s"}]' \
      "$origin" "$board" "$bare" "$board"
  }
  shot() {  # name url width height fullPage [rewrite-json] [extra JSON fields, e.g. "scrollY":1400]
    printf '{"name":"%s","url":"%s","width":%s,"height":%s,"fullPage":%s,"settleMs":14000,"rewrite":%s%s}' \
      "$1" "$2" "$3" "$4" "$5" "${6:-[]}" "${7:+,$7}"
  }
  extras() {  # name-prefix url rewrite-json: part-way scrolled, laptop size, quick-launch search
    echo ","; shot "$1-scrolled" "$2" 1600 1000 false "$3" '"scrollY":1400'
    echo ","; shot "$1-1366x768" "$2" 1366 768 false "$3"
    echo ","; shot "$1-search" "$2" 1600 1000 false "$3" '"typeText":"cas"'
  }
  if [ "$mode" = live ]; then
    pub="http://100.72.169.40:3012"
    tail="https://homepage.tilapia-skilift.ts.net/"
    {
      echo "["
      shot "$label-tailnet" "$tail" 1600 1000 true; echo ","
      shot "$label-tailnet-phone" "$tail" 390 844 true
      extras "$label-tailnet" "$tail" "[]"; echo ","
      shot "$label-public-login" "https://homepage.int.mitechconsult.com/" 1600 1000 false; echo ","
      shot "$label-public-content" "$pub/" 1600 1000 true "$(rewrite "$pub")"
      echo "]"
    } > "$plan"
    status=0
    node "$work/shoot.mjs" "$plan" "$work/out" || status=$?
    exit "$status"
  else
    hp="$HOME/portal-work/homepage-v1.13.2"
    if [ ! -f "$hp/app/server.js" ]; then
      # linux/amd64 manifest of ghcr.io/gethomepage/homepage:v1.13.2
      # (index sha256:a0b71c8e757298d02560186bab9fbe3fc2d375c523a62cc1019177b37e48aa28, pinned in the Dockerfile)
      manifest="sha256:c881120b024d6a8e2f3c9664efc568984e4352e47df459d6b32e225374c71955"
      mkdir -p "$hp/rootfs"
      token="$(curl -fsS "https://ghcr.io/token?scope=repository:gethomepage/homepage:pull" | jq -r .token)"
      curl -fsS -H "Authorization: Bearer $token" \
        -H "Accept: application/vnd.oci.image.manifest.v1+json,application/vnd.docker.distribution.manifest.v2+json" \
        "https://ghcr.io/v2/gethomepage/homepage/manifests/$manifest" > "$hp/manifest.json"
      for layer in $(jq -r '.layers[].digest' "$hp/manifest.json"); do
        curl -fsSL -H "Authorization: Bearer $token" "https://ghcr.io/v2/gethomepage/homepage/blobs/$layer" |
          tar -xz -C "$hp/rootfs" app 2>/dev/null || true
      done
      mv "$hp/rootfs/app" "$hp/app"
    fi
    # One instance at a time: both processes would share the unpacked app's page cache (.next).
    port=3990
    if curl -s -o /dev/null "http://127.0.0.1:$port/"; then
      echo "port $port already in use in the devbox; stop that process first" >&2
      exit 1
    fi
    status=0
    for inst in tailnet public; do
      title="Propria Project Portal"; [ "$inst" = public ] && title="Propria Project Portal (Public)"
      cfg="$work/config-$inst"
      mkdir -p "$cfg"
      cp -R "$work/src/shared/." "$cfg/"
      cp -R "$work/src/$inst/." "$cfg/"
      cd "$hp/app"
      PORT="$port" HOSTNAME=127.0.0.1 HOMEPAGE_CONFIG_DIR="$cfg" \
        HOMEPAGE_ALLOWED_HOSTS="127.0.0.1:$port" HOMEPAGE_VAR_PORTAL_TITLE="$title" \
        HOMEPAGE_BUILDTIME="$(date +%s)" nohup node server.js > "$work/server-$inst.log" 2>&1 &
      pid=$!
      cd "$work"
      for _ in $(seq 1 60); do
        curl -fsS -o /dev/null "http://127.0.0.1:$port/api/healthcheck" 2>/dev/null && break
        sleep 1
      done
      # Render the index with this config now (the deployed containers do the same from their
      # healthcheck; see deploy/portal.yaml).
      curl -fsS -o /dev/null "http://127.0.0.1:$port/api/revalidate" || true
      origin="http://127.0.0.1:$port"
      {
        echo "["
        shot "$label-$inst" "$origin/" 1600 1000 true "$(rewrite "$origin")"
        if [ "$inst" = tailnet ]; then
          echo ","; shot "$label-$inst-phone" "$origin/" 390 844 true "$(rewrite "$origin")"
          extras "$label-$inst" "$origin/" "$(rewrite "$origin")"
        fi
        echo "]"
      } > "$plan"
      node "$work/shoot.mjs" "$plan" "$work/out" || status=$?
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    done
    exit "$status"
  fi
fi

# -------------------------------------------------------------------- runs on the calling machine
mode="${1:?usage: shoot.sh live|preview <local-out-dir> [label]}"
out="${2:?usage: shoot.sh live|preview <local-out-dir> [label]}"
label="${3:-$mode}"
case "$mode" in live|preview) ;; *) echo "mode must be live or preview" >&2; exit 2 ;; esac
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(git -C "$here" rev-parse --show-toplevel)"
rel="$(git -C "$here" rev-parse --show-prefix)"; rel="${rel%/}"
host="${PORTAL_SHOOT_HOST:-root@100.91.190.107}"   # ovh-files on the tailnet
key="${PORTAL_SHOOT_SSH_KEY:-$HOME/.ssh/ovh}"
vps() { MSYS_NO_PATHCONV=1 ssh -i "$key" -o BatchMode=yes "$host" "$@"; }

box="$(vps "docker ps --filter label=com.docker.compose.service=devbox --format '{{.Names}}' | head -n 1")"
[ -n "$box" ] || { echo "no running devbox container on $host" >&2; exit 1; }
work="/home/kasm-user/portal-shoot/$label-$(date +%Y%m%dT%H%M%S)"
echo "portal screenshots: $mode as $label in $box:$work"

# Committed screenshot code; the config under preview comes from this checkout's working tree.
# core.autocrlf=false: a Windows checkout would otherwise archive shoot.sh with CRLF line ends.
stage="$(mktemp -d)"
git -c core.autocrlf=false -C "$repo" archive --format=tar "HEAD:$rel" shoot.mjs shoot.sh | tar -x -C "$stage"
mkdir -p "$stage/src"
cp -R "$here/shared" "$here/tailnet" "$here/public" "$stage/src/"
tar -c -C "$stage" . | vps "docker exec -i -u 1000 $box sh -c 'mkdir -p $work && tar -x -C $work'"

status=0
vps "docker exec -u 1000 $box bash $work/shoot.sh --in-devbox $mode $label $work" || status=$?
mkdir -p "$out"
vps "docker exec -u 1000 $box tar -c -C $work/out ." | tar -x --force-local -C "$out"
ls -1 "$out"
exit "$status"
