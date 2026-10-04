"""The cycle's stages as plain async functions: bounded inputs, small outputs, one job each, no
hidden orchestration.

> Byline: Claude Code · Sonnet 5.5 · 2026-10-02

These are the bodies of the Temporal Activities (temporal_worker.py wraps each 1:1) and of the
``casebible-corpus
cycle`` command (the direct-call shape: the same units in sequence on a developer machine). Nothing
here sequences
stages, retries, or decides what runs next: that is the workflow's job (modules/engine/superindex in
the Go engine).

Every stage takes ``(cycle_id, params, beat)`` and returns a JSON-able dict that is also written as
an immutable
receipt under ``cycles/<cycle_id>/`` (ledger.py). References in, counts out: no stage returns
document text.

    discover   catalog watermark + classification of every object            (reads the catalog
    only)
    extract    CocoIndex incremental pass: extract + chunk + document rows    (subprocess; reads B2
    for new/changed)
    summarize  separate summary pass over documents the index picks           (NIM chat; reads
    Parquet only)
    embed      one vector slot for every distinct chunk text without one     (embedding API;
    reads/writes Parquet)
    publish    chunks + vectors -> Weaviate; follow moved files              (Weaviate; reads
    Parquet)
    graph      the active snapshot -> surreal-intake file graph, at most daily  (Surreal; reads
    Parquet)
    commit     advance the catalog watermark                                 (writes one receipt)

The image stage (``INTAKE_IMAGE_STAGE=on``; image_slice.py and image_stage.py) is five more units,
run as a loop of
slices after the text passes: ``image_fetch`` (a bounded slice from the bucket to the spool),
``image_facts`` (time,
device, screenshot/photo/scan), ``image_ocr`` (text in the picture), ``image_embed`` (one vector
slot) and
``image_publish`` (Weaviate ``IntakeImageV1``, then release the spool). They extend Intake's image
index.

Extract and chunk are ONE pass on purpose: the extracted text streams into the chunker and is never
materialized
(that is what keeps a 1.3 GB export bounded), so they cannot be two units without writing the whole
text to disk.
"""

from __future__ import annotations

import asyncio
import json
import os
import sys
import time
from collections.abc import Callable
from datetime import UTC, datetime
from typing import Any

import httpx

from . import discovery, ledger
from .cas_store import COLLECTION, ChunkCollection
from .config import Settings
from .embed_stage import DEFAULT_MAX_TEXTS, run_embed
from .embedders import SlotEmbedder, load_slots
from .image_slice import fetch_slice, load_slice
from .image_stage import ImageStageSettings
from .image_stage import run_embed as run_image_embed
from .image_stage import run_facts as run_image_facts
from .image_stage import run_ocr as run_image_ocr
from .image_stage import run_publish as run_image_publish
from .publish_stage import DEFAULT_MAX_OBJECTS, publish_pending, relocate_moved
from .run_status import latest_run_status
from .secrets import get_secret
from .summary_stage import policy_from_env, run_summary

Beat = Callable[[str], None]


def _noop(_: str) -> None:
    return None


def load_settings() -> Settings:
    settings = Settings.from_env().resolved()
    settings.validate(require_source=False)
    return settings


def _catalog_dsn() -> str:
    dsn = get_secret("INTAKE_CATALOG_DSN")
    if not dsn:
        raise ValueError("INTAKE_CATALOG_DSN is not configured")
    return dsn


def _weaviate_base() -> str:
    url = (os.getenv("INTAKE_WEAVIATE_URL") or "").strip()
    if not url.startswith(("http://", "https://")):
        raise ValueError("INTAKE_WEAVIATE_URL must be an absolute http(s) URL")
    return url.rstrip("/")


def chunk_collection_name() -> str:
    return os.getenv("INTAKE_CHUNK_COLLECTION", "").strip() or COLLECTION


# -------------------------------------------------- discover


