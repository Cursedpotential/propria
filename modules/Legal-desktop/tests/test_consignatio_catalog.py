"""Consignatio catalog desk: fail-closed behavior and binding safety.

Byline: Claude Code · Fable 5.1 · 2026-09-21
No database stand-in: query correctness is verified against the live catalog
(see docs receipt). These tests cover what must hold with no catalog at all.
"""

import json

import pytest
from legal_workspace.services import consignatio_catalog as catalog


@pytest.fixture(autouse=True)
def no_catalog(monkeypatch):
    monkeypatch.delenv("CONSIGNATIO_CATALOG_URL", raising=False)
    monkeypatch.delenv("CONSIGNATIO_CATALOG_BINDINGS", raising=False)


def test_status_reports_not_configured(client_with_auth):
    body = client_with_auth.get("/v1/evidence-catalog/status").json()
    assert body["configured"] is False
    assert body["promotion_binding"] == "unbound"


def test_objects_fail_closed_without_catalog(client_with_auth):
    assert client_with_auth.get("/v1/evidence-catalog/objects").status_code == 503


def test_promoted_view_is_empty_while_promotion_relation_is_unbound(client_with_auth):
    for state in ("promoted", "pending"):
        response = client_with_auth.get(f"/v1/evidence-catalog/promotions?state={state}")
        assert response.status_code == 200
        assert response.json() == []


def test_folder_name_never_makes_evidence():
    bindings = catalog.load_bindings()
    assert catalog._lifecycle(bindings, None) == "context"


def test_lifecycle_follows_the_promotion_record_only():
    bindings = catalog.load_bindings()
    bindings["promotions"].update(
        state="lifecycle_state", pending_values=["staged"], completed_values=["evidence"]
    )
    assert catalog._lifecycle(bindings, {"lifecycle_state": "evidence"}) == "promoted"
    assert catalog._lifecycle(bindings, {"lifecycle_state": "staged"}) == "pending_promotion"
    assert catalog._lifecycle(bindings, {"lifecycle_state": "anything else"}) == "context"


def test_binding_override_merges_and_rejects_non_identifiers(tmp_path, monkeypatch):
    override = tmp_path / "bindings.json"
    override.write_text(json.dumps({"objects": {"relation": "x.y; drop table z"}}), "utf-8")
    monkeypatch.setenv("CONSIGNATIO_CATALOG_BINDINGS", str(override))
    bindings = catalog.load_bindings()
    assert bindings["objects"]["key"] == "object_key"
    with pytest.raises(catalog.CatalogUnavailable):
        catalog._ident(bindings["objects"]["relation"])


def test_artifact_url_uses_configured_template_only():
    bindings = catalog.load_bindings()
    assert catalog._artifact_url(bindings, "b", "k") is None
    bindings["artifact_url_template"] = "https://files.example/d/{bucket}/{key}"
    url = catalog._artifact_url(bindings, "salem-data", "consignatio/vault/a b.pdf")
    assert url == "https://files.example/d/salem-data/consignatio/vault/a%20b.pdf"
