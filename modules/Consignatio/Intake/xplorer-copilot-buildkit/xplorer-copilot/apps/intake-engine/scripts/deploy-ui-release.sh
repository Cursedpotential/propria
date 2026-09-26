#!/usr/bin/env bash
# Byline: Claude Code · Opus 5 · 2026-09-17
# Copy the UI built on ovh-files (remote-ui-build.sh) into a new progress-board release on ovh-app
# and make it current. Previous current.json is kept as current.json.bak-<release> for rollback.
set -euo pipefail
REL=${1:-$(date -u +%Y-%m-%dT%H-%M-%S-000Z)}
SHA=$(git -C "$(dirname "$0")" rev-parse --short HEAD)
MSYS_NO_PATHCONV=1 ssh -i ~/.ssh/ovh root@100.91.190.107 "cd /data/probata/build/intake-ui/src/apps/client/dist && tar czf - ." \
 | MSYS_NO_PATHCONV=1 ssh -i ~/.ssh/ovh root@100.72.169.40 "set -e; B=/data/dashboards/progress-board/intake-build; \
   PREV=\$(python3 -c 'import json;print(json.load(open(\"'\$B'/current.json\"))[\"release\"])'); \
   mkdir -p \$B/releases/$REL/xplorer && tar xzf - -C \$B/releases/$REL/xplorer; \
   cp -a \$B/releases/\$PREV/metadata \$B/releases/$REL/metadata; cp \$B/current.json \$B/current.json.bak-$REL; \
   printf '{\n  \"release\": \"$REL\",\n  \"built_at\": \"$REL\",\n  \"source_fingerprint\": \"Intake-desktop feat/hosted-intake-engine $SHA\",\n  \"note\": \"Hosted Intake engine v1 UI. Rollback: copy current.json.bak-$REL over current.json.\"\n}\n' > \$B/current.json; echo \"current=$REL prev=\$PREV\""
