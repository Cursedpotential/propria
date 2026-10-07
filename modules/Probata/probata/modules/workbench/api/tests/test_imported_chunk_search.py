"""Mobile search over ProfferChunks20261002: a hit is a chunk, opens its thread at the first message.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest
from app.repo import imported_pg as pg
from app.runtime import imported as runtime
from app.service import imported as service
from fastapi import FastAPI
from fastapi.testclient import TestClient

MATTER = "01a0f751-e07b-75cc-9ad5-63ad9449a8ba"
KEY = "b2://salem-data/consignatio/casevault/SourceCorpus/messaging/sms-backup-restore/8102689630/sms-2024-11-24.xml"
SV = "01a0fd15-c14c-7851-96b8-9ef5870988fd"
CALL_SV = "01a0fcea-20b7-711f-ba27-32ab9debee64"
PEOPLE = [
    {"entity_id": "e-kat", "person": "Katrina", "display_name": "Katrina Kinzel", "role_in_case": "co_parent",
     "verification_state": "confirmed", "identifier": "8102689630", "kind": "phone"},
]

CHUNK = {
    "text": "[2024-11-24 14:00] Katrina Kinzel: are you picking him up\n"
            "[2024-11-24 14:02] Matthew S. Salem: yes at six\n"
            "[2024-11-24 14:03] Katrina Kinzel: the custody order says five\n"
            "[2024-11-24 14:05] Matthew S. Salem: ok five then\n"
            "[2024-11-24 14:06] Katrina Kinzel: thanks",
    "participant_names": ["Katrina Kinzel", "Matthew S. Salem"], "start_at": "2024-11-24T14:00:00Z",
    "first_message_id": "01a0fd18-fc2e-743c-bc75-de44dc411ec5", "source_version_ids": [SV], "source_version_id": None,
    "record_kind": "conversation_chunk", "message_count": 5, "_additional": {"score": "1.5"},
}
CALLS = {
    "text": "[2022-09-24 04:02] missed call from Katrina Kinzel, 0s", "participant_names": ["Katrina Kinzel"],
    "start_at": "2022-09-24T04:02:08Z", "first_message_id": None, "source_version_ids": [CALL_SV],
    "source_version_id": CALL_SV, "record_kind": "call_log_file", "message_count": 1, "_additional": {"score": "0.9"},
}


@pytest.fixture(autouse=True)
def _fresh(monkeypatch):
    service._cache.clear()
    monkeypatch.setattr(service.settings, "proffer_matter_id", MATTER)
    monkeypatch.setattr(pg, "people", lambda: PEOPLE)
    monkeypatch.setattr(pg, "versions_to_threads", lambda matter, ids: [
        {"id": SV, "export_key": KEY, "conv": "8102689630"}] if SV in ids else [])


def _app() -> FastAPI:
    app = FastAPI()
    app.include_router(runtime.router)
    return app


def _weaviate(monkeypatch, hits, sent):
    class Client:
        def __init__(self, *a, **k): ...
        async def __aenter__(self): return self
        async def __aexit__(self, *a): return False

        async def post(self, url, json=None, **k):
            sent.append((url, json))

            class Response:
                status_code = 200

                def json(self_inner):
                    return {"data": {"Get": {service.settings.imported_weaviate_class: hits}}}

            return Response()

    monkeypatch.setattr(service.httpx, "AsyncClient", Client)


def test_search_targets_the_chunk_collection_and_filters_to_the_live_matter(monkeypatch):
    sent: list = []
    _weaviate(monkeypatch, [], sent)
    TestClient(_app()).get("/api/imported/search", params={"q": "custody order"})
    url, body = sent[0]
    assert service.settings.imported_weaviate_class == "ProfferChunks20261002"
    assert url.endswith("/v1/graphql") and "ProfferChunks20261002(" in body["query"]
    assert 'properties: ["text"]' in body["query"] and f'valueText: {json.dumps(MATTER)}' in body["query"]
    assert "ProfferMsgEvents" not in body["query"]


def test_deployment_uses_the_chunk_reader_collection():
    """Reject deployment drift that makes Read query chunk fields on raw events.

    Input: tracked Workbench compose. Output: matching reader collection. No
    network or writes; this checks the deployment boundary omitted by unit mocks.
    """
    compose = Path(__file__).resolve().parents[4] / "deploy" / "workbench.yaml"
    value = next(line.split(":", 1)[1].strip() for line in compose.read_text(encoding="utf-8").splitlines()
                 if line.strip().startswith("IMPORTED_WEAVIATE_CLASS:"))
    assert value == service.settings.imported_weaviate_class


def test_a_chunk_hit_opens_its_thread_at_the_first_message_with_the_matching_lines(monkeypatch):
    sent: list = []
    _weaviate(monkeypatch, [CHUNK, CALLS], sent)
    body = TestClient(_app()).get("/api/imported/search", params={"q": "custody order"}).json()
    chunk, calls = body["items"]
    assert chunk["id"] == CHUNK["first_message_id"]  # the Workbench opens the thread with ?focus=<this id>
    assert chunk["thread_id"] == service.encode_id(KEY, "8102689630")
    assert chunk["kind"] == "conversation" and chunk["message_count"] == 5
    assert chunk["sender"] == "Katrina Kinzel, Matthew S. Salem"
    lines = chunk["body"].split("\n")
    assert "custody order" in lines[1] and 1 < len(lines) <= 4  # the matching line with its neighbours, not the whole chunk
    assert calls["kind"] == "call_log" and calls["thread_id"] is None and calls["sender"].startswith("Call log: ")
    assert calls["id"] == CALL_SV


def test_snippet_and_names_label_edges():
    assert service._snippet("a\nb\nc", "zzz").split("\n") == ["a", "b", "c"]  # no word matches: the first lines
    long = "x " * 400
    assert len(service._snippet(long, "x")) <= 424
    assert service._names_label([]) == "Unknown"
    assert service._names_label(["A", "B", "C", "D", "A"]) == "A, B, C +1"


def test_search_keeps_all_chunk_source_versions_and_distinct_object_identities(monkeypatch):
    """Synthetic overlapping chunks keep provenance without inventing missing locators."""
    first, second = "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
    resolved = "b2://synthetic/original-one.txt"
    requested = []
    def resolve(matter, ids):
        requested.extend(ids)
        return [{"id": first, "export_key": resolved, "conv": "synthetic"}]
    monkeypatch.setattr(pg, "versions_to_threads", resolve)
    monkeypatch.setattr(service, "describe_export", lambda *args: {"file_name": "original-one.txt", "format": "Other", "device": None})
    hits = [{"text": "same synthetic passage", "source_version_ids": [first, second],
             "source_version_id": first, "participant_names": [],
             "_additional": {"id": object_id, "score": 1}}
            for object_id in ("chunk-one", "chunk-two")]
    sent = []
    _weaviate(monkeypatch, hits, sent)
    body = TestClient(_app()).get("/api/imported/search", params={"q": "synthetic passage"}).json()
    assert requested == [first, second]
    assert [item["object_id"] for item in body["items"]] == ["chunk-one", "chunk-two"]
    assert all(item["source_version_ids"] == [first, second] for item in body["items"])
    assert body["items"][0]["source_versions"] == [
        {"id": first, "source_uri": resolved}, {"id": second, "source_uri": None}]
    assert "_additional { id score }" in sent[0][1]["query"]
