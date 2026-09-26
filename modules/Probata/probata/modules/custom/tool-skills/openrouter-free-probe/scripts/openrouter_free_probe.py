#!/usr/bin/env python3
"""openrouter-free-probe — LIVE-test every OpenRouter FREE model: plain chat, tool calling,
vision (image input), with latency + token usage per call. Sister of nim-chat-probe.

Byline: Claude Code · Fable 5 · 2026-08-26

Usage:
  python openrouter_free_probe.py                    # all free models, all 3 checks, save + diff
  python openrouter_free_probe.py --tests tools      # one/some checks only: chat,tools,vision
  python openrouter_free_probe.py --only a/b:free    # specific ids
  python openrouter_free_probe.py --extra a/b        # extra ids (need not be free)
  python openrouter_free_probe.py --workers 3        # concurrency (default 2 — free tier is ~20 req/min)
  python openrouter_free_probe.py --last | --history

"Free" = id ends with ":free" OR pricing.prompt == pricing.completion == "0".
/models also tells us what OpenRouter *claims* (input_modalities, supported_parameters); we
record the claim next to the live result so claim-vs-reality drift is visible.
Key: env OPENROUTER_API_KEY. Runs persist in ../runs/<UTC-stamp>.json + latest.md.
"""
import os, sys, json, time, glob, argparse, urllib.request, urllib.error
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone

BASE = os.environ.get("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1").rstrip("/")
RUNS = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "runs")
PNG_B64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="
TOOL = {"type": "function", "function": {"name": "get_weather", "description": "Get current weather for a city",
        "parameters": {"type": "object", "properties": {"city": {"type": "string"}}, "required": ["city"]}}}
TESTS = {"chat", "tools", "vision"}
SKIP = {"ok": False, "called": False, "skipped": True}


def key():
    k = os.environ.get("OPENROUTER_API_KEY")
    if not k:
        sys.exit("OPENROUTER_API_KEY not set")
    return k


def call(path, body=None, timeout=90, retries=3):
    H = {"Authorization": f"Bearer {key()}", "Content-Type": "application/json",
         "HTTP-Referer": "https://github.com/Cursedpotential", "X-Title": "openrouter-free-probe"}
    t = time.time()
    for attempt in range(retries + 1):
        req = urllib.request.Request(BASE + path, data=json.dumps(body).encode() if body else None,
                                     headers=H, method="POST" if body else "GET")
        try:
            with urllib.request.urlopen(req, timeout=timeout) as r:
                return r.status, json.loads(r.read()), round(time.time() - t, 2)
        except urllib.error.HTTPError as e:
            raw = e.read().decode(errors="replace")
            if e.code == 429 and attempt < retries:
                time.sleep(4 * (attempt + 1))
                continue
            try:
                return e.code, json.loads(raw), round(time.time() - t, 2)
            except Exception:
                return e.code, raw[:300], round(time.time() - t, 2)
        except Exception as e:
            return -1, str(e)[:300], round(time.time() - t, 2)


def detail(r):
    if isinstance(r, dict):
        d = r.get("error") or r.get("detail") or r.get("message") or json.dumps(r)
        if isinstance(d, dict):
            d = d.get("message") or json.dumps(d)
        return str(d)[:160]
    return str(r)[:160]


def usage(r):
    u = r.get("usage") or {}
    return {"prompt": u.get("prompt_tokens"), "completion": u.get("completion_tokens"), "total": u.get("total_tokens")}


def choice_msg(r):
    """OpenRouter sometimes returns 200 with an error body or an empty choices list."""
    if isinstance(r, dict) and r.get("choices"):
        return r["choices"][0].get("message", {}) or {}
    return None


def t_chat(m):
    st, r, dt = call("/chat/completions", {"model": m, "max_tokens": 16, "temperature": 0,
                                           "messages": [{"role": "user", "content": "Reply with exactly the word PONG."}]})
    msg = choice_msg(r) if st == 200 else None
    if msg is not None:
        txt = (msg.get("content") or "") or (msg.get("reasoning") or "")
        return {"ok": True, "secs": dt, "usage": usage(r), "reply": txt.strip()[:40],
                "reasoning": bool(msg.get("reasoning")), "provider": r.get("provider")}
    return {"ok": False, "secs": dt, "status": st, "detail": detail(r)}


def t_tools(m):
    st, r, dt = call("/chat/completions", {"model": m, "max_tokens": 128, "temperature": 0, "tools": [TOOL],
                                           "tool_choice": "auto",
                                           "messages": [{"role": "user", "content": "What's the weather in Paris right now? Use the tool."}]})
    msg = choice_msg(r) if st == 200 else None
    if msg is not None:
        tc = msg.get("tool_calls") or []
        if tc:
            fn = tc[0].get("function", {})
            return {"ok": True, "called": True, "secs": dt, "usage": usage(r), "name": fn.get("name"),
                    "args": (fn.get("arguments") or "")[:60]}
        return {"ok": True, "called": False, "secs": dt, "usage": usage(r), "reply": (msg.get("content") or "")[:60]}
    return {"ok": False, "called": False, "secs": dt, "status": st, "detail": detail(r)}


