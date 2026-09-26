"""Automation HTTP surface. Mount from main.py — do not import main here.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, HTTPException

from legal_workspace.services.automation.runner import UnknownPlaybookError, list_playbooks, run_playbook
from legal_workspace.services.automation.scheduler import list_jobs

router = APIRouter()


@router.get("/v1/automations/jobs")
def get_jobs() -> list[dict[str, Any]]:
    return list_jobs()


@router.get("/v1/automations/playbooks")
def get_playbooks() -> list[dict[str, Any]]:
    return list_playbooks()


@router.post("/v1/automations/playbooks/{playbook_id}:run")
def post_run_playbook(playbook_id: str) -> dict[str, Any]:
    try:
        return run_playbook(playbook_id)
    except UnknownPlaybookError as exc:
        raise HTTPException(status_code=404, detail="unknown playbook") from exc
