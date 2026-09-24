"""Build the owner's bout review page: day bouts with Opus tone stretches and shifts (2024 texts, her phone).

Byline: Claude Code · Opus 5.5 · 2026-09-24. The owner judges each bout: right, wrong, or missed a shift,
plus a note. Answers are saved to the artifact's `db` (collection `bouts`, doc per bout id) and read back
with read_db. The output embeds case messages: it is built on the devbox or in a scratch path and never
committed.

Usage: python bout_review_page.py <bouts.jsonl> <raw/bout_tone dir> <out.html>
"""

import collections
import json
import pathlib
import sys

TONES = ["affectionate", "friendly", "neutral", "tense", "hostile", "distressed", "conciliatory"]
bouts_path, tone_dir, out_path = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2]), pathlib.Path(sys.argv[3])
bouts = [json.loads(x) for x in bouts_path.read_text(encoding="utf-8").split("\n") if x.strip()]
tone = {}
for p in tone_dir.glob("c2024-*.json"):
    d = json.loads(p.read_text(encoding="utf-8"))
    if d.get("ok"):
        tone[d["bout_id"]] = d["output"]

data, counts, shifts_n, abrupt_n = [], collections.Counter(), 0, 0
for b in bouts:
    t = tone.get(b["bout_id"])
    st = [[s["from_i"], s["to_i"], TONES.index(s["tone"]), s["driver"], s["note"]] for s in (t or {}).get("stretches", [])]
    sh = [[s["at_i"], TONES.index(s["from_tone"]), TONES.index(s["to_tone"]), s["speed"], s["trigger_i"], s["note"]]
          for s in (t or {}).get("shifts", [])]
    for s in st:
        counts[TONES[s[2]]] += s[1] - s[0] + 1
    shifts_n += len(sh)
    abrupt_n += sum(1 for s in sh if s[3] == "abrupt")
    data.append({"id": b["bout_id"], "day": b["day"], "s": b["start_local"][11:16], "e": b["end_local"][11:16],
                 "m": [[m["ts_local"], 0 if m["who"] == "Matt" else 1, m["text"]] for m in b["messages"]],
                 "st": st, "sh": sh, "sum": (t or {}).get("summary", ""), "ok": t is not None})

meta = {"bouts": len(data), "labelled": len(tone), "days": len({d["day"] for d in data}),
        "messages": sum(len(d["m"]) for d in data), "shifts": shifts_n, "abrupt": abrupt_n,
        "tone_msgs": {k: counts[k] for k in TONES}}

