"""The image stage: kind, selection, fetch (images and scanned PDF pages), facts, OCR, embed,
publish, schema.

Byline: Claude Code · Sonnet 5.5 · 2026-10-03

Every image here is synthetic and generated in the test. The bucket is a fake store, the embedders
are fake
functions, Weaviate is a recording writer or an httpx MockTransport: nothing leaves the process.
"""

import hashlib
import io
import shutil
from datetime import UTC, datetime
from pathlib import Path

import httpx
import pyarrow as pa
import pyarrow.parquet as pq
import pytest
from PIL import Image, ImageDraw

from casebible_index import image_bench, image_stage
from casebible_index.discovery import REPRESENTED_SCHEMA
from casebible_index.image_kind import classify_image
from casebible_index.image_slice import (
    fetch_slice,
    item_path,
    load_slice,
    open_slice_id,
    select_images,
    select_scanned_pdfs,
    table_path,
)
from casebible_index.image_stage import (
    ImageStageSettings,
    run_embed,
    run_facts,
    run_ocr,
    run_publish,
)
from casebible_index.image_target import ImageObjectWriter, collection_schema

NOW = datetime(2026, 10, 3, tzinfo=UTC)


def settings(monkeypatch, **env) -> ImageStageSettings:
    for name in [n for n in list(__import__("os").environ) if n.startswith("INTAKE_IMAGE")]:
        monkeypatch.delenv(name)
    base = {
        "INTAKE_IMAGE_STAGE": "on",
        "INTAKE_IMAGES_WEAVIATE_URL": "http://weaviate.test",
        "INTAKE_IMAGES_CLIP": "0",
    }
    for name, value in {**base, **env}.items():
        monkeypatch.setenv(name, value)
    s = ImageStageSettings.from_env()
    s.validate()
    return s


def png(text: str, size=(900, 500)) -> bytes:
    image = Image.new("RGB", size, "white")
    ImageDraw.Draw(image).text((40, 200), text, fill="black", font=image_bench._font(44))
    out = io.BytesIO()
    image.save(out, format="PNG")
    return out.getvalue()


def jpeg(color) -> bytes:
    out = io.BytesIO()
    Image.new("RGB", (640, 480), color).save(out, format="JPEG")
    return out.getvalue()


class FakeStore:
    def __init__(self, objects: dict[str, bytes]):
        self.objects = objects
        self.reads: list[str] = []

    async def stream(self, key: str, *, window_bytes: int = 0):
        self.reads.append(key)
        if key not in self.objects:
            raise RuntimeError("missing")
        data = self.objects[key]
        for start in range(0, len(data), 7000):
            yield data[start : start + 7000]


def inventory(out: Path, rows: list[tuple], folder="represented") -> None:
    table = pa.Table.from_pylist(
        [
            {
                "provider": "b2",
                "bucket": "salem-data",
                "key": k,
                "size": s,
                "sha1": h,
                "kind": kind,
                "status": "media_not_extracted",
                "listed_at": NOW,
                "cycle_id": "20261003T000000Z",
            }
            for k, s, h, kind in rows
        ],
        schema=REPRESENTED_SCHEMA,
    )
    path = out / "inventory" / folder / "20261003T000000Z.parquet"
    path.parent.mkdir(parents=True, exist_ok=True)
    pq.write_table(table, path)


# -------------------------------------------------- kind


def test_kind_rules_run_in_order_and_name_their_basis():
    assert (
        classify_image("a.pdf", 1700, 2200, is_pdf_page=True, format_name="PNG").basis == "pdf_page"
    )
    assert (
        classify_image("Screenshot_2026-01-02.jpg", 4000, 3000, has_camera_exif=True).kind
        == "screenshot"
    )
    assert classify_image("IMG_1.jpg", 4000, 3000, has_camera_exif=True).basis == "camera_exif"
    assert classify_image("x.png", 800, 600, format_name="PNG").basis == "png_no_camera"
    shot = classify_image("x.jpg", 1080, 2400, format_name="JPEG")
    assert (shot.kind, shot.basis) == ("screenshot", "screen_geometry")
    assert classify_image("x.jpg", 4032, 3024, format_name="JPEG").kind == "photo"
    assert classify_image("x.jpg", 100, 100, user_comment="screenshot").basis == "name_hint"


