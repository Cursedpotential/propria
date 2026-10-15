"""Synthetic contract tests for the optional screenshot/app proposal in the image stage."""

import base64
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
    APPS,
    DECISIONS_MODEL,
    DECISIONS_TOTAL_BYTES,
    VERSION,
    cache_path,
    classifier_version,
    classify,
    load_reference_bundle,
    target_jpeg,
    validate_decisions_reply,
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


def _decisions_reply(decision: str = "yes", app: str = "messages", *, probability: float = 0.8) -> dict:
    """Build complete typed distributions without a real provider call.

    Inputs: selected screenshot/app choices and winning probability. Output: Decisions API
    response. Side effects: none. Pick this for strict response and routing tests.
    """
    def answer(choice: str, labels: list[str]) -> dict:
        """Distribute remaining probability across every unselected label.

        Inputs: winning label and options. Output: complete choice answer. Side effects: none.
        Pick this only for synthetic Decisions fixtures.
        """
        others = (1 - probability) / (len(labels) - 1)
        return {
            "type": "choice",
            "choice": choice,
            "probabilities": {label: probability if label == choice else others for label in labels},
            "confidence": 0.33,
        }
    return {
        "model": "perplexity/pplx-decider-v1.1-27b-20261006",
        "answers": {
            "screenshot": answer(decision, ["yes", "no", "unknown"]),
            "app": answer(app, sorted(APPS)),
        },
        "usage": {"input_tokens": 100, "output_tokens": 0, "cost": 0.001},
    }


def _bundle(root: Path, *, version: str = "refs-v1"):
    """Create two synthetic, labeled JPEG references and their verified manifest.

    Inputs: test output root and set version. Output: manifest path and loaded bundle. Side
    effects: fixture files under the test root. Pick this for single-image provider tests.
    """
    folder = root / "datasets" / "images" / "classifier-references" / version
    folder.mkdir(parents=True, exist_ok=True)
    refs = []
    for index, app in enumerate(("messages", "snapchat")):
        image = io.BytesIO()
        Image.new("RGB", (300, 600), (index * 100, 80, 170)).save(image, format="JPEG")
        content = image.getvalue()
        filename = f"{app}.jpg"
        (folder / filename).write_bytes(content)
        refs.append({
            "image": filename,
            "source_sha256": hashlib.sha256(f"source-{app}".encode()).hexdigest(),
            "derived_sha256": hashlib.sha256(content).hexdigest(),
            "expected": {"screenshot_decision": "yes", "app": app, "confidence": 90, "basis": "visible UI controls"},
        })
    manifest = folder / "manifest.json"
    manifest.write_text(json.dumps({"schema_version": 1, "reference_set_version": version, "references": refs}))
    return manifest, load_reference_bundle(manifest, root)


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
async def test_classifier_sends_native_two_shot_turns_and_reuses_versioned_sha_cache(tmp_path: Path):
    """Send distinct image-answer examples before target and cache identical target bytes.

    Inputs: synthetic image and mock HTTP client. Output: proposal and one call count. Side
    effects: local lake cache only. Pick this to verify provider shape and SHA/version reuse.
    """
    source = tmp_path / "renamed.jpg"
    source.write_bytes(_image())
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    _, bundle = _bundle(tmp_path)
    calls = []

    def respond(request: httpx.Request) -> httpx.Response:
        calls.append(request)
        body = json.loads(request.content)
        assert body["model"] == "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning"
        assert body["response_format"] == {"type": "json_object"}
        assert body["reasoning_budget"] == 128 and body["max_tokens"] == 512
        messages = body["messages"]
        assert [message["role"] for message in messages] == ["system", "user", "assistant", "user", "assistant", "user"]
        assert "never conversation topics or filenames" in messages[0]["content"]
        assert "without transcribing messages, names, credentials" in messages[0]["content"]
        for user_index, assistant_index, reference_index in ((1, 2, 0), (3, 4, 1)):
            content = messages[user_index]["content"]
            assert [part["type"] for part in content] == ["image_url", "text"]
            assert f"REFERENCE {'AB'[reference_index]} only" in content[1]["text"]
            assert base64.b64decode(content[0]["image_url"]["url"].split(",", 1)[1]) == bundle.images[reference_index]
            assert json.loads(messages[assistant_index]["content"]) == bundle.expected[reference_index]
        target_content = messages[5]["content"]
        assert [part["type"] for part in target_content] == ["image_url", "text"]
        assert "TARGET only" in target_content[1]["text"]
        target_bytes = base64.b64decode(target_content[0]["image_url"]["url"].split(",", 1)[1])
        assert target_bytes not in bundle.images
        target_image = Image.open(io.BytesIO(target_bytes))
        assert target_image.format == "JPEG" and max(target_image.size) <= 1024
        return httpx.Response(200, json=_reply("yes", "facebook-messenger", 76))

    async with httpx.AsyncClient(transport=httpx.MockTransport(respond)) as client:
        first = await classify(source, digest, tmp_path, "placeholder-key", client, bundle)
        second = await classify(source, digest, tmp_path, "placeholder-key", client, bundle)
    assert {k: v for k, v in first.items() if not k.startswith("_")} == {k: v for k, v in second.items() if not k.startswith("_")}
    assert first["_request_made"] and second["_cache_hit"] and len(calls) == 1
    assert (first["screenshot_app"], first["screenshot_classifier_status"]) == ("facebook-messenger", "model_ok")
    assert first["screenshot_classifier_version"] == bundle.version


