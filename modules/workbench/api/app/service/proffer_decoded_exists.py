"""Does a decode manifest exist for these sources? One HEAD per source, batched.

Byline: Claude Code · Opus 5 · 2026-09-22.

The Sources screen paints a state mark on every row. "decoded" means SBV's
derivation manifest object exists for that source — never a guess from the
file name (owner rule: nothing about a file is inferred from its name). The
viewer's own `GET /decoded/manifest` answers the same question but reads the
manifest body; this asks the object store to HEAD it instead, for a page of
rows in one request.
"""

from __future__ import annotations

from app.repo.proffer_media_store import object_exists
from app.service.proffer_decoded import MANIFEST_NAME, _locate
from app.service.proffer_errors import ProfferError
from app.types.proffer_decoded_exists import DecodedExistsItem, DecodedExistsResponse

MAX_SOURCES = 200


def decoded_exists(source_refs: list[str]) -> DecodedExistsResponse:
    """One entry per requested source, in the order asked."""
    if len(source_refs) > MAX_SOURCES:
        raise ProfferError(f"ask about at most {MAX_SOURCES} sources in one request", 422)
    items: list[DecodedExistsItem] = []
    for source_ref in source_refs:
        items.append(_one(source_ref))
    return DecodedExistsResponse(items=items)


def _one(source_ref: str) -> DecodedExistsItem:
    try:
        scheme, bucket, base = _locate(source_ref)
    except ProfferError as error:
        # An unreachable or out-of-root source is reported per row, not as a
        # page-wide failure: one bad row must not blank every state mark.
        return DecodedExistsItem(source_ref=source_ref, decoded=False, reason=error.detail)
    try:
        found = object_exists(scheme, bucket, base + MANIFEST_NAME)
    except RuntimeError:
        return DecodedExistsItem(
            source_ref=source_ref, decoded=False, reason="object store is unreachable"
        )
    return DecodedExistsItem(source_ref=source_ref, decoded=found, reason="")
