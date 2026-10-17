"""Verify atomic native AI decoding and search/result boundaries with synthetic sources.

Inputs: fictional JSON/Markdown files and bounded provider doubles. Outputs:
real DuckDB/file assertions and explicit simulated provider receipts. Effects:
temporary test files only; pick before any authorized historical-source import.
"""
from __future__ import annotations

import hashlib
import json
from pathlib import Path
from types import SimpleNamespace

import pytest

from server.analysis import ai_context as context
from server.analysis.ai_context_native import NativeDecodeInvalid, project_records


def _source(tmp_path, monkeypatch, native, source_format="chatgpt"):
    """Create one exact fictional original and native request.

    Inputs: temporary root/native value/format. Output: pinned request and bytes.
    Effects: synthetic file/environment only; pick for real decoder I/O checks.
    """
    source_root = tmp_path / "source"
    source_root.mkdir()
    raw = native.encode("utf-8") if isinstance(native, str) else json.dumps(native, ensure_ascii=False).encode("utf-8")
    original = source_root / ("original.md" if source_format == "gemini_markdown" else "original.json")
    original.write_bytes(raw)
    monkeypatch.setenv("AI_CONTEXT_ROOT", str(tmp_path / "derived"))
    monkeypatch.setenv("AI_CONTEXT_SOURCE_ROOT", str(source_root))
    params = {"contract_version": context.VERSION, "source_ref": original.as_uri(), "source_format": source_format,
              "request_id": "synthetic-atomic-source", "temporal_run_id": "synthetic-run",
              "workflow_id": "context-source:synthetic-atomic-source",
              "source_version_id": "10000000-0000-4000-8000-000000000001",
              "historical_source_version_id": "10000000-0000-4000-8000-000000000002",
              "source_sha256": hashlib.sha256(raw).hexdigest()}
    return params, raw


def _chatgpt():
    """Return a multipart native branch with identical work occurrences.

    Inputs: none. Output: fictional ChatGPT mapping. Effects: none; pick for
    multipart identity, discarded-branch preservation and native role tests.
    """
    return [{"conversation_id": "fictional-conversation", "title": "Fictional work", "current_node": "a/1~x",
             "mapping": {"u0": {"parent": None, "message": {"author": {"role": "user"}, "create_time": 1704153660.125,
                          "content": {"parts": ["😀 Alice reported a plan on 2024-01-02."]}}},
                         "a/1~x": {"parent": "u0", "message": {"author": {"role": "assistant"}, "create_time": 1704153661,
                                    "content": {"parts": ["```text\nfirst complete work\n```\n", "```text\nsecond complete work\n```\n", "```text\nfirst complete work\n```\n"]}}},
                         "discarded": {"parent": "u0", "message": {"author": {"role": "assistant"},
                                       "content": {"parts": ["alternate branch retained only in original"]}}}}}]


def test_actual_duckdb_native_fields_preserve_unicode_roles_clocks_and_branch(tmp_path, monkeypatch):
    """Decode real synthetic files through DuckDB without PG or topic inference.

    Inputs: multipart mapping. Outputs: four exact records and retained original.
    Effects: temporary files; pick to detect parser substitution and clock drift.
    """
    params, raw = _source(tmp_path, monkeypatch, _chatgpt())
    receipt = context.decode(params)
    decoded = context._read(receipt["bundle_ref"], params, "decoded")
    assert receipt["records"] == 4 and "chunks" not in decoded
    assert decoded["decoder"]["backend"] == "duckdb" and decoded["decoder"]["version"]
    assert [r["role"] for r in decoded["records"]] == ["user", "assistant", "assistant", "assistant"]
    assert decoded["records"][0]["native_time"] == 1704153660.125
    assert decoded["records"][0]["source_available_from"] == "2024-01-02T00:01:00.125000Z"
    assert decoded["records"][1]["native_json_pointer"] == "/0/mapping/a~11~0x/message/content/parts/0"
    assert Path(decoded["original_ref"].removeprefix("file://")).read_bytes() == raw
    assert b"alternate branch" in raw
    assert not any("alternate branch" in r["body"] for r in decoded["records"])
    assert context.decode(params) == receipt
    assert decoded["source"]["source_version_id"] == params["source_version_id"]


