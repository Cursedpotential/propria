"""The extract-and-chunk stage end to end on bytes. Byline: Claude Code · Sonnet 5.5 · 2026-10-02"""

import hashlib
import io
import zipfile
from pathlib import Path

import duckdb
import pytest
from test_conversation_pipeline import SMS_XML, EveryTwo

from casebible_index.indexer import Hooks, IndexParams, ObjectSpec, index_object, local_zip_handle
from casebible_index.pipeline import ChunkAccumulator


def params(out: Path, **kw) -> IndexParams:
    base = dict(
        source_id="t",
        output_dir=out,
        chunk_size=400,
        chunk_overlap=50,
        chunk_flush_size=512,
        summary_max_chars=2000,
        embed_batch_size=8,
        embed_model="m",
        summary_model="s",
        embed_dimensions=4,
        embed_enabled=False,
        summary_enabled=False,
        spool_dir=out / "spool",
    )
    base.update(kw)
    return IndexParams(**base)


def spec(
    key: str, data: bytes, *, vault_key: str | None = None, identity: str | None = None
) -> ObjectSpec:
    async def windows():
        for start in range(0, len(data), 1024):
            yield data[start : start + 1024]

    async def head(n: int) -> bytes:
        return data[:n]

    return ObjectSpec(
        key=key,
        relative_path=vault_key or key,
        vault_key=vault_key or key,
        member_path="",
        identity=identity or "sha1:" + hashlib.sha1(data).hexdigest(),
        byte_size=len(data),
        resolution="content_on_b2",
        windows=windows,
        head=head,
        provider="b2",
        bucket="salem-data",
    )


def rows(out: Path, dataset: str) -> list[dict]:
    files = list((out / "datasets" / dataset).glob("*.parquet"))
    if not files:
        return []
    con = duckdb.connect()
    try:
        cursor = con.execute(
            f"SELECT * FROM read_parquet({[f.as_posix() for f in files]!r}, union_by_name = true)"
        )
        names = [d[0] for d in cursor.description]
        return [dict(zip(names, r, strict=True)) for r in cursor.fetchall()]
    finally:
        con.close()


@pytest.mark.asyncio
async def test_document_chunks_are_content_addressed_and_pending(tmp_path: Path):
    text = ("The court entered an order on parenting time. " * 40).encode()
    result = await index_object(
        spec("a/order.txt", text), params(tmp_path), Hooks(), ChunkAccumulator
    )
    assert result["status"] == "indexed" and result["chunks"] >= 2
    chunks = rows(tmp_path, "chunks")
    assert {c["embedding_status"] for c in chunks} == {"pending"}
    assert all(
        c["content_hash"] and c["chunker_version"].startswith("recursive_markdown|size=400")
        for c in chunks
    )
    assert all(
        c["chunk_kind"] == "document" and c["provider"] == "b2" and c["bucket"] == "salem-data"
        for c in chunks
    )
    [doc] = rows(tmp_path, "documents")
    assert doc["index_status"] == "indexed" and doc["chunk_count"] == len(chunks)


@pytest.mark.asyncio
async def test_a_moved_file_is_not_streamed_again_and_leaves_a_locator(tmp_path: Path):
    data = ("A sentence about a school pickup. " * 30).encode()
    first = await index_object(
        spec("old/place.txt", data), params(tmp_path), Hooks(), ChunkAccumulator
    )
    assert first["status"] == "indexed"

    reads = {"n": 0}
    moved = spec("new/place.txt", data)
    original = moved.windows

    def counting():
        reads["n"] += 1
        return original()

    moved.windows = counting
    second = await index_object(moved, params(tmp_path), Hooks(), ChunkAccumulator)
    assert second["status"] == "already_indexed" and reads["n"] == 0
    [locator] = rows(tmp_path, "locators")
    assert locator["vault_key"] == "new/place.txt"
    assert len(rows(tmp_path, "documents")) == 1  # nothing re-derived


@pytest.mark.asyncio
async def test_ai_chat_export_is_represented_but_never_chunked(tmp_path: Path):
    export = b'[{"title": "t", "mapping": {"a": {"message": null}}, "current_node": "a"}]'
    result = await index_object(
        spec("moved/data-2025/conversations.json", export),
        params(tmp_path),
        Hooks(),
        ChunkAccumulator,
    )
    assert result["status"] == "routed_ai_chat" and result["chunks"] == 0
    assert rows(tmp_path, "chunks") == []
    [doc] = rows(tmp_path, "documents")
    assert (
        doc["index_status"] == "routed_ai_chat"
        and doc["vault_key"] == "moved/data-2025/conversations.json"
    )