# -------------------------------------------------- selection and fetch


@pytest.mark.asyncio
async def test_fetch_groups_copies_streams_once_skips_ledgered_and_resumes_an_open_slice(
    tmp_path: Path,
):
    out = tmp_path / "lake"
    a, b = png("alpha"), jpeg((200, 10, 10))
    inventory(
        out,
        [
            ("v/one.png", len(a), "aa", "media"),
            ("w/copy-of-one.png", len(a), "aa", "media"),  # one image, two occurrences
            ("v/two.jpg", len(b), "bb", "media"),
            ("v/huge.jpg", 99_000_000, "cc", "media"),
            ("v/phone.heic", 50, "dd", "media"),
            ("v/song.mp3", 50, "ee", "media"),
            ("v/note.txt", 5, "ff", "document"),
        ],
    )
    store = FakeStore({"v/one.png": a, "w/copy-of-one.png": a, "v/two.jpg": b})
    first = await fetch_slice(out, tmp_path / "spool", lambda p, bk: store, max_files=10)
    assert first["count"] == 2 and first["more"] is False and first["slice_id"]
    assert first["skipped"] == {"unsupported_format": 1, "over_max_file_bytes": 1}
    sl = load_slice(out, first["slice_id"])
    assert {i.identity: i.occurrences for i in sl.items} == {"aa": 2, "bb": 1}
    assert len(store.reads) == 2, "an exact copy must not be read twice"
    again = await fetch_slice(out, tmp_path / "spool", lambda p, bk: store, max_files=10)
    assert again["resumed"] and again["slice_id"] == first["slice_id"] and len(store.reads) == 2
    assert open_slice_id(out) == first["slice_id"]
    # Once ledgered, the images are not selected again.
    table_path(out, "published", first["slice_id"]).parent.mkdir(parents=True)
    pq.write_table(
        pa.Table.from_pylist(
            [
                {
                    "identity": i,
                    "status": "ok",
                    "reason": "",
                    "object_id": "x",
                    "slice_id": "s",
                    "run_id": "r",
                    "published_at": NOW,
                }
                for i in ("aa", "bb")
            ],
            schema=image_stage.LEDGER_SCHEMA,
        ),
        table_path(out, "published", first["slice_id"]),
    )
    picked, more, _ = select_images(out, 10, 25_000_000)
    assert picked == [] and more is False
    assert (await fetch_slice(out, tmp_path / "spool", lambda p, bk: store))["slice_id"] == ""


@pytest.mark.asyncio
async def test_slice_respects_file_and_byte_bounds_and_records_a_failed_object(tmp_path: Path):
    out = tmp_path / "lake"
    data = {f"v/{n}.png": png(f"item {n}") for n in range(5)}
    inventory(
        out,
        [(k, len(v), f"h{n}", "media") for n, (k, v) in enumerate(data.items())]
        + [("v/gone.png", 100, "a-gone", "media")],
    )
    store = FakeStore(data)
    result = await fetch_slice(out, None, lambda p, bk: store, max_files=3)
    assert (
        result["count"] == 2 and result["failed"] == 1 and result["more"] is True
    )  # a-gone failed, h0,h1 fetched (sorted)
    sl = load_slice(out, result["slice_id"])
    assert [f["identity"] for f in sl.failed] == ["a-gone"]
    inventory(
        tmp_path / "bytes", [(k, len(v), f"h{n}", "media") for n, (k, v) in enumerate(data.items())]
    )
    reads_before = len(store.reads)
    by_bytes = await fetch_slice(
        tmp_path / "bytes", None, lambda p, bk: store, max_files=10, max_bytes=1
    )
    assert by_bytes["count"] == 0 and by_bytes["failed"] == len(data)
    assert len(store.reads) == reads_before, "objects beyond the byte budget are never fetched"


