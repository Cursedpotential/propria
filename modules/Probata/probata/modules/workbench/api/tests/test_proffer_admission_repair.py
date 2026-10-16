"""Verify authenticated BFF repair forwarding and real failure status without old mode inference.

Inputs: synthetic actual actor/version/case and engine replies. Outputs: source-bound
Temporal request/status assertions. Effects: mocked requests only; run on the VPS.
"""
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient
from app.runtime.proffer import router
from app.service import proffer_admission_repair as service


def test_explicit_repair_forwards_actual_actor_and_preserves_failure(monkeypatch):
    """Use existing Workbench identity and relay actual failed workflow details.

    Inputs: explicit LIVE owner action and failed engine status. Outputs: exact
    source/key/actor/body and failure assertions. Effects: mocked forwarding only.
    """
    calls = []
    monkeypatch.setattr(service, "require_scope", lambda *args: None)

    async def engine(method, path, **kwargs):
        """Record forwarding and return actual-shaped Temporal IDs or failure."""
        calls.append((method, path, kwargs))
        if method == "POST":
            return {"workflow_id": "preview-admission-repair:actual", "run_id": "actual-run"}
        return {"status": "failed", "reason": "source version differs", "workflow_id": "preview-admission-repair:actual", "run_id": "actual-run"}

    monkeypatch.setattr(service.proffer, "_request", engine)
    monkeypatch.setattr(service.proffer, "_json_payload", lambda value, _: value)
    app = FastAPI()

    @app.middleware("http")
    async def identity(request: Request, call_next):
        """Supply the actual-shaped existing identity seam without fabricating it in adapters."""
        request.state.subject_uid, request.state.principal = "owner-uid", "owner"
        return await call_next(request)

    app.include_router(router)
    client = TestClient(app)
    body = {"source_version_ref": "11111111-1111-4111-8111-111111111111", "matter_id": "22222222-2222-4222-8222-222222222222",
            "court_case_id": "33333333-3333-4333-8333-333333333333", "operating_mode": "LIVE", "reason": "Current owner readmission"}
    path = "/api/proffer/previews/synthetic-existing-handle/admission-repair"
    assert client.post(path, json={k:v for k,v in body.items() if k != "operating_mode"}, headers={"Idempotency-Key": "click-0001"}).status_code == 422
    response = client.post(path, json=body, headers={"Idempotency-Key": "click-0001"})
    assert response.status_code == 202, response.text
    assert calls[0][2]["json"] == body
    assert calls[0][2]["headers"] == {"X-authentik-uid": "owner-uid", "X-authentik-username": "owner", "Idempotency-Key": "click-0001"}
    result = client.get(response.json()["status_path"])
    assert result.status_code == 200 and result.json()["status"] == "failed"
    assert result.json()["reason"] == "source version differs"