@pytest.mark.asyncio
async def test_ai_chat_mode_index_restores_the_old_behaviour(tmp_path: Path):
    export = b'[{"title": "t", "mapping": {"a": {}}, "current_node": "a"}]'
    result = await index_object(
        spec("x/conversations.json", export),
        params(tmp_path, ai_chat_mode="index"),
        Hooks(),
        ChunkAccumulator,
    )
    assert result["status"] == "indexed"


@pytest.mark.asyncio
async def test_sms_backup_becomes_conversation_chunks(tmp_path: Path):
    pytest.importorskip("server.context_chunks.chunker")
    result = await index_object(
        spec("xml/sms-backup.xml", SMS_XML),
        params(tmp_path),
        Hooks(engine=EveryTwo(), calls_engine=EveryTwo()),
        ChunkAccumulator,
    )
    assert result["conversation"] and result["chunks"] >= 3
    chunks = rows(tmp_path, "chunks")
    assert {c["chunk_kind"] for c in chunks} == {"conversation"}
    assert all(
        c["content_hash"] and c["thread_id"].startswith("sha1:") and c["message_count"] >= 1
        for c in chunks
    )
    assert not any("Resolved Name" in c["text"] for c in chunks)  # source-stated senders only
    assert any("device -> +18105550101" in c["text"] for c in chunks)
    spool_dir = tmp_path / "spool"
    assert [path.name for path in spool_dir.iterdir()] == ["to_be_deleted"]
    assert len(list((spool_dir / "to_be_deleted" / "message-spool").glob("*/*.sqlite"))) == 1


@pytest.mark.asyncio
async def test_route_mode_keeps_a_document_row_for_a_message_export(tmp_path: Path):
    result = await index_object(
        spec("xml/sms-backup.xml", SMS_XML),
        params(tmp_path, conversation_mode="route"),
        Hooks(),
        ChunkAccumulator,
    )
    assert result["status"] == "routed_message_export" and rows(tmp_path, "chunks") == []


@pytest.mark.asyncio
async def test_zip_members_nested_zip_and_media_are_all_represented(tmp_path: Path):
    inner = io.BytesIO()
    with zipfile.ZipFile(inner, "w") as z:
        z.writestr("deep/letter.txt", "Dear court, " * 100)
    outer_path = tmp_path / "outer.zip"
    with zipfile.ZipFile(outer_path, "w") as z:
        z.writestr("notes/summary.txt", "Meeting notes about the schedule. " * 40)
        z.writestr("photos/img1.jpg", b"\xff\xd8\xff not really a jpeg")
        z.writestr("data.bin", b"\x00\x01\x02")
        z.writestr("inner.zip", inner.getvalue())
    data = outer_path.read_bytes()

    async def open_archive(_: ObjectSpec):
        return local_zip_handle(outer_path)

    out = tmp_path / "lake"
    result = await index_object(
        spec("takeout/outer.zip", data),
        params(out, index_archive_members=True),
        Hooks(open_archive=open_archive),
        ChunkAccumulator,
    )
    assert (
        result["status"] == "container"
        and result["members_indexed"] == 2
        and result["members_inventoried"] >= 3
    )
    documents = {d["member_path"]: d for d in rows(out, "documents")}
    assert set(documents) == {"", "notes/summary.txt", "inner.zip!/deep/letter.txt"}
    assert all(d["vault_key"] == "takeout/outer.zip" for d in documents.values())
    assert documents["notes/summary.txt"]["index_status"] == "indexed"
    inventory = {r["member_path"]: r["status"] for r in rows(out, "members")}
    assert inventory["photos/img1.jpg"] == "media_not_extracted"
    assert inventory["data.bin"] == "unsupported"
    assert inventory["inner.zip"] == "nested_archive"
    assert not list((out / "spool").glob("*"))


@pytest.mark.asyncio
async def test_a_corrupt_archive_keeps_its_container_row(tmp_path: Path):
    bad = tmp_path / "bad.zip"
    bad.write_bytes(b"not a zip")

    async def open_archive(_: ObjectSpec):
        return local_zip_handle(bad)

    out = tmp_path / "lake"
    result = await index_object(
        spec("x/bad.zip", b"not a zip"),
        params(out, index_archive_members=True),
        Hooks(open_archive=open_archive),
        ChunkAccumulator,
    )
    assert result["status"] == "container" and result["members_indexed"] == 0
    [doc] = rows(out, "documents")
    assert "Archive listing failed" in " ".join(doc["extraction_notes"])