@pytest.mark.asyncio
async def test_scanned_pdf_becomes_page_images_and_is_selected_only_when_it_had_no_text(
    tmp_path: Path,
):
    out = tmp_path / "lake"
    pages = [Image.new("RGB", (612, 792), "white") for _ in range(3)]
    for page, label in zip(pages, ("first page", "second page", "third page"), strict=True):
        ImageDraw.Draw(page).text((60, 60), label, fill="black", font=image_bench._font(40))
    buffer = io.BytesIO()
    pages[0].save(buffer, format="PDF", save_all=True, append_images=pages[1:])
    scanned = buffer.getvalue()
    inventory(
        out,
        [
            ("docs/scan.pdf", len(scanned), "s1", "document"),
            ("docs/text.pdf", len(scanned), "s2", "document"),
        ],
        "pdf",
    )
    documents = out / "datasets" / "documents"
    documents.mkdir(parents=True)
    pq.write_table(
        pa.Table.from_pylist(
            [
                {
                    "content_sha256": "sha1:s1",
                    "index_status": "skipped_no_text",
                    "extension": ".pdf",
                    "member_path": "",
                },
                {
                    "content_sha256": "sha1:s2",
                    "index_status": "indexed",
                    "extension": ".pdf",
                    "member_path": "",
                },
            ]
        ),
        documents / "d.parquet",
    )
    picked, _ = select_scanned_pdfs(out, 10)
    assert [p["identity"] for p in picked] == ["s1"]
    result = await fetch_slice(
        out,
        tmp_path / "spool",
        lambda p, bk: FakeStore({"docs/scan.pdf": scanned}),
        pdf_max_pages=2,
    )
    sl = load_slice(out, result["slice_id"])
    assert result["kind"] == "pdf" and result["count"] == 2
    assert [i.identity for i in sl.items] == ["s1#p1", "s1#p2"] and all(
        i.page_count == 3 for i in sl.items
    )
    assert sl.skipped == {"s1": 1}, "a truncated document says how many pages were not rendered"


@pytest.mark.asyncio
@pytest.mark.parametrize("max_items", [0, 1])
async def test_pdf_page_budget_resumes_remainder_and_only_then_marks_source_complete(
    tmp_path, monkeypatch, max_items
):
    """A capped PDF remains selectable until every page is published.

    Inputs: a synthetic three-page PDF and one-page budget. Output: ledger assertions.
    Side effects: retained local fixture artifacts only. Pick this to verify page-window
    continuation.
    """
    out = tmp_path / "lake"
    spool = tmp_path / "spool"
    pages = [Image.new("RGB", (72, 72), color) for color in ("white", "red", "blue")]
    buffer = io.BytesIO()
    pages[0].save(buffer, format="PDF", save_all=True, append_images=pages[1:])
    data = buffer.getvalue()
    source_sha1 = hashlib.sha1(data).hexdigest()
    inventory(out, [("proof/scan.pdf", len(data), source_sha1, "document")], "pdf")
    docs = out / "datasets" / "documents"
    docs.mkdir(parents=True)
    pq.write_table(
        pa.Table.from_pylist(
            [
                {
                    "content_sha256": "sha1:" + source_sha1,
                    "index_status": "skipped_no_text",
                    "extension": ".pdf",
                    "member_path": "",
                }
            ]
        ),
        docs / "scan.parquet",
    )
    image_buffer = io.BytesIO()
    Image.new("RGB", (72, 72), "green").save(image_buffer, format="PNG")
    image_data = image_buffer.getvalue()
    image_sha1 = hashlib.sha1(image_data).hexdigest()
    inventory(out, [("proof/photo.png", len(image_data), image_sha1, "media")])
    store = FakeStore({"proof/scan.pdf": data, "proof/photo.png": image_data})
    s = settings(monkeypatch, INTAKE_IMAGES_SLOTS="image_maxsim")
    writer = RecordingWriter()

    async def bag(data, ext):
        return [[0.5] * 128]

    first = await fetch_slice(out, spool, lambda *_: store, max_files=1)
    assert first["kind"] == "image" and first["count"] == 1 and first["more"] is True
    image_slice = load_slice(out, first["slice_id"])
    await run_facts(out, spool, image_slice)
    await run_embed(out, spool, image_slice, "image_maxsim", bag, "synthetic", s)
    await run_publish(out, spool, image_slice, writer, s, source_id="proof", run_id="image")

    page_hashes = []
    for page_number in (1, 2, 3):
        result = await fetch_slice(
            out, spool, lambda *_: store, max_items=max_items, max_files=1, pdf_max_pages=20
        )
        assert result["count"] == 1
        sl = load_slice(out, result["slice_id"])
        assert sl.items[0].page == page_number
        expected_hash = hashlib.sha256(
            item_path(out, spool, sl.slice_id, sl.items[0]).read_bytes()
        ).hexdigest()
        await run_facts(out, spool, sl)
        await run_embed(out, spool, sl, "image_maxsim", bag, "synthetic", s)
        published = await run_publish(
            out, spool, sl, writer, s, source_id="proof", run_id=str(page_number)
        )
        assert published["published"] == 1 and published["partial"] == 0
        properties = writer.actions[-1].spec.properties
        assert properties["content_sha256"] == expected_hash
        assert properties["source_content_sha1"] == source_sha1
        assert len(properties["content_sha256"]) == 64 and len(source_sha1) == 40
        page_hashes.append(properties["content_sha256"])
        parent = [
            r
            for r in pq.read_table(table_path(out, "published", sl.slice_id)).to_pylist()
            if r["identity"] == source_sha1
        ][0]
        assert parent["status"] == ("ok" if page_number == 3 else "partial")
    assert select_scanned_pdfs(out, 10)[0] == []
    assert len({a.key for a in writer.actions}) == 4
    assert len(set(page_hashes)) == 3, "Rendered pages have distinct hashes, one source PDF SHA1"


