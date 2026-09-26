#!/usr/bin/env python3
"""Merge a full streamed NIM sweep with one or more targeted rechecks."""

from __future__ import annotations

import argparse
import json
from pathlib import Path


def render(run: dict) -> str:
    results = run["results"]
    working = [r for r in results if r.get("classification") == "visible_text"]
    unavailable = [r for r in results if r.get("status") in (404, 410)]
    unresolved = [r for r in results if r not in working and r not in unavailable]
    lines = [
        "# NVIDIA NIM general-chat liveness — corrected streamed report", "",
        "> _Byline: Codex · streamed plain-chat probe · 2026-08-29_", "",
        f"- Catalog: {run['catalog_count']} models; general-chat candidates: {len(results)}; specialized endpoints excluded: {len(run['excluded'])}",
        "- Method: stream `POST /v1/chat/completions`; wait up to 20 seconds for first SSE event, then recheck that slow set with 60 seconds. Once streaming begins, consume through `[DONE]` without a whole-answer timeout.",
        f"- Good general chat: {len(working)} models with visible text; {sum(r.get('exact_pong') for r in working)} followed the exact PONG instruction.",
        f"- Unavailable: {len(unavailable)} catalog entries returned 404/410. Non-text stream: {sum(r.get('classification') == 'events_without_text' for r in results)}. No headers after 60 seconds: {sum(r.get('classification') == 'no_headers' for r in results)}.", "",
        "## Good general-chat models", "",
        "| Model | First streamed text | Full response | Exact PONG |", "|---|---:|---:|---|",
    ]
    for r in sorted(working, key=lambda row: row.get("first_data_s", 9999)):
        lines.append(f"| `{r['model']}` | {r.get('first_data_s')}s | {r.get('elapsed_s')}s | {'yes' if r.get('exact_pong') else 'no'} |")
    lines += ["", "## Not usable as general chat in this run", "", "| Model | Result | Status |", "|---|---|---:|"]
    for r in sorted(unavailable + unresolved, key=lambda row: row["model"]):
        lines.append(f"| `{r['model']}` | {r.get('classification')} | {r.get('status', '-')} |")
    lines += ["", "## Excluded specialized endpoints", ""]
    lines += [f"- `{model}`" for model in run["excluded"]]
    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("full", type=Path)
    parser.add_argument("recheck", type=Path, nargs="+")
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()

    full = json.loads(args.full.read_text(encoding="utf-8"))
    merged = {r["model"]: r for r in full["results"]}
    for path in args.recheck:
        retry = json.loads(path.read_text(encoding="utf-8"))
        merged.update({r["model"]: r for r in retry["results"]})
    output = {**full, "results": [merged[m] for m in sorted(merged)], "rechecks": [str(p) for p in args.recheck]}
    args.out.with_suffix(".json").write_text(json.dumps(output, indent=2), encoding="utf-8")
    args.out.with_suffix(".md").write_text(render(output), encoding="utf-8")
    print(args.out.with_suffix(".json"))
    print(args.out.with_suffix(".md"))


if __name__ == "__main__":
    main()
