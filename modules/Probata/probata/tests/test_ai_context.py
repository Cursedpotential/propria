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


def test_gemini_markdown_preserves_eight_native_turns_and_unicode_spans():
    """Preserve eight exact native turns without timestamp or JSON-pointer invention.

    Inputs: Unicode CRLF Markdown with an export header and role-like fenced code.
    Outputs: exact original-document slices and eight native roles. Effects: none;
    choose to catch text stripping, byte/codepoint confusion and fence splitting.
    """
    context = _module()
    bodies = ["\r\nFirst 😀 question\r\n", "\r\n# Keep this heading\r\n```md\r\n**You:**\r\nnot a turn\r\n```\r\n",
              "\r\nSecond question\r\n", "\r\nSecond answer\r\n", "\r\nThird question\r\n",
              "\r\nThird answer\r\n", "\r\nFourth question\r\n", "\r\nFourth answer\r\n"]
    text = "\ufeff# Native title\r\nExported on: 4/14/2026, 7:08:32 PM\r\n---\r\n" + "".join(
        ("**You:**\r\n" if index % 2 == 0 else "**Gemini:**\r\n") + body
        for index, body in enumerate(bodies))
    params = {**_request(), "source_format": "gemini_markdown"}
    records = context.decode_markdown_records(text, params)
    assert len(records) == 8
    assert [record["role"] for record in records] == ["user", "assistant"] * 4
    assert [record["body"] for record in records] == bodies
    for record in records:
        span = record["source_span"]
        assert text[span["start"]:span["end"]] == record["body"]
        assert record["native_json_pointer"] == ""
        assert record["native_time"] is None and record["source_available_from"] is None
    assert "# Keep this heading" in records[1]["body"]


def test_markdown_grounding_uses_absolute_native_document_offsets():
    """Ground a quote after an astral character in whole-source coordinates.

    Inputs: native Markdown and a bounded chunk slice. Outputs: exact quote and
    absolute codepoint occurrence. Effects: none; choose for Go review verifier
    interoperability without converting the transcript to synthetic JSON.
    """
    context = _module()
    text = "# Topic\n**You:**\n😀 ask\n**Gemini:**\n\nA precise response.\n"
    records = context.decode_markdown_records(text, {**_request(), "source_format": "gemini_markdown"})
    record = records[1]
    segment = {**{k: v for k, v in record.items() if k != "body"}, "body_start": 3,
               "body_end": len(record["body"]), "text": record["body"][3:]}
    candidate = {"record_id": record["record_id"], "quote": "precise response", "kind": "fact",
                 "statement": "Reported statement", "predicate": "reported"}
    accepted, rejected = context._ground({"segments": [segment]}, {"candidates": [candidate]})
    assert rejected == 0 and len(accepted) == 1
    occurrence = accepted[0]["occurrences"][0]
    assert text[occurrence["start"]:occurrence["end"]] == candidate["quote"]
    assert occurrence["native_json_pointer"] == ""
    assert occurrence["source_available_from"] is None
    assert accepted[0]["reported"]["statement"] == candidate["statement"]
    assert accepted[0]["reported"]["predicate"] == candidate["predicate"]


def test_markdown_prepare_retains_exact_original_bytes(monkeypatch, tmp_path):
    """Retain the full Markdown original alongside native turn records on retry.

    Inputs: bounded in-memory source and temporary context root. Outputs: exact
    original.md bytes and stable stage receipt. Effects: temporary test files;
    choose for original preservation independently of model inference.
    """
    context = _module()
    raw = "# Title\r\n**You:**\r\n 😀 question \r\n**Gemini:**\r\n answer \r\n".encode()
    import hashlib
    digest = hashlib.sha256(raw).hexdigest()
    params = {**_request(), "source_format": "gemini_markdown", "source_sha256": digest}
    monkeypatch.setenv("AI_CONTEXT_ROOT", str(tmp_path.resolve()))
    monkeypatch.setattr(context, "_source_bytes", lambda request: raw)
    monkeypatch.setattr(context, "topic_chunks", lambda records, request, neural=None: [])
    receipt = context.prepare(params)
    prepared = context._read(receipt["bundle_ref"], params, "prepared")
    assert Path(prepared["original_ref"].removeprefix("file://")).name == "original.md"
    original = context._root() / context._scope(params) / "original.md"
    assert original.read_bytes() == raw
    assert prepared["source"]["source_sha256"] == prepared["original_sha256"] == digest
    assert receipt["records"] == 2
    assert context.prepare(params) == receipt
    original.write_bytes(raw + b"tampered")
    with pytest.raises(context.ContextInvalid, match="retained original SHA256 differs"):
        context.prepare(params)


