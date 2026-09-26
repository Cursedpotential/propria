"""Build a local planning navigator and verify the planning package using stdlib only."""
from pathlib import Path
import csv
import html
import json
import re

ROOT = Path(__file__).resolve().parent
requirements = []
for line in (ROOT / "REQUIREMENTS.md").read_text(encoding="utf-8").splitlines():
    match = re.match(r"\| (R\d+) \| (.*?) \| (.*?) \|$", line)
    if match:
        requirements.append(dict(zip(("id", "direction", "acceptance"), match.groups())))
assert [r["id"] for r in requirements] == [f"R{i:02d}" for i in range(1, 53)]

groups = {
    "0": [1, 4, 5, 6, 9, 12, 37, 38, 40, 41, 42, 47, 48, 50, 52],
    "1": [2, 3, 7, 16, 39, 44, 49],
    "2": [8, 10, 11, 13, 14, 15, 45, 47, 48, 50, 51],
    "3": [25, 26, 27, 28, 29, 30, 31],
    "4": [13, 15, 16, 17, 18, 19],
    "5": [21, 22, 23, 24],
    "6": [20, 36, 42],
    "7": [13, 22, 32, 33, 34, 35, 36, 45],
    "8": [2, 37, 38, 39, 40, 41, 43, 44, 46, 49, 51, 52],
}
for req in requirements:
    number = int(req["id"][1:])
    req["phases"] = [f"F{k}/B{k}" for k, values in groups.items() if number in values]
    assert req["phases"], req["id"]
    req["source"] = "Owner discussion; proposed mechanics labeled in REQUIREMENTS.md"
    req["status"] = "planned"

