#!/usr/bin/env python3
"""Recheck NIM embedding and reranking models with max-length inputs and a generous timeout.

Embed/rerank copy of nim_streaming_recheck.py (which excludes these models from its chat list).

> _Byline: Claude Code · Opus 5 · 2026-09-10_

Per model, three requests:
  1. short   - 4 texts (embed) / 4 passages (rerank): dimension, batching, semantic sanity.
  2. over    - one over-long input with truncate=NONE: the 4xx names the real token ceiling.
  3. long    - a batch of long inputs with truncate=END: the model working AT its ceiling.

Limits observed live 2026-09-10: nemotron-3-embed-1b 4096 tokens + 65536 chars per input;
llama-nemotron-rerank-vl-1b-v2 10240 tokens per query+passage.
"""

from __future__ import annotations

import asyncio
import argparse
import json
import math
import os
import re
import time
from datetime import datetime, timezone
from pathlib import Path

import httpx

BASE = os.environ.get("NVIDIA_NIM_API_BASE", "https://integrate.api.nvidia.com/v1").rstrip("/")
# Rerankers are not in /models; each one lives on its own retrieval endpoint (dots -> underscores).
RERANK_BASE = os.environ.get("NVIDIA_NIM_RETRIEVAL_BASE", "https://ai.api.nvidia.com/v1/retrieval").rstrip("/")
OUT_DIR = Path(__file__).resolve().parent
EMBED_MARKERS = ("embed", "bge", "e5-", "minilm", "arctic", "gte", "nomic", "clip", "siglip")
# Retired rerankers stay listed so an end-of-life (410) shows up instead of silently vanishing.
KNOWN_RERANKERS = (
    "nvidia/llama-nemotron-rerank-vl-1b-v2",
    "nvidia/llama-nemotron-rerank-1b-v2",
    "nvidia/llama-3.2-nv-rerankqa-1b-v2",
    "nvidia/llama-3.2-nemoretriever-500m-rerank-v2",
    "nvidia/nv-rerankqa-mistral-4b-v3",
    "nvidia/rerank-qa-mistral-4b",
)
# Full sentences: bare 3-word phrases gave noisy similarities. [0]~[1] is the paraphrase pair.
SHORT_TEXTS = [
    "A dog chased the ball across the park.",
    "The puppy ran after a toy in the garden.",
    "Estimated quarterly tax payments are due in April.",
    "Photosynthesis converts light into chemical energy.",
]
RERANK_QUERY = "When is the quarterly tax filing deadline?"
RERANK_PASSAGES = [
    "The quick brown fox jumps over the lazy dog.",
    "Estimated quarterly tax payments are due in April, June, September and January.",
    "Photosynthesis converts light energy into chemical energy in plants.",
    "The school pickup schedule changes on the first Monday of each month.",
]
RERANK_EXPECTED_TOP = 1


def api_key() -> str:
    value = os.environ.get("NVIDIA_API_KEY") or os.environ.get("NVIDIA_NIM_API_KEY")
    if not value:
        raise SystemExit("NVIDIA_API_KEY not set")
    return value


def is_embed_candidate(model: str) -> bool:
    name = model.lower()
    return "rerank" not in name and any(marker in name for marker in EMBED_MARKERS)


def long_text(chars: int, seed: int = 0) -> str:
    """Deterministic, non-repeating filler so tokenizers can't collapse it."""
    parts, size, i = [], 0, 0
    while size < chars:
        sentence = f"Exhibit {seed}-{i} logs a message on day {i % 365} about pickup at {1 + i % 12}pm and school attendance."
        parts.append(sentence)
        size += len(sentence) + 1
        i += 1
    return " ".join(parts)[:chars]


def cosine(a: list[float], b: list[float]) -> float:
    dot = sum(x * y for x, y in zip(a, b))
    return dot / ((math.sqrt(sum(x * x for x in a)) * math.sqrt(sum(y * y for y in b))) or 1.0)


async def post(client: httpx.AsyncClient, url: str, body: dict) -> dict:
    started = time.monotonic()
    try:
        response = await client.post(
            url,
            headers={"Authorization": f"Bearer {api_key()}", "Content-Type": "application/json", "Accept": "application/json"},
            json=body,
        )
    except httpx.TimeoutException as exc:
        return {"ok": False, "classification": "timeout", "detail": repr(exc), "elapsed_s": round(time.monotonic() - started, 2)}
    except Exception as exc:
        return {"ok": False, "classification": "connection_error", "detail": repr(exc), "elapsed_s": round(time.monotonic() - started, 2)}
    out = {"status": response.status_code, "elapsed_s": round(time.monotonic() - started, 2)}
    try:
        data = response.json()
    except ValueError:
        data = None
    if response.status_code != 200 or not isinstance(data, dict):
        out.update(ok=False, classification="http_error", detail=response.text[:1000])
    else:
        out.update(ok=True, data=data)
    return out


