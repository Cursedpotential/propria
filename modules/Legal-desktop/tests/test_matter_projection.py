"""Home Matter identity is an Agno projection. Fail closed. No evidence clone.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Drives Workspace.home() — the same function GET /v1/matter calls.
"""

from uuid import uuid4

import httpx
from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.agno_client import HEALTH_TIMEOUT
from legal_workspace.services.workspace import Workspace


def test_home_projects_agno_identity_and_persists(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    mid = uuid4()

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.url.path == "/v1/matters"
        return httpx.Response(
            200,
            json={
                "data": [
                    {
                        "id": str(mid),
                        "title": "Agno Genesee Matter",
                        "evidence_items": [{"hash": "do-not-clone"}],
                    }
                ]
            },
        )

    home = workspace.home(
        agno_http=httpx.Client(transport=httpx.MockTransport(handler), timeout=HEALTH_TIMEOUT)
    )
    assert home.matter.display_name == "Agno Genesee Matter"
    assert home.matter.matter_id == mid
    assert home.court_case.matter_id == mid
    dumped = home.matter.model_dump()
    assert "evidence_items" not in dumped
    reloaded = Workspace(tmp_path).load()
    assert reloaded.matter.display_name == "Agno Genesee Matter"
    assert reloaded.matter.matter_id == mid


def test_home_fail_closed_keeps_local_identity(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    before = workspace.load().matter.display_name
    before_id = workspace.load().matter.matter_id

    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("no evidence-platform")

    home = workspace.home(
        agno_http=httpx.Client(transport=httpx.MockTransport(handler), timeout=HEALTH_TIMEOUT)
    )
    assert home.matter.display_name == before
    assert home.matter.matter_id == before_id


def test_http_matter_home_ok_when_agno_down(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    local_name = store.load().matter.display_name
    response = TestClient(app).get("/v1/matter")
    assert response.status_code == 200
    body = response.json()
    assert body["matter"]["display_name"] == local_name
    assert "evidence_items" not in body["matter"]
