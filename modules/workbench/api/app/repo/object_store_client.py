# Byline: Claude Code · Sonnet (agent) · 2026-07-19
# Byline: Codex · GPT-5.6-Sol · 2026-08-30 (fixed source/staging buckets and runtime credentials)
# Byline: Claude Code · Fable 5.1 · 2026-09-20 (object stores and source roots are configuration, not code)
"""S3-compatible object store repo layer for allowlisted Platform-owned R2 roots.

Adapted from the donor kit's b2_client.py. All boto3 usage is confined to this
module (enforced by tests/test_structure.py::test_boto3_only_in_repo). B2-specific
naming (user-agent string, "B2" identifiers) has been stripped in favor of the
The runtime credential document configures the account endpoint and credentials;
Browser input can select only a named root from :data:`SOURCE_ROOTS`; it can
never supply an arbitrary provider, endpoint, or bucket.  Every Case Bible root
is read-only. Workbench staging writes remain fixed to ``nexus``.
"""

from __future__ import annotations

import io
import logging
import mimetypes
import re
from typing import IO

import boto3  # noqa: F401  (tests patch the SDK through this module; clients are built in object_store_factory)
from botocore.exceptions import ClientError
from functools import lru_cache

from app.config import settings
from app.repo import object_store_factory
from app.types.source_roots import SourceRoot, configured_object_stores, configured_source_roots

logger = logging.getLogger(__name__)

CASEBIBLE_SORTED_BUCKET = "casebible-sorted"
CASEBIBLE_SORTED_PREFIX = ""
STAGING_BUCKET = "nexus"
MAX_SOURCE_KEY_LENGTH = 1024
_SAFE_SOURCE_KEY = re.compile(r"^[^\x00\r\n\\]+$")


def get_casebible_r2_config_path() -> str:
    """Return the runtime secret path; settings integration may replace this accessor."""
    return str(getattr(settings, "casebible_r2_config_path", "")).strip()


SOURCE_ROOTS: dict[str, SourceRoot] = configured_source_roots()
# The first configured root is the browser default. The fixed Case Bible Sorted
# helpers below name their bucket explicitly and keep their own root id.
DEFAULT_SOURCE_ROOT_ID = next(iter(SOURCE_ROOTS))
CASEBIBLE_SORTED_ROOT_ID = "r2-sorted"


@lru_cache(maxsize=1)
def get_r2_client():
    """The store fixed Workbench staging (``nexus``) and Case Bible Sorted live in."""
    stores = configured_object_stores(legacy_r2_path=get_casebible_r2_config_path())
    return object_store_factory.build_store_client(str(stores.get("r2", "")).strip(), "r2")


@lru_cache(maxsize=8)
def _other_store_client(scheme: str):
    stores = configured_object_stores(legacy_r2_path=get_casebible_r2_config_path())
    return object_store_factory.build_store_client(str(stores.get(scheme, "")).strip(), scheme)


def get_store_client(scheme: str):
    """The shared S3-compatible client for one configured object store."""
    return get_r2_client() if scheme == "r2" else _other_store_client(scheme)


def get_casebible_sorted_client():
    """Return the shared client used for fixed Case Bible Sorted reads."""
    return get_r2_client()


def get_source_root(root_id: str) -> SourceRoot:
    """Resolve a browser root through the code-owned allowlist."""
    try:
        return SOURCE_ROOTS[root_id]
    except KeyError:
        raise ValueError("unknown or unavailable source root") from None


def validate_source_key(key: str, *, allow_empty: bool = False) -> str:
    """Validate a relative object key without allowing a root escape."""
    normalized = key.strip()
    if allow_empty and not normalized:
        return ""
    if (
        not normalized
        or len(normalized) > MAX_SOURCE_KEY_LENGTH
        or normalized.startswith("/")
        or not _SAFE_SOURCE_KEY.fullmatch(normalized)
        or ".." in normalized.split("/")
    ):
        raise ValueError("invalid source object key")
    return normalized


def list_source_objects(
    *,
    root_id: str,
    prefix: str = "",
    continuation_token: str | None = None,
    start_after: str | None = None,
    max_keys: int = 100,
    delimiter: str | None = "/",
) -> dict:
    """List one page from an allowlisted read-only source root."""
    root = get_source_root(root_id)
    validated_prefix = validate_source_key(prefix, allow_empty=True)
    if validated_prefix and prefix.endswith("/") and not validated_prefix.endswith("/"):
        validated_prefix += "/"
    request: dict[str, object] = {
        "Bucket": root.bucket,
        "Prefix": root.key_prefix + validated_prefix,
        "MaxKeys": max_keys,
    }
    if delimiter is not None:
        request["Delimiter"] = delimiter
    if continuation_token:
        request["ContinuationToken"] = continuation_token
    if start_after:
        request["StartAfter"] = root.key_prefix + validate_source_key(start_after)
    try:
        page = get_store_client(root.scheme).list_objects_v2(**request)
    except ClientError as error:
        raise RuntimeError(f"{root.label} source listing failed") from error
    return _relative_page(root, page)