def test_actual_duckdb_claude_blocks_wrapped_root_and_no_silent_fallback():
    """Use existing Claude text precedence and exact block pointers in DuckDB.

    Inputs: wrapped native messages and invalid ChatGPT graph. Outputs: literal
    strings/locators or fail-closed error. Effects: ephemeral DuckDB only.
    """
    native = {"conversations": [{"uuid": "fictional-claude", "chat_messages": [
        {"uuid": "m1", "sender": "human", "created_at": "2024-01-02T00:00:00Z", "text": "  exact 😀  ",
         "content": [{"type": "text", "text": "unused"}]},
        {"uuid": "m2", "sender": "assistant", "text": "", "content": [
            {"type": "tool", "text": "not text"}, {"type": "text", "text": "first"}, {"type": "text", "text": "second"}]}]}]}
    records = project_records(json.dumps(native).encode(), "claude", 10)
    assert [r["body"] for r in records] == ["  exact 😀  ", "first", "second"]
    assert records[-1]["native_json_pointer"] == "/conversations/0/chat_messages/1/content/2/text"
    assert records[0]["role"] == "human"
    invalid = _chatgpt()
    invalid[0]["current_node"] = "missing"
    with pytest.raises(NativeDecodeInvalid, match="explicit active path"):
        project_records(json.dumps(invalid).encode(), "chatgpt", 10)
    with pytest.raises(NativeDecodeInvalid, match="DuckDB projection rejected"):
        project_records(b'{"mapping": []}', "chatgpt", 10)
    with pytest.raises(NativeDecodeInvalid, match="record bound"):
        project_records(json.dumps(native).encode(), "claude", 2)


def test_works_before_chunks_keep_multipart_and_repeated_occurrences(tmp_path, monkeypatch):
    """Retain complete work occurrences independently of chunk availability.

    Inputs: three native assistant parts. Outputs: three unique exact files and
    chunk-only preparation. Effects: temporary files; no text deduplication.
    """
    params, _ = _source(tmp_path, monkeypatch, _chatgpt())
    params["decoded_ref"] = context.decode(params)["bundle_ref"]
    works_receipt = context.extract_work_products(params)
    works = context._read(works_receipt["bundle_ref"], params, "work_products")
    assert works_receipt["work_products"] == 3
    assert "prepared_ref" not in works and works["decoded_ref"] == params["decoded_ref"]
    paths = [Path(w["file_ref"].removeprefix("file://")) for w in works["work_products"]]
    assert len(set(paths)) == 3
    assert [p.read_text() for p in paths] == [w["content"] for w in works["work_products"]]
    assert paths[0].read_bytes() == paths[2].read_bytes() and paths[0] != paths[2]
    assert context.extract_work_products(params) == works_receipt
    monkeypatch.setattr(context, "_source_bytes", lambda p: pytest.fail("chunk Activity re-read source"))
    monkeypatch.setattr(context, "decode_records", lambda *a: pytest.fail("chunk Activity decoded source"))
    params["work_products_ref"] = works_receipt["bundle_ref"]
    params["prepared_ref"] = context.prepare(params, segmenter=False)["bundle_ref"]
    prepared = context._read(params["prepared_ref"], params, "prepared")
    assert prepared["decoded_ref"] == params["decoded_ref"]
    assert prepared["records"] == context._read(params["decoded_ref"], params, "decoded")["records"]
    assert prepared["counts"]["chunks"] > 0
    for chunk in prepared["chunks"]:
        for segment in chunk["segments"]:
            assert segment["text"] == next(r["body"] for r in prepared["records"] if r["record_id"] == segment["record_id"])[segment["body_start"]:segment["body_end"]]