def summarise(response: dict, sent_chars: int) -> dict:
    """Strip vectors/rankings; keep status, token usage and any ceiling named in the error."""
    row = {key: response.get(key) for key in ("ok", "status", "classification", "elapsed_s", "detail") if response.get(key) is not None}
    row["sent_chars"] = sent_chars
    usage = (response.get("data") or {}).get("usage") or {}
    if usage.get("prompt_tokens") is not None:
        row["prompt_tokens"] = usage["prompt_tokens"]
    detail = str(response.get("detail", ""))
    match = re.search(r"maximum\D{0,30}?(\d+)", detail)
    if match:
        row["char_limit" if "character" in detail.lower() else "token_limit"] = int(match.group(1))
    return row


def unavailable(result: dict, response: dict) -> dict:
    status = response.get("status")
    result.update(
        ok=False,
        classification="unavailable" if status in (404, 410) else response.get("classification", "short_failed"),
        status=status,
        detail=str(response.get("detail", ""))[:1000],
        elapsed_s=response.get("elapsed_s"),
    )
    return result


async def probe_embed(client: httpx.AsyncClient, model: str, args: argparse.Namespace, listed: bool) -> dict:
    url = f"{BASE}/embeddings"
    result = {"model": model, "kind": "embed", "listed": listed}
    extra: dict = {}
    short = await post(client, url, {"model": model, "input": SHORT_TEXTS, "encoding_format": "float"})
    if not short["ok"] and short.get("status") not in (404, 410):
        # Asymmetric embedders (embedqa, vl) reject calls without input_type.
        retry = await post(client, url, {"model": model, "input": SHORT_TEXTS, "encoding_format": "float", "input_type": "passage"})
        if retry["ok"]:
            short, extra = retry, {"input_type": "passage"}
    if not short["ok"]:
        return unavailable(result, short)
    vectors = [row["embedding"] for row in sorted(short["data"]["data"], key=lambda row: row.get("index", 0))]
    scores = {f"0~{i}": round(cosine(vectors[0], vectors[i]), 3) for i in range(1, len(vectors))}
    result.update(
        mode="asymmetric (input_type required)" if extra else "symmetric",
        dim=len(vectors[0]),
        short_count=len(vectors),
        short_s=short["elapsed_s"],
        sanity_scores=scores,
        # The paraphrase pair must beat both unrelated sentences.
        sanity_ok=len(vectors) == len(SHORT_TEXTS) and scores["0~1"] > max(scores["0~2"], scores["0~3"]),
    )

    over = await post(client, url, {"model": model, "input": [long_text(args.long_chars)], "encoding_format": "float", "truncate": "NONE", **extra})
    result["untruncated"] = summarise(over, args.long_chars)

    batch_input = [long_text(args.long_chars, seed) for seed in range(args.batch)]
    long = await post(client, url, {"model": model, "input": batch_input, "encoding_format": "float", "truncate": "END", **extra})
    result["long_batch"] = summarise(long, args.long_chars * args.batch)
    if long["ok"]:
        result["long_batch"]["vectors"] = len(long["data"].get("data", []))
        result["long_batch"]["complete"] = result["long_batch"]["vectors"] == args.batch
        # truncate=END can cut below the ceiling the API error names (vl-1b-v2: 8192 named, 4096 kept).
        if result["long_batch"].get("prompt_tokens") is not None:
            result["long_batch"]["tokens_per_input"] = result["long_batch"]["prompt_tokens"] // args.batch
    result["ok"] = bool(long["ok"] and result["long_batch"].get("complete"))
    result["classification"] = "long_batch_ok" if result["ok"] else "short_only"
    return result


