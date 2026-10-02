"""html.beautifulsoup4 — BeautifulSoup4 (lxml tree builder): visible text, one block per line.

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
    id="html.beautifulsoup4",
    capability="extract.html_text",
    description="BeautifulSoup4 over the lxml parser: visible text, script/style/head/noscript removed, one block per line.",
    accept=accepts_html,
    provenance="beautifulsoup4 over lxml",
    tool_version=TOOL_VERSION["beautifulsoup4"],
    formats=HTML_FORMATS,
    quality=QUALITY["beautifulsoup4"],
)
def extract_html_beautifulsoup4(payload: dict[str, Any]) -> dict[str, Any]:
    from bs4 import BeautifulSoup

    def extract(path, html):
        soup = BeautifulSoup(html, "lxml")
        for node in soup(["script", "style", "head", "noscript"]):
            node.decompose()
        return soup.get_text("\n")

    return run_text_tool(payload, "beautifulsoup4", "beautifulsoup4", extract)
