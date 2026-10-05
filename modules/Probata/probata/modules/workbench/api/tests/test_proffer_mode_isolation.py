"""Single approved case, default-Live policy, durable receipts and Dev write isolation.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
"""

from __future__ import annotations

import asyncio
from uuid import UUID

import httpx
import pytest
from app.config import settings
from app.runtime import case_identity, case_management
from app.service import case_scope, matter_mode, preview_mode_recovery, proffer, proffer_batch
from app.types.matter_mode import MatterMode
from app.types.proffer import ProfferStartRequest
from app.types.proffer_batch import ProfferBatchStartRequest
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient
from main import app as production_app
from main import development_write_guard
from pydantic import TypeAdapter, ValidationError
from starlette.middleware.base import BaseHTTPMiddleware

MATTER = "11111111-1111-4111-8111-111111111111"
COURT = "22222222-2222-4222-8222-222222222222"
OTHER = "33333333-3333-4333-8333-333333333333"
HANDLE = "preview_handle_abcdefghijklmnopqrstuvwxyz"
PERSON = "44444444-4444-4444-8444-444444444444"
ALIAS = "55555555-5555-4555-8555-555555555555"


@pytest.fixture(autouse=True)
def canonical_case(monkeypatch):
    monkeypatch.setattr(settings, "proffer_matter_id", MATTER)
    monkeypatch.setattr(settings, "proffer_court_case_id", COURT)
    matter_mode._clear_preview_modes_for_tests()
    yield
    matter_mode._clear_preview_modes_for_tests()


def _header(mode="LIVE"):
    return {"mode": mode, "matter": {"id": MATTER}, "court_case": {"id": COURT},
            "people": [{"id": PERSON}], "source_versions": [{"id": OTHER}],
            "history": [], "probata_counts": [], "probata_unknowns": [], "dismissed": []}


def _app():
    app = FastAPI()

    @app.middleware("http")
    async def actor(request: Request, call_next):
        request.state.subject_uid = "subject-1"
        request.state.principal = "operator"
        return await call_next(request)

    app.include_router(case_identity.router)
    app.include_router(case_management.router)
    return app


@pytest.mark.parametrize(("value", "expected"),
                         [(None, "LIVE"), ("LIVE", "LIVE"), ("DEV", "DEV"), ("REAL", "LIVE"), ("TEST", "DEV")])
def test_boundary_aliases_are_input_only(value, expected):
    adapter = TypeAdapter(MatterMode)
    mode = "LIVE" if value is None else adapter.validate_python(value)
    assert mode == expected
    assert adapter.dump_python(mode) in {"DEV", "LIVE"}
    assert matter_mode.configured_matter_id(mode) == UUID(MATTER)
    assert matter_mode.configured_court_case_id(mode) == UUID(COURT)


@pytest.mark.parametrize("value", ["", "live", "dev", "unknown", 7])
def test_unknown_modes_are_rejected(value):
    with pytest.raises(ValidationError):
        TypeAdapter(MatterMode).validate_python(value)


@pytest.mark.parametrize("field", ["proffer_matter_id", "proffer_court_case_id"])
@pytest.mark.parametrize("value", ["", "not-a-uuid", "00000000-0000-0000-0000-000000000000",
                                 "deadbeef-dead-beef-dead-beefdeadbeef", "cafebabe-cafe-babe-cafe-babecafebabe"])
def test_missing_or_sentinel_configuration_fails_closed(monkeypatch, field, value):
    monkeypatch.setattr(settings, field, value)
    monkeypatch.setattr(settings, "proffer_real_matter_id", "")
    monkeypatch.setattr(settings, "proffer_real_court_case_id", "")
    with pytest.raises(matter_mode.MatterModeError) as error:
        resolver = matter_mode.configured_matter_id if field.endswith("matter_id") else matter_mode.configured_court_case_id
        resolver("LIVE")
    assert error.value.status_code == 503


