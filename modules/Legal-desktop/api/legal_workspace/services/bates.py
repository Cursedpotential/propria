"""Bates stamp overlay on owner-produced PDFs. Not Agno evidence.

> _Byline: Grok · grok-4.6 · 2026-08-18_
pypdf page overlay of `{prefix}-{n:06d}`. No evidence bytes. Not court-safe.
"""

from __future__ import annotations

from pathlib import Path

from pydantic import BaseModel

from legal_workspace.domain.exhibits import bates_prefix, next_bates_number


class BatesStampResult(BaseModel):
    ok: bool
    bates_number: str
    output_path: str
    page_count: int
    court_safe: bool = False
    exportable: bool = False


def _footer_overlay(path: Path, text: str, width: float, height: float) -> Path:
    from pikepdf import Dictionary, Name, Pdf

    escaped = text.replace("\\", "\\\\").replace("(", "\\(").replace(")", "\\)")
    pdf = Pdf.new()
    font = pdf.make_indirect(
        Dictionary(Type=Name.Font, Subtype=Name.Type1, BaseFont=Name.Helvetica)
    )
    stream = pdf.make_stream(f"BT /F1 9 Tf 36 18 Td ({escaped}) Tj ET".encode("latin-1"))
    page = pdf.add_blank_page(page_size=(width, height))
    page.Contents = stream
    page.Resources = Dictionary(Font=Dictionary(F1=font))
    path.parent.mkdir(parents=True, exist_ok=True)
    pdf.save(path)
    return path


def stamp_bates_pdf(src: Path, dest: Path, bates_number: str) -> BatesStampResult:
    """Merge a footer overlay containing `bates_number` onto every page."""
    from pypdf import PdfReader, PdfWriter

    if not bates_number.strip():
        raise ValueError("bates_number is required")
    dest.parent.mkdir(parents=True, exist_ok=True)
    writer = PdfWriter(clone_from=str(src))
    overlay_path = dest.with_name(dest.stem + ".overlay.pdf")
    first = writer.pages[0]
    _footer_overlay(
        overlay_path,
        bates_number,
        float(first.mediabox.width),
        float(first.mediabox.height),
    )
    stamp_page = PdfReader(str(overlay_path)).pages[0]
    for page in writer.pages:
        page.merge_page(stamp_page, over=True)
    with dest.open("wb") as handle:
        writer.write(handle)
    return BatesStampResult(
        ok=True,
        bates_number=bates_number,
        output_path=str(dest),
        page_count=len(writer.pages),
        court_safe=False,
        exportable=False,
    )


def next_stamp_for_matter(display_name: str, existing: list[str]) -> str:
    return next_bates_number(bates_prefix(display_name), existing)
