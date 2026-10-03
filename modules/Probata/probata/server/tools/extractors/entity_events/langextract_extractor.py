"""LangExtract as an entity and event extractor, configured for kimi-k3 on NVIDIA NIM.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Google's ``langextract`` (PyPI, pinned in ``requirements-extractors.txt``) runs few-shot extraction over a window of
messages and aligns every extraction to the exact span it came from. The model is the default extractor's: the same
``ENTITY_MODEL_*`` environment, the same OpenAI-compatible NIM endpoint, the same rules:

- ``response_format: json_object``; the reply must parse as one JSON object.
- ``max_tokens`` at least 1500 (a reasoning model returns empty content below that).
- Empty content, a reply that is one character repeated, a reply cut off at the token limit, or invalid JSON is a
  failure: the call is retried once with thinking on. A second failure raises, so the Activity fails loudly with the
  reason. Nothing is ever replaced by a silent default.
- Thinking is off for a short prompt and on for a long one (a long prompt with thinking off came back as a run of
  ``!``). Models whose id contains ``glm`` are refused.
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from server.tools.extractors.entity_events.pages import Message, PageBuilder

EXTRACTOR = "langextract"

ENV_BASE_URL = "ENTITY_MODEL_BASE_URL"
ENV_MODEL_ID = "ENTITY_MODEL_ID"
ENV_API_KEY_FILE = "ENTITY_MODEL_API_KEY_FILE"
ENV_MAX_TOKENS = "ENTITY_MODEL_MAX_TOKENS"

DEFAULT_BASE_URL = "https://integrate.api.nvidia.com/v1"
DEFAULT_MODEL_ID = "moonshotai/kimi-k3"
DEFAULT_MAX_TOKENS = 6000
MIN_MAX_TOKENS = 1500

# Where kimi-k3 must think: with thinking off, prompts of roughly 10k-43k tokens came back as a run of "!".
LONG_PROMPT_TOKENS = 8000

# One call covers at most this many characters of the window.
MAX_CHAR_BUFFER = 6000

PROMPT = (
    "Read the text messages and extract (1) the people, places and organizations they mention and (2) the events "
    "worth putting on a timeline. You extract; you do not interpret, judge, diagnose, or infer motives, feelings or "
    "relationships. Every extraction_text must be copied exactly from the text, in order of appearance. Use the "
    "classes person, place, organization and event. An event is something that happened or was arranged "
    "(an appointment, a court date, a hand-off of a child, a trip, a move, an incident); its attributes give "
    "title and description restating what the message says and nothing more, type (one of appointment, court, "
    "medical, school, custody_exchange, travel, incident, communication, financial, residence, work, other), and "
    "date only when the message states a complete calendar date with a year as YYYY-MM-DD, otherwise when with the "
    "phrase as written."
)


class ExtractorUnavailable(Exception):
    """The extractor cannot run in this process (no key mounted, no package); the message says why."""


@dataclass(frozen=True)
class Config:
    """The model endpoint, read from the default extractor's environment."""

    base_url: str
    model_id: str
    api_key: str
    max_tokens: int


def config_from_env() -> Config:
    """Read the model configuration; raises :class:`ExtractorUnavailable` when no key file is mounted.

    A present but invalid configuration (a GLM model, too few tokens, a key file that is not one line) raises
    ``ValueError``: that is a deployment error, not a reason to skip.
    """
    key_file = os.environ.get(ENV_API_KEY_FILE, "").strip()
    if not key_file:
        raise ExtractorUnavailable(f"{ENV_API_KEY_FILE} is not set on this worker; no model key is mounted")
    path = Path(key_file)
    if not path.is_absolute():
        raise ValueError(f"{ENV_API_KEY_FILE} must be an absolute path")
    if not path.is_file():
        raise ExtractorUnavailable(f"{ENV_API_KEY_FILE} points at a file that is not mounted")
    key = path.read_text(encoding="utf-8").strip()
    # Tolerate a KEY=value line so an env-style secret file can be mounted.
    name, sep, value = key.partition("=")
    if sep and "KEY" in name and not any(c.isspace() for c in name):
        key = value.strip().strip("\"'")
    if not key or any(c.isspace() for c in key):
        raise ValueError(f"{ENV_API_KEY_FILE} does not hold a single key")
    model_id = os.environ.get(ENV_MODEL_ID, "").strip() or DEFAULT_MODEL_ID
    if "glm" in model_id.lower():
        raise ValueError(f"model {model_id!r} refused: GLM models are not used (owner rule 2026-09-25)")
    max_tokens = int(os.environ.get(ENV_MAX_TOKENS, "").strip() or DEFAULT_MAX_TOKENS)
    if max_tokens < MIN_MAX_TOKENS:
        raise ValueError(f"max tokens {max_tokens} is below {MIN_MAX_TOKENS}; a reasoning model returns empty content")
    base_url = (os.environ.get(ENV_BASE_URL, "").strip() or DEFAULT_BASE_URL).rstrip("/")
    return Config(base_url=base_url, model_id=model_id, api_key=key, max_tokens=max_tokens)


