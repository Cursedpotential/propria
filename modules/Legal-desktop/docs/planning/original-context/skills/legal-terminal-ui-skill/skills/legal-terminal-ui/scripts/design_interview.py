#!/usr/bin/env python3
"""
Interactive design interview for legal-terminal UI/UX decisions.
Collects requirements per module, generates a research_brief for Claude to
execute via search_web, and stores everything (including the eventual
final_recommendation) in assets/design-decisions.json keyed by module name.
"""
import json
import os
import sys
from datetime import datetime, timezone

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DECISIONS_PATH = os.path.join(BASE_DIR, "assets", "design-decisions.json")

QUESTIONS = [
    {"key": "module_name", "prompt": "Short name/id for this module (e.g. 'case-dashboard', 'ai-chat-panel'):", "options": None},
    {"key": "module_purpose", "prompt": "What is the primary purpose of this screen/module?", "options": None},
    {"key": "user_role", "prompt": "Who is the primary user of this screen?", "options": ["Attorney", "Paralegal", "Admin/Ops", "Mixed roles"]},
    {"key": "density", "prompt": "How dense should this screen be?", "options": [
        "Maximal density (Bloomberg-style)",
        "Moderate density",
        "Low density",
    ]},
    {"key": "tech_stack", "prompt": "Frontend stack/framework?", "options": None},
    {"key": "aesthetic_direction", "prompt": "Describe the aesthetic direction (e.g. 'Bloomberg terminal', 'modern SaaS dark'):", "options": None},
    {"key": "data_viz_needs", "prompt": "What data visualization does this screen need?", "options": [
        "Dense tables only",
        "Tables + trend charts",
        "Ingestion/pipeline status monitoring",
        "Relationship graphs",
        "AI chat / agent interaction panel",
        "Database explorer / query panel",
        "Combination — describe which",
    ]},
    {"key": "must_haves", "prompt": "Any must-have constraints? (licensing, accessibility, existing design system)", "options": None},
]


def ask(question):
    print(f"\n{question['prompt']}")
    if question["options"]:
        for i, opt in enumerate(question["options"], 1):
            print(f"  {i}. {opt}")
        print("Answer with a number, or type free text:")
    raw = input("> ").strip()
    if question["options"]:
        try:
            idx = int(raw)
            if 1 <= idx <= len(question["options"]):
                return question["options"][idx - 1]
        except ValueError:
            pass
    return raw


def build_research_brief(answers):
    return {
        "instruction": (
            "Claude: use search_web to research CURRENT, LIVE options based on these answers before "
            "recommending anything. Treat references/component-libraries.md and references/color-schemes.md "
            "as starting hypotheses only. Search for component libraries and color/theme directions fitting "
            "tech_stack, aesthetic_direction, and data_viz_needs. Present 3-4 live options with tradeoffs, "
            "ask the user to pick or refine, record final choice into final_recommendation, then run "
            "scripts/generate_handoff.py --module <module_name> to compile the handoff doc."
        ),
        "suggested_queries": [
            f"{answers.get('tech_stack', '')} dark mode component library {answers.get('aesthetic_direction', '')}".strip(),
            f"best terminal style UI kit {answers.get('tech_stack', '')}".strip(),
            f"{answers.get('aesthetic_direction', '')} dark color palette design system".strip(),
        ],
    }


def load_decisions():
    if os.path.exists(DECISIONS_PATH):
        with open(DECISIONS_PATH, "r") as f:
            try:
                return json.load(f)
            except json.JSONDecodeError:
                return {}
    return {}


def main():
    answers = {}
    for q in QUESTIONS:
        answers[q["key"]] = ask(q)

    module_name = answers.get("module_name") or "unnamed-module"

    record = {
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "answers": answers,
        "research_brief": build_research_brief(answers),
        "final_recommendation": None,
    }

    decisions = load_decisions()
    decisions[module_name] = record

    os.makedirs(os.path.dirname(DECISIONS_PATH), exist_ok=True)
    with open(DECISIONS_PATH, "w") as f:
        json.dump(decisions, f, indent=2)

    print(f"\nSaved interview + research brief for module '{module_name}' to {DECISIONS_PATH}")
    print(json.dumps(record, indent=2))
    print(
        "\nNEXT STEP FOR CLAUDE: run search_web using suggested_queries above, synthesize live options, "
        "present to user, update final_recommendation for this module in design-decisions.json, then run:\n"
        f"  python3 scripts/generate_handoff.py --module {module_name}"
    )


if __name__ == "__main__":
    main()
