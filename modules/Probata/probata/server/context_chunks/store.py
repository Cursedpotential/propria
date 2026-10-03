"""The Weaviate REST boundary for the chunk collection (ProfferChunks20261002).

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Same shape as ProfferMsgEvents20261002 (read live 2026-10-02): one named vector ``text_nim``, vectorizer none (the
writer supplies the NIM vector), HNSW, cosine. REST over httpx, three endpoints like the Go store (engine/weaviate):
schema, batch objects, graphql. The batch endpoint REPLACES an existing id (verified 2026-09-26: the count does not
grow), so every write here is an idempotent upsert.

``dict`` filters only, never a FilterExpr (the repository AGENTS.md: a FilterExpr applies zero filters in production).
"""

from __future__ import annotations

import json
from collections.abc import Iterator
from typing import Any

import httpx

from server.context_chunks.config import VECTOR_NAME

DESCRIPTION = (
    "Proffer conversation chunks (owner 2026-10-02): Chonkie Neural distilbert chunks of committed Postgres "
    "messages, each linking back to its Postgres message ids, plus one entry per call-log file. Postgres holds every "
    "message and call; this collection holds only chunks. Rebuildable from Postgres."
)

RECORD_KIND_CHUNK = "conversation_chunk"
RECORD_KIND_CALL_FILE = "call_log_file"


def _text(name: str, tokenization: str = "word") -> dict:
    return {"name": name, "dataType": ["text"], "tokenization": tokenization}


def _texts(name: str, tokenization: str = "field") -> dict:
    return {"name": name, "dataType": ["text[]"], "tokenization": tokenization}


PROPERTIES: list[dict] = [
    _text("text"),  # the chunk text, one line per message; for a call file one line per call
    _text("record_kind", "field"),  # conversation_chunk | call_log_file
    _text("corpus", "field"),  # first_party | acquired_third_party | call_log
    _text("thread_id", "field"),
    _text("thread_digest", "field"),
    _text(
        "content_hash", "field"
    ),  # ids.chunk_content_hash: with chunker_version, what lets another system reuse this chunk
    _text("reused_from_object_id", "field"),  # the object whose vectors were copied, when they were
    _text("reused_from_collection", "field"),  # one chunking of one thread; stale generations are deleted by it
    _text("first_message_id", "field"),
    _text("last_message_id", "field"),
    _texts("message_ids"),
    # The same ids under their pre-commit name: a committed working.message id IS the normalized record id (the
    # first-party import copies it), so a chunk cut from a run's normalized generation needs no rewrite after the commit.
    _texts("normalized_record_ids"),
    _text("normalized_generation_id", "field"),  # Postgres working.message / third_party_message ids the chunk covers
    _texts("call_log_ids"),  # Postgres working.call_log ids, for a call-log file
    _texts("participant_entity_ids"),
    _texts("participant_names", "word"),
    {"name": "start_at", "dataType": ["date"]},
    {"name": "end_at", "dataType": ["date"]},
    _texts("source_version_ids"),  # first element = the source version of the chunk's first message
    _text("source_version_id", "field"),  # a call-log file's own source version
    _text("matter_id", "field"),
    _text("chunker", "field"),
    _text("chunker_version", "field"),
    {"name": "overlap", "dataType": ["int"]},
    {"name": "chunk_index", "dataType": ["int"]},
    {"name": "message_count", "dataType": ["int"]},
    _text("embed_model", "field"),
    _text("origin_system", "field"),
    _text("ingest_run_id", "field"),  # the Proffer run that published it: joins a chunk to that run's preview decision
    _text("object_id_construction", "field"),
    {"name": "indexed_at", "dataType": ["date"]},
]


class StoreError(RuntimeError):
    pass


