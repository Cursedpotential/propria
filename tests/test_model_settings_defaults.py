"""Default model selection contract for server/core/settings.py.

Owner 2026-09-25: glm-5.1 is banned outright and Ollama Cloud is no longer primary;
NVIDIA NIM ``moonshotai/kimi-k3`` is the default and reasons unless an agent turns it off
(``build_model(thinking=False)``) — with thinking off, long prompts came back as junk.

Byline: Claude Code · Opus 5.5 · 2026-09-25
"""

from __future__ import annotations

import pytest

from server.core import settings

_PROVIDER_ENV = (
    "DEFAULT_MODEL_PROVIDER",
    "DEFAULT_MODEL_ID",
    "NVIDIA_MODEL_ID",
    "KIMI_MODEL_ID",
    "OLLAMA_MODEL_ID",
    "NVIDIA_API_KEY",
    "NVIDIA_BASE_URL",
    "MOONSHOT_API_KEY",
    "OLLAMA_API_KEY",
    "OLLAMA_HOST",
    "OPENROUTER_API_KEY",
    "ANTHROPIC_API_KEY",
    "ANTHROPIC_AUTH_TOKEN",
    "OPENAI_API_KEY",
    "GOOGLE_API_KEY",
    "GROQ_API_KEY",
)


@pytest.fixture(autouse=True)
def _clean_provider_env(monkeypatch: pytest.MonkeyPatch) -> None:
    for name in _PROVIDER_ENV:
        monkeypatch.delenv(name, raising=False)


def test_nvidia_is_first_and_no_glm_is_pinned() -> None:
    assert settings._provider_order()[0] == "nvidia"
    assert not any("glm" in model_id for model_id in settings._PINNED.values())
    assert "ollama" not in settings._PINNED


def test_default_is_kimi_k3_on_nim_with_thinking_on(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("NVIDIA_API_KEY", "test-key")
    monkeypatch.setenv("OLLAMA_API_KEY", "test-key")  # Ollama credentials alone never make it the default
    model = settings.default_model()
    assert model.id == "moonshotai/kimi-k3"
    assert model.base_url == settings.NVIDIA_BASE_URL_DEFAULT
    assert model.extra_body == {"chat_template_kwargs": {"thinking": True}}


def test_an_agent_can_turn_thinking_off(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("NVIDIA_API_KEY", "test-key")
    assert settings.build_model(thinking=False).extra_body == {"chat_template_kwargs": {"thinking": False}}


def test_other_nim_models_get_no_extra_body(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("NVIDIA_API_KEY", "test-key")
    assert settings.build_model(provider="nvidia", model_id="meta/llama-3.3-70b-instruct").extra_body is None


def test_ollama_without_a_named_model_is_skipped(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("OLLAMA_API_KEY", "test-key")
    with pytest.raises(ValueError, match="No model provider configured"):
        settings.build_model()
    monkeypatch.setenv("OLLAMA_MODEL_ID", "kimi-k3")
    assert settings.build_model().id == "kimi-k3"


def test_kimi_via_moonshot_direct_sends_no_nim_body(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("MOONSHOT_API_KEY", "test-key")
    model = settings.build_model(provider="kimi")
    assert model.id == "moonshotai/kimi-k3"
    assert model.extra_body is None
