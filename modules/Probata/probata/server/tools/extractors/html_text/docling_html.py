"""html.docling — Docling DocumentConverter, HTML backend: structured document exported as Markdown.

One library, one job: HTML file in, text out. Selectable by id; the default per
file type is the ``primary`` rank in ``_ranks.QUALITY`` (set from the bench in
``docs/receipts/2026-10-02-html-tool-bench/``). The library is imported inside
the call so registry discovery stays safe in an image without it. The HTML
backend needs no OCR or layout model; it is the same Docling that
``documents.extract-docling`` runs for PDF and Office files.

Byline: Claude Code · Sonnet · 2026-10-02
"""

from __future__ import annotations

from typing import Any

from server.tools.registry import register

from ._common import HTML_FORMATS, accepts_html, run_text_tool
from ._ranks import QUALITY, TOOL_VERSION


@register(
    id="html.docling",
    capability="extract.html_text",
    description="Docling HTML backend: structured document (headings, lists, tables) exported as Markdown.",
    accept=accepts_html,
    provenance="docling (IBM) HTML backend",
    tool_version=TOOL_VERSION["docling"],
    formats=HTML_FORMATS,
    quality=QUALITY["docling"],
)
def extract_html_docling(payload: dict[str, Any]) -> dict[str, Any]:
    from docling.document_converter import DocumentConverter

    def extract(path, html):
        return DocumentConverter().convert(str(path)).document.export_to_markdown()

    return run_text_tool(payload, "docling", "docling", extract)
