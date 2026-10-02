"""Exact tool versions and the per-file-type selector ranks of the html_text tools (not a tool module).

GENERATED from docs/receipts/2026-10-02-html-tool-bench/ranks.json by rank_from_bench.py; the rule is stated
in that receipt's README. `primary` = best fidelity for the file type, `fallback` = within 0.15 of the primary,
`experimental` = the rest. A file type with no primary (imessage_export_html: the messages sit inside a
script string no HTML text tool reads) is handled by a dedicated DuckDB template, not by these tools.

Byline: Claude Code · Sonnet · 2026-10-02
"""

from __future__ import annotations

TOOL_VERSION = {
    "docling": "docling-slim-2.132.0",
    "unstructured": "unstructured-0.27.10",
    "markitdown": "markitdown-0.1.8",
    "html2text": "html2text-2025.4.15",
    "beautifulsoup4": "beautifulsoup4-4.15.0",
    "lxml": "lxml-6.1.1",
    "selectolax": "selectolax-0.4.13",
}

# Best tool per file type at generation time (fidelity in brackets): facebook_export_section_html: html2text (0.843); facebook_messenger_html: html2text (0.966); generic_html_document: markitdown (0.758); google_takeout_activity_html: html2text (1.0); google_voice_html: docling (0.922); imessage_export_html: none (0.001); snapchat_export_html: html2text (1.0); whatsapp_chat_html: html2text (1.0)
QUALITY = {
    "docling": {
        "facebook_export_section_html": "experimental",
        "facebook_messenger_html": "experimental",
        "generic_html_document": "fallback",
        "google_takeout_activity_html": "experimental",
        "google_voice_html": "primary",
        "imessage_export_html": "experimental",
        "snapchat_export_html": "experimental",
        "whatsapp_chat_html": "fallback",
    },
    "unstructured": {
        "facebook_export_section_html": "experimental",
        "facebook_messenger_html": "fallback",
        "generic_html_document": "experimental",
        "google_takeout_activity_html": "fallback",
        "google_voice_html": "fallback",
        "imessage_export_html": "experimental",
        "snapchat_export_html": "experimental",
        "whatsapp_chat_html": "fallback",
    },
    "markitdown": {
        "facebook_export_section_html": "fallback",
        "facebook_messenger_html": "fallback",
        "generic_html_document": "primary",
        "google_takeout_activity_html": "experimental",
        "google_voice_html": "fallback",
        "imessage_export_html": "experimental",
        "snapchat_export_html": "fallback",
        "whatsapp_chat_html": "fallback",
    },
    "html2text": {
        "facebook_export_section_html": "primary",
        "facebook_messenger_html": "primary",
        "generic_html_document": "fallback",
        "google_takeout_activity_html": "primary",
        "google_voice_html": "fallback",
        "imessage_export_html": "experimental",
        "snapchat_export_html": "primary",
        "whatsapp_chat_html": "primary",
    },
    "beautifulsoup4": {
        "facebook_export_section_html": "fallback",
        "facebook_messenger_html": "fallback",
        "generic_html_document": "fallback",
        "google_takeout_activity_html": "fallback",
        "google_voice_html": "fallback",
        "imessage_export_html": "experimental",
        "snapchat_export_html": "experimental",
        "whatsapp_chat_html": "fallback",
    },
    "lxml": {
        "facebook_export_section_html": "fallback",
        "facebook_messenger_html": "fallback",
        "generic_html_document": "fallback",
        "google_takeout_activity_html": "fallback",
        "google_voice_html": "fallback",
        "imessage_export_html": "experimental",
        "snapchat_export_html": "experimental",
        "whatsapp_chat_html": "fallback",
    },
    "selectolax": {
        "facebook_export_section_html": "fallback",
        "facebook_messenger_html": "fallback",
        "generic_html_document": "fallback",
        "google_takeout_activity_html": "fallback",
        "google_voice_html": "fallback",
        "imessage_export_html": "experimental",
        "snapchat_export_html": "experimental",
        "whatsapp_chat_html": "fallback",
    },
}
