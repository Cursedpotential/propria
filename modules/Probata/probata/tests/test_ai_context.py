"""Exercise the context-only AI contract without custody or live services.

Inputs: synthetic native exports and in-memory provider/search doubles.
Outputs: bounded assertions. Effects: none outside test memory. Choose to prove
that exact source locators and partial enrichment preserve publishable chunks.
"""
from __future__ import annotations

import importlib.util
import io
import sys
from pathlib import Path
from types import SimpleNamespace

import pytest


def _module():
    """Load the owned context module without importing legacy providers.

    Inputs: test source path. Outputs: module. Effects: module load only. Choose
    to keep this contract independent of ai_content_provider availability.
    """
    source = Path(__file__).resolve().parents[1] / "server" / "analysis" / "ai_context.py"
    spec = importlib.util.spec_from_file_location("ai_context_under_test", source)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _request():
    """Return one synthetic B2 source identity.

    Inputs: none. Outputs: ai-context-v1 request. Effects: none. Choose for
    source-version and no-gate contract assertions.
    """
    return {"contract_version": "ai-context-v1", "source_ref": "b2://bucket/path/export.json",
            "provider_version_id": "v7", "request_id": "synthetic-run", "source_format": "claude"}


def test_b2_exact_version_without_evidence_pins(monkeypatch):
    """Read exact B2 version and reject a response from another version.

    Inputs: in-memory S3 responses. Outputs: exact kwargs and mismatch error.
    Effects: environment and SDK mocks only. Choose for object-store semantics.
    """
    context = _module()
    calls = []

    def client(kind, endpoint_url=None):
        """Supply a tiny S3-compatible test client.

        Inputs: service and endpoint. Outputs: fake client. Effects: call log.
        Choose to avoid any network or original-source fixture files.
        """
        assert kind == "s3" and endpoint_url == "https://synthetic.invalid"

        def get_object(**kwargs):
            """Return exact synthetic native bytes.

            Inputs: S3 coordinates. Outputs: response. Effects: call log.
            Choose for VersionId request and response checks.
            """
            calls.append(kwargs)
            return {"ContentLength": 2, "VersionId": "v7", "Body": io.BytesIO(b"{}")}

        return SimpleNamespace(get_object=get_object)

    monkeypatch.setenv("AI_CONTEXT_B2_ENDPOINT_URL", "https://synthetic.invalid")
    monkeypatch.setitem(sys.modules, "boto3", SimpleNamespace(client=client))
    assert context._source_bytes(_request()) == b"{}"
    assert calls == [{"Bucket": "bucket", "Key": "path/export.json", "VersionId": "v7"}]
    assert "source_version_id" not in context.identity(_request())
    monkeypatch.setitem(sys.modules, "boto3", SimpleNamespace(client=lambda *args, **kwargs:
        SimpleNamespace(get_object=lambda **kwargs: {"ContentLength": 2, "VersionId": "v8", "Body": io.BytesIO(b"{}") })))
    with pytest.raises(context.ContextInvalid, match="version differs"):
        context._source_bytes(_request())


def test_native_pointers_and_partial_candidates_remain_searchable(monkeypatch):
    """Retain Claude pointers, role/time and publishable partial search text.

    Inputs: synthetic Claude export and failed optional model. Outputs: exact
    native coordinates and search object. Effects: in-memory module mocks only.
    Choose for context continuity after enrichment failure.
    """
    context = _module()
    native = [{"uuid": "conversation-1", "name": "Topic", "chat_messages": [
        {"uuid": "message-1", "sender": "human", "created_at": "2024-01-02T00:00:00Z", "text": "hello"},
        {"uuid": "message-2", "sender": "assistant", "created_at": "2024-01-02T00:01:00Z", "text": "alice@example.com"}]}]
    records = context.decode_records(native, _request())
    assert [record["native_json_pointer"] for record in records] == ["/0/chat_messages/0/text", "/0/chat_messages/1/text"]
    assert records[1]["role"] == "assistant" and records[1]["native_time"] == "2024-01-02T00:01:00Z"
    assert records[1]["source_span"] == {"start": 0, "end": 17, "unit": "unicode_codepoint"}
    chunk = {"conversation_index": 0, "conversation_id": "conversation-1", "chunk_index": 0,
             "text": "hello\n\nalice@example.com", "segments": [{**{key: value for key, value in record.items() if key != "body"},
                 "text": record["body"], "body_start": 0, "body_end": len(record["body"])} for record in records]}
    accepted, rejected = context._ground(chunk, {"candidates": [
        {"kind": "entity", "title": "account", "record_id": records[1]["record_id"], "quote": "alice@example.com"},
        {"kind": "fact", "title": "invented", "record_id": records[1]["record_id"], "quote": "not in source"}]})
    assert len(accepted) == rejected == 1
    assert accepted[0]["occurrences"][0]["start"] == 0
    prepared = {"original_ref": "file:///synthetic/original.json", "chunks": [chunk]}
    candidates = {"chunks": [{"status": "partial_enrichment"}]}
    embedded = {"vectors": None}
    request = {**_request(), "prepared_ref": "file:///synthetic/prepared.json",
               "candidates_ref": "file:///synthetic/candidates.json", "work_products_ref": "file:///synthetic/works.json"}
    obj = context.search_objects(request, prepared, candidates, embedded, "AiChatEvents20260918", "text_nim")[0]
    assert obj["properties"]["body"] == chunk["text"]
    assert obj["properties"]["promotion_policy"] == "forbidden"
    assert "vectors" not in obj
    assert "source_sha256" not in obj["properties"]["provenance"][0]


