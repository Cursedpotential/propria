"""server/temporal/html_tool_activities.py — one Temporal Activity per HTML text-extraction tool.

Byline: Claude Code · Sonnet · 2026-10-02

The DuckDB webbed templates are the Go engine's primary HTML handlers
(``facebook_messenger_html_v1``, ``generic_html_document_v1``). These Activities
are the selectable alternatives and fallbacks: one library, one job, one
Activity, each running exactly the registry tool of the same name
(``server/tools/extractors/html_text/``) so the Activity, the tool facade and a
direct import all execute the same code.

    extract_html_docling_activity        html.docling
    extract_html_unstructured_activity   html.unstructured
    extract_html_markitdown_activity     html.markitdown
    extract_html_html2text_activity      html.html2text
    extract_html_beautifulsoup4_activity html.beautifulsoup4
    extract_html_lxml_activity           html.lxml
    extract_html_selectolax_activity     html.selectolax

Scheduled on the ``evidence-pipeline`` queue, like ``build_timeline_generation_activity``.
Everything that crosses Temporal history is a reference or a count: the text goes
to ``output_dir`` and only its path, size and timing return. ``path`` must be
readable on this worker (the Activity does not fetch from object storage).

Import rule (as in timeline_activities.py): temporalio + stdlib at module level;
the tool modules, and through them the heavy libraries, are imported inside the body.
"""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any, Callable

from temporalio import activity

HTML_TOOL_IDS = (
    "docling",
    "unstructured",
    "markitdown",
    "html2text",
    "beautifulsoup4",
    "lxml",
    "selectolax",
)


def activity_name(tool: str) -> str:
    return f"extract_html_{tool}_activity"


@dataclass
class HtmlToolParams:
    """Input of every html tool Activity."""

    path: str
    output_dir: str = ""  # when set, the extracted text is written here as <stem>.<tool>.txt


def _run_tool(tool: str, params: HtmlToolParams) -> dict[str, Any]:
    from server.tools.registry import load_builtin_tools, registry

    load_builtin_tools()
    registered = registry.get(f"html.{tool}")
    source = Path(params.path)
    if not registered.accepts(source.name, source.stat().st_size if source.is_file() else 0):
        raise ValueError(f"html.{tool}: not an HTML file name: {source.name}")
    result = registered.run({"path": str(source)})
    stats = dict(result["stats"])
    output_path = ""
    if params.output_dir:
        out_dir = Path(params.output_dir)
        out_dir.mkdir(parents=True, exist_ok=True)
        target = out_dir / f"{source.stem}.{tool}.txt"
        target.write_text(result["text"], encoding="utf-8")
        output_path = str(target)
    activity.logger.info("html.%s %s: %s chars in %ss", tool, source.name, stats["char_count"], stats["elapsed_s"])
    return {
        "tool_id": registered.id,
        "tool_version": registered.tool_version,
        "library_version": stats["library_version"],
        "char_count": stats["char_count"],
        "elapsed_s": stats["elapsed_s"],
        "low_confidence": stats["low_confidence"],
        "output_path": output_path,
    }


def _make_activity(tool: str) -> Callable[[HtmlToolParams], dict[str, Any]]:
    def run(params: HtmlToolParams) -> dict[str, Any]:
        return _run_tool(tool, params)

    run.__name__ = activity_name(tool)
    run.__qualname__ = activity_name(tool)
    return activity.defn(name=activity_name(tool))(run)


extract_html_docling_activity = _make_activity("docling")
extract_html_unstructured_activity = _make_activity("unstructured")
extract_html_markitdown_activity = _make_activity("markitdown")
extract_html_html2text_activity = _make_activity("html2text")
extract_html_beautifulsoup4_activity = _make_activity("beautifulsoup4")
extract_html_lxml_activity = _make_activity("lxml")
extract_html_selectolax_activity = _make_activity("selectolax")

HTML_TOOL_ACTIVITIES = [
    extract_html_docling_activity,
    extract_html_unstructured_activity,
    extract_html_markitdown_activity,
    extract_html_html2text_activity,
    extract_html_beautifulsoup4_activity,
    extract_html_lxml_activity,
    extract_html_selectolax_activity,
]
