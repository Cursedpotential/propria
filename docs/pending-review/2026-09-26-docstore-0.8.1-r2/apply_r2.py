"""Docstore 0.8.1-r2: fast long-question search, named API failures, a published API, full-size syncs.

Byline: Claude Code · Opus 5.5 · 2026-09-26

Runs on ovh-files against the release build context (default /data/propria/releases/docstore-0.8.1,
which is in no git repository; this script and the before-copies in ./before/ are the record).
Edits four files IN PLACE after asserting every anchor occurs exactly once. Only the anchored
regions change; every other byte (mixed CRLF/LF endings included) is preserved. Each edited
file is compiled before it is written and backed up as <name>.bak-<utc>-r2. Idempotent: a file
that already carries the marker is reported and left alone. Dry run unless --apply.

    python3 apply_r2.py [release_dir] [--apply]

1. scripts/docstore/recall.py   - the any-term keyword fallback ran one BM25 query per term in
   sequence (16 terms: 7.7 s warm, over 30 s cold). Terms are de-duplicated and queried
   concurrently (8 in flight, 32 at most) on the one multiplexed connection, and the vector leg
   runs alongside the keyword leg.
2. plugins/docstore/control/server.py - the ctl-to-API budget was a fixed 30 s and every failure
   read "Docstore unavailable or invalid response". Budget DOCSTORE_API_TIMEOUT_S (default 55 s,
   under ContextForge's 60 s tool timeout); a timeout, HTTP status or unreachable API is named.
   Upstream bodies are still never echoed.
3. scripts/docstore/service.py - DOCSTORE_API_HOST (127.0.0.1 default, or 0.0.0.0) so the compose
   can publish the worker API on the canonical Docstore port 8072 for svc:docstore-api.
4. plugins/docstore/control/hosted.py - the MCP SDK refuses request bodies over 4 MiB (HTTP 413),
   so the complete five-root source manifest (about 12 MB) could never be planned or applied.
   The hosted ctl accepts DOCSTORE_MCP_MAX_BODY_BYTES (default 40 MiB, the worker API's own bound).
"""
from __future__ import annotations

import sys
import time
from pathlib import Path

MARKER = "0.8.1-r2"

RECALL_CONSTANTS_OLD = "RERANK_POOL = 25\n"
RECALL_CONSTANTS_NEW = (
    "RERANK_POOL = 25\n"
    "FALLBACK_CONCURRENCY = 8  # 0.8.1-r2: per-term BM25 queries in flight on the one connection\n"
    "MAX_FALLBACK_TERMS = 32  # 0.8.1-r2: bound for long (up to 2048-character) questions\n"
)

RECALL_LEGS_OLD = '''    db = await sq.connect("docs", "probata", "docs")
    embed_task = asyncio.create_task(asyncio.to_thread(embed, query))
    kw = _rows(await db.query(
        "SELECT id, text, search::score(1) AS score, (->chunk_of->document)[0] AS doc FROM chunk "
        "WHERE text @1@ $q" + extra + " ORDER BY score DESC LIMIT 80;", params))
    kw_mode = "all-terms"
    terms = params["q"].split()
    if len({str(c["doc"]) for c in kw if c.get("doc") is not None}) < 3 and len(terms) > 1:
        # The BM25 index ANDs every term; a single OR query scores ~10k chunks (4.4 s), so run each
        # term separately (fast, indexed) and sum scores per chunk instead.
        merged = {str(c["id"]): dict(c) for c in kw}
        for term in terms:
            for c in _rows(await db.query(
                    "SELECT id, text, search::score(1) AS score, (->chunk_of->document)[0] AS doc FROM chunk "
                    "WHERE text @1@ $t" + extra + " ORDER BY score DESC LIMIT 30;", dict(params, t=term))):
                key = str(c["id"])
                if key in merged:
                    merged[key]["score"] += c["score"]
                else:
                    merged[key] = dict(c)
        kw = list(merged.values())
        kw_mode = "any-term"
    params["v"] = await embed_task
    vec = _rows(await db.query(
        "SELECT id, text, vector::distance::knn() AS dist, (->chunk_of->document)[0] AS doc FROM chunk "
        "WHERE embedding <|80,200|> $v" + extra + " ORDER BY dist;", params))
    t_search = time.perf_counter()
'''

