"""Allowlisted Case Bible source-browser contract tests.

Byline: Codex · GPT-5 · 2026-08-29.
Byline amendment: Codex · GPT-6.1-Sol · 2026-10-05 — explicit active B2 fixtures, historical R2 reads.
"""

from __future__ import annotations

import hashlib
import io
import json
from datetime import UTC, datetime

import pytest
from app.repo import object_store_client
from app.runtime import source_inspection as source_runtime
from app.service import proffer, proffer_sources, source_inspection
from app.types.source_context import SourceContextReceipt
from app.types.source_roots import parse_source_roots
from fastapi import FastAPI
from fastapi.testclient import TestClient


@pytest.fixture(autouse=True)
def configured_b2_root(monkeypatch):
    """Use one explicit active B2 root while retaining provider-boundary mocks."""
    raw = '[{"id":"b2-vault","label":"B2 Casevault","url":"b2://salem-data/consignatio/casevault/"}]'
    roots = parse_source_roots(raw)
    monkeypatch.setenv("SOURCE_ROOTS_JSON", raw)
    monkeypatch.setattr(object_store_client, "SOURCE_ROOTS", roots)
    monkeypatch.setattr(proffer_sources, "SOURCE_ROOTS", roots)


def test_browser_lists_selected_root_with_delimiter_and_file_details(monkeypatch) -> None:
    captured = {}

    def fake_list(**kwargs):
        captured.update(kwargs)
        return {
            "IsTruncated": True,
            "NextContinuationToken": "opaque-next",
            "CommonPrefixes": [{"Prefix": "exports/messages/"}, {"Prefix": "exports/photos/"}],
            "Contents": [
                {
                    "Key": "exports/messages/thread.json",
                    "Size": 41,
                    "ETag": '"not-a-sha256"',
                    "LastModified": datetime(2026, 8, 29, tzinfo=UTC),
                },
                {"Key": "exports/photos/photo.jpg", "Size": 99},
            ],
        }

    monkeypatch.setattr(proffer_sources, "list_source_objects", fake_list)
    result = proffer.browse_sources(
        mode="LIVE", root_id="b2-vault", prefix="exports/", continuation_token="opaque-current", page_size=25
    )

    assert captured == {
        "root_id": "b2-vault",
        "prefix": "exports/",
        "continuation_token": "opaque-current",
        "max_keys": 25,
    }
    assert result.source == "salem-data" and result.active_root_id == "b2-vault"
    assert result.delimiter == "/"
    assert result.filter == "" and result.filter_applied is False
    assert result.is_truncated is True and result.continuation_token == "opaque-next"
    assert [item.prefix for item in result.prefixes] == ["exports/messages/", "exports/photos/"]
    assert [item.key for item in result.objects] == ["exports/messages/thread.json", "exports/photos/photo.jpg"]
    assert result.objects[0].source_ref == "b2://salem-data/consignatio/casevault/exports/messages/thread.json"
    assert result.objects[0].file_kind == "structured_data"
    assert {root.root_id for root in result.available_roots} == {"b2-vault"}
    assert not hasattr(result.objects[0], "sha256"), "remote listings must not claim an acquisition digest"


def test_root_search_scans_beyond_the_first_provider_page_and_filters_type(monkeypatch) -> None:
    calls = []

    def fake_list(**kwargs):
        calls.append(kwargs)
        if len(calls) == 1:
            return {
                "IsTruncated": True,
                "NextContinuationToken": "provider-page-2",
                "Contents": [{"Key": "unrelated/photo.jpg", "Size": 1}],
            }
        return {
            "IsTruncated": False,
            "Contents": [{"Key": "AI_Chats/export.zip", "Size": 14_794_205}],
        }

    monkeypatch.setattr(proffer_sources, "list_source_objects", fake_list)
    result = proffer.browse_sources(
        mode="LIVE",
        root_id="b2-vault",
        prefix="ignored-while-root-searching/",
        filter_text="AI_Chats",
        filter_scope="root",
        file_types=("archive",),
        page_size=25,
    )

    assert len(calls) == 2
    assert calls[0]["prefix"] == "" and calls[0]["delimiter"] is None
    assert calls[1]["continuation_token"] == "provider-page-2"
    assert [item.key for item in result.objects] == ["AI_Chats/export.zip"]
    assert result.objects[0].archive_format == "zip"
    assert result.search_complete is True and result.scanned_count == 2


