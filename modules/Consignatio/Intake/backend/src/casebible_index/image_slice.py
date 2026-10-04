"""The image stage's unit of work: a bounded SLICE of images, its selection, its spool and its
ledger. One job: fetch.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

The Super Index image stage extends Intake's image index (``image_pipeline.py``: facts, embedders,
Weaviate target,
search) with a catalog source. Instead of walking a directory it takes images from the represented
inventory that
discovery writes (``inventory/represented/<cycle>.parquet``, kind ``media``) and scanned PDFs from
the extract stage
(documents with status ``skipped_no_text`` joined to ``inventory/pdf/<cycle>.parquet``), a bounded
slice at a time:

    fetch    this module     select a slice, stream each object from the bucket into the spool (PDF
    -> page PNGs)
    facts    image_stage     original time, device, GPS, screenshot/photo/scan
    ocr      image_stage     text in the image (engine selectable)
    embed    image_stage     one vector slot (single vector, MaxSim bag), one Activity per slot
    publish  image_stage     facts + text + vectors -> Weaviate IntakeImageV1, release the spool

Every unit reads the slice manifest and writes one small Parquet keyed by ``identity`` under
``datasets/images/``, so a
retry of any unit repeats only that unit and the bucket is read once per image. Identity is content:
the catalog SHA-1
(``k-<md5>`` of the locator when the catalog has none), and ``<sha1>#p<n>`` for a PDF page. Exact
copies in several
places are ONE image object that records how many occurrences the catalog lists; every occurrence
stays in the catalog.

Memory: one object streams to disk in 4 MiB windows; a slice holds at most ``max_files`` files and
``max_bytes`` bytes
on the spool (default 40 files, 256 MiB). Nothing here holds image bytes in memory beyond a window.
"""

from __future__ import annotations

import json
import shutil
import uuid
from collections.abc import Callable
from dataclasses import asdict, dataclass, field
from datetime import UTC, datetime
from pathlib import Path, PurePosixPath
from typing import Any

import duckdb

from .parquet_store import write_json_immutable
from .pdf_pages import DEFAULT_DPI, DEFAULT_MAX_PAGES, render_pdf_pages

# Formats the embedders accept (image_embedders._MIME). HEIC/HEIF are inventoried by discovery but
# counted, not
# processed: Pillow has no HEIC codec here and nothing in the chain would read them.
PROCESSABLE_EXTENSIONS = (".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tif", ".tiff")
DEFAULT_MAX_FILES = 40
DEFAULT_MAX_BYTES = 256 * 1024 * 1024
DEFAULT_MAX_FILE_BYTES = 25 * 1024 * 1024
WINDOW_BYTES = 4 * 1024 * 1024

Beat = Callable[[str], None]
StoreFor = Callable[[str, str], Any]  # (provider, bucket) -> ObjectStore


def lake(output_dir: Path) -> Path:
    return output_dir / "datasets" / "images"


def table_path(output_dir: Path, table: str, slice_id: str) -> Path:
    """Where one unit's result for a slice lives: ``datasets/images/<table>/<slice_id>.parquet``."""
    return lake(output_dir) / table / f"{slice_id}.parquet"


def manifest_path(output_dir: Path, slice_id: str) -> Path:
    return lake(output_dir) / "slices" / f"{slice_id}.json"


def spool_root(output_dir: Path, spool_dir: Path | None) -> Path:
    return (spool_dir if spool_dir else output_dir / "spool") / "images"


def new_slice_id() -> str:
    return datetime.now(UTC).strftime("%Y%m%dT%H%M%S") + "-" + uuid.uuid4().hex[:6]


@dataclass
class ImageItem:
    """One image of a slice. ``page`` is 0 for an image file and the 1-based page for a rendered PDF
    page."""

    identity: str
    sha1: str
    provider: str
    bucket: str
    key: str
    size: int
    occurrences: int
    file: str  # file name inside the slice's spool folder
    page: int = 0
    page_count: int = 0
    width: int = 0
    height: int = 0

    @property
    def source_kind(self) -> str:
        return "pdf_page" if self.page else "image"

    @property
    def name(self) -> str:
        return PurePosixPath(self.key).name


