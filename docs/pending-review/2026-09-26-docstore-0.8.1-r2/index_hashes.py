"""Print every stored document's source_path, status and content_hash as JSON (read-only).

Byline: Claude Code · Opus 5.5 · 2026-09-26

Runs INSIDE the propria-docstore container, SELECT only. Used to prove that specific desktop
files (for example the 61 docs recovered after the 2026-09-20 git flatten) are indexed with
their current content: compare the printed content_hash with the file's sha256 (identical for
files without non-BMP characters, which the store folds to names).

    docker exec -i docstore-<uuid> sh -c "cd /app/scripts/docstore && python -" < index_hashes.py > hashes.json
"""
import asyncio
import json
import sys

sys.path.insert(0, "/app/scripts/docstore")
import sq  # noqa: E402
from upgrade import rows  # noqa: E402


async def main() -> None:
    db = await sq.connect("docs", "probata", "docs")
    try:
        found = rows(await db.query("SELECT source_path, status, content_hash, observed_at FROM document;"))
    finally:
        await db.close()
    print(json.dumps([{"source_path": r.get("source_path"), "status": r.get("status"),
                       "content_hash": r.get("content_hash"), "observed_at": str(r.get("observed_at"))}
                      for r in found]))


asyncio.run(main())
