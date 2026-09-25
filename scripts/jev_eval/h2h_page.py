"""Build the owner's head-to-head page: 25 reviewed chunks x every working model, with a scoreboard.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 21:40: run his reviewed chunks through every working model head-to-head.
Per chunk the owner sees his own earlier verdict/note, the chunk with its context, and each model's labels; he picks the
best model(s) and can note why (artifact db collection `h2h`, doc per chunk). The scoreboard adds an automatic check
against his earlier verdicts (GOLD below, derived only from what he said: "Jev right" = tension was there; "Opus right"
on a neutral bout = no conflict/hostility; the Bout Review verdicts on distress/hostility/affection). It is a rough guide,
not a grade: his picks on this page are the real one.
Window mode (items-h2h-window-v1, owner 22:01-22:02): each item is the reviewed bout plus 25 messages either side, all
labelled; the model decides where conversations start and end. The verdict check reads only the episodes that overlap
the reviewed messages, and the page draws where every model split the window next to the 30-minute bouts and, when a
Chonkie output dir is given, Chonkie's semantic and Slumber splits (owner 22:03 "try chonky?").
The output embeds case messages: build it on the devbox or in a scratch path, never commit it.
Usage: python h2h_page.py <out.html> <h2h dir> [<chonkie dir>]   (run from the jev-eval dir: reads bouts/)
"""

import json
import pathlib
import sys

NEGATIVE = {"edgy", "conflict", "hostile", "threat", "deflecting", "leverage", "stonewalling"}
POSITIVE = {"warm", "cooperative", "playful", "conciliatory"}
TENSION = {"edgy", "conflict", "hostile", "stonewalling", "threat", "deflecting"}
# expect: at least one of these labels; avoid: none of these at intensity >= avoid_min; who: expected shower of an expect label
GOLD = {
    "c2024-b0007": {"expect": {"upset"}, "basis": "Bout Review: Opus 'distressed' marked right"},
    "c2024-b0142": {"expect": {"hostile", "conflict"}, "basis": "Bout Review: Opus 'hostile' marked right"},
    "c2024-b0407": {"expect": {"warm"}, "basis": "Bout Review: Opus 'affectionate' marked right"},
    "c2024-b0592": {"avoid": {"hostile", "conflict"}, "basis": "Bout Review: Opus 'neutral' marked right"},
    "c2024-b0027": {"expect": {"edgy"}, "avoid": {"hostile"}, "basis": "owner: attitude was there; Jev caught too much"},
    "c2024-b0229": {"expect": {"edgy", "conflict"}, "avoid": {"hostile"}, "avoid_min": 3,
                    "basis": "owner: Opus didn't catch enough, Jev caught too much"},
    "c2024-b0211": {"expect": {"conflict", "hostile"}, "who": "Katrina", "basis": "owner: a conflict, not by him"},
}
for b in ("c2024-b0076", "c2024-b0084", "c2024-b0085", "c2024-b0140", "c2024-b0179", "c2024-b0181"):
    GOLD[b] = {"expect": TENSION, "basis": "owner: Jev right (tense)"}
for b in ("c2024-b0141", "c2024-b0180", "c2024-b0230", "c2024-b0276"):
    GOLD[b] = {"avoid": {"conflict", "hostile"}, "basis": "owner: Opus right (neutral)"}


BOUT_FILES = {"c2024": "bouts/c2024_bouts_v2.jsonl", "f2024": "bouts/f2024_bouts_v1.jsonl"}


def in_focus(e: dict, ff: int, ft: int) -> bool:
    try:
        return int(e.get("from_i")) <= ft and int(e.get("to_i")) >= ff
    except (TypeError, ValueError):
        return True


def gold_check(bout_id: str, out: dict, ff: int, ft: int) -> str | None:
    g = GOLD.get(bout_id)
    if not g or not out:
        return None
    labs = [l for e in out.get("episodes", []) if in_focus(e, ff, ft) for l in e.get("labels", [])]
    ok = True
    if g.get("expect"):
        hits = [l for l in labs if l.get("label") in g["expect"] and (not g.get("who") or l.get("who") in (g["who"], "both"))]
        ok = ok and bool(hits)
    if g.get("avoid"):
        ok = ok and not any(l.get("label") in g["avoid"] and (l.get("intensity") or 1) >= g.get("avoid_min", 1) for l in labs)
    return "pass" if ok else "miss"


