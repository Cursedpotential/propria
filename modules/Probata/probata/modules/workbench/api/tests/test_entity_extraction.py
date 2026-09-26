"""Entity/event extraction BFF: mode-bound, actor-bound, idempotent pass-through.

Byline: Claude Code · Opus 5.5 · 2026-09-25
"""

from __future__ import annotations

import httpx
import pytest
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient

from app.runtime import entity_extraction as runtime
from app.service import proffer
from app.service.proffer_errors import ProfferError

HANDLE = "handle_abcdefghijklmnopqrstuvwxyz0123"
DIGEST = "a" * 64


def _app() -> FastAPI:
    app = FastAPI()

    @app.middleware("http")
    async def actor(request: Request, call_next):
        request.state.subject_uid = "subject-1"
        request.state.principal = "operator"
        return await call_next(request)

    app.include_router(runtime.router)
    return app


class _Engine:
    """Records every starter call and answers from a script."""

    def __init__(self, answers: dict[tuple[str, str], tuple[int, object]]):
        self.answers = answers
        self.calls: list[dict] = []

    async def request(self, method: str, path: str, **kwargs):
        self.calls.append({"method": method, "path": path, **kwargs})
        status, payload = self.answers[(method, path)]
        if status >= 400:
            raise ProfferError(payload if isinstance(payload, str) else "rejected", status)
        return httpx.Response(status, json=payload)


@pytest.fixture
def engine(monkeypatch):
    modes: list[tuple[str, str]] = []

    async def require_mode(handle, mode):
        modes.append((handle, mode))

    stub = _Engine({})
    stub.modes = modes
    monkeypatch.setattr(proffer, "_require_mode", require_mode)
    monkeypatch.setattr(proffer, "_request", stub.request)
    return stub


def test_extract_forwards_actor_key_and_mode(engine) -> None:
    engine.answers[("POST", "/reference-import/entities/extract")] = (
        202,
        {
            "workflow_id": f"entity-extraction:{HANDLE}:x",
            "run_id": "r",
            "extraction_id": "e",
            "normalized_generation_id": "g",
            "use_model": True,
            "matter_mode": "REAL",
        },
    )
    response = TestClient(_app()).post(
        "/api/entities/extract",
        params={"mode": "REAL"},
        json={"preview_handle": HANDLE},
        headers={"Idempotency-Key": "click-0001"},
    )
    assert response.status_code == 202, response.text
    call = engine.calls[0]
    assert call["json"] == {"preview_handle": HANDLE, "use_model": True, "matter_mode": "REAL"}
    assert call["headers"] == {
        "X-authentik-uid": "subject-1",
        "X-authentik-username": "operator",
        "Idempotency-Key": "click-0001",
    }
    assert engine.modes == [(HANDLE, "REAL")]


def test_writes_require_an_idempotency_key(engine) -> None:
    client = TestClient(_app())
    for path, body in (
        ("/api/entities/extract", {"preview_handle": HANDLE}),
        ("/api/entities/commit", {"preview_handle": HANDLE, "digest": DIGEST}),
        ("/api/events/from-record", {"preview_handle": HANDLE, "record_id": "r-1"}),
    ):
        assert client.post(path, params={"mode": "REAL"}, json=body).status_code == 422
    assert engine.calls == []


def test_progress_is_scoped_to_the_runs_own_workflows(engine) -> None:
    client = TestClient(_app())
    foreign = client.get(
        "/api/entities/commits/proffer-run-12345678",
        params={"mode": "REAL", "preview_handle": HANDLE},
    )
    assert foreign.status_code == 404
    assert engine.calls == []
    workflow_id = f"extraction-commit:{HANDLE}:abc"
    engine.answers[("GET", f"/reference-import/entities/commits/extraction-commit%3A{HANDLE}%3Aabc")] = (
        200,
        {"workflow_id": workflow_id, "outcome": "committed", "steps": [{"step": "validate", "status": "completed"}]},
    )
    own = client.get(f"/api/entities/commits/{workflow_id}", params={"mode": "REAL", "preview_handle": HANDLE})
    assert own.status_code == 200, own.text
    assert own.json()["steps"][0]["status"] == "completed"
    assert own.json()["matter_mode"] == "REAL"


