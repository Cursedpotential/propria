"""Find and read the files that accompany one source file (its sidecars).

Byline: Claude Code · Opus 5.5 · 2026-09-26.

Owner 2026-09-25 19:15: the metadata screen covers "any available accompanying
sidecar" — a Google Takeout `<file>.json` / `.supplemental-metadata.json`,
`.xmp`, Apple `.aae`, the owner's own `*.sidecar.md` / `*.EXTRACTION.md`
analyses (shown, never flagged or moved), and any same-stem companion.

Two bounded lookups, never a bucket scan: the objects directly beside the file
whose names start with its stem (one delimited listing), and the files the
catalog records in the file's original folder with that stem (the catalog is
the source of truth for what sat beside a file). Sidecar values are shown with
where they came from; a disagreement with the file's own values is reported by
`sidecar_conflicts`, never merged.
"""

from __future__ import annotations

import json
from typing import Any

from app.repo.intake_discovery import DiscoveryError, catalog_companions, configured
from app.repo.proffer_media_store import list_objects_beside, read_small_object
from app.service.proffer_media import _split_source_ref
from app.types.source_metadata import Sidecar, SidecarField, SidecarKind, SidecarLookup
from app.types.source_roots import SourceRoot, configured_source_roots, in_configured_root

READ_MAX_BYTES = 256 * 1024
TEXT_SHOW_MAX = 64 * 1024
FIELDS_MAX = 300
VALUE_TEXT_MAX = 2000
LOOKUP_MAX = 50
_STEM_SEPARATORS = ".( -_~"
_TAKEOUT_KEYS = {"photoTakenTime", "creationTime", "geoData", "geoDataExif", "photoLastModifiedTime", "people"}
_READABLE: set[SidecarKind] = {"json", "xmp", "apple_aae", "owner_sidecar_md", "owner_extraction_md"}


def stem_of(name: str) -> str:
    """`IMG_1234.jpg` -> `IMG_1234`; a name without an extension is its own stem."""
    head, dot, _ = name.rpartition(".")
    return head if dot and head else name


def classify(subject_name: str, candidate: str) -> SidecarKind | None:
    """Name-based sidecar kind of `candidate` for `subject_name`, or None if unrelated."""
    subject, lower, stem = subject_name.casefold(), candidate.casefold(), stem_of(subject_name).casefold()
    if lower == subject or not lower.startswith(stem):
        return None
    rest = lower[len(stem) :]
    if rest and rest[0] not in _STEM_SEPARATORS:
        return None  # IMG_12345.jpg is not a companion of IMG_1234.jpg
    if lower.endswith(".extraction.md"):
        return "owner_extraction_md"
    if lower.endswith(".sidecar.md"):
        return "owner_sidecar_md"
    if lower.endswith(".json"):
        return "json"
    if lower.endswith(".xmp"):
        return "xmp"
    if lower.endswith(".aae"):
        return "apple_aae"
    return "companion"


def _scalar(value: Any) -> Any:
    if isinstance(value, str) and len(value) > VALUE_TEXT_MAX:
        return value[:VALUE_TEXT_MAX] + "…"
    return value


def flatten(value: Any, prefix: str = "") -> tuple[list[SidecarField], bool]:
    """Dotted-path fields of a JSON document, bounded to FIELDS_MAX."""
    fields: list[SidecarField] = []
    stack: list[tuple[str, Any]] = [(prefix, value)]
    while stack:
        path, item = stack.pop()
        if isinstance(item, dict) and item:
            stack.extend((f"{path}.{key}" if path else str(key), child) for key, child in reversed(list(item.items())))
        elif isinstance(item, list) and item:
            stack.extend((f"{path}[{index}]", child) for index, child in reversed(list(enumerate(item))))
        else:
            if len(fields) >= FIELDS_MAX:
                return fields, True
            fields.append(
                SidecarField(
                    path=path or "(document)", value=_scalar(item if not isinstance(item, (dict, list)) else None)
                )
            )
    return fields, False


