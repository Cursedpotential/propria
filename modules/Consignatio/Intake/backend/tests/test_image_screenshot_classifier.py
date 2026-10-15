"""Synthetic contract tests for the optional screenshot/app proposal in the image stage."""

import hashlib
import io
import json
from datetime import UTC, datetime
from pathlib import Path
from types import SimpleNamespace
from uuid import uuid4

import httpx
import pyarrow as pa
import pyarrow.parquet as pq
import pytest
from PIL import Image

from casebible_index import image_stage
from casebible_index.image_screenshot_classifier import (
    VERSION,
    cache_path,
    classify,
    validate_reply,
)
from casebible_index.image_slice import ImageItem, Slice, table_path


def _image() -> bytes:
    """Return a synthetic PNG with no source or personal data.

    Inputs: none. Output: bytes. Side effects: none. Pick this for classifier request tests.
    """
    data = io.BytesIO()
    Image.new("RGB", (400, 800), "white").save(data, format="PNG")
    return data.getvalue()


def _reply(decision: str, app: str, confidence: int) -> dict:
    """Build the exact hosted JSON response shape for a mocked request.

    Inputs: decision, app and score. Output: provider response. Side effects: none. Pick this for
    request and stage contract tests without external inference.
    """
    return {
        "choices": [{"finish_reason": "stop", "message": {"content": json.dumps({
            "screenshot_decision": decision,
            "app": app,
            "confidence": confidence,
            "basis": "Visible navigation chrome",
        })}}],
        "usage": {"prompt_tokens": 100, "completion_tokens": 20, "total_tokens": 120},
    }


def test_classifier_rejects_structurally_invalid_result():
    """Reject unexpected model shapes and avoid treating its score as a boolean.

    Inputs: synthetic JSON. Output: validation assertions. Side effects: none. Pick this for the
    boundary before a model proposal reaches Parquet or Weaviate.
    """
    with pytest.raises(ValueError):
        validate_reply(json.dumps({"screenshot_decision": "yes", "app": "messages", "confidence": True, "basis": "UI"}))
    with pytest.raises(ValueError):
        validate_reply(json.dumps({"screenshot_decision": "no", "app": "whatsapp", "confidence": 50, "basis": "UI"}))
    with pytest.raises(ValueError):
        validate_reply(json.dumps({"screenshot_decision": "yes", "app": "messages", "confidence": 90, "basis": "UI", "extra": "text"}))
    valid = _reply("yes", "messages", 77)["choices"][0]["message"]["content"]
    assert validate_reply(f"```json\n{valid}\n```")["screenshot_app"] == "messages"
    with pytest.raises(ValueError):
        validate_reply(f"Here is the answer: {valid}")


@pytest.mark.asyncio
async def test_classifier_sends_image_first_and_reuses_versioned_sha_cache(tmp_path: Path):
    """Use the verified hosted request order and avoid a second call for identical bytes.

    Inputs: synthetic image and mock HTTP client. Output: proposal and one call count. Side
    effects: local lake cache only. Pick this to verify provider shape and SHA/version reuse.
    """
    source = tmp_path / "renamed.jpg"
    source.write_bytes(_image())
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    calls = []

    def respond(request: httpx.Request) -> httpx.Response:
        calls.append(request)
        body = json.loads(request.content)
        assert body["model"] == "meta/llama-3.2-11b-vision-instruct"
        assert body["response_format"] == {"type": "json_object"}
        content = body["messages"][0]["content"]
        assert content[0]["type"] == "image_url" and content[1]["type"] == "text"
        assert content[0]["image_url"]["url"].startswith("data:image/jpeg;base64,")
        return httpx.Response(200, json=_reply("yes", "facebook-messenger", 76))

    async with httpx.AsyncClient(transport=httpx.MockTransport(respond)) as client:
        first = await classify(source, digest, tmp_path, "placeholder-key", client)
        second = await classify(source, digest, tmp_path, "placeholder-key", client)
    assert {k: v for k, v in first.items() if not k.startswith("_")} == {k: v for k, v in second.items() if not k.startswith("_")}
    assert first["_request_made"] and second["_cache_hit"] and len(calls) == 1
    assert (first["screenshot_app"], first["screenshot_classifier_status"]) == ("facebook-messenger", "model_ok")
    assert first["screenshot_classifier_version"] == VERSION


