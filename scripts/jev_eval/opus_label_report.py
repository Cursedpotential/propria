"""Build the owner's review report of Opus's reference labels (Jev eval, Phase 3 checkpoint).

Byline: Claude Code · Opus 5.5 · 2026-09-23. The owner reviews this report before any Jev comparison
(owner 2026-09-23 14:17). The output is one self-contained HTML file with the case data embedded, so
it stays on ovh-files and the owner's machine and never goes into git.

Usage (from the work dir): .venv/bin/python code/opus_label_report.py sample/sample_v2.jsonl reports/opus_labels_report.html
"""

import collections
import datetime
import html
import json
import pathlib
import sys

SAMPLE, OUT = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
WORK = SAMPLE.parent.parent
TAGS = ["case_relevant", "parenting_time_denial", "child_referenced", "info_gatekeeping", "hostility_threat",
        "parent_disparagement", "financial", "logistics", "legal_reference", "third_party", "child_wellbeing",
        "cooperation_offer", "admission", "reframes_prior_event", "blame_shift"]
EXPERIMENTAL = {"reframes_prior_event", "blame_shift"}
STRATA = {"A_fb": "Facebook 2018–25", "B_sms_2021_22": "Texts 2021–22", "C_sms_her_phone": "Texts 2024 (her phone)",
          "D_sms_2025_26": "Texts 2025–26"}


def load(p):
    return [json.loads(line) for line in p.read_text(encoding="utf-8").splitlines() if line.strip()] if p.exists() else []


sample = {r["msg_id"]: r for r in load(SAMPLE)}
labels = {r["msg_id"]: r for r in load(WORK / "labels_opus.jsonl")}
repeat = {r["msg_id"]: r for r in load(WORK / "labels_opus_repeat.jsonl")}
prompt = json.loads((WORK / "raw" / "opus" / "_prompt.json").read_text(encoding="utf-8"))

# Summary: tag counts overall, per stratum, per sender.
cnt, by_s, by_d = collections.Counter(), collections.defaultdict(collections.Counter), collections.defaultdict(collections.Counter)
n_s, n_d = collections.Counter(), collections.Counter()
for mid, lab in labels.items():
    s = sample[mid]
    n_s[s["stratum"]] += 1
    n_d[s["sender_label"]] += 1
    for t in TAGS:
        if lab.get(t):
            cnt[t] += 1
            by_s[t][s["stratum"]] += 1
            by_d[t][s["sender_label"]] += 1
reg = collections.Counter(lab["register"] for lab in labels.values())
none = sum(1 for lab in labels.values() if not any(lab.get(t) for t in TAGS))

# Self-consistency on the repeat sample.
cons_rows, cons_diffs = [], []
if repeat:
    for t in TAGS + ["register"]:
        pairs = [(labels[m].get(t), repeat[m].get(t)) for m in repeat if m in labels]
        agree = sum(1 for a, b in pairs if a == b)
        cons_rows.append((t, agree, len(pairs)))
        for m in repeat:
            if m in labels and labels[m].get(t) != repeat[m].get(t):
                cons_diffs.append({"msg_id": m, "tag": t, "first": labels[m].get(t), "second": repeat[m].get(t)})


def pct(a, b):
    return f"{(100 * a / b):.0f}%" if b else "–"


rows_html = []
for t in TAGS:
    cells = "".join(f"<td>{by_s[t][s]}</td>" for s in STRATA)
    rows_html.append(
        f"<tr><th>{t}{' <span class=exp>experimental</span>' if t in EXPERIMENTAL else ''}</th><td class=num>{cnt[t]}</td>"
        f"<td class=num>{pct(cnt[t], len(labels))}</td>{cells}<td>{by_d[t]['Matt']}</td><td>{by_d[t]['Katrina']}</td></tr>")
cons_html = "".join(
    f"<tr><th>{t}</th><td class=num>{a}/{n}</td><td class=num>{pct(a, n)}</td></tr>" for t, a, n in cons_rows)

messages = []
for mid, s in sorted(sample.items(), key=lambda kv: kv[1]["ts_utc"]):
    lab = labels.get(mid)
    messages.append({
        "id": mid, "stratum": s["stratum"], "source": STRATA[s["stratum"]], "ts": s["ts_utc"][:16].replace("T", " "),
        "sender": s["sender_label"], "text": s["text"], "context": s["prior_context"],
        "labeled": lab is not None,
        "tags": [t for t in TAGS if lab and lab.get(t)],
        "register": lab["register"] if lab else None,
        "rationales": lab["rationales"] if lab else {},
        "repeat_diff": [d["tag"] for d in cons_diffs if d["msg_id"] == mid],
    })

