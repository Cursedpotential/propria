"""Docstore 0.8.1-r4: enrichment retries a mis-shaped reply and an overloaded provider once.

Byline: Claude Code · Opus 5.5 · 2026-09-26

Cause of the two ADR enrichment failures (docs/adr/generated/0016.md, 0085.md; ValueError), found live
2026-09-26:
  - nvidia/nemotron-3.5-lightning-30b-a3b sometimes returns `statements` as one string instead of an array.
    knowledge.extract() retried only unparseable or truncated JSON; a parseable reply of the wrong shape
    failed the document on the first try ("Invalid semantic response lists"). 0085 passed on a lucky later
    run; 0016 kept failing.
  - An overloaded provider answered 429/5xx (88 HTTPStatusError failures in run 6f397d56, 23:10 UTC) and
    `raise_for_status()` failed the document without the retry a TransportError already gets.
Fix: shape validation moves inside the retry loop, and 429/500/502/503/504 get one retry after 5 s.
A document still fails, visibly, when both attempts fail. A release test covers the shape retry.

Same mechanics as apply_r2.py / apply_r3.py: anchored, byte-preserving, compiled before writing,
backups <name>.bak-<utc>-r4, idempotent (marker 0.8.1-r4), dry run unless --apply.

    python3 apply_r4.py [release_dir] [--apply]
"""
from __future__ import annotations

import sys
import time
from pathlib import Path

MARKER = "0.8.1-r4"

EXTRACT_OLD = """                response.raise_for_status()
                choice=response.json()['choices'][0]
                try:
                    value=json.loads(choice['message']['content'])
                    if not isinstance(value,dict) or choice.get('finish_reason')=='length':raise ValueError('Truncated semantic response')
                    break
                except (TypeError,ValueError):
                    if attempt:raise ValueError('Provider returned invalid or truncated semantic JSON twice') from None
"""
EXTRACT_NEW = """                try:
                    response.raise_for_status()
                except httpx.HTTPStatusError as exc:
                    # 0.8.1-r4 (Claude Code · Opus 5.5, 2026-09-26): an overloaded provider (429/5xx; 88 failures in
                    # one run on 2026-09-26) gets one delayed retry, like a transport error already does.
                    if attempt or exc.response.status_code not in {429,500,502,503,504}:raise
                    await asyncio.sleep(5)
                    continue
                choice=response.json()['choices'][0]
                try:
                    value=json.loads(choice['message']['content'])
                    if not isinstance(value,dict) or choice.get('finish_reason')=='length':raise ValueError('Truncated semantic response')
                    # 0.8.1-r4: a parseable reply of the wrong shape (seen 2026-09-26: `statements` as one string for
                    # ADR 0016 and 0085) is retried like invalid JSON instead of failing the document outright.
                    if (not isinstance(value.get('summary'),str) or not isinstance(value.get('classification'),str)
                            or not isinstance(value.get('entities',[]),list) or not isinstance(value.get('statements',[]),list)):
                        raise ValueError('Semantic response has the wrong shape')
                    break
                except (TypeError,ValueError):
                    if attempt:raise ValueError('Provider returned invalid, truncated or mis-shaped semantic JSON twice') from None
"""

TEST_ANCHOR = """async def test_enrichment_bounds_parallel_provider_work(database,monkeypatch):
"""
TEST_NEW = """async def test_semantic_retry_reshapes_misshaped_reply(monkeypatch):
    # 0.8.1-r4 (Claude Code · Opus 5.5, 2026-09-26): statements as one string is retried, not a document failure.
    for key,value in {'DOCSTORE_LLM_BASE_URL':'https://provider.invalid/v1','DOCSTORE_LLM_MODEL':'test','DOCSTORE_LLM_API_KEY':'synthetic'}.items():
        monkeypatch.setenv(key,value)
    requests=[]
    class Client:
        def __init__(self,**kwargs):pass
        async def __aenter__(self):return self
        async def __aexit__(self,*args):pass
        async def post(self,url,**kwargs):
            requests.append(kwargs['json'])
            statements='one; two' if len(requests)==1 else ['one','two']
            content=json.dumps({'summary':'s','classification':'reference','entities':[],'statements':statements})
            return SimpleNamespace(raise_for_status=lambda:None,json=lambda:{'choices':[{'finish_reason':'stop','message':{'content':content}}]})
    monkeypatch.setattr(knowledge.httpx,'AsyncClient',Client)
    result=await knowledge.extract('short body')
    assert len(requests)==2 and result['statements']==['one','two']


async def test_enrichment_bounds_parallel_provider_work(database,monkeypatch):
"""

EDITS = {
    "scripts/docstore/knowledge.py": [(EXTRACT_OLD, EXTRACT_NEW)],
    "tests/test_release.py": [(TEST_ANCHOR, TEST_NEW)],
}


def anchored(text: str, old: str, new: str) -> str:
    for newline in ("\r\n", "\n"):
        old_n = old.replace("\n", newline)
        count = text.count(old_n)
        if count == 1:
            return text.replace(old_n, new.replace("\n", newline), 1)
        if count > 1:
            break
    raise SystemExit(f"anchor not found exactly once; refusing to edit: {old.splitlines()[0][:80]!r}")


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
        new_text = text
        for old, new in edits:
            new_text = anchored(new_text, old, new)
        compile(new_text, str(target), "exec")
        planned.append((target, raw, new_text.encode("utf-8")))
        print(f"{'patching' if apply else 'would patch'}  {relative} (+{len(new_text.encode()) - len(raw)} bytes)")
    if not apply:
        return
    for target, raw, data in planned:
        target.with_name(f"{target.name}.bak-{stamp}-r4").write_bytes(raw)
        target.write_bytes(data)
    print(f"done; backups end in .bak-{stamp}-r4")


if __name__ == "__main__":
    main()
