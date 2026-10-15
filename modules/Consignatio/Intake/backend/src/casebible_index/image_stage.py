"""The image stage's units after fetch: facts, ocr, embed (one slot), publish. One job each.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

Each ``run_*`` function takes a slice (``image_slice.py``) and writes one small Parquet keyed by
``identity`` under
``datasets/images/<table>/<slice_id>.parquet``. A retry finds its own output already written and
returns the same counts
without repeating the work. None of them knows its caller (direct call, Temporal Activity, n8n
node).

    run_facts      original time, device, GPS, size, format, screenshot / photo / scan        ->
    facts
    run_ocr        the text in the picture with the selected engine                            ->
    ocr
    run_embed      one vector slot: ``image_maxsim`` (Jina bag), ``image_single`` (NIM/Google),
    ``image_colqwen``  -> vectors/<slot>
    run_publish    facts + text + vectors -> Weaviate ``IntakeImageV1``; ledger; release the spool
    -> published

They extend Intake's image index rather than replace it: the embedders (``image_embedders``), facts
reader
(``image_facts``), original-time resolver, Weaviate writer/schema (``image_target``) and search
(``image_search``) are
the existing ones. What changed is the SOURCE (the catalog and bucket instead of a directory) and
the WRITE path
(direct upserts from Parquet-tracked slices instead of CocoIndex per-object target state, which the
text lane measured at
3.16 GiB for a 61 MB export).
"""

from __future__ import annotations

import asyncio
import base64
import hashlib
import os
from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any
from uuid import NAMESPACE_URL, UUID, uuid5

import pyarrow as pa
import pyarrow.parquet as pq

from .image_decode import NATIVE_PROVIDER_EXTENSIONS, jpeg_derivative, open_image
from .image_embedders import SINGLE_DIMENSIONS
from .image_facts import read_image_facts
from .image_kind import KIND_PHOTO, KIND_SCAN, KIND_SCREENSHOT, classify_image
from .image_ocr import DEFAULT_ENGINE, read_text
from .image_screenshot_classifier import CHAIN_MODEL as SCREENSHOT_CHAIN_MODEL
from .image_screenshot_classifier import CHAIN_PROVIDER as SCREENSHOT_CHAIN_PROVIDER
from .image_screenshot_classifier import DECISIONS_MODEL as SCREENSHOT_DECISIONS_MODEL
from .image_screenshot_classifier import MODEL as SCREENSHOT_NIM_MODEL
from .image_screenshot_classifier import (
    cached_classification,
    classifier_version,
    load_reference_bundle,
)
from .image_screenshot_classifier import classify as classify_screenshot
from .image_screenshot_classifier import unknown as unknown_screenshot
from .image_slice import (
    DEFAULT_MAX_BYTES,
    DEFAULT_MAX_FILE_BYTES,
    DEFAULT_MAX_FILES,
    ImageItem,
    Slice,
    item_path,
    lake,
    load_slice,
    parse_image_allowlist,
    published_pdf_pages,
    quarantine_spool,
    spool_root,
    table_path,
)
from .image_target import (
    CLIP_BLOB,
    CLIP_VECTOR,
    COLQWEN_VECTOR,
    MULTI_VECTOR,
    SINGLE_VECTOR,
    ImageObjectAction,
    ImageObjectSpec,
)

Beat = Callable[[str], None]

SLOT_SINGLE = SINGLE_VECTOR
SLOT_MAXSIM = MULTI_VECTOR
SLOT_COLQWEN = COLQWEN_VECTOR
MAX_EMBED_EDGE = 2048
MAX_EMBED_BYTES = 4 * 1024 * 1024
THUMB_EDGE = 224


SLOT_CLIP = CLIP_VECTOR
ALL_SLOTS = (SLOT_MAXSIM, SLOT_SINGLE, SLOT_COLQWEN)
TIERS = ("all", "clip_first")
POLICIES = ("screenshots", "all", "none")

# Why each hosted slot may be used on this corpus. Recorded in every embed receipt (owner decisions
# 2026-10-03).
SLOT_LICENCE = {
    SLOT_MAXSIM: "Jina jina-embeddings-v4 (Qwen2-VL base, Qwen Research License: "
    "research and non-commercial use). "
    "Owner 2026-10-03 09:34: personal, non-commercial case use; "
    "accepted for screenshots and scanned pages.",
    SLOT_SINGLE: "NVIDIA NIM nvidia/llama-nemotron-embed-vl-1b-v2 on the existing account "
    "(hosted trial terms; "
    "NVIDIA's data-retention terms not yet read).",
    SLOT_COLQWEN: "ColQwen2.5 (vidore/colqwen2.5-v0.2, Apache-2.0 adapter on the Qwen2.5-VL base) "
    "self-hosted on the "
    "owner's own GPU endpoint; images never leave that endpoint.",
}


