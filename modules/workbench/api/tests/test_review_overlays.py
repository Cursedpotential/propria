"""Review metadata screen, sidecars, corrections and context review (BFF boundary).

Byline: Claude Code · Opus 5.5 · 2026-09-26.

The engine is replaced by recorded answers here; these tests pin the BFF's own
rules (mode first, fail-closed correlation, the as-lived horizon never carrying
the foreshadowing member, actor-bound idempotent writes, sidecar lookup and
conflict reporting). They are not live proof of the deployed stack.
"""

from __future__ import annotations

import asyncio
import json

import httpx
import pytest
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient

from app.runtime import review_overlays as runtime
from app.service import context_review, source_metadata, source_sidecars
from app.service.proffer_errors import ProfferError
from app.types.context_review import ContextReviewRequest, ForeshadowingRequest
from app.types.proffer import ProfferDecisionActor
from app.types.source_metadata import MetadataCorrectionRequest, MetadataRow, Sidecar, SidecarLookup

HANDLE = "review_overlay_handle_abcdefghijklmnop"
MESSAGE = "0190a000-0000-7000-8000-0000000000e1"
SHA = "ab" * 32
ACTOR = ProfferDecisionActor(subject_uid="authentik-user-1", username="owner")
NOW = "2026-09-26T12:00:00Z"


def _engine_view(**overrides):
    view = {
        "preview_handle": HANDLE,
        "request_id": "request-1",
        "source_ref": "b2://proof-bucket/vault/IMG_0001.jpg",
        "subject_kind": "source",
        "subject_sha256": SHA,
        "source": {
            "source_version_ref": "44444444-4444-4444-4444-444444444444",
            "declared_format": "image",
            "original": {
                "object_ref": "o1",
                "storage_class": "filesystem",
                "sha256": SHA,
                "byte_length": 10,
                "immutable_at": NOW,
            },
        },
        "metadata": [
            {
                "metadata_ref": "m1",
                "metadata_class": "embedded",
                "extractor_id": "reader",
                "generated_at": NOW,
                "receipt_ref": "r1",
                "fields": {"EXIF:DateTimeOriginal": "2021:05:01 10:00:00"},
                "fields_bytes": 40,
            }
        ],
        "members": [],
        "hashes": [],
        "corrections": [],
        "corrections_available": True,
    }
    view.update(overrides)
    return view


def _wire(monkeypatch, module, answers: list, calls: list) -> None:
    async def fake_require_mode(handle, mode):
        calls.append(("mode", handle, mode))

    async def fake_request(method, path, **kwargs):
        calls.append((method, path, kwargs))
        status, payload = answers.pop(0)
        return httpx.Response(status, json=payload)

    monkeypatch.setattr(module, "_require_mode", fake_require_mode)
    monkeypatch.setattr(module, "_request", fake_request)


def test_metadata_screen_proves_mode_first_and_adds_sidecars(monkeypatch) -> None:
    calls: list = []
    _wire(monkeypatch, source_metadata, [(200, _engine_view())], calls)
    sidecar = Sidecar(
        name="IMG_0001.jpg.json",
        key="vault/IMG_0001.jpg.json",
        kind="takeout_json",
        found_via="beside_object",
        fields=[{"path": "photoTakenTime.timestamp", "value": "1620468000"}],
    )
    monkeypatch.setattr(
        source_metadata,
        "find_sidecars",
        lambda ref: ([sidecar], SidecarLookup(beside_object="ok", catalog_folder="not_configured")),
    )

    screen = asyncio.run(source_metadata.metadata_screen(HANDLE, mode="TEST"))

    assert calls[0] == ("mode", HANDLE, "TEST")
    assert calls[1][:2] == ("GET", f"/reference-import/previews/{HANDLE}/metadata")
    assert screen.matter_mode == "TEST"
    assert [item.name for item in screen.sidecars] == ["IMG_0001.jpg.json"]
    # 2021-05-08 10:00 UTC (sidecar) vs 2021-05-01 10:00 wall clock (file): a week apart.
    assert [conflict.topic for conflict in screen.sidecar_conflicts] == ["capture_time"]
    assert screen.sidecar_lookup.catalog_folder == "not_configured"


