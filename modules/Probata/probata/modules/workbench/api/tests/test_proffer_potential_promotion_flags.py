"""Potential-promotion annotations stay reversible, attempt-bound, and actor-bound."""

from __future__ import annotations

import asyncio
import hashlib
import hmac
import json
from typing import Any, Literal

import httpx
import pytest
from app.runtime import proffer as proffer_runtime
from app.service import flags as flags_service
from app.service import proffer as proffer_service
from app.service import proffer_flags
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferContentResponse, ProfferDecisionActor
from app.types.proffer_flags import (
    ProfferPotentialPromotionFlag,
    ProfferPotentialPromotionFlagRequest,
)
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient

PREVIEW_HANDLE = "preview_handle_abcdefghijklmnopqrstuvwxyz"
ATTEMPT_ID = "attempt://normalized-1"


def _content() -> ProfferContentResponse:
    return ProfferContentResponse.model_validate(
        {
            "preview_handle": PREVIEW_HANDLE,
            "matter_mode": "LIVE",
            "package": {
                "source_version_ref": "source-version-1",
                "declared_format": "document",
                "status": "retained",
                "metadata_count": 1,
                "attachment_count": 0,
            },
            "attempt": {
                "attempt_ref": ATTEMPT_ID,
                "projection_ref": "normalized-1",
                "source_version_ref": "source-version-1",
                "raw_generation_ref": "raw-1",
                "normalized_generation_ref": "normalized-1",
                "receipts": [],
            },
            "attempts_complete": False,
            "attempts_reason": "Current attempt only",
            "records": [
                {
                    "record_id": "record-1",
                    "ordinal": 0,
                    "record_type": "document",
                    "payload": {"title": "Visible record"},
                    "source_locator_ref": "locator://record-1",
                }
            ],
            "attachments": [],
            "chunk_generation": {
                "generation_ref": "generation-1",
                "generation_ordinal": 1,
                "status": "sealed",
                "policy_id": "document",
                "policy_version": "1",
                "chunker_id": "offsets",
                "chunker_version": "1",
                "schema_version": "1",
                "source_view": "original",
                "source_sha256": "a" * 64,
                "receipt_ref": "receipt-1",
            },
            "chunks": [
                {
                    "chunk_ref": "chunk-1",
                    "index": 0,
                    "content": "Visible chunk",
                    "sha256": "b" * 64,
                    "derivation_mode": "verbatim_span",
                    "locator_ref": "locator://chunk-1",
                    "byte_start": 0,
                    "byte_end": 13,
                }
            ],
        }
    )


def _app() -> FastAPI:
    app = FastAPI()

    @app.middleware("http")
    async def actor(request: Request, call_next):
        request.state.subject_uid = "subject-1"
        request.state.principal = "operator"
        return await call_next(request)

    app.include_router(proffer_runtime.router)
    return app


def test_flag_route_binds_exact_off_page_target_attempt_and_authenticated_actor(monkeypatch) -> None:
    captured: dict[str, Any] = {}

    async def require_mode(*args, **kwargs):
        captured["mode_check"] = (args, kwargs)

    def create(preview_handle, mode, body, actor):
        captured.update(
            preview_handle=preview_handle,
            mode=mode,
            body=body,
            actor=actor,
        )
        return ProfferPotentialPromotionFlag(
            flag_id="flag-1",
            preview_handle=preview_handle,
            matter_mode=mode,
            scope=body.scope,
            target_id=body.target_id,
            attempt_id=body.attempt_id,
            reason=body.reason,
            actor_subject_uid=actor.subject_uid,
            actor_username=actor.username,
            flagged_at="2026-09-13T20:00:00Z",
            status="open",
        )

    monkeypatch.setattr(proffer_runtime, "require_preview_mode", require_mode)
    monkeypatch.setattr(
        proffer_runtime,
        "preview_content",
        lambda *_args, **_kwargs: (_ for _ in ()).throw(AssertionError("presentation page must not run")),
    )
    monkeypatch.setattr(proffer_runtime, "create_potential_promotion_flag", create)
    response = TestClient(_app()).post(
        f"/api/proffer/previews/{PREVIEW_HANDLE}/potential-promotion-flags",
        params={"mode": "LIVE"},
        json={
            "scope": "record",
            "target_id": "record-301",
            "attempt_id": ATTEMPT_ID,
            "reason": "Review this record later",
        },
    )

    assert response.status_code == 201
    assert response.json()["classification"] == "potential_promotion"
    assert captured["actor"] == ProfferDecisionActor(subject_uid="subject-1", username="operator")
    assert captured["body"].attempt_id == ATTEMPT_ID
    assert captured["preview_handle"] == PREVIEW_HANDLE
    assert captured["mode_check"][1] == {"mode": "LIVE"}


