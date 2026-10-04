"""CocoIndex v1 root target for the image collection: named single + multi-vector objects.

> _Byline: Claude Code · Fable 5.1 · 2026-09-21_
Same contract as `weaviate_target.py` (idempotent objects, retirement marks `active=false`,
never deletes, schema provisioned outside the indexing process) but its own provider id,
collection and writer: the image index is a separate app and never shares the text
collection's identity. `image_maxsim` is a bag of 128-d vectors scored with MaxSim.
"""

from __future__ import annotations

import hashlib
import json
import math
from collections.abc import Collection, Sequence
from dataclasses import dataclass
from uuid import UUID

import cocoindex as coco
import httpx

from .image_embedders import MULTI_DIMENSIONS

SINGLE_VECTOR = "image_single"
MULTI_VECTOR = "image_maxsim"
TEXT_PROPERTIES = (
    "source_id",
    "source_path",
    "filename",
    "content_sha256",
    "source_content_sha1",
    "original_time",
    "original_time_source",
    "original_time_confidence",
    "device",
    "software",
    "gps",
    "ocr_text",
    "embed_single_model",
    "embed_multi_model",
    "notes",
    # Catalog-source additions (Claude Code · Sonnet 5.5 · 2026-10-03): where the image lives and
    # what kind it is.
    "provider",
    "bucket",
    "vault_key",
    "image_kind",
    "image_kind_basis",
    "source_kind",
    "ocr_engine",
    "identity",
    "embed_slots",
)
BOOL_PROPERTIES = ("active", "is_screenshot", "original_time_conflict")
INT_PROPERTIES = ("width", "height", "page", "page_count", "occurrences")
CLIP_VECTOR = "image_clip"
COLQWEN_VECTOR = "image_colqwen"
CLIP_BLOB = "thumb"


def collection_schema(collection: str, *, clip: bool = False, colqwen: bool = False) -> dict:
    """The schema this writer requires; provisioned explicitly by `image-provision` or
    `ensure_schema`.

    ``clip=True`` adds the in-database CLIP option: a ``thumb`` blob property (a 224-px JPEG, what
    CLIP sees) and an
    ``image_clip`` named vector that Weaviate's ``multi2vec-clip`` module computes on insert from
    that blob. It needs the
    module and its inference container enabled on the Weaviate instance; without them the collection
    create is refused."""
    hnsw = {"distance": "cosine"}
    properties = (
        [{"name": name, "dataType": ["text"]} for name in TEXT_PROPERTIES]
        + [{"name": name, "dataType": ["boolean"]} for name in BOOL_PROPERTIES]
        + [{"name": name, "dataType": ["int"]} for name in INT_PROPERTIES]
    )
    if clip:
        properties.append({"name": CLIP_BLOB, "dataType": ["blob"]})
    schema = {
        "class": collection,
        "properties": properties,
        "vectorConfig": {
            SINGLE_VECTOR: {
                "vectorizer": {"none": {}},
                "vectorIndexType": "hnsw",
                "vectorIndexConfig": hnsw,
            },
            MULTI_VECTOR: {
                "vectorizer": {"none": {}},
                "vectorIndexType": "hnsw",
                "vectorIndexConfig": {**hnsw, "multivector": {"enabled": True}},
            },
        },
    }
    if colqwen:
        schema["vectorConfig"][COLQWEN_VECTOR] = {
            "vectorizer": {"none": {}},
            "vectorIndexType": "hnsw",
            "vectorIndexConfig": {**hnsw, "multivector": {"enabled": True}},
        }
    if clip:
        schema["vectorConfig"][CLIP_VECTOR] = {
            "vectorizer": {
                "multi2vec-clip": {"imageFields": [CLIP_BLOB], "vectorizeCollectionName": False}
            },
            "vectorIndexType": "hnsw",
            "vectorIndexConfig": hnsw,
        }
    return schema


@dataclass(frozen=True)
class ImageObjectSpec:
    properties: dict[str, str | bool | int]
    single: list[float] | None
    multi: list[list[float]] | None
    # Further named multi-vectors (image_colqwen), vector name -> bag of 128-d vectors.
    extra: dict[str, list[list[float]]] | None = None


@dataclass(frozen=True)
class ImageObjectAction:
    key: tuple[str, str, str]  # origin, collection, UUID
    spec: ImageObjectSpec | None


