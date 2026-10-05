"""Conversation actions: Extract and Send to Surreal forwarding, the workflow guard, and extraction grouping.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""

from __future__ import annotations

import inspect
import json
from datetime import UTC, datetime

import httpx
import pytest
from app.repo import extractions_pg as pg
from app.runtime import conversation_actions as runtime
from app.service import conversation_actions as service
from app.service import imported, proffer
from fastapi import FastAPI, Request
from fastapi.testclient import TestClient

MATTER = "01a0f751-e07b-75cc-9ad5-63ad9449a8ba"
KEY = "b2://salem-data/consignatio/casevault/SourceCorpus/messaging/sms-backup-restore/8102689630/sms-2024-11-24.xml"
THREAD = imported.encode_id(KEY, "8103099590")
OTHER = imported.encode_id(KEY, "8105550101")

REGISTRY = {
    "extractors": [
        {"id": "go-kimi-k3", "label": "Go + kimi-k3 (default)", "kind": "internal", "default": True, "compare_only": False,
         "run_extractor": "probata.extract.model",
         "run_extractors": ["probata.entities.rules", "probata.extract.model", "probata.entities.reconcile"]},
        {"id": "semantica", "label": "Semantica (patterns)", "kind": "external", "default": False, "compare_only": True,
         "run_extractor": "semantica", "run_extractors": ["semantica"]},
        {"id": "langextract", "label": "LangExtract (kimi-k3)", "kind": "external", "default": False, "compare_only": True,
         "run_extractor": "langextract", "run_extractors": ["langextract"]},
    ],
    "default": "go-kimi-k3",
}


class Starter:
    """Records what the BFF forwards to the Proffer starter and answers like it."""

    def __init__(self) -> None:
        self.calls: list[dict] = []

    async def request(self, method: str, path: str, **kwargs):
        self.calls.append({"method": method, "path": path, **kwargs})
        if path == "/reference-import/extractors":
            return httpx.Response(200, json=REGISTRY)
        if path.endswith("/extract"):
            return httpx.Response(202, json={"workflow_id": "conversation-extraction:abc12345", "run_id": "r", "kind": "extraction", "conversations": len(kwargs["json"]["conversations"])})
        if path.endswith("/send-to-surreal"):
            return httpx.Response(202, json={"workflow_id": "send-to-surreal:abc12345", "run_id": "r", "kind": "send_to_surreal", "conversations": len(kwargs["json"]["conversations"])})
        if "/workflows/" in path:
            return httpx.Response(200, json={"workflow_id": path.rsplit("/", 1)[1], "kind": "extraction", "outcome": "running", "steps": [], "receipts": []})
        return httpx.Response(404, json={"error": "unknown"})


@pytest.fixture
def starter(monkeypatch):
    fake = Starter()
    monkeypatch.setattr(proffer, "_request", fake.request)
    monkeypatch.setattr(imported.settings, "proffer_matter_id", MATTER)
    service._reset_registry_cache()
    return fake


def _client(subject: bool = True) -> TestClient:
    app = FastAPI()

    @app.middleware("http")
    async def identity(request: Request, call_next):
        if subject:
            request.state.subject_uid = "uid-owner"
            request.state.principal = "owner"
        return await call_next(request)

    app.include_router(runtime.router)
    return TestClient(app)


def test_extract_forwards_decoded_conversations_actor_key_and_the_live_matter(starter):
    response = _client().post(
        "/api/imported/threads/extract",
        json={"thread_ids": [THREAD, OTHER, THREAD], "extractors": ["go-kimi-k3", "semantica", "semantica"]},
        headers={"Idempotency-Key": "click-0001"},
    )
    assert response.status_code == 202, response.text
    assert response.json()["workflow_id"] == "conversation-extraction:abc12345"
    call = starter.calls[0]
    assert call["path"] == "/reference-import/conversations/extract"
    assert call["json"]["matter_id"] == MATTER
    assert call["json"]["conversations"] == [{"export_key": KEY, "conv": "8103099590"}, {"export_key": KEY, "conv": "8105550101"}]
    assert call["json"]["extractors"] == ["go-kimi-k3", "semantica"]
    assert call["headers"]["Idempotency-Key"] == "click-0001"
    assert call["headers"]["X-authentik-username"] == "owner"


def test_extract_without_a_key_or_a_subject_starts_nothing(starter):
    body = {"thread_ids": [THREAD]}
    assert _client().post("/api/imported/threads/extract", json=body).status_code == 422
    assert _client(subject=False).post("/api/imported/threads/extract", json=body, headers={"Idempotency-Key": "click-0001"}).status_code == 401
    assert starter.calls == []


def test_extract_refuses_an_unknown_conversation_or_a_bad_body(starter):
    headers = {"Idempotency-Key": "click-0001"}
    assert _client().post("/api/imported/threads/extract", json={"thread_ids": ["not-a-real-token"]}, headers=headers).status_code == 404
    assert _client().post("/api/imported/threads/extract", json={"thread_ids": []}, headers=headers).status_code == 422
    assert _client().post("/api/imported/threads/extract", json={"thread_ids": [THREAD], "extra": 1}, headers=headers).status_code == 422
    assert _client().post("/api/imported/threads/extract", json={"thread_ids": [THREAD], "extractors": ["Bad Id"]}, headers=headers).status_code == 422
    assert starter.calls == []


def test_send_to_surreal_forwards_the_include_flag(starter):
    response = _client().post(
        "/api/imported/threads/send-to-surreal",
        json={"thread_ids": [THREAD], "include_extractions": False},
        headers={"Idempotency-Key": "send-0001"},
    )
    assert response.status_code == 202, response.text
    call = starter.calls[0]
    assert call["path"] == "/reference-import/conversations/send-to-surreal"
    assert call["json"]["include_extractions"] is False and call["json"]["matter_id"] == MATTER


def test_extractors_lists_the_engine_registry(starter):
    response = _client().get("/api/extractors")
    assert response.status_code == 200
    assert [e["id"] for e in response.json()["extractors"]] == ["go-kimi-k3", "semantica", "langextract"]
    assert response.json()["default"] == "go-kimi-k3"


def test_workflow_status_reads_only_workflows_this_surface_starts(starter):
    assert _client().get("/api/imported/workflows/conversation-extraction:abc12345").status_code == 200
    assert _client().get("/api/imported/workflows/send-to-surreal:abc12345").status_code == 200
    assert _client().get("/api/imported/workflows/proffer-workflow-123456").status_code == 404
    assert [c["path"] for c in starter.calls] == [
        "/reference-import/conversations/workflows/conversation-extraction:abc12345",
        "/reference-import/conversations/workflows/send-to-surreal:abc12345",
    ]


def _run(run_id: str, extractor: str, generation: str = "g1", status: str = "completed") -> dict:
    return {"run_id": run_id, "extractor": extractor, "extractor_version": "1", "model_id": "", "status": status, "error": "",
            "started_at": datetime(2026, 10, 2, tzinfo=UTC), "finished_at": datetime(2026, 10, 2, tzinfo=UTC),
            "stats": {"messages": 5}, "generation_id": generation}


def _entity(entity_id: str, run_id: str, name: str) -> dict:
    return {"id": entity_id, "run_id": run_id, "name": name, "entity_type": "person", "confidence": 0.7, "review_state": "pending",
            "aliases": [{"text": "Kat"}], "model_mentions": 2, "mention_count": 0,
            "mentions": [{"record_id": "r1", "surface": name, "snippet": "hi " + name}]}


def _event(event_id: str, run_id: str, title: str) -> dict:
    return {"id": event_id, "run_id": run_id, "title": title, "event_type": "court", "occurred_at": datetime(2025, 7, 2, tzinfo=UTC),
            "precision": "point", "confidence": 0.6, "review_state": "pending", "record_ids": ["r1", None], "description": "", "when_stated": None}


def test_extractions_are_grouped_by_the_extractor_that_made_them(starter, monkeypatch):
    monkeypatch.setattr(pg, "runs", lambda *a: [
        _run("run-rules", "probata.entities.rules"), _run("run-model", "probata.extract.model"), _run("run-rec", "probata.entities.reconcile"),
        _run("run-sem", "semantica"), _run("run-odd", "someone.else", status="failed"),
    ])
    monkeypatch.setattr(pg, "entities", lambda *a: [_entity("e1", "run-rec", "Katherine"), _entity("e2", "run-sem", "Katherine Doe"), _entity("e3", "run-sem", "Flint")])
    monkeypatch.setattr(pg, "events", lambda *a: [_event("v1", "run-model", "Court date"), _event("v2", "run-sem", "court: hearing")])
    response = _client().get(f"/api/imported/threads/{THREAD}/extractions")
    assert response.status_code == 200, response.text
    groups = response.json()["extractors"]
    assert [g["id"] for g in groups] == ["go-kimi-k3", "semantica", "someone.else"]
    default, semantica, odd = groups
    assert default["counts"] == {"entities": 1, "events": 1, "runs": 3} and default["compare_only"] is False
    assert semantica["counts"] == {"entities": 2, "events": 1, "runs": 1} and semantica["compare_only"] is True
    assert odd["status"] == "failed" and odd["counts"]["entities"] == 0
    first = default["entities"][0]
    assert first["aliases"] == ["Kat"] and first["mention_count"] == 2 and first["mentions"][0]["snippet"] == "hi Katherine"
    assert default["events"][0]["record_ids"] == ["r1"]
    assert response.json()["truncated"] is False


def test_extractions_group_under_their_own_names_when_the_engine_is_unreachable(monkeypatch):
    async def down(method, path, **kwargs):
        raise proffer.ProfferError("down", 503)

    monkeypatch.setattr(proffer, "_request", down)
    monkeypatch.setattr(imported.settings, "proffer_matter_id", MATTER)
    service._reset_registry_cache()
    monkeypatch.setattr(pg, "runs", lambda *a: [_run("run-sem", "semantica")])
    monkeypatch.setattr(pg, "entities", lambda *a: [_entity("e2", "run-sem", "Flint")])
    monkeypatch.setattr(pg, "events", lambda *a: [])
    groups = _client().get(f"/api/imported/threads/{THREAD}/extractions").json()["extractors"]
    assert [g["id"] for g in groups] == ["semantica"] and groups[0]["counts"]["entities"] == 1


def test_a_conversation_nobody_extracted_has_no_groups(starter, monkeypatch):
    monkeypatch.setattr(pg, "runs", lambda *a: [])
    monkeypatch.setattr(pg, "entities", lambda *a: [])
    monkeypatch.setattr(pg, "events", lambda *a: [])
    assert _client().get(f"/api/imported/threads/{THREAD}/extractions").json() == {"thread_id": THREAD, "extractors": [], "truncated": False}


def test_the_extraction_reads_never_write():
    """Every SQL statement in the repo module is a SELECT (the statements are the module's non-docstring string constants)."""
    import ast
    import re

    tree = ast.parse(inspect.getsource(pg))
    statements = []
    for node in ast.walk(tree):
        if isinstance(node, ast.FunctionDef):
            for child in ast.walk(node):
                if isinstance(child, ast.Constant) and isinstance(child.value, str) and child is not node.body[0].value:
                    if "SELECT" in child.value:
                        statements.append(child.value)
    assert statements, "no SQL found"
    for sql in statements:
        assert not re.search(r"(INSERT|UPDATE|DELETE|DROP|ALTER|GRANT|CREATE|TRUNCATE)", sql), sql[:80]


def test_the_default_extractor_is_listed_once_and_marked(starter):
    listing = json.loads(_client().get("/api/extractors").text)
    assert [e["id"] for e in listing["extractors"] if e["default"]] == ["go-kimi-k3"]