def test_only_authoritative_legacy_environment_is_a_rollout_fallback(monkeypatch):
    monkeypatch.setattr(settings, "proffer_matter_id", "")
    monkeypatch.setattr(settings, "proffer_court_case_id", "")
    monkeypatch.setattr(settings, "proffer_test_matter_id", OTHER)
    monkeypatch.setattr(settings, "proffer_test_court_case_id", OTHER)
    monkeypatch.setattr(settings, "proffer_real_matter_id", MATTER)
    monkeypatch.setattr(settings, "proffer_real_court_case_id", COURT)
    for mode in ("DEV", "LIVE"):
        assert matter_mode.require_scope(mode, UUID(MATTER), UUID(COURT)) == (UUID(MATTER), UUID(COURT))
    monkeypatch.setattr(settings, "proffer_matter_id", OTHER)
    assert matter_mode.configured_matter_id("LIVE") == UUID(OTHER)
    with pytest.raises(matter_mode.MatterModeError):
        matter_mode.require_matter("LIVE", UUID(MATTER))


def test_authoritative_header_rejects_a_wrong_non_sentinel_config(monkeypatch):
    monkeypatch.setattr(settings, "proffer_matter_id", OTHER)
    with pytest.raises(proffer.ProfferError) as error:
        asyncio.run(case_scope.verify_case_scope("LIVE", _header()))
    assert error.value.status_code == 502


def test_bad_config_cannot_select_an_unrelated_spine_case(monkeypatch):
    monkeypatch.setattr(settings, "proffer_matter_id", OTHER)
    calls = []

    async def engine(method, path, **kwargs):
        calls.append((method, path))
        return httpx.Response(200, json=_header())

    monkeypatch.setattr(proffer, "_request", engine)
    monkeypatch.setattr(case_management.service, "get_matter", lambda *_: pytest.fail("spine must not run"))
    response = TestClient(_app()).get("/api/matters")
    assert response.status_code == 502
    assert calls == [("GET", "/case-identity")]


def test_default_live_and_both_modes_preserve_case_people_and_source_ids(monkeypatch):
    async def engine(method, path, **kwargs):
        assert method == "GET"
        assert path == "/case-identity"
        return httpx.Response(200, json=_header(kwargs["params"]["mode"]))

    monkeypatch.setattr(proffer, "_request", engine)
    monkeypatch.setattr(case_identity.service, "_catalog_section", lambda _: {})
    client = TestClient(_app())
    responses = [client.get("/api/case-identity", params=params)
                 for params in ({}, {"mode": "LIVE"}, {"mode": "DEV"}, {"mode": "REAL"}, {"mode": "TEST"})]
    assert all(response.status_code == 200 for response in responses)
    views = [response.json() for response in responses]
    assert [view["mode"] for view in views] == ["LIVE", "LIVE", "DEV", "LIVE", "DEV"]
    for key in ("matter", "court_case", "people", "source_versions"):
        assert all(view[key] == views[0][key] for view in views)


@pytest.mark.parametrize("value", ["", "unknown", "dev"])
def test_unknown_route_policy_never_dispatches(monkeypatch, value):
    monkeypatch.setattr(proffer, "_request", lambda *_args, **_kwargs: pytest.fail("engine must not run"))
    response = TestClient(_app()).get("/api/case-identity", params={"mode": value})
    assert response.status_code == 422


CASE_WRITES = [
    "/identifiers", f"/identifiers/{ALIAS}", f"/identifiers/{ALIAS}/delete",
    "/header", "/people", f"/people/{PERSON}", "/triage", "/placeholders",
    "/contact-people", f"/people/{PERSON}/merge",
]


@pytest.mark.parametrize("path", CASE_WRITES)
@pytest.mark.parametrize("mode", ["DEV", "TEST"])
def test_every_case_mutation_denies_dev_before_upstream(monkeypatch, path, mode):
    monkeypatch.setattr(proffer, "_request", lambda *_args, **_kwargs: pytest.fail("engine must not run"))
    response = TestClient(_app()).post("/api/case-identity" + path, params={"mode": mode},
                                      json={}, headers={"Idempotency-Key": "key"})
    assert response.status_code == 409
    assert "isolated data workspace" in response.json()["detail"]


