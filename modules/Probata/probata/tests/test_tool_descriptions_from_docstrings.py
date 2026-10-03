# Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""A tool's catalog description is its implementing function's docstring (owner 2026-10-02).

"these tools description is written in doc strings or whatever and then it can be automatically extracted from that
when it does the category update and the documentation and everything." The registry reads the description from the
docstring, GET /tools and the ``atomic_tools`` MCP entry serve it, and these tests keep it that way:

* a registered tool with no docstring (and no legacy description) cannot register;
* a docstring and an explicit ``description=`` together are refused, so a tool has ONE source;
* every tool outside ``LEGACY_EXPLICIT_DESCRIPTION`` is described by its docstring, and the docstring names its formats,
  its side effects and when to pick it;
* ``LEGACY_EXPLICIT_DESCRIPTION`` is the follow-up list: the tools registered before this rule that still carry a typed
  description. It can only shrink: a tool that gains a docstring must leave the list, and a new tool cannot join it.
"""

from __future__ import annotations

import json

import pytest

from server.tools.registry import docstring_description, load_builtin_tools, register, registry

# 41 tools registered before the docstring rule, still described by a typed ``description=``. Converting one means giving
# its function a docstring (first paragraph = the description; then formats, side effects, when to pick) and deleting
# the ``description=`` argument, then deleting its id here.
LEGACY_EXPLICIT_DESCRIPTION = frozenset(
    {
        "documents.extract-docling",
        "documents.extract-text",
        "ingest.context-drain",
        "transcripts.chatgpt-custom-gpt-md",
        "transcripts.chatgpt-official",
        "transcripts.chatgpt-share",
        "transcripts.claude-ai-export",
        "transcripts.claude-code",
        "transcripts.claude-code-jsonl",
        "transcripts.claude-md",
        "transcripts.gemini-chrome",
        "transcripts.gemini-json",
        "transcripts.gemini-md",
        "transcripts.perplexity-contexts",
        "transcripts.perplexity-gdpr",
        "transcripts.perplexity-md",
        "transcripts.perplexity-plugin",
        "transcripts.generic-md",
        "transcripts.markdown",
        "messages.facebook-html",
        "messages.facebook-json",
        "messages.imessage-html",
        "messages.imessage-txt",
        "messages.imessage-pdf",
        "messages.messaging-csv",
        "messages.transcript-marker",
        "messages.sms-xml-sbv",
        "messages.sms-xml",
        "messages.snapchat-json",
        "messages.whatsapp-txt",
        "repair.capabilities",
        "repair.detect",
        "repair.preview",
        "repair.write-derived",
        "repair.pdf-inspect",
        "repair.pdf-derived",
        "repair.flag-damaged",
        "repair.quarantine-plan",
        "repair.quarantine-copy",
        "repair.audit-verify",
        "geo_map.leaflet",
    }
)

# Registered with the docstring rule on 2026-10-02.
DOCSTRING_TOOLS = {
    "engine.poppler-inspect": "engine.inspect",
    "engine.poppler-certify-text": "engine.certify",
    "html.docling": "extract.html_text",
    "html.unstructured": "extract.html_text",
    "html.markitdown": "extract.html_text",
    "html.html2text": "extract.html_text",
    "html.beautifulsoup4": "extract.html_text",
    "html.lxml": "extract.html_text",
    "html.selectolax": "extract.html_text",
    "chunking.chonkie-token": "chunk.message_spans",
    "chunking.chonkie-fast": "chunk.message_spans",
    "chunking.chonkie-sentence": "chunk.message_spans",
    "chunking.chonkie-recursive": "chunk.message_spans",
    "repair.json-repair": "repair.json",
}


def _by_id():
    load_builtin_tools()
    return {tool.id: tool for tool in registry.all()}


def test_description_is_the_first_paragraph_of_the_docstring_joined_to_one_line():
    def tool(payload):
        """Do one thing to a file,
        wrapped over two lines.

        Formats: none.
        """

    assert docstring_description(tool) == "Do one thing to a file, wrapped over two lines."


def test_a_tool_without_a_docstring_or_description_cannot_register():
    with pytest.raises(ValueError, match="no description"):

        @register(id="test.no-docs", capability="test.nothing")
        def undocumented(payload):
            return {}

    assert "test.no-docs" not in _by_id()


def test_a_docstring_and_an_explicit_description_are_two_sources_and_are_refused():
    with pytest.raises(ValueError, match="docstring is the only source"):

        @register(id="test.two-sources", capability="test.nothing", description="typed text")
        def documented(payload):
            """Docstring text."""
            return {}

    assert "test.two-sources" not in _by_id()


def test_a_registered_docstring_becomes_the_catalog_description():
    @register(id="test.from-doc", capability="test.nothing", provenance="test")
    def documented(payload):
        """Echo the payload back.

        Formats: any.
        """
        return payload

    try:
        entry = {row["id"]: row for row in registry.contract_manifest()}["test.from-doc"]
        assert entry["description"] == "Echo the payload back."
    finally:
        registry._tools.pop("test.from-doc")


def test_every_tool_has_a_description_and_the_new_ones_are_registered():
    tools = _by_id()
    for tool in tools.values():
        assert tool.description.strip(), tool.id
    for tool_id, capability in DOCSTRING_TOOLS.items():
        assert tool_id in tools, f"{tool_id} is not registered"
        assert tools[tool_id].capability == capability


def test_tools_outside_the_legacy_list_are_described_by_their_docstring_alone():
    for tool in _by_id().values():
        if tool.id in LEGACY_EXPLICIT_DESCRIPTION:
            continue
        documented = docstring_description(tool.fn)
        assert documented, f"{tool.id}: no docstring; the catalog description comes from it"
        assert tool.description == documented, tool.id
        assert len(documented) <= 300, f"{tool.id}: the first paragraph is the one-sentence description"


def test_docstrings_say_formats_side_effects_and_when_to_pick():
    tools = _by_id()
    for tool_id in DOCSTRING_TOOLS:
        doc = tools[tool_id].fn.__doc__ or ""
        assert "Formats:" in doc, f"{tool_id}: name the formats it handles"
        assert "Side effects:" in doc, f"{tool_id}: say what it changes (nothing, or what)"
        assert "Pick " in doc, f"{tool_id}: say when to pick it over its siblings"


def test_the_legacy_list_only_shrinks():
    tools = _by_id()
    for tool_id in LEGACY_EXPLICIT_DESCRIPTION:
        assert tool_id in tools, f"{tool_id} is listed as legacy but is not registered; remove it from the list"
        assert not docstring_description(tools[tool_id].fn), (
            f"{tool_id} now has a docstring: delete its description= argument and remove it from the legacy list"
        )
    undocumented = {tool_id for tool_id, tool in tools.items() if not docstring_description(tool.fn)}
    assert undocumented == set(LEGACY_EXPLICIT_DESCRIPTION), sorted(undocumented ^ set(LEGACY_EXPLICIT_DESCRIPTION))


# --- the new tools run for real -------------------------------------------------------------------------------------


def test_chonkie_chunker_tools_cut_a_thread_like_the_chunk_pipeline():
    pytest.importorskip("chonkie")
    from server.context_chunks.chunker import chunk_spans

    lines = [f"m{index} says hello number {index} " + "x" * 60 for index in range(30)]
    tools = _by_id()
    for key, engine in (
        ("token", "token_1000"),
        ("fast", "fast_1000"),
        ("sentence", "sentence_1000"),
        ("recursive", "recursive_1000"),
    ):
        result = tools[f"chunking.chonkie-{key}"].run({"lines": lines})
        assert result["spans"] == [list(span) for span in chunk_spans(lines, engine)], key
        assert result["spans"][0][0] == 0 and result["spans"][-1][1] == len(lines) - 1
        assert result["stats"]["message_count"] == 30
        assert result["stats"]["chunk_count"] == len(result["spans"])


def test_chonkie_chunker_tool_reads_a_file_and_rejects_a_payload_without_input(tmp_path):
    pytest.importorskip("chonkie")
    thread = tmp_path / "thread.txt"
    thread.write_text("\n".join(f"line {index} " + "y" * 80 for index in range(20)), encoding="utf-8")
    tool = _by_id()["chunking.chonkie-fast"]
    assert tool.run({"path": str(thread), "overlap": 0})["stats"]["message_count"] == 20
    with pytest.raises(ValueError, match="lines"):
        tool.run({})
    with pytest.raises(FileNotFoundError):
        tool.run({"path": str(tmp_path / "missing.txt")})


def test_json_repair_tool_repairs_a_truncated_document_and_leaves_valid_json_alone(tmp_path):
    pytest.importorskip("json_repair")
    tool = _by_id()["repair.json-repair"]

    broken = tmp_path / "broken.json"
    broken.write_text('{"messages": [{"id": 1, "body": "hi"}, {"id": 2, "body": "cut off', encoding="utf-8")
    result = tool.run({"path": str(broken)})
    assert result["was_valid"] is False and result["repair_count"] >= 1
    repaired = json.loads(result["repaired_json"])
    assert repaired["messages"][0] == {"id": 1, "body": "hi"}
    assert repaired["messages"][1]["id"] == 2
    assert broken.read_text(encoding="utf-8").endswith("cut off")  # the source is never rewritten

    valid = tmp_path / "valid.json"
    valid.write_text('{"a": 1}', encoding="utf-8")
    unchanged = tool.run({"path": str(valid)})
    assert (
        unchanged["was_valid"] is True and unchanged["repaired_json"] == '{"a": 1}' and unchanged["repair_count"] == 0
    )

    with pytest.raises(FileNotFoundError):
        tool.run({"path": str(tmp_path / "missing.json")})
