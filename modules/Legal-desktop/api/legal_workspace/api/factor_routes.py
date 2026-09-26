"""Issue-tree and both-parent factor analysis routes.

> _Byline: Grok · grok-4.6 · 2026-08-18_

Read-only against the live workspace. Parent mounts this router:

    from legal_workspace.api.factor_routes import router as factor_router
    app.include_router(factor_router)

``GET /v1/issues`` returns ``WORKSPACE.load().issue`` (the persisted tree).
The domain helper ``issue_tree()`` is the structural seed when no live
issue is passed. Do not invent package citations here.
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException

from legal_workspace.domain.factors import (
    FactorAnalysis,
    FactorEntry,
    FactorLetter,
    FactorNoteCreate,
    analyze_factor,
    lookup_factor,
)
from legal_workspace.domain.issue import (
    IssueChildCreate,
    IssueElement,
    IssueElementCreate,
    IssuePatch,
    LegalIssue,
)

router = APIRouter()


def live_workspace():
    """Read-only workspace handle.

    Prefers ``legal_workspace.api.main.WORKSPACE`` so HTTP tests that
    rebind the API module stay consistent. Falls back to ``get_workspace()``.
    """
    from legal_workspace.api import main as main_mod
    from legal_workspace.services.workspace import get_workspace

    bound = getattr(main_mod, "WORKSPACE", None)
    if bound is not None:
        return bound
    return get_workspace()


@router.get("/v1/issues", response_model=LegalIssue)
def get_issue_tree() -> LegalIssue:
    """Parent may instead mount this as ``return WORKSPACE.load().issue``."""
    return live_workspace().load().issue


@router.post("/v1/issues", response_model=LegalIssue)
def add_issue(body: IssueChildCreate) -> LegalIssue:
    try:
        return live_workspace().add_issue_child(body)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail="unknown parent issue") from exc


@router.put("/v1/issues/{issue_id}", response_model=LegalIssue)
def edit_issue(issue_id, body: IssuePatch) -> LegalIssue:
    from uuid import UUID

    try:
        return live_workspace().patch_issue(UUID(str(issue_id)), body)
    except (StopIteration, ValueError) as exc:
        raise HTTPException(status_code=404, detail="unknown issue") from exc


@router.post("/v1/issues/{issue_id}/elements", response_model=IssueElement)
def add_issue_element(issue_id, body: IssueElementCreate) -> IssueElement:
    from uuid import UUID

    try:
        return live_workspace().add_issue_element(UUID(str(issue_id)), body)
    except (StopIteration, ValueError) as exc:
        raise HTTPException(status_code=404, detail="unknown issue") from exc


@router.post("/v1/factors/{letter}/notes", response_model=FactorEntry)
def add_factor_note(letter: FactorLetter, body: FactorNoteCreate) -> FactorEntry:
    try:
        return live_workspace().add_factor_note(letter, body.side, body.text)
    except StopIteration as exc:
        raise HTTPException(status_code=404, detail=f"factor ({letter.value}) not in workspace") from exc
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc


@router.get("/v1/factors/{letter}/analysis", response_model=FactorAnalysis)
def get_factor_analysis(letter: FactorLetter) -> FactorAnalysis:
    entry = lookup_factor(live_workspace().load().factors, letter)
    if entry is None:
        raise HTTPException(status_code=404, detail=f"factor ({letter.value}) not in workspace")
    return analyze_factor(entry)
