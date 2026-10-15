"""Bounded hosted screenshot and app recognition for retained image slices.

Results are machine proposals. The score is the model's reported certainty, not a calibrated
probability, and neither a filename nor OCR topics establish an app identity.
"""

from __future__ import annotations

import asyncio
import base64
import hashlib
import io
import json
import math
import re
import time
import uuid
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import httpx
from PIL import ImageOps

from .image_decode import open_image

MODEL = "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning"
ENDPOINT = "https://integrate.api.nvidia.com/v1/chat/completions"
DECISIONS_ENDPOINT = "https://openrouter.ai/api/alpha/decisions"
DECISIONS_MODEL = "cloudflare/clef-flash"
CHAIN_PROVIDER = "openrouter-decisions-chain"
CHAIN_MODEL = "mercury-free-then-luna-20261006"
CHAIN_SCREENSHOT_MODEL = "inception/mercury-decide:free"
CHAIN_APP_MODEL = "openai/gpt-6-luna-decisions-20261006"
DECISIONS_MODELS = frozenset({
    DECISIONS_MODEL,
    "perplexity/pplx-decider-v1.1-27b",
    "perplexity/pplx-decider-v1.1-27b-20261006",
    "typesafe/jev-1.14",
    "typesafe/jev-1.14-20261014",
    "openai/gpt-6-luna-decisions",
    "openai/gpt-6-luna-decisions-20261006",
    "inception/mercury-decide:free",
})
DECISIONS_PROMPT_VERSION = "screenshot-app-ui-v1-decisions-two-reference"
DECISIONS_PREPROCESS_VERSION = "jpeg-768q70-640q60-512q50-384q40-max60k-v1"
DECISIONS_IMAGE_BYTES = 60_000
DECISIONS_TOTAL_BYTES = 180_000
PROMPT_VERSION = "screenshot-app-ui-v5-native-two-shot"
PREPROCESS_VERSION = "native-target-1024-jpeg-q85-v1"
VERSION = f"{MODEL}:{PROMPT_VERSION}:{PREPROCESS_VERSION}"
MAX_REFERENCE_BYTES = 2_000_000
MAX_TARGET_BYTES = 1_000_000
APPS = frozenset(
    {"facebook-messenger", "messages", "snapchat", "whatsapp", "instagram", "browser", "other", "mixed", "unknown"}
)
DECISIONS = frozenset({"yes", "no", "unknown"})
SYSTEM_PROMPT = (
    "Classify whether the current image is a screenshot and identify its visible app UI. "
    "Use the two prior image-and-answer turns only as examples, not as evidence about the final image. "
    "Judge app from visible controls, chrome, or logos, never conversation topics or filenames. "
    "A cropped or renamed screenshot still counts; PNG format alone proves nothing. "
    "Valid screenshot_decision: yes, no, unknown. Valid app: facebook-messenger, messages, "
    "snapchat, whatsapp, instagram, browser, other, mixed, unknown. Mixed requires multiple "
    "app UIs within the current image. Return only JSON with exactly screenshot_decision, app, "
    "confidence, basis. Confidence is an uncalibrated integer 0..100; set zero if decision or "
    "app is unknown. For no/unknown decision, app must be unknown and confidence zero. "
    "Basis must briefly describe visible UI without transcribing messages, names, credentials, "
    "or secrets."
)
REFERENCE_A_PROMPT = "Classify REFERENCE A only. Return the four-field JSON answer for this image."
REFERENCE_B_PROMPT = "Classify REFERENCE B only. Return the four-field JSON answer for this image."
TARGET_PROMPT = "Classify TARGET only. Ignore the reference images when judging TARGET. Return TARGET JSON only."

FENCE = re.compile(r"\A```(?:json)?\s*\n(?P<body>[\s\S]*?)\n```\s*\Z", re.IGNORECASE)
_mercury_start_lock = asyncio.Lock()
_next_mercury_start = 0.0


@dataclass(frozen=True)
class ReferenceBundle:
    """Two hash-checked derived UI references and their cache/provenance fingerprint.

    Inputs: validated manifest and JPEG bytes. Output: immutable runtime bundle. Side effects:
    none. Pick this only for classifier example turns, never OCR or embedding.
    """

    version: str
    images: tuple[bytes, bytes]
    expected: tuple[dict[str, Any], dict[str, Any]]
    provider: str = "nvidia-nim"
    model: str = MODEL
    components: tuple[ReferenceBundle, ReferenceBundle] | None = None


