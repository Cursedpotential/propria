#!/usr/bin/env bash
# Byline: Claude Code · Opus 5 · 2026-09-17
# Build the hosted Intake web UI on ovh-files (nothing builds on the owner's desktop).
# Ships tracked + untracked (non-ignored) files except the Rust/marketplace apps, runs
# pnpm install + vite build in a throwaway node container; output stays on the host at
# /data/probata/build/intake-ui/src/dist (vite empties it on each build).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
HOST=root@100.91.190.107
REMOTE=/data/probata/build/intake-ui/src
cd "$ROOT"
git ls-files -z --cached --others --exclude-standard -- . ':!apps/src-tauri' ':!apps/intake-engine' \
  | tar czf - --null -T - \
  | MSYS_NO_PATHCONV=1 ssh -i ~/.ssh/ovh "$HOST" "mkdir -p $REMOTE && tar xzf - -C $REMOTE"
MSYS_NO_PATHCONV=1 ssh -i ~/.ssh/ovh "$HOST" "docker run --rm --name intake-ui-build --cpus 4 --memory 8g \
  -v $REMOTE:/src -v intake-ui-pnpm-store:/pnpm-store -w /src \
  -e VITE_API_MODE=http -e VITE_API_URL=../storage -e VITE_INTAKE_MODE=1 \
  -e VITE_INTAKE_CHAT_MODEL=portkey:nemotron-3-super -e CI=true -e HUSKY=0 \
  node:22-bookworm sh -c 'corepack enable && pnpm config set store-dir /pnpm-store && \
    pnpm install --frozen-lockfile --ignore-scripts 2>&1 | tail -n 15 && \
    npx vite build --base ./ --emptyOutDir 2>&1 | tail -n 25'"