def test_metadata_screen_fails_closed_on_a_crossed_answer(monkeypatch) -> None:
    _wire(
        monkeypatch, source_metadata, [(200, _engine_view(preview_handle="another_handle_abcdefghijklmnopqrstu"))], []
    )
    monkeypatch.setattr(
        source_metadata, "find_sidecars", lambda ref: ([], SidecarLookup(beside_object="ok", catalog_folder="ok"))
    )
    with pytest.raises(ProfferError) as crossed:
        asyncio.run(source_metadata.metadata_screen(HANDLE, mode="TEST"))
    assert crossed.value.status_code == 502

    _wire(monkeypatch, source_metadata, [(200, _engine_view(subject_kind="attachment", subject_sha256="cd" * 32))], [])
    with pytest.raises(ProfferError) as other_file:
        asyncio.run(source_metadata.metadata_screen(HANDLE, mode="TEST", subject_sha256="ef" * 32))
    assert other_file.value.status_code == 502


def test_corrections_are_actor_bound_and_idempotent(monkeypatch) -> None:
    calls: list = []
    receipt = {
        "correction_ref": "c1",
        "receipt_ref": "metadata-correction://c1",
        "content_digest": "d" * 64,
        "revision": 1,
        "recorded_at": NOW,
    }
    _wire(monkeypatch, source_metadata, [(201, receipt), (201, receipt)], calls)
    request = MetadataCorrectionRequest(
        subject_sha256=SHA,
        field_key="embedded:EXIF:DateTimeOriginal",
        action="correct",
        source_value="2021:05:01 10:00:00",
        corrected_value="2021:05:01 22:00:00",
        change_reason="camera clock was twelve hours off",
    )

    first = asyncio.run(source_metadata.correct_metadata(HANDLE, request, ACTOR, mode="REAL"))
    asyncio.run(source_metadata.correct_metadata(HANDLE, request, ACTOR, mode="REAL"))

    posts = [call for call in calls if call[0] == "POST"]
    assert first.matter_mode == "REAL" and first.revision == 1
    assert posts[0][2]["headers"]["X-authentik-uid"] == "authentik-user-1"
    assert posts[0][2]["headers"]["Idempotency-Key"].startswith("metadata-correction:")
    assert posts[0][2]["headers"]["Idempotency-Key"] == posts[1][2]["headers"]["Idempotency-Key"], (
        "a retry reuses its key"
    )
    assert posts[0][2]["json"]["corrected_value"] == "2021:05:01 22:00:00"


def test_correction_request_rules() -> None:
    with pytest.raises(ValueError):
        MetadataCorrectionRequest(
            subject_sha256=SHA, field_key="embedded:EXIF:Make", action="correct", change_reason="x"
        )
    with pytest.raises(ValueError):
        MetadataCorrectionRequest(
            subject_sha256=SHA, field_key="embedded:EXIF:Make", action="retract", change_reason="x"
        )
    with pytest.raises(ValueError):
        MetadataCorrectionRequest(
            subject_sha256=SHA, field_key="Make", action="correct", corrected_value="x", change_reason="x"
        )


def _review(**overrides):
    view = {
        "preview_handle": HANDLE,
        "message_id": MESSAGE,
        "horizon": "as_lived",
        "reviews": [],
        "foreshadowing": [
            {
                "flag_ref": "f1",
                "revision": 1,
                "foreshadowing": True,
                "note": "",
                "horizon": "hindsight",
                "knowledge_time": NOW,
                "change_reason": "x",
                "actor_username": "owner",
                "receipt_ref": "foreshadowing://f1",
            }
        ],
    }
    view.update(overrides)
    return view


def test_as_lived_read_never_carries_foreshadowing_even_if_the_engine_leaks_it(monkeypatch) -> None:
    calls: list = []
    _wire(monkeypatch, context_review, [(200, _review())], calls)
    view = asyncio.run(context_review.read_context_review(HANDLE, MESSAGE, mode="TEST"))
    assert calls[1][1].endswith("/context-review?horizon=as_lived")
    assert view.foreshadowing is None

    _wire(monkeypatch, context_review, [(200, _review(horizon="hindsight"))], [])
    hindsight = asyncio.run(context_review.read_context_review(HANDLE, MESSAGE, mode="TEST", horizon="hindsight"))
    assert hindsight.foreshadowing and hindsight.foreshadowing[0].horizon == "hindsight"

    _wire(monkeypatch, context_review, [(200, _review(horizon="hindsight"))], [])
    with pytest.raises(ProfferError):
        asyncio.run(context_review.read_context_review(HANDLE, MESSAGE, mode="TEST", horizon="as_lived"))