def test_typed_grounding_holds_invalid_results_without_inventing_fields():
    """Separate exact quote grounding from usable typed candidate coverage.

    Inputs: fictional segment and complete/incomplete model proposals. Outputs:
    typed/held distinctions, unchanged source spans and no fabricated dates.
    Effects: none; choose before the Go writer sees a derived proposal.
    """
    segment = {"record_id": "/0/text", "native_json_pointer": "/0/text", "role": "assistant", "native_time": None,
               "source_available_from": None, "body_start": 0, "text": "Alice reported a plan."}
    base = {"record_id": "/0/text", "quote": "Alice"}
    proposals = [{**base, "kind": "entity", "name": "Alice", "entity_type": "person", "confidence": 0.8},
                 {**base, "kind": "entity", "title": "Alice"},
                 {**base, "kind": "entity", "name": "Alice", "entity_type": "invented", "confidence": 0.8},
                 {**base, "kind": "event", "statement": "Reported event", "event_type": "reported", "occurred_at": "unknown", "confidence": 0.8},
                 {**base, "kind": "work_product", "confidence": 0.8},
                 {**base, "kind": "fact", "predicate": "reported", "statement": "A reported account", "confidence": 0.7},
                 {**base, "kind": "entity", "name": "Ghost", "entity_type": "person", "confidence": 0.8, "quote": "Ghost"}]
    accepted, rejected = context._ground({"segments": [segment]}, {"candidates": proposals}, typed=True)
    assert len(accepted) == 6 and rejected == 1
    assert sum(c["classification_status"] == "typed" for c in accepted) == 2
    assert sum(c["classification_status"] == "held" for c in accepted) == 4
    for c in accepted:
        assert c["occurrences"][0]["start"] == 0 and c["occurrences"][0]["end"] == 5
        if c["classification_status"] == "held":
            assert "confidence" not in c and c["reported"]
    assert accepted[3]["reported"]["occurred_at"] == "unknown" and "occurred_at" not in accepted[3]


def test_markdown_fallback_is_explicit_and_preserves_whole_document(tmp_path, monkeypatch):
    """Decode only the unsupported Markdown span shape through its named fallback.

    Inputs: fictional CRLF/Unicode transcript. Outputs: unchanged original and
    absolute turn spans; effects: temporary files. Pick for Gemini native text.
    """
    params, raw = _source(tmp_path, monkeypatch, "# Fictional\r\n**You:**\r\n😀 ask\r\n**Gemini:**\r\n answer \r\n", "gemini_markdown")
    receipt = context.decode(params)
    decoded = context._read(receipt["bundle_ref"], params, "decoded")
    assert decoded["decoder"]["backend"] == "native_markdown_spans"
    for record in decoded["records"]:
        assert raw.decode()[record["source_span"]["start"]:record["source_span"]["end"]] == record["body"]
        assert record["native_time"] is None


def test_gemini_exporter_headings_require_source_origin_and_preserve_spans(tmp_path, monkeypatch):
    """Decode Gemini exporter turns with original clocks and source slices intact.

    Inputs: fictional URL-backed CRLF export with a fenced role-like heading.
    Outputs: four exact turns, original byte identity and unknown timestamps.
    Effects: temporary files only; pick for the exporter Markdown variant.
    """
    text = ("# Fictional export\r\n[Original](https://gemini.google.com/app/abc123)\r\n"
            "## Prompt:\r\n😀 first ask\r\n## Response:\r\n"
            "```markdown\r\n## Prompt:\r\nquoted heading\r\n```\r\n"
            "## Prompt:\r\nsecond ask\r\n## Response:\r\n second answer \r\n")
    params, raw = _source(tmp_path, monkeypatch, text, "gemini_markdown")
    receipt = context.decode(params)
    decoded = context._read(receipt["bundle_ref"], params, "decoded")
    records = decoded["records"]
    assert [r["role"] for r in records] == ["user", "assistant", "user", "assistant"]
    assert "## Prompt:\r\nquoted heading" in records[1]["body"]
    assert records[-1]["body"] == " second answer \r\n"
    for record in records:
        span = record["source_span"]
        assert text[span["start"]:span["end"]] == record["body"]
        assert record["native_time"] is None and record["source_available_from"] is None
        assert record["native_message_id"] is None and record["conversation_id"] is None
    assert Path(decoded["original_ref"].removeprefix("file://")).read_bytes() == raw


@pytest.mark.parametrize("origin", ["", "https://chatgpt.com/c/abc123", "https://gemini.google.com.evil.example/app/abc123",
                                    "https://gemini.google.com/", "```\nhttps://gemini.google.com/app/abc123\n```"])
def test_exporter_headings_without_gemini_conversation_origin_are_rejected(origin):
    """Reject generic or unrelated Markdown headings as Gemini role evidence.

    Inputs: absent, unrelated, spoofed or fenced origin. Outputs: permanent
    rejection; effects: none. Pick to guard the narrow exporter compatibility.
    """
    with pytest.raises(context.ContextInvalid, match="Gemini conversation URL"):
        context.decode_markdown_records(origin + "\n## Prompt:\nask\n## Response:\nanswer\n", {})