@dataclass(frozen=True)
class ImageStageSettings:
    """Every image-stage setting, from ``INTAKE_IMAGE*`` environment names. Off unless
    ``INTAKE_IMAGE_STAGE=on``.

    Slots (``INTAKE_IMAGES_SLOTS``, default ``image_maxsim,image_single``) are the hosted embedders
    that fill a named
    vector each: ``image_maxsim`` Jina v4 multi-vector (the PRIMARY, owner 2026-10-03),
    ``image_single`` NVIDIA NIM or
    Google single vector, ``image_colqwen`` a ColQwen2.5 endpoint (selectable, off until its URL is
    set). ``clip``
    adds Weaviate's in-database CLIP vector (``image_clip``) on a 224-px thumbnail. Which images
    each multi-vector
    slot covers is a policy (``INTAKE_IMAGES_MAXSIM`` / ``INTAKE_IMAGES_COLQWEN``: screenshots, all,
    none). The photo
    tier (``INTAKE_IMAGES_PHOTO_TIER``) decides how photos are covered: ``all`` gives every image
    every enabled slot
    (CLIP, when on, is a second opinion); ``clip_first`` gives photos CLIP only, as the free first
    pass, and keeps the
    hosted slots for screenshots and scans (a shortlisted photo is promoted later with
    ``image_promote``). ``INTAKE_IMAGES_ALLOWED_PREFIXES_JSON`` is a persistent JSON array of
    exact provider, bucket and relative folder path_prefix entries; automatic fetch requires it.
    An explicit exact locator proof can run without that persistent setting. The optional hosted
    screenshot classifier requires ``INTAKE_IMAGES_SCREENSHOT_REFERENCE_MANIFEST`` to point to
    two hash-checked derived examples inside this image output lake; missing references fail
    closed before any provider request."""

    enabled: bool
    weaviate_url: str
    collection: str
    single_provider: str
    slots: tuple[str, ...]
    maxsim: str
    colqwen: str
    colqwen_url: str
    photo_tier: str
    ocr_engine: str
    max_ocr_chars: int
    max_files: int
    max_bytes: int
    max_file_bytes: int
    pdf_max_pages: int
    pdf_dpi: int
    clip: bool
    inflight: int
    allowed_prefixes: tuple[dict[str, str], ...] = ()
    screenshot_classifier_enabled: bool = False
    screenshot_classifier_max_calls: int = 5
    screenshot_reference_manifest: str = ""
    screenshot_classifier_provider: str = "nvidia-nim"
    screenshot_classifier_model: str = ""

    @classmethod
    def from_env(cls) -> ImageStageSettings:
        env = os.environ
        return cls(
            enabled=env.get("INTAKE_IMAGE_STAGE", "off").strip().casefold() == "on",
            weaviate_url=(
                env.get("INTAKE_IMAGES_WEAVIATE_URL") or env.get("INTAKE_WEAVIATE_URL") or ""
            ).rstrip("/"),
            collection=env.get("INTAKE_IMAGES_COLLECTION", "IntakeImageV1").strip()
            or "IntakeImageV1",
            single_provider=env.get("INTAKE_IMAGES_SINGLE_PROVIDER", "nim").strip(),
            slots=tuple(
                s.strip()
                for s in env.get("INTAKE_IMAGES_SLOTS", "image_maxsim,image_single").split(",")
                if s.strip()
            ),
            maxsim=env.get("INTAKE_IMAGES_MAXSIM", "all").strip(),
            colqwen=env.get("INTAKE_IMAGES_COLQWEN", "screenshots").strip(),
            colqwen_url=env.get("INTAKE_IMAGES_COLQWEN_URL", "").strip(),
            photo_tier=env.get("INTAKE_IMAGES_PHOTO_TIER", "all").strip(),
            ocr_engine=env.get("INTAKE_IMAGES_OCR_ENGINE", DEFAULT_ENGINE).strip(),
            max_ocr_chars=int(env.get("INTAKE_IMAGES_MAX_OCR_CHARS", "20000")),
            max_files=int(env.get("INTAKE_IMAGES_SLICE_FILES", str(DEFAULT_MAX_FILES))),
            max_bytes=int(env.get("INTAKE_IMAGES_SLICE_BYTES", str(DEFAULT_MAX_BYTES))),
            max_file_bytes=int(
                env.get("INTAKE_IMAGES_MAX_FILE_BYTES", str(DEFAULT_MAX_FILE_BYTES))
            ),
            pdf_max_pages=int(env.get("INTAKE_IMAGES_PDF_MAX_PAGES", "20")),
            pdf_dpi=int(env.get("INTAKE_IMAGES_PDF_DPI", "150")),
            clip=env.get("INTAKE_IMAGES_CLIP", "0").strip() == "1",
            inflight=int(env.get("INTAKE_IMAGES_MAX_INFLIGHT", "4")),
            allowed_prefixes=parse_image_allowlist(
                env.get("INTAKE_IMAGES_ALLOWED_PREFIXES_JSON", "")
            ),
            screenshot_classifier_enabled=env.get("INTAKE_IMAGES_SCREENSHOT_CLASSIFIER", "off").strip().casefold() == "on",
            screenshot_classifier_max_calls=int(env.get("INTAKE_IMAGES_SCREENSHOT_CLASSIFIER_MAX_CALLS", "5")),
            screenshot_reference_manifest=env.get("INTAKE_IMAGES_SCREENSHOT_REFERENCE_MANIFEST", "").strip(),
            screenshot_classifier_provider=env.get("INTAKE_IMAGES_SCREENSHOT_PROVIDER", "nvidia-nim").strip(),
            screenshot_classifier_model=env.get("INTAKE_IMAGES_SCREENSHOT_MODEL", "").strip(),
        )

    def validate(self) -> None:
        if self.single_provider not in SINGLE_DIMENSIONS:
            raise ValueError("INTAKE_IMAGES_SINGLE_PROVIDER must be 'nim' or 'google'")
        for name, value in (
            ("INTAKE_IMAGES_MAXSIM", self.maxsim),
            ("INTAKE_IMAGES_COLQWEN", self.colqwen),
        ):
            if value not in POLICIES:
                raise ValueError(f"{name} must be one of {POLICIES}")
        if self.photo_tier not in TIERS:
            raise ValueError(f"INTAKE_IMAGES_PHOTO_TIER must be one of {TIERS}")
        unknown = [s for s in self.slots if s not in ALL_SLOTS]
        if unknown or len(set(self.slots)) != len(self.slots):
            raise ValueError(
                f"INTAKE_IMAGES_SLOTS must list distinct slots from {ALL_SLOTS}, got {self.slots}"
            )
        if not self.slots and not self.clip:
            raise ValueError("The image stage needs at least one slot or INTAKE_IMAGES_CLIP=1")
        if self.photo_tier == "clip_first" and not self.clip:
            raise ValueError("INTAKE_IMAGES_PHOTO_TIER=clip_first needs INTAKE_IMAGES_CLIP=1")
        if SLOT_COLQWEN in self.slots and not self.colqwen_url:
            raise ValueError("image_colqwen is enabled but INTAKE_IMAGES_COLQWEN_URL is not set")
        if (
            min(
                self.max_files,
                self.max_bytes,
                self.max_file_bytes,
                self.pdf_max_pages,
                self.inflight,
            )
            < 1
        ):
            raise ValueError("INTAKE_IMAGES_* limits must be positive")
        if not 1 <= self.screenshot_classifier_max_calls <= 5:
            raise ValueError("INTAKE_IMAGES_SCREENSHOT_CLASSIFIER_MAX_CALLS must be 1..5")
        model = self.screenshot_classifier_model or (
            SCREENSHOT_CHAIN_MODEL
            if self.screenshot_classifier_provider == SCREENSHOT_CHAIN_PROVIDER
            else SCREENSHOT_DECISIONS_MODEL
            if self.screenshot_classifier_provider == "openrouter-decisions"
            else SCREENSHOT_NIM_MODEL
        )
        classifier_version(self.screenshot_classifier_provider, model)

    def slot_applies(self, slot: str, kind: str) -> bool:
        """Does ``slot`` embed an image of this kind? The one place the policy and the photo tier
        are decided."""
        if slot not in self.slots:
            return False
        if kind == KIND_PHOTO and self.photo_tier == "clip_first":
            return False
        policy = {SLOT_SINGLE: "all", SLOT_MAXSIM: self.maxsim, SLOT_COLQWEN: self.colqwen}[slot]
        return policy == "all" or (policy == "screenshots" and kind in (KIND_SCREENSHOT, KIND_SCAN))

    def licence_basis(self) -> dict[str, str]:
        return {slot: SLOT_LICENCE[slot] for slot in self.slots}


