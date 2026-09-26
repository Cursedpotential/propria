"""Bulk rename old 4-letter/legal routes to plain English.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
"""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

# UI route path renames. API routes (e.g. /v1/automations/jobs) are kept out.
PATH_RENAMES: dict[str, str] = {
    "/chat": "/assistant",
    "/prec": "/case-search",
    "/stat": "/laws",
    "/cite": "/citation-check",
    "/rqst": "/open-questions",
    "/issue": "/questions",
    "/fctr": "/custody-factors",
    "/ctrx": "/agreements",
    "/doc": "/documents",
    "/priv": "/confidentiality-check",
    "/drft": "/drafts",
    "/tmpl": "/templates",
    "/rvw": "/review",
    "/rels": "/final-copy",
    "/file": "/filing-checklist",
    "/jobs": "/analysis-queue",
    "/wkfl": "/playbooks",
    "/autm": "/scheduled-jobs",
    "/trig": "/notices",
    "/audt": "/activity-log",
    "/live": "/external-sources",
    "/cal": "/calendar",
    "/disc": "/evidence-requests",
    "/exh": "/evidence",
    "/miss": "/missing-evidence",
    "/todo": "/tasks",
    "/timl": "/timeline",
    "/agnt": "/assistant-log",
    "/strat": "/private-notes",
    "/team": "/challenge-draft",
}

# Exact label/help renames. Longer phrases first to avoid partial collisions.
LABEL_RENAMES: dict[str, str] = {
    "Filing readiness checklist": "Filing readiness checklist",  # no-op placeholder for ordering
    "Filing checklist": "Filing readiness checklist",
    "Analysis Queue": "Analysis queue",
    "Precedent Search": "Case search",
    "Research questions": "Open questions",
    "Issue tree": "Questions the judge decides",
    "Best-interest factors": "What the judge must consider",
    "Contract Workbench": "Agreement review",
    "Document Analyzer": "Document viewer",
    "Privilege Check": "Confidentiality check",
    "Brief Builder": "Motion writer",
    "Motion outlines": "Starting templates",
    "Owner review": "Your review",
    "Release candidate": "Final review copy",
    "Workflows": "Playbooks",
    "Automations": "Scheduled jobs",
    "Triggers": "Inbound notices",
    "Audit Log": "Activity log",
    "Integrations": "External sources",
    "Docket Watch": "Court dates",
    "Exhibit list": "Evidence list",
    "Agent log": "Assistant activity log",
    "Private strategy": "My private notes",
    "Red team": "Devil's advocate review",
    "Paralegal": "Ask the assistant",
    "Statutes": "Laws",
    "Citations": "Citation check",
    "Discovery": "Evidence requests",
    "Tasks": "Your tasks",
    "Matter command center": "Overview of your case",
}

# Internal route abbreviations used in user-facing strings.
ABBREVIATIONS: dict[str, str] = {
    "DRFT": "drafts",
    "RVW": "review",
    "RELS": "final-copy",
}


def replace_path(text: str) -> str:
    # Use word boundary after the path so /chat does not match /challenge-draft.
    for old, new in PATH_RENAMES.items():
        text = re.sub(rf"{re.escape(old)}(?!\w|-)", new, text)
    return text


def replace_labels(text: str) -> str:
    for old, new in LABEL_RENAMES.items():
        text = text.replace(old, new)
    for old, new in ABBREVIATIONS.items():
        text = text.replace(old, new)
    return text


def process_file(path: Path) -> bool:
    original = path.read_text(encoding="utf-8")
    updated = replace_path(original)
    updated = replace_labels(updated)
    if updated != original:
        path.write_text(updated, encoding="utf-8")
        return True
    return False


def main() -> None:
    patterns = [
        "web/src/**/*.tsx",
        "web/src/**/*.ts",
        "api/legal_workspace/**/*.py",
        "tests/**/*.py",
        "config/routing.json",
    ]
    touched: list[Path] = []
    for pattern in patterns:
        for file in ROOT.glob(pattern):
            if process_file(file):
                touched.append(file)
    for file in sorted(touched):
        print(file.relative_to(ROOT))


if __name__ == "__main__":
    main()