def splits(out: dict | None) -> list[int] | None:
    """Window indexes where a new conversation starts (the first episode's start is not a split)."""
    if not out:
        return None
    try:
        return sorted({int(e["from_i"]) for e in out.get("episodes", [])} - {0})
    except (KeyError, TypeError, ValueError):
        return None


def agree(a: list[int], b: list[int], tol: int = 1) -> float:
    """F1 of two split sets, a split within tol messages counting as the same place; both empty = 1."""
    if not a and not b:
        return 1.0
    hit_a = sum(1 for x in a if any(abs(x - y) <= tol for y in b))
    hit_b = sum(1 for y in b if any(abs(x - y) <= tol for x in a))
    p, r = (hit_a / len(a) if a else 0), (hit_b / len(b) if b else 0)
    return 2 * p * r / (p + r) if p + r else 0.0


out_path, h2h = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
chonkie_dir = pathlib.Path(sys.argv[3]) if len(sys.argv) > 3 else None
run = json.loads((h2h / "_run.json").read_text(encoding="utf-8"))
models = [d for d in sorted(h2h.iterdir()) if d.is_dir()]
flat, first_k = {}, {}
for src, path in BOUT_FILES.items():
    # split on "\n" only: splitlines() also breaks on U+2028 etc. inside message text
    bs = [json.loads(x) for x in pathlib.Path(path).read_text(encoding="utf-8").split("\n") if x.strip()]
    bs.sort(key=lambda b: b["start_local"])
    flat[src] = [b["bout_id"] for b in bs for _ in b["messages"]]
    for k, b_id in enumerate(flat[src]):
        first_k.setdefault(b_id, (src, k))
chonkie = {}  # label -> sorted global chunk starts (semantic) or {bout_id: starts} (slumber)
if chonkie_dir:
    for f in sorted(chonkie_dir.glob("semantic_*.json")):
        r = json.loads(f.read_text(encoding="utf-8"))
        chonkie["Chonkie semantic " + f.stem.removeprefix("semantic_")] = sorted(c["first"] for c in r["chunks"])
    for f in sorted(chonkie_dir.glob("slumber_*.json")):
        r = json.loads(f.read_text(encoding="utf-8"))
        chonkie["Chonkie slumber " + f.stem.removeprefix("slumber_")] = {
            b: sorted(c["first"] for c in w.get("chunks", [])) for b, w in r["windows"].items() if w.get("chunks")}
