"""repair.json-repair — the json-repair library as its own selectable atomic tool (capability ``repair.json``).

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

The repair package already uses json-repair as the last step behind the streaming ijson reader
(``chunkers._json_repair_fallback``, reached through ``repair.preview``). This module is the same library on its own: one
JSON file in, one repaired JSON document and the list of repairs out, with nothing else attached (owner rule 2026-10-02:
one tool, one job, everything selectable).

The library is imported inside the call, as everywhere in this package, so registry discovery stays safe in an image
without it (see ``engines.py``).
"""

from __future__ import annotations

import json
from importlib import metadata
from pathlib import Path
from typing import Any

from server.tools.registry import register

# Cap on the repair log returned to the caller; the full count is always reported.
_MAX_REPAIRS_RETURNED = 200


@register(
    id="repair.json-repair",
    capability="repair.json",
    accept=lambda hint, size: hint.lower().endswith((".json", ".jsonl", ".txt")),
    provenance="json-repair (mangiucugna/json_repair) over server/tools/repair/encoding.open_text",
    tool_version="json-repair-0.62.0",
    formats=("json",),
    quality={"json": "fallback"},
)
def repair_json_document(payload: dict[str, Any]) -> dict[str, Any]:
    """Repair one damaged JSON file (truncated, unquoted, trailing commas, stray text) and return the repaired document.

    Formats: json, a single document. A file that already parses is returned unchanged (json-repair can rewrite valid
    JSON, so it is not run on it); ndjson is repaired per line by repair.preview, not here.
    Side effects: none; reads payload['path'] and returns {repaired_json, was_valid, repair_count, repairs, stats}. It
    writes no file; use repair.write-derived to keep a repaired copy. Refused above the 64 MiB whole-document cap,
    because json-repair cannot stream.
    Pick it when you want only the library's repair of a JSON document. Pick repair.preview instead when the file is
    large, an array of records, or you need the streaming reader (ijson) with this repair only as its fallback.
    """
    from server.tools.repair.chunkers import JSON_REPAIR_MAX_BYTES
    from server.tools.repair.encoding import open_text
    from server.tools.repair.types import RepairEvent

    path = Path(str(payload["path"]))
    if not path.is_file():
        raise FileNotFoundError(path)
    size = path.stat().st_size
    if size > JSON_REPAIR_MAX_BYTES:
        raise ValueError(
            f"{size:,} bytes exceeds the {JSON_REPAIR_MAX_BYTES:,}-byte whole-document repair cap; "
            "use repair.preview, which streams"
        )

    events: list[RepairEvent] = []
    with open_text(path, events=events) as handle:
        raw = handle.read()
    decoding = [event.as_row() for event in events]

    try:
        json.loads(raw)
    except ValueError:
        pass
    else:
        return _result(raw, True, [], decoding, size, "unchanged")

    import json_repair

    repaired, log = json_repair.repair_json(raw, return_objects=True, logging=True)
    if repaired == "":
        # json-repair returns "" when it finds no JSON structure at all; say so instead of returning an empty document.
        raise ValueError("json-repair found no JSON structure to recover in this file")
    text = json.dumps(repaired, ensure_ascii=False)
    return _result(text, False, log, decoding, size, "json-repair")


def _result(
    text: str, was_valid: bool, log: list[dict[str, str]], decoding: list[dict[str, Any]], size: int, method: str
) -> dict[str, Any]:
    try:
        version = metadata.version("json-repair")
    except metadata.PackageNotFoundError:
        version = "unavailable"
    return {
        "repaired_json": text,
        "was_valid": was_valid,
        "repair_count": len(log),
        "repairs": list(log[:_MAX_REPAIRS_RETURNED]),
        "decoding_events": decoding,
        "stats": {
            "method": method,
            "library_version": version,
            "source_bytes": size,
            "char_count": len(text),
            "repairs_truncated": len(log) > _MAX_REPAIRS_RETURNED,
        },
    }
