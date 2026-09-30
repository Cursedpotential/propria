#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-27
# Watch the Workbench after a deploy (its last two failures killed it ~10 minutes after start).
# Read-only. Every INTERVAL seconds for DURATION seconds: the container's start time, restart count
# and health on ovh-app, the tailnet door https://workbench.tilapia-skilift.ts.net/ (needs a tailnet
# client) and the public door https://workbench.int.mitechconsult.com/ logged out (must land on the
# Authentik flow).   watch_workbench.sh [DURATION=630] [INTERVAL=30]
set -u
dur="${1:-630}"; step="${2:-30}"; end=$(( $(date +%s) + dur )); fails=0
while [ "$(date +%s)" -lt "$end" ]; do
  state=$(MSYS_NO_PATHCONV=1 ssh -i "$HOME/.ssh/ovh" -o BatchMode=yes -o ConnectTimeout=10 root@100.72.169.40 \
    "c=\$(docker ps -a --format '{{.Names}}' | grep '^workbench-' | head -1); docker inspect \$c --format '{{.Name}} started={{.State.StartedAt}} status={{.State.Status}} restarts={{.RestartCount}} health={{.State.Health.Status}}'" 2>&1)
  tail=$(curl -sS -o /dev/null -m 15 -w '%{http_code}' https://workbench.tilapia-skilift.ts.net/ 2>&1)
  pub=$(curl -sS -L -o /dev/null -m 20 -w '%{http_code} %{url_effective}' https://workbench.int.mitechconsult.com/ 2>&1 | cut -d'?' -f1)
  case "$state $tail $pub" in *healthy*" 200 200 https://auth.int.mitechconsult.com/if/flow/"*) v=OK;; *) v=FAIL; fails=$((fails+1));; esac
  echo "$(date -u +%H:%M:%SZ) $v | $state | tailnet=$tail | public=$pub"
  sleep "$step"
done
echo "WATCH DONE fails=$fails"
