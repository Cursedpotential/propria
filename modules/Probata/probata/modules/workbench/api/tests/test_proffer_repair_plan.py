"""Repair workflow builder passthrough: /api/proffer/repair/{tools,propose,validate,run,runs/{id}}.

Byline: Claude Code · Opus 5.5 · 2026-09-26.

The engine is replaced by a test double at the HTTP client boundary only, so the real
`proffer._request` path runs: the same client, the mounted service token, and the upstream
status and `detail` translation every other Proffer route uses.
"""

from __future__ import annotations

from uuid import UUID

import httpx
import pytest
from app.runtime.proffer_repair_plan import router
from app.service import matter_mode, proffer
from app.service.matter_mode import _clear_preview_modes_for_tests, bind_preview_mode
from app.service.proffer_repair_plan import _clear_run_modes_for_tests
from fastapi import FastAPI
from fastapi.testclient import TestClient

STARTER = "https://starter.internal"
TOKEN = "t" * 40
HANDLE = "live_policy_review_run_handle_abcdefghij"
DEV_HANDLE = "dev_policy_review_run_handle_abcdefghij"
SOURCE = "b2://salem-data/consignatio/vault/v1/sms/sms-20240101.xml"
WORKFLOW_ID = "repair-plan-rp-abcdefgh-0123456789ab"
CASE_MATTER = UUID("11111111-1111-4111-8111-111111111111")

FIND_SCHEMA = {
    "type": "object",
    "additionalProperties": False,
    "properties": {"max_candidates": {"type": "integer", "minimum": 1, "maximum": 20, "default": 5}},
}


class FakeEngine:
    """Answers (method, path) with a status and JSON body, and records every call."""

    def __init__(self) -> None:
        self.routes: dict[tuple[str, str], tuple[int, object]] = {}
        self.calls: list[dict] = []

    def reply(self, method: str, path: str, status: int, payload: object) -> None:
        self.routes[(method, path)] = (status, payload)

    def client_class(self):
        engine = self

        class Client:
            def __init__(self, *args, **kwargs):
                pass

            async def __aenter__(self):
                return self

            async def __aexit__(self, *args):
                return None

            async def request(self, method, url, **kwargs):
                path = url.removeprefix(STARTER)
                engine.calls.append({"method": method, "path": path, **kwargs})
                status, payload = engine.routes.get((method, path), (404, {"detail": "no such engine route"}))
                return httpx.Response(status, json=payload)

        return Client


@pytest.fixture
def engine(monkeypatch, tmp_path):
    from app.runtime import operating_mode
    async def verified_scope(_mode):
        return None
    monkeypatch.setattr(operating_mode, "verify_case_scope", verified_scope)
    secret = tmp_path / "proffer-service-token"
    secret.write_text(TOKEN, encoding="utf-8")
    fake = FakeEngine()
    monkeypatch.setattr(proffer.settings, "proffer_starter_url", STARTER)
    monkeypatch.setattr(proffer.settings, "proffer_service_token_file", str(secret))
    monkeypatch.setattr(proffer.httpx, "AsyncClient", fake.client_class())
    configured = lambda mode: CASE_MATTER
    monkeypatch.setattr(proffer, "configured_matter_id", configured)
    monkeypatch.setattr(matter_mode, "configured_matter_id", configured)
    _clear_preview_modes_for_tests()
    _clear_run_modes_for_tests()
    bind_preview_mode(HANDLE, "LIVE")
    bind_preview_mode(DEV_HANDLE, "DEV")
    yield fake
    _clear_preview_modes_for_tests()
    _clear_run_modes_for_tests()


@pytest.fixture
def client():
    app = FastAPI()
    app.include_router(router)
    return TestClient(app)


def _plan(**overrides) -> dict:
    plan = {
        "plan_id": "rp-abcdefgh",
        "source_ref": SOURCE,
        "preview_handle": HANDLE,
        "matter_mode": "LIVE",
        "steps": [
            {"step_id": "s1", "activity": "repair.find_other_version", "params": {"max_candidates": 3}},
            {"step_id": "s2", "activity": "repair.salvage_truncated_xml", "params": None},
        ],
    }
    plan.update(overrides)
    return plan


