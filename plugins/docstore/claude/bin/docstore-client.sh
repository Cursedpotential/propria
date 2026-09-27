#!/usr/bin/env bash
# Byline: Claude Code · Opus 5.5 · 2026-09-26
# Run the portable Docstore client with the plugin's own pinned requirements (fastmcp==3.2.4), in an
# isolated uv environment. The desktop's global Python has fastmcp 3.4.7 + mcp 2.0.0, which cannot import
# fastmcp's client, so `python3 client.py` fails there. Usage: bin/docstore-client.sh call docstore_health
here="$(cd "$(dirname "$0")/.." && pwd)"
exec uv run --no-project --quiet --with-requirements "$here/requirements.txt" python "$here/client.py" "$@"