@pytest.mark.asyncio
async def test_decisions_request_routes_three_bounded_images_and_uses_separate_cache(tmp_path: Path):
    """Send reference-answer pairs and final target to the Decisions endpoint only.

    Inputs: synthetic images and mock HTTP response. Output: validated proposal and isolated
    cache. Side effects: test-only derived cache. Pick this to verify provider selection without
    sending original source bytes or mixing prior NIM results.
    """
    source = tmp_path / "target.png"
    source.write_bytes(_image())
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    manifest, nim_bundle = _bundle(tmp_path)
    bundle = load_reference_bundle(manifest, tmp_path, provider="openrouter-decisions", model=DECISIONS_MODEL)
    assert bundle.version != nim_bundle.version
    assert cache_path(tmp_path, digest, version=bundle.version) != cache_path(tmp_path, digest, version=nim_bundle.version)
    calls = []

    def respond(request: httpx.Request) -> httpx.Response:
        calls.append(request)
        assert str(request.url) == "https://openrouter.ai/api/alpha/decisions"
        body = json.loads(request.content)
        assert body["model"] == DECISIONS_MODEL and set(body) == {"model", "state", "questions"}
        assert list(body["questions"]) == ["screenshot", "app"]
        assert set(body["questions"]["screenshot"]["criteria"]) == {"yes", "no", "unknown"}
        assert set(body["questions"]["app"]["criteria"]) == APPS
        state = body["state"]
        assert len(state) == 6
        assert all(isinstance(state[index], str) for index in (0, 2, 4))
        assert "REFERENCE A only" in state[0] and "REFERENCE B only" in state[2]
        assert "TARGET only" in state[4]
        assert json.dumps(bundle.expected[0], sort_keys=True, separators=(",", ":")) in state[0]
        assert json.dumps(bundle.expected[1], sort_keys=True, separators=(",", ":")) in state[2]
        decoded = [base64.b64decode(state[index]["image_url"]["url"].split(",", 1)[1]) for index in (1, 3, 5)]
        assert all(state[index]["type"] == "image_url" for index in (1, 3, 5))
        assert sum(map(len, decoded)) <= DECISIONS_TOTAL_BYTES
        assert decoded[2] not in decoded[:2]
        assert all(Image.open(io.BytesIO(data)).format == "JPEG" for data in decoded)
        return httpx.Response(200, json=_decisions_reply())

    async with httpx.AsyncClient(transport=httpx.MockTransport(respond)) as client:
        first = await classify(source, digest, tmp_path, "placeholder-key", client, bundle)
        second = await classify(source, digest, tmp_path, "placeholder-key", client, bundle)
    assert len(calls) == 1 and second["_cache_hit"]
    assert first["screenshot_classifier_provider"] == "openrouter-decisions"
    assert (first["screenshot_decision"], first["screenshot_app"], first["screenshot_app_confidence"]) == ("yes", "messages", 80)
    assert first["_usage"]["prompt_tokens"] == 100
    assert first["_billed_cost_usd"] == 0.001
    assert first["_diagnostic"]["served_model"] == "perplexity/pplx-decider-v1.1-27b-20261006"
    assert '"billed_cost_usd": 0.001' in cache_path(tmp_path, digest, version=bundle.version).read_text()


