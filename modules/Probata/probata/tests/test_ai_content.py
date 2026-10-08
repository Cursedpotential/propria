"""Verify bounded AI content stages with retained synthetic source occurrences.

Inputs: VPS-only AI_CONTENT_TEST_ROOT and synthetic fixtures. Outputs: assertions
and retained bundles. Effects: files under a unique retained directory, fake
provider/search calls; no canonical writes or fixture deletion. Choose for the
five independent AI content stages, not native parsing or human chunk regressions.
Byline: Codex / GPT-6.1-Sol / 2026-10-06.
"""
from __future__ import annotations

from contextlib import contextmanager
from copy import deepcopy
import json
import hashlib
import os
from pathlib import Path
from types import SimpleNamespace
from uuid import uuid4

import httpx
import pytest

from server.analysis import ai_content as ai


def test_native_selected_path_preserves_pointer_and_unknown_time():
    """Use the selected ChatGPT branch and retain native source coordinates.

    Inputs: branching synthetic native export. Outputs: path and pointer checks.
    Effects: none; choose for source-only decoder regression coverage.
    """
    source = {"source_version_id": str(uuid4()), "original_object_id": str(uuid4()),
              "original_sha256": "a" * 64, "version_id": None}
    native = [{"id": "chat", "current_node": "chosen", "mapping": {
        "root": {"parent": None, "message": None},
        "other": {"parent": "root", "message": {"author": {"role": "assistant"},
                  "content": {"parts": ["abandoned branch"]}}},
        "chosen": {"parent": "root", "message": {"author": {"role": "assistant"},
                   "content": {"parts": ["selected text"]}}}}}]
    records = ai.decode_native_conversations(native, source, 10)
    assert len(records) == 1 and records[0]["body"] == "selected text"
    assert records[0]["native_json_pointer"] == "/0/mapping/chosen/message/content/parts/0"
    assert records[0]["occurred_at"] is None and records[0]["version_id"] is None
    assert records[0]["span_unit"] == "unicode_codepoint"


def test_claude_native_pointer_and_invalid_quote_fail_closed():
    """Verify Claude text pointers and refuse altered quote/span/pointer values.

    Inputs: synthetic Claude export. Outputs: exact citations or ContentInvalid.
    Effects: none; choose for cross-provider source pointer integrity.
    """
    source = {"source_version_id": str(uuid4()), "original_object_id": str(uuid4()),
              "original_sha256": "a" * 64, "version_id": None}
    native = [{"chat_messages": [{"sender": "assistant", "content": [
        {"type": "text", "text": "Résumé of work"}]}]}]
    records = ai.decode_native_conversations(native, source, 10)
    row = records[0]
    assert row["native_json_pointer"] == "/0/chat_messages/0/content/0/text"
    ai.verify_native_citation(native, row, "Résumé of work")
    with pytest.raises(ai.ContentInvalid):
        ai.verify_native_citation(native, row, "Resume of work")
    with pytest.raises(ai.ContentInvalid):
        ai.verify_native_citation(native, {**row, "native_json_pointer": "/0/chat_messages/9/text"}, row["body"])
    bad_span = {**row, "source_span": {**row["source_span"], "sha256": "0" * 64}}
    with pytest.raises(ai.ContentInvalid):
        ai.verify_native_citation(native, bad_span, row["body"])


def test_inline_original_requires_exact_bytes_and_sha256():
    """Reject a retained inline object whose full source hash changes.

    Inputs: synthetic inline JSON and fake read-only row. Outputs: parsed source
    or permanent mismatch. Effects: none; choose for retained-source gate proof.
    """
    body = b'[{"chat_messages":[{"sender":"assistant","text":"Hello"}]}]'

    class Result:
        """Expose one inline byte value as a read-only scalar result.

        Inputs: fixture bytes. Outputs: scalar. Effects: none.
        """
        def scalar_one(self):
            """Return the retained fixture bytes without persistence.

            Inputs: none. Outputs: source bytes. Effects: none.
            """
            return body

    class Connection:
        """Resolve the retained inline byte SELECT for one fixture.

        Inputs: source query. Outputs: scalar result. Effects: none.
        """
        def execute(self, statement, params):
            """Return bytes for the exact original-object query.

            Inputs: query and object id. Outputs: result. Effects: none.
            """
            assert "inline_bytes" in str(statement) and params["id"] == "object-id"
            return Result()

    source = {"storage_class": "inline", "original_object_id": "object-id",
              "original_bytes": len(body), "original_sha256": hashlib.sha256(body).hexdigest()}
    assert ai.read_verified_original(Connection(), source)[0]["chat_messages"][0]["text"] == "Hello"
    with pytest.raises(ai.ContentInvalid):
        ai.read_verified_original(Connection(), {**source, "original_sha256": "0" * 64})


def test_cached_preparation_rehashes_retained_original(scope, monkeypatch):
    """Reject a changed sealed original even when a prepared bundle already exists.

    Inputs: mutable inline fixture with stable metadata. Outputs: a first bundle
    and a retry refusal. Effects: retained synthetic file; choose for cache custody.
    """
    from server.context_chunks import db, chunker
    original = b'[{"chat_messages":[{"sender":"assistant","text":"Hello"}]}]'
    current = [original]
    bound = {**source(scope), "original_bytes": len(original), "original_sha256": hashlib.sha256(original).hexdigest()}
    monkeypatch.setattr(ai, "source_binding", lambda conn, pin: bound)
    monkeypatch.setattr(chunker, "neural_text_spans", lambda text, **_: [(0, len(text))])

    class Connection:
        """Return current inline source bytes for each exact object read."""
        def execute(self, statement, params):
            """Expose an immutable-looking object id with mutable test bytes."""
            assert params["id"] == bound["original_object_id"]
            return SimpleNamespace(scalar_one=lambda: current[0])

    @contextmanager
    def connection():
        """Yield the fixture connection without any production DB access."""
        yield Connection()

    monkeypatch.setattr(db, "read_only_connection", connection)
    with pytest.raises(ai.ContentInvalid, match="native source-only"):
        ai.prepare_content({**scope, "original_ref": str(uuid4())})
    with pytest.raises(ai.ContentInvalid, match="native source-only"):
        ai.prepare_content({**scope, "native_source_only": False})
    first = ai.prepare_content(scope)
    assert first["bundle_sha256"] == hashlib.sha256(Path(first["bundle_ref"].removeprefix("file://")).read_bytes()).hexdigest()
    current[0] = original.replace(b"Hello", b"Jello")
    with pytest.raises(ai.ContentInvalid, match="SHA256 differs"):
        ai.prepare_content(scope)


def test_grounded_candidate_rejects_nonfinite_confidence():
    """Refuse a nonfinite model score before retaining candidate JSON.

    Inputs: one exact native segment and NaN confidence. Outputs: permanent
    validation error. Effects: none; choose for finite candidate bundle proof.
    """
    row = {"record_id": "r", "body": "Exact quote", "conversation_index": 0,
           "source_version_id": str(uuid4()), "source_object_id": str(uuid4()),
           "source_sha256": "a" * 64, "version_id": None,
           "native_json_pointer": "/0/chat_messages/0/text", "span_unit": "unicode_codepoint"}
    chunk = {"conversation_index": "0", "conversation_id": None, "chunk_index": 0,
             "content_key": "synthetic", "segments": [{**row, "text": row["body"],
                 "body_start": 0, "body_end": len(row["body"])}]}
    with pytest.raises(ai.ContentInvalid, match="confidence"):
        ai.ground_candidates(chunk, {"candidates": [{"kind": "event", "title": "Event",
            "record_id": "r", "quote": "Exact quote", "confidence": float("nan")}]})


def test_grounded_candidate_emits_narrow_repeated_native_spans():
    """Emit a candidate with exact per-occurrence native quote coordinates.

    Inputs: repeated quote in one native text field. Outputs: two codepoint
    citations plus top-level span unit for the Go review bridge. Effects: none.
    """
    body = "Alice met Alice"
    row = {"record_id": "r", "conversation_index": 0,
           "source_version_id": str(uuid4()), "source_object_id": str(uuid4()),
           "source_sha256": "a" * 64, "version_id": None,
           "native_json_pointer": "/0/chat_messages/0/text", "span_unit": "unicode_codepoint"}
    chunk = {"conversation_index": "0", "conversation_id": None, "chunk_index": 0,
             "content_key": "synthetic", "segments": [{**row, "text": body,
                 "body_start": 0, "body_end": len(body)}]}
    candidate = ai.ground_candidates(chunk, {"candidates": [{"kind": "entity", "title": "Alice",
        "record_id": "r", "quote": "Alice"}]})[0]
    assert candidate["span_unit"] == "unicode_codepoint"
    assert candidate["evidence_quote"] == "Alice"
    assert [part["source_span"]["start"] for part in candidate["occurrences"]] == [0, 10]
    assert all(part["evidence_quote"] == "Alice" and part["span_unit"] == "unicode_codepoint"
               for part in candidate["occurrences"])


def test_created_work_crosses_native_turns_and_saves_openable_file(scope):
    """Keep one complete email draft and both native source slices.

    Inputs: two synthetic assistant turns. Outputs: one retained plain text file
    and two pointer spans. Effects: files in the retained VPS test scope only.
    """
    source = {"source_version_id": scope["source_version_id"], "original_object_id": str(uuid4()),
              "original_sha256": "b" * 64, "version_id": None}
    native = [{"chat_messages": [
        {"sender": "assistant", "text": "Subject: Status\nHello,\n"},
        {"sender": "assistant", "text": "The work is complete.\nRegards,\nMatt"}]}]
    records = ai.decode_native_conversations(native, source, 10)
    products = ai.conversation_created_works(records)
    assert len(products) == 1 and len(products[0]["source_segments"]) == 2
    assert all(segment["span_unit"] == "unicode_codepoint" for segment in products[0]["source_segments"])
    saved = ai._save_created_work(ai.pins(scope), products[0])
    assert Path(saved["file_ref"].replace("file://", "")).read_text(encoding="utf-8") == products[0]["content"]
    assert saved["case_vault_relative_path"].startswith("created-works/0/")
    assert ai._save_created_work(ai.pins(scope), products[0]) == saved


