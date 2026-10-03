"""html.html2text — html2text: HTML to unwrapped Markdown.

One library, one job: HTML file in, text out. Selectable by id; the default per
file type is the ``primary`` rank in ``_ranks.QUALITY`` (set from the bench in
``docs/receipts/2026-10-02-html-tool-bench/``). The library is imported inside
the call so registry discovery stays safe in an image without it.

Byline: Claude Code · Sonnet · 2026-10-02; docstring-described by Claude Code · Sonnet 5.5 · 2026-10-02
"""

from __future__ import annotations

from typing import Any

from server.tools.registry import register

from ._common import HTML_FORMATS, accepts_html, run_text_tool
from ._ranks import QUALITY, TOOL_VERSION


@register(
    id="html.html2text",
    capability="extract.html_text",
    accept=accepts_html,
    provenance="html2text",
    tool_version=TOOL_VERSION["html2text"],
    formats=HTML_FORMATS,
    quality=QUALITY["html2text"],
)
def extract_html_html2text(payload: dict[str, Any]) -> dict[str, Any]:
    """html2text: HTML to Markdown with no line wrapping; links and images kept as Markdown.

    Formats: every HTML family in HTML_FORMATS (Facebook Messenger and export sections, Google Takeout activity and Voice, iMessage, Snapchat, WhatsApp, generic documents); accepts a .html, .htm or .xhtml file name.
    Side effects: none; reads the file named by payload['path'] and returns {text, pages, stats}.
    Pick it first for message exports: it is the primary tool of the bench for Facebook Messenger and export sections, Google Takeout
    activity, Snapchat and WhatsApp pages; a fallback for generic pages and Google Voice.
    """
    import html2text

    def extract(path, html):
        converter = html2text.HTML2Text()
        converter.body_width = 0
        converter.ignore_images = False
        return converter.handle(html)

    return run_text_tool(payload, "html2text", "html2text", extract)