def test_corrections_forward_one_target_and_pass_conflicts_through(engine) -> None:
    client = TestClient(_app())
    mismatched = client.post(
        "/api/entities/corrections",
        params={"mode": "REAL"},
        json={"preview_handle": HANDLE, "target": "entity", "event": {"op": "reject"}},
        headers={"Idempotency-Key": "fix-000001"},
    )
    assert mismatched.status_code == 422
    assert engine.calls == []
    engine.answers[("POST", "/reference-import/entities/corrections")] = (409, "proposal changed since it was loaded")
    stale = client.post(
        "/api/entities/corrections",
        params={"mode": "REAL"},
        json={
            "preview_handle": HANDLE,
            "target": "entity",
            "entity": {"op": "rename", "candidate_ids": ["x"], "name": "Y"},
        },
        headers={"Idempotency-Key": "fix-000002"},
    )
    assert stale.status_code == 409
    assert engine.calls[0]["json"]["entity"]["op"] == "rename"


def test_validate_commit_and_mark_event(engine) -> None:
    engine.answers[("POST", "/reference-import/entities/validate")] = (
        200,
        {
            "ok": False,
            "digest": DIGEST,
            "checks": [{"rule": "live_mode", "status": "fail", "reason": "TEST run"}],
            "counts": {},
        },
    )
    engine.answers[("POST", "/reference-import/entities/commit")] = (422, "validation failed; fix the failing checks")
    engine.answers[("POST", "/reference-import/events/from-record")] = (
        201,
        {"event": {"title": "Pickup", "detected_by": "owner"}, "source_available_from": "2026-09-21T01:14:39Z"},
    )
    client = TestClient(_app())
    report = client.post("/api/entities/validate", params={"mode": "TEST"}, json={"preview_handle": HANDLE})
    assert report.status_code == 200
    assert report.json()["checks"][0] == {"rule": "live_mode", "status": "fail", "reason": "TEST run"}
    commit = client.post(
        "/api/entities/commit",
        params={"mode": "TEST"},
        json={"preview_handle": HANDLE, "digest": DIGEST},
        headers={"Idempotency-Key": "commit-0001"},
    )
    assert commit.status_code == 422
    marked = client.post(
        "/api/events/from-record",
        params={"mode": "TEST"},
        json={"preview_handle": HANDLE, "record_id": "r-1", "title": "Pickup"},
        headers={"Idempotency-Key": "mark-00001"},
    )
    assert marked.status_code == 201, marked.text
    assert marked.json()["event"]["detected_by"] == "owner"
    sent = engine.calls[-1]["json"]
    assert sent == {"preview_handle": HANDLE, "record_id": "r-1", "title": "Pickup", "matter_mode": "TEST"}


def test_record_and_registry_reads(engine) -> None:
    engine.answers[("GET", "/reference-import/entities/records/r-1")] = (
        200,
        {
            "record_id": "r-1",
            "ordinal": 4,
            "occurred_at": "2025-06-01T14:46:00Z",
            "body": "Court on Thursday",
            "participants": [],
        },
    )
    engine.answers[("GET", "/reference-import/entities/registry")] = (
        200,
        {"entities": [{"id": "e-1", "display_name": "Katherine Doe", "registry_type": "person"}]},
    )
    client = TestClient(_app())
    record = client.get("/api/entities/records/r-1", params={"mode": "REAL", "preview_handle": HANDLE})
    assert record.status_code == 200
    assert record.json()["body"] == "Court on Thursday"
    registry = client.get("/api/entities/registry", params={"q": "kath"})
    assert registry.status_code == 200
    assert registry.json()["entities"][0]["display_name"] == "Katherine Doe"


def test_main_app_mounts_the_extraction_routes_beside_governed_entities() -> None:
    import main

    paths = {getattr(route, "path", "") for route in main.app.routes}
    assert "/api/entities" in paths  # governed search/create stays
    for path in ("/api/entities/extract", "/api/entities/proposals", "/api/entities/commit", "/api/events/from-record"):
        assert path in paths