def test_object_store_client_never_accepts_a_bucket_from_the_browser(monkeypatch) -> None:
    class Client:
        def list_objects_v2(self, **kwargs):
            assert kwargs["Bucket"] == "salem-data"
            assert kwargs["Prefix"] == "consignatio/casevault/"
            assert kwargs["Delimiter"] == "/"
            return {"Contents": [], "CommonPrefixes": []}

    monkeypatch.setattr(object_store_client, "get_store_client", lambda scheme: Client() if scheme == "b2" else pytest.fail("wrong provider"))
    object_store_client.list_source_objects(root_id="b2-vault", prefix="", max_keys=10)


def test_explicit_b2_credentials_have_no_browser_configurable_bucket(monkeypatch, tmp_path) -> None:
    secret_path = tmp_path / "active-b2.json"
    secret_path.write_text(
        json.dumps(
            {
                "endpoint_url": "https://s3.example.backblazeb2.com",
                "region": "auto",
                "access_key_id": "test-ak",
                "secret_access_key": "test-sk",
            }
        ),
        encoding="utf-8",
    )
    captured = {}

    def fake_client(service, **kwargs):
        captured.update(service=service, **kwargs)
        return object()

    monkeypatch.setattr(object_store_client, "configured_object_stores", lambda **_: {"b2": str(secret_path)})
    monkeypatch.setattr(object_store_client.boto3, "client", fake_client)
    object_store_client._other_store_client.cache_clear()
    object_store_client.get_store_client("b2")

    assert captured["service"] == "s3"
    assert captured["endpoint_url"] == "https://s3.example.backblazeb2.com"
    assert "bucket" not in captured
    object_store_client._other_store_client.cache_clear()


def test_legacy_r2_credential_fallback_rejects_before_sdk(monkeypatch):
    monkeypatch.delenv("OBJECT_STORES_JSON", raising=False)
    monkeypatch.setattr(object_store_client, "get_casebible_r2_config_path", lambda: "/fixture/retired-r2.json")
    monkeypatch.setattr(object_store_client.boto3, "client", lambda *_args, **_kwargs: pytest.fail("SDK must not run"))
    object_store_client.get_r2_client.cache_clear()
    try:
        with pytest.raises(ValueError, match="legacy R2 credential fallback is retired"):
            object_store_client.get_r2_client()
    finally:
        object_store_client.get_r2_client.cache_clear()


def test_historical_staged_reads_preserve_fixed_nexus_identity_without_new_writes(monkeypatch):
    calls = []

    class Client:
        def get_object(self, **kwargs):
            calls.append(("get", kwargs))
            return {"Body": io.BytesIO(b"source")}

        def head_object(self, **kwargs):
            calls.append(("exists", kwargs))

        def generate_presigned_url(self, operation, **kwargs):
            calls.append((operation, kwargs))
            return "https://example.invalid/object"

        def put_object(self, **kwargs):
            pytest.fail("historical read must never write")

    monkeypatch.setattr(object_store_client, "get_client", lambda: Client())
    key = "workbench/staging/sha/source.md"
    assert object_store_client.get_object(key) == b"source"
    assert object_store_client.object_exists(key) is True
    assert object_store_client.presigned_get(key)
    assert calls[0][1] == {"Bucket": "nexus", "Key": key}
    assert calls[1][1] == {"Bucket": "nexus", "Key": key}
    assert calls[2][1]["Params"] == {"Bucket": "nexus", "Key": key}


def test_browser_rejects_escape_prefix_before_object_store_call(monkeypatch) -> None:
    monkeypatch.setattr(
        proffer_sources,
        "list_source_objects",
        lambda **_: (_ for _ in ()).throw(AssertionError("object store must not be called")),
    )
    try:
        proffer.browse_sources(mode="LIVE", prefix="../wrong-case/")
    except proffer.ProfferError as error:
        assert error.status_code == 422
    else:
        raise AssertionError("escaping prefix accepted")