async def stage_discover(
    cycle_id: str, params: dict[str, Any], beat: Beat = _noop
) -> dict[str, Any]:
    settings = load_settings()
    previous = ledger.last_commit(settings.output_dir)
    receipt = await asyncio.to_thread(
        discovery.discover,
        settings,
        _catalog_dsn(),
        cycle_id,
        (previous or {}).get("watermark"),
        force=bool(params.get("force")),
    )
    receipt["previous_commit"] = (previous or {}).get("cycle_id")
    ledger.write_stage_receipt(settings.output_dir, cycle_id, "discover", receipt)
    return receipt


# -------------------------------------------------- extract


def stage_environment(base: dict[str, str] | None = None) -> dict[str, str]:
    """The environment of the extract subprocess: embedding, summary and the CocoIndex Weaviate
    target are OFF.

    Those are separate stages now. The row is written ``pending`` and a later stage fills the
    vector."""
    env = dict(os.environ if base is None else base)
    env.update(
        {
            "INTAKE_EMBED_MODE": "deferred",
            "INTAKE_SUMMARY_MODE": "off",
            "INTAKE_WEAVIATE_INDEX_ENABLED": "0",
        }
    )
    return env


def _rss_mb(pid: int) -> float | None:
    try:
        with open(f"/proc/{pid}/status", encoding="utf-8") as handle:
            for line in handle:
                if line.startswith("VmRSS:"):
                    return int(line.split()[1]) / 1024
    except OSError:
        return None
    return None


async def stage_extract(
    cycle_id: str, params: dict[str, Any], beat: Beat = _noop
) -> dict[str, Any]:
    """Run ``casebible-corpus index`` as a child process; heartbeat from its run status; kill it on
    cancel or when it
    outgrows ``INTAKE_EXTRACT_RSS_LIMIT_MB`` (a clean failure beats the container OOM killer)."""
    settings = load_settings()
    started = time.time()
    # The file graph is its own stage (graph_stage.py): projecting it per slice would rewrite the
    # corpus each time.
    command = [
        sys.executable,
        "-c",
        "from casebible_index.cli import app; app()",
        "index",
        "--no-project-graph",
    ]
    if params.get("limit"):
        command += ["--limit", str(int(params["limit"]))]
    if params.get("path_prefix"):
        command += ["--path-prefix", str(params["path_prefix"])]
    rss_limit = float(os.getenv("INTAKE_EXTRACT_RSS_LIMIT_MB", "5000"))
    process = await asyncio.create_subprocess_exec(
        *command,
        env=stage_environment(),
        stdout=asyncio.subprocess.DEVNULL,
        stderr=asyncio.subprocess.PIPE,
    )
    peak = 0.0
    killed = ""
    budget = float(
        params.get("time_budget_s") or os.getenv("INTAKE_EXTRACT_TIME_BUDGET_S", "0") or 0
    )
    stopped_for_budget = False
    try:
        while True:
            try:
                await asyncio.wait_for(process.wait(), timeout=20.0)
                break
            except TimeoutError:
                status = await asyncio.to_thread(latest_run_status, settings.output_dir)
                rss = _rss_mb(process.pid) or 0.0
                peak = max(peak, rss)
                beat(
                    f"extract: transformed {status.get('files_transformed')} "
                    f"of {status.get('files_observed')} "
                    f"failures {status.get('failure_events')} rss {rss:.0f} MB"
                )
                if budget and time.time() - started > budget:
                    # Slice the first full run: stop here, publish what exists, and run extract
                    # again. Finished
                    # objects keep their memo state and document rows, so the next slice resumes,
                    # not restarts.
                    stopped_for_budget = True
                    process.terminate()
                    try:
                        await asyncio.wait_for(process.wait(), timeout=120.0)
                    except TimeoutError:
                        process.kill()
                        await process.wait()
                    break
                if rss_limit and rss > rss_limit:
                    killed = (
                        "extract subprocess exceeded "
                        f"INTAKE_EXTRACT_RSS_LIMIT_MB={rss_limit:.0f} ({rss:.0f} MB)"
                    )
                    process.kill()
                    await process.wait()
                    break
    except asyncio.CancelledError:
        process.kill()
        await process.wait()
        raise
    snapshot_rebuilt = ""
    if stopped_for_budget:
        # The index command builds the snapshot only after a clean finish; build it here so embed
        # and publish see
        # every document the slice completed.
        from .snapshots import build_active_snapshot

        snapshot_rebuilt = str(await asyncio.to_thread(build_active_snapshot, settings))
    stderr = (
        (await process.stderr.read()).decode("utf-8", errors="replace")[-600:]
        if process.stderr
        else ""
    )
    receipts = sorted(
        (settings.output_dir / "receipts").glob("*-index-*.json"), key=lambda p: p.stat().st_mtime
    )
    fresh = [p for p in receipts if p.stat().st_mtime >= started - 1]
    detail: dict[str, Any] = json.loads(fresh[-1].read_text(encoding="utf-8")) if fresh else {}
    result = {
        "exit_code": process.returncode,
        "duration_s": round(time.time() - started, 1),
        "peak_rss_mb": round(peak),
        "killed": killed,
        "more": stopped_for_budget,
        "stopped_for_time_budget": stopped_for_budget,
        "run_status": detail.get("run_status", {}) or latest_run_status(settings.output_dir),
        "snapshot": detail.get("snapshot", "") or snapshot_rebuilt,
        "b2_requests": detail.get("b2_requests"),
        "b2_bytes_read": detail.get("b2_bytes_read"),
        "b2_objects_touched": detail.get("b2_objects_touched"),
        "graph": detail.get("graph", {}),
        "index_receipt": str(fresh[-1]) if fresh else "",
    }
    ledger.write_stage_receipt(
        settings.output_dir,
        cycle_id,
        "extract",
        {**result, "stderr_tail": stderr if process.returncode else ""},
    )
    if killed or (process.returncode and not stopped_for_budget):
        raise RuntimeError(
            killed or f"extract subprocess exited {process.returncode}: {stderr[-300:]}"
        )
    return result


