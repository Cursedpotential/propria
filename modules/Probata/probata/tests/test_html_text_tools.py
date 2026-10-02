# Byline: Claude Code · Sonnet · 2026-10-02
"""The html_text registry tools and their Temporal Activities.

Registry/declaration checks need no HTML library. The extraction checks run the
real library on a small HTML file and skip when that library is not installed
in this interpreter (the temporal-worker image installs all of them; the pins are
in requirements-html-tools.txt).
"""

from __future__ import annotations

import importlib.util
import re

import pytest

from server.temporal import html_tool_activities as activities
from server.tools.extractors.html_text._common import HTML_FORMATS
from server.tools.registry import load_builtin_tools, registry

PAGE = (
    "<!DOCTYPE html><html><head><title>t</title><style>.x{color:red}</style><script>var hidden=1</script></head>"
    "<body><h1>Thread</h1><div><p>first \U0001f602\U0001f44d\U0001f3fd message with café</p>"
    '<p>second <a href="photos/a.jpg">photo</a></p></div></body></html>'
)

LIBRARY_OF = {
    "docling": "docling",
    "unstructured": "unstructured",
    "markitdown": "markitdown",
    "html2text": "html2text",
    "beautifulsoup4": "bs4",
    "lxml": "lxml",
    "selectolax": "selectolax",
}


def test_every_html_tool_is_registered_with_formats_and_a_primary_somewhere():
    load_builtin_tools()
    tools = {t.id: t for t in registry.all() if t.capability == "extract.html_text"}
    assert set(tools) == {f"html.{name}" for name in activities.HTML_TOOL_IDS}
    primaries = {fmt: [] for fmt in HTML_FORMATS}
    for tool in tools.values():
        assert tool.formats == HTML_FORMATS
        assert re.search(r"-\d+(\.\d+)+$", tool.tool_version), tool.tool_version  # exact pin, never unversioned
        assert tool.accepts("thread.html", 10) and tool.accepts("PAGE.HTM", 10) and not tool.accepts("a.pdf", 10)
        for fmt, rank in tool.quality:
            if rank == "primary":
                primaries[fmt].append(tool.id)
    for fmt, ids in primaries.items():
        if fmt == "imessage_export_html":
            # The messages sit inside a script string; no HTML text tool reads them, a DuckDB template does.
            assert ids == [], f"{fmt} has no HTML text tool that works, but {ids} are marked primary"
            continue
        assert len(ids) == 1, f"{fmt} must have exactly one primary tool, has {ids}"


def test_one_activity_per_tool_with_its_own_name():
    names = [a.__temporal_activity_definition.name for a in activities.HTML_TOOL_ACTIVITIES]
    assert names == [f"extract_html_{tool}_activity" for tool in activities.HTML_TOOL_IDS]
    assert len(set(names)) == len(names) == 7


@pytest.mark.parametrize("tool", activities.HTML_TOOL_IDS)
def test_tool_extracts_text_keeps_emoji_and_drops_script_style(tool, tmp_path):
    if importlib.util.find_spec(LIBRARY_OF[tool]) is None:
        pytest.skip(f"{LIBRARY_OF[tool]} not installed in this interpreter")
    page = tmp_path / "thread.html"
    page.write_text(PAGE, encoding="utf-8")
    load_builtin_tools()
    result = registry.get(f"html.{tool}").run({"path": str(page)})
    text = result["text"]
    assert "first \U0001f602\U0001f44d\U0001f3fd message with café" in text.replace("\n", " ") or (
        "\U0001f602\U0001f44d\U0001f3fd" in text and "café" in text
    )
    assert "hidden" not in text and "color:red" not in text
    assert result["stats"]["method"] == tool
    assert result["stats"]["library_version"] != "unavailable"


@pytest.mark.parametrize("tool", ["beautifulsoup4", "lxml", "html2text"])
def test_activity_body_writes_output_and_returns_references_only(tool, tmp_path):
    if importlib.util.find_spec(LIBRARY_OF[tool]) is None:
        pytest.skip(f"{LIBRARY_OF[tool]} not installed in this interpreter")
    page = tmp_path / "thread.html"
    page.write_text(PAGE, encoding="utf-8")
    out = tmp_path / "out"
    result = activities._run_tool(tool, activities.HtmlToolParams(path=str(page), output_dir=str(out)))
    assert result["tool_id"] == f"html.{tool}"
    assert result["output_path"] == str(out / f"thread.{tool}.txt")
    assert (out / f"thread.{tool}.txt").read_text(encoding="utf-8").strip()
    assert set(result) == {
        "tool_id",
        "tool_version",
        "library_version",
        "char_count",
        "elapsed_s",
        "low_confidence",
        "output_path",
    }
