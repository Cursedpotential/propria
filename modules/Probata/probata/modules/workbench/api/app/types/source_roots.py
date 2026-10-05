# Byline: Claude Code · Fable 5.1 · 2026-09-20; Codex · GPT-6-Luna · 2026-10-04
"""Parse explicitly configured S3-compatible stores and roots for Workbench acquisition.

The Workbench API and Go engine share ``OBJECT_STORES_JSON`` (scheme to
credential path) and ``SOURCE_ROOTS_JSON`` (source locator roots). Provider
primitives stay generic; the only built-in root is the owner-declared B2
Casevault home, and credentials are never implicit. This module is stdlib-only
because ``types`` is the bottom layer.
"""

from __future__ import annotations

import json
import os
import re
from dataclasses import dataclass
from urllib.parse import parse_qsl, unquote, urlsplit

STORES_ENV = "OBJECT_STORES_JSON"
ROOTS_ENV = "SOURCE_ROOTS_JSON"
_SCHEME = re.compile(r"^[a-z][a-z0-9]{0,15}$")
_RESERVED_SCHEMES = {"upload", "file", "http", "https", "s3"}

# Owner-declared current source root. This keeps the existing Workbench default-root
# contract while eliminating the retired R2 roots and preserving generic explicit configs.
_DEFAULT_ROOTS = (
    {"id": "b2-vault", "label": "B2 / Casevault", "url": "b2://salem-data/consignatio/casevault/"},
)


@dataclass(frozen=True)
class SourceRoot:
    """One configured browser root: ``<scheme>://<bucket>/<key_prefix>``.

    Browser keys are relative to ``key_prefix``; the repo layer prepends it on
    every request and strips it from every response, so the browser can never
    leave the root. ``scheme`` names a configured object store.
    """

    root_id: str
    label: str
    bucket: str
    root_ref: str
    temporary: bool = True
    scheme: str = ""
    key_prefix: str = ""

    def source_ref(self, key: str) -> str:
        """The locator Proffer receives for a browser-relative key."""
        return f"{self.scheme}://{self.bucket}/{self.key_prefix}{key}"


def parse_source_roots(raw: str) -> dict[str, SourceRoot]:
    """Parse configured roots and use only the owner-declared B2 default when unset.

    Inputs: raw SOURCE_ROOTS_JSON.
    Outputs: roots keyed by configured id, or the canonical B2 Casevault root when unset.
    Side effects: none; malformed explicit configuration raises ValueError.
    When to choose it: use for source-root config; choose ``parse_object_stores`` for credentials.
    Only the retired R2 default is removed; this B2 default preserves the repository adapter's
    first-root selection contract without providing credentials or rewriting persisted locators.
    """
    entries = json.loads(raw) if raw.strip() else list(_DEFAULT_ROOTS)
    if not isinstance(entries, list) or not entries:
        raise ValueError(f"{ROOTS_ENV} must be a non-empty JSON array")
    roots: dict[str, SourceRoot] = {}
    for entry in entries:
        if not isinstance(entry, dict):
            raise ValueError("every source root must be an object")
        root_id, label, url = (str(entry.get(name, "")).strip() for name in ("id", "label", "url"))
        scheme, separator, rest = url.partition("://")
        bucket, _, prefix = rest.partition("/")
        if not root_id or not label or not separator or not bucket or root_id in roots:
            raise ValueError("every source root needs a unique id, a label and a <scheme>://<bucket>/<prefix/> url")
        if not _SCHEME.fullmatch(scheme) or "@" in bucket or "?" in url or "#" in url:
            raise ValueError(f"source root {root_id} has an invalid url")
        segments = prefix.split("/")
        if prefix and (not prefix.endswith("/") or ".." in segments or "." in segments):
            raise ValueError(f"source root {root_id} prefix must end in / and hold no dot segments")
        roots[root_id] = SourceRoot(
            root_id=root_id,
            label=label,
            bucket=bucket,
            root_ref=f"{scheme}://{bucket}/{prefix}",
            scheme=scheme,
            temporary=bool(entry.get("temporary", False)),
            key_prefix=prefix,
        )
    return roots