class ImageObjectWriter:
    def __init__(
        self, url: str, collection: str, single_dimensions: int, client: httpx.AsyncClient
    ):
        if not url.startswith(("http://", "https://")) or not collection.isidentifier():
            raise ValueError("Image index needs a Weaviate URL and a plain collection name")
        self.url = url.rstrip("/")
        self.collection = collection
        self.single_dimensions = single_dimensions
        self.client = client

    async def ensure_schema(self, *, clip: bool = False, colqwen: bool = False) -> list[str]:
        """Create the collection, or add the properties an older one lacks. Additive only: never
        alters or drops.

        Returns the names created or added. A missing named vector on an existing collection is an
        error that names
        it (Weaviate cannot add a vector to a populated collection through the properties
        endpoint)."""
        response = await self.client.get(f"{self.url}/v1/schema/{self.collection}")
        if response.status_code == 404:
            wanted = collection_schema(self.collection, clip=clip, colqwen=colqwen)
            created = await self.client.post(f"{self.url}/v1/schema", json=wanted)
            if created.status_code != 200:
                raise RuntimeError(
                    f"create {self.collection}: HTTP {created.status_code}: {created.text[:300]}"
                )
            return [p["name"] for p in wanted["properties"]]
        response.raise_for_status()
        schema = response.json()
        have = {item["name"] for item in schema.get("properties", [])}
        wanted = collection_schema(self.collection, clip=clip, colqwen=colqwen)
        self._verify_vectors(schema)
        for needed, flag in ((CLIP_VECTOR, clip), (COLQWEN_VECTOR, colqwen)):
            if flag and needed not in (schema.get("vectorConfig") or {}):
                raise RuntimeError(
                    f"collection {self.collection} has no named vector {needed!r}; "
                    "recreate it with the option enabled"
                )
        added: list[str] = []
        for prop in wanted["properties"]:
            if prop["name"] in have:
                continue
            posted = await self.client.post(
                f"{self.url}/v1/schema/{self.collection}/properties", json=prop
            )
            if posted.status_code != 200:
                raise RuntimeError(
                    f"add property {prop['name']}: HTTP {posted.status_code}: {posted.text[:200]}"
                )
            added.append(prop["name"])
        return added

    async def verify_schema(self) -> None:
        """Check required properties and named vectors before image publication.

        Inputs: configured collection. Output: none, or ValueError on incompatibility.
        Side effects: one schema read. Pick this before writes to an existing collection.
        """
        response = await self.client.get(f"{self.url}/v1/schema/{self.collection}")
        response.raise_for_status()
        schema = response.json()
        properties = {item["name"]: item["dataType"] for item in schema.get("properties", [])}
        for name in TEXT_PROPERTIES:
            if properties.get(name) != ["text"]:
                raise ValueError(f"Image collection missing text property: {name}")
        for name in BOOL_PROPERTIES:
            if properties.get(name) != ["boolean"]:
                raise ValueError(f"Image collection missing boolean property: {name}")
        for name in INT_PROPERTIES:
            if properties.get(name) != ["int"]:
                raise ValueError(f"Image collection missing int property: {name}")
        self._verify_vectors(schema)

    def _verify_vectors(self, schema: dict) -> None:
        """Reject incompatible vector layouts without changing existing schema.

        Inputs: Weaviate schema. Output: none or ValueError. Side effects: none.
        Pick this before additive property updates so an incompatible collection stays intact.
        """
        vectors = schema.get("vectorConfig", {})
        for name in (SINGLE_VECTOR, MULTI_VECTOR):
            if vectors.get(name, {}).get("vectorizer") != {"none": {}}:
                raise ValueError(f"Named vector {name} must explicitly use the none vectorizer")
        multivector = vectors[MULTI_VECTOR].get("vectorIndexConfig", {}).get("multivector", {})
        if not multivector.get("enabled"):
            raise ValueError("image_maxsim must be a multi-vector index")

    def _check(self, spec: ImageObjectSpec) -> None:
        if spec.single is None and spec.multi is None and not spec.extra:
            raise ValueError("An image object needs at least one vector")
        if spec.single is not None and (
            len(spec.single) != self.single_dimensions
            or not all(math.isfinite(v) for v in spec.single)
        ):
            raise ValueError("Invalid single image vector")
        for bag in (spec.extra or {}).values():
            if not bag or any(len(v) != MULTI_DIMENSIONS for v in bag):
                raise ValueError("Invalid extra MaxSim vector bag")
        if spec.multi is not None and (
            not spec.multi or any(len(v) != MULTI_DIMENSIONS for v in spec.multi)
        ):
            raise ValueError("Invalid MaxSim vector bag")

    async def apply(self, action: ImageObjectAction) -> None:
        origin, collection, object_id = action.key
        UUID(object_id)  # Bound path segment, never arbitrary input URL.
        if (origin, collection) != (self.url, self.collection):
            raise ValueError("Target identity changed; restore the original writer configuration")
        path = f"{origin}/v1/objects/{collection}/{object_id}"
        try:
            existing = await self.client.get(path)
            if existing.status_code != 404:
                existing.raise_for_status()
            if action.spec is None:
                if existing.status_code != 404:
                    result = await self.client.patch(
                        path,
                        json={
                            "class": collection,
                            "id": object_id,
                            "properties": {"active": False},
                        },
                    )
                    result.raise_for_status()
                return
            self._check(action.spec)
            vectors: dict[str, object] = {}
            if action.spec.single is not None:
                vectors[SINGLE_VECTOR] = action.spec.single
            if action.spec.multi is not None:
                vectors[MULTI_VECTOR] = action.spec.multi
            vectors.update(action.spec.extra or {})
            payload = {
                "class": collection,
                "id": object_id,
                "properties": {**action.spec.properties, "active": True},
                "vectors": vectors,
            }
            if existing.status_code == 404:
                result = await self.client.post(f"{origin}/v1/objects", json=payload)
                if result.status_code in {409, 422}:
                    check = await self.client.get(path)
                    check.raise_for_status()
                    result = await self.client.put(path, json=payload)
            else:
                result = await self.client.put(path, json=payload)
            result.raise_for_status()
        except httpx.HTTPError as exc:
            raise RuntimeError("Weaviate image write failed; state remains retryable") from exc