def test_prepared_topic_segments_reassemble_exact_native_slices(monkeypatch):
    """Reject a source slice that changes while the displayed topic text stays intact.

    Inputs: two native fields and one synthetic topic span. Outputs: exact
    reassembly or ContentInvalid. Effects: none; choose for native span custody.
    """
    from server.context_chunks import chunker

    records = [record(0, "Résumé begins."), record(1, "The second turn continues.")]
    monkeypatch.setattr(chunker, "neural_text_spans", lambda text, **_: [(0, len(text))])
    chunks = ai.conversation_windows(records, 2)
    ai.validate_prepared_chunks(records, chunks)
    changed = deepcopy(chunks)
    changed[0]["segments"][1]["body_start"] += 1
    with pytest.raises(ai.ContentInvalid, match="reassemble"):
        ai.validate_prepared_chunks(records, changed)
    changed = deepcopy(chunks)
    changed[0]["segments"][0]["source_sha256"] = "0" * 64
    with pytest.raises(ai.ContentInvalid, match="differs from its native source"):
        ai.validate_prepared_chunks(records, changed)
    changed = deepcopy(chunks)
    changed[0]["segments"][0]["occurred_at"] = "2026-01-01T00:00:00Z"
    with pytest.raises(ai.ContentInvalid, match="differs from its native source"):
        ai.validate_prepared_chunks(records, changed)


