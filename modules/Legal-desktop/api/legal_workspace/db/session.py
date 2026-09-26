"""Database session utilities.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
"""

from __future__ import annotations

from contextlib import contextmanager
from typing import Generator

from sqlalchemy.orm import Session, sessionmaker

from legal_workspace.db.engine import get_engine


@contextmanager
def db_session(store_dir: str | None = None) -> Generator[Session, None, None]:
    """Yield a short-lived SQLAlchemy session and commit/rollback/close it."""
    maker = sessionmaker(bind=get_engine(store_dir=store_dir))
    session = maker()
    try:
        yield session
        session.commit()
    except Exception:
        session.rollback()
        raise
    finally:
        session.close()