@dataclass
class Slice:
    slice_id: str
    kind: str  # "image" | "pdf"
    items: list[ImageItem] = field(default_factory=list)
    failed: list[dict[str, str]] = field(default_factory=list)
    skipped: dict[str, int] = field(default_factory=dict)

    def to_json(self) -> dict[str, Any]:
        return {
            "slice_id": self.slice_id,
            "kind": self.kind,
            "items": [asdict(i) for i in self.items],
            "failed": self.failed,
            "skipped": self.skipped,
        }

    @classmethod
    def from_json(cls, data: dict[str, Any]) -> Slice:
        return cls(
            data["slice_id"],
            data["kind"],
            [ImageItem(**i) for i in data["items"]],
            list(data.get("failed", [])),
            dict(data.get("skipped", {})),
        )


def load_slice(output_dir: Path, slice_id: str) -> Slice:
    path = manifest_path(output_dir, slice_id)
    if not path.is_file():
        raise ValueError(f"image slice {slice_id!r} has no manifest; run image_fetch first")
    return Slice.from_json(json.loads(path.read_text(encoding="utf-8")))


def open_slice_id(output_dir: Path) -> str | None:
    """The oldest slice whose manifest exists but whose publish ledger does not (a cycle that died
    mid-slice)."""
    folder = lake(output_dir) / "slices"
    if not folder.is_dir():
        return None
    for manifest in sorted(folder.glob("*.json")):
        if not table_path(output_dir, "published", manifest.stem).is_file():
            return manifest.stem
    return None


# -------------------------------------------------- selection


def _posix(path: Path) -> str:
    return path.resolve().as_posix().replace("'", "''")


def newest_inventory(output_dir: Path, folder: str) -> Path | None:
    files = sorted((output_dir / "inventory" / folder).glob("*.parquet"))
    return files[-1] if files else None


def _ledger_sql(output_dir: Path, retry_failed: bool) -> str:
    done = lake(output_dir) / "published"
    if not (done.is_dir() and any(done.glob("*.parquet"))):
        return "SELECT NULL::VARCHAR AS identity WHERE false"
    where = "WHERE status = 'ok'" if retry_failed else "WHERE reason <> 'pdf partial'"
    return (
        f"SELECT DISTINCT identity FROM read_parquet('{_posix(done / '*.parquet')}', "
        f"union_by_name = true) {where}"
    )


IDENTITY_SQL = (
    "CASE WHEN coalesce(sha1, '') <> '' THEN sha1 "
    "ELSE 'k-' || md5(provider || '|' || bucket || '|' || key || '|' || CAST(size AS VARCHAR)) END"
)


def _prefix_sql(path_prefix: str) -> str:
    if not path_prefix:
        return ""
    return " AND starts_with(key, '" + path_prefix.replace("'", "''") + "')"


def _locator_sql(locators: list[dict[str, str]] | None) -> str:
    """Restrict inventory rows to exact approved object locators.

    Inputs: optional provider/bucket/key dictionaries. Output: an escaped SQL predicate.
    Side effects: none. Pick this for a proof manifest instead of a broad path prefix.
    An explicitly empty manifest selects nothing.
    """
    if locators is None:
        return ""
    clauses = []
    for locator in locators:
        if set(locator) != {"provider", "bucket", "key"} or any(
            not isinstance(value, str) or not value for value in locator.values()
        ):
            raise ValueError("image locators require non-empty provider, bucket and key")
        clauses.append(
            "("
            + " AND ".join(
                name + " = '" + locator[name].replace("'", "''") + "'"
                for name in ("provider", "bucket", "key")
            )
            + ")"
        )
    return " AND (" + " OR ".join(clauses) + ")" if clauses else " AND false"


def quarantine_spool(output_dir: Path, spool_dir: Path | None, source: Path) -> dict[str, int]:
    """Move derived spool material into the owning output mount's quarantine.

    Inputs: output mount, configured spool and a child path. Output: retained file/byte counts.
    Side effects: moves material into output_dir/to_be_deleted/image-spool; never deletes.
    Pick this when releasing a slice or retaining an incomplete download for owner disposal.
    """
    root = spool_root(output_dir, spool_dir).resolve()
    resolved = source.resolve()
    if resolved == root or not resolved.is_relative_to(root):
        raise ValueError("quarantine source must be a child of the configured image spool")
    if not source.exists():
        return {"files": 0, "bytes": 0}
    files = [source] if source.is_file() else [p for p in source.rglob("*") if p.is_file()]
    counts = {"files": len(files), "bytes": sum(p.stat().st_size for p in files)}
    target = output_dir / "to_be_deleted" / "image-spool" / uuid.uuid4().hex
    target.mkdir(parents=True)
    shutil.move(str(source), str(target / source.name))
    return counts


