"""Document rendering HTTP surface.

> _Byline: Claude Code · Fable 5.1 · 2026-09-20_
POST /v1/documents:convert sends an owner-produced office file to the
`legal-renderer` sidecar (LibreOffice) and stores the PDF under
`<store>/renders/`. GET /v1/documents/renders/{name} returns it. Not Agno
evidence. Not court-safe. Parent mounts this router on the FastAPI app.
"""

from __future__ import annotations

import tempfile
from pathlib import Path
from typing import Annotated

from fastapi import APIRouter, File, Form, HTTPException, UploadFile
from fastapi.responses import FileResponse

from legal_workspace.services.metadata import (
    ExiftoolUnavailable,
    MetadataReport,
    MetadataScrubResult,
    read_file_metadata,
    scrub_pdf_metadata,
)
from legal_workspace.services.ocr import OcrResult, OcrUnavailable, ocr_image
from legal_workspace.services.renderer import (
    OFFICE_SUFFIXES,
    RendererUnavailable,
    RenderResult,
    convert_office_to_pdf,
    safe_document_name,
)
from legal_workspace.services.workspace import WORKSPACE

router = APIRouter()

def _renders_dir() -> Path:
    folder = WORKSPACE.store_dir / "renders"
    folder.mkdir(parents=True, exist_ok=True)
    return folder


@router.post("/v1/documents:convert", response_model=RenderResult)
async def convert_owner_document(file: Annotated[UploadFile, File()]) -> RenderResult:
    """LibreOffice conversion of an owner-produced office file to PDF."""
    try:
        return convert_document_bytes(file.filename or "", await file.read())
    except RendererUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc


def convert_document_bytes(filename: str, data: bytes) -> RenderResult:
    """Store an office upload under renders/ and convert it. Raises ValueError / RendererUnavailable."""
    name = safe_document_name(filename)
    # Refuse before anything touches disk.
    if Path(name).suffix.lower() not in OFFICE_SUFFIXES:
        raise ValueError(f"unsupported office format: {Path(name).suffix or '(none)'}")
    folder = _renders_dir()
    src = folder / name
    src.write_bytes(data)
    return convert_office_to_pdf(src, folder / f"{Path(name).stem}.pdf")


def _checked_pdf_name(filename: str, data: bytes) -> str:
    name = safe_document_name(filename)
    if Path(name).suffix.lower() != ".pdf" or not data.startswith(b"%PDF"):
        raise ValueError("a PDF is required")
    return name


def read_metadata_bytes(
    filename: str, data: bytes, *, takeout_sidecar_json: str | None = None
) -> MetadataReport:
    """exiftool report for any file (image, video, audio, office, PDF). Temp folder only."""
    name = safe_document_name(filename)
    with tempfile.TemporaryDirectory() as work:
        src = Path(work) / name
        src.write_bytes(data)
        return read_file_metadata(src, takeout_sidecar_json=takeout_sidecar_json)


def scrub_metadata_bytes(filename: str, data: bytes) -> MetadataScrubResult:
    """Store `<stem>.scrubbed.pdf` with document info and XMP removed; the upload is not kept."""
    name = _checked_pdf_name(filename, data)
    with tempfile.TemporaryDirectory() as work:
        src = Path(work) / name
        src.write_bytes(data)
        return scrub_pdf_metadata(src, _renders_dir() / f"{Path(name).stem}.scrubbed.pdf")


@router.post("/v1/documents:metadata", response_model=MetadataReport)
async def read_owner_file_metadata(
    file: Annotated[UploadFile, File()],
    takeout_sidecar_json: Annotated[str | None, Form()] = None,
) -> MetadataReport:
    """exiftool report for any uploaded file. Streamed to a temp folder, never kept, never changed."""
    try:
        name = safe_document_name(file.filename or "")
        with tempfile.TemporaryDirectory() as work:
            src = Path(work) / name
            with src.open("wb") as handle:
                while block := await file.read(1024 * 1024):
                    handle.write(block)
            return read_file_metadata(src, takeout_sidecar_json=takeout_sidecar_json or None)
    except ExiftoolUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc


@router.post("/v1/documents:scrub-metadata", response_model=MetadataScrubResult)
async def scrub_owner_pdf_metadata(file: Annotated[UploadFile, File()]) -> MetadataScrubResult:
    try:
        return scrub_metadata_bytes(file.filename or "", await file.read())
    except ExiftoolUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc


def ocr_image_bytes(
    filename: str, data: bytes, *, language: str = "eng", layout: str = "auto"
) -> OcrResult:
    """Tesseract OCR of a screenshot or photographed document. Temp folder only."""
    name = safe_document_name(filename)
    with tempfile.TemporaryDirectory() as work:
        src = Path(work) / name
        src.write_bytes(data)
        return ocr_image(src, language=language, layout=layout)


@router.post("/v1/documents:ocr", response_model=OcrResult)
async def ocr_owner_image(
    file: Annotated[UploadFile, File()],
    language: Annotated[str, Form()] = "eng",
    layout: Annotated[str, Form()] = "auto",
) -> OcrResult:
    try:
        return ocr_image_bytes(
            file.filename or "", await file.read(), language=language, layout=layout
        )
    except OcrUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc


@router.get("/v1/documents/renders/{name}")
def get_rendered_document(name: str) -> FileResponse:
    try:
        safe = safe_document_name(name)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    path = _renders_dir() / safe
    if safe != name or not path.is_file():
        raise HTTPException(status_code=404, detail="render not found")
    return FileResponse(path, filename=safe)
