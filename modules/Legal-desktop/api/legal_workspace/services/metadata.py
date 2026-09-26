"""File metadata: read anything with exiftool, scrub PDFs with pikepdf.

> _Byline: Claude Code · Fable 5.1 · 2026-09-21_
CAT5 section E / DOC-10. exiftool is installed in the legal-api image and reads
images (EXIF, GPS, maker notes, XMP edit history), video, audio, office files
and PDFs. Reading never changes the file. A scrub writes a new PDF copy and
reports what exiftool still sees afterwards; it is never for an evidence
original. Not court-safe.
"""

from __future__ import annotations

import hashlib
import json
import shutil
import subprocess
from pathlib import Path

from pydantic import BaseModel

from legal_workspace.services.original_time import OriginalTime, resolve_original_time

# exiftool groups that describe the file on disk or the tool run, not the content's own metadata.
_STRUCTURAL_GROUPS = frozenset({"ExifTool", "System", "File"})
_STRUCTURAL_PDF = frozenset({"PDF:PDFVersion", "PDF:Linearized", "PDF:PageCount"})

# What an image/video reviewer looks at first, keyed by the tag name after the group.
_SUMMARY_TAGS = {
    "captured": ("DateTimeOriginal", "CreateDate", "MediaCreateDate", "CreationDate"),
    "captured_offset": ("OffsetTimeOriginal", "OffsetTime"),
    "modified": ("ModifyDate", "MetadataDate", "MediaModifyDate"),
    "device_make": ("Make",),
    "device_model": ("Model",),
    "device_serial": ("BodySerialNumber", "SerialNumber", "InternalSerialNumber"),
    "lens": ("LensModel", "LensID"),
    "host_computer": ("HostComputer",),
    "image_unique_id": ("ImageUniqueID", "DocumentID", "OriginalDocumentID"),
    "user_comment": ("UserComment",),
    "software": ("Software", "CreatorTool", "HistorySoftwareAgent", "Producer"),
    "gps_latitude": ("GPSLatitude",),
    "gps_longitude": ("GPSLongitude",),
    "gps_timestamp": ("GPSDateTime", "GPSDateStamp"),
    "width": ("ImageWidth", "ExifImageWidth"),
    "height": ("ImageHeight", "ExifImageHeight"),
    "duration": ("Duration",),
    "author": ("Author", "Artist", "Creator", "By-line"),
}


class ExiftoolUnavailable(RuntimeError):
    """The exiftool binary is not installed where legal-api runs."""


class MetadataReport(BaseModel):
    ok: bool
    engine: str
    source_name: str
    content_hash: str
    size_bytes: int
    file_type: str | None
    mime_type: str | None
    summary: dict[str, object]
    original_time: OriginalTime
    has_gps: bool
    metadata: dict[str, object]
    authored_fields: list[str]
    court_safe: bool = False


class MetadataScrubResult(BaseModel):
    ok: bool
    source_name: str
    output_name: str
    output_path: str
    content_hash: str
    removed_fields: list[str]
    remaining_authored_fields: list[str]
    court_safe: bool = False
    exportable: bool = False


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _summary(fields: dict[str, object]) -> dict[str, object]:
    by_tag: dict[str, object] = {}
    for key, value in fields.items():
        by_tag.setdefault(key.split(":", 1)[-1], value)
    # Raw GPS tags are unsigned with a separate N/S/E/W ref; exiftool's Composite values
    # carry the sign (west and south negative), so they win.
    for key, value in fields.items():
        if key.startswith("Composite:GPS"):
            by_tag[key.split(":", 1)[-1]] = value
    for tag in ("GPSLatitude", "GPSLongitude"):
        try:
            by_tag[tag] = float(str(by_tag[tag]))
        except (KeyError, ValueError):
            pass
    out: dict[str, object] = {}
    for label, tags in _SUMMARY_TAGS.items():
        for tag in tags:
            if tag in by_tag:
                out[label] = by_tag[tag]
                break
    return out


def read_file_metadata(src: Path, *, takeout_sidecar_json: str | None = None) -> MetadataReport:
    """Every metadata field exiftool reports for `src`, grouped (`EXIF:`, `GPS:`, `XMP-…:`)."""
    binary = shutil.which("exiftool")
    if binary is None:
        raise ExiftoolUnavailable("exiftool is not installed on this host")
    try:
        done = subprocess.run(  # fixed argv, no shell
            [binary, "-json", "-G1", "-a", "-struct", "-c", "%+.6f",
             "-api", "largefilesupport=1", str(src)],
            capture_output=True,
            timeout=120,
            check=False,
        )
    except subprocess.TimeoutExpired as exc:
        raise ValueError(f"exiftool timed out on {src.name}") from exc
    try:
        fields = dict(json.loads(done.stdout.decode("utf-8", "replace"))[0])
    except (ValueError, IndexError) as exc:
        detail = done.stderr.decode("utf-8", "replace").strip()[:200]
        raise ValueError(f"exiftool could not read {src.name}: {detail}") from exc
    fields.pop("SourceFile", None)
    authored = sorted(
        key
        for key in fields
        if key.split(":", 1)[0] not in _STRUCTURAL_GROUPS and key not in _STRUCTURAL_PDF
    )
    summary = _summary(fields)
    return MetadataReport(
        ok=True,
        engine=f"exiftool {fields.get('ExifTool:ExifToolVersion', '')}".strip(),
        source_name=src.name,
        content_hash=_sha256(src),
        size_bytes=src.stat().st_size,
        file_type=fields.get("File:FileType"),
        mime_type=fields.get("File:MIMEType"),
        summary=summary,
        original_time=resolve_original_time(fields, src.name, takeout_sidecar_json),
        has_gps="gps_latitude" in summary and "gps_longitude" in summary,
        metadata=fields,
        authored_fields=authored,
    )


def scrub_pdf_metadata(src: Path, dest: Path) -> MetadataScrubResult:
    """Write a copy of `src` with the document-info dictionary and XMP packet removed."""
    from pikepdf import Pdf

    before = read_file_metadata(src)
    with Pdf.open(src) as pdf:
        if "/Info" in pdf.trailer:
            del pdf.trailer["/Info"]
        if "/Metadata" in pdf.Root:
            del pdf.Root["/Metadata"]
        dest.parent.mkdir(parents=True, exist_ok=True)
        # A fresh, non-incremental save drops the unreferenced old objects too.
        pdf.save(dest, deterministic_id=True)
    after = read_file_metadata(dest)
    return MetadataScrubResult(
        ok=True,
        source_name=src.name,
        output_name=dest.name,
        output_path=str(dest),
        content_hash=after.content_hash,
        removed_fields=sorted(set(before.authored_fields) - set(after.authored_fields)),
        remaining_authored_fields=after.authored_fields,
    )
