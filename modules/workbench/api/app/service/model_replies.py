"""Read a model's JSON reply for Workbench classification and sentiment.

One job: ask the model, take its reply apart, and never invent an answer. A reply that is
empty, junk (one character repeated, e.g. 32 × '!' from kimi-k3 on NIM) or invalid JSON
fails that attempt; after the last attempt the call raises ModelReplyError, which the
routes answer with HTTP 502 naming the model and each attempt's failure. This replaces
the old silent ``categories[0]`` / 0.5 classification fallback.

Byline: Claude Code · Opus 5.5 · 2026-09-25 (owner decision relayed by the parent session:
one retry, then fail clearly; delete the silent fallback; detect empty and junk replies;
one INFO line per attempt — model, mode, prompt size, outcome, latency; never prompt text)
"""

from __future__ import annotations

import json
import logging
import time
from collections.abc import Callable, Sequence
from typing import Any, Protocol, TypeVar

from agno.models.message import Message

logger = logging.getLogger(__name__)
_T = TypeVar("_T")


class ModelReplyError(Exception):
    """The model answered, but gave nothing usable in any attempt. Routes map this to HTTP 502."""


class Answers(Protocol):
    """Anything with Agno's ``aresponse`` coroutine."""

    async def aresponse(self, messages: list[Message], **kwargs: Any) -> Any: ...


def reply_problem(raw: str) -> str | None:
    """Name what is wrong with a reply that carries nothing; None when there is text to parse."""
    if not raw:
        return "empty reply"
    if len(raw) > 1 and len(set(raw)) == 1:
        return f"junk reply ({len(raw)} x {raw[0]!r})"
    return None


def _log_attempt(model: Any, mode: str, prompt_chars: int, outcome: str, started: float) -> None:
    """One INFO line per model attempt: no prompt text, no reply text, no credentials."""
    logger.info(
        "model attempt model=%s mode=%s prompt_chars=%d outcome=%s latency_ms=%d",
        getattr(model, "id", "unknown"),
        mode,
        prompt_chars,
        outcome,
        round((time.perf_counter() - started) * 1000),
    )


def json_object(raw: str) -> dict[str, Any]:
    """Return the JSON object in *raw* (code fences tolerated); ValueError when there is none."""
    start, end = raw.find("{"), raw.rfind("}") + 1
    if start < 0 or end <= start:
        raise ValueError("no JSON object in the reply")
    parsed = json.loads(raw[start:end])
    if not isinstance(parsed, dict):
        raise ValueError("the reply's JSON is not an object")
    return parsed


async def ask_json(
    attempts: Sequence[tuple[str, Answers]],
    messages: list[Message],
    read: Callable[[dict[str, Any]], _T],
    request: dict[str, Any] | None = None,
) -> tuple[_T, str]:
    """Ask each ``(label, model)`` attempt in turn until one reply reads cleanly.

    *request* holds extra ``aresponse`` keyword arguments (e.g. ``response_format``).
    *read* raises KeyError/TypeError/ValueError for an object it cannot use. Returns
    ``(read(object), raw_text)``; raises ModelReplyError when every attempt fails.
    """
    problems: list[str] = []
    prompt_chars = sum(len(str(message.content or "")) for message in messages)
    for label, model in attempts:
        started = time.perf_counter()
        try:
            response = await model.aresponse(messages, **(request or {}))
        except Exception as error:
            _log_attempt(model, label, prompt_chars, f"error ({type(error).__name__})", started)
            raise
        raw = str(response.content).strip() if response.content else ""
        problem = reply_problem(raw)
        if problem is None:
            try:
                result = read(json_object(raw))
            except (KeyError, TypeError, ValueError) as error:
                problem = f"invalid JSON reply ({error})"
            else:
                _log_attempt(model, label, prompt_chars, "ok", started)
                return result, raw
        _log_attempt(model, label, prompt_chars, problem.split(" reply", 1)[0], started)
        problems.append(f"{label}: {problem}")
    model_id = getattr(attempts[0][1], "id", "unknown") if attempts else "unknown"
    raise ModelReplyError(
        f"Model '{model_id}' gave no usable reply in {len(attempts)} attempts ({'; '.join(problems)})."
    )
