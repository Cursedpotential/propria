"""Filing-readiness is computed and never marks filed.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

import httpx
from fastapi.testclient import TestClient

from legal_workspace.api import main as main_mod
from legal_workspace.api.main import app
from legal_workspace.domain.filing import CheckState, FilingOverrideCreate
from legal_workspace.services import workspace as workspace_mod
from legal_workspace.services.workspace import Workspace
from test_first_slice import _approved_package
from conftest import seed_synthetic_approved_package


def test_blank_matter_is_not_ready(tmp_path) -> None:
    report = Workspace(tmp_path).filing_readiness()
    assert report.ready is False
    assert report.filed is False
    assert report.blocking_count >= 1
    states = {item.check_id: item.state for item in report.checks}
    assert states["docket"] is CheckState.FAIL
    assert states["package"] is CheckState.FAIL
    assert states["agno-verify"] is CheckState.FAIL
    assert states["signature"] is CheckState.UNKNOWN


def test_owner_override_persists(tmp_path) -> None:
    first = Workspace(tmp_path)
    first.add_filing_override(
        FilingOverrideCreate(
            check_id="signature",
            state=CheckState.PASS,
            note="Owner verified notary block on the current form.",
        )
    )
    try:
        first.add_filing_override(
            FilingOverrideCreate(
                check_id="signature",
                state=CheckState.PASS,
                note="agent",
                reviewer="agent",
            )
        )
        raise AssertionError("agent override must fail")
    except ValueError as exc:
        assert "owner" in str(exc)
    reloaded = Workspace(tmp_path).filing_readiness()
    signature = next(item for item in reloaded.checks if item.check_id == "signature")
    assert signature.state is CheckState.PASS
    assert reloaded.ready is False


def test_http_filing_readiness(tmp_path) -> None:
    store = workspace_mod.get_workspace(tmp_path)
    main_mod.WORKSPACE = store
    client = TestClient(app)
    response = client.get("/v1/filing-readiness")
    assert response.status_code == 200
    body = response.json()
    assert body["ready"] is False
    assert body["filed"] is False
    assert any(row["path"] == "/filing-checklist" and row["label"] == "Filing readiness checklist" for row in client.get("/v1/matter").json()["next_surfaces"])
    blocked = client.post(
        "/v1/filing-overrides",
        json={"check_id": "fee", "state": "pass", "note": "no", "reviewer": "agent"},
    )
    assert blocked.status_code == 409


def test_filing_calls_agno_verify_and_fails_closed_when_unreachable(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    package, _citation = _approved_package(workspace.load().matter.matter_id)
    seed_synthetic_approved_package(workspace, package)

    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(503)

    report = workspace.filing_readiness(
        agno_http=httpx.Client(transport=httpx.MockTransport(handler))
    )
    check = next(item for item in report.checks if item.check_id == "agno-verify")
    assert check.state is CheckState.FAIL
    assert "unreachable" in check.reason
    assert Workspace(tmp_path).load().last_agno_verify


def test_filing_agno_verify_pass_when_agno_returns_intact(tmp_path) -> None:
    workspace = Workspace(tmp_path)
    package, _citation = _approved_package(workspace.load().matter.matter_id)
    hex_digest = "ab" * 32
    package.items[0].content_hash = f"sha256:{hex_digest}"
    seed_synthetic_approved_package(workspace, package)

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "POST"
        assert request.url.path == f"/v1/verify/{hex_digest}"
        return httpx.Response(200, json={"sha256_match": True, "verdict": "hash-only-ok"})

    report = workspace.filing_readiness(agno_http=httpx.Client(transport=httpx.MockTransport(handler)))
    check = next(item for item in report.checks if item.check_id == "agno-verify")
    assert check.state is CheckState.PASS
    assert "verified by Agno" in check.reason