# -------------------------------------------------- summarize


async def stage_summarize(
    cycle_id: str, params: dict[str, Any], beat: Beat = _noop
) -> dict[str, Any]:
    from .nim import NimClient

    settings = load_settings()
    if os.getenv("INTAKE_SUMMARY_STAGE", "on").strip().casefold() != "on":
        result = {
            "skipped": "INTAKE_SUMMARY_STAGE is off",
            "selected": 0,
            "summarized": 0,
            "failed": 0,
            "more": False,
        }
    else:
        key = get_secret("NVIDIA_API_KEY")
        if not key:
            raise ValueError("NVIDIA_API_KEY is not configured")
        policy = policy_from_env()
        if params.get("max_docs"):
            policy["max_docs"] = int(params["max_docs"])
        async with NimClient(
            api_key=key,
            base_url=settings.nim_base_url,
            embed_model=settings.embed_model,
            summary_model=settings.summary_model,
            dimensions=settings.embed_dimensions,
            timeout_seconds=settings.timeout_seconds,
            max_retries=settings.max_retries,
            max_concurrency=settings.max_concurrency,
        ) as client:
            result = await run_summary(
                settings.output_dir,
                client.summarize,
                settings.summary_model,
                max_chars=settings.summary_max_chars,
                policy=policy,
                beat=beat,
            )
    ledger.write_stage_receipt(settings.output_dir, cycle_id, "summarize", result)
    return result


# -------------------------------------------------- embed


async def stage_embed(cycle_id: str, params: dict[str, Any], beat: Beat = _noop) -> dict[str, Any]:
    """Embed ONE slot (``params['slot']``); the workflow runs one activity per slot, in parallel."""
    settings = load_settings()
    slots = {s.name: s for s in load_slots()}
    slot = slots.get(str(params.get("slot") or "text_nim"))
    if slot is None:
        raise ValueError(f"slot {params.get('slot')!r} is not enabled (INTAKE_VECTOR_SLOTS)")
    async with httpx.AsyncClient(follow_redirects=False) as client:
        embedder = SlotEmbedder.for_slot(slot, client)
        result = await run_embed(
            settings.output_dir,
            embedder,
            max_texts=int(params.get("max_texts") or DEFAULT_MAX_TEXTS),
            beat=beat,
        )
    ledger.write_stage_receipt(settings.output_dir, cycle_id, f"embed-{slot.name}", result)
    return result


# -------------------------------------------------- publish


