#!/usr/bin/env python3
"""nim-chat-probe — LIVE-test every NVIDIA NIM chat/text model: plain chat, tool calling,
vision (image input), with latency + token usage per call.

Byline: Claude Code · Fable 5 · 2026-08-26

Usage:
  python nim_chat_probe.py                     # probe all non-embedding models, save run, diff vs previous
  python nim_chat_probe.py --only a/b c/d      # probe just these ids
  python nim_chat_probe.py --extra a/b         # probe extra ids not in /models
  python nim_chat_probe.py --workers 8         # concurrency (default 6)
  python nim_chat_probe.py --tests vision      # run only one/some checks: chat,tools,vision (default all)
  python nim_chat_probe.py --last              # print newest saved report (no API calls)
  python nim_chat_probe.py --history           # one line per saved run

Per model, three real calls to /chat/completions:
  chat   : "Reply with exactly the word PONG."            -> latency, prompt/completion tokens
  tools  : a get_weather tool + "What's the weather in Paris?" -> did it emit a tool_call?
  vision : 1x1 PNG data-URI + "What color is this image?"  -> accepted image content parts?
Key: env NVIDIA_API_KEY (User-scope registry var) or NVIDIA_NIM_API_KEY.
Runs persist in ../runs/<UTC-stamp>.json + latest.md for later recall.
"""
import os, sys, json, time, glob, argparse, urllib.request, urllib.error
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone

BASE = os.environ.get("NVIDIA_NIM_API_BASE", "https://integrate.api.nvidia.com/v1").rstrip("/")
RUNS = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "runs")
EMBED_KW = ("embed", "bge", "e5-", "minilm", "arctic", "gte", "nomic", "clip", "siglip", "rerank")
# 1x1 red PNG
PNG_B64 = ("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==")
TOOL = {"type": "function", "function": {"name": "get_weather", "description": "Get current weather for a city",
        "parameters": {"type": "object", "properties": {"city": {"type": "string"}}, "required": ["city"]}}}


def key():
    k = os.environ.get("NVIDIA_API_KEY") or os.environ.get("NVIDIA_NIM_API_KEY")
    if not k:
        sys.exit("NVIDIA_API_KEY not set")
    return k


def call(path, body=None, timeout=90):
    H = {"Authorization": f"Bearer {key()}", "Content-Type": "application/json"}
    req = urllib.request.Request(BASE + path, data=json.dumps(body).encode() if body else None,
                                 headers=H, method="POST" if body else "GET")
    t = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, json.loads(r.read()), round(time.time() - t, 2)
    except urllib.error.HTTPError as e:
        raw = e.read().decode(errors="replace")
        try:
            return e.code, json.loads(raw), round(time.time() - t, 2)
        except Exception:
            return e.code, raw[:300], round(time.time() - t, 2)
    except Exception as e:
        return -1, str(e)[:300], round(time.time() - t, 2)


def detail(r):
    if isinstance(r, dict):
        d = r.get("detail") or r.get("error") or r.get("message") or json.dumps(r)
        if isinstance(d, dict):
            d = d.get("message") or json.dumps(d)
        return str(d)[:160]
    return str(r)[:160]


def usage(r):
    u = r.get("usage") or {}
    return {"prompt": u.get("prompt_tokens"), "completion": u.get("completion_tokens"), "total": u.get("total_tokens")}


def t_chat(m):
    st, r, dt = call("/chat/completions", {"model": m, "max_tokens": 16, "temperature": 0,
                                           "messages": [{"role": "user", "content": "Reply with exactly the word PONG."}]})
    if st == 200 and isinstance(r, dict) and r.get("choices"):
        msg = r["choices"][0].get("message", {})
        txt = (msg.get("content") or "") or (msg.get("reasoning_content") or "")
        return {"ok": True, "secs": dt, "usage": usage(r), "reply": txt.strip()[:40],
                "reasoning": bool(msg.get("reasoning_content"))}
    return {"ok": False, "secs": dt, "status": st, "detail": detail(r)}


