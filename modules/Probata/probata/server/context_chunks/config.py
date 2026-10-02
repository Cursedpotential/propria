"""Configuration for the chunk collection. Secrets are read from a mounted file, never printed.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""

from __future__ import annotations

import os
from collections.abc import Mapping
from dataclasses import dataclass, field
from pathlib import Path

# Owner-approved collection name (2026-10-02). Vector-store collection names have no other default.
COLLECTION = "ProfferChunks20261002"
VECTOR_NAME = "text_nim"
EMBED_MODEL = "nvidia/nemotron-3-embed-1b"
EMBED_DIMENSIONS = 2048
EMBED_BASE_URL = "https://integrate.api.nvidia.com/v1"
# One NIM request embeds a whole batch (probed 2026-09-18 and 2026-09-26: a real array in, N vectors out).
EMBED_BATCH = 32
# The Case Bible writers' and the Go embedder's 8000-character cut for one input.
MAX_INPUT_CHARS = 8000

# The Neural chunker classifies only the first ~512 tokens of what it is given (measured 2026-10-02 on transformers
# 5.17/5.18: a 1000-message thread came back as 3 chunks, every split inside the first window), so a thread is fed to it
# in overlapping windows of at most this many model tokens (chunker.WindowedNeural).
NEURAL_WINDOW_TOKENS = 450
# A chunk longer than this is cut into even parts by message, so the whole chunk fits the embedder's 8000-character
# input (nim_input cuts there). 0 disables it.
MAX_CHUNK_CHARS = 7500

DEFAULT_CHUNKER = "neural_distilbert"
# Every chunk reaches at least this many messages into the next chunk (owner 2026-10-02).
DEFAULT_OVERLAP = 2


@dataclass(frozen=True)
class ChunkConfig:
    weaviate_url: str
    collection: str = COLLECTION
    vector_name: str = VECTOR_NAME
    embed_base_url: str = EMBED_BASE_URL
    embed_model: str = EMBED_MODEL
    embed_batch: int = EMBED_BATCH
    api_key: str = field(default="", repr=False)


def load_config(env: Mapping[str, str] | None = None) -> ChunkConfig:
    """Read CONTEXT_CHUNKS_*; the NIM key comes from CONTEXT_CHUNKS_EMBED_API_KEY_FILE, else NVIDIA_API_KEY."""
    env = os.environ if env is None else env
    url = env.get("CONTEXT_CHUNKS_WEAVIATE_URL", "").strip().rstrip("/")
    if not url.startswith(("http://", "https://")):
        raise RuntimeError("CONTEXT_CHUNKS_WEAVIATE_URL must be an absolute http(s) URL")
    key_file = env.get("CONTEXT_CHUNKS_EMBED_API_KEY_FILE", "").strip()
    if key_file:
        try:
            key = Path(key_file).read_text(encoding="utf-8").strip()
        except OSError:
            raise RuntimeError("CONTEXT_CHUNKS_EMBED_API_KEY_FILE is unreadable") from None
    else:
        key = env.get("NVIDIA_API_KEY", "").strip()
    if not key:
        raise RuntimeError("no NIM API key: set CONTEXT_CHUNKS_EMBED_API_KEY_FILE (or NVIDIA_API_KEY)")
    return ChunkConfig(
        weaviate_url=url,
        collection=env.get("CONTEXT_CHUNKS_COLLECTION", "").strip() or COLLECTION,
        embed_base_url=(env.get("CONTEXT_CHUNKS_EMBED_BASE_URL", "").strip() or EMBED_BASE_URL).rstrip("/"),
        embed_model=env.get("CONTEXT_CHUNKS_EMBED_MODEL", "").strip() or EMBED_MODEL,
        api_key=key,
    )
