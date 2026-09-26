"""Optional model-gateway client. Fail closed. Never court-safe.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Uses the service name in settings. This desktop has no Docker, so
the default host will not resolve unless the owner deploys it.
"""

from __future__ import annotations

from dataclasses import dataclass

import httpx

from legal_workspace.config import get_settings
from legal_workspace.domain.provider_grid import confidential_blocked_reason

SYSTEM = (
    "You are a narrow legal-workspace helper for one Michigan family matter. "
    "Your output is a hypothesis, not a finding, not legal advice, and not "
    "court-safe. Do not invent citations, docket numbers, hearing dates, "
    "or facts. Do not approve, file, serve, or diagnose."
)


@dataclass(frozen=True)
class GatewayResult:
    ok: bool
    text: str
    model: str


def invoke_chat(
    prompt: str,
    model: str,
    client: httpx.Client | None = None,
    *,
    confidential: bool = False,
) -> GatewayResult:
    if confidential and confidential_blocked_reason(model) is not None:
        return GatewayResult(ok=False, text="confidential_blocked", model=model)
    settings = get_settings()
    url = settings.model_gateway_base_url.rstrip("/") + "/v1/chat/completions"
    payload = {
        "model": model,
        "messages": [
            {"role": "system", "content": SYSTEM},
            {"role": "user", "content": prompt},
        ],
        "temperature": 0,
    }
    closer = False
    if client is None:
        client = httpx.Client(timeout=2.0)
        closer = True
    try:
        response = client.post(url, json=payload)
        response.raise_for_status()
        body = response.json()
        text = body["choices"][0]["message"]["content"]
        return GatewayResult(ok=True, text=text, model=model)
    except Exception as exc:
        return GatewayResult(
            ok=False,
            text=f"Gateway unreachable or refused: {exc.__class__.__name__}. No model output.",
            model="gateway-unreachable",
        )
    finally:
        if closer:
            client.close()