(ROOT / "requirements.json").write_text(json.dumps(requirements, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
with (ROOT / "TRACEABILITY.csv").open("w", newline="", encoding="utf-8-sig") as handle:
    writer = csv.DictWriter(handle, fieldnames=["id", "direction", "acceptance", "phases", "status"])
    writer.writeheader()
    for req in requirements:
        writer.writerow({key: "; ".join(req[key]) if key == "phases" else req[key] for key in writer.fieldnames})

json_checks = []
for path in ROOT.rglob("*.json"):
    json.loads(path.read_text(encoding="utf-8-sig"))
    json_checks.append(str(path.relative_to(ROOT)))

required_reports = [
    "inputs/current-app/REPORT.md", "inputs/current-app/VISUALIZATION-RECOVERY.md",
    "inputs/current-app/LEGAL-CAPABILITY-DISCOVERY.md", "inputs/stack/STACK-RECOVERY.md",
    "inputs/stack/TOOLKIT-CAPABILITIES.md", "inputs/handoffs/HANDOFF-RECOVERY.md",
    "inputs/handoffs/LEGAL-MCP-PACK-REVIEW.md",
]
missing_reports = [name for name in required_reports if not (ROOT / name).is_file()]
broken = []
checked = 0
for path in ROOT.glob("*.md"):
    for match in re.finditer(r"\[[^\]]*\]\((?:<([^>]+)>|([^\)\n]+))\)", path.read_text(encoding="utf-8")):
        target = match.group(1) or match.group(2)
        if re.match(r"https?://", target) or target.startswith("#"):
            continue
        target = re.sub(r":\d+$", "", target.split("#", 1)[0])
        resolved = Path(target) if re.match(r"^[A-Za-z]:[/\\]", target) else path.parent / target
        checked += 1
        if not resolved.exists():
            broken.append({"file": path.name, "target": target})

rediscovery = json.loads((ROOT / "inputs/handoffs/rediscovery.json").read_text(encoding="utf-8"))
def plain(value):
    return html.escape(str(value).replace("**", "").replace("`", ""))
rows = "".join(f'<tr><td>{r["id"]}</td><td>{plain(r["direction"])}</td><td>{plain(r["acceptance"])}</td><td>{plain(", ".join(r["phases"]))}</td></tr>' for r in requirements)
recovered = "".join(f'<li><span class="tag">{plain(i["id"])}</span> {plain(i["requirement"])}</li>' for i in rediscovery["items"])
integrations = "".join(f'<li>{plain(i["name"])}</li>' for i in rediscovery["external_integrations"])
navigator = """<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Advocatio — plan and reconciliation</title>
<style>:root{color-scheme:dark}body{font:15px/1.55 system-ui,sans-serif;background:#171b20;color:#e5e4de;margin:0}main{max-width:1500px;margin:auto;padding:28px}h1{font:32px Georgia,serif;margin-bottom:6px}h2{font-size:20px;margin-top:30px}p{max-width:1050px;color:#c4c7cb}a{color:#b4cddd}nav{display:flex;flex-wrap:wrap;gap:10px;margin:18px 0}nav a,.tag{border:1px solid #454b53;padding:4px 9px;border-radius:3px}input{width:min(720px,90%);padding:12px;background:#222830;color:#eee;border:1px solid #66717d;border-radius:3px}table{width:100%;border-collapse:collapse;font-size:14px;margin-top:15px}td,th{padding:11px;text-align:left;vertical-align:top;border-bottom:1px solid #363c44}th{background:#242a32;position:sticky;top:0}td:first-child{white-space:nowrap;color:#aebcc8}td:last-child{min-width:90px}li{margin:10px 0}.columns{display:grid;grid-template-columns:2fr 1fr;gap:30px}.tag{font-size:12px;white-space:nowrap}button{background:#28313b;color:#eee;border:1px solid #66717d;padding:8px;cursor:pointer}@media(max-width:800px){main{padding:15px}.columns{grid-template-columns:1fr}.scroll{overflow:auto}table{min-width:850px}h1{font-size:27px}}</style>
<main><h1>Advocatio</h1><p>One unified legal work surface. Shared references and methods, source-linked case work, document review, timelines, a digital firm, and tools people and agents can use. A staged port builds on the toolkit and services already present.</p>
<nav><a href="BUILD_GUIDE.md">Direction and deliverables</a><a href="CURRENT-COMPARISON.md">Current application</a><a href="REDISCUSSION.md">Recovered commitments</a><a href="STACK.md">Stack decision</a><a href="MCP-CLIENT-AND-PORTS.md">Tools, MCP and ports</a><a href="STATUS-AND-FLAGS.md">Compact flags</a><a href="PHASES.md">Build phases</a><a href="README.md">All documents</a></nav>
<h2>Requirements and acceptance</h2><label for="filter">Find a requirement, capability or phase</label><br><input id="filter" placeholder="Try timeline, LibreOffice, MCP, translation…"><p id="count"></p><div class="scroll"><table><thead><tr><th>ID</th><th>Direction</th><th>Deliverable / acceptance</th><th>Phases</th></tr></thead><tbody>""" + rows + """</tbody></table></div>
<div class="columns"><section><h2>Recovered handoff requirements</h2><p>Each item has original and later source references in the <a href="inputs/handoffs/HANDOFF-RECOVERY.md">handoff review</a>.</p><ul>""" + recovered + """</ul></section><section><h2>Named outside projects</h2><p>Intent, existing code and disposition are recorded separately.</p><ul>""" + integrations + """</ul><h2>Additional inventories</h2><p><a href="inputs/stack/TOOLKIT-CAPABILITIES.md">Full toolkit and remote store</a></p><p><a href="inputs/current-app/LEGAL-CAPABILITY-DISCOVERY.md">Claude, Codex and .agents legal capabilities</a></p><p><a href="inputs/handoffs/LEGAL-MCP-PACK-REVIEW.md">Original Legal MCP ZIP content review</a></p><p><a href="TRACEABILITY.csv">Download requirement traceability</a></p></section></div>
<script>const box=document.querySelector('#filter'),rows=[...document.querySelectorAll('tbody tr')],count=document.querySelector('#count');function update(){const q=box.value.toLowerCase();let shown=0;for(const r of rows){r.hidden=!r.textContent.toLowerCase().includes(q);if(!r.hidden)shown++;}count.textContent=shown+' of '+rows.length+' requirements';}box.addEventListener('input',update);update();</script></main></html>"""
(ROOT / "INDEX.html").write_text(navigator, encoding="utf-8")
receipt = {"requirements": len(requirements), "requirements_with_phase_mapping": sum(bool(r["phases"]) for r in requirements), "recovered_requirements": len(rediscovery["items"]), "external_integrations": len(rediscovery["external_integrations"]), "json_files_parsed": json_checks, "root_markdown_file_links_checked": checked, "broken_links": broken, "missing_reports": missing_reports, "checks": "artifact structure, JSON, requirements numbering/coverage and local file links", "runtime_tests": "not run in planning review", "docstore_registration": "See DOCSTORE-PUBLICATION.md for registration and readback; indexing is separate"}
(ROOT / "VERIFICATION.json").write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
print(json.dumps(receipt, indent=2))
if broken or missing_reports:
    raise SystemExit(1)
