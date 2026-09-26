"""kimi-k3 on NVIDIA NIM: which reasoning mode to ask for, chosen per prompt.

Measured live 2026-09-25 against NIM ``moonshotai/kimi-k3``:

- thinking OFF: short prompts answered cleanly (17 of 17), but prompts of ~10k and
  ~43k tokens came back as 32 '!' tokens (3 of 3).
- thinking ON: about 1 short prompt in 8 came back empty (the same 32 junk tokens, in the
  reasoning field); a 43k-token prompt answered correctly once direct and was junk once
  through OpenCode.

So a short prompt asks with thinking OFF first and a long one with thinking ON first; the
one retry (``model_replies.ask_json``) uses the other mode. Every attempt gets a fresh
model object, because a batch shares one provider across concurrent calls.

Byline: Claude Code · Opus 5.5 · 2026-09-25 (owner decision relayed by the parent session)
"""

from __future__ import annotations

import os
from typing import Any

from agno.models.message import Message
from agno.models.openai.like import OpenAILike

NVIDIA_BASE_URL_DEFAULT = "https://integrate.api.nvidia.com/v1"
KIMI_K3_MODEL_ID = "moonshotai/kimi-k3"
SHORT_PROMPT_CHARS = 32_000  # about 8k tokens at ~4 characters per token


def is_kimi_k3_on_nim(model: Any) -> bool:
    """Whether *model* calls moonshotai/kimi-k3 on the NVIDIA NIM endpoint."""
    base = str(getattr(model, "base_url", "") or "").rstrip("/")
    nim = os.getenv("NVIDIA_BASE_URL", NVIDIA_BASE_URL_DEFAULT).rstrip("/")
    return getattr(model, "id", None) == KIMI_K3_MODEL_ID and base == nim


def json_mode(model: Any) -> dict[str, Any]:
    """``aresponse`` arguments for a JSON answer: JSON mode for kimi-k3 on NIM, else none."""
    return {"response_format": {"type": "json_object"}} if is_kimi_k3_on_nim(model) else {}


def _with_thinking(model: Any, thinking: bool) -> OpenAILike:
    """A fresh copy of a kimi-k3-on-NIM model that asks for *thinking*; the shared one is untouched."""
    return OpenAILike(
        id=model.id,
        api_key=model.api_key,
        base_url=model.base_url,
        extra_body={"chat_template_kwargs": {"thinking": thinking}},
        retries=model.retries,
        delay_between_retries=model.delay_between_retries,
        exponential_backoff=model.exponential_backoff,
        temperature=model.temperature,
        max_tokens=model.max_tokens,
    )


def attempts(model: Any, messages: list[Message]) -> list[tuple[str, Any]]:
    """``(label, model)`` for each attempt: the mode that suits the prompt first, then the other."""
    if not is_kimi_k3_on_nim(model):
        return [("first try", model), ("retry", model)]
    long_prompt = sum(len(str(message.content or "")) for message in messages) > SHORT_PROMPT_CHARS
    order = (True, False) if long_prompt else (False, True)
    return [(f"thinking {'on' if thinking else 'off'}", _with_thinking(model, thinking)) for thinking in order]
