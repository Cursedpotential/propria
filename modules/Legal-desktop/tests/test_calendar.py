"""Docket events persist. No invented seed dates.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from datetime import UTC, datetime, timedelta

import pytest
from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.domain.calendar import DocketEventCreate, EventKind
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace


def test_blank_workspace_has_no_docket_dates(tmp_path) -> None:
    state = Workspace(tmp_path).load()
    assert state.docket_events == []
    assert Workspace(tmp_path).upcoming_event_count() == 0


def test_past_and_upcoming_roundtrip(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    past = workspace.add_docket_event(
        DocketEventCreate(
            occurs_at=datetime.now(UTC) - timedelta(days=30),
            title="Prior FOC conference — owner recorded",
            kind=EventKind.FOC,
            source="owner",
            confirmed=False,
        )
    )
    future = workspace.add_docket_event(
        DocketEventCreate(
            occurs_at=datetime.now(UTC) + timedelta(days=14),
            title="Owner-entered hearing placeholder only if real",
            kind=EventKind.HEARING,
            source="owner",
            confirmed=False,
        )
    )
    reloaded = Workspace(tmp_path)
    ids = {item.event_id for item in reloaded.list_docket_events()}
    assert past.event_id in ids
    assert future.event_id in ids
    assert reloaded.upcoming_event_count() == 1
    ordered = reloaded.list_docket_events()
    assert ordered[0].occurs_at <= ordered[-1].occurs_at


def test_http_docket(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    # ContextForge JWT auth was added to the app after this test was written, so
    # the bare client 401s on every request. Same header as tests/test_api.py:43,
    # matching the secret conftest.py sets. — Claude Code · Opus 5 · 2026-08-23
    client.headers = {"Authorization": "Bearer test-jwt-secret-for-testing"}
    empty = client.get("/v1/docket-events")
    assert empty.status_code == 200
    assert empty.json() == []
    created = client.post(
        "/v1/docket-events",
        json={
            "occurs_at": (datetime.now(UTC) + timedelta(days=7)).isoformat(),
            "title": "Clerk-set date to be confirmed",
            "kind": "deadline",
            "source": "owner",
            "confirmed": False,
        },
    )
    assert created.status_code == 200, created.text
    home = client.get("/v1/matter")
    assert home.json()["upcoming_event_count"] == 1
    paths = {row["path"] for row in home.json()["next_surfaces"]}
    labels = {row["label"] for row in home.json()["next_surfaces"]}
    assert "/calendar" in paths and "Court dates" in labels
    assert "/timeline" in paths and "Timeline" in labels
    assert home.json()["upcoming_events"]
    event_id = created.json()["event_id"]
    removed = client.delete(f"/v1/docket-events/{event_id}")
    assert removed.status_code == 200
    assert client.get("/v1/docket-events").json() == []
    assert client.delete(f"/v1/docket-events/{event_id}").status_code == 404


# ---------------------------------------------------------------------------
# DELETE /v1/calendar/events/{event_id}
#
# Distinct from /v1/docket-events/{id} above — this is calendar_routes.py's own
# delete, mounted at main.py:900. It was previously untested.
#
# Byline amendment: Claude Code · Opus 5 · 2026-08-23
# ---------------------------------------------------------------------------


def test_calendar_delete_rejects_malformed_id_with_400() -> None:
    """A malformed event id must return 400, not crash the handler.

    Regression test: the handler called UUID(event_id) without importing UUID,
    so it raised NameError. NameError is not caught by the route's
    `except ValueError` / `except StopIteration` clauses, so every call to this
    endpoint returned an unhandled 500 instead of the intended 400.
    """
    client = TestClient(app)
    client.headers = {"Authorization": "Bearer test-jwt-secret-for-testing"}

    response = client.delete("/v1/calendar/events/not-a-uuid")

    assert response.status_code == 400, response.text
    assert "Invalid event ID format" in response.text


@pytest.mark.xfail(
    reason=(
        "Blocked by URGENT-TODO B4: the dev SQLite at data/workspace/legal.sqlite is "
        "columns behind the ORM, so this route's workspace lookup raises "
        "OperationalError: no such column: legal_core_matter_ref.last_agno_verify. "
        "The assertion below is the intended behavior; remove this xfail once the dev "
        "DB is rebuilt from the ORM."
    ),
    strict=False,
)
def test_calendar_delete_unknown_id_returns_404() -> None:
    """A well-formed id that matches no event must return 404."""
    client = TestClient(app)
    client.headers = {"Authorization": "Bearer test-jwt-secret-for-testing"}

    response = client.delete("/v1/calendar/events/00000000-0000-4000-8000-000000000000")

    assert response.status_code == 404, response.text