def _write_table(path: Path, table: pa.Table) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    partial = path.with_suffix(".partial")
    pq.write_table(table, partial, compression="zstd")
    partial.replace(path)


def _read_rows(path: Path) -> dict[str, dict[str, Any]]:
    return (
        {row["identity"]: row for row in pq.read_table(path).to_pylist()} if path.is_file() else {}
    )


# -------------------------------------------------- facts

FACTS_SCHEMA = pa.schema(
    [
        ("identity", pa.string()),
        ("content_sha256", pa.string()),
        ("original_time", pa.string()),
        ("original_time_source", pa.string()),
        ("original_time_confidence", pa.string()),
        ("original_time_conflict", pa.bool_()),
        ("device", pa.string()),
        ("software", pa.string()),
        ("gps", pa.string()),
        ("width", pa.int32()),
        ("height", pa.int32()),
        ("format", pa.string()),
        ("kind", pa.string()),
        ("kind_basis", pa.string()),
        ("notes", pa.string()),
        ("screenshot_decision", pa.string()),
        ("screenshot_app", pa.string()),
        ("screenshot_app_confidence", pa.int32()),
        ("screenshot_app_basis", pa.string()),
        ("screenshot_classifier_version", pa.string()),
        ("screenshot_classifier_provider", pa.string()),
        ("screenshot_classifier_status", pa.string()),
    ]
)


def _facts_for(path: Path, item: ImageItem) -> dict[str, Any]:
    """Read metadata and stream the retained image or PDF page's actual SHA256 digest.

    Inputs: retained path and slice item. Output: facts row. Side effects: file reads and metadata
    inspection. Pick this for catalog image facts; source SHA1 remains on the slice item.
    """
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for block in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(block)
    with open_image(path) as image:
        width, height, fmt = image.width, image.height, image.format or ""
    facts = read_image_facts(path, item.name, takeout_sidecar_json=None, ocr=False, max_ocr_chars=0)
    kind = classify_image(
        item.name,
        width,
        height,
        format_name=fmt,
        has_camera_exif=bool(facts.device),
        is_pdf_page=bool(item.page),
        user_comment="screenshot" if facts.is_screenshot else "",
    )
    time = facts.original_time
    return {
        "identity": item.identity,
        "content_sha256": digest.hexdigest(),
        "original_time": time.value or "",
        "original_time_source": time.source or "",
        "original_time_confidence": time.confidence or "",
        "original_time_conflict": bool(time.conflict),
        "device": facts.device,
        "software": facts.software,
        "gps": facts.gps,
        "width": width,
        "height": height,
        "format": fmt,
        "kind": kind.kind,
        "kind_basis": kind.basis,
        "notes": "; ".join(facts.notes),
    }


def _heuristic_screenshot(row: dict[str, Any]) -> dict[str, Any]:
    """Record what strong metadata rules establish while leaving visual app identity unknown.

    Inputs: legacy facts row. Output: flat classifier fields. Side effects: none. Pick this when
    hosted recognition is disabled; PNG format and screen geometry alone stay unknown.
    """
    basis = row.get("kind_basis", "")
    result = unknown_screenshot("heuristic", basis)
    result["screenshot_classifier_provider"] = "metadata"
    if basis == "name_hint":
        result["screenshot_decision"] = "yes"
    elif basis in {"camera_exif", "pdf_page"}:
        result["screenshot_decision"] = "no"
    return result


