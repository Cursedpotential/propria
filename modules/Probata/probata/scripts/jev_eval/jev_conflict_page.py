"""Build the owner's Opus-vs-Jev disagreement page: every 2024 bout where the two models name a different main tone.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 13:46 "Let me see the conflict ones" after the Jev run (bout-q-v1,
645 bouts, main tone agrees with Opus on 56.6%). For each bout the owner says who was right: Opus, Jev, or neither,
plus a note. Answers save to the artifact's `db` (collection `conflicts`, doc per bout id) and are read back with
read_db. The output embeds case messages: build it on the devbox or in a scratch path, never commit it.

Usage: python jev_conflict_page.py <bouts.jsonl> <raw/bout_tone dir> <raw/jev_bouts dir> <out.html>
"""

import collections
import json
import pathlib
import sys

TONES = ["affectionate", "friendly", "neutral", "tense", "hostile", "distressed", "conciliatory"]
FLAGS = ["any_warm", "any_conflict", "any_hostile", "any_distress", "any_conciliatory", "tone_changes", "abrupt_change"]
bouts_path, tone_dir, jev_dir, out_path = (pathlib.Path(a) for a in sys.argv[1:5])
bouts = {b["bout_id"]: b for b in (json.loads(x) for x in bouts_path.read_text(encoding="utf-8").split("\n") if x.strip())}

opus = {}
for p in tone_dir.glob("c2024-*.json"):
    d = json.loads(p.read_text(encoding="utf-8"))
    if d.get("ok"):
        opus[d["bout_id"]] = d["output"]

jev = collections.defaultdict(list)
for p in jev_dir.glob("c2024-*.json"):
    r = json.loads(p.read_text(encoding="utf-8"))
    try:
        ans = json.loads(r["response_raw"])["answers"]
    except (KeyError, ValueError, TypeError):
        continue
    jev[r["bout_id"]].append((r["n_messages"], r["from_i"], ans))

data, pairs, agree = [], collections.Counter(), 0
for bid, out in sorted(opus.items()):
    if bid not in jev or bid not in bouts:
        continue
    st = out.get("stretches", [])
    o_main = max(st, key=lambda s: (s["to_i"] - s["from_i"], -s["from_i"]))["tone"] if st else None
    wins = sorted(jev[bid], key=lambda w: w[1])
    big = max(wins, key=lambda w: w[0])[2]
    j_main = (big.get("main_tone") or {}).get("choice")
    if o_main == j_main:
        agree += 1
        continue
    probs = (big.get("main_tone") or {}).get("probabilities") or {}
    flags = {f: max(((w[2].get(f) or {}).get("noul") or 0) for w in wins) for f in FLAGS}
    driver = (big.get("conflict_driver") or {}).get("choice")
    b = bouts[bid]
    pairs[(o_main, j_main)] += 1
    data.append({
        "id": bid, "day": b["day"], "s": b["start_local"][11:16], "e": b["end_local"][11:16],
        "m": [[m["ts_local"], 0 if m["who"] == "Matt" else 1, m["text"]] for m in b["messages"]],
        "o": TONES.index(o_main), "j": TONES.index(j_main),
        "st": [[s["from_i"], s["to_i"], TONES.index(s["tone"]), s.get("driver", ""), s.get("note", "")] for s in st],
        "sum": out.get("summary", ""),
        "p": [[TONES.index(t), round(v, 2)] for t, v in sorted(probs.items(), key=lambda x: -x[1]) if v >= 0.05][:4],
        "f": {k: round(v, 2) for k, v in flags.items()}, "drv": driver, "pieces": len(wins)})

meta = {"total": len(data) + agree, "agree": agree, "conflicts": len(data),
        "pairs": [[TONES.index(a), TONES.index(b), n] for (a, b), n in pairs.most_common()]}

