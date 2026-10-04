"""Render PDF pages to image files, one page at a time. One job: rasterize.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

Used by the image stage for PDFs that have no text layer (the extract stage marks them
``skipped_no_text``): each
page becomes a PNG that goes through the same facts, OCR and embedding units as any screenshot.
pypdfium2 (the
PDFium engine, already the bake-off's fast reader) renders; Pillow encodes. Memory is one page
bitmap at a time: a
300-page scan is rendered and written page by page, never held. Pages beyond ``max_pages`` are
counted, not rendered,
so a receipt can say a document was only partly covered.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path

import pypdfium2 as pdfium

DEFAULT_DPI = 150
DEFAULT_MAX_PAGES = 20
# a page bitmap above this is rendered at a lower scale instead of risking the memory cap
MAX_PIXELS = 40_000_000


@dataclass(frozen=True)
class RenderedPage:
    """One rendered page: its 1-based number and the PNG file written for it."""

    page: int
    path: Path
    width: int
    height: int


@dataclass(frozen=True)
class RenderResult:
    pages: list[RenderedPage]
    page_count: int  # pages in the document, rendered or not
    deferred: list[RenderedPage] = field(default_factory=list)

    @property
    def truncated(self) -> int:
        return self.page_count - len(self.pages)


def render_pdf_pages(
    pdf_path: Path,
    out_dir: Path,
    *,
    max_pages: int = DEFAULT_MAX_PAGES,
    dpi: int = DEFAULT_DPI,
    stem: str = "page",
    start_page: int = 1,
    max_output_bytes: int | None = None,
) -> RenderResult:
    """Render up to ``max_pages`` pages of ``pdf_path`` to ``out_dir/<stem>-<n>.png`` at ``dpi``.

    Inputs: PDF file, output folder, page cap, resolution, start page and optional output byte cap.
    Output: accepted pages, at most one deferred overflow page and the
    document's page count. Side effects: writes the PNG files. A file PDFium cannot open raises
    ``ValueError`` (the
    caller records the document as unreadable); the source PDF is only read. Pick this over a tool
    call when the page
    images feed the image stage directly; pick the PDF text tools when the PDF has a text layer."""
    try:
        document = pdfium.PdfDocument(str(pdf_path))
    except pdfium.PdfiumError as exc:
        raise ValueError(f"not a readable PDF: {pdf_path.name}") from exc
    try:
        count = len(document)
        pages: list[RenderedPage] = []
        deferred: list[RenderedPage] = []
        output_bytes = 0
        if start_page < 1 or max_pages < 1:
            raise ValueError("PDF start_page and max_pages must be positive")
        if max_output_bytes is not None and max_output_bytes < 0:
            raise ValueError("PDF output byte cap cannot be negative")
        for index in range(start_page - 1, min(count, start_page - 1 + max_pages)):
            page = document[index]
            try:
                scale = dpi / 72.0
                width, height = page.get_size()
                if width * height * scale * scale > MAX_PIXELS:
                    scale = (MAX_PIXELS / (width * height)) ** 0.5
                bitmap = page.render(scale=scale)
                image = bitmap.to_pil().convert("RGB")
                target = out_dir / f"{stem}-{index + 1:04d}.png"
                image.save(target, format="PNG")
                rendered = RenderedPage(index + 1, target, image.width, image.height)
                size = target.stat().st_size
                image.close()
                if max_output_bytes is not None and output_bytes + size > max_output_bytes:
                    deferred.append(rendered)
                    break  # Caller retains this one bounded overflow page; it is never published.
                pages.append(rendered)
                output_bytes += size
            finally:
                page.close()
        return RenderResult(pages, count, deferred)
    finally:
        document.close()
