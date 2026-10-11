"""Read-only Weaviate boundary for Intake, separate from evidence search.

Uses externally supplied query embeddings, never a server-side default vectorizer.
Collection population and coverage publication are separate indexing operations.
"""

from __future__ import annotations

import json
import math
import re
from dataclasses import dataclass
from typing import Literal
from urllib.parse import urlsplit

import httpx
from pydantic import BaseModel, Field, field_validator, model_validator

from .image_search import ImageHit


class FilesystemSearchRequest(BaseModel):
    query: str = Field(min_length=1, max_length=4096)
    limit: int = Field(default=20, ge=1, le=100)
    mode: Literal["hybrid", "keyword", "vector"] = "hybrid"
    vector: list[float] | None = Field(default=None, min_length=1, max_length=8192)

    @field_validator("query")
    @classmethod
    def query_has_text(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("Search query must contain text")
        return value

    @model_validator(mode="after")
    def vector_matches_mode(self) -> FilesystemSearchRequest:
        """Require a caller-supplied vector only for pure vector mode.

        Inputs: validated mode and optional externally generated query vector.
        Output: the validated request. Side effects: none. Pick `vector` to use an existing
        embedding without asking this API to initialize an embedding provider.
        """
        if self.mode == "vector" and self.vector is None:
            raise ValueError("Vector mode requires a caller-supplied query vector")
        if self.mode != "vector" and self.vector is not None:
            raise ValueError("A caller-supplied vector is only accepted in vector mode")
        return self


class FilesystemHit(BaseModel):
    object_id: str
    source_id: str
    source_path: str
    # The B2 object key and the catalog resolution, so a hit can be opened from the vault
    # rather than from a desktop path (Claude Code · Opus 5 · 2026-09-22; audit item I-8).
    vault_key: str = ""
    resolution: str = "unknown"
    document_id: str
    chunk_id: str
    filename: str
    text: str
    score: float | None = None
    distance: float | None = None


class FilesystemSearchResponse(BaseModel):
    query: str
    backend: str = "weaviate"
    collection: str
    target_vector: str | None
    coverage: str = "unknown"  # Successful query is not proof of complete indexing.
    hits: list[FilesystemHit]
    # Image lane (Claude Code · Fable 5.1 · 2026-09-22): empty when INTAKE_IMAGES_* is unset.
    image_collection: str | None = None
    image_hits: list[ImageHit] = Field(default_factory=list)


class FilesystemSearchError(RuntimeError):
    pass


@dataclass(frozen=True)
class WeaviateSearchConfig:
    url: str
    collection: str
    target_vector: str
    dimensions: int
    api_key: str = ""
    timeout_seconds: float = 20.0

    def validate(self) -> None:
        parsed = urlsplit(self.url)
        if parsed.scheme not in {"http", "https"} or not parsed.hostname:
            raise ValueError("Invalid filesystem Weaviate URL")
        if parsed.username or parsed.password or parsed.query or parsed.fragment:
            raise ValueError("Weaviate URL must not contain credentials, query, or fragment")
        if parsed.path not in {"", "/"}:
            raise ValueError("Weaviate URL must be an origin without an API path")
        if not re.fullmatch(r"[A-Z][A-Za-z0-9_]*", self.collection):
            raise ValueError("Explicit valid filesystem collection name required")
        if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", self.target_vector):
            raise ValueError("Explicit valid named vector required")
        if self.dimensions < 1 or not 0 < self.timeout_seconds <= 120:
            raise ValueError("Invalid vector dimensions or query timeout")


class WeaviateFilesystemSearcher:
    def __init__(self, config: WeaviateSearchConfig) -> None:
        config.validate()
        self.config = config

    async def search(
        self,
        request: FilesystemSearchRequest,
        *,
        vector: list[float] | None = None,
        transport: httpx.AsyncBaseTransport | None = None,
    ) -> FilesystemSearchResponse:
        """Search active filesystem objects with keyword, hybrid, or supplied-vector ranking.

        Inputs: a bounded request, its optional query vector, and an optional test transport.
        Output: source-preserving Weaviate hits with mode-appropriate ranking metadata.
        Side effects: one bounded Weaviate GraphQL read. Pick vector mode when NIM supplied an
        embedding upstream and keyword/hybrid when lexical or combined ranking is intended.
        """
        if not request.query.strip():
            raise ValueError("Search query must contain text")
        query_text = json.dumps(request.query, ensure_ascii=True)
        if request.mode in {"hybrid", "vector"}:
            if request.mode == "vector" and vector is None:
                vector = request.vector
            if vector is None or len(vector) != self.config.dimensions:
                raise ValueError("Query embedding dimension mismatch")
            if not all(math.isfinite(v) for v in vector) or not any(vector):
                raise ValueError("Query embedding must be finite and nonzero")
            encoded_vector = json.dumps(vector, allow_nan=False)
            target_vector = json.dumps(self.config.target_vector)
            if request.mode == "vector":
                operator = (
                    f"nearVector: {{vector: {encoded_vector}, targetVectors: [{target_vector}]}}"
                )
            else:
                operator = (
                    f"hybrid: {{query: {query_text}, alpha: 0.7, "
                    f"vector: {encoded_vector}, targetVectors: [{target_vector}]}}"
                )
        else:
            operator = f"bm25: {{query: {query_text}}}"
        additional_fields = "id distance" if request.mode == "vector" else "id score"
        query = (
            f"{{ Get {{ {self.config.collection}(limit: {request.limit}, {operator}, "
            'where: {path: ["active"], operator: Equal, valueBoolean: true}) { '
            "source_id source_path vault_key resolution document_id chunk_id filename text "
            f"_additional {{{additional_fields}}}"
            " } } }"
        )
        headers = {"Authorization": f"Bearer {self.config.api_key}"} if self.config.api_key else {}
        try:
            async with httpx.AsyncClient(
                timeout=self.config.timeout_seconds,
                headers=headers,
                transport=transport,
                follow_redirects=False,
            ) as client:
                response = await client.post(
                    self.config.url.rstrip("/") + "/v1/graphql", json={"query": query}
                )
                response.raise_for_status()
                body = response.json()
            if body.get("errors"):
                raise FilesystemSearchError("Weaviate rejected the filesystem query")
            rows = body["data"]["Get"][self.config.collection]
            if not isinstance(rows, list):
                raise ValueError("Missing result list")
            hits = []
            for row in rows:
                additional = row["_additional"]
                score = None if request.mode == "vector" else float(additional["score"])
                distance = float(additional["distance"]) if request.mode == "vector" else None
                hit = FilesystemHit(
                    object_id=additional["id"],
                    score=score,
                    distance=distance,
                    vault_key=row.get("vault_key") or "",
                    resolution=row.get("resolution") or "unknown",
                    **{
                        key: row[key]
                        for key in (
                            "source_id",
                            "source_path",
                            "document_id",
                            "chunk_id",
                            "filename",
                            "text",
                        )
                    },
                )
                hits.append(hit)
            if any(
                (hit.score is not None and not math.isfinite(hit.score))
                or (hit.distance is not None and not math.isfinite(hit.distance))
                for hit in hits
            ):
                raise ValueError("Nonfinite score")
        except (httpx.HTTPError, ValueError, KeyError, TypeError) as exc:
            # Never return provider bodies, credentials, or corpus excerpts as error details.
            raise FilesystemSearchError(
                "Filesystem search service unavailable or incompatible"
            ) from exc
        return FilesystemSearchResponse(
            query=request.query,
            collection=self.config.collection,
            target_vector=(
                self.config.target_vector if request.mode in {"hybrid", "vector"} else None
            ),
            hits=hits[: request.limit],
        )
