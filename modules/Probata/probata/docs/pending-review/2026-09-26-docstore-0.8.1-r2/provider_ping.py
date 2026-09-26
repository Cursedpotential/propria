"""Tiny read-only liveness ping of the Docstore enrichment provider (and the owner's primary NIM model).

Byline: Claude Code · Opus 5.5 · 2026-09-26

Runs INSIDE the propria-docstore container with its own DOCSTORE_LLM_* environment (never printed).
Sends one short chat request per model and prints HTTP status, latency and finish_reason only.
Writes nothing.

    docker exec -i docstore-<uuid> sh -c "cd /app/scripts/docstore && python -" < provider_ping.py
"""
import json
import os
import time

import httpx

BASE = os.environ["DOCSTORE_LLM_BASE_URL"].rstrip("/")
KEY = os.environ["DOCSTORE_LLM_API_KEY"]
MODELS = [os.environ.get("DOCSTORE_LLM_MODEL", ""), "moonshotai/kimi-k3"]

for model in [m for m in MODELS if m]:
    started = time.perf_counter()
    try:
        response = httpx.post(BASE + "/chat/completions", headers={"Authorization": "Bearer " + KEY}, timeout=90,
                              json={"model": model, "max_tokens": 1600, "temperature": 0,
                                    "messages": [{"role": "user", "content": "Reply with the single word: ok"}]})
        body = response.json() if response.headers.get("content-type", "").startswith("application/json") else {}
        choice = (body.get("choices") or [{}])[0]
        print(json.dumps({"model": model, "http": response.status_code, "seconds": round(time.perf_counter() - started, 1),
                          "finish_reason": choice.get("finish_reason"),
                          "content_chars": len((choice.get("message") or {}).get("content") or "")}))
    except httpx.HTTPError as exc:
        print(json.dumps({"model": model, "error": type(exc).__name__, "seconds": round(time.perf_counter() - started, 1)}))
