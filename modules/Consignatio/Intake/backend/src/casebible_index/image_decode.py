"""Bounded Pillow decoding and JPEG derivatives for the image index."""

from __future__ import annotations

import io
from pathlib import Path

from PIL import Image, ImageOps
from pillow_heif import register_heif_opener

HEIF_EXTENSIONS = frozenset({".heic", ".heif"})
NATIVE_PROVIDER_EXTENSIONS = frozenset({".png", ".jpg", ".jpeg", ".webp"})
MAX_DECODE_PIXELS = 50_000_000


def open_image(source: Path | io.BytesIO) -> Image.Image:
    """Open one image with HEIF support and reject excessive decoded dimensions.

    Inputs: a retained path or in-memory image stream. Output: a Pillow image that the caller
    closes. Side effects: registers the HEIF Pillow plugin process-wide and reads image headers.
    Pick this for image-lane decoding, including HEIC/HEIF and ordinary Pillow formats.
    """
    register_heif_opener(thumbnails=False)
    image = Image.open(source)
    if image.width * image.height > MAX_DECODE_PIXELS:
        image.close()
        raise Image.DecompressionBombError("image exceeds the image-lane pixel limit")
    return image


def jpeg_derivative(source: Path | bytes, *, edge: int, quality: int = 90) -> bytes:
    """Make an oriented, bounded JPEG from an original image without changing the original.

    Inputs: retained path or bytes, maximum long edge, and JPEG quality. Output: JPEG bytes.
    Side effects: decodes one image in memory. Pick this for providers and OCR that cannot read
    HEIC/HEIF; original metadata and content hashes must still come from the source.
    """
    if edge <= 0:
        raise ValueError("edge must be positive")
    stream = io.BytesIO(source) if isinstance(source, bytes) else source
    with open_image(stream) as image:
        image.seek(0)
        image.draft(None, (edge, edge))
        oriented = ImageOps.exif_transpose(image)
        converted = oriented.convert("RGB")
        converted.thumbnail((edge, edge))
        buffer = io.BytesIO()
        converted.save(buffer, format="JPEG", quality=quality)
        return buffer.getvalue()