async def probe_rerank(client: httpx.AsyncClient, model: str, args: argparse.Namespace) -> dict:
    url = f"{RERANK_BASE}/{model.replace('.', '_')}/reranking"
    result = {"model": model, "kind": "rerank", "listed": False}
    query = {"text": RERANK_QUERY}
    short = await post(client, url, {"model": model, "query": query, "passages": [{"text": p} for p in RERANK_PASSAGES], "truncate": "END"})
    if not short["ok"]:
        return unavailable(result, short)
    rankings = short["data"].get("rankings") or []
    result.update(
        short_count=len(rankings),
        short_s=short["elapsed_s"],
        sanity_ok=bool(rankings) and rankings[0].get("index") == RERANK_EXPECTED_TOP,
    )

    over = await post(client, url, {"model": model, "query": query, "passages": [{"text": long_text(args.rerank_long_chars)}], "truncate": "NONE"})
    result["untruncated"] = summarise(over, args.rerank_long_chars)

    # Long filler passages with the one relevant passage in the middle: truncation must not break ranking.
    passages = [{"text": long_text(args.rerank_long_chars, seed)} for seed in range(args.passages - 1)]
    needle = len(passages) // 2
    passages.insert(needle, {"text": RERANK_PASSAGES[RERANK_EXPECTED_TOP]})
    long = await post(client, url, {"model": model, "query": query, "passages": passages, "truncate": "END"})
    result["long_batch"] = summarise(long, args.rerank_long_chars * (args.passages - 1))
    if long["ok"]:
        long_rankings = long["data"].get("rankings") or []
        result["long_batch"]["rankings"] = len(long_rankings)
        result["long_batch"]["complete"] = len(long_rankings) == len(passages)
        result["long_batch"]["needle_top"] = bool(long_rankings) and long_rankings[0].get("index") == needle
    result["ok"] = bool(long["ok"] and result["long_batch"].get("complete"))
    result["classification"] = "long_batch_ok" if result["ok"] else "short_only"
    return result


def ceiling(row: dict) -> str:
    over = row.get("untruncated") or {}
    if over.get("token_limit"):
        return f"{over['token_limit']} tok"
    if over.get("char_limit"):
        return f"{over['char_limit']} chars"
    if over.get("ok"):
        return f">= {over.get('prompt_tokens', '?')} tok"
    return "-"


def render(run: dict) -> str:
    results = run["results"]
    working = [r for r in results if r.get("ok")]
    short_only = [r for r in results if r.get("classification") == "short_only"]
    unavailable_rows = [r for r in results if r.get("status") in (404, 410)]
    other = [r for r in results if r not in working and r not in short_only and r not in unavailable_rows]
    lines = [
        f"# NVIDIA NIM embed + rerank liveness — {run['timestamp']}", "",
        "> _Byline: Claude Code · Opus 5 · embed/rerank max-length probe · 2026-09-10_", "",
        f"- Catalog: {run['catalog_count']} models; embedders probed: {sum(r['kind'] == 'embed' for r in results)}; rerankers probed: {sum(r['kind'] == 'rerank' for r in results)} (not in /models)",
        f"- Method: per-request timeout {run['timeout_s']}s; over-long input with truncate=NONE to read the ceiling; then truncate=END batch "
        f"({run['batch']} × {run['long_chars']} chars for embed, {run['passages']} passages × {run['rerank_long_chars']} chars for rerank).",
        f"- Working at ceiling: {len(working)}; short-only: {len(short_only)}; unavailable (404/410): {len(unavailable_rows)}; other failures: {len(other)}", "",
    ]
    for kind, title in (("embed", "Embedders"), ("rerank", "Rerankers")):
        rows = [r for r in working + short_only if r["kind"] == kind]
        lines += [f"## {title} that answered", ""]
        if kind == "embed":
            lines += ["| Model | Dim | Mode | Sanity (cos 0~1 / 0~2 / 0~3) | Ceiling | Long batch tokens | Tokens/input | Long batch | Short | In /models |", "|---|---:|---|---|---|---:|---:|---|---:|---|"]
        else:
            lines += ["| Model | Sanity | Ceiling | Long batch tokens | Needle top | Long batch | Short |", "|---|---|---|---:|---|---|---:|"]
        for r in sorted(rows, key=lambda row: row["model"]):
            batch = r.get("long_batch") or {}
            batch_cell = f"{batch.get('elapsed_s')}s ok" if r.get("ok") else f"FAIL {batch.get('status', batch.get('classification', '-'))}"
            if kind == "embed":
                scores = r.get("sanity_scores") or {}
                sanity = f"{'yes' if r.get('sanity_ok') else 'no'} ({' / '.join(str(v) for v in scores.values())})"
                lines.append(f"| `{r['model']}` | {r.get('dim')} | {r.get('mode')} | {sanity} | {ceiling(r)} | {batch.get('prompt_tokens', '-')} | {batch.get('tokens_per_input', '-')} | {batch_cell} | {r.get('short_s')}s | {'yes' if r.get('listed') else 'no'} |")
            else:
                lines.append(f"| `{r['model']}` | {'yes' if r.get('sanity_ok') else 'no'} | {ceiling(r)} | {batch.get('prompt_tokens', '-')} | {'yes' if batch.get('needle_top') else 'no'} | {batch_cell} | {r.get('short_s')}s |")
        lines.append("")
    lines += ["## Unavailable or failed", "", "| Model | Kind | Outcome | Status | Detail |", "|---|---|---|---:|---|"]
    failures = unavailable_rows + other + [r for r in short_only if (r.get("long_batch") or {}).get("detail")]
    for r in sorted(failures, key=lambda row: row["model"]):
        detail = str(r.get("detail") or (r.get("long_batch") or {}).get("detail", "")).replace("|", "/").replace("\n", " ")[:240]
        lines.append(f"| `{r['model']}` | {r['kind']} | {r.get('classification')} | {r.get('status', (r.get('long_batch') or {}).get('status', '-'))} | {detail} |")
    return "\n".join(lines) + "\n"


