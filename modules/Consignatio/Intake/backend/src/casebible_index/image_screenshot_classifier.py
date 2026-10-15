"""Bounded hosted screenshot and app recognition for retained image slices.

Results are machine proposals. The score is the model's reported certainty, not a calibrated
probability, and neither a filename nor OCR topics establish an app identity.
"""

from __future__ import annotations

import base64
import hashlib
import json
import time
import uuid
from pathlib import Path
from typing import Any

import httpx

from .image_decode import jpeg_derivative

MODEL = "meta/llama-3.2-11b-vision-instruct"
ENDPOINT = "https://integrate.api.nvidia.com/v1/chat/completions"
PROMPT_VERSION = "screenshot-app-ui-v1"
PREPROCESS_VERSION = "jpeg1024-q75-v1"
VERSION = f"{MODEL}:{PROMPT_VERSION}:{PREPROCESS_VERSION}"
APPS = frozenset(
    {"facebook-messenger", "messages", "snapchat", "whatsapp", "instagram", "browser", "other", "mixed", "unknown"}
)
DECISIONS = frozenset({"yes", "no", "unknown"})
PROMPT = (
    "Look at this image's visible pixels. Decide whether it is a screenshot of a digital UI. "
    "A renamed or cropped screenshot may have no filename or phone frame. A PNG without camera "
    "metadata is not necessarily a screenshot. If uncertain, answer unknown. For a screenshot, "
    "identify the app from visible interface chrome, logos, navigation and controls; do not "
    "infer the app from conversation topics, names, OCR text alone, or file metadata. Use mixed "
    "only when multiple app UIs are actually visible; browser when browser chrome identifies "
    "the container; other for an identifiable unlisted app; unknown if app UI is inconclusive. "
    "Return ONLY a JSON object with exactly these keys: screenshot_decision (yes/no/unknown), "
    "screenshot_app (facebook-messenger/messages/snapchat/whatsapp/instagram/browser/other/mixed/unknown), "
    "screenshot_app_confidence (integer 0..100), screenshot_app_basis (short visual UI reason). "
    "The confidence is your uncalibrated estimate. For no or unknown screenshot decision, "
    "set app unknown and confidence 0. If it is a screenshot but the app is unknown, also set "
    "confidence 0. Do not quote message content or personal names."
)


def unknown(status: str, basis: str = "") -> dict[str, Any]:
    """Return a safe unknown proposal with a controlled status and no source content.

    Inputs: non-sensitive status and short reason. Output: flat classifier fields. Side effects:
    none. Pick this for disabled, missing-key and provider failures instead of guessing an app.
    """
    return {
        "screenshot_decision": "unknown",
        "screenshot_app": "unknown",
        "screenshot_app_confidence": 0,
        "screenshot_app_basis": basis[:120],
        "screenshot_classifier_version": VERSION,
        "screenshot_classifier_provider": "nvidia-nim",
        "screenshot_classifier_status": status,
    }


def validate_reply(content: str) -> dict[str, Any]:
    """Validate the model's exact JSON contract before any proposal is persisted.

    Inputs: one provider response content string. Output: normalized flat fields. Side effects:
    none. Pick this after the hosted call; rejected output becomes an unknown failure upstream.
    """
    data = json.loads(content)
    expected = {"screenshot_decision", "screenshot_app", "screenshot_app_confidence", "screenshot_app_basis"}
    if not isinstance(data, dict) or set(data) != expected:
        raise ValueError("classifier JSON fields are invalid")
    decision, app, confidence, basis = (data[k] for k in (
        "screenshot_decision", "screenshot_app", "screenshot_app_confidence", "screenshot_app_basis"
    ))
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
        "screenshot_classifier_version": VERSION,
        "screenshot_classifier_provider": "nvidia-nim",
        "screenshot_classifier_status": "model_ok" if decision != "unknown" else "model_unknown",
    }


def cache_path(output_dir: Path, digest: str) -> Path:
    """Locate one versioned SHA-256 result in the existing image output lake.

    Inputs: output lake root and retained-byte SHA-256. Output: JSON cache path. Side effects:
    none. Pick this for cross-slice reuse without creating a source catalog.
    """
    if len(digest) != 64 or any(c not in "0123456789abcdef" for c in digest):
        raise ValueError("invalid retained image SHA-256")
    key = hashlib.sha256(f"{digest}:{VERSION}".encode()).hexdigest()
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
        partial.write_text(json.dumps(payload, sort_keys=True), encoding="utf-8")
        partial.replace(target)
    except OSError:
        # Classification remains usable even if an optional cache write fails.
        return


async def classify(path: Path, digest: str, output_dir: Path, key: str, client: httpx.AsyncClient) -> dict[str, Any]:
    """Classify a retained image through the hosted vision model, reusing a versioned result.

    Inputs: fetched path, its SHA-256, output lake, NVIDIA key and HTTP client. Output: flat
    proposal with explicit model status. Side effects: bounded decode, one HTTPS request on cache
    miss, atomic JSON cache write for validated replies. Pick this within image_facts; OCR and
    embedding have separate jobs. Provider errors return unknown without response body or secrets.
    """
    target = cache_path(output_dir, digest)
    if target.is_file():
        try:
            stored = json.loads(target.read_text(encoding="utf-8"))
            cached = stored.get("result", {})
            if cached.get("screenshot_classifier_version") == VERSION and (cached.get("screenshot_classifier_status") in {"model_ok", "model_unknown"} or stored.get("retry_after", 0) > time.time()):
                return {**cached, "_cache_hit": True}
        except (OSError, ValueError, TypeError, AttributeError):
            pass
    if not key:
        return unknown("missing_key")
    request_made = False
    try:
        image = jpeg_derivative(path, edge=1024, quality=75)
        if len(image) > 1_000_000:
            image = jpeg_derivative(path, edge=768, quality=65)
        if len(image) > 1_000_000:
            return unknown("preprocess_too_large")
        body = {
            "model": MODEL,
            "temperature": 0,
            "max_tokens": 256,
            "stream": False,
            "response_format": {"type": "json_object"},
            "messages": [{"role": "user", "content": [
                {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64," + base64.b64encode(image).decode()}},
                {"type": "text", "text": PROMPT},
            ]}],
        }
        request_made = True
        response = await client.post(ENDPOINT, headers={"Authorization": f"Bearer {key}"}, json=body)
        if response.status_code != 200:
            failure = unknown(f"model_http_{response.status_code}")
            _cache_result(target, failure, retry_after=time.time() + 300)
            return {**failure, "_request_made": True}
        payload = response.json()
        result = validate_reply(payload["choices"][0]["message"]["content"])
        raw_usage = payload.get("usage", {})
        if isinstance(raw_usage, dict):
            usage = {
                field: value
                for field in ("prompt_tokens", "completion_tokens", "total_tokens")
                if type(value := raw_usage.get(field)) is int and 0 <= value <= 1_000_000
            }
            if usage:
                result["_usage"] = usage
    except (httpx.HTTPError, OSError, ValueError, KeyError, IndexError, TypeError) as exc:
        failure = unknown(f"model_failed_{type(exc).__name__}")
        _cache_result(target, failure, retry_after=time.time() + 300)
        return {**failure, "_request_made": request_made}
    _cache_result(target, result)
    return {**result, "_request_made": True}