def t_vision(m):
    st, r, dt = call("/chat/completions", {"model": m, "max_tokens": 32, "temperature": 0,
                                           "messages": [{"role": "user", "content": [
                                               {"type": "text", "text": "What color is this image? One word."},
                                               {"type": "image_url", "image_url": {"url": f"data:image/png;base64,{PNG_B64}"}}]}]})
    msg = choice_msg(r) if st == 200 else None
    if msg is not None:
        return {"ok": True, "secs": dt, "usage": usage(r), "reply": (msg.get("content") or "").strip()[:40]}
    return {"ok": False, "secs": dt, "status": st, "detail": detail(r)}


def probe(m):
    r = {"chat": dict(SKIP), "tools": dict(SKIP), "vision": dict(SKIP)}
    if "chat" in TESTS:
        r["chat"] = t_chat(m)
        if not r["chat"]["ok"]:
            return r
    if "tools" in TESTS:
        r["tools"] = t_tools(m)
    if "vision" in TESTS:
        r["vision"] = t_vision(m)
    return r


def fmt(m, r):
    c, t, v = r["chat"], r["tools"], r["vision"]
    if "chat" in TESTS and not c["ok"]:
        return f"FAIL {m:52s} chat {c['status']} {c['detail']}"
    parts = []
    if "chat" in TESTS:
        u = c["usage"]
        parts.append(f"chat {c['secs']:5.1f}s tok={u['prompt']}/{u['completion']} {'R' if c.get('reasoning') else ' '}")
    if "tools" in TESTS:
        parts.append(f"tools {'CALL' if t.get('called') else ('text' if t['ok'] else 'ERR ' + str(t.get('status')))} {t.get('secs', 0):5.1f}s")
    if "vision" in TESTS:
        parts.append(f"vision {'OK ' if v['ok'] else 'no ' + str(v.get('status', ''))} {v.get('secs', 0):5.1f}s")
    anyok = c["ok"] or t["ok"] or v["ok"]
    return f"{'OK  ' if anyok else 'FAIL'} {m:52s} " + " | ".join(parts)


def is_free(md):
    p = md.get("pricing") or {}
    return md["id"].endswith(":free") or (str(p.get("prompt")) in ("0", "0.0") and str(p.get("completion")) in ("0", "0.0"))


def run(only, extra, workers):
    st, models, _ = call("/models")
    if st != 200:
        sys.exit(f"/models failed: {st} {detail(models)}")
    meta = {x["id"]: x for x in models["data"]}
    free = sorted(i for i, x in meta.items() if is_free(x))
    targets = list(dict.fromkeys(only or free + extra))
    print(f"/models: {len(meta)} total, {len(free)} free; probing {len(targets)} with {workers} workers\n", flush=True)
    results = {}
    with ThreadPoolExecutor(max_workers=workers) as ex:
        for m, r in zip(targets, ex.map(probe, targets)):
            x = meta.get(m, {})
            r["listed"] = m in meta
            r["free"] = m in free
            r["claims"] = {"input": (x.get("architecture") or {}).get("input_modalities"),
                           "tools": "tools" in (x.get("supported_parameters") or []),
                           "context": x.get("context_length")}
            results[m] = r
            print(fmt(m, r), flush=True)
    return {"ts": datetime.now(timezone.utc).isoformat(timespec="seconds"), "base": BASE, "tests": sorted(TESTS),
            "total_listed": len(meta), "total_free": len(free), "results": results}


def saved_runs():
    return sorted(glob.glob(os.path.join(RUNS, "*.json")))


def caps(r):
    return (r["chat"]["ok"], bool(r["tools"].get("called")), r["vision"]["ok"])


def diff(prev, cur):
    lines = []
    for m, r in cur["results"].items():
        p = prev["results"].get(m)
        if p is None:
            lines.append(f"+ NEW      {m}: chat={caps(r)[0]} tools={caps(r)[1]} vision={caps(r)[2]}")
        elif caps(p) != caps(r):
            lines.append(f"! CHANGED  {m}: {caps(p)} -> {caps(r)} (chat,tools,vision)")
    for m in prev["results"]:
        if m not in cur["results"]:
            lines.append(f"- DROPPED  {m}")
    return lines


