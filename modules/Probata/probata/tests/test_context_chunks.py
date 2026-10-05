"""Unit tests for server/context_chunks: chunk spans, the overlap rule, ids, thread replacement, the REST shapes.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
Updated: Codex · GPT-5 · 2026-10-05 — configure neutral scope in the mutation starter unit fixture.

The chunker's model is replaced by a fake that cuts at known offsets (the span arithmetic is what is tested); the real
distilbert run is opt-in (CONTEXT_CHUNKS_TEST_NEURAL=1). Live proof (Postgres, NIM, Weaviate) is the deploy's job.
"""

from __future__ import annotations

import json
import os
import random
import uuid
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from itertools import pairwise

import httpx
import pytest

from server.context_chunks import ids
from server.context_chunks.chunker import (
    CHUNKERS,
    chunk_spans,
    chunker_version,
    firsts_from_offsets,
    line_starts,
    spans_from_firsts,
    widen,
)
from server.context_chunks.config import EMBED_DIMENSIONS, ChunkConfig, load_config
from server.context_chunks.embed import EmbedError, NimEmbedder
from server.context_chunks.model import CORPUS_FIRST_PARTY, CallFile, Message, Thread, ThreadPlan, ThreadRef
from server.context_chunks.render import BLANK_PLACEHOLDER, nim_input, one_line, render_line
from server.context_chunks.service import (
    PlanStale,
    build_call_file_object,
    build_chunk_objects,
    plan_thread,
    publish_call_files,
    publish_thread,
    thread_lines,
)
from server.context_chunks.source import Resolver
from server.context_chunks.store import PROPERTIES, RECORD_KIND_CALL_FILE, ChunkStore

T0 = datetime(2026, 1, 5, 14, 30, tzinfo=UTC)


# ------------------------------------------------------------------ fakes
@dataclass
class FakeChunk:
    start_index: int


class CutAt:
    """A chunker that starts a new chunk at every message index in ``cuts`` (offsets from the joined text)."""

    def __init__(self, cuts: list[int], lines_ref: list[str]):
        self.cuts, self.lines = cuts, lines_ref

    def chunk(self, text: str):
        starts = line_starts(self.lines)
        return [FakeChunk(starts[c]) for c in self.cuts]


def make_thread(n: int, thread_id: str = "t-1", *, prefix: str = "m") -> Thread:
    messages = [
        Message(
            id=f"{prefix}{i:04d}",
            at=T0 + timedelta(minutes=i),
            sender="Alice" if i % 2 else "Bob",
            body=f"message number {i}",
            source_version_id="sv-1",
            sender_entity_id="e-a" if i % 2 else "e-b",
            participant_entity_ids=["e-a", "e-b"],
            participant_names=["Alice", "Bob"],
        )
        for i in range(n)
    ]
    return Thread(ref=ThreadRef(CORPUS_FIRST_PARTY, thread_id), matter_id="matter-1", messages=messages)


class FakeSource:
    def __init__(self, thread: Thread, calls: dict[str, CallFile] | None = None):
        self.thread, self.calls = thread, calls or {}

    def load_thread(self, ref):
        assert ref == self.thread.ref
        return self.thread

    def call_source_versions(self, source_version_id=None):
        return [k for k in sorted(self.calls) if source_version_id in (None, k)]

    def load_call_file(self, source_version_id):
        return self.calls[source_version_id]


class FakeEmbedder:
    model = "nvidia/nemotron-3-embed-1b"

    def __init__(self):
        self.calls, self.texts = 0, []

    def embed(self, texts):
        self.calls += 1
        self.texts += texts
        return [[0.0] * 4 for _ in texts]


class FakeStore:
    def __init__(self):
        self.objects: dict[str, dict] = {}
        self.ensured = 0

    def ensure_collection(self):
        self.ensured += 1
        return []

    def vector_names(self):
        return ["text_nim"]

    def upsert(self, objects):
        for o in objects:
            self.objects[o["id"]] = o
        return len(objects)

    def delete_other_generations(self, thread_id, digest):
        stale = [
            k
            for k, o in self.objects.items()
            if o["properties"].get("thread_id") == thread_id and o["properties"]["thread_digest"] != digest
        ]
        for k in stale:
            del self.objects[k]
        return len(stale)

    def has_generation(self, thread_id, digest):
        return any(
            o["properties"].get("thread_id") == thread_id and o["properties"]["thread_digest"] == digest
            for o in self.objects.values()
        )


def plan_with_cuts(thread: Thread, cuts: list[int], overlap: int = 2) -> ThreadPlan:
    lines = thread_lines(thread)
    return plan_thread(thread, "token_1000", overlap, engine=CutAt(cuts, lines))


# ------------------------------------------------------------------ render and guards
def test_render_line_is_one_line_per_message_in_utc():
    assert render_line(T0, "Alice", "hi\nthere  you") == "[2026-01-05 14:30] Alice: hi there you"
    detroit = T0.astimezone(timezone_east())
    assert render_line(detroit, "Bob", "x") == "[2026-01-05 14:30] Bob: x"
    assert render_line(None, "", "") == "[unknown time] Unknown: (attachment)"
    assert "\n" not in render_line(T0, "A\nB", "a\r\nb")


def timezone_east():
    from datetime import timezone

    return timezone(timedelta(hours=-5))


def test_nim_input_guards():
    assert nim_input("see data:image/png;base64,AAA") == "see data: image/png;base64,AAA"
    assert nim_input("   \n") == BLANK_PLACEHOLDER
    assert nim_input("") == BLANK_PLACEHOLDER
    assert len(nim_input("x" * 9000)) == 8000
    assert one_line(None) == ""