RECALL_LEGS_NEW = '''    db = await sq.connect("docs", "probata", "docs")
    embed_task = asyncio.create_task(asyncio.to_thread(embed, query))
    keyword_select = ("SELECT id, text, search::score(1) AS score, (->chunk_of->document)[0] AS doc FROM chunk "
                      "WHERE text @1@ ")

    # 0.8.1-r2 (Claude Code · Opus 5.5, 2026-09-26): the any-term fallback used to run one BM25 query per
    # term, one after another, repeated terms included (16 terms: 7.7 s warm and over 30 s cold, past the
    # ctl client's 30 s budget, which surfaced as "Docstore unavailable"). Terms are now de-duplicated and
    # queried concurrently on the one multiplexed connection (1.7 s measured for the same 16 terms), and
    # the vector leg (question embedding + KNN) runs alongside the keyword leg instead of after it.
    async def keyword_leg():
        kw = _rows(await db.query(keyword_select + "$q" + extra + " ORDER BY score DESC LIMIT 80;", params))
        terms = []
        for term in params["q"].split():
            if term.lower() not in {t.lower() for t in terms}:
                terms.append(term)
        terms = terms[:MAX_FALLBACK_TERMS]
        if len({str(c["doc"]) for c in kw if c.get("doc") is not None}) >= 3 or len(terms) < 2:
            return kw, "all-terms"
        # The BM25 index ANDs every term; a single OR query scores ~10k chunks (4.4 s), so query each
        # term separately (fast, indexed) and sum scores per chunk instead.
        gate = asyncio.Semaphore(FALLBACK_CONCURRENCY)

        async def one_term(term):
            async with gate:
                return _rows(await db.query(keyword_select + "$t" + extra + " ORDER BY score DESC LIMIT 30;",
                                            dict(params, t=term)))

        merged = {str(c["id"]): dict(c) for c in kw}
        for chunks in await asyncio.gather(*[one_term(term) for term in terms]):
            for c in chunks:
                key = str(c["id"])
                if key in merged:
                    merged[key]["score"] += c["score"]
                else:
                    merged[key] = dict(c)
        return list(merged.values()), "any-term"

    async def vector_leg():
        vector = await embed_task
        return _rows(await db.query(
            "SELECT id, text, vector::distance::knn() AS dist, (->chunk_of->document)[0] AS doc FROM chunk "
            "WHERE embedding <|80,200|> $v" + extra + " ORDER BY dist;", dict(params, v=vector)))

    legs = await asyncio.gather(keyword_leg(), vector_leg(), return_exceptions=True)
    failure = next((leg for leg in legs if isinstance(leg, BaseException)), None)
    if failure is not None:
        await db.close()
        raise failure
    (kw, kw_mode), vec = legs
    t_search = time.perf_counter()
'''

SERVER_REQUEST_OLD = '''    async def request(method: str, path: str, params=None, payload=None):
        headers = {"Authorization": f"Bearer {config.token}"} if config.token else {}
        try:
            async with httpx.AsyncClient(timeout=30, headers=headers, transport=transport,
                                         follow_redirects=False, trust_env=False) as client:
'''

SERVER_REQUEST_NEW = '''    # 0.8.1-r2 (Claude Code · Opus 5.5, 2026-09-26): a fixed 30 s budget turned a slow recall into a bare
    # "unavailable". The budget is DOCSTORE_API_TIMEOUT_S (default 55 s, under ContextForge's 60 s tool
    # timeout), and a timeout, HTTP status or unreachable API is named. Upstream bodies are never echoed.
    api_timeout = float(os.environ.get("DOCSTORE_API_TIMEOUT_S") or 55)
    if not 5 <= api_timeout <= 300:
        raise ValueError("DOCSTORE_API_TIMEOUT_S must be between 5 and 300 seconds")

    async def request(method: str, path: str, params=None, payload=None):
        headers = {"Authorization": f"Bearer {config.token}"} if config.token else {}
        try:
            async with httpx.AsyncClient(timeout=api_timeout, headers=headers, transport=transport,
                                         follow_redirects=False, trust_env=False) as client:
'''

SERVER_EXCEPT_OLD = '''        except (httpx.HTTPError, ValueError):
            raise ToolError("Docstore unavailable or invalid response; no filesystem fallback") from None

    async def get(path: str, params=None):
'''

SERVER_EXCEPT_NEW = '''        except httpx.TimeoutException:
            raise ToolError(f"Docstore unavailable: the API did not answer within {api_timeout:g} s; "
                            "no filesystem fallback") from None
        except httpx.HTTPStatusError as exc:
            raise ToolError(f"Docstore unavailable: the API answered HTTP {exc.response.status_code}; "
                            "no filesystem fallback") from None
        except httpx.TransportError:
            raise ToolError("Docstore unavailable: the API could not be reached; no filesystem fallback") from None
        except (httpx.HTTPError, ValueError):
            raise ToolError("Docstore unavailable or invalid response; no filesystem fallback") from None

    async def get(path: str, params=None):
'''

