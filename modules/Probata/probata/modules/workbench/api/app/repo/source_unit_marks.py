"""Durable file store for hand-marked source units.

Byline: Claude Code · Opus 5 · 2026-09-22.

The catalog tables (`raw_duck.atomic_units`) are read-only to the Workbench, so
a hand-marked unit is kept in the Workbench's own data volume as one JSON
document — the same shape the copilot presets file uses. Writes are atomic
(temporary file + replace) so a crash mid-write cannot truncate the store.
"""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Any

from app.config import settings


class UnitMarkStoreError(Exception):
    def __init__(self, message: str, status: int = 503):
        self.message, self.status = message, status


def _path() -> Path:
    return Path(settings.source_unit_marks_path)


def storage_description() -> str:
    return f"Workbench data volume file {settings.source_unit_marks_path}"


def read_all() -> list[dict[str, Any]]:
    path = _path()
    if not path.is_file():
        return []
    try:
        payload = json.loads(path.read_text(encoding="utf-8") or "{}")
    except (OSError, ValueError):
        raise UnitMarkStoreError("The hand-marked unit store could not be read") from None
    items = payload.get("items") if isinstance(payload, dict) else None
    return [item for item in items if isinstance(item, dict)] if isinstance(items, list) else []


def write_all(items: list[dict[str, Any]]) -> None:
    path = _path()
    temporary = path.with_suffix(path.suffix + ".writing")
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        temporary.write_text(json.dumps({"items": items}, indent=2), encoding="utf-8")
        os.replace(temporary, path)
    except OSError:
        raise UnitMarkStoreError("The hand-marked unit store could not be written") from None
