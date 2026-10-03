"""html.lxml — lxml.html: visible text in document order.

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
    id="html.lxml",
    capability="extract.html_text",
    accept=accepts_html,
    provenance="lxml.html",
    tool_version=TOOL_VERSION["lxml"],
    formats=HTML_FORMATS,
    quality=QUALITY["lxml"],
)
def extract_html_lxml(payload: dict[str, Any]) -> dict[str, Any]:
    """lxml.html (libxml2): visible text in document order, script/style/head/noscript dropped.

    Formats: every HTML family in HTML_FORMATS (Facebook Messenger and export sections, Google Takeout activity and Voice, iMessage, Snapchat, WhatsApp, generic documents); accepts a .html, .htm or .xhtml file name.
    Side effects: none; reads the file named by payload['path'] and returns {text, pages, stats}.
    Pick it as a fast plain-text fallback; it declares the encoding as UTF-8 so emoji do not turn into mojibake. Never the primary
    of a family.
    """
    import lxml.html

    def extract(path, html):
        # Bytes WITHOUT an explicit UTF-8 parser make lxml guess Latin-1 whenever the page has no <meta charset>,
        # which turns every emoji into mojibake. The encoding is stated, never guessed.
        document = lxml.html.document_fromstring(html.encode("utf-8"), parser=lxml.html.HTMLParser(encoding="utf-8"))
        for node in document.xpath("//script|//style|//head|//noscript"):
            node.drop_tree()
        return "\n".join(document.itertext())

    return run_text_tool(payload, "lxml", "lxml", extract)
