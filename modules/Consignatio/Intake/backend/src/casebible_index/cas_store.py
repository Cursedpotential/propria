"""The Weaviate boundary for the Case Bible chunk collection ``CaseBibleChunks20261002``. One unit,
one job: write.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Direct, batched, idempotent writes. This replaces CocoIndex holding one declared target state per
chunk (the
2026-09-22 receipt measured 3.16 GiB for a 61 MB export): the publish stage reads Parquet shards of
bounded size and
posts them in batches, so memory is one shard whatever the object. Ids are content-addressed
(chunk_identity.py), so
every write is an upsert and a re-run costs nothing.

One named vector per embedding slot (embedders.py), vectorizer ``none``, HNSW cosine. The property
names that exist in
the loaders' schema (``chunk_publish.PROPERTIES``) keep their names and types, so one query reads
both writers' rows.
An existing collection is never altered here: a missing property is added (additive), a missing
named vector is an
error that names it.

Dict filters only (a ``FilterExpr`` applies zero filters in production; repository AGENTS.md).
"""

from __future__ import annotations

from collections.abc import Mapping, Sequence
from typing import Any

import httpx

from .chunk_identity import CHUNK_ID_CONSTRUCTION

COLLECTION = "CaseBibleChunks20261002"
ORIGIN_SYSTEM = "superindex"
RECORD_KIND_CONVERSATION = "conversation_chunk"
RECORD_KIND_DOCUMENT = "document_chunk"

DESCRIPTION = (
    "Case Bible chunks (owner 2026-10-02) written by the Coco Super Index: "
    "Chonkie Neural conversation chunks of "
    "message exports and recursive-split chunks of documents, "
    "one object per distinct chunk text and chunker "
    "version, with one named vector per embedding model. Rebuildable from the catalog and B2."
)


def _text(name: str, tokenization: str = "word") -> dict:
    return {"name": name, "dataType": ["text"], "tokenization": tokenization}


def _texts(name: str, tokenization: str = "field") -> dict:
    return {"name": name, "dataType": ["text[]"], "tokenization": tokenization}


# Names and types shared with comm_timeline_mvp/chunk_publish.py come first; the rest are the Super
# Index's own.
PROPERTIES: list[dict] = [
    _text("text"),
    _text("record_kind", "field"),
    _text("corpus", "field"),
    _text("thread_id", "field"),
    _text("thread_digest", "field"),
    _text("conversation_id", "field"),
    _text("conversation_title"),
    _text("vault_key", "field"),
    _text("catalog_path", "field"),
    _text("sha1", "field"),
    _text("source_format", "field"),
    _text("platform", "field"),
    _text("custodian", "field"),
    _text("source_device", "field"),
    _texts("content_keys"),
    _texts("dedup_keys"),
    _text("first_content_key", "field"),
    _text("last_content_key", "field"),
    _text("first_dedup_key", "field"),
    _text("last_dedup_key", "field"),
    _texts("participant_names", "word"),
    {"name": "start_at", "dataType": ["date"]},
    {"name": "end_at", "dataType": ["date"]},
    _text("chunker", "field"),
    _text("chunker_version", "field"),
    {"name": "overlap", "dataType": ["int"]},
    {"name": "chunk_index", "dataType": ["int"]},
    {"name": "message_count", "dataType": ["int"]},
    _text("embed_model", "field"),
    _text("origin_system", "field"),
    _text("ingest_run_id", "field"),
    _text("object_id_construction", "field"),
    {"name": "indexed_at", "dataType": ["date"]},
    # Super Index additions
    _text("content_hash", "field"),
    _text("document_id", "field"),
    _text("version_id", "field"),
    _text("source_id", "field"),
    _text("provider", "field"),
    _text("bucket", "field"),
    _text("member_path", "field"),
    _text("filename"),
    _text("resolution", "field"),
    {"name": "chunk_ordinal", "dataType": ["int"]},
    {"name": "char_start", "dataType": ["int"]},
    {"name": "char_end", "dataType": ["int"]},
    {"name": "first_message_index", "dataType": ["int"]},
    {"name": "last_message_index", "dataType": ["int"]},
    _texts("embed_models", "field"),
    {"name": "active", "dataType": ["boolean"]},
]
PROPERTY_NAMES = frozenset(p["name"] for p in PROPERTIES)


class StoreError(RuntimeError):
    """A Weaviate call failed. Carries no key and no chunk text."""