def test_writes_are_checked_against_their_horizon(monkeypatch) -> None:
    receipt = {
        "ref": "r1",
        "receipt_ref": "context-review://r1",
        "content_digest": "d" * 64,
        "revision": 1,
        "recorded_at": NOW,
    }
    _wire(monkeypatch, context_review, [(201, {**receipt, "horizon": "as_lived"})], [])
    review = ContextReviewRequest(
        addressed_to=[{"label": "Recipient A"}], about_child="yes", relevant=True, change_reason="read the thread"
    )
    assert asyncio.run(context_review.write_context_review(HANDLE, MESSAGE, review, ACTOR, mode="TEST")).revision == 1

    _wire(monkeypatch, context_review, [(201, {**receipt, "horizon": "as_lived"})], [])
    flag = ForeshadowingRequest(foreshadowing=True, change_reason="significant in hindsight")
    with pytest.raises(ProfferError):
        asyncio.run(context_review.write_foreshadowing(HANDLE, MESSAGE, flag, ACTOR, mode="TEST"))
    with pytest.raises(ValueError):
        ContextReviewRequest.model_validate({"foreshadowing": True, "change_reason": "smuggled"})
    with pytest.raises(ValueError):
        ContextReviewRequest(about=[{"label": "Same"}, {"label": "same"}], change_reason="duplicate")


def _client() -> TestClient:
    app = FastAPI()

    @app.middleware("http")
    async def authenticated(request: Request, call_next):
        request.state.subject_uid, request.state.principal = "authentik-user-1", "owner"
        return await call_next(request)

    app.include_router(runtime.router)
    return TestClient(app)


def test_as_lived_http_answer_has_no_foreshadowing_key(monkeypatch) -> None:
    async def fake_read(handle, message_id, *, mode, horizon):
        return context_review.ContextReviewView.model_validate({**_review(horizon=horizon), "matter_mode": mode})

    monkeypatch.setattr(runtime, "read_context_review", fake_read)
    response = _client().get(f"/api/proffer/previews/{HANDLE}/messages/{MESSAGE}/context-review?mode=TEST")
    assert response.status_code == 200
    assert "foreshadowing" not in response.json()
    hindsight = _client().get(
        f"/api/proffer/previews/{HANDLE}/messages/{MESSAGE}/context-review?mode=TEST&horizon=hindsight"
    )
    assert "foreshadowing" in hindsight.json()
    bad = _client().post(
        f"/api/proffer/previews/{HANDLE}/messages/{MESSAGE}/context-review?mode=TEST",
        json={"foreshadowing": True, "change_reason": "smuggled"},
    )
    assert bad.status_code == 422


def test_sidecar_classification_and_flattening() -> None:
    assert source_sidecars.stem_of("IMG_0001.jpg") == "IMG_0001"
    assert source_sidecars.classify("IMG_0001.jpg", "IMG_0001.jpg.supplemental-metadata.json") == "json"
    assert source_sidecars.classify("IMG_0001.jpg", "IMG_0001.XMP") == "xmp"
    assert source_sidecars.classify("IMG_0001.jpg", "IMG_0001.AAE") == "apple_aae"
    assert source_sidecars.classify("IMG_0001.jpg", "IMG_0001.jpg.sidecar.md") == "owner_sidecar_md"
    assert source_sidecars.classify("IMG_0001.jpg", "IMG_0001.EXTRACTION.md") == "owner_extraction_md"
    assert source_sidecars.classify("IMG_0001.jpg", "IMG_0001.MOV") == "companion"
    assert source_sidecars.classify("IMG_0001.jpg", "IMG_00012.jpg") is None
    assert source_sidecars.classify("IMG_0001.jpg", "IMG_0001.jpg") is None
    fields, truncated = source_sidecars.flatten({"photoTakenTime": {"timestamp": "1"}, "people": [{"name": "A"}]})
    assert [(field.path, field.value) for field in fields] == [
        ("photoTakenTime.timestamp", "1"),
        ("people[0].name", "A"),
    ]
    assert not truncated


