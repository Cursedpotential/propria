"""Read-only timing probe for the Docstore recall keyword leg.

Byline: Claude Code · Opus 5.5 · 2026-09-26

Runs INSIDE the propria-docstore 0.8.1 container (it imports /app/scripts/docstore). It only
SELECTs: it times the per-term BM25 fallback that recall.py runs when the all-terms query
finds fewer than three documents, sequentially and concurrently, so the 0.8.1-r2 fix is
chosen from measurements rather than guesses.

    docker exec -i docstore-<uuid> sh -c "cd /app/scripts/docstore && python -" < recall_timing.py
"""
import asyncio
import sys
import time

sys.path.insert(0, "/app/scripts/docstore")
import recall as R  # noqa: E402
import sq  # noqa: E402

QUERY = ("Docstore API deployment: which Coolify service or app serves the Docstore API, its port, "
         "and svc:docstore-api endpoint after the 0.8 release")
SELECT = ("SELECT id, text, search::score(1) AS score, (->chunk_of->document)[0] AS doc FROM chunk "
          "WHERE text @1@ $t{extra} ORDER BY score DESC LIMIT 30;")


async def main() -> None:
    started = time.perf_counter()
    db = await sq.connect("docs", "probata", "docs")
    print("connect", round(time.perf_counter() - started, 2))
    terms = R.keyword_terms(QUERY).split()
    print("terms", len(terms), terms)
    with_domain = SELECT.format(extra=" AND $dom INSIDE domains")
    without_domain = SELECT.format(extra="")

    total = time.perf_counter()
    for term in terms:
        started = time.perf_counter()
        rows = R._rows(await db.query(with_domain, {"t": term, "dom": "infra"}))
        print(f"  sequential {term!r:24} {time.perf_counter() - started:6.2f}s rows={len(rows)}")
    print("sequential total (domain filter)", round(time.perf_counter() - total, 2))

    total = time.perf_counter()
    for term in terms:
        await db.query(without_domain, {"t": term})
    print("sequential total (no domain filter)", round(time.perf_counter() - total, 2))

    total = time.perf_counter()
    await asyncio.gather(*[db.query(with_domain, {"t": term, "dom": "infra"}) for term in terms])
    print("concurrent, one connection", round(time.perf_counter() - total, 2))
    await db.close()

    async def own_connection(term: str):
        connection = await sq.connect("docs", "probata", "docs")
        try:
            return await connection.query(with_domain, {"t": term, "dom": "infra"})
        finally:
            await connection.close()

    total = time.perf_counter()
    await asyncio.gather(*[own_connection(term) for term in terms])
    print("concurrent, separate connections", round(time.perf_counter() - total, 2))


asyncio.run(main())
