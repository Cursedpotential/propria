"""Read-only search over the Intake image collection (single vector, MaxSim, OCR text).

> _Byline: Claude Code · Fable 5.1 · 2026-09-22_
Owner 2026-09-22: the image index is wired into Intake search. This lane runs beside the
text lane in /filesystem/search when INTAKE_IMAGES_* is configured. Hybrid mode: the
question is embedded with the same hosted embedders the index used; MaxSim scores the
screenshots' Jina bag, the single vector scores everything, and the OCR fallback text is
matched by keyword. Keyword mode uses only the OCR text. Nothing is written.
"""

# Updated by: Codex (D03 Case Bible search) | Date: 2026-09-23 | Rev: 2 |
# Platform: Codex / win32 | Changes: verify literal OCR spans and label visual hits |
# Context: BM25 candidates cannot establish that a requested phrase occurs in OCR text.

from __future__ import annotations

import json
import math
import unicodedata
from typing import Literal
from urllib.parse import urlsplit

import httpx
from pydantic import BaseModel

from .image_embedders import ColQwenEmbedder, ImageEmbedders
from .image_target import CLIP_VECTOR, COLQWEN_VECTOR, MULTI_VECTOR, SINGLE_VECTOR

_FIELDS = (
    "source_id source_path filename content_sha256 source_content_sha1 "
    "original_time original_time_source "
    "original_time_confidence original_time_conflict device software gps is_screenshot "
    "ocr_text provider bucket vault_key image_kind source_kind page identity "
    "screenshot_decision screenshot_app screenshot_app_confidence screenshot_app_basis "
    "screenshot_classifier_version screenshot_classifier_provider screenshot_classifier_status "
    "_additional { id distance score }"
)
KINDS = ("screenshot", "photo", "scan")
APPS = (
    "facebook-messenger", "messages", "snapchat", "whatsapp", "instagram",
    "browser", "other", "mixed", "unknown",
)


class ImageHit(BaseModel):
    object_id: str
    source_id: str
    source_path: str
    filename: str
    content_sha256: str
    source_content_sha1: str = ""
    original_time: str
    original_time_source: str
    original_time_confidence: str
    original_time_conflict: bool
    device: str
    software: str
    gps: str
    is_screenshot: bool
    ocr_excerpt: str
    matched_by: Literal["maxsim", "colqwen", "single", "clip", "ocr_literal"]
    channel: Literal["ocr_literal", "visual_similarity"]
    score_basis: Literal["weaviate_bm25", "weaviate_cosine", "weaviate_maxsim"]
    # Catalog-source fields (the Super Index image stage). With ``provider``, ``bucket``
    #  and ``source_content_sha1`` the
    # ``vault_key`` is the locator that opens the original; ``image_kind`` is screenshot,
    #  photo or scan; a rendered
    # PDF page has ``source_kind`` pdf_page and its 1-based ``page``. Empty on objects
    #  from the directory lane.
    provider: str = ""
    bucket: str = ""
    vault_key: str = ""
    image_kind: str = ""
    source_kind: str = ""
    page: int = 0
    identity: str = ""
    screenshot_decision: str = "unknown"
    screenshot_app: str = "unknown"
    screenshot_app_confidence: int = 0
    screenshot_app_basis: str = ""
    screenshot_classifier_version: str = ""
    screenshot_classifier_provider: str = ""
    screenshot_classifier_status: str = "unknown"
    ocr_span_start: int | None = None
    ocr_span_end: int | None = None
    region_status: Literal["not_recorded"] = "not_recorded"
    locator_status: Literal["provisional"] = "provisional"
    score: float


class ImageSearchError(RuntimeError):
    pass


def _normalized_chars(value: str) -> tuple[str, list[tuple[int, int]]]:
    """NFC + casefold + collapsed Unicode whitespace, retaining source offsets.

    Punctuation is unchanged and matching is a substring, including within words.
    Offsets are Python character offsets in the retained OCR text, end exclusive.
    """
    chars: list[str] = []
    spans: list[tuple[int, int]] = []
    index = 0
    while index < len(value):
        start = index
        index += 1
        if value[start].isspace():
            while index < len(value) and value[index].isspace():
                index += 1
            normalized = " "
        else:
            # NFC also composes Hangul Jamo across combining-class-zero starters.
            while index < len(value) and (
                unicodedata.combining(value[index])
                or len(unicodedata.normalize("NFC", value[start : index + 1]))
                < len(unicodedata.normalize("NFC", value[start:index])) + 1
            ):
                index += 1
            normalized = unicodedata.normalize("NFC", value[start:index]).casefold()
        chars.extend(normalized)
        spans.extend([(start, index)] * len(normalized))
    return "".join(chars), spans


def _literal_span(ocr_text: str, query: str) -> tuple[int, int] | None:
    normalized_text, offsets = _normalized_chars(ocr_text)
    normalized_query, _ = _normalized_chars(query.strip())
    if not normalized_query:
        return None
    start = normalized_text.find(normalized_query)
    if start < 0:
        return None
    return offsets[start][0], offsets[start + len(normalized_query) - 1][1]