def estimate_tokens(prompt: str) -> int:
    """A conservative prompt size (about 3.5 characters a token), as the Go client estimates it."""
    return len(prompt) * 2 // 7 + 1


def reply_problem(content: str | None, finish_reason: str | None) -> str | None:
    """Why a reply is unusable, or None. Empty, one repeated character, cut off, or not one JSON object."""
    if content is None or not content.strip():
        return "empty reply"
    stripped = content.strip()
    if len(stripped) >= 4 and len(set(stripped)) == 1:
        return "junk reply: one character repeated"
    if finish_reason == "length":
        return "reply was cut off at the token limit"
    try:
        value = json.loads(stripped)
    except json.JSONDecodeError as error:
        return f"reply is not JSON: {error.msg}"
    if not isinstance(value, dict):
        return "reply is not one JSON object"
    return None


def window_text(window: list[Message]) -> tuple[str, list[tuple[int, int, str]]]:
    """The window as one text and the character span of each message body in it.

    Each line reads ``sender -> recipients: body``; the returned spans are (start, end, record_id) over the body only,
    so an extraction's span maps back to the message it came from.
    """
    parts: list[str] = []
    spans: list[tuple[int, int, str]] = []
    cursor = 0
    for message in window:
        body = " ".join(message.body.split("\n"))
        head = f"{message.sender or '?'} -> {', '.join(message.recipients) or '?'}: "
        start = cursor + len(head)
        parts.append(head + body + "\n")
        spans.append((start, start + len(body), message.record_id))
        cursor += len(parts[-1])
    return "".join(parts), spans


def _message_for(span: tuple[int, int] | None, surface: str, text: str, spans: list[tuple[int, int, str]]) -> str | None:
    """The record id of the message an extraction came from: by its aligned span, else by where the text occurs."""
    if span is not None:
        start, end = span
        for first, last, record_id in spans:
            if first <= start and end <= last:
                return record_id
    for first, last, record_id in spans:
        if surface and surface in text[first:last]:
            return record_id
    return None


_CLASS_KIND = {"person": "people", "place": "places", "organization": "organizations"}


def page_from_extractions(
    extractions: list[Any], window: list[Message], text: str, spans: list[tuple[int, int, str]],
    after_ordinal: int, *, version: str, model_id: str,
) -> dict[str, Any]:
    """Convert LangExtract extractions of one window into the shared page. An extraction that maps to no message is dropped."""
    builder = PageBuilder(EXTRACTOR, version, model_id=model_id)
    for extraction in extractions:
        surface = (getattr(extraction, "extraction_text", "") or "").strip()
        interval = getattr(extraction, "char_interval", None)
        span = None
        if interval is not None and interval.start_pos is not None and interval.end_pos is not None:
            span = (interval.start_pos, interval.end_pos)
        record_id = _message_for(span, surface, text, spans)
        if record_id is None or not surface:
            continue
        kind = getattr(extraction, "extraction_class", "")
        attributes = getattr(extraction, "attributes", None) or {}
        if kind in _CLASS_KIND:
            builder.add_entity(_CLASS_KIND[kind], surface, record_id, surface)
        elif kind == "event":
            date = attributes.get("date")
            when = attributes.get("when")
            builder.add_event(
                str(attributes.get("title") or surface),
                record_id,
                description=str(attributes.get("description") or ""),
                date=date if isinstance(date, str) else None,
                when=when if isinstance(when, str) else None,
                event_type=str(attributes.get("type") or "other"),
            )
    return builder.page(window, after_ordinal)