def test_decisions_reply_rejects_malformed_weak_and_contradictory_answers():
    """Reject malformed distributions and abstain on weak or contradictory choices.

    Inputs: synthetic typed Decisions replies. Output: safe unknown or ValueError. Side effects:
    none. Pick this to guard conversion of provider probabilities into flat proposal fields.
    """
    version = classifier_version("openrouter-decisions", DECISIONS_MODEL)
    assert validate_decisions_reply(_decisions_reply(), version=version)["screenshot_app_confidence"] == 80
    weak = _decisions_reply(probability=0.60)
    assert validate_decisions_reply(weak, version=version)["screenshot_decision"] == "unknown"
    contradictory = _decisions_reply("no", "messages")
    assert validate_decisions_reply(contradictory, version=version)["screenshot_decision"] == "unknown"
    photo = _decisions_reply("no", "unknown")
    assert validate_decisions_reply(photo, version=version)["screenshot_decision"] == "no"
    malformed = _decisions_reply()
    malformed["answers"]["app"]["probabilities"]["messages"] = float("nan")
    with pytest.raises(ValueError, match="probabilities"):
        validate_decisions_reply(malformed, version=version)
    malformed = _decisions_reply()
    del malformed["answers"]["app"]["probabilities"]["unknown"]
    with pytest.raises(ValueError, match="labels"):
        validate_decisions_reply(malformed, version=version)


@pytest.mark.asyncio
async def test_malformed_decisions_response_stays_unknown_without_raw_text(tmp_path: Path):
    """Cache only a controlled failure when the Decisions provider returns malformed answers.

    Inputs: synthetic image and mocked malformed HTTP 200. Output: unknown and short cooldown.
    Side effects: private test cache only. Pick this to prevent provider text from reaching facts.
    """
    source = tmp_path / "target.png"
    source.write_bytes(_image())
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    manifest, _ = _bundle(tmp_path)
    bundle = load_reference_bundle(manifest, tmp_path, provider="openrouter-decisions", model=DECISIONS_MODEL)
    reply = _decisions_reply()
    reply["answers"]["app"]["probabilities"] = {"messages": 1.0, "PRIVATE_SOURCE_TEXT": 0.0}
    async with httpx.AsyncClient(transport=httpx.MockTransport(lambda request: httpx.Response(200, json=reply))) as client:
        result = await classify(source, digest, tmp_path, "placeholder-key", client, bundle)
    assert (result["screenshot_decision"], result["screenshot_classifier_status"]) == ("unknown", "model_failed_ValueError")
    stored = cache_path(tmp_path, digest, version=bundle.version).read_text()
    assert "PRIVATE_SOURCE_TEXT" not in stored and '"billed_cost_usd": 0.001' in stored


@pytest.mark.asyncio
@pytest.mark.parametrize("provider,key_name", [
    ("nvidia-nim", "NVIDIA_API_KEY"),
    ("openrouter-decisions", "OPENROUTER_API_KEY"),
])
async def test_facts_stage_reads_only_selected_provider_key(tmp_path: Path, monkeypatch, provider: str, key_name: str):
    """Route classifier credentials through the chosen provider without touching the other key.

    Inputs: mocked stage settings and secret reader. Output: exactly one selected key name.
    Side effects: none beyond test stubs. Pick this to guard runtime provider selection.
    """
    from casebible_index import stage_runner

    settings = SimpleNamespace(enabled=True, screenshot_classifier_enabled=True, screenshot_classifier_provider=provider)
    monkeypatch.setattr(stage_runner, "_image_settings", lambda: settings)
    monkeypatch.setattr(stage_runner, "load_settings", lambda: SimpleNamespace(output_dir=tmp_path, spool_dir=tmp_path))
    monkeypatch.setattr(stage_runner, "_scoped_image_slice", lambda *args: object())
    seen = []
    monkeypatch.setattr(stage_runner, "get_secret", lambda name: seen.append(name) or "placeholder-key")
    async def facts(*args, **kwargs):
        """Accept the selected key without running image work.

        Inputs: stage call arguments. Output: empty receipt. Side effects: assertion only.
        Pick this solely to isolate the stage's credential route.
        """
        assert kwargs["classifier_key"] == "placeholder-key"
        return {"images": 0}
    monkeypatch.setattr(stage_runner, "run_image_facts", facts)
    monkeypatch.setattr(stage_runner.ledger, "write_stage_receipt", lambda *args: None)
    assert await stage_runner.stage_image_facts("cycle", {"slice_id": "slice"}) == {"images": 0}
    assert seen == [key_name]


