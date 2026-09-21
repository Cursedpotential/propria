"""Contract tests for GET /api/proffer/previews/{preview_handle}/media/{sha256}.

Byline: Claude Code · Sonnet 5 · 2026-09-21.

No live network: the object store is a FakeClient that ignores the requested
Prefix and returns every stored key, so tests that expect refusal are proving
this module's OWN defensive filtering, not the store's Prefix behaviour.
"""

from __future__ import annotations

import hashlib
import io
import json

import pytest
from botocore.exceptions import ClientError
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.repo import proffer_media_store
from app.runtime import proffer as proffer_runtime
from app.service import matter_mode, proffer_media
from app.service.proffer_media_prefix import derived_media_prefix
from app.types.source_roots import ROOTS_ENV

SHA_A = hashlib.sha256(b"attachment-a").hexdigest()
SHA_B = hashlib.sha256(b"attachment-b").hexdigest()
HANDLE_A = "preview_handle_media_run_a_0123456789"
HANDLE_B = "preview_handle_media_run_b_0123456789"
SOURCE_KEY_A = "consignatio/vault/v1/calls-20260912155315.xml"
SOURCE_KEY_B = "consignatio/vault/v1/sms-20260609173028.xml"
BUCKET = "salem-data"


class _Run:
    def __init__(self, source_ref: str):
        self.source_ref = source_ref


class FakeClient:
    """Fakes the boto3 S3 surface this module reads; no live network."""

    def __init__(self, objects: dict[str, bytes], content_types: dict[str, str] | None = None):
        self.objects = objects
        self.content_types = content_types or {}
        self.list_prefixes: list[str] = []

    def list_objects_v2(self, **kwargs):
        self.list_prefixes.append(kwargs.get("Prefix", ""))
        # Deliberately ignore Prefix: the service layer must filter on its own.
        return {"Contents": [{"Key": key, "Size": len(value)} for key, value in self.objects.items()]}

    def get_object(self, **kwargs):
        key = kwargs["Key"]
        if key not in self.objects:
            raise ClientError({"Error": {"Code": "NoSuchKey"}}, "GetObject")
        data = self.objects[key]
        range_header = kwargs.get("Range")
        if range_header:
            start, end = _parse_fake_range(range_header, len(data))
            if start is None:
                raise ClientError({"Error": {"Code": "InvalidRange"}}, "GetObject")
            data = data[start : end + 1]
        return {"Body": io.BytesIO(data), "ContentType": self.content_types.get(key, "application/octet-stream")}


def _parse_fake_range(header: str, size: int):
    spec = header.removeprefix("bytes=")
    start_text, _, end_text = spec.partition("-")
    start, end = int(start_text), int(end_text) if end_text else size - 1
    if start < 0 or end < start or start >= size:
        return None, None
    return start, min(end, size - 1)


def _app() -> TestClient:
    app = FastAPI()
    app.include_router(proffer_runtime.router)
    return TestClient(app)


@pytest.fixture(autouse=True)
def _bindings_and_root(monkeypatch):
    monkeypatch.setenv(
        ROOTS_ENV,
        json.dumps([{"id": "b2-vault", "label": "B2 Vault", "url": "b2://salem-data/consignatio/vault/v1/"}]),
    )
    matter_mode._clear_preview_modes_for_tests()
    matter_mode.bind_preview_mode(HANDLE_A, "TEST")
    matter_mode.bind_preview_mode(HANDLE_B, "TEST")
    yield
    matter_mode._clear_preview_modes_for_tests()


def _wire_run(monkeypatch, handle: str, source_key: str):
    async def fake_operation(preview_handle: str):
        assert preview_handle == handle
        return _Run(f"b2://{BUCKET}/{source_key}")

    monkeypatch.setattr(proffer_media, "operation", fake_operation)


def _wire_store(monkeypatch, client: FakeClient):
    monkeypatch.setattr(proffer_media_store, "get_store_client", lambda scheme: client)


