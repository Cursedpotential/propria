"""Head-to-head: the owner's reviewed chunks, each labelled by every working model with the same prompt (v2).

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 21:40: take the chunks he has reviewed, rated or confirmed (not mostly
neutral; some neutral; the ones that matter), run 25 through every working model head-to-head, same prompt or tuned per
model; try older Gemini (2.5), the Claude SDK models incl. 4.x, NVIDIA (Kimi K3, GLM 5.3, DeepSeek V4.1 Flash, Laguna XS,
Nemotron, Mistral) and OpenRouter free models (north-mini-code, nex 2.5 mini/pro, Qwen 3.8 27B, GLM 5.2).
- Prompt: block_review_llm.py v2 (background, behaviours for both, worked examples from the owner's notes). The 6 example
  bouts are never in the test set.
- Each chunk goes with the 10 messages before it and the 10 after it as context (owner notes: chunks were cut short, "there
  was more to this conversation"); only the chunk's own messages [0..n-1] are labelled.
- Same prompt for every model; only the JSON mechanism differs by provider (Gemini schema mode, OpenAI-style json_schema
  for NIM/OpenRouter, SDK output_format for Claude). Per-model tuning comes after seeing where each one fails.
- Keys from --env-file (GEMINI_*, NVIDIA_API_KEY, OPENROUTER_API_KEY, CLAUDE_CODE_OAUTH_TOKEN); never printed.
Runs in the ovh-files devbox with the jev-eval venv (Claude needs claude_agent_sdk):
    .venv/bin/python code/items_h2h.py <items.json> <examples.json> <outdir> <provider:model> [...]
items.json: [{"bout_id": ..., "source": "c2024"|"f2024", "why": ...}]; bouts files: bouts/<source>_bouts_v2|v1.jsonl.
Writes <outdir>/<provider>_<model>/<bout_id>.json per item (skips items already ok, so reruns resume).
"""

import asyncio
import concurrent.futures
import datetime
import hashlib
import json
import os
import pathlib
import sys
import time

import block_review_llm as B

WINDOW = int(os.environ.get("H2H_WINDOW", "25"))  # messages on each side of the reviewed bout, all labelled
BOUT_FILES = {"c2024": "bouts/c2024_bouts_v2.jsonl", "f2024": "bouts/f2024_bouts_v1.jsonl"}
WINDOW_NOTE = ("\n\nThe block is a window cut from a longer history: it may start or end in the middle of a conversation. "
               "Split all of it into conversations and label every message.")


def load_bouts() -> dict:
    out = {}
    for src, path in BOUT_FILES.items():
        bouts = [json.loads(x) for x in pathlib.Path(path).read_text(encoding="utf-8").split("\n") if x.strip()]
        bouts.sort(key=lambda b: b["start_local"])
        flat = [(b["bout_id"], b["day"], m) for b in bouts for m in b["messages"]]
        out[src] = (bouts, flat)
    return out


def item_text(src_data, bout_id: str) -> tuple[str, int, int, int]:
    """Window mode (owner 22:01-22:02: the 30-minute bouts were never real chunks; messages a minute later belong to the
    same conversation and must be labelled; the LLM is there to find the real chunks). The model gets WINDOW messages
    either side of the reviewed bout, splits the whole window into conversations itself and labels every message.
    Returns (text, window size, first and last index of the reviewed bout inside the window)."""
    bouts, flat = src_data
    idx = [k for k, (bid, _, _) in enumerate(flat) if bid == bout_id]
    lo, hi = max(0, idx[0] - WINDOW), min(len(flat), idx[-1] + 1 + WINDOW)
    lines, prev = [], None
    for j, k in enumerate(range(lo, hi)):
        day, m = flat[k][1], flat[k][2]
        ts = datetime.datetime.fromisoformat(day + "T" + m["ts_local"])
        if prev is not None and (ts - prev).total_seconds() >= 3600:
            lines.append(f"[— {round((ts - prev).total_seconds() / 3600, 1)} h no messages —]")
        lines.append(f"[{j}] {day} {m['ts_local']} {m['who']}: {m['text']}")
        prev = ts
    return "\n".join(lines), hi - lo, idx[0] - lo, idx[-1] - lo


def call_openrouter(model: str, text: str, system: str, schema: dict) -> tuple[str, dict]:
    body = {"model": model, "temperature": 0.2, "max_tokens": 12000,
            "messages": [{"role": "system", "content": system}, {"role": "user", "content": text}],
            "response_format": {"type": "json_schema", "json_schema": {"name": "episodes", "schema": schema, "strict": False}}}
    d = B.post("https://openrouter.ai/api/v1/chat/completions", body,
               {"Authorization": f"Bearer {os.environ['OPENROUTER_API_KEY']}"})
    return d["choices"][0]["message"]["content"] or "", {"usage": d.get("usage"), "model_version": d.get("model")}


def call_claude(model: str, text: str, system: str, schema: dict) -> tuple[str, dict]:
    from claude_agent_sdk import ClaudeAgentOptions, ResultMessage, query

    async def run():
        opts = ClaudeAgentOptions(model=model, system_prompt=system, tools=[], allowed_tools=[], setting_sources=[],
                                  max_turns=4, output_format={"type": "json_schema", "schema": schema})
        res = None
        async for m in query(prompt=text, options=opts):
            if isinstance(m, ResultMessage):
                res = m
        if not (res and res.subtype == "success" and res.structured_output):
            raise RuntimeError(f"claude: no structured output ({getattr(res, 'subtype', None)}: {str(getattr(res, 'result', ''))[:200]})")
        return json.dumps(res.structured_output), {"usage": getattr(res, "usage", None), "model_version": model,
                                                   "cost_usd": getattr(res, "total_cost_usd", None)}
    return asyncio.run(run())


