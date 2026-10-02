"""NVIDIA NIM embedder for chunk text: nvidia/nemotron-3-embed-1b, 2048-d, batched, input_type passage.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The OpenAI-compatible /embeddings endpoint takes a real array and returns N vectors in one request (probed
2026-09-26). Same call shape and guards as the Go embedder (engine/embedding/nim.go): ``data:image/`` rewritten,
blank -> placeholder, 8000-character cut, ``truncate: END``. The key is never logged and never in an error.
"""

from __future__ import annotations

import time
from collections.abc import Callable

import httpx

from server.context_chunks.config import EMBED_DIMENSIONS, ChunkConfig
from server.context_chunks.render import nim_input


class EmbedError(RuntimeError):
    pass


class NimEmbedder:
    def __init__(
        self,
        config: ChunkConfig,
        *,
        http: httpx.Client | None = None,
        sleep: Callable[[float], None] = time.sleep,
        retries: int = 4,
    ):
        self._config = config
        self._http = http or httpx.Client(timeout=180.0)
        self._sleep = sleep
        self._retries = retries
        self.calls = 0  # embed requests made, for the dry-run estimate and the run receipt
        self.texts = 0

    @property
    def model(self) -> str:
        return self._config.embed_model

    def embed(self, texts: list[str]) -> list[list[float]]:
        """One vector per text, in order, batched ``embed_batch`` per request. Never a partial result."""
        out: list[list[float]] = []
        batch = self._config.embed_batch
        for start in range(0, len(texts), batch):
            out.extend(self._embed_batch([nim_input(t) for t in texts[start : start + batch]]))
        if len(out) != len(texts):
            raise EmbedError(f"embedder returned {len(out)} vectors for {len(texts)} texts")
        return out

    def _embed_batch(self, inputs: list[str]) -> list[list[float]]:
        body = {
            "model": self._config.embed_model,
            "input": inputs,
            "input_type": "passage",
            "encoding_format": "float",
            "truncate": "END",
        }
        headers = {"Authorization": f"Bearer {self._config.api_key}", "Content-Type": "application/json"}
        last = "no attempt made"
        for attempt in range(self._retries + 1):
            if attempt:
                self._sleep(5.0 * attempt)
            self.calls += 1
            try:
                response = self._http.post(f"{self._config.embed_base_url}/embeddings", json=body, headers=headers)
            except httpx.HTTPError as error:
                last = f"transport error {type(error).__name__}"
                continue
            if response.status_code == 429 or response.status_code >= 500:
                last = f"HTTP {response.status_code}"
                continue
            if response.status_code != 200:
                raise EmbedError(f"NIM returned HTTP {response.status_code}: {response.text[:300]}")
            data = sorted(response.json().get("data", []), key=lambda item: item.get("index", 0))
            vectors = [item["embedding"] for item in data]
            if len(vectors) != len(inputs):
                raise EmbedError(f"NIM returned {len(vectors)} vectors for {len(inputs)} inputs")
            if any(len(v) != EMBED_DIMENSIONS for v in vectors):
                raise EmbedError(f"NIM returned a vector that is not {EMBED_DIMENSIONS}-d")
            self.texts += len(inputs)
            return vectors
        raise EmbedError(f"NIM embedding failed after {self._retries + 1} attempts: {last}")