# ------------------------------------------------------------------ spans and the overlap rule
def test_firsts_from_offsets_maps_to_messages_and_merges_mid_message_cuts():
    lines = ["aaaa", "bbbb", "cccc", "dddd"]
    starts = line_starts(lines)  # 0, 5, 10, 15
    assert firsts_from_offsets([0, 5, 15], starts) == [0, 1, 3]
    assert firsts_from_offsets([7, 12], starts) == [0, 1, 2]  # cuts inside messages 1 and 2; 0 is added
    assert firsts_from_offsets([5, 6, 7], starts) == [0, 1]  # three cuts in one message merge
    assert firsts_from_offsets([], starts) == [0]


def test_spans_partition_the_thread_then_widen_forward():
    spans = spans_from_firsts([0, 4, 9], 12)
    assert spans == [(0, 3), (4, 8), (9, 11)]
    assert widen(spans, 12, 2) == [(0, 5), (4, 10), (9, 11)]
    assert widen(spans, 12, 0) == spans


@pytest.mark.parametrize("seed", range(40))
def test_every_chunk_overlaps_the_next_by_at_least_two_messages(seed):
    rng = random.Random(seed)
    n = rng.randint(2, 80)
    cuts = sorted({0, *rng.sample(range(1, n), k=rng.randint(0, min(n - 1, 25)))})
    lines = [f"line {i}" for i in range(n)]
    spans = chunk_spans(lines, chunker=CutAt(cuts, lines), overlap=2)
    assert spans[0][0] == 0 and spans[-1][1] == n - 1
    covered = {i for a, b in spans for i in range(a, b + 1)}
    assert covered == set(range(n))  # nothing dropped
    for (a1, b1), (a2, b2) in pairwise(spans):
        shared = len(set(range(a1, b1 + 1)) & set(range(a2, b2 + 1)))
        assert shared >= min(2, n - a2), (spans, shared)
        assert a1 < a2  # chunk starts still advance


def test_chunk_spans_edge_cases():
    assert chunk_spans([], chunker=object()) == []
    assert chunk_spans(["one"], chunker=object()) == [(0, 0)]
    lines = ["a", "b"]
    assert chunk_spans(lines, chunker=CutAt([0, 1], lines), overlap=2) == [(0, 1), (1, 1)]


def test_toolbox_keeps_the_other_chonkie_chunkers_selectable_and_versions_name_the_construction():
    assert {
        "neural_distilbert",
        "neural_modernbert",
        "token_1000",
        "fast_1000",
        "sentence_1000",
        "recursive_1000",
    } <= set(CHUNKERS)
    v = chunker_version("neural_distilbert", 2)
    assert v.startswith("neural_distilbert|mirth/chonky_distilbert_base_uncased_1|chonkie-") and "overlap=2" in v
    assert chunker_version("neural_distilbert", 3) != v
    with pytest.raises(ValueError):
        chunker_version("nope")


# ------------------------------------------------------------------ ids
def test_chunk_object_id_is_uuid5_over_thread_first_last_and_chunker_version():
    expected = str(uuid.uuid5(ids.CHUNK_NAMESPACE, "proffer-chunk-v1|T|F|L|V"))
    assert ids.chunk_object_id("T", "F", "L", "V") == expected
    assert ids.chunk_object_id("T", "F", "L", "V") == expected  # stable
    assert ids.chunk_object_id("T", "F", "L", "V2") != expected
    assert ids.chunk_object_id("T", "F", "L2", "V") != expected
    assert ids.chunk_object_id("T2", "F", "L", "V") != expected


def test_thread_digest_binds_order_membership_and_version():
    base = ids.thread_digest("T", ["a", "b", "c"], "V")
    assert base == ids.thread_digest("T", ["a", "b", "c"], "V")
    assert base != ids.thread_digest("T", ["a", "c", "b"], "V")
    assert base != ids.thread_digest("T", ["a", "b"], "V")
    assert base != ids.thread_digest("T", ["a", "b", "c"], "V2")


# ------------------------------------------------------------------ chunk objects link back to Postgres
def test_chunk_objects_carry_the_postgres_message_ids_and_metadata():
    thread = make_thread(12)
    plan = plan_with_cuts(thread, [0, 5, 9])
    assert plan.spans == [[0, 6], [5, 10], [9, 11]]
    objects = build_chunk_objects(thread, plan, embed_model="m", now=T0)
    assert len(objects) == 3
    first = objects[0]["properties"]
    assert first["message_ids"] == [f"m{i:04d}" for i in range(7)]
    assert first["first_message_id"] == "m0000" and first["last_message_id"] == "m0006"
    assert first["thread_id"] == "t-1" and first["corpus"] == CORPUS_FIRST_PARTY
    assert first["matter_id"] == "matter-1" and first["source_version_ids"] == ["sv-1"]
    assert first["participant_entity_ids"] == ["e-a", "e-b"] and first["participant_names"] == ["Alice", "Bob"]
    assert first["start_at"] == "2026-01-05T14:30:00Z" and first["end_at"] == "2026-01-05T14:36:00Z"
    assert first["chunker"] == "token_1000" and first["overlap"] == 2 and first["message_count"] == 7
    assert first["text"].split("\n")[0] == "[2026-01-05 14:30] Bob: message number 0"
    assert len(first["text"].split("\n")) == 7  # one line per message
    assert first["thread_digest"] == plan.digest and first["record_kind"] == "conversation_chunk"
    # the object id is the owner's construction, from the chunk's own first and last message
    assert objects[1]["id"] == ids.chunk_object_id("t-1", "m0005", "m0010", plan.chunker_version)
    # every message is in at least one chunk and the overlap shares messages
    assert {m for o in objects for m in o["properties"]["message_ids"]} == {m.id for m in thread.messages}
    assert set(objects[0]["properties"]["message_ids"]) & set(objects[1]["properties"]["message_ids"]) == {
        "m0005",
        "m0006",
    }