def _validate_origin(url: str) -> str:
    parsed = urlsplit(url)
    if parsed.scheme not in {"http", "https"} or not parsed.hostname:
        raise ValueError("Invalid image Weaviate URL")
    if parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise ValueError("Weaviate URL must not contain credentials, query, or fragment")
    return url.rstrip("/")


class WeaviateImageSearcher:
    def __init__(self, url: str, collection: str, api_key: str = "", timeout: float = 30.0):
        if not collection.isidentifier() or not collection[0].isupper():
            raise ValueError("Explicit valid image collection name required")
        self.url = _validate_origin(url)
        self.collection = collection
        self.api_key = api_key
        self.timeout = timeout

    @staticmethod
    def _where(kind: str | None, app: str | None = None) -> str:
        """Build an active image filter with optional kind and named screenshot app.

        Inputs: validated kind and app names. Output: a Weaviate GraphQL filter string.
        Side effects: none. Use this for image queries rather than the filesystem text filter.
        The unknown app is checked after retrieval so old rows with no app remain eligible.
        """
        active = '{path: ["active"], operator: Equal, valueBoolean: true}'
        if kind is not None and kind not in KINDS:
            raise ValueError(f"image kind must be one of {KINDS}")
        if app is not None and app not in APPS:
            raise ValueError(f"screenshot app must be one of {APPS}")
        operands = [active]
        if kind:
            operands.append(
                '{path: ["image_kind"], operator: Equal, valueText: '
                + json.dumps(kind) + "}"
            )
        if app and app != "unknown":
            operands.append(
                '{path: ["screenshot_app"], operator: Equal, valueText: '
                + json.dumps(app) + "}"
            )
        return active if len(operands) == 1 else "{operator: And, operands: [" + ", ".join(operands) + "]}"

    async def _get(
        self, client: httpx.AsyncClient, operator: str, limit: int,
        kind: str | None = None, app: str | None = None,
    ) -> list[dict]:
        """Fetch one ranked candidate lane from the active Intake image collection.

        Inputs: GraphQL operator, candidate limit, kind and app filters. Output: result rows.
        Side effect: one read-only Weaviate request. Use for each OCR or vector lane.
        """
        query = (
            f"{{ Get {{ {self.collection}(limit: {limit}, {operator}, "
            f"where: {self._where(kind, app)}) {{ "
            f"{_FIELDS} }} }} }}"
        )
        response = await client.post(f"{self.url}/v1/graphql", json={"query": query})
        response.raise_for_status()
        body = response.json()
        if body.get("errors"):
            raise ImageSearchError("Weaviate rejected the image query")
        rows = body["data"]["Get"][self.collection]
        if not isinstance(rows, list):
            raise ValueError("Missing result list")
        return rows

    async def search(
        self,
        query: str,
        *,
        limit: int,
        mode: str,
        embedders: ImageEmbedders | None,
        transport: httpx.AsyncBaseTransport | None = None,
        kind: str | None = None,
        app: str | None = None,
        lanes: tuple[str, ...] | None = None,
        colqwen: ColQwenEmbedder | None = None,
    ) -> list[ImageHit]:
        """Find images by a text question: literal OCR text, plus (hybrid) one vector lane per \
enabled slot.

        ``kind`` restricts to screenshot, photo or scan. ``app`` restricts to a
        classifier app; unknown also includes old unclassified rows. ``lanes`` \
names the vector lanes to
        query (``image_single``, ``image_maxsim``, ``image_colqwen``, ``image_clip``); None \
means single plus the Jina
        bag when the embedders have a Jina key (the directory lane's behaviour). A lane needs \
its named vector in the
        collection, so the caller passes the slots that were provisioned.
        Inputs: text, candidate limit, mode, embedders, optional filters and lanes.
        Output: ranked ImageHit records with source and classifier provenance.
        Side effect: read-only Weaviate and embedder requests; no indexed data changes.
        Use this for image results rather than the filesystem text searcher.
        """
        if not query.strip():
            raise ValueError("Search query must contain text")
        # Validate before making a request, including keyword searches with no matching rows.
        try:
            self._where(kind, app)
        except ValueError as exc:
            raise ImageSearchError("Invalid image search filter") from exc
        headers = {"Authorization": f"Bearer {self.api_key}"} if self.api_key else {}
        text = json.dumps(query, ensure_ascii=True)
        merged: dict[str, tuple[str, float, dict, tuple[int, int] | None]] = {}

        rank = {"ocr_literal": 0, "maxsim": 1, "colqwen": 2, "single": 3, "clip": 4}
        candidate_limit = min(limit * 4, 400) if app == "unknown" else limit

        def keep(
            row: dict,
            matched_by: str,
            score: float,
            span: tuple[int, int] | None = None,
        ) -> None:
            if not math.isfinite(score):
                raise ValueError("Nonfinite score")
            if app == "unknown" and (row.get("screenshot_app") or "unknown") != "unknown":
                return
            object_id = row["_additional"]["id"]
            prior = merged.get(object_id)
            if prior is None or (rank[matched_by], -score) < (rank[prior[0]], -prior[1]):
                merged[object_id] = (matched_by, score, row, span)

        try:
            async with httpx.AsyncClient(
                timeout=self.timeout, headers=headers, transport=transport, follow_redirects=False
            ) as client:
                for row in await self._get(
                    client, f'bm25: {{query: {text}, properties: ["ocr_text"]}}', candidate_limit, kind, app
                ):
                    span = _literal_span(row.get("ocr_text") or "", query)
                    if span is not None:
                        keep(row, "ocr_literal", float(row["_additional"]["score"]), span)
                if mode == "hybrid":
                    if embedders is None:
                        raise ValueError("Image embedders are required for hybrid search")
                    wanted = lanes if lanes is not None else (SINGLE_VECTOR, MULTI_VECTOR)
                    plan: list[tuple[str, object, str]] = []
                    if embedders is not None and SINGLE_VECTOR in wanted:
                        single, multi = await embedders.embed_query(query)
                        plan.append((SINGLE_VECTOR, single, "single"))
                        if multi is not None and MULTI_VECTOR in wanted:
                            plan.append((MULTI_VECTOR, multi, "maxsim"))
                    elif embedders is not None and MULTI_VECTOR in wanted:
                        plan.append(
                            (MULTI_VECTOR, await embedders.embed_query_multi(query), "maxsim")
                        )
                    if COLQWEN_VECTOR in wanted and colqwen is not None:
                        plan.append((COLQWEN_VECTOR, await colqwen.embed_query(query), "colqwen"))
                    for target, vector, label in plan:
                        near = (
                            f"nearVector: {{vector: {json.dumps(vector, allow_nan=False)}, "
                            f'targetVectors: ["{target}"]}}'
                        )
                        for row in await self._get(client, near, candidate_limit, kind, app):
                            # MaxSim distances are negative sums; cosine distances are 0..2.
                            # Both are turned into "higher is better" on their own scale.
                            distance = float(row["_additional"]["distance"])
                            keep(
                                row,
                                label,
                                -distance if label in {"maxsim", "colqwen"} else 1.0 - distance,
                            )
                    if CLIP_VECTOR in wanted:
                        # Weaviate's multi2vec-clip embeds the question itself
                        #  (nearText), no client call.
                        near = f'nearText: {{concepts: [{text}], targetVectors: ["{CLIP_VECTOR}"]}}'
                        for row in await self._get(client, near, candidate_limit, kind, app):
                            keep(row, "clip", 1.0 - float(row["_additional"]["distance"]))
        except (httpx.HTTPError, ValueError, KeyError, TypeError) as exc:
            raise ImageSearchError("Image search service unavailable or incompatible") from exc

        hits = []
        for matched_by, score, row, span in merged.values():
            ocr_text = row.get("ocr_text") or ""
            excerpt_start = max(0, span[0] - 80) if span else 0
            excerpt_end = max(span[1] + 80, excerpt_start + 300) if span else 0
            hits.append(
                ImageHit(
                    object_id=row["_additional"]["id"],
                    matched_by=matched_by,
                    channel="ocr_literal" if span else "visual_similarity",
                    score_basis={
                        "ocr_literal": "weaviate_bm25",
                        "maxsim": "weaviate_maxsim",
                        "colqwen": "weaviate_maxsim",
                        "single": "weaviate_cosine",
                        "clip": "weaviate_cosine",
                    }[matched_by],
                    score=round(score, 4),
                    ocr_excerpt=ocr_text[excerpt_start:excerpt_end] if span else "",
                    ocr_span_start=span[0] if span else None,
                    ocr_span_end=span[1] if span else None,
                    **{
                        key: row[key]
                        for key in (
                            "source_id",
                            "source_path",
                            "filename",
                            "content_sha256",
                            "original_time",
                            "original_time_source",
                            "original_time_confidence",
                            "original_time_conflict",
                            "device",
                            "software",
                            "gps",
                            "is_screenshot",
                        )
                    },
                    **{
                        key: row[key]
                        for key in (
                            "provider",
                            "bucket",
                            "vault_key",
                            "image_kind",
                            "source_kind",
                            "page",
                            "identity",
                            "source_content_sha1",
                        )
                        if row.get(key) is not None
                    },
                    screenshot_decision=row.get("screenshot_decision") or "unknown",
                    screenshot_app=row.get("screenshot_app") or "unknown",
                    screenshot_app_confidence=row.get("screenshot_app_confidence") or 0,
                    screenshot_app_basis=row.get("screenshot_app_basis") or "",
                    screenshot_classifier_version=row.get("screenshot_classifier_version") or "",
                    screenshot_classifier_provider=row.get("screenshot_classifier_provider") or "",
                    screenshot_classifier_status=row.get("screenshot_classifier_status") or "unknown",
                )
            )
        # Rank channels first: cross-channel scores are not directly comparable.
        hits.sort(key=lambda hit: (rank[hit.matched_by], -hit.score))
        return hits[:limit]