def test_source_inspection_hashes_immediately_without_claiming_a_custody_digest(monkeypatch) -> None:
    payload = b"%PDF-1.7\nsmall source preview"
    modified = datetime(2026, 8, 30, tzinfo=UTC)
    monkeypatch.setattr(
        source_inspection,
        "head_source_object",
        lambda root_id, key: {
            "ContentLength": len(payload),
            "ETag": '"object-etag"',
            "ContentType": "application/pdf",
            "LastModified": modified,
        },
    )
    monkeypatch.setattr(
        source_inspection,
        "open_source_object",
        lambda root_id, key, **kwargs: {"Body": io.BytesIO(payload), "ETag": '"object-etag"', "ContentLength": len(payload)},
    )
    app = FastAPI()
    app.include_router(source_runtime.router)

    response = TestClient(app).post(
        "/api/proffer/source-inspection?mode=LIVE&root_id=b2-vault",
        json={
            "root_id": "b2-vault",
            "source_ref": "b2://salem-data/consignatio/casevault/filings/source.pdf",
            "key": "filings/source.pdf",
            "expected_byte_length": len(payload),
            "expected_etag": '"object-etag"',
        },
    )

    assert response.status_code == 200
    body = response.json()
    assert body["sha256"] == hashlib.sha256(payload).hexdigest()
    assert body["digest_status"] == "preview_only"
    assert body["matter_mode"] == "LIVE"
    assert body["active_root_id"] == "b2-vault"
    assert body["preview_kind"] == "pdf"
    assert body["preview_url"].startswith("/api/proffer/source-content?")
    assert body["parser_preflight"] == {
        "declared_format": "pdf",
        "route_label": "PDF document route",
        "basis": "filename_extension",
        "authoritative": False,
    }


@pytest.mark.parametrize("key,content_type,kind", [
    ("source.pdf", "application/octet-stream", "pdf"),
    ("photo.png", "application/octet-stream", "image"),
    ("recording.mp3", "audio/mpeg", "audio"),
    ("clip.mp4", "video/mp4", "video"),
    ("export.jsonl", "application/octet-stream", "text"),
])
def test_selection_preview_reads_only_metadata_even_for_large_files(monkeypatch, key, content_type, kind) -> None:
    """A file selection above the old hash limit returns a reader without GET bytes.

    Inputs: allowlisted B2 listing and mocked HEAD. Output: reader descriptor.
    Effects: no source read or write; legacy POST checksum has its own test.
    """
    size = source_inspection.MAX_IMMEDIATE_HASH_BYTES + 1
    monkeypatch.setattr(source_inspection, "head_source_object", lambda *_: {
        "ContentLength": size, "ETag": '"etag"', "ContentType": content_type,
    })
    monkeypatch.setattr(source_inspection, "open_source_object", lambda *_args, **_kwargs: pytest.fail("selection must not read source bytes"))
    app = FastAPI()
    app.include_router(source_runtime.router)
    response = TestClient(app).get("/api/proffer/source-inspection", params={
        "root_id": "b2-vault", "key": key,
        "source_ref": f"b2://salem-data/consignatio/casevault/{key}",
        "expected_byte_length": size, "expected_etag": '"etag"',
    })
    assert response.status_code == 200
    body = response.json()
    assert body["sha256"] is None and body["digest_status"] == "not_computed"
    assert body["source_location"] == "b2" and body["preview_kind"] == kind
    assert body["preview_text"] == "" and body["matter_mode"] == "LIVE"
    assert "etag=%22etag%22" in body["preview_url"]


