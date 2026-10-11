"""Plan bounded Neural conversation chunks from immutable native SMS bundles.

Inputs are source-pinned extraction manifests; outputs are external chunk plans and compact counts.
Effects are derivative files only. Original bytes, PostgreSQL messages and vector collections are untouched.
Choose this path for human XML context; legacy generation planners require normalized identities.
"""
from __future__ import annotations

import hashlib
import json
import os
import sqlite3
import tempfile
from pathlib import Path
from urllib.parse import unquote, urlsplit

CONTRACT = "human-source-v1"
INDEX_KIND = "native-human-conversation-chunks-v1"
BOUNDARY_VERSION = "native-ordinal-window256-bytes131072-overlap2-utf8parts6000-nimprepared8000-v2"
MAX_ROW = 2 * 1024 * 1024
WINDOW_ROWS = 256
WINDOW_BYTES = 131072
MAX_TEXT_BYTES = 7500
MAX_PREPARED_BYTES = 8000
SEGMENT_BYTES = 6000
OVERLAP = 2


def bundle_path(root: Path, ref: str) -> Path:
    """Resolve an existing unlinked file inside the configured native bundle root.

    Inputs: configured root and file URI. Outputs: checked path. Effects: read-only path checks.
    """
    uri = urlsplit(ref)
    if uri.scheme != "file" or uri.netloc or uri.query or uri.fragment:
        raise ValueError("native bundle requires local file reference")
    root = root.absolute()
    path = Path(unquote(uri.path))
    relative = path.relative_to(root)
    if ".." in relative.parts or not relative.parts:
        raise ValueError("native bundle reference escaped root")
    cursor = root
    for part in ("", *relative.parts):
        cursor = cursor / part
        if cursor.is_symlink() or not cursor.exists():
            raise ValueError("native bundle path unavailable or linked")
    return path


def bounded_json_lines(path: Path):
    """Yield bounded JSON rows without reading an export or conversation into memory.

    Inputs: immutable JSONL path. Outputs: parsed rows. Effects: bounded file reads only.
    """
    with path.open("rb") as stream:
        while raw := stream.readline(MAX_ROW + 1):
            if len(raw) > MAX_ROW:
                raise ValueError("native row exceeds bound")
            yield json.loads(raw)


def _segments(text: str):
    """Split an oversize native body at UTF-8 boundaries, preserving exact decoded character spans."""
    start = 0
    while start < len(text):
        end, size = start, 0
        while end < len(text):
            width = len(text[end].encode("utf-8"))
            if size + width > SEGMENT_BYTES:
                break
            size += width
            end += 1
        yield start, end, text[start:end]
        start = end
    if not text:
        yield 0, 0, ""


def _spool(connection, records_path, source, beat):
    """Spool complete native records by explicit conversation key and source ordinal.

    Outputs retain exact locators/body segments. Disk SQLite is a derivative spool, never a new corpus.
    """
    connection.execute("CREATE TABLE pieces (thread TEXT, ordinal INTEGER, segment INTEGER, payload TEXT)")
    connection.execute("CREATE INDEX ordered_pieces ON pieces(thread,ordinal,segment)")
    complete = pending = total = 0
    for record in bounded_json_lines(records_path):
        total += 1
        if record["status"] != "complete":
            pending += 1
            continue
        native = record["native"]
        if native.get("record_kind") != "message":
            raise ValueError("complete native record is not a message")
        locator = native["source_locator"]
        if locator.get("source_version_ref") != source["source_version_ref"] or locator.get("source_ref") != source["source_ref"] or locator.get("record_ordinal") != record["ordinal"]:
            raise ValueError("native record locator differs from registered source")
        addresses = sorted({p if isinstance(p, str) else p.get("address") for p in (native.get("participants") or [])} - {None, "", "insert-address-token"})
        key = json.dumps([source["source_version_ref"], source.get("owner_person_ref", ""), addresses], ensure_ascii=False, separators=(",", ":"))
        # Unknown participants stay separate: there is no basis for merging them into one conversation.
        if not addresses:
            key += ":unknown:" + str(record["ordinal"])
        thread = "native-conversation:" + hashlib.sha256(key.encode()).hexdigest()
        sender = native.get("sender_address") or ("source owner" if native.get("sender_role") == "source_owner" else "unknown sender")
        for segment, (first, last, body) in enumerate(_segments(native.get("body") or "")):
            # Literal text retains whitespace. Search rendering is derived; native JSON remains unchanged.
            line = str(sender)[:200] + ": " + (body or "(attachment; original range retained)")
            item = {"ordinal": record["ordinal"], "segment": segment, "body_start": first, "body_end": last,
                    "locator": locator, "text": line, "at": native.get("occurred_at"), "participants": addresses}
            connection.execute("INSERT INTO pieces VALUES (?,?,?,?)", (thread, record["ordinal"], segment, json.dumps(item, ensure_ascii=False)))
        complete += 1
        if total % 128 == 0:
            connection.commit()
            beat("native spool " + str(total))
    connection.commit()
    return total, complete, pending


