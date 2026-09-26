#!/usr/bin/env python3
# Byline: Claude Code · Fable 5.1 · 2026-09-21
"""Generate the Level-2 vault scaffold from LEVEL-2-ARCHITECTURE.md.

Reads the design doc's domain headings and section tables and writes the
navigation files it describes (INDEX.md, Dashboard.md, AGENTS.md,
MANIFEST.json, _Incoming/.keep per domain; INDEX.md per Level-2 section)
into an output directory that mirrors the logical vault paths. Nothing here
touches B2; the upload is a separate rclone copy. Deployed 2026-09-21 to
b2:salem-data/consignatio/casevault/ (a new tree beside vault/).

Usage: python vault_level2_scaffold_20260921.py <design.md> <out_dir>
"""
import json
import re
import sys
from pathlib import Path

REVISION = "2026-09-21"
DESIGN_NAME = "LEVEL-2-ARCHITECTURE"
BYLINE = f"> _Byline: Claude Code · Fable 5.1 · {REVISION} — generated from [[{DESIGN_NAME}]]; planning scaffold, not live data._"

DOMAIN_RE = re.compile(r"^## (\d)\. (\w+)\s*$")
ROW_RE = re.compile(r"^\| `([^`]+)/` \| (.+?) \| (.+?) \| (.+?) \|\s*$")
DASH_RE = re.compile(r"for: (.+?)\. Report scope")


def parse(design_text):
    domains = []
    current = None
    in_messaging = False
    awaiting_purpose = False
    for line in design_text.splitlines():
        match = DOMAIN_RE.match(line)
        if match:
            current = {"order": int(match.group(1)), "name": match.group(2),
                       "purpose": "", "sections": [], "messaging": [], "dashboard": []}
            domains.append(current)
            in_messaging = False
            awaiting_purpose = True
            continue
        if current is None:
            continue
        if line.startswith("## "):
            current = None
            continue
        if awaiting_purpose and line.strip():
            current["purpose"] = line.strip()
            awaiting_purpose = False
            continue
        if line.startswith("### Messaging detail"):
            in_messaging = True
            continue
        if line.startswith("### Dashboard population"):
            in_messaging = False
            continue
        row = ROW_RE.match(line)
        if row:
            entry = {"name": row.group(1), "populated_by": row.group(2),
                     "contents": row.group(3), "boundary": row.group(4)}
            (current["messaging"] if in_messaging else current["sections"]).append(entry)
            continue
        dash = DASH_RE.search(line)
        if dash:
            current["dashboard"] = [part.strip() for part in dash.group(1).split(",")]
    return domains


def section_index(domain, logical_path, entry):
    return "\n".join([
        f"# {logical_path}/",
        "",
        BYLINE,
        "",
        f"Part of [[{domain}/INDEX|{domain}]]. Everything here remains **context** until an explicit promotion.",
        "",
        "## Populated by",
        "",
        entry["populated_by"],
        "",
        "## Contents",
        "",
        entry["contents"],
        "",
        "## Boundary",
        "",
        entry["boundary"],
        "",
    ])


def domain_index(domain):
    name = domain["name"]
    lines = [f"# {name}", "", BYLINE, "", domain["purpose"], "",
             "## Sections", "", "| Section | Contents |", "|---|---|"]
    for entry in domain["sections"]:
        lines.append(f"| [[{name}/{entry['name']}/INDEX\\|{entry['name']}/]] | {entry['contents']} |")
    lines += ["", "## Landing area", "",
              f"`{name}/_Incoming/` is the coarse-sort landing and review area for this domain.", "",
              "## See also", "",
              f"- [[{name}/Dashboard|Dashboard]]", f"- [[{name}/AGENTS|AGENTS]]",
              f"- [[{DESIGN_NAME}]] section {domain['order']}", ""]
    return "\n".join(lines)


def domain_dashboard(domain):
    name = domain["name"]
    lines = [f"# {name} — Dashboard", "", BYLINE, "",
             "**Status: not connected.** No catalog feed is bound to this page, so it shows no counts.", "",
             "## Views to bind to the catalog", ""]
    lines += [f"- {view} — _not connected_" for view in domain["dashboard"]]
    lines += ["", "Each bound view must show its scope and last refresh.", "",
              f"Back to [[{name}/INDEX|{name}]].", ""]
    return "\n".join(lines)


def domain_agents(domain):
    name = domain["name"]
    return "\n".join([
        f"# {name} — local guide", "", BYLINE, "",
        "Descriptive planning guide. It activates no automation, installs no behavioral rules "
        "and does not override owner instructions.", "",
        f"- Design: [[{DESIGN_NAME}]] section {domain['order']}.",
        f"- Navigation: [[{name}/INDEX|{name}]].",
        "- Original collected bytes are never rewritten; corrections are documented derived versions.",
        "- Processing, review or relevance never creates evidence status; only a completed explicit promotion does.",
        "",
    ])


def domain_manifest(domain):
    name = domain["name"]
    return json.dumps({
        "kind": "planning-only scaffold descriptor",
        "not": ["live catalog", "evidence-package manifest", "integration contract"],
        "design": f"{DESIGN_NAME}.md",
        "planning_revision": REVISION,
        "domain": name,
        "order": domain["order"],
        "sections": [entry["name"] for entry in domain["sections"]],
        "subsections": {"messaging": [entry["name"] for entry in domain["messaging"]]} if domain["messaging"] else {},
        "byline": f"Claude Code · Fable 5.1 · {REVISION}",
    }, indent=2) + "\n"


def write(out_dir, relative, text):
    target = out_dir / relative
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(text, encoding="utf-8", newline="\n")


def main():
    design_path, out_dir = Path(sys.argv[1]), Path(sys.argv[2])
    design_text = design_path.read_text(encoding="utf-8")
    domains = parse(design_text)
    write(out_dir, f"{DESIGN_NAME}.md", design_text)
    for domain in domains:
        name = domain["name"]
        write(out_dir, f"{name}/INDEX.md", domain_index(domain))
        write(out_dir, f"{name}/Dashboard.md", domain_dashboard(domain))
        write(out_dir, f"{name}/AGENTS.md", domain_agents(domain))
        write(out_dir, f"{name}/MANIFEST.json", domain_manifest(domain))
        write(out_dir, f"{name}/_Incoming/.keep", f"vault skeleton marker (level-2 {REVISION})\n")
        for entry in domain["sections"]:
            write(out_dir, f"{name}/{entry['name']}/INDEX.md",
                  section_index(name, f"{name}/{entry['name']}", entry))
        for entry in domain["messaging"]:
            write(out_dir, f"{name}/messaging/{entry['name']}/INDEX.md",
                  section_index(name, f"{name}/messaging/{entry['name']}", entry))
    print(f"domains={len(domains)} sections={sum(len(d['sections']) for d in domains)} "
          f"messaging={sum(len(d['messaging']) for d in domains)}")


if __name__ == "__main__":
    main()
