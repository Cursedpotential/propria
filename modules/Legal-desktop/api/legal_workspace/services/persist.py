"""Atomic JSON persistence. RAM is never the source of truth.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

import json
import os
from datetime import UTC, datetime
from pathlib import Path
from typing import Any


def default_store_dir() -> Path:
    """Return the workspace state directory.

    ``LEGAL_WORKSPACE_STORE_DIR`` wins when set. It is required in the container:
    the package is installed into site-packages, so the repo-relative default below
    resolves to an ephemeral path inside the image that is destroyed on every
    redeploy. On the VPS this points at a bind-mounted host directory.
    """
    override = os.environ.get("LEGAL_WORKSPACE_STORE_DIR", "").strip()
    if override:
        directory = Path(override)
        directory.mkdir(parents=True, exist_ok=True)
        return directory
    return Path(__file__).resolve().parents[3] / "data" / "workspace"


def atomic_write_json(path: Path, payload: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(path.suffix + ".tmp")
    text = json.dumps(payload, indent=2, default=str)
    tmp.write_text(text, encoding="utf-8")
    os.replace(tmp, path)


def append_jsonl(path: Path, payload: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    record = {"at": datetime.now(UTC).isoformat(), **payload}
    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(record, default=str) + "\n")
        handle.flush()
        os.fsync(handle.fileno())


def read_json(path: Path) -> Any | None:
    if not path.is_file():
        return None
    return json.loads(path.read_text(encoding="utf-8"))


def read_jsonl(path: Path, *, limit: int = 200) -> list[dict[str, Any]]:
    if not path.is_file():
        return []
    rows: list[dict[str, Any]] = []
    with path.open(encoding="utf-8") as handle:
        for line in handle:
            text = line.strip()
            if not text:
                continue
            try:
                parsed = json.loads(text)
            except json.JSONDecodeError:
                continue
            if isinstance(parsed, dict):
                rows.append(parsed)
    if limit <= 0:
        return rows
    return rows[-limit:]
