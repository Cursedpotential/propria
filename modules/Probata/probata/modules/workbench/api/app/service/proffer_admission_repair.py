"""Forward explicit readmission to existing Temporal starter with actual owner attribution.

Inputs: existing preview/version/case and actor/key. Outputs: durable attempt and
real status. Effects: service-authenticated forwarding only, no local repair engine.
"""
from urllib.parse import quote
from app.service import proffer
from app.service.matter_mode import MatterModeError, require_scope
from app.service.proffer_errors import ProfferError
from app.types.proffer_admission_repair import AdmissionRepairRequest


async def start_repair(handle, body: AdmissionRepairRequest, actor, key: str) -> dict:
    """Start the standalone repair workflow without requiring the invalid old mode cache.

    Inputs: explicit current scope and actual owner. Output: workflow/run/BFF status path.
    Effects: existing Proffer request only; no source changes or inferred old admission.
    """
    try:
        require_scope(body.operating_mode, body.matter_id, body.court_case_id)
    except MatterModeError as exc:
        raise ProfferError(exc.detail, exc.status_code) from exc
    headers = {"X-authentik-uid": actor.subject_uid, "X-authentik-username": actor.username, "Idempotency-Key": key}
    response = await proffer._request("POST", f"/reference-import/previews/{quote(handle, safe='')}/admission-repair",
        json=body.model_dump(mode="json"), headers=headers)
    value = proffer._json_payload(response, "admission repair")
    if not isinstance(value, dict) or not value.get("workflow_id") or not value.get("run_id"):
        raise ProfferError("Repair returned no durable Temporal attempt", 502)
    value["status_path"] = f"/api/proffer/admission-repairs/{quote(value['workflow_id'], safe='')}"
    return value


async def repair_status(workflow_id: str, actor) -> dict:
    """Read real repair failure/details or the appended receipt with actual actor forwarding.

    Inputs: returned workflow ID and authenticated owner. Output: durable result.
    Effects: existing service read; choose for in-app status, never direct Activity invocation.
    """
    if not workflow_id.startswith("preview-admission-repair:") or len(workflow_id) > 220:
        raise ProfferError("Unknown admission repair attempt", 404)
    response = await proffer._request("GET", f"/reference-import/admission-repairs/{quote(workflow_id, safe='')}",
        headers={"X-authentik-uid": actor.subject_uid, "X-authentik-username": actor.username})
    return proffer._json_payload(response, "admission repair status")