async def main() -> None:
    parser = argparse.ArgumentParser(description="NIM embed + rerank liveness probe at max input length")
    parser.add_argument("--only", nargs="*", default=[], help="Embed model IDs to probe (default: catalog embedders)")
    parser.add_argument("--rerankers", nargs="*", default=list(KNOWN_RERANKERS), help="Reranker model IDs to probe")
    parser.add_argument("--skip-embed", action="store_true")
    parser.add_argument("--skip-rerank", action="store_true")
    parser.add_argument("--timeout", type=float, default=600.0, help="Per-request read timeout in seconds")
    parser.add_argument("--long-chars", type=int, default=60000, help="Chars per long embed input (NIM caps embed inputs at 65536 chars)")
    parser.add_argument("--batch", type=int, default=8, help="Long inputs per embed batch request")
    parser.add_argument("--rerank-long-chars", type=int, default=75000, help="Chars per long rerank passage (~14k tokens, over the 10240 ceiling)")
    parser.add_argument("--passages", type=int, default=16, help="Passages in the long rerank request")
    args = parser.parse_args()
    timeout = httpx.Timeout(connect=30, read=args.timeout, write=120, pool=args.timeout)
    limits = httpx.Limits(max_connections=2, max_keepalive_connections=2)
    async with httpx.AsyncClient(timeout=timeout, limits=limits) as client:
        catalog_response = await client.get(f"{BASE}/models", headers={"Authorization": f"Bearer {api_key()}"})
        catalog_response.raise_for_status()
        listed = sorted(item["id"] for item in catalog_response.json().get("data", []))
        embed_targets = [] if args.skip_embed else sorted(set(args.only or [m for m in listed if is_embed_candidate(m)]))
        rerank_targets = [] if args.skip_rerank else list(dict.fromkeys(args.rerankers))
        jobs = [("embed", m) for m in embed_targets] + [("rerank", m) for m in rerank_targets]
        print(f"Catalog {len(listed)}; embedders {len(embed_targets)}; rerankers {len(rerank_targets)}; timeout {args.timeout}s", flush=True)
        gate = asyncio.Semaphore(2)

        async def one(index: int, kind: str, model: str) -> dict:
            async with gate:
                print(f"[{index}/{len(jobs)}] {kind} {model}", flush=True)
                if kind == "embed":
                    result = await probe_embed(client, model, args, listed=model in listed)
                else:
                    result = await probe_rerank(client, model, args)
                batch = result.get("long_batch") or {}
                print(f"  -> {model}: {result.get('classification')} status={result.get('status', batch.get('status'))} ceiling={ceiling(result)} long_tokens={batch.get('prompt_tokens')} long={batch.get('elapsed_s')}s", flush=True)
                return result

        results = await asyncio.gather(*(one(index, kind, model) for index, (kind, model) in enumerate(jobs, 1)))
    run = {
        "timestamp": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "base": BASE,
        "rerank_base": RERANK_BASE,
        "catalog_count": len(listed),
        "timeout_s": args.timeout,
        "long_chars": args.long_chars,
        "batch": args.batch,
        "rerank_long_chars": args.rerank_long_chars,
        "passages": args.passages,
        "results": results,
    }
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    json_path = OUT_DIR / f"embed-rerank-{stamp}.json"
    md_path = OUT_DIR / f"embed-rerank-{stamp}.md"
    json_path.write_text(json.dumps(run, indent=2), encoding="utf-8")
    md_path.write_text(render(run), encoding="utf-8")
    (OUT_DIR / "latest-embed-rerank.json").write_text(json.dumps(run, indent=2), encoding="utf-8")
    (OUT_DIR / "latest-embed-rerank.md").write_text(render(run), encoding="utf-8")
    print(f"DONE: {json_path}", flush=True)
    print(md_path, flush=True)


if __name__ == "__main__":
    asyncio.run(main())
