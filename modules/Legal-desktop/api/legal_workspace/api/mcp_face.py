"""MCP face of legal-api: one `advocatio` gateway for ContextForge.

> _Byline: Claude Code · Fable 5.1 · 2026-09-21_
Owner decision 2026-09-21 (option A): every workdesk tool is registered in
ContextForge by arriving through this one MCP endpoint, mounted at /mcp behind
the same auth middleware as the HTTP API. A tool here is a thin function over
an existing service; add one per capability as it goes live. Output is work
product: not evidence, not court-safe.
"""

from __future__ import annotations

import base64
import binascii

from fastmcp import FastMCP

from legal_workspace.api.document_routes import (
    convert_document_bytes,
    ocr_image_bytes,
    read_metadata_bytes,
    scrub_metadata_bytes,
)

mcp = FastMCP("advocatio")

# MCP arguments are JSON, so files arrive base64-encoded and are capped.
_INLINE_MAX_BYTES = 25 * 1024 * 1024


def _decode(content_base64: str) -> bytes:
    try:
        data = base64.b64decode(content_base64, validate=True)
    except (binascii.Error, ValueError) as exc:
        raise ValueError("content_base64 is not valid base64") from exc
    if not data:
        raise ValueError("content_base64 is empty")
    if len(data) > _INLINE_MAX_BYTES:
        raise ValueError("inline documents are limited to 25 MiB")
    return data


@mcp.tool
def convert_office_document_to_pdf(filename: str, content_base64: str) -> dict:
    """Convert an owner-produced office file (DOCX, DOC, ODT, RTF, TXT, XLSX, ODS, PPTX, ODP) to PDF
    with LibreOffice. `filename` keeps its office extension; `content_base64` is the file's bytes.
    Returns the stored PDF name, sha256 and size; fetch it from /v1/documents/renders/{output_name}.
    """
    return convert_document_bytes(filename, _decode(content_base64)).model_dump()


@mcp.tool
def read_file_metadata(
    filename: str, content_base64: str, takeout_sidecar_json: str | None = None
) -> dict:
    """Read every metadata field exiftool finds in a file: photos and screenshots (EXIF capture time,
    GPS, phone/camera make, model and serial, editing software, XMP edit history), video, audio,
    office files, PDFs. `original_time` resolves when the image was originally captured and names its
    source (EXIF, Takeout sidecar, embedded creation time, or the device-generated filename), with
    every candidate listed and a conflict flag when they disagree. Pass the ORIGINAL filename: names
    like Screenshot_20240312-141502.png carry the capture time. `takeout_sidecar_json` is the text of
    a Google Takeout `<image>.json` sidecar if one exists. The file is never changed or kept.
    Files over 25 MiB go to POST /v1/documents:metadata instead.
    """
    return read_metadata_bytes(
        filename, _decode(content_base64), takeout_sidecar_json=takeout_sidecar_json
    ).model_dump()


@mcp.tool
def order_images_by_original_time(images: list[dict]) -> dict:
    """Put a series of screenshots or photos in chronological order by original capture time.
    `images` is a list of {"filename", "content_base64", optional "takeout_sidecar_json"}; use the
    original filenames. Returns `ordered` (earliest first: filename, sha256, resolved time, source,
    confidence, conflict flag, device) and `unresolved` for files with no recoverable time. Times
    without a timezone are device-local wall-clock; mixing them with UTC values is flagged.
    """
    rows = []
    for item in images:
        report = read_metadata_bytes(
            item["filename"],
            _decode(item["content_base64"]),
            takeout_sidecar_json=item.get("takeout_sidecar_json"),
        )
        device = " ".join(
            str(report.summary[key]) for key in ("device_make", "device_model") if key in report.summary
        )
        rows.append(
            {
                "filename": item["filename"],
                "content_hash": report.content_hash,
                "original_time": report.original_time.value,
                "source": report.original_time.source,
                "confidence": report.original_time.confidence,
                "timezone_known": report.original_time.timezone_known,
                "conflict": report.original_time.conflict,
                "device": device or None,
            }
        )
    resolved = sorted(
        (row for row in rows if row["original_time"]), key=lambda row: row["original_time"][:19]
    )
    return {
        "ordered": resolved,
        "unresolved": [row for row in rows if not row["original_time"]],
        "mixed_timezone_basis": len({row["timezone_known"] for row in resolved}) > 1,
    }


@mcp.tool
def scrub_pdf_metadata(filename: str, content_base64: str) -> dict:
    """Write a copy of an owner-produced PDF with its document-info dictionary and XMP packet removed,
    then re-read it with exiftool. Returns the stored `<stem>.scrubbed.pdf` name, sha256, the fields
    removed and any authored fields still present. Never use on an evidence original.
    """
    return scrub_metadata_bytes(filename, _decode(content_base64)).model_dump()


@mcp.tool
def ocr_image(
    filename: str, content_base64: str, language: str = "eng", layout: str = "auto"
) -> dict:
    """Read the text in a screenshot or photographed document with Tesseract OCR (PNG, JPG, WEBP,
    TIFF, BMP, GIF). Returns the full text, each line with its confidence and pixel box, word count
    and mean confidence. `layout`: "auto", "block" (one column, e.g. a message thread), or "sparse"
    (scattered UI text). OCR text is a machine-read derivative, never equal to a native export.
    """
    return ocr_image_bytes(
        filename, _decode(content_base64), language=language, layout=layout
    ).model_dump()


mcp_app = mcp.http_app(path="/", stateless_http=True, json_response=True)