def classifier_version(provider: str = "nvidia-nim", model: str = MODEL) -> str:
    """Identify the exact provider, model, prompt and preprocessing contract for cache isolation.

    Inputs: controlled provider and configured model. Output: stable version string. Side effects:
    none. Pick this before reading references or recording an unavailable classifier result.
    """
    if provider == "nvidia-nim" and model == MODEL:
        return VERSION
    if provider == "openrouter-decisions" and model in DECISIONS_MODELS:
        return f"{provider}:{model}:{DECISIONS_PROMPT_VERSION}:{DECISIONS_PREPROCESS_VERSION}"
    if provider == CHAIN_PROVIDER and model == CHAIN_MODEL:
        return f"{provider}:{model}:v1"
    raise ValueError("unsupported screenshot classifier provider or model")


def load_reference_bundle(
    manifest_path: Path, output_dir: Path, *, provider: str = "nvidia-nim", model: str = MODEL
) -> ReferenceBundle:
    """Validate exactly two reviewed reference derivatives beneath the output lake.

    Inputs: absolute manifest path and attached output root. Output: hash-checked JPEG bundle
    with a version derived from provider, model, prompt, preprocessing, labels and source/derived hashes. Side
    effects: bounded reads only. Pick this once per facts activity before any provider request.
    Invalid or unavailable inputs raise ValueError/OSError for a safe unknown result upstream.
    """
    if provider == CHAIN_PROVIDER:
        classifier_version(provider, model)
        screenshot = load_reference_bundle(
            manifest_path, output_dir, provider="openrouter-decisions", model=CHAIN_SCREENSHOT_MODEL
        )
        app = load_reference_bundle(
            manifest_path, output_dir, provider="openrouter-decisions", model=CHAIN_APP_MODEL
        )
        components = (screenshot.version, app.version)
        fingerprint = hashlib.sha256(
            json.dumps(components, separators=(",", ":")).encode()
        ).hexdigest()
        return ReferenceBundle(
            f"{classifier_version(provider, model)}:{fingerprint}",
            screenshot.images,
            screenshot.expected,
            provider,
            model,
            (screenshot, app),
        )
    if not manifest_path.is_absolute():
        raise ValueError("reference manifest must be absolute")
    base = output_dir.resolve()
    manifest = manifest_path.resolve(strict=True)
    if not manifest.is_relative_to(base) or manifest.stat().st_size > 16_000:
        raise ValueError("reference manifest outside output lake or too large")
    data = json.loads(manifest.read_text(encoding="utf-8"))
    if not isinstance(data, dict) or set(data) != {"schema_version", "reference_set_version", "references"}:
        raise ValueError("reference manifest shape invalid")
    set_version = data["reference_set_version"]
    if data["schema_version"] != 1 or not isinstance(set_version, str) or not re.fullmatch(r"[A-Za-z0-9._-]{1,80}", set_version):
        raise ValueError("reference manifest version invalid")
    refs = data["references"]
    if not isinstance(refs, list) or len(refs) != 2:
        raise ValueError("exactly two references required")
    images: list[bytes] = []
    for index, ref in enumerate(refs):
        if not isinstance(ref, dict) or set(ref) != {"image", "source_sha256", "derived_sha256", "expected"}:
            raise ValueError("reference entry invalid")
        expected = ref["expected"]
        app = ("messages", "snapchat")[index]
        if not isinstance(expected, dict) or set(expected) != {"screenshot_decision", "app", "confidence", "basis"}:
            raise ValueError("reference label shape invalid")
        if expected["screenshot_decision"] != "yes" or expected["app"] != app or type(expected["confidence"]) is not int or not 0 <= expected["confidence"] <= 100:
            raise ValueError("reference label invalid")
        if not isinstance(expected["basis"], str) or not 1 <= len(expected["basis"]) <= 120:
            raise ValueError("reference basis invalid")
        for name in ("source_sha256", "derived_sha256"):
            digest = ref[name]
            if not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest):
                raise ValueError("reference hash invalid")
        rel = ref["image"]
        if not isinstance(rel, str) or not rel or Path(rel).is_absolute() or ".." in Path(rel).parts:
            raise ValueError("reference path invalid")
        path = (manifest.parent / rel).resolve(strict=True)
        if not path.is_relative_to(manifest.parent) or not path.is_relative_to(base) or path.stat().st_size > MAX_REFERENCE_BYTES:
            raise ValueError("reference image outside bundle or too large")
        content = path.read_bytes()
        if hashlib.sha256(content).hexdigest() != ref["derived_sha256"]:
            raise ValueError("reference derivative hash mismatch")
        with open_image(io.BytesIO(content)) as image:
            if image.format != "JPEG":
                raise ValueError("reference derivative must be JPEG")
        images.append(content)
    base_version = classifier_version(provider, model)
    canonical = json.dumps({"base_version": base_version, "set_version": set_version, "references": refs}, sort_keys=True, separators=(",", ":"))
    fingerprint = hashlib.sha256(canonical.encode()).hexdigest()
    return ReferenceBundle(
        f"{base_version}:{fingerprint}",
        (images[0], images[1]),
        (refs[0]["expected"], refs[1]["expected"]),
        provider,
        model,
    )