@pytest.mark.asyncio
async def test_existing_facts_gain_model_fields_and_published_row_is_patched(tmp_path: Path, monkeypatch):
    """Upgrade a cached facts shard and patch a published row without source re-upload.

    Inputs: synthetic retained PNG and old ledger. Output: updated kind, fields and patch receipt.
    Side effects: test-only Parquet and mock HTTP. Pick this for classifier-version migration.
    """
    source = tmp_path / "spool" / "images" / "s1" / "00000.png"
    source.parent.mkdir(parents=True)
    source.write_bytes(_image())
    item = ImageItem("image-1", "catalog-sha1", "b2", "bucket", "renamed.png", source.stat().st_size, 1, "00000.png")
    sl = Slice("s1", "image", [item])
    monkeypatch.setattr(image_stage, "read_image_facts", lambda *a, **k: SimpleNamespace(
        original_time=SimpleNamespace(value="", source="", confidence="", conflict=False),
        device="", software="", gps="", is_screenshot=False, notes=[],
    ))
    original = await image_stage.run_facts(tmp_path, tmp_path / "spool", sl)
    assert original["images"] == 1
    legacy = pq.read_table(table_path(tmp_path, "facts", "s1")).to_pylist()[0]
    assert legacy["kind_basis"] == "png_no_camera" and legacy["screenshot_decision"] == "unknown"
    enabled = SimpleNamespace(screenshot_classifier_enabled=True, screenshot_classifier_max_calls=5)
    async with httpx.AsyncClient(transport=httpx.MockTransport(lambda r: httpx.Response(200, json=_reply("no", "unknown", 0)))) as client:
        receipt = await image_stage.run_facts(tmp_path, tmp_path / "spool", sl, settings=enabled, classifier_key="placeholder-key", classifier_client=client)
    upgraded = pq.read_table(table_path(tmp_path, "facts", "s1")).to_pylist()[0]
    assert receipt["screenshot_classifier_requests"] == 1
    assert receipt["screenshot_classifier_usage"] == {"prompt_tokens": 100, "completion_tokens": 20, "total_tokens": 120}
    assert (upgraded["kind"], upgraded["screenshot_decision"], upgraded["screenshot_app"]) == ("photo", "no", "unknown")
    old_ledger = image_stage._ledger(item.identity, "ok", "", str(uuid4()), sl, "run", datetime.now(UTC))
    path = table_path(tmp_path, "published", "s1")
    path.parent.mkdir(parents=True)
    pq.write_table(pa.Table.from_pylist([old_ledger], schema=image_stage.LEDGER_SCHEMA), path)
    patches = []

    class Writer:
        """Record classification-only patches for the existing-ledger branch."""
        collection = "IntakeImageV1"

        async def patch_classification(self, object_id, properties):
            """Record the object and changed fields without a network side effect."""
            patches.append((object_id, properties))

    result = await image_stage.run_publish(tmp_path, tmp_path / "spool", sl, Writer(), enabled, source_id="case", run_id="run", release=False)
    assert result["classification_patched"] == 1
    assert patches[0][1]["image_kind"] == "photo"
    assert patches[0][1]["screenshot_classifier_version"] == VERSION