def test_find_sidecars_reads_beside_the_object_and_names_where_it_looked(monkeypatch) -> None:
    monkeypatch.setenv(
        "SOURCE_ROOTS_JSON", json.dumps([{"id": "vault", "label": "Vault", "url": "b2://proof-bucket/vault/"}])
    )
    listing = [
        {"Key": "vault/IMG_0001.jpg", "Size": 10},
        {"Key": "vault/IMG_0001.jpg.json", "Size": 60},
        {"Key": "vault/IMG_0001.MOV", "Size": 999},
        {"Key": "vault/IMG_00011.jpg", "Size": 5},
    ]
    monkeypatch.setattr(source_sidecars, "list_objects_beside", lambda scheme, bucket, prefix, max_keys: listing)
    monkeypatch.setattr(
        source_sidecars,
        "read_small_object",
        lambda scheme, bucket, key, max_bytes: b'{"photoTakenTime": {"timestamp": "1620468000"}}',
    )
    monkeypatch.setattr(source_sidecars, "configured", lambda: False)

    sidecars, lookup = source_sidecars.find_sidecars("b2://proof-bucket/vault/IMG_0001.jpg")

    assert [(item.name, item.kind) for item in sidecars] == [
        ("IMG_0001.jpg.json", "takeout_json"),
        ("IMG_0001.MOV", "companion"),
    ]
    assert lookup.beside_object == "ok" and lookup.catalog_folder == "not_configured"
    assert source_sidecars.find_sidecars("b2://elsewhere/IMG_0001.jpg")[1].beside_object == "not_applicable"


def test_conflicts_respect_the_timezone_tolerance() -> None:
    sidecar = Sidecar(
        name="a.json",
        key="a.json",
        kind="takeout_json",
        found_via="beside_object",
        fields=[
            {"path": "photoTakenTime.timestamp", "value": "1620468000"},
            {"path": "geoData.latitude", "value": 41.5},
            {"path": "geoData.longitude", "value": 0.0},
        ],
    )
    close = MetadataRow.model_validate(
        {
            "metadata_ref": "m",
            "metadata_class": "embedded",
            "extractor_id": "x",
            "generated_at": NOW,
            "receipt_ref": "r",
            "fields": {"EXIF:DateTimeOriginal": "2021:05:08 05:00:00", "GPSLatitude": "41.5"},
        }
    )
    assert source_metadata.sidecar_conflicts([sidecar], [close]) == []
    far = close.model_copy(
        update={"fields": {"EXIF:DateTimeOriginal": "2021:05:08 10:00:00-07:00", "GPSLatitude": 40.0}}
    )
    topics = sorted(conflict.topic for conflict in source_metadata.sidecar_conflicts([sidecar], [far]))
    assert topics == ["capture_time", "gps"]


def test_find_sidecars_asks_the_catalog_for_the_original_folder(monkeypatch) -> None:
    monkeypatch.setenv(
        "SOURCE_ROOTS_JSON", json.dumps([{"id": "vault", "label": "Vault", "url": "b2://proof-bucket/vault/"}])
    )
    monkeypatch.setattr(source_sidecars, "list_objects_beside", lambda scheme, bucket, prefix, max_keys: [])
    monkeypatch.setattr(source_sidecars, "configured", lambda: True)
    asked: list = []

    def fake_companions(vault_key, stem, *, limit):
        asked.append((vault_key, stem))
        return {
            "items": [
                {"rel": "phone/IMG_0001.xmp", "name": "IMG_0001.xmp", "size": 30, "vault_key": "x/IMG_0001.xmp"},
                {"rel": "phone/IMG_0001.AAE", "name": "IMG_0001.AAE", "size": 20, "vault_key": None},
                {
                    "rel": "phone/IMG_0001.jpg.json",
                    "name": "IMG_0001.jpg.json",
                    "size": 9,
                    "vault_key": "../escape.json",
                },
            ]
        }

    monkeypatch.setattr(source_sidecars, "catalog_companions", fake_companions)
    monkeypatch.setattr(source_sidecars, "read_small_object", lambda scheme, bucket, key, max_bytes: b"<x:xmpmeta/>")

    sidecars, lookup = source_sidecars.find_sidecars("b2://proof-bucket/vault/a/IMG_0001.jpg")

    assert asked == [("a/IMG_0001.jpg", "IMG_0001")], "the catalog is asked with the root-relative key"
    by_name = {item.name: item for item in sidecars}
    assert by_name["IMG_0001.xmp"].text == "<x:xmpmeta/>" and by_name["IMG_0001.xmp"].found_via == "catalog_folder"
    assert by_name["IMG_0001.AAE"].unread_reason and by_name["IMG_0001.AAE"].text is None
    assert by_name["IMG_0001.jpg.json"].unread_reason, "a catalog key outside the root is never read"
    assert lookup.catalog_folder == "ok"