def target_jpeg(source: Path | io.BytesIO | bytes) -> bytes:
    """Make a bounded native JPEG of one retained target without reference pixels.

    Inputs: fetched source path or in-memory source bytes. Output: JPEG at quality 85 with
    longest edge at most 1024 and at most one megabyte. Side effects: image decoding only;
    original source is never written. Pick this only for classifier inference, while OCR and
    embeddings continue to use the original fetched image.
    """
    image_source = io.BytesIO(source) if isinstance(source, bytes) else source
    with open_image(image_source) as image:
        image.seek(0)
        oriented = ImageOps.exif_transpose(image).convert("RGB")
        oriented.thumbnail((1024, 1024))
        buffer = io.BytesIO()
        oriented.save(buffer, format="JPEG", quality=85)
    if buffer.tell() > MAX_TARGET_BYTES:
        raise ValueError("target JPEG exceeds provider byte cap")
    return buffer.getvalue()


def decisions_jpeg(source: Path | io.BytesIO | bytes) -> bytes:
    """Bound one Decisions API image to at most 60 KB without changing its original source.

    Inputs: a checked reference JPEG or fetched target path/bytes. Output: oriented JPEG no more
    than 768 pixels on its longest side and 60 KB. Side effects: decode only. Pick this for each
    of the three Decisions API image parts; OCR and embeddings retain their original sources.
    """
    image_source = io.BytesIO(source) if isinstance(source, bytes) else source
    with open_image(image_source) as image:
        image.seek(0)
        oriented = ImageOps.exif_transpose(image).convert("RGB")
        for edge, quality in ((768, 70), (640, 60), (512, 50), (384, 40)):
            resized = oriented.copy()
            resized.thumbnail((edge, edge))
            buffer = io.BytesIO()
            resized.save(buffer, format="JPEG", quality=quality)
            if buffer.tell() <= DECISIONS_IMAGE_BYTES:
                return buffer.getvalue()
    raise ValueError("decision image exceeds provider byte cap")