async def stage_publish(
    cycle_id: str, params: dict[str, Any], beat: Beat = _noop
) -> dict[str, Any]:
    settings = load_settings()
    slots = load_slots()
    headers = {}
    weaviate_key = get_secret("INTAKE_WEAVIATE_API_KEY")
    if weaviate_key:
        headers["Authorization"] = f"Bearer {weaviate_key}"
    async with httpx.AsyncClient(headers=headers, timeout=120.0, follow_redirects=False) as client:
        store = ChunkCollection(_weaviate_base(), chunk_collection_name(), client)
        limit = int(params.get("max_objects") or DEFAULT_MAX_OBJECTS)
        published = await publish_pending(
            settings.output_dir, store, slots, max_objects=limit, run_id=cycle_id, beat=beat
        )
        relocated = await relocate_moved(
            settings.output_dir, store, max_objects=limit, run_id=cycle_id
        )
        try:
            collection_count = await store.count()
        except Exception:  # noqa: BLE001 - a count failure must not fail a publish that succeeded
            collection_count = -1
    result = {
        **published,
        "relocated": relocated,
        "collection": chunk_collection_name(),
        "collection_count": collection_count,
        "slots": [s.name for s in slots],
    }
    ledger.write_stage_receipt(settings.output_dir, cycle_id, "publish", result)
    return result


# -------------------------------------------------- graph


async def stage_graph(cycle_id: str, params: dict[str, Any], beat: Beat = _noop) -> dict[str, Any]:
    """Project the active snapshot into surreal-intake when it changed and the interval has
    passed."""
    from .graph_stage import run_graph_stage

    settings = load_settings()
    beat("graph: checking whether a projection is due")
    result = await run_graph_stage(settings, force=bool(params.get("force")))
    ledger.write_stage_receipt(settings.output_dir, cycle_id, "graph", result)
    return result


# -------------------------------------------------- commit


async def stage_commit(cycle_id: str, params: dict[str, Any], beat: Beat = _noop) -> dict[str, Any]:
    """Advance the watermark. ``params['watermark']`` is the discovery stage's watermark, passed by
    reference."""
    settings = load_settings()
    path = ledger.commit_cycle(
        settings.output_dir,
        cycle_id,
        list(params.get("watermark") or []),
        {"committed_at": datetime.now(UTC).isoformat(), **(params.get("summary") or {})},
    )
    return {
        "committed": True,
        "receipt": str(path),
        "watermark_buckets": len(params.get("watermark") or []),
    }


# -------------------------------------------------- images


def _image_settings() -> ImageStageSettings:
    settings = ImageStageSettings.from_env()
    settings.validate()
    return settings


def _image_off() -> dict[str, Any]:
    return {"skipped": "INTAKE_IMAGE_STAGE is off", "slice_id": "", "count": 0, "more": False}


def _no_slice(stage: str) -> dict[str, Any]:
    return {
        "skipped": f"{stage}: no slice id (image_fetch found nothing to do)",
        "images": 0,
        "more": False,
    }


async def stage_image_fetch(
    cycle_id: str, params: dict[str, Any], beat: Beat = _noop
) -> dict[str, Any]:
    """Select the next slice of images (or scanned PDF pages) and stream them from the bucket to the
    spool."""
    from .object_store import ObjectStore, configured_credentials

    image = _image_settings()
    if not image.enabled:
        return _image_off()
    settings = load_settings()
    stores: dict[tuple[str, str], ObjectStore] = {}
    async with httpx.AsyncClient(timeout=120.0, follow_redirects=False) as client:

        def store_for(provider: str, bucket: str) -> ObjectStore:
            if (provider, bucket) not in stores:
                stores[(provider, bucket)] = ObjectStore(
                    configured_credentials(provider), bucket, client
                )
            return stores[(provider, bucket)]

        result = await fetch_slice(
            settings.output_dir,
            settings.spool_dir,
            store_for,
            max_files=min(image.max_files, int(params.get("max_files") or image.max_files)),
            max_bytes=image.max_bytes,
            max_file_bytes=image.max_file_bytes,
            pdf_max_pages=image.pdf_max_pages,
            pdf_dpi=image.pdf_dpi,
            retry_failed=bool(params.get("retry_failed")),
            path_prefix=str(params.get("path_prefix") or ""),
            locators=params.get("locators"),
            max_items=int(params.get("max_items") or 0),
            beat=beat,
        )
    result["slots"] = list(image.slots)
    ledger.write_stage_receipt(settings.output_dir, cycle_id, "image-fetch", result)
    return result


