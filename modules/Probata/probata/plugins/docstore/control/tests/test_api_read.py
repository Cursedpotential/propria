"""Verify read-only API reads with synthetic IDs/rows; output current bodies, version metadata or errors, never mutate stores, and complement MCP forwarding tests."""
import hashlib
import importlib.util
import sys
from pathlib import Path

import httpx
import pytest
from fastmcp import Client
from fastapi import HTTPException

_API_DIR = Path(__file__).resolve().parents[4] / "scripts" / "docstore"
_CONTROL_DIR = Path(__file__).resolve().parents[1]
if str(_CONTROL_DIR) not in sys.path:
    sys.path.insert(0, str(_CONTROL_DIR))
if str(_API_DIR) not in sys.path:
    sys.path.insert(0, str(_API_DIR))
_spec = importlib.util.spec_from_file_location("docstore_api_read_test_target", _API_DIR / "api.py")
_api = importlib.util.module_from_spec(_spec)
sys.modules[_spec.name] = _api
_spec.loader.exec_module(_api)
_CONTROL_SPEC = importlib.util.spec_from_file_location(
    "docstore_control_api_read_test_target", Path(__file__).parents[1] / "server.py")
_control = importlib.util.module_from_spec(_CONTROL_SPEC)
sys.modules[_CONTROL_SPEC.name] = _control
_CONTROL_SPEC.loader.exec_module(_control)
_revisions = importlib.import_module("revisions")
emitted_document_id = _revisions.document_id
emitted_revision_id = _revisions.revision_id


class FakeDB:
    def __init__(self, head, revision, projection):
        self.head = head
        self.revision = revision
        self.projection = projection
        self.calls = []
        self.closed = False

    async def query(self, statement, params=None):
        self.calls.append((statement, params))
        record_id = params["r"]
        if record_id.startswith("docstore_document:"):
            return [self.head]
        if record_id.startswith("docstore_revision:"):
            return [self.revision]
        return [self.projection]

    async def close(self):
        self.closed = True


@pytest.fixture
def revision_rows(monkeypatch):
    """Provide synthetic revision and projection rows with exact multiline bodies.

    Input: pytest monkeypatch; output: fake database, IDs, and revision body.
    Side effects: Replaces only the API module's connector for these tests.
    Use for read roundtrips rather than tests requiring a live Docstore.
    Byline: Codex · GPT-6 · 2026-10-07.
    """
    monkeypatch.setattr(_api, "TOKEN", "synthetic-token")
    document_key = "note:synthetic_revision_read"
    document_id = emitted_document_id(document_key)
    body = "Current governed body - synthetic only.\n\n    Indented line  with  spaces.\n"
    body_hash = hashlib.sha256(body.encode("utf-8")).hexdigest()
    revision_id = emitted_revision_id(document_key, 3)
    head = {
        "id": document_id, "document_key": document_key, "source_path": "docs/synthetic.md",
        "title": "Synthetic  revision", "generation": 3, "latest_number": 3,
        "current_revision": revision_id, "approved_revision": None, "current_hash": body_hash,
        "updated_at": "2026-10-07 12:00:00",
    }
    revision = {
        "id": revision_id, "document": document_id, "number": 3, "body": body,
        "raw_sha256": body_hash, "projection_sha256": "b" * 64,
        "projection_profile": "test-profile", "source_path": "docs/synthetic.md",
        "actor": "test", "source_ref": "synthetic:test", "at": "2026-10-07 12:00:00",
    }
    projection = {
        "id": "document:docs_legacy_md", "source_path": "docs/legacy.md", "title": "Legacy",
        "doc_type": "doc", "domains": ["docs"], "status": "active", "observed_at": "today",
        "body": "Legacy projected body.\n\n    Indented line  with  spaces.\n",
    }
    db = FakeDB(head, revision, projection)

    async def connect(*_args):
        return db

    monkeypatch.setattr(_api.sq, "connect", connect)
    return db, document_id, revision_id, body