def test_classifier_settings_reject_unsupported_provider_model(monkeypatch):
    """Keep runtime model selection explicit and fail on unsupported provider contracts.

    Inputs: synthetic classifier environment values. Output: validated or rejected settings.
    Side effects: test process environment only. Pick this before allowing model configuration.
    """
    monkeypatch.setenv("INTAKE_IMAGES_SCREENSHOT_PROVIDER", "openrouter-decisions")
    monkeypatch.setenv("INTAKE_IMAGES_SCREENSHOT_MODEL", DECISIONS_MODEL)
    settings = image_stage.ImageStageSettings.from_env()
    settings.validate()
    assert (settings.screenshot_classifier_provider, settings.screenshot_classifier_model) == ("openrouter-decisions", DECISIONS_MODEL)
    for model in ("perplexity/pplx-decider-v1.1-27b-20261006", "typesafe/jev-1.14-20261014"):
        monkeypatch.setenv("INTAKE_IMAGES_SCREENSHOT_MODEL", model)
        image_stage.ImageStageSettings.from_env().validate()
    monkeypatch.setenv("INTAKE_IMAGES_SCREENSHOT_MODEL", "scaledown/classify-1")
    with pytest.raises(ValueError, match="unsupported screenshot classifier"):
        image_stage.ImageStageSettings.from_env().validate()
    monkeypatch.setenv("INTAKE_IMAGES_SCREENSHOT_MODEL", DECISIONS_MODEL)
    monkeypatch.setenv("INTAKE_IMAGES_SCREENSHOT_PROVIDER", "unsupported-provider")
    with pytest.raises(ValueError, match="unsupported screenshot classifier"):
        image_stage.ImageStageSettings.from_env().validate()


def test_reference_manifest_hashes_labels_and_native_target_are_bound(tmp_path: Path):
    """Bind example labels and derived hashes to cache identity while keeping target separate.

    Inputs: synthetic references and two target images. Output: version and panel assertions.
    Side effects: test bundle only. Pick this to prevent stale examples or target confusion.
    """
    manifest, bundle = _bundle(tmp_path)
    assert bundle.version.startswith(VERSION + ":")
    first_target = _image()
    changed_target = io.BytesIO()
    Image.new("RGB", (400, 800), "red").save(changed_target, format="PNG")
    first_jpeg = target_jpeg(first_target)
    changed_jpeg = target_jpeg(changed_target.getvalue())
    assert first_jpeg != changed_jpeg
    assert Image.open(io.BytesIO(first_jpeg)).size == (400, 800)
    manifest_data = json.loads(manifest.read_text())
    manifest_data["references"][0]["expected"]["basis"] = "changed visual controls"
    manifest.write_text(json.dumps(manifest_data))
    updated = load_reference_bundle(manifest, tmp_path)
    assert updated.version != bundle.version
    assert updated.expected != bundle.expected
    manifest_data["references"][0]["derived_sha256"] = "0" * 64
    manifest.write_text(json.dumps(manifest_data))
    with pytest.raises(ValueError, match="hash mismatch"):
        load_reference_bundle(manifest, tmp_path)


@pytest.mark.asyncio
async def test_missing_reference_manifest_fails_closed_before_provider(tmp_path: Path, monkeypatch):
    """Record unknown without a hosted call when two checked examples are unavailable.

    Inputs: one synthetic fetched image and missing manifest path. Output: reference_unavailable
    facts row and zero requests. Side effects: test facts shard only. Pick this for fail-closed mode.
    """
    source = tmp_path / "spool" / "images" / "missing" / "00000.png"
    source.parent.mkdir(parents=True)
    source.write_bytes(_image())
    item = ImageItem("missing-ref-image", "catalog-sha1", "b2", "bucket", "renamed.png", source.stat().st_size, 1, "00000.png")
    monkeypatch.setattr(image_stage, "read_image_facts", lambda *a, **k: SimpleNamespace(
        original_time=SimpleNamespace(value="", source="", confidence="", conflict=False),
        device="", software="", gps="", is_screenshot=False, notes=[],
    ))
    settings = SimpleNamespace(screenshot_classifier_enabled=True, screenshot_classifier_max_calls=5, screenshot_reference_manifest=str(tmp_path / "missing.json"))
    def forbidden(_request):
        raise AssertionError("provider must not be called without references")
    async with httpx.AsyncClient(transport=httpx.MockTransport(forbidden)) as client:
        receipt = await image_stage.run_facts(tmp_path, tmp_path / "spool", Slice("missing", "image", [item]), settings=settings, classifier_key="placeholder-key", classifier_client=client)
    assert receipt["screenshot_classifier_requests"] == 0
    row = pq.read_table(table_path(tmp_path, "facts", "missing")).to_pylist()[0]
    assert (row["screenshot_decision"], row["screenshot_classifier_status"]) == ("unknown", "reference_unavailable")


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
    manifest, bundle = _bundle(tmp_path)
    enabled = SimpleNamespace(screenshot_classifier_enabled=True, screenshot_classifier_max_calls=5, screenshot_reference_manifest=str(manifest))
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
    assert patches[0][1]["screenshot_classifier_version"] == bundle.version