def test_happy_path_streams_bytes_with_correct_headers(monkeypatch) -> None:
    _wire_run(monkeypatch, HANDLE_A, SOURCE_KEY_A)
    prefix = derived_media_prefix(SOURCE_KEY_A)
    data = b"\xff\xd8\xff-not-a-real-jpeg-but-bytes"
    _wire_store(monkeypatch, FakeClient({f"{prefix}{SHA_A}.jpg": data}))

    response = _app().get(f"/api/proffer/previews/{HANDLE_A}/media/{SHA_A}", params={"mode": "TEST"})

    assert response.status_code == 200
    assert response.content == data
    assert response.headers["content-type"] == "image/jpeg"
    assert response.headers["content-length"] == str(len(data))
    assert response.headers["cache-control"] == "private, max-age=31536000, immutable"
    assert response.headers["x-content-type-options"] == "nosniff"
    assert response.headers["accept-ranges"] == "bytes"
    assert response.headers["content-disposition"] == f'inline; filename="{SHA_A}.jpg"'
    assert "content-range" not in response.headers


def test_range_request_returns_206_with_correct_slice(monkeypatch) -> None:
    _wire_run(monkeypatch, HANDLE_A, SOURCE_KEY_A)
    prefix = derived_media_prefix(SOURCE_KEY_A)
    data = b"0123456789abcdef"
    _wire_store(monkeypatch, FakeClient({f"{prefix}{SHA_A}.mp4": data}))

    response = _app().get(
        f"/api/proffer/previews/{HANDLE_A}/media/{SHA_A}",
        params={"mode": "TEST"},
        headers={"Range": "bytes=2-5"},
    )

    assert response.status_code == 206
    assert response.content == data[2:6]
    assert response.headers["content-range"] == f"bytes 2-5/{len(data)}"
    assert response.headers["content-length"] == "4"
    assert response.headers["content-type"] == "video/mp4"
    assert response.headers["content-disposition"] == f'inline; filename="{SHA_A}.mp4"'


def test_malformed_range_is_refused_with_416(monkeypatch) -> None:
    _wire_run(monkeypatch, HANDLE_A, SOURCE_KEY_A)
    prefix = derived_media_prefix(SOURCE_KEY_A)
    data = b"short"
    _wire_store(monkeypatch, FakeClient({f"{prefix}{SHA_A}.bin": data}))

    response = _app().get(
        f"/api/proffer/previews/{HANDLE_A}/media/{SHA_A}",
        params={"mode": "TEST"},
        headers={"Range": "bytes=999-1000"},
    )

    assert response.status_code == 416


def test_object_outside_the_allowed_prefix_is_never_selected(monkeypatch) -> None:
    """The fake store ignores Prefix; only this module's own filter protects it."""
    _wire_run(monkeypatch, HANDLE_A, SOURCE_KEY_A)
    # A decoy sitting under a completely different (still-configured) key.
    decoy_prefix = derived_media_prefix(SOURCE_KEY_B)
    _wire_store(monkeypatch, FakeClient({f"{decoy_prefix}{SHA_A}.jpg": b"someone else's photo"}))

    response = _app().get(f"/api/proffer/previews/{HANDLE_A}/media/{SHA_A}", params={"mode": "TEST"})

    assert response.status_code == 404


def test_sha256_prefix_collision_is_not_served(monkeypatch) -> None:
    """A key that merely BEGINS with the sha256 must not match (content-addressed == exact)."""
    _wire_run(monkeypatch, HANDLE_A, SOURCE_KEY_A)
    prefix = derived_media_prefix(SOURCE_KEY_A)
    _wire_store(monkeypatch, FakeClient({f"{prefix}{SHA_A}deadbeef.jpg": b"not this one"}))

    response = _app().get(f"/api/proffer/previews/{HANDLE_A}/media/{SHA_A}", params={"mode": "TEST"})

    assert response.status_code == 404