def test_selection_preview_rejects_changed_listing_before_opening_reader(monkeypatch) -> None:
    """Reject stale selection identity without reading or hashing any source bytes."""
    monkeypatch.setattr(source_inspection, "head_source_object", lambda *_: {"ContentLength": 10, "ETag": '"new"'})
    monkeypatch.setattr(source_inspection, "open_source_object", lambda *_args, **_kwargs: pytest.fail("stale selection must not read bytes"))
    app = FastAPI()
    app.include_router(source_runtime.router)
    response = TestClient(app).get("/api/proffer/source-inspection", params={
        "root_id": "b2-vault", "key": "source.pdf",
        "source_ref": "b2://salem-data/consignatio/casevault/source.pdf",
        "expected_byte_length": 10, "expected_etag": '"old"',
    })
    assert response.status_code == 409


def test_selection_preview_rejects_a_locator_outside_the_selected_root(monkeypatch) -> None:
    """A mismatched source locator is rejected before provider I/O."""
    monkeypatch.setattr(source_inspection, "head_source_object", lambda *_: pytest.fail("wrong root must not reach provider"))
    app = FastAPI()
    app.include_router(source_runtime.router)
    response = TestClient(app).get("/api/proffer/source-inspection", params={
        "root_id": "b2-vault", "key": "source.pdf",
        "source_ref": "b2://another-bucket/source.pdf", "expected_byte_length": 10,
    })
    assert response.status_code == 422


def test_image_extensions_share_one_preview_and_preflight_classification() -> None:
    expected_extensions = {".avif", ".bmp", ".gif", ".jpeg", ".jpg", ".png", ".tif", ".tiff", ".webp"}

    assert source_inspection._IMAGE_EXTENSIONS == expected_extensions
    for extension in expected_extensions:
        preflight = source_inspection._preflight(f"photos/source{extension}")
        assert preflight.declared_format == "image"
        assert preflight.route_label == "Image processing route"
        assert preflight.basis == "filename_extension"
        assert preflight.authoritative is False


def test_source_inspection_rejects_a_changed_listing_identity(monkeypatch) -> None:
    monkeypatch.setattr(
        source_inspection,
        "head_source_object",
        lambda root_id, key: {"ContentLength": 12, "ETag": '"new-etag"'},
    )
    app = FastAPI()
    app.include_router(source_runtime.router)

    response = TestClient(app).post(
        "/api/proffer/source-inspection?mode=LIVE&root_id=b2-vault",
        json={
            "root_id": "b2-vault",
            "source_ref": "b2://salem-data/consignatio/casevault/source.pdf",
            "key": "source.pdf",
            "expected_byte_length": 11,
            "expected_etag": '"old-etag"',
        },
    )

    assert response.status_code == 409
    assert "choose it again" in response.json()["detail"]


def test_source_content_is_same_origin_etag_pinned_and_range_bounded(monkeypatch) -> None:
    payload = b"0123456789"
    captured = {}
    monkeypatch.setattr(
        source_inspection,
        "head_source_object",
        lambda root_id, key: {"ContentLength": len(payload), "ETag": '"etag"', "ContentType": "application/octet-stream"},
    )

    def fake_open(root_id, key, **kwargs):
        captured.update(kwargs)
        return {"Body": io.BytesIO(payload[2:6]), "ETag": '"etag"', "ContentLength": 4, "ContentRange": "bytes 2-5/10"}

    monkeypatch.setattr(source_inspection, "open_source_object", fake_open)
    app = FastAPI()
    app.include_router(source_runtime.router)

    response = TestClient(app).get(
        "/api/proffer/source-content",
        params={"root_id": "b2-vault", "key": "source.pdf", "etag": '"etag"', "download": "true"},
        headers={"Range": "bytes=2-5"},
    )

    assert response.status_code == 206
    assert response.content == b"2345"
    assert response.headers["content-range"] == "bytes 2-5/10"
    assert response.headers["accept-ranges"] == "bytes"
    assert response.headers["content-type"] == "application/pdf"
    assert response.headers["content-disposition"] == "attachment; filename*=UTF-8''source.pdf"
    assert captured == {"if_match": '"etag"', "byte_range": "bytes=2-5", "version_id": None}