def _prepared_size(text: str) -> int:
    """Measure NIM's exact lowercase image-marker expansion without altering source text.

    Input: rendered nonblank chunk text. Output: UTF-8 byte count after PrepareInput's
    data:image/ -> data: image/ replacement. Effects: none. Choose for embedding budgets;
    original spans and emitted text remain unchanged, and no truncation is simulated.
    """
    return len(text.encode("utf-8")) + text.count("data:image/")


def _bounded_ranges(items, spans):
    """Split Neural spans against both original-text and actual NIM-prepared byte budgets.

    Inputs: source-pinned rendered pieces and Neural ranges. Outputs: bounded piece ranges.
    Effects: none. Choose after overlap so separators, sender prefixes and lowercase marker
    expansion count toward embedding limits without modifying exact original body spans.
    """
    seen = set()
    for first, last in spans:
        start, size, prepared_size = first, 0, 0
        for index in range(first, last + 1):
            separator = 1 if index > start else 0
            width = len(items[index]["text"].encode("utf-8")) + separator
            prepared_width = _prepared_size(items[index]["text"]) + separator
            if (size + width > MAX_TEXT_BYTES or prepared_size + prepared_width > MAX_PREPARED_BYTES) and index > start:
                pair = (start, index - 1)
                if pair not in seen:
                    seen.add(pair)
                    yield pair
                start, size, prepared_size = index, 0, 0
                width = len(items[index]["text"].encode("utf-8"))
                prepared_width = _prepared_size(items[index]["text"])
            if width > MAX_TEXT_BYTES or prepared_width > MAX_PREPARED_BYTES:
                raise ValueError("native text segment exceeds embedding bound")
            size += width
            prepared_size += prepared_width
        pair = (start, last)
        if pair not in seen:
            seen.add(pair)
            yield pair


def validate_source_pins(expected: dict, observed: dict) -> None:
    """Validate manifest identity, attribution and known provider pins against the registered request.

    Inputs: expected registration pins and observed manifest source. Output: None or ValueError.
    Effects: none. Choose before any derivative spool or planning; only unknown provider version,
    byte length and ETag may become observed. Mirrors humancontext.ValidateSourcePins and SameETag.
    """
    if any(not expected.get(field) for field in ("source_ref", "source_version_ref", "registration_receipt_ref")):
        raise ValueError("human source requires registered identity references")
    for field in ("source_ref", "source_version_ref", "registration_receipt_ref", "format",
                  "owner_person_ref", "perspective_person_ref", "perspective"):
        if expected.get(field, "") != observed.get(field, ""):
            raise ValueError(f"human source {field} pin differs")
    if expected.get("provider_version_id") and expected["provider_version_id"] != observed.get("provider_version_id", ""):
        raise ValueError("human source provider version pin differs")
    expected_bytes, observed_bytes = expected.get("byte_length", 0), observed.get("byte_length", 0)
    if expected_bytes < 0 or observed_bytes < 0 or (expected_bytes > 0 and expected_bytes != observed_bytes):
        raise ValueError("human source byte length pin differs")
    if expected.get("etag") and expected["etag"].strip().strip('"') != observed.get("etag", "").strip().strip('"'):
        raise ValueError("human source ETag pin differs")


