# Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""Chunked publish for the Case Bible loaders: conversations become Chonkie Neural chunks in CaseBibleChunks20261002.

Owner 2026-10-02: the Case Bible does not need one Weaviate entry per message either; it needs a real basic search.
New loads are chunked from now on (the 366,912 per-message objects already in MsgEvents20260918 stay as they are):

  * rows are grouped by (vault_key, conversation_id) and ordered by sort_ts_final;
  * each group goes through the same chunker as Proffer (Chonkie Neural distilbert, windowed, every chunk overlapping the
    next by at least 2 messages) -- the code is Probata's server/context_chunks, imported, not copied;
  * each chunk is one object: its text (one line per message), and the links back to the Case Bible's records
    (vault_key, catalog_path, conversation_id, and the member content_key / dedup_key values);
  * ids are uuid5 over (vault_key#conversation_id, first dedup_key, last dedup_key, chunker version), so a rerun upserts;
    a conversation that was extracted again is re-chunked whole and its old chunks replaced; chunks of the same file
    left by an earlier run id are retired after the new ones are in (the bundle and the catalog keep the record).

Needs the Probata chunk core on PYTHONPATH:  PYTHONPATH=<propria checkout>/modules/Probata/probata  (the directory that
holds server/context_chunks; on the devbox, copy server/__init__.py and server/context_chunks/ next to this script). It
needs chonkie[neural] with torch and the distilbert model in the venv (the jev-eval venv has them) for a real run.

Env: WEAVIATE_URL, NVIDIA_API_KEY (as the other loaders), CHUNK_COLLECTION (default CaseBibleChunks20261002), RUN_ID.
"""
from __future__ import annotations

import math
import os
import sys
import time
import uuid
from collections import defaultdict
from datetime import datetime, timezone

try:
    from server.context_chunks import ids
    from server.context_chunks.chunker import chunk_spans, chunker_version
    from server.context_chunks.config import DEFAULT_CHUNKER, DEFAULT_OVERLAP, EMBED_BATCH, ChunkConfig
    from server.context_chunks.embed import NimEmbedder
    from server.context_chunks.model import Message, Thread, ThreadRef
    from server.context_chunks.render import one_line, render_line
    from server.context_chunks.store import ChunkStore
except ImportError as error:  # a clear message beats a bare traceback on the devbox
    sys.exit(f"chunk_publish needs Probata's server/context_chunks on PYTHONPATH ({error}); see this file's docstring")

COLLECTION = os.environ.get("CHUNK_COLLECTION", "CaseBibleChunks20261002")
ORIGIN = "casebible"
RECORD_KIND = "conversation_chunk"
# Messages per chunk measured on the Proffer threads (2026-10-02), used only for a probe's estimate.
EST_MESSAGES_PER_CHUNK = 32.0


def _text(name: str, tokenization: str = "word") -> dict:
    return {"name": name, "dataType": ["text"], "tokenization": tokenization}


def _texts(name: str, tokenization: str = "field") -> dict:
    return {"name": name, "dataType": ["text[]"], "tokenization": tokenization}


PROPERTIES: list[dict] = [
    _text("text"), _text("record_kind", "field"), _text("corpus", "field"),
    _text("thread_id", "field"), _text("thread_digest", "field"),
    _text("conversation_id", "field"), _text("conversation_title"),
    _text("vault_key", "field"), _text("catalog_path", "field"), _text("sha1", "field"),
    _text("source_format", "field"), _text("platform", "field"), _text("custodian", "field"),
    _text("source_device", "field"),
    _texts("content_keys"), _texts("dedup_keys"), _text("first_content_key", "field"), _text("last_content_key", "field"),
    _text("first_dedup_key", "field"), _text("last_dedup_key", "field"),
    _texts("participant_names", "word"),
    {"name": "start_at", "dataType": ["date"]}, {"name": "end_at", "dataType": ["date"]},
    _text("chunker", "field"), _text("chunker_version", "field"),
    {"name": "overlap", "dataType": ["int"]}, {"name": "chunk_index", "dataType": ["int"]},
    {"name": "message_count", "dataType": ["int"]},
    _text("embed_model", "field"), _text("origin_system", "field"), _text("ingest_run_id", "field"),
    _text("object_id_construction", "field"), {"name": "indexed_at", "dataType": ["date"]},
]
DESCRIPTION = ("Case Bible conversation chunks (owner 2026-10-02): Chonkie Neural chunks of the messages the comm_timeline_mvp "
               "loaders extract, each linking back to its vault_key, catalog_path, conversation_id and member content/dedup "
               "keys. New loads only; MsgEvents20260918 keeps the earlier per-message objects. Rebuildable from B2.")
ID_CONSTRUCTION = ("uuid5(ns=b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34, "
                   "'casebible-chunk-v1|<vault_key>#<conversation_id>|<first_dedup_key>|<last_dedup_key>|<chunker_version>')")


def make_store(client, weaviate_url: str, collection: str = COLLECTION) -> ChunkStore:
    return ChunkStore(weaviate_url, collection, http=client, properties=PROPERTIES, description=DESCRIPTION)


def make_embedder(client, api_key: str, model: str, base_url: str, batch: int = EMBED_BATCH) -> NimEmbedder:
    return NimEmbedder(ChunkConfig(weaviate_url="-", api_key=api_key, embed_model=model, embed_base_url=base_url,
                                   embed_batch=batch), http=client)


def _when(row: dict) -> datetime | None:
    for key in ("sort_ts_final", "event_ts_utc"):
        value = row.get(key)
        if value is not None:
            return value if value.tzinfo else value.replace(tzinfo=timezone.utc)
    return None


def _line_body(row: dict) -> str:
    body = one_line(row.get("body"))
    if body:
        return body
    # a call, or a message of attachments only: the same description the per-message loader embedded
    return one_line(" ".join(x for x in [row.get("conversation_title"), row.get("event_kind"), row.get("direction"),
                                          ("attachments: " + row["attachments"]) if row.get("attachments") else None] if x))


def group_rows(rows: list[dict]) -> dict[tuple[str, str], list[dict]]:
    """(vault_key, conversation_id) -> its rows in conversation order."""
    groups: dict[tuple[str, str], list[dict]] = defaultdict(list)
    for r in rows:
        groups[(r["vault_key"], r.get("conversation_id") or "")].append(r)
    far = datetime.max.replace(tzinfo=timezone.utc)
    for g in groups.values():
        g.sort(key=lambda r: (_when(r) or far, r.get("record_index") or 0, r["dedup_key"]))
    return groups


def group_thread(vault_key: str, conversation_id: str, rows: list[dict]) -> Thread:
    """The core's Thread for one conversation. Message id = the row's dedup_key (unique per copy)."""
    thread_id = f"{vault_key}#{conversation_id}"
    messages = [Message(id=r["dedup_key"], at=_when(r), sender=r.get("sender") or r.get("contact_name") or "",
                        body=_line_body(r),
                        participant_names=[p for p in (r.get("participants") or []) if p]) for r in rows]
    return Thread(ref=ThreadRef(ORIGIN, thread_id), matter_id=None, messages=messages)


def _iso(at: datetime | None) -> str | None:
    return None if at is None else at.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def chunk_group(vault_key: str, conversation_id: str, rows: list[dict], *, run_id: str, model: str,
                chunker: str = DEFAULT_CHUNKER, overlap: int = DEFAULT_OVERLAP, engine=None,
                corpus: str = ORIGIN) -> tuple[str, list[dict]]:
    """(thread digest, chunk objects without vectors) for one conversation."""
    thread = group_thread(vault_key, conversation_id, rows)
    version = chunker_version(chunker, overlap)
    lines = [render_line(m.at, m.sender, m.body) for m in thread.messages]
    spans = chunk_spans(lines, chunker, overlap, chunker=engine)
    digest = ids.thread_digest(thread.ref.thread_id, [m.id for m in thread.messages], version)
    head = rows[0]
    now = _iso(datetime.now(timezone.utc))
    objects = []
    for index, (first, last) in enumerate(spans):
        members = rows[first:last + 1]
        names = [n for m in thread.messages[first:last + 1] for n in [m.sender, *m.participant_names] if n]
        times = [t for t in (_when(r) for r in members) if t is not None]
        props = {
            "text": "\n".join(lines[first:last + 1]), "record_kind": RECORD_KIND, "corpus": corpus,
            "thread_id": thread.ref.thread_id, "thread_digest": digest, "conversation_id": conversation_id,
            "conversation_title": head.get("conversation_title") or "", "vault_key": vault_key,
            "catalog_path": head.get("catalog_rel") or "", "sha1": head.get("sha1") or "",
            "source_format": head.get("source_format") or "", "platform": head.get("platform") or "",
            "custodian": head.get("custodian") or "", "source_device": head.get("source_device") or "",
            "content_keys": list(dict.fromkeys(r["content_key"] for r in members if r.get("content_key"))),
            "dedup_keys": [r["dedup_key"] for r in members],
            "first_content_key": members[0].get("content_key") or "", "last_content_key": members[-1].get("content_key") or "",
            "first_dedup_key": members[0]["dedup_key"], "last_dedup_key": members[-1]["dedup_key"],
            "participant_names": list(dict.fromkeys(names)), "chunker": chunker, "chunker_version": version,
            "overlap": overlap, "chunk_index": index, "message_count": len(members), "embed_model": model,
            "origin_system": ORIGIN, "ingest_run_id": run_id, "object_id_construction": ID_CONSTRUCTION, "indexed_at": now,
        }
        if times:
            props["start_at"], props["end_at"] = _iso(min(times)), _iso(max(times))
        key = f"casebible-chunk-v1|{thread.ref.thread_id}|{members[0]['dedup_key']}|{members[-1]['dedup_key']}|{version}"
        objects.append({"id": str(uuid.uuid5(ids.CHUNK_NAMESPACE, key)), "properties": props})
    return digest, objects


def retire_old_file(store: ChunkStore, vault_key: str, run_id: str, *, dry: bool = False) -> int:
    """Chunks of this file left by any earlier run id (derived index objects; the bundle and the catalog keep the record)."""
    where = {"operator": "And", "operands": [
        {"path": ["vault_key"], "operator": "Equal", "valueText": vault_key},
        {"path": ["ingest_run_id"], "operator": "NotEqual", "valueText": run_id}]}
    return store.delete_matching(where, dry_run=dry)


def publish_file(store: ChunkStore, embedder: NimEmbedder, rows: list[dict], *, run_id: str, pace: float = 0.0,
                 chunker: str = DEFAULT_CHUNKER, overlap: int = DEFAULT_OVERLAP, engine=None, batch: int = EMBED_BATCH,
                 corpus: str = ORIGIN) -> dict:
    """Chunk, embed and write every conversation of one file (all rows share a vault_key); returns the counts."""
    if not rows:
        return {"conversations": 0, "chunks": 0, "skipped_current": 0, "retired": 0}
    vault_key = rows[0]["vault_key"]
    store.ensure_collection()
    out = {"conversations": 0, "chunks": 0, "skipped_current": 0, "retired": 0}
    for (vk, conv), group in group_rows(rows).items():
        digest, objects = chunk_group(vk, conv, group, run_id=run_id, model=embedder.model, chunker=chunker,
                                      overlap=overlap, engine=engine, corpus=corpus)
        out["conversations"] += 1
        thread_id = objects[0]["properties"]["thread_id"]
        current = {"operator": "And", "operands": [
            {"path": ["thread_id"], "operator": "Equal", "valueText": thread_id},
            {"path": ["thread_digest"], "operator": "Equal", "valueText": digest},
            {"path": ["ingest_run_id"], "operator": "Equal", "valueText": run_id}]}
        if store.count(current) >= len(objects):
            out["skipped_current"] += 1  # this run id already wrote exactly this chunking of the conversation (a resumed run)
            continue
        for start in range(0, len(objects), batch):
            part = objects[start:start + batch]
            vectors = embedder.embed([o["properties"]["text"] for o in part])
            for o, v in zip(part, vectors, strict=True):
                o["vector"] = v
            if store.upsert(part) != len(part):
                raise RuntimeError(f"store wrote fewer than {len(part)} chunks for {thread_id}")
            out["chunks"] += len(part)
            if pace:
                time.sleep(pace)
        store.delete_other_generations(thread_id, digest)
    out["retired"] = retire_old_file(store, vault_key, run_id)
    return out


def estimate(conversations: int, messages: int, *, batch: int = EMBED_BATCH) -> dict:
    """A probe's estimate (no model run): chunks at EST_MESSAGES_PER_CHUNK, one embed call per ``batch`` chunks per
    conversation, at least one call per conversation."""
    chunks = max(conversations, math.ceil(messages / EST_MESSAGES_PER_CHUNK))
    calls = max(conversations, math.ceil(chunks / batch))
    return {"conversations": conversations, "messages": messages, "estimated_chunks": chunks, "estimated_embed_calls": calls}
