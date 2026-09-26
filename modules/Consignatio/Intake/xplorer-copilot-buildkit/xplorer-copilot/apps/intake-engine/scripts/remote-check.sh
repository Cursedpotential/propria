#!/usr/bin/env bash
# Byline: Claude Code · Opus 5 · 2026-09-17
# Dev loop: ship the engine sources to ovh-files and run cargo there in a throwaway
# build container (nothing builds or runs on the owner's desktop). Usage:
#   apps/intake-engine/scripts/remote-check.sh [cargo args...]   (default: build --release)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
HOST=root@100.91.190.107
REMOTE=/data/probata/build/intake-engine/src
ARGS=${*:-build --release}
cd "$ROOT"
tar czf - --exclude=apps/intake-engine/target --exclude='*.log' \
  rust-toolchain.toml apps/src-tauri/src apps/src-tauri/Cargo.toml apps/intake-engine \
  |MSYS_NO_PATHCONV=1 ssh -i ~/.ssh/ovh "$HOST" "mkdir -p $REMOTE && tar xzf - -C $REMOTE"
MSYS_NO_PATHCONV=1 ssh -i ~/.ssh/ovh "$HOST" "docker run --rm --name intake-engine-build \
  --cpus 6 --memory 12g \
  -v $REMOTE:/src -v intake-engine-cargo:/usr/local/cargo/registry -v intake-engine-target:/target \
  -e CARGO_TARGET_DIR=/target -e CARGO_BUILD_JOBS=6 -w /src/apps/intake-engine \
  rust:1.91.1-bookworm sh -c 'cargo $ARGS 2>&1 | tail -n 400'"
