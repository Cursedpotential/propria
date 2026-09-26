#!/usr/bin/env python3
"""Recheck ambiguous NIM chat results using SSE streaming without a post-first-token timeout."""

from __future__ import annotations

import asyncio
import argparse
import json
import os
import time
from datetime import datetime, timezone
from pathlib import Path

import httpx

BASE = os.environ.get("NVIDIA_NIM_API_BASE", "https://integrate.api.nvidia.com/v1").rstrip("/")
OUT_DIR = Path(__file__).resolve().parent
EMBED_MARKERS = ("embed", "bge", "e5-", "minilm", "arctic", "gte", "nomic", "clip", "siglip", "rerank")
# These are deliberately excluded from the *general chat* list, rather than
# reported as failed chat models.  They are classifiers, guards, parsers, or
# translation-only endpoints.
SPECIALIZED_MARKERS = (
    "content-safety", "safety-guard", "nemoguard", "llama-guard", "topic-control",
    "synthetic-video-detector", "nemotron-parse", "riva-translate", "ising-calibration",
)


def api_key() -> str:
    value = os.environ.get("NVIDIA_API_KEY") or os.environ.get("NVIDIA_NIM_API_KEY")
    if not value:
        raise SystemExit("NVIDIA_API_KEY not set")
    return value


def is_general_chat_candidate(model: str) -> bool:
    name = model.lower()
    return not any(marker in name for marker in EMBED_MARKERS + SPECIALIZED_MARKERS)


async def consume_stream(client: httpx.AsyncClient, model: str, first_event_timeout: float) -> dict:
    started = time.monotonic()
    request = client.build_request(
        "POST",
        f"{BASE}/chat/completions",
        headers={"Authorization": f"Bearer {api_key()}", "Content-Type": "application/json"},
        json={
            "model": model,
            "stream": True,
            "max_tokens": 8192,
            "temperature": 0,
            "messages": [{"role": "user", "content": "Reply with exactly the word PONG."}],
        },
    )
    try:
        response = await asyncio.wait_for(client.send(request, stream=True), timeout=first_event_timeout)
    except asyncio.TimeoutError:
        return {"model": model, "ok": False, "classification": "no_headers", "elapsed_s": round(time.monotonic() - started, 2)}
    except Exception as exc:
        return {"model": model, "ok": False, "classification": "connection_error", "detail": repr(exc), "elapsed_s": round(time.monotonic() - started, 2)}

    result = {"model": model, "status": response.status_code, "ok": False}
    if response.status_code != 200:
        raw = await response.aread()
        await response.aclose()
        result.update(
            classification="http_error",
            detail=raw.decode(errors="replace")[:1000],
            elapsed_s=round(time.monotonic() - started, 2),
        )
        return result

    visible_parts: list[str] = []
    reasoning_parts: list[str] = []
    first_data_s = None
    finish_reason = None
    usage = None
    event_count = 0
    iterator = response.aiter_lines()

    try:
        # Only the first real SSE data event has a timeout. After that, consume until [DONE].
        while True:
            try:
                line = await asyncio.wait_for(iterator.__anext__(), timeout=first_event_timeout)
            except StopAsyncIteration:
                break
            except asyncio.TimeoutError:
                result.update(
                    classification="no_first_data_event",
                    elapsed_s=round(time.monotonic() - started, 2),
                )
                return result
            if not line.startswith("data:"):
                continue
            payload = line[5:].strip()
            if not payload:
                continue
            first_data_s = round(time.monotonic() - started, 2)
            if payload == "[DONE]":
                break
            event_count += 1
            try:
                chunk = json.loads(payload)
            except json.JSONDecodeError:
                continue
            usage = chunk.get("usage") or usage
            choices = chunk.get("choices") or []
            if not choices:
                continue
            choice = choices[0]
            finish_reason = choice.get("finish_reason") or finish_reason
            delta = choice.get("delta") or choice.get("message") or {}
            content = delta.get("content")
            reasoning = delta.get("reasoning_content") or delta.get("reasoning")
            if content:
                visible_parts.append(content)
            if reasoning:
                reasoning_parts.append(reasoning)
            break

        # Streaming has started. Deliberately no timeout from here through [DONE].
        async for line in iterator:
            if not line.startswith("data:"):
                continue
            payload = line[5:].strip()
            if not payload:
                continue
            if payload == "[DONE]":
                break
            event_count += 1
            try:
                chunk = json.loads(payload)
            except json.JSONDecodeError:
                continue
            usage = chunk.get("usage") or usage
            choices = chunk.get("choices") or []
            if not choices:
                continue
            choice = choices[0]
            finish_reason = choice.get("finish_reason") or finish_reason
            delta = choice.get("delta") or choice.get("message") or {}
            content = delta.get("content")
            reasoning = delta.get("reasoning_content") or delta.get("reasoning")
            if content:
                visible_parts.append(content)
            if reasoning:
                reasoning_parts.append(reasoning)
    except Exception as exc:
        result.update(classification="stream_error", detail=repr(exc))
    finally:
        await response.aclose()

    visible = "".join(visible_parts).strip()
    reasoning = "".join(reasoning_parts).strip()
    result.update(
        ok=bool(visible or reasoning or event_count),
        classification=("visible_text" if visible else "reasoning_only" if reasoning else "events_without_text" if event_count else "empty_stream"),
        first_data_s=first_data_s,
        elapsed_s=round(time.monotonic() - started, 2),
        event_count=event_count,
        visible_text=visible,
        reasoning_text=reasoning,
        exact_pong=visible == "PONG",
        finish_reason=finish_reason,
        usage=usage,
    )
    return result


