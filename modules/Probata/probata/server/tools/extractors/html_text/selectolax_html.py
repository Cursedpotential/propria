"""html.selectolax — selectolax (lexbor): fast HTML5 parser, visible text.

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
    id="html.selectolax",
    capability="extract.html_text",
    description="selectolax (lexbor): visible text, script/style/head/noscript stripped; the fastest parser measured.",
    accept=accepts_html,
    provenance="selectolax.lexbor",
    tool_version=TOOL_VERSION["selectolax"],
    formats=HTML_FORMATS,
    quality=QUALITY["selectolax"],
)
def extract_html_selectolax(payload: dict[str, Any]) -> dict[str, Any]:
    from selectolax.lexbor import LexborHTMLParser

    def extract(path, html):
        tree = LexborHTMLParser(html)
        tree.strip_tags(["script", "style", "head", "noscript"])
        return tree.body.text(separator="\n") if tree.body else tree.text(separator="\n")

    return run_text_tool(payload, "selectolax", "selectolax", extract)