@pytest.mark.asyncio
async def test_facts_caps_hosted_requests_at_five_for_six_images(tmp_path: Path, monkeypatch):
    """Keep a larger slice within the hard hosted-call budget and mark overflow unknown.

    Inputs: six distinct synthetic PNGs and a mock provider. Output: five requests and one
    budget_exhausted row. Side effects: test-only Parquet/cache writes. Pick this for cost bounds.
    """
    items = []
    for n in range(6):
        file = f"{n:05d}.png"
        path = tmp_path / "spool" / "images" / "s2" / file
        path.parent.mkdir(parents=True, exist_ok=True)
        image = io.BytesIO()
        Image.new("RGB", (400, 800), (n * 30, 255, 255)).save(image, format="PNG")
        path.write_bytes(image.getvalue())
        items.append(ImageItem(f"image-{n}", f"sha1-{n}", "b2", "bucket", f"renamed-{n}.png", path.stat().st_size, 1, file))
    monkeypatch.setattr(image_stage, "read_image_facts", lambda *a, **k: SimpleNamespace(
        original_time=SimpleNamespace(value="", source="", confidence="", conflict=False),
        device="", software="", gps="", is_screenshot=False, notes=[],
    ))
    requests = []

    def respond(request):
        requests.append(request)
        return httpx.Response(200, json=_reply("yes", "browser", 70))

    settings = SimpleNamespace(screenshot_classifier_enabled=True, screenshot_classifier_max_calls=5)
    async with httpx.AsyncClient(transport=httpx.MockTransport(respond)) as client:
        receipt = await image_stage.run_facts(tmp_path, tmp_path / "spool", Slice("s2", "image", items), settings=settings, classifier_key="placeholder-key", classifier_client=client)
    assert receipt["screenshot_classifier_requests"] == len(requests) == 5
    rows = pq.read_table(table_path(tmp_path, "facts", "s2")).to_pylist()
    assert sum(r["screenshot_classifier_status"] == "budget_exhausted" for r in rows) == 1


@pytest.mark.asyncio
async def test_http_failure_is_unknown_cached_briefly_without_response_body(tmp_path: Path):
    """Keep provider failures unknown and prevent rapid repeated calls or body leakage.

    Inputs: synthetic image and mocked HTTP 429. Output: unknown/cache hit. Side effects:
    private test cache only. Pick this for failure isolation and bounded retry behavior.
    """
    source = tmp_path / "source.png"
    source.write_bytes(_image())
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    calls = []

    def respond(request):
        calls.append(request)
        return httpx.Response(429, text="PRIVATE_PROVIDER_BODY_DO_NOT_PERSIST")

    async with httpx.AsyncClient(transport=httpx.MockTransport(respond)) as client:
        first = await classify(source, digest, tmp_path, "placeholder-key", client)
        second = await classify(source, digest, tmp_path, "placeholder-key", client)
    assert first["screenshot_classifier_status"] == "model_http_429"
    assert first["screenshot_decision"] == second["screenshot_decision"] == "unknown"
    assert len(calls) == 1 and second["_cache_hit"]
    assert "PRIVATE_PROVIDER_BODY_DO_NOT_PERSIST" not in cache_path(tmp_path, digest).read_text()


@pytest.mark.asyncio
async def test_invalid_json_keeps_usage_and_shape_only(tmp_path: Path):
    """Retain token cost and controlled format diagnostics when HTTP 200 carries prose.

    Inputs: synthetic image and mocked prose response. Output: unknown failure with usage and
    content category. Side effects: private test cache only. Pick this for the real-proof failure.
    """
    source = tmp_path / "source.png"
    source.write_bytes(_image())
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    secret_prose = "PRIVATE_MODEL_PROSE_DO_NOT_PERSIST"
    payload = {
        "choices": [{"finish_reason": "stop", "message": {"content": secret_prose}}],
        "usage": {"prompt_tokens": 95, "completion_tokens": 38, "total_tokens": 133},
    }
    async with httpx.AsyncClient(transport=httpx.MockTransport(lambda r: httpx.Response(200, json=payload))) as client:
        result = await classify(source, digest, tmp_path, "placeholder-key", client)
    assert result["screenshot_decision"] == "unknown"
    assert result["screenshot_classifier_status"] == "model_failed_JSONDecodeError"
    assert result["_usage"]["total_tokens"] == 133
    assert result["_diagnostic"] == {"finish_reason": "stop", "content_format": "prose"}
    stored = cache_path(tmp_path, digest).read_text()
    assert secret_prose not in stored and '"total_tokens": 133' in stored