class ChunkStore:
    def __init__(
        self,
        base_url: str,
        collection: str,
        *,
        vector_name: str = VECTOR_NAME,
        http: httpx.Client | None = None,
        batch_size: int = 100,
        properties: list[dict] | None = None,
        description: str = DESCRIPTION,
    ):
        """``properties`` and ``description`` default to the Proffer chunk collection's; the Case Bible chunk
        collection passes its own (casebible/tools/comm_timeline_mvp/chunk_publish.py)."""
        self.properties = PROPERTIES if properties is None else properties
        self.description = description
        self.base = base_url.rstrip("/")
        self.collection = collection
        self.vector_name = vector_name
        self._http = http or httpx.Client(timeout=120.0)
        self._batch = batch_size

    # ---------------------------------------------------------------- schema
    def ensure_collection(self) -> list[str]:
        """Create the collection when absent (owner-approved name); otherwise add any property it lacks. Returns the
        property names created or added. Never drops or alters an existing property."""
        r = self._http.get(f"{self.base}/v1/schema/{self.collection}")
        if r.status_code == 404:
            body = {
                "class": self.collection,
                "description": self.description,
                "properties": self.properties,
                "vectorConfig": {
                    self.vector_name: {
                        "vectorizer": {"none": {}},
                        "vectorIndexType": "hnsw",
                        "vectorIndexConfig": {"distance": "cosine"},
                    }
                },
            }
            c = self._http.post(f"{self.base}/v1/schema", json=body)
            if c.status_code != 200:
                raise StoreError(f"create collection {self.collection}: HTTP {c.status_code}: {c.text[:300]}")
            return [p["name"] for p in self.properties]
        if r.status_code != 200:
            raise StoreError(f"inspect collection {self.collection}: HTTP {r.status_code}")
        schema = r.json()
        if self.vector_name not in (schema.get("vectorConfig") or {}):
            raise StoreError(f"collection {self.collection} has no named vector {self.vector_name!r}")
        have = {p["name"]: p for p in schema.get("properties", [])}
        added = []
        for want in self.properties:
            got = have.get(want["name"])
            if got is not None:
                if got["dataType"] != want["dataType"]:
                    raise StoreError(f"property {want['name']} is {got['dataType']}, want {want['dataType']}")
                continue
            a = self._http.post(f"{self.base}/v1/schema/{self.collection}/properties", json=want)
            if a.status_code != 200:
                raise StoreError(f"add property {want['name']}: HTTP {a.status_code}: {a.text[:300]}")
            added.append(want["name"])
        return added

    # ---------------------------------------------------------------- writes
    def upsert(self, objects: list[dict]) -> int:
        """Write objects ``{"id", "properties", "vector"}``; returns the number written. Fail-closed on any rejection."""
        written = 0
        for start in range(0, len(objects), self._batch):
            part = objects[start : start + self._batch]
            body = {
                "objects": [
                    {
                        "class": self.collection,
                        "id": o["id"],
                        "properties": o["properties"],
                        "vectors": o.get("vectors") or {self.vector_name: o["vector"]},
                    }
                    for o in part
                ]
            }
            r = self._http.post(f"{self.base}/v1/batch/objects", json=body)
            if r.status_code != 200:
                raise StoreError(f"batch write: HTTP {r.status_code}: {r.text[:300]}")
            results = r.json()
            if len(results) != len(part):
                raise StoreError(f"batch response covered {len(results)} of {len(part)} objects")
            for item in results:
                errors = ((item.get("result") or {}).get("errors") or {}).get("error") or []
                if errors:
                    raise StoreError(f"Weaviate rejected object {item.get('id')}: {errors[0].get('message')}")
                written += 1
        return written

    def _where_thread_not_digest(self, thread_id: str, digest: str) -> dict:
        return {
            "operator": "And",
            "operands": [
                {"path": ["thread_id"], "operator": "Equal", "valueText": thread_id},
                {"path": ["thread_digest"], "operator": "NotEqual", "valueText": digest},
            ],
        }

    def delete_matching(self, where: dict, *, dry_run: bool = False) -> int:
        """Batch-delete every object matching ``where`` (loops past the server's per-call limit). Returns the count
        deleted, or, with ``dry_run``, the count that WOULD match (one call; capped at the server limit per call, so
        use ``count`` for an exact figure)."""
        total = 0
        while True:
            body = {"match": {"class": self.collection, "where": where}, "output": "minimal", "dryRun": dry_run}
            r = self._http.request("DELETE", f"{self.base}/v1/batch/objects", json=body)
            if r.status_code != 200:
                raise StoreError(f"batch delete: HTTP {r.status_code}: {r.text[:300]}")
            res = r.json().get("results") or {}
            if res.get("failed"):
                raise StoreError(f"batch delete failed for {res['failed']} objects")
            n = int(res.get("successful", 0) if not dry_run else res.get("matches", 0))
            total += n
            if dry_run or n == 0:
                return total

    def delete_object(self, object_id: str) -> None:
        r = self._http.delete(f"{self.base}/v1/objects/{self.collection}/{object_id}")
        if r.status_code not in (204, 404):
            raise StoreError(f"delete object {object_id}: HTTP {r.status_code}")

    def delete_other_generations(self, thread_id: str, digest: str) -> int:
        """Replace a thread's chunks: delete every chunk of the thread that is not of the current chunking."""
        return self.delete_matching(self._where_thread_not_digest(thread_id, digest))

    # ---------------------------------------------------------------- reads
    def exists(self) -> bool:
        return self._http.get(f"{self.base}/v1/schema/{self.collection}").status_code == 200

    def vector_names(self) -> list[str]:
        """The named vectors this collection has (the ones a reused object must bring)."""
        r = self._http.get(f"{self.base}/v1/schema/{self.collection}")
        if r.status_code != 200:
            raise StoreError(f"inspect collection {self.collection}: HTTP {r.status_code}")
        return sorted((r.json().get("vectorConfig") or {}).keys())

    def find_by_content_hashes(
        self, hashes: list[str], chunker_version: str, vector_names: list[str], *, page: int = 40
    ) -> dict[str, dict]:
        """content_hash -> {"id", "vectors": {name: vector}} for the objects cut with ``chunker_version`` that hold one of
        ``hashes`` (first match per hash).

        Only objects that carry EVERY name in ``vector_names`` are returned: a chunk missing one of the vectors the
        destination collection has is not reusable as it stands."""
        found: dict[str, dict] = {}
        wanted = " ".join(vector_names)
        for start in range(0, len(hashes), page):
            part = hashes[start : start + page]
            where = _gql(
                {
                    "operator": "And",
                    "operands": [
                        {"path": ["content_hash"], "operator": "ContainsAny", "valueText": part},
                        {"path": ["chunker_version"], "operator": "Equal", "valueText": chunker_version},
                    ],
                }
            )
            data = self._graphql(
                f"{{ Get {{ {self.collection}(where: {where}, limit: {len(part) * 2}) "
                f"{{ content_hash _additional {{ id vectors {{ {wanted} }} }} }} }} }}"
            )
            for o in data["Get"][self.collection] or []:
                vectors = (o.get("_additional") or {}).get("vectors") or {}
                if (
                    o.get("content_hash")
                    and o["content_hash"] not in found
                    and all(vectors.get(n) for n in vector_names)
                ):
                    found[o["content_hash"]] = {"id": o["_additional"]["id"], "vectors": vectors}
        return found

    def _graphql(self, query: str) -> dict:
        r = self._http.post(f"{self.base}/v1/graphql", json={"query": query})
        payload = r.json() if r.status_code == 200 else {}
        if r.status_code != 200 or payload.get("errors"):
            raise StoreError(f"graphql: HTTP {r.status_code}: {str(payload.get('errors'))[:300]}")
        return payload["data"]

    def count(self, where: dict | None = None) -> int:
        clause = f"(where: {_gql(where)})" if where else ""
        data = self._graphql(f"{{ Aggregate {{ {self.collection}{clause} {{ meta {{ count }} }} }} }}")
        return int(data["Aggregate"][self.collection][0]["meta"]["count"])

    def has_generation(self, thread_id: str, digest: str) -> bool:
        """True when the thread already has chunks of exactly this chunking (resume after a crash)."""
        where = {
            "operator": "And",
            "operands": [
                {"path": ["thread_id"], "operator": "Equal", "valueText": thread_id},
                {"path": ["thread_digest"], "operator": "Equal", "valueText": digest},
            ],
        }
        return self.count(where) > 0

    def iter_objects(self, properties: list[str], *, page: int = 500) -> Iterator[dict]:
        """Every object's ``properties`` and ``id`` by id cursor (REST ``after``)."""
        after = ""
        while True:
            url = f"{self.base}/v1/objects?class={self.collection}&limit={page}"
            if after:
                url += f"&after={after}"
            r = self._http.get(url)
            if r.status_code != 200:
                raise StoreError(f"list objects: HTTP {r.status_code}")
            objects = r.json().get("objects", [])
            if not objects:
                return
            for o in objects:
                yield {"id": o["id"], **{k: (o.get("properties") or {}).get(k) for k in properties}}
            after = objects[-1]["id"]


def _gql(value: Any) -> str:
    """A dict filter as GraphQL input syntax (unquoted keys, enum-style operators)."""
    if isinstance(value, dict):
        return "{" + ", ".join(f"{k}: {_gql_operator(k, v)}" for k, v in value.items()) + "}"
    if isinstance(value, list):
        return "[" + ", ".join(_gql(v) for v in value) + "]"
    return json.dumps(value)


def _gql_operator(key: str, value: Any) -> str:
    return str(value) if key == "operator" else _gql(value)