def image_part(jpeg: bytes) -> dict[str, Any]:
    """Wrap one bounded JPEG as a provider image part for its own user turn.

    Inputs: validated reference or derived target JPEG bytes. Output: image_url content part.
    Side effects: none. Pick this for the classifier's native multimodal example/target turns.
    """
    return {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64," + base64.b64encode(jpeg).decode()}}


def unknown(
    status: str, basis: str = "", *, version: str = VERSION, provider: str = "nvidia-nim"
) -> dict[str, Any]:
    """Return a safe unknown proposal with a controlled status and no source content.

    Inputs: non-sensitive status and short reason. Output: flat classifier fields. Side effects:
    none. Pick this for disabled, missing-key and provider failures instead of guessing an app.
    """
    return {
        "screenshot_decision": "unknown",
        "screenshot_app": "unknown",
        "screenshot_app_confidence": 0,
        "screenshot_app_basis": basis[:120],
        "screenshot_classifier_version": version,
        "screenshot_classifier_provider": provider,
        "screenshot_classifier_status": status,
    }


def content_format(content: Any) -> str:
    """Describe a response's shape without retaining or printing its text.

    Inputs: provider message content. Output: one controlled format category. Side effects: none.
    Pick this for private cache/receipt diagnostics when JSON validation fails.
    """
    if not isinstance(content, str):
        return "non_string"
    value = content.strip()
    if not value:
        return "empty"
    if value.startswith("```"):
        return "fenced_json" if FENCE.fullmatch(value) else "invalid_fence"
    return "json_object" if value.startswith("{") else "prose"


def validate_reply(content: str, *, version: str = VERSION) -> dict[str, Any]:
    """Validate the model's exact JSON contract before any proposal is persisted.

    Inputs: one provider response content string. Output: normalized flat fields. Side effects:
    none. Pick this after the hosted call; rejected output becomes an unknown failure upstream.
    """
    if not isinstance(content, str):
        raise TypeError("classifier content is not text")
    value = content.strip()
    if value.startswith("```"):
        match = FENCE.fullmatch(value)
        if match is None:
            raise ValueError("classifier fence is invalid")
        value = match.group("body").strip()
    data = json.loads(value)
    expected = {"screenshot_decision", "app", "confidence", "basis"}
    if not isinstance(data, dict) or set(data) != expected:
        raise ValueError("classifier JSON fields are invalid")
    decision, app, confidence, basis = (data[k] for k in ("screenshot_decision", "app", "confidence", "basis"))
    if decision not in DECISIONS or app not in APPS:
        raise ValueError("classifier decision or app is invalid")
    if type(confidence) is not int or not 0 <= confidence <= 100:
        raise ValueError("classifier confidence is invalid")
    if not isinstance(basis, str) or not 1 <= len(basis.strip()) <= 120:
        raise ValueError("classifier basis is invalid")
    if decision != "yes" and (app != "unknown" or confidence != 0):
        raise ValueError("non-screenshot cannot name an app")
    if decision == "yes" and app == "unknown" and confidence != 0:
        raise ValueError("unknown app must have zero confidence")
    # Provider prose is not stored. A compact reason code avoids retaining quoted messages or PII.
    reason = f"visible_ui:{app}" if app != "unknown" else "screen_ui_uncertain"
    return {
        "screenshot_decision": decision,
        "screenshot_app": app,
        "screenshot_app_confidence": confidence,
        "screenshot_app_basis": reason,
        "screenshot_classifier_version": version,
        "screenshot_classifier_provider": "nvidia-nim",
        "screenshot_classifier_status": "model_ok" if decision != "unknown" else "model_unknown",
    }


def decisions_body(bundle: ReferenceBundle, target: Path | io.BytesIO | bytes) -> dict[str, Any]:
    """Build one bounded Decisions request with two labeled examples and one unanswered target.

    Inputs: hash-checked references and fetched target. Output: OpenRouter Decisions request.
    Side effects: image decode only. Pick this for the selected Decisions provider; NIM uses
    its existing multi-turn chat request and neither path writes source bytes.
    """
    images = [decisions_jpeg(bundle.images[0]), decisions_jpeg(bundle.images[1]), decisions_jpeg(target)]
    if sum(map(len, images)) > DECISIONS_TOTAL_BYTES:
        raise ValueError("decision images exceed combined byte cap")
    state = [
        "REFERENCE A only; its reviewed answer is " + json.dumps(bundle.expected[0], sort_keys=True, separators=(",", ":")) + ". Use this answer only for the following image.",
        image_part(images[0]),
        "REFERENCE B only; its reviewed answer is " + json.dumps(bundle.expected[1], sort_keys=True, separators=(",", ":")) + ". Use this answer only for the following image.",
        image_part(images[1]),
        "TARGET only. Classify this final image from visible UI controls, chrome, and logos. The two preceding labels are examples, not evidence about TARGET. Never infer app from message topics or names; never transcribe messages, names, credentials, or secrets. A cropped screenshot counts; PNG alone proves nothing.",
        image_part(images[2]),
    ]
    return {
        "model": bundle.model,
        "state": state,
        "questions": {
            "screenshot": {
                "type": "choice",
                "instructions": "For TARGET only, is this a screenshot of a visible user interface? Choose unknown if uncertain; do not infer from file type or references.",
                "criteria": {value: None for value in ("yes", "no", "unknown")},
            },
            "app": {
                "type": "choice",
                "instructions": "For TARGET only, which app has visible UI controls or logo? Choose unknown if not a screenshot or uncertain. Mixed requires multiple app UIs within TARGET.",
                "criteria": {value: None for value in sorted(APPS)},
            },
        },
    }


def validate_decisions_reply(payload: dict[str, Any], *, version: str) -> dict[str, Any]:
    """Normalize strictly checked choice distributions into the existing flat proposal.

    Inputs: one Decisions API JSON object and cache version. Output: app/decision fields with
    an uncalibrated corpus score derived from the winning app probability. Side effects: none.
    Pick this for Decisions replies; NIM chat replies use validate_reply instead. Weak,
    contradictory or malformed answers become unknown or raise for the caller's safe failure.
    """
    if not isinstance(payload, dict) or not isinstance(payload.get("answers"), dict):
        raise TypeError("decision answer shape invalid")
    answers = payload["answers"]
    if set(answers) != {"screenshot", "app"}:
        raise ValueError("decision answer names invalid")

    def checked_choice(name: str, allowed: frozenset[str]) -> tuple[str, float]:
        """Validate one complete probability distribution and return its selected probability.

        Inputs: answer name and allowed choices. Output: selected label and probability. Side
        effects: none. Pick this inside Decisions reply validation for either typed question.
        """
        answer = answers[name]
        if not isinstance(answer, dict) or set(answer) != {"type", "choice", "probabilities", "confidence"} or answer["type"] != "choice":
            raise ValueError("decision choice shape invalid")
        choice, probabilities = answer["choice"], answer["probabilities"]
        if choice not in allowed or not isinstance(probabilities, dict) or set(probabilities) != allowed:
            raise ValueError("decision choice labels invalid")
        values = probabilities.values()
        if any(type(value) not in (int, float) or not 0 <= value <= 1 or not math.isfinite(value) for value in values):
            raise ValueError("decision probabilities invalid")
        if not 0.98 <= sum(probabilities.values()) <= 1.02:
            raise ValueError("decision probability sum invalid")
        confidence = answer["confidence"]
        if type(confidence) not in (int, float) or not 0 <= confidence <= 1 or not math.isfinite(confidence):
            raise ValueError("decision distribution confidence invalid")
        selected = probabilities[choice]
        if selected + 0.001 < max(probabilities.values()):
            raise ValueError("decision selected choice is not highest")
        return choice, selected

    decision, decision_probability = checked_choice("screenshot", DECISIONS)
    app, app_probability = checked_choice("app", APPS)
    if decision == "unknown" or (decision != "unknown" and decision_probability < 0.8):
        return unknown("model_unknown", "screen_ui_uncertain", version=version, provider="openrouter-decisions")
    if decision == "no":
        if app != "unknown" or app_probability < 0.8:
            return unknown("model_unknown", "screen_ui_uncertain", version=version, provider="openrouter-decisions")
        return {
            **unknown("model_ok", "visible_ui:none", version=version, provider="openrouter-decisions"),
            "screenshot_decision": "no",
        }
    if app != "unknown" and app_probability < 0.8:
        return unknown("model_unknown", "screen_ui_uncertain", version=version, provider="openrouter-decisions")
    return {
        **unknown("model_ok", f"visible_ui:{app}" if app != "unknown" else "screen_ui_uncertain", version=version, provider="openrouter-decisions"),
        "screenshot_decision": "yes",
        "screenshot_app": app,
        "screenshot_app_confidence": round(app_probability * 100) if app != "unknown" else 0,
    }


def cache_path(output_dir: Path, digest: str, *, version: str = VERSION) -> Path:
    """Locate one versioned SHA-256 result in the existing image output lake.

    Inputs: output lake root and retained-byte SHA-256. Output: JSON cache path. Side effects:
    none. Pick this for cross-slice reuse without creating a source catalog.
    """
    if len(digest) != 64 or any(c not in "0123456789abcdef" for c in digest):
        raise ValueError("invalid retained image SHA-256")
    key = hashlib.sha256(f"{digest}:{version}".encode()).hexdigest()
    return output_dir / "datasets" / "images" / "classification-cache" / key[:2] / f"{key}.json"


def _cache_result(target: Path, result: dict[str, Any], *, retry_after: float = 0) -> None:
    """Atomically retain a small result and optional provider-failure cooldown.

    Inputs: versioned cache path, flat result and Unix retry time. Output: none. Side effects:
    writes only inside the existing image lake. Pick this for validated model responses and
    brief provider-error cooldowns; source bytes and request bodies are never cached.
    """
    try:
        target.parent.mkdir(parents=True, exist_ok=True)
        partial = target.with_name(f"{target.name}.{uuid.uuid4().hex}.partial")
        payload = {"result": {k: v for k, v in result.items() if not k.startswith("_")}, "retry_after": retry_after}
        if result.get("_usage"):
            payload["usage"] = result["_usage"]
        if result.get("_billed_cost_usd") is not None:
            payload["billed_cost_usd"] = result["_billed_cost_usd"]
        if result.get("_diagnostic"):
            payload["diagnostic"] = result["_diagnostic"]
        partial.write_text(json.dumps(payload, sort_keys=True), encoding="utf-8")
        partial.replace(target)
    except OSError:
        # Classification remains usable even if an optional cache write fails.
        return


def cached_classification(output_dir: Path, digest: str, *, version: str) -> dict[str, Any] | None:
    """Read a reusable validated result or active failure cooldown without a provider call.

    Inputs are the output lake, retained source SHA-256 and exact classifier version.
    Output is a safe cached proposal with cache-hit metadata, or None. Side effects are
    bounded cache reads only. Pick this before enforcing an external-call budget; use
    ``classify`` for a cache miss that may require a provider request.
    """
    target = cache_path(output_dir, digest, version=version)
    if target.is_file():
        try:
            stored = json.loads(target.read_text(encoding="utf-8"))
            cached = stored.get("result", {})
            if cached.get("screenshot_classifier_version") == version and (
                cached.get("screenshot_classifier_status") in {"model_ok", "model_unknown"}
                or stored.get("retry_after", 0) > time.time()
            ):
                return {**cached, "_cache_hit": True, "_diagnostic": stored.get("diagnostic", {})}
        except (OSError, ValueError, TypeError, AttributeError):
            pass
    return None


async def _pace_mercury_start() -> None:
    """Space real Mercury-free request starts by at least 3.2 seconds in this worker.

    Inputs and output are none. Side effect is a short cooperative wait on a cache miss.
    Pick this immediately before a Mercury-free provider call, never for a cache hit.
    """
    global _next_mercury_start
    async with _mercury_start_lock:
        delay = max(0.0, _next_mercury_start - time.monotonic())
        if delay:
            await asyncio.sleep(delay)
        _next_mercury_start = time.monotonic() + 3.2


async def _classify_chain(
    path: Path | io.BytesIO | bytes,
    digest: str,
    output_dir: Path,
    key: str,
    client: httpx.AsyncClient,
    bundle: ReferenceBundle,
    request_budget: int | None,
) -> dict[str, Any]:
    """Gate screenshot detection through Mercury, then ask Luna only for an app on yes.

    Inputs are one retained source, its hash, lake, OpenRouter client/key, fixed chain bundle
    and remaining external-call budget. Output is a flat proposal with actual request count.
    Side effects are at most two component requests and versioned component/final cache writes.
    Pick this only for the owner's explicit Mercury-then-Luna provider.
    """
    if bundle.components is None:
        raise ValueError("chain reference components unavailable")
    screenshot_bundle, app_bundle = bundle.components
    remaining = 2 if request_budget is None else max(0, request_budget)
    request_count = 0
    cache_hits = 0
    usage = {"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0}
    billed_cost = 0.0
    cost_missing = 0
    diagnostic: dict[str, str] = {}

    async def component(child: ReferenceBundle, label: str) -> dict[str, Any] | None:
        """Reuse a child cache or make one bounded call; output proposal or budget miss."""
        nonlocal remaining, request_count, cache_hits, billed_cost, cost_missing
        result = cached_classification(output_dir, digest, version=child.version)
        if result is None:
            if remaining <= 0:
                return None
            if label == "mercury" and key:
                await _pace_mercury_start()
            result = await classify(
                path, digest, output_dir, key, client, child, request_budget=1
            )
        actual = result.get("_request_count", 0)
        request_count += actual
        remaining -= actual
        cache_hits += int(bool(result.get("_cache_hit")))
        for field in usage:
            usage[field] += result.get("_usage", {}).get(field, 0)
        if actual:
            cost = result.get("_billed_cost_usd")
            if type(cost) in (int, float) and math.isfinite(cost) and cost >= 0:
                billed_cost += cost
            else:
                cost_missing += actual
        diagnostic[f"{label}_status"] = result["screenshot_classifier_status"]
        served = result.get("_diagnostic", {}).get("served_model")
        if isinstance(served, str) and re.fullmatch(r"[A-Za-z0-9._/:-]{1,120}", served):
            diagnostic[f"{label}_served_model"] = served
        return result

    def finish(
        result: dict[str, Any], *, cache: bool = False, cooldown: bool = False
    ) -> dict[str, Any]:
        """Attach safe request provenance and optionally retain a complete chain result."""
        result["_request_count"] = request_count
        result["_request_made"] = request_count > 0
        result["_component_cache_hits"] = cache_hits
        if cache_hits:
            result["_cache_hit"] = True
        if any(usage.values()):
            result["_usage"] = usage
        if request_count and not cost_missing:
            result["_billed_cost_usd"] = billed_cost
        if cost_missing:
            diagnostic["missing_cost_requests"] = str(cost_missing)
        diagnostic["component_cache_hits"] = str(cache_hits)
        result["_diagnostic"] = diagnostic
        if cache:
            _cache_result(
                cache_path(output_dir, digest, version=bundle.version), result,
                retry_after=time.time() + 300 if cooldown else 0,
            )
        return result

    mercury = await component(screenshot_bundle, "mercury")
    if mercury is None:
        return finish(unknown("budget_exhausted", version=bundle.version, provider=bundle.provider))
    mercury_status = mercury["screenshot_classifier_status"]
    if mercury_status == "model_ok" and mercury["screenshot_decision"] == "no":
        result = unknown(
            "model_ok", "visible_ui:none", version=bundle.version, provider=bundle.provider
        )
        result["screenshot_decision"] = "no"
        return finish(result, cache=True)
    if mercury_status == "model_unknown":
        return finish(
            unknown("model_unknown", version=bundle.version, provider=bundle.provider), cache=True
        )
    if mercury_status != "model_ok" or mercury["screenshot_decision"] != "yes":
        failure = unknown(
            f"mercury_{mercury_status}", version=bundle.version, provider=bundle.provider
        )
        return finish(failure, cache=True, cooldown=True)

    luna = await component(app_bundle, "luna")
    if luna is None:
        partial = unknown("budget_exhausted", version=bundle.version, provider=bundle.provider)
        partial["screenshot_decision"] = "yes"
        return finish(partial)
    luna_status = luna["screenshot_classifier_status"]
    status = "model_ok" if luna_status in {"model_ok", "model_unknown"} else f"luna_{luna_status}"
    result = unknown(
        status, "screen_ui_uncertain", version=bundle.version, provider=bundle.provider
    )
    result["screenshot_decision"] = "yes"
    if (
        luna_status == "model_ok"
        and luna["screenshot_decision"] == "yes"
        and luna["screenshot_app"] != "unknown"
    ):
        result["screenshot_app"] = luna["screenshot_app"]
        result["screenshot_app_confidence"] = luna["screenshot_app_confidence"]
        result["screenshot_app_basis"] = f"visible_ui:{luna['screenshot_app']}"
    return finish(result, cache=True, cooldown=status != "model_ok")


async def classify(
    path: Path | io.BytesIO | bytes,
    digest: str,
    output_dir: Path,
    key: str,
    client: httpx.AsyncClient,
    bundle: ReferenceBundle,
    *,
    request_budget: int | None = None,
) -> dict[str, Any]:
    """Classify a retained image through the hosted vision model, reusing a versioned result.

    Inputs: fetched path, its SHA-256, output lake, selected provider key and HTTP client. Output: flat
    proposal with explicit model status. Side effects: bounded decode, one HTTPS request on cache
    miss, atomic JSON cache write for validated replies. ``request_budget`` caps external calls,
    including both chain components. Pick this within image_facts; OCR and embedding have separate
    jobs. Provider errors return unknown without response body or secrets.
    """
    cached = cached_classification(output_dir, digest, version=bundle.version)
    if cached is not None:
        return {**cached, "_request_count": 0}
    if bundle.provider == CHAIN_PROVIDER:
        return await _classify_chain(path, digest, output_dir, key, client, bundle, request_budget)
    if request_budget is not None and request_budget <= 0:
        return {
            **unknown("budget_exhausted", version=bundle.version, provider=bundle.provider),
            "_request_count": 0,
        }
    target = cache_path(output_dir, digest, version=bundle.version)
    if not key:
        return {
            **unknown("missing_key", version=bundle.version, provider=bundle.provider),
            "_request_count": 0,
        }
    request_made = False
    usage: dict[str, int] = {}
    billed_cost_usd: float | None = None
    diagnostic: dict[str, str] = {}
    try:
        if bundle.provider == "openrouter-decisions":
            endpoint = DECISIONS_ENDPOINT
            body = decisions_body(bundle, path)
        else:
            image = target_jpeg(path)
            endpoint = ENDPOINT
            body = {
                "model": MODEL,
                "temperature": 0,
                "max_tokens": 512,
                "reasoning_budget": 128,
                "stream": False,
                "response_format": {"type": "json_object"},
                "messages": [
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user", "content": [image_part(bundle.images[0]), {"type": "text", "text": REFERENCE_A_PROMPT}]},
                    {"role": "assistant", "content": json.dumps(bundle.expected[0], sort_keys=True, separators=(",", ":"))},
                    {"role": "user", "content": [image_part(bundle.images[1]), {"type": "text", "text": REFERENCE_B_PROMPT}]},
                    {"role": "assistant", "content": json.dumps(bundle.expected[1], sort_keys=True, separators=(",", ":"))},
                    {"role": "user", "content": [image_part(image), {"type": "text", "text": TARGET_PROMPT}]},
                ],
            }
        request_made = True
        response = await client.post(endpoint, headers={"Authorization": f"Bearer {key}"}, json=body)
        if response.status_code != 200:
            failure = unknown(f"model_http_{response.status_code}", version=bundle.version, provider=bundle.provider)
            _cache_result(target, failure, retry_after=time.time() + 300)
            return {**failure, "_request_made": True, "_request_count": 1}
        payload = response.json()
        raw_usage = payload.get("usage", {})
        if isinstance(raw_usage, dict):
            if bundle.provider == "openrouter-decisions":
                raw_cost = raw_usage.get("cost")
                if type(raw_cost) in (int, float) and 0 <= raw_cost <= 100 and math.isfinite(raw_cost):
                    billed_cost_usd = float(raw_cost)
                raw_usage = {
                    "prompt_tokens": raw_usage.get("input_tokens"),
                    "completion_tokens": raw_usage.get("output_tokens"),
                    "total_tokens": raw_usage.get("total_tokens"),
                }
            usage = {
                field: value
                for field in ("prompt_tokens", "completion_tokens", "total_tokens")
                if type(value := raw_usage.get(field)) is int and 0 <= value <= 1_000_000
            }
            if bundle.provider == "openrouter-decisions" and "total_tokens" not in usage and {
                "prompt_tokens", "completion_tokens"
            } <= usage.keys():
                total = usage["prompt_tokens"] + usage["completion_tokens"]
                if total <= 1_000_000:
                    usage["total_tokens"] = total
        if bundle.provider == "openrouter-decisions":
            result = validate_decisions_reply(payload, version=bundle.version)
            diagnostic["content_format"] = "typed_decisions"
            served_model = payload.get("model")
            if isinstance(served_model, str) and re.fullmatch(r"[A-Za-z0-9._/:-]{1,120}", served_model):
                diagnostic["served_model"] = served_model
        else:
            choice = payload["choices"][0]
            finish = choice.get("finish_reason", "")
            diagnostic["finish_reason"] = finish if isinstance(finish, str) and re.fullmatch(r"[a-z_]{1,32}", finish) else "other"
            content = choice["message"]["content"]
            diagnostic["content_format"] = content_format(content)
            result = validate_reply(content, version=bundle.version)
        if usage:
            result["_usage"] = usage
        if billed_cost_usd is not None:
            result["_billed_cost_usd"] = billed_cost_usd
        result["_diagnostic"] = diagnostic
    except (httpx.HTTPError, OSError, ValueError, KeyError, IndexError, TypeError, AttributeError, OverflowError) as exc:
        failure = unknown(f"model_failed_{type(exc).__name__}", version=bundle.version, provider=bundle.provider)
        if usage:
            failure["_usage"] = usage
        if billed_cost_usd is not None:
            failure["_billed_cost_usd"] = billed_cost_usd
        if diagnostic:
            failure["_diagnostic"] = diagnostic
        _cache_result(target, failure, retry_after=time.time() + 300)
        return {**failure, "_request_made": request_made, "_request_count": int(request_made)}
    _cache_result(target, result)
    return {**result, "_request_made": True, "_request_count": 1}
