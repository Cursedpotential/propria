"""Liveness sweep of candidate models through the llm_probe service (the owner's NIM probe tool).

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 21:40: check which requested models work (NVIDIA NIM, OpenRouter,
Google) before running the 25 reviewed chunks head-to-head; "use the NIM probe tool". Results are persisted to
llm_probe's own board (persist=true) and printed as a table. Standard library only.
    python3 probe_models_sweep.py <llm_probe base url> <list file: "provider model" per line> [probe] [workers]
"""

import concurrent.futures
import json
import sys
import urllib.request

base, list_path = sys.argv[1].rstrip("/"), sys.argv[2]
probe = sys.argv[3] if len(sys.argv) > 3 else "liveness"
workers = int(sys.argv[4]) if len(sys.argv) > 4 else 8
pairs = [ln.split(None, 1) for ln in open(list_path, encoding="utf-8").read().splitlines() if ln.strip() and not ln.startswith("#")]


def run(provider: str, model: str) -> tuple[str, str, dict]:
    body = {"provider": provider, "model": model, "probe": probe, "max_tokens": 500, "persist": True,
            "run_note": "2026-09-24 owner model sweep for bout labelling"}
    req = urllib.request.Request(f"{base}/probe/run", data=json.dumps(body).encode(), headers={"content-type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=240) as r:
            return provider, model, json.loads(r.read().decode())
    except Exception as e:  # recorded, never silent
        return provider, model, {"ok": False, "reason": type(e).__name__, "error": str(e)[:200]}


with concurrent.futures.ThreadPoolExecutor(workers) as pool:
    results = list(pool.map(lambda p: run(*p), pairs))
for provider, model, d in sorted(results, key=lambda x: (not x[2].get("ok"), x[0], x[1])):
    why = d.get("reason") or ""
    err = str(d.get("error") or d.get("detail") or "")[:110].replace("\n", " ")
    print(f"{'OK  ' if d.get('ok') else 'FAIL'} {provider:<11} {model:<45} {d.get('latency_s') or '':>7} {why} {err if not d.get('ok') else ''}")
