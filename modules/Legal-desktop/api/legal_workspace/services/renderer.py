"""Office-document rendering through the `legal-renderer` sidecar.

> _Byline: Claude Code · Fable 5.1 · 2026-09-20_
The sidecar is Gotenberg (headless LibreOffice behind an HTTP API). This
module only converts owner-produced work product (DOCX/ODT/RTF/...) to PDF.
Not Agno evidence. Not court-safe. Fails closed when the sidecar is down —
there is no local fallback renderer.
"""

from __future__ import annotations

import hashlib
import re
from pathlib import Path

import httpx
from pydantic import BaseModel

from legal_workspace.config import get_settings

# Formats LibreOffice converts reliably; anything else is rejected before upload.
OFFICE_SUFFIXES = frozenset(
    {".docx", ".doc", ".odt", ".rtf", ".txt", ".xlsx", ".xls", ".ods", ".pptx", ".ppt", ".odp"}
)

_SAFE_NAME_RE = re.compile(r"[^A-Za-z0-9._-]+")


class RendererUnavailable(RuntimeError):
    """The legal-renderer sidecar could not be reached."""


class RenderResult(BaseModel):
    ok: bool
    engine: str
    source_name: str
    output_name: str
    output_path: str
    content_hash: str
    size_bytes: int
    court_safe: bool = False
    exportable: bool = False


def safe_document_name(name: str) -> str:
    """Flatten an uploaded filename to one safe path segment."""
    cleaned = _SAFE_NAME_RE.sub("_", Path(name).name).strip("._")
    if not cleaned:
        raise ValueError("a file name is required")
    return cleaned


def convert_office_to_pdf(src: Path, dest: Path, *, base_url: str | None = None) -> RenderResult:
    """Send `src` to LibreOffice (via Gotenberg) and write the PDF to `dest`."""
    if src.suffix.lower() not in OFFICE_SUFFIXES:
        raise ValueError(f"unsupported office format: {src.suffix or '(none)'}")
    url = (base_url or get_settings().legal_renderer_base_url).rstrip("/")
    try:
        with src.open("rb") as handle, httpx.Client(timeout=180.0) as client:
            response = client.post(
                f"{url}/forms/libreoffice/convert",
                files={"files": (src.name, handle)},
            )
    except httpx.HTTPError as exc:
        raise RendererUnavailable(f"legal-renderer unreachable: {exc}") from exc
    if response.status_code != 200:
        raise ValueError(
            f"legal-renderer rejected {src.name}: {response.status_code} {response.text[:200]}"
        )
    body = response.content
    if not body.startswith(b"%PDF"):
        raise ValueError(f"legal-renderer returned a non-PDF body for {src.name}")
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_bytes(body)
    return RenderResult(
        ok=True,
        engine="libreoffice/gotenberg",
        source_name=src.name,
        output_name=dest.name,
        output_path=str(dest),
        content_hash=hashlib.sha256(body).hexdigest(),
        size_bytes=len(body),
    )
