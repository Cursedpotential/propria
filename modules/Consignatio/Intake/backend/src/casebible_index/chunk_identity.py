"""Content-addressed chunk identity. Pure functions, no I/O.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

One definition, shared in words with Proffer so Proffer can COPY a Super Index chunk (and its
vector) when a file
moves into the system instead of re-chunking and re-embedding it (owner 2026-10-02):

    chunk_text   = "\\n".join(render_line(at, sender, body) for every member message, in thread
    order, overlap
                   messages included)            # the shared module's
                   server.context_chunks.render.render_line
    content_hash = sha256(chunk_text as UTF-8).hexdigest()
    chunk_key    = chunker_version + "|" + content_hash
    object id    = uuid5(CHUNK_NAMESPACE, "content-chunk-v1|" + chunk_key)

``chunker_version`` is the exact construction string of the chunker
(server.context_chunks.chunker.chunker_version
for conversations; ``document_chunker_version`` below for plain documents), so a new model, setting
or overlap is a
new identity. The id carries no location: the same text chunked the same way in two files is one
object, one
vector. Where a chunk came from (``vault_key``, ``document_id``, ``sha1``) is a property, and the
authoritative
file-to-chunk map is the lake (Parquet) and the Surreal file graph.

The namespace is the Proffer chunk namespace on purpose: both systems produce ids in one space, and
a Proffer chunk
of identical rendered text and identical chunker version resolves to the same object.
"""

from __future__ import annotations

import hashlib
import uuid

CHUNK_NAMESPACE = uuid.UUID("b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34")
CHUNK_ID_VERSION = "content-chunk-v1"
CHUNK_ID_CONSTRUCTION = (
    "uuid5(ns=b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34, "
    "'content-chunk-v1|<chunker_version>|<sha256(chunk_text)>')"
)
DOCUMENT_TEXT_FORMAT = "raw-v1"


def content_hash(chunk_text: str) -> str:
    return hashlib.sha256(chunk_text.encode("utf-8")).hexdigest()


def chunk_key(chunker_version: str, text_hash: str) -> str:
    return f"{chunker_version}|{text_hash}"


def chunk_object_id(chunker_version: str, text_hash: str) -> str:
    return str(
        uuid.uuid5(CHUNK_NAMESPACE, f"{CHUNK_ID_VERSION}|{chunk_key(chunker_version, text_hash)}")
    )


def document_chunker_version(chunk_size: int, chunk_overlap: int) -> str:
    """The plain-document splitter's construction (CocoIndex RecursiveSplitter, markdown
    language)."""
    return (
        f"recursive_markdown|size={chunk_size}|overlap={chunk_overlap}|text={DOCUMENT_TEXT_FORMAT}"
    )
