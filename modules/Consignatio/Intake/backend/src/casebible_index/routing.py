"""Where each catalog object goes. Pure functions, no I/O: one unit, one job.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Every object in the catalog is REPRESENTED in the Super Index (the Case Bible holds everything;
being in it does not
mean a file is relevant). What differs is how far it goes:

    document        a supported text/PDF/office/web format: extracted, chunked, embedded, published
    message_export  an SMS/call backup (smsbackuprestore XML) and other message exports:
    conversation chunks
                    (INTAKE_CONVERSATION_MODE=chunk, the default) or a document row only (route)
    ai_chat_export  an AI chat export (ChatGPT/Claude ``conversations.json``): NEVER chunked here.
    The AI-chat
                    workstream owns it through Proffer (topic chunking). The Super Index keeps a
                    document row with
                    the locator and status ``routed_ai_chat``.
    archive         ZIP: members are listed over ranged reads and each member is routed again
    media           image/audio/video: represented in the inventory with its locator, no text
    extraction here
    unsupported     any other extension: represented in the inventory

``classify_key`` looks at the key only (used by discovery, which reads no bytes). ``sniff_*`` look
at the first bytes
of an object and are used by the extract stage, which has the stream open anyway.
"""

from __future__ import annotations

from pathlib import PurePosixPath

from .config import SUPPORTED_EXTENSIONS
from .stream_extract import ARCHIVE_EXTENSIONS

KIND_DOCUMENT = "document"
KIND_ARCHIVE = "archive"
KIND_MEDIA = "media"
KIND_UNSUPPORTED = "unsupported"
KIND_MESSAGE_EXPORT = "message_export"
KIND_AI_CHAT_EXPORT = "ai_chat_export"

STATUS_ROUTED_AI_CHAT = "routed_ai_chat"
STATUS_ROUTED_MESSAGES = "routed_message_export"
STATUS_CONTAINER = "container"
STATUS_MEDIA = "media_not_extracted"
STATUS_UNSUPPORTED = "unsupported"

IMAGE_EXTENSIONS = frozenset(
    {".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".bmp", ".tif", ".tiff"}
)
AUDIO_EXTENSIONS = frozenset({".mp3", ".m4a", ".wav", ".aac", ".ogg", ".amr", ".flac", ".opus"})
VIDEO_EXTENSIONS = frozenset({".mp4", ".mov", ".avi", ".mkv", ".3gp", ".webm", ".wmv", ".m4v"})
MEDIA_EXTENSIONS = IMAGE_EXTENSIONS | AUDIO_EXTENSIONS | VIDEO_EXTENSIONS
EXCLUDED_SEGMENTS = frozenset({".git", ".review_hold", "to_be_deleted", "__pycache__"})
_TEXT_EXTENSIONS = frozenset(SUPPORTED_EXTENSIONS)

CONVERSATION_MODES = ("chunk", "route")

# AI chat export file names (ChatGPT data export, Claude data export). A name match alone is not
# enough: the head of
# the object must also look like one (sniff_ai_chat), because other tools also write
# conversations.json.
AI_CHAT_FILE_NAMES = frozenset({"conversations.json"})


def is_excluded(key: str) -> bool:
    path = PurePosixPath(key)
    if not key or path.is_absolute() or ".." in path.parts:
        return True
    return bool(EXCLUDED_SEGMENTS.intersection(path.parts))


def classify_key(key: str) -> str:
    """The kind a key can be sent to without reading it. XML and conversations.json are refined by
    sniffing."""
    extension = PurePosixPath(key).suffix.casefold()
    if extension in ARCHIVE_EXTENSIONS:
        return KIND_ARCHIVE
    if extension in _TEXT_EXTENSIONS:
        return KIND_DOCUMENT
    if extension in MEDIA_EXTENSIONS:
        return KIND_MEDIA
    return KIND_UNSUPPORTED


def is_indexable_kind(kind: str) -> bool:
    """Kinds that become document rows and chunks (the extract stage). Everything else is inventory
    only."""
    return kind in {KIND_DOCUMENT, KIND_ARCHIVE}


def inventory_status(kind: str) -> str:
    return {
        KIND_MEDIA: STATUS_MEDIA,
        KIND_UNSUPPORTED: STATUS_UNSUPPORTED,
        KIND_ARCHIVE: STATUS_CONTAINER,
    }.get(kind, STATUS_UNSUPPORTED)


def maybe_ai_chat_name(key: str) -> bool:
    return PurePosixPath(key).name.casefold() in AI_CHAT_FILE_NAMES


def sniff_ai_chat(head: bytes) -> bool:
    """True when the first bytes look like a ChatGPT or Claude export: an array of conversations
    whose messages sit
    in a ``mapping`` tree (ChatGPT) or a ``chat_messages`` list (Claude). Looks only at a bounded
    head."""
    text = head[:131072].decode("utf-8", errors="ignore")
    if not text.lstrip().startswith("["):
        return False
    chatgpt = '"mapping"' in text and ('"current_node"' in text or '"conversation_id"' in text)
    claude = '"chat_messages"' in text and '"sender"' in text
    return chatgpt or claude


def sniff_message_export(head: bytes) -> bool:
    """True for an smsbackuprestore-style XML (``<smses>``/``<calls>`` roots, ``<sms>``/``<mms>``
    records)."""
    text = head[:65536].decode("utf-8", errors="ignore").casefold()
    return "<smses" in text or "<calls" in text or "<sms " in text or "<mms " in text
