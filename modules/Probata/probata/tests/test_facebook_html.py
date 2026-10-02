# Byline amendment: Codex · GPT-5 · 2026-08-18 (combined-change hygiene)
"""Unit tests for server.tools.parsers.messaging.facebook_messenger_html.

Covers both DOM layouts Facebook emits (legacy div.message, card _a6-g), fuzzy
timestamp parsing, owner-based direction tagging, and format rejection.
"""

from __future__ import annotations

from typing import Any

import pytest

from server.tools.parsers.messaging.facebook_messenger_html import parse


def _run(tmp_path, html, *, owner=None, name="thread.html"):
    p = tmp_path / name
    p.write_text(html, encoding="utf-8")
    payload: dict[str, Any] = {"path": str(p)}
    if owner:
        payload["source_meta"] = {"owner_name": owner}
    return parse(payload)


def test_legacy_layout_with_direction(tmp_path):
    html = """<html><body>
      <div class="message"><div class="meta">Ex Partner - May 17, 2022, 5:29 PM</div><p>Why are you late again</p></div>
      <div class="message"><div class="meta">Me - May 17, 2022, 5:30 PM</div><p>Traffic on the bridge</p></div>
    </body></html>"""
    result = _run(tmp_path, html, owner="Me")

    assert result["stats"]["messages"] == 2
    assert result["stats"]["layout"] == "legacy"
    assert result["stats"]["participants"] == 2

    ex, me = result["records"]  # chronological
    assert ex["role"] == "Ex Partner"
    assert ex["content"] == "Why are you late again"
    assert ex["attrs"]["direction"] == "inbound"
    assert ex["occurred_at"].startswith("2022-05-17")
    assert me["role"] == "Me" and me["attrs"]["direction"] == "outbound"


def test_card_layout(tmp_path):
    html = """<html><body>
      <div class="_a6-g"><div class="_a6-h">Ex</div><div class="_a6-p">hello there</div></div>
      <div class="_a6-o"><div class="_a72d">May 17, 2022, 5:29 PM</div></div>
    </body></html>"""
    result = _run(tmp_path, html)
    assert result["stats"]["layout"] == "card"
    assert result["records"][0]["content"] == "hello there"
    assert result["records"][0]["role"] == "Ex"


def test_rejects_non_facebook_html(tmp_path):
    with pytest.raises(ValueError, match="not a Facebook HTML export"):
        _run(tmp_path, "<html><body><p>just a webpage</p></body></html>")


def test_card_layout_2024_timestamp_inside_card(tmp_path):
    # Current 2024 export: sender/body/timestamp are all inside the card; seconds present, "pm" glued to the time.
    html = """<html><body><div class="_a6-g"><div class="_2ph_ _a6-h _a6-i">Matt Salem</div>
      <div class="_2ph_ _a6-p"><div><div></div><div>Hi, is this available?</div></div></div>
      <div class="_3-94 _a6-o"><div class="_a72d">Jul 12, 2024 4:54:10pm</div></div></div></body></html>"""
    result = _run(tmp_path, html)
    assert result["stats"]["layout"] == "card"
    record = result["records"][0]
    assert record["content"] == "Hi, is this available?"
    assert record["role"] == "Matt Salem"
    assert record["occurred_at"].startswith("2024-07-12T16:54:10")


def test_card_layout_2025_section_h2_footer_and_reactions(tmp_path):
    # Current 2025 export: section card, h2 sender, footer timestamp, reactions in ul._a6-q kept out of the body.
    html = """<html><body><main><section class="_3-95 _a6-g"><h2 class="_2ph_ _a6-h _a6-i">Aleksandrs Petrovs</h2>
      <div class="_2ph_ _a6-p"><div><div></div><div>\U0001F602\U0001F602 same here</div>
      <div><ul class="_a6-q"><li><span>\U0001F606Jeffery Cooper (Jun 01, 2025 3:32:01 pm)</span></li></ul></div></div></div>
      <footer class="_3-94 _a6-o"><div class="_a72d">Jun 01, 2025 3:31:29 pm</div></footer></section></main></body></html>"""
    result = _run(tmp_path, html)
    record = result["records"][0]
    assert record["content"] == "\U0001F602\U0001F602 same here"  # emoji intact, reaction not merged into the body
    assert record["role"] == "Aleksandrs Petrovs"
    assert record["occurred_at"].startswith("2025-06-01T15:31:29")
    assert "Jeffery Cooper" in record["attrs"]["meta"]