def parse_object_stores(raw: str, *, legacy_r2_path: str = "") -> dict[str, str]:
    """Parse explicit credential paths and refuse the retired R2 fallback.

    Inputs: raw OBJECT_STORES_JSON and an optional legacy R2 path from older callers.
    Outputs: configured schemes mapped to absolute credential paths.
    Side effects: none; legacy fallback use or malformed config raises ValueError.
    When to choose it: use for credential config; choose ``parse_source_roots`` for locator roots.
    """
    if not raw.strip():
        if legacy_r2_path:
            raise ValueError("legacy R2 credential fallback is retired; configure an active object store explicitly")
        return {}
    stores = json.loads(raw)
    if not isinstance(stores, dict) or not stores:
        raise ValueError(f"{STORES_ENV} must be a non-empty JSON object")
    for scheme, path in stores.items():
        if not _SCHEME.fullmatch(str(scheme)) or scheme in _RESERVED_SCHEMES:
            raise ValueError(f"{STORES_ENV} scheme {scheme!r} must be lower-case letters and digits and not reserved")
        if not isinstance(path, str) or not path.startswith("/") or path != path.strip():
            raise ValueError(f"{STORES_ENV} credential path for {scheme!r} must be absolute")
    return stores


def configured_source_roots() -> dict[str, SourceRoot]:
    """Read configured roots or the canonical B2 Casevault default from the environment.

    Inputs: SOURCE_ROOTS_JSON environment value.
    Outputs: configured roots, or the canonical B2 Casevault root when unset.
    Side effects: reads process environment only.
    When to choose it: use for environment config; choose ``parse_source_roots`` for supplied JSON.
    """
    return parse_source_roots(os.environ.get(ROOTS_ENV, ""))


def configured_object_stores(*, legacy_r2_path: str = "") -> dict[str, str]:
    """Read explicit object-store credentials from the process environment.

    Inputs: OBJECT_STORES_JSON environment value and an optional legacy R2 path.
    Outputs: configured stores, or an empty mapping when unset.
    Side effects: reads process environment only; requesting the retired fallback raises ValueError.
    When to choose it: use for environment config; choose ``parse_object_stores`` for supplied JSON.
    """
    return parse_object_stores(os.environ.get(STORES_ENV, ""), legacy_r2_path=legacy_r2_path)


# Byline: Codex · GPT-6 · 2026-10-05.
def valid_object_version_query(raw: str) -> bool:
    """Admit one exact retained provider version while rejecting ambiguous locator queries.

    Inputs: undecoded query string. Outputs: validity boolean. Side effects: none.
    Choose at configured acquisition boundaries; an empty query retains legacy unversioned reads.
    """
    if not raw:
        return True
    if re.search(r"%(?![0-9A-Fa-f]{2})", raw):
        return False
    try:
        items = parse_qsl(raw, keep_blank_values=True, strict_parsing=True, errors="strict")
    except (ValueError, UnicodeError):
        return False
    if len(items) != 1 or items[0][0] != "versionId":
        return False
    value = items[0][1]
    return bool(value) and value != "null" and len(value.encode("utf-8")) <= 2048 and not any(c in value for c in "\r\n\x00")


def in_configured_root(value: str) -> bool:
    """Check whether a locator names an object inside an explicit source root.

    Inputs: an object locator string.
    Outputs: true only when its scheme, bucket, and key are inside a configured root.
    Side effects: reads SOURCE_ROOTS_JSON from the process environment.
    When to choose it: use for generic root membership; choose ``validate_authorized_source_ref``
    for acquisition validation that also rejects retired R2 references.
    """
    parsed = urlsplit(value)
    if not valid_object_version_query(parsed.query) or parsed.fragment or "@" in parsed.netloc or not parsed.netloc:
        return False
    key = unquote(parsed.path.removeprefix("/"))
    if not key or key.startswith("/") or "\\" in key or ".." in key.split("/"):
        return False
    return any(
        root.scheme == parsed.scheme.lower()
        and root.bucket == parsed.netloc
        and key.startswith(root.key_prefix)
        and len(key) > len(root.key_prefix)
        for root in configured_source_roots().values()
    )


def validate_authorized_source_ref(value: str) -> str:
    """Validate a current acquisition locator and reject retired R2 references.

    Inputs: an opaque upload digest locator or object locator string.
    Outputs: the unchanged authorized locator; otherwise raises ValueError.
    Side effects: reads SOURCE_ROOTS_JSON and never rewrites a source/version identity.
    When to choose it: use at current acquisition boundaries; historical R2 provenance remains
    stored as-is and is outside this validation path.
    """
    parsed = urlsplit(value)
    upload_digest = parsed.netloc.casefold()
    if (
        parsed.scheme == "upload"
        and len(upload_digest) == 64
        and all(character in "0123456789abcdef" for character in upload_digest)
        and not parsed.path
        and not parsed.query
        and not parsed.fragment
    ):
        return value
    if parsed.scheme == "r2":
        raise ValueError("r2:// is retired for new acquisitions; use the configured B2 Casevault source root")
    if in_configured_root(value):
        return value
    raise ValueError("source_ref must be an upload reference or an object inside a configured source root")
