"""Read SBV's decoded output for one source, before any ingest run exists.

Byline: Claude Code · Fable 5.1 · 2026-09-21.

Owner, 2026-09-21: SBV "was literally built to read these files and will
display the base64 inline". Read from SBV's source: its importer decodes the
inline base64 parts to bytes while it imports, and its viewer then serves the
decoded bytes. In this platform that importer is the derive step, which writes
`manifest.json`, thread files and a media folder. This module is the viewer's
read side: the manifest gives the conversation list, a thread file gives the
messages, and app.service.proffer_media serves the pictures by sha256.

Nothing here accepts a caller-supplied bucket or path. The source must sit in a
configured source root, and a thread file is accepted only when the source's own
manifest lists it.
"""

from __future__ import annotations

import json

from app.repo.proffer_media_store import iter_object_lines, read_small_object
from app.service.proffer import ProfferError
from app.service.proffer_media import _split_source_ref
from app.service.proffer_media_prefix import derived_base_prefix
from app.types.proffer_decoded import (
    DecodedAttachment,
    DecodedManifest,
    DecodedMessage,
    DecodedThread,
    DecodedThreadPage,
)
from app.types.source_roots import in_configured_root

MANIFEST_NAME = "manifest.json"
MANIFEST_MAX_BYTES = 32 * 1024 * 1024
PAGE_MAX = 500


def _locate(source_ref: str) -> tuple[str, str, str]:
    if not in_configured_root(source_ref):
        raise ProfferError("this source is outside a configured source root", 403)
    scheme, bucket, source_key = _split_source_ref(source_ref)
    base = derived_base_prefix(source_key)
    if not in_configured_root(f"{scheme}://{bucket}/{base}{MANIFEST_NAME}"):
        raise ProfferError("decoded output is outside a configured source root", 502)
    return scheme, bucket, base


def _load_manifest(scheme: str, bucket: str, base: str) -> dict:
    try:
        raw = read_small_object(scheme, bucket, base + MANIFEST_NAME, max_bytes=MANIFEST_MAX_BYTES)
    except FileNotFoundError:
        raise ProfferError("this source has not been decoded yet", 404) from None
    except ValueError:
        raise ProfferError("the decode manifest is larger than the viewer reads", 502) from None
    except RuntimeError as error:
        raise ProfferError("Object store is unreachable", 503) from error
    try:
        manifest = json.loads(raw)
    except json.JSONDecodeError:
        raise ProfferError("the decode manifest is not valid JSON", 502) from None
    if not isinstance(manifest, dict) or not isinstance(manifest.get("threads"), list):
        raise ProfferError("the decode manifest has no thread list", 502)
    return manifest


def _thread_entries(manifest: dict, base: str) -> dict[str, dict]:
    """Thread files keyed by their id: the object key relative to the decoded prefix."""
    entries: dict[str, dict] = {}
    for item in manifest["threads"]:
        key = str(item.get("key", "")) if isinstance(item, dict) else ""
        if key.startswith(base) and len(key) > len(base):
            entries[key[len(base) :]] = item
    return entries


def decoded_manifest(source_ref: str) -> DecodedManifest:
    scheme, bucket, base = _locate(source_ref)
    manifest = _load_manifest(scheme, bucket, base)
    threads = [
        DecodedThread(
            file=file_id,
            thread=str(item.get("thread", "")),
            chunk=int(item.get("chunk") or 0),
            participants=[str(value) for value in item.get("participants") or []],
            records=int(item.get("records") or 0),
            first_occurred_at=item.get("first_occurred_at") or None,
            last_occurred_at=item.get("last_occurred_at") or None,
        )
        for file_id, item in _thread_entries(manifest, base).items()
    ]
    return DecodedManifest(
        source_ref=source_ref,
        decoded_at=manifest.get("derived_at") or None,
        decoder=manifest.get("decoder") or None,
        records=int(manifest.get("records") or 0),
        rejected=int(manifest.get("rejected") or 0),
        media_objects=int(manifest.get("media_objects") or 0),
        media_bytes=int(manifest.get("media_bytes") or 0),
        threads=threads,
    )


def _message(ordinal: int, line: dict) -> DecodedMessage:
    attachments = [
        DecodedAttachment(
            ordinal=int(item.get("ordinal") or 0),
            name=item.get("name") or None,
            media_type=item.get("mime") or None,
            sha256=item.get("sha256") or None,
            byte_length=item.get("bytes") if isinstance(item.get("bytes"), int) else None,
        )
        for item in line.get("attachments") or []
        if isinstance(item, dict)
    ]
    missing = len(line.get("attachment_references") or []) + len(line.get("attachment_failures") or [])
    return DecodedMessage(
        ordinal=ordinal,
        kind=str(line.get("kind", "")),
        status=str(line.get("status", "")),
        occurred_at=line.get("occurred_at") or None,
        sender=line.get("sender") or None,
        participants=[str(value) for value in line.get("participants") or []],
        body=str(line.get("content") or ""),
        attachments=attachments,
        missing_attachments=missing,
    )


def decoded_thread_page(source_ref: str, file_id: str, *, offset: int, limit: int) -> DecodedThreadPage:
    """One page of one thread file, oldest first (the order SBV wrote it)."""
    scheme, bucket, base = _locate(source_ref)
    entry = _thread_entries(_load_manifest(scheme, bucket, base), base).get(file_id)
    if entry is None:
        raise ProfferError("that thread file is not listed in this source's decode manifest", 404)
    limit = max(1, min(limit, PAGE_MAX))
    total = int(entry.get("records") or 0)
    messages: list[DecodedMessage] = []
    seen = 0
    try:
        for raw_line in iter_object_lines(scheme, bucket, str(entry["key"])):
            if not raw_line.strip():
                continue
            if seen >= offset:
                try:
                    line = json.loads(raw_line)
                except json.JSONDecodeError:
                    raise ProfferError(f"thread file line {seen} is not valid JSON", 502) from None
                messages.append(_message(seen, line if isinstance(line, dict) else {}))
            seen += 1
            if len(messages) >= limit:
                break
    except FileNotFoundError:
        raise ProfferError("the thread file listed in the manifest is missing", 502) from None
    except RuntimeError as error:
        raise ProfferError("Object store is unreachable", 503) from error
    next_offset = offset + len(messages) if messages and offset + len(messages) < total else None
    return DecodedThreadPage(
        file=file_id, offset=offset, total_records=total, messages=messages, next_offset=next_offset
    )