def test_neural_text_spans_cover_long_single_turn_and_late_topic():
    """Check exact reconstruction and late cuts with a bounded fake chunker.

    Inputs: long synthetic text and an offset-compatible model stub. Outputs:
    complete adjacent spans. Effects: none; choose before remote real-model proof.
    """
    from server.context_chunks.chunker import neural_text_spans

    class Stub:
        """Return one mid-window topic cut with Chonkie's start_index shape.

        Inputs: bounded text. Outputs: synthetic chunks. Effects: none.
        """
        def chunk(self, text):
            """Expose native Chonkie-compatible offset attributes for span tests.

            Inputs: window. Outputs: chunk starts. Effects: none.
            """
            return [SimpleNamespace(start_index=0), SimpleNamespace(start_index=len(text) // 2)]

    body = "alpha " * 1700 + "late topic " * 250
    spans = neural_text_spans(body, max_chars=3000, neural=Stub())
    assert spans[0][0] == 0 and spans[-1][1] == len(body)
    assert all(left < right and right - left <= 3000 for left, right in spans)
    assert all(spans[i][1] == spans[i + 1][0] for i in range(len(spans) - 1))
    assert "".join(body[left:right] for left, right in spans) == body
    assert any(left > 8000 for left, _ in spans)


def test_neural_text_spans_compact_three_character_model_fragment():
    """Merge tiny adjacent model cuts while retaining exact source coverage.

    Inputs: Chonkie-compatible synthetic starts three characters apart.
    Outputs: useful adjacent spans without a three-character search object.
    Effects: none; choose for the real excerpt's observed tiny-cut regression.
    """
    from server.context_chunks.chunker import neural_text_spans

    class TinyCut:
        """Expose two adjacent Chonkie-compatible topic starts.

        Inputs: bounded window. Outputs: chunk offsets. Effects: none.
        """
        def chunk(self, text):
            """Return a short middle span in the first window.

            Inputs: text. Outputs: offset objects. Effects: none.
            """
            return [SimpleNamespace(start_index=0), SimpleNamespace(start_index=300),
                    SimpleNamespace(start_index=303)]

    body = "x" * 1200
    spans = neural_text_spans(body, max_chars=1500, neural=TinyCut())
    assert "".join(body[a:b] for a, b in spans) == body
    assert all(b - a >= 80 for a, b in spans)


def test_real_chonkie_neural_offsets_on_late_topic(scope):
    """Check real Chonkie chunk offsets and late-turn reconstruction on the VPS.

    Inputs: synthetic long turn and installed NeuralChunker 1.7. Outputs: exact
    text coverage and a model-origin cut beyond the first window. Effects:
    model inference only in the approved remote runtime; no provider call.
    """
    from server.context_chunks.chunker import _built, neural_text_spans

    text = ("The garden inventory lists trees, soil and watering dates. " * 180 +
            "The court filing discusses exhibits, service, and hearing dates. " * 180)
    neural = _built("neural_distilbert").neural
    direct = neural.chunk(text[:2500])
    assert direct and all(hasattr(part, "start_index") for part in direct)
    spans = neural_text_spans(text, max_chars=3000, neural=neural)
    assert spans[0][0] == 0 and spans[-1][1] == len(text)
    assert "".join(text[left:right] for left, right in spans) == text
    assert all(spans[i][1] == spans[i + 1][0] for i in range(len(spans) - 1))
    assert any(left > 8000 for left, _ in spans)


@pytest.fixture
def scope(monkeypatch):
    """Create an independently retained synthetic scope without temporary cleanup.

    Inputs: explicit VPS test root. Outputs: common pins. Effects: a unique
    retained directory and scoped environment; choose over auto-deleted TempDir.
    """
    root = Path(os.environ["AI_CONTENT_TEST_ROOT"]) / uuid4().hex
    root.mkdir(parents=True)
    monkeypatch.setenv("AI_CONTENT_ROOT", str(root))
    return {"request_id": "synthetic-ai-content-" + uuid4().hex,
            **{key: str(uuid4()) for key in ("source_version_id", "matter_id", "court_case_id")},
            "operating_mode": "LIVE", "original_ref": str(uuid4()), "native_source_only": True}


def record(ordinal, body, *, conversation=0, native_id="native-conversation"):
    """Build a synthetic native text field with exact source locators.

    Inputs: ordinal/body/native conversation. Outputs: record. Effects: none;
    choose for source-span and conversation-boundary assertions.
    """
    return {"record_id": str(uuid4()), "ordinal": ordinal, "body": body,
            "conversation_index": conversation, "conversation_id": native_id,
            "native_message_id": "native-node-" + str(ordinal), "mapping_key": "slot-" + str(ordinal),
            "native_message_index": None, "conversation_title": None, "role": "assistant",
            "occurred_at": None, "source_version_id": str(uuid4()), "source_object_id": str(uuid4()),
            "version_id": None, "source_sha256": "a" * 64,
            "native_json_pointer": f"/0/mapping/slot-{ordinal}/message/content/parts/0",
            "span_unit": "unicode_codepoint",
            "source_span": {"start": 0, "end": len(body), "sha256": hashlib.sha256(body.encode()).hexdigest(),
                            "unit": "unicode_codepoint"}}


def source(scope):
    """Bind a synthetic retained original to the exact common scope.

    Inputs: scope. Outputs: metadata binding. Effects: none; choose instead of
    creating real source rows for pure provider/publication tests.
    """
    return {"source_version_id": scope["source_version_id"], "original_object_id": scope["original_ref"],
            "source_key": "inline://synthetic/conversations.json", "storage_class": "inline",
            "version_id": None, "original_sha256": "a" * 64,
            "original_bytes": 476547, "format_id": "chatgpt_json_array"}


def prepared(scope, monkeypatch, records=None):
    """Retain one synthetic prepared bundle with complete source occurrences.

    Inputs: scope and optional records. Outputs: reference/bundle. Effects:
    retained files and fake read-only binding; choose for downstream stage tests.
    """
    records = records or [record(0, "Alice drafted the motion."), record(1, "The hearing is on 2026-10-07.")]
    bound = source(scope)
    monkeypatch.setattr(ai, "_bound_source", lambda _: bound)
    from server.context_chunks import chunker
    monkeypatch.setattr(chunker, "neural_text_spans", lambda text, **_: [(0, len(text))])
    chunks = ai.conversation_windows(records, 128)
    data = {"source": bound, "method": ai.METHOD, "records": records, "chunks": chunks,
            "counts": {"records": len(records), "conversations": len({r["conversation_index"] for r in records}), "chunks": len(chunks)}}
    ref = ai._save(ai._path(scope, "prepared", "fixture"), scope, "prepared", data)
    return ref, ai._read(ref, scope, "prepared")


def dependencies(scope, monkeypatch):
    """Retain the three exact linked prerequisites without remote side effects.

    Inputs: scope. Outputs: publication request/bundles. Effects: retained files;
    choose for independent mutation and readback boundary tests.
    """
    ref, prep = prepared(scope, monkeypatch)
    product_ref = ai.extract_work_products({**scope, "prepared_ref": ref})["bundle_ref"]
    candidate = {"source": prep["source"], "prepared_ref": ref, "work_products_ref": product_ref, "model_id": "moonshotai/kimi-k3", "candidates": []}
    candidate_ref = ai._save(ai._path(scope, "candidates", "fixture"), scope, "candidates", candidate)
    embedded = {"source": prep["source"], "prepared_ref": ref, "model_id": "nvidia/nemotron-3-embed-1b",
                "vectors": [[0.25] * 2048 for _ in prep["chunks"]]}
    embed_ref = ai._save(ai._path(scope, "embedded", "fixture"), scope, "embedded", embedded)
    return {**scope, "prepared_ref": ref, "work_products_ref": product_ref, "candidates_ref": candidate_ref, "embeddings_ref": embed_ref}, prep, candidate, embedded


class MemoryStore:
    """Model the existing additive REST contract with visible read/write counts.

    Inputs: test objects. Outputs: deep-copied reads. Effects: in-memory only;
    choose for immutable collision and independent verification tests.
    """
    base, collection, vector_name = "http://synthetic", "AiChatEvents20260918", "text_nim"

    def __init__(self):
        """Initialize empty synthetic search state.

        Inputs: none. Outputs: store. Effects: memory allocation; choose for tests.
        """
        self.objects, self.adds, self.reads = {}, 0, 0

    def schema(self):
        """Accept the known fixture schema.

        Inputs: none. Outputs: none. Effects: none; REST schema has separate tests.
        """

    def get(self, key):
        """Read independent object bytes from fixture state.

        Inputs: UUID. Outputs: copied object or absence. Effects: read counter.
        """
        self.reads += 1
        return deepcopy(self.objects.get(key))

    def add(self, value):
        """Add an absent fixture object and forbid replacement.

        Inputs: object. Outputs: none. Effects: state/count; choose over upsert.
        """
        assert value["id"] not in self.objects
        self.adds += 1
        self.objects[value["id"]] = deepcopy(value)


def test_windows_preserve_every_text_occurrence_without_per_message_objects(scope, monkeypatch):
    """Prove complete source reconstruction and coherent grouping; input fixture scope, output assertions, no external effects; choose over object-count-only tests."""
    from server.context_chunks import chunker
    monkeypatch.setattr(chunker, "neural_text_spans", lambda text, **_: [(start, min(start + ai.CHUNK_CHARS, len(text))) for start in range(0, len(text), ai.CHUNK_CHARS)])
    rows = [record(0, "alpha\n\n" + "x" * 15000), record(1, "alpha alpha"), record(2, ""), record(3, "other", conversation=1)]
    chunks = ai.conversation_windows(rows, 128)
    assert len(chunks) >= 3
    for row in rows:
        segments = [s for c in chunks for s in c["segments"] if s["record_id"] == row["record_id"]]
        assert "".join(s["text"] for s in segments) == row["body"]
        assert all(s["text"] == row["body"][s["body_start"]:s["body_end"]] for s in segments)
    assert all(len(c["text"]) <= ai.CHUNK_CHARS for c in chunks)
    short = ai.conversation_windows([record(i, "short text") for i in range(132)], 128)
    assert len(short) == 1
    assert len(short[0]["segments"]) == 132


def test_chunk_bound_missing_native_coordinate_and_empty_only_fail(scope, monkeypatch):
    """Reject incomplete grouping and exceeded bounds; input fixture scope, output assertions, no external effects; choose for fail-closed admission."""
    from server.context_chunks import chunker
    monkeypatch.setattr(chunker, "neural_text_spans", lambda text, **_: [(start, min(start + ai.CHUNK_CHARS, len(text))) for start in range(0, len(text), ai.CHUNK_CHARS)])
    for rows, maximum in [([record(0, "x" * 15000)], 1), ([{**record(0, "x"), "conversation_index": None}], 128), ([record(0, "")], 128)]:
        with pytest.raises(ai.ContentInvalid):
            ai.conversation_windows(rows, maximum)


def test_exact_grounding_preserves_repeated_and_overlapping_occurrences(scope, monkeypatch):
    """Prove exact repeated quote locators; input synthetic source, output assertions, no external effects; choose over deduplicated candidate checks."""
    from server.context_chunks import chunker
    monkeypatch.setattr(chunker, "neural_text_spans", lambda text, **_: [(0, len(text))])
    row = record(7, "aaaa; Alice drafted a motion; Alice drafted a motion")
    chunk = ai.conversation_windows([row], 128)[0]
    candidates = ai.ground_candidates(chunk, {"candidates": [{"kind": "artifact", "title": "Motion", "record_id": row["record_id"], "quote": "Alice drafted a motion"},
        {"kind": "entity", "title": "Repeated letters", "record_id": row["record_id"], "quote": "aa"}]})
    assert [len(c["occurrences"]) for c in candidates] == [2, 3]
    for candidate in candidates:
        for occurrence in candidate["occurrences"]:
            assert row["body"][occurrence["body_start"]:occurrence["body_end"]] == candidate["quote"]
            assert occurrence["mapping_key"] == "slot-7" and occurrence["native_json_pointer"] == row["native_json_pointer"]
            assert occurrence["occurred_at"] is None


@pytest.mark.parametrize("quote,record_id", [("fabricated", None), ("Alice", "wrong-record"), ("", None)])
def test_ungrounded_candidates_fail_closed(scope, monkeypatch, quote, record_id):
    """Reject fabricated or misbound quotes; inputs fixture scope and quote, output assertions, no external effects; choose for candidate grounding."""
    from server.context_chunks import chunker
    monkeypatch.setattr(chunker, "neural_text_spans", lambda text, **_: [(0, len(text))])
    row = record(0, "Alice")
    chunk = ai.conversation_windows([row], 128)[0]
    with pytest.raises(ai.ContentInvalid):
        ai.ground_candidates(chunk, {"candidates": [{"kind": "entity", "quote": quote, "record_id": record_id or row["record_id"]}]})


def test_immutable_ref_pins_stage_and_tampering(scope):
    """Prove immutable pin-bound files; input retained fixture scope, output assertions, effects retained test files; choose over mutable cache tests."""
    path = ai._path(scope, "prepared", "same")
    ref = ai._save(path, scope, "prepared", {"value": 1})
    assert ai._save(path, scope, "prepared", {"value": 1}) == ref
    with pytest.raises(ai.ContentInvalid):
        ai._save(path, scope, "prepared", {"value": 2})
    for wrong, stage in [({**scope, "source_version_id": str(uuid4())}, "prepared"), (scope, "embedded")]:
        with pytest.raises(ai.ContentInvalid):
            ai._read(ref, wrong, stage)
    path.chmod(0o600)
    path.write_text('{"changed":true}', encoding="utf-8")
    with pytest.raises(ai.ContentInvalid):
        ai._read(ref, scope, "prepared")


def test_extract_all_categories_exact_quotes_and_model_bound_cache(scope, monkeypatch):
    """Prove categories and model-bound cache; inputs fixture and fake provider, output assertions, effects retained files; choose for extraction contracts."""
    ref, prep = prepared(scope, monkeypatch)
    calls = []

    class Model:
        """Return a bounded deterministic synthetic remote result.

        Inputs: prompt. Outputs: provider-compatible scores. Effects: call log;
        choose instead of real inference in contract tests.
        """
        def infer(self, prompts, **kwargs):
            """Emit source-grounded candidates for every required category.

            Inputs: prompts/config. Outputs: scored JSON. Effects: records call.
            """
            calls.extend(prompts)
            value = {"candidates": [{"kind": kind, "title": kind, "record_id": prep["records"][0]["record_id"], "quote": "Alice"} for kind in sorted(ai.KINDS)]}
            yield [SimpleNamespace(output=json.dumps(value))]

    config = SimpleNamespace(base_url="https://configured.example/v1", model_id="moonshotai/kimi-k3", max_tokens=6000)
    params = {**scope, "prepared_ref": ref, "work_products_ref": ai.extract_work_products({**scope, "prepared_ref": ref})["bundle_ref"]}
    result = ai.extract_candidates(params, model=Model(), config=config)
    bundle = ai._read(result["bundle_ref"], scope, "candidates")
    assert {c["kind"] for c in bundle["candidates"]} == ai.KINDS
    assert result["model_calls"] == 1
    assert ai.extract_candidates(params, model=Model(), config=config)["bundle_ref"] == result["bundle_ref"]
    assert len(calls) == 1
    changed = SimpleNamespace(**{**vars(config), "model_id": "configured-next-model"})
    assert ai.extract_candidates(params, model=Model(), config=changed)["bundle_ref"] != result["bundle_ref"]
    with pytest.raises(ai.ContentInvalid):
        ai.extract_candidates({**params, "max_model_calls": 1}, model=Model(), config=config)


def test_embed_remote_only_exact_count_finite_and_cache(scope, monkeypatch):
    """Prove retained vector count and cache; inputs fixture and fake embedder, output assertions, effects retained files; choose for embedding contracts."""
    ref, prep = prepared(scope, monkeypatch)
    config = SimpleNamespace(embed_base_url="https://configured.example/v1", embed_model="nvidia/nemotron-3-embed-1b")
    calls = []

    class Embedder:
        """Return synthetic provider vectors without local inference.

        Inputs: strings. Outputs: vectors. Effects: call log; choose for tests.
        """
        calls = 1
        def embed(self, texts):
            """Embed each supplied fixture string as a complete named vector.

            Inputs: text array. Outputs: 2048 values each. Effects: fixture log.
            """
            calls.append(texts)
            return [[0.1] * 2048 for _ in texts]

    params = {**scope, "prepared_ref": ref}
    result = ai.embed_content(params, embedder=Embedder(), config=config)
    assert result["chunks"] == len(prep["chunks"]) and result["embed_requests"] == 1
    assert ai.embed_content(params, embedder=Embedder(), config=config)["bundle_ref"] == result["bundle_ref"]
    assert len(calls) == 1
    assert ai._read(result["bundle_ref"], scope, "embedded")["vectors"][0][0] != 0.1  # persisted float32 representation
    for vector in ([float("nan")] * 2048, [0.25] * 2047):
        bad_config = SimpleNamespace(**{**vars(config), "embed_model": "invalid-fixture-" + uuid4().hex})
        with pytest.raises(ai.ContentInvalid):
            ai.embed_content(params, embedder=SimpleNamespace(calls=1, embed=lambda _: [vector]), config=bad_config)


def test_publication_and_fresh_readback_preserve_retained_source_and_no_turn_objects(scope, monkeypatch):
    """Prove additive conversation publication and fresh full readback; inputs fixture/store, output assertions, effects retained files; choose for publication integrity."""
    params, prep, candidates, embedded = dependencies(scope, monkeypatch)
    from server.temporal import chunk_write_guard
    admitted = []
    monkeypatch.setattr(chunk_write_guard, "require_live_chunk_write", lambda *p: admitted.append(p))
    store = MemoryStore()
    result = ai.publish_content(params, store=store)
    assert result["stage"] == "published" and result["objects_written"] == result["chunks"] == 1
    assert result["records"] == 2 and store.adds == 1 and admitted
    assert ai.publish_content(params, store=store)["bundle_ref"] == result["bundle_ref"] and store.adds == 1
    obj = next(iter(store.objects.values()))
    assert obj["properties"]["record_kind"] == "ai_conversation_chunk"
    assert not {"turn_index", "pg_row_id", "vault_key", "sha1", "catalog_path"} & obj["properties"].keys()
    citation = json.loads(obj["properties"]["provenance"][0])
    assert citation["source"]["original_sha256"] == "a" * 64
    assert citation["candidates_ref"] == params["candidates_ref"] and citation["pins"] == scope
    assert citation["method"] == ai.METHOD and len(citation["segments"]) == 2
    before = store.reads
    verify = {**params, "publication_ref": result["bundle_ref"]}
    verified = ai.verify_publication(verify, store=store)
    assert verified["stage"] == "verified" and verified["objects_verified"] == 1 and store.reads > before
    obj["vectors"]["text_nim"][0] = 0.75
    with pytest.raises(ai.ContentInvalid):
        ai.verify_publication(verify, store=store)
    with pytest.raises(ai.ContentInvalid):
        ai.publish_content(params, store=store)
    assert store.adds == 1


def test_publication_denied_before_bundle_or_search_read(scope, monkeypatch):
    """Prove LIVE authority precedes content access; input denied fixture scope, output assertions, no external effects; choose for write fence order."""
    from server.temporal import chunk_write_guard
    monkeypatch.setattr(ai, "publication_inputs", lambda _: pytest.fail("read content before authority"))
    def denied(*_):
        """Reject fixture authority before any content access.

        Inputs: scope. Outputs: exception. Effects: none; tests write fence order.
        """
        raise ai.ContentInvalid("fixture authority denied")
    monkeypatch.setattr(chunk_write_guard, "require_live_chunk_write", denied)
    with pytest.raises(ai.ContentInvalid):
        ai.publish_content(scope)


def test_scope_dev_nil_and_mismatched_source_rejected(scope, monkeypatch):
    """Reject mismatched scope before downstream use; inputs invalid pins, output assertions, effects retained fixtures; choose for independent admission."""
    with pytest.raises(ai.ContentInvalid):
        ai.pins({**scope, "operating_mode": "DEV"})
    with pytest.raises(ai.ContentInvalid):
        ai.pins({**scope, "source_version_id": "00000000-0000-0000-0000-000000000000"})
    params, _, _, _ = dependencies(scope, monkeypatch)
    with pytest.raises(ai.ContentInvalid):
        ai.publication_inputs({**params, "source_version_id": str(uuid4())})


def test_rest_schema_read_only_incompatible_type_and_add_contract(scope):
    """Prove REST schema and additive collision contracts; input mocked HTTP, output assertions, no network effects; choose for client serialization."""
    calls, objects = [], {}
    schema = {"properties": [{"name": n, "dataType": [t],
                              **({"indexFilterable": True, "tokenization": "field"} if n in ai.SEARCH_SCOPE_FIELDS else {})}
                             for n, t in ai.SEARCH_PROPERTIES.items()],
              "vectorConfig": {"text_nim": {"vectorizer": {"none": {}}}}}
    def handle(request):
        """Serve a synthetic read-only schema and additive object endpoint.

        Inputs: REST request. Outputs: response. Effects: fixture call/object log;
        choose for actual httpx request and named-vector contract validation.
        """
        calls.append((request.method, request.url.path))
        if "/schema/" in request.url.path:
            return httpx.Response(200, json=schema)
        if request.method == "POST":
            obj = json.loads(request.content)
            if obj["id"] in objects:
                return httpx.Response(422, json={"error": "already exists"})
            objects[obj["id"]] = obj
            return httpx.Response(200, json=obj)
        key = request.url.path.rsplit("/", 1)[-1]
        return httpx.Response(200 if key in objects else 404, json=objects.get(key))
    store = ai.AIChatStore("http://synthetic", "AiChatEvents20260918", "text_nim", http=httpx.Client(transport=httpx.MockTransport(handle)))
    store.schema()
    assert all(method == "GET" for method, _ in calls)
    obj = {"id": str(uuid4()), "class": store.collection, "properties": {"body": "synthetic complete conversation"}, "vectors": {"text_nim": [0.25] * 2048}}
    store.add(obj)
    ai.compare_object(store.get(obj["id"]), obj, store.vector_name)
    store.add(obj)  # duplicate POST is rejected, independent GET proves identical
    assert len(objects) == 1
    with pytest.raises(ai.ContentInvalid):
        store.add({**obj, "properties": {"body": "different output"}})
    assert objects[obj["id"]] == obj
    schema["properties"][0]["dataType"] = ["object"]
    with pytest.raises(ai.ContentInvalid):
        store.schema()


def test_temporal_names_shared_go_fields_and_permanent_errors(scope, monkeypatch):
    """Prove wire names and retry semantics; inputs shared Go shape, output assertions, no external effects; choose for Activity registration contracts."""
    from temporalio.exceptions import ApplicationError
    from temporalio import activity
    from server.temporal import ai_content_activities as activities
    expected = ("ai_prepare_content_activity", "ai_extract_work_products_activity", "ai_extract_candidates_activity", "ai_embed_content_activity",
                "ai_publish_content_activity", "ai_verify_content_publication_activity")
    assert tuple(activity._Definition.must_from_callable(f).name for f in activities.AI_CONTENT_ACTIVITIES) == expected
    params = activities.AIContentParams(**scope, max_records=256, max_text_bytes=2097152, max_chunks=128, max_model_calls=128,
        prepared_ref="file:///prepared", candidates_ref="file:///candidates", embeddings_ref="file:///embeddings", publication_ref="file:///publication")
    def invalid(_):
        """Raise the permanent fixture admission error.

        Inputs: request. Outputs: exception. Effects: none; tests retry semantics.
        """
        raise ai.ContentInvalid("bad fixture pins")
    monkeypatch.setattr(ai, "prepare_content", invalid)
    with pytest.raises(ApplicationError) as error:
        activities.ai_prepare_content_activity(params)
    assert error.value.type == "AIContentInvalid" and error.value.non_retryable
    worker = Path(__file__).parents[1] / "server/temporal/worker.py"
    assert "*AI_CONTENT_ACTIVITIES" in worker.read_text(encoding="utf-8")


def test_complete_work_product_content_spans_repetition_and_truncation(scope, monkeypatch):
    """Prove full repeated artifact and draft spans; input retained synthetic bodies, output assertions, effects retained files; choose over short quotation checks."""
    body = "Intro\n```python\nprint('Alice')\n```\n\n```python\nprint('Alice')\n```\nDRAFT Motion\nEntire draft here.\n"
    row = record(0, body)
    second = record(1, "~~~sql\nSELECT 1;\n", conversation=1)
    ref, _ = prepared(scope, monkeypatch, [row, second])
    result = ai.extract_work_products({**scope, "prepared_ref": ref})
    bundle = ai._read(result["bundle_ref"], scope, "work_products")
    products = bundle["work_products"]
    assert result["stage"] == "work_products" and result["work_products"] == 4
    assert products[0]["content"] == products[1]["content"]
    assert products[0]["body_start"] != products[1]["body_start"]
    assert products[2]["content"] == "DRAFT Motion\nEntire draft here.\n"
    assert products[3]["closed"] is False and products[3]["content"] == second["body"]
    for product in products:
        original = row if product["conversation_index"] == 0 else second
        assert product["content"] == original["body"][product["body_start"]:product["body_end"]]
        assert product["source_segments"][0]["native_json_pointer"] == original["native_json_pointer"]
        assert Path(product["file_ref"].removeprefix("file://")).read_text(encoding="utf-8") == product["content"]
    assert ai.extract_work_products({**scope, "prepared_ref": ref})["bundle_ref"] == result["bundle_ref"]
    drafts = ai.full_work_product_spans(record(2, "DRAFT One\nComplete first.\nSubject: Two\nComplete second.\n"))
    assert [d["content"] for d in drafts] == ["DRAFT One\nComplete first.\n", "Subject: Two\nComplete second.\n"]


def test_source_binding_original_membership_and_durable_case_scope(scope):
    """Bind exact original membership and LIVE case scope without generation reads.

    Inputs: source/object metadata fixture. Outputs: pinned source binding.
    Effects: no DB writes; choose for source-only preparation admission.
    """
    row = {**source(scope), "workflow_id": scope["request_id"], "source_status": "retained",
           "declared_format": "chatgpt_json_array", "matter_id": scope["matter_id"], "court_case_id": scope["court_case_id"],
           "admission": {k: scope[k] for k in ("operating_mode", "matter_id", "court_case_id")},
           "source_version_id": scope["source_version_id"], "original_object_id": str(uuid4()),
           "original_bytes": 128, "storage_class": "inline",
           "source_key": "b2://synthetic/conversations.json#exact-native-version"}

    class Connection:
        """Expose a single exact metadata result without payload reads.

        Inputs: SQL query/pins. Outputs: bound row. Effects: fixture assertions;
        choose for binding tests independently from source extraction tests.
        """
        def execute(self, query, values):
            """Check exact source/original membership lookup coordinates.

            Inputs: SQL and params. Outputs: result facade. Effects: assertions.
            """
            assert values == {"source": scope["source_version_id"]}
            assert "JOIN context.source src" in str(query)
            assert "JOIN context.source_version_object so" in str(query)
            assert "normalized_generation" not in str(query)
            return SimpleNamespace(mappings=lambda: SimpleNamespace(one_or_none=lambda: row))

    assert ai.source_binding(Connection(), scope)["source_key"].endswith("exact-native-version")
    for field, value in [("original_bytes", ai.MAX_SOURCE_BYTES + 1), ("matter_id", str(uuid4())),
                         ("source_status", "registered"), ("declared_format", "codex_rollout_jsonl")]:
        previous = row[field]
        row[field] = value
        with pytest.raises(ai.ContentInvalid):
            ai.source_binding(Connection(), scope)
        row[field] = previous
    row["admission"]["operating_mode"] = "DEV"
    with pytest.raises(ai.ContentInvalid):
        ai.source_binding(Connection(), scope)


def test_per_record_projection_cannot_cross_publication_gate(scope, monkeypatch):
    """Reject fully rebound per-record projections at the conversation guard; input forged fixtures, output assertions, effects retained files; choose for an independent anti-message fence."""
    params, prep, _, _ = dependencies(scope, monkeypatch)
    bad_chunks = [ai.conversation_windows([r], 128)[0] for r in prep["records"]]
    bad = {k: v for k, v in prep.items() if k not in {"version", "stage", "pins", "bundle_fingerprint"}}
    bad["chunks"] = bad_chunks
    bad["counts"] = {**bad["counts"], "chunks": len(bad_chunks)}
    ref = ai._save(ai._path(scope, "prepared", "forged-per-message"), scope, "prepared", bad)
    forged = {**params, "prepared_ref": ref}
    for stage, field in (("work_products", "work_products_ref"), ("candidates", "candidates_ref"), ("embedded", "embeddings_ref")):
        original = ai._read(params[field], scope, stage)
        rebound = {k: v for k, v in original.items() if k not in {"version", "stage", "pins", "bundle_fingerprint"}}
        rebound["prepared_ref"] = ref
        if stage == "candidates":
            rebound["work_products_ref"] = forged["work_products_ref"]
        if stage == "embedded":
            rebound["vectors"] = [original["vectors"][0] for _ in bad_chunks]
        forged[field] = ai._save(ai._path(scope, stage, "forged-per-message"), scope, stage, rebound)
    with pytest.raises(ai.ContentInvalid, match="prepared native topic chunks"):
        ai.publication_inputs(forged)


def test_prepare_uses_exact_reader_ordinals_and_accounts_empty_records(scope, monkeypatch):
    """Prepare only verified native fields and account for empty source text.

    Inputs: retained-source fixture. Outputs: exact native records and counts.
    Effects: retained bundle file; choose over normalized generation readers.
    """
    from server.context_chunks import db
    bound = source(scope)
    monkeypatch.setattr(ai, "source_binding", lambda conn, pin: bound)
    native = [{"id": "conversation-1", "current_node": "second", "mapping": {
        "first": {"parent": None, "message": {"author": {"role": "assistant"}, "content": {"parts": ["full source text"]}}},
        "second": {"parent": "first", "message": {"author": {"role": "assistant"}, "content": {"parts": [""]}}}}}]
    monkeypatch.setattr(ai, "read_verified_original", lambda conn, source: native)
    from server.context_chunks import chunker
    monkeypatch.setattr(chunker, "neural_text_spans", lambda text, **_: [(0, len(text))])
    class Connection:
        """Stand in for a source-metadata connection with no generation query."""
        pass
    @contextmanager
    def connection():
        """Yield the retained fixture's read-only connection facade.

        Inputs: none. Outputs: fake connection. Effects: none; choose for tests.
        """
        yield Connection()
    monkeypatch.setattr(db, "read_only_connection", connection)
    result = ai.prepare_content(scope)
    bundle = ai._read(result["bundle_ref"], scope, "prepared")
    assert result["records"] == 1 and result["chunks"] == 1
    assert bundle["empty_record_ids"] == []
    assert [r["ordinal"] for r in bundle["records"]] == [0]
    assert all(r["native_json_pointer"].startswith("/0/mapping/") for r in bundle["records"])
    assert all("raw_occurrences" not in r for r in bundle["records"])
    assert ai.prepare_content(scope)["bundle_ref"] == result["bundle_ref"]
    with pytest.raises(ai.ContentInvalid):
        ai.prepare_content({**scope, "max_records": 0})


def test_native_decoding_612_fields_preserves_exact_source_pointers(scope, monkeypatch):
    """Decode an actual-sized native mapping without a normalized generation.

    Inputs: 612 bounded synthetic native fields. Outputs: exact ordered pointer
    and source span assertions. Effects: none; choose for large retained exports.
    """
    native = [{"id": "conversation-1", "mapping": {
        f"node-{index}": {"parent": f"node-{index - 1}" if index else None,
                           "message": {"author": {"role": "assistant"},
                                    "content": {"parts": [f"body-{index}"]}}}
        for index in range(612)}}]
    rows = ai.decode_native_conversations(native, source(scope), 612)
    assert len(rows) == 612
    assert [r["body"] for r in rows] == [f"body-{i}" for i in range(612)]
    assert rows[611]["native_json_pointer"] == "/0/mapping/node-611/message/content/parts/0"
    assert all(row["source_span"]["sha256"] == hashlib.sha256(row["body"].encode()).hexdigest() for row in rows)
    with pytest.raises(ai.ContentInvalid):
        ai.decode_native_conversations(native, source(scope), 611)
    from server.temporal.ai_content_activities import AIContentParams
    defaults = AIContentParams()
    assert (defaults.max_records, defaults.max_text_bytes, defaults.max_chunks, defaults.max_model_calls) == (1024, 2097152, 256, 512)
    assert ai.VERSION == "ai-content-native-v3"

def test_same_model_output_repair_is_bounded_and_both_attempts_retained(scope, monkeypatch):
    """Prove one schema correction retains both attempts; inputs fake provider and source, output assertions, effects retained files; choose over fallback-model tests."""
    ref, prep = prepared(scope, monkeypatch)
    products = ai.extract_work_products({**scope, "prepared_ref": ref})
    calls = []
    class Model:
        """Emit one invalid taxonomy followed by one exact source-grounded result.

        Inputs: same source prompt. Outputs: scored replies. Effects: fixture log;
        choose for bounded output repair without provider/model fallback.
        """
        def infer(self, prompts, **kwargs):
            """Return deterministic malformed/valid candidate shapes in order.

            Inputs: prompt/config. Outputs: scored JSON. Effects: append call log.
            """
            calls.append(prompts[0])
            kind = "person" if len(calls) == 1 else "entity"
            yield [SimpleNamespace(output=json.dumps({"candidates": [{"kind": kind, "title": "Alice", "record_id": prep["records"][0]["record_id"], "quote": "Alice"}]}))]
    config = SimpleNamespace(base_url="https://configured.example/v1", model_id="moonshotai/kimi-k3", max_tokens=6000)
    result = ai.extract_candidates({**scope, "prepared_ref": ref, "work_products_ref": products["bundle_ref"]}, model=Model(), config=config)
    assert result["model_calls"] == 2 and result["candidates"] == 1
    assert len(calls) == 2 and calls[1].startswith(calls[0])
    retained = list(ai._root().glob("*/model_reply/*.json"))
    assert len(retained) == 2
    assert {json.loads(path.read_bytes())["repair_attempt"] for path in retained} == {0, 1}


def checkpoint_fixture(scope, monkeypatch, *, chunks=1):
    """Create exact extraction inputs for durable retries without a provider.

    Inputs: synthetic scope and chunk count. Outputs: params/preparation/config.
    Effects: retained synthetic files; choose for checkpoint and budget tests.
    """
    ref, prep = prepared(scope, monkeypatch, [record(i, "Alice draft " + str(i), conversation=i) for i in range(chunks)])
    products = ai.extract_work_products({**scope, "prepared_ref": ref})
    params = {**scope, "prepared_ref": ref, "work_products_ref": products["bundle_ref"], "max_model_calls": chunks * 2}
    config = SimpleNamespace(base_url="https://configured.example/v1", model_id="moonshotai/kimi-k3", max_tokens=6000)
    return params, prep, config


class CheckpointModel:
    """Return exact quoted candidates and optionally fail one synthetic request.

    Inputs: failure request number. Outputs: scored replies. Effects: call log;
    choose for retry durability without any real inference.
    """
    def __init__(self, fail_at=None):
        """Initialize the fixture; inputs failure ordinal, outputs model, effects memory only; choose for deterministic retry tests."""
        self.calls = []
        self.fail_at = fail_at

    def infer(self, prompts, **kwargs):
        """Ground the fixture in the supplied window; inputs prompt, outputs scored JSON, effects call log; choose over invented fixture IDs."""
        self.calls.append(prompts[0])
        if len(self.calls) == self.fail_at:
            raise RuntimeError("synthetic transport failure")
        segments = json.loads(prompts[0].split("SOURCE WINDOW:\n", 1)[1])
        yield [SimpleNamespace(output=json.dumps({"candidates": [{"kind": "entity", "title": "Alice", "record_id": segments[0]["record_id"], "quote": "Alice"}]}))]


def test_late_failure_reuses_only_validated_chunks_without_repeat_calls(scope, monkeypatch):
    """Reuse the first grounded chunk after a later failure; inputs two chunks, outputs ordered results, effects retained fake attempts; choose for durable retry."""
    params, prep, config = checkpoint_fixture(scope, monkeypatch, chunks=2)
    model = CheckpointModel(fail_at=2)
    with pytest.raises(RuntimeError, match="no fallback"):
        ai.extract_candidates(params, model=model, config=config)
    assert len(list(ai._root().glob("*/candidate_chunk/*.json"))) == 1
    result = ai.extract_candidates(params, model=model, config=config)
    assert len(model.calls) == 3 and model.calls[1] == model.calls[2]
    assert result["cached_chunks"] == 1 and result["new_model_calls"] == 1
    assert result["provider_budget_consumed"] == result["model_calls"] == 3
    bundle = ai._read(result["bundle_ref"], scope, "candidates")
    assert [item["content_key"] for item in bundle["replies"]] == [chunk["content_key"] for chunk in prep["chunks"]]
    assert [item["conversation_index"] for item in bundle["candidates"]] == ["0", "1"]
    again = ai.extract_candidates(params, model=model, config=config)
    assert again["new_model_calls"] == 0 and len(model.calls) == 3


@pytest.mark.parametrize("change", ["prompt", "model", "endpoint", "max_tokens"])
def test_changed_prompt_identity_cannot_reuse_completed_chunk(scope, monkeypatch, change):
    """Require fresh inference after exact prompt bytes change; inputs one chunk, outputs distinct identities, effects fake retained calls; choose for cache invalidation."""
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    params["max_model_calls"] = 4
    model = CheckpointModel()
    first = ai.extract_candidates(params, model=model, config=config)
    if change == "prompt":
        monkeypatch.setattr(ai, "PROMPT", "Changed policy.\n" + ai.PROMPT)
    else:
        field = {"model": "model_id", "endpoint": "base_url", "max_tokens": "max_tokens"}[change]
        monkeypatch.setattr(config, field, 7000 if field == "max_tokens" else getattr(config, field) + "-changed")
    second = ai.extract_candidates(params, model=model, config=config)
    assert first["bundle_ref"] != second["bundle_ref"] and len(model.calls) == 2
    assert second["cached_chunks"] == 0 and second["provider_budget_consumed"] == 2
    monkeypatch.setattr(ai, "OUTPUT_CORRECTION", "Different correction")
    with pytest.raises(ai.ContentInvalid, match="two durable"):
        ai.extract_candidates(params, model=model, config=config)
    assert len(model.calls) == 2


def test_unknown_outcome_consumes_slot_and_exhaustion_is_permanent(scope, monkeypatch):
    """Charge interrupted intents rather than resetting usage; inputs unknown requests, outputs bounded failure, effects retained claims; choose for crash recovery."""
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    identity = ai.candidate_identity(params["prepared_ref"], params["work_products_ref"], config, 2)
    chunk = prep["chunks"][0]
    ai._reserve_provider_call(scope, prep["source"], identity, chunk["content_key"], {"synthetic_unknown": 1})
    model = CheckpointModel()
    result = ai.extract_candidates(params, model=model, config=config)
    assert len(model.calls) == 1 and result["model_calls"] == 2
    monkeypatch.setattr(ai, "PROMPT", "New identity.\n" + ai.PROMPT)
    with pytest.raises(ai.ContentInvalid, match="budget exhausted"):
        ai.extract_candidates(params, model=model, config=config)
    assert len(model.calls) == 1


def test_forged_checkpoint_is_regrounded_and_legacy_usage_fails_closed(scope, monkeypatch):
    """Reject forged grounded output and unbound legacy usage; inputs retained forgeries, outputs explicit errors, effects fixture files; choose over trusting cache metadata."""
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    chunk = prep["chunks"][0]
    identity = ai.candidate_identity(params["prepared_ref"], params["work_products_ref"], config, 2)
    prompt = ai.PROMPT + json.dumps([{"record_id": s["record_id"], "text": s["text"], "role": s.get("role")} for s in chunk["segments"]], ensure_ascii=False)
    path, _ = ai._candidate_checkpoint(scope, prep["source"], identity, chunk, prompt)
    raw = json.dumps({"candidates": []})
    ai._save(path, scope, "candidate_chunk", {"source": prep["source"], "identity": identity,
        "content_key": chunk["content_key"], "prompt_digest": ai._key(prompt), "raw_reply": raw,
        "reply_ref": "file:///missing", "candidates": [{"invented": "candidate"}]})
    model = CheckpointModel()
    with pytest.raises(ai.ContentInvalid, match="grounding differs"):
        ai.extract_candidates(params, model=model, config=config)
    assert model.calls == []
    ai._save(ai._path(scope, "provider_attempt", "old-unbound"), scope, "provider_attempt", {"source": prep["source"], "response": {"error_type": "InternalServerError"}})
    with pytest.raises(ai.ContentInvalid, match="legacy provider attempt lacks durable intent"):
        ai.extract_candidates(params, model=model, config=config)
    assert model.calls == []


def test_exclusive_request_claim_cannot_be_issued_twice(scope):
    """Reject duplicate atomic intent claims; inputs one immutable identity, outputs exclusive failure, effects retained files; choose for concurrent retry safety."""
    path = ai._path(scope, "candidate_intent", ["same-content", 1])
    data = {"source": source(scope), "request": {"model": "configured"}}
    ai._save(path, scope, "candidate_intent", data, require_new=True)
    with pytest.raises(FileExistsError):
        ai._save(path, scope, "candidate_intent", data, require_new=True)


def test_exclusive_claim_syncs_file_then_link_then_directory_ancestry(scope, monkeypatch):
    """Check the reservation fsync protocol with real retained fixture writes.

    Inputs: synthetic scope and transparent filesystem spies. Outputs: ordered
    file/link/directory assertions. Effects: fixture files only; choose for protocol
    ordering rather than claiming hardware power-loss testing.
    Byline: Codex / GPT-6.1-Sol / 2026-10-07.
    """
    events = []
    descriptors = {}
    real_open, real_fsync, real_link = ai.os.open, ai.os.fsync, ai.os.link
    def opened(path, flags, *args, **kwargs):
        """Track actual descriptors; inputs open arguments, outputs fd, effects fixture I/O; choose for transparent ordering proof."""
        fd = real_open(path, flags, *args, **kwargs)
        descriptors[fd] = ai.Path(path)
        return fd
    def synced(fd):
        """Record actual fsync; inputs fd, outputs none, effects fixture sync; choose for ordering assertions."""
        events.append(("sync", descriptors[fd]))
        return real_fsync(fd)
    def linked(pending, target):
        """Record actual exclusive link; inputs paths, outputs none, effects retained fixture link; choose for claim timing."""
        result = real_link(pending, target)
        events.append(("link", target))
        return result
    monkeypatch.setattr(ai.os, "open", opened)
    monkeypatch.setattr(ai.os, "fsync", synced)
    monkeypatch.setattr(ai.os, "link", linked)
    path = ai._path(scope, "candidate_intent", ["ordered", 1])
    ai._save(path, scope, "candidate_intent", {"source": source(scope)}, require_new=True)
    assert events[0][0] == "sync" and events[0][1].suffix == ".pending"
    assert events[1] == ("link", path)
    assert events[2:] == [("sync", directory) for directory in
                         (path.parent, path.parent.parent, ai._root())]


def test_failed_budget_directory_sync_prevents_call_and_retains_consumption(scope, monkeypatch):
    """Fail closed before model dispatch when budget namespace durability fails.

    Inputs: synthetic extraction and injected directory fsync failure. Outputs:
    zero calls and one retained intent/budget. Effects: retained fixture claims;
    choose for conservative unknown-outcome accounting without hardware disruption.
    Byline: Codex / GPT-6.1-Sol / 2026-10-07.
    """
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    descriptors = {}
    real_open, real_fsync = ai.os.open, ai.os.fsync
    def opened(path, flags, *args, **kwargs):
        """Track actual descriptors; inputs open arguments, outputs fd, effects fixture I/O; choose for targeted directory failure."""
        fd = real_open(path, flags, *args, **kwargs)
        descriptors[fd] = ai.Path(path)
        return fd
    def synced(fd):
        """Reject only budget directory fsync; inputs fd, outputs error or sync, effects fixture I/O; choose for pre-dispatch failure proof."""
        if descriptors[fd].name == "candidate_budget":
            raise OSError("synthetic directory sync failure")
        return real_fsync(fd)
    monkeypatch.setattr(ai.os, "open", opened)
    monkeypatch.setattr(ai.os, "fsync", synced)
    model = CheckpointModel()
    with pytest.raises(RuntimeError, match="no fallback"):
        ai.extract_candidates(params, model=model, config=config)
    assert model.calls == []
    assert len(ai._provider_budget(scope, prep["source"])) == 1
    assert len(ai._chunk_intents(scope, prep["chunks"][0]["content_key"])) == 1
    assert list(ai._root().glob("*/provider_attempt/*.json")) == []


def test_sdk_request_is_claimed_before_call_and_error_receipt_binds_intent(scope, monkeypatch):
    """Prove production SDK interception retains pre-call claims; inputs fake SDK, outputs receipt bindings, effects synthetic files only; choose over testing only the injected model seam."""
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    calls = []
    class Client:
        """Model the configured SDK transport; inputs options/request, outputs failure, effects fixture assertions; choose without network calls."""
        def __init__(self):
            """Expose the SDK request surface; inputs none, outputs facade, effects memory only; choose for interception tests."""
            self.chat = SimpleNamespace(completions=SimpleNamespace(create=self.create))
        def with_options(self, **kwargs):
            """Require disabled SDK retries; inputs options, outputs same client, effects assertions; choose for bounded transport behavior."""
            assert kwargs == {"max_retries": 0}
            return self
        def create(self, **kwargs):
            """Inspect durable claims before simulated request failure; inputs SDK arguments, outputs exception, effects call log; choose for intent timing proof."""
            assert len(list(ai._root().glob("*/candidate_intent/*.json"))) == 1
            assert len(list(ai._root().glob("*/candidate_budget/*.json"))) == 1
            calls.append(kwargs)
            raise TimeoutError("synthetic SDK timeout")
    class Model:
        """Use the intercepted SDK request; inputs prompt, outputs exception, effects fake client only; choose for the real callback route."""
        def __init__(self):
            """Install the fixture SDK; inputs none, outputs model, effects memory only; choose for callback coverage."""
            self._client = Client()
        def infer(self, prompts, **kwargs):
            """Dispatch one request; inputs source prompt, outputs transport exception, effects fake SDK call; choose for retained failure proof."""
            self._client.chat.completions.create(model=config.model_id, messages=[{"role": "user", "content": prompts[0]}],
                max_tokens=config.max_tokens, temperature=0, response_format={"type": "json_object"})
            yield []
    with pytest.raises(RuntimeError, match="no fallback"):
        ai.extract_candidates(params, model=Model(), config=config)
    paths = list(ai._root().glob("*/provider_attempt/*.json"))
    assert len(calls) == len(paths) == 1
    attempt = ai._read(paths[0].as_uri(), scope, "provider_attempt")
    assert attempt["response"] == {"error_type": "TimeoutError"}
    intent = ai._read(attempt["intent_ref"], scope, "candidate_intent")
    budget = ai._read(attempt["budget_ref"], scope, "candidate_budget")
    assert intent["request"] == calls[0] and budget["intent_ref"] == attempt["intent_ref"]
    assert len(ai._provider_budget(scope, prep["source"])) == 1


def test_cancellation_after_claim_prevents_sdk_request_and_preserves_consumption(scope, monkeypatch):
    """Stop SDK work after a canceled beat; inputs wrapper cancellation and fake SDK, outputs zero calls/one claim, effects retained fixture only; choose for the production cancellation boundary."""
    from server.temporal import ai_content_activities as activities
    from temporalio.exceptions import CancelledError
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    canceled = [False]
    calls = []
    monkeypatch.setattr(activities.activity, "in_activity", lambda: True)
    monkeypatch.setattr(activities.activity, "is_cancelled", lambda: canceled[0])
    monkeypatch.setattr(activities.activity, "heartbeat", lambda *args: None)
    class Client:
        """Expose a fake SDK without remote access; inputs requests, outputs failure if called, effects counter; choose for cancellation proof."""
        def __init__(self):
            """Build the transport facade; inputs none, outputs client, effects memory only; choose for SDK interception."""
            self.chat = SimpleNamespace(completions=SimpleNamespace(create=self.create))
        def with_options(self, **kwargs):
            """Return the fake configured client; inputs options, outputs self, effects none; choose without SDK construction."""
            return self
        def create(self, **kwargs):
            """Reject any request; inputs arguments, outputs assertion, effects call log; choose to prove cancellation stops transport."""
            calls.append(kwargs)
            raise AssertionError("canceled request reached SDK")
    class Model:
        """Simulate cancellation delivered before intercepted SDK dispatch; inputs prompt, outputs exception, effects cancellation flag; choose for synchronous boundaries."""
        def __init__(self):
            """Install the fake client; inputs none, outputs model, effects memory only; choose for boundary coverage."""
            self._client = Client()
        def infer(self, prompts, **kwargs):
            """Deliver cancellation before dispatch; inputs prompt, outputs propagated cancellation, effects flag; choose for conservative reservation."""
            canceled[0] = True
            self._client.chat.completions.create(model=config.model_id, messages=[{"role": "user", "content": prompts[0]}])
            yield []
    with activities._heartbeats() as beat:
        with pytest.raises(CancelledError, match="cancellation was requested"):
            ai.extract_candidates(params, model=Model(), config=config, beat=beat)
    assert calls == []
    assert len(ai._provider_budget(scope, prep["source"])) == 1
    assert len(ai._chunk_intents(scope, prep["chunks"][0]["content_key"])) == 1
    assert list(ai._root().glob("*/provider_attempt/*.json")) == []


def routed_fixture(scope, monkeypatch, *, chunks=1, first_failure=False):
    """Bind the actual routed extraction seam to a retained fake SDK.

    Inputs: synthetic scope, chunk count and first-call failure. Outputs: params,
    router, clock and calls. Effects: fixture files only; choose for durable route
    integration without production inference. Byline: Codex / GPT-6.1-Sol / 2026-10-07.
    """
    from server.analysis import ai_content_provider as provider
    params, prep, _ = checkpoint_fixture(scope, monkeypatch, chunks=chunks)
    clock, calls = [1000.0], []
    options = {"max_tokens": 6000, "temperature": 0, "response_format": {"type": "json_object"}}
    profiles = tuple(provider.Profile("nvidia", model, provider.NIM_URL, "ai-nim-primary", "synthetic", dict(options))
                     for model in (*provider.PRIMARY_MODELS, provider.BACKUP_MODEL))
    def factory(profile):
        """Build one fake routed transport; inputs profile, outputs SDK facade, effects none; choose for actual request binding."""
        def create(**request):
            """Return exact grounded fenced JSON; inputs SDK request, outputs completion/error, effects call log; choose without remote calls."""
            calls.append(request)
            if first_failure and len(calls) == 1:
                error = RuntimeError("synthetic provider failure")
                error.status_code = 503
                error.response = SimpleNamespace(headers={})
                raise error
            prompt = request["messages"][-1]["content"]
            segments = json.loads(prompt.split("SOURCE WINDOW:\n", 1)[1].split("\nOUTPUT CORRECTION:", 1)[0])
            raw = "```json\n" + json.dumps({"candidates": [{"kind": "entity", "title": "Alice", "record_id": segments[0]["record_id"], "quote": "Alice"}]}) + "\n```"
            data = {"model": profile.model_id, "usage": {"total_tokens": 12},
                    "choices": [{"message": {"content": raw}, "finish_reason": "stop"}]}
            return SimpleNamespace(choices=[SimpleNamespace(message=SimpleNamespace(content=raw), finish_reason="stop")],
                                   model_dump=lambda **kwargs: data)
        return SimpleNamespace(chat=SimpleNamespace(completions=SimpleNamespace(create=create)))
    def sleep(seconds):
        """Advance fixture clock; inputs seconds, outputs none, effects clock only; choose without real waiting."""
        clock[0] += seconds
    router = provider.ProviderRouter(ai._root(), profiles, clock=lambda: clock[0], monotonic=lambda: clock[0],
                                     sleep=sleep, jitter=lambda: 1, client_factory=factory)
    return params, router, clock, calls


def test_routed_primaries_keep_exact_winners_and_reuse_all_checkpoints(scope, monkeypatch):
    """Bind alternating model winners and reuse their grounded fenced replies.

    Inputs: two retained synthetic chunks. Outputs: actual model/ref proof and
    zero retry calls. Effects: fake SDK/files only; choose for routed production seam.
    """
    from server.analysis import ai_content_provider as provider
    params, router, clock, calls = routed_fixture(scope, monkeypatch, chunks=2)
    result = ai.extract_candidates(params, router=router)
    candidates = ai._read(result["bundle_ref"], scope, "candidates")
    assert [reply["actual_provider"]["model_id"] for reply in candidates["replies"]] == list(provider.PRIMARY_MODELS)
    assert candidates["model_id"] is None and candidates["model_ids"] == sorted(provider.PRIMARY_MODELS)
    assert all(reply["reply"].startswith("```json\n") for reply in candidates["replies"])
    assert result["new_model_calls"] == result["model_calls"] == 2
    prep = ai._read(params["prepared_ref"], scope, "prepared")
    embedded = {"model_id": "synthetic-remote-embed", "vectors": [[0.25] * 2048 for _ in prep["chunks"]]}
    objects = ai.search_objects({**params, "candidates_ref": result["bundle_ref"], "embeddings_ref": "file:///synthetic-embedding"},
                                prep, candidates, embedded, "AiChatEvents20260918", "text_nim")
    citations = [json.loads(obj["properties"]["provenance"][0]) for obj in objects]
    assert [citation["extraction_model"] for citation in citations] == list(provider.PRIMARY_MODELS)
    assert all(citation["extraction_provider"] == "nvidia" and citation["extraction_reply_ref"] for citation in citations)
    assert all(obj["properties"]["source_version_id"] == scope["source_version_id"] and
               obj["properties"]["promotion_policy"] == "forbidden" for obj in objects)
    cached = ai.extract_candidates(params, router=router)
    assert cached["new_model_calls"] == 0 and cached["cached_chunks"] == 2 and len(calls) == 2


def test_routed_alternate_success_reuses_exact_attempt_and_cannot_reset_budget(scope, monkeypatch):
    """Reuse a successful alternate after failure and reject a fresh third request.

    Inputs: fake503 then grounded alternate. Outputs: preserved actual winner,
    persistent two-slot accounting and no recall. Effects: retained fake fixtures.
    Choose for backup durability under plan changes and Temporal retries.
    """
    from server.analysis import ai_content_provider as provider
    params, router, clock, calls = routed_fixture(scope, monkeypatch, first_failure=True)
    with pytest.raises(provider.ProviderDeferred) as error:
        ai.extract_candidates(params, router=router)
    clock[0] += error.value.delay
    result = ai.extract_candidates(params, router=router)
    bundle = ai._read(result["bundle_ref"], scope, "candidates")
    assert bundle["replies"][0]["actual_provider"]["model_id"] == provider.PRIMARY_MODELS[1]
    assert result["model_calls"] == 2 and result["new_model_calls"] == 1
    again = ai.extract_candidates(params, router=router)
    assert again["new_model_calls"] == 0 and len(calls) == 2
    monkeypatch.setattr(ai, "PROMPT", "changed exact policy\n" + ai.PROMPT)
    with pytest.raises(ai.ContentInvalid, match="budget exhausted"):
        ai.extract_candidates(params, router=router)
    assert len(calls) == 2


def legacy_review_fixture(scope, prep, *, charge=6, change=None):
    """Retain two unbound failures and an explicit exact-set synthetic review.

    Inputs: fixture pins/source, charged upper bound and optional review edit.
    Outputs: original receipt bytes and review reference. Effects: additive
    fixture files only; choose for legacy recovery without inventing old intents.
    """
    originals = {}
    for error in ("APIConnectionError", "InternalServerError"):
        path = ai._path(scope, "provider_attempt", ["legacy", error])
        ai._save(path, scope, "provider_attempt", {"source": prep["source"],
            "content_key": prep["chunks"][0]["content_key"], "provider_call": 1,
            "response": {"error_type": error}})
        originals[path] = path.read_bytes()
    data = {"source": prep["source"], "legacy_attempts": [
        {"ref": path.as_uri(), "sha256": hashlib.sha256(raw).hexdigest()}
        for path, raw in sorted(originals.items())],
        "historical_charged_upper_bound": charge,
        "authorization": "fresh_bounded_attempts_for_reviewed_legacy_chunks",
        "reason": "At most three historical Activity attempts with two SDK requests each; not actual six calls.",
        "proof_refs": ["file:///synthetic/temporal-history-event19", "git:synthetic-historical-max-retries-zero"]}
    if change:
        change(data)
    path = ai._path(scope, "provider_accounting_review", prep["source"])
    ref = ai._save(path, scope, "provider_accounting_review", data)
    return originals, ref


@pytest.mark.parametrize("routed", [False, True])
def test_explicit_legacy_review_resumes_with_separate_historical_charge(scope, monkeypatch, routed):
    """Resume reviewed legacy chunks and preserve historical evidence and charges.

    Inputs: exact synthetic review and legacy or routed provider. Outputs:
    six historical plus actual new usage, cached no-call resume. Effects: fake
    calls and additive fixture files; choose for the existing Activity recovery.
    """
    if routed:
        params, router, clock, calls = routed_fixture(scope, monkeypatch)
        prep = ai._read(params["prepared_ref"], scope, "prepared")
        options = {"router": router}
    else:
        params, prep, config = checkpoint_fixture(scope, monkeypatch)
        model = CheckpointModel()
        calls, options = model.calls, {"model": model, "config": config}
    params["max_model_calls"] = 8
    originals, review_ref = legacy_review_fixture(scope, prep)
    result = ai.extract_candidates(params, **options)
    assert len(calls) == result["new_provider_budget_consumed"] == result["new_model_calls"] == 1
    assert result["historical_provider_charge"] == 6
    assert result["model_calls"] == result["provider_budget_consumed"] == 7
    assert len(ai._provider_budget(scope, prep["source"])) == 1
    assert len(ai._chunk_intents(scope, prep["chunks"][0]["content_key"])) == 1
    again = ai.extract_candidates(params, **options)
    assert again["new_model_calls"] == 0 and len(calls) == 1
    assert again["historical_provider_charge"] == 6 and again["model_calls"] == 7
    assert all(path.read_bytes() == raw for path, raw in originals.items())
    review = ai._read(review_ref, scope, "provider_accounting_review")
    with pytest.raises(ai.ContentInvalid, match="different retained output"):
        ai._save(ai._path(scope, "provider_accounting_review", prep["source"]), scope,
                 "provider_accounting_review", {**{key: value for key, value in review.items()
                    if key not in ("version", "pins", "stage", "bundle_fingerprint")}, "historical_charged_upper_bound": 2})


@pytest.mark.parametrize("change", ["source", "hash", "ref", "missing", "authorization", "charge", "proof"])
def test_legacy_review_must_match_complete_evidence_and_authorization(scope, monkeypatch, change):
    """Refuse mismatched legacy review source, full bytes, receipt set or authority.

    Inputs: synthetic altered review. Outputs: permanent refusal before dispatch.
    Effects: retained fixture files only; choose over broad legacy exemptions.
    """
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    params["max_model_calls"] = 8
    def edit(data):
        """Alter one review assertion; input review, output none, effects fixture memory; choose for fail-closed validation."""
        if change == "source":
            data["source"] = {**data["source"], "original_sha256": "b" * 64}
        elif change in ("hash", "ref"):
            data["legacy_attempts"][0]["sha256" if change == "hash" else "ref"] = "wrong"
        elif change == "missing":
            data["legacy_attempts"] = data["legacy_attempts"][:1]
        elif change == "authorization":
            data["authorization"] = "unreviewed"
        elif change == "charge":
            data["historical_charged_upper_bound"] = 1
        else:
            data["proof_refs"] = []
    legacy_review_fixture(scope, prep, change=edit)
    model = CheckpointModel()
    with pytest.raises(ai.ContentInvalid, match="accounting review differs"):
        ai.extract_candidates(params, model=model, config=config)
    assert model.calls == [] and not ai._chunk_intents(scope, prep["chunks"][0]["content_key"])


def test_new_unreviewed_legacy_receipt_invalidates_exact_review(scope, monkeypatch):
    """Refuse an additional historical receipt after an exact-set review exists.

    Inputs: reviewed fixture plus new unbound receipt. Outputs: refusal with no
    model call. Effects: additive fixture files; choose for historical drift.
    """
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    params["max_model_calls"] = 8
    originals, _ = legacy_review_fixture(scope, prep)
    ai._save(ai._path(scope, "provider_attempt", "new-unreviewed"), scope, "provider_attempt",
             {"source": prep["source"], "response": {"error_type": "AnotherFailure"}})
    model = CheckpointModel()
    with pytest.raises(ai.ContentInvalid, match="accounting review differs"):
        ai.extract_candidates(params, model=model, config=config)
    assert model.calls == [] and all(path.read_bytes() == raw for path, raw in originals.items())


def test_legacy_review_wrong_pins_and_fingerprint_refuse(scope, monkeypatch):
    """Refuse a review whose immutable envelope is tampered or has another scope.

    Inputs: synthetic review mutation. Outputs: stage/pin/fingerprint refusal.
    Effects: modifies only disposable review fixture bytes; choose for read checks.
    """
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    _, ref = legacy_review_fixture(scope, prep)
    path = ai._path(scope, "provider_accounting_review", prep["source"])
    review = ai._read(ref, scope, "provider_accounting_review")
    path.chmod(0o600)
    path.write_bytes(ai._json({**review, "reason": "tampered"}))
    with pytest.raises(ai.ContentInvalid, match="fingerprint differs"):
        ai._provider_budget(scope, prep["source"])
    review["pins"] = {**scope, "source_version_id": str(uuid4())}
    review["bundle_fingerprint"] = ai._key({key: value for key, value in review.items() if key != "bundle_fingerprint"})
    path.write_bytes(ai._json(review))
    with pytest.raises(ai.ContentInvalid, match="pins differ"):
        ai._provider_budget(scope, prep["source"])


def test_historical_charge_reduces_source_ceiling_without_creating_slots(scope, monkeypatch):
    """Enforce the unchanged 512 ceiling with historical usage outside new slots.

    Inputs: 510 conservative historical charges and fresh requests. Outputs:
    exactly two real reservations and permanent third refusal. Effects: fixture
    ledgers only; choose for concurrent-safe reduced reservation namespace.
    """
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    legacy_review_fixture(scope, prep, charge=510)
    identity = ai.candidate_identity(params["prepared_ref"], params["work_products_ref"], config, ai.MAX_MODEL_CALLS)
    assert ai._provider_budget(scope, prep["source"]) == []
    for index in range(2):
        ai._reserve_provider_call(scope, prep["source"], identity, "fresh-chunk-" + str(index), {})
    assert ai._provider_accounting(scope, prep["source"]) == {
        "historical_provider_charge": 510, "new_provider_budget_consumed": 2,
        "provider_budget_consumed": 512, "model_calls": 512}
    with pytest.raises(ai.ContentInvalid, match="budget exhausted"):
        ai._reserve_provider_call(scope, prep["source"], identity, "fresh-chunk-3", {})
    assert [item["slot"] for item in ai._provider_budget(scope, prep["source"])] == [1, 2]


def test_review_preserves_two_fresh_requests_per_chunk(scope, monkeypatch):
    """Authorize fresh reviewed chunk requests without widening their two-slot cap.

    Inputs: six historical charges and fresh reviewed chunk. Outputs: two new
    intent/budget slots and third refusal. Effects: fixture files only; choose
    over manufacturing historical intents or resetting charged consumption.
    """
    params, prep, config = checkpoint_fixture(scope, monkeypatch)
    legacy_review_fixture(scope, prep)
    identity = ai.candidate_identity(params["prepared_ref"], params["work_products_ref"], config, 512)
    content_key = prep["chunks"][0]["content_key"]
    for _ in range(2):
        ai._reserve_provider_call(scope, prep["source"], identity, content_key, {})
    with pytest.raises(ai.ContentInvalid, match="two durable"):
        ai._reserve_provider_call(scope, prep["source"], identity, content_key, {})
    assert len(ai._chunk_intents(scope, content_key)) == 2
    assert ai._provider_accounting(scope, prep["source"])["provider_budget_consumed"] == 8


def semantic_router_fixture(scope, monkeypatch, *, failed=False, chunks=2):
    """Return malformed then partially grounded completions through fake actual SDK profiles.

    Inputs: fixture scope and all-failed option. Outputs: routed inputs/call log.
    Effects: synthetic SDK state only; choose for optional enrichment recovery.
    """
    params, router, clock, calls = routed_fixture(scope, monkeypatch, chunks=chunks)
    factory = router.client_factory
    def build(profile):
        """Wrap fake SDK; input approved profile, output client, effects fixture calls; choose for semantic rejection tests."""
        original = factory(profile)
        def create(**request):
            """Return controlled content; input request, output completion, effects fixture log; choose without model calls."""
            response = original.chat.completions.create(**request)
            if len(calls) <= 2:
                decoded = json.loads(response.choices[0].message.content[8:-4])
                raw = json.dumps({"wrong_shape": []}) if failed or len(calls) == 1 else json.dumps({
                    "candidates": [decoded["candidates"][0], {**decoded["candidates"][0], "quote": "not in this source"}]})
                response.choices[0].message.content = raw
                data = response.model_dump()
                data["choices"][0]["message"]["content"] = raw
                response.model_dump = lambda **kwargs: data
            return response
        return SimpleNamespace(chat=SimpleNamespace(completions=SimpleNamespace(create=create)))
    router.client_factory = build
    return params, router, calls


@pytest.mark.parametrize("failed", [False, True])
def test_semantic_failure_preserves_windows_alternates_primary_and_reports_rejections(scope, monkeypatch, failed):
    """Continue every real window after partial or failed optional enrichment.

    Inputs: malformed first response and corrected partial/failure response.
    Outputs: alternate primary, exact accepted subset, rejection/status citations
    and stable cached accounting. Effects: fake SDK and retained fixture files.
    """
    from server.analysis import ai_content_provider as provider
    params, router, calls = semantic_router_fixture(scope, monkeypatch, failed=failed)
    result = ai.extract_candidates(params, router=router)
    assert [request["model"] for request in calls[:2]] == list(provider.PRIMARY_MODELS)
    assert result["model_calls"] == 3 and result["chunks"] == 2
    bundle = ai._read(result["bundle_ref"], scope, "candidates")
    first = bundle["replies"][0]
    assert first["enrichment_status"] == ("failed_enrichment" if failed else "partial_candidates_rejected")
    assert first["grounded_candidates"] == (0 if failed else 1)
    assert len(first["rejected_candidates"]) == 1 and len(first["reply_validations"]) == 2
    assert all(item["provider_attempt_ref"] for item in first["reply_validations"])
    assert bundle["replies"][1]["enrichment_status"] == "grounded_candidates"
    prep = ai._read(params["prepared_ref"], scope, "prepared")
    embedded = {"model_id": "synthetic", "vectors": [[0.25] * 2048 for _ in prep["chunks"]]}
    objects = ai.search_objects({**params, "candidates_ref": result["bundle_ref"], "embeddings_ref": "file:///synthetic"},
                                prep, bundle, embedded, "AiChatEvents20260918", "text_nim")
    assert len(objects) == 2
    citation = json.loads(objects[0]["properties"]["provenance"][0])
    assert citation["enrichment_status"] == first["enrichment_status"] and citation["rejected_model_candidates"]
    assert len(citation["enrichment_reply_validations"]) == 2
    again = ai.extract_candidates(params, router=router)
    assert again["new_model_calls"] == 0 and len(calls) == 3


def test_existing_exhausted_replies_recover_partial_chunk_without_calls(scope, monkeypatch):
    """Recover a genuine exact subset from existing charged replies with no checkpoint.

    Inputs: two retained original semantic completions and durable claims.
    Outputs: partial checkpoint, rejection report and zero new calls. Effects:
    additive derived fixture files; choose for the real failed first-chunk case.
    """
    params, router, calls = semantic_router_fixture(scope, monkeypatch, chunks=1)
    prep = ai._read(params["prepared_ref"], scope, "prepared")
    chunk, source = prep["chunks"][0], prep["source"]
    identity = {"prepared_ref": params["prepared_ref"], "work_products_ref": params["work_products_ref"],
                "route_plan": router.identity(), "prompt_digest": ai._key([ai.PROMPT, ai.OUTPUT_CORRECTION]),
                "policy": "exact-quote-routed-checkpoint-v1", "maximum": 2, "requests_per_chunk": 2}
    originals = {}
    for repair in range(2):
        attempt_refs = []
        def reserve(request, actual):
            """Claim fixture request; input actual completion/profile, output refs, effects immutable fixture slots; choose before fake SDK."""
            return ai._reserve_provider_call(scope, source, identity, chunk["content_key"],
                {"completion": request, "actual_provider": actual})
        def retain(attempt):
            """Keep actual fixture response; input attempt, output none, effects retained file; choose for recovery evidence."""
            path = ai._path(scope, "provider_attempt", [attempt["intent_ref"], attempt["response"]])
            attempt_refs.append(ai._save(path, scope, "provider_attempt", attempt))
            originals[path] = path.read_bytes()
        prompt = ai._candidate_prompt(chunk) + (ai.OUTPUT_CORRECTION if repair else "")
        response = router.complete(scope, source, 0, prompt, reserve, retain, lambda _: None)
        path = ai._path(scope, "model_reply", [identity, chunk["content_key"], response["intent_ref"]])
        ai._save(path, scope, "model_reply", {"source": source, "identity": identity, "content_key": chunk["content_key"],
            "request": prompt, "raw_reply": response["raw_reply"], "actual_provider": response["actual_provider"],
            "finish_reason": response["finish_reason"], "reported_model": response["reported_model"], "usage": response["usage"],
            "intent_ref": response["intent_ref"], "budget_ref": response["budget_ref"], "provider_attempt_ref": attempt_refs[-1]})
        originals[path] = path.read_bytes()
    result = ai.extract_candidates(params, router=router)
    assert len(calls) == 2 and result["new_model_calls"] == 0 and result["model_calls"] == 2
    assert result["candidates"] == 1 and result["partial_enrichment_chunks"] == 1
    assert all(path.read_bytes() == raw for path, raw in originals.items())
    checkpoint = ai._read(ai._path(scope, "candidate_chunk", [identity, chunk["content_key"], ai._key(ai._candidate_prompt(chunk))]).as_uri(),
                          scope, "candidate_chunk")
    assert checkpoint["validation"]["rejected_candidates"][0]["index"] == 1
    path = ai._path(scope, "candidate_chunk", [identity, chunk["content_key"], ai._key(ai._candidate_prompt(chunk))])
    path.chmod(0o600)
    checkpoint["validation"]["rejected_candidates"] = []
    checkpoint["bundle_fingerprint"] = ai._key({key: value for key, value in checkpoint.items() if key != "bundle_fingerprint"})
    path.write_bytes(ai._json(checkpoint))
    with pytest.raises(ai.ContentInvalid, match="re-grounded"):
        ai.extract_candidates(params, router=router)
    assert len(calls) == 2