def test_source_context_route_uses_authenticated_actor_and_returns_only_receipt(monkeypatch) -> None:
    captured = {}

    async def fake_create(body, actor, *, mode):
        captured.update(body=body, actor=actor, mode=mode)
        return SourceContextReceipt(
            source_context_ref="33333333-3333-3333-3333-333333333333",
            receipt_ref="proffer-source-context://33333333-3333-3333-3333-333333333333",
            content_digest="a" * 64,
            revision=1,
            recorded_at=datetime(2026, 8, 30, tzinfo=UTC),
            matter_mode="LIVE",
        )

    monkeypatch.setattr(source_runtime, "create_source_context", fake_create)
    app = FastAPI()

    @app.middleware("http")
    async def identity(request, call_next):
        request.state.subject_uid = "authentik-user-1"
        request.state.principal = "operator"
        return await call_next(request)

    app.include_router(source_runtime.router)
    response = TestClient(app).post(
        "/api/proffer/source-contexts?mode=LIVE",
        json={
            "request_id": "request-1",
            "matter_id": "11111111-1111-4111-8111-111111111111",
            "court_case_id": "22222222-2222-4222-8222-222222222222",
            "source_ref": "b2://salem-data/consignatio/casevault/source.pdf",
            "observed_source": {
                "key": "source.pdf",
                "name": "source.pdf",
                "byte_length": 10,
                "etag": '"etag"',
                "preview_sha256": "b" * 64,
            },
            "assertions": {"source_class": "acquired_third_party", "other_party": "Other party"},
            "change_reason": "Operator supplied source context during intake",
            "matter_mode": "LIVE",
        },
    )

    assert response.status_code == 201
    assert response.json()["source_context_ref"] == "33333333-3333-3333-3333-333333333333"
    assert captured["actor"].subject_uid == "authentik-user-1"
    assert captured["actor"].username == "operator"
    assert captured["body"].assertions.other_party == "Other party"
    assert captured["mode"] == "LIVE"


@pytest.mark.parametrize("field,bad", [("ETag", '"different"'), ("VersionId", "wrong-version"), ("ContentLength", 10), ("ContentRange", "bytes 0-3/10")])
def test_reader_rejects_provider_get_identity_and_range_mismatch(monkeypatch, field, bad):
    """No bytes escape when provider GET disagrees with the exact inspected version.

    Inputs: each malformed GET header. Output: 502 and a closed body. Effects:
    memory-only fixtures; choose to prevent false success from fabricated headers.
    """
    body = io.BytesIO(b"2345")
    monkeypatch.setattr(source_inspection, "head_source_object", lambda *args, **kwargs: {"ContentLength": 10, "ETag": '"etag"', "VersionId": "version-a"})
    response = {"Body": body, "ETag": '"etag"', "VersionId": "version-a", "ContentLength": 4, "ContentRange": "bytes 2-5/10", field: bad}
    monkeypatch.setattr(source_inspection, "open_source_object", lambda *args, **kwargs: response)
    with pytest.raises(source_inspection.SourceInspectionError, match="did not confirm"):
        source_inspection.open_source_content("b2-vault", "photo.jpg", '"etag"', "bytes=2-5", version_id="version-a", expected_byte_length=10)
    assert body.closed


def test_reader_pins_head_and_get_to_the_same_version_and_preserves_original_bytes(monkeypatch):
    """Reader metadata and original links share the exact provider version and bytes.

    Input: arbitrary bytes including non-decodable media. Output: unchanged 206.
    Effects: mocked reads only; choose to separate byte delivery from browser decoding.
    """
    payload = b"\x00\xff\xfe\x80original"
    calls = []
    def head(root, key, **kwargs):
        calls.append(("HEAD", kwargs))
        return {"ContentLength": len(payload), "ETag": '"etag"', "VersionId": "exact-version", "ContentType": "image/jpeg"}
    def read(root, key, **kwargs):
        calls.append(("GET", kwargs))
        return {"Body": io.BytesIO(payload[1:5]), "ContentLength": 4, "ETag": '"etag"', "VersionId": "exact-version", "ContentRange": f"bytes 1-4/{len(payload)}"}
    monkeypatch.setattr(source_inspection, "head_source_object", head)
    monkeypatch.setattr(source_inspection, "open_source_object", read)
    app = FastAPI()
    app.include_router(source_runtime.router)
    response = TestClient(app).get("/api/proffer/source-content", params={"root_id": "b2-vault", "key": "photo.jpg", "etag": '"etag"', "version_id": "exact-version", "expected_byte_length": len(payload)}, headers={"Range": "bytes=1-4"})
    assert response.status_code == 206 and response.content == payload[1:5]
    assert response.headers["x-source-version-id"] == "exact-version"
    assert all(kwargs["version_id"] == "exact-version" for _, kwargs in calls)
    assert calls[-1][1]["if_match"] == '"etag"'