def _merge_screenshot(row: dict[str, Any], result: dict[str, Any]) -> None:
    """Merge a validated classifier proposal into one facts row without changing provenance.

    Inputs: mutable facts row and flat proposal. Output: none. Side effects: row mutation only.
    Pick this after a model or metadata decision, before embedding policy reads image kind.
    """
    row.update({key: value for key, value in result.items() if not key.startswith("_")})
    if (
        result["screenshot_classifier_provider"]
        in {"nvidia-nim", "openrouter-decisions", SCREENSHOT_CHAIN_PROVIDER}
        and result["screenshot_classifier_status"] == "model_ok"
    ):
        if result["screenshot_decision"] == "yes":
            row["kind"] = KIND_SCREENSHOT
            row["kind_basis"] = "vision_model"
        elif result["screenshot_decision"] == "no" and row["kind"] != KIND_SCAN:
            row["kind"] = KIND_PHOTO
            row["kind_basis"] = "vision_model"


async def run_facts(
    output_dir: Path, spool_dir: Path | None, sl: Slice, *,
    settings: ImageStageSettings | None = None,
    classifier_key: str = "",
    classifier_client: Any = None,
    beat: Beat | None = None,
) -> dict[str, Any]:
    """Read the facts of every image in the slice and classify it as screenshot, photo or scan.

    Inputs: the slice. Output: counts by kind and how many had an original time. Side effects: runs
    ``exiftool`` per
    image (optional binary: its absence is noted per image, never guessed) and writes
    ``facts/<slice>.parquet``, including streamed SHA256 of the retained image/page bytes.
    PDF page hashes identify rendered PNG bytes; the catalog SHA1 continues to identify the PDF
    original. Legacy facts without this digest are regenerated from the retained files. An
    unreadable image gets a row with kind ``photo``, basis ``unreadable`` and a note, so later units
    can skip it. Optional screenshot recognition reads two checked references for a model-only
    canvas; OCR and embeddings still receive the original fetched image. Pick this before OCR and
    embedding to establish image facts and byte provenance.
    """
    out = table_path(output_dir, "facts", sl.slice_id)
    existing = out.is_file() and "content_sha256" in pq.read_schema(out).names
    if existing:
        rows = list(_read_rows(out).values())
    else:
        rows = []
        for n, item in enumerate(sl.items):
            path = item_path(output_dir, spool_dir, sl.slice_id, item)
            try:
                rows.append(await asyncio.to_thread(_facts_for, path, item))
            except Exception as exc:  # noqa: BLE001 - one unreadable file must not fail the slice
                rows.append(
                    {
                        "identity": item.identity,
                        "content_sha256": "",
                        "original_time": "",
                        "original_time_source": "",
                        "original_time_confidence": "",
                        "original_time_conflict": False,
                        "device": "",
                        "software": "",
                        "gps": "",
                        "width": 0,
                        "height": 0,
                        "format": "",
                        "kind": "photo",
                        "kind_basis": "unreadable",
                        "notes": f"unreadable: {type(exc).__name__}",
                    }
                )
            if beat is not None and n % 10 == 0:
                beat(f"facts {n + 1}/{len(sl.items)}")
    by_identity = {row["identity"]: row for row in rows}
    enabled = bool(settings and settings.screenshot_classifier_enabled)
    provider = getattr(settings, "screenshot_classifier_provider", "nvidia-nim")
    configured_model = getattr(settings, "screenshot_classifier_model", "")
    model = configured_model or (
        SCREENSHOT_CHAIN_MODEL if provider == SCREENSHOT_CHAIN_PROVIDER
        else SCREENSHOT_DECISIONS_MODEL if provider == "openrouter-decisions"
        else SCREENSHOT_NIM_MODEL
    )
    version = classifier_version(provider, model)
    bundle = None
    if enabled and getattr(settings, "screenshot_reference_manifest", ""):
        try:
            bundle = await asyncio.to_thread(
                load_reference_bundle,
                Path(settings.screenshot_reference_manifest),
                output_dir,
                provider=provider,
                model=model,
            )
        except (OSError, ValueError, TypeError):
            bundle = None
    calls = 0
    requests = 0
    cache_hits = 0
    usage = {"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0}
    billed_cost_usd = 0.0
    formats: dict[str, int] = {}
    finish_reasons: dict[str, int] = {}
    changed = not existing
    for item_index, item in enumerate(sl.items):
        row = by_identity.get(item.identity)
        if row is None:
            continue
        if not enabled:
            if "screenshot_decision" not in row:
                _merge_screenshot(row, _heuristic_screenshot(row))
                changed = True
            continue
        if bundle is not None and row.get("screenshot_classifier_version") == bundle.version and row.get("screenshot_classifier_status") in {"model_ok", "model_unknown"}:
            continue
        if item.page:
            _merge_screenshot(row, _heuristic_screenshot(row))
        elif not row.get("content_sha256"):
            _merge_screenshot(row, unknown_screenshot("unreadable", version=version, provider=provider))
        elif bundle is None:
            _merge_screenshot(row, unknown_screenshot("reference_unavailable", version=version, provider=provider))
        else:
            result = cached_classification(
                output_dir, row["content_sha256"], version=bundle.version
            )
            if result is None:
                path = item_path(output_dir, spool_dir, sl.slice_id, item)
                if path.is_file() and classifier_client is not None:
                    result = await classify_screenshot(
                        path, row["content_sha256"], output_dir, classifier_key,
                        classifier_client, bundle,
                        request_budget=max(
                            0, (settings.screenshot_classifier_max_calls if settings else 5) - calls
                        ),
                    )
                    calls += result.get("_request_count", int(bool(result.get("_request_made"))))
                elif not path.is_file() or classifier_client is None:
                    unavailable = unknown_screenshot(
                        "source_unavailable", version=version, provider=provider
                    )
                    _merge_screenshot(row, unavailable)
            if result is not None:
                requests += result.get("_request_count", int(bool(result.get("_request_made"))))
                cache_hits += int(bool(result.get("_cache_hit")))
                for field in usage:
                    usage[field] += result.get("_usage", {}).get(field, 0)
                billed_cost_usd += result.get("_billed_cost_usd", 0.0)
                diagnostic = result.get("_diagnostic", {})
                if diagnostic.get("content_format"):
                    fmt = diagnostic["content_format"]
                    formats[fmt] = formats.get(fmt, 0) + 1
                if diagnostic.get("finish_reason"):
                    finish = diagnostic["finish_reason"]
                    finish_reasons[finish] = finish_reasons.get(finish, 0) + 1
                _merge_screenshot(row, result)
        changed = True
        if beat is not None:
            beat(f"classified {item_index + 1}/{len(sl.items)}")
    if changed:
        _write_table(out, pa.Table.from_pylist(rows, schema=FACTS_SCHEMA))
    kinds: dict[str, int] = {}
    for row in rows:
        kinds[row["kind"]] = kinds.get(row["kind"], 0) + 1
    return {
        "slice_id": sl.slice_id,
        "images": len(rows),
        "kinds": kinds,
        "with_original_time": sum(1 for r in rows if r["original_time"]),
        "unreadable": sum(1 for r in rows if r["kind_basis"] == "unreadable"),
        "screenshot_classifier_budget_units": calls,
        "screenshot_classifier_requests": requests,
        "screenshot_classifier_cache_hits": cache_hits,
        "screenshot_classifier_usage": usage,
        "screenshot_classifier_billed_cost_usd": round(billed_cost_usd, 9),
        "screenshot_reference_version": bundle.version if bundle is not None else "",
        "screenshot_classifier_content_formats": formats,
        "screenshot_classifier_finish_reasons": finish_reasons,
        "screenshot_classifier_statuses": {status: sum(1 for r in rows if r.get("screenshot_classifier_status") == status) for status in sorted({r.get("screenshot_classifier_status", "") for r in rows})},
    }


# -------------------------------------------------- ocr

OCR_SCHEMA = pa.schema(
    [
        ("identity", pa.string()),
        ("ocr_text", pa.string()),
        ("ocr_confidence", pa.float32()),
        ("ocr_engine", pa.string()),
    ]
)


async def run_ocr(
    output_dir: Path,
    spool_dir: Path | None,
    sl: Slice,
    *,
    engine: str = DEFAULT_ENGINE,
    max_chars: int = 20000,
    beat: Beat | None = None,
) -> dict[str, Any]:
    """OCR every image in the slice with the named engine.

    Inputs: the slice, an engine name (``image_ocr.OCR_ENGINES``) and a character cap. Output:
    counts and the mean
    confidence. Side effects: runs the engine per image and writes ``ocr/<slice>.parquet``. An
    engine that cannot run
    raises ``OcrUnavailable`` (non-retryable); a picture with no text is a row with empty text. A
    single image the
    engine fails on is a row with empty text and engine ``<name>:failed``, so a retry does not loop
    on it."""
    out = table_path(output_dir, "ocr", sl.slice_id)
    if out.is_file():
        rows = list(_read_rows(out).values())
    else:
        rows = []
        for n, item in enumerate(sl.items):
            path = item_path(output_dir, spool_dir, sl.slice_id, item)
            try:
                text, confidence = await asyncio.to_thread(read_text, engine, path, max_chars)
                rows.append(
                    {
                        "identity": item.identity,
                        "ocr_text": text,
                        "ocr_confidence": confidence,
                        "ocr_engine": engine,
                    }
                )
            except ValueError:
                raise  # OcrUnavailable and unknown engines are configuration, not a bad image
            except Exception:  # noqa: BLE001
                rows.append(
                    {
                        "identity": item.identity,
                        "ocr_text": "",
                        "ocr_confidence": 0.0,
                        "ocr_engine": f"{engine}:failed",
                    }
                )
            if beat is not None and n % 10 == 0:
                beat(f"ocr {n + 1}/{len(sl.items)}")
        _write_table(out, pa.Table.from_pylist(rows, schema=OCR_SCHEMA))
    with_text = [r for r in rows if r["ocr_text"].strip()]
    return {
        "slice_id": sl.slice_id,
        "images": len(rows),
        "with_text": len(with_text),
        "failed": sum(1 for r in rows if r["ocr_engine"].endswith(":failed")),
        "engine": engine,
        "chars": sum(len(r["ocr_text"]) for r in rows),
        "mean_confidence": round(sum(r["ocr_confidence"] for r in with_text) / len(with_text), 1)
        if with_text
        else 0.0,
    }


# -------------------------------------------------- embed


def prepare_for_embedding(path: Path) -> tuple[bytes, str]:
    """Provide original native bytes or a bounded JPEG for a hosted embedder.

    Inputs: retained image path. Output: provider bytes and matching extension. Side effects:
    reads and sometimes decodes the image. Pick this for hosted image slots; source hashes and
    facts are always computed separately from the unchanged original.
    """
    size = path.stat().st_size
    ext = path.suffix.casefold()
    with open_image(path) as image:
        if (
            ext in NATIVE_PROVIDER_EXTENSIONS
            and size <= MAX_EMBED_BYTES
            and max(image.size) <= MAX_EMBED_EDGE
        ):
            return path.read_bytes(), ext
    return jpeg_derivative(path, edge=MAX_EMBED_EDGE), ".jpg"


def thumbnail_b64(path: Path, edge: int = THUMB_EDGE) -> str:
    """Encode a bounded, oriented JPEG thumbnail for Weaviate's CLIP blob.

    Inputs: retained image path and long-edge cap. Output: base64 JPEG. Side effects: image
    decode only. Pick this when CLIP needs the thumbnail, including for HEIC/HEIF originals.
    """
    return base64.b64encode(jpeg_derivative(path, edge=edge, quality=85)).decode()


def _vector_schema(slot: str) -> pa.Schema:
    embedding = pa.list_(pa.float32()) if slot == SLOT_SINGLE else pa.list_(pa.list_(pa.float32()))
    return pa.schema(
        [
            ("identity", pa.string()),
            ("slot", pa.string()),
            ("model", pa.string()),
            ("status", pa.string()),
            ("embedding", embedding),
            ("embedded_at", pa.timestamp("us", tz="UTC")),
        ]
    )


EmbedFn = Callable[[bytes, str], Awaitable[Any]]


async def run_embed(
    output_dir: Path,
    spool_dir: Path | None,
    sl: Slice,
    slot: str,
    embed: EmbedFn,
    model: str,
    settings: ImageStageSettings,
    *,
    beat: Beat | None = None,
) -> dict[str, Any]:
    """Embed every image of the slice that ``slot`` covers, for ONE slot.

    Slots: ``image_single`` (one vector per image: NIM or Google), ``image_maxsim`` (Jina v4
    multi-vector bag) and
    ``image_colqwen`` (ColQwen2.5 bag). ``embed(bytes, extension)`` is the slot's embedder call;
    ``model`` its name.
    Coverage comes from ``settings.slot_applies(slot, kind)`` (policy and photo tier) and needs the
    slice's facts for
    the kinds. Side effects: one hosted request per covered image, at most ``settings.inflight`` at
    once (the embedders
    add their own gate and retries), and ``vectors/<slot>/<slice>.parquet``. A provider failure for
    one image is a row
    with status ``failed:<ErrorName>`` and no vector; publish ledgers the image ``partial`` or
    ``failed``. Receipt carries the
    slot's licence basis."""
    if slot not in ALL_SLOTS:
        raise ValueError(f"unknown image slot {slot!r}; known: {ALL_SLOTS}")
    if slot not in settings.slots:
        raise ValueError(
            f"image slot {slot!r} is not enabled (INTAKE_IMAGES_SLOTS={','.join(settings.slots)})"
        )
    out = table_path(output_dir, f"vectors/{slot}", sl.slice_id)
    facts = _read_rows(table_path(output_dir, "facts", sl.slice_id))
    if not facts:
        raise ValueError("embedding needs the slice's facts (kinds); run image_facts first")
    wanted = [
        i
        for i in sl.items
        if settings.slot_applies(slot, facts.get(i.identity, {}).get("kind", "photo"))
    ]
    if out.is_file():
        rows = list(_read_rows(out).values())
    else:
        gate = asyncio.Semaphore(max(settings.inflight, 1))
        done = 0

        async def one(item: ImageItem) -> dict[str, Any]:
            nonlocal done
            async with gate:
                row = {
                    "identity": item.identity,
                    "slot": slot,
                    "model": model,
                    "status": "ok",
                    "embedding": None,
                    "embedded_at": datetime.now(UTC),
                }
                try:
                    data, ext = await asyncio.to_thread(
                        prepare_for_embedding, item_path(output_dir, spool_dir, sl.slice_id, item)
                    )
                    row["embedding"] = await embed(data, ext)
                except Exception as exc:  # noqa: BLE001
                    row["status"] = f"failed:{type(exc).__name__}"
                done += 1
                if beat is not None and done % 10 == 0:
                    beat(f"embed {slot} {done}/{len(wanted)}")
                return row

        rows = list(await asyncio.gather(*(one(i) for i in wanted)))
        _write_table(out, pa.Table.from_pylist(rows, schema=_vector_schema(slot)))
    ok = sum(1 for r in rows if r["status"] == "ok")
    return {
        "slot": slot,
        "slice_id": sl.slice_id,
        "model": model,
        "selected": len(rows),
        "embedded_ok": ok,
        "embedded_failed": len(rows) - ok,
        "skipped_by_policy": len(sl.items) - len(wanted),
        "photo_tier": settings.photo_tier,
        "licence_basis": SLOT_LICENCE[slot],
    }


# -------------------------------------------------- publish

LEDGER_SCHEMA = pa.schema(
    [
        ("identity", pa.string()),
        ("status", pa.string()),
        ("reason", pa.string()),
        ("object_id", pa.string()),
        ("slice_id", pa.string()),
        ("run_id", pa.string()),
        ("published_at", pa.timestamp("us", tz="UTC")),
        ("screenshot_classifier_version", pa.string()),
        ("screenshot_classifier_status", pa.string()),
    ]
)


def image_object_id(identity: str) -> str:
    """Content-addressed Weaviate id: the same image (or PDF page) found again is the same
    object."""
    return str(uuid5(NAMESPACE_URL, f"intake-image-content:{identity}"))


def build_properties(
    item: ImageItem,
    facts: dict[str, Any],
    ocr: dict[str, Any] | None,
    *,
    source_id: str,
    models: dict[str, str],
) -> dict[str, str | bool | int]:
    """Build image properties with separate retained-content SHA256 and catalog-source SHA1.

    Inputs: slice item, facts, OCR and model names. Output: Weaviate properties; no side effects.
    ``content_sha256`` hashes the retained image, or rendered PNG for a PDF page.
    ``source_content_sha1`` preserves the catalog SHA1 of the original bucket object, including
    the PDF original for every page. Provider, bucket and vault_key locate that original.
    Pick this for catalog images; the directory lane builds its properties in image_pipeline.
    """
    kind = facts["kind"]
    return {
        "source_id": source_id,
        "source_path": item.key,
        "filename": item.name,
        "content_sha256": facts["content_sha256"],
        "source_content_sha1": item.sha1,
        "original_time": facts["original_time"],
        "original_time_source": facts["original_time_source"],
        "original_time_confidence": facts["original_time_confidence"],
        "original_time_conflict": facts["original_time_conflict"],
        "device": facts["device"],
        "software": facts["software"],
        "gps": facts["gps"],
        "is_screenshot": kind == KIND_SCREENSHOT,
        "screenshot_decision": facts.get("screenshot_decision", "unknown"),
        "screenshot_app": facts.get("screenshot_app", "unknown"),
        "screenshot_app_confidence": facts.get("screenshot_app_confidence", 0),
        "screenshot_app_basis": facts.get("screenshot_app_basis", ""),
        "screenshot_classifier_version": facts.get("screenshot_classifier_version", ""),
        "screenshot_classifier_provider": facts.get("screenshot_classifier_provider", ""),
        "screenshot_classifier_status": facts.get("screenshot_classifier_status", ""),
        "ocr_text": (ocr or {}).get("ocr_text", ""),
        "embed_single_model": models.get(SLOT_SINGLE, ""),
        "embed_multi_model": models.get(SLOT_MAXSIM, ""),
        "embed_slots": ",".join(f"{slot}={model}" for slot, model in sorted(models.items())),
        "notes": facts["notes"],
        "provider": item.provider,
        "bucket": item.bucket,
        "vault_key": item.key,
        "image_kind": kind,
        "image_kind_basis": facts["kind_basis"],
        "source_kind": item.source_kind,
        "ocr_engine": (ocr or {}).get("ocr_engine", ""),
        "identity": item.identity,
        "width": facts["width"],
        "height": facts["height"],
        "page": item.page,
        "page_count": item.page_count,
        "occurrences": item.occurrences,
    }


async def run_publish(
    output_dir: Path,
    spool_dir: Path | None,
    sl: Slice,
    writer: Any,
    settings: ImageStageSettings,
    *,
    source_id: str,
    run_id: str,
    release: bool = True,
    beat: Beat | None = None,
) -> dict[str, Any]:
    """Write the slice's images to Weaviate and record each in the ledger; then release the slice's
    spool folder.

    Inputs: the slice, a writer (``image_target.ImageObjectWriter``), the settings, source id and
    run id. Output:
    counts. Side effects: one upsert per image that has facts and at least one vector (idempotent:
    the id is
    content-addressed), the ledger ``published/<slice>.parquet``, and moving retained files from
    ``<spool>/images/<slice>/`` into the owning output folder's ``to_be_deleted/image-spool/``.
    Counts include quarantine files and bytes. Bucket originals remain available.
    Ledger status: ``ok`` (every covered slot has
    its vector),
    ``partial`` (written, but a covered slot failed: selected again by ``retry_failed``) or
    ``failed`` (nothing written;
    also not retried unless asked, so a bad file does not loop). With ``clip`` a 224-px thumbnail
    blob is attached for
    Weaviate's CLIP module to vectorize, and a photo in the ``clip_first`` tier needs nothing
    else."""
    ledger_path = table_path(output_dir, "published", sl.slice_id)
    patched = 0
    if ledger_path.is_file():
        rows = list(_read_rows(ledger_path).values())
        facts = _read_rows(table_path(output_dir, "facts", sl.slice_id))
        for row in rows:
            f = facts.get(row["identity"])
            if not row.get("object_id") or f is None:
                continue
            version = f.get("screenshot_classifier_version", "")
            if not version or (row.get("screenshot_classifier_version") == version and row.get("screenshot_classifier_status") == f.get("screenshot_classifier_status")):
                continue
            if not hasattr(writer, "patch_classification"):
                continue
            kind = f["kind"]
            await writer.patch_classification(row["object_id"], {
                "is_screenshot": kind == KIND_SCREENSHOT,
                "image_kind": kind,
                "image_kind_basis": f["kind_basis"],
                "screenshot_decision": f.get("screenshot_decision", "unknown"),
                "screenshot_app": f.get("screenshot_app", "unknown"),
                "screenshot_app_confidence": f.get("screenshot_app_confidence", 0),
                "screenshot_app_basis": f.get("screenshot_app_basis", ""),
                "screenshot_classifier_version": version,
                "screenshot_classifier_provider": f.get("screenshot_classifier_provider", ""),
                "screenshot_classifier_status": f.get("screenshot_classifier_status", ""),
            })
            row["screenshot_classifier_version"] = version
            row["screenshot_classifier_status"] = f.get("screenshot_classifier_status", "")
            patched += 1
        if patched:
            _write_table(ledger_path, pa.Table.from_pylist(rows, schema=LEDGER_SCHEMA))
    else:
        facts = _read_rows(table_path(output_dir, "facts", sl.slice_id))
        ocr = _read_rows(table_path(output_dir, "ocr", sl.slice_id))
        tables = {
            slot: _read_rows(table_path(output_dir, f"vectors/{slot}", sl.slice_id))
            for slot in settings.slots
        }
        now = datetime.now(UTC)
        rows = []
        pdf_sources: dict[str, str] = {i.identity.split("#", 1)[0]: "" for i in sl.items if i.page}
        for n, item in enumerate(sl.items):
            object_id = image_object_id(item.identity)
            f = facts.get(item.identity)
            if f is None:
                rows.append(_ledger(item.identity, "failed", "no facts", "", sl, run_id, now))
                continue
            good: dict[str, Any] = {}
            missing: list[str] = []
            for slot in settings.slots:
                if not settings.slot_applies(slot, f["kind"]):
                    continue
                row = tables[slot].get(item.identity)
                if row is not None and row["status"] == "ok":
                    good[slot] = row
                else:
                    missing.append(f"{slot}:{(row or {}).get('status', 'no row')}")
            clip_here = settings.clip
            if not good and not clip_here:
                rows.append(
                    _ledger(
                        item.identity,
                        "failed",
                        "; ".join(missing) or "no slot covers this image",
                        "",
                        sl,
                        run_id,
                        now,
                    )
                )
                continue
            props = build_properties(
                item,
                f,
                ocr.get(item.identity),
                source_id=source_id,
                models={slot: r["model"] for slot, r in good.items()},
            )
            if clip_here:
                props[CLIP_BLOB] = await asyncio.to_thread(
                    thumbnail_b64, item_path(output_dir, spool_dir, sl.slice_id, item)
                )
            extra = (
                {SLOT_COLQWEN: [list(v) for v in good[SLOT_COLQWEN]["embedding"]]}
                if SLOT_COLQWEN in good
                else None
            )
            if extra:
                extra = {COLQWEN_VECTOR: extra[SLOT_COLQWEN]}
            spec = (
                ImageObjectSpec(
                    props,
                    list(good[SLOT_SINGLE]["embedding"]) if SLOT_SINGLE in good else None,
                    [list(v) for v in good[SLOT_MAXSIM]["embedding"]]
                    if SLOT_MAXSIM in good
                    else None,
                    extra,
                )
                if good
                else ImageObjectSpec(props, None, None, None)
            )
            await writer.apply(
                ImageObjectAction((writer.url, writer.collection, str(UUID(object_id))), spec)
            )
            rows.append(
                _ledger(
                    item.identity,
                    "partial" if missing else "ok",
                    "; ".join(missing),
                    object_id,
                    sl,
                    run_id,
                    now,
                )
            )
            rows[-1]["screenshot_classifier_version"] = f.get("screenshot_classifier_version", "")
            rows[-1]["screenshot_classifier_status"] = f.get("screenshot_classifier_status", "")
            if item.page:
                pdf_sources[item.identity.split("#", 1)[0]] = object_id
            if beat is not None and n % 10 == 0:
                beat(f"published {n + 1}/{len(sl.items)}")
        for failure in sl.failed:
            rows.append(
                _ledger(
                    failure["identity"],
                    "failed",
                    "fetch: " + failure["reason"],
                    "",
                    sl,
                    run_id,
                    now,
                )
            )
        # Parent completion means every page has a published object, including earlier slices.
        for pdf_identity, object_id in pdf_sources.items():
            pdf_items = [
                i for i in sl.items if i.page and i.identity.split("#", 1)[0] == pdf_identity
            ]
            published = published_pdf_pages(output_dir, pdf_identity)
            published.update(
                i.page
                for i in pdf_items
                if any(r["identity"] == i.identity and r["object_id"] for r in rows)
            )
            complete = set(range(1, pdf_items[0].page_count + 1)).issubset(published)
            failed = any(
                r["identity"].startswith(pdf_identity + "#p") and not r["object_id"] for r in rows
            )
            status = "ok" if complete else "failed" if failed else "partial"
            reason = "pdf covered" if complete else "pdf failed" if failed else "pdf partial"
            rows.append(_ledger(pdf_identity, status, reason, object_id, sl, run_id, now))
        _write_table(ledger_path, pa.Table.from_pylist(rows, schema=LEDGER_SCHEMA))
    retained = {"files": 0, "bytes": 0}
    if release:
        retained = quarantine_spool(
            output_dir, spool_dir, spool_root(output_dir, spool_dir) / sl.slice_id
        )
    image_rows = [r for r in rows if not r["reason"].startswith("pdf ")]
    written = [r for r in image_rows if r["object_id"]]
    return {
        "slice_id": sl.slice_id,
        "published": len(written),
        "partial": sum(1 for r in image_rows if r["status"] == "partial"),
        "failed": sum(1 for r in image_rows if r["status"] == "failed"),
        "released": release,
        "quarantined_files": retained["files"],
        "quarantined_bytes": retained["bytes"],
        "collection": getattr(writer, "collection", ""),
        "classification_patched": patched,
    }


def _ledger(
    identity: str, status: str, reason: str, object_id: str, sl: Slice, run_id: str, now: datetime
) -> dict[str, Any]:
    return {
        "identity": identity,
        "status": status,
        "reason": reason,
        "object_id": object_id,
        "slice_id": sl.slice_id,
        "run_id": run_id,
        "published_at": now,
        "screenshot_classifier_version": "",
        "screenshot_classifier_status": "",
    }


def lake_counts(output_dir: Path) -> dict[str, int]:
    """Images ledgered ``ok`` and ``failed`` so far (a cheap status for receipts)."""
    folder = lake(output_dir) / "published"
    counts = {"ok": 0, "failed": 0}
    for path in folder.glob("*.parquet") if folder.is_dir() else []:
        for row in pq.read_table(path, columns=["status", "object_id"]).to_pylist():
            if row["status"] == "ok" and row["object_id"]:
                counts["ok"] += 1
            elif row["status"] == "failed":
                counts["failed"] += 1
    return counts


__all__ = ["ImageStageSettings", "load_slice", "run_embed", "run_facts", "run_ocr", "run_publish"]