TEMPLATE = r"""<title>Tone Disagreements · 2024 Texts</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap">
<style>
/* Palette: Propria design contract 1.0.0 (same tokens as the Bout Review page). Tone colours are data colours. */
:root {
  --ground:#f5f3ee; --panel:#fffefb; --ink:#1d2228; --muted:#687078; --line:#d5d1c9; --soft:#ebe8e0; --focus:#4051b9;
  --accent:#4051b9; --accent-ink:#fff; --ok:#247047; --ok-bg:#e2efe6; --bad:#b5433b; --bad-bg:#f8e5e3; --jev:#6b4fa3; --jev-bg:#ece6f6;
  --matt:#376f72; --kat:#8a4f7d;
  --t0:#c2477a; --t1:#3d8b5a; --t2:#9aa1ab; --t3:#c98a1c; --t4:#c0392b; --t5:#7a4bb5; --t6:#1f8a8a;
}
@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --accent-ink:#111820; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a; --jev:#b9a2ec; --jev-bg:#352c4a;
  --matt:#82bdc0; --kat:#d69cc6;
  --t0:#e98ab0; --t1:#7cc79a; --t2:#7d8791; --t3:#e2b25a; --t4:#ef7b6e; --t5:#b39be6; --t6:#6cc9c9; } }
:root[data-theme="dark"] {
  color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --focus:#8591f0;
  --accent:#8591f0; --accent-ink:#111820; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a; --jev:#b9a2ec; --jev-bg:#352c4a;
  --matt:#82bdc0; --kat:#d69cc6;
  --t0:#e98ab0; --t1:#7cc79a; --t2:#7d8791; --t3:#e2b25a; --t4:#ef7b6e; --t5:#b39be6; --t6:#6cc9c9; }
* { box-sizing:border-box; }
body { margin:0; background:var(--ground); color:var(--ink); font:15px/1.5 "Instrument Sans", "Segoe UI", system-ui, sans-serif; padding-inline:16px; padding-block:20px 80px; }
.wrap { max-width:1100px; margin:0 auto; display:grid; gap:16px; }
h1 { font-size:24px; margin:0; letter-spacing:-.01em; text-wrap:balance; }
h2 { font-size:15px; margin:0; }
.lede { color:var(--muted); margin:4px 0 0; max-width:72ch; }
.mono { font-family:"IBM Plex Mono", ui-monospace, Consolas, monospace; font-size:12.5px; font-variant-numeric:tabular-nums; }
.panel { background:var(--panel); border:1px solid var(--line); border-radius:6px; padding:14px 16px; }
.facts { display:flex; flex-wrap:wrap; gap:6px 20px; font-size:14px; }
.facts b { font-variant-numeric:tabular-nums; }
.pairs { display:grid; grid-template-columns:repeat(auto-fill, minmax(230px, 1fr)); gap:6px; margin-top:10px; }
.pair { display:flex; align-items:center; gap:8px; font:inherit; font-size:13.5px; text-align:left; background:var(--panel); color:var(--ink); border:1px solid var(--line); border-radius:4px; padding:6px 10px; cursor:pointer; }
.pair:hover { background:var(--soft); }
.pair.on { border-color:var(--accent); box-shadow:inset 0 0 0 1px var(--accent); }
.pair .n { margin-left:auto; font:500 12.5px "IBM Plex Mono", ui-monospace, monospace; color:var(--muted); }
.chip { display:inline-flex; align-items:center; gap:5px; font-size:13px; font-weight:600; padding:2px 8px; border-radius:999px; border:1px solid var(--line); background:var(--panel); white-space:nowrap; }
.chip .sw, .sw { display:inline-block; width:10px; height:10px; border-radius:2px; }
.arrow { color:var(--muted); }
.filters { position:sticky; top:env(safe-area-inset-top, 0px); z-index:5; background:var(--ground); padding-block:10px; display:flex; flex-wrap:wrap; gap:8px; align-items:center; border-bottom:1px solid var(--line); }
select, textarea { font:inherit; font-size:14px; color:var(--ink); background:var(--panel); border:1px solid var(--line); border-radius:4px; padding:6px 8px; }
.filters select { flex:1 1 170px; }
.filters .count { color:var(--muted); font-size:13px; }
.save { font-size:13px; color:var(--muted); } .save.err { color:var(--bad); }
:focus-visible { outline:2px solid var(--focus); outline-offset:2px; }
.bout { background:var(--panel); border:1px solid var(--line); border-radius:6px; padding:12px 14px; display:grid; gap:10px; }
.bhead { display:flex; flex-wrap:wrap; gap:6px 12px; align-items:baseline; font-size:13px; color:var(--muted); }
.bhead .d { font-weight:700; color:var(--ink); font-size:14px; }
.calls { display:grid; grid-template-columns:1fr 1fr; gap:10px; }
@media (max-width:640px) { .calls { grid-template-columns:1fr; } }
.call { border:1px solid var(--line); border-radius:4px; padding:8px 10px; display:grid; gap:6px; align-content:start; }
.call .lab { font-size:11.5px; font-weight:700; letter-spacing:.06em; text-transform:uppercase; color:var(--muted); }
.call.jev .lab { color:var(--jev); }
.call .txt { font-size:13.5px; }
.probs { display:flex; flex-wrap:wrap; gap:4px 10px; font-size:12.5px; color:var(--muted); }
.probs b { font-variant-numeric:tabular-nums; color:var(--ink); font-weight:600; }
.flags { display:flex; flex-wrap:wrap; gap:4px; }
.flag { font-size:12px; padding:1px 7px; border-radius:3px; background:var(--jev-bg); color:var(--jev); font-variant-numeric:tabular-nums; }
.msgs { display:grid; gap:1px; }
.msg { display:grid; grid-template-columns:4px 46px 60px 1fr; gap:8px; align-items:start; padding:2px 0; font-size:14px; }
.msg .tone { align-self:stretch; border-radius:2px; }
.msg .time { font:12px "IBM Plex Mono", ui-monospace, monospace; color:var(--muted); padding-top:2px; }
.msg .who { font-weight:600; font-size:13px; padding-top:1px; }
.msg .who.m { color:var(--matt); } .msg .who.k { color:var(--kat); }
.msg .txt { white-space:pre-wrap; overflow-wrap:anywhere; }
.stnote { font-size:12.5px; color:var(--muted); margin-left:58px; }
.fb { display:flex; flex-wrap:wrap; gap:6px; align-items:center; }
.fb button { font:inherit; font-size:13px; font-weight:600; border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:4px; padding:5px 10px; cursor:pointer; }
.fb button.on.opus { background:var(--ok-bg); color:var(--ok); border-color:var(--ok); }
.fb button.on.jev { background:var(--jev-bg); color:var(--jev); border-color:var(--jev); }
.fb button.on.neither { background:var(--bad-bg); color:var(--bad); border-color:var(--bad); }
.fb textarea { flex:1 1 280px; min-height:36px; }
.more { font:inherit; font-weight:600; justify-self:start; background:var(--accent); color:var(--accent-ink); border:0; border-radius:4px; padding:8px 14px; cursor:pointer; }
.empty { color:var(--muted); }
</style>
<div class="wrap">
  <header>
    <h1>Tone Disagreements · 2024 Texts</h1>
    <p class="lede">Bouts from Katrina's phone where Opus and Jev named a different main tone. The coloured bar beside each message is Opus's tone for that stretch. Read the messages, then say who got the bout's tone right.</p>
  </header>
  <section class="panel">
    <div class="facts" id="facts"></div>
    <div class="pairs" id="pairs" role="group" aria-label="Kind of disagreement"></div>
  </section>
  <div class="filters" role="search">
    <select id="f-pair" aria-label="Kind of disagreement"><option value="">Every disagreement</option></select>
    <select id="f-month" aria-label="Month"><option value="">All months</option></select>
    <select id="f-review" aria-label="Your answer"><option value="">Any answer</option><option value="todo">Not answered</option><option value="opus">Opus right</option><option value="jev">Jev right</option><option value="neither">Both wrong</option></select>
    <span class="count" id="count"></span><span class="save" id="save-state">Connecting…</span>
  </div>
  <div id="list" class="wrap"></div>
</div>
<script>
const DATA = __DATA__;
const META = __META__;
const TONES = __TONES__;
const FLAG_NAMES = {any_warm:"warmth", any_conflict:"conflict", any_hostile:"hostility", any_distress:"distress", any_conciliatory:"making up", tone_changes:"tone changes", abrupt_change:"abrupt change"};
const $ = id => document.getElementById(id);
const el = (t, c, x) => { const e = document.createElement(t); if (c) e.className = c; if (x != null) e.textContent = x; return e; };
const tv = i => "var(--t" + i + ")";
const WHO = ["Matt", "Katrina"];
const PAGE = 25;
const answers = {}; let db = null, shown = PAGE;

function chip(i, prefix) { const c = el("span", "chip"); const s = el("span", "sw"); s.style.background = tv(i); c.append(s, document.createTextNode((prefix || "") + TONES[i])); return c; }
function facts() {
  const f = $("facts");
  const add = (label, v) => { const s = el("span"); s.append(document.createTextNode(label + " ")); s.append(el("b", null, v)); f.append(s); };
  add("Bouts scored", META.total); add("Same main tone", META.agree + " (" + Math.round(100 * META.agree / META.total) + "%)");
  add("Disagree", META.conflicts);
  for (const [o, j, n] of META.pairs) {
    const b = el("button", "pair"); b.type = "button"; b.dataset.key = o + ">" + j;
    b.append(chip(o, "Opus "), el("span", "arrow", "→"), chip(j, "Jev "), el("span", "n", String(n)));
    b.addEventListener("click", () => { $("f-pair").value = $("f-pair").value === b.dataset.key ? "" : b.dataset.key; shown = PAGE; render(); });
    $("pairs").append(b);
    const opt = el("option", null, "Opus " + TONES[o] + " → Jev " + TONES[j] + " (" + n + ")"); opt.value = o + ">" + j; $("f-pair").append(opt);
  }
}
function toneOf(b, i) { for (const s of b.st) if (i >= s[0] && i <= s[1]) return s; return null; }
function card(b) {
  const c = el("article", "bout"); c.id = "bout-" + b.id;
  const nM = b.m.filter(m => m[1] === 0).length;
  const head = el("div", "bhead");
  head.append(el("span", "d", b.day), el("span", "mono", b.s + "–" + b.e), el("span", null, b.m.length + " messages · Matt " + nM + " · Katrina " + (b.m.length - nM)), el("span", "mono", b.id));
  c.append(head);
  const calls = el("div", "calls");
  const o = el("div", "call"); o.append(el("span", "lab", "Opus says"), chip(b.o)); if (b.sum) o.append(el("div", "txt", b.sum));
  const j = el("div", "call jev"); j.append(el("span", "lab", "Jev says" + (b.pieces > 1 ? " (long bout, read in " + b.pieces + " pieces)" : "")), chip(b.j));
  const pr = el("div", "probs"); for (const [t, v] of b.p) { const s = el("span"); s.append(document.createTextNode(TONES[t] + " ")); s.append(el("b", null, Math.round(v * 100) + "%")); pr.append(s); } j.append(pr);
  const fl = el("div", "flags"); for (const k in b.f) if (b.f[k] >= 0.5) fl.append(el("span", "flag", FLAG_NAMES[k] + " " + Math.round(b.f[k] * 100) + "%"));
  if (b.drv && b.drv !== "no conflict") fl.append(el("span", "flag", "conflict driven by " + b.drv));
  if (fl.childElementCount) j.append(fl);
  calls.append(o, j); c.append(calls);
  const list = el("div", "msgs"); const stStart = new Map(b.st.map(s => [s[0], s]));
  b.m.forEach((m, i) => {
    const s0 = stStart.get(i); if (s0 && b.st.length > 1) list.append(el("div", "stnote", "Opus: " + TONES[s0[2]] + (s0[3] ? " · driven by " + s0[3] : "") + (s0[4] ? " · " + s0[4] : "")));
    const row = el("div", "msg"); const bar = el("span", "tone"); const s = toneOf(b, i); bar.style.background = s ? tv(s[2]) : "var(--line)";
    row.append(bar, el("span", "time", m[0]), el("span", "who " + (m[1] ? "k" : "m"), WHO[m[1]]), el("span", "txt", m[2])); list.append(row);
  });
  c.append(list);
  const fb = el("div", "fb"); const btns = {};
  for (const [k, label] of [["opus", "Opus right"], ["jev", "Jev right"], ["neither", "Both wrong"]]) {
    const bt = el("button", k, label); bt.type = "button"; bt.id = "v-" + b.id + "-" + k;
    bt.addEventListener("click", () => save(b, { verdict: (answers[b.id] || {}).verdict === k ? "" : k }));
    btns[k] = bt; fb.append(bt);
  }
  const ta = el("textarea"); ta.id = "n-" + b.id; ta.placeholder = "What's the right tone, and why? (optional)";
  ta.addEventListener("input", () => save(b, { note: ta.value }));
  fb.append(ta); c.append(fb); c._btns = btns; c._ta = ta; paint(b, c);
  return c;
}
function paint(b, c) {
  c = c || $("bout-" + b.id); if (!c) return; const r = answers[b.id] || {};
  for (const k in c._btns) { c._btns[k].classList.toggle("on", r.verdict === k); c._btns[k].setAttribute("aria-pressed", r.verdict === k ? "true" : "false"); }
  if (document.activeElement !== c._ta) c._ta.value = r.note || "";
}
function matches(b) {
  const p = $("f-pair").value, mo = $("f-month").value, rv = $("f-review").value, v = (answers[b.id] || {}).verdict || "";
  if (p && p !== b.o + ">" + b.j) return false;
  if (mo && !b.day.startsWith(mo)) return false;
  if (rv === "todo") return !v;
  if (rv && v !== rv) return false;
  return true;
}
function render() {
  const hits = DATA.filter(matches); const box = $("list"); box.replaceChildren();
  for (const b of hits.slice(0, shown)) box.append(card(b));
  if (!hits.length) box.append(el("p", "empty", "No bouts match these filters."));
  if (hits.length > shown) { const m = el("button", "more", "Show " + Math.min(PAGE, hits.length - shown) + " more"); m.type = "button"; m.addEventListener("click", () => { shown += PAGE; render(); }); box.append(m); }
  $("count").textContent = hits.length + " bouts" + (hits.length > shown ? " · showing " + shown : "");
  document.querySelectorAll(".pair").forEach(b => b.classList.toggle("on", b.dataset.key === $("f-pair").value));
}
const timers = {}, inflight = {}, queued = {};
function setSave(t, err) { const s = $("save-state"); s.textContent = t; s.classList.toggle("err", !!err); }
function save(b, patch) {
  const next = Object.assign({ bout_id: b.id, day: b.day, opus: TONES[b.o], jev: TONES[b.j], verdict: "", note: "" }, answers[b.id] || {}, patch, { updated_at: new Date().toISOString() });
  answers[b.id] = next; paint(b);
  const path = "conflicts/" + b.id; queued[path] = next; clearTimeout(timers[path]);
  timers[path] = setTimeout(() => flush(path), 600); setSave("Saving…");
}
async function flush(path) {
  if (!db) { setSave("Not saved: storage unavailable in this view", true); return; }
  if (inflight[path]) return; const payload = queued[path]; delete queued[path]; if (!payload) return;
  inflight[path] = true;
  try { await db.doc(path).set(payload); setSave("Saved " + new Date().toLocaleTimeString()); }
  catch (e) { setSave("Not saved (" + (e && e.code || "error") + "). Your answer is still on screen.", true); }
  finally { inflight[path] = false; if (queued[path]) flush(path); }
}
async function connect() {
  db = await window.claude?.use?.("db") ?? null;
  if (!db) { setSave("Saving unavailable in this view: answers stay on screen only", true); return; }
  setSave("Connected");
  db.collection("conflicts").onSnapshot(qs => {
    for (const d of qs.docs) { const v = d.data(); if (v && v.bout_id && !queued["conflicts/" + v.bout_id] && !inflight["conflicts/" + v.bout_id]) answers[v.bout_id] = v; }
    for (const b of DATA) paint(b);
  }, e => setSave("Live sync stopped (" + (e && e.code || "error") + "); reload to reconnect", true));
}
function init() {
  facts();
  [...new Set(DATA.map(b => b.day.slice(0, 7)))].sort().forEach(m => { const o = el("option", null, m); o.value = m; $("f-month").append(o); });
  ["f-pair", "f-month", "f-review"].forEach(id => $(id).addEventListener("change", () => { shown = PAGE; render(); }));
  if (META.pairs.length) $("f-pair").value = META.pairs[0][0] + ">" + META.pairs[0][1];
  render(); connect();
}
init();
</script>
"""

page = (TEMPLATE.replace("__DATA__", json.dumps(data, ensure_ascii=False)).replace("__META__", json.dumps(meta))
        .replace("__TONES__", json.dumps(TONES)))
out_path.write_text(page, encoding="utf-8")
print(f"wrote {out_path}: {meta['conflicts']} disagreements of {meta['total']} bouts, {len(page) // 1024} KB")
