"""Resolve and stream one decoded media object beside a Proffer run's source.

Byline: Claude Code · Sonnet 5 · 2026-09-21.

Backs `GET /api/proffer/previews/{preview_handle}/media/{sha256}`. This module
resolves WHICH object is allowed and reads it; app.runtime.proffer_media only
turns the result into an HTTP response (status, headers, StreamingResponse).
"""

from __future__ import annotations

import mimetypes
import re
from dataclasses import dataclass
from typing import Iterator

from app.repo.proffer_media_store import list_media_objects, open_media_object
from app.service.proffer import ProfferError, _require_mode
from app.service.proffer_media_prefix import derived_media_prefix
from app.service.proffer_operations import operation
from app.types.matter_mode import MatterMode
from app.types.source_roots import in_configured_root

CHUNK_BYTES = 256 * 1024
_SHA256 = re.compile(r"^[0-9a-f]{64}$")
_SAFE_EXTENSION = re.compile(r"^\.[A-Za-z0-9]{1,16}$")
_INLINE_TYPES = {"application/pdf", "text/vcard", "text/x-vcard"}
_INLINE_PREFIXES = ("image/", "audio/", "video/")


@dataclass(frozen=True)
class ResolvedMedia:
    scheme: str
    bucket: str
    key: str
    size: int
    extension: str


@dataclass(frozen=True)
class MediaStreamResult:
    iterator: Iterator[bytes]
    status_code: int
    content_type: str
    content_length: int
    total_size: int
    extension: str
    disposition: str
    byte_range: tuple[int, int] | None


def _validate_sha256(value: str) -> str:
    if not _SHA256.fullmatch(value):
        raise ProfferError("sha256 must be exactly 64 lowercase hex characters", 422)
    return value


def _split_source_ref(source_ref: str) -> tuple[str, str, str]:
    """Split "<scheme>://<bucket>/<key>" without treating the key as a URL.

    `source_ref` names a literal object-store key, not a URL: it may contain
    spaces or "%" sequences that are unremarkable in an S3 key but would be
    silently rewritten by urllib's URL parsing/unquoting. A plain partition
    keeps every byte of the key exactly as the run recorded it.
    """
    scheme, separator, rest = source_ref.partition("://")
    bucket, separator2, key = rest.partition("/")
    if not separator or not separator2 or not key:
        raise ProfferError("this run's source_ref is malformed", 502)
    return scheme.lower(), bucket, key


def _content_type_for(extension: str, metadata_type: str | None) -> str:
    if metadata_type and metadata_type != "application/octet-stream":
        return metadata_type
    guessed = mimetypes.guess_type(f"object{extension}")[0]
    return guessed or metadata_type or "application/octet-stream"


def _disposition_for(content_type: str) -> str:
    if content_type in _INLINE_TYPES or content_type.startswith(_INLINE_PREFIXES):
        return "inline"
    return "attachment"


def _parse_range(header: str | None, size: int) -> tuple[int, int] | None:
    """Return the inclusive (start, end) byte range, or None for a full read."""
    if not header:
        return None
    if not header.startswith("bytes="):
        raise ProfferError("Range header must use the bytes unit", 416)
    spec = header[len("bytes=") :]
    if "," in spec or "-" not in spec:
        raise ProfferError("only a single byte range is supported", 416)
    start_text, _, end_text = spec.partition("-")
    if not start_text and not end_text:
        raise ProfferError("Range header is malformed", 416)
    try:
        if not start_text:
            length = int(end_text)
            if length <= 0 or size == 0:
                raise ProfferError("Range is not satisfiable", 416)
            start, end = max(size - length, 0), size - 1
        else:
            start = int(start_text)
            end = int(end_text) if end_text else size - 1
    except ValueError:
        raise ProfferError("Range header is malformed", 416) from None
    if start < 0 or end < start or start >= size:
        raise ProfferError("Range is not satisfiable", 416)
    return start, min(end, size - 1)


