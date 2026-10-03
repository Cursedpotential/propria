"""html.unstructured — unstructured partition_html: typed elements joined as text.

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
    id="html.unstructured",
    capability="extract.html_text",
    accept=accepts_html,
    provenance="unstructured.partition.html",
    tool_version=TOOL_VERSION["unstructured"],
    formats=HTML_FORMATS,
    quality=QUALITY["unstructured"],
)
def extract_html_unstructured(payload: dict[str, Any]) -> dict[str, Any]:
    """unstructured partition_html: Title / NarrativeText / ListItem / Table elements, joined one per line.

    Formats: every HTML family in HTML_FORMATS (Facebook Messenger and export sections, Google Takeout activity and Voice, iMessage, Snapchat, WhatsApp, generic documents); accepts a .html, .htm or .xhtml file name.
    Side effects: none; reads the file named by payload['path'] and returns {text, pages, stats}.
    Pick it when element types matter more than layout; it is a fallback for Facebook Messenger, Google Takeout, Google Voice and
    WhatsApp pages and never the primary of a family. Slower and heavier than the other six.
    """
    from unstructured.partition.html import partition_html

    def extract(path, html):
        return "\n".join(element.text for element in partition_html(filename=str(path)))

    return run_text_tool(payload, "unstructured", "unstructured", extract)