def test_source_hash_mismatch_rejects_before_retention(monkeypatch, tmp_path):
    """Reject a supplied source hash mismatch before any original is retained.

    Inputs: bounded native source with deliberately mismatched digest. Outputs:
    permanent rejection and no retained files. Effects: temporary test root only;
    choose to prove the source pin is checked rather than merely propagated.
    """
    context = _module()
    params = {**_request(), "source_format": "gemini_markdown", "source_sha256": "a" * 64}
    monkeypatch.setenv("AI_CONTEXT_ROOT", str(tmp_path.resolve()))
    monkeypatch.setattr(context, "_source_bytes", lambda request: b"**You:**\nsource\n")
    with pytest.raises(context.ContextInvalid, match="source SHA256 differs"):
        context.prepare(params)
    assert list(tmp_path.iterdir()) == []


def test_actual_original_hash_without_identity_mutation(monkeypatch, tmp_path):
    """Record the actual original digest without adding a new source identity pin.

    Inputs: native source with no supplied digest. Outputs: prepared original
    hash and unchanged request identity. Effects: temporary retained files only;
    choose for callers that rely on exact provider version rather than SHA pins.
    """
    import hashlib
    context = _module()
    raw = b"**You:**\nsource\n"
    params = {**_request(), "source_format": "gemini_markdown"}
    monkeypatch.setenv("AI_CONTEXT_ROOT", str(tmp_path.resolve()))
    monkeypatch.setattr(context, "_source_bytes", lambda request: raw)
    monkeypatch.setattr(context, "topic_chunks", lambda records, request, neural=None: [])
    before = context.identity(params)
    receipt = context.prepare(params)
    prepared = context._read(receipt["bundle_ref"], params, "prepared")
    assert prepared["original_sha256"] == hashlib.sha256(raw).hexdigest()
    assert prepared["source"] == context.identity(params) == before
    assert "source_sha256" not in before


def test_json_native_record_time_carries_existing_source_availability():
    """Carry native first-party message time independently of candidate event time.

    Inputs: Claude original-message metadata. Outputs: source availability copied
    from that original metadata. Effects: none; choose to keep knowledge time out
    of model-invented occurrence times and import/approval timestamps.
    """
    context = _module()
    record = context.decode_records({"chat_messages": [{"sender": "human", "text": "Statement",
        "created_at": "2024-01-02T00:01:00Z"}]}, _request())[0]
    assert record["source_available_from"] == "2024-01-02T00:01:00.000000Z"


def test_source_available_time_is_verified_native_utc():
    """Serialize real native epoch and ISO times without inventing missing zones.

    Inputs: native epoch and timezone-bearing values plus invalid/unknown values.
    Outputs: microsecond UTC RFC3339 or null. Effects: none. Choose to protect
    the Go time.Time contract and keep source knowledge separate from event time.
    """
    context = _module()
    assert context.source_available_time(1704153660.123456) == "2024-01-02T00:01:00.123456Z"
    assert context.source_available_time("2024-01-01T19:01:00.123456-05:00") == "2024-01-02T00:01:00.123456Z"
    for missing in (None, True, "2024-01-02T00:01:00", "not a date", float("nan"), 1e100):
        assert context.source_available_time(missing) is None


def test_context_primary_call_reuses_existing_ai_only_options(monkeypatch):
    """Apply the existing AI-only primary completion settings without SDK retries.

    Inputs: configured primary model and mocked SDK. Outputs: exact JSON/token/
    thinking/n request options and one call. Effects: memory only. Choose to keep
    max_model_calls meaningful and prevent thinking prose from breaking JSON.
    """
    import sys
    from types import SimpleNamespace, ModuleType
    context = _module()
    provider = ModuleType("server.analysis.ai_content_provider")
    provider.PRIMARY_MODELS = ("moonshotai/kimi-k3", "nvidia/nemotron-3-super-120b-a12b")
    provider.NIM_URL = "https://integrate.api.nvidia.com/v1"
    provider.MAX_OUTPUT_TOKENS = 6000
    monkeypatch.setitem(sys.modules, provider.__name__, provider)
    calls = {}

    def create(**kwargs):
        """Capture one test completion without provider traffic.

        Inputs: SDK keyword arguments. Outputs: synthetic empty candidate JSON.
        Effects: memory assignment only; choose to test the exact wire request.
        """
        calls["request"] = kwargs
        return SimpleNamespace(choices=[SimpleNamespace(message=SimpleNamespace(content='{"candidates":[]}'))])

    def client(**kwargs):
        """Capture SDK retry and timeout configuration without allocating a client.

        Inputs: constructor arguments. Outputs: mock completion client. Effects:
        memory only; choose to verify bounded provider call behavior.
        """
        calls["client"] = kwargs
        return SimpleNamespace(chat=SimpleNamespace(completions=SimpleNamespace(create=create)))

    monkeypatch.setitem(sys.modules, "openai", SimpleNamespace(OpenAI=client))
    monkeypatch.setenv("AI_CONTEXT_MODEL_BASE_URL", provider.NIM_URL)
    monkeypatch.setenv("AI_CONTEXT_MODEL_ID", provider.PRIMARY_MODELS[0])
    monkeypatch.setenv("AI_CONTEXT_MODEL_API_KEY", "unit-test-key")
    assert context._model_reply({"segments": []}, None) == {"candidates": []}
    assert calls["client"]["max_retries"] == 0
    assert calls["request"]["model"] == provider.PRIMARY_MODELS[0]
    assert calls["request"]["response_format"] == {"type": "json_object"}
    assert calls["request"]["max_tokens"] == 6000 and calls["request"]["n"] == 1
    assert calls["request"]["extra_body"] == {"chat_template_kwargs": {"thinking": False}}