# -------------------------------------------------- facts, ocr


@pytest.mark.asyncio
async def test_image_digest_is_sha256_and_legacy_facts_are_rebuilt(tmp_path: Path):
    """Keep the retained image digest distinct from its catalog SHA1 on a facts retry.

    Inputs: synthetic PNG and a legacy facts shard. Outputs: hash/property assertions.
    Side effects: retained local fixtures only. Pick this to catch mislabeled source hashes
    and stale cached facts after the additive digest field is introduced.
    """
    data = png("Synthetic hash proof")
    source_sha1 = hashlib.sha1(data).hexdigest()
    expected_sha256 = hashlib.sha256(data).hexdigest()
    out, spool = tmp_path / "lake", tmp_path / "spool"
    inventory(out, [("proof/image.png", len(data), source_sha1, "media")])
    store = FakeStore({"proof/image.png": data})
    fetched = await fetch_slice(out, spool, lambda *_: store)
    sl = load_slice(out, fetched["slice_id"])
    await run_facts(out, spool, sl)
    path = table_path(out, "facts", sl.slice_id)
    legacy = pq.read_table(path).drop(["content_sha256"])
    pq.write_table(legacy, path)
    await run_facts(out, spool, sl)
    facts = pq.read_table(path).to_pylist()[0]
    properties = image_stage.build_properties(
        sl.items[0], facts, None, source_id="proof", models={}
    )
    assert facts["content_sha256"] == properties["content_sha256"] == expected_sha256
    assert properties["source_content_sha1"] == source_sha1 == sl.items[0].identity
    assert len(expected_sha256) == 64 and len(source_sha1) == 40
    assert expected_sha256 != source_sha1


@pytest.mark.asyncio
async def test_facts_and_ocr_read_the_slice(tmp_path: Path):
    out = tmp_path / "lake"
    shot = png("Pick up Mia at 4:30 on Friday")
    inventory(
        out,
        [
            ("v/Screenshot_2026-01-02.png", len(shot), "p1", "media"),
            ("v/photo.jpg", 9000, "p2", "media"),
        ],
    )
    store = FakeStore({"v/Screenshot_2026-01-02.png": shot, "v/photo.jpg": jpeg((30, 90, 200))})
    fetched = await fetch_slice(out, tmp_path / "spool", lambda p, bk: store)
    sl = load_slice(out, fetched["slice_id"])
    facts = await run_facts(out, tmp_path / "spool", sl)
    assert facts["kinds"] == {"screenshot": 1, "photo": 1}
    assert (await run_facts(out, tmp_path / "spool", sl))["images"] == 2, (
        "a retry reads its own output"
    )
    if shutil.which("tesseract"):
        ocr = await run_ocr(out, tmp_path / "spool", sl, engine="tesseract")
        assert ocr["with_text"] >= 1 and ocr["failed"] == 0
        rows = {
            r["identity"]: r for r in pq.read_table(table_path(out, "ocr", sl.slice_id)).to_pylist()
        }
        assert "friday" in rows["p1"]["ocr_text"].lower()
    from casebible_index.image_ocr import OcrUnavailable, read_text

    with pytest.raises(OcrUnavailable):
        read_text("no-such-engine", Path("x.png"), 10)
    with pytest.raises(OcrUnavailable):
        read_text(
            "doctr", Path("x.png"), 10
        )  # python-doctr is not installed here: a clear configuration error


