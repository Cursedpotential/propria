#!/usr/bin/env python3
"""nim-embed-probe — list NVIDIA NIM embedding models and LIVE-test each one.

Byline: Claude Code · Fable 5 · 2026-08-26

Usage:
  python nim_embed_probe.py                # list + probe, save run, diff vs previous
  python nim_embed_probe.py --extra a/b    # also probe models not in /models (retired?)
  python nim_embed_probe.py --last         # just print the most recent saved run (no API calls)
  python nim_embed_probe.py --history      # one line per saved run

Key comes from env NVIDIA_API_KEY (User-scope registry var on this box) or NVIDIA_NIM_API_KEY.
Runs are saved next to this skill in ../runs/<UTC-stamp>.json + latest.md so any later session
can recall "what was working on <date>" without re-hitting the API.
"""
import os, sys, json, time, glob, argparse, urllib.request, urllib.error
from datetime import datetime, timezone

BASE = os.environ.get("NVIDIA_NIM_API_BASE", "https://integrate.api.nvidia.com/v1").rstrip("/")
RUNS = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "runs")
KEYWORDS = ("embed", "bge", "e5", "minilm", "arctic", "gte", "nomic", "clip", "siglip")
# Models known to have existed on NIM; always probed so retirements are caught even after delisting.
KNOWN = ["nvidia/nv-embed-v1", "baai/bge-m3", "nvidia/nv-embedqa-e5-v5",
         "nvidia/llama-3.2-nv-embedqa-1b-v2", "nvidia/nemotron-3-embed-1b",
         "nvidia/llama-nemotron-embed-vl-1b-v2"]
TEXTS = ["the quick brown fox", "a fast auburn dog", "quarterly tax filing deadline", "photosynthesis in plants"]


def key():
    k = os.environ.get("NVIDIA_API_KEY") or os.environ.get("NVIDIA_NIM_API_KEY")
    if not k:
        sys.exit("NVIDIA_API_KEY not set")
    return k


def call(path, body=None, timeout=60):
    H = {"Authorization": f"Bearer {key()}", "Content-Type": "application/json"}
    req = urllib.request.Request(BASE + path, data=json.dumps(body).encode() if body else None,
                                 headers=H, method="POST" if body else "GET")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, json.loads(r.read())
    except urllib.error.HTTPError as e:
        raw = e.read().decode(errors="replace")
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, raw[:300]
    except Exception as e:
        return -1, str(e)[:300]


def detail(r):
    if isinstance(r, dict):
        return r.get("detail") or r.get("error") or json.dumps(r)[:200]
    return str(r)[:200]


def probe(model, extra=None):
    body = {"model": model, "input": TEXTS, "encoding_format": "float"}
    if extra:
        body.update(extra)
    t = time.time()
    st, r = call("/embeddings", body)
    dt = round(time.time() - t, 2)
    if st == 200 and isinstance(r, dict) and "data" in r:
        vecs = [d["embedding"] for d in r["data"]]
        return {"ok": True, "n": len(vecs), "dim": len(vecs[0]), "secs": dt}
    return {"ok": False, "status": st, "detail": detail(r), "secs": dt}


def run(extra_models):
    st, models = call("/models")
    if st != 200:
        sys.exit(f"/models failed: {st} {detail(models)}")
    listed = sorted(m["id"] for m in models["data"])
    emb = [m for m in listed if any(k in m.lower() for k in KEYWORDS)]
    targets = list(dict.fromkeys(emb + KNOWN + extra_models))
    results = {}
    for m in targets:
        r = probe(m)
        mode = "symmetric"
        if not r["ok"]:
            r2 = probe(m, {"input_type": "passage", "truncate": "NONE"})
            if r2["ok"]:
                r, mode = r2, "asymmetric(input_type required)"
        r["listed"] = m in listed
        r["mode"] = mode if r["ok"] else None
        results[m] = r
        flag = "OK  " if r["ok"] else "FAIL"
        info = (f"dim={r['dim']} n={r['n']} {r['secs']}s {r['mode']}" if r["ok"]
                else f"{r['status']} {r['detail']}")
        print(f"{flag} {m:48s} {'listed' if r['listed'] else 'UNLISTED':8s} {info}", flush=True)
    return {"ts": datetime.now(timezone.utc).isoformat(timespec="seconds"), "base": BASE,
            "total_listed": len(listed), "results": results}


def saved_runs():
    return sorted(glob.glob(os.path.join(RUNS, "*.json")))


def diff(prev, cur):
    lines = []
    for m, r in cur["results"].items():
        p = prev["results"].get(m)
        if p is None:
            lines.append(f"+ NEW      {m}: {'OK' if r['ok'] else 'FAIL'}")
        elif p["ok"] != r["ok"]:
            lines.append(f"! CHANGED  {m}: {'OK' if p['ok'] else 'FAIL'} -> "
                         f"{'OK' if r['ok'] else 'FAIL'} ({r.get('detail', '')})")
    for m in prev["results"]:
        if m not in cur["results"]:
            lines.append(f"- DROPPED  {m}")
    return lines


def write_md(cur, difflines):
    ok = [m for m, r in cur["results"].items() if r["ok"]]
    bad = [m for m, r in cur["results"].items() if not r["ok"]]
    L = [f"# NIM embedders — live probe {cur['ts']}", "",
         f"> _Byline: nim-embed-probe · {cur['ts'][:10]}_", "",
         f"- `/models` listed {cur['total_listed']} models total",
         f"- endpoint: `{cur['base']}`", "",
         "## WORKING", "",
         "| model | dim | mode | batch of 4 (s) | in /models |", "|---|---|---|---|---|"]
    for m in ok:
        r = cur["results"][m]
        L.append(f"| `{m}` | {r['dim']} | {r['mode']} | {r['secs']} | {'yes' if r['listed'] else 'no'} |")
    L += ["", "## NOT WORKING", "", "| model | status | detail |", "|---|---|---|"]
    for m in bad:
        r = cur["results"][m]
        L.append(f"| `{m}` | {r['status']} | {r['detail'][:140].replace('|', '/')} |")
    L += ["", "## Changes vs previous run", ""]
    L += [f"- {d}" for d in difflines] or ["- (no previous run / no changes)"]
    with open(os.path.join(RUNS, "latest.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(L) + "\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--extra", nargs="*", default=[], help="extra model ids to probe")
    ap.add_argument("--last", action="store_true")
    ap.add_argument("--history", action="store_true")
    a = ap.parse_args()
    os.makedirs(RUNS, exist_ok=True)
    if a.last:
        p = os.path.join(RUNS, "latest.md")
        print(open(p, encoding="utf-8").read() if os.path.exists(p) else "no runs yet")
        return
    if a.history:
        for f in saved_runs():
            d = json.load(open(f, encoding="utf-8"))
            ok = sum(r["ok"] for r in d["results"].values())
            print(f"{d['ts']}  ok={ok}/{len(d['results'])}  {os.path.basename(f)}")
        return
    prev_files = saved_runs()
    cur = run(a.extra)
    difflines = diff(json.load(open(prev_files[-1], encoding="utf-8")), cur) if prev_files else []
    stamp = cur["ts"].replace(":", "").replace("+00:00", "Z").replace("+0000", "Z")
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
