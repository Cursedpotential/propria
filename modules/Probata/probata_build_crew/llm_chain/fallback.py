"""FallbackLLM — try a list of CrewAI model strings in order until one answers.

CrewAI 1.15 has no cross-provider fallback of its own (its only "fallback" is the LiteLLM path for unknown
providers), so this BaseLLM wrapper delegates each call to the first member that succeeds. Members are built
lazily with ``crewai.llm.LLM(model=...)`` so every native provider (ollama, openrouter, gemini, hosted_vllm, ...)
keeps its own client, events and token accounting. Any exception from a member (429, 5xx, timeout, connection
error, context overflow) moves to the next member; the failure list is kept on the instance for reporting.

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import sys
from typing import Any

from crewai.llm import LLM
from crewai.llms.base_llm import BaseLLM
from pydantic import Field, PrivateAttr


def _looks_like_tool_calls(result: Any) -> bool:
    """True when a provider handed back tool-call objects instead of text."""
    if isinstance(result, (str, bytes)) or result is None:
        return False
    if isinstance(result, list):
        return bool(result) and all(hasattr(item, "function") or (isinstance(item, dict) and "function" in item) for item in result)
    return hasattr(result, "tool_calls") and not getattr(result, "content", None)


_FORCED_FINAL_MARKERS = (
    "Maximum iterations reached. Requesting final answer.",      # crewai/utilities/agent_utils.py::handle_max_iterations_exceeded
    "MUST give your absolute best final answer",                  # translations/en.json errors.force_final_answer
)


def _is_forced_final_answer(messages: str | list[Any]) -> bool:
    """True only when CrewAI is forcing the final answer after max_iter (the one place tool calls must not come back)."""
    if isinstance(messages, str):
        return any(m in messages for m in _FORCED_FINAL_MARKERS)
    for msg in reversed(list(messages)[-4:]):
        content = msg.get("content") if isinstance(msg, dict) else getattr(msg, "content", None)
        if isinstance(content, str) and any(m in content for m in _FORCED_FINAL_MARKERS):
            return True
    return False


def _as_message_list(messages: str | list[Any]) -> list[Any]:
    if isinstance(messages, str):
        return [{"role": "user", "content": messages}]
    return list(messages)


class FallbackLLM(BaseLLM):
    """Ordered chain of model strings; the first that succeeds answers."""

    chain: list[str] = Field(default_factory=list, description="Model strings in priority order, e.g. ['ollama/nemotron-3-super', 'openrouter/google/gemma-4-31b-it:free']")
    context_window: int | None = Field(default=None, description="Explicit context window in tokens. CrewAI assumes ~8k for models it does not know, which makes respect_context_window summarize far too early; set the real (conservative) value for the chain.")

    _members: list[BaseLLM] = PrivateAttr(default_factory=list)
    _failures: list[str] = PrivateAttr(default_factory=list)

    def __init__(self, chain: list[str] | None = None, **kwargs: Any) -> None:
        chain = list(chain or kwargs.pop("chain", []) or [])
        if not chain:
            raise ValueError("FallbackLLM needs at least one model string in `chain`")
        kwargs.setdefault("model", chain[0])
        kwargs.setdefault("provider", chain[0].split("/", 1)[0])
        super().__init__(chain=chain, **kwargs)

    # -- members -------------------------------------------------------------------------------------
    def members(self) -> list[BaseLLM]:
        if not self._members:
            shared: dict[str, Any] = {}
            if self.max_tokens is not None:
                shared["max_tokens"] = self.max_tokens
            if self.temperature is not None:
                shared["temperature"] = self.temperature
            for spec in self.chain:
                try:
                    self._members.append(LLM(model=spec, **shared))
                except Exception as exc:  # noqa: BLE001 - a member that cannot even be built is skipped, not fatal
                    self._failures.append(f"{spec}: build failed: {type(exc).__name__}: {str(exc)[:200]}")
                    print(f"[FallbackLLM] cannot build {spec}: {exc}", file=sys.stderr)
            if not self._members:
                raise RuntimeError("FallbackLLM: no member could be built:\n" + "\n".join(self._failures))
        return self._members

    @property
    def failures(self) -> list[str]:
        return list(self._failures)

    # -- BaseLLM contract ----------------------------------------------------------------------------
    def call(
        self,
        messages: str | list[Any],
        tools: list[dict[str, Any]] | None = None,
        callbacks: list[Any] | None = None,
        available_functions: dict[str, Any] | None = None,
        from_task: Any = None,
        from_agent: Any = None,
        response_model: Any = None,
    ) -> str | Any:
        errors: list[str] = []
        for member in self.members():
            try:
                if self.stop and getattr(member, "stop", None) != self.stop:
                    member.stop = list(self.stop)
                result = member.call(
                    messages,
                    tools=tools,
                    callbacks=callbacks,
                    available_functions=available_functions,
                    from_task=from_task,
                    from_agent=from_agent,
                    response_model=response_model,
                )
                if _looks_like_tool_calls(result) and response_model is None and not tools and _is_forced_final_answer(messages):
                    # CrewAI 1.15 edge: at "Maximum iterations reached" the executor asks for a final answer, the
                    # model answers with a tool call, and the list lands in TaskOutput.raw (a str field) -> crash.
                    # Only at that forced-final step (never during the normal ReAct loop, where tool calls are the
                    # expected reply) ask the same member once more for plain text, with no tools offered.
                    print(f"[FallbackLLM] {member.model} returned tool calls where text was required; re-asking for text", file=sys.stderr)
                    retry_messages = _as_message_list(messages) + [
                        {
                            "role": "user",
                            "content": "Tool calls are not accepted at this point. Write your complete final answer now as plain markdown text, using only the information already gathered. Do not call any tool.",
                        }
                    ]
                    result = member.call(retry_messages, tools=None, callbacks=callbacks, from_task=from_task, from_agent=from_agent)
                    if _looks_like_tool_calls(result):
                        result = "FINAL ANSWER UNAVAILABLE: the model kept emitting tool calls after the iteration limit. Partial work is in the execution log."
                return result
            except Exception as exc:  # noqa: BLE001 - deliberate: any member failure falls through to the next
                line = f"{member.model}: {type(exc).__name__}: {str(exc)[:300]}"
                errors.append(line)
                self._failures.append(line)
                print(f"[FallbackLLM] {line} -> trying next member", file=sys.stderr)
        raise RuntimeError("FallbackLLM: every member failed:\n" + "\n".join(errors))

    def supports_function_calling(self) -> bool:
        return bool(getattr(self.members()[0], "supports_function_calling", lambda: True)())

    def supports_stop_words(self) -> bool:
        return bool(self.members()[0].supports_stop_words())

    def get_context_window_size(self) -> int:
        if self.context_window:
            return int(self.context_window)
        return min(m.get_context_window_size() for m in self.members())

    def supports_multimodal(self) -> bool:
        return False