def test_flag_route_rejects_stale_and_advanced_attempts_at_atomic_admission(monkeypatch) -> None:
    current_attempt = ATTEMPT_ID
    persisted: list[ProfferPotentialPromotionFlag] = []

    async def require_mode(*_args, **_kwargs):
        return None

    def atomic_create(_handle, _mode, request, _actor):
        if request.attempt_id != current_attempt or request.target_id != "chunk-1":
            raise proffer_runtime.ProfferError("Preview attempt changed or target is no longer present", 409)
        persisted.append(
            ProfferPotentialPromotionFlag(
                flag_id="flag-1",
                preview_handle=PREVIEW_HANDLE,
                matter_mode="LIVE",
                scope=request.scope,
                target_id=request.target_id,
                attempt_id=request.attempt_id,
                reason=request.reason,
                actor_subject_uid="subject-1",
                actor_username="operator",
                flagged_at="2026-09-13T20:00:00Z",
                status="open",
            )
        )
        return persisted[-1]

    monkeypatch.setattr(proffer_runtime, "require_preview_mode", require_mode)
    monkeypatch.setattr(proffer_runtime, "create_potential_promotion_flag", atomic_create)
    client = TestClient(_app())
    base = {
        "scope": "chunk",
        "target_id": "chunk-1",
        "attempt_id": ATTEMPT_ID,
        "reason": "Review this chunk later",
    }

    stale = client.post(
        f"/api/proffer/previews/{PREVIEW_HANDLE}/potential-promotion-flags",
        params={"mode": "LIVE"},
        json={**base, "attempt_id": "attempt://stale"},
    )
    current_attempt = "attempt://advanced-after-form-submit"
    advanced = client.post(
        f"/api/proffer/previews/{PREVIEW_HANDLE}/potential-promotion-flags",
        params={"mode": "LIVE"},
        json=base,
    )
    unseen = client.post(
        f"/api/proffer/previews/{PREVIEW_HANDLE}/potential-promotion-flags",
        params={"mode": "LIVE"},
        json={**base, "target_id": "chunk-unseen"},
    )

    assert stale.status_code == 409
    assert "changed" in stale.json()["detail"]
    assert advanced.status_code == 409
    assert "changed" in advanced.json()["detail"]
    assert unseen.status_code == 409
    assert "no longer present" in unseen.json()["detail"]
    assert persisted == []


def test_flag_route_rejects_entity_without_a_governed_reader(monkeypatch) -> None:
    async def require_mode(*_args, **_kwargs):
        return None

    monkeypatch.setattr(proffer_runtime, "require_preview_mode", require_mode)
    monkeypatch.setattr(
        proffer_runtime,
        "create_potential_promotion_flag",
        lambda *_: (_ for _ in ()).throw(AssertionError("flag store must not run")),
    )
    response = TestClient(_app()).post(
        f"/api/proffer/previews/{PREVIEW_HANDLE}/potential-promotion-flags",
        params={"mode": "LIVE"},
        json={
            "scope": "entity",
            "target_id": "entity-1",
            "attempt_id": ATTEMPT_ID,
            "reason": "Inspect this entity",
        },
    )
    assert response.status_code == 422


def test_exact_target_service_binds_mode_and_rejects_uncorrelated_response(monkeypatch) -> None:
    calls: list[tuple[Literal["mode"], str, MatterMode] | tuple[Literal["request"], str, str, dict[str, str]]] = []

    async def require_mode(handle, mode):
        calls.append(("mode", handle, mode))

    async def request(method, path, **kwargs):
        calls.append(("request", method, path, kwargs["params"]))
        return httpx.Response(200, json={"preview_handle": PREVIEW_HANDLE, "attempt_id": ATTEMPT_ID, "found": True})

    monkeypatch.setattr(proffer_service, "_require_mode", require_mode)
    monkeypatch.setattr(proffer_service, "_request", request)
    assert asyncio.run(
        proffer_service.preview_content_target(PREVIEW_HANDLE, mode="LIVE", scope="record", target_id="record-301")
    ) == (ATTEMPT_ID, True)
    assert calls == [
        ("mode", PREVIEW_HANDLE, "LIVE"),
        (
            "request",
            "GET",
            f"/reference-import/previews/{PREVIEW_HANDLE}/content-target",
            {"scope": "record", "target_id": "record-301"},
        ),
    ]

    async def wrong_handle(*_args, **_kwargs):
        return httpx.Response(200, json={"preview_handle": "other", "attempt_id": ATTEMPT_ID, "found": True})

    monkeypatch.setattr(proffer_service, "_request", wrong_handle)
    with pytest.raises(proffer_service.ProfferError, match="invalid exact preview content target"):
        asyncio.run(
            proffer_service.preview_content_target(PREVIEW_HANDLE, mode="LIVE", scope="record", target_id="record-301")
        )