def _status(**overrides) -> dict:
    status = {
        "workflow_id": WORKFLOW_ID,
        "plan_id": "rp-abcdefgh",
        "preview_handle": HANDLE,
        "matter_mode": "LIVE",
        "status": "running",
        "steps": [
            {
                "step_id": "s1",
                "activity": "repair.find_other_version",
                "status": "succeeded",
                "receipt_ref": "r-1",
                "output_ref": SOURCE + ".copy",
                "summary": {"candidates": 1},
            },
            {"step_id": "s2", "activity": "repair.salvage_truncated_xml", "status": "pending", "receipt_ref": ""},
        ],
    }
    status.update(overrides)
    return status


# --- tools -------------------------------------------------------------------


def test_tools_pass_through_with_null_lists_emptied_and_the_mode_echoed(client, engine):
    engine.reply(
        "GET",
        "/reference-import/repair/tools",
        200,
        {
            "tools": [
                {
                    "id": "repair.find_other_version",
                    "description": "Find another copy",
                    "input_types": ["any"],
                    "output_types": ["same-as-input"],
                    "params_schema": FIND_SCHEMA,
                    "writes": "none",
                    "needs_n8n": False,
                },
                {
                    "id": "repair.lenient_decode",
                    "description": "Lenient decode",
                    "input_types": None,
                    "output_types": None,
                    "params_schema": None,
                    "writes": "derived",
                    "needs_n8n": False,
                },
            ]
        },
    )

    response = client.get("/api/proffer/repair/tools?mode=LIVE")

    assert response.status_code == 200
    body = response.json()
    assert body["matter_mode"] == "LIVE"
    assert body["tools"][0]["params_schema"] == FIND_SCHEMA
    assert body["tools"][1] == {
        "id": "repair.lenient_decode",
        "description": "Lenient decode",
        "input_types": [],
        "output_types": [],
        "params_schema": {},
        "writes": "derived",
        "needs_n8n": False,
    }
    # Same client, same mounted service token as every other Proffer route.
    assert engine.calls[0]["headers"]["Authorization"] == f"Bearer {TOKEN}"


def test_a_null_tool_list_is_an_empty_list(client, engine):
    engine.reply("GET", "/reference-import/repair/tools", 200, {"tools": None})
    assert client.get("/api/proffer/repair/tools?mode=DEV").json() == {"tools": [], "matter_mode": "DEV"}


def test_tools_keep_the_engines_status_and_detail(client, engine):
    engine.reply(
        "GET", "/reference-import/repair/tools", 503, {"detail": "repair plans are not configured on this service"}
    )
    response = client.get("/api/proffer/repair/tools?mode=LIVE")
    assert response.status_code == 503
    assert response.json() == {"detail": "repair plans are not configured on this service"}


def test_every_route_defaults_to_live_policy(client, engine):
    engine.reply("GET", "/reference-import/repair/tools", 200, {"tools": []})
    engine.reply("POST", "/reference-import/repair/validate", 200, {"ok": True, "checks": []})
    engine.reply("GET", f"/reference-import/repair/runs/{WORKFLOW_ID}", 200, _status())
    assert client.get("/api/proffer/repair/tools").json()["matter_mode"] == "LIVE"
    assert client.post("/api/proffer/repair/validate", json=_plan()).json()["matter_mode"] == "LIVE"
    assert client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}").json()["matter_mode"] == "LIVE"


# --- propose -----------------------------------------------------------------