# -------------------------------------------------- settings, embed, publish


def test_settings_validation_and_slot_coverage(monkeypatch):
    s = settings(monkeypatch)
    assert s.slots == ("image_maxsim", "image_single") and s.maxsim == "all"
    assert s.slot_applies("image_maxsim", "photo") and s.slot_applies("image_single", "scan")
    only_shots = settings(monkeypatch, INTAKE_IMAGES_MAXSIM="screenshots")
    assert only_shots.slot_applies("image_maxsim", "screenshot") and not only_shots.slot_applies(
        "image_maxsim", "photo"
    )
    tier = settings(monkeypatch, INTAKE_IMAGES_CLIP="1", INTAKE_IMAGES_PHOTO_TIER="clip_first")
    assert not tier.slot_applies("image_single", "photo") and tier.slot_applies(
        "image_single", "screenshot"
    )
    assert "Qwen Research License" in tier.licence_basis()["image_maxsim"]
    for bad in (
        {"INTAKE_IMAGES_PHOTO_TIER": "clip_first"},
        {"INTAKE_IMAGES_SLOTS": "image_colqwen"},
        {"INTAKE_IMAGES_SLOTS": "nope"},
        {"INTAKE_IMAGES_MAXSIM": "some"},
    ):
        with pytest.raises(ValueError):
            settings(monkeypatch, **bad)


class RecordingWriter:
    url, collection = "http://weaviate.test", "IntakeImageV1"

    def __init__(self):
        self.actions = []

    async def apply(self, action):
        self.actions.append(action)


async def prepared_slice(tmp_path: Path):
    out = tmp_path / "lake"
    shot, photo = png("rent receipt paid on the 3rd"), jpeg((10, 120, 30))
    inventory(
        out,
        [
            ("v/Screenshot_a.png", len(shot), "p1", "media"),
            ("v/trip.jpg", len(photo), "p2", "media"),
        ],
    )
    store = FakeStore({"v/Screenshot_a.png": shot, "v/trip.jpg": photo})
    fetched = await fetch_slice(out, tmp_path / "spool", lambda p, bk: store)
    sl = load_slice(out, fetched["slice_id"])
    await run_facts(out, tmp_path / "spool", sl)
    await run_ocr_stub(out, tmp_path / "spool", sl)
    return out, sl


async def run_ocr_stub(out, spool, sl):
    """OCR rows written directly: the OCR unit has its own test; this keeps the publish tests
    independent of tesseract."""
    rows = [
        {
            "identity": i.identity,
            "ocr_text": "rent receipt" if i.identity == "p1" else "",
            "ocr_confidence": 90.0,
            "ocr_engine": "tesseract",
        }
        for i in sl.items
    ]
    path = table_path(out, "ocr", sl.slice_id)
    path.parent.mkdir(parents=True, exist_ok=True)
    pq.write_table(pa.Table.from_pylist(rows, schema=image_stage.OCR_SCHEMA), path)


