"""Prove HEIF decoding and persistent image scope with synthetic retained originals."""

import hashlib
import io
import json
import shutil
import subprocess
from pathlib import Path

import pyarrow as pa
import pyarrow.parquet as pq
import pytest
from PIL import Image, ImageOps, UnidentifiedImageError
from pillow_heif import register_heif_opener

from casebible_index.image_decode import jpeg_derivative, open_image
from casebible_index.image_embedders import mime_for
from casebible_index.image_facts import read_image_facts
from casebible_index.image_slice import (
    ImageItem,
    Slice,
    fetch_slice,
    manifest_path,
    parse_image_allowlist,
    select_images,
)
from casebible_index.image_stage import prepare_for_embedding, thumbnail_b64
from casebible_index.parquet_store import write_json_immutable
from casebible_index.stage_runner import _scoped_image_slice
from casebible_index.image_stage import ImageStageSettings
from casebible_index.image_ocr import read_text
from casebible_index import image_ocr
from casebible_index.stage_runner import stage_image_fetch


def _heif(path: Path) -> None:
    """Encode a tiny real HEIF original for decoder integration checks.

    Inputs: fixture path. Output: none. Side effects: writes only the test fixture. Pick this over
    a renamed JPEG so the libheif codec is exercised.
    """
    register_heif_opener(thumbnails=False)
    with Image.new("RGB", (32, 20), "red") as image:
        image.save(path, format="HEIF")


@pytest.mark.parametrize("extension", [".heic", ".heif"])
def test_real_heif_decodes_to_oriented_provider_jpeg_and_keeps_source_bytes(tmp_path, extension):
    original = tmp_path / f"camera{extension}"
    _heif(original)
    before = hashlib.sha256(original.read_bytes()).hexdigest()
    with open_image(original) as decoded:
        assert decoded.format == "HEIF" and decoded.size == (32, 20)
    data, ext = prepare_for_embedding(original)
    assert ext == ".jpg" and data[:3] == b"\xff\xd8\xff"
    with Image.open(io.BytesIO(data)) as derivative:
        assert derivative.format == "JPEG" and derivative.size == (32, 20)
    assert thumbnail_b64(original)
    assert hashlib.sha256(original.read_bytes()).hexdigest() == before
    assert mime_for(extension) == f"image/{extension[1:]}"


def test_heif_embedded_facts_are_read_from_original_not_jpeg(tmp_path):
    binary = shutil.which("exiftool")
    if binary is None:
        pytest.skip("exiftool is required for original HEIF metadata proof")
    original = tmp_path / "camera.heic"
    _heif(original)
    command = [
        binary, "-overwrite_original", "-EXIF:DateTimeOriginal=2024:05:06 12:13:14",
        "-EXIF:Make=ProofCamera", "-EXIF:Model=Model1",
        "-EXIF:Orientation=Rotate 90 CW", "-EXIF:GPSLatitude=42.5",
        "-EXIF:GPSLatitudeRef=N", "-EXIF:GPSLongitude=83.25",
        "-EXIF:GPSLongitudeRef=W", str(original),
    ]
    done = subprocess.run(command, capture_output=True, check=False, timeout=30)
    assert done.returncode == 0, done.stderr.decode("utf-8", "replace")
    facts = read_image_facts(
        original, original.name, takeout_sidecar_json=None, ocr=False, max_ocr_chars=0
    )
    assert "2024" in (facts.original_time.value or "")
    assert facts.device == "ProofCamera Model1"
    assert "42.5" in facts.gps and "83.25" in facts.gps
    with open_image(original) as decoded:
        oriented_size = ImageOps.exif_transpose(decoded).size
    with Image.open(io.BytesIO(jpeg_derivative(original, edge=2048))) as derivative:
        assert derivative.size == oriented_size


def test_jpeg_regression_and_corrupt_heif_error(tmp_path):
    photo = tmp_path / "photo.jpg"
    Image.new("RGB", (24, 12), "blue").save(photo, format="JPEG")
    data, ext = prepare_for_embedding(photo)
    assert (data, ext) == (photo.read_bytes(), ".jpg")
    invalid = tmp_path / "broken.heic"
    invalid.write_bytes(b"broken HEIC")
    with pytest.raises((UnidentifiedImageError, OSError, ValueError)):
        prepare_for_embedding(invalid)


