"""Named embedding slots: one Weaviate named vector per embedding model.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Owner 2026-10-02: more than one embedding model, side by side. The shape follows CocoIndex's
named-vector support
(``vectors={"text": ..., "image": ...}`` on one point/object, one ``VectorDef`` per name; CocoIndex
v1 docs,
https://cocoindex.io/docs-v1/connectors/qdrant, and the multi-vector write-up
https://cocoindex.io/blogs/multi-vector/).
Weaviate takes the same shape: ``vectors: {slot: [...]}`` on the object, one ``vectorConfig`` entry
per slot.

Slots (``INTAKE_VECTOR_SLOTS``, comma separated, default ``text_nim``):

* ``text_nim``  NVIDIA NIM ``nvidia/nemotron-3-embed-1b``, 2048-d, a real batched ``/embeddings``
array, with the
  input guards the endpoint needs: the lowercase text ``data:image/`` is rewritten to ``data:
  image/`` and a blank
  input gets a placeholder (either one fails the WHOLE request, owner notes 2026-09-26).
* ``legal``     a pluggable slot for the legal embedder another agent is choosing (current MLEB
leader; voyage-law-2
  selectable). Nothing is assumed: ``INTAKE_VECTOR_LEGAL_MODEL`` must be set to enable it, and its
  dimensions, base
  URL, key name and request style are settings, so the choice is configuration, not code.

Asymmetric models need a per-call ``input_type`` (owner 2026-07-11): documents are embedded as
``passage``/``document``
here; a query client must send the query side itself.
"""

from __future__ import annotations

import asyncio
import math
import os
import random
from collections.abc import Mapping, Sequence
from dataclasses import dataclass, field

import httpx

from .secrets import get_secret

BLANK_PLACEHOLDER = "(empty)"
MAX_INPUT_CHARS = 8000
TRANSIENT_STATUS = {408, 409, 425, 429, 500, 502, 503, 504}
EMBED_FLOOR_CHARS = 256

DEFAULT_SLOT = "text_nim"
STYLES = ("nim", "voyage", "openai")


def guard_input(text: str) -> str:
    """The NIM input guards. Rewrites rather than drops, so a batch keeps its length and order."""
    text = text.replace("data:image/", "data: image/")
    if len(text) > MAX_INPUT_CHARS:
        text = text[:MAX_INPUT_CHARS]
    return text if text.strip() else BLANK_PLACEHOLDER


class EmbedError(RuntimeError):
    """The provider failed (outage, rate limit). Carries no key and no corpus text. Retry later;
    never bisect."""


class EmbedRejected(EmbedError):
    """The provider refused THIS input (HTTP 4xx, bad shape): bisecting the batch can isolate the
    culprit."""


@dataclass(frozen=True)
class VectorSlot:
    name: str
    model: str
    dimensions: int
    base_url: str
    key_secret: str
    style: str = "nim"
    batch_size: int = 32
    document_input_type: str | None = "passage"
    timeout_seconds: float = 120.0
    max_retries: int = 4

    def validate(self) -> None:
        if not self.name.replace("_", "").isalnum():
            raise ValueError(f"Vector slot name {self.name!r} must be alphanumeric/underscore")
        if self.style not in STYLES:
            raise ValueError(f"Vector slot {self.name}: style must be one of {STYLES}")
        if self.dimensions <= 0 or self.batch_size <= 0:
            raise ValueError(f"Vector slot {self.name}: dimensions and batch size must be positive")
        if not self.model:
            raise ValueError(f"Vector slot {self.name}: no model configured")

    def request_body(self, inputs: Sequence[str]) -> dict:
        body: dict = {"model": self.model, "input": list(inputs), "encoding_format": "float"}
        if self.style == "nim":
            body["input_type"] = self.document_input_type or "passage"
            body["truncate"] = "END"
        elif self.style == "voyage":
            body["input_type"] = self.document_input_type or "document"
            body["truncation"] = True
            body.pop("encoding_format")
        return body


def _int(env: Mapping[str, str], name: str, default: int) -> int:
    raw = env.get(name, "").strip()
    return int(raw) if raw else default


