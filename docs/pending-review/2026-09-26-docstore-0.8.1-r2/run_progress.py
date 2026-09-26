"""Read-only progress probe for a running Docstore index run (enrichment stage).

Byline: Claude Code · Opus 5.5 · 2026-09-26

Runs INSIDE the propria-docstore container. Counts enrichment records written since the run's start
time (argv[1], ISO UTC) and the documents/chunks currently active. SELECT only.

    docker exec -i docstore-<uuid> sh -c "cd /app/scripts/docstore && python - 2026-09-26T22:24:05Z" < run_progress.py
"""
import asyncio
import sys

sys.path.insert(0, "/app/scripts/docstore")
import sq  # noqa: E402
from upgrade import rows  # noqa: E402

SINCE = sys.argv[1] if len(sys.argv) > 1 else "2026-09-26T00:00:00Z"


async def main() -> None:
    db = await sq.connect("docs", "probata", "docs")
    try:
        enriched = rows(await db.query(
            "SELECT count() AS n FROM docstore_enrichment WHERE updated_at > <datetime>$since GROUP ALL;",
            {"since": SINCE}))
        methods = rows(await db.query(
            "SELECT method, count() AS n FROM docstore_enrichment WHERE updated_at > <datetime>$since GROUP BY method;",
            {"since": SINCE}))
        docs = rows(await db.query("SELECT status, count() AS n FROM document GROUP BY status;"))
        print({"enriched_since": (enriched[0]["n"] if enriched else 0), "methods": methods, "documents_by_status": docs})
    finally:
        await db.close()


asyncio.run(main())