page = f"""<title>Opus labels review</title>
<style>
:root {{ --bg:#f7f7f5; --card:#fff; --ink:#1d1d1b; --muted:#6b6b66; --line:#e2e1dc; --accent:#3257a8; --chip:#e8eefb; --warn:#9a5b00; --warnbg:#fff4e0; --me:#eef6ee; }}
@media (prefers-color-scheme: dark) {{ :root:not([data-theme="light"]) {{ color-scheme:dark; --bg:#16171a; --card:#1f2024; --ink:#e9e8e4; --muted:#9b9a95; --line:#34353a; --accent:#8aa8ee; --chip:#26314a; --warn:#f0b35a; --warnbg:#3a2c14; --me:#1f2c22; }} }}
:root[data-theme="dark"] {{ color-scheme:dark; --bg:#16171a; --card:#1f2024; --ink:#e9e8e4; --muted:#9b9a95; --line:#34353a; --accent:#8aa8ee; --chip:#26314a; --warn:#f0b35a; --warnbg:#3a2c14; --me:#1f2c22; }}
body {{ background:var(--bg); color:var(--ink); font:15px/1.5 system-ui,-apple-system,Segoe UI,sans-serif; margin:0; padding:24px 16px 64px; }}
main {{ max-width:1100px; margin:0 auto; }}
h1 {{ font-size:22px; margin:0 0 4px; }} h2 {{ font-size:17px; margin:28px 0 8px; }}
.sub {{ color:var(--muted); font-size:13px; }}
.facts {{ display:flex; flex-wrap:wrap; gap:8px 20px; margin:12px 0; font-size:14px; }}
.facts b {{ font-variant-numeric:tabular-nums; }}
.tablewrap {{ overflow-x:auto; }}
table {{ border-collapse:collapse; width:100%; font-size:13.5px; background:var(--card); border:1px solid var(--line); }}
th, td {{ padding:6px 10px; border-bottom:1px solid var(--line); text-align:left; white-space:nowrap; }}
td.num, td {{ font-variant-numeric:tabular-nums; }}
thead th {{ color:var(--muted); font-weight:600; font-size:12px; }}
.exp {{ color:var(--warn); font-size:11px; font-weight:500; }}
.filters {{ position:sticky; top:env(safe-area-inset-top,0px); background:var(--bg); padding:10px 0; z-index:2; display:flex; flex-wrap:wrap; gap:6px; align-items:center; border-bottom:1px solid var(--line); }}
.filters select, .filters button {{ font:inherit; font-size:13px; padding:4px 8px; border:1px solid var(--line); border-radius:6px; background:var(--card); color:var(--ink); }}
.filters .count {{ color:var(--muted); font-size:13px; margin-left:auto; }}
.msg {{ background:var(--card); border:1px solid var(--line); border-radius:8px; padding:12px 14px; margin:10px 0; }}
.msg.me {{ border-left:4px solid #5c9a5c; }} .msg.her {{ border-left:4px solid var(--accent); }}
.meta {{ color:var(--muted); font-size:12.5px; margin-bottom:4px; }}
.text {{ white-space:pre-wrap; overflow-wrap:anywhere; font-size:15px; }}
.chips {{ margin-top:8px; display:flex; flex-wrap:wrap; gap:6px; }}
.chip {{ background:var(--chip); border-radius:999px; padding:2px 10px; font-size:12.5px; }}
.chip.reg {{ background:transparent; border:1px solid var(--line); }}
.chip.flag {{ background:var(--warnbg); color:var(--warn); }}
.rats {{ margin:6px 0 0; padding-left:18px; font-size:13.5px; }}
.rats b {{ font-weight:600; }}
details {{ margin-top:8px; font-size:13px; color:var(--muted); }}
details .ctx {{ margin:4px 0; white-space:pre-wrap; overflow-wrap:anywhere; }}
.none {{ color:var(--muted); font-size:13px; margin-top:6px; }}
</style>
<main>
<h1>Opus reference labels: review before any Jev comparison</h1>
<div class=sub>Sample v2 (300 Matt ↔ Katrina messages, seed 20260923) · question set tags-v0 · model {html.escape(prompt["model"])} · prompt {prompt["prompt_sha256"][:12]} · built {datetime.datetime.now(datetime.timezone.utc):%Y-%m-%d %H:%M} UTC</div>
<div class=facts>
<span>Labelled <b>{len(labels)}</b> / {len(sample)}</span>
<span>No tag at all <b>{none}</b></span>
<span>Register: {" · ".join(f"{k} <b>{v}</b>" for k, v in reg.most_common())}</span>
<span>Repeat pass <b>{len(repeat)}</b> messages</span>
</div>

<h2>How often each tag fired</h2>
<div class=tablewrap><table><thead><tr><th>Tag</th><th>True</th><th>Share</th>{"".join(f"<th>{v}<br><span class=sub>n={n_s[k]}</span></th>" for k, v in STRATA.items())}<th>Matt<br><span class=sub>n={n_d['Matt']}</span></th><th>Katrina<br><span class=sub>n={n_d['Katrina']}</span></th></tr></thead>
<tbody>{"".join(rows_html)}</tbody></table></div>

<h2>Is Opus consistent with itself? (same messages labelled twice)</h2>
{"<div class=tablewrap><table><thead><tr><th>Tag</th><th>Same answer</th><th>Agreement</th></tr></thead><tbody>" + cons_html + "</tbody></table></div>" if repeat else "<p class=sub>Repeat pass not finished yet.</p>"}

<h2>Every message</h2>
<div class=filters>
<select id=fTag><option value="">Any tag</option><option value="__none">No tag</option>{"".join(f'<option value="{t}">{t}</option>' for t in TAGS)}<option value="__diff">Repeat disagreed</option></select>
<select id=fSrc><option value="">All sources</option>{"".join(f'<option value="{k}">{v}</option>' for k, v in STRATA.items())}</select>
<select id=fWho><option value="">Both senders</option><option>Matt</option><option>Katrina</option></select>
<span class=count id=count></span>
</div>
<div id=list></div>
</main>
<script>
const DATA = {json.dumps(messages, ensure_ascii=False)};
const el = (tag, cls, text) => {{ const e = document.createElement(tag); if (cls) e.className = cls; if (text != null) e.textContent = text; return e; }};
function render() {{
  const tag = fTag.value, src = fSrc.value, who = fWho.value, list = document.getElementById('list');
  list.replaceChildren();
  const rows = DATA.filter(m => (!src || m.stratum === src) && (!who || m.sender === who) &&
    (!tag || (tag === '__none' ? m.labeled && m.tags.length === 0 : tag === '__diff' ? m.repeat_diff.length > 0 : m.tags.includes(tag))));
  document.getElementById('count').textContent = rows.length + ' messages';
  for (const m of rows) {{
    const card = el('div', 'msg ' + (m.sender === 'Matt' ? 'me' : 'her'));
    card.append(el('div', 'meta', m.ts + ' UTC · ' + m.source + ' · ' + m.sender));
    card.append(el('div', 'text', m.text));
    const chips = el('div', 'chips');
    if (!m.labeled) chips.append(el('span', 'chip flag', 'not labelled'));
    m.tags.forEach(t => chips.append(el('span', 'chip', t)));
    if (m.register) chips.append(el('span', 'chip reg', 'register: ' + m.register));
    if (m.repeat_diff.length) chips.append(el('span', 'chip flag', 'repeat disagreed: ' + m.repeat_diff.join(', ')));
    card.append(chips);
    if (m.labeled && !m.tags.length) card.append(el('div', 'none', 'No tag applies (Opus).'));
    if (m.tags.length) {{
      const ul = el('ul', 'rats');
      m.tags.forEach(t => {{ const li = el('li'); li.append(el('b', null, t + ': ')); li.append(document.createTextNode(m.rationales[t] || '')); ul.append(li); }});
      card.append(ul);
    }}
    const d = el('details'); d.append(el('summary', null, 'Earlier messages (' + m.context.length + ')'));
    m.context.forEach(c => d.append(el('div', 'ctx', (c.ts || '').slice(0, 16).replace('T', ' ') + '  ' + c.sender + ': ' + c.text)));
    card.append(d);
    list.append(card);
  }}
}}
['fTag', 'fSrc', 'fWho'].forEach(id => document.getElementById(id).addEventListener('change', render));
render();
</script>
"""
OUT.parent.mkdir(parents=True, exist_ok=True)
OUT.write_text(page, encoding="utf-8")
print(f"report -> {OUT} ({len(labels)} labelled, {len(repeat)} repeat)")
