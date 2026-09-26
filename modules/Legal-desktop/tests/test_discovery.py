"""Evidence requests persists and cannot be marked served without a date.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from datetime import UTC, datetime

from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.domain.discovery import (
    DiscoveryCreate,
    DiscoveryKind,
    DiscoveryStatus,
    DiscoveryStatusUpdate,
)
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace


def test_seeded_requests_are_unserved(tmp_path) -> None:
    state = Workspace(tmp_path).load()
    assert state.discovery
    assert all(item.status is DiscoveryStatus.DRAFT for item in state.discovery)
    assert all(item.served_on is None for item in state.discovery)
    assert all(item.linked_issue for item in state.discovery)


def test_served_requires_date(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    added = workspace.add_discovery(
        DiscoveryCreate(
            kind=DiscoveryKind.RFA,
            text="Admit that no written specific-terms schedule currently exists.",
            purpose="Narrow the specific-terms request.",
            linked_issue="What the judge must consider (a)–(l) considered individually",
        )
    )
    try:
        workspace.set_discovery_status(
            added.request_id,
            DiscoveryStatusUpdate(status=DiscoveryStatus.SERVED),
        )
        raise AssertionError("served without date must fail")
    except ValueError as exc:
        assert "served_on" in str(exc)
    marked = workspace.set_discovery_status(
        added.request_id,
        DiscoveryStatusUpdate(status=DiscoveryStatus.SERVED, served_on=datetime.now(UTC)),
    )
    assert marked.served_on is not None
    assert Workspace(tmp_path).load().discovery


def test_http_discovery(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    listed = client.get("/v1/discovery")
    assert listed.status_code == 200
    assert len(listed.json()) >= 2
    created = client.post(
        "/v1/discovery",
        json={
            "kind": "rfp",
            "text": "Produce school attendance records for the last semester.",
            "purpose": "Factor (b)/(h) only if those records become an issue.",
            "linked_issue": "What the judge must consider (a)–(l) considered individually",
            "missing_proof": "No school records in an approved package.",
        },
    )
    assert created.status_code == 200, created.text
    request_id = created.json()["request_id"]
    blocked = client.post(
        f"/v1/discovery/{request_id}:status",
        json={"status": "served"},
    )
    assert blocked.status_code == 409
    home = client.get("/v1/matter")
    assert any(row["path"] == "/evidence-requests" and row["label"] == "Evidence requests" for row in home.json()["next_surfaces"])
    assert home.json()["discovery_count"] >= 3
