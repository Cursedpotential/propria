"""Focused tests for the mode-bound Proffer proposal resource catalog.

Byline amendment: Claude Code · Opus 5.5 · 2026-09-26 — unprovable runs are left out and counted.
"""

from __future__ import annotations

import asyncio
from uuid import UUID

import pytest
from app.runtime import proffer_resources as runtime
from app.service import matter_mode, proffer, proffer_operations, proffer_resources
from app.service.matter_mode import _clear_preview_modes_for_tests, bind_preview_mode
from app.types.proffer_content import ProfferContentResponse
from app.types.proffer_operations import ProfferOperationListResponse
from fastapi import HTTPException

HANDLE = "proposal_resource_handle_abcdefghijkl"
OTHER_HANDLE = "other_mode_resource_handle_abcdefghij"
MATTER_ID = "11111111-1111-4111-8111-111111111111"
COURT_CASE_ID = "22222222-2222-4222-8222-222222222222"


def _operation(handle: str = HANDLE) -> dict:
    return {
        "preview_handle": handle,
        "request_id": "request-1",
        "source_ref": "upload://" + "a" * 64,
        "service": "proffer",
        "created_at": "2026-09-13T12:00:00Z",
        "lifecycle": "awaiting_preview_decision",
        "current_stage": "await_preview_decision",
        "active_stages": [],
        "wait": "preview_decision",
        "terminal": False,
        "source_version_ref": "source-version-ref",
        "completed_stage_count": 6,
    }


def _content() -> ProfferContentResponse:
    return ProfferContentResponse.model_validate(
        {
            "preview_handle": HANDLE,
            "matter_mode": "TEST",
            "package": {
                "source_version_ref": "source-version-ref",
                "declared_format": "sms_xml",
                "status": "ready",
                "metadata_count": 0,
                "attachment_count": 0,
            },
            "attempt": {
                "projection_ref": "projection-ref",
                "source_version_ref": "source-version-ref",
                "raw_generation_ref": "raw-ref",
                "normalized_generation_ref": "normalized-ref",
                "receipts": [],
            },
            "attempts_complete": True,
            "records": [
                {
                    "record_id": "record-1",
                    "ordinal": 0,
                    "record_type": "message",
                    "payload": {"body": "hello"},
                    "source_locator_ref": "locator://record/1",
                }
            ],
            "attachments": [],
            "chunks": [],
        }
    )


@pytest.fixture(autouse=True)
def bindings():
    _clear_preview_modes_for_tests()
    bind_preview_mode(HANDLE, "TEST")
    bind_preview_mode(OTHER_HANDLE, "REAL")
    yield
    _clear_preview_modes_for_tests()


def _wire(monkeypatch: pytest.MonkeyPatch, *, items: list[dict] | None = None) -> None:
    async def fake_operations(**kwargs):
        return ProfferOperationListResponse.model_validate(
            {"items": items if items is not None else [_operation(), _operation(OTHER_HANDLE)]}
        )

    async def fake_content(*args, **kwargs):
        return _content()

    monkeypatch.setattr(proffer_operations, "list_operations", fake_operations)
    monkeypatch.setattr(proffer, "preview_content", fake_content)
    monkeypatch.setattr(matter_mode, "configured_matter_id", lambda mode: MATTER_ID)
    monkeypatch.setattr(matter_mode, "configured_court_case_id", lambda mode: COURT_CASE_ID)


def test_catalog_lists_only_proven_mode_bound_operations_and_open_paths(monkeypatch) -> None:
    _wire(monkeypatch)

    result = asyncio.run(proffer_resources.list_proposal_resources(mode="TEST", status=None, cursor=None, limit=50))

    assert result.scope == "context_review_resources"
    assert str(result.matter_id) == MATTER_ID
    assert str(result.court_case_id) == COURT_CASE_ID
    assert result.approval_destination == "neo4j"
    assert result.later_manual_projection == "surrealdb"
    assert [item.preview_handle for item in result.items] == [HANDLE]
    assert result.unbound_count == 0
    assert result.items[0].representation_state == "committed_readback"
    assert "already-persisted PostgreSQL Context rows" in result.items[0].representation_detail
    assert "per-attempt DuckDB manifest and digest" in result.items[0].representation_detail
    assert result.items[0].content_status == "available"
    assert result.items[0].record_preview_available is True
    assert result.items[0].open_path == f"/api/proffer/previews/{HANDLE}?mode=TEST"


