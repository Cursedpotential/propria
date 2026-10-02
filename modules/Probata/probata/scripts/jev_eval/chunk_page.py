"""Owner review page for the chunking comparison: 25 stretches x every Chonkie chunker, break points side by side.

Byline: Claude Code · Sonnet · 2026-10-02. Companion of chunk_all.py (which writes <dir>/<method>.json) and h2h_page.py (same
stretches, same pick-saving approach: artifact db collection, here "chunks"; one doc per stretch:
{bout_id, models: [method ids he marked as right], note, updated_at}).
The output embeds case messages: build it on the devbox or in a scratch path, never commit it.
Usage (from the jev-eval dir, which holds bouts/ and raw/): python code/chunk_page.py <out.html> raw/chunk_all_v1
"""

import json
import pathlib
import sys

BOUT_FILES = {"c2024": "bouts/c2024_bouts_v2.jsonl", "f2024": "bouts/f2024_bouts_v1.jsonl"}
H2H = "raw/h2h_v2/_run.json"
# (method id, short code, family) in display order
ORDER = [("token_1000", "T1", "size"), ("token_2500", "T2", "size"), ("fast_1000", "F1", "size"), ("fast_2500", "F2", "size"),
         ("sentence_1000", "S1", "size"), ("sentence_2500", "S2", "size"), ("recursive_1000", "R1", "size"),
         ("recursive_2500", "R2", "size"), ("semantic_t65_w3", "M1", "sem"), ("semantic_t50_w3", "M2", "sem"),
         ("semantic_t50_w8_min1", "M3", "sem"), ("late_256", "L", "late"), ("neural_distilbert", "N1", "neural"),
         ("neural_modernbert", "N2", "neural"), ("slumber_kimi_k3", "K", "llm")]
MSG_PREFIX = None

out_path, d = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
run = json.loads(pathlib.Path(H2H).read_text(encoding="utf-8"))
flat, first_k = {}, {}
for src, path in BOUT_FILES.items():
    bs = [json.loads(x) for x in pathlib.Path(path).read_text(encoding="utf-8").split("\n") if x.strip()]
    bs.sort(key=lambda b: b["start_local"])
    flat[src] = [b["bout_id"] for b in bs for _ in b["messages"]]
    for k, b_id in enumerate(flat[src]):
        first_k.setdefault(b_id, (src, k))

methods, recs = [], {}
for mid, code, fam in ORDER:
    f = d / f"{mid}.json"
    if not f.exists():
        continue
    r = json.loads(f.read_text(encoding="utf-8"))
    recs[mid] = r
    t = r["totals"]
    methods.append({"id": mid, "code": code, "fam": fam, "label": r["label"], "chunker": r["chunker"],
                    "settings": r["settings"], "n_chunks": t["n_chunks"], "median": t["median_msgs"],
                    "failed": t["windows_failed"], "calls": r.get("llm_calls")})

