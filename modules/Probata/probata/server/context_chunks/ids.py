"""Deterministic identifiers for chunk objects. No I/O.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Weaviate object id = uuid5(CHUNK_NAMESPACE, "proffer-chunk-v1|<thread_id>|<first message id>|<last message id>|
<chunker version>"), the construction the owner specified. Writing the same chunk again replaces it in place; a
thread that grew is re-chunked whole, its new chunks get new ids, and the old ones are deleted by digest
(store.delete_other_generations).
"""

from __future__ import annotations

import hashlib
import uuid

CHUNK_NAMESPACE = uuid.UUID("b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34")
CHUNK_ID_CONSTRUCTION = (
    "uuid5(ns=b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34, "
    "'proffer-chunk-v1|<thread_id>|<first_message_id>|<last_message_id>|<chunker_version>')"
)
CALL_FILE_ID_CONSTRUCTION = "uuid5(ns=b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34, 'proffer-call-file-v1|<source_version_id>')"


def chunk_object_id(thread_id: str, first_message_id: str, last_message_id: str, chunker_version: str) -> str:
    key = f"proffer-chunk-v1|{thread_id}|{first_message_id}|{last_message_id}|{chunker_version}"
    return str(uuid.uuid5(CHUNK_NAMESPACE, key))


def call_file_object_id(source_version_id: str) -> str:
    return str(uuid.uuid5(CHUNK_NAMESPACE, f"proffer-call-file-v1|{source_version_id}"))


def thread_digest(thread_id: str, message_ids: list[str], chunker_version: str) -> str:
    """sha256 over the thread's ordered message ids and the chunker version: the identity of one chunking."""
    h = hashlib.sha256()
    h.update(f"proffer-thread-digest-v1|{thread_id}|{chunker_version}|{len(message_ids)}\n".encode())
    for message_id in message_ids:
        h.update(message_id.encode())
        h.update(b"\n")
    return h.hexdigest()


CONTENT_CHUNK_ID_CONSTRUCTION = "uuid5(ns=b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34, 'content-chunk-v1|' + chunk_key)"


def chunk_text(lines: list[str]) -> str:
    """The text of a chunk: its member messages' rendered lines (render.render_line), in thread order, one per line,
    including the overlap messages."""
    return chr(10).join(lines)


def chunk_content_hash(lines: list[str]) -> str:
    """sha256 of the chunk text. Holds only what the source said and how it was rendered: no database id, no resolved
    name, no collection name."""
    return hashlib.sha256(chunk_text(lines).encode("utf-8")).hexdigest()


def chunk_key(chunker_version: str, content_hash: str) -> str:
    """THE shared chunk identity: ``<chunker version>|<content hash>``. Two systems that cut the same messages with the same
    chunker version and render them the same way (``[YYYY-MM-DD HH:MM] <source-stated sender>: <body>``, UTC) get the same
    key, so one can copy the other's chunk and its vectors instead of chunking and embedding it again."""
    return f"{chunker_version}|{content_hash}"


def content_chunk_id(key: str) -> str:
    """The Weaviate id of a chunk in the content-addressed scheme (the Super Index's); the same key, the same id."""
    return str(uuid.uuid5(CHUNK_NAMESPACE, "content-chunk-v1|" + key))
