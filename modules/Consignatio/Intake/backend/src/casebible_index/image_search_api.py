"""``POST /images/search``: find screenshots, photos and scanned pages by a text question. One job:
read.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

Wraps ``image_search.WeaviateImageSearcher`` for the catalog-sourced image index the Super Index
image stage fills.
What it queries follows the same settings that built the index (``INTAKE_IMAGES_SLOTS``,
``INTAKE_IMAGES_CLIP``), so a
lane is queried only when its named vector exists:

* literal OCR text (BM25, verified as a substring of the retained OCR text) is always searched;
* hybrid mode adds one vector lane per enabled slot: ``image_maxsim`` (Jina bag), ``image_single``
(NIM or Google),
  ``image_colqwen`` (ColQwen endpoint) and ``image_clip`` (Weaviate's CLIP module, no client call);
* ``kind`` restricts to ``screenshot``, ``photo`` or ``scan`` (the "only screenshots" filter).

Every hit carries its locator (``provider``, ``bucket``, ``vault_key``, ``content_sha256``), its
``image_kind`` and its
channel, so OCR text matches and visual-similarity matches are never mixed in one ranking.
"""

from __future__ import annotations

from typing import Literal

import httpx
from pydantic import BaseModel, Field, field_validator

from .image_embedders import ColQwenEmbedder, ImageEmbedders
from .image_search import ImageHit, WeaviateImageSearcher
from .image_stage import ImageStageSettings
from .image_target import CLIP_VECTOR
from .secrets import get_secret


class ImageSearchRequest(BaseModel):
    query: str = Field(min_length=1, max_length=4096)
    limit: int = Field(default=20, ge=1, le=100)
    mode: Literal["hybrid", "keyword"] = "hybrid"
    kind: Literal["screenshot", "photo", "scan"] | None = None

    @field_validator("query")
    @classmethod
    def query_has_text(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("Search query must contain text")
        return value


class ImageSearchResponse(BaseModel):
    query: str
    collection: str
    kind: str | None
    lanes: list[str]
    hits: list[ImageHit]


async def search_images(request: ImageSearchRequest) -> ImageSearchResponse:
    """Run one image search against the configured collection.

    Inputs: the request and the environment (``INTAKE_IMAGES_*``, keys by name). Output: hits
    grouped by channel
    (OCR literal first, then each vector lane). Raises ``ValueError`` for a missing URL or key and
    ``ImageSearchError`` when Weaviate is unavailable or incompatible. Nothing is written."""
    settings = ImageStageSettings.from_env()
    if not settings.weaviate_url:
        raise ValueError("INTAKE_IMAGES_WEAVIATE_URL (or INTAKE_WEAVIATE_URL) is not configured")
    lanes = tuple(settings.slots) + ((CLIP_VECTOR,) if settings.clip else ())
    searcher = WeaviateImageSearcher(
        settings.weaviate_url, settings.collection, get_secret("INTAKE_WEAVIATE_API_KEY") or ""
    )
    if request.mode == "keyword":
        hits = await searcher.search(
            request.query, limit=request.limit, mode="keyword", embedders=None, kind=request.kind
        )
        return ImageSearchResponse(
            query=request.query,
            collection=settings.collection,
            kind=request.kind,
            lanes=[],
            hits=hits,
        )
    key_name = "NVIDIA_API_KEY" if settings.single_provider == "nim" else "GOOGLE_API_KEY"
    single_key = get_secret(key_name) if "image_single" in lanes else "unused"
    if not single_key:
        raise ValueError(f"{key_name} is not configured for image search")
    jina_key = get_secret("JINA_API_KEY") if "image_maxsim" in lanes else None
    if "image_maxsim" in lanes and not jina_key:
        raise ValueError("JINA_API_KEY is not configured for image search")
    async with httpx.AsyncClient(timeout=120.0, follow_redirects=False) as client:
        embedders = ImageEmbedders(client, settings.single_provider, single_key, jina_key)
        colqwen = (
            ColQwenEmbedder(client, settings.colqwen_url, get_secret("COLQWEN_API_TOKEN") or "")
            if "image_colqwen" in lanes and settings.colqwen_url
            else None
        )
        hits = await searcher.search(
            request.query,
            limit=request.limit,
            mode="hybrid",
            embedders=embedders,
            kind=request.kind,
            lanes=lanes,
            colqwen=colqwen,
        )
    return ImageSearchResponse(
        query=request.query,
        collection=settings.collection,
        kind=request.kind,
        lanes=list(lanes),
        hits=hits,
    )