chunks, board, cagree = [], {}, {}
for it in run["items"]:
    bid = it["bout_id"]
    tup = run["texts"][bid]
    text, n = tup[0], tup[1]
    ff, ft = (tup[2], tup[3]) if len(tup) > 3 else (0, n - 1)
    src, k0 = first_k[bid]
    base = k0 - ff  # global index of window line 0 (items_h2h.py puts WINDOW messages before the bout)
    rows = [["30-minute bouts", [j for j in range(1, n) if flat[src][base + j] != flat[src][base + j - 1]]]]
    if src == "c2024":
        for label, starts in chonkie.items():
            st = starts.get(bid) if isinstance(starts, dict) else starts
            if st is not None:
                rows.append([label, [g - base for g in st if base < g < base + n]])
    res = {}
    for d in models:
        f = d / f"{bid}.json"
        if not f.exists():
            continue
        r = json.loads(f.read_text(encoding="utf-8"))
        out = r.get("output") if r.get("ok") else None
        gc = gold_check(bid, out, ff, ft)
        res[d.name] = {"ok": bool(r.get("ok")), "s": r.get("seconds"), "err": (r.get("error") or "")[:200],
                       "problems": r.get("problems") or [], "gold": gc, "splits": splits(out),
                       "eps": [{"f": e.get("from_i"), "t": e.get("to_i"), "topic": e.get("topic", ""), "sum": e.get("summary", ""),
                                "labs": [[l.get("label"), l.get("who"), l.get("directed_at"), l.get("intensity"), l.get("note", ""),
                                          l.get("responds_to_i")] for l in e.get("labels", [])],
                                "un": [[u.get("who"), u.get("from_i"), u.get("to_i"), u.get("note", "")] for u in e.get("unanswered", [])]}
                               for e in (out or {}).get("episodes", [])]}
        b = board.setdefault(d.name, {"ok": 0, "fail": 0, "secs": [], "pass": 0, "miss": 0, "pos": 0, "neg": 0, "logistics": 0,
                                      "labels": 0, "unanswered": 0, "problems": 0})
        if r.get("ok"):
            b["ok"] += 1
            b["secs"].append(r.get("seconds") or 0)
            labs = [l for e in out.get("episodes", []) for l in e.get("labels", [])]
            b["labels"] += len(labs)
            b["pos"] += sum(1 for l in labs if l.get("label") in POSITIVE)
            b["neg"] += sum(1 for l in labs if l.get("label") in NEGATIVE)
            b["logistics"] += sum(1 for l in labs if l.get("label") == "logistics")
            b["unanswered"] += sum(len(e.get("unanswered", [])) for e in out.get("episodes", []))
            b["problems"] += 1 if r.get("problems") else 0
            if gc:
                b[gc] += 1
                if gc == "miss":
                    b.setdefault("misses", []).append(bid)
        else:
            b["fail"] += 1
    msplits = {m: v["splits"] for m, v in res.items() if v["splits"] is not None}
    for m, sp in msplits.items():
        others = [agree(sp, o) for m2, o in msplits.items() if m2 != m]
        if others:
            board[m].setdefault("agree", []).append(sum(others) / len(others))
    for label, sp in rows:
        if msplits:
            cagree.setdefault(label, []).append(sum(agree(sp, o) for o in msplits.values()) / len(msplits))
    chunks.append({"id": bid, "why": it.get("why", ""), "gold": GOLD.get(bid, {}).get("basis", ""), "n": n, "text": text,
                   "ff": ff, "ft": ft, "rows": rows, "res": res})
scores = [{"m": m, "ok": v["ok"], "fail": v["fail"], "avg_s": round(sum(v["secs"]) / len(v["secs"]), 1) if v["secs"] else None,
           "pass": v["pass"], "graded": v["pass"] + v["miss"], "pos": v["pos"], "neg": v["neg"], "logistics": v["logistics"],
           "labels": v["labels"], "unanswered": v["unanswered"], "problems": v["problems"],
           "agree": round(100 * sum(v["agree"]) / len(v["agree"])) if v.get("agree") else None,
           "misses": v.get("misses", [])} for m, v in board.items()]
# Owner 22:24 "the math doesn't math": a ratio over only the stretches a model has answered ranked a model with 5 of 17
# done above one with 16 of 17 done. Every model is now counted against all of the owner's calls, unanswered shown.
gold_ids = [it["bout_id"] for it in run["items"] if it["bout_id"] in GOLD]
for s_ in scores:
    s_["pending"] = len(gold_ids) - s_["graded"]
scores.sort(key=lambda s: (-s["pass"], s["graded"] - s["pass"], -s["ok"], s["avg_s"] or 9999))


def rule_text(g: dict) -> str:
    parts = []
    if g.get("expect"):
        parts.append("needs at least one of: " + ", ".join(sorted(g["expect"])) + (f" (shown by {g['who']})" if g.get("who") else ""))
    if g.get("avoid"):
        parts.append("must not say: " + ", ".join(sorted(g["avoid"])) + (f" at strength {g['avoid_min']}" if g.get("avoid_min") else ""))
    return "; ".join(parts)


calls = [{"id": b, "basis": GOLD[b]["basis"], "rule": rule_text(GOLD[b])} for b in gold_ids]
csplit = [{"m": k, "agree": round(100 * sum(v) / len(v)), "windows": len(v)} for k, v in cagree.items()]