def test_failed_candidate_provider_still_publishes_chunks(monkeypatch):
    """Publish prepared context after optional extraction fails.

    Inputs: in-memory stage bundles, failing model and search store. Outputs:
    partial publication with one searchable chunk. Effects: memory only. Choose
    to protect the core context path from enrichment availability.
    """
    context = _module()
    params = {**_request(), "prepared_ref": "file:///synthetic/prepared.json",
              "work_products_ref": "file:///synthetic/work_products.json",
              "candidates_ref": "file:///synthetic/candidates.json",
              "embeddings_ref": "file:///synthetic/embedded.json"}
    segment = {"record_id": "/0/chat_messages/0/text", "native_json_pointer": "/0/chat_messages/0/text",
               "role": "assistant", "native_time": None, "text": "source text", "body_start": 0,
               "body_end": 11, "conversation_title": "Synthetic"}
    chunk = {"conversation_index": 0, "conversation_id": "conversation-1", "chunk_index": 0,
             "text": "source text", "segments": [segment]}
    prepared = {"contract_version": context.VERSION, "stage": "prepared", "source": context.identity(params),
                "original_ref": "file:///synthetic/original.json", "chunks": [chunk], "counts": {"chunks": 1}}
    works = {"contract_version": context.VERSION, "stage": "work_products", "source": context.identity(params),
             "prepared_ref": params["prepared_ref"], "work_products": []}
    embedded = {"contract_version": context.VERSION, "stage": "embedded", "source": context.identity(params),
                "prepared_ref": params["prepared_ref"], "vectors": None, "status": "partial_enrichment"}
    bundles = {"prepared": prepared, "work_products": works, "embedded": embedded}

    def read(ref, request, stage):
        """Return a stage bundle from memory.

        Inputs: ref, request and stage. Outputs: bundle. Effects: none. Choose
        to avoid filesystem activity in this focused test.
        """
        assert ref == params["embeddings_ref" if stage == "embedded" else f"{stage}_ref"]
        return bundles[stage]

    def save(path, bundle):
        """Retain a stage bundle in memory.

        Inputs: path and bundle. Outputs: synthetic URI. Effects: memory map.
        Choose to test stage boundaries without disk writes.
        """
        bundles[bundle["stage"]] = bundle
        return params.get(f"{bundle['stage']}_ref", "file:///synthetic/published.json")

    monkeypatch.setattr(context, "_read", read)
    monkeypatch.setattr(context, "_write", save)
    monkeypatch.setattr(context, "_model_reply", lambda *args: (_ for _ in ()).throw(RuntimeError("offline")))
    result = context.extract_candidates(params)
    assert result["status"] == "partial_enrichment"
    assert bundles["candidates"]["chunks"][0]["reason"] == "RuntimeError"

    class Store:
        """Hold one additive search object in memory.

        Inputs: context object. Outputs: readback. Effects: memory write.
        Choose to prove publication without a production search service.
        """
        collection = "AiChatEvents20260918"
        vector_name = "text_nim"

        def __init__(self):
            """Initialize an empty object map.

            Inputs: none. Outputs: store. Effects: memory allocation. Choose
            for a fresh additive publish test.
            """
            self.objects = {}

        def get(self, object_id):
            """Read a synthetic search object.

            Inputs: object ID. Outputs: object or absence. Effects: none. Choose
            for publish collision checks.
            """
            return self.objects.get(object_id)

        def add(self, obj):
            """Insert a synthetic search object.

            Inputs: object. Outputs: none. Effects: memory write. Choose for
            in-memory partial publication proof.
            """
            self.objects[obj["id"]] = obj

    store = Store()
    result = context.publish(params, store=store)
    assert result["status"] == "partial_enrichment" and result["objects_written"] == 1
    assert next(iter(store.objects.values()))["properties"]["body"] == "source text"
