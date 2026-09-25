"""Build the owner's head-to-head page: his reviewed stretches (plus clearly hostile ones) x every working model.

Byline: Claude Code · Opus 5.5 · 2026-09-24/25. Owner 21:40: run his reviewed chunks through every working model head-to-head.
Per stretch the owner sees his own review, the whole stretch with his reviewed messages highlighted, where each model split
it into conversations, and each model's labels; he picks the best model(s) and can note why (artifact db collection h2h).
~~The scoreboard adds an automatic check against his earlier verdicts (GOLD).~~ Removed 2026-09-25 (owner 22:24-00:29: "the
math doesn't math"; the check was a keyword rule on 17 of 25 stretches and a ratio over unfinished runs). The scoreboard
now only counts: done out of N, failed, not run yet, speed, and his picks; a model still running is shown as such.
Window mode: each item is the reviewed bout plus 25 messages either side, all labelled; the model decides where
conversations start and end (prompt v3). Strips show every model's splits next to the old 30-minute rule and Chonkie.
Items may carry owner_note (his own words from the review pages) and group ("reviewed" or e.g. "clearly hostile").
The output embeds case messages: build it on the devbox or in a scratch path, never commit it.
Usage: python h2h_page.py <out.html> <h2h dir> [<chonkie dir>]   (run from the jev-eval dir: reads bouts/)
"""

import json
import pathlib
import sys

NEGATIVE = {"edgy", "conflict", "hostile", "threat", "deflecting", "leverage", "stonewalling"}
POSITIVE = {"warm", "cooperative", "playful", "conciliatory"}
BOUT_FILES = {"c2024": "bouts/c2024_bouts_v2.jsonl", "f2024": "bouts/f2024_bouts_v1.jsonl"}
CHONKIE_SHOWN = ("semantic_t65_w3",)  # one meaning-based setting on the strips; all four are in the chonkie dir


def splits(out: dict | None) -> list[int] | None:
    """Window indexes where a new conversation starts (the first episode's start is not a split)."""
    if not out:
        return None
    try:
        return sorted({int(e["from_i"]) for e in out.get("episodes", [])} - {0})
    except (KeyError, TypeError, ValueError):
        return None


out_path, h2h = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
chonkie_dir = pathlib.Path(sys.argv[3]) if len(sys.argv) > 3 else None
run = json.loads((h2h / "_run.json").read_text(encoding="utf-8"))
models = [d for d in sorted(h2h.iterdir()) if d.is_dir()]
n_items = len(run["items"])
flat, first_k = {}, {}
for src, path in BOUT_FILES.items():
    # split on "\n" only: splitlines() also breaks on U+2028 etc. inside message text
    bs = [json.loads(x) for x in pathlib.Path(path).read_text(encoding="utf-8").split("\n") if x.strip()]
    bs.sort(key=lambda b: b["start_local"])
    flat[src] = [b["bout_id"] for b in bs for _ in b["messages"]]
    for k, b_id in enumerate(flat[src]):
        first_k.setdefault(b_id, (src, k))
chonkie = {}  # label -> sorted global chunk starts (whole stream) or {bout_id: starts} (per window)
if chonkie_dir:
    for f in sorted(chonkie_dir.glob("semantic_*.json")):
        if f.stem in CHONKIE_SHOWN:
            r = json.loads(f.read_text(encoding="utf-8"))
            chonkie["Chonkie meaning-based"] = sorted(c["first"] for c in r["chunks"])
    for f in sorted(chonkie_dir.glob("*_windows_*.json")) + sorted(chonkie_dir.glob("slumber_*.json")):
        r = json.loads(f.read_text(encoding="utf-8"))
        chonkie[r.get("label") or ("Chonkie " + f.stem)] = {
            b: sorted(c["first"] for c in w.get("chunks", [])) for b, w in r["windows"].items() if w.get("chunks")}
