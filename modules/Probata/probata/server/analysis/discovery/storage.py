# Byline: Codex · GPT-6.1-sol · 2026-10-11
"""Retain bounded discovery JSON outside workflow history with exact replay.

Inputs: explicit shared scratch root and immutable references. Outputs: verified
JSON and content-addressed references. Effects: bounded files only; choose over
canonical persistence, which belongs to PostgreSQL and the Go writer.
"""

from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
from typing import Any
from urllib.parse import unquote, urlsplit
from uuid import uuid4

MAX_BYTES = 8 * 1024 * 1024


class DiscoveryInvalid(ValueError):
    """Reject permanently invalid discovery snapshots without exposing source text.

    Inputs: safe diagnostic. Outputs: exception. Effects: none; choose instead of
    retrying malformed or mismatched references.
    """


def encode(value: Any) -> bytes:
    """Encode deterministic finite JSON for derivative cache identity.

    Inputs: JSON value. Outputs: UTF-8 bytes. Effects: none; choose for derived
    files, never custody hashes or source reinterpretation.
    """
    try:
        return json.dumps(value, sort_keys=True, ensure_ascii=False, separators=(",", ":"), allow_nan=False).encode()
    except (TypeError, ValueError):
        raise DiscoveryInvalid("discovery JSON contains an invalid or nonfinite value") from None


def fingerprint(value: Any) -> str:
    """Identify one derived configuration without implying source verification.

    Inputs: JSON value. Outputs: SHA256 identity. Effects: none; choose for cache
    pins and retry identity, never evidence custody.
    """
    return hashlib.sha256(encode(value)).hexdigest()


def root() -> Path:
    """Resolve the explicit existing shared discovery scratch directory.

    Inputs: ADAPTIVE_DISCOVERY_ROOT. Outputs: absolute root. Effects: environment
    read; choose fail-closed configuration over creating an implicit store.
    """
    value = os.environ.get("ADAPTIVE_DISCOVERY_ROOT", "")
    path = Path(value)
    if not value or not path.is_absolute() or not path.is_dir():
        raise DiscoveryInvalid("ADAPTIVE_DISCOVERY_ROOT must be an existing absolute directory")
    if any(part.is_symlink() for part in [path, *path.parents]):
        raise DiscoveryInvalid("discovery root cannot traverse symlinks")
    return path


def safe_path(uri: str) -> Path:
    """Admit only plain local file references inside the configured scratch root.

    Inputs: file URI. Outputs: safe path. Effects: path metadata reads; choose
    before every retained file operation rather than arbitrary file access.
    """
    if not isinstance(uri, str):
        raise DiscoveryInvalid("discovery reference URI must be text")
    parsed = urlsplit(uri)
    path = Path(unquote(parsed.path))
    if parsed.scheme != "file" or parsed.netloc or parsed.query or parsed.fragment or not path.is_absolute():
        raise DiscoveryInvalid("discovery reference must be a plain absolute local file URI")
    base = root()
    if ".." in path.parts or not path.is_relative_to(base):
        raise DiscoveryInvalid("discovery reference escaped configured root")
    if any(part.is_symlink() for part in [path, *path.parents]):
        raise DiscoveryInvalid("discovery reference cannot traverse symlinks")
    return path


def read(ref: dict[str, Any]) -> dict[str, Any]:
    """Read one bounded immutable JSON reference and verify its exact bytes.

    Inputs: uri/SHA256/bytes reference. Outputs: dictionary. Effects: bounded
    file read; choose for snapshot and result replay outside Temporal history.
    """
    if not isinstance(ref, dict) or set(ref) != {"uri", "sha256", "bytes"}:
        raise DiscoveryInvalid("discovery reference requires uri, sha256 and bytes")
    size = ref["bytes"]
    digest = ref["sha256"]
    if isinstance(size, bool) or not isinstance(size, int) or not 0 < size <= MAX_BYTES:
        raise DiscoveryInvalid("discovery reference bytes exceed limit")
    if not isinstance(digest, str) or len(digest) != 64 or any(c not in "0123456789abcdef" for c in digest):
        raise DiscoveryInvalid("discovery reference SHA256 is invalid")
    path = safe_path(ref["uri"])
    try:
        with path.open("rb") as stream:
            data = stream.read(MAX_BYTES + 1)
    except OSError:
        raise DiscoveryInvalid("discovery reference is unavailable") from None
    if len(data) != size or hashlib.sha256(data).hexdigest() != digest:
        raise DiscoveryInvalid("discovery reference bytes or digest differ")
    try:
        value = json.loads(data, parse_constant=_reject_nonfinite)
    except (ValueError, UnicodeError):
        raise DiscoveryInvalid("discovery reference is invalid JSON") from None
    if not isinstance(value, dict):
        raise DiscoveryInvalid("discovery snapshot must be an object")
    return value


def _reject_nonfinite(value: str) -> None:
    """Reject nonfinite JSON literals instead of silently changing snapshot content.

    Inputs: invalid literal. Outputs: exception. Effects: none; choose strict
    finite derivative JSON for cross-runtime score/reference agreement.
    """
    raise DiscoveryInvalid("discovery JSON contains a nonfinite literal")


def save(value: dict[str, Any], identity: Any) -> dict[str, Any]:
    """Atomically retain one deterministic derived result without overwriting history.

    Inputs: JSON result and complete revision identity. Outputs: immutable file
    reference. Effects: fsynced pending file, exclusive hardlink and directories;
    choose for retry/restart replay, never canonical state or source replacement.
    """
    data = encode(value)
    if not 0 < len(data) <= MAX_BYTES:
        raise DiscoveryInvalid("discovery result exceeds retained byte bound")
    base = root()
    parent = base / "results"
    parent.mkdir(exist_ok=True)
    path = safe_path((parent / (fingerprint(identity) + ".json")).as_uri())
    if path.exists():
        if not _matches(path, data):
            raise DiscoveryInvalid("different discovery result occupies pinned identity")
    else:
        pending = parent / (path.stem + "." + uuid4().hex + ".pending")
        descriptor = os.open(pending, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o400)
        with os.fdopen(descriptor, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        try:
            os.link(pending, path)
        except FileExistsError:
            if not _matches(path, data):
                raise DiscoveryInvalid("concurrent discovery output differs") from None
    for directory in (parent, base):
        descriptor = os.open(directory, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(descriptor)
        finally:
            os.close(descriptor)
    return {"uri": path.as_uri(), "sha256": hashlib.sha256(data).hexdigest(), "bytes": len(data)}


def _matches(path: Path, data: bytes) -> bool:
    """Compare an existing exclusive output using bounded safe-path reads.

    Inputs: retained path and expected bytes. Outputs: exact equality. Effects:
    bounded file read; choose over overwriting or trusting a concurrent symlink.
    """
    checked = safe_path(path.as_uri())
    with checked.open("rb") as stream:
        return stream.read(MAX_BYTES + 1) == data
