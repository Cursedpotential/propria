"""The ONE place that knows where decoded media lives relative to a source key.

Byline: Claude Code · Sonnet 5 · 2026-09-21.

Owner ruling (2026-09-21): derived output is moving OUT from beside the
original object (today's layout) to a separate top-level vault directory
whose inner path mirrors the source tree. The new directory's name has not
been chosen and will be configuration, not code. This module exists so that
move touches exactly one function; every caller already treats its return
value as an opaque bucket-relative prefix ending in "/".

Current (pre-move) layout, verified live 2026-09-21 against
`context.proffer_preview_binding.source_ref` rows:

    source key (original XML):
        consignatio/vault/v1/calls-20260912155315.xml
    -> media prefix:
        consignatio/vault/v1/calls-20260912155315.xml.derived/media/

    source key (a derived thread chunk beside the same original):
        consignatio/vault/v1/sms-20260110021338.xml.derived/threads/8103533592_8103535467_self.ndjson
    -> media prefix:
        consignatio/vault/v1/sms-20260110021338.xml.derived/media/

The rule anchors on the LAST "<suffix>/" marker (default suffix ".derived",
configured below) rather than on a file extension: when the key already
contains that marker (it is itself a derived child, e.g. a thread ndjson),
the prefix is everything up to and including that marker, plus the media
dirname. When it does not (it is the original source object, of whatever
format), the prefix is the whole key plus the marker plus the dirname. This
deliberately does NOT search for ".xml" or any other extension, so it never
mistakes a directory that happens to contain the marker text for the file's
own boundary (e.g. a directory literally named "v1.xml-archive/" resolves as
a directory, not as the anchor), and it keeps working once non-XML sources
(json, txt, mbox, ...) start producing derived media of their own.

Object keys are opaque byte strings here — they may contain spaces or other
characters that are unremarkable in an S3 key but meaningful in a URL, so
this function never encodes, decodes, strips, or otherwise normalizes
`source_key`; it only slices and concatenates.
"""

from __future__ import annotations

from app.config import settings


def derived_media_prefix(source_key: str) -> str:
    """Map a source object key to its decoded-media prefix (bucket-relative)."""
    marker = f"{settings.proffer_derived_media_suffix}/"
    dirname = settings.proffer_derived_media_dirname
    marker_index = source_key.rfind(marker)
    if marker_index >= 0:
        base_with_marker = source_key[: marker_index + len(marker)]
        return f"{base_with_marker}{dirname}/"
    return f"{source_key}{marker}{dirname}/"