@pytest.mark.asyncio
@pytest.mark.parametrize("provider", ["nvidia-nim", "openrouter-decisions"])
async def test_facts_caps_hosted_requests_at_five_for_six_images(tmp_path: Path, monkeypatch, provider: str):
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
        if provider == "openrouter-decisions":
            return httpx.Response(200, json=_decisions_reply("yes", "browser"))
        return httpx.Response(200, json=_reply("yes", "browser", 70))

    manifest, _ = _bundle(tmp_path)
    settings = SimpleNamespace(
        screenshot_classifier_enabled=True,
        screenshot_classifier_max_calls=5,
        screenshot_reference_manifest=str(manifest),
        screenshot_classifier_provider=provider,
        screenshot_classifier_model="",
    )
    async with httpx.AsyncClient(transport=httpx.MockTransport(respond)) as client:
        receipt = await image_stage.run_facts(tmp_path, tmp_path / "spool", Slice("s2", "image", items), settings=settings, classifier_key="placeholder-key", classifier_client=client)
    assert receipt["screenshot_classifier_requests"] == len(requests) == 5
    assert receipt["screenshot_classifier_billed_cost_usd"] == (0.005 if provider == "openrouter-decisions" else 0.0)
    rows = pq.read_table(table_path(tmp_path, "facts", "s2")).to_pylist()
    assert sum(r["screenshot_classifier_status"] == "budget_exhausted" for r in rows) == 1


@pytest.mark.asyncio
@pytest.mark.parametrize("provider", ["nvidia-nim", "openrouter-decisions"])
async def test_http_failure_is_unknown_cached_briefly_without_response_body(tmp_path: Path, provider: str):
    """Keep provider failures unknown and prevent rapid repeated calls or body leakage.

    Inputs: synthetic image and mocked HTTP 429. Output: unknown/cache hit. Side effects:
    private test cache only. Pick this for failure isolation and bounded retry behavior.
    """
    source = tmp_path / "source.png"
    source.write_bytes(_image())
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    manifest, bundle = _bundle(tmp_path)
    if provider == "openrouter-decisions":
        bundle = load_reference_bundle(manifest, tmp_path, provider=provider, model=DECISIONS_MODEL)
    calls = []

    def respond(request):
        calls.append(request)
        return httpx.Response(429, text="PRIVATE_PROVIDER_BODY_DO_NOT_PERSIST")

    async with httpx.AsyncClient(transport=httpx.MockTransport(respond)) as client:
        first = await classify(source, digest, tmp_path, "placeholder-key", client, bundle)
        second = await classify(source, digest, tmp_path, "placeholder-key", client, bundle)
    assert first["screenshot_classifier_status"] == "model_http_429"
    assert first["screenshot_classifier_provider"] == provider
    assert first["screenshot_decision"] == second["screenshot_decision"] == "unknown"
    assert len(calls) == 1 and second["_cache_hit"]
    assert "PRIVATE_PROVIDER_BODY_DO_NOT_PERSIST" not in cache_path(tmp_path, digest, version=bundle.version).read_text()


@pytest.mark.asyncio
async def test_invalid_json_keeps_usage_and_shape_only(tmp_path: Path):
    """Retain token cost and controlled format diagnostics when HTTP 200 carries prose.

    Inputs: synthetic image and mocked prose response. Output: unknown failure with usage and
    content category. Side effects: private test cache only. Pick this for the real-proof failure.
    """
    source = tmp_path / "source.png"
    source.write_bytes(_image())
    digest = hashlib.sha256(source.read_bytes()).hexdigest()
    _, bundle = _bundle(tmp_path)
    secret_prose = "PRIVATE_MODEL_PROSE_DO_NOT_PERSIST"
    payload = {
        "choices": [{"finish_reason": "stop", "message": {"content": secret_prose}}],
        "usage": {"prompt_tokens": 95, "completion_tokens": 38, "total_tokens": 133},
    }
    async with httpx.AsyncClient(transport=httpx.MockTransport(lambda r: httpx.Response(200, json=payload))) as client:
        result = await classify(source, digest, tmp_path, "placeholder-key", client, bundle)
    assert result["screenshot_decision"] == "unknown"
    assert result["screenshot_classifier_status"] == "model_failed_JSONDecodeError"
    assert result["_usage"]["total_tokens"] == 133
    assert result["_diagnostic"] == {"finish_reason": "stop", "content_format": "prose"}
    stored = cache_path(tmp_path, digest, version=bundle.version).read_text()
    assert secret_prose not in stored and '"total_tokens": 133' in stored
