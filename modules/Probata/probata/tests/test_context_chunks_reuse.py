"""The shared chunk key, and Proffer copying a chunk (with its vectors) another system already cut and embedded.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta

from server.context_chunks import (
    chunk_content_hash,
    chunk_key,
    chunk_text,
    content_chunk_id,
    render_line,
)
from server.context_chunks.model import CORPUS_FIRST_PARTY, Message, Thread, ThreadRef
from server.context_chunks.service import plan_thread, publish_thread

T0 = datetime(2026, 1, 5, 14, 30, tzinfo=UTC)
VERSION = "neural_distilbert|mirth/chonky_distilbert_base_uncased_1|chonkie-1.7.0|window=450|maxchars=7500|overlap=2|text=line-v1"


def test_the_shared_chunk_key_vector():
    """The vector the Super Index (CocoIndex) flow and Proffer both assert. Change the render, the hash or the id and
    chunks stop being shareable: this test fails on purpose, and so must the other side's."""
    lines = [
        render_line(T0, "+18102959303", "are you picking him up\nnow"),
        render_line(T0 + timedelta(minutes=1), "device -> +18102959303", "yes at six"),
        render_line(T0 + timedelta(minutes=2), "+18102959303", ""),
    ]
    assert chunk_text(lines) == (
        "[2026-01-05 14:30] +18102959303: are you picking him up now\n"
        "[2026-01-05 14:31] device -> +18102959303: yes at six\n"
        "[2026-01-05 14:32] +18102959303: (attachment)"
    )
    digest = chunk_content_hash(lines)
    assert digest == "473919a1ddc7d5850c8a2b665c1d1daeb8b2b91a1d43a181cecd0bb611287bb0"
    key = chunk_key(VERSION, digest)
    assert key == VERSION + "|" + digest
    assert content_chunk_id(key) == "96961b7f-e21d-5269-8a85-c79dc69d526a"


def _thread(n=10):
    messages = [
        Message(
            id=f"m{i:02d}",
            at=T0 + timedelta(minutes=i),
            sender="+18102959303" if i % 2 else "self",
            sender_name="Katrina Kinzel" if i % 2 else "Matthew S. Salem",
            body=f"message {i}",
            source_version_id="sv-1",
            participant_names=["Katrina Kinzel", "Matthew S. Salem"],
        )
        for i in range(n)
    ]
    return Thread(ThreadRef(CORPUS_FIRST_PARTY, "t-1"), "matter-1", messages)


class _Engine:
    def firsts(self, lines, beat=None):
        return [0, 5]


class _Store:
    def __init__(self):
        self.objects = {}

    def ensure_collection(self):
        return []

    def vector_names(self):
        return ["text_nim"]

    def upsert(self, objects):
        for o in objects:
            self.objects[o["id"]] = o
        return len(objects)

    def has_generation(self, *_):
        return False

    def delete_other_generations(self, *_):
        return 0


class _Embedder:
    model = "nvidia/nemotron-3-embed-1b"

    def __init__(self):
        self.texts = []

    def embed(self, texts):
        self.texts += texts
        return [[1.0, 1.0] for _ in texts]


class _Source:
    def __init__(self, thread):
        self.thread = thread

    def load_thread(self, ref):
        return self.thread


class _Reuse:
    collection = "CaseBibleChunks20261002"

    def __init__(self, vectors_by_hash, exists=True):
        self.vectors_by_hash, self._exists, self.asked = vectors_by_hash, exists, None

    def exists(self):
        return self._exists

    def find_by_content_hashes(self, hashes, chunker_version, vector_names):
        self.asked = (hashes, chunker_version, vector_names)
        return {h: {"id": f"cb-{h[:6]}", "vectors": v} for h, v in self.vectors_by_hash.items() if h in hashes}


def _plan(thread):
    from server.context_chunks.chunker import register

    register("test_two_chunks", lambda: _Engine())
    return plan_thread(thread, "test_two_chunks", 2)


def test_the_embedded_text_carries_the_source_stated_sender_and_the_names_ride_as_properties():
    thread = _thread()
    store = _Store()
    publish_thread(_Source(thread), store, _Embedder(), _plan(thread))
    first = next(o for o in store.objects.values() if o["properties"]["chunk_index"] == 0)["properties"]
    assert first["text"].split("\n")[0] == "[2026-01-05 14:30] self: message 0"  # not "Matthew S. Salem"
    assert "Matthew S. Salem" not in first["text"] and "Katrina Kinzel" not in first["text"]
    assert {"Katrina Kinzel", "Matthew S. Salem"} <= set(first["participant_names"])
    version = first["chunker_version"]
    assert first["content_hash"] == chunk_content_hash(first["text"].split("\n"))
    assert chunk_key(version, first["content_hash"]).endswith(first["content_hash"])


def test_a_matching_chunk_is_copied_with_its_vectors_and_only_the_rest_is_embedded():
    thread = _thread()
    plan = _plan(thread)
    # what the Super Index would hold for the first chunk: the same text, so the same content hash
    from server.context_chunks.service import build_chunk_objects

    objects = build_chunk_objects(thread, plan, embed_model="m")
    shared = objects[0]["properties"]["content_hash"]
    reuse = _Reuse({shared: {"text_nim": [9.0, 9.0]}})
    store, embedder = _Store(), _Embedder()
    result = publish_thread(_Source(thread), store, embedder, plan, reuse=reuse)  # type: ignore[arg-type]
    assert result.reused == 1 and result.chunks_written == 2 and len(embedder.texts) == 1  # one chunk embedded afresh
    assert reuse.asked and reuse.asked[1] == plan.chunker_version and reuse.asked[2] == ["text_nim"]
    copied = next(o for o in store.objects.values() if o["properties"]["chunk_index"] == 0)
    assert copied["vectors"] == {"text_nim": [9.0, 9.0]}  # the other system's vector, not a new one
    assert copied["properties"]["reused_from_object_id"] == f"cb-{shared[:6]}"
    assert copied["properties"]["reused_from_collection"] == "CaseBibleChunks20261002"
    # Proffer's own identity and links are added to the copy
    assert copied["id"] != f"cb-{shared[:6]}" and copied["properties"]["message_ids"][0] == "m00"
    assert copied["properties"]["normalized_record_ids"] == copied["properties"]["message_ids"]
    fresh = next(o for o in store.objects.values() if o["properties"]["chunk_index"] == 1)
    assert fresh["vector"] == [1.0, 1.0] and "reused_from_object_id" not in fresh["properties"]


def test_no_match_or_no_such_collection_means_chunk_and_embed_as_before():
    thread = _thread()
    plan = _plan(thread)
    for reuse in (_Reuse({}), _Reuse({"x": {"text_nim": [1.0]}}, exists=False)):
        embedder = _Embedder()
        result = publish_thread(_Source(thread), _Store(), embedder, plan, reuse=reuse)  # type: ignore[arg-type]
        assert result.reused == 0 and len(embedder.texts) == 2