@pytest.mark.parametrize("path", CASE_WRITES)
@pytest.mark.parametrize("params", [{}, {"mode": "LIVE"}, {"mode": "REAL"}])
def test_every_case_mutation_forwards_selected_canonical_policy(monkeypatch, path, params):
    calls = []

    async def engine(method, upstream, **kwargs):
        calls.append((method, upstream, kwargs))
        if method == "GET":
            return httpx.Response(200, json=_header())
        return httpx.Response(201, json={"ref": "receipt", "kind": "identity", "recorded_at": "now"})

    monkeypatch.setattr(proffer, "_request", engine)
    response = TestClient(_app()).post("/api/case-identity" + path, params=params,
                                      json={}, headers={"Idempotency-Key": "key"})
    assert response.status_code == 201
    posts = [call for call in calls if call[0] == "POST"]
    assert len(posts) == 1
    assert posts[0][1] == "/case-identity" + path
    assert posts[0][2]["params"] == {"mode": "LIVE"}
    assert posts[0][2]["headers"]["Idempotency-Key"] == "key"


def test_production_guard_blocks_all_api_mutation_routes_before_any_dispatch(monkeypatch):
    app = FastAPI()
    app.add_middleware(BaseHTTPMiddleware, dispatch=development_write_guard)
    calls = []

    @app.api_route("/api/{tail:path}", methods=["POST", "PUT", "PATCH", "DELETE"])
    async def forbidden(tail: str):
        calls.append(tail)
        return {}

    client = TestClient(app)
    tested = 0
    for path, methods in production_app.openapi()["paths"].items():
        for method in methods.keys() & {"post", "put", "patch", "delete"}:
            if not path.startswith("/api/"):
                continue
            response = client.request(method, path, params={"mode": "DEV"}, json={})
            assert response.status_code == 409, (method, path, response.text)
            tested += 1
    assert tested > 40
    assert calls == []


@pytest.mark.parametrize("mode", [None, "TEST", "REAL", "unknown"])
def test_historical_identity_never_implies_durable_mode(mode):
    assert preview_mode_recovery.rebind(HANDLE, UUID(MATTER), mode) is None
    with pytest.raises(matter_mode.MatterModeError):
        matter_mode.require_preview_mode(HANDLE, "LIVE")


def test_explicit_durable_mode_recovers_after_restart_but_wrong_scope_does_not():
    assert preview_mode_recovery.rebind(HANDLE, UUID(OTHER), "LIVE") is None
    assert preview_mode_recovery.rebind(HANDLE, UUID(MATTER), "DEV") == "DEV"
    matter_mode.require_preview_mode(HANDLE, "DEV")
    with pytest.raises(matter_mode.MatterModeError):
        matter_mode.require_preview_mode(HANDLE, "LIVE")


def test_operation_binding_recovers_mode_without_identity_inference(monkeypatch):
    async def engine(method, path, **kwargs):
        return httpx.Response(200, json={"preview_handle": HANDLE, "matter_id": MATTER, "operating_mode": "LIVE"})
    monkeypatch.setattr(proffer, "_request", engine)
    asyncio.run(proffer._require_mode(HANDLE, "LIVE"))
    matter_mode.require_preview_mode(HANDLE, "LIVE")


def _start():
    return ProfferStartRequest(request_id="one", matter_id=MATTER, court_case_id=COURT,
                               source_ref="r2://casebible-raw/source.xml", declared_format="xml",
                               parser_options_ref="parser-options://default")


def test_single_start_defaults_live_and_sends_explicit_durable_flag(monkeypatch):
    calls = []
    async def engine(method, path, **kwargs):
        calls.append((method, path, kwargs))
        if method == "GET":
            return httpx.Response(200, json={"preview_handle": HANDLE, "matter_id": MATTER, "operating_mode": "LIVE"})
        return httpx.Response(201, json={"preview_handle": HANDLE})
    monkeypatch.setattr(proffer, "_request", engine)
    result = asyncio.run(proffer.start(_start(), mode="LIVE"))
    assert result.matter_mode == "LIVE"
    assert calls[0][2]["json"]["operating_mode"] == "LIVE"
    assert "matter_mode" not in calls[0][2]["json"]
    assert calls[0][2]["json"]["matter_id"] == MATTER


