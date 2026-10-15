"""Bounded hosted screenshot and app recognition for retained image slices.

Results are machine proposals. The score is the model's reported certainty, not a calibrated
probability, and neither a filename nor OCR topics establish an app identity.
"""

from __future__ import annotations

import base64
import hashlib
import io
import json
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


@dataclass(frozen=True)
class ReferenceBundle:
    """Two hash-checked derived UI references and their cache/provenance fingerprint.

    Inputs: validated manifest and JPEG bytes. Output: immutable runtime bundle. Side effects:
    none. Pick this only for classifier example turns, never OCR or embedding.
    """

    version: str
    images: tuple[bytes, bytes]
    expected: tuple[dict[str, Any], dict[str, Any]]


def load_reference_bundle(manifest_path: Path, output_dir: Path) -> ReferenceBundle:
    """Validate exactly two reviewed reference derivatives beneath the output lake.

    Inputs: absolute manifest path and attached output root. Output: hash-checked JPEG bundle
    with a version derived from model, prompt, preprocessing, labels and source/derived hashes. Side
    effects: bounded reads only. Pick this once per facts activity before any provider request.
    Invalid or unavailable inputs raise ValueError/OSError for a safe unknown result upstream.
    """
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
    canonical = json.dumps({"base_version": VERSION, "set_version": set_version, "references": refs}, sort_keys=True, separators=(",", ":"))
    fingerprint = hashlib.sha256(canonical.encode()).hexdigest()
    return ReferenceBundle(
        f"{VERSION}:{fingerprint}",
        (images[0], images[1]),
        (refs[0]["expected"], refs[1]["expected"]),
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


def image_part(jpeg: bytes) -> dict[str, Any]:
    """Wrap one bounded JPEG as a provider image part for its own user turn.

    Inputs: validated reference or derived target JPEG bytes. Output: image_url content part.
    Side effects: none. Pick this for the classifier's native multimodal example/target turns.
    """
    return {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64," + base64.b64encode(jpeg).decode()}}


def unknown(status: str, basis: str = "", *, version: str = VERSION) -> dict[str, Any]:
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
        "screenshot_classifier_provider": "nvidia-nim",
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
        if result.get("_diagnostic"):
            payload["diagnostic"] = result["_diagnostic"]
        partial.write_text(json.dumps(payload, sort_keys=True), encoding="utf-8")
        partial.replace(target)
    except OSError:
        # Classification remains usable even if an optional cache write fails.
        return


async def classify(
    path: Path | io.BytesIO | bytes,
    digest: str,
    output_dir: Path,
    key: str,
    client: httpx.AsyncClient,
    bundle: ReferenceBundle,
) -> dict[str, Any]:
    """Classify a retained image through the hosted vision model, reusing a versioned result.

    Inputs: fetched path, its SHA-256, output lake, NVIDIA key and HTTP client. Output: flat
    proposal with explicit model status. Side effects: bounded decode, one HTTPS request on cache
    miss, atomic JSON cache write for validated replies. Pick this within image_facts; OCR and
    embedding have separate jobs. Provider errors return unknown without response body or secrets.
    """
    target = cache_path(output_dir, digest, version=bundle.version)
    if target.is_file():
        try:
            stored = json.loads(target.read_text(encoding="utf-8"))
            cached = stored.get("result", {})
            if cached.get("screenshot_classifier_version") == bundle.version and (cached.get("screenshot_classifier_status") in {"model_ok", "model_unknown"} or stored.get("retry_after", 0) > time.time()):
                return {**cached, "_cache_hit": True, "_diagnostic": stored.get("diagnostic", {})}
        except (OSError, ValueError, TypeError, AttributeError):
            pass
    if not key:
        return unknown("missing_key", version=bundle.version)
    request_made = False
    usage: dict[str, int] = {}
    diagnostic: dict[str, str] = {}
    try:
        image = target_jpeg(path)
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
        response = await client.post(ENDPOINT, headers={"Authorization": f"Bearer {key}"}, json=body)
        if response.status_code != 200:
            failure = unknown(f"model_http_{response.status_code}", version=bundle.version)
            _cache_result(target, failure, retry_after=time.time() + 300)
            return {**failure, "_request_made": True}
        payload = response.json()
        raw_usage = payload.get("usage", {})
        if isinstance(raw_usage, dict):
            usage = {
                field: value
                for field in ("prompt_tokens", "completion_tokens", "total_tokens")
                if type(value := raw_usage.get(field)) is int and 0 <= value <= 1_000_000
            }
        choice = payload["choices"][0]
        finish = choice.get("finish_reason", "")
        diagnostic["finish_reason"] = finish if isinstance(finish, str) and re.fullmatch(r"[a-z_]{1,32}", finish) else "other"
        content = choice["message"]["content"]
        diagnostic["content_format"] = content_format(content)
        result = validate_reply(content, version=bundle.version)
        if usage:
            result["_usage"] = usage
        result["_diagnostic"] = diagnostic
    except (httpx.HTTPError, OSError, ValueError, KeyError, IndexError, TypeError) as exc:
        failure = unknown(f"model_failed_{type(exc).__name__}", version=bundle.version)
        if usage:
            failure["_usage"] = usage
        if diagnostic:
            failure["_diagnostic"] = diagnostic
        _cache_result(target, failure, retry_after=time.time() + 300)
        return {**failure, "_request_made": request_made}
    _cache_result(target, result)
    return {**result, "_request_made": True}
