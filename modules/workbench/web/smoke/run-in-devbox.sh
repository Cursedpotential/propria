#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-26
# Runs the Workbench browser journeys (smoke/matter-flow.smoke.test.mjs) inside the Probata devbox,
# the agents' Kasm sandbox on ovh-files (deploy/devbox.yaml). Nothing browser-related runs on the
# calling machine: this packs the committed web source with `git archive` and streams it over SSH
# (owner rule 2026-09-24: no Chrome, Edge, headless browser or Playwright on the owner's desktop).
# The devbox already has Node 22 and Google Chrome; the journeys run as its non-root user and use
# Chrome's real binary, not the desktop launcher, so Chrome keeps its own sandbox.
# Uncommitted edits are not tested.
#
# Usage, from any directory (Git Bash on the desktop is fine):
#   modules/workbench/web/smoke/run-in-devbox.sh [git-ref]    # default: HEAD
# Each run unpacks into ~/browser-journeys-<sha> in the devbox; the journeys' browser profiles
# stay under its to_be_deleted/ for owner-only cleanup.
set -euo pipefail

ref="${1:-HEAD}"
host="${BROWSER_SMOKE_HOST:-root@100.91.190.107}"   # ovh-files on the tailnet
key="${BROWSER_SMOKE_SSH_KEY:-$HOME/.ssh/ovh}"
repo="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
sha="$(git -C "$repo" rev-parse --short "$ref")"
work="browser-journeys-$sha"

vps() { MSYS_NO_PATHCONV=1 ssh -i "$key" -o BatchMode=yes "$host" "$@"; }

box="$(vps "docker ps --filter label=com.docker.compose.service=devbox --format '{{.Names}}' | head -n 1")"
if [ -z "$box" ]; then
  echo "no running devbox container on $host" >&2
  exit 1
fi
echo "browser journeys: $sha in $box on $host"

git -C "$repo" archive --format=tar "$ref" modules/workbench/web |
  vps "docker exec -i $box sh -c 'cd && mkdir -p $work && tar -x -C $work && cd $work/modules/workbench/web && npm ci --no-audit --no-fund --loglevel=error && SMOKE_BROWSER=/opt/google/chrome/chrome npm run smoke:matter-flow'"
