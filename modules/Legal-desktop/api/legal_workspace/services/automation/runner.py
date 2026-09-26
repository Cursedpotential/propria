"""Sequential playbook runner. No Temporal / Prefect / Dagster.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Steps are structural labels. Do not invent hearing dates.
"""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Any
from uuid import uuid4

from legal_workspace.services.automation import get_store_dir
from legal_workspace.services.automation.bus import publish
from legal_workspace.services.persist import append_jsonl, atomic_write_json

FOC_HEARING_PREP_ID = "foc-hearing-prep"
FOC_HEARING_PREP_STEPS = (
    "confirm-clerk-docket",
    "review-factor-j",
    "build-release",
)

PLAYBOOKS: dict[str, dict[str, Any]] = {
    FOC_HEARING_PREP_ID: {
        "playbook_id": FOC_HEARING_PREP_ID,
        "title": "FOC hearing prep",
        "steps": list(FOC_HEARING_PREP_STEPS),
        "kind": "structural_labels",
        "court_safe": False,
        "note": "Labels only. No hearing date is invented. Clerk docket is not fetched.",
    }
}


class UnknownPlaybookError(KeyError):
    """Playbook id is not in the seeded catalog."""


def list_playbooks() -> list[dict[str, Any]]:
    return [dict(row) for row in PLAYBOOKS.values()]


def run_playbook(playbook_id: str) -> dict[str, Any]:
    playbook = PLAYBOOKS.get(playbook_id)
    if playbook is None:
        raise UnknownPlaybookError(playbook_id)
    step_results = [
        {"step": name, "passed": True, "kind": "label"} for name in playbook["steps"]
    ]
    run = {
        "run_id": str(uuid4()),
        "playbook_id": playbook_id,
        "steps": step_results,
        "ok": all(row["passed"] for row in step_results),
        "court_safe": False,
        "kind": "structural_labels",
        "finished_at": datetime.now(UTC).isoformat(),
    }
    root = get_store_dir()
    append_jsonl(root / "playbook_runs.jsonl", run)
    atomic_write_json(root / "last_playbook_run.json", run)
    publish("job_complete")
    return run
