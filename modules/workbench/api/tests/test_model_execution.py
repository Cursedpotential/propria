"""Observed adapter-contract tests for Workbench model execution.

Byline: Codex · GPT-5 · 2026-08-16
Byline: Claude Code · Opus 5.5 · 2026-09-25 (empty/junk/invalid-JSON reply retry, 502/429 mapping, kimi-k3 mode by prompt size)
"""

from __future__ import annotations

import asyncio
import logging
from types import SimpleNamespace
from typing import Any

import pytest
from agno.exceptions import ModelRateLimitError
from agno.models.message import Message
from fastapi.testclient import TestClient

from app.runtime import auth
from app.service import classification as classification_module
from app.service.model_providers import (
    KIMI_K3_MODEL_ID,
    NVIDIA_BASE_URL_DEFAULT,
    build_provider,
    run_classification,
    run_sentiment,
)
from app.service import nim_kimi
from app.service.model_replies import ModelReplyError, ask_json
from app.types.classification import ProviderName
from main import app


class FakeAgnoModel:
    """Implements Agno 2.8's aresponse API and deliberately has no arun."""

    def __init__(self, content: str) -> None:
        self.id = "fake-model"
        self.temperature: float | None = None
        self.max_tokens: int | None = None
        self.content = content
        self.messages: list[Message] = []

    async def aresponse(self, messages: list[Message], **_kwargs: Any) -> Any:
        self.messages = messages
        return SimpleNamespace(content=self.content)


def test_classification_uses_installed_agno_aresponse_contract() -> None:
    model = FakeAgnoModel('{"category":"legal","confidence":0.9,"reasoning":"filing"}')

    result = asyncio.run(
        run_classification(
            model,
            "Motion for parenting time",
            ["legal", "other"],
            temperature=0.2,
            max_tokens=321,
        )
    )

    assert result[:3] == ("legal", 0.9, "filing")
    assert model.temperature == 0.2
    assert model.max_tokens == 321
    assert [message.role for message in model.messages] == ["system", "user"]


def test_sentiment_uses_installed_agno_aresponse_contract() -> None:
    model = FakeAgnoModel('{"sentiment":"mixed","score":0.1,"emotions":{"trust":0.4},"reasoning":"conflicted"}')

    result = asyncio.run(
        run_sentiment(
            model,
            "I want to trust this but remain worried.",
            temperature=0.1,
            max_tokens=222,
        )
    )

    assert result[:4] == ("mixed", 0.1, {"trust": 0.4}, "conflicted")
    assert model.temperature == 0.1
    assert model.max_tokens == 222


class ScriptedModel:
    """Answers with a scripted list of contents, one per call, and records each call's kwargs."""

    def __init__(self, contents: list[str | None], model_id: str = "scripted-model", base_url: str = "") -> None:
        self.id = model_id
        self.base_url = base_url
        self.temperature: float | None = None
        self.max_tokens: int | None = None
        self._contents = list(contents)
        self.calls: list[dict[str, Any]] = []

    async def aresponse(self, messages: list[Message], **kwargs: Any) -> Any:
        self.calls.append(kwargs)
        return SimpleNamespace(content=self._contents.pop(0))


VALID = '{"category":"legal","confidence":0.8,"reasoning":"filing"}'


def test_empty_reply_is_retried_once() -> None:
    model = ScriptedModel(["", VALID])
    result = asyncio.run(run_classification(model, "Motion", ["legal", "other"]))
    assert result[:3] == ("legal", 0.8, "filing")
    assert len(model.calls) == 2


def test_two_empty_replies_raise_an_error_naming_the_model() -> None:
    model = ScriptedModel([None, "  "], model_id=KIMI_K3_MODEL_ID)
    with pytest.raises(ModelReplyError) as caught:
        asyncio.run(run_classification(model, "Motion", ["legal", "other"]))
    assert KIMI_K3_MODEL_ID in str(caught.value)
    assert "empty reply" in str(caught.value)


def test_invalid_json_never_falls_back_to_the_first_category() -> None:
    model = ScriptedModel(["It is probably legal.", '{"confidence": 0.9}'])
    with pytest.raises(ModelReplyError) as caught:
        asyncio.run(run_classification(model, "Motion", ["legal", "other"]))
    assert "invalid JSON reply" in str(caught.value)
    assert len(model.calls) == 2


def test_junk_reply_is_retried_and_named_when_it_repeats() -> None:
    junk = "!" * 32  # the kimi-k3-on-NIM signature measured 2026-09-25
    rescued = ScriptedModel([junk, VALID])
    assert asyncio.run(run_classification(rescued, "Motion", ["legal", "other"]))[0] == "legal"
    with pytest.raises(ModelReplyError) as caught:
        asyncio.run(run_classification(ScriptedModel([junk, junk]), "Motion", ["legal", "other"]))
    assert "junk reply (32 x '!')" in str(caught.value)


def test_sentiment_label_outside_the_set_is_an_error_not_neutral() -> None:
    bad = '{"sentiment":"ecstatic","score":0.9,"emotions":{},"reasoning":"x"}'
    with pytest.raises(ModelReplyError):
        asyncio.run(run_sentiment(ScriptedModel([bad, bad]), "Great news"))


def test_each_attempt_uses_its_own_model_and_the_json_request() -> None:
    first, second = ScriptedModel(["!" * 32]), ScriptedModel([VALID])
    request = {"response_format": {"type": "json_object"}}
    parsed, raw = asyncio.run(ask_json([("thinking off", first), ("thinking on", second)], [], dict, request))
    assert parsed["category"] == "legal" and raw == VALID
    assert first.calls == [request] and second.calls == [request]