def test_single_and_batch_dev_start_deny_before_dispatch(monkeypatch):
    monkeypatch.setattr(proffer, "_request", lambda *_args, **_kwargs: pytest.fail("dispatch forbidden"))
    monkeypatch.setattr(proffer_batch, "_request", lambda *_args, **_kwargs: pytest.fail("dispatch forbidden"))
    batch = ProfferBatchStartRequest(batch_id="batch_" + "a"*32, matter_id=MATTER, court_case_id=COURT,
                                    folder_ref="r2://casebible-raw/folder", declared_format="xml",
                                    parser_options_ref="parser-options://default", matter_mode="DEV")
    for coroutine in (proffer.start(_start(), mode="DEV"), proffer_batch.start_batch(batch, mode="DEV")):
        with pytest.raises(proffer.ProfferError) as error:
            asyncio.run(coroutine)
        assert error.value.status_code == 409


@pytest.mark.parametrize("receipt_mode", [None, "", "REAL", "DEV", "unknown"])
def test_fresh_start_never_binds_requested_live_without_durable_proof(monkeypatch, receipt_mode):
    async def engine(method, path, **kwargs):
        if method == "POST":
            return httpx.Response(201, json={"preview_handle": HANDLE})
        return httpx.Response(200, json={"preview_handle": HANDLE, "matter_id": MATTER,
                                        "operating_mode": receipt_mode})
    monkeypatch.setattr(proffer, "_request", engine)
    with pytest.raises(proffer.ProfferError) as error:
        asyncio.run(proffer.start(_start(), mode="LIVE"))
    assert error.value.status_code == 502
    with pytest.raises(matter_mode.MatterModeError):
        matter_mode.require_preview_mode(HANDLE, "LIVE")


@pytest.mark.parametrize("field", ["matter_id", "court_case_id"])
def test_wrong_single_start_case_scope_denies_before_dispatch(monkeypatch, field):
    monkeypatch.setattr(proffer, "_request", lambda *_args, **_kwargs: pytest.fail("dispatch forbidden"))
    request = _start().model_copy(update={field: UUID(OTHER)})
    with pytest.raises(proffer.ProfferError) as error:
        asyncio.run(proffer.start(request, mode="LIVE"))
    assert error.value.status_code == 409


def _case_route_requests(matter_id: str) -> list[tuple[str, str, dict | None]]:
    evidence_item_id = "88888888-8888-4888-8888-888888888888"
    source = {
        "lane": "evidence",
        "partition_key": "primary",
        "artifact_id": "33333333-3333-4333-8333-333333333333",
        "sha256": "a" * 64,
        "retrieval_ref": "hit-1",
    }
    return [
        ("POST", f"/api/matters/{matter_id}/court-cases", {"caption": "Configured case"}),
        ("POST", f"/api/matters/{matter_id}/knowledge/resolve", source),
        (
            "POST",
            f"/api/matters/{matter_id}/evidence-items",
            {
                "court_case_id": COURT,
                "source": {
                    **source,
                    "normalized_record_id": "44444444-4444-4444-8444-444444444444",
                },
                "title": "Draft evidence",
            },
        ),
        ("GET", f"/api/matters/{matter_id}/evidence-items", None),
        ("GET", f"/api/matters/{matter_id}/evidence-items/{evidence_item_id}", None),
        (
            "GET",
            f"/api/matters/{matter_id}/evidence-items/{evidence_item_id}/source-content",
            None,
        ),
        (
            "GET",
            f"/api/matters/{matter_id}/evidence-items/{evidence_item_id}/conversation-context",
            None,
        ),
        (
            "GET",
            f"/api/matters/{matter_id}/evidence-items/{evidence_item_id}/court-readiness",
            None,
        ),
        (
            "POST",
            f"/api/matters/{matter_id}/evidence-items/{evidence_item_id}/reviews",
            {"decision": "approved", "rationale": "Exact record reviewed."},
        ),
        ("GET", f"/api/matters/{matter_id}/evidence-items/{evidence_item_id}/reviews", None),
    ]