def test_unsafe_extension_is_dropped_not_reflected_into_headers(monkeypatch) -> None:
    """A key with a header-hostile 'extension' must not leak into Content-Disposition."""
    _wire_run(monkeypatch, HANDLE_A, SOURCE_KEY_A)
    prefix = derived_media_prefix(SOURCE_KEY_A)
    hostile_key = f'{prefix}{SHA_A}."evil\r\nX-Injected: 1'
    _wire_store(monkeypatch, FakeClient({hostile_key: b"payload"}))

    response = _app().get(f"/api/proffer/previews/{HANDLE_A}/media/{SHA_A}", params={"mode": "TEST"})

    assert response.status_code == 200
    assert response.content == b"payload"
    assert "X-Injected" not in response.headers
    assert response.headers["content-disposition"] == f'attachment; filename="{SHA_A}"'
    assert "\r" not in response.headers["content-disposition"]


def test_media_from_a_different_run_is_refused(monkeypatch) -> None:
    """Run B's media must not be reachable through run A's preview handle."""
    _wire_run(monkeypatch, HANDLE_A, SOURCE_KEY_A)
    prefix_b = derived_media_prefix(SOURCE_KEY_B)
    _wire_store(monkeypatch, FakeClient({f"{prefix_b}{SHA_B}.jpg": b"belongs to run b"}))

    response = _app().get(f"/api/proffer/previews/{HANDLE_A}/media/{SHA_B}", params={"mode": "TEST"})

    assert response.status_code == 404


def test_unknown_sha256_is_404(monkeypatch) -> None:
    _wire_run(monkeypatch, HANDLE_A, SOURCE_KEY_A)
    _wire_store(monkeypatch, FakeClient({}))

    response = _app().get(f"/api/proffer/previews/{HANDLE_A}/media/{SHA_A}", params={"mode": "TEST"})

    assert response.status_code == 404


@pytest.mark.parametrize(
    "bad_sha",
    [
        "not-hex-not-hex-not-hex-not-hex-not-hex-not-hex-not-hex-not-he",  # 64 chars, not hex
        "ABCDEF0000000000000000000000000000000000000000000000000000000",  # uppercase, wrong length too
        "deadbeef",  # too short
        SHA_A + "0",  # 65 chars
    ],
)
def test_malformed_sha256_is_422(monkeypatch, bad_sha) -> None:
    _wire_run(monkeypatch, HANDLE_A, SOURCE_KEY_A)
    _wire_store(monkeypatch, FakeClient({}))

    response = _app().get(f"/api/proffer/previews/{HANDLE_A}/media/{bad_sha}", params={"mode": "TEST"})

    assert response.status_code == 422


class TestDerivedMediaPrefix:
    """Pure-function coverage for app.service.proffer_media_prefix."""

    def test_original_xml_gets_a_new_derived_media_sibling(self) -> None:
        assert derived_media_prefix("consignatio/vault/v1/calls-20260912155315.xml") == (
            "consignatio/vault/v1/calls-20260912155315.xml.derived/media/"
        )

    def test_derived_thread_chunk_resolves_to_the_same_media_sibling(self) -> None:
        key = "consignatio/vault/v1/sms-20260110021338.xml.derived/threads/8103533592_8103535467_self.ndjson"
        assert derived_media_prefix(key) == "consignatio/vault/v1/sms-20260110021338.xml.derived/media/"

    def test_directory_containing_xml_extension_does_not_false_anchor(self) -> None:
        key = "consignatio/vault/v1.xml-archive/sms-20260110021338.xml"
        assert derived_media_prefix(key) == (
            "consignatio/vault/v1.xml-archive/sms-20260110021338.xml.derived/media/"
        )

    def test_trailing_space_directory_is_preserved_verbatim(self) -> None:
        key = "consignatio/vault/v1/sms /sms-20260609173028.xml.derived/threads/8103919475.0001.ndjson"
        assert derived_media_prefix(key) == "consignatio/vault/v1/sms /sms-20260609173028.xml.derived/media/"

    def test_source_key_with_no_xml_extension_still_resolves_deliberately(self) -> None:
        # No ".derived/" marker is present, so the whole key is treated as the
        # original object and the marker + dirname are appended — the same
        # rule as the ".xml" case, generalized to any source format.
        assert derived_media_prefix("consignatio/vault/v1/export.json") == (
            "consignatio/vault/v1/export.json.derived/media/"
        )
