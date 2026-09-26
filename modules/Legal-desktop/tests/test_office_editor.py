"""Protocol tests exercise actual durable document service and middleware boundary."""

from urllib.parse import parse_qs, urlsplit

import httpx
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient
from legal_workspace.api import office_routes
from legal_workspace.api.auth import LegalWorkspaceAuthMiddleware
from legal_workspace.services import office_editor
from legal_workspace.services.office_editor import OfficeEditor, OfficeError, request_token
from legal_workspace.services.work_documents import DocumentLocked, WorkDocumentService


@pytest.fixture
def setup(tmp_path):
    docs = WorkDocumentService(tmp_path, "office-test-matter")
    doc = docs.create_blank("Office round trip", actor="owner")
    bridge = OfficeEditor(docs, tmp_path)
    session = bridge.create_session(
        doc.document_id, "owner", "https://example.test/office/browser/edit"
    )
    return docs, doc, bridge, session


def test_scope_expiry_and_restart(setup, tmp_path):
    _docs, doc, bridge, session = setup
    token = session["access_token"]
    assert token not in (tmp_path / "legal.sqlite").read_bytes().decode(errors="ignore")
    reloaded = OfficeEditor(WorkDocumentService(tmp_path, "office-test-matter"), tmp_path)
    assert reloaded.authorize(doc.document_id, token)["actor"] == "owner"
    with pytest.raises(OfficeError, match="invalid"):
        reloaded.authorize("another-document", token)
    with bridge.connection() as conn:
        conn.execute("UPDATE office_editor_sessions SET expires=0")
    with pytest.raises(OfficeError, match="expired"):
        reloaded.authorize(doc.document_id, token)


def test_lock_save_reopen_and_original_retained(setup, tmp_path):
    docs, doc, bridge, session = setup
    token = session["access_token"]
    original = docs.read_bytes(doc.document_id)
    bridge.lock(doc.document_id, token, "LOCK", "test-lock")
    assert bridge.lock(doc.document_id, token, "GET_LOCK") == "test-lock"
    result = bridge.save(doc.document_id, token, "test-lock", original)
    assert result.current_revision == 2
    result = bridge.save(doc.document_id, token, "test-lock", original)
    assert result.current_revision == 3
    assert docs.read_bytes(doc.document_id, 1) == original
    restarted = OfficeEditor(WorkDocumentService(tmp_path, "office-test-matter"), tmp_path)
    assert restarted.info(doc.document_id, token)["Version"] == "3"
    assert restarted.read(doc.document_id, token) == (original, 3)


def test_competing_lock_and_external_write_rejected(setup):
    docs, doc, bridge, session = setup
    token = session["access_token"]
    bridge.lock(doc.document_id, token, "LOCK", "A")
    with pytest.raises(OfficeError) as conflict:
        bridge.lock(doc.document_id, token, "LOCK", "B")
    assert conflict.value.status == 409 and conflict.value.lock == "A"
    with pytest.raises(DocumentLocked):
        docs.save(doc.document_id, docs.read_bytes(doc.document_id), 1, actor="outside")
    with pytest.raises(OfficeError) as conflict:
        bridge.save(doc.document_id, token, "B", docs.read_bytes(doc.document_id))
    assert conflict.value.lock == "A"


def test_stale_session_cannot_overwrite_external_revision(setup):
    docs, doc, bridge, session = setup
    docs.save(doc.document_id, docs.read_bytes(doc.document_id), 1, actor="outside")
    with pytest.raises(OfficeError, match="changed"):
        bridge.lock(doc.document_id, session["access_token"], "LOCK", "old-editor")
    assert docs.get(doc.document_id).current_revision == 2


def test_refresh_replace_unlock_and_expiry(setup):
    docs, doc, bridge, session = setup
    token = session["access_token"]
    bridge.lock(doc.document_id, token, "LOCK", "A")
    bridge.lock(doc.document_id, token, "REFRESH_LOCK", "A")
    bridge.lock(doc.document_id, token, "LOCK", "B", "A")
    bridge.lock(doc.document_id, token, "UNLOCK", "B")
    with pytest.raises(OfficeError) as error:
        bridge.save(doc.document_id, token, "B", docs.read_bytes(doc.document_id))
    assert error.value.lock == ""
    bridge.lock(doc.document_id, token, "LOCK", "C")
    with bridge.connection() as conn:
        conn.execute("UPDATE office_editor_locks SET expires=0")
    with pytest.raises(OfficeError, match="expired"):
        bridge.lock(doc.document_id, token, "REFRESH_LOCK", "C")