def test_flag_service_persists_governed_metadata_without_promoting(monkeypatch) -> None:
    captured: dict[str, str] = {}
    returned_notes = ""

    def create(payload):
        nonlocal returned_notes
        captured.update(payload)
        returned_notes = json.dumps(
            {
                "actor_subject_uid": payload["actor_subject_uid"],
                "actor_username": payload["actor_username"],
                "attempt_id": payload["attempt_id"],
                "classification": "potential_promotion",
                "contract": "proffer-potential-promotion/v1",
                "matter_mode": payload["matter_mode"],
                "preview_handle": payload["preview_handle"],
                "scope": payload["scope"],
                "target_id": payload["target_id"],
            }
        )
        return {
            "flag_id": "flag-1",
            "target_kind": "run",
            "target_id": payload["preview_handle"],
            "claim": payload["claim"],
            "notes": returned_notes,
            "created_at": "2026-09-13T20:00:00Z",
            "status": "open",
        }

    monkeypatch.setattr(proffer_flags.flags_service, "create_proffer_potential_promotion_flag", create)
    request = ProfferPotentialPromotionFlagRequest(
        scope="chunk",
        target_id="chunk-1",
        attempt_id=ATTEMPT_ID,
        reason="Review this chunk later",
    )
    actor = ProfferDecisionActor(subject_uid="subject-1", username="operator")

    result = proffer_flags.create_potential_promotion_flag(PREVIEW_HANDLE, "LIVE", request, actor)

    notes = json.loads(returned_notes)
    assert captured == {
        "preview_handle": PREVIEW_HANDLE,
        "matter_mode": "LIVE",
        "scope": "chunk",
        "target_id": "chunk-1",
        "attempt_id": ATTEMPT_ID,
        "actor_subject_uid": "subject-1",
        "actor_username": "operator",
        "claim": "Review this chunk later",
    }
    assert notes["preview_handle"] == PREVIEW_HANDLE
    assert notes["attempt_id"] == ATTEMPT_ID
    assert notes["scope"] == "chunk"
    assert notes["target_id"] == "chunk-1"
    assert result.classification == "potential_promotion"
    assert result.attempt_id == ATTEMPT_ID


def test_flag_list_is_scoped_to_preview_handle(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def spine_json(method, path, **kwargs):
        captured.update(method=method, path=path, params=kwargs["params"])
        return {"flags": []}

    monkeypatch.setattr(proffer_flags.flags_service, "spine_json", spine_json)

    assert proffer_flags.list_potential_promotion_flags(PREVIEW_HANDLE, "LIVE") == []
    assert captured == {
        "method": "GET",
        "path": "/v1/flags/proffer-potential-promotion",
        "params": {"preview_handle": PREVIEW_HANDLE},
    }


def test_ordinary_run_notes_are_not_governed_proffer_flags() -> None:
    assert proffer_flags._normalize({"notes": "ordinary free text"}) is None
    assert proffer_flags._normalize({"notes": json.dumps({"contract": "another-contract"})}) is None


def test_flag_service_signs_authenticated_actor_and_exact_payload(monkeypatch, tmp_path) -> None:
    key = b"test-only-proffer-delegation-key-1234567890"
    key_file = tmp_path / "delegation-key"
    key_file.write_bytes(key)
    monkeypatch.setattr(flags_service, "_PROFFER_DELEGATION_KEY_FILE", key_file)
    captured: dict[str, Any] = {}

    def spine_json(method, path, **kwargs):
        captured.update(method=method, path=path, **kwargs)
        return {"flag_id": "flag-1"}

    monkeypatch.setattr(flags_service, "spine_json", spine_json)
    payload = {
        "preview_handle": PREVIEW_HANDLE,
        "matter_mode": "LIVE",
        "scope": "record",
        "target_id": "33333333-3333-3333-3333-333333333333",
        "attempt_id": "11111111-1111-1111-1111-111111111111",
        "actor_subject_uid": "subject-1",
        "actor_username": "operator",
        "claim": "Review later",
    }
    assert flags_service.create_proffer_potential_promotion_flag(payload) == {"flag_id": "flag-1"}
    issued_at = captured["headers"]["X-Proffer-Flag-Issued-At"]
    request_bytes = json.dumps(payload, sort_keys=True, separators=(",", ":")).encode()
    expected_key = hashlib.sha256(request_bytes).hexdigest()
    assert captured["json"] == {**payload, "idempotency_key": expected_key}
    signed = json.dumps(captured["json"], sort_keys=True, separators=(",", ":")).encode()
    expected = hmac.new(key, issued_at.encode() + b"." + signed, hashlib.sha256).hexdigest()
    assert captured["headers"]["X-Proffer-Flag-Signature"] == expected
