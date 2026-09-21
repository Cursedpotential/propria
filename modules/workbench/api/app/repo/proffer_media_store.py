"""Generic bucket/key reads for an already-resolved, allowlisted media object.

Byline: Claude Code · Sonnet 5 · 2026-09-21.

Callers here already hold a fully resolved (scheme, bucket, key) coordinate
that app.service.proffer_media validated against app.types.source_roots
before this module is touched, so unlike object_store_client's browser
helpers, no root_id -> bucket/prefix translation happens here. boto3/botocore
usage stays confined to repo/ (tests/test_structure.py::test_boto3_only_in_repo).
"""

from __future__ import annotations

from botocore.exceptions import ClientError

from app.repo.object_store_client import get_store_client


def list_media_objects(scheme: str, bucket: str, prefix: str, *, max_keys: int = 8) -> list[dict]:
    """List up to `max_keys` objects under `prefix` in one configured store."""
    try:
        page = get_store_client(scheme).list_objects_v2(Bucket=bucket, Prefix=prefix, MaxKeys=max_keys)
    except ClientError as error:
        raise RuntimeError("media listing failed") from error
    return list(page.get("Contents", []))


def open_media_object(scheme: str, bucket: str, key: str, *, byte_range: str | None = None) -> dict:
    """Open a GetObject stream, optionally honouring a single S3 byte range."""
    request: dict[str, str] = {"Bucket": bucket, "Key": key}
    if byte_range:
        request["Range"] = byte_range
    try:
        return get_store_client(scheme).get_object(**request)
    except ClientError as error:
        code = error.response.get("Error", {}).get("Code", "")
        if code in ("404", "NoSuchKey"):
            raise FileNotFoundError(key) from error
        if code == "InvalidRange":
            raise ValueError("unsatisfiable range") from error
        raise RuntimeError("media read failed") from error