def select_images(
    output_dir: Path,
    limit: int,
    max_file_bytes: int,
    *,
    retry_failed: bool = False,
    path_prefix: str = "",
    locators: list[dict[str, str]] | None = None,
) -> tuple[list[dict[str, Any]], bool, dict[str, int]]:
    """Up to ``limit`` distinct images not yet in the ledger, whether more remain, and counts of
    what was passed over.

    Reads the newest represented inventory only; groups exact copies (same identity) into one row
    with an occurrence
    count; skips objects above ``max_file_bytes`` and formats the chain cannot read (counted in the
    third value)."""
    inventory = newest_inventory(output_dir, "represented")
    if inventory is None:
        return [], False, {}
    suffix = "lower(regexp_extract(key, '(\\.[^./]+)$', 1))"
    allowed = ", ".join(f"'{e}'" for e in PROCESSABLE_EXTENSIONS)
    con = duckdb.connect()
    try:
        con.execute("SET memory_limit='1GB'")
        base = (
            f"SELECT *, {IDENTITY_SQL} AS identity, {suffix} AS ext "
            f"FROM read_parquet('{_posix(inventory)}') WHERE kind = 'media'"
            f"{_prefix_sql(path_prefix)}{_locator_sql(locators)}"
        )
        skipped_rows = con.execute(
            f"WITH b AS ({base}) SELECT "
            "sum(CASE WHEN ext IN ('.heic', '.heif') THEN 1 ELSE 0 END), "
            f"sum(CASE WHEN ext IN ({allowed}) AND size > {int(max_file_bytes)} "
            "THEN 1 ELSE 0 END) FROM b"
        ).fetchone()
        rows = con.execute(
            f"""
            WITH b AS ({base}), ok AS (
                SELECT * FROM b WHERE ext IN ({allowed}) AND size <= {int(max_file_bytes)}
            ), todo AS (SELECT o.* FROM ok o ANTI JOIN """
            f"""({_ledger_sql(output_dir, retry_failed)}) d USING (identity))
            SELECT identity, any_value(sha1) AS sha1, any_value(provider) AS provider, """
            f"""any_value(bucket) AS bucket,
                   min(key) AS key, max(size) AS size, count(*) AS occurrences
            FROM todo GROUP BY identity ORDER BY identity LIMIT {int(limit) + 1}
            """
        ).fetchall()
    finally:
        con.close()
    cols = ("identity", "sha1", "provider", "bucket", "key", "size", "occurrences")
    picked = [dict(zip(cols, r, strict=True)) for r in rows]
    skipped = {
        "unsupported_format": int(skipped_rows[0] or 0),
        "over_max_file_bytes": int(skipped_rows[1] or 0),
    }
    return picked[:limit], len(picked) > limit, skipped


def select_scanned_pdfs(
    output_dir: Path,
    limit: int,
    *,
    retry_failed: bool = False,
    path_prefix: str = "",
    locators: list[dict[str, str]] | None = None,
) -> tuple[list[dict[str, Any]], bool]:
    """Up to ``limit`` top-level PDFs the extract stage found no text in and the ledger has not
    covered."""
    inventory = newest_inventory(output_dir, "pdf")
    documents = output_dir / "datasets" / "documents"
    if inventory is None or not any(documents.glob("*.parquet")):
        return [], False
    con = duckdb.connect()
    try:
        con.execute("SET memory_limit='1GB'")
        rows = con.execute(
            f"""
            WITH scanned AS (
                SELECT DISTINCT replace(content_sha256, 'sha1:', '') AS sha1
                FROM read_parquet('{_posix(documents / "*.parquet")}', union_by_name = true)
                WHERE index_status = 'skipped_no_text' AND lower(extension) = '.pdf' """
            f"""AND coalesce(member_path, '') = ''
            ), pdf AS (
                SELECT *, {IDENTITY_SQL} AS identity """
            f"""FROM read_parquet('{_posix(inventory)}') WHERE coalesce(sha1, '') <> ''"""
            f"""{_prefix_sql(path_prefix)}{_locator_sql(locators)}
            ), todo AS (
                SELECT p.* FROM pdf p SEMI JOIN scanned s USING (sha1)
                ANTI JOIN ({_ledger_sql(output_dir, retry_failed)}) d USING (identity)
            )
            SELECT identity, any_value(sha1), any_value(provider), any_value(bucket), """
            f"""min(key), max(size), count(*)
            FROM todo GROUP BY identity ORDER BY identity LIMIT {int(limit) + 1}
            """
        ).fetchall()
    finally:
        con.close()
    cols = ("identity", "sha1", "provider", "bucket", "key", "size", "occurrences")
    picked = [dict(zip(cols, r, strict=True)) for r in rows]
    return picked[:limit], len(picked) > limit


