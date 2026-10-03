"""html.beautifulsoup4 — BeautifulSoup4 (lxml tree builder): visible text, one block per line.

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
    id="html.beautifulsoup4",
    capability="extract.html_text",
    accept=accepts_html,
    provenance="beautifulsoup4 over lxml",
    tool_version=TOOL_VERSION["beautifulsoup4"],
    formats=HTML_FORMATS,
    quality=QUALITY["beautifulsoup4"],
)
def extract_html_beautifulsoup4(payload: dict[str, Any]) -> dict[str, Any]:
    """BeautifulSoup4 over the lxml parser: visible text, script/style/head/noscript removed, one block per line.

    Formats: every HTML family in HTML_FORMATS (Facebook Messenger and export sections, Google Takeout activity and Voice, iMessage, Snapchat, WhatsApp, generic documents); accepts a .html, .htm or .xhtml file name.
    Side effects: none; reads the file named by payload['path'] and returns {text, pages, stats}.
    Pick it as a plain-text fallback when the Markdown tools add noise; it is a fallback for most families and never the primary.
    """
    from bs4 import BeautifulSoup

    def extract(path, html):
        soup = BeautifulSoup(html, "lxml")
        for node in soup(["script", "style", "head", "noscript"]):
            node.decompose()
        return soup.get_text("\n")

    return run_text_tool(payload, "beautifulsoup4", "beautifulsoup4", extract)
