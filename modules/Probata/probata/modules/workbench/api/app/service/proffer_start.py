"""Single-source start and durable operation-policy proof.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
"""

from uuid import UUID

from app.service import preview_mode_recovery
from app.service.matter_mode import bind_preview_mode, require_scope
from app.service.proffer_errors import ProfferError
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferStartRequest, ProfferStartResponse


async def start(request: ProfferStartRequest, *, mode: MatterMode) -> ProfferStartResponse:
    """Start one Live import and bind only its explicit durable operating receipt.

    Inputs: fixed-case source request and canonical policy. Output: preview handle.
    Effects: engine POST then durable GET; Dev rejects before I/O. Pick over batch
    start for one source, never use request echo as durable mode evidence.
    """
    from app.service import proffer

    if mode == "DEV":
        raise ProfferError("Development writes require an isolated data workspace; no canonical write was dispatched", 409)
    if request.matter_mode != mode:
        raise ProfferError("matter_mode in the start body must match the mode query", 409)
    proffer._mode_call(require_scope, mode, request.matter_id, request.court_case_id)
    response = await proffer._request(
        "POST",
        "/reference-import/start",
        json={**request.model_dump(mode="json", exclude={"matter_mode"}), "operating_mode": mode},
    )
    result = proffer._validated(
        ProfferStartResponse,
        proffer._mode_payload(proffer._json_payload(response, "start response"), "start response", mode),
        "start response",
    )
    matter_id, operating_mode = await operation_binding(result.preview_handle)
    if preview_mode_recovery.mode_for_operation(matter_id, operating_mode) != mode:
        raise ProfferError("Started preview has no matching explicit durable operating mode", 502)
    proffer._mode_call(bind_preview_mode, result.preview_handle, mode)
    return result


async def operation_binding(preview_handle: str) -> tuple[UUID | None, str | None]:
    """Read durable case scope and explicit operating mode for a preview.

    Input: opaque preview handle. Output: matter UUID and canonical/raw flag.
    Effects: authenticated GET. Pick for restart/start proof; never infer a flag
    from source identity or accept a mismatched handle.
    """
    from app.service import proffer

    response = await proffer._request("GET", f"/reference-import/operations/{preview_handle}")
    raw = proffer._json_payload(response, "operation detail")
    if not isinstance(raw, dict) or raw.get("preview_handle") != preview_handle:
        raise ProfferError("Proffer operation detail correlation failed", 502)
    value = raw.get("matter_id") if isinstance(raw, dict) else None
    try:
        matter_id = UUID(value) if isinstance(value, str) else None
    except ValueError:
        matter_id = None
    operating_mode = raw.get("operating_mode") if isinstance(raw, dict) else None
    return matter_id, operating_mode if isinstance(operating_mode, str) else None