@pytest.mark.asyncio
async def test_embed_by_slot_publish_with_partial_and_release(tmp_path: Path, monkeypatch):
    s = settings(monkeypatch)
    out, sl = await prepared_slice(tmp_path)
    spool = tmp_path / "spool"

    async def bag(data, ext):
        return [[0.5] * 128 for _ in range(3)]

    async def single(data, ext):
        if ext == ".jpg":
            raise RuntimeError("provider down")
        return [0.25] * 2048

    maxsim = await run_embed(out, spool, sl, "image_maxsim", bag, "jina-embeddings-v4", s)
    one = await run_embed(out, spool, sl, "image_single", single, "nim-vl", s)
    assert maxsim["embedded_ok"] == 2 and "Qwen Research License" in maxsim["licence_basis"]
    assert one["embedded_ok"] == 1 and one["embedded_failed"] == 1
    writer = RecordingWriter()
    result = await run_publish(
        out, spool, sl, writer, s, source_id="consignatio-vault-v1", run_id="20261003T000000Z"
    )
    assert result["published"] == 2 and result["partial"] == 1 and result["failed"] == 0
    by_identity = {a.spec.properties["identity"]: a.spec for a in writer.actions}
    shot = by_identity["p1"]
    assert (
        shot.properties["image_kind"] == "screenshot" and shot.properties["is_screenshot"] is True
    )
    assert (
        shot.properties["ocr_text"] == "rent receipt"
        and shot.single is not None
        and len(shot.multi) == 3
    )
    assert (
        shot.properties["vault_key"] == "v/Screenshot_a.png" and shot.properties["provider"] == "b2"
    )
    assert (
        by_identity["p2"].single is None and by_identity["p2"].multi is not None
    )  # NIM failed: written, ledgered partial
    assert not (spool / "images" / sl.slice_id).exists(), "the spool is released after publish"
    # A retry returns the ledger's counts and writes nothing more.
    again = await run_publish(out, spool, sl, writer, s, source_id="x", run_id="y")
    assert again["published"] == 2 and len(writer.actions) == 2
    # The failed one comes back only when asked.
    assert select_images(out, 10, 25_000_000)[0] == []
    assert [p["identity"] for p in select_images(out, 10, 25_000_000, retry_failed=True)[0]] == [
        "p2"
    ]


@pytest.mark.asyncio
async def test_clip_first_tier_gives_photos_only_the_in_database_vector(
    tmp_path: Path, monkeypatch
):
    s = settings(monkeypatch, INTAKE_IMAGES_CLIP="1", INTAKE_IMAGES_PHOTO_TIER="clip_first")
    out, sl = await prepared_slice(tmp_path)
    spool = tmp_path / "spool"

    async def single(data, ext):
        return [0.25] * 2048

    async def bag(data, ext):
        return [[0.5] * 128]

    one = await run_embed(out, spool, sl, "image_single", single, "nim-vl", s)
    await run_embed(out, spool, sl, "image_maxsim", bag, "jina", s)
    assert one["selected"] == 1 and one["skipped_by_policy"] == 1  # only the screenshot goes to NIM
    writer = RecordingWriter()
    await run_publish(out, spool, sl, writer, s, source_id="s", run_id="r")
    photo = next(a.spec for a in writer.actions if a.spec.properties["identity"] == "p2")
    assert photo.single is None and photo.multi is None and photo.properties["thumb"], (
        "CLIP vectorizes the thumbnail"
    )
    assert len(__import__("base64").b64decode(photo.properties["thumb"])) < 30_000


@pytest.mark.asyncio
async def test_embed_refuses_unknown_or_disabled_slots_and_missing_facts(
    tmp_path: Path, monkeypatch
):
    s = settings(monkeypatch, INTAKE_IMAGES_SLOTS="image_single")
    out, sl = await prepared_slice(tmp_path)

    async def noop(data, ext):
        return [1.0]

    with pytest.raises(ValueError):
        await run_embed(out, tmp_path / "spool", sl, "image_maxsim", noop, "m", s)
    with pytest.raises(ValueError):
        await run_embed(out, tmp_path / "spool", sl, "bogus", noop, "m", s)
    facts_path = table_path(out, "facts", sl.slice_id)
    held = out / "to_be_deleted" / "test-facts"
    held.mkdir(parents=True)
    facts_path.rename(held / facts_path.name)
    with pytest.raises(ValueError):
        await run_embed(out, tmp_path / "spool", sl, "image_single", noop, "m", s)


# -------------------------------------------------- schema, writer


def test_collection_schema_carries_catalog_fields_and_the_optional_vectors():
    base = collection_schema("IntakeImageV1")
    names = {p["name"] for p in base["properties"]}
    assert {
        "vault_key",
        "image_kind",
        "provider",
        "bucket",
        "width",
        "occurrences",
        "embed_slots",
    } <= names
    assert set(base["vectorConfig"]) == {"image_single", "image_maxsim"}
    full = collection_schema("IntakeImageV1", clip=True, colqwen=True)
    assert set(full["vectorConfig"]) == {
        "image_single",
        "image_maxsim",
        "image_clip",
        "image_colqwen",
    }
    assert "multi2vec-clip" in full["vectorConfig"]["image_clip"]["vectorizer"]
    assert full["vectorConfig"]["image_colqwen"]["vectorIndexConfig"]["multivector"] == {
        "enabled": True
    }
    assert {"name": "thumb", "dataType": ["blob"]} in full["properties"]