def _read(scheme: str, bucket: str, key: str, kind: SidecarKind, sidecar: Sidecar) -> Sidecar:
    if kind not in _READABLE:
        return sidecar
    if sidecar.byte_length is not None and sidecar.byte_length > READ_MAX_BYTES:
        sidecar.unread_reason = f"larger than {READ_MAX_BYTES // 1024} KiB"
        return sidecar
    try:
        raw = read_small_object(scheme, bucket, key, max_bytes=READ_MAX_BYTES)
    except ValueError:
        sidecar.unread_reason = f"larger than {READ_MAX_BYTES // 1024} KiB"
        return sidecar
    except (FileNotFoundError, RuntimeError):
        sidecar.unread_reason = "could not be read"
        return sidecar
    text = raw.decode("utf-8", errors="replace")
    if kind == "json":
        try:
            document = json.loads(text)
        except ValueError:
            sidecar.unread_reason = "not valid JSON"
        else:
            sidecar.fields, sidecar.fields_truncated = flatten(document)
            if isinstance(document, dict) and _TAKEOUT_KEYS & set(document):
                sidecar.kind = "takeout_json"
            return sidecar
    sidecar.text = text[:TEXT_SHOW_MAX]
    sidecar.text_truncated = len(text) > TEXT_SHOW_MAX
    return sidecar


def _root_for(scheme: str, bucket: str, key: str) -> SourceRoot | None:
    matches = [
        root
        for root in configured_source_roots().values()
        if root.scheme == scheme and root.bucket == bucket and key.startswith(root.key_prefix)
    ]
    return max(matches, key=lambda root: len(root.key_prefix), default=None)


def find_sidecars(source_ref: str) -> tuple[list[Sidecar], SidecarLookup]:
    """Sidecars of one source object. Blocking (object store + catalog): call from a thread."""
    if not in_configured_root(source_ref):
        return [], SidecarLookup(beside_object="not_applicable", catalog_folder="not_applicable")
    scheme, bucket, key = _split_source_ref(source_ref)
    directory, _, name = key.rpartition("/")
    stem = stem_of(name)
    folder = f"{directory}/" if directory else ""
    found: dict[str, Sidecar] = {}

    beside_state = "ok"
    try:
        for row in list_objects_beside(scheme, bucket, folder + stem, max_keys=LOOKUP_MAX):
            candidate_key = str(row.get("Key", ""))
            candidate_name = candidate_key.rpartition("/")[2]
            kind = classify(name, candidate_name)
            if kind is None or candidate_key == key:
                continue
            size = row.get("Size")
            sidecar = Sidecar(
                name=candidate_name,
                key=candidate_key,
                kind=kind,
                found_via="beside_object",
                byte_length=int(size) if isinstance(size, int) else None,
            )
            found[candidate_key] = _read(scheme, bucket, candidate_key, kind, sidecar)
    except RuntimeError:
        beside_state = "unavailable"

    catalog_state = "ok"
    root = _root_for(scheme, bucket, key)
    if not configured():
        catalog_state = "not_configured"
    elif root is None:
        catalog_state = "not_applicable"
    else:
        try:
            rows = catalog_companions(key[len(root.key_prefix) :], stem, limit=LOOKUP_MAX)["items"]
        except DiscoveryError:
            rows, catalog_state = [], "unavailable"
        for row in rows:
            candidate_name = str(row.get("name") or "")
            kind = classify(name, candidate_name)
            if kind is None:
                continue
            vault_key = row.get("vault_key")
            target = f"{root.key_prefix}{vault_key}" if vault_key else None
            if target is not None and not in_configured_root(f"{scheme}://{bucket}/{target}"):
                target = None  # never read outside a configured root, whatever the catalog says
            if target is not None and target in found:
                continue
            size = row.get("size")
            sidecar = Sidecar(
                name=candidate_name,
                key=target or str(row.get("rel") or candidate_name),
                kind=kind,
                found_via="catalog_folder",
                byte_length=int(size) if isinstance(size, int) else None,
            )
            if target is None:
                sidecar.unread_reason = "recorded in the catalog; not stored in this source root"
                found[f"catalog:{sidecar.key}"] = sidecar
            else:
                found[target] = _read(scheme, bucket, target, kind, sidecar)

    ordered = sorted(found.values(), key=lambda item: (item.kind == "companion", item.name.casefold()))
    return ordered, SidecarLookup(beside_object=beside_state, catalog_folder=catalog_state)
