"""Bounded hosted screenshot and app recognition for retained image slices.

Results are machine proposals. The score is the model's reported certainty, not a calibrated
probability, and neither a filename nor OCR topics establish an app identity.
"""

from __future__ import annotations

import base64
import hashlib
import json
import re
import time
import uuid
from pathlib import Path
from typing import Any

import httpx

from .image_decode import jpeg_derivative

MODEL = "meta/llama-3.2-11b-vision-instruct"
ENDPOINT = "https://integrate.api.nvidia.com/v1/chat/completions"
PROMPT_VERSION = "screenshot-app-ui-v2"
PREPROCESS_VERSION = "jpeg1024-q75-v1"
VERSION = f"{MODEL}:{PROMPT_VERSION}:{PREPROCESS_VERSION}"
APPS = frozenset(
    {"facebook-messenger", "messages", "snapchat", "whatsapp", "instagram", "browser", "other", "mixed", "unknown"}
)
DECISIONS = frozenset({"yes", "no", "unknown"})
PROMPT = (
    "Classify THIS image from visible pixels and app UI controls. A cropped or renamed image can "
    "be a screenshot; a PNG alone proves nothing. Set screenshot_decision to yes, no, or unknown. For yes, "
    "identify app from visible chrome/logo/navigation, not message topics or names. App must be "
    "facebook-messenger, messages, snapchat, whatsapp, instagram, browser, other, mixed, or "
    "unknown. Use mixed only if multiple app UIs are visible. Confidence is an uncalibrated "
    "integer 0..100; set it to 0 when app is unknown or screenshot_decision is no/unknown. Basis must briefly "
    "describe visual UI, without quoting conversation content. Return ONLY JSON with exactly four "
    "keys. Example: {\"screenshot_decision\":\"yes\",\"app\":\"messages\",\"confidence\":85,"
    "\"basis\":\"visible chat header and compose controls\"}. "
    "Now return JSON for THIS image."
)

FENCE = re.compile(r"\A```(?:json)?\s*\n(?P<body>[\s\S]*?)\n```\s*\Z", re.IGNORECASE)


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


def validate_reply(content: str) -> dict[str, Any]:
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
        if result.get("_diagnostic"):
            payload["diagnostic"] = result["_diagnostic"]
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
                return {**cached, "_cache_hit": True, "_diagnostic": stored.get("diagnostic", {})}
        except (OSError, ValueError, TypeError, AttributeError):
            pass
    if not key:
        return unknown("missing_key")
    request_made = False
    usage: dict[str, int] = {}
    diagnostic: dict[str, str] = {}
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
        result = validate_reply(content)
        if usage:
            result["_usage"] = usage
        result["_diagnostic"] = diagnostic
    except (httpx.HTTPError, OSError, ValueError, KeyError, IndexError, TypeError) as exc:
        failure = unknown(f"model_failed_{type(exc).__name__}")
        if usage:
            failure["_usage"] = usage
        if diagnostic:
            failure["_diagnostic"] = diagnostic
        _cache_result(target, failure, retry_after=time.time() + 300)
        return {**failure, "_request_made": request_made}
    _cache_result(target, result)
    return {**result, "_request_made": True}