@pytest.mark.parametrize("range_header,expected", [("bytes=0-", "bytes 0-9/10"), ("bytes=-3", "bytes 7-9/10"), ("bytes=7-99", "bytes 7-9/10")])
def test_range_normalization_is_exact(range_header, expected):
    """Normalize open, suffix and clipped ranges to the provider's exact byte interval.

    Input: supported Range syntax. Output: exact Content-Range. Effects: none;
    choose to protect browser media seeking and bounded original inspection.
    """
    assert source_inspection._range_header(range_header, 10)[2] == expected


def test_reader_rejects_missing_expected_version_before_get(monkeypatch):
    """A supplied immutable version cannot silently fall back to the latest object.

    Input: HEAD lacking the requested version. Output: 409 before GET. Effects:
    mocked metadata only; choose for Back/retry preservation of original identity.
    """
    monkeypatch.setattr(source_inspection, "head_source_object", lambda *args, **kwargs: {"ContentLength": 10, "ETag": '"etag"'})
    monkeypatch.setattr(source_inspection, "open_source_object", lambda *args, **kwargs: pytest.fail("GET must not run"))
    with pytest.raises(source_inspection.SourceInspectionError) as error:
        source_inspection.open_source_content("b2-vault", "photo.jpg", '"etag"', None, version_id="exact-version")
    assert error.value.status_code == 409


@pytest.mark.parametrize("payload", [b"short", b"too many bytes"])
def test_reader_rejects_body_length_disagreement_and_closes(payload):
    """Detect truncated or overlong streams without changing their source bytes.

    Input: body disagreeing with validated headers. Output: stream error and
    closed reader. Effects: memory-only reads; choose after header validation.
    """
    body = io.BytesIO(payload)
    content = source_inspection.SourceContent(body, "image/jpeg", 10, '"etag"', 200)
    with pytest.raises(source_inspection.SourceInspectionError):
        list(source_inspection.stream_source_content(content))
    assert body.closed


def test_object_store_passes_version_to_both_provider_reads(monkeypatch):
    """The object repo carries immutable version pins through both SDK calls.

    Input: versioned HEAD and GET. Output: unchanged provider requests. Effects:
    mocked SDK only; choose to prove pins are not dropped below the service seam.
    """
    calls = []
    class Client:
        def head_object(self, **kwargs):
            calls.append(kwargs)
            return {}
        def get_object(self, **kwargs):
            calls.append(kwargs)
            return {}
    monkeypatch.setattr(object_store_client, "get_store_client", lambda _: Client())
    object_store_client.head_source_object("b2-vault", "photo.jpg", version_id="exact-version")
    object_store_client.open_source_object("b2-vault", "photo.jpg", if_match='"etag"', byte_range="bytes=0-3", version_id="exact-version")
    assert all(call["VersionId"] == "exact-version" for call in calls)
    assert calls[1]["IfMatch"] == '"etag"' and calls[1]["Range"] == "bytes=0-3"


def test_legacy_version_locator_is_authorized_without_rewriting_or_broadening_root():
    """The existing starter/context locator validator admits one exact provider version.

    Input: encoded versioned B2 source URI. Output: unchanged locator; wrong
    roots fail. Effects: reads fixture configuration only, never starts processing.
    """
    from app.types.source_roots import validate_authorized_source_ref
    locator = "b2://salem-data/consignatio/casevault/source.pdf?versionId=retained%2B%2Fversion"
    assert validate_authorized_source_ref(locator) == locator
    with pytest.raises(ValueError):
        validate_authorized_source_ref(locator.replace("salem-data", "unapproved-bucket"))
