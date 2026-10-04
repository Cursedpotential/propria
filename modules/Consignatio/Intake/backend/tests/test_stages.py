"""Embed, publish, relocate, summary and discovery stages. Byline: Claude Code · Sonnet 5.5 ·
2026-10-02"""

import json
from pathlib import Path

import duckdb
import httpx
import pytest
from test_indexer import params, rows, spec

from casebible_index import discovery, ledger
from casebible_index.cas_store import ChunkCollection, StoreError, chunk_properties
from casebible_index.chunk_identity import chunk_object_id
from casebible_index.config import Settings
from casebible_index.embed_stage import run_embed, select_pending
from casebible_index.embedders import SlotEmbedder, VectorSlot
from casebible_index.indexer import Hooks, index_object
from casebible_index.models import DocumentEnrichment
from casebible_index.pipeline import ChunkAccumulator
from casebible_index.publish_stage import publish_pending, relocate_moved
from casebible_index.search import list_documents
from casebible_index.snapshots import build_active_snapshot
from casebible_index.summary_stage import run_summary

SLOT = VectorSlot(
    name="text_nim",
    model="fake-embed",
    dimensions=3,
    base_url="https://nim.test/v1",
    key_secret="K",
    batch_size=4,
)
LEGAL = VectorSlot(
    name="legal",
    model="fake-law",
    dimensions=2,
    base_url="https://law.test/v1",
    key_secret="K",
    style="voyage",
    batch_size=4,
)


def settings_for(out: Path, monkeypatch) -> Settings:
    monkeypatch.setenv("CASEBIBLE_OUTPUT_DIR", str(out))
    monkeypatch.setenv("CASEBIBLE_SOURCE_DIR", str(out / "src"))
    monkeypatch.setenv("INTAKE_SOURCE_MODE", "catalog")
    monkeypatch.setenv("INTAKE_VAULT_BUCKET", "salem-data")
    monkeypatch.setenv("INTAKE_LOCK_DIR", str(out / "locks"))
    return Settings.from_env().resolved()


def embed_client(dimensions: int, seen: list | None = None) -> httpx.AsyncClient:
    def handler(request: httpx.Request) -> httpx.Response:
        texts = json.loads(request.content)["input"]
        if seen is not None:
            seen.extend(texts)
        return httpx.Response(
            200,
            json={
                "data": [
                    {"index": i, "embedding": [1.0 + i] + [0.5] * (dimensions - 1)}
                    for i, _ in enumerate(texts)
                ]
            },
        )

    return httpx.AsyncClient(transport=httpx.MockTransport(handler))


async def build_lake(out: Path, monkeypatch) -> Settings:
    for name, body in (
        ("a/one.txt", "Parenting time order. " * 60),
        ("b/two.txt", "School pickup schedule. " * 60),
    ):
        await index_object(spec(name, body.encode()), params(out), Hooks(), ChunkAccumulator)
    settings = settings_for(out, monkeypatch)
    build_active_snapshot(settings)
    return settings


@pytest.mark.asyncio
async def test_embed_gives_each_distinct_text_one_vector_and_is_idempotent(
    tmp_path: Path, monkeypatch
):
    out = tmp_path / "lake"
    await build_lake(out, monkeypatch)
    distinct = {c["content_hash"] for c in rows(out, "chunks")}
    seen: list[str] = []
    async with embed_client(3, seen) as client:
        result = await run_embed(out, SlotEmbedder(SLOT, client, api_key="k"), max_texts=1000)
    assert result["embedded_ok"] == len(distinct) == len(seen) and result["more"] is False
    files = list((out / "datasets" / "vectors" / "text_nim").glob("*.parquet"))
    assert files
    async with embed_client(3, seen) as client:
        again = await run_embed(out, SlotEmbedder(SLOT, client, api_key="k"), max_texts=1000)
    assert again["selected"] == 0 and again["embedded_ok"] == 0  # nothing is embedded twice


@pytest.mark.asyncio
async def test_embed_is_bounded_and_says_when_more_remain(tmp_path: Path, monkeypatch):
    out = tmp_path / "lake"
    await build_lake(out, monkeypatch)
    async with embed_client(3) as client:
        first = await run_embed(out, SlotEmbedder(SLOT, client, api_key="k"), max_texts=1)
    assert first["selected"] == 1 and first["more"] is True
    assert (
        len(select_pending(out, "text_nim", 1000)[0])
        == len({c["content_hash"] for c in rows(out, "chunks")}) - 1
    )