def test_propose_forwards_the_run_and_empties_null_lists(client, engine):
    engine.reply(
        "POST",
        "/reference-import/repair/propose",
        200,
        {
            "signature": "sms_backup_xml:truncated",
            "proposals": [
                {
                    "signature": "sms_backup_xml:truncated",
                    "rationale": "Find another copy.",
                    "by": "rule",
                    "agent_available": False,
                    "steps": [{"step_id": "s1", "activity": "repair.find_other_version", "params": None}],
                },
                {
                    "signature": "sms_backup_xml:truncated",
                    "rationale": "Wait: re-run after parse.",
                    "by": "rule",
                    "agent_available": False,
                    "steps": None,
                },
            ],
            "agent_available": False,
        },
    )

    response = client.post(
        "/api/proffer/repair/propose?mode=LIVE", json={"source_ref": SOURCE, "preview_handle": HANDLE}
    )

    assert response.status_code == 200
    body = response.json()
    assert body["signature"] == "sms_backup_xml:truncated" and body["matter_mode"] == "LIVE"
    assert body["proposals"][0]["steps"] == [{"step_id": "s1", "activity": "repair.find_other_version", "params": {}}]
    assert body["proposals"][1]["steps"] == []
    assert engine.calls[0]["json"] == {"source_ref": SOURCE, "preview_handle": HANDLE}


def test_an_uncovered_signature_proposes_nothing(client, engine):
    engine.reply(
        "POST",
        "/reference-import/repair/propose",
        200,
        {"signature": "pdf:damaged", "proposals": None, "agent_available": False},
    )
    body = client.post(
        "/api/proffer/repair/propose?mode=LIVE", json={"source_ref": SOURCE, "preview_handle": HANDLE}
    ).json()
    assert body["proposals"] == [] and body["signature"] == "pdf:damaged"


def test_propose_refuses_a_run_of_the_other_mode_before_the_engine(client, engine):
    response = client.post(
        "/api/proffer/repair/propose?mode=LIVE", json={"source_ref": SOURCE, "preview_handle": DEV_HANDLE}
    )
    assert response.status_code == 409
    assert "different matter mode" in response.json()["detail"]
    assert engine.calls == []


def test_propose_rebinds_a_run_from_its_durable_matter_after_a_restart(client, engine):
    _clear_preview_modes_for_tests()
    engine.reply("GET", f"/reference-import/operations/{HANDLE}", 200, {"preview_handle": HANDLE, "matter_id": str(CASE_MATTER), "operating_mode": "LIVE"})
    engine.reply("POST", "/reference-import/repair/propose", 200, {"signature": "xml:clean", "proposals": []})

    assert (
        client.post(
            "/api/proffer/repair/propose?mode=LIVE", json={"source_ref": SOURCE, "preview_handle": HANDLE}
        ).status_code
        == 200
    )
    _clear_preview_modes_for_tests()
    wrong = client.post("/api/proffer/repair/propose?mode=DEV", json={"source_ref": SOURCE, "preview_handle": HANDLE})
    assert wrong.status_code == 409


def test_propose_keeps_the_engines_422(client, engine):
    detail = "no Review run exists for this source; start it in Review first"
    engine.reply("POST", "/reference-import/repair/propose", 422, {"detail": detail})
    response = client.post(
        "/api/proffer/repair/propose?mode=LIVE", json={"source_ref": SOURCE, "preview_handle": HANDLE}
    )
    assert response.status_code == 422
    assert response.json() == {"detail": detail}


# --- validate ----------------------------------------------------------------


def test_validate_passes_the_plan_through_and_returns_every_check(client, engine):
    checks = [
        {"rule": "anchor_resolves", "status": "pass", "reason": "Anchored to Review run."},
        {"rule": "type_chain", "status": "fail", "reason": "Step s2 reads sms_backup_xml or xml but receives pdf."},
    ]
    engine.reply("POST", "/reference-import/repair/validate", 200, {"ok": False, "checks": checks})

    response = client.post("/api/proffer/repair/validate?mode=LIVE", json=_plan())

    assert response.status_code == 200
    assert response.json() == {"ok": False, "checks": checks, "matter_mode": "LIVE"}
    forwarded = engine.calls[0]["json"]
    assert forwarded["matter_mode"] == "LIVE" and forwarded["preview_handle"] == HANDLE
    assert forwarded["steps"][1] == {"step_id": "s2", "activity": "repair.salvage_truncated_xml", "params": {}}


