"""Shared plumbing for the html_text tools (underscore prefix: not a tool module).

Every tool returns the same shape as the other ``extract.*`` tools:
``{"text", "pages", "stats"}`` where ``stats`` names the library and its exact
version, so a result can always be tied to the tool that produced it.

Byline: Claude Code · Sonnet · 2026-10-02
"""

from __future__ import annotations

import time
from importlib import metadata
from pathlib import Path
from typing import Any, Callable

HTML_SUFFIXES = (".html", ".htm", ".xhtml")

# HTML file families held in the Case Bible catalog (raw_duck, 2026-10-02) and scored per family in
# docs/receipts/2026-10-02-html-tool-bench/. The Go engine detects facebook_messenger_html and generic_html_document today
# (context.handler_detected_format); the others are declared so the registry can already route each
# family to its own best tool once the engine detects it. Ids are sorted, as the registry requires.
FORMAT_FACEBOOK_MESSENGER_HTML = "facebook_messenger_html"
FORMAT_GENERIC_HTML = "generic_html_document"
HTML_FORMATS = (
    "facebook_export_section_html",  # logins, search history, friends, ads ... (Facebook cards and tables)
    "facebook_messenger_html",  # message_N.html thread file
    "generic_html_document",  # saved web pages, software documentation, anything else
    "google_takeout_activity_html",  # Takeout My Activity (very large single page)
    "google_voice_html",  # Takeout Voice calls / texts / voicemail (XHTML)
    "imessage_export_html",  # iMessage export page (messages embedded in a script string)
    "snapchat_export_html",  # Snapchat chat_history / snap_history pages
    "whatsapp_chat_html",  # WhatsApp _chat.html
)


def accepts_html(hint: str, size: int) -> bool:
    return hint.lower().endswith(HTML_SUFFIXES)


def distribution_version(distribution: str) -> str:
    """Exact installed version of the pip distribution, or ``unavailable``."""
    try:
        return metadata.version(distribution)
    except metadata.PackageNotFoundError:
        return "unavailable"


def read_html(payload: dict[str, Any]) -> tuple[Path, str]:
    """The file as UTF-8 text. Facebook HTML is real UTF-8 (no mojibake); undecodable bytes are replaced, never dropped silently into a different alphabet."""
    path = Path(payload["path"])
    if not path.is_file():
        raise FileNotFoundError(path)
    return path, path.read_bytes().decode("utf-8", errors="replace")


def run_text_tool(
    payload: dict[str, Any], library: str, distribution: str, extract: Callable[[Path, str], str]
) -> dict[str, Any]:
    path, html = read_html(payload)
    started = time.perf_counter()
    text = extract(path, html)
    elapsed = time.perf_counter() - started
    return {
        "text": text,
        "pages": [text],
        "stats": {
            "method": library,
            "library_version": distribution_version(distribution),
            "char_count": len(text),
            "elapsed_s": round(elapsed, 3),
            "low_confidence": not text.strip(),
            "structured": False,
        },
    }