def weaviate_mock(created: dict, calls: list):
    schema: dict = {}

    def handler(request: httpx.Request) -> httpx.Response:
        path = request.url.path
        if request.method == "GET" and path.startswith("/v1/schema/"):
            return httpx.Response(200, json=schema) if schema else httpx.Response(404)
        if request.method == "POST" and path == "/v1/schema":
            body = json.loads(request.content)
            schema.update(
                {
                    "class": body["class"],
                    "properties": body["properties"],
                    "vectorConfig": body["vectorConfig"],
                }
            )
            calls.append(("schema", sorted(body["vectorConfig"])))
            return httpx.Response(200, json=body)
        if request.method == "POST" and path == "/v1/batch/objects":
            objects = json.loads(request.content)["objects"]
            for o in objects:
                created[o["id"]] = o
            calls.append(("batch", len(objects)))
            return httpx.Response(200, json=[{"id": o["id"], "result": {}} for o in objects])
        if request.method == "PATCH" and path.startswith("/v1/objects/"):
            object_id = path.rsplit("/", 1)[-1]
            if object_id not in created:
                return httpx.Response(404)
            created[object_id]["properties"].update(json.loads(request.content)["properties"])
            calls.append(("patch", object_id))
            return httpx.Response(204)
        if request.method == "POST" and path == "/v1/graphql":
            return httpx.Response(
                200,
                json={
                    "data": {
                        "Aggregate": {
                            "CaseBibleChunks20261002": [{"meta": {"count": len(created)}}]
                        }
                    }
                },
            )
        raise AssertionError(f"unexpected {request.method} {path}")

    return handler


@pytest.mark.asyncio
async def test_publish_waits_for_every_slot_then_writes_content_addressed_objects_once(
    tmp_path: Path, monkeypatch
):
    out = tmp_path / "lake"
    await build_lake(out, monkeypatch)
    created: dict = {}
    calls: list = []
    slots = [SLOT, LEGAL]
    async with httpx.AsyncClient(
        transport=httpx.MockTransport(weaviate_mock(created, calls))
    ) as http:
        store = ChunkCollection("http://weaviate.test", "CaseBibleChunks20261002", http)
        async with embed_client(3) as client:
            await run_embed(out, SlotEmbedder(SLOT, client, api_key="k"), max_texts=1000)
        # Only text_nim is embedded so far: with two slots enabled nothing is ready to publish.
        early = await publish_pending(out, store, slots, run_id="c1")
        assert early["published"] == 0 and created == {}
        async with embed_client(2) as client:
            await run_embed(out, SlotEmbedder(LEGAL, client, api_key="k"), max_texts=1000)
        done = await publish_pending(out, store, slots, run_id="c1")
        distinct = {(c["chunker_version"], c["content_hash"]) for c in rows(out, "chunks")}
        assert done["published"] == len(distinct) == len(created) and done["more"] is False
        sample = next(iter(created.values()))
        assert (
            set(sample["vectors"]) == {"text_nim", "legal"} and len(sample["vectors"]["legal"]) == 2
        )
        p = sample["properties"]
        assert (
            p["origin_system"] == "superindex"
            and p["active"] is True
            and p["content_hash"]
            and p["vault_key"]
        )
        assert sample["id"] == chunk_object_id(p["chunker_version"], p["content_hash"])
        assert p["embed_models"] == ["fake-embed", "fake-law"]
        again = await publish_pending(out, store, slots, run_id="c2")
        assert again["published"] == 0  # the ledger makes a re-run free
    assert calls[0] == ("schema", ["legal", "text_nim"])


@pytest.mark.asyncio
async def test_a_moved_file_is_patched_in_weaviate_without_a_new_embedding(
    tmp_path: Path, monkeypatch
):
    out = tmp_path / "lake"
    settings = await build_lake(out, monkeypatch)
    created: dict = {}
    async with httpx.AsyncClient(transport=httpx.MockTransport(weaviate_mock(created, []))) as http:
        store = ChunkCollection("http://weaviate.test", "CaseBibleChunks20261002", http)
        async with embed_client(3) as client:
            await run_embed(out, SlotEmbedder(SLOT, client, api_key="k"), max_texts=1000)
        await publish_pending(out, store, [SLOT], run_id="c1")
        # The file moves: same bytes, new key. Indexing again writes a locator and nothing else.
        moved = spec("final/place/one.txt", ("Parenting time order. " * 60).encode())
        result = await index_object(moved, params(out), Hooks(), ChunkAccumulator)
        assert result["status"] == "already_indexed"
        relocated = await relocate_moved(out, store, run_id="c2")
        assert relocated["patched"] >= 1
        assert {o["properties"]["vault_key"] for o in created.values()} >= {
            "final/place/one.txt",
            "b/two.txt",
        }
        assert (await relocate_moved(out, store, run_id="c3"))[
            "patched"
        ] == 0  # settled: no ping-pong
    del settings


