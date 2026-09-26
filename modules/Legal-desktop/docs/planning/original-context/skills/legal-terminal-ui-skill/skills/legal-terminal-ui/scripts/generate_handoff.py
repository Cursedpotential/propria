#!/usr/bin/env python3
"""
Compiles a full UI/UX design handoff for a module by merging:
  1. The module's interview answers + final_recommendation (assets/design-decisions.json)
  2. EVERY relevant guidance file in references/ (principles, layout, dark-mode tokens,
     data-viz/ingestion, legal-domain UX, AI chat UX, database UX) — filtered to what
     the module's data_viz_needs actually require, plus the always-applicable core files.
  3. The checklist.

Output: handoffs/<module>-handoff.md — a single, self-contained, developer-ready spec.
This is the actual deliverable. Nobody should need to re-read the interview transcript
or hunt through reference/ files after this doc exists — it IS the guide for building
that specific module's UI/UX.

Usage: python3 generate_handoff.py --module case-dashboard
"""
import argparse
import json
import os
import re
from datetime import datetime, timezone

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DECISIONS_PATH = os.path.join(BASE_DIR, "assets", "design-decisions.json")
REFERENCES_DIR = os.path.join(BASE_DIR, "references")
HANDOFFS_DIR = os.path.join(BASE_DIR, "handoffs")

ALWAYS_INCLUDE = ["dark-mode-tokens.md", "layout-patterns.md"]

CONDITIONAL_MAP = {
    "ingestion": "data-viz-ingestion.md",
    "chart": "data-viz-ingestion.md",
    "trend": "data-viz-ingestion.md",
    "table": "data-viz-ingestion.md",
    "relationship": "legal-domain-ux.md",
    "graph": "legal-domain-ux.md",
    "ai chat": "ai-chat-interaction.md",
    "agent": "ai-chat-interaction.md",
    "database": "database-productivity-ux.md",
    "query": "database-productivity-ux.md",
}


def read_ref(filename):
    path = os.path.join(REFERENCES_DIR, filename)
    if not os.path.exists(path):
        return f"_[Missing reference file: {filename}]_"
    with open(path, "r") as f:
        return f.read()


def strip_h1(md_text):
    return re.sub(r"^# .+\n+", "", md_text, count=1)


def determine_conditional_refs(data_viz_needs):
    needs_lower = (data_viz_needs or "").lower()
    matched = []
    for keyword, filename in CONDITIONAL_MAP.items():
        if keyword in needs_lower and filename not in matched:
            matched.append(filename)
    if "case" in needs_lower or "document" in needs_lower or "party" in needs_lower:
        if "legal-domain-ux.md" not in matched:
            matched.append("legal-domain-ux.md")
    if not matched:
        matched.append("legal-domain-ux.md")
    return matched


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--module", required=True)
    args = parser.parse_args()
    module_name = args.module

    if not os.path.exists(DECISIONS_PATH):
        print(f"ERROR: {DECISIONS_PATH} not found. Run design_interview.py first.")
        return
    with open(DECISIONS_PATH, "r") as f:
        decisions = json.load(f)

    record = decisions.get(module_name)
    if not record:
        print(f"ERROR: No interview record found for module '{module_name}'.")
        return
    if not record.get("final_recommendation"):
        print(
            f"ERROR: 'final_recommendation' is not yet set for module '{module_name}'. "
            "Complete the live search_web research step and write the chosen option into "
            "design-decisions.json before generating the handoff."
        )
        return

    answers = record["answers"]
    final_rec = record["final_recommendation"]

    conditional_files = determine_conditional_refs(answers.get("data_viz_needs", ""))
    all_ref_files = ALWAYS_INCLUDE + [f for f in conditional_files if f not in ALWAYS_INCLUDE]

    core_principles = read_ref("legal-domain-ux.md") if False else None  # placeholder, unused
    checklist = read_ref("checklist.md")

    sections = []
    sections.append(f"# Design Handoff: {module_name}")
    sections.append(f"_Generated {datetime.now(timezone.utc).isoformat()} from interview + live research._")

    sections.append("## Module Brief")
    sections.append(f"- **Purpose:** {answers.get('module_purpose', 'N/A')}")
    sections.append(f"- **Primary user role:** {answers.get('user_role', 'N/A')}")
    sections.append(f"- **Density:** {answers.get('density', 'N/A')}")
    sections.append(f"- **Tech stack:** {answers.get('tech_stack', 'N/A')}")
    sections.append(f"- **Aesthetic direction:** {answers.get('aesthetic_direction', 'N/A')}")
    sections.append(f"- **Data/viz/interaction needs:** {answers.get('data_viz_needs', 'N/A')}")
    sections.append(f"- **Must-haves/constraints:** {answers.get('must_haves', 'N/A')}")

    sections.append("## Final Recommendation (from live research)")
    if isinstance(final_rec, dict):
        sections.append("```json\n" + json.dumps(final_rec, indent=2) + "\n```")
    else:
        sections.append(str(final_rec))

    sections.append("## Core Principles (apply to every screen)")
    sections.append(
        "- Conceal complexity: dense defaults, advanced options one interaction away, never deleted.\n"
        "- Progressive disclosure: layer novice guidance on top of expert density on the same screen.\n"
        "- Status must always be visible and actionable.\n"
        "- Every visualization matches its data type; every table/chart is exportable as CSV/JSON.\n"
        "- Provenance (source, timestamp, version) shown wherever data is displayed."
    )

    for filename in all_ref_files:
        content = strip_h1(read_ref(filename))
        title = filename.replace(".md", "").replace("-", " ").title()
        sections.append(f"## Guidance: {title}")
        sections.append(content.strip())

    sections.append("## Pre-Ship Checklist for This Module")
    sections.append(strip_h1(checklist).strip())

    output_path = os.path.join(HANDOFFS_DIR, f"{module_name}-handoff.md")
    os.makedirs(HANDOFFS_DIR, exist_ok=True)
    with open(output_path, "w") as f:
        f.write("\n\n".join(sections) + "\n")

    print(f"Handoff document written to {output_path}")
    print("This file is now the single source of truth for building this module's UI/UX.")


if __name__ == "__main__":
    main()