chunks, board = [], {}
for it in run["items"]:
    bid = it["bout_id"]
    key = it.get("id") or bid  # an item may target part of a bout (hostile picks)
    tup = run["texts"][key]
    text, n = tup[0], tup[1]
    ff, ft = (tup[2], tup[3]) if len(tup) > 3 else (0, n - 1)
    src, k0 = first_k[bid]
    base = tup[4] if len(tup) > 4 else k0 - ff  # stream index of window line 0
    rows = [["Old 30-minute rule", [j for j in range(1, n) if flat[src][base + j] != flat[src][base + j - 1]]]]
    if src == "c2024":
        for label, starts in chonkie.items():
            st = starts.get(key) if isinstance(starts, dict) else starts
            if st is not None:
                rows.append([label, [g - base for g in st if base < g < base + n]])
    res = {}
    for d in models:
        b = board.setdefault(d.name, {"ok": 0, "fail": 0, "secs": []})
        f = d / f"{key}.json"
        if not f.exists():
            continue
        r = json.loads(f.read_text(encoding="utf-8"))
        out = r.get("output") if r.get("ok") else None
        res[d.name] = {"ok": bool(r.get("ok")), "s": r.get("seconds"), "err": (r.get("error") or "")[:200],
                       "splits": splits(out),
                       "eps": [{"f": e.get("from_i"), "t": e.get("to_i"), "topic": e.get("topic", ""), "sum": e.get("summary", ""),
                                "labs": [[l.get("label"), l.get("who"), l.get("directed_at"), l.get("intensity"), l.get("note", ""),
                                          l.get("responds_to_i")] for l in e.get("labels", [])],
                                "un": [[u.get("who"), u.get("from_i"), u.get("to_i"), u.get("note", "")] for u in e.get("unanswered", [])]}
                               for e in (out or {}).get("episodes", [])]}
        if r.get("ok"):
            b["ok"] += 1
            b["secs"].append(r.get("seconds") or 0)
        else:
            b["fail"] += 1
    chunks.append({"id": key, "group": it.get("group", "reviewed"), "review": it.get("owner_note") or "",
                   "why": it.get("why", ""), "n": n, "text": text, "ff": ff, "ft": ft, "rows": rows, "res": res})
scores = [{"m": m, "ok": v["ok"], "fail": v["fail"], "todo": n_items - v["ok"] - v["fail"],
           "avg_s": round(sum(v["secs"]) / len(v["secs"]), 1) if v["secs"] else None} for m, v in board.items()]
scores.sort(key=lambda s: (-s["ok"], s["fail"], s["avg_s"] or 9999))
running = [s["m"] for s in scores if s["todo"]]

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
select { max-width:100%; }
.save { font-size:13px; color:var(--muted); } .save.err { color:var(--bad); }
:focus-visible { outline:2px solid var(--focus); outline-offset:2px; }
.chunk { display:grid; gap:10px; }
.why { font-size:14px; } .gold { font-size:13px; color:var(--muted); }
pre.ctx { white-space:pre-wrap; overflow-wrap:anywhere; margin:0; font:13px/1.5 "IBM Plex Mono", ui-monospace, monospace; background:var(--soft); border-radius:4px; padding:10px; max-height:420px; overflow:auto; }
.models { display:grid; grid-template-columns:repeat(auto-fill, minmax(min(330px, 100%), 1fr)); gap:10px; }
.m { border:1px solid var(--line); border-radius:6px; padding:10px; display:grid; gap:6px; background:var(--panel); align-content:start; }
.m.pick { border-color:var(--ok); box-shadow:inset 0 0 0 1px var(--ok); }
.mh { display:flex; flex-wrap:wrap; gap:6px 10px; align-items:baseline; }
.mh .n { font-weight:700; font-size:13.5px; overflow-wrap:anywhere; }
.tag { font-size:11.5px; font-weight:700; padding:1px 6px; border-radius:3px; }
.tag.fail { background:var(--soft); color:var(--muted); }
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
.strip { display:grid; grid-template-columns:minmax(110px, 210px) 1fr; gap:8px; align-items:center; font-size:12px; }
.strip .lbl { color:var(--muted); overflow-wrap:anywhere; }
.cells { display:flex; height:14px; border:1px solid var(--line); border-radius:3px; overflow:hidden; }
.cells span { flex:1; min-width:1px; }
.cells span.a { background:var(--oth-bg); } .cells span.b { background:var(--pos-bg); }
.cells span.f { box-shadow:inset 0 -3px 0 var(--accent); }
</style>
<div class="wrap">
  <header>
    <h1>Model Head-to-Head · 2024 Texts</h1>
    <p class="lede">Stretches of the 2024 texts: the chunks you reviewed, plus clearly hostile ones. Each model got the whole stretch (the chunk plus about 25 messages either side), decided where each conversation starts and ends, and labelled every message. Your messages from the review are highlighted. Pick the best model for each stretch; your picks are the only score.</p>
  </header>
  <section class="panel"><h2>Scoreboard</h2>
    <p class="gold" id="run-state"></p>
    <div class="tablewrap"><table id="board"><thead><tr><th>Model</th><th>Done</th><th>Failed</th><th>Not run yet</th><th>Seconds per stretch</th><th>Your picks</th></tr></thead><tbody></tbody></table></div>
  </section>
  <div class="bar"><select id="pick-chunk" aria-label="Stretch"></select><select id="filter" aria-label="Show models"><option value="ok">Models that answered</option><option value="all">All models</option></select><span class="save" id="save-state">Connecting…</span></div>
  <div id="chunk" class="wrap"></div>