@pytest.mark.asyncio
async def test_store_refuses_a_collection_missing_a_named_vector(tmp_path: Path):
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200, json={"class": "C", "properties": [], "vectorConfig": {"text_nim": {}}}
        )

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as http:
        store = ChunkCollection("http://w.test", "C", http)
        with pytest.raises(StoreError, match="legal"):
            await store.ensure_collection({"text_nim": 3, "legal": 2})


def test_chunk_properties_omit_none_and_keep_conversation_fields():
    from datetime import UTC, datetime

    props = chunk_properties(
        {
            "text": "t",
            "content_hash": "h",
            "chunk_kind": "conversation",
            "participants": ["a"],
            "first_message_index": 0,
            "last_message_index": 3,
            "start_at": datetime(2026, 1, 1, tzinfo=UTC),
            "end_at": None,
            "vault_key": "k",
        },
        embed_models=["m"],
        run_id="r",
        indexed_at="2026-01-01T00:00:00Z",
    )
    assert props["record_kind"] == "conversation_chunk" and props["participant_names"] == ["a"]
    assert (
        props["start_at"] == "2026-01-01T00:00:00Z"
        and "end_at" not in props
        and props["last_message_index"] == 3
    )


@pytest.mark.asyncio
async def test_summary_is_a_separate_pass_chosen_from_the_index_and_overlays_the_document(
    tmp_path: Path, monkeypatch
):
    out = tmp_path / "lake"
    settings = await build_lake(out, monkeypatch)
    asked: list[str] = []

    async def summarize(**kwargs):
        asked.append(kwargs["filename"])
        return DocumentEnrichment(
            title="Order on parenting time",
            document_type="court_order",
            short_summary="An order.",
            confidence=0.9,
        )

    policy = {"extensions": (".txt",), "min_chars": 100, "max_docs": 10}
    result = await run_summary(out, summarize, "fake-summary", max_chars=2000, policy=policy)
    assert result["summarized"] == 2 and sorted(asked) == ["one.txt", "two.txt"]
    again = await run_summary(out, summarize, "fake-summary", max_chars=2000, policy=policy)
    assert again["selected"] == 0  # already summarized for this version
    titles = {d["title"] for d in list_documents(settings, limit=10)}
    assert titles == {"Order on parenting time"}
    [first, *_] = list_documents(settings, limit=1)
    assert first["document_type"] == "court_order" and first["short_summary"] == "An order."
    # The document row on disk is untouched: the overlay is a relation, not a rewrite.
    assert {d["document_type"] for d in rows(out, "documents")} == {"unknown"}


@pytest.mark.asyncio
async def test_a_failed_summary_leaves_a_marker_and_is_not_retried(tmp_path: Path, monkeypatch):
    out = tmp_path / "lake"
    await build_lake(out, monkeypatch)

    async def broken(**kwargs):
        raise RuntimeError("provider down")

    policy = {"extensions": (".txt",), "min_chars": 100, "max_docs": 10}
    first = await run_summary(out, broken, "fake-summary", max_chars=2000, policy=policy)
    assert first["failed"] == 2 and first["summarized"] == 0
    assert (await run_summary(out, broken, "fake-summary", max_chars=2000, policy=policy))[
        "selected"
    ] == 0


def test_watermark_comparison_and_commit_ledger(tmp_path: Path):
    marks = [discovery.BucketMark("b2", "salem-data", "2026-10-02T23:07:24+00:00", 10, 100)]
    assert discovery.changed_buckets(None, marks) == ["b2:salem-data"]
    committed = [m.as_dict() for m in marks]
    assert discovery.changed_buckets(committed, marks) == []
    grown = [discovery.BucketMark("b2", "salem-data", "2026-10-02T23:07:24+00:00", 11, 100)]
    assert discovery.changed_buckets(committed, grown) == ["b2:salem-data"]
    assert ledger.last_commit(tmp_path) is None
    ledger.write_stage_receipt(tmp_path, "20261003T000000Z", "discover", {"changed": True})
    assert ledger.last_commit(tmp_path) is None  # a cycle that did not commit moves nothing
    ledger.commit_cycle(tmp_path, "20261003T000000Z", committed, {"published": 1})
    assert ledger.last_commit(tmp_path)["watermark"] == committed
    with pytest.raises(ValueError):
        ledger.cycle_dir(tmp_path, "../escape")


