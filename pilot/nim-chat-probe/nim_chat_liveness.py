#!/usr/bin/env python3
"""Fast two-stage NVIDIA NIM plain-chat liveness probe."""

from __future__ import annotations

import argparse
import json
import os
import time
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from pathlib import Path

BASE = os.environ.get("NVIDIA_NIM_API_BASE", "https://integrate.api.nvidia.com/v1").rstrip("/")
EMBED_KW = ("embed", "bge", "e5-", "minilm", "arctic", "gte", "nomic", "clip", "siglip", "rerank")


def api_key() -> str:
    value = os.environ.get("NVIDIA_API_KEY") or os.environ.get("NVIDIA_NIM_API_KEY")
    if not value:
        raise SystemExit("NVIDIA_API_KEY not set")
    return value


def request(path: str, body: dict | None, timeout: float) -> tuple[int, object, float]:
    headers = {"Authorization": f"Bearer {api_key()}", "Content-Type": "application/json"}
    req = urllib.request.Request(
        BASE + path,
        data=json.dumps(body).encode() if body else None,
        headers=headers,
        method="POST" if body else "GET",
    )
    started = time.monotonic()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as response:
            return response.status, json.loads(response.read()), round(time.monotonic() - started, 2)
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode(errors="replace")
        try:
            payload: object = json.loads(raw)
        except Exception:
            payload = raw[:500]
        return exc.code, payload, round(time.monotonic() - started, 2)
    except Exception as exc:
        return -1, str(exc)[:500], round(time.monotonic() - started, 2)


def detail(payload: object) -> str:
    if isinstance(payload, dict):
        value = payload.get("detail") or payload.get("error") or payload.get("message") or payload
        if isinstance(value, dict):
            value = value.get("message") or value
        return str(value)[:240]
    return str(payload)[:240]


def probe(model: str, timeout: float, stage: str) -> dict:
    status, payload, latency = request(
        "/chat/completions",
        {
            "model": model,
            "max_tokens": 64,
            "temperature": 0,
            "messages": [{"role": "user", "content": "Reply with exactly the word PONG."}],
        },
        timeout,
    )
    result = {"model": model, "stage": stage, "status": status, "latency_s": latency, "ok": False}
    if status == 200 and isinstance(payload, dict) and payload.get("choices"):
        msg = payload["choices"][0].get("message", {})
        visible = (msg.get("content") or "").strip()
        reasoning = (msg.get("reasoning_content") or msg.get("reasoning") or "").strip()
        usage = payload.get("usage") or {}
        result.update(
            ok=True,
            reply=(visible or reasoning)[:120],
            visible_reply=visible[:120],
            reasoning_only=bool(reasoning and not visible),
            finish_reason=payload["choices"][0].get("finish_reason"),
            prompt_tokens=usage.get("prompt_tokens"),
            completion_tokens=usage.get("completion_tokens"),
        )
    else:
        result["detail"] = detail(payload)
    return result


def retryable(result: dict) -> bool:
    return result["status"] == -1 or result["status"] == 429 or result["status"] >= 500