@pytest.mark.asyncio
async def test_ensure_schema_creates_or_adds_missing_properties_only():
    calls = []

    def handler(request: httpx.Request) -> httpx.Response:
        calls.append((request.method, request.url.path))
        if request.method == "GET":
            if len(calls) == 1:
                return httpx.Response(404)
            old = collection_schema("IntakeImageV1")
            old["properties"] = [
                p for p in old["properties"] if p["name"] not in ("image_kind", "width")
            ]
            return httpx.Response(200, json=old)
        return httpx.Response(200, json={})

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    writer = ImageObjectWriter("http://weaviate.test", "IntakeImageV1", 2048, client)
    created = await writer.ensure_schema()
    assert "image_kind" in created and calls[-1] == ("POST", "/v1/schema")
    calls.clear()
    calls.append(("GET", "x"))  # the next GET answers with the old schema
    added = await writer.ensure_schema()
    assert sorted(added) == ["image_kind", "width"]
    assert all(c[1].endswith("/properties") for c in calls if c[0] == "POST")


@pytest.mark.asyncio
async def test_writer_accepts_a_spec_with_only_a_colqwen_bag_and_refuses_an_empty_one():
    puts = []

    def handler(request: httpx.Request) -> httpx.Response:
        puts.append(request)
        return httpx.Response(404 if request.method == "GET" else 200, json={})

    writer = ImageObjectWriter(
        "http://w.test",
        "IntakeImageV1",
        2048,
        httpx.AsyncClient(transport=httpx.MockTransport(handler)),
    )
    from casebible_index.image_target import ImageObjectAction, ImageObjectSpec

    key = ("http://w.test", "IntakeImageV1", "5b9f5f3e-0f5a-5c1a-8a4d-0123456789ab")
    await writer.apply(
        ImageObjectAction(
            key, ImageObjectSpec({"filename": "a"}, None, None, {"image_colqwen": [[0.1] * 128]})
        )
    )
    import json

    body = json.loads(next(r for r in puts if r.method == "POST").content)
    assert list(body["vectors"]) == ["image_colqwen"] and body["properties"]["active"] is True
    with pytest.raises(ValueError):
        await writer.apply(
            ImageObjectAction(key, ImageObjectSpec({"filename": "a"}, None, None, None))
        )


# -------------------------------------------------- colqwen client, wiring


@pytest.mark.asyncio
async def test_colqwen_client_follows_its_contract_and_retries_a_cold_endpoint():
    from casebible_index.image_embedders import ColQwenEmbedder

    seen = []

    def handler(request: httpx.Request) -> httpx.Response:
        seen.append((request.url.path, request.headers.get("authorization")))
        if len(seen) == 1:
            return httpx.Response(503)
        return httpx.Response(200, json={"embeddings": [[0.1] * 128, [0.2] * 128]})

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    colqwen = ColQwenEmbedder(client, "https://gpu.test/", "tok", max_retries=2)
    import casebible_index.image_embedders as module

    module.asyncio.sleep = _no_sleep  # type: ignore[assignment]
    bag = await colqwen.embed_multi(b"img")
    assert len(bag) == 2 and seen[-1] == ("/embed_image", "Bearer tok")
    assert len(await colqwen.embed_query("pickup")) == 2 and seen[-1][0] == "/embed_query"


async def _no_sleep(_):
    return None


@pytest.mark.asyncio
async def test_stage_runner_image_stages_are_registered_and_off_by_default(monkeypatch):
    from casebible_index import stage_runner

    for name in ("image_fetch", "image_facts", "image_ocr", "image_embed", "image_publish"):
        assert name in stage_runner.STAGES
    monkeypatch.delenv("INTAKE_IMAGE_STAGE", raising=False)
    skipped = await stage_runner.stage_image_fetch("20261003T000000Z", {})
    assert skipped["slice_id"] == "" and "off" in skipped["skipped"]
