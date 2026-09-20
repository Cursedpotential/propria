# Byline: Claude Code · Fable 5.1 · 2026-09-20 (owner: storage providers are configuration, not code)
"""Configured object stores and source roots.

Cloudflare R2, Backblaze B2 and any other S3-compatible provider differ only in
an endpoint and a credential file, so neither is named in code. The Workbench
API and the Go engine read the same two documents:

    OBJECT_STORES_JSON = {"b2": "/run/secrets/casebible-b2.json", "r2": "/run/secrets/casebible-r2.json"}
    SOURCE_ROOTS_JSON  = [{"id": "b2-vault", "label": "B2 / Vault", "url": "b2://bucket/prefix/"}]

A store's key is the locator scheme. Switching or adding a provider is a
credential file plus these two values. This module is stdlib-only because
``types`` is the bottom layer.
"""

from __future__ import annotations

import json
import os
import re
from dataclasses import dataclass
from urllib.parse import unquote, urlsplit

STORES_ENV = "OBJECT_STORES_JSON"
ROOTS_ENV = "SOURCE_ROOTS_JSON"
_SCHEME = re.compile(r"^[a-z][a-z0-9]{0,15}$")
_RESERVED_SCHEMES = {"upload", "file", "http", "https", "s3"}

# Behaviour before 2026-09-20, kept for a deploy that has not set the variables.
_LEGACY_ROOTS = (
    # r2-sorted first: the first root is the browser default, as it was before.
    {"id": "r2-sorted", "label": "R2 / Case Bible Sorted", "url": "r2://casebible-sorted/", "temporary": True},
    {"id": "r2-raw", "label": "R2 / Case Bible Raw", "url": "r2://casebible-raw/", "temporary": True},
    {
        "id": "r2-quarantine",
        "label": "R2 / Case Bible Quarantine",
        "url": "r2://casebible-quarantine/",
        "temporary": True,
    },
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
    scheme: str = "r2"
    key_prefix: str = ""

    def source_ref(self, key: str) -> str:
        """The locator Proffer receives for a browser-relative key."""
        return f"{self.scheme}://{self.bucket}/{self.key_prefix}{key}"


def parse_source_roots(raw: str) -> dict[str, SourceRoot]:
    """Decode SOURCE_ROOTS_JSON; empty input keeps the legacy R2 roots."""
    entries = json.loads(raw) if raw.strip() else list(_LEGACY_ROOTS)
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
            root_id,
            label,
            bucket,
            f"{scheme}://{bucket}/{prefix}",
            temporary=bool(entry.get("temporary", False)),
            scheme=scheme,
            key_prefix=prefix,
        )
    return roots


def parse_object_stores(raw: str, *, legacy_r2_path: str = "") -> dict[str, str]:
    """Decode OBJECT_STORES_JSON: locator scheme -> credential document path."""
    if not raw.strip():
        return {"r2": legacy_r2_path} if legacy_r2_path else {}
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
    return parse_source_roots(os.environ.get(ROOTS_ENV, ""))


def configured_object_stores(*, legacy_r2_path: str = "") -> dict[str, str]:
    return parse_object_stores(os.environ.get(STORES_ENV, ""), legacy_r2_path=legacy_r2_path)


def in_configured_root(value: str) -> bool:
    """True when ``value`` names an object inside a configured source root."""
    parsed = urlsplit(value)
    if parsed.query or parsed.fragment or "@" in parsed.netloc or not parsed.netloc:
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
    """Accept an opaque upload, the fixed Case Bible R2 scope, or an object in a configured root."""
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
    if (
        parsed.scheme == "r2"
        and (
            parsed.netloc in {"casebible-raw", "casebible-sorted", "casebible-quarantine"}
            or (parsed.netloc == "nexus" and re.fullmatch(r"/workbench/staging/[0-9a-f]{64}/.+", unquote(parsed.path)))
        )
        and not parsed.query
        and not parsed.fragment
    ):
        key = unquote(parsed.path.removeprefix("/"))
        if key and not key.startswith("/") and "\\" not in key and ".." not in key.split("/"):
            return value
    if in_configured_root(value):
        return value
    raise ValueError("source_ref must be an upload reference or an object inside a configured source root")
