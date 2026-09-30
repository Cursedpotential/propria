#!/usr/bin/env bash
# Byline: Claude Code · Sonnet 5 · 2026-09-07
# STALE, DO NOT RUN (Claude Code · Sonnet 5 · 2026-09-27): deploy/docker/family-court-console/src/
# was restructured on 2026-09-26 (commit 3b545875) to build mcp-app inside the Docker image
# (Dockerfile.cloud stage 1: npm ci + node build.mjs), because a synced dist/ built on the host is
# not committed and Coolify builds from a clean clone -- that mismatch was the exact reason this
# app never deployed successfully. Running this script quarantines that committed source tree
# (src/, widgets/, tests/, build.mjs, tsconfig.json) and replaces Dockerfile.cloud with the OLD
# single-stage version that COPYs a dist/ nobody commits, reintroducing the original bug. Verified
# live: running it on 2026-09-27 reverted the fix; `git restore deploy/docker/family-court-console/`
# undid the damage before anything was pushed. If mcp-app/src changes again, update
# deploy/docker/family-court-console/src/{mcp-app/src,widgets,tests,build.mjs,tsconfig.json}
# directly (or write a new sync step that copies THOSE, never dist/), not this script.
#
# Copies the family-court-toolkit MCP console's runtime subset out of the
# desktop plugin checkout — which lives OUTSIDE this repo, at
# E:/AI_Workspace/plugins/plugins/family-court-toolkit/ — into
# deploy/docker/family-court-console/src/, which IS committed to this repo so
# Coolify's git-based "Docker Compose" build (deploy/family-court-console.yaml)
# can see it. The plugin root itself is never reachable as a Coolify build
# source.
#
# This script only COPIES; it does not build. Before running it, build the
# plugin's mcp-app so dist/ is fresh:
#   cd E:/AI_Workspace/plugins/plugins/family-court-toolkit/mcp-app && node build.mjs
#
# Owner rulings 2026-09-07 16:16-17:12: the console runs in the cloud as its
# own Coolify app, federated by ContextForge — no client downloads anything.
# See deploy/docker/family-court-console/README.md and
# deploy/family-court-console.yaml.
#
# This script never deletes (owner hard rule): an existing src/ is quarantined
# under deploy/docker/family-court-console/_stale/ before a fresh copy lands.
#
# Usage: scripts/sync_family_court_console.sh [PLUGIN_ROOT]
#   PLUGIN_ROOT defaults to E:/AI_Workspace/plugins/plugins/family-court-toolkit

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_ROOT="${1:-E:/AI_Workspace/plugins/plugins/family-court-toolkit}"
DEST_DIR="$REPO_ROOT/deploy/docker/family-court-console/src"
STALE_DIR="$REPO_ROOT/deploy/docker/family-court-console/_stale"

if [ ! -d "$PLUGIN_ROOT" ]; then
  echo "sync_family_court_console.sh: plugin root not found: $PLUGIN_ROOT" >&2
  exit 1
fi
if [ ! -d "$PLUGIN_ROOT/mcp-app/dist" ]; then
  echo "sync_family_court_console.sh: $PLUGIN_ROOT/mcp-app/dist is missing — run 'node build.mjs' in the plugin's mcp-app/ first" >&2
  exit 1
fi
if [ ! -f "$PLUGIN_ROOT/mcp-app/Dockerfile.cloud" ]; then
  echo "sync_family_court_console.sh: $PLUGIN_ROOT/mcp-app/Dockerfile.cloud is missing" >&2
  exit 1
fi

if [ -d "$DEST_DIR" ]; then
  mkdir -p "$STALE_DIR"
  quarantine="$STALE_DIR/src-$(date +%Y%m%d-%H%M%S)"
  echo "sync_family_court_console.sh: quarantining existing $DEST_DIR -> $quarantine"
  mv "$DEST_DIR" "$quarantine"
fi

mkdir -p "$DEST_DIR/mcp-app"

echo "sync_family_court_console.sh: copying mcp-app/dist"
cp -R "$PLUGIN_ROOT/mcp-app/dist" "$DEST_DIR/mcp-app/dist"

echo "sync_family_court_console.sh: copying mcp-app/package.json + package-lock.json"
cp "$PLUGIN_ROOT/mcp-app/package.json" "$DEST_DIR/mcp-app/package.json"
cp "$PLUGIN_ROOT/mcp-app/package-lock.json" "$DEST_DIR/mcp-app/package-lock.json"

echo "sync_family_court_console.sh: copying content/"
cp -R "$PLUGIN_ROOT/content" "$DEST_DIR/content"

echo "sync_family_court_console.sh: copying skills/"
cp -R "$PLUGIN_ROOT/skills" "$DEST_DIR/skills"

echo "sync_family_court_console.sh: copying Dockerfile.cloud + .dockerignore"
cp "$PLUGIN_ROOT/mcp-app/Dockerfile.cloud" "$DEST_DIR/Dockerfile.cloud"
if [ -f "$PLUGIN_ROOT/.dockerignore" ]; then
  cp "$PLUGIN_ROOT/.dockerignore" "$DEST_DIR/.dockerignore"
fi

echo "sync_family_court_console.sh: done. Review deploy/docker/family-court-console/src/ and commit it."
du -sh "$DEST_DIR" 2>/dev/null || true
