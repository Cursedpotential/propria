"""Workbench BFF routes for the repair workflow builder.

Byline: Claude Code · Opus 5.5 · 2026-09-26.

`/api/proffer/repair/{tools,propose,validate,run,runs/{workflow_id}}` pass the
engine's `/reference-import/repair/*` routes through (shared contract:
`docs/pending-review/2026-09-25-repair-workflow-builder.md`). Every route takes
the TEST/REAL `mode` query like the other Proffer routes. Engine errors keep
their status and `detail`; a refused run (422) also keeps its checklist.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import APIRouter, HTTPException, Path, Query
from fastapi.responses import JSONResponse

from app.service.proffer_errors import ProfferError
from app.service.proffer_repair_plan import (
    RepairRunRefusedError,
    propose,
    run,
    run_status,
    tools,
    validate,
)
from app.types.matter_mode import MatterMode
from app.types.proffer_repair_plan import (
    RepairPlan,
    RepairProposeRequest,
    RepairProposeResponse,
    RepairRunRefused,
    RepairRunResponse,
    RepairRunStatus,
    RepairToolsResponse,
    RepairValidateResponse,
)

router = APIRouter(prefix="/api/proffer/repair", tags=["proffer"])

WorkflowID = Annotated[str, Path(pattern=r"^[A-Za-z0-9_-]{8,160}$")]
Mode = Annotated[MatterMode, Query()]


def _translate(error: ProfferError) -> HTTPException:
    return HTTPException(status_code=error.status_code, detail=error.detail)


@router.get("/tools", response_model=RepairToolsResponse)
async def repair_tools_endpoint(mode: Mode):
    try:
        return await tools(mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/propose", response_model=RepairProposeResponse)
async def repair_propose_endpoint(body: RepairProposeRequest, mode: Mode):
    try:
        return await propose(body, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post("/validate", response_model=RepairValidateResponse)
async def repair_validate_endpoint(body: RepairPlan, mode: Mode):
    try:
        return await validate(body, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None


@router.post(
    "/run",
    response_model=RepairRunResponse,
    status_code=201,
    responses={422: {"model": RepairRunRefused, "description": "The engine refused the plan; its checks say why."}},
)
async def repair_run_endpoint(body: RepairPlan, mode: Mode):
    try:
        return await run(body, mode=mode)
    except RepairRunRefusedError as refused:
        return JSONResponse(status_code=422, content=refused.refused.model_dump(mode="json"))
    except ProfferError as error:
        raise _translate(error) from None


@router.get("/runs/{workflow_id}", response_model=RepairRunStatus)
async def repair_run_status_endpoint(workflow_id: WorkflowID, mode: Mode):
    try:
        return await run_status(workflow_id, mode=mode)
    except ProfferError as error:
        raise _translate(error) from None
