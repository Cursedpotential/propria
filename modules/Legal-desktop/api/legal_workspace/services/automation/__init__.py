"""In-process automation: APScheduler, event bus, sequential playbooks.

> _Byline: Grok · grok-4.6 · 2026-08-18_
File-only jobstore locally. legal_core.automation_job is the later PG store
(Type 1 HOLD — do not apply). n8n outbound stays HOLD. No tab timers.
"""

from __future__ import annotations

from pathlib import Path

from legal_workspace.services.persist import default_store_dir

_store_dir: Path | None = None


def set_store_dir(path: Path | None) -> None:
    """Point mutations at the workspace store or a test tmp_path."""
    global _store_dir
    _store_dir = Path(path) if path is not None else None


def get_store_dir() -> Path:
    return _store_dir if _store_dir is not None else default_store_dir()