async def resolve_media(preview_handle: str, sha256: str, *, mode: MatterMode) -> ResolvedMedia:
    """Resolve `sha256` to one object, refusing anything outside the run's own prefix."""
    sha256 = _validate_sha256(sha256)
    await _require_mode(preview_handle, mode)
    run = await operation(preview_handle)

    # in_configured_root() unquotes internally for its own "../" traversal check
    # (so an encoded traversal can't sneak past it); it is given the literal
    # key here, on purpose — the store calls below need the literal key, and
    # the root prefixes it compares against are early path segments where a
    # later "%" cannot change the containment outcome. Fetch the literal,
    # validate the decoded — do not "fix" this into a matched unquote/no-op pair.
    if not in_configured_root(run.source_ref):
        raise ProfferError("this run's source is outside a configured source root", 403)
    scheme, bucket, source_key = _split_source_ref(run.source_ref)

    media_prefix = derived_media_prefix(source_key)
    if not in_configured_root(f"{scheme}://{bucket}/{media_prefix}"):  # same literal-key note as above
        raise ProfferError("derived media prefix is outside a configured source root", 502)

    key_prefix = media_prefix + sha256
    try:
        candidates = list_media_objects(scheme, bucket, key_prefix)
    except RuntimeError as error:
        raise ProfferError("Media store is unreachable", 503) from error

    matches: list[tuple[dict, str]] = []
    for item in candidates:
        key = str(item.get("Key", ""))
        if not key.startswith(key_prefix):
            continue  # a listing-provider quirk, or a sha256 that is a prefix of another one
        remainder = key[len(key_prefix) :]
        if remainder and not remainder.startswith("."):
            continue  # content-addressed names match exactly: "" or ".<ext>", never "extra.jpg"
        if not in_configured_root(f"{scheme}://{bucket}/{key}"):  # same literal-key note as above
            continue
        matches.append((item, remainder))
    if not matches:
        raise ProfferError("media object was not found", 404)
    chosen, extension = min(matches, key=lambda pair: str(pair[0]["Key"]))
    return ResolvedMedia(
        scheme=scheme,
        bucket=bucket,
        key=str(chosen["Key"]),
        size=int(chosen.get("Size", 0)),
        # Never trust the bucket key into a response header: keep only a short
        # alnum extension, drop anything else (CR/LF, quotes, extra dots, ...).
        extension=extension if _SAFE_EXTENSION.fullmatch(extension) else "",
    )


async def stream_preview_media(
    preview_handle: str, sha256: str, *, mode: MatterMode, range_header: str | None
) -> MediaStreamResult:
    media = await resolve_media(preview_handle, sha256, mode=mode)
    byte_range = _parse_range(range_header, media.size)
    s3_range = f"bytes={byte_range[0]}-{byte_range[1]}" if byte_range else None
    try:
        response = open_media_object(media.scheme, media.bucket, media.key, byte_range=s3_range)
    except FileNotFoundError:
        raise ProfferError("media object was not found", 404) from None
    except ValueError:
        raise ProfferError("Range is not satisfiable", 416) from None
    except RuntimeError as error:
        raise ProfferError("Media store is unreachable", 503) from error

    body = response["Body"]

    def iterate() -> Iterator[bytes]:
        try:
            while True:
                chunk = body.read(CHUNK_BYTES)
                if not chunk:
                    break
                yield chunk
        finally:
            close = getattr(body, "close", None)
            if callable(close):
                close()

    content_type = _content_type_for(media.extension, response.get("ContentType"))
    content_length = (byte_range[1] - byte_range[0] + 1) if byte_range else media.size
    return MediaStreamResult(
        iterator=iterate(),
        status_code=206 if byte_range else 200,
        content_type=content_type,
        content_length=content_length,
        total_size=media.size,
        extension=media.extension,
        disposition=_disposition_for(content_type),
        byte_range=byte_range,
    )
