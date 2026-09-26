"""Authenticated pass-through to the engine's repair workflow builder routes.

Byline: Claude Code · Opus 5.5 · 2026-09-26.

The engine owns the tool registry, the signature table, the fail-closed
validator and the Temporal `RepairPlanWorkflow` (`modules/engine/repairplan`;
routes in `modules/engine/runtimeapi/repair_plan_http.go`). This module adds
the Workbench's TEST/REAL discipline and nothing else, through the same client
and service token as every other Proffer route (`app.service.proffer`):

- propose, validate and run prove the plan's mode from its Review run's preview
  handle, with the same restart-safe recovery the preview routes use;
- run records workflow_id -> mode; runs/{id} proves the mode from the engine's
  own `matter_mode` echo, so the proof survives a BFF restart, and falls back to
  the recorded binding only when the engine names none. An id that neither can
  prove is refused with 409, never guessed.
"""

from __future__ import annotations

from threading import RLock
from typing import Any, TypeVar, cast

import httpx
from pydantic import BaseModel, ValidationError

from app.service import proffer
from app.service.proffer_errors import ProfferError
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

_MODES: tuple[MatterMode, ...] = ("TEST", "REAL")
_Model = TypeVar("_Model", bound=BaseModel)
_run_modes: dict[str, MatterMode] = {}
_run_modes_lock = RLock()


class RepairRunRefusedError(ProfferError):
    """The engine refused to start the plan (422). Carries its full checklist."""

    def __init__(self, refused: RepairRunRefused):
        super().__init__(refused.detail, 422)
        self.refused = refused


def _record_run_mode(workflow_id: str, mode: MatterMode) -> None:
    with _run_modes_lock:
        existing = _run_modes.get(workflow_id)
        if existing is not None and existing != mode:
            raise ProfferError("repair run is already bound to a different matter mode", 502)
        _run_modes[workflow_id] = mode


def _recorded_run_mode(workflow_id: str) -> MatterMode | None:
    with _run_modes_lock:
        return _run_modes.get(workflow_id)


def _clear_run_modes_for_tests() -> None:
    with _run_modes_lock:
        _run_modes.clear()


async def _require_plan_mode(plan: RepairPlan, mode: MatterMode) -> None:
    if plan.matter_mode != mode:
        raise ProfferError("matter_mode in the plan must match the mode query", 409)
    await proffer._require_mode(plan.preview_handle, mode)


def _engine_body(model: RepairPlan | RepairProposeRequest) -> dict[str, Any]:
    return model.model_dump(mode="json")


def _passthrough(response: httpx.Response, model: type[_Model], label: str, mode: MatterMode) -> _Model:
    payload = proffer._mode_payload(proffer._json_payload(response, label), label, mode)
    return proffer._validated(model, payload, label)


async def tools(*, mode: MatterMode) -> RepairToolsResponse:
    """Every repair-capable Activity, for the step picker."""
    proffer._require_mode_configuration(mode)
    response = await proffer._request("GET", "/reference-import/repair/tools")
    return _passthrough(response, RepairToolsResponse, "repair tool list", mode)


async def propose(request: RepairProposeRequest, *, mode: MatterMode) -> RepairProposeResponse:
    """The signature table's candidate plans for one Review run's source."""
    await proffer._require_mode(request.preview_handle, mode)
    response = await proffer._request("POST", "/reference-import/repair/propose", json=_engine_body(request))
    return _passthrough(response, RepairProposeResponse, "repair proposals", mode)


async def validate(plan: RepairPlan, *, mode: MatterMode) -> RepairValidateResponse:
    """Every named check against the plan; the engine fails closed."""
    await _require_plan_mode(plan, mode)
    response = await proffer._request("POST", "/reference-import/repair/validate", json=_engine_body(plan))
    return _passthrough(response, RepairValidateResponse, "repair plan validation", mode)


def _refusal(error: ProfferError) -> RepairRunRefusedError:
    body = error.payload if isinstance(error.payload, dict) else {}
    try:
        refused = RepairRunRefused.model_validate({"detail": error.detail, "checks": body.get("checks")})
    except ValidationError:
        # The refusal and its detail are the contract; a malformed checklist is dropped, not invented.
        refused = RepairRunRefused(detail=error.detail)
    return RepairRunRefusedError(refused)


async def run(plan: RepairPlan, *, mode: MatterMode) -> RepairRunResponse:
    """Start the plan on Temporal, only when the engine's own validation passes."""
    await _require_plan_mode(plan, mode)
    try:
        response = await proffer._request("POST", "/reference-import/repair/run", json=_engine_body(plan))
    except ProfferError as error:
        if error.status_code == 422:
            raise _refusal(error) from None
        raise
    result = _passthrough(response, RepairRunResponse, "repair run", mode)
    _record_run_mode(result.workflow_id, mode)
    return result


def _proven_run_mode(workflow_id: str, upstream: Any) -> MatterMode:
    recorded = _recorded_run_mode(workflow_id)
    if upstream in _MODES:
        if recorded is not None and recorded != upstream:
            raise ProfferError("the engine reports a different matter mode than this repair run was started in", 502)
        return cast(MatterMode, upstream)
    if recorded is None:
        raise ProfferError("repair run has no provable TEST/REAL binding; it is refused rather than guessed", 409)
    return recorded


async def run_status(workflow_id: str, *, mode: MatterMode) -> RepairRunStatus:
    """One run's per-step progress, receipts and re-entry, mode-checked from the engine's echo."""
    response = await proffer._request("GET", f"/reference-import/repair/runs/{workflow_id}")
    payload = proffer._json_payload(response, "repair run status")
    if not isinstance(payload, dict):
        raise ProfferError("Proffer starter returned an invalid repair run status", 502)
    if payload.get("workflow_id") != workflow_id:
        raise ProfferError("Proffer repair run status correlation failed", 502)
    proven = _proven_run_mode(workflow_id, payload.get("matter_mode"))
    if proven != mode:
        raise ProfferError("repair run belongs to a different matter mode", 409)
    result = proffer._validated(RepairRunStatus, {**payload, "matter_mode": proven}, "repair run status")
    _record_run_mode(workflow_id, proven)
    return result