def test_catalog_leaves_out_and_counts_runs_whose_mode_cannot_be_proven(monkeypatch) -> None:
    # Was test_catalog_fails_truthfully_when_operation_mode_cannot_be_proven: one unprovable run
    # used to refuse the whole catalog with a 503 and blank the owner's Review page.
    # Byline amendment: Claude Code · Opus 5.5 · 2026-09-26.
    unknown = "unbound_resource_handle_abcdefghijklmn"
    unknown_matter = "33333333-3333-4333-8333-333333333333"  # neither mode's configured matter
    _wire(
        monkeypatch,
        items=[
            _operation(),
            _operation(unknown),
            {**_operation("unknown_matter_resource_handle_abcdef"), "matter_id": unknown_matter},
            _operation(OTHER_HANDLE),
        ],
    )

    for mode, visible in (("TEST", [HANDLE]), ("REAL", [OTHER_HANDLE])):
        result = asyncio.run(proffer_resources.list_proposal_resources(mode=mode, status=None, cursor=None, limit=50))
        # Provable runs are listed under their own mode only; the two unprovable ones under neither.
        assert [item.preview_handle for item in result.items] == visible
        assert result.unbound_count == 2

    # Leaving a run out never binds it to a mode by implication.
    with pytest.raises(matter_mode.MatterModeError, match="no active TEST/REAL binding"):
        matter_mode.require_preview_mode(unknown, "TEST")


def test_catalog_rebinds_a_run_its_durable_matter_proves(monkeypatch) -> None:
    rebound = "rebound_resource_handle_abcdefghijklm"
    _wire(monkeypatch, items=[{**_operation(rebound), "matter_id": MATTER_ID}])
    real_matter = UUID("44444444-4444-4444-8444-444444444444")
    monkeypatch.setattr(
        matter_mode, "configured_matter_id", lambda mode: UUID(MATTER_ID) if mode == "TEST" else real_matter
    )

    result = asyncio.run(proffer_resources.list_proposal_resources(mode="TEST", status=None, cursor=None, limit=50))

    assert [item.preview_handle for item in result.items] == [rebound]
    assert result.unbound_count == 0


def test_catalog_fails_when_content_store_is_unavailable(monkeypatch) -> None:
    _wire(monkeypatch, items=[_operation()])

    async def unavailable(*args, **kwargs):
        raise proffer.ProfferError("Proffer starter unreachable", 503)

    monkeypatch.setattr(proffer, "preview_content", unavailable)

    with pytest.raises(proffer.ProfferError, match="starter unreachable"):
        asyncio.run(proffer_resources.list_proposal_resources(mode="TEST", status=None, cursor=None, limit=50))


def test_catalog_reports_not_ready_content_without_inventing_resources(monkeypatch) -> None:
    _wire(monkeypatch, items=[_operation()])

    async def pending(*args, **kwargs):
        raise proffer.ProfferError("preview projection is not ready", 409)

    monkeypatch.setattr(proffer, "preview_content", pending)
    result = asyncio.run(proffer_resources.list_proposal_resources(mode="TEST", status=None, cursor=None, limit=50))

    assert len(result.items) == 1
    assert result.items[0].content_status == "pending"
    assert result.items[0].content_reason == "preview projection is not ready"
    assert result.items[0].record_preview_available is False


def test_route_translates_catalog_store_failure(monkeypatch) -> None:
    async def unavailable(**kwargs):
        raise proffer.ProfferError("operation store unavailable", 503)

    monkeypatch.setattr(runtime, "list_proposal_resources", unavailable)
    with pytest.raises(HTTPException) as caught:
        asyncio.run(runtime.proposal_resources_endpoint("TEST", None, None, 50))

    assert caught.value.status_code == 503
    assert caught.value.detail == "operation store unavailable"