def test_each_attempt_logs_one_line_without_prompt_text(caplog: pytest.LogCaptureFixture) -> None:
    private = "PRIVATE-CASE-TEXT-7731"
    caplog.set_level(logging.INFO, logger="app.service.model_replies")
    model = ScriptedModel(["!" * 32, VALID], model_id="m1")
    asyncio.run(run_classification(model, private, ["legal", "other"]))
    lines = [r.getMessage() for r in caplog.records if r.name == "app.service.model_replies"]
    assert len(lines) == 2
    assert lines[0].startswith("model attempt model=m1 mode=first try prompt_chars=")
    assert "outcome=junk latency_ms=" in lines[0]
    assert "mode=retry" in lines[1] and "outcome=ok" in lines[1]
    assert all(private not in line and "legal" not in line for line in lines)


def test_a_provider_error_is_logged_then_raised(caplog: pytest.LogCaptureFixture) -> None:
    class Failing(ScriptedModel):
        async def aresponse(self, messages: list[Message], **kwargs: Any) -> Any:
            raise ModelRateLimitError("Unknown model error", model_id=KIMI_K3_MODEL_ID)

    caplog.set_level(logging.INFO, logger="app.service.model_replies")
    with pytest.raises(ModelRateLimitError):
        asyncio.run(run_classification(Failing([], model_id=KIMI_K3_MODEL_ID), "Motion", ["legal", "other"]))
    lines = [r.getMessage() for r in caplog.records if r.name == "app.service.model_replies"]
    assert len(lines) == 1 and "outcome=error (ModelRateLimitError)" in lines[0]


def _nim_provider(monkeypatch: pytest.MonkeyPatch) -> Any:
    for name in ("DEFAULT_MODEL_ID", "NVIDIA_MODEL_ID", "NVIDIA_BASE_URL"):
        monkeypatch.delenv(name, raising=False)
    monkeypatch.setenv("NVIDIA_API_KEY", "test-key")
    return build_provider(ProviderName.NVIDIA)


def test_kimi_k3_on_nim_picks_the_thinking_mode_by_prompt_size(monkeypatch: pytest.MonkeyPatch) -> None:
    provider = _nim_provider(monkeypatch)
    assert provider.id == KIMI_K3_MODEL_ID
    # Rate limits are retried with backoff before the call gives up.
    assert (getattr(provider, "retries"), getattr(provider, "exponential_backoff")) == (3, True)
    provider.temperature, provider.max_tokens = 0.0, 2048
    assert nim_kimi.json_mode(provider) == {"response_format": {"type": "json_object"}}

    def plan(chars: int) -> list[tuple[str, Any]]:
        return nim_kimi.attempts(provider, [Message(role="user", content="x" * chars)])

    short, long = plan(1_000), plan(nim_kimi.SHORT_PROMPT_CHARS + 1)
    assert [label for label, _ in short] == ["thinking off", "thinking on"]
    assert [label for label, _ in long] == ["thinking on", "thinking on again"]  # owner 2026-09-26 option B
    assert [m.extra_body["chat_template_kwargs"]["thinking"] for _, m in short] == [False, True]
    for _, model in short + long:
        assert model is not provider  # a batch shares the provider; attempts never mutate it
        assert (model.id, model.max_tokens, model.retries) == (KIMI_K3_MODEL_ID, 2048, 3)


def test_other_models_retry_as_themselves_without_json_mode() -> None:
    other = ScriptedModel([VALID], model_id="other-model", base_url=NVIDIA_BASE_URL_DEFAULT)
    assert nim_kimi.attempts(other, []) == [("first try", other), ("retry", other)]
    assert nim_kimi.json_mode(other) == {}


def test_model_reply_error_is_a_502_naming_the_model(monkeypatch: pytest.MonkeyPatch) -> None:
    async def failing_classify(_request: Any) -> Any:
        raise ModelReplyError(
            f"Model '{KIMI_K3_MODEL_ID}' gave no usable reply in 2 attempts "
            "(thinking off: junk reply (32 x '!'); thinking on: empty reply)."
        )

    monkeypatch.setattr(auth.settings, "trusted_auth_proxy_cidrs", "10.0.0.0/8")
    monkeypatch.setattr(auth.settings, "tailnet_auth_bypass_enabled", False)
    monkeypatch.setattr(classification_module.classification_service, "classify", failing_classify)
    client = TestClient(app, client=("10.1.2.3", 50000))
    response = client.post(
        "/api/classification/classify",
        json={"text": "Motion", "categories": ["legal", "other"]},
        headers={"X-authentik-uid": "user-123", "X-authentik-username": "owner@example.test"},
    )
    assert response.status_code == 502
    assert KIMI_K3_MODEL_ID in response.json()["detail"]
    assert "empty reply" in response.json()["detail"]


def test_provider_rate_limit_is_a_429_not_an_opaque_500(monkeypatch: pytest.MonkeyPatch) -> None:
    async def rate_limited(_request: Any) -> Any:
        raise ModelRateLimitError("Unknown model error", model_id=KIMI_K3_MODEL_ID)

    monkeypatch.setattr(auth.settings, "trusted_auth_proxy_cidrs", "10.0.0.0/8")
    monkeypatch.setattr(auth.settings, "tailnet_auth_bypass_enabled", False)
    monkeypatch.setattr(classification_module.classification_service, "classify", rate_limited)
    client = TestClient(app, client=("10.1.2.3", 50000))
    response = client.post(
        "/api/classification/classify",
        json={"text": "Motion", "categories": ["legal", "other"]},
        headers={"X-authentik-uid": "user-123", "X-authentik-username": "owner@example.test"},
    )
    assert response.status_code == 429
    assert KIMI_K3_MODEL_ID in response.json()["detail"]