def test_discovery_represents_every_object_and_reads_only_the_catalog(tmp_path: Path, monkeypatch):
    """The catalog is faked with an in-memory DuckDB schema of the same name; no bucket is
    touched."""

    def fake_attach(con, dsn):
        con.execute("ATTACH ':memory:' AS catalog")
        con.execute("CREATE SCHEMA catalog.raw_duck")
        con.execute(
            "CREATE TABLE catalog.raw_duck.bucket_objects_current "
            "(provider VARCHAR, bucket VARCHAR, key VARCHAR,"
            " size BIGINT, sha1 VARCHAR, listed_at TIMESTAMPTZ)"
        )
        con.execute(
            "INSERT INTO catalog.raw_duck.bucket_objects_current VALUES "
            "('b2','salem-data','v/note.txt',10,'aa','2026-10-02 23:00:00+00'),"
            "('b2','salem-data','v/photo.jpg',20,'bb','2026-10-02 23:00:00+00'),"
            "('b2','salem-data','v/clip.mp4',30,'cc','2026-10-02 23:00:00+00'),"
            "('b2','salem-data','v/blob.xyz',40,'dd','2026-10-02 23:00:00+00'),"
            "('b2','salem-data','v/takeout.zip',50,'ee','2026-10-02 23:00:00+00'),"
            "('b2','salem-data','v/.git/config',1,'ff','2026-10-02 23:00:00+00'),"
            "('b2','salem-data','v/scan.pdf',60,'hh','2026-10-02 23:00:00+00'),"
            "('r2','casebible-raw','x/other.txt',5,'gg','2026-10-02 23:01:00+00')"
        )

    monkeypatch.setattr(discovery, "_attach", fake_attach)
    settings = settings_for(tmp_path / "lake", monkeypatch)
    monkeypatch.setenv("INTAKE_SOURCE_BUCKETS", "b2:salem-data")
    settings = Settings.from_env().resolved()
    receipt = discovery.discover(settings, "dsn", "20261003T000000Z", None)
    assert receipt["changed"] and receipt["changed_buckets"] == ["b2:salem-data"]
    assert receipt["kinds"] == {"document": 2, "media": 2, "unsupported": 1, "archive": 1}
    assert receipt["excluded"] == 1 and receipt["represented_rows"] == 3
    represented = {
        r[0]: r[1]
        for r in duckdb.connect()
        .execute(f"SELECT key, status FROM read_parquet('{receipt['represented_inventory']}')")
        .fetchall()
    }
    assert represented == {
        "v/photo.jpg": "media_not_extracted",
        "v/clip.mp4": "media_not_extracted",
        "v/blob.xyz": "unsupported",
    }
    # Top-level PDFs are listed once for the image stage (scanned ones are found by joining the
    # extract stage's rows).
    assert receipt["pdf_rows"] == 1
    assert duckdb.connect().execute(
        f"SELECT key, sha1 FROM read_parquet('{receipt['pdf_inventory']}')"
    ).fetchall() == [("v/scan.pdf", "hh")]
    quiet = discovery.discover(settings, "dsn", "20261003T000100Z", receipt["watermark"])
    assert quiet["changed"] is False and "kinds" not in quiet
    forced = discovery.discover(
        settings, "dsn", "20261003T000200Z", receipt["watermark"], force=True
    )
    assert forced["changed"] is True


def test_graph_projection_runs_only_when_the_snapshot_changed_and_the_interval_passed(
    tmp_path: Path,
):
    from datetime import UTC, datetime, timedelta

    from casebible_index.graph_stage import graph_due

    snap_a, snap_b = tmp_path / "s" / "a.parquet", tmp_path / "s" / "b.parquet"
    now = datetime(2026, 10, 3, 12, tzinfo=UTC)
    assert graph_due(tmp_path, None, now=now) == (False, "no active snapshot yet")
    assert graph_due(tmp_path, snap_a, now=now) == (True, "never projected")
    ledger.write_stage_receipt(
        tmp_path, "20261003T000000Z", "graph", {"projected": True, "snapshot": str(snap_a)}
    )
    receipts = list((tmp_path / "cycles").glob("*/*-graph.json"))
    data = json.loads(receipts[0].read_text(encoding="utf-8"))
    data["recorded_at"] = (now - timedelta(hours=2)).isoformat()
    receipts[0].write_text(json.dumps(data), encoding="utf-8")
    assert graph_due(tmp_path, snap_a, now=now)[0] is False  # unchanged snapshot
    assert (
        graph_due(tmp_path, snap_b, now=now, min_hours=24)[0] is False
    )  # changed, but only 2 h ago
    assert graph_due(tmp_path, snap_b, now=now, min_hours=1)[0] is True
    assert graph_due(tmp_path, snap_a, now=now, force=True) == (True, "forced")
