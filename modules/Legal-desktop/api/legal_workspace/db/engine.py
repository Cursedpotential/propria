"""Database engine factory. SQLite locally; PostgreSQL on the platform.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
"""

from __future__ import annotations

import re
from functools import lru_cache
from pathlib import Path
from typing import Any

from sqlalchemy import create_engine, event
from sqlalchemy.engine import Engine
from sqlalchemy.pool import NullPool


@lru_cache(maxsize=8)
def get_engine(database_url: str | None = None, store_dir: str | None = None) -> Engine:
    """Return a cached engine. Accepts an explicit URL or falls back to config.

    For local SQLite, ``store_dir`` lets callers place the DB beside the workspace
    state directory instead of the global default.
    """
    if database_url is not None:
        url = database_url
    elif store_dir is not None:
        url = default_sqlite_url(Path(store_dir))
    else:
        # Lazy import to avoid a circular dependency at module load time.
        from legal_workspace.config import get_settings as _get_settings

        url = _get_settings().database_url

    if url.startswith("sqlite"):
        engine = create_engine(
            url,
            connect_args={"check_same_thread": False},
            poolclass=NullPool,
            echo=False,
        )
        _configure_sqlite(engine)
    elif url.startswith("postgresql"):
        engine = create_engine(url, echo=False, future=True)
    else:
        raise ValueError(f"unsupported database URL scheme: {url}")

    return engine


def _configure_sqlite(engine: Engine) -> None:
    """Apply WAL, foreign keys, and schema-prefix parsing for local SQLite."""

    @event.listens_for(engine, "connect")
    def _on_connect(dbapi_conn: Any, _: Any) -> None:
        dbapi_conn.execute("PRAGMA journal_mode=WAL")
        dbapi_conn.execute("PRAGMA foreign_keys=ON")


def default_sqlite_url(store_dir: Path | None = None) -> str:
    """Return the default SQLite URL for local development/testing."""
    from legal_workspace.services.persist import default_store_dir

    directory = store_dir or default_store_dir()
    directory.mkdir(parents=True, exist_ok=True)
    # Windows paths need extra slashes for SQLite URI; use absolute path.
    path = directory / "legal.sqlite"
    return f"sqlite:///{path.resolve().as_posix()}"


def is_sqlite(engine: Engine | None = None) -> bool:
    """True if the active engine is SQLite."""
    target = engine or get_engine()
    return target.dialect.name == "sqlite"