def test_plan_is_references_only():
    thread = make_thread(6)
    plan = plan_with_cuts(thread, [0, 3])
    blob = json.dumps(plan.to_dict())
    assert "message number" not in blob and "Alice" not in blob  # indexes and a digest, never message text
    assert ThreadPlan.from_dict(json.loads(blob)) == plan


# ------------------------------------------------------------------ publish and re-chunk replacement
def test_publish_thread_embeds_in_batches_and_is_idempotent():
    thread = make_thread(100)
    plan = plan_with_cuts(thread, list(range(0, 100, 5)))
    source, store, emb = FakeSource(thread), FakeStore(), FakeEmbedder()
    result = publish_thread(source, store, emb, plan, batch=8)
    assert result.chunks_written == 20 and len(store.objects) == 20
    assert emb.calls == 3 and len(emb.texts) == 20  # 8 + 8 + 4
    again = publish_thread(source, store, emb, plan, batch=8)  # resume: already current, nothing embedded
    assert again.skipped_existing and again.chunks_written == 0 and emb.calls == 3
    forced = publish_thread(source, store, emb, plan, resume=False, batch=8)  # same ids: replaced in place
    assert forced.chunks_written == 20 and len(store.objects) == 20


def test_a_thread_that_grew_is_rechunked_whole_and_its_old_chunks_are_replaced():
    store, emb = FakeStore(), FakeEmbedder()
    small = make_thread(10)
    publish_thread(FakeSource(small), store, emb, plan_with_cuts(small, [0, 4]))
    old_ids = set(store.objects)
    other = make_thread(6, "t-other", prefix="x")  # another thread's chunks must survive
    publish_thread(FakeSource(other), store, emb, plan_with_cuts(other, [0, 3]))

    grown = make_thread(16)  # later chunks of the same backup: the thread's message list is longer
    plan = plan_with_cuts(grown, [0, 6, 11])
    result = publish_thread(FakeSource(grown), store, emb, plan)
    assert result.stale_deleted == len(old_ids) and not (old_ids & set(store.objects))
    mine = [o for o in store.objects.values() if o["properties"]["thread_id"] == "t-1"]
    assert {o["properties"]["thread_digest"] for o in mine} == {plan.digest}
    assert {m for o in mine for m in o["properties"]["message_ids"]} == {m.id for m in grown.messages}
    assert len([o for o in store.objects.values() if o["properties"]["thread_id"] == "t-other"]) == 2


def test_publish_refuses_a_plan_whose_thread_changed():
    store, emb = FakeStore(), FakeEmbedder()
    thread = make_thread(10)
    plan = plan_with_cuts(thread, [0, 5])
    with pytest.raises(PlanStale):
        publish_thread(FakeSource(make_thread(11)), store, emb, plan)
    assert not store.objects and emb.calls == 0


# ------------------------------------------------------------------ call-log files
def test_one_object_per_call_log_file_linking_its_call_ids():
    files = {
        "sv-a": CallFile(
            "sv-a",
            "matter-1",
            ["c1", "c2"],
            ["[2022-09-24 04:02] missed call from Kat, 0s", "[2022-09-25 10:00] outgoing call to Kat, 62s"],
            T0,
            T0 + timedelta(days=1),
            ["e-k"],
            ["Kat"],
        ),
        "sv-b": CallFile(
            "sv-b", "matter-1", ["c3"], ["[2022-10-01 09:00] incoming call from Bob, 5s"], T0, T0, [], ["Bob"]
        ),
    }
    store, emb = FakeStore(), FakeEmbedder()
    out = publish_call_files(FakeSource(make_thread(2), files), store, emb)
    assert out == {"files": 2, "calls": 3} and len(store.objects) == 2  # one entry per file, not per call
    obj = build_call_file_object(files["sv-a"], embed_model="m", now=T0)
    p = obj["properties"]
    assert p["record_kind"] == RECORD_KIND_CALL_FILE and p["call_log_ids"] == ["c1", "c2"]
    assert p["source_version_id"] == "sv-a" and p["text"].count("\n") == 1
    assert obj["id"] == ids.call_file_object_id("sv-a")
    only = publish_call_files(FakeSource(make_thread(2), files), FakeStore(), FakeEmbedder(), "sv-b")
    assert only == {"files": 1, "calls": 1}


# ------------------------------------------------------------------ the REST boundary
def _client(handler) -> httpx.Client:
    return httpx.Client(transport=httpx.MockTransport(handler))


def test_collection_schema_carries_the_properties_the_owner_listed():
    names = {p["name"] for p in PROPERTIES}
    assert {
        "message_ids",
        "thread_id",
        "corpus",
        "participant_entity_ids",
        "participant_names",
        "start_at",
        "end_at",
        "source_version_ids",
        "matter_id",
        "chunker",
        "chunker_version",
        "overlap",
        "text",
    } <= names
    kinds = {p["name"]: p["dataType"] for p in PROPERTIES}
    assert kinds["message_ids"] == ["text[]"] and kinds["start_at"] == ["date"] and kinds["overlap"] == ["int"]