TEMPLATE = r"""<title>Model Head-to-Head · 2024 Texts</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap">
<style>
/* Palette: Propria design contract 1.0.0 (same tokens as the other review pages). */
:root {
  --ground:#f5f3ee; --panel:#fffefb; --ink:#1d2228; --muted:#687078; --line:#d5d1c9; --soft:#ebe8e0; --focus:#4051b9;
  --accent:#4051b9; --accent-ink:#fff; --ok:#247047; --ok-bg:#e2efe6; --bad:#b5433b; --bad-bg:#f8e5e3;
  --matt:#376f72; --kat:#8a4f7d; --pos:#2f7d4f; --pos-bg:#e3f1e8; --neg:#b5433b; --neg-bg:#f8e5e3; --oth:#5b5f9e; --oth-bg:#e8e9f6;
}
@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --accent-ink:#111820; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a;
  --matt:#82bdc0; --kat:#d69cc6; --pos:#7cc79a; --pos-bg:#1f3a2b; --neg:#ef7b6e; --neg-bg:#43282a; --oth:#aeb2f0; --oth-bg:#2e3150; } }
:root[data-theme="dark"] {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --accent-ink:#111820; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a;
  --matt:#82bdc0; --kat:#d69cc6; --pos:#7cc79a; --pos-bg:#1f3a2b; --neg:#ef7b6e; --neg-bg:#43282a; --oth:#aeb2f0; --oth-bg:#2e3150; }
* { box-sizing:border-box; }
body { margin:0; background:var(--ground); color:var(--ink); font:15px/1.5 "Instrument Sans", "Segoe UI", system-ui, sans-serif; padding-inline:16px; padding-block:20px 80px; }
.wrap { max-width:1150px; margin:0 auto; display:grid; gap:14px; }
h1 { font-size:24px; margin:0; letter-spacing:-.01em; text-wrap:balance; } h2 { font-size:16px; margin:0; }
.lede { color:var(--muted); margin:4px 0 0; max-width:74ch; }
.mono { font-family:"IBM Plex Mono", ui-monospace, Consolas, monospace; font-size:12.5px; font-variant-numeric:tabular-nums; }
.panel { background:var(--panel); border:1px solid var(--line); border-radius:6px; padding:12px 14px; }
.tablewrap { overflow-x:auto; }
table { border-collapse:collapse; width:100%; font-size:13px; }
th, td { text-align:left; padding:4px 8px; border-bottom:1px solid var(--line); white-space:nowrap; font-variant-numeric:tabular-nums; }
th { font-weight:600; color:var(--muted); }
.bar { position:sticky; top:env(safe-area-inset-top, 0px); z-index:5; background:var(--ground); padding-block:10px; display:flex; flex-wrap:wrap; gap:8px; align-items:center; border-bottom:1px solid var(--line); }
select, textarea { font:inherit; font-size:14px; color:var(--ink); background:var(--panel); border:1px solid var(--line); border-radius:4px; padding:6px 8px; }
.save { font-size:13px; color:var(--muted); } .save.err { color:var(--bad); }
:focus-visible { outline:2px solid var(--focus); outline-offset:2px; }
.chunk { display:grid; gap:10px; }
.why { font-size:14px; } .gold { font-size:13px; color:var(--muted); }
pre.ctx { white-space:pre-wrap; overflow-wrap:anywhere; margin:0; font:13px/1.5 "IBM Plex Mono", ui-monospace, monospace; background:var(--soft); border-radius:4px; padding:10px; max-height:420px; overflow:auto; }
.models { display:grid; grid-template-columns:repeat(auto-fill, minmax(330px, 1fr)); gap:10px; }
.m { border:1px solid var(--line); border-radius:6px; padding:10px; display:grid; gap:6px; background:var(--panel); align-content:start; }
.m.pick { border-color:var(--ok); box-shadow:inset 0 0 0 1px var(--ok); }
.mh { display:flex; flex-wrap:wrap; gap:6px 10px; align-items:baseline; }
.mh .n { font-weight:700; font-size:13.5px; overflow-wrap:anywhere; }
.tag { font-size:11.5px; font-weight:700; padding:1px 6px; border-radius:3px; }
.tag.pass { background:var(--ok-bg); color:var(--ok); } .tag.miss { background:var(--bad-bg); color:var(--bad); } .tag.fail { background:var(--soft); color:var(--muted); }
.ep { font-size:13px; display:grid; gap:4px; border-top:1px solid var(--line); padding-top:6px; }
.ep .t { font-weight:600; }
.labs { display:flex; flex-wrap:wrap; gap:4px; }
.lab { font-size:12px; padding:2px 6px; border-radius:3px; background:var(--oth-bg); color:var(--oth); }
.lab.pos { background:var(--pos-bg); color:var(--pos); } .lab.neg { background:var(--neg-bg); color:var(--neg); }
.un { font-size:12px; color:var(--bad); }
.pickbtn { font:inherit; font-size:12.5px; font-weight:600; border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:4px; padding:3px 8px; cursor:pointer; justify-self:start; }
.pickbtn.on { background:var(--ok-bg); color:var(--ok); border-color:var(--ok); }
.note textarea { width:100%; min-height:38px; }
.err { font-size:12.5px; color:var(--muted); overflow-wrap:anywhere; }
.ctx .fl { background:var(--oth-bg); box-shadow:inset 3px 0 0 var(--accent); display:block; }
.strips { display:grid; gap:3px; }
.strip { display:grid; grid-template-columns:minmax(120px, 210px) 1fr; gap:8px; align-items:center; font-size:12px; }
.strip .lbl { color:var(--muted); overflow-wrap:anywhere; }
.cells { display:flex; height:14px; border:1px solid var(--line); border-radius:3px; overflow:hidden; }
.cells span { flex:1; min-width:1px; }
.cells span.a { background:var(--oth-bg); } .cells span.b { background:var(--pos-bg); }
.cells span.f { box-shadow:inset 0 -3px 0 var(--accent); }
</style>
<div class="wrap">
  <header>
    <h1>Model Head-to-Head · 2024 Texts</h1>
    <p class="lede">25 stretches of the 2024 texts around chunks you had already reviewed. Each model got the whole stretch (your chunk plus about 25 messages either side), decided for itself where each conversation starts and ends, and labelled every message. Your reviewed messages are highlighted. The strips show where each model, the old 30-minute bouts and Chonkie split the stretch. Pick the best model for each stretch; the automatic check only compares against your earlier verdicts, on the conversations that include your reviewed messages.</p>
  </header>
  <section class="panel"><h2>Scoreboard</h2>
    <div class="tablewrap"><table id="board"><thead><tr><th>Model</th><th>Answered</th><th>Failed</th><th>Avg s</th><th>Your calls: matched</th><th>missed</th><th>not answered yet</th><th>Positive labels</th><th>Negative labels</th><th>"Logistics"</th><th>Unanswered runs</th><th>Structure warnings</th><th>Splits agree with other models</th><th>Your picks</th></tr></thead><tbody></tbody></table></div>
    <details style="margin-top:10px"><summary id="calls-sum"></summary><div class="tablewrap"><table id="calls"><thead><tr><th>Stretch</th><th>Your earlier call</th><th>A model matches when its labels on your messages…</th></tr></thead><tbody></tbody></table></div></details>
    <h2 style="margin-top:12px">Splitters without labels</h2>
    <div class="tablewrap"><table id="cboard"><thead><tr><th>Splitter</th><th>Stretches</th><th>Agrees with the models on where conversations split</th></tr></thead><tbody></tbody></table></div>
  </section>
  <div class="bar"><select id="pick-chunk" aria-label="Chunk"></select><select id="filter" aria-label="Show models"><option value="ok">Models that answered</option><option value="all">All models</option></select><span class="save" id="save-state">Connecting…</span></div>
  <div id="chunk" class="wrap"></div>
</div>
<script>
const CHUNKS = __CHUNKS__;
const SCORES = __SCORES__;
const CSPLIT = __CSPLIT__;
const CALLS = __CALLS__;
const POS = new Set(__POS__), NEG = new Set(__NEG__);
const $ = id => document.getElementById(id);
const el = (t, c, x) => { const e = document.createElement(t); if (c) e.className = c; if (x != null) e.textContent = x; return e; };
const short = m => m.replace(/^(gemini|gemma|nim|openrouter|claude)_/, "").replace(/_free$/, " (free)").replace(/_/g, "/");
let cur = 0, db = null; const picks = {};
function board() {
  const tb = $("board").querySelector("tbody"); tb.replaceChildren();
  const counts = {}; for (const k in picks) for (const m of picks[k].models || []) counts[m] = (counts[m] || 0) + 1;
  for (const s of SCORES) {
    const tr = el("tr");
    const cells = [short(s.m), s.ok, s.fail, s.avg_s ?? "–", s.pass, s.graded - s.pass, s.pending, s.pos, s.neg, s.logistics, s.unanswered, s.problems, s.agree != null ? s.agree + "%" : "–", counts[s.m] || 0];
    cells.forEach((v, i) => { const td = el("td", i === 0 ? "mono" : null, String(v)); tr.append(td); });
    tb.append(tr);
  }
  $("calls-sum").textContent = CALLS.length + " of the " + CHUNKS.length + " stretches have an earlier call of yours the page can check (the other " + (CHUNKS.length - CALLS.length) + " had notes but no clear call). Each model is counted against all " + CALLS.length + ": matched + missed + not answered yet = " + CALLS.length + ". Show them";
  const ct = $("calls").querySelector("tbody"); ct.replaceChildren();
  for (const c of CALLS) { const tr = el("tr"); [c.id, c.basis, c.rule].forEach(v => tr.append(el("td", null, v))); ct.append(tr); }
  const cb = $("cboard").querySelector("tbody"); cb.replaceChildren();
  for (const c of CSPLIT) { const tr = el("tr"); [c.m, c.windows, c.agree + "%"].forEach(v => tr.append(el("td", null, String(v)))); cb.append(tr); }
}
function strip(label, sp, n, ff, ft) {
  const row = el("div", "strip"); row.append(el("span", "lbl", label));
  const cells = el("div", "cells"); const set = new Set(sp || []); let seg = 0;
  for (let j = 0; j < n; j++) { if (set.has(j)) seg++; const c = el("span", (seg % 2 ? "b" : "a") + (j >= ff && j <= ft ? " f" : "")); c.title = "message " + j; cells.append(c); }
  row.append(cells); return row;
}
function render() {
  const c = CHUNKS[cur]; const box = $("chunk"); box.replaceChildren();
  const p = el("section", "panel chunk");
  p.append(el("h2", null, c.id + " · " + c.n + " messages in the stretch, " + (c.ft - c.ff + 1) + " of them yours (highlighted)"), el("div", "why", "Why it's here: " + c.why));
  if (c.gold) p.append(el("div", "gold", "Automatic check uses: " + c.gold));
  const pre = el("pre", "ctx");
  for (const line of c.text.split("\n")) {
    const m = line.match(/^\[(\d+)\]/); const i = m ? +m[1] : -1;
    pre.append(el("span", i >= c.ff && i <= c.ft ? "fl" : null, line + "\n"));
  }
  p.append(pre);
  const strips = el("div", "strips");
  for (const [label, sp] of c.rows) strips.append(strip(label, sp, c.n, c.ff, c.ft));
  for (const s of SCORES) { const r = c.res[s.m]; if (r && r.ok && r.splits) strips.append(strip(short(s.m), r.splits, c.n, c.ff, c.ft)); }
  p.append(el("div", "gold", "Where each one split this stretch (a colour change = a new conversation; underlined = your reviewed messages):"), strips);
  const note = el("div", "note"); const ta = el("textarea"); ta.id = "note-" + c.id; ta.placeholder = "What did the best ones get right? What did they all miss? (optional)";
  ta.value = (picks[c.id] || {}).note || ""; ta.addEventListener("input", () => save(c.id, { note: ta.value }, true)); note.append(ta); p.append(note);
  box.append(p);
  const grid = el("div", "models"); const show = $("filter").value;
  const mine = new Set((picks[c.id] || {}).models || []);
  for (const s of SCORES) {
    const r = c.res[s.m]; if (!r) continue; if (show === "ok" && !r.ok) continue;
    const card = el("article", "m" + (mine.has(s.m) ? " pick" : ""));
    const h = el("div", "mh"); h.append(el("span", "n", short(s.m)));
    if (!r.ok) h.append(el("span", "tag fail", "failed")); else if (r.gold) h.append(el("span", "tag " + r.gold, r.gold === "pass" ? "matches your verdict" : "misses your verdict"));
    if (r.s != null) h.append(el("span", "mono", r.s + " s"));
    card.append(h);
    if (!r.ok) card.append(el("div", "err", r.err));
    for (const e of r.eps) {
      const ep = el("div", "ep"); ep.append(el("span", "t", "[" + e.f + "–" + e.t + "] " + e.topic));
      if (e.sum) ep.append(el("span", null, e.sum));
      const labs = el("div", "labs");
      for (const [lab, who, at, inten, note, rt] of e.labs) {
        const s2 = el("span", "lab " + (POS.has(lab) ? "pos" : NEG.has(lab) ? "neg" : ""), lab + " · " + who + "→" + at + " " + "●".repeat(Math.max(1, Math.min(3, inten || 1))) + (rt != null && rt !== -1 ? " · reacts to " + rt : ""));
        s2.title = note || ""; labs.append(s2);
      }
      ep.append(labs);
      for (const [who, f, t, n2] of e.un) ep.append(el("span", "un", "Unanswered: " + who + " [" + f + "–" + t + "] " + n2));
      card.append(ep);
    }
    if (r.ok) {
      const b = el("button", "pickbtn" + (mine.has(s.m) ? " on" : ""), mine.has(s.m) ? "Picked as best" : "Pick as best"); b.type = "button"; b.id = "pick-" + c.id + "-" + s.m;
      b.setAttribute("aria-pressed", mine.has(s.m) ? "true" : "false");
      b.addEventListener("click", () => { const set = new Set((picks[c.id] || {}).models || []); set.has(s.m) ? set.delete(s.m) : set.add(s.m); save(c.id, { models: [...set] }); });
      card.append(b);
    }
    grid.append(card);
  }
  box.append(grid);
}
const timers = {}, queued = {}, inflight = {};
function setSave(t, err) { const s = $("save-state"); s.textContent = t; s.classList.toggle("err", !!err); }
function save(id, patch, quiet) {
  picks[id] = Object.assign({ bout_id: id, models: [], note: "" }, picks[id] || {}, patch, { updated_at: new Date().toISOString() });
  if (!quiet) { render(); board(); }
  queued[id] = picks[id]; clearTimeout(timers[id]); timers[id] = setTimeout(() => flush(id), 600); setSave("Saving…");
}
async function flush(id) {
  if (!db) { setSave("Not saved: storage unavailable in this view", true); return; }
  if (inflight[id]) return; const payload = queued[id]; delete queued[id]; if (!payload) return; inflight[id] = true;
  try { await db.doc("h2h/" + id).set(payload); setSave("Saved " + new Date().toLocaleTimeString()); }
  catch (e) { setSave("Not saved (" + (e && e.code || "error") + "). Your answer is still on screen.", true); }
  finally { inflight[id] = false; if (queued[id]) flush(id); }
}
async function connect() {
  db = await window.claude?.use?.("db") ?? null;
  if (!db) { setSave("Saving unavailable in this view: answers stay on screen only", true); return; }
  setSave("Connected");
  db.collection("h2h").onSnapshot(qs => {
    for (const d of qs.docs) { if (!queued[d.id] && !inflight[d.id]) picks[d.id] = d.data(); }
    board(); if (document.activeElement?.tagName !== "TEXTAREA") render();
  }, e => setSave("Live sync stopped (" + (e && e.code || "error") + "); reload to reconnect", true));
}
CHUNKS.forEach((c, i) => { const o = el("option", null, (i + 1) + ". " + c.id + " — " + c.why.slice(0, 70)); o.value = String(i); $("pick-chunk").append(o); });
$("pick-chunk").addEventListener("change", () => { cur = +$("pick-chunk").value; render(); });
$("filter").addEventListener("change", render);
board(); render(); connect();
</script>
"""

page = (TEMPLATE.replace("__CHUNKS__", json.dumps(chunks, ensure_ascii=False)).replace("__SCORES__", json.dumps(scores))
        .replace("__CSPLIT__", json.dumps(csplit)).replace("__CALLS__", json.dumps(calls, ensure_ascii=False))
        .replace("__POS__", json.dumps(sorted(POSITIVE))).replace("__NEG__", json.dumps(sorted(NEGATIVE))))
out_path.write_text(page, encoding="utf-8")
print(f"wrote {out_path}: {len(chunks)} chunks x {len(models)} models, {len(page) // 1024} KB")
for s in scores:
    print(f"  {s['m']:<50} ok={s['ok']:>2} fail={s['fail']:>2} avg_s={s['avg_s']} verdicts={s['pass']}/{s['graded']} pos={s['pos']} neg={s['neg']} logistics={s['logistics']} unanswered={s['unanswered']} agree={s['agree']}")
for c in csplit:
    print(f"  {c['m']:<50} windows={c['windows']} agree_with_models={c['agree']}%")
