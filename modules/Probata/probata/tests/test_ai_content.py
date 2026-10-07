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
import os
from pathlib import Path
from types import SimpleNamespace
from uuid import uuid4

import httpx
import pytest

from server.analysis import ai_content as ai


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
            **{key: str(uuid4()) for key in ("source_version_id", "normalized_generation_id", "verification_id", "matter_id", "court_case_id")},
            "operating_mode": "LIVE"}


def record(ordinal, body, *, conversation=0, native_id="native-conversation"):
    """Build a synthetic normalized occurrence with actual native locators.

    Inputs: ordinal/body/native conversation. Outputs: record. Effects: none;
    choose for source-span and conversation-boundary assertions.
    """
    return {"record_id": str(uuid4()), "ordinal": ordinal, "body": body,
            "conversation_index": conversation, "conversation_id": native_id,
            "native_message_id": "native-node-" + str(ordinal), "mapping_key": "slot-" + str(ordinal),
            "native_message_index": None, "conversation_title": None, "role": "assistant",
            "occurred_at": None, "raw_occurrences": [{"raw_record_id": str(uuid4()), "raw_ordinal": ordinal, "role": "direct"}]}


def source(scope):
    """Bind a synthetic retained original to the exact common scope.

    Inputs: scope. Outputs: metadata binding. Effects: none; choose instead of
    creating real source rows for pure provider/publication tests.
    """
    return {"original_uri": "b2://synthetic/conversations.json#version=fixture", "original_sha256": "a" * 64,
            "original_bytes": 476547, "format_id": "chatgpt_json_array", "raw_generation_id": str(uuid4())}


def prepared(scope, monkeypatch, records=None):
    """Retain one synthetic prepared bundle with complete source occurrences.

    Inputs: scope and optional records. Outputs: reference/bundle. Effects:
    retained files and fake read-only binding; choose for downstream stage tests.
    """
    records = records or [record(0, "Alice drafted the motion."), record(1, "The hearing is on 2026-10-07.")]
    bound = source(scope)
    monkeypatch.setattr(ai, "_bound_source", lambda _: bound)
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


def test_windows_preserve_every_text_occurrence_without_per_message_objects(scope):
    """Prove complete source reconstruction and coherent grouping; input fixture scope, output assertions, no external effects; choose over object-count-only tests."""
    rows = [record(0, "alpha\n\n" + "x" * 15000), record(1, "alpha alpha"), record(2, ""), record(3, "other", conversation=1)]
    chunks = ai.conversation_windows(rows, 128)
    assert len(chunks) == 5
    for row in rows:
        segments = [s for c in chunks for s in c["segments"] if s["record_id"] == row["record_id"]]
        assert "".join(s["text"] for s in segments) == row["body"]
        assert all(s["text"] == row["body"][s["body_start"]:s["body_end"]] for s in segments)
    assert all(len(c["text"]) <= ai.CHUNK_CHARS for c in chunks)
    short = ai.conversation_windows([record(i, "short text") for i in range(132)], 128)
    assert len(short) == 1
    assert len(short[0]["segments"]) == 132


def test_chunk_bound_missing_native_coordinate_and_empty_only_fail(scope):
    """Reject incomplete grouping and exceeded bounds; input fixture scope, output assertions, no external effects; choose for fail-closed admission."""
    for rows, maximum in [([record(0, "x" * 15000)], 1), ([{**record(0, "x"), "conversation_index": None}], 128), ([record(0, "")], 128)]:
        with pytest.raises(ai.ContentInvalid):
            ai.conversation_windows(rows, maximum)


def test_exact_grounding_preserves_repeated_and_overlapping_occurrences(scope):
    """Prove exact repeated quote locators; input synthetic source, output assertions, no external effects; choose over deduplicated candidate checks."""
    row = record(7, "aaaa; Alice drafted a motion; Alice drafted a motion")
    chunk = ai.conversation_windows([row], 128)[0]
    candidates = ai.ground_candidates(chunk, {"candidates": [{"kind": "artifact", "title": "Motion", "record_id": row["record_id"], "quote": "Alice drafted a motion"},
        {"kind": "entity", "title": "Repeated letters", "record_id": row["record_id"], "quote": "aa"}]})
    assert [len(c["occurrences"]) for c in candidates] == [2, 3]
    for candidate in candidates:
        for occurrence in candidate["occurrences"]:
            assert row["body"][occurrence["body_start"]:occurrence["body_end"]] == candidate["quote"]
            assert occurrence["mapping_key"] == "slot-7" and occurrence["raw_occurrences"] == row["raw_occurrences"]
            assert occurrence["occurred_at"] is None