def test_ensure_collection_creates_text_nim_with_no_vectorizer_when_absent():
    seen = {}

    def handler(request: httpx.Request) -> httpx.Response:
        if request.method == "GET":
            return httpx.Response(404)
        seen["body"] = json.loads(request.content)
        return httpx.Response(200, json={})

    store = ChunkStore("http://w:8082", "ProfferChunks20261002", http=_client(handler))
    added = store.ensure_collection()
    body = seen["body"]
    assert body["class"] == "ProfferChunks20261002" and set(added) == {p["name"] for p in PROPERTIES}
    assert body["vectorConfig"]["text_nim"]["vectorizer"] == {"none": {}}
    assert body["vectorConfig"]["text_nim"]["vectorIndexConfig"] == {"distance": "cosine"}


def test_ensure_collection_only_adds_missing_properties_and_refuses_a_conflicting_type():
    posted = []
    have = [{"name": p["name"], "dataType": p["dataType"]} for p in PROPERTIES if p["name"] != "overlap"]

    def handler(request: httpx.Request) -> httpx.Response:
        if request.method == "GET":
            return httpx.Response(200, json={"vectorConfig": {"text_nim": {}}, "properties": have})
        posted.append(json.loads(request.content)["name"])
        return httpx.Response(200, json={})

    assert ChunkStore("http://w", "C", http=_client(handler)).ensure_collection() == ["overlap"]
    have[0] = {"name": have[0]["name"], "dataType": ["int"]}
    with pytest.raises(Exception, match="want"):
        ChunkStore("http://w", "C", http=_client(handler)).ensure_collection()


def test_upsert_writes_the_named_vector_and_fails_closed_on_a_rejected_object():
    sent = []

    def ok(request):
        body = json.loads(request.content)
        sent.append(body)
        return httpx.Response(200, json=[{"id": o["id"], "result": {"status": "SUCCESS"}} for o in body["objects"]])

    objs = [{"id": f"id{i}", "properties": {"text": "t"}, "vector": [0.1, 0.2]} for i in range(250)]
    assert ChunkStore("http://w", "C", http=_client(ok), batch_size=100).upsert(objs) == 250
    assert [len(b["objects"]) for b in sent] == [100, 100, 50]
    assert sent[0]["objects"][0]["vectors"] == {"text_nim": [0.1, 0.2]} and sent[0]["objects"][0]["class"] == "C"

    def reject(request):
        body = json.loads(request.content)
        return httpx.Response(
            200, json=[{"id": o["id"], "result": {"errors": {"error": [{"message": "boom"}]}}} for o in body["objects"]]
        )

    with pytest.raises(Exception, match="boom"):
        ChunkStore("http://w", "C", http=_client(reject)).upsert(objs[:2])


def test_delete_other_generations_filters_on_thread_and_a_different_digest():
    calls = []

    def handler(request):
        body = json.loads(request.content)
        calls.append(body)
        return httpx.Response(
            200,
            json={
                "results": {
                    "matches": 3 if len(calls) == 1 else 0,
                    "successful": 3 if len(calls) == 1 else 0,
                    "failed": 0,
                }
            },
        )

    n = ChunkStore("http://w", "C", http=_client(handler)).delete_other_generations("T", "D")
    assert n == 3 and len(calls) == 2  # loops until nothing matches
    where = calls[0]["match"]["where"]
    assert calls[0]["match"]["class"] == "C" and calls[0]["dryRun"] is False
    assert {"path": ["thread_id"], "operator": "Equal", "valueText": "T"} in where["operands"]
    assert {"path": ["thread_digest"], "operator": "NotEqual", "valueText": "D"} in where["operands"]


# ------------------------------------------------------------------ the NIM embedder
def _cfg(**kw) -> ChunkConfig:
    return ChunkConfig(weaviate_url="http://w", api_key="secret-key", **kw)


def test_embedder_batches_applies_the_input_guards_and_never_leaks_the_key():
    requests = []

    def handler(request):
        body = json.loads(request.content)
        requests.append((request, body))
        data = [{"index": i, "embedding": [0.0] * EMBED_DIMENSIONS} for i in range(len(body["input"]))]
        return httpx.Response(200, json={"data": list(reversed(data))})  # out of order: sorted back by index

    emb = NimEmbedder(_cfg(embed_batch=2), http=_client(handler), sleep=lambda s: None)
    out = emb.embed(["a data:image/png;base64,X", "", "plain", "x" * 9000, "last"])
    assert len(out) == 5 and emb.calls == 3 and emb.texts == 5
    inputs = [i for _, b in requests for i in b["input"]]
    assert inputs[0] == "a data: image/png;base64,X" and inputs[1] == BLANK_PLACEHOLDER and len(inputs[3]) == 8000
    body = requests[0][1]
    assert (
        body["model"] == "nvidia/nemotron-3-embed-1b" and body["input_type"] == "passage" and body["truncate"] == "END"
    )
    assert requests[0][0].headers["authorization"] == "Bearer secret-key"
    assert "secret-key" not in repr(_cfg())


