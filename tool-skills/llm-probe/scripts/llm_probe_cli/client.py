"""Thin sync client for the llm-probe service (deployed on ovh-files,
tailnet-only). Shared by the CLI and the TUI so there's exactly one place
that knows the API shape.

Byline: Claude Code · Sonnet 5 · 2026-08-27
"""
from __future__ import annotations

import os
from typing import Any, Optional

import httpx

DEFAULT_BASE_URL = "http://100.91.190.107:8030"


def base_url() -> str:
    return os.environ.get("LLM_PROBE_URL", DEFAULT_BASE_URL).rstrip("/")


class LLMProbeClient:
    def __init__(self, base: Optional[str] = None, timeout: float = 60.0):
        self.base = (base or base_url()).rstrip("/")
        self._client = httpx.Client(base_url=self.base, timeout=timeout)

    def close(self) -> None:
        self._client.close()

    def __enter__(self) -> "LLMProbeClient":
        return self

    def __exit__(self, *exc) -> None:
        self.close()

    def health(self) -> dict[str, Any]:
        return self._client.get("/health").json()

    def providers(self) -> list[dict[str, Any]]:
        return self._client.get("/providers").json()

    def models(self, provider: str) -> list[dict[str, Any]]:
        r = self._client.get(f"/providers/{provider}/models")
        r.raise_for_status()
        return r.json()

    def probe_defs(self) -> dict[str, Any]:
        return self._client.get("/probes").json()

    def run_probe(self, provider: str, model: str, probe: str, *,
                  max_tokens: int = 500, temperature: float = 0,
                  reasoning_effort: Optional[str] = None, persist: bool = True,
                  run_note: Optional[str] = None) -> dict[str, Any]:
        body = {
            "provider": provider, "model": model, "probe": probe,
            "max_tokens": max_tokens, "temperature": temperature,
            "reasoning_effort": reasoning_effort, "persist": persist, "run_note": run_note,
        }
        r = self._client.post("/probe/run", json=body, timeout=90)
        r.raise_for_status()
        return r.json()

    def run_playground(self, provider: str, model: str, prompt: str, *,
                        max_tokens: int = 500, temperature: float = 0,
                        reasoning_effort: Optional[str] = None, label: Optional[str] = None,
                        persist: bool = True) -> dict[str, Any]:
        body = {
            "provider": provider, "model": model, "prompt": prompt,
            "max_tokens": max_tokens, "temperature": temperature,
            "reasoning_effort": reasoning_effort, "label": label, "persist": persist,
        }
        r = self._client.post("/playground/run", json=body, timeout=90)
        r.raise_for_status()
        return r.json()

    def playground_history(self, provider: Optional[str] = None, model: Optional[str] = None,
                            limit: int = 50) -> list[dict[str, Any]]:
        params = {"limit": limit}
        if provider:
            params["provider"] = provider
        if model:
            params["model"] = model
        r = self._client.get("/playground/history", params=params)
        r.raise_for_status()
        return r.json()

    def board(self) -> list[dict[str, Any]]:
        r = self._client.get("/results/board", timeout=30)
        r.raise_for_status()
        return r.json()

    def summary(self) -> dict[str, Any]:
        r = self._client.get("/results/summary", timeout=30)
        r.raise_for_status()
        return r.json()