@pytest.mark.parametrize("quote,record_id", [("fabricated", None), ("Alice", "wrong-record"), ("", None)])
def test_ungrounded_candidates_fail_closed(scope, quote, record_id):
    """Reject fabricated or misbound quotes; inputs fixture scope and quote, output assertions, no external effects; choose for candidate grounding."""
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
    for wrong, stage in [({**scope, "verification_id": str(uuid4())}, "prepared"), (scope, "embedded")]:
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


def test_publication_and_fresh_readback_preserve_generation_source_and_no_turn_objects(scope, monkeypatch):
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


def test_scope_dev_nil_and_mismatched_generation_rejected(scope, monkeypatch):
    """Reject mismatched scope before downstream use; inputs invalid pins, output assertions, effects retained fixtures; choose for independent admission."""
    with pytest.raises(ai.ContentInvalid):
        ai.pins({**scope, "operating_mode": "DEV"})
    with pytest.raises(ai.ContentInvalid):
        ai.pins({**scope, "source_version_id": "00000000-0000-0000-0000-000000000000"})
    params, _, _, _ = dependencies(scope, monkeypatch)
    with pytest.raises(ai.ContentInvalid):
        ai.publication_inputs({**params, "normalized_generation_id": str(uuid4())})


def test_rest_schema_read_only_incompatible_type_and_add_contract(scope):
    """Prove REST schema and additive collision contracts; input mocked HTTP, output assertions, no network effects; choose for client serialization."""
    calls, objects = [], {}
    schema = {"properties": [{"name": n, "dataType": [t]} for n, t in ai.SEARCH_PROPERTIES.items()],
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
    second = record(1, "~~~sql\nSELECT 1;\n")
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
        original = row if product["locator"]["record_id"] == row["record_id"] else second
        assert product["content"] == original["body"][product["body_start"]:product["body_end"]]
        assert product["locator"]["raw_occurrences"] == original["raw_occurrences"]
    assert ai.extract_work_products({**scope, "prepared_ref": ref})["bundle_ref"] == result["bundle_ref"]
    drafts = ai.full_work_product_spans(record(2, "DRAFT One\nComplete first.\nSubject: Two\nComplete second.\n"))
    assert [d["content"] for d in drafts] == ["DRAFT One\nComplete first.\n", "Subject: Two\nComplete second.\n"]


def test_source_binding_verified_open_count_and_durable_case_scope(scope):
    """Prove verified OPEN source admission and durable case binding; input metadata fixture, output assertions, no DB effects; choose for pre-preview lifecycle."""
    expected = {"member_count": 132, "normalized_generation_manifest_digest": "f" * 64,
                "construction": "normalized-manifest-fixture", "verification_mode": "independent_recomputation"}
    row = {**source(scope), "workflow_id": scope["request_id"], "source_status": "retained",
           "declared_format": "chatgpt_json_array", "matter_id": scope["matter_id"], "court_case_id": scope["court_case_id"],
           "admission": {k: scope[k] for k in ("operating_mode", "matter_id", "court_case_id")},
           "verification_status": "success", "verification_expected": expected,
           "verification_observed": {k: v for k, v in expected.items() if k != "member_count"}, "record_count": 132,
           "source_key": "b2://synthetic/conversations.json#exact-native-version"}

    class Connection:
        """Expose a single exact metadata result without payload reads.

        Inputs: SQL query/pins. Outputs: bound row. Effects: fixture assertions;
        choose for binding tests independently from source extraction tests.
        """
        def execute(self, query, values):
            """Check exact generation/source/verification lookup coordinates.

            Inputs: SQL and params. Outputs: result facade. Effects: assertions.
            """
            assert values == {"source": scope["source_version_id"], "generation": scope["normalized_generation_id"], "verification": scope["verification_id"]}
            assert "JOIN context.source src" in str(query)
            assert "g.status" not in str(query)  # verified OPEN is the existing pre-preview seam
            return SimpleNamespace(mappings=lambda: SimpleNamespace(one_or_none=lambda: row))

    assert ai.source_binding(Connection(), scope)["source_key"].endswith("exact-native-version")
    for field, value in [("record_count", 133), ("matter_id", str(uuid4())), ("verification_status", "failed"), ("declared_format", "codex_rollout_jsonl")]:
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
    with pytest.raises(ai.ContentInvalid, match="coherent conversation-window"):
        ai.publication_inputs(forged)


def test_prepare_uses_exact_reader_ordinals_and_accounts_empty_records(scope, monkeypatch):
    """Prove preparation accounts exact reader rows; input fake read-only generation, output assertions, effects retained files; choose for bounded preparation."""
    from server.context_chunks import db
    from server.tools.extractors.entity_events import pages
    rows = [record(0, "full source text"), record(1, "")]
    bound = source(scope)
    monkeypatch.setattr(ai, "source_binding", lambda conn, pin: bound)
    calls = []
    def reader(conn, generation, after, limit):
        """Return exact synthetic reader rows and capture generation admission.

        Inputs: reader coordinates. Outputs: message rows. Effects: fixture log;
        choose for testing the existing reader seam without database mutation.
        """
        calls.append((generation, after, limit))
        return [pages.Message(r["record_id"], r["ordinal"], None, r["body"]) for r in rows]
    monkeypatch.setattr(pages, "read_window", reader)
    totals = {"records": 2, "text_bytes": len(rows[0]["body"].encode()), "other_records": 0}
    class Connection:
        """Provide exact totals and native locators from a synthetic generation.

        Inputs: query. Outputs: fixture rows. Effects: none; choose for reader scope.
        """
        def execute(self, query, values):
            """Answer only exact-generation totals and locator queries.

            Inputs: SQL/pins. Outputs: mappings. Effects: assertions only.
            """
            if "other_records" in str(query):
                assert values == {"g": scope["normalized_generation_id"]}
                return SimpleNamespace(mappings=lambda: SimpleNamespace(one=lambda: totals))
            assert values == {"g": scope["normalized_generation_id"], "limit": 3}
            return SimpleNamespace(mappings=lambda: [{"record_id": r["record_id"],
                "native_fields": {"conversation_id": r["conversation_id"], "message_id": r["native_message_id"], "source_role": r["role"]},
                "native_metadata": {"conversation_index": r["conversation_index"], "mapping_key": r["mapping_key"]},
                "raw_occurrences": r["raw_occurrences"]} for r in rows])
    @contextmanager
    def connection():
        """Yield the retained fixture's read-only connection facade.

        Inputs: none. Outputs: fake connection. Effects: none; choose for tests.
        """
        yield Connection()
    monkeypatch.setattr(db, "read_only_connection", connection)
    result = ai.prepare_content(scope)
    assert calls == [(scope["normalized_generation_id"], -1, 2)]
    bundle = ai._read(result["bundle_ref"], scope, "prepared")
    assert result["records"] == 2 and result["chunks"] == 1
    assert bundle["empty_record_ids"] == [rows[1]["record_id"]]
    assert [r["ordinal"] for r in bundle["records"]] == [0, 1]
    totals["records"] = 1025
    calls.clear()
    with pytest.raises(ai.ContentInvalid):
        ai.prepare_content(scope)
    assert calls == []


def test_native_coordinates_support_exact_sbv_shape_and_direct_fallback():
    """Preserve both known metadata shapes; inputs synthetic native IDs, outputs exact coordinates, effects none; choose for SBV grouping regression."""
    nested = {"conversation_index": 0, "conversation_id": "native-c", "conversation_title": "Native title",
              "message_id": "native-m", "node_id": "slot-7", "message_index": 0, "role": "assistant"}
    expected = {"conversation_index": 0, "conversation_id": "native-c", "conversation_title": "Native title",
                "native_message_id": "native-m", "mapping_key": "slot-7", "native_message_index": 0, "role": "assistant"}
    assert ai.native_coordinates({}, {"sbv_kind": "message", "sbv_source_pos": "display string ignored", "source_metadata": nested}) == expected
    fields = {"conversation_id": "native-c", "conversation_title": "Native title", "message_id": "native-m", "source_role": "assistant"}
    direct = {"conversation_index": 0, "mapping_key": "slot-7", "message_index": 0}
    assert ai.native_coordinates(fields, direct) == expected
    assert ai.native_coordinates({}, {"sbv_source_pos": "conversation=23/message=42"})["conversation_index"] is None
    with pytest.raises(ai.ContentInvalid):
        ai.native_coordinates({}, {"source_metadata": ["not an object"]})


def test_exact_generation_pages_612_records_without_losing_occurrences(scope, monkeypatch):
    """Account three bounded pages and every locator; inputs synthetic SBV generation, outputs ordered assertions, effects retained fixture only; choose for the actual-size reader regression."""
    from server.tools.extractors.entity_events import pages
    rows = [record(i, "body-" + str(i), conversation=i // 27) for i in range(612)]
    calls = []
    def reader(conn, generation, after, limit):
        """Return one exact keyset page; inputs pinned coordinates, outputs bounded rows, effects captured calls; choose for reader coverage."""
        calls.append((generation, after, limit))
        return [pages.Message(r["record_id"], r["ordinal"], None, r["body"]) for r in rows if r["ordinal"] > after][:limit]
    monkeypatch.setattr(pages, "read_window", reader)
    metadata = [{"record_id": r["record_id"], "native_fields": {}, "native_metadata": {"source_metadata": {
        "conversation_index": r["conversation_index"], "conversation_id": r["conversation_id"],
        "message_id": r["native_message_id"], "node_id": r["mapping_key"], "message_index": r["ordinal"], "role": r["role"]}},
        "raw_occurrences": r["raw_occurrences"]} for r in rows]
    class Connection:
        """Serve bounded exact metadata; inputs SELECT, outputs fixture locators, effects assertions; choose for paging tests."""
        def execute(self, query, values):
            """Check metadata scope; inputs SQL and pins, outputs mappings, effects assertions; choose for source binding."""
            assert values == {"g": scope["normalized_generation_id"], "limit": 613}
            return SimpleNamespace(mappings=lambda: metadata)
    result = ai.read_generation_records(Connection(), scope, 612)
    assert calls == [(scope["normalized_generation_id"], -1, 256), (scope["normalized_generation_id"], 255, 256),
                     (scope["normalized_generation_id"], 511, 100)]
    assert [r["record_id"] for r in result] == [r["record_id"] for r in rows]
    assert [r["body"] for r in result] == [r["body"] for r in rows]
    assert all(r["raw_occurrences"] == original["raw_occurrences"] and r["native_message_id"] == original["native_message_id"]
               and r["mapping_key"] == original["mapping_key"] for r, original in zip(result, rows))
    assert len(ai.conversation_windows(result, ai.MAX_CHUNKS)) == 23
    monkeypatch.setattr(pages, "read_window", lambda *args: [])
    with pytest.raises(ai.ContentInvalid, match="every normalized record"):
        ai.read_generation_records(Connection(), scope, 612)
    monkeypatch.setattr(pages, "read_window", lambda *args: [pages.Message(rows[0]["record_id"], -1, None, "body")])
    with pytest.raises(ai.ContentInvalid, match="non-increasing"):
        ai.read_generation_records(Connection(), scope, 612)
    from server.temporal.ai_content_activities import AIContentParams
    defaults = AIContentParams()
    assert (defaults.max_records, defaults.max_text_bytes, defaults.max_chunks, defaults.max_model_calls) == (1024, 2097152, 256, 512)
    assert ai.VERSION == "ai-content-v2"


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
    retained = list(ai._root().glob("*/*/model_reply/*.json"))
    assert len(retained) == 2
    assert {json.loads(path.read_bytes())["repair_attempt"] for path in retained} == {0, 1}