def test_embedder_retries_transient_errors_then_fails_without_the_key():
    attempts = []

    def flaky(request):
        attempts.append(1)
        return (
            httpx.Response(503, text="busy")
            if len(attempts) < 3
            else httpx.Response(200, json={"data": [{"index": 0, "embedding": [0.0] * EMBED_DIMENSIONS}]})
        )

    assert len(NimEmbedder(_cfg(), http=_client(flaky), sleep=lambda s: None).embed(["x"])) == 1 and len(attempts) == 3
    dead = NimEmbedder(_cfg(), http=_client(lambda r: httpx.Response(500)), sleep=lambda s: None, retries=2)
    with pytest.raises(EmbedError) as caught:
        dead.embed(["x"])
    assert "secret-key" not in str(caught.value)
    short = NimEmbedder(_cfg(), http=_client(lambda r: httpx.Response(200, json={"data": []})), sleep=lambda s: None)
    with pytest.raises(EmbedError, match="0 vectors"):
        short.embed(["x"])


def test_load_config_requires_the_weaviate_url_and_a_key(tmp_path):
    with pytest.raises(RuntimeError, match="WEAVIATE_URL"):
        load_config({})
    with pytest.raises(RuntimeError, match="no NIM API key"):
        load_config({"CONTEXT_CHUNKS_WEAVIATE_URL": "http://w:8082"})
    key = tmp_path / "k"
    key.write_text("abc\n", encoding="utf-8")
    cfg = load_config({"CONTEXT_CHUNKS_WEAVIATE_URL": "http://w:8082/", "CONTEXT_CHUNKS_EMBED_API_KEY_FILE": str(key)})
    assert cfg.api_key == "abc" and cfg.weaviate_url == "http://w:8082" and cfg.collection == "ProfferChunks20261002"


# ------------------------------------------------------------------ names
def test_resolver_maps_numbers_names_and_self():
    r = Resolver(
        [
            {"identifier": "8102959303", "entity_id": "e-k", "display_name": "Katrina Kinzel", "kind": "phone"},
            {"identifier": "matt salem", "entity_id": "e-m", "display_name": "Matthew S. Salem", "kind": "name"},
        ]
    )
    assert r.resolve("+18102959303") == ("e-k", "Katrina Kinzel")
    assert r.resolve("Matt Salem") == ("e-m", "Matthew S. Salem")
    assert r.resolve("self", ("e-m", "Matthew S. Salem")) == ("e-m", "Matthew S. Salem")
    assert r.resolve("+15550001111") == (None, "+15550001111")
    assert r.resolve(None) == (None, "Unknown")


# ------------------------------------------------------------------ the temporal wiring
def test_the_worker_registers_the_three_chunk_activities_once_each():
    pytest.importorskip("temporalio")
    from server.temporal import chunk_activities as ca

    assert ca.CHUNK_ACTIVITY_NAMES == (
        "chunk_context_threads_activity",
        "publish_context_chunks_activity",
        "publish_call_log_files_activity",
    )
    import inspect

    from server.temporal import worker

    source = inspect.getsource(worker)
    for fn in ("chunk_context_threads_activity", "publish_context_chunks_activity", "publish_call_log_files_activity"):
        assert source.count(f"                {fn},") == 1  # in the Worker(activities=[...]) list, once
    for fn in (
        ca.chunk_context_threads_activity,
        ca.publish_context_chunks_activity,
        ca.publish_call_log_files_activity,
    ):
        assert fn.__temporal_activity_definition.name in ca.CHUNK_ACTIVITY_NAMES


# ------------------------------------------------------------------ the real model (opt-in)
@pytest.mark.skipif(os.environ.get("CONTEXT_CHUNKS_TEST_NEURAL") != "1", reason="downloads the distilbert model")
def test_neural_distilbert_cuts_a_real_thread_and_keeps_the_overlap():
    lines = [
        render_line(T0 + timedelta(minutes=i * (1 if i < 40 else 600)), "A" if i % 2 else "B", t)
        for i, t in enumerate(
            ["ok see you at six", "sounds good", "did you eat?", "yes thanks"] * 10
            + [
                "Court hearing is on the 14th about the custody order",
                "I got the notice",
                "Can you bring the papers",
                "Yes I will",
            ]
            * 10
        )
    ]
    spans = chunk_spans(lines, "neural_distilbert", 2)
    assert spans[0][0] == 0 and spans[-1][1] == len(lines) - 1
    for (a1, b1), (a2, b2) in pairwise(spans):
        assert len(set(range(a1, b1 + 1)) & set(range(a2, b2 + 1))) >= 2


