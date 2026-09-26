"""Build the owner's block comparison page: one block of texts, each model's episodes and labels side by side.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 20:46-20:53: the bouts were a hot mess (chunking cut conversations
apart; "neutral" hid the good); run a block of texts through Gemini Flash / Pro, Gemma 4 31B and Nemotron 3.5 Lightning.
For each model the owner can say Better / Worse / Mixed and leave a note; answers save to the artifact's `db`
(collection `block_reviews`, doc per block+model). The output embeds case messages: build it on the devbox or in a
scratch path, never commit it.

Usage: python block_review_page.py <out.html> <results dir> [<results dir> ...]
Each results dir holds _block_<version>.json (or _block.json) and <provider>_<model>[_<version>].json from
block_review_llm.py.
"""

import json
import pathlib
import sys

POS = {"warm", "cooperative", "playful", "conciliatory"}
NEG = {"edgy", "conflict", "hostile", "threat", "deflecting", "leverage"}
out_path = pathlib.Path(sys.argv[1])
blocks = []
for d in map(pathlib.Path, sys.argv[2:]):
    for bf in sorted(d.glob("_block*.json")):
        b = json.loads(bf.read_text(encoding="utf-8"))
        version = b.get("version", "block-episodes-v1")
        runs = []
        for rf in sorted(d.glob("*.json")):
            if rf.name.startswith("_"):
                continue
            r = json.loads(rf.read_text(encoding="utf-8"))
            if r.get("version", "block-episodes-v1") != version:
                continue
            runs.append({"key": r["provider"] + ":" + r["model"], "ok": r.get("ok"), "error": (r.get("error") or "")[:300],
                         "seconds": r.get("seconds"), "problems": r.get("problems") or [],
                         "episodes": (r.get("output") or {}).get("episodes", []),
                         "overall": (r.get("output") or {}).get("overall", "")})
        m = b["messages"]
        if any(x["id"] == f"{m[0]['day']}_{version}" for x in blocks):
            continue  # the same block saved under the older file name (_block.json) and the versioned one
        blocks.append({"id": f"{m[0]['day']}_{version}", "version": version, "range": f"{m[0]['day']} {m[0]['ts']} → {m[-1]['day']} {m[-1]['ts']}",
                       "msgs": [[x["i"], x["day"], x["ts"], 0 if x["who"] == "Matt" else 1, x["text"]] for x in m],
                       "runs": runs})