class ChunkCollection:
    def __init__(
        self, base_url: str, collection: str, client: httpx.AsyncClient, *, batch_size: int = 100
    ) -> None:
        self.base = base_url.rstrip("/")
        self.collection = collection
        self._client = client
        self._batch = batch_size

    async def ensure_collection(self, slots: Mapping[str, int]) -> list[str]:
        """Create the collection with one named vector per slot, or verify an existing one has them
        all.

        Returns the property names created or added. Never drops or alters an existing property."""
        response = await self._client.get(f"{self.base}/v1/schema/{self.collection}")
        if response.status_code == 404:
            body = {
                "class": self.collection,
                "description": DESCRIPTION,
                "properties": PROPERTIES,
                "vectorConfig": {
                    name: {
                        "vectorizer": {"none": {}},
                        "vectorIndexType": "hnsw",
                        "vectorIndexConfig": {"distance": "cosine"},
                    }
                    for name in slots
                },
            }
            created = await self._client.post(f"{self.base}/v1/schema", json=body)
            if created.status_code != 200:
                raise StoreError(
                    f"create {self.collection}: HTTP {created.status_code}: {created.text[:300]}"
                )
            return [p["name"] for p in PROPERTIES]
        if response.status_code != 200:
            raise StoreError(f"inspect {self.collection}: HTTP {response.status_code}")
        schema = response.json()
        have_vectors = set((schema.get("vectorConfig") or {}).keys())
        missing_vectors = sorted(set(slots) - have_vectors)
        if missing_vectors:
            raise StoreError(
                f"collection {self.collection} has no named vector {missing_vectors}; "
                "add the vector to the collection first (Weaviate schema), then enable the slot"
            )
        have = {p["name"]: p for p in schema.get("properties", [])}
        added: list[str] = []
        for want in PROPERTIES:
            got = have.get(want["name"])
            if got is not None:
                if got["dataType"] != want["dataType"]:
                    raise StoreError(
                        f"property {want['name']} is {got['dataType']}, want {want['dataType']}"
                    )
                continue
            posted = await self._client.post(
                f"{self.base}/v1/schema/{self.collection}/properties", json=want
            )
            if posted.status_code != 200:
                raise StoreError(
                    f"add property {want['name']}: HTTP {posted.status_code}: {posted.text[:200]}"
                )
            added.append(want["name"])
        return added

    async def upsert(self, objects: Sequence[Mapping[str, Any]]) -> int:
        """Write ``{"id", "properties", "vectors": {slot: [...]}}``. Fail-closed on any per-object
        rejection."""
        written = 0
        for start in range(0, len(objects), self._batch):
            part = objects[start : start + self._batch]
            body = {
                "objects": [
                    {
                        "class": self.collection,
                        "id": o["id"],
                        "properties": o["properties"],
                        "vectors": o["vectors"],
                    }
                    for o in part
                ]
            }
            response = await self._client.post(f"{self.base}/v1/batch/objects", json=body)
            if response.status_code != 200:
                raise StoreError(f"batch write: HTTP {response.status_code}: {response.text[:300]}")
            results = response.json()
            if len(results) != len(part):
                raise StoreError(f"batch response covered {len(results)} of {len(part)} objects")
            for item in results:
                errors = ((item.get("result") or {}).get("errors") or {}).get("error") or []
                if errors:
                    raise StoreError(
                        f"Weaviate rejected object {item.get('id')}: {errors[0].get('message')}"
                    )
                written += 1
        return written

    async def patch(self, object_id: str, properties: Mapping[str, Any]) -> bool:
        """Merge properties into one object (a moved file's new locator). False when the object does
        not exist."""
        response = await self._client.patch(
            f"{self.base}/v1/objects/{self.collection}/{object_id}",
            json={"class": self.collection, "id": object_id, "properties": dict(properties)},
        )
        if response.status_code == 404:
            return False
        if response.status_code not in (200, 204):
            raise StoreError(f"patch {object_id}: HTTP {response.status_code}")
        return True

    async def count(self) -> int:
        response = await self._client.post(
            f"{self.base}/v1/graphql",
            json={"query": f"{{ Aggregate {{ {self.collection} {{ meta {{ count }} }} }} }}"},
        )
        payload = response.json() if response.status_code == 200 else {}
        if response.status_code != 200 or payload.get("errors"):
            raise StoreError(f"count: HTTP {response.status_code}")
        return int(payload["data"]["Aggregate"][self.collection][0]["meta"]["count"])


def chunk_properties(
    row: Mapping[str, Any], *, embed_models: Sequence[str], run_id: str, indexed_at: str
) -> dict:
    """Weaviate properties for one chunk row of the lake (the publish stage's mapping; None values
    are omitted)."""
    conversation = (row.get("chunk_kind") or "document") == "conversation"
    props: dict[str, Any] = {
        "text": row["text"],
        "record_kind": RECORD_KIND_CONVERSATION if conversation else RECORD_KIND_DOCUMENT,
        "corpus": "casebible",
        "thread_id": row.get("thread_id") or "",
        "vault_key": row.get("vault_key") or "",
        "catalog_path": row.get("relative_path") or "",
        "sha1": row.get("sha1") or "",
        "source_format": row.get("extension") or "",
        "chunker": row.get("chunker") or "",
        "chunker_version": row.get("chunker_version") or "",
        "overlap": int(row.get("overlap") or 0),
        "chunk_index": int(row.get("chunk_ordinal") or 0),
        "message_count": int(row.get("message_count") or 0),
        "embed_model": embed_models[0] if embed_models else "",
        "embed_models": list(embed_models),
        "origin_system": ORIGIN_SYSTEM,
        "ingest_run_id": run_id,
        "object_id_construction": CHUNK_ID_CONSTRUCTION,
        "indexed_at": indexed_at,
        "content_hash": row["content_hash"],
        "document_id": row.get("document_id") or "",
        "version_id": row.get("version_id") or "",
        "source_id": row.get("source_id") or "",
        "provider": row.get("provider") or "",
        "bucket": row.get("bucket") or "",
        "member_path": row.get("member_path") or "",
        "filename": row.get("filename") or "",
        "resolution": row.get("resolution") or "",
        "chunk_ordinal": int(row.get("chunk_ordinal") or 0),
        "char_start": int(row.get("char_start") or 0),
        "char_end": int(row.get("char_end") or 0),
        "active": True,
    }
    participants = row.get("participants")
    if participants:
        props["participant_names"] = list(participants)
    for source, target in (
        ("first_message_index", "first_message_index"),
        ("last_message_index", "last_message_index"),
    ):
        if row.get(source) is not None:
            props[target] = int(row[source])
    for source, target in (("start_at", "start_at"), ("end_at", "end_at")):
        value = row.get(source)
        if value is not None:
            props[target] = (
                value.strftime("%Y-%m-%dT%H:%M:%SZ") if hasattr(value, "strftime") else str(value)
            )
    return props