def write_md(cur, difflines):
    R = cur["results"]
    alive = lambda m: R[m]["chat"]["ok"] or R[m]["tools"]["ok"] or R[m]["vision"]["ok"]
    ok = sorted([m for m in R if alive(m)], key=lambda m: R[m]["chat"].get("secs") or R[m]["tools"].get("secs") or R[m]["vision"].get("secs") or 0)
    bad = [m for m in R if not alive(m)]
    L = [f"# OpenRouter FREE models — live probe {cur['ts']}", "",
         f"> _Byline: openrouter-free-probe · {cur['ts'][:10]}_", "",
         f"- `/models` listed {cur['total_listed']} models, {cur['total_free']} free; {len(R)} probed; {len(ok)} alive; "
         f"{sum(1 for m in ok if R[m]['tools'].get('called'))} emitted a tool call; "
         f"{sum(1 for m in ok if R[m]['vision']['ok'])} accepted an image", f"- endpoint: `{cur['base']}`", "",
         "## WORKING (sorted by chat latency)", "",
         "| model | ctx | chat s | prompt/completion tok | reasoning | tool call | claims tools | tools s | vision | claims image | vision s | vision reply |",
         "|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for m in ok:
        c, t, v, cl = R[m]["chat"], R[m]["tools"], R[m]["vision"], R[m].get("claims", {})
        u = c.get("usage") or {"prompt": "-", "completion": "-"}
        tool = "CALL " + str(t.get("name")) if t.get("called") else ("text-only" if t["ok"] else f"ERR {t.get('status')}")
        vis = "yes" if v["ok"] else f"no ({v.get('status')})"
        L.append(f"| `{m}` | {cl.get('context', '')} | {c.get('secs', '-')} | {u['prompt']}/{u['completion']} | {'yes' if c.get('reasoning') else ''} | "
                 f"{tool} | {'yes' if cl.get('tools') else 'no'} | {t.get('secs', '')} | {vis} | "
                 f"{'yes' if 'image' in (cl.get('input') or []) else 'no'} | {v.get('secs', '')} | {str(v.get('reply', '')).replace('|', '/')} |")
    L += ["", "## NOT WORKING", "", "| model | status | detail |", "|---|---|---|"]
    for m in bad:
        c = next((R[m][k] for k in ("chat", "tools", "vision") if not R[m][k].get("skipped")), R[m]["chat"])
        L.append(f"| `{m}` | {c.get('status')} | {str(c.get('detail', '')).replace('|', '/')} |")
    L += ["", "## Changes vs previous run", ""]
    L += [f"- {d}" for d in difflines] or ["- (no previous run / no changes)"]
    with open(os.path.join(RUNS, "latest.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(L) + "\n")


def main():
    global TESTS
    ap = argparse.ArgumentParser()
    ap.add_argument("--only", nargs="*", default=[])
    ap.add_argument("--extra", nargs="*", default=[])
    ap.add_argument("--workers", type=int, default=2)
    ap.add_argument("--tests", default="chat,tools,vision")
    ap.add_argument("--last", action="store_true")
    ap.add_argument("--history", action="store_true")
    a = ap.parse_args()
    TESTS = {t.strip() for t in a.tests.split(",")} & {"chat", "tools", "vision"} or TESTS
    os.makedirs(RUNS, exist_ok=True)
    if a.last:
        p = os.path.join(RUNS, "latest.md")
        print(open(p, encoding="utf-8").read() if os.path.exists(p) else "no runs yet")
        return
    if a.history:
        for f in saved_runs():
            d = json.load(open(f, encoding="utf-8"))
            R = d["results"]
            print(f"{d['ts']}  chat={sum(R[m]['chat']['ok'] for m in R)} tools={sum(bool(R[m]['tools'].get('called')) for m in R)} "
                  f"vision={sum(R[m]['vision']['ok'] for m in R)} /{len(R)}  {os.path.basename(f)}")
        return
    prev_files = saved_runs()
    prev = json.load(open(prev_files[-1], encoding="utf-8")) if prev_files else None
    cur = run(a.only, a.extra, a.workers)
    probed = set(cur["results"])
    if prev and (a.only or set(TESTS) != {"chat", "tools", "vision"}):
        merged = dict(prev["results"])
        for m, r in cur["results"].items():
            base = dict(merged.get(m, {}))
            base.update({k: v for k, v in r.items() if not (isinstance(v, dict) and v.get("skipped"))})
            merged[m] = base
        cur["results"] = merged
        cur["partial"] = {"models": sorted(probed), "tests": sorted(TESTS)}
        difflines = diff({"results": {m: prev["results"][m] for m in probed if m in prev["results"]}},
                         {"results": {m: merged[m] for m in probed}})
    else:
        difflines = diff(prev, cur) if prev else []
    stamp = cur["ts"].replace(":", "").replace("+00:00", "Z")
    with open(os.path.join(RUNS, f"{stamp}.json"), "w", encoding="utf-8") as f:
        json.dump(cur, f, indent=1)
    write_md(cur, difflines)
    if prev:
        print("\n" + ("\n".join(difflines) if difflines else "(no changes vs previous run)"))
    else:
        print("\n(first run saved)")
    print(f"saved -> {RUNS}")


if __name__ == "__main__":
    main()