def test_b2_mounted_credentials_preserve_exact_version(monkeypatch, tmp_path):
    """Use mounted existing B2 credential shape while preserving VersionId checks.

    Inputs: temporary synthetic credentials and an in-memory object client.
    Outputs: bounded body and exact provider-version request. Effects: temporary
    test file only; choose to verify the managed worker's private mount contract.
    """
    import json
    context = _module()
    credentials = {"endpoint_url": "https://synthetic.invalid", "region": "region-test",
                   "access_key_id": "test-id", "secret_access_key": "test-secret"}
    path = tmp_path / "credentials.json"
    path.write_text(json.dumps(credentials))
    monkeypatch.setenv("AI_CONTEXT_B2_CREDENTIALS_FILE", str(path))
    monkeypatch.delenv("AI_CONTEXT_B2_ENDPOINT_URL", raising=False)
    calls = []

    def client(kind, **options):
        """Assert file-derived connection options and return a bounded fake body.

        Inputs: service/options. Outputs: fake S3 client. Effects: request capture;
        choose to prove secrets need not become process command-line arguments.
        """
        assert kind == "s3"
        assert options["endpoint_url"] == credentials["endpoint_url"]
        assert options["aws_access_key_id"] == credentials["access_key_id"]
        assert options["aws_secret_access_key"] == credentials["secret_access_key"]

        def get_object(**kwargs):
            """Capture versioned object coordinates and return original bytes.

            Inputs: bucket/key/version. Outputs: bounded object response. Effects:
            memory capture only; choose instead of live provider calls in tests.
            """
            calls.append(kwargs)
            return {"ContentLength": 2, "VersionId": "v7", "Body": io.BytesIO(b"{}")}
        return SimpleNamespace(get_object=get_object)

    monkeypatch.setitem(sys.modules, "boto3", SimpleNamespace(client=client))
    assert context._source_bytes(_request()) == b"{}"
    assert calls[0]["VersionId"] == "v7"


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


def test_chatgpt_conversation_id_is_retained():
    """Keep ChatGPT's native conversation_id in derived source locators.

    Inputs: one synthetic mapping export. Outputs: native conversation ID.
    Effects: none. Choose for ChatGPT export citation identity.
    """
    context = _module()
    native = [{"conversation_id": "native-conversation-7", "current_node": "turn-1",
               "mapping": {"turn-1": {"parent": None, "message": {"author": {"role": "user"},
                           "content": {"parts": ["hello"]}}}}}]
    records = context.decode_records(native, {**_request(), "source_format": "chatgpt"})
    assert records[0]["conversation_id"] == "native-conversation-7"
    assert records[0]["native_json_pointer"] == "/0/mapping/turn-1/message/content/parts/0"


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


def test_rate_limit_defers_remaining_enrichment_calls(monkeypatch):
    """Stop provider requests after a throttle while retaining every topic chunk.

    Inputs: three synthetic chunks and a 429 model response. Outputs: one call
    and three partial entries. Effects: in-memory bundle writes only. Choose to
    prove provider cooldown does not discard context or hammer the endpoint.
    """
    context = _module()
    params = {**_request(), "prepared_ref": "file:///synthetic/prepared.json",
              "work_products_ref": "file:///synthetic/work_products.json"}
    chunks = [{"conversation_index": 0, "chunk_index": index, "segments": []}
              for index in range(3)]
    prepared = {"chunks": chunks, "counts": {"chunks": 3}}
    products = {"prepared_ref": params["prepared_ref"]}
    saved = {}
    monkeypatch.setattr(context, "_read", lambda ref, request, stage:
                        prepared if stage == "prepared" else products)

    def save(path, bundle):
        """Capture the candidate bundle without writing a file.

        Inputs: output path and bundle. Outputs: synthetic URI. Effects: memory
        assignment only. Choose for focused cooldown contract validation.
        """
        saved["bundle"] = bundle
        return "file:///synthetic/candidates.json"

    monkeypatch.setattr(context, "_write", save)

    class Throttle(Exception):
        """Model a provider HTTP 429 response.

        Inputs: none. Outputs: exception. Effects: none. Choose to exercise
        cooldown detection without an SDK dependency.
        """
        status_code = 429

    calls = []

    def throttled_model(prompt):
        """Record one attempted model request and return a throttle.

        Inputs: prompt. Outputs: exception. Effects: call count only. Choose
        to prove later chunks are not sent to the overloaded provider.
        """
        calls.append(prompt)
        raise Throttle()

    result = context.extract_candidates(params, model=throttled_model)
    assert len(calls) == 1
    assert result["status"] == "partial_enrichment"
    assert [entry["reason"] for entry in saved["bundle"]["chunks"]] == ["provider_cooldown"] * 3
    assert saved["bundle"]["counts"]["model_calls"] == 1
