"""Build the owner's interactive label-review page (Jev eval, Phase 3 checkpoint).

Byline: Claude Code · Opus 5.5 · 2026-09-23. Owner 15:06: "turn this into an artifact where I can actually
type in a response". The page embeds the 300 sample messages with Opus's labels. The owner flips any
label, sets the register, comments per message, and writes what each tag should mean. Answers are saved
to the artifact's own `db` (collections `reviews`, `notes`) so Claude can read them back with read_db.
The output contains case data: it is written to a local scratch path and never committed.

Usage: python review_page.py <sample_v2.jsonl> <labels_opus.jsonl> <out.html>
"""

import json
import pathlib
import sys

TAGS = [
    ("case_relevant", "Relates to the child, parenting time, custody, co-parenting, money for the child, or court proceedings"),
    ("parenting_time_denial", "Refuses, cancels, shortens, delays, or puts conditions on a parent's time or contact with the child"),
    ("child_referenced", "Mentions or discusses the child (Kailah)"),
    ("info_gatekeeping", "Withholds or refuses information about the child's school, health, location, or activities"),
    ("hostility_threat", "Insults, threats, intimidation, or demeaning language"),
    ("parent_disparagement", "Criticizes or runs down a parent's character or fitness as a parent"),
    ("financial", "About child support, payments, expenses, or money"),
    ("logistics", "About scheduling, pickup, drop-off, or exchange arrangements"),
    ("legal_reference", "Mentions court, attorneys, orders, filings, police, or the Friend of the Court"),
    ("third_party", "Involves a third party in the child's life, such as a partner, relative, or Mike Joubran"),
    ("child_wellbeing", "Addresses the child's health, safety, school, or emotional state"),
    ("cooperation_offer", "Offers flexibility, a compromise, or extra time"),
    ("admission", "The sender admits, concedes, or apologizes for something"),
    ("reframes_prior_event", "Denies or recasts something that happened earlier in the conversation (experimental)"),
    ("blame_shift", "Shifts responsibility for a problem onto the other person (experimental)"),
]
SOURCES = {"A_fb": "Facebook", "B_sms_2021_22": "Texts 2021–22", "C_sms_her_phone": "Texts 2024 · her phone",
           "D_sms_2025_26": "Texts 2025–26"}

sample_path, labels_path, out_path = map(pathlib.Path, sys.argv[1:4])
sample = [json.loads(line) for line in sample_path.read_text(encoding="utf-8").splitlines() if line.strip()]
labels = {json.loads(line)["msg_id"]: json.loads(line) for line in labels_path.read_text(encoding="utf-8").splitlines() if line.strip()}

data = []
for i, s in enumerate(sample, 1):
    lab = labels[s["msg_id"]]
    data.append({
        "k": f"m{i:03d}", "id": s["msg_id"], "src": s["stratum"], "ts": s["ts_utc"][:16].replace("T", " "),
        "who": s["sender_label"], "text": s["text"],
        "ctx": [{"who": c["sender"], "ts": (c["ts"] or "")[:16].replace("T", " "), "t": c["text"]} for c in s["prior_context"]],
        "opus": {t: bool(lab[t]) for t, _ in TAGS}, "reg": lab["register"], "why": lab["rationales"],
    })
data.sort(key=lambda d: d["ts"])

