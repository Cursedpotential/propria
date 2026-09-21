"""Contract tests for the decoded-source viewer: GET /api/proffer/decoded/*.

Byline: Claude Code · Fable 5.1 · 2026-09-21.

UNIT tests. The object store is an in-memory fake with the two boto3 calls the
repo layer uses; nothing here proves the deployed service reads B2. The live
proof is recorded in docs/planning/2026-09-20-TODO.md.
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
from app.types.source_roots import ROOTS_ENV

BUCKET = "salem-data"
# A real key shape: the folder name ends in a space.
SOURCE_KEY = "consignatio/vault/v1/sms /sms-20260609173028.xml"
SOURCE_REF = f"b2://{BUCKET}/{SOURCE_KEY}"
BASE = SOURCE_KEY + ".derived/"
THREAD_KEY = BASE + "threads/8102594380_8106100987.0001.ndjson"
PICTURE = b"\x89PNG-not-really"
PICTURE_SHA = hashlib.sha256(PICTURE).hexdigest()


class _Body(io.BytesIO):
    def iter_lines(self, chunk_size: int = 1024):
        yield from self.getvalue().splitlines()


class FakeClient:
    def __init__(self, objects: dict[str, bytes]):
        self.objects = objects

    def list_objects_v2(self, **kwargs):
        prefix = kwargs.get("Prefix", "")
        return {"Contents": [{"Key": k, "Size": len(v)} for k, v in self.objects.items() if k.startswith(prefix)]}

    def get_object(self, **kwargs):
        key = kwargs["Key"]
        if key not in self.objects:
            raise ClientError({"Error": {"Code": "NoSuchKey"}}, "GetObject")
        data = self.objects[key]
        return {"Body": _Body(data), "ContentLength": len(data), "ContentType": "application/octet-stream"}


def _lines(count: int) -> bytes:
    rows = []
    for index in range(count):
        row = {
            "thread": "8102594380_8106100987",
            "kind": "mms" if index == 1 else "sms",
            "status": "parsed",
            "occurred_at": f"2026-01-0{index + 1}T12:00:00Z",
            "sender": "8106100987",
            "participants": ["8102594380", "8106100987"],
            "content": f"message {index}",
        }
        if index == 1:
            row["attachments"] = [
                {
                    "ordinal": 0,
                    "name": "image000000.png",
                    "mime": "image/png",
                    "sha256": PICTURE_SHA,
                    "bytes": len(PICTURE),
                    "uri": f"b2://{BUCKET}/{BASE}media/{PICTURE_SHA}.png",
                }
            ]
            row["attachment_references"] = [{"name": "clip.3gp"}]
        rows.append(json.dumps(row))
    return ("\n".join(rows) + "\n").encode()


def _objects() -> dict[str, bytes]:
    manifest = {
        "schema": "smsthreads/v2",
        "source": SOURCE_REF,
        "derived_at": "2026-09-20T22:00:00Z",
        "decoder": "sbv-parseonly",
        "records": 3,
        "rejected": 0,
        "media_objects": 1,
        "media_bytes": len(PICTURE),
        "threads": [
            {
                "thread": "8102594380_8106100987",
                "chunk": 1,
                "participants": ["8102594380", "8106100987"],
                "key": THREAD_KEY,
                "records": 3,
                "first_occurred_at": "2026-01-01T12:00:00Z",
                "last_occurred_at": "2026-01-03T12:00:00Z",
            },
            # A manifest entry pointing outside the decoded prefix must never become readable.
            {"thread": "escape", "key": "consignatio/vault/v1/other.xml", "records": 1, "participants": []},
        ],
    }
    return {
        BASE + "manifest.json": json.dumps(manifest).encode(),
        THREAD_KEY: _lines(3),
        f"{BASE}media/{PICTURE_SHA}.png": PICTURE,
        "consignatio/vault/v1/other.xml": b"<secret/>",
    }


@pytest.fixture()
def client(monkeypatch) -> TestClient:
    monkeypatch.setenv(
        ROOTS_ENV, json.dumps([{"id": "b2-vault", "label": "B2 Vault", "url": "b2://salem-data/consignatio/vault/v1/"}])
    )
    fake = FakeClient(_objects())
    monkeypatch.setattr(proffer_media_store, "get_store_client", lambda scheme: fake)
    app = FastAPI()
    app.include_router(proffer_runtime.router)
    return TestClient(app)


def test_manifest_lists_only_thread_files_inside_the_decoded_prefix(client) -> None:
    response = client.get("/api/proffer/decoded/manifest", params={"source_ref": SOURCE_REF})
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["records"] == 3 and body["media_objects"] == 1
    assert [thread["file"] for thread in body["threads"]] == ["threads/8102594380_8106100987.0001.ndjson"]


def test_thread_page_is_oldest_first_and_pages_forward(client) -> None:
    params = {"source_ref": SOURCE_REF, "file": "threads/8102594380_8106100987.0001.ndjson", "limit": 2}
    first = client.get("/api/proffer/decoded/thread", params=params).json()
    assert [m["body"] for m in first["messages"]] == ["message 0", "message 1"]
    assert first["next_offset"] == 2 and first["total_records"] == 3
    picture = first["messages"][1]
    assert picture["attachments"][0]["sha256"] == PICTURE_SHA
    assert picture["missing_attachments"] == 1
    last = client.get("/api/proffer/decoded/thread", params={**params, "offset": 2}).json()
    assert [m["ordinal"] for m in last["messages"]] == [2] and last["next_offset"] is None


def test_thread_file_not_in_the_manifest_is_refused(client) -> None:
    for file_id in ("../other.xml", "threads/unknown.ndjson", "consignatio/vault/v1/other.xml"):
        response = client.get("/api/proffer/decoded/thread", params={"source_ref": SOURCE_REF, "file": file_id})
        assert response.status_code == 404, file_id


def test_media_streams_the_decoded_bytes_for_a_source_with_no_run(client) -> None:
    response = client.get(f"/api/proffer/decoded/media/{PICTURE_SHA}", params={"source_ref": SOURCE_REF})
    assert response.status_code == 200
    assert response.content == PICTURE
    assert response.headers["content-type"].startswith("image/png")


def test_source_outside_a_configured_root_is_refused(client) -> None:
    response = client.get("/api/proffer/decoded/manifest", params={"source_ref": "b2://other-bucket/x/file.xml"})
    assert response.status_code == 403


def test_source_that_was_never_decoded_says_so(client) -> None:
    response = client.get(
        "/api/proffer/decoded/manifest", params={"source_ref": f"b2://{BUCKET}/consignatio/vault/v1/never.xml"}
    )
    assert response.status_code == 404
    assert "not been decoded" in response.json()["detail"]