def _examples() -> list[Any]:
    from langextract import data

    return [
        data.ExampleData(
            text="self -> +18105550101: Morning Katherine, can you get Emma from Lincoln Elementary at 3? Court is on 2025-07-02.\n",
            extractions=[
                data.Extraction(extraction_class="person", extraction_text="Katherine"),
                data.Extraction(extraction_class="person", extraction_text="Emma"),
                data.Extraction(extraction_class="organization", extraction_text="Lincoln Elementary"),
                data.Extraction(
                    extraction_class="event",
                    extraction_text="get Emma from Lincoln Elementary at 3",
                    attributes={
                        "title": "School pickup", "description": "Asked Katherine to get Emma from Lincoln Elementary at 3",
                        "type": "custody_exchange", "when": "at 3",
                    },
                ),
                data.Extraction(
                    extraction_class="event",
                    extraction_text="Court is on 2025-07-02",
                    attributes={"title": "Court date", "description": "Court is on 2025-07-02", "type": "court", "date": "2025-07-02"},
                ),
            ],
        )
    ]


def build_model(config: Config) -> Any:
    """The LangExtract model for NIM: an OpenAI-compatible provider whose replies are validated and retried once with thinking on."""
    try:
        from langextract.core import exceptions
        from langextract.core import types as core_types
        from langextract.providers.openai import OpenAILanguageModel
    except ImportError as error:
        raise ExtractorUnavailable(f"langextract[openai] is not installed on this worker: {error}") from error

    class NimKimiModel(OpenAILanguageModel):
        """OpenAILanguageModel with the default extractor's reply rules (validate, retry once with thinking on)."""

        def _process_single_prompt(self, prompt: str, config: dict) -> Any:  # noqa: D401 - provider hook
            api_params = self._build_chat_completions_params(prompt, config)
            first = estimate_tokens(prompt) > LONG_PROMPT_TOKENS
            failure = "no attempt"
            for thinking in (first, True):
                params = dict(api_params)
                params["extra_body"] = {"chat_template_kwargs": {"thinking": thinking}}
                try:
                    response = self._client.chat.completions.create(**params)
                except Exception as error:  # the SDK's own error types are many; the runtime error keeps the cause
                    raise exceptions.InferenceRuntimeError(
                        f"NIM request failed: {error}", original=error, provider="OpenAI"
                    ) from error
                choice = response.choices[0] if response.choices else None
                content = choice.message.content if choice else None
                problem = reply_problem(content, choice.finish_reason if choice else None)
                if problem is None:
                    return core_types.ScoredOutput(score=1.0, output=content)
                failure = f"{problem} (thinking {'on' if thinking else 'off'})"
            raise exceptions.InferenceRuntimeError(
                f"{self.model_id} reply unusable after one retry with thinking on: {failure}", provider="OpenAI"
            )

    return NimKimiModel(
        model_id=config.model_id,
        api_key=config.api_key,
        base_url=config.base_url,
        max_workers=4,
        max_output_tokens=config.max_tokens,
    )


def extract_page(window: list[Message], after_ordinal: int, config: Config | None = None, model: Any = None) -> dict[str, Any]:
    """Run LangExtract over a window of messages and return the shared page.

    ``config`` is read from the environment when absent (raising :class:`ExtractorUnavailable` when no key is
    mounted). ``model`` lets a caller supply the LangExtract model; the Activity never does.
    """
    from importlib.metadata import version as package_version

    config = config or config_from_env()
    if not window:
        return PageBuilder(EXTRACTOR, package_version("langextract"), model_id=config.model_id).page(window, after_ordinal)
    try:
        import langextract as lx
        from langextract import prompt_validation as pv
    except ImportError as error:
        raise ExtractorUnavailable(f"langextract is not installed on this worker: {error}") from error
    text, spans = window_text(window)
    result = lx.extract(
        text_or_documents=text,
        prompt_description=PROMPT,
        examples=_examples(),
        model=model or build_model(config),
        format_type=lx.data.FormatType.JSON,
        fence_output=False,
        use_schema_constraints=False,
        max_char_buffer=MAX_CHAR_BUFFER,
        batch_length=4,
        max_workers=4,
        extraction_passes=1,
        show_progress=False,
        # The few-shot example's events deliberately overlap its entity spans; the alignment check only warns about that.
        prompt_validation_level=pv.PromptValidationLevel.OFF,
    )
    return page_from_extractions(
        list(result.extractions or []), window, text, spans, after_ordinal,
        version=package_version("langextract"), model_id=config.model_id,
    )
