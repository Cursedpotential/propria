"""html.unstructured — unstructured partition_html: typed elements joined as text.

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
    id="html.unstructured",
    capability="extract.html_text",
    description="unstructured partition_html: Title / NarrativeText / ListItem / Table elements, joined one per line.",
    accept=accepts_html,
    provenance="unstructured.partition.html",
    tool_version=TOOL_VERSION["unstructured"],
    formats=HTML_FORMATS,
    quality=QUALITY["unstructured"],
)
def extract_html_unstructured(payload: dict[str, Any]) -> dict[str, Any]:
    from unstructured.partition.html import partition_html

    def extract(path, html):
        return "\n".join(element.text for element in partition_html(filename=str(path)))

    return run_text_tool(payload, "unstructured", "unstructured", extract)