def render(run: dict) -> str:
    results = run["results"]
    working = [r for r in results if r.get("classification") == "visible_text"]
    unavailable = [r for r in results if r.get("status") in (404, 410)]
    pending = [r for r in results if r not in working and r not in unavailable]
    lines = [
        f"# NVIDIA NIM general-chat liveness — {run['timestamp']}", "",
        "> _Byline: Codex · streamed plain-chat probe · 2026-08-29_", "",
        f"- Catalog: {run['catalog_count']} models; general-chat candidates: {len(results)}; excluded specialized endpoints: {len(run['excluded'])}",
        f"- Method: SSE streaming; {run['first_event_timeout_s']}s maximum wait for response/first event; once streaming starts, read continues until `[DONE]` with no whole-answer timeout.",
        f"- Visible chat response: {len(working)}; exact `PONG`: {sum(r.get('exact_pong') for r in working)}; unavailable (404/410): {len(unavailable)}; unresolved: {len(pending)}", "",
        "## General chat models that answered", "",
        "| Model | First token | Total | Exact PONG | Finish |", "|---|---:|---:|---|---|",
    ]
    for r in sorted(working, key=lambda row: row.get("first_data_s", 9999)):
        lines.append(f"| `{r['model']}` | {r.get('first_data_s', '-')}s | {r.get('elapsed_s', '-')}s | {'yes' if r.get('exact_pong') else 'no'} | {r.get('finish_reason') or '-'} |")
    lines += ["", "## Unavailable or unresolved", "", "| Model | Outcome | Status | Detail |", "|---|---|---:|---|"]
    for r in sorted(unavailable + pending, key=lambda row: row["model"]):
        detail = str(r.get("detail", "")).replace("|", "/")[:240]
        lines.append(f"| `{r['model']}` | {r.get('classification')} | {r.get('status', '-')} | {detail} |")
    lines += ["", "## Excluded specialized endpoints", ""]
    lines += [f"- `{model}`" for model in run["excluded"]]
    return "\n".join(lines) + "\n"


async def main() -> None:
    parser = argparse.ArgumentParser(description="Streamed NIM general-chat liveness probe")
    parser.add_argument("--only", nargs="*", default=[], help="Run only these catalog model IDs")
    parser.add_argument("--first-event-timeout", type=float, default=20.0)
    args = parser.parse_args()
    timeout = httpx.Timeout(connect=15, read=None, write=30, pool=15)
    limits = httpx.Limits(max_connections=2, max_keepalive_connections=2)
    results = []
    async with httpx.AsyncClient(timeout=timeout, limits=limits) as client:
        catalog_response = await client.get(f"{BASE}/models", headers={"Authorization": f"Bearer {api_key()}"})
        catalog_response.raise_for_status()
        catalog = catalog_response.json().get("data", [])
        listed = sorted(item["id"] for item in catalog)
        general_chat_models = [model for model in listed if is_general_chat_candidate(model)]
        targets = args.only or general_chat_models
        unknown = [model for model in targets if model not in listed]
        if unknown:
            raise SystemExit(f"Not present in current NIM catalog: {', '.join(unknown)}")
        targets = sorted(set(targets))
        excluded = [model for model in listed if model not in general_chat_models]
        print(f"Catalog {len(listed)}; streamed general-chat candidates {len(targets)}; excluded specialized {len(excluded)}", flush=True)
        gate = asyncio.Semaphore(2)

        async def one(index: int, model: str) -> dict:
            async with gate:
                print(f"[{index}/{len(targets)}] {model}", flush=True)
                result = await consume_stream(client, model, first_event_timeout=args.first_event_timeout)
                print(f"  -> {result.get('classification')} status={result.get('status')} first={result.get('first_data_s')}s total={result.get('elapsed_s')}s", flush=True)
                return result

        results = await asyncio.gather(*(one(index, model) for index, model in enumerate(targets, 1)))
    run = {
        "timestamp": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "base": BASE,
        "catalog_count": len(listed),
        "first_event_timeout_s": args.first_event_timeout,
        "results": results,
        "excluded": excluded,
    }
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    json_path = OUT_DIR / f"streamed-general-chat-{stamp}.json"
    md_path = OUT_DIR / f"streamed-general-chat-{stamp}.md"
    json_path.write_text(json.dumps(run, indent=2), encoding="utf-8")
    md_path.write_text(render(run), encoding="utf-8")
    (OUT_DIR / "latest-streamed-general-chat.json").write_text(json.dumps(run, indent=2), encoding="utf-8")
    (OUT_DIR / "latest-streamed-general-chat.md").write_text(render(run), encoding="utf-8")
    print(f"DONE: {json_path}", flush=True)
    print(md_path, flush=True)


if __name__ == "__main__":
    asyncio.run(main())
