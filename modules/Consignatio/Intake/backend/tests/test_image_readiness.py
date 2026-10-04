"""Exercise proof fences, retained spool material and collection compatibility.

Inputs are synthetic inventories and mocked providers; outputs are assertions.
Side effects stay in pytest's retained temporary directory; no original corpus is read.
Pick these focused checks before a bounded server proof.
"""

import io
from pathlib import Path

import httpx
import pyarrow as pa
import pyarrow.parquet as pq
import pytest
from PIL import Image

from casebible_index.image_slice import fetch_slice, quarantine_spool, select_images
from casebible_index.image_target import ImageObjectWriter, collection_schema


def _inventory(out: Path, keys: list[str]) -> bytes:
    """Write a synthetic inventory and return its image payload.

    Inputs: output folder and exact keys. Output: PNG bytes. Side effects: Parquet write.
    Pick this fixture to test catalog selection without a real bucket.
    """
    data = io.BytesIO()
    Image.new("RGB", (20, 20), "white").save(data, format="PNG")
    payload = data.getvalue()
    path = out / "inventory" / "represented" / "proof.parquet"
    path.parent.mkdir(parents=True)
    pq.write_table(
        pa.Table.from_pylist(
            [
                {
                    "provider": "b2",
                    "bucket": "proof",
                    "key": key,
                    "size": len(payload),
                    "sha1": str(n),
                    "kind": "media",
                }
                for n, key in enumerate(keys)
            ]
        ),
        path,
    )
    return payload


@pytest.mark.asyncio
async def test_exact_manifest_and_remaining_budget_do_not_fetch_neighbouring_originals(tmp_path):
    out = tmp_path / "out"
    payload = _inventory(out, ["proof/a.png", "proof/b.png", "proof/unapproved.png"])
    reads = []

    class Store:
        async def stream(self, key, *, window_bytes):
            reads.append(key)
            yield payload

    locators = [
        {"provider": "b2", "bucket": "proof", "key": key} for key in ("proof/a.png", "proof/b.png")
    ]
    result = await fetch_slice(out, None, lambda *_: Store(), max_items=1, locators=locators)
    assert result["count"] == 1
    assert reads == ["proof/a.png"]
    with pytest.raises(ValueError, match="outside this proof"):
        await fetch_slice(out, None, lambda *_: Store(), path_prefix="other/")
    assert reads == ["proof/a.png"]
    assert select_images(out, 5, 1000, locators=[])[0] == []


def test_spool_release_retains_exact_bytes_and_refuses_paths_outside_spool(tmp_path):
    out = tmp_path / "out"
    source = out / "spool" / "images" / "slice"
    source.mkdir(parents=True)
    (source / "original-copy.png").write_bytes(b"retained")
    counts = quarantine_spool(out, None, source)
    assert counts == {"files": 1, "bytes": 8}
    held = list((out / "to_be_deleted").rglob("original-copy.png"))
    assert len(held) == 1 and held[0].read_bytes() == b"retained"
    assert not source.exists()
    with pytest.raises(ValueError, match="child of"):
        quarantine_spool(out, None, tmp_path / "outside")


@pytest.mark.asyncio
async def test_incompatible_existing_maxsim_shape_is_rejected_before_any_schema_write():
    calls = []
    schema = collection_schema("ProofImages")
    schema["vectorConfig"]["image_maxsim"]["vectorIndexConfig"]["multivector"] = {"enabled": False}
    schema["properties"] = []

    def handler(request):
        calls.append(request.method)
        return httpx.Response(200, json=schema)

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
        writer = ImageObjectWriter("http://weaviate.test", "ProofImages", 2048, client)
        with pytest.raises(ValueError, match="multi-vector"):
            await writer.ensure_schema()
    assert calls == ["GET"]


@pytest.mark.asyncio
async def test_pdf_source_and_rendered_page_byte_caps_retain_overflow_and_never_publish_it(
    tmp_path,
):
    """Oversized originals are not read and one overflowing raster is retained as a terminal
    failure.

    Inputs: a synthetic PDF and restrictive source or slice bounds. Output: receipt assertions.
    Side effects: writes retained fixtures only. Pick this to test both sides of the PDF byte fence.
    """
    buffer = io.BytesIO()
    Image.new("RGB", (72, 72), "white").save(buffer, format="PDF")
    payload = buffer.getvalue()
    reads = []

    class Store:
        async def stream(self, key, *, window_bytes):
            reads.append(key)
            yield payload

    for name, max_file_bytes, max_bytes, expected_reason in (
        ("source", len(payload) - 1, 256 * 1024 * 1024, "over_max_file_bytes"),
        ("page", len(payload) + 4096, len(payload) + 1, "pdf_page_exceeds_slice_byte_budget"),
    ):
        out = tmp_path / name
        inventory = out / "inventory" / "pdf" / "proof.parquet"
        inventory.parent.mkdir(parents=True)
        pq.write_table(
            pa.Table.from_pylist(
                [
                    {
                        "provider": "b2",
                        "bucket": "proof",
                        "key": "proof/scan.pdf",
                        "size": len(payload),
                        "sha1": "pdf1",
                        "kind": "document",
                    }
                ]
            ),
            inventory,
        )
        docs = out / "datasets" / "documents"
        docs.mkdir(parents=True)
        pq.write_table(
            pa.Table.from_pylist(
                [
                    {
                        "content_sha256": "sha1:pdf1",
                        "index_status": "skipped_no_text",
                        "extension": ".pdf",
                        "member_path": "",
                    }
                ]
            ),
            docs / "proof.parquet",
        )
        result = await fetch_slice(
            out, None, lambda *_: Store(), max_file_bytes=max_file_bytes, max_bytes=max_bytes
        )
        assert result["count"] == 0 and result["failed"] == 1
        from casebible_index.image_slice import load_slice

        sl = load_slice(out, result["slice_id"])
        assert sl.failed[0]["reason"] == expected_reason
        if name == "source":
            assert reads == [] and result["quarantined_files"] == 0
        else:
            assert reads == ["proof/scan.pdf"]
            assert result["quarantined_files"] == 2
            assert result["quarantined_bytes"] > len(payload)
            assert not any((out / "spool" / "images").rglob("*.png"))