def test_validate_empties_a_null_checklist(client, engine):
    engine.reply("POST", "/reference-import/repair/validate", 200, {"ok": True, "checks": None})
    assert client.post("/api/proffer/repair/validate?mode=LIVE", json=_plan()).json()["checks"] == []


def test_a_plan_whose_mode_contradicts_the_query_never_reaches_the_engine(client, engine):
    response = client.post("/api/proffer/repair/validate?mode=DEV", json=_plan())
    assert response.status_code == 409
    assert engine.calls == []


def test_a_plan_anchored_to_the_other_modes_run_is_refused(client, engine):
    response = client.post("/api/proffer/repair/validate?mode=LIVE", json=_plan(preview_handle=DEV_HANDLE))
    assert response.status_code == 409
    assert engine.calls == []


def test_a_plan_without_a_provable_run_is_refused(client, engine):
    assert client.post("/api/proffer/repair/validate?mode=LIVE", json=_plan(preview_handle=None)).status_code == 422
    assert engine.calls == []


def test_validate_keeps_the_engines_400_for_a_malformed_plan(client, engine):
    engine.reply(
        "POST", "/reference-import/repair/validate", 400, {"detail": "plan_id must be 8-96 URL-safe characters"}
    )
    response = client.post("/api/proffer/repair/validate?mode=LIVE", json=_plan(plan_id="short"))
    assert response.status_code == 400
    assert response.json() == {"detail": "plan_id must be 8-96 URL-safe characters"}


def test_an_upstream_answer_for_the_other_mode_is_a_bad_gateway(client, engine):
    engine.reply("POST", "/reference-import/repair/validate", 200, {"ok": True, "checks": [], "matter_mode": "DEV"})
    assert client.post("/api/proffer/repair/validate?mode=LIVE", json=_plan()).status_code == 502


# --- run ---------------------------------------------------------------------


def test_run_starts_the_plan_and_records_its_mode(client, engine):
    engine.reply("POST", "/reference-import/repair/run", 201, {"workflow_id": WORKFLOW_ID, "run_id": "run-1"})

    response = client.post("/api/proffer/repair/run?mode=LIVE", json=_plan())

    assert response.status_code == 201
    assert response.json() == {"workflow_id": WORKFLOW_ID, "run_id": "run-1", "matter_mode": "LIVE"}
    # An engine status without a mode echo is proven by the binding /run recorded ...
    engine.reply(
        "GET",
        f"/reference-import/repair/runs/{WORKFLOW_ID}",
        200,
        {k: v for k, v in _status().items() if k != "matter_mode"},
    )
    assert client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}?mode=LIVE").json()["matter_mode"] == "LIVE"
    # ... and still refuses the other mode.
    assert client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}?mode=DEV").status_code == 409


def test_a_refused_run_keeps_its_detail_and_checks(client, engine):
    checks = [
        {"rule": "anchor_resolves", "status": "pass", "reason": "Anchored."},
        {"rule": "bounded", "status": "fail", "reason": "13 steps; at most 12."},
    ]
    detail = "plan failed validation — bounded: 13 steps; at most 12."
    engine.reply("POST", "/reference-import/repair/run", 422, {"detail": detail, "ok": False, "checks": checks})

    response = client.post("/api/proffer/repair/run?mode=LIVE", json=_plan())

    assert response.status_code == 422
    assert response.json() == {"detail": detail, "ok": False, "checks": checks}


def test_a_refused_run_with_no_checklist_keeps_its_detail(client, engine):
    engine.reply("POST", "/reference-import/repair/run", 422, {"detail": "plan failed validation", "checks": None})
    response = client.post("/api/proffer/repair/run?mode=LIVE", json=_plan())
    assert response.status_code == 422
    assert response.json() == {"detail": "plan failed validation", "ok": False, "checks": []}


def test_a_run_the_engine_cannot_start_keeps_its_status(client, engine):
    engine.reply(
        "POST", "/reference-import/repair/run", 503, {"detail": "the repair run could not start: temporal unavailable"}
    )
    response = client.post("/api/proffer/repair/run?mode=LIVE", json=_plan())
    assert response.status_code == 503
    assert response.json()["detail"] == "the repair run could not start: temporal unavailable"


