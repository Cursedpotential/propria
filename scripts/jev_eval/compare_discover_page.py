"""Build the owner's before/after page: discovery prompt v1 (test) next to v2 (narrow, observational), bout by bout.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 06:46: flag the bouts already run as a test, so he can see the
results before and after the prompt change and say which look better. Answers are saved to the artifact's `db`
(collection `compare`, one doc per bout id) and read back with read_db. The output embeds case messages; it is built
on the devbox and published as a private artifact, never committed.

Usage: python compare_discover_page.py <out.html> <label=dir>[,<label=dir>...] <bouts.jsonl> [<bouts.jsonl> ...]
Example: ... "v1 (test)=raw/bout_discover,v2=raw/bout_discover_v2,v2.1=raw/bout_discover_v21" bouts/f2024_bouts_v1.jsonl
Owner 07:17: add prompt v2.1 as a further column on the same bouts.
"""

import json
import pathlib
import sys

out_path = pathlib.Path(sys.argv[1])
VERS = [tuple(x.split("=", 1)) for x in sys.argv[2].split(",")]
bouts = {}
for f in sys.argv[3:]:
    for line in pathlib.Path(f).read_text(encoding="utf-8").split("\n"):
        if line.strip():
            b = json.loads(line)
            bouts[b["bout_id"]] = b


def load(d: pathlib.Path) -> dict:
    res = {}
    for p in d.glob("*-b*.json"):
        r = json.loads(p.read_text(encoding="utf-8"))
        if r.get("ok"):
            res[r["bout_id"]] = r["output"]
    return res


def slim(o: dict) -> dict:
    """v1-v2.1 observations have one `category`; v3 has `categories`, `what_happens` and a bout `topic`."""
    return {"sum": ((o["topic"] + ". ") if o.get("topic") else "") + o.get("summary", ""),
            "p": [[" + ".join(p.get("categories") or [p.get("category", "")]), p["from_guide"], p["who"], p["message_is"],
                   p["confidence"], (p.get("what_happens", "") + " " + p["note"]).strip(), p["quote"],
                   p.get("child_related", False)]
                  for p in o.get("patterns", [])],
            "new": [[c["name"], c["definition"]] for c in o.get("new_categories", [])]}


res = [load(pathlib.Path(d)) for _, d in VERS]
ids = sorted(set.intersection(*(set(r) for r in res)), key=lambda i: (bouts[i]["day"], bouts[i]["start_local"], i))
data = [{"id": i, "day": bouts[i]["day"], "s": bouts[i]["start_local"][11:16], "e": bouts[i]["end_local"][11:16],
         "src": "Facebook" if i.startswith("f") else "Texts",
         "m": [[m["ts_local"], m["who"], m["text"]] for m in bouts[i]["messages"]],
         "v": [slim(r[i]) for r in res]} for i in ids]
LABELS = [label for label, _ in VERS]

