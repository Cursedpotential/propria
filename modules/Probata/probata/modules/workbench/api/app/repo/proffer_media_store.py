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


def list_objects_beside(scheme: str, bucket: str, prefix: str, *, max_keys: int = 50) -> list[dict]:
    """List objects directly under `prefix` without descending into sub-folders.

    Byline: Claude Code · Opus 5.5 · 2026-09-26 — the metadata screen's
    same-stem sidecar lookup ("IMG_1234" -> IMG_1234.jpg.json, IMG_1234.xmp, ...).
    The delimiter keeps a decoded "<key>.derived/" folder out of the answer.
    """
    try:
        page = get_store_client(scheme).list_objects_v2(Bucket=bucket, Prefix=prefix, MaxKeys=max_keys, Delimiter="/")
    except ClientError as error:
        raise RuntimeError("sidecar listing failed") from error
    return list(page.get("Contents", []))


def object_exists(scheme: str, bucket: str, key: str) -> bool:
    """HEAD one object. True when it exists, False when it does not.

    Byline: Claude Code · Opus 5 · 2026-09-22 — the cheap half of "has this
    source been decoded?", so the Sources list never reads a manifest body
    just to learn that one exists.
    """
    try:
        get_store_client(scheme).head_object(Bucket=bucket, Key=key)
    except ClientError as error:
        code = error.response.get("Error", {}).get("Code", "")
        if code in ("404", "NoSuchKey", "NotFound"):
            return False
        raise RuntimeError("media head failed") from error
    return True


def read_small_object(scheme: str, bucket: str, key: str, *, max_bytes: int) -> bytes:
    """Read one whole small object (a manifest). Refuses anything over `max_bytes`."""
    response = open_media_object(scheme, bucket, key)
    body = response["Body"]
    try:
        if int(response.get("ContentLength", 0)) > max_bytes:
            raise ValueError("object is larger than the read limit")
        data = body.read(max_bytes + 1)
    finally:
        body.close()
    if len(data) > max_bytes:
        raise ValueError("object is larger than the read limit")
    return data


def iter_object_lines(scheme: str, bucket: str, key: str):
    """Stream one object line by line (NDJSON); the object is never held whole."""
    body = open_media_object(scheme, bucket, key)["Body"]
    try:
        yield from body.iter_lines(chunk_size=256 * 1024)
    finally:
        body.close()


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