def _relative_page(root: SourceRoot, page: dict) -> dict:
    """Return the page with keys relative to the root's fixed prefix."""
    if not root.key_prefix:
        return page
    cut = len(root.key_prefix)
    relative = dict(page)
    relative["Contents"] = [
        {**row, "Key": str(row["Key"])[cut:]}
        for row in page.get("Contents", [])
        if str(row.get("Key", "")).startswith(root.key_prefix) and len(str(row["Key"])) > cut
    ]
    relative["CommonPrefixes"] = [
        {**row, "Prefix": str(row["Prefix"])[cut:]}
        for row in page.get("CommonPrefixes", [])
        if str(row.get("Prefix", "")).startswith(root.key_prefix) and len(str(row["Prefix"])) > cut
    ]
    return relative


def head_source_object(root_id: str, key: str) -> dict:
    root = get_source_root(root_id)
    validated = validate_source_key(key)
    try:
        return get_store_client(root.scheme).head_object(Bucket=root.bucket, Key=root.key_prefix + validated)
    except ClientError as error:
        raise RuntimeError(f"{root.label} source inspection failed") from error


def open_source_object(
    root_id: str,
    key: str,
    *,
    if_match: str | None = None,
    byte_range: str | None = None,
) -> dict:
    root = get_source_root(root_id)
    request: dict[str, str] = {"Bucket": root.bucket, "Key": root.key_prefix + validate_source_key(key)}
    if if_match:
        request["IfMatch"] = if_match
    if byte_range:
        request["Range"] = byte_range
    try:
        return get_store_client(root.scheme).get_object(**request)
    except ClientError as error:
        raise RuntimeError(f"{root.label} source read failed") from error


def list_casebible_sorted_objects(
    *, prefix: str = "", continuation_token: str | None = None, max_keys: int = 100
) -> dict:
    """List one delimiter-bounded page from the fixed Case Bible Sorted bucket."""
    return list_source_objects(
        root_id=CASEBIBLE_SORTED_ROOT_ID,
        prefix=prefix,
        continuation_token=continuation_token,
        max_keys=max_keys,
    )


def validate_casebible_sorted_key(key: str) -> str:
    """Validate one browser-supplied coordinate inside the fixed source bucket."""
    try:
        return validate_source_key(key)
    except ValueError:
        raise ValueError("invalid Case Bible Sorted object key") from None


def head_casebible_sorted_object(key: str) -> dict:
    """Read immutable-object coordinates from the fixed source bucket."""
    return head_source_object(CASEBIBLE_SORTED_ROOT_ID, validate_casebible_sorted_key(key))


def open_casebible_sorted_object(
    key: str,
    *,
    if_match: str | None = None,
    byte_range: str | None = None,
) -> dict:
    """Open a source stream without allowing the caller to choose storage scope."""
    return open_source_object(
        CASEBIBLE_SORTED_ROOT_ID,
        validate_casebible_sorted_key(key),
        if_match=if_match,
        byte_range=byte_range,
    )


def get_client():
    """Compatibility accessor for the fixed Nexus staging client."""
    return get_r2_client()


def check_connectivity() -> bool:
    """Prove that both fixed buckets are reachable with the runtime credential."""
    try:
        client = get_r2_client()
        client.head_bucket(Bucket=CASEBIBLE_SORTED_BUCKET)
        client.head_bucket(Bucket=STAGING_BUCKET)
        return True
    except Exception:
        logger.warning("Object store connectivity check failed", exc_info=True)
        return False


def put_object(key: str, data: bytes | IO[bytes], content_type: str | None = None) -> None:
    """Upload bytes or a file-like object to `key`. Raises RuntimeError on failure."""
    body = data if hasattr(data, "read") else io.BytesIO(data)  # type: ignore[arg-type]
    guessed_type = content_type or mimetypes.guess_type(key)[0] or "application/octet-stream"
    try:
        get_client().put_object(
            Bucket=STAGING_BUCKET,
            Key=key,
            Body=body,
            ContentType=guessed_type,
        )
    except ClientError as e:
        raise RuntimeError(f"Object store upload failed for '{key}': {e}") from e


def get_object(key: str) -> bytes:
    """Download and return the raw bytes stored at `key`. Raises RuntimeError on failure."""
    try:
        response = get_client().get_object(Bucket=STAGING_BUCKET, Key=key)
        return response["Body"].read()
    except ClientError as e:
        raise RuntimeError(f"Object store download failed for '{key}': {e}") from e


def object_exists(key: str) -> bool:
    """Check whether `key` exists in the bucket. Re-raises on non-404 errors."""
    try:
        get_client().head_object(Bucket=STAGING_BUCKET, Key=key)
        return True
    except ClientError as e:
        code = e.response.get("Error", {}).get("Code", "")
        if code in ("404", "NoSuchKey"):
            return False
        raise


def presigned_get(key: str, expires: int = 600) -> str:
    """Generate a presigned GET URL for `key`, valid for `expires` seconds."""
    try:
        return get_client().generate_presigned_url(
            "get_object",
            Params={"Bucket": STAGING_BUCKET, "Key": key},
            ExpiresIn=expires,
        )
    except ClientError as e:
        raise RuntimeError(f"Object store presign failed for '{key}': {e}") from e