async def stage_image_facts(
    cycle_id: str, params: dict[str, Any], beat: Beat = _noop
) -> dict[str, Any]:
    """Original time, device, GPS, size and screenshot/photo/scan for the slice's images."""
    if not _image_settings().enabled or not params.get("slice_id"):
        return _no_slice("image_facts")
    settings = load_settings()
    result = await run_image_facts(
        settings.output_dir,
        settings.spool_dir,
        load_slice(settings.output_dir, params["slice_id"]),
        beat=beat,
    )
    ledger.write_stage_receipt(settings.output_dir, cycle_id, "image-facts", result)
    return result


async def stage_image_ocr(
    cycle_id: str, params: dict[str, Any], beat: Beat = _noop
) -> dict[str, Any]:
    """The text in the slice's pictures, with the engine named by ``INTAKE_IMAGES_OCR_ENGINE``."""
    image = _image_settings()
    if not image.enabled or not params.get("slice_id"):
        return _no_slice("image_ocr")
    settings = load_settings()
    result = await run_image_ocr(
        settings.output_dir,
        settings.spool_dir,
        load_slice(settings.output_dir, params["slice_id"]),
        engine=str(params.get("engine") or image.ocr_engine),
        max_chars=image.max_ocr_chars,
        beat=beat,
    )
    ledger.write_stage_receipt(settings.output_dir, cycle_id, "image-ocr", result)
    return result


async def stage_image_embed(
    cycle_id: str, params: dict[str, Any], beat: Beat = _noop
) -> dict[str, Any]:
    """Embed ONE image slot (``params['slot']``: image_maxsim, image_single or image_colqwen) for
    the slice."""
    from .image_embedders import (
        GOOGLE_MODEL,
        JINA_MODEL,
        NIM_MODEL,
        ColQwenEmbedder,
        ImageEmbedders,
    )

    image = _image_settings()
    if not image.enabled or not params.get("slice_id"):
        return _no_slice("image_embed")
    slot = str(params.get("slot") or image.slots[0])
    if slot not in image.slots:
        raise ValueError(
            f"image slot {slot!r} is not enabled (INTAKE_IMAGES_SLOTS={','.join(image.slots)})"
        )
    settings = load_settings()
    async with httpx.AsyncClient(timeout=180.0, follow_redirects=False) as client:
        if slot == "image_colqwen":
            colqwen = ColQwenEmbedder(
                client, image.colqwen_url, get_secret("COLQWEN_API_TOKEN") or ""
            )
            embed, model = colqwen.embed_multi, colqwen.model
        elif slot == "image_single":
            key_name = "NVIDIA_API_KEY" if image.single_provider == "nim" else "GOOGLE_API_KEY"
            key = get_secret(key_name)
            if not key:
                raise ValueError(f"{key_name} is not configured")
            embedders = ImageEmbedders(client, image.single_provider, key, None)
            embed, model = (
                embedders.embed_single,
                NIM_MODEL if image.single_provider == "nim" else GOOGLE_MODEL,
            )
        else:
            jina = get_secret("JINA_API_KEY")
            if not jina:
                raise ValueError("JINA_API_KEY is not configured")
            embedders = ImageEmbedders(client, image.single_provider, "unused", jina)

            async def embed(data: bytes, extension: str) -> Any:
                return await embedders.embed_multi(data)

            model = JINA_MODEL
        result = await run_image_embed(
            settings.output_dir,
            settings.spool_dir,
            load_slice(settings.output_dir, params["slice_id"]),
            slot,
            embed,
            model,
            image,
            beat=beat,
        )
    ledger.write_stage_receipt(settings.output_dir, cycle_id, f"image-embed-{slot}", result)
    return result