def call_openrouter_plain(model: str, text: str, system: str, schema: dict) -> tuple[str, dict]:
    """For OpenRouter models that return nothing in json_schema mode (nex-n2.5-mini, 2026-09-24): the schema goes into
    the instructions and the JSON is parsed from the reply."""
    body = {"model": model, "temperature": 0.2, "max_tokens": 12000,
            "messages": [{"role": "system", "content": system + "\n\nJSON schema to follow exactly:\n" + json.dumps(schema)},
                         {"role": "user", "content": text}]}
    d = B.post("https://openrouter.ai/api/v1/chat/completions", body,
               {"Authorization": f"Bearer {os.environ['OPENROUTER_API_KEY']}"})
    return d["choices"][0]["message"]["content"] or "", {"usage": d.get("usage"), "model_version": d.get("model")}


def call(provider: str, model: str, text: str, system: str, schema: dict) -> tuple[str, dict]:
    if provider == "openrouter":
        return call_openrouter(model, text, system, schema)
    if provider == "openrouter-plain":
        return call_openrouter_plain(model, text, system, schema)
    if provider == "claude":
        return call_claude(model, text, system, schema)
    return B.call(provider, model, text, system, schema)


def parse(text: str) -> dict:
    s = text.strip()
    if "```" in s:
        s = s.split("```", 1)[1].split("\n", 1)[1].rsplit("```", 1)[0]
    elif not s.startswith("{"):
        s = s[s.find("{"): s.rfind("}") + 1]
    return json.loads(s)


def run_model(spec: str, items: list, texts: dict, system: str, schema: dict, sha: str, outdir: pathlib.Path) -> str:
    provider, model = spec.split(":", 1)
    mdir = outdir / f"{provider}_{model.replace('/', '_').replace(':', '_')}"
    mdir.mkdir(parents=True, exist_ok=True)
    ok = fail = 0
    for it in items:
        f = mdir / f"{it['bout_id']}.json"
        if f.exists() and json.loads(f.read_text(encoding="utf-8")).get("ok"):
            ok += 1
            continue
        text, n, ff, ft = texts[it["bout_id"]]
        rec = {"version": "items-h2h-window-v1", "prompt_sha256": sha, "provider": provider, "model": model,
               "bout_id": it["bout_id"], "n_messages": n, "focus_from": ff, "focus_to": ft, "request_ts": datetime.datetime.now(datetime.timezone.utc).isoformat()}
        t0 = time.time()
        try:
            raw, meta = call(provider, model, text, system, schema)
            rec.update(meta, raw=raw)
            rec["output"] = parse(raw)
            rec["problems"] = B.check(rec["output"], n, B.LABELS_V2)
            rec["ok"] = True
            ok += 1
        except Exception as e:  # recorded per item, never silent
            rec.update(ok=False, error=f"{type(e).__name__}: {e}"[:1500])
            fail += 1
        rec["seconds"] = round(time.time() - t0, 1)
        f.write_text(json.dumps(rec, ensure_ascii=False, indent=1, default=str), encoding="utf-8")
    line = f"{spec}: ok={ok} failed={fail}"
    print(line, flush=True)
    return line


def main() -> int:
    items = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
    examples = json.loads(pathlib.Path(sys.argv[2]).read_text(encoding="utf-8"))
    outdir = pathlib.Path(sys.argv[3])
    specs = sys.argv[4:]
    data = load_bouts()
    by_id = {b["bout_id"]: b for b in data["c2024"][0]}
    for e in examples:
        b = by_id[e["bout_id"]]
        e["text"] = "\n".join(f"[{i}] {b['day']} {m['ts_local']} {m['who']}: {m['text']}" for i, m in enumerate(b["messages"]))
    assert not {e["bout_id"] for e in examples} & {it["bout_id"] for it in items}, "an example bout is in the test set"
    system = B.system_v2(examples) + WINDOW_NOTE
    schema = B.SCHEMA_V2
    sha = hashlib.sha256((system + json.dumps(schema, sort_keys=True)).encode()).hexdigest()
    texts = {it["bout_id"]: item_text(data[it.get("source", "c2024")], it["bout_id"]) for it in items}
    outdir.mkdir(parents=True, exist_ok=True)
    (outdir / "_run.json").write_text(json.dumps({"version": "items-h2h-window-v1", "prompt_sha256": sha, "system": system,
                                                  "schema": schema, "items": items, "texts": texts, "models": specs},
                                                 ensure_ascii=False, indent=1), encoding="utf-8")
    print(f"items-h2h-v2 prompt {sha[:12]}: {len(items)} items x {len(specs)} models", flush=True)
    with concurrent.futures.ThreadPoolExecutor(len(specs)) as pool:
        list(pool.map(lambda s: run_model(s, items, texts, system, schema, sha, outdir), specs))
    return 0


if __name__ == "__main__":
    sys.exit(main())