def t_tools(m):
    st, r, dt = call("/chat/completions", {"model": m, "max_tokens": 128, "temperature": 0, "tools": [TOOL],
                                           "tool_choice": "auto",
                                           "messages": [{"role": "user", "content": "What's the weather in Paris right now? Use the tool."}]})
    if st == 200 and isinstance(r, dict) and r.get("choices"):
        msg = r["choices"][0].get("message", {})
        tc = msg.get("tool_calls") or []
        if tc:
            fn = tc[0].get("function", {})
            return {"ok": True, "called": True, "secs": dt, "usage": usage(r),
                    "name": fn.get("name"), "args": (fn.get("arguments") or "")[:60]}
        return {"ok": True, "called": False, "secs": dt, "usage": usage(r), "reply": (msg.get("content") or "")[:60]}
    return {"ok": False, "called": False, "secs": dt, "status": st, "detail": detail(r)}


def t_vision(m):
    st, r, dt = call("/chat/completions", {"model": m, "max_tokens": 32, "temperature": 0,
                                           "messages": [{"role": "user", "content": [
                                               {"type": "text", "text": "What color is this image? One word."},
                                               {"type": "image_url", "image_url": {"url": f"data:image/png;base64,{PNG_B64}"}}]}]})
    if st == 200 and isinstance(r, dict) and r.get("choices"):
        msg = r["choices"][0].get("message", {})
        return {"ok": True, "secs": dt, "usage": usage(r), "reply": (msg.get("content") or "").strip()[:40]}
    return {"ok": False, "secs": dt, "status": st, "detail": detail(r)}


TESTS = {"chat", "tools", "vision"}
SKIP = {"ok": False, "called": False, "skipped": True}


def probe(m):
    r = {"chat": dict(SKIP), "tools": dict(SKIP), "vision": dict(SKIP)}
    if "chat" in TESTS:
        r["chat"] = t_chat(m)
        if not r["chat"]["ok"]:
            return r  # dead model: don't burn two more calls
    if "tools" in TESTS:
        r["tools"] = t_tools(m)
    if "vision" in TESTS:
        r["vision"] = t_vision(m)
    return r


def fmt(m, r):
    c, t, v = r["chat"], r["tools"], r["vision"]
    if "chat" in TESTS and not c["ok"]:
        return f"FAIL {m:50s} chat {c['status']} {c['detail']}"
    parts = []
    if "chat" in TESTS:
        u = c["usage"]
        parts.append(f"chat {c['secs']:5.1f}s tok={u['prompt']}/{u['completion']} {'R' if c.get('reasoning') else ' '}")
    if "tools" in TESTS:
        parts.append(f"tools {'CALL' if t.get('called') else ('text' if t['ok'] else 'ERR ' + str(t.get('status')))} {t.get('secs', 0):5.1f}s")
    if "vision" in TESTS:
        parts.append(f"vision {'OK ' if v['ok'] else 'no ' + str(v.get('status', ''))} {v.get('secs', 0):5.1f}s")
    anyok = c["ok"] or t["ok"] or v["ok"]
    return f"{'OK  ' if anyok else 'FAIL'} {m:50s} " + " | ".join(parts)