def test_a_run_of_the_other_mode_is_refused_before_the_engine(client, engine):
    assert client.post("/api/proffer/repair/run?mode=DEV", json=_plan()).status_code == 409
    assert engine.calls == []


# --- runs/{workflow_id} ------------------------------------------------------


def test_run_status_is_proven_from_the_engine_after_a_restart(client, engine):
    engine.reply("GET", f"/reference-import/repair/runs/{WORKFLOW_ID}", 200, _status(checks=None))

    response = client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}?mode=LIVE")

    assert response.status_code == 200
    body = response.json()
    assert body["matter_mode"] == "LIVE" and body["status"] == "running"
    assert body["checks"] == []
    assert [step["status"] for step in body["steps"]] == ["succeeded", "pending"]
    assert body["steps"][0]["summary"] == {"candidates": 1}
    assert body["steps"][1]["receipt_ref"] == ""
    wrong = client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}?mode=DEV")
    assert wrong.status_code == 409
    assert "different matter mode" in wrong.json()["detail"]


def test_run_status_carries_the_reentry(client, engine):
    engine.reply(
        "GET",
        f"/reference-import/repair/runs/{WORKFLOW_ID}",
        200,
        _status(
            status="completed",
            steps=None,
            reentry_batch_id="repair-" + "a" * 40,
            reentry_receipt_ref="r-9",
        ),
    )
    body = client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}?mode=LIVE").json()
    assert body["steps"] == []
    assert body["reentry_batch_id"] == "repair-" + "a" * 40
    assert body["reentry_receipt_ref"] == "r-9"
    assert body["reentry_preview_handle"] is None


def test_an_unbound_run_the_engine_cannot_prove_fails_closed(client, engine):
    engine.reply("GET", f"/reference-import/repair/runs/{WORKFLOW_ID}", 200, _status(matter_mode=""))
    response = client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}?mode=LIVE")
    assert response.status_code == 409
    assert "no provable TEST/REAL binding" in response.json()["detail"]


def test_a_binding_the_engine_contradicts_is_a_bad_gateway(client, engine):
    engine.reply("POST", "/reference-import/repair/run", 201, {"workflow_id": WORKFLOW_ID, "run_id": "run-1"})
    assert client.post("/api/proffer/repair/run?mode=LIVE", json=_plan()).status_code == 201
    engine.reply("GET", f"/reference-import/repair/runs/{WORKFLOW_ID}", 200, _status(matter_mode="DEV"))
    assert client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}?mode=LIVE").status_code == 502


def test_a_status_for_another_workflow_is_a_bad_gateway(client, engine):
    engine.reply(
        "GET",
        f"/reference-import/repair/runs/{WORKFLOW_ID}",
        200,
        _status(workflow_id="repair-plan-other-000000000000"),
    )
    assert client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}?mode=LIVE").status_code == 502


def test_an_unknown_run_keeps_the_engines_404(client, engine):
    engine.reply("GET", f"/reference-import/repair/runs/{WORKFLOW_ID}", 404, {"detail": "repair run not found"})
    response = client.get(f"/api/proffer/repair/runs/{WORKFLOW_ID}?mode=LIVE")
    assert response.status_code == 404
    assert response.json() == {"detail": "repair run not found"}


def test_a_malformed_workflow_id_never_reaches_the_engine(client, engine):
    assert client.get("/api/proffer/repair/runs/short?mode=LIVE").status_code == 422
    assert client.get("/api/proffer/repair/runs/has%2Fslash-0123456789?mode=LIVE").status_code in {404, 422}
    assert engine.calls == []


def test_the_workbench_app_mounts_every_repair_route():
    from main import app

    paths = set(app.openapi()["paths"])  # FastAPI 0.141 nests included routers; the schema lists them all
    assert {
        "/api/proffer/repair/tools",
        "/api/proffer/repair/propose",
        "/api/proffer/repair/validate",
        "/api/proffer/repair/run",
        "/api/proffer/repair/runs/{workflow_id}",
    } <= paths