async def stage_image_publish(
    cycle_id: str, params: dict[str, Any], beat: Beat = _noop
) -> dict[str, Any]:
    """Write the slice to Weaviate ``IntakeImageV1`` (schema added if needed), ledger it, release
    the spool."""
    from .image_embedders import SINGLE_DIMENSIONS
    from .image_target import ImageObjectWriter

    image = _image_settings()
    if not image.enabled or not params.get("slice_id"):
        return _no_slice("image_publish")
    if not image.weaviate_url.startswith(("http://", "https://")):
        raise ValueError(
            "INTAKE_IMAGES_WEAVIATE_URL (or INTAKE_WEAVIATE_URL) must be an absolute http(s) URL"
        )
    settings = load_settings()
    headers = {}
    weaviate_key = get_secret("INTAKE_WEAVIATE_API_KEY")
    if weaviate_key:
        headers["Authorization"] = f"Bearer {weaviate_key}"
    async with httpx.AsyncClient(headers=headers, timeout=120.0, follow_redirects=False) as client:
        writer = ImageObjectWriter(
            image.weaviate_url, image.collection, SINGLE_DIMENSIONS[image.single_provider], client
        )
        added = await writer.ensure_schema(clip=image.clip, colqwen="image_colqwen" in image.slots)
        await writer.verify_schema()
        result = await run_image_publish(
            settings.output_dir,
            settings.spool_dir,
            load_slice(settings.output_dir, params["slice_id"]),
            writer,
            image,
            source_id=settings.source_id,
            run_id=cycle_id,
            beat=beat,
        )
    result["schema_properties_added"] = len(added)
    ledger.write_stage_receipt(settings.output_dir, cycle_id, "image-publish", result)
    return result


STAGES = {
    "discover": stage_discover,
    "extract": stage_extract,
    "summarize": stage_summarize,
    "embed": stage_embed,
    "publish": stage_publish,
    "graph": stage_graph,
    "commit": stage_commit,
    "image_fetch": stage_image_fetch,
    "image_facts": stage_image_facts,
    "image_ocr": stage_image_ocr,
    "image_embed": stage_image_embed,
    "image_publish": stage_image_publish,
}


async def run_cycle_direct(
    params: dict[str, Any] | None = None, beat: Beat = _noop
) -> dict[str, Any]:
    """The direct-call shape: every stage in order in this process. For a developer machine and the
    live proof;
    production runs the same units as Temporal Activities under the Go workflow."""
    params = params or {}
    cycle_id = ledger.new_cycle_id()
    out: dict[str, Any] = {"cycle_id": cycle_id}
    discovered = await stage_discover(cycle_id, params, beat)
    out["discover"] = discovered
    if not discovered["changed"]:
        return out
    out["extract"] = await stage_extract(cycle_id, params, beat)
    out["summarize"] = await stage_summarize(cycle_id, params, beat)
    out["embed"] = {}
    for slot in load_slots():
        while True:
            result = await stage_embed(cycle_id, {**params, "slot": slot.name}, beat)
            out["embed"][slot.name] = result
            if not result["more"]:
                break
    while True:
        published = await stage_publish(cycle_id, params, beat)
        out["publish"] = published
        if not published["more"]:
            break
    out["images"] = []
    if _image_settings().enabled:
        for _ in range(int(params.get("max_image_slices") or 500)):
            fetched = await stage_image_fetch(cycle_id, params, beat)
            if not fetched["slice_id"]:
                break
            chain = {**params, "slice_id": fetched["slice_id"]}
            step = {
                "fetch": fetched,
                "facts": await stage_image_facts(cycle_id, chain, beat),
                "ocr": await stage_image_ocr(cycle_id, chain, beat),
            }
            for slot in _image_settings().slots:
                step[f"embed_{slot}"] = await stage_image_embed(
                    cycle_id, {**chain, "slot": slot}, beat
                )
            step["publish"] = await stage_image_publish(cycle_id, chain, beat)
            out["images"].append(step)
            if not fetched["more"]:
                break
    out["graph"] = await stage_graph(cycle_id, params, beat)
    out["commit"] = await stage_commit(cycle_id, {"watermark": discovered["watermark"]}, beat)
    return out