TEMPLATE = r"""<title>Opus Label Review</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Public+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap">
<style>
:root {
  --ground:#f3f4f6; --panel:#ffffff; --ink:#191c21; --muted:#5d6470; --line:#dcdfe5; --soft:#eceef2;
  --accent:#2f5d8a; --accent-ink:#ffffff; --matt:#1f7a6d; --kat:#6a4bb5;
  --on:#2f5d8a; --on-bg:#e3ecf6; --changed:#a8611a; --changed-bg:#fbeedd; --ok:#2e7d4f; --ok-bg:#e3f2e8;
  --focus:#2f5d8a;
}
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    color-scheme: dark;
    --ground:#121418; --panel:#1b1e24; --ink:#e6e8ec; --muted:#9aa1ad; --line:#2d323b; --soft:#23272f;
    --accent:#7fa8d6; --accent-ink:#0f1318; --matt:#5fc0b0; --kat:#a99ae6;
    --on:#8db3de; --on-bg:#223245; --changed:#e2a25c; --changed-bg:#3a2a18; --ok:#6cc792; --ok-bg:#1d3326;
    --focus:#8db3de;
  }
}
:root[data-theme="dark"] {
  color-scheme: dark;
  --ground:#121418; --panel:#1b1e24; --ink:#e6e8ec; --muted:#9aa1ad; --line:#2d323b; --soft:#23272f;
  --accent:#7fa8d6; --accent-ink:#0f1318; --matt:#5fc0b0; --kat:#a99ae6;
  --on:#8db3de; --on-bg:#223245; --changed:#e2a25c; --changed-bg:#3a2a18; --ok:#6cc792; --ok-bg:#1d3326;
  --focus:#8db3de;
}
* { box-sizing: border-box; }
body { margin:0; background:var(--ground); color:var(--ink); font:15px/1.5 "Public Sans", system-ui, "Segoe UI", sans-serif; padding-inline:16px; padding-block:20px 80px; }
.wrap { max-width:1040px; margin:0 auto; display:grid; gap:18px; }
h1 { font-size:24px; font-weight:700; margin:0; text-wrap:balance; letter-spacing:-0.01em; }
h2 { font-size:16px; font-weight:700; margin:0; }
.lede { color:var(--muted); margin:4px 0 0; max-width:70ch; }
.mono { font-family:"IBM Plex Mono", ui-monospace, Consolas, monospace; font-size:12.5px; }
.panel { background:var(--panel); border:1px solid var(--line); border-radius:10px; padding:16px; }
.save { font-size:13px; color:var(--muted); }
.save.err { color:var(--changed); }
.progress { display:flex; flex-wrap:wrap; gap:6px 18px; align-items:center; font-size:14px; }
.progress b { font-variant-numeric:tabular-nums; }
.bar { flex:1 1 220px; height:8px; background:var(--soft); border-radius:99px; overflow:hidden; min-width:160px; }
.bar i { display:block; height:100%; background:var(--ok); width:0; transition:width .3s; }
details.defs > summary { cursor:pointer; font-weight:700; }
.defgrid { display:grid; gap:12px; margin-top:12px; }
.def label { display:block; font-weight:600; font-size:13.5px; }
.def .opusdef { color:var(--muted); font-size:13px; margin:2px 0 6px; }
textarea, select, input[type=search] { font:inherit; font-size:14px; color:var(--ink); background:var(--panel); border:1px solid var(--line); border-radius:8px; padding:8px 10px; width:100%; }
textarea { min-height:60px; resize:vertical; }
:focus-visible { outline:2px solid var(--focus); outline-offset:2px; }
.filters { position:sticky; top:env(safe-area-inset-top, 0px); z-index:5; background:var(--ground); padding-block:10px; display:flex; flex-wrap:wrap; gap:8px; align-items:center; border-bottom:1px solid var(--line); }
.filters select, .filters input { width:auto; flex:1 1 150px; padding:6px 8px; }
.filters .count { color:var(--muted); font-size:13px; flex:0 0 auto; }
.msg { background:var(--panel); border:1px solid var(--line); border-left:4px solid var(--line); border-radius:10px; padding:14px 16px; display:grid; gap:10px; }
.msg.matt { border-left-color:var(--matt); } .msg.kat { border-left-color:var(--kat); }
.msg.done { box-shadow: inset 0 0 0 1px var(--ok-bg); }
.head { display:flex; flex-wrap:wrap; gap:6px 12px; align-items:baseline; font-size:13px; color:var(--muted); }
.who { font-weight:700; } .matt .who { color:var(--matt); } .kat .who { color:var(--kat); }
.status { margin-left:auto; font-size:12px; font-weight:600; padding:2px 8px; border-radius:99px; background:var(--soft); color:var(--muted); }
.status.agree { background:var(--ok-bg); color:var(--ok); } .status.corrected { background:var(--changed-bg); color:var(--changed); } .status.commented { background:var(--on-bg); color:var(--on); }
.text { font-size:16px; white-space:pre-wrap; overflow-wrap:anywhere; }
details.ctx summary { cursor:pointer; font-size:13px; color:var(--muted); }
details.ctx .line { font-size:13.5px; padding:3px 0; border-bottom:1px dashed var(--line); white-space:pre-wrap; overflow-wrap:anywhere; }
details.ctx .line span { color:var(--muted); font-size:12px; margin-right:6px; }
.tags { display:flex; flex-wrap:wrap; gap:6px; }
.tag { font:500 12.5px "IBM Plex Mono", ui-monospace, Consolas, monospace; border:1px solid var(--line); background:var(--panel); color:var(--muted); border-radius:6px; padding:4px 8px; cursor:pointer; }
.tag.on { background:var(--on-bg); border-color:var(--on); color:var(--on); }
.tag.changed { border-style:dashed; border-color:var(--changed); color:var(--changed); background:var(--changed-bg); }
.tag.gate { font-weight:600; }
.why { font-size:13px; color:var(--muted); display:grid; gap:2px; }
.why b { color:var(--ink); font-weight:600; font-family:"IBM Plex Mono", ui-monospace, Consolas, monospace; font-size:12px; }
.row { display:flex; flex-wrap:wrap; gap:8px; align-items:center; }
.row select { width:auto; padding:5px 8px; }
button.act { font:inherit; font-size:13px; font-weight:600; border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:8px; padding:6px 12px; cursor:pointer; }
button.act.primary { background:var(--accent); color:var(--accent-ink); border-color:var(--accent); }
.legend { font-size:12.5px; color:var(--muted); display:flex; flex-wrap:wrap; gap:10px; }
.legend .tag { cursor:default; }
@media (prefers-reduced-motion: reduce) { .bar i { transition:none; } }
</style>
<div class="wrap">
  <header>
    <h1>Opus Label Review</h1>
    <p class="lede">300 messages between Matt and Katrina, each labelled by Opus. Click a label to change it; changes save automatically. Nothing goes to Jev until you're done here.</p>
  </header>

  <section class="panel progress" aria-live="polite">
    <span>Reviewed <b id="n-done">0</b> / <b id="n-all">300</b></span>
    <div class="bar"><i id="bar"></i></div>
    <span>Opus right <b id="n-agree">0</b></span>
    <span>Corrected <b id="n-corr">0</b></span>
    <span class="save" id="save-state">Connecting…</span>
  </section>

  <details class="panel defs">
    <summary>What each label should mean (your words). Start with case_relevant</summary>
    <p class="lede">Opus used the wording in grey. Write what the label means to you, or what Opus is getting wrong. This is what I'll use to fix the definitions.</p>
    <div class="defgrid" id="defs"></div>
    <div class="def"><label for="def-general">Anything else about the labels</label><textarea id="def-general" placeholder="General notes"></textarea></div>
  </details>

  <div class="legend"><span class="tag on">Opus: yes</span><span class="tag">Opus: no</span><span class="tag changed">you changed it</span></div>

  <div class="filters" role="search">
    <select id="f-tag" aria-label="Label"><option value="">Any label</option><option value="__none">Opus: no label</option></select>
    <select id="f-src" aria-label="Source"><option value="">All sources</option></select>
    <select id="f-who" aria-label="Sender"><option value="">Both senders</option><option>Matt</option><option>Katrina</option></select>
    <select id="f-status" aria-label="Review status"><option value="">Any status</option><option value="todo">Not reviewed</option><option value="agree">Opus right</option><option value="corrected">Corrected</option><option value="commented">Comment only</option></select>
    <input type="search" id="f-q" placeholder="Search text" aria-label="Search text">
    <span class="count" id="count"></span>
  </div>

  <div id="list" class="wrap"></div>
</div>
<script>
const DATA = __DATA__;
const TAGS = __TAGS__;
const SOURCES = __SOURCES__;
const REGISTERS = ["factual", "opinion", "emotional", "mixed"];
const reviews = {};   // k -> saved review doc
let notes = {};       // tag -> text, plus general
let db = null;

const $ = id => document.getElementById(id);
const el = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text != null) e.textContent = text; return e; };
const byK = Object.fromEntries(DATA.map(d => [d.k, d]));

function answer(d) {
  const r = reviews[d.k];
  const tags = Object.assign({}, d.opus, r && r.tags ? r.tags : {});
  return { tags, reg: (r && r.reg) || d.reg, comment: (r && r.comment) || "" };
}
function statusOf(d) {
  const r = reviews[d.k];
  if (!r) return "todo";
  const a = answer(d);
  const changed = TAGS.some(([t]) => a.tags[t] !== d.opus[t]) || a.reg !== d.reg;
  if (changed) return "corrected";
  if (r.agree) return "agree";
  return a.comment.trim() ? "commented" : "todo";
}
const STATUS_TEXT = { todo: "Not reviewed", agree: "Opus right", corrected: "Corrected", commented: "Comment" };

// ---- saving: one write in flight per doc, latest state wins ----
const timers = {}, inflight = {}, queued = {};
function setSave(text, err) { const s = $("save-state"); s.textContent = text; s.classList.toggle("err", !!err); }
function scheduleSave(path, payload) {
  queued[path] = payload;
  clearTimeout(timers[path]);
  timers[path] = setTimeout(() => flush(path), 600);
  setSave("Saving…");
}
async function flush(path) {
  if (!db) { setSave("Not saved: storage unavailable in this view", true); return; }
  if (inflight[path]) return;
  const payload = queued[path]; delete queued[path];
  if (!payload) return;
  inflight[path] = true;
  try { await db.doc(path).set(payload); setSave("Saved " + new Date().toLocaleTimeString()); }
  catch (e) { setSave("Not saved (" + (e && e.code || "error") + "). Your change is still on screen.", true); }
  finally { inflight[path] = false; if (queued[path]) flush(path); }
}
function saveReview(d, patch) {
  const cur = reviews[d.k] || {};
  const a = answer(d);
  const next = Object.assign({ msg_id: d.id, k: d.k, tags: a.tags, reg: a.reg, comment: a.comment, agree: !!cur.agree }, patch, { updated_at: new Date().toISOString() });
  reviews[d.k] = next;
  scheduleSave("reviews/" + d.k, next);
  refreshCard(d); refreshProgress();
}

// ---- definitions panel ----
function buildDefs() {
  const box = $("defs");
  for (const [t, desc] of TAGS) {
    const w = el("div", "def");
    const lab = el("label", null, t); lab.htmlFor = "def-" + t;
    const g = el("div", "opusdef", "Opus used: " + desc);
    const ta = el("textarea"); ta.id = "def-" + t; ta.placeholder = t === "case_relevant" ? "What makes a message case-relevant?" : "Your definition or correction";
    ta.addEventListener("input", () => { notes[t] = ta.value; scheduleSave("notes/definitions", Object.assign({}, notes, { updated_at: new Date().toISOString() })); });
    w.append(lab, g, ta); box.append(w);
  }
  $("def-general").addEventListener("input", e => { notes.general = e.target.value; scheduleSave("notes/definitions", Object.assign({}, notes, { updated_at: new Date().toISOString() })); });
}
function fillDefs() {
  for (const [t] of TAGS) { const ta = $("def-" + t); if (ta && document.activeElement !== ta) ta.value = notes[t] || ""; }
  const g = $("def-general"); if (g && document.activeElement !== g) g.value = notes.general || "";
}

// ---- message cards ----
const cards = {};
function buildCard(d) {
  const card = el("article", "msg " + (d.who === "Matt" ? "matt" : "kat")); card.id = "card-" + d.k;
  const head = el("div", "head");
  head.append(el("span", "who", d.who), el("span", null, d.ts + " UTC"), el("span", null, SOURCES[d.src]), el("span", "mono", d.k));
  const st = el("span", "status"); head.append(st);
  const text = el("div", "text", d.text);
  const ctx = el("details", "ctx"); ctx.append(el("summary", null, "Earlier messages (" + d.ctx.length + ")"));
  for (const c of d.ctx) { const line = el("div", "line"); line.append(el("span", null, c.ts + " " + c.who)); line.append(document.createTextNode(c.t)); ctx.append(line); }
  const tags = el("div", "tags");
  const btns = {};
  for (const [t, desc] of TAGS) {
    const b = el("button", "tag" + (t === "case_relevant" ? " gate" : ""), t); b.type = "button"; b.id = "t-" + d.k + "-" + t; b.title = desc;
    b.addEventListener("click", () => { const a = answer(d); const tagsNext = Object.assign({}, a.tags, { [t]: !a.tags[t] }); saveReview(d, { tags: tagsNext, agree: false }); });
    btns[t] = b; tags.append(b);
  }
  const why = el("div", "why");
  for (const [t] of TAGS) if (d.opus[t] && d.why[t]) { const line = el("div"); line.append(el("b", null, t + " "), document.createTextNode(d.why[t])); why.append(line); }
  const row = el("div", "row");
  const regLab = el("label", "save", "Register"); regLab.htmlFor = "r-" + d.k;
  const reg = el("select"); reg.id = "r-" + d.k;
  for (const r of REGISTERS) { const o = el("option", null, r + (r === d.reg ? " (Opus)" : "")); o.value = r; reg.append(o); }
  reg.addEventListener("change", () => saveReview(d, { reg: reg.value, agree: false }));
  const agree = el("button", "act primary", "Opus is right"); agree.type = "button"; agree.id = "a-" + d.k;
  agree.addEventListener("click", () => saveReview(d, { tags: Object.assign({}, d.opus), reg: d.reg, agree: true }));
  row.append(regLab, reg, agree);
  const com = el("textarea"); com.id = "c-" + d.k; com.placeholder = "Your note on this message (optional)";
  com.addEventListener("input", () => saveReview(d, { comment: com.value }));
  card.append(head, text, ctx, tags, why, row, com);
  cards[d.k] = { card, st, btns, reg, com };
  refreshCard(d);
  return card;
}
function refreshCard(d) {
  const c = cards[d.k]; if (!c) return;
  const a = answer(d), s = statusOf(d);
  c.st.textContent = STATUS_TEXT[s]; c.st.className = "status " + s;
  c.card.classList.toggle("done", s !== "todo");
  for (const [t] of TAGS) {
    const on = a.tags[t], changed = on !== d.opus[t];
    c.btns[t].className = "tag" + (t === "case_relevant" ? " gate" : "") + (on ? " on" : "") + (changed ? " changed" : "");
    c.btns[t].setAttribute("aria-pressed", on ? "true" : "false");
  }
  if (document.activeElement !== c.reg) c.reg.value = a.reg;
  if (document.activeElement !== c.com) c.com.value = a.comment;
}
function refreshProgress() {
  let done = 0, agree = 0, corr = 0;
  for (const d of DATA) { const s = statusOf(d); if (s !== "todo") done++; if (s === "agree") agree++; if (s === "corrected") corr++; }
  $("n-done").textContent = done; $("n-agree").textContent = agree; $("n-corr").textContent = corr;
  $("bar").style.width = (100 * done / DATA.length) + "%";
}
function applyFilters() {
  const tag = $("f-tag").value, src = $("f-src").value, who = $("f-who").value, st = $("f-status").value, q = $("f-q").value.trim().toLowerCase();
  let n = 0;
  for (const d of DATA) {
    const ok = (!src || d.src === src) && (!who || d.who === who) && (!st || statusOf(d) === st) &&
      (!q || d.text.toLowerCase().includes(q)) &&
      (!tag || (tag === "__none" ? !TAGS.some(([t]) => d.opus[t]) : d.opus[tag]));
    cards[d.k].card.hidden = !ok; if (ok) n++;
  }
  $("count").textContent = n + " shown";
}

function init() {
  $("n-all").textContent = DATA.length;
  for (const [t] of TAGS) { const o = el("option", null, "Opus: " + t); o.value = t; $("f-tag").append(o); }
  for (const [k, v] of Object.entries(SOURCES)) { const o = el("option", null, v); o.value = k; $("f-src").append(o); }
  buildDefs();
  const list = $("list");
  for (const d of DATA) list.append(buildCard(d));
  ["f-tag", "f-src", "f-who", "f-status"].forEach(id => $(id).addEventListener("change", applyFilters));
  $("f-q").addEventListener("input", applyFilters);
  applyFilters(); refreshProgress();
  connect();
}
async function connect() {
  db = await window.claude?.use?.("db") ?? null;
  if (!db) { setSave("Saving unavailable in this view: changes stay on screen only", true); return; }
  setSave("Connected");
  db.collection("reviews").onSnapshot(qs => {
    for (const doc of qs.docs) { const v = doc.data(); if (v && v.k && !queued["reviews/" + v.k] && !inflight["reviews/" + v.k]) reviews[v.k] = v; }
    for (const d of DATA) refreshCard(d);
    refreshProgress(); applyFilters();
  }, e => setSave("Live sync stopped (" + (e && e.code || "error") + "); reload to reconnect", true));
  db.doc("notes/definitions").onSnapshot(snap => { if (snap.exists && !queued["notes/definitions"]) { notes = snap.data() || {}; fillDefs(); } });
}
init();
</script>
"""

page = (TEMPLATE.replace("__DATA__", json.dumps(data, ensure_ascii=False))
        .replace("__TAGS__", json.dumps(TAGS, ensure_ascii=False))
        .replace("__SOURCES__", json.dumps(SOURCES, ensure_ascii=False)))
out_path.write_text(page, encoding="utf-8")
print(f"wrote {out_path} ({len(data)} messages, {len(page)//1024} KB)")
