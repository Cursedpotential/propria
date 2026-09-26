"""Before/after recall check for Docstore 0.8.1-r2 (read-only).

Byline: Claude Code · Opus 5.5 · 2026-09-26

Runs INSIDE the propria-docstore container against the loopback worker API with the
container's own DOCSTORE_API_TOKEN (never printed). For each query it prints wall time,
the API's own search timing, the keyword mode and the top result paths, once with
rerank=false (deterministic ranking, comparable before/after) and once with rerank=true.

    docker exec -i docstore-<uuid> sh -c "cd /app/scripts/docstore && python -" < recall_compare.py
"""
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request

QUERIES = [
    ("tailscale service", "infra"),
    ("Docstore API deployment: which Coolify service or app serves the Docstore API, its port, "
     "and svc:docstore-api endpoint after the 0.8 release", "infra"),
    ("what did we decide about the embedding model for the docstore and which NIM model replaced the retired ones",
     "docs"),
    ("custody hash chain genesis", "probata"),
]


def recall(query: str, domain: str, rerank: bool) -> dict:
    params = urllib.parse.urlencode({"q": query, "domain": domain, "kind": "doc", "status": "all", "k": 8,
                                     "rerank": str(rerank).lower()})
    request = urllib.request.Request("http://127.0.0.1:8000/recall?" + params,
                                     headers={"Authorization": "Bearer " + os.environ.get("DOCSTORE_API_TOKEN", "")})
    started = time.perf_counter()
    try:
        body = json.loads(urllib.request.urlopen(request, timeout=180).read().decode())
    except urllib.error.HTTPError as exc:
        return {"http": exc.code, "seconds": round(time.perf_counter() - started, 1)}
    stats = body.get("stats") or {}
    return {"seconds": round(time.perf_counter() - started, 1), "search_ms": stats.get("search_ms"),
            "kw_mode": stats.get("kw_mode"), "kw_docs": stats.get("kw_docs"), "vec_docs": stats.get("vec_docs"),
            "reranked": stats.get("reranked"), "top": [r.get("path") for r in body.get("results", [])][:5]}


for query, domain in QUERIES:
    for rerank in (False, True):
        print(json.dumps({"query": query[:60], "domain": domain, "rerank": rerank, **recall(query, domain, rerank)}))