TEMPLATE = r"""<title>Prompt Compare · Discovery</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap">
<style>
/* Palette: Propria design contract 1.0.0 (Probata graphite/indigo), owner direction 2026-09-24. */
:root { --ground:#f5f3ee; --panel:#fffefb; --ink:#1d2228; --muted:#687078; --line:#d5d1c9; --soft:#ebe8e0; --accent:#4051b9;
  --ok:#247047; --ok-bg:#e2efe6; --bad:#b5433b; --bad-bg:#f8e5e3; --miss:#9a5a12; --miss-bg:#f7ecdc; --matt:#376f72; --kat:#8a4f7d; }
@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) { color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef;
  --muted:#b1b8bd; --line:#43505a; --soft:#2c373f; --accent:#8591f0; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a;
  --miss:#e6b55d; --miss-bg:#40341d; --matt:#82bdc0; --kat:#d69cc6; } }
:root[data-theme="dark"] { color-scheme:dark; --ground:#1d252c; --panel:#242e36; --ink:#f0f1ef; --muted:#b1b8bd; --line:#43505a;
  --soft:#2c373f; --accent:#8591f0; --ok:#6cc392; --ok-bg:#1f3a2b; --bad:#e06e65; --bad-bg:#43282a; --miss:#e6b55d; --miss-bg:#40341d;
  --matt:#82bdc0; --kat:#d69cc6; }
* { box-sizing:border-box; }
body { margin:0; background:var(--ground); color:var(--ink); font:15px/1.5 "Instrument Sans","Segoe UI",system-ui,sans-serif; padding-inline:16px; padding-block:20px 80px; }
.wrap { max-width:1200px; margin:0 auto; display:grid; gap:14px; }
h1 { font-size:24px; margin:0; } .lede { color:var(--muted); margin:4px 0 0; max-width:80ch; }
.bar { position:sticky; top:env(safe-area-inset-top,0px); z-index:5; background:var(--ground); padding-block:10px; display:flex; flex-wrap:wrap; gap:8px 14px; align-items:center; border-bottom:1px solid var(--line); font-size:14px; }
select { font:inherit; font-size:14px; color:var(--ink); background:var(--panel); border:1px solid var(--line); border-radius:4px; padding:5px 8px; }
.muted { color:var(--muted); } .err { color:var(--bad); }
.bout { background:var(--panel); border:1px solid var(--line); border-radius:6px; padding:12px 14px; display:grid; gap:10px; }
.head { display:flex; flex-wrap:wrap; gap:6px 14px; align-items:baseline; font-size:13px; color:var(--muted); }
.head b { color:var(--ink); font-size:15px; }
details.msgs summary { cursor:pointer; font-size:13px; color:var(--accent); }
.msg { display:grid; grid-template-columns:46px 62px 1fr; gap:8px; font-size:14px; padding:1px 0; }
.msg .t { font:12px "IBM Plex Mono",ui-monospace,monospace; color:var(--muted); padding-top:2px; }
.msg .w { font-weight:600; font-size:13px; } .w.Matt { color:var(--matt); } .w.Katrina { color:var(--kat); }
.msg .x { white-space:pre-wrap; overflow-wrap:anywhere; }
.cols { display:grid; grid-template-columns:repeat(auto-fit, minmax(300px, 1fr)); gap:12px; }
@media (max-width:760px) { .cols { grid-template-columns:1fr; } }
.col { border:1px solid var(--line); border-radius:4px; padding:10px; display:grid; gap:6px; align-content:start; }
.col h3 { margin:0; font-size:14px; } .col .sum { font-size:14px; }
.pat { font-size:13px; border-top:1px solid var(--soft); padding-top:5px; }
.pat .c { font-weight:600; } .pat .new { color:var(--miss); font-weight:700; font-size:11px; margin-left:4px; }
.pat .q { color:var(--muted); font-style:italic; }
.none { color:var(--muted); font-size:13px; }
.fb { display:flex; flex-wrap:wrap; gap:6px; align-items:center; }
.fb button { font:inherit; font-size:13px; font-weight:600; border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:4px; padding:5px 10px; cursor:pointer; }
.fb button.on { background:var(--ok-bg); color:var(--ok); border-color:var(--ok); }
.fb textarea { flex:1 1 280px; min-height:34px; font:inherit; font-size:14px; color:var(--ink); background:var(--panel); border:1px solid var(--line); border-radius:4px; padding:5px 8px; }
:focus-visible { outline:2px solid var(--accent); outline-offset:2px; }
</style>
<div class="wrap">
  <header>
    <h1>Prompt Compare · Discovery</h1>
    <p class="lede">The same bouts read by each prompt version. <b>v1 (test)</b> read behaviour, including provocation, reaction and blame. <b>v2</b> only records acts it can see in the chunk, without judging anyone. <b>v2.1</b> is v2 plus a child-related flag and a blocked-contact category. <b>v3</b> observes whole conversation groups inside the bout, not single messages. Pick the one that reads best; your picks decide the prompt for the full run.</p>
  </header>
  <div class="bar" role="search">
    <select id="f-show" aria-label="Show"><option value="some">Bouts with observations</option><option value="all">All bouts</option></select>
    <select id="f-pick" aria-label="Your pick"><option value="">Any pick</option><option value="todo">Not picked</option>__PICK_OPTIONS__</select>
    <span id="count" class="muted"></span><span id="tally" class="muted"></span><span id="save" class="muted">Connecting…</span>
  </div>
  <div id="list" class="wrap"></div>
</div>
<script>
const DATA = __DATA__;
const LABELS = __LABELS__;
const $ = id => document.getElementById(id);
const el = (t, c, x) => { const e = document.createElement(t); if (c) e.className = c; if (x != null) e.textContent = x; return e; };
const picks = {}; let db = null; const timers = {}, queued = {}, inflight = {};
function col(title, r) {
  const c = el("div", "col"); c.append(el("h3", null, title));
  c.append(el("div", "sum", r.sum || ""));
  if (!r.p.length) c.append(el("div", "none", "No observations."));
  for (const p of r.p) {
    const d = el("div", "pat"); const h = el("div"); h.append(el("span", "c", p[0]));
    if (!p[1]) h.append(el("span", "new", "NEW"));
    h.append(document.createTextNode(" · " + p[2] + " · msgs " + p[3].join(",") + " · " + p[4]));
    d.append(h, el("div", null, p[5]));
    if (p[6]) d.append(el("div", "q", "“" + p[6] + "”"));
    if (p[7]) d.append(el("div", "none", "child-related"));
    c.append(d);
  }
  for (const n of r.new) c.append(el("div", "none", "New category: " + n[0] + " — " + n[1]));
  return c;
}
function card(b) {
  const a = el("article", "bout"); a.id = "b-" + b.id;
  const h = el("div", "head"); h.append(el("b", null, b.day + " " + b.s + "–" + b.e), el("span", null, b.src + " · " + b.m.length + " messages · " + b.id));
  a.append(h);
  const det = el("details", "msgs"); det.open = true; det.append(el("summary", null, "Messages (all, in order)"));
  for (const m of b.m) { const r = el("div", "msg"); r.append(el("span", "t", m[0]), el("span", "w " + m[1], m[1]), el("span", "x", m[2])); det.append(r); }
  a.append(det);
  const cols = el("div", "cols"); LABELS.forEach((l, k) => cols.append(col(l, b.v[k]))); a.append(cols);
  const fb = el("div", "fb"); a._btn = {};
  for (const [k, label] of [...LABELS.map(l => [l, l + " reads best"]), ["same", "About the same"]]) {
    const bt = el("button", null, label); bt.type = "button";
    bt.addEventListener("click", () => save(b, { pick: (picks[b.id] || {}).pick === k ? "" : k }));
    a._btn[k] = bt; fb.append(bt);
  }
  const ta = el("textarea"); ta.placeholder = "What's better or worse? (optional)"; ta.addEventListener("input", () => save(b, { note: ta.value }));
  a._ta = ta; fb.append(ta); a.append(fb); paint(b, a); return a;
}
function paint(b, a) {
  a = a || $("b-" + b.id); if (!a) return; const r = picks[b.id] || {};
  for (const k in a._btn) { a._btn[k].classList.toggle("on", r.pick === k); a._btn[k].setAttribute("aria-pressed", r.pick === k ? "true" : "false"); }
  if (document.activeElement !== a._ta) a._ta.value = r.note || "";
}
function visible(b) {
  const show = $("f-show").value, pk = $("f-pick").value, p = (picks[b.id] || {}).pick || "";
  if (show === "some" && !b.v.some(r => r.p.length)) return false;
  if (pk === "todo" && p) return false; if (pk && pk !== "todo" && p !== pk) return false; return true;
}
function render() {
  const list = $("list"); list.replaceChildren(); let n = 0;
  for (const b of DATA) if (visible(b)) { list.append(card(b)); n++; }
  $("count").textContent = n + " bouts"; tally();
}
function tally() {
  const t = {}; [...LABELS, "same"].forEach(l => t[l] = 0); for (const k in picks) if (t[picks[k].pick] != null) t[picks[k].pick]++;
  $("tally").textContent = Object.entries(t).map(([l, n]) => l + " " + n).join(" · ");
}
function setSave(t, e) { const s = $("save"); s.textContent = t; s.className = e ? "err" : "muted"; }
function save(b, patch) {
  picks[b.id] = Object.assign({ bout_id: b.id, pick: "", note: "" }, picks[b.id] || {}, patch, { updated_at: new Date().toISOString() });
  paint(b); tally(); const path = "compare/" + b.id; queued[path] = picks[b.id];
  clearTimeout(timers[path]); timers[path] = setTimeout(() => flush(path), 600); setSave("Saving…");
}
async function flush(path) {
  if (!db) { setSave("Not saved: storage unavailable in this view", true); return; }
  if (inflight[path]) return; const v = queued[path]; delete queued[path]; if (!v) return; inflight[path] = true;
  try { await db.doc(path).set(v); setSave("Saved " + new Date().toLocaleTimeString()); }
  catch (e) { setSave("Not saved (" + (e && e.code || "error") + "). Your pick is still on screen.", true); }
  finally { inflight[path] = false; if (queued[path]) flush(path); }
}
async function connect() {
  db = await window.claude?.use?.("db") ?? null;
  if (!db) { setSave("Saving unavailable in this view", true); return; }
  setSave("Connected");
  db.collection("compare").onSnapshot(qs => {
    for (const d of qs.docs) { const v = d.data(); if (v && v.bout_id && !queued["compare/" + v.bout_id] && !inflight["compare/" + v.bout_id]) picks[v.bout_id] = v; }
    for (const b of DATA) paint(b); tally();
  }, e => setSave("Live sync stopped (" + (e && e.code || "error") + ")", true));
}
["f-show", "f-pick"].forEach(id => $(id).addEventListener("change", render));
render(); connect();
</script>
"""

opts = "".join(f'<option value="{l}">{l} best</option>' for l in LABELS) + '<option value="same">About the same</option>'
page = (TEMPLATE.replace("__DATA__", json.dumps(data, ensure_ascii=False)).replace("__LABELS__", json.dumps(LABELS))
        .replace("__PICK_OPTIONS__", opts))
out_path.write_text(page, encoding="utf-8")
print(f"wrote {out_path}: {len(data)} bouts with all {len(LABELS)} versions ({sum(1 for d in data if any(r['p'] for r in d['v']))} with observations)")