def test_http_auth_and_put_protocol(setup, monkeypatch):
    docs, doc, bridge, session = setup
    monkeypatch.setattr(office_routes, "editor", lambda: bridge)
    app = FastAPI()
    app.add_middleware(LegalWorkspaceAuthMiddleware)
    app.include_router(office_routes.router)
    client = TestClient(app)
    path = f"/wopi/files/{doc.document_id}"
    assert client.get(path).status_code == 401  # even explicit test auth bypass cannot grant WOPI
    auth = {"Authorization": f"Bearer {session['access_token']}"}
    assert client.get(path, headers=auth).json()["BaseFileName"] == doc.filename
    wrong = client.get(path, headers=auth, params={"access_token": "wrong"})
    assert wrong.status_code == 401
    assert (
        client.post(
            path, headers={**auth, "X-WOPI-Override": "LOCK", "X-WOPI-Lock": "a"}
        ).status_code
        == 200
    )
    content = docs.read_bytes(doc.document_id)
    bad = client.post(
        path + "/contents",
        content=content,
        headers={**auth, "X-WOPI-Override": "PUT", "X-WOPI-Lock": "b"},
    )
    assert bad.status_code == 409 and bad.headers["X-WOPI-Lock"] == "a"
    good = client.post(
        path + "/contents",
        content=content,
        headers={**auth, "X-WOPI-Override": "PUT", "X-WOPI-Lock": "a"},
    )
    assert good.status_code == 200 and good.headers["X-WOPI-ItemVersion"] == "2"
    assert client.get(path + "/contents", headers=auth).content == content


def test_token_cannot_authorize_normal_api_route(setup, monkeypatch):
    _docs, _doc, bridge, session = setup
    monkeypatch.setenv("LEGAL_WORKSPACE_BYPASS_AUTH", "false")
    monkeypatch.setattr(office_routes, "editor", lambda: bridge)
    app = FastAPI()
    app.add_middleware(LegalWorkspaceAuthMiddleware)
    app.include_router(office_routes.router)
    result = TestClient(app).get(
        "/v1/office/status", headers={"Authorization": f"Bearer {session['access_token']}"}
    )
    assert result.status_code in {401, 503}


def test_discovery_retains_prefix_and_no_token_in_url(monkeypatch):
    monkeypatch.setenv("OFFICE_EDITOR_PUBLIC_URL", "https://office.test/office")
    monkeypatch.setenv("OFFICE_EDITOR_INTERNAL_URL", "http://legal-office:9980/office")
    monkeypatch.setenv("OFFICE_WOPI_BASE_URL", "http://legal-api:8010")
    office_editor._discovery.cache_clear()

    def discovery(url, **kwargs):
        assert url == "http://legal-office:9980/office/hosting/discovery"
        return httpx.Response(
            200,
            request=httpx.Request("GET", url),
            text="""
        <wopi-discovery><net-zone><app><action ext="odt" name="edit"
        urlsrc="https://internal/office/browser/version/cool.html?lang=&lt;lang&gt;&amp;"/>
        </app></net-zone></wopi-discovery>""",
        )

    monkeypatch.setattr(office_editor.httpx, "get", discovery)
    url = office_editor.discover_editor_url("odt", "doc-id")
    assert urlsplit(url).path == "/office/browser/version/cool.html"
    assert parse_qs(urlsplit(url).query) == {"WOPISrc": ["http://legal-api:8010/wopi/files/doc-id"]}
    office_editor._discovery.cache_clear()


def test_unconfigured_discovery_fails_closed(monkeypatch):
    monkeypatch.delenv("OFFICE_EDITOR_PUBLIC_URL", raising=False)
    with pytest.raises(OfficeError) as failure:
        office_editor.discover_editor_url("odt", "doc-id")
    assert failure.value.status == 503


def test_conflicting_token_sources_rejected():
    assert request_token("valid", "Bearer valid") == "valid"
    with pytest.raises(OfficeError):
        request_token("valid", "Bearer wrong")
