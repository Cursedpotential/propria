"""Content-addressed chunk identity. Byline: Claude Code · Sonnet 5.5 · 2026-10-02"""

import hashlib
import uuid

from casebible_index import chunk_identity as ci

TEXT = "[2026-01-02 03:04] +18105550101: hello\n[2026-01-02 03:05] device -> +18105550101: hi"
VERSION = (
    "neural_distilbert|mirth/chonky_distilbert_base_uncased_1|chonkie-1.7.0|"
    "window=450|maxchars=7500|overlap=2|text=line-v1"
)


def test_definition_is_sha256_of_text_and_uuid5_of_version_and_hash():
    digest = hashlib.sha256(TEXT.encode("utf-8")).hexdigest()
    assert ci.content_hash(TEXT) == digest
    assert ci.chunk_key(VERSION, digest) == f"{VERSION}|{digest}"
    expected = str(
        uuid.uuid5(
            uuid.UUID("b6f5c3a2-6d1e-4f6a-8f0e-2a9c5d7e1b34"),
            f"content-chunk-v1|{VERSION}|{digest}",
        )
    )
    assert ci.chunk_object_id(VERSION, digest) == expected


def test_test_vector_for_the_chunk_agent():
    # Frozen so Proffer's copy path can assert it computes the same id for the same rendered text.
    assert (
        ci.content_hash(TEXT) == "ad70b309b1c71dd78f7417a1a739663672e919e843f3264d60eeafa5895c523b"
    )
    assert (
        ci.chunk_object_id(VERSION, ci.content_hash(TEXT)) == "d877f60e-322f-5339-9d4b-097e4e05d67c"
    )


def test_identity_carries_no_location_and_changes_with_version_or_text():
    h = ci.content_hash(TEXT)
    assert ci.chunk_object_id(VERSION, h) == ci.chunk_object_id(VERSION, ci.content_hash(TEXT))
    assert ci.chunk_object_id(VERSION + "x", h) != ci.chunk_object_id(VERSION, h)
    assert ci.chunk_object_id(VERSION, ci.content_hash(TEXT + " ")) != ci.chunk_object_id(
        VERSION, h
    )


def test_document_chunker_version_pins_size_and_overlap():
    assert ci.document_chunker_version(2400, 300) != ci.document_chunker_version(2400, 200)