def run(only, extra, workers):
    st, models, _ = call("/models")
    if st != 200:
        sys.exit(f"/models failed: {st} {detail(models)}")
    listed = sorted(x["id"] for x in models["data"])
    targets = only or [m for m in listed if not any(k in m.lower() for k in EMBED_KW)] + extra
    targets = list(dict.fromkeys(targets))
    print(f"/models: {len(listed)} total; probing {len(targets)} chat candidates with {workers} workers\n", flush=True)
    results = {}
    with ThreadPoolExecutor(max_workers=workers) as ex:
        for m, r in zip(targets, ex.map(probe, targets)):
            r["listed"] = m in listed
            results[m] = r
            print(fmt(m, r), flush=True)
    return {"ts": datetime.now(timezone.utc).isoformat(timespec="seconds"), "base": BASE,
            "tests": sorted(TESTS), "total_listed": len(listed), "results": results}


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
    L = [f"# NIM chat/text models — live probe {cur['ts']}", "",
         f"> _Byline: nim-chat-probe · {cur['ts'][:10]}_", "",
         f"- `/models` listed {cur['total_listed']} models; {len(R)} probed; {len(ok)} answered chat; "
         f"{sum(1 for m in ok if R[m]['tools'].get('called'))} emitted a tool call; "
         f"{sum(1 for m in ok if R[m]['vision']['ok'])} accepted an image", f"- endpoint: `{cur['base']}`", "",
         "## WORKING (sorted by chat latency)", "",
         "| model | chat s | prompt/completion tok | reasoning field | tool call | tools s | vision | vision s | vision reply |",
         "|---|---|---|---|---|---|---|---|---|"]
    for m in ok:
        c, t, v = R[m]["chat"], R[m]["tools"], R[m]["vision"]
        u = c.get("usage") or {"prompt": "-", "completion": "-"}
        tool = "CALL " + str(t.get("name")) if t.get("called") else ("text-only" if t["ok"] else f"ERR {t.get('status')}")
        vis = "yes" if v["ok"] else f"no ({v.get('status')})"
        L.append(f"| `{m}` | {c.get('secs', '-')} | {u['prompt']}/{u['completion']} | {'yes' if c.get('reasoning') else ''} | {tool} | "
                 f"{t.get('secs', '')} | {vis} | {v.get('secs', '')} | {str(v.get('reply', '')).replace('|', '/')} |")
    L += ["", "## NOT WORKING", "", "| model | status | detail |", "|---|---|---|"]
    for m in bad:
        c = next((R[m][k] for k in ("chat", "tools", "vision") if not R[m][k].get("skipped")), R[m]["chat"])
        L.append(f"| `{m}` | {c.get('status')} | {str(c.get('detail', '')).replace('|', '/')} |")
    L += ["", "## Changes vs previous run", ""]
    L += [f"- {d}" for d in difflines] or ["- (no previous run / no changes)"]
    with open(os.path.join(RUNS, "latest.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(L) + "\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--only", nargs="*", default=[])
    ap.add_argument("--extra", nargs="*", default=[])
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--tests", default="chat,tools,vision", help="comma list of chat,tools,vision")
    ap.add_argument("--last", action="store_true")
    ap.add_argument("--history", action="store_true")
    a = ap.parse_args()
    global TESTS
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
        # Partial run (subset of models and/or checks): overlay onto the previous full picture so
        # latest.md never regresses to a fragment; diff only what was actually re-probed.
        merged = dict(prev["results"])
        for m, r in cur["results"].items():
            base = dict(merged.get(m, {}))
            base.update({k: v for k, v in r.items() if k == "listed" or not v.get("skipped")} if isinstance(r, dict) else r)
            merged[m] = base
        cur["results"] = merged
        cur["partial"] = {"models": sorted(probed), "tests": sorted(TESTS)}
        sub_prev = {"results": {m: prev["results"][m] for m in probed if m in prev["results"]}}
        sub_cur = {"results": {m: merged[m] for m in probed}}
        difflines = diff(sub_prev, sub_cur)
    else:
        difflines = diff(prev, cur) if prev else []
    stamp = cur["ts"].replace(":", "").replace("+00:00", "Z")
    with open(os.path.join(RUNS, f"{stamp}.json"), "w", encoding="utf-8") as f:
        json.dump(cur, f, indent=1)
    write_md(cur, difflines)
    if prev_files:
        print("\n" + ("\n".join(difflines) if difflines else "(no changes vs previous run)"))
    else:
        print("\n(first run saved)")
    print(f"saved -> {RUNS}")


if __name__ == "__main__":
    main()