def test_gemini_exporter_role_order_and_record_bound_are_enforced():
    """Reject ambiguous exporter role sequences and work beyond requested bounds.

    Inputs: URL-backed malformed sequences or three turns with a two-turn limit.
    Outputs: explicit permanent errors; effects: none. Pick for bounded decoding.
    """
    origin = "https://gemini.google.com/share/abc123\n"
    for body in ["## Response:\nanswer", "## Prompt:\nask\n## Prompt:\nask"]:
        with pytest.raises(context.ContextInvalid, match="alternate"):
            context.decode_markdown_records(origin + body, {})
    with pytest.raises(context.ContextInvalid, match="record bound"):
        context.decode_markdown_records(origin + "## Prompt:\na\n## Response:\nb\n## Prompt:\nc", {"max_records": 2})


def test_native_markdown_role_style_keeps_exporter_like_body_headings():
    """Keep exporter-like headings literal within the existing native role style.

    Inputs: explicit native roles and a Prompt heading inside the answer.
    Outputs: two unchanged turns; effects: none. Pick for replay compatibility.
    """
    records = context.decode_markdown_records("**You:**\nask\n**Gemini:**\n## Prompt:\nexample\n", {})
    assert len(records) == 2 and records[1]["body"] == "## Prompt:\nexample\n"


class _Store:
    """Hold additive synthetic search objects for independent readback.

    Inputs: expected objects. Outputs: exact readback. Effects: memory only;
    choose to exercise real publication code without a live search mutation.
    """
    collection = "AiChatEvents20260918"
    vector_name = "text_nim"

    def __init__(self):
        """Allocate a fresh memory store.

        Inputs: none. Output: store; effects: memory only. Pick per proof run.
        """
        self.objects = {}

    def get(self, key):
        """Read one exact object.

        Inputs: UUID. Output: object or none; effects: none. Pick for verification.
        """
        return self.objects.get(key)

    def add(self, obj):
        """Insert one absent object.

        Inputs: object. Output: none; effects: memory. Pick for additive proof.
        """
        assert obj["id"] not in self.objects
        self.objects[obj["id"]] = obj


def test_publication_precedes_enrichment_and_retains_actual_models_and_pins(tmp_path, monkeypatch):
    """Publish and verify all chunks without an optional candidate bundle.

    Inputs: real decoded files plus named embedding double and failed model.
    Outputs: stable search readback, actual model metadata and honest failure.
    Effects: synthetic files/memory only; never claims real model extraction.
    """
    params, _ = _source(tmp_path, monkeypatch, _chatgpt())
    params["decoded_ref"] = context.decode(params)["bundle_ref"]
    params["work_products_ref"] = context.extract_work_products(params)["bundle_ref"]
    prepared_receipt = context.prepare(params, segmenter=False)
    params["prepared_ref"] = prepared_receipt["bundle_ref"]
    embedder = SimpleNamespace(model="synthetic-explicit-embedding", calls=1,
                              embed=lambda texts: [[0.0] * 2048 for _ in texts])
    embedded = context.embed(params, embedder=embedder)
    assert embedded["model_id"] == embedder.model and embedded["model_calls"] == 1
    params["embeddings_ref"] = embedded["bundle_ref"]
    store = _Store()
    published = context.publish(params, store=store)
    params["publication_ref"] = published["bundle_ref"]
    verified = context.verify_publication(params, store=store)
    assert published["objects_written"] == verified["objects_verified"] == prepared_receipt["chunks"]
    assert published["status"] == "complete"
    publication = context._read(params["publication_ref"], params, "published")
    assert publication["candidate_status"] == "not_run" and "candidates_ref" not in publication
    before = json.dumps(store.objects, sort_keys=True)
    for obj in store.objects.values():
        props = obj["properties"]
        assert props["embed_model"] == embedder.model
        assert props["source_version_id"] == params["source_version_id"]
        provenance = json.loads(props["provenance"][0])
        assert provenance["historical_source_version_id"] == params["historical_source_version_id"]
        assert provenance["workflow_id"] == params["workflow_id"]
        assert provenance["temporal_run_id"] == params["temporal_run_id"]
        assert "source_version_ids" not in props
        assert provenance["source_sha256"] == params["source_sha256"]
        assert provenance["enrichment_status"] == "not_run" and "candidates_ref" not in provenance
    model = SimpleNamespace(model_id="synthetic-failing-model")
    def fail(prompt):
        """Simulate provider failure after core publication.

        Inputs: synthetic prompt. Output: exception; effects: none. Pick to
        prove optional extraction cannot alter already verified search objects.
        """
        raise RuntimeError("synthetic offline")
    fail.model_id = model.model_id
    extracted = context.extract_candidates(params, model=fail)
    assert extracted["status"] == "partial_enrichment" and extracted["model_id"] == model.model_id
    params["candidates_ref"] = extracted["bundle_ref"]
    assert context.verify_publication(params, store=store)["objects_verified"] == verified["objects_verified"]
    assert context.publish(params, store=store) == published
    assert json.dumps(store.objects, sort_keys=True) == before