def _deny_all_case_management_upstream(monkeypatch) -> None:
    def forbidden(*args, **kwargs):
        raise AssertionError("case-management upstream must not run")

    for name in (
        "create_matter",
        "create_court_case",
        "resolve_knowledge_source",
        "create_evidence_item",
        "list_evidence_items",
        "get_evidence_detail",
        "get_original_source_content",
        "get_conversation_context",
        "get_court_readiness",
        "review_evidence_item",
        "list_evidence_reviews",
    ):
        monkeypatch.setattr(case_management.service, name, forbidden)



async def _case_header_request(method, path, **kwargs):
    assert (method, path) == ("GET", "/case-identity")
    return httpx.Response(200, json=_header(kwargs.get("params", {}).get("mode", "LIVE")))


@pytest.mark.parametrize(("method", "path", "payload"), _case_route_requests(OTHER))
def test_arbitrary_matter_routes_deny_before_spine(monkeypatch, method, path, payload):
    _deny_all_case_management_upstream(monkeypatch)
    monkeypatch.setattr(proffer, "_request", _case_header_request)
    monkeypatch.setattr(case_management.service, "get_matter", lambda *_: pytest.fail("spine must not run"))
    response = TestClient(_app()).request(method, path, params={"mode": "LIVE"}, json=payload)
    assert response.status_code == 409


@pytest.mark.parametrize(("method", "path", "payload"),
                         [("GET", "/api/matters", None), ("POST", "/api/matters", {"title": "Unavailable"})]
                         + _case_route_requests(MATTER))
def test_unconfigured_case_denies_every_spine_route_before_dispatch(monkeypatch, method, path, payload):
    _deny_all_case_management_upstream(monkeypatch)
    monkeypatch.setattr(settings, "proffer_matter_id", "")
    monkeypatch.setattr(settings, "proffer_real_matter_id", "")
    monkeypatch.setattr(proffer, "_request", lambda *_args, **_kwargs: pytest.fail("engine must not run"))
    response = TestClient(_app()).request(method, path, params={"mode": "LIVE"}, json=payload)
    assert response.status_code == 503


def test_evidence_wrong_court_scope_and_matter_creation_deny_before_spine(monkeypatch):
    _deny_all_case_management_upstream(monkeypatch)
    monkeypatch.setattr(proffer, "_request", _case_header_request)
    method, path, payload = _case_route_requests(MATTER)[2]
    payload["court_case_id"] = OTHER
    client = TestClient(_app())
    response = client.request(method, path, params={"mode": "LIVE"}, json=payload)
    assert response.status_code == 409
    response = client.post("/api/matters?mode=LIVE", json={"title": "No alternate case"})
    assert response.status_code == 409
    assert "Matter creation is disabled" in response.json()["detail"]


def test_unconfigured_source_browser_has_zero_provider_io(monkeypatch):
    from app.service import proffer_sources
    monkeypatch.setattr(settings, "proffer_matter_id", "")
    monkeypatch.setattr(settings, "proffer_real_matter_id", "")
    monkeypatch.setattr(proffer_sources, "list_source_objects", lambda *_args, **_kwargs: pytest.fail("provider must not run"))
    with pytest.raises(proffer.ProfferError) as error:
        proffer.browse_sources(mode="LIVE")
    assert error.value.status_code == 503


def test_openapi_policy_defaults_live_and_advertises_only_canonical_values():
    schema = _app().openapi()
    for path, operations in schema["paths"].items():
        if path == "/api/case-identity" or path.startswith("/api/matters") or (
            path.startswith("/api/case-identity") and "post" in operations
        ):
            for operation in operations.values():
                parameters = operation.get("parameters", [])
                mode = next((p for p in parameters if p["name"] == "mode"), None)
                if mode is not None:
                    assert mode["required"] is False
                    assert mode["schema"]["default"] == "LIVE"
                    assert mode["schema"]["enum"] == ["DEV", "LIVE"]