def plan_native_chunks(root: Path, request: dict, *, span_planner=None, version=None, beat=lambda _: None) -> dict:
    """Write immutable conversation chunk plans using the existing Neural distilbert chunker.

    Inputs: reference-only registered extraction request and optional test planner.
    Outputs: chunk manifest/counts. Effects: bounded derivative spool/plans; no index/schema/original writes.
    Window overlap and oversize body segments retain native ordinals and exact decoded/native byte locators.
    """
    if span_planner is None:
        from server.context_chunks.chunker import chunk_spans, chunker_version
        span_planner = chunk_spans
        version = chunker_version("neural_distilbert", OVERLAP)
    if not version:
        raise ValueError("chunker version is required")
    if request["contract_version"] != CONTRACT:
        raise ValueError("native chunk contract differs")
    manifest_path = bundle_path(root, request["manifest_ref"])
    if manifest_path.stat().st_size > MAX_ROW:
        raise ValueError("native extraction manifest exceeds bound")
    manifest = json.loads(manifest_path.read_bytes())
    if manifest["contract_version"] != CONTRACT or manifest["stage"] != "extracted" or not request["workflow_id"] or manifest["workflow_id"] != request["workflow_id"]:
        raise ValueError("native chunk extraction coordinates differ")
    validate_source_pins(request["source"], manifest["source"])
    directory = Path(tempfile.mkdtemp(prefix="chunked-", dir=root))
    chunks_path = directory / "chunks.jsonl"
    chunks, windows = 0, 0
    with sqlite3.connect(directory / "ordinal-spool.sqlite") as connection, chunks_path.open("x", encoding="utf-8") as output:
        connection.execute("PRAGMA cache_size=-4096")
        total, complete, pending = _spool(connection, bundle_path(root, manifest["records_ref"]), manifest["source"], beat)
        if total != manifest["records"] or pending != manifest["pending"]:
            raise ValueError("native chunk extraction accounting differs")
        thread_cursor = connection.execute("SELECT DISTINCT thread FROM pieces ORDER BY thread")
        for (thread,) in thread_cursor:
            rows = connection.execute("SELECT payload FROM pieces WHERE thread=? ORDER BY ordinal,segment", (thread,))
            window, size, index, window_id = [], 0, 0, 0

            def emit(items):
                nonlocal chunks, windows, index, window_id
                if not items:
                    return
                spans = span_planner([item["text"] for item in items], "neural_distilbert", OVERLAP, max_chars=MAX_TEXT_BYTES, beat=beat)
                covered = set()
                ranges = list(_bounded_ranges(items, spans))
                for range_index, (first, last) in enumerate(ranges):
                    members = items[first:last + 1]
                    covered.update(range(first, last + 1))
                    refs = ["native-record:" + manifest["source"]["source_version_ref"] + ":" + str(m["ordinal"]) + ":" + str(m["body_start"]) + ":" + str(m["body_end"]) for m in members]
                    chunk = {"thread_id": thread, "chunk_index": index, "window_index": window_id,
                             "text": "\n".join(m["text"] for m in members), "message_refs": refs,
                             "members": members, "chunker": "neural_distilbert", "chunker_version": version + ";" + BOUNDARY_VERSION,
                             "boundary_version": BOUNDARY_VERSION, "requested_overlap": OVERLAP,
                             "overlap": max(0, min(last, ranges[range_index + 1][1]) - max(first, ranges[range_index + 1][0]) + 1) if range_index + 1 < len(ranges) else 0}
                    output.write(json.dumps(chunk, ensure_ascii=False, separators=(",", ":")) + "\n")
                    index += 1
                    chunks += 1
                if covered != set(range(len(items))):
                    raise ValueError("native Neural plan omitted source segments")
                windows += 1
                window_id += 1
                beat("native chunk windows " + str(windows))

            for (payload,) in rows:
                item = json.loads(payload)
                width = len(item["text"].encode("utf-8"))
                if window and (len(window) >= WINDOW_ROWS or size + width > WINDOW_BYTES):
                    emit(window)
                    window = window[-OVERLAP:]
                    size = sum(len(m["text"].encode("utf-8")) for m in window)
                window.append(item)
                size += width
            emit(window)
        output.flush()
        os.fsync(output.fileno())
    manifest.update(stage="chunked", chunks_ref=chunks_path.as_uri(), chunks=chunks,
                    index_kind=INDEX_KIND, chunker_version=version + ";" + BOUNDARY_VERSION,
                    boundary_version=BOUNDARY_VERSION, chunk_windows=windows, chunked_records=complete)
    result_path = directory / "manifest.json"
    with result_path.open("x", encoding="utf-8") as output:
        json.dump(manifest, output, ensure_ascii=False, separators=(",", ":"))
        output.flush()
        os.fsync(output.fileno())
    return {"contract_version": CONTRACT, "stage": "chunked", "status": "partial" if pending or not manifest["framing"]["complete"] else "complete",
            "manifest_ref": result_path.as_uri(), "index_kind": INDEX_KIND, "records": total, "pending": pending,
            "chunks": chunks, "published": 0, "verified": 0}