# -------------------------------------------------- fetch


def published_pdf_pages(output_dir: Path, identity: str) -> set[int]:
    """Read the successful page positions for one PDF from retained publish ledgers.

    Inputs: owning output folder and PDF identity. Output: 1-based page numbers.
    Side effects: bounded-column Parquet reads. Pick this to resume a capped PDF or retry a page
    gap.
    """
    import pyarrow.parquet as pq

    pages = set()
    prefix = identity + "#p"
    for path in (lake(output_dir) / "published").glob("*.parquet"):
        for row in pq.read_table(path, columns=["identity", "object_id"]).to_pylist():
            if row["identity"].startswith(prefix) and row["object_id"]:
                page = row["identity"][len(prefix) :]
                if page.isdigit():
                    pages.add(int(page))
    return pages


async def _download(store: Any, key: str, target: Path, limit: int) -> int:
    """Stream one object into ``target`` in windows; refuse to exceed ``limit`` bytes. Returns the
    bytes written."""
    written = 0
    with target.open("wb") as handle:
        async for window in store.stream(key, window_bytes=WINDOW_BYTES):
            written += len(window)
            if written > limit:
                raise ValueError("object is larger than its catalog size allows")
            handle.write(window)
    return written


async def fetch_slice(
    output_dir: Path,
    spool_dir: Path | None,
    store_for: StoreFor,
    *,
    max_files: int = DEFAULT_MAX_FILES,
    max_bytes: int = DEFAULT_MAX_BYTES,
    max_file_bytes: int = DEFAULT_MAX_FILE_BYTES,
    pdf_max_pages: int = DEFAULT_MAX_PAGES,
    pdf_dpi: int = DEFAULT_DPI,
    retry_failed: bool = False,
    path_prefix: str = "",
    locators: list[dict[str, str]] | None = None,
    max_items: int = 0,
    beat: Beat | None = None,
) -> dict[str, Any]:
    """Materialize the next slice on the spool and write its manifest. Idempotent: an unpublished
    slice is resumed.

    Inputs: the lake, the spool folder, a ``(provider, bucket) -> ObjectStore`` factory and the
    bounds. Output: a
    small dict (slice id, kind, count, more, skipped counts); never image bytes. Side effects:
    bucket reads (one per
    image, windowed), files under ``<spool>/images/<slice_id>/``, one manifest. Images come first;
    scanned PDFs are
    taken when no image is left. A failed object is recorded in the manifest and ledgered as
    ``failed`` by publish."""
    _locator_sql(locators)  # Validate even when a pending slice exists.
    if max_items < 0 or min(max_files, max_bytes, max_file_bytes) < 1:
        raise ValueError("image slice bounds must be positive; max_items cannot be negative")
    max_items = min(max_items or max_files, max_files)
    max_files = min(max_files, max_items)
    resumed = open_slice_id(output_dir)
    if resumed:
        sl = load_slice(output_dir, resumed)
        allowed = (
            None
            if locators is None
            else {(item["provider"], item["bucket"], item["key"]) for item in locators}
        )
        if (
            (max_items and len(sl.items) + len(sl.failed) > max_items)
            or any(
                not item.key.startswith(path_prefix)
                or (allowed is not None and (item.provider, item.bucket, item.key) not in allowed)
                for item in sl.items
            )
            or ((path_prefix or locators is not None) and sl.failed)
        ):
            raise ValueError(
                "pending image slice is outside this proof's scope or remaining budget"
            )
        return _summary(sl, more=True, resumed=True)

    picked, more, skipped = select_images(
        output_dir,
        max_files,
        max_file_bytes,
        retry_failed=retry_failed,
        path_prefix=path_prefix,
        locators=locators,
    )
    kind = "image"
    if not picked:
        picked, more = select_scanned_pdfs(
            output_dir,
            max_files,
            retry_failed=retry_failed,
            path_prefix=path_prefix,
            locators=locators,
        )
        kind = "pdf"
    sl = Slice(new_slice_id(), kind, skipped=skipped)
    if not picked:
        return _summary(sl, more=False, resumed=False)

    folder = spool_root(output_dir, spool_dir) / sl.slice_id
    folder.mkdir(parents=True, exist_ok=True)
    total = 0
    quarantined = {"files": 0, "bytes": 0}

    def retain(source: Path) -> dict[str, int]:
        """Retain one source or overflow page and account for its quarantine totals.

        Inputs: a child spool path. Output: retained file/byte counts.
        Side effects: moves to owner quarantine. Pick this for all fetch-stage spool releases.
        """
        counts = quarantine_spool(output_dir, spool_dir, source)
        for name in quarantined:
            quarantined[name] += counts[name]
        return counts

    for row in picked:
        if max_items and len(sl.items) + len(sl.failed) >= max_items:
            more = True
            break
        size = int(row["size"])
        if size > max_file_bytes:
            sl.failed.append({"identity": row["identity"], "reason": "over_max_file_bytes"})
            continue
        if size > max_bytes:
            sl.failed.append({"identity": row["identity"], "reason": "over_slice_byte_budget"})
            continue
        if total + size > max_bytes:
            more = True  # The remaining original waits for the next slice.
            break
        store = store_for(row["provider"], row["bucket"])
        suffix = PurePosixPath(row["key"]).suffix.casefold() or ".bin"
        local = folder / f"{len(sl.items) + len(sl.failed):05d}{suffix}"
        try:
            written = await _download(
                store,
                row["key"],
                local,
                min(max(size, 1) + 4096, max_file_bytes, max_bytes - total),
            )
        # noqa: BLE001 - one bad object must not fail the slice; it is ledgered as failed
        except Exception as exc:
            retained = retain(local)
            total += retained["bytes"]
            sl.failed.append({"identity": row["identity"], "reason": type(exc).__name__})
            continue
        total += written
        if kind == "image":
            sl.items.append(
                ImageItem(
                    row["identity"],
                    row["sha1"] or "",
                    row["provider"],
                    row["bucket"],
                    row["key"],
                    int(row["size"]),
                    int(row["occurrences"]),
                    local.name,
                )
            )
        else:
            try:
                pages = (
                    min(pdf_max_pages, max_items - len(sl.items) - len(sl.failed))
                    if max_items
                    else pdf_max_pages
                )
                published = published_pdf_pages(output_dir, row["identity"])
                start_page = 1
                while start_page in published:
                    start_page += 1
                rendered = render_pdf_pages(
                    local,
                    folder,
                    max_pages=pages,
                    dpi=pdf_dpi,
                    stem=local.stem,
                    start_page=start_page,
                    max_output_bytes=max_bytes - total,
                )
            except ValueError as exc:
                sl.failed.append({"identity": row["identity"], "reason": str(exc)[:120]})
                retain(local)
                continue
            retain(local)
            for deferred in rendered.deferred:
                retain(deferred.path)
            total += sum(page.path.stat().st_size for page in rendered.pages)
            if not rendered.pages:
                sl.failed.append(
                    {"identity": row["identity"], "reason": "pdf_page_exceeds_slice_byte_budget"}
                )
            if rendered.deferred:
                more = True
            remaining_pages = rendered.page_count - len(
                published | {p.page for p in rendered.pages}
            )
            if remaining_pages:
                sl.skipped[row["identity"]] = remaining_pages
            for page in rendered.pages:
                sl.items.append(
                    ImageItem(
                        f"{row['identity']}#p{page.page}",
                        row["sha1"] or "",
                        row["provider"],
                        row["bucket"],
                        row["key"],
                        page.path.stat().st_size,
                        int(row["occurrences"]),
                        page.path.name,
                        page.page,
                        rendered.page_count,
                        page.width,
                        page.height,
                    )
                )
        if beat is not None:
            beat(f"fetched {len(sl.items)} images ({total // 1024} KiB) of slice {sl.slice_id}")
    write_json_immutable(manifest_path(output_dir, sl.slice_id), sl.to_json())
    result = _summary(sl, more=more, resumed=False, bytes_read=total)
    result["quarantined_files"] = quarantined["files"]
    result["quarantined_bytes"] = quarantined["bytes"]
    return result


def _summary(sl: Slice, *, more: bool, resumed: bool, bytes_read: int = 0) -> dict[str, Any]:
    return {
        "slice_id": sl.slice_id if sl.items or sl.failed else "",
        "kind": sl.kind,
        "count": len(sl.items),
        "failed": len(sl.failed),
        "more": more,
        "resumed": resumed,
        "bytes_read": bytes_read,
        "skipped": sl.skipped,
    }


def item_path(output_dir: Path, spool_dir: Path | None, slice_id: str, item: ImageItem) -> Path:
    return spool_root(output_dir, spool_dir) / slice_id / item.file