TEMPLATE = r"""<title>Bout Review · 2024 Texts</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap">
<style>
/* Palette: Propria design contract 1.0.0 (Probata graphite/indigo), owner direction 2026-09-24. Tone colours are data colours. */
:root {
  --ground:#f5f3ee; --panel:#fffefb; --ink:#1d2228; --muted:#687078; --line:#d5d1c9; --soft:#ebe8e0; --focus:#4051b9;
  --accent:#4051b9; --accent-ink:#fff; --ok:#247047; --ok-bg:#e2efe6; --bad:#b5433b; --bad-bg:#f8e5e3; --miss:#9a5a12; --miss-bg:#f7ecdc;
  --matt:#376f72; --kat:#8a4f7d;
  --t0:#c2477a; --t1:#3d8b5a; --t2:#9aa1ab; --t3:#c98a1c; --t4:#c0392b; --t5:#7a4bb5; --t6:#1f8a8a;
}
@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --accent-ink:#111820; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a; --miss:#e6b55d; --miss-bg:#40341d;
  --matt:#82bdc0; --kat:#d69cc6;
  --t0:#e98ab0; --t1:#7cc79a; --t2:#7d8791; --t3:#e2b25a; --t4:#ef7b6e; --t5:#b39be6; --t6:#6cc9c9; } }
:root[data-theme="dark"] {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --accent-ink:#111820; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a; --miss:#e6b55d; --miss-bg:#40341d;
  --matt:#82bdc0; --kat:#d69cc6;
  --t0:#e98ab0; --t1:#7cc79a; --t2:#7d8791; --t3:#e2b25a; --t4:#ef7b6e; --t5:#b39be6; --t6:#6cc9c9; }
* { box-sizing:border-box; }
body { margin:0; background:var(--ground); color:var(--ink); font:15px/1.5 "Instrument Sans", "Segoe UI", system-ui, sans-serif; padding-inline:16px; padding-block:20px 80px; }
.wrap { max-width:1100px; margin:0 auto; display:grid; gap:16px; }
h1 { font-size:24px; margin:0; letter-spacing:-.01em; text-wrap:balance; }
.lede { color:var(--muted); margin:4px 0 0; max-width:72ch; }
.mono { font-family:"IBM Plex Mono", ui-monospace, Consolas, monospace; font-size:12.5px; font-variant-numeric:tabular-nums; }
.panel { background:var(--panel); border:1px solid var(--line); border-radius:6px; padding:14px 16px; }
.facts { display:flex; flex-wrap:wrap; gap:6px 20px; font-size:14px; }
.facts b { font-variant-numeric:tabular-nums; }
.legend { display:flex; flex-wrap:wrap; gap:6px 14px; font-size:13px; color:var(--muted); }
.sw { display:inline-block; width:12px; height:12px; border-radius:3px; vertical-align:-1px; margin-right:5px; }
.save { font-size:13px; color:var(--muted); } .save.err { color:var(--bad); }
.strip { display:grid; gap:2px; }
.day-row { display:grid; grid-template-columns:92px 1fr 44px; gap:8px; align-items:center; cursor:pointer; border-radius:4px; padding:1px 2px; }
.day-row:hover { background:var(--soft); }
.day-row .d { font:500 12px "IBM Plex Mono", ui-monospace, monospace; color:var(--muted); }
.day-row .n { font:12px "IBM Plex Mono", ui-monospace, monospace; color:var(--muted); text-align:right; }
.bar { display:flex; gap:2px; height:14px; }
.bout-seg { display:flex; height:100%; border-radius:2px; overflow:hidden; min-width:3px; }
.bout-seg i { display:block; height:100%; }
.filters { position:sticky; top:env(safe-area-inset-top, 0px); z-index:5; background:var(--ground); padding-block:10px; display:flex; flex-wrap:wrap; gap:8px; align-items:center; border-bottom:1px solid var(--line); }
select, textarea { font:inherit; font-size:14px; color:var(--ink); background:var(--panel); border:1px solid var(--line); border-radius:4px; padding:6px 8px; }
.filters select { flex:1 1 160px; }
.filters .count { color:var(--muted); font-size:13px; }
:focus-visible { outline:2px solid var(--focus); outline-offset:2px; }
details.day { background:var(--panel); border:1px solid var(--line); border-radius:6px; }
details.day > summary { cursor:pointer; padding:10px 14px; display:flex; flex-wrap:wrap; gap:6px 14px; align-items:center; }
details.day > summary .t { font-weight:700; }
details.day .body { padding:0 14px 14px; display:grid; gap:12px; }
.bout { border:1px solid var(--line); border-radius:4px; padding:10px 12px; display:grid; gap:8px; }
.bhead { display:flex; flex-wrap:wrap; gap:6px 12px; align-items:baseline; font-size:13px; color:var(--muted); }
.bsum { font-size:14px; }
.msgs { display:grid; gap:1px; }
.msg { display:grid; grid-template-columns:4px 46px 60px 1fr; gap:8px; align-items:start; padding:2px 0; font-size:14px; }
.msg .tone { align-self:stretch; border-radius:2px; }
.msg .time { font:12px "IBM Plex Mono", ui-monospace, monospace; color:var(--muted); padding-top:2px; }
.msg .who { font-weight:600; font-size:13px; padding-top:1px; }
.msg .who.m { color:var(--matt); } .msg .who.k { color:var(--kat); }
.msg .txt { white-space:pre-wrap; overflow-wrap:anywhere; }
.shift { margin:4px 0 4px 58px; font-size:13px; padding:4px 8px; border-left:3px solid var(--line); background:var(--soft); border-radius:0 6px 6px 0; }
.shift b { font-weight:700; }
.stnote { font-size:12.5px; color:var(--muted); margin-left:58px; }
.fb { display:flex; flex-wrap:wrap; gap:6px; align-items:center; }
.fb button { font:inherit; font-size:13px; font-weight:600; border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:4px; padding:5px 10px; cursor:pointer; }
.fb button.on.right { background:var(--ok-bg); color:var(--ok); border-color:var(--ok); }
.fb button.on.wrong { background:var(--bad-bg); color:var(--bad); border-color:var(--bad); }
.fb button.on.missed { background:var(--miss-bg); color:var(--miss); border-color:var(--miss); }
.fb textarea { flex:1 1 280px; min-height:36px; }
.none { color:var(--muted); font-size:13px; }
</style>
<div class="wrap">
  <header>
    <h1>Bout Review · 2024 Texts</h1>
    <p class="lede">Texts from Katrina's phone, __RANGE__, cut into bouts (a new bout after 30 minutes of silence, within a day). Opus marked the tone of each stretch and every point where it turned. For each bout, say whether Opus got it right.</p>
  </header>
  <section class="panel facts" id="facts"></section>
  <section class="panel">
    <div class="legend" id="legend"></div>
    <p class="lede" style="margin:8px 0 10px">Each row is a day; each block is a bout, split into its tone stretches. Click a row to open that day.</p>
    <div class="strip" id="strip"></div>
  </section>
  <div class="filters" role="search">
    <select id="f-tone" aria-label="Tone"><option value="">Any tone</option></select>
    <select id="f-shift" aria-label="Shifts"><option value="">Any bout</option><option value="abrupt">Has an abrupt shift</option><option value="any">Has any shift</option><option value="two">Both of you talking</option></select>
    <select id="f-month" aria-label="Month"><option value="">All months</option></select>
    <select id="f-review" aria-label="Your review"><option value="">Any review state</option><option value="todo">Not reviewed</option><option value="right">Right</option><option value="wrong">Wrong</option><option value="missed">Missed a shift</option></select>
    <span class="count" id="count"></span><span class="save" id="save-state">Connecting…</span>
  </div>
  <div id="days" class="wrap"></div>
</div>
<script>
const DATA = __DATA__;
const META = __META__;
const TONES = __TONES__;
const $ = id => document.getElementById(id);
const el = (t, c, x) => { const e = document.createElement(t); if (c) e.className = c; if (x != null) e.textContent = x; return e; };
const tv = i => "var(--t" + i + ")";
const WHO = ["Matt", "Katrina"];
const reviews = {}; let db = null;
const byDay = new Map();
for (const b of DATA) { if (!byDay.has(b.day)) byDay.set(b.day, []); byDay.get(b.day).push(b); }
const days = [...byDay.keys()].sort();

function facts() {
  const f = $("facts");
  const add = (label, v) => { const s = el("span"); s.append(document.createTextNode(label + " ")); s.append(el("b", null, v)); f.append(s); };
  add("Days", META.days); add("Bouts", META.bouts); add("Messages", META.messages.toLocaleString());
  add("Tone shifts", META.shifts); add("Abrupt", META.abrupt); add("Labelled", META.labelled + " / " + META.bouts);
  const lg = $("legend");
  TONES.forEach((t, i) => { const s = el("span"); const sw = el("span", "sw"); sw.style.background = tv(i); s.append(sw, document.createTextNode(t + " · " + META.tone_msgs[t].toLocaleString() + " msgs")); lg.append(s); });
}
function segFor(b) {
  const seg = el("span", "bout-seg"); seg.style.flexGrow = String(Math.max(1, b.m.length)); seg.title = b.s + "–" + b.e + " · " + b.m.length + " messages";
  if (!b.st.length) { const i = el("i"); i.style.flexGrow = "1"; i.style.background = "var(--soft)"; seg.append(i); }
  for (const s of b.st) { const i = el("i"); i.style.flexGrow = String(s[1] - s[0] + 1); i.style.background = tv(s[2]); seg.append(i); }
  return seg;
}
function strip() {
  const box = $("strip");
  for (const d of days) {
    const row = el("div", "day-row"); row.tabIndex = 0; row.setAttribute("role", "button");
    const bouts = byDay.get(d);
    const bar = el("div", "bar"); bouts.forEach(b => bar.append(segFor(b)));
    row.append(el("span", "d", d), bar, el("span", "n", String(bouts.reduce((a, b) => a + b.m.length, 0))));
    const open = () => { const det = $("day-" + d); if (!det) return; det.hidden = false; det.open = true; det.scrollIntoView({ behavior: "smooth", block: "start" }); };
    row.addEventListener("click", open); row.addEventListener("keydown", e => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); open(); } });
    box.append(row);
  }
}
function toneOf(b, i) { for (const s of b.st) if (i >= s[0] && i <= s[1]) return s; return null; }
function renderBout(b) {
  const card = el("article", "bout"); card.id = "bout-" + b.id;
  const nM = b.m.filter(m => m[1] === 0).length, nK = b.m.length - nM;
  const head = el("div", "bhead");
  head.append(el("span", "mono", b.s + "–" + b.e), el("span", null, b.m.length + " messages · Matt " + nM + " · Katrina " + nK), el("span", "mono", b.id));
  card.append(head);
  if (b.sum) card.append(el("div", "bsum", b.sum));
  if (!b.ok) card.append(el("div", "none", "Opus has not labelled this bout yet."));
  const list = el("div", "msgs");
  const shiftsAt = new Map(); for (const s of b.sh) { if (!shiftsAt.has(s[0])) shiftsAt.set(s[0], []); shiftsAt.get(s[0]).push(s); }
  const stStart = new Map(b.st.map(s => [s[0], s]));
  b.m.forEach((m, i) => {
    for (const s of shiftsAt.get(i) || []) {
      const box = el("div", "shift"); box.style.borderLeftColor = tv(s[2]);
      box.append(el("b", null, TONES[s[1]] + " → " + TONES[s[2]]), document.createTextNode(" · " + s[3] + " · " + s[5]));
      list.append(box);
    }
    const st0 = stStart.get(i);
    if (st0) list.append(el("div", "stnote", TONES[st0[2]] + " · driven by " + st0[3] + " · " + st0[4]));
    const row = el("div", "msg");
    const bar = el("span", "tone"); const s = toneOf(b, i); bar.style.background = s ? tv(s[2]) : "var(--line)";
    row.append(bar, el("span", "time", m[0]), el("span", "who " + (m[1] ? "k" : "m"), WHO[m[1]]), el("span", "txt", m[2]));
    list.append(row);
  });
  card.append(list);
  const fb = el("div", "fb");
  const btns = {};
  for (const [k, label] of [["right", "Right"], ["wrong", "Wrong"], ["missed", "Missed a shift"]]) {
    const bt = el("button", k, label); bt.type = "button"; bt.id = "v-" + b.id + "-" + k;
    bt.addEventListener("click", () => save(b, { verdict: (reviews[b.id] || {}).verdict === k ? "" : k }));
    btns[k] = bt; fb.append(bt);
  }
  const ta = el("textarea"); ta.id = "n-" + b.id; ta.placeholder = "What did Opus miss or get wrong? (optional)";
  ta.addEventListener("input", () => save(b, { note: ta.value }));
  fb.append(ta); card.append(fb);
  card._btns = btns; card._ta = ta;
  paintReview(b, card);
  return card;
}
function paintReview(b, card) {
  card = card || $("bout-" + b.id); if (!card) return;
  const r = reviews[b.id] || {};
  for (const k in card._btns) card._btns[k].classList.toggle("on", r.verdict === k), card._btns[k].setAttribute("aria-pressed", r.verdict === k ? "true" : "false");
  if (document.activeElement !== card._ta) card._ta.value = r.note || "";
}
function daySection(d) {
  const det = el("details", "day"); det.id = "day-" + d;
  const sum = el("summary");
  const bouts = byDay.get(d);
  const mini = el("div", "bar"); mini.style.flex = "1 1 240px"; bouts.forEach(b => mini.append(segFor(b)));
  sum.append(el("span", "t", d), el("span", "mono", bouts.length + " bouts · " + bouts.reduce((a, b) => a + b.m.length, 0) + " msgs"), mini);
  det.append(sum);
  const body = el("div", "body"); det.append(body);
  det.addEventListener("toggle", () => { if (det.open && !body.childElementCount) { for (const b of bouts) if (matches(b)) body.append(renderBout(b)); } });
  return det;
}
function matches(b) {
  const tone = $("f-tone").value, sh = $("f-shift").value, mo = $("f-month").value, rv = $("f-review").value;
  if (mo && !b.day.startsWith(mo)) return false;
  if (tone !== "" && !b.st.some(s => s[2] === +tone)) return false;
  if (sh === "abrupt" && !b.sh.some(s => s[3] === "abrupt")) return false;
  if (sh === "any" && !b.sh.length) return false;
  if (sh === "two" && !(b.m.some(m => m[1] === 0) && b.m.some(m => m[1] === 1))) return false;
  const v = (reviews[b.id] || {}).verdict || "";
  if (rv === "todo" && v) return false;
  if (rv && rv !== "todo" && v !== rv) return false;
  return true;
}
function applyFilters() {
  let n = 0;
  for (const d of days) {
    const det = $("day-" + d); const hits = byDay.get(d).filter(matches);
    det.hidden = hits.length === 0; n += hits.length;
    const body = det.querySelector(".body"); if (body.childElementCount) { body.replaceChildren(); if (det.open) for (const b of hits) body.append(renderBout(b)); }
  }
  $("count").textContent = n + " bouts";
}
const timers = {}, inflight = {}, queued = {};
function setSave(t, err) { const s = $("save-state"); s.textContent = t; s.classList.toggle("err", !!err); }
function save(b, patch) {
  const next = Object.assign({ bout_id: b.id, day: b.day, verdict: "", note: "" }, reviews[b.id] || {}, patch, { updated_at: new Date().toISOString() });
  reviews[b.id] = next; paintReview(b);
  const path = "bouts/" + b.id; queued[path] = next; clearTimeout(timers[path]);
  timers[path] = setTimeout(() => flush(path), 600); setSave("Saving…");
}
async function flush(path) {
  if (!db) { setSave("Not saved: storage unavailable in this view", true); return; }
  if (inflight[path]) return;
  const payload = queued[path]; delete queued[path]; if (!payload) return;
  inflight[path] = true;
  try { await db.doc(path).set(payload); setSave("Saved " + new Date().toLocaleTimeString()); }
  catch (e) { setSave("Not saved (" + (e && e.code || "error") + "). Your answer is still on screen.", true); }
  finally { inflight[path] = false; if (queued[path]) flush(path); }
}
function init() {
  facts(); strip();
  TONES.forEach((t, i) => { const o = el("option", null, t); o.value = String(i); $("f-tone").append(o); });
  [...new Set(days.map(d => d.slice(0, 7)))].forEach(m => { const o = el("option", null, m); o.value = m; $("f-month").append(o); });
  const box = $("days"); for (const d of days) box.append(daySection(d));
  ["f-tone", "f-shift", "f-month", "f-review"].forEach(id => $(id).addEventListener("change", applyFilters));
  applyFilters(); connect();
}
async function connect() {
  db = await window.claude?.use?.("db") ?? null;
  if (!db) { setSave("Saving unavailable in this view: answers stay on screen only", true); return; }
  setSave("Connected");
  db.collection("bouts").onSnapshot(qs => {
    for (const doc of qs.docs) { const v = doc.data(); if (v && v.bout_id && !queued["bouts/" + v.bout_id] && !inflight["bouts/" + v.bout_id]) reviews[v.bout_id] = v; }
    for (const b of DATA) paintReview(b);
  }, e => setSave("Live sync stopped (" + (e && e.code || "error") + "); reload to reconnect", true));
}
init();
</script>
"""

page = (TEMPLATE.replace("__DATA__", json.dumps(data, ensure_ascii=False)).replace("__RANGE__", f"{min(d['day'] for d in data)} to {max(d['day'] for d in data)}")
        .replace("__META__", json.dumps(meta)).replace("__TONES__", json.dumps(TONES)))
out_path.write_text(page, encoding="utf-8")
print(f"wrote {out_path}: {meta['bouts']} bouts, {meta['labelled']} labelled, {len(page)//1024} KB")