stretches = []
for it in run["items"]:
    key = it.get("id") or it["bout_id"]
    tup = run["texts"][key]
    text, n, ff, ft = tup[0], tup[1], tup[2], tup[3]
    src, k0 = first_k[it["bout_id"]]
    base = k0 - ff
    old = [j for j in range(1, n) if flat[src][base + j] != flat[src][base + j - 1]]
    lines, gap = [], ""
    for ln in text.split("\n"):
        if ln.startswith("[") and ln[1:2].isdigit():
            j = int(ln[1:ln.index("]")])
            lines.append({"j": j, "t": ln[ln.index("]") + 2:], "gap": gap})
            gap = ""
        else:
            gap = ln.strip("[]— ")
    rows = {"old": old}
    stats = {}
    for m in methods:
        w = recs[m["id"]]["windows"].get(key, {})
        if w.get("chunks"):
            ch = w["chunks"]
            rows[m["id"]] = [c["first"] for c in ch if c["first"] > 0]
            sz = sorted(c["n"] for c in ch)
            stats[m["id"]] = [len(ch), sz[len(sz) // 2]]
        else:
            rows[m["id"]] = None
            stats[m["id"]] = [0, 0, (w.get("error") or "not run")[:160]]
    stretches.append({"id": key, "group": it.get("group", "reviewed"), "review": it.get("owner_note") or "",
                      "why": it.get("why", ""), "n": n, "ff": ff, "ft": ft, "lines": lines, "rows": rows, "stats": stats})

TEMPLATE = r"""<title>Chunk Method Review</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap">
<style>
/* Layout: one stretch at a time. Messages with break markers on top, one row per method below with its own strip and a
   "got it right" button. Palette: Propria design contract 1.0.0 tokens, plus one hue per chunker family. */
:root {
  --ground:#f5f3ee; --panel:#fffefb; --ink:#1d2228; --muted:#687078; --line:#d5d1c9; --soft:#ebe8e0; --focus:#4051b9;
  --accent:#4051b9; --ok:#247047; --ok-bg:#e2efe6; --bad:#b5433b; --matt:#376f72; --kat:#8a4f7d;
  --f-size:#5a6a78; --f-sem:#2f7d4f; --f-late:#a8701d; --f-neural:#6b4fb0; --f-llm:#b5433b; --f-old:#8a8f94;
  --fs:#5a6a781f; --fsem:#2f7d4f1f; --flate:#a8701d1f; --fneural:#6b4fb01f; --fllm:#b5433b1f;
}
@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --matt:#82bdc0; --kat:#d69cc6;
  --f-size:#9fb0bf; --f-sem:#7cc79a; --f-late:#e0a85a; --f-neural:#b9a2f0; --f-llm:#ef8c82; --f-old:#8f969b; } }
:root[data-theme="dark"] {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --matt:#82bdc0; --kat:#d69cc6;
  --f-size:#9fb0bf; --f-sem:#7cc79a; --f-late:#e0a85a; --f-neural:#b9a2f0; --f-llm:#ef8c82; --f-old:#8f969b; }
* { box-sizing:border-box; }
body { margin:0; background:var(--ground); color:var(--ink); font:15px/1.5 "Instrument Sans","Segoe UI",system-ui,sans-serif; padding-inline:16px; padding-block:20px 80px; }
.wrap { max-width:1100px; margin:0 auto; display:grid; gap:14px; min-width:0; }
h1 { font-size:24px; margin:0; letter-spacing:-.01em; text-wrap:balance; } h2 { font-size:16px; margin:0; }
.lede { color:var(--muted); margin:4px 0 0; max-width:74ch; }
.mono { font-family:"IBM Plex Mono",ui-monospace,Consolas,monospace; font-size:12.5px; font-variant-numeric:tabular-nums; }
.panel { background:var(--panel); border:1px solid var(--line); border-radius:6px; padding:12px 14px; min-width:0; }
.tablewrap { overflow-x:auto; }
table { border-collapse:collapse; width:100%; font-size:13px; }
th, td { text-align:left; padding:4px 8px; border-bottom:1px solid var(--line); white-space:nowrap; font-variant-numeric:tabular-nums; }
th { font-weight:600; color:var(--muted); }
.bar { position:sticky; top:env(safe-area-inset-top,0px); z-index:5; background:var(--ground); padding-block:10px; display:flex; flex-wrap:wrap; gap:8px; align-items:center; border-bottom:1px solid var(--line); }
select, textarea { font:inherit; font-size:14px; color:var(--ink); background:var(--panel); border:1px solid var(--line); border-radius:4px; padding:6px 8px; max-width:100%; }
.save { font-size:13px; color:var(--muted); } .save.err { color:var(--bad); }
:focus-visible { outline:2px solid var(--focus); outline-offset:2px; }
.why { font-size:14px; } .gold { font-size:13px; color:var(--muted); }
.msgs { margin:0; font:13px/1.45 "IBM Plex Mono",ui-monospace,monospace; background:var(--soft); border-radius:4px; padding:8px 10px; max-height:520px; overflow:auto; }
.msg { display:block; padding-block:1px; overflow-wrap:anywhere; white-space:pre-wrap; }
.msg.fl { background:var(--panel); box-shadow:inset 3px 0 0 var(--accent); padding-left:6px; }
.gapl { display:block; color:var(--muted); font-size:11.5px; padding-block:2px; }
.brk { display:flex; flex-wrap:wrap; gap:3px; align-items:center; border-top:2px solid var(--muted); margin-block:5px 3px; padding-top:2px; min-height:10px; }
.tag { font:700 10.5px/1 "IBM Plex Mono",monospace; padding:2px 4px; border-radius:3px; color:var(--ground); }
.tag.size { background:var(--f-size); } .tag.sem { background:var(--f-sem); } .tag.late { background:var(--f-late); }
.tag.neural { background:var(--f-neural); } .tag.llm { background:var(--f-llm); } .tag.old { background:var(--f-old); }
.chips { display:flex; flex-wrap:wrap; gap:6px; }
.chip { font:inherit; font-size:12.5px; border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:12px; padding:2px 9px; cursor:pointer; }
.chip[aria-pressed="false"] { opacity:.45; text-decoration:line-through; }
.mrows { display:grid; gap:2px; }
.mrow { display:grid; grid-template-columns:minmax(150px,230px) 1fr auto; gap:8px; align-items:center; font-size:12.5px; padding-block:2px; border-radius:4px; }
.mrow.pick { background:var(--ok-bg); }
.mrow .lbl { overflow-wrap:anywhere; min-width:0; } .mrow .lbl b { font-weight:600; } .mrow .lbl span { color:var(--muted); }
.cells { display:flex; height:14px; border:1px solid var(--line); border-radius:3px; overflow:hidden; min-width:0; }
.cells span { flex:1; min-width:1px; } .cells span.a { background:var(--soft); } .cells span.b { background:var(--oth, #9aa4f033); background:color-mix(in srgb, var(--accent) 28%, transparent); }
.cells span.f { box-shadow:inset 0 -3px 0 var(--accent); }
.pickbtn { font:inherit; font-size:12.5px; font-weight:600; border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:4px; padding:3px 8px; cursor:pointer; }
.pickbtn.on { background:var(--ok-bg); color:var(--ok); border-color:var(--ok); }
.err { font-size:12px; color:var(--muted); }
.note textarea { width:100%; min-height:38px; }
@media (max-width:640px) { .mrow { grid-template-columns:1fr auto; } .mrow .cells { grid-column:1 / -1; order:3; } }
</style>
<div class="wrap">
  <header>
    <h1>Chunk Method Review</h1>
    <p class="lede">The same 25 stretches of the 2024 texts, split by every Chonkie chunker that fits a text stream. A line above a message means a method starts a new chunk there; the letters say which methods. For each stretch, mark the methods whose breaks you would accept as conversations. Your marks are saved as you go.</p>
  </header>
  <section class="panel"><h2>The methods</h2>
    <p class="gold">Counts are over all 25 stretches. Size-based methods (T, F, S, R) cut by length and only prefer to break at line ends; M splits where the meaning shifts; L embeds the whole stretch first; N is a trained topic-shift model; K is an LLM choosing the cut points.</p>
    <div class="tablewrap"><table id="board"><thead><tr><th>Code</th><th>Method</th><th>Chunks</th><th>Median messages</th><th>Settings</th><th>Marked right</th></tr></thead><tbody></tbody></table></div>
  </section>
  <div class="bar"><select id="pick-chunk" aria-label="Stretch"></select><span class="save" id="save-state">Connecting…</span></div>
  <div id="chunk" class="wrap"></div>
</div>
<script>
const CHUNKS = __CHUNKS__;
const METHODS = __METHODS__;
const $ = id => document.getElementById(id);
const el = (t, c, x) => { const e = document.createElement(t); if (c) e.className = c; if (x != null) e.textContent = x; return e; };
let cur = 0, db = null; const picks = {}; const hidden = new Set();
try { const h = JSON.parse(localStorage.getItem("chunkHidden") || "[]"); h.forEach(x => hidden.add(x)); } catch (e) {}
function board() {
  const tb = $("board").querySelector("tbody"); tb.replaceChildren();
  const counts = {}; for (const k in picks) for (const m of picks[k].models || []) counts[m] = (counts[m] || 0) + 1;
  for (const m of METHODS) {
    const tr = el("tr");
    const cfg = Object.entries(m.settings).map(([k, v]) => k + " " + (typeof v === "object" ? JSON.stringify(v) : v)).join(", ");
    const cells = [m.code, m.label + (m.failed ? " (" + m.failed + " stretches failed)" : ""), m.n_chunks, m.median, cfg, counts[m.id] || 0];
    cells.forEach((v, i) => { const td = el("td", i === 0 || i >= 2 ? "mono" : null, String(v)); if (i === 0) { td.textContent = ""; td.append(el("span", "tag " + m.fam, v)); } tr.append(td); });
    tb.append(tr);
  }
}
function strip(sp, n, ff, ft) {
  const cells = el("div", "cells"); const set = new Set(sp || []); let seg = 0;
  for (let j = 0; j < n; j++) { if (set.has(j)) seg++; const c = el("span", (seg % 2 ? "b" : "a") + (j >= ff && j <= ft ? " f" : "")); c.title = "message " + j; cells.append(c); }
  return cells;
}
function render() {
  const c = CHUNKS[cur]; const box = $("chunk"); box.replaceChildren();
  const p = el("section", "panel wrap");
  p.append(el("h2", null, c.id + " · " + c.n + " messages · " + (c.group === "reviewed" ? "you reviewed this" : c.group)));
  if (c.review) p.append(el("div", "why", "Your review: " + c.review));
  if (c.why) p.append(el("div", "gold", "Why it's in the test: " + c.why));
  const chips = el("div", "chips");
  for (const m of METHODS) {
    const b = el("button", "chip"); b.type = "button"; b.setAttribute("aria-pressed", hidden.has(m.id) ? "false" : "true");
    b.append(el("span", "tag " + m.fam, m.code), document.createTextNode(" " + m.label));
    b.addEventListener("click", () => { hidden.has(m.id) ? hidden.delete(m.id) : hidden.add(m.id); try { localStorage.setItem("chunkHidden", JSON.stringify([...hidden])); } catch (e) {} render(); });
    chips.append(b);
  }
  p.append(el("div", "gold", "Break markers shown on the messages (tap a method to hide or show its marker):"), chips);
  const pre = el("div", "msgs");
  c.lines.forEach(l => {
    const here = METHODS.filter(m => !hidden.has(m.id) && (c.rows[m.id] || []).includes(l.j));
    if (here.length) { const b = el("div", "brk"); here.forEach(m => b.append(el("span", "tag " + m.fam, m.code))); pre.append(b); }
    if (l.gap) pre.append(el("span", "gapl", "· " + l.gap));
    pre.append(el("span", "msg" + (l.j >= c.ff && l.j <= c.ft ? " fl" : ""), "[" + l.j + "] " + l.t));
  });
  p.append(pre);
  const rows = el("div", "mrows");
  const old = el("div", "mrow"); const ol = el("div", "lbl"); ol.append(el("b", null, "Old 30-minute rule"), el("span", null, " · " + (c.rows.old.length + 1) + " chunks"));
  old.append(ol, strip(c.rows.old, c.n, c.ff, c.ft), el("span")); rows.append(old);
  const mine = new Set((picks[c.id] || {}).models || []);
  for (const m of METHODS) {
    const r = c.rows[m.id]; const st = c.stats[m.id];
    const row = el("div", "mrow" + (mine.has(m.id) ? " pick" : ""));
    const l = el("div", "lbl"); l.append(el("span", "tag " + m.fam, m.code), el("b", null, " " + m.label), el("span", null, r ? " · " + st[0] + " chunks, median " + st[1] : " · no result"));
    row.append(l);
    if (r) row.append(strip(r, c.n, c.ff, c.ft)); else row.append(el("div", "err", st[2]));
    const b = el("button", "pickbtn" + (mine.has(m.id) ? " on" : ""), mine.has(m.id) ? "Marked right" : "Mark right"); b.type = "button"; b.id = "pick-" + c.id + "-" + m.id;
    b.setAttribute("aria-pressed", mine.has(m.id) ? "true" : "false");
    if (r) { b.addEventListener("click", () => { const set = new Set((picks[c.id] || {}).models || []); set.has(m.id) ? set.delete(m.id) : set.add(m.id); save(c.id, { models: [...set] }); }); row.append(b); } else row.append(el("span"));
    rows.append(row);
  }
  p.append(el("div", "gold", "Each strip is the stretch left to right; a colour change is a new chunk, underlined = the messages you reviewed."), rows);
  const note = el("div", "note"); const ta = el("textarea"); ta.id = "note-" + c.id; ta.placeholder = "What did the right ones get right? What did they all miss? (optional)";
  ta.value = (picks[c.id] || {}).note || ""; ta.addEventListener("input", () => save(c.id, { note: ta.value }, true)); note.append(ta); p.append(note);
  box.append(p);
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
  try { await db.doc("chunks/" + id).set(payload); setSave("Saved " + new Date().toLocaleTimeString()); }
  catch (e) { setSave("Not saved (" + (e && e.code || "error") + "). Your answer is still on screen.", true); }
  finally { inflight[id] = false; if (queued[id]) flush(id); }
}
async function connect() {
  db = await window.claude?.use?.("db") ?? null;
  if (!db) { setSave("Saving unavailable in this view: answers stay on screen only", true); return; }
  setSave("Connected");
  db.collection("chunks").onSnapshot(qs => {
    for (const d of qs.docs) { if (!queued[d.id] && !inflight[d.id]) picks[d.id] = d.data(); }
    board(); if (document.activeElement?.tagName !== "TEXTAREA") render();
  }, e => setSave("Live sync stopped (" + (e && e.code || "error") + "); reload to reconnect", true));
}
CHUNKS.forEach((c, i) => { const o = el("option", null, (i + 1) + ". " + (c.group === "reviewed" ? "" : "[" + c.group + "] ") + c.id + " — " + (c.review || c.why).slice(0, 70)); o.value = String(i); $("pick-chunk").append(o); });
$("pick-chunk").addEventListener("change", () => { cur = +$("pick-chunk").value; render(); });
board(); render(); connect();
</script>
"""

page = TEMPLATE.replace("__CHUNKS__", json.dumps(stretches, ensure_ascii=False)).replace("__METHODS__", json.dumps(methods))
out_path.write_text(page, encoding="utf-8")
print(f"wrote {out_path}: {len(stretches)} stretches x {len(methods)} methods, {len(page) // 1024} KB")
for m in methods:
    print(f"  {m['code']:<3} {m['label']:<36} chunks={m['n_chunks']:>5} median={m['median']} failed={m['failed']}")