</div>
<script>
const CHUNKS = __CHUNKS__;
const SCORES = __SCORES__;
const RUNNING = __RUNNING__;
const POS = new Set(__POS__), NEG = new Set(__NEG__);
const $ = id => document.getElementById(id);
const el = (t, c, x) => { const e = document.createElement(t); if (c) e.className = c; if (x != null) e.textContent = x; return e; };
const short = m => m.replace(/^(gemini|gemma|nim|openrouter-plain|openrouter|claude)_/, "").replace(/_free$/, " (free)").replace(/_/g, "/");
let cur = 0, db = null; const picks = {};
function board() {
  const tb = $("board").querySelector("tbody"); tb.replaceChildren();
  const counts = {}; for (const k in picks) for (const m of picks[k].models || []) counts[m] = (counts[m] || 0) + 1;
  for (const s of SCORES) {
    const tr = el("tr");
    const cells = [short(s.m), s.ok + " of " + CHUNKS.length, s.fail, s.todo, s.avg_s ?? "–", counts[s.m] || 0];
    cells.forEach((v, i) => { const td = el("td", i === 0 ? "mono" : null, String(v)); tr.append(td); });
    tb.append(tr);
  }
  $("run-state").textContent = RUNNING.length ? "Still running: " + RUNNING.length + " model(s) have stretches not run yet." : "Finished: every model has an answer or a failure for every stretch.";
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
  p.append(el("h2", null, c.id + " · " + c.n + " messages · " + (c.group === "reviewed" ? "you reviewed this" : c.group)));
  if (c.review) p.append(el("div", "why", "Your review: " + c.review));
  if (c.why) p.append(el("div", "gold", "Why it's in the test: " + c.why));
  const pre = el("pre", "ctx");
  for (const line of c.text.split("\n")) {
    const m = line.match(/^\[(\d+)\]/); const i = m ? +m[1] : -1;
    pre.append(el("span", i >= c.ff && i <= c.ft ? "fl" : null, line + "\n"));
  }
  p.append(pre);
  const strips = el("div", "strips");
  for (const [label, sp] of c.rows) strips.append(strip(label, sp, c.n, c.ff, c.ft));
  for (const s of SCORES) { const r = c.res[s.m]; if (r && r.ok && r.splits) strips.append(strip(short(s.m), r.splits, c.n, c.ff, c.ft)); }
  p.append(el("div", "gold", "Where each one split this stretch into conversations (a colour change = a new conversation; underlined = your reviewed messages):"), strips);
  const note = el("div", "note"); const ta = el("textarea"); ta.id = "note-" + c.id; ta.placeholder = "What did the best ones get right? What did they all miss? (optional)";
  ta.value = (picks[c.id] || {}).note || ""; ta.addEventListener("input", () => save(c.id, { note: ta.value }, true)); note.append(ta); p.append(note);
  box.append(p);
  const grid = el("div", "models"); const show = $("filter").value;
  const mine = new Set((picks[c.id] || {}).models || []);
  for (const s of SCORES) {
    const r = c.res[s.m]; if (!r) continue; if (show === "ok" && !r.ok) continue;
    const card = el("article", "m" + (mine.has(s.m) ? " pick" : ""));
    const h = el("div", "mh"); h.append(el("span", "n", short(s.m)));
    if (!r.ok) h.append(el("span", "tag fail", "failed"));
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
CHUNKS.forEach((c, i) => { const o = el("option", null, (i + 1) + ". " + (c.group === "reviewed" ? "" : "[" + c.group + "] ") + c.id + " — " + (c.review || c.why).slice(0, 70)); o.value = String(i); $("pick-chunk").append(o); });
$("pick-chunk").addEventListener("change", () => { cur = +$("pick-chunk").value; render(); });
$("filter").addEventListener("change", render);
board(); render(); connect();
</script>
"""

page = (TEMPLATE.replace("__CHUNKS__", json.dumps(chunks, ensure_ascii=False)).replace("__SCORES__", json.dumps(scores))
        .replace("__RUNNING__", json.dumps(running))
        .replace("__POS__", json.dumps(sorted(POSITIVE))).replace("__NEG__", json.dumps(sorted(NEGATIVE))))
out_path.write_text(page, encoding="utf-8")
print(f"wrote {out_path}: {len(chunks)} stretches x {len(models)} models, {len(page) // 1024} KB; still running: {len(running)}")
for s in scores:
    print(f"  {s['m']:<50} done={s['ok']:>2}/{n_items} failed={s['fail']:>2} not_run={s['todo']:>2} avg_s={s['avg_s']}")