def load_slots(env: Mapping[str, str] | None = None) -> list[VectorSlot]:
    """Slots from ``INTAKE_VECTOR_SLOTS``. A slot that is named but not configured is an error,
    never a silent skip."""
    env = os.environ if env is None else env
    names = [
        n.strip() for n in env.get("INTAKE_VECTOR_SLOTS", DEFAULT_SLOT).split(",") if n.strip()
    ]
    if not names:
        raise ValueError("INTAKE_VECTOR_SLOTS names no slot")
    slots: list[VectorSlot] = []
    for name in names:
        if name == "text_nim":
            slot = VectorSlot(
                name="text_nim",
                model=env.get("NIM_EMBED_MODEL", "").strip() or "nvidia/nemotron-3-embed-1b",
                dimensions=_int(env, "NIM_EMBED_DIMENSIONS", 2048),
                base_url=(
                    env.get("NIM_BASE_URL", "").strip() or "https://integrate.api.nvidia.com/v1"
                ).rstrip("/"),
                key_secret="NVIDIA_API_KEY",
                style="nim",
                batch_size=_int(env, "NIM_EMBED_BATCH_SIZE", 32),
            )
        elif name == "legal":
            model = env.get("INTAKE_VECTOR_LEGAL_MODEL", "").strip()
            if not model:
                raise ValueError(
                    "Slot 'legal' is enabled but INTAKE_VECTOR_LEGAL_MODEL is not set; "
                    "the legal embedder is "
                    "chosen by configuration"
                )
            style = env.get("INTAKE_VECTOR_LEGAL_STYLE", "voyage").strip()
            slot = VectorSlot(
                name="legal",
                model=model,
                dimensions=_int(env, "INTAKE_VECTOR_LEGAL_DIMENSIONS", 1024),
                base_url=(
                    env.get("INTAKE_VECTOR_LEGAL_BASE_URL", "").strip()
                    or "https://api.voyageai.com/v1"
                ).rstrip("/"),
                key_secret=env.get("INTAKE_VECTOR_LEGAL_KEY_SECRET", "").strip()
                or "VOYAGE_API_KEY",
                style=style,
                batch_size=_int(env, "INTAKE_VECTOR_LEGAL_BATCH_SIZE", 32),
                document_input_type=env.get("INTAKE_VECTOR_LEGAL_INPUT_TYPE", "").strip() or None,
            )
        else:
            raise ValueError(f"Unknown vector slot {name!r}; known: text_nim, legal")
        slot.validate()
        slots.append(slot)
    if len({s.name for s in slots}) != len(slots):
        raise ValueError("INTAKE_VECTOR_SLOTS repeats a slot")
    return slots


@dataclass
class SlotEmbedder:
    """Embeds document text for one slot; never returns a partial batch and never raises on one bad
    text."""

    slot: VectorSlot
    client: httpx.AsyncClient
    api_key: str = field(repr=False, default="")
    requests: int = 0

    @classmethod
    def for_slot(cls, slot: VectorSlot, client: httpx.AsyncClient) -> SlotEmbedder:
        key = get_secret(slot.key_secret)
        if not key:
            raise ValueError(f"{slot.key_secret} is not configured for vector slot {slot.name}")
        return cls(slot=slot, client=client, api_key=key)

    async def _post(self, inputs: list[str]) -> list[list[float]]:
        headers = {"Authorization": f"Bearer {self.api_key}", "Content-Type": "application/json"}
        last = "no attempt made"
        for attempt in range(self.slot.max_retries + 1):
            if attempt:
                await asyncio.sleep(min(12.0, 0.75 * 2**attempt) + random.random() * 0.25)
            self.requests += 1
            try:
                response = await self.client.post(
                    f"{self.slot.base_url}/embeddings",
                    json=self.slot.request_body(inputs),
                    headers=headers,
                    timeout=self.slot.timeout_seconds,
                )
            except httpx.HTTPError as error:
                last = f"transport error {type(error).__name__}"
                continue
            if response.status_code in TRANSIENT_STATUS:
                last = f"HTTP {response.status_code}"
                continue
            if response.status_code != 200:
                raise EmbedRejected(
                    f"{self.slot.name}: provider returned HTTP {response.status_code}"
                )
            data = sorted(response.json().get("data", []), key=lambda item: item.get("index", 0))
            vectors = [[float(x) for x in item["embedding"]] for item in data]
            if len(vectors) != len(inputs):
                raise EmbedError(
                    f"{self.slot.name}: {len(vectors)} vectors for {len(inputs)} inputs"
                )
            for vector in vectors:
                if len(vector) != self.slot.dimensions:
                    raise EmbedRejected(f"{self.slot.name}: vector is not {self.slot.dimensions}-d")
                if not all(math.isfinite(x) for x in vector) or not any(vector):
                    raise EmbedRejected(
                        f"{self.slot.name}: provider returned a zero or non-finite vector"
                    )
            return vectors
        raise EmbedError(
            f"{self.slot.name}: failed after {self.slot.max_retries + 1} attempts ({last})"
        )

    async def embed(self, texts: Sequence[str]) -> tuple[list[list[float]], list[str]]:
        """One vector and one status per text, in order. Status: ok | truncated | failed.

        A chunk is bounded in characters, the provider in tokens; dense text (minified CSS or JSON
        inside an export)
        can be refused. The batch is bisected, then a single refused text is halved down to a floor
        and recorded
        ``truncated``; one that still fails is ``failed`` with a zero vector the publisher will not
        write.
        """
        out_vectors: list[list[float]] = []
        out_status: list[str] = []
        for start in range(0, len(texts), self.slot.batch_size):
            part = [guard_input(t) for t in texts[start : start + self.slot.batch_size]]
            vectors, statuses = await self._bisect(part)
            out_vectors.extend(vectors)
            out_status.extend(statuses)
        return out_vectors, out_status

    async def _bisect(self, part: list[str]) -> tuple[list[list[float]], list[str]]:
        try:
            return await self._post(part), ["ok"] * len(part)
        except EmbedRejected:
            if len(part) > 1:
                middle = len(part) // 2
                left = await self._bisect(part[:middle])
                right = await self._bisect(part[middle:])
                return left[0] + right[0], left[1] + right[1]
        text = part[0]
        while len(text) > EMBED_FLOOR_CHARS:
            text = text[: len(text) // 2]
            try:
                return await self._post([text]), ["truncated"]
            except EmbedRejected:
                continue
        return [[0.0] * self.slot.dimensions], ["failed"]
