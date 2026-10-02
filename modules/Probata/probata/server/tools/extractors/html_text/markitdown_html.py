"""html.markitdown — Microsoft MarkItDown: HTML to Markdown.

One library, one job: HTML file in, text out. Selectable by id; the default per
file type is the ``primary`` rank in ``_ranks.QUALITY`` (set from the bench in
``docs/receipts/2026-10-02-html-tool-bench/``). The library is imported inside
the call so registry discovery stays safe in an image without it.

Byline: Claude Code · Sonnet · 2026-10-02
"""

from __future__ import annotations

from typing import Any

from server.tools.registry import register

from ._common import HTML_FORMATS, accepts_html, run_text_tool
from ._ranks import QUALITY, TOOL_VERSION


@register(
    id="html.markitdown",
    capability="extract.html_text",
    description="MarkItDown (Microsoft): HTML to Markdown.",
    accept=accepts_html,
    provenance="markitdown",
    tool_version=TOOL_VERSION["markitdown"],
    formats=HTML_FORMATS,
    quality=QUALITY["markitdown"],
)
def extract_html_markitdown(payload: dict[str, Any]) -> dict[str, Any]:
    from markitdown import MarkItDown

    def extract(path, html):
        return MarkItDown().convert(str(path)).text_content

    return run_text_tool(payload, "markitdown", "markitdown", extract)
