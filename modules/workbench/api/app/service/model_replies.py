"""Read a model's JSON reply for Workbench classification and sentiment.

One job: ask the model, take its reply apart, and never invent an answer. An empty
reply, or one the caller's reader rejects, gets one retry; after that the call fails with
ModelReplyError (the routes answer HTTP 502 naming the model) instead of a fallback such
as the old silent ``categories[0]`` / 0.5 classification.

Byline: Claude Code · Opus 5.5 · 2026-09-25 (owner decision relayed by the parent session:
retry an empty reply once, then fail clearly; delete the silent fallback)
"""

from __future__ import annotations

import json
from collections.abc import Callable
from typing import Any, Protocol, TypeVar

from agno.models.message import Message

REPLY_ATTEMPTS = 2  # the first answer plus one retry
_T = TypeVar("_T")


class ModelReplyError(Exception):
    """The model answered, but gave nothing usable in any attempt. Routes map this to HTTP 502."""


class Answers(Protocol):
    """Anything with Agno's ``aresponse`` coroutine."""

    async def aresponse(self, messages: list[Message], **kwargs: Any) -> Any: ...


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
    model: Answers,
    messages: list[Message],
    read: Callable[[dict[str, Any]], _T],
    request: dict[str, Any] | None = None,
) -> tuple[_T, str]:
    """Ask *model* for a JSON object and turn it into a result with *read*.

    *request* holds extra ``aresponse`` keyword arguments (e.g. ``response_format``).
    *read* raises KeyError/TypeError/ValueError for an object it cannot use. Returns
    ``(read(object), raw_text)``; raises ModelReplyError after REPLY_ATTEMPTS attempts.
    """
    problem = "empty reply"
    for _attempt in range(REPLY_ATTEMPTS):
        response = await model.aresponse(messages, **(request or {}))
        raw = str(response.content).strip() if response.content else ""
        if not raw:
            problem = "empty reply"
            continue
        try:
            return read(json_object(raw)), raw
        except (KeyError, TypeError, ValueError) as error:
            problem = f"unparseable reply ({error})"
    raise ModelReplyError(
        f"Model '{getattr(model, 'id', 'unknown')}' gave no usable reply in {REPLY_ATTEMPTS} attempts; "
        f"last attempt: {problem}."
    )