# ------------------------------------------------------------------ windowing the Neural chunker
class FakeNeural:
    """Behaves like the real Neural chunker on this stack: it only classifies the first ~600 characters it is given
    (about 512 tokens), and splits at every line that says NEWTOPIC."""

    class tokenizer:
        @staticmethod
        def count_tokens_batch(lines):
            return [len(line) // 3 + 1 for line in lines]

    def chunk(self, text):
        starts, position = [], 0
        for line in text.split("\n"):
            if "NEWTOPIC" in line and position < 600:
                starts.append(position)
            position += len(line) + 1
        return [FakeChunk(s) for s in starts] or [FakeChunk(0)]


def _topic_lines(n=600, every=37):
    return [f"[2026-01-05 14:{i % 60:02d}] A: {'NEWTOPIC ' if i and i % every == 0 else ''}msg {i}" for i in range(n)]


def test_unwindowed_neural_misses_every_split_past_the_first_window_and_windowing_finds_them():
    from server.context_chunks.chunker import TextEngine, WindowedNeural

    lines = _topic_lines()
    wanted = [i for i in range(len(lines)) if i and i % 37 == 0]
    whole = TextEngine(FakeNeural()).firsts(lines)
    assert max(whole) < 40 and len(whole) < 4  # the failure that was measured on the real model
    windowed = WindowedNeural(FakeNeural(), budget=150).firsts(lines)
    assert windowed == [0, *wanted]  # every split, exactly once, nothing invented


def test_windows_overlap_by_half_and_respect_the_token_budget():
    from server.context_chunks.chunker import WindowedNeural

    windows = WindowedNeural(FakeNeural(), budget=100).windows([10] * 95)
    assert windows[0] == (0, 10) and windows[1][0] == 5 and windows[-1][1] == 95
    assert all(e - s <= 10 for s, e in windows)
    assert WindowedNeural(FakeNeural(), budget=5).windows([10, 10, 10]) == [(0, 1), (1, 2), (2, 3)]  # one message each
    assert WindowedNeural(FakeNeural(), budget=500).windows([10, 10, 10]) == [(0, 3)]


def test_oversize_chunks_are_cut_evenly_at_message_boundaries():
    from server.context_chunks.chunker import split_oversize

    lines = ["x" * 99] * 40  # 100 characters each with the newline
    assert split_oversize([(0, 39)], lines, 1000) == [(0, 9), (10, 19), (20, 29), (30, 39)]
    assert split_oversize([(0, 39)], lines, 0) == [(0, 39)]  # disabled
    assert split_oversize([(0, 0)], ["y" * 5000], 1000) == [(0, 0)]  # one message is never cut
    assert split_oversize([(0, 3)], lines, 1000) == [(0, 3)]


def test_chunk_spans_cuts_an_oversize_chunk_before_widening_and_keeps_the_overlap():
    lines = ["z" * 199] * 60  # one 12000-character chunk
    spans = chunk_spans(lines, chunker=CutAt([0], lines), overlap=2, max_chars=5000)
    assert len(spans) >= 3 and all(sum(len(lines[i]) + 1 for i in range(a, b + 1)) <= 5000 + 3 * 200 for a, b in spans)
    for (a1, b1), (a2, b2) in pairwise(spans):
        assert len(set(range(a1, b1 + 1)) & set(range(a2, b2 + 1))) >= 2


# ------------------------------------------------------------------ PgSource assembly over rows shaped like the SQL's
class _Result:
    def __init__(self, rows):
        self.rows = rows

    def mappings(self):
        return self

    def scalars(self):
        return _Result([r if not isinstance(r, tuple) else r[0] for r in self.rows])

    def all(self):
        return list(self.rows)

    def one_or_none(self):
        return self.rows[0] if self.rows else None


class _FakeConn:
    """Answers each query of PgSource by a marker in its SQL, with rows shaped like that SQL's columns."""

    def __init__(self, answers):
        self.answers = answers

    def execute(self, statement, params=None):
        sql = str(statement)
        for marker, rows in self.answers.items():
            if marker in sql:
                return _Result(rows)
        raise AssertionError(f"unexpected SQL: {sql[:80]}")


def test_pgsource_names_senders_and_participants_and_builds_call_file_lines():
    from server.context_chunks.source import PgSource

    ident = [
        {"entity_id": "e-k", "display_name": "Katrina Kinzel", "identifier": "8102959303", "kind": "phone"},
        {"entity_id": "e-m", "display_name": "Matthew S. Salem", "identifier": "matt salem", "kind": "name"},
    ]
    t = datetime(2024, 4, 14, 15, 23, tzinfo=UTC)
    conn = _FakeConn(
        {
            "FROM registry.vw_case_identifier": ident,
            "FROM working.third_party_message t\nJOIN context": [
                {
                    "id": "m1",
                    "at": t,
                    "sender_raw": "+18102959303",
                    "sender_entity_id": None,
                    "sender_entity_name": None,
                    "body": "hello\nthere",
                    "svid": "sv-9",
                    "matter_id": "matter-1",
                },
                {
                    "id": "m2",
                    "at": t + timedelta(minutes=1),
                    "sender_raw": "Matt Salem",
                    "sender_entity_id": "e-m",
                    "sender_entity_name": "Matthew S. Salem",
                    "body": None,
                    "svid": "sv-9",
                    "matter_id": "matter-1",
                },
            ],
            "FROM working.third_party_message_participant": [
                {"message_id": "m1", "entity_id": None, "participant_raw": "+18102959303", "participant_e164": None},
                {"message_id": "m1", "entity_id": "e-m", "participant_raw": "self", "participant_e164": None},
            ],
            "FROM working.call_log c": [
                {
                    "id": "c1",
                    "started_at": t,
                    "call_type": "missed",
                    "direction": "inbound",
                    "duration_s": 0,
                    "from_raw": "+18102959303",
                    "to_raw": "self",
                    "from_entity_id": None,
                    "to_entity_id": None,
                    "from_name": None,
                    "to_name": None,
                    "matter_id": "matter-1",
                },
                {
                    "id": "c2",
                    "started_at": t + timedelta(days=1),
                    "call_type": "outgoing",
                    "direction": "outbound",
                    "duration_s": 62,
                    "from_raw": "self",
                    "to_raw": "+18102959303",
                    "from_entity_id": None,
                    "to_entity_id": "e-k",
                    "from_name": None,
                    "to_name": "Katrina Kinzel",
                    "matter_id": "matter-1",
                },
            ],
            "FROM working.first_party_context_thread t": [("owner-id", "Matthew S. Salem")],
        }
    )
    source = PgSource(conn)  # type: ignore[arg-type]
    thread = source.load_thread(ThreadRef("acquired_third_party", "conv-1"))
    assert thread.matter_id == "matter-1" and [m.id for m in thread.messages] == ["m1", "m2"]
    first, second = thread.messages
    # the sender is what the source stated (it is in the embedded text); the registry's name rides along as a property
    assert first.sender == "+18102959303" and first.sender_name == "Katrina Kinzel" and first.sender_entity_id == "e-k"
    assert second.sender == "Matt Salem" and second.sender_name == "Matthew S. Salem" and second.body == ""
    assert set(first.participant_names) == {"Katrina Kinzel", "Device owner"} and "e-m" in first.participant_entity_ids
    assert render_line(first.at, first.sender, first.body) == "[2024-04-14 15:23] +18102959303: hello there"
    calls = source.load_call_file("sv-calls")
    assert calls.call_ids == ["c1", "c2"] and calls.matter_id == "matter-1"
    assert calls.lines == [
        "[2024-04-14 15:23] missed call from Katrina Kinzel, 0s",
        "[2024-04-15 15:23] outgoing call to Katrina Kinzel, 62s",
    ]
    assert (
        calls.first_at == t
        and calls.participant_entity_ids == ["e-k"]
        and calls.participant_names == ["Katrina Kinzel"]
    )


# ------------------------------------------------------------------ the re-chunk entry point
class _Every:
    """A registered chunker that starts a chunk every ``n`` messages."""

    def __init__(self, n):
        self.n = n

    def firsts(self, lines, beat=None):
        return list(range(0, len(lines), self.n))


class _MultiSource:
    def __init__(self, threads: dict[ThreadRef, Thread], calls: dict | None = None):
        self.threads, self.calls = threads, calls or {}

    def all_threads(self):
        return list(self.threads)

    def thread_message_counts(self):
        return {r: len(t.messages) for r, t in self.threads.items()}

    def load_thread(self, ref):
        if ref.thread_id == "boom":
            raise RuntimeError("database went away")
        return self.threads[ref]

    def call_source_versions(self, source_version_id=None):
        return sorted(self.calls)

    def load_call_file(self, source_version_id):
        return self.calls[source_version_id]


def _multi_source():
    from server.context_chunks.chunker import register

    register("test_every5", lambda: _Every(5))
    threads = {}
    for tid, n in (("a", 12), ("b", 40), ("c", 3)):
        thread = make_thread(n, tid, prefix=tid)
        threads[thread.ref] = thread
    third = ThreadRef("acquired_third_party", "t3")
    threads[third] = Thread(ref=third, matter_id="m", messages=make_thread(7, "t3", prefix="z").messages)
    calls = {"sv-a": CallFile("sv-a", "m", ["c1"], ["[2022-09-24 04:02] missed call from K, 0s"], T0, T0)}
    return _MultiSource(threads, calls)


def test_rechunk_dry_run_reports_threads_messages_chunks_embed_calls_and_writes_nothing():
    from server.context_chunks.rechunk import dry_run

    source = _multi_source()
    exact = dry_run(source, "test_every5", 2, sample=3, exact=True, sample_cap=100)  # type: ignore[arg-type]
    assert exact["threads"] == {"first_party": 3, "acquired_third_party": 1, "total": 4}
    assert exact["messages"]["total"] == 62 and exact["messages"]["acquired_third_party"] == 7
    assert exact["chunks"] == 3 + 8 + 1 + 2 and exact["embed_calls"] == 4  # one batch per thread
    assert exact["call_log_files"] == 1 and exact["call_file_embed_calls"] == 1
    estimate = dry_run(source, "test_every5", 2, sample=3, exact=False, sample_cap=100)  # type: ignore[arg-type]
    assert estimate["basis"].startswith("estimate") and estimate["chunks"] >= 4


# ------------------------------------------------------------------ the owner-run removal of the per-message objects
class _ListStore:
    def __init__(self, objects):
        self.objects, self.deleted = objects, []

    def iter_objects(self, properties, *, page=500):
        for o in self.objects:  # like the real store: every requested property, None when absent
            yield {**dict.fromkeys(properties), **o}

    def count(self, where):
        return sum(1 for o in self.objects if o["record_kind"] == where["operands"][1]["valueText"])

    def delete_matching(self, where, *, dry_run=False):
        kind = where["operands"][1]["valueText"]
        self.deleted.append(kind)
        return self.count(where)

    def delete_object(self, object_id):
        self.deleted.append(object_id)


def test_removal_verifies_chunk_coverage_before_it_deletes_anything():
    from server.context_chunks.remove_per_message import verify

    source = _MultiSource({ThreadRef(CORPUS_FIRST_PARTY, "a"): make_thread(4, "a", prefix="a")})
    source.thread_message_ids = lambda ref: [f"a{i:04d}" for i in range(4)]  # type: ignore[method-assign]
    chunks = _ListStore(
        [
            {
                "id": "c1",
                "record_kind": "conversation_chunk",
                "thread_id": "a",
                "message_ids": ["a0000", "a0001", "a0002"],
            },
            {
                "id": "c2",
                "record_kind": "conversation_chunk",
                "normalized_record_ids": [],
                "thread_id": "a",
                "message_ids": ["a0002", "a0003"],
            },
            {"id": "f1", "record_kind": "call_log_file", "normalized_record_ids": [], "call_log_ids": ["call-1"]},
        ]
    )
    old = _ListStore(
        [
            {"id": "o1", "record_kind": "message", "pg_row_id": "a0000", "origin_system": "probata"},
            {"id": "o2", "record_kind": "message", "pg_row_id": "a0003", "origin_system": "probata"},
            {"id": "o3", "record_kind": "call", "pg_row_id": "call-1", "origin_system": "probata"},
        ]
    )
    report = verify(source, chunks, old)  # type: ignore[arg-type]
    assert report["verified"] and report["uncovered_pg_messages"] == 0 and report["old_message_objects"] == 2
    assert report["old_message_objects_uncovered"] == 0 and report["old_call_objects_uncovered"] == 0
    # a message no chunk covers, a Postgres message no chunk covers: not verified, and the ids are named
    old.objects.append({"id": "o4", "record_kind": "message", "pg_row_id": "gone", "origin_system": "probata"})
    chunks.objects[1]["message_ids"] = ["a0002"]
    gap = verify(source, chunks, old)  # type: ignore[arg-type]
    assert not gap["verified"] and gap["uncovered_pg_messages"] == 1 and gap["threads_with_uncovered_messages"] == 1
    assert gap["_uncovered_ids"]["message"] == ["o2", "o4"]


def test_list_threads_can_leave_out_what_chunks_already_cover():
    from server.context_chunks.rechunk import list_threads

    source = _multi_source()
    source.thread_message_ids = lambda ref: [m.id for m in source.threads[ref].messages]  # type: ignore[attr-defined]
    everything, calls = list_threads(source)  # type: ignore[arg-type]
    assert [t["thread_id"] for t in everything] == ["t3", "a", "b", "c"] and calls == ["sv-a"]
    covered = {m.id for m in source.threads[ThreadRef(CORPUS_FIRST_PARTY, "b")].messages}
    rest, _ = list_threads(source, covered=covered)  # type: ignore[arg-type]
    assert [t["thread_id"] for t in rest] == ["t3", "a", "c"]
    assert [t["thread_id"] for t in list_threads(source, limit=2)[0]] == ["t3", "a"]  # type: ignore[arg-type]


def test_removal_refuses_an_unverified_collection_and_counts_in_a_dry_run():
    from server.context_chunks.remove_per_message import NotVerified, remove

    def stores(extra_old=()):
        source = _MultiSource({ThreadRef(CORPUS_FIRST_PARTY, "a"): make_thread(2, "a", prefix="a")})
        source.thread_message_ids = lambda ref: ["a0000", "a0001"]  # type: ignore[method-assign]
        chunks = _ListStore([{"id": "c1", "record_kind": "conversation_chunk", "message_ids": ["a0000", "a0001"]}])
        old = _ListStore(
            [
                {"id": "o1", "record_kind": "message", "pg_row_id": "a0000", "origin_system": "probata"},
                {"id": "o2", "record_kind": "message", "pg_row_id": "a0001", "origin_system": "probata"},
                *extra_old,
            ]
        )
        return source, chunks, old

    source, chunks, old = stores()
    dry = remove(source, chunks, old, dry_run=True)  # type: ignore[arg-type]
    assert (
        dry["verified"] and dry["deleted"] == {"message": 2, "call": 0} and old.deleted == []
    )  # a dry run deletes nothing
    done = remove(source, chunks, old, dry_run=False)  # type: ignore[arg-type]
    assert done["deleted"]["message"] == 2 and old.deleted == ["message", "call"]
    source, chunks, old = stores(
        [{"id": "o3", "record_kind": "message", "pg_row_id": "gone", "origin_system": "probata"}]
    )
    with pytest.raises(NotVerified):
        remove(source, chunks, old, dry_run=False)  # type: ignore[arg-type]
    assert old.deleted == []
    partial = remove(source, chunks, old, dry_run=False, only_covered=True)  # type: ignore[arg-type]
    assert partial["kept_uncovered"]["message"] == 1 and partial["deleted"]["message"] == 2
    assert "_uncovered_ids" not in partial


def test_the_starter_builds_the_workflow_inputs_the_go_structs_decode(monkeypatch):
    from server.context_chunks.start import BACKFILL_WORKFLOW, REMOVAL_WORKFLOW, build_input, parser

    monkeypatch.setenv("PROFFER_MATTER_ID", str(uuid.uuid4()))
    monkeypatch.setenv("PROFFER_COURT_CASE_ID", str(uuid.uuid4()))
    name, workflow_id, body = build_input(
        parser().parse_args(["rechunk", "--dry-run", "--exact", "--request-id", "x1"])
    )
    assert name == BACKFILL_WORKFLOW == "proffer_conversation_chunks_backfill_workflow" and workflow_id.endswith("x1")
    assert body["dry_run"] is True and body["exact"] is True and body["request_id"] == "x1"
    name, workflow_id, body = build_input(parser().parse_args(["remove", "--only-covered"]))
    assert name == REMOVAL_WORKFLOW == "proffer_conversation_chunks_removal_workflow"
    assert body["dry_run"] is False and body["only_covered"] is True and body["request_id"]


def test_the_worker_registers_the_backfill_and_removal_activities_once_each():
    pytest.importorskip("temporalio")
    import inspect

    from server.temporal import chunk_backfill_activities as ba
    from server.temporal import worker

    source = inspect.getsource(worker)
    for fn in (
        ba.list_context_threads_activity,
        ba.estimate_context_chunks_activity,
        ba.verify_chunk_coverage_activity,
        ba.remove_per_message_objects_activity,
    ):
        assert source.count(f"                {fn.__name__},") == 1
        assert fn.__temporal_activity_definition.name in ba.CHUNK_BACKFILL_ACTIVITY_NAMES