def render(run: dict) -> str:
    results = list(run["results"].values())
    live = [row for row in results if row["ok"]]
    visible = [row for row in live if row.get("visible_reply")]
    exact = [row for row in live if (row.get("visible_reply") or "").strip() == "PONG"]
    reasoning_only = [row for row in live if row.get("reasoning_only")]
    blank = [row for row in live if not row.get("visible_reply") and not row.get("reasoning_only")]
    unavailable = [row for row in results if not row["ok"] and row["status"] in (404, 410)]
    inconclusive = [row for row in results if not row["ok"] and row not in unavailable]
    lines = [
        f"# NVIDIA NIM chat-generation liveness — {run['timestamp']}",
        "",
        f"- Catalog: {run['catalog_count']} models; chat candidates: {len(results)}",
        f"- Endpoint live (HTTP 200 + choices): {len(live)}; visible text: {len(visible)}; exact PONG: {len(exact)}",
        f"- Reasoning-only: {len(reasoning_only)}; empty choice: {len(blank)}; unavailable (404/410): {len(unavailable)}; inconclusive: {len(inconclusive)}",
        f"- First pass: {run['fast_timeout_s']}s with {run['workers']} workers; retry: {run['retry_timeout_s']}s serialized",
        "",
        "## Live",
        "",
        "| Model | Outcome | Stage | Latency | Reply | Tokens |",
        "|---|---|---:|---:|---|---:|",
    ]
    for row in sorted(live, key=lambda item: item["latency_s"]):
        tokens = f"{row.get('prompt_tokens')}/{row.get('completion_tokens')}"
        reply = (row.get("reply") or "").replace("|", "\\|")
        if (row.get("visible_reply") or "").strip() == "PONG":
            outcome = "exact PONG"
        elif row.get("visible_reply"):
            outcome = "visible, non-exact"
        elif row.get("reasoning_only"):
            outcome = "reasoning only"
        else:
            outcome = "empty choice"
        lines.append(
            f"| `{row['model']}` | {outcome} | {row['stage']} | {row['latency_s']:.2f}s | {reply} | {tokens} |"
        )
    lines += [
        "",
        "## Unavailable or inconclusive",
        "",
        "| Model | Classification | Status | Stage | Detail |",
        "|---|---|---:|---:|---|",
    ]
    for row in sorted(unavailable + inconclusive, key=lambda item: item["model"]):
        text = (row.get("detail") or "").replace("|", "\\|")
        classification = "unavailable" if row["status"] in (404, 410) else "inconclusive"
        lines.append(
            f"| `{row['model']}` | {classification} | {row['status']} | {row['stage']} | {text} |"
        )
    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--fast-timeout", type=float, default=8.0)
    parser.add_argument("--retry-timeout", type=float, default=20.0)
    parser.add_argument("--workers", type=int, default=2)
    args = parser.parse_args()

    status, payload, _ = request("/models", None, 10)
    if status != 200 or not isinstance(payload, dict):
        raise SystemExit(f"/models failed: {status} {detail(payload)}")
    listed = [item["id"] for item in payload.get("data", [])]
    targets = [model for model in listed if not any(word in model.lower() for word in EMBED_KW)]
    print(f"/models: {len(listed)} total; {len(targets)} chat candidates", flush=True)

    with ThreadPoolExecutor(max_workers=args.workers) as executor:
        first_pass = list(executor.map(lambda model: probe(model, args.fast_timeout, "fast"), targets))
    results = {row["model"]: row for row in first_pass}
    retries = [row["model"] for row in first_pass if retryable(row)]
    print(f"fast pass complete; retrying {len(retries)} inconclusive models serially", flush=True)
    for model in retries:
        results[model] = probe(model, args.retry_timeout, "retry")
        row = results[model]
        print(f"{model}: {'LIVE' if row['ok'] else 'NOT LIVE'} ({row['status']}, {row['latency_s']}s)", flush=True)

    run = {
        "timestamp": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "base": BASE,
        "catalog_count": len(listed),
        "fast_timeout_s": args.fast_timeout,
        "retry_timeout_s": args.retry_timeout,
        "workers": args.workers,
        "results": results,
    }
    output = Path(__file__).resolve().parent
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    json_path = output / f"liveness-{stamp}.json"
    md_path = output / f"liveness-{stamp}.md"
    json_path.write_text(json.dumps(run, indent=2), encoding="utf-8")
    md_path.write_text(render(run), encoding="utf-8")
    (output / "latest-liveness.json").write_text(json.dumps(run, indent=2), encoding="utf-8")
    (output / "latest-liveness.md").write_text(render(run), encoding="utf-8")
    live = sum(1 for row in results.values() if row["ok"])
    print(f"DONE: {live}/{len(results)} live", flush=True)
    print(json_path, flush=True)
    print(md_path, flush=True)


if __name__ == "__main__":
    main()