TEMPLATE = r"""<title>Block Review · 2024 Texts</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap">
<style>
/* Palette: Propria design contract 1.0.0 (same tokens as the Bout Review and Tone Disagreements pages). */
:root {
  --ground:#f5f3ee; --panel:#fffefb; --ink:#1d2228; --muted:#687078; --line:#d5d1c9; --soft:#ebe8e0; --focus:#4051b9;
  --accent:#4051b9; --accent-ink:#fff; --ok:#247047; --ok-bg:#e2efe6; --bad:#b5433b; --bad-bg:#f8e5e3; --mix:#9a5a12; --mix-bg:#f7ecdc;
  --matt:#376f72; --kat:#8a4f7d; --pos:#2f7d4f; --pos-bg:#e3f1e8; --neg:#b5433b; --neg-bg:#f8e5e3; --oth:#5b5f9e; --oth-bg:#e8e9f6;
}
@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --accent-ink:#111820; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a; --mix:#e6b55d; --mix-bg:#40341d;
  --matt:#82bdc0; --kat:#d69cc6; --pos:#7cc79a; --pos-bg:#1f3a2b; --neg:#ef7b6e; --neg-bg:#43282a; --oth:#aeb2f0; --oth-bg:#2e3150; } }
:root[data-theme="dark"] {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --accent-ink:#111820; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a; --mix:#e6b55d; --mix-bg:#40341d;
  --matt:#82bdc0; --kat:#d69cc6; --pos:#7cc79a; --pos-bg:#1f3a2b; --neg:#ef7b6e; --neg-bg:#43282a; --oth:#aeb2f0; --oth-bg:#2e3150; }
* { box-sizing:border-box; }
body { margin:0; background:var(--ground); color:var(--ink); font:15px/1.5 "Instrument Sans", "Segoe UI", system-ui, sans-serif; padding-inline:16px; padding-block:20px 80px; }
.wrap { max-width:1100px; margin:0 auto; display:grid; gap:14px; }
h1 { font-size:24px; margin:0; letter-spacing:-.01em; text-wrap:balance; }
.lede { color:var(--muted); margin:4px 0 0; max-width:72ch; }
.mono { font-family:"IBM Plex Mono", ui-monospace, Consolas, monospace; font-size:12.5px; font-variant-numeric:tabular-nums; }
.panel { background:var(--panel); border:1px solid var(--line); border-radius:6px; padding:12px 14px; }
.bar { position:sticky; top:env(safe-area-inset-top, 0px); z-index:5; background:var(--ground); padding-block:10px; display:flex; flex-wrap:wrap; gap:8px; align-items:center; border-bottom:1px solid var(--line); }
select, textarea { font:inherit; font-size:14px; color:var(--ink); background:var(--panel); border:1px solid var(--line); border-radius:4px; padding:6px 8px; }
.tabs { display:flex; flex-wrap:wrap; gap:6px; }
.tab { font:inherit; font-size:13.5px; font-weight:600; background:var(--panel); color:var(--ink); border:1px solid var(--line); border-radius:4px; padding:6px 10px; cursor:pointer; }
.tab.on { border-color:var(--accent); box-shadow:inset 0 0 0 1px var(--accent); }
.tab .err { color:var(--bad); font-weight:500; }
:focus-visible { outline:2px solid var(--focus); outline-offset:2px; }
.save { font-size:13px; color:var(--muted); } .save.err { color:var(--bad); }
.maps { display:grid; gap:4px; }
.maprow { display:grid; grid-template-columns:230px 1fr; gap:8px; align-items:center; font-size:12.5px; }
.map { display:flex; height:14px; gap:1px; }
.map i { display:block; height:100%; border-radius:2px; }
.ep { background:var(--panel); border:1px solid var(--line); border-radius:6px; padding:12px 14px; display:grid; gap:8px; }
.ephead { display:flex; flex-wrap:wrap; gap:6px 12px; align-items:baseline; }
.ephead .t { font-weight:700; }
.sum { font-size:14px; }
.labs { display:flex; flex-wrap:wrap; gap:6px; }
.lab { font-size:12.5px; padding:3px 8px; border-radius:4px; background:var(--oth-bg); color:var(--oth); max-width:100%; }
.lab.pos { background:var(--pos-bg); color:var(--pos); } .lab.neg { background:var(--neg-bg); color:var(--neg); }
.lab b { font-weight:700; } .lab .dots { letter-spacing:1px; }
.lab .n { display:block; color:var(--ink); opacity:.85; margin-top:1px; }
.un { font-size:13px; color:var(--mix); }
.msgs { display:grid; gap:1px; border-top:1px solid var(--line); padding-top:6px; }
.msg { display:grid; grid-template-columns:34px 88px 60px 1fr; gap:8px; font-size:14px; padding:1px 0; }
.msg.hit { background:var(--soft); border-radius:3px; }
.msg .i, .msg .time { font:12px "IBM Plex Mono", ui-monospace, monospace; color:var(--muted); padding-top:2px; }
.msg .who { font-weight:600; font-size:13px; } .msg .who.m { color:var(--matt); } .msg .who.k { color:var(--kat); }
.msg .txt { white-space:pre-wrap; overflow-wrap:anywhere; }
.verdict { display:flex; flex-wrap:wrap; gap:6px; align-items:center; }
.verdict button { font:inherit; font-size:13px; font-weight:600; border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:4px; padding:5px 10px; cursor:pointer; }
.verdict button.on.better { background:var(--ok-bg); color:var(--ok); border-color:var(--ok); }
.verdict button.on.worse { background:var(--bad-bg); color:var(--bad); border-color:var(--bad); }
.verdict button.on.mixed { background:var(--mix-bg); color:var(--mix); border-color:var(--mix); }
.verdict textarea { flex:1 1 280px; min-height:36px; }
.none { color:var(--muted); }
</style>
<div class="wrap">
  <header>
    <h1>Block Review · 2024 Texts</h1>
    <p class="lede">One block of consecutive texts from Katrina's phone, read whole by each model. Each model splits it into conversations by topic (not by the clock) and labels every conversation: who, at whom, how strongly, and which messages show it. Pick a model to read its take, then say whether it is better than the old bouts.</p>
  </header>
  <div class="bar">
    <select id="blk" aria-label="Block and prompt version"></select>
    <div class="tabs" id="tabs" role="tablist" aria-label="Model"></div>
    <span class="save" id="save-state">Connecting…</span>
  </div>
  <section class="panel maps" id="maps" aria-label="Where each model split the block"></section>
  <section class="panel verdict" id="verdict"></section>
  <div id="eps" class="wrap"></div>
</div>
<script>
const BLOCKS = __BLOCKS__;
const POS = new Set(__POS__), NEG = new Set(__NEG__);
const $ = id => document.getElementById(id);
const el = (t, c, x) => { const e = document.createElement(t); if (c) e.className = c; if (x != null) e.textContent = x; return e; };
const WHO = ["Matt", "Katrina"];
let blk = BLOCKS[0], run = null, db = null; const reviews = {};
const tone = l => POS.has(l) ? "pos" : NEG.has(l) ? "neg" : "oth";
function mainTone(ep) {
  const score = { pos: 0, neg: 0, oth: 0 };
  for (const l of ep.labels || []) score[tone(l.label)] += (l.intensity || 1);
  return score.neg > score.pos ? "neg" : score.pos > 0 ? "pos" : "oth";
}
function maps() {
  const box = $("maps"); box.replaceChildren(el("div", "lede", "Where each model split the block (green = mostly positive labels, red = mostly negative, purple = other):"));
  const n = blk.msgs.length;
  for (const r of blk.runs) {
    const row = el("div", "maprow"); row.append(el("span", "mono", r.key.split(":")[1]));
    const map = el("div", "map");
    if (!r.ok) { const i = el("i"); i.style.flexGrow = "1"; i.style.background = "var(--line)"; map.append(i); }
    for (const ep of r.episodes) { const i = el("i"); i.style.flexGrow = String(Math.max(1, ep.to_i - ep.from_i + 1)); i.style.background = "var(--" + mainTone(ep) + ")"; i.title = ep.from_i + "–" + ep.to_i + " · " + ep.topic; map.append(i); }
    row.append(map); box.append(row);
  }
}
function tabs() {
  const box = $("tabs"); box.replaceChildren();
  for (const r of blk.runs) {
    const b = el("button", "tab" + (r === run ? " on" : "")); b.type = "button"; b.setAttribute("role", "tab"); b.setAttribute("aria-selected", r === run ? "true" : "false");
    b.append(document.createTextNode(r.key.split(":")[1] + " "));
    if (!r.ok) b.append(el("span", "err", "(failed)")); else b.append(el("span", "mono", r.episodes.length + " conv."));
    b.addEventListener("click", () => { run = r; render(); }); box.append(b);
  }
}
function verdict() {
  const box = $("verdict"); box.replaceChildren();
  if (!run) return;
  const key = blk.id + "__" + run.key.replace(/[^A-Za-z0-9.-]/g, "_"); const cur = reviews[key] || {};
  box.append(el("strong", null, "Your verdict on " + run.key.split(":")[1] + ": "));
  for (const [k, label] of [["better", "Better than the bouts"], ["mixed", "Mixed"], ["worse", "Worse"]]) {
    const bt = el("button", k + (cur.verdict === k ? " on" : ""), label); bt.type = "button"; bt.id = "v-" + key + "-" + k;
    bt.setAttribute("aria-pressed", cur.verdict === k ? "true" : "false");
    bt.addEventListener("click", () => save(key, { verdict: cur.verdict === k ? "" : k })); box.append(bt);
  }
  const ta = el("textarea"); ta.id = "n-" + key; ta.placeholder = "What did it get right or wrong? (optional)"; ta.value = cur.note || "";
  ta.addEventListener("input", () => save(key, { note: ta.value }, true)); box.append(ta);
}
function render() {
  tabs(); verdict();
  const box = $("eps"); box.replaceChildren();
  if (!run) { box.append(el("p", "none", "No model result for this block yet.")); return; }
  if (!run.ok) { box.append(el("p", "none", "This model did not return a result: " + run.error)); return; }
  if (run.overall) { const o = el("div", "panel"); o.append(el("strong", null, "Overall: "), document.createTextNode(run.overall)); box.append(o); }
  if (run.problems.length) box.append(el("p", "none", "Structure warnings: " + run.problems.join("; ")));
  for (const ep of run.episodes) {
    const c = el("article", "ep");
    const h = el("div", "ephead"); h.append(el("span", "t", ep.topic), el("span", "mono", "messages " + ep.from_i + "–" + ep.to_i)); c.append(h);
    if (ep.summary) c.append(el("div", "sum", ep.summary));
    const labs = el("div", "labs"); const hit = new Set();
    for (const l of ep.labels || []) {
      const s = el("span", "lab " + tone(l.label));
      s.append(el("b", null, l.label), document.createTextNode(" · " + l.who + " → " + l.directed_at + " "), el("span", "dots", "●".repeat(Math.max(1, Math.min(3, l.intensity || 1)))));
      if (l.responds_to_i != null && l.responds_to_i >= 0) s.append(document.createTextNode(" · reacts to #" + l.responds_to_i));
      if (l.note) s.append(el("span", "n", l.note));
      (l.evidence_i || []).forEach(i => hit.add(i)); labs.append(s);
    }
    c.append(labs);
    for (const u of ep.unanswered || []) c.append(el("div", "un", "Unanswered: " + u.who + " · messages " + u.from_i + "–" + u.to_i + " · " + u.note));
    const ms = el("div", "msgs");
    for (const m of blk.msgs.slice(ep.from_i, ep.to_i + 1)) {
      const row = el("div", "msg" + (hit.has(m[0]) ? " hit" : ""));
      row.append(el("span", "i", "#" + m[0]), el("span", "time", m[1].slice(5) + " " + m[2]), el("span", "who " + (m[3] ? "k" : "m"), WHO[m[3]]), el("span", "txt", m[4]));
      ms.append(row);
    }
    c.append(ms); box.append(c);
  }
}
const timers = {}, queued = {}, inflight = {};
function setSave(t, err) { const s = $("save-state"); s.textContent = t; s.classList.toggle("err", !!err); }
function save(key, patch, quiet) {
  reviews[key] = Object.assign({ block: blk.id, model: run.key, verdict: "", note: "" }, reviews[key] || {}, patch, { updated_at: new Date().toISOString() });
  if (!quiet) verdict();
  queued[key] = reviews[key]; clearTimeout(timers[key]); timers[key] = setTimeout(() => flush(key), 600); setSave("Saving…");
}
async function flush(key) {
  if (!db) { setSave("Not saved: storage unavailable in this view", true); return; }
  if (inflight[key]) return; const payload = queued[key]; delete queued[key]; if (!payload) return; inflight[key] = true;
  try { await db.doc("block_reviews/" + key).set(payload); setSave("Saved " + new Date().toLocaleTimeString()); }
  catch (e) { setSave("Not saved (" + (e && e.code || "error") + "). Your answer is still on screen.", true); }
  finally { inflight[key] = false; if (queued[key]) flush(key); }
}
async function connect() {
  db = await window.claude?.use?.("db") ?? null;
  if (!db) { setSave("Saving unavailable in this view: answers stay on screen only", true); return; }
  setSave("Connected");
  db.collection("block_reviews").onSnapshot(qs => {
    for (const d of qs.docs) { if (!queued[d.id] && !inflight[d.id]) reviews[d.id] = d.data(); }
    if (document.activeElement?.tagName !== "TEXTAREA") verdict();
  }, e => setSave("Live sync stopped (" + (e && e.code || "error") + "); reload to reconnect", true));
}
function init() {
  BLOCKS.forEach((b, i) => { const o = el("option", null, b.range + " · " + (b.version.endsWith("v2") ? "prompt v2 (background + examples)" : "prompt v1")); o.value = String(i); $("blk").append(o); });
  $("blk").addEventListener("change", () => { blk = BLOCKS[+$("blk").value]; run = blk.runs.find(r => r.ok) || blk.runs[0] || null; maps(); render(); });
  run = blk.runs.find(r => r.ok) || blk.runs[0] || null; maps(); render(); connect();
}
init();
</script>
"""

page = (TEMPLATE.replace("__BLOCKS__", json.dumps(blocks, ensure_ascii=False)).replace("__POS__", json.dumps(sorted(POS)))
        .replace("__NEG__", json.dumps(sorted(NEG))))
out_path.write_text(page, encoding="utf-8")
print(f"wrote {out_path}: {len(blocks)} block(s), runs: " + "; ".join(f"{b['id']}: " + ", ".join(
    f"{r['key'].split(':')[1]}={'ok' if r['ok'] else 'failed'}" for r in b["runs"]) for b in blocks))