@pytest.mark.asyncio
async def test_emitted_revision_document_id_reads_verified_current_body(revision_rows):
    """Read an exact multiline revision body through MCP with its integrity hash.

    Input: synthetic revision rows; output: assertions on the returned document.
    Side effects: Only fake database reads; use for governed revision IDs.
    Byline: Codex · GPT-6 · 2026-10-07.
    """
    db, document_id, revision_id, body = revision_rows
    _api.TOKEN = "synthetic-token"
    source_root = Path(__file__).resolve().parents[4]
    config = _control.Config(
        "https://docstore.invalid", source_root, source_root.parent / "test-control-state",
        "docstore-api-roundtrip", "synthetic-token")
    server = _control.build_server(config, httpx.ASGITransport(app=_api.app))
    async with Client(server) as client:
        result = (await client.call_tool("docstore_get", {"record_id": document_id})).data
    assert result["id"] == document_id
    assert result["document_key"] == "note:synthetic_revision_read"
    assert result["body"] == body
    assert result["title"] == "Synthetic revision"
    assert result["generation"] == result["latest_number"] == result["revision"]["number"] == 3
    assert result["current_revision"] == result["revision"]["id"] == revision_id
    assert result["revision"]["raw_sha256"] == result["current_hash"]
    assert len(db.calls) == 2
    assert all("type::record($r)" in statement for statement, _ in db.calls)
    assert db.calls[0][1] == {"r": document_id}
    assert db.calls[1][1] == {"r": revision_id}
    assert db.closed


@pytest.mark.asyncio
async def test_legacy_projection_id_still_uses_existing_document_read(revision_rows):
    """Read an exact multiline legacy body through the API and MCP surfaces.

    Input: synthetic projection row; output: assertions on both read surfaces.
    Side effects: Only fake database reads; use for legacy document IDs.
    Byline: Codex · GPT-6 · 2026-10-07.
    """
    db, *_ = revision_rows
    result = await _api.doc("document:docs_legacy_md", authorization="Bearer synthetic-token")
    assert result["id"] == "document:docs_legacy_md"
    assert result["body"] == db.projection["body"]
    source_root = Path(__file__).resolve().parents[4]
    config = _control.Config(
        "https://docstore.invalid", source_root, source_root.parent / "test-control-state",
        "docstore-api-roundtrip", "synthetic-token")
    server = _control.build_server(config, httpx.ASGITransport(app=_api.app))
    async with Client(server) as client:
        via_mcp = (await client.call_tool("docstore_get", {"record_id": "document:docs_legacy_md"})).data
    assert via_mcp["body"] == db.projection["body"]
    assert len(db.calls) == 2
    assert all(params == {"r": "document:docs_legacy_md"} for _, params in db.calls)


@pytest.mark.asyncio
@pytest.mark.parametrize("record_id", ["docstore_document:" + "a" * 63, "docstore_document:" + "A" * 64, "docstore_revision:" + "a" * 64 + "_1"])
async def test_malformed_or_unknown_id_family_rejected_before_query(revision_rows, record_id):
    db, *_ = revision_rows
    with pytest.raises(HTTPException) as exc:
        await _api.doc(record_id, authorization="Bearer synthetic-token")
    assert exc.value.status_code == 400
    assert db.calls == []


@pytest.mark.asyncio
async def test_current_revision_integrity_mismatch_is_not_returned(revision_rows):
    db, document_id, *_ = revision_rows
    db.revision["number"] = 2
    with pytest.raises(HTTPException) as exc:
        await _api.doc(document_id, authorization="Bearer synthetic-token")
    assert exc.value.status_code == 502
    assert db.closed


@pytest.mark.asyncio
@pytest.mark.parametrize("field,value", [
    ("document_key", "note:different_key"),
    ("document_key", "unknown:different_key"),
    ("current_revision", "docstore_revision:" + "f" * 64 + "_3"),
])
async def test_revision_head_identity_and_pointer_mismatch_rejected(revision_rows, field, value):
    db, document_id, *_ = revision_rows
    db.head[field] = value
    with pytest.raises(HTTPException) as exc:
        await _api.doc(document_id, authorization="Bearer synthetic-token")
    assert exc.value.status_code == 502
    assert len(db.calls) == 1
    assert db.closed