IMAGE_WRITER = coco.ContextKey[ImageObjectWriter]("intake_images_weaviate_writer")


class ImageObjectHandler:
    def __init__(self):
        self.sink = coco.TargetActionSink.from_async_fn(self.apply_actions)

    async def apply_actions(
        self,
        context_provider: coco.ContextProvider,
        actions: Sequence[ImageObjectAction],
        /,
    ) -> None:
        writer = context_provider.get(IMAGE_WRITER)
        for action in actions:  # sequential: a MaxSim bag is ~0.4 MB per request
            await writer.apply(action)

    def reconcile(
        self,
        key: coco.StableKey,
        desired_target_state: ImageObjectSpec | coco.NonExistenceType,
        prev_possible_records: Collection[str],
        prev_may_be_missing: bool,
        /,
    ) -> coco.TargetReconcileOutput | None:
        if not isinstance(key, tuple) or len(key) != 3:
            raise ValueError("Invalid Weaviate image target identity")
        if coco.is_non_existence(desired_target_state):
            if not prev_possible_records and not prev_may_be_missing:
                return None
            return coco.TargetReconcileOutput(
                ImageObjectAction(key, None), self.sink, coco.NON_EXISTENCE
            )
        serialized = json.dumps(
            {
                "properties": desired_target_state.properties,
                "single": desired_target_state.single,
                "multi": desired_target_state.multi,
                "extra": desired_target_state.extra,
            },
            sort_keys=True,
            separators=(",", ":"),
            allow_nan=False,
        )
        fingerprint = hashlib.sha256(serialized.encode()).hexdigest()
        if (
            prev_possible_records
            and not prev_may_be_missing
            and all(record == fingerprint for record in prev_possible_records)
        ):
            return None
        return coco.TargetReconcileOutput(
            ImageObjectAction(key, desired_target_state),
            self.sink,
            fingerprint,
        )


_provider = coco.register_root_target_states_provider(
    "intake/images/weaviate/object",
    ImageObjectHandler(),
)


@coco.fn
def declare_image(origin: str, collection: str, object_id: str, spec: ImageObjectSpec) -> None:
    coco.declare_target_state(
        _provider.target_state(
            (origin.rstrip("/"), collection, str(UUID(object_id))),
            spec,
        )
    )