SERVICE_HOST_OLD = "        raise ValueError('Worker and control tokens are mandatory')\n"
SERVICE_HOST_NEW = (
    "        raise ValueError('Worker and control tokens are mandatory')\n"
    "    # 0.8.1-r2 (Claude Code · Opus 5.5, 2026-09-26): DOCSTORE_API_HOST=0.0.0.0 lets the compose publish the\n"
    "    # worker API on the canonical Docstore port 8072 behind svc:docstore-api (owner 2026-09-10: \"make sure\n"
    "    # there's an API exposed\"; 2026-09-12 port families). The default stays loopback.\n"
    "    api_host=os.environ.get('DOCSTORE_API_HOST','127.0.0.1')\n"
    "    if api_host not in {'127.0.0.1','0.0.0.0'}:\n"
    "        raise ValueError('DOCSTORE_API_HOST must be 127.0.0.1 or 0.0.0.0')\n"
)
SERVICE_BIND_OLD = "'api:app','--host','127.0.0.1','--port','8000']"
SERVICE_BIND_NEW = "'api:app','--host',api_host,'--port','8000']"

HOSTED_FUNCS_OLD = "def main():\n"
HOSTED_FUNCS_NEW = '''# 0.8.1-r2 (Claude Code · Opus 5.5, 2026-09-26): docstore_source_plan/apply carry the complete five-root
# manifest in one tool call (about 12 MB on 2026-09-26). The MCP SDK's Streamable HTTP transport refuses
# request bodies over 4 MiB with HTTP 413, so a client sync could never reach the source mirror. The hosted
# ctl accepts the worker API's own source-upload bound (40 MiB) instead.
def mcp_body_limit():
    limit=int(os.environ.get('DOCSTORE_MCP_MAX_BODY_BYTES',str(40*1024*1024)))
    if not 4*1024*1024<=limit<=64*1024*1024:
        raise ValueError('DOCSTORE_MCP_MAX_BODY_BYTES must be between 4 MiB and 64 MiB')
    return limit


def allow_source_manifests(limit):
    from mcp.server.streamable_http_manager import StreamableHTTPSessionManager
    original=StreamableHTTPSessionManager.__init__

    def __init__(self,*args,max_request_body_size=limit,**kwargs):
        original(self,*args,max_request_body_size=max_request_body_size,**kwargs)
    StreamableHTTPSessionManager.__init__=__init__


def main():
'''
HOSTED_RUN_OLD = "    server.run(**runtime)\n"
HOSTED_RUN_NEW = "    allow_source_manifests(mcp_body_limit())\n    server.run(**runtime)\n"

EDITS = {
    "scripts/docstore/recall.py": [(RECALL_CONSTANTS_OLD, RECALL_CONSTANTS_NEW), (RECALL_LEGS_OLD, RECALL_LEGS_NEW)],
    "plugins/docstore/control/server.py": [(SERVER_REQUEST_OLD, SERVER_REQUEST_NEW),
                                           (SERVER_EXCEPT_OLD, SERVER_EXCEPT_NEW)],
    "scripts/docstore/service.py": [(SERVICE_HOST_OLD, SERVICE_HOST_NEW), (SERVICE_BIND_OLD, SERVICE_BIND_NEW)],
    "plugins/docstore/control/hosted.py": [(HOSTED_FUNCS_OLD, HOSTED_FUNCS_NEW), (HOSTED_RUN_OLD, HOSTED_RUN_NEW)],
}


def patch(text: str, edits) -> str:
    newline = "\r\n" if "\r\n" in text else "\n"
    for old, new in edits:
        old_n, new_n = old.replace("\n", newline), new.replace("\n", newline)
        count = text.count(old_n)
        if count != 1:
            raise SystemExit(f"anchor found {count} times; refusing to edit: {old.splitlines()[0][:70]!r}")
        text = text.replace(old_n, new_n, 1)
    return text


def main() -> None:
    args = [a for a in sys.argv[1:] if a != "--apply"]
    apply = "--apply" in sys.argv[1:]
    root = Path(args[0] if args else "/data/propria/releases/docstore-0.8.1")
    stamp = time.strftime("%Y%m%dT%H%M%SZ", time.gmtime())
    planned = []
    for relative, edits in EDITS.items():
        target = root / relative
        raw = target.read_bytes()
        text = raw.decode("utf-8")
        if MARKER in text:
            print(f"already applied  {relative}")
            continue
        new_text = patch(text, edits)
        compile(new_text, str(target), "exec")
        planned.append((target, raw, new_text.encode("utf-8")))
        print(f"{'patching' if apply else 'would patch'}  {relative} (+{len(new_text.encode()) - len(raw)} bytes)")
    if not apply:
        return
    for target, raw, data in planned:
        target.with_name(f"{target.name}.bak-{stamp}-r2").write_bytes(raw)
        target.write_bytes(data)
    print(f"done; backups end in .bak-{stamp}-r2")


if __name__ == "__main__":
    main()
