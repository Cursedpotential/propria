#!/usr/bin/env bash
# Byline: Codex / GPT-6, 2026-09-20. Optional hosted-control diagnostic.
# Automatic hooks are disabled in hooks.json; this remains directly callable.
set -uo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PY="${DOCSTORE_PYTHON:-python3}"
exec "$PY" "$DIR/../client.py" call docstore_diagnostics