def test_real_candidate_unit_reports_grounded_typed_held_and_model_coverage(tmp_path, monkeypatch):
    """Run actual candidate grounding with a deterministic fictional model reply.

    Inputs: real decoded/prepared files and an explicitly simulated typed reply.
    Outputs: six usable typed proposals, two held and one ungrounded rejection;
    effects: temporary derivatives only, no real model or PostgreSQL calls.
    """
    params, _ = _source(tmp_path, monkeypatch, _chatgpt())
    params["decoded_ref"] = context.decode(params)["bundle_ref"]
    params["work_products_ref"] = context.extract_work_products(params)["bundle_ref"]
    params["prepared_ref"] = context.prepare(params, segmenter=False)["bundle_ref"]
    prepared = context._read(params["prepared_ref"], params, "prepared")
    record = prepared["records"][0]
    base = {"record_id": record["record_id"], "quote": "Alice", "confidence": 0.8}
    proposals = [{**base, "kind": "entity", "name": "Alice", "entity_type": "person"},
                 {**base, "kind": "event", "quote": "reported a plan on 2024-01-02", "statement": "A reported plan", "event_type": "reported_plan", "occurred_at": "2024-01-02T00:00:00Z"},
                 {**base, "kind": "event", "statement": "An undated account", "event_type": "reported", "occurred_at": None},
                 {**base, "kind": "fact", "statement": "A reported account", "predicate": "reported"},
                 {**base, "kind": "strategy", "statement": "Reported strategy account"},
                 {**base, "kind": "history", "statement": "Reported historical account"},
                 {**base, "kind": "work_product"},
                 {**base, "kind": "entity", "name": "Alice", "entity_type": "person", "confidence": "high"},
                 {**base, "kind": "entity", "name": "Ghost", "entity_type": "person", "quote": "Ghost"}]

    def model(prompt):
        """Return one explicit simulated reply after checking the typed prompt.

        Inputs: prompt. Output: fictional structured proposals; effects: none.
        Pick to verify complete schemas/counts without claiming real extraction.
        """
        assert '"entity_type":"person","confidence":0.8' in prompt
        assert "Strategy/history require their original kind" in prompt
        return {"candidates": proposals}
    model.model_id = "synthetic-typed-model"
    result = context.extract_candidates(params, model=model)
    assert result["model_id"] == "synthetic-typed-model" and result["model_calls"] == 1
    assert result["grounded_candidates"] == result["candidates"] == 8
    assert result["typed_candidates"] == 6 and result["held_candidates"] == 2 and result["rejected_candidates"] == 1
    assert result["status"] == "partial_enrichment"
    bundle = context._read(result["bundle_ref"], params, "candidates")
    assert all(c["occurrences"][0]["source_available_from"] == record["source_available_from"] for c in bundle["candidates"])
    assert bundle["candidates"][2]["occurred_at"] is None


def test_nonfinite_model_json_and_optional_typed_field_corruption_fail_closed():
    """Reject invalid model JSON and hold malformed optional bridge fields.

    Inputs: fictional provider values. Outputs: bounded error or held candidate.
    Effects: none; pick to prevent one malformed value breaking all Go staging.
    """
    chunk = {"segments": [{"record_id": "/text", "native_json_pointer": "/text", "role": "assistant",
                           "native_time": None, "text": "Alice", "body_start": 0}]}
    with pytest.raises(context.ContextInvalid, match="nonfinite"):
        context._model_reply(chunk, lambda prompt: '{"candidates":[],"invalid":NaN}')
    candidate = {"kind": "entity", "record_id": "/text", "quote": "Alice", "name": "Alice",
                 "entity_type": "person", "confidence": 0.8, "statement": ["invalid string shape"]}
    accepted, rejected = context._ground(chunk, {"candidates": [candidate]}, typed=True)
    assert rejected == 0 and accepted[0]["classification_status"] == "held"
    assert "confidence" not in accepted[0] and "statement" not in accepted[0]