def test_heif_ocr_receives_jpeg_bytes_without_writing_a_derivative(tmp_path, monkeypatch):
    original = tmp_path / "camera.heic"
    _heif(original)
    before = sorted(tmp_path.iterdir())
    seen = []

    def record(source, max_chars):
        seen.append((source, max_chars))
        return "proof", 99.0

    monkeypatch.setitem(image_ocr.OCR_ENGINES, "tesseract", record)
    assert read_text("tesseract", original, 100) == ("proof", 99.0)
    assert seen[0][0][:3] == b"\xff\xd8\xff"
    assert sorted(tmp_path.iterdir()) == before


def _inventory(root: Path) -> None:
    """Write an invented catalog with two allowed and two disallowed source folders.

    Inputs: test lake path. Output: none. Side effects: fixture Parquet only. Pick this to prove
    selection filters before content identities are grouped.
    """
    path = root / "inventory" / "represented" / "proof.parquet"
    path.parent.mkdir(parents=True)
    rows = [
        {"provider": "b2", "bucket": "salem-data", "key": "selected/a.heic", "size": 100, "sha1": "a", "kind": "media"},
        {"provider": "b2", "bucket": "salem-data", "key": "selected/b.heif", "size": 100, "sha1": "b", "kind": "media"},
        {"provider": "b2", "bucket": "salem-data", "key": "selected-neighbor/c.heic", "size": 100, "sha1": "c", "kind": "media"},
        {"provider": "b2", "bucket": "other", "key": "selected/d.heic", "size": 100, "sha1": "d", "kind": "media"},
    ]
    pq.write_table(pa.Table.from_pylist(rows), path)


def test_persistent_allowlist_intersects_manual_selector_before_query(tmp_path):
    _inventory(tmp_path)
    scope = parse_image_allowlist(json.dumps([
        {"provider": "b2", "bucket": "salem-data", "path_prefix": "selected/"}
    ]))
    rows, more, skipped = select_images(tmp_path, 10, 1000, allowlist=scope)
    assert {row["key"] for row in rows} == {"selected/a.heic", "selected/b.heif"}
    assert not more and skipped["unsupported_format"] == 0
    rows, _, _ = select_images(tmp_path, 10, 1000, allowlist=scope, locators=[
        {"provider": "b2", "bucket": "salem-data", "key": "selected/b.heif"},
        {"provider": "b2", "bucket": "salem-data", "key": "selected-neighbor/c.heic"},
    ])
    assert [row["key"] for row in rows] == ["selected/b.heif"]
    for bad in ("[]", "{", '[{"provider":"b2","bucket":"salem-data","path_prefix":""}]'):
        with pytest.raises(ValueError):
            parse_image_allowlist(bad)


@pytest.mark.asyncio
async def test_old_pending_slice_outside_current_scope_rejected_before_bucket_read(tmp_path):
    sl = Slice("old", "image", [ImageItem("a", "a", "b2", "salem-data", "outside/a.heic", 10, 1, "000.heic")])
    write_json_immutable(manifest_path(tmp_path, "old"), sl.to_json())
    scope = parse_image_allowlist(json.dumps([
        {"provider": "b2", "bucket": "salem-data", "path_prefix": "selected/"}
    ]))
    reads = []
    def store_for(*args):
        reads.append(args)
        raise AssertionError("bucket must not be read")
    with pytest.raises(ValueError, match="outside"):
        await fetch_slice(tmp_path, None, store_for, allowlist=scope)
    assert reads == []
    image = ImageStageSettings.from_env()
    image = __import__("dataclasses").replace(image, allowed_prefixes=scope)
    with pytest.raises(ValueError, match="outside"):
        _scoped_image_slice(tmp_path, "old", image)


@pytest.mark.asyncio
async def test_automatic_fetch_requires_persistent_allowlist_or_exact_locators(monkeypatch):
    monkeypatch.setenv("INTAKE_IMAGE_STAGE", "on")
    monkeypatch.delenv("INTAKE_IMAGES_ALLOWED_PREFIXES_JSON", raising=False)
    with pytest.raises(ValueError, match="requires INTAKE_IMAGES_ALLOWED_PREFIXES_JSON"):
        await stage_image_fetch("proof", {})
