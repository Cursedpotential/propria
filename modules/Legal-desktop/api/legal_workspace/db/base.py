"""SQLAlchemy declarative base shared across Legal-Workspace persistence.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
"""

from __future__ import annotations

from sqlalchemy.orm import DeclarativeBase


class Base(DeclarativeBase):
    """Shared declarative base. Schemas are mirrored by table prefixes until
    the app moves to PostgreSQL, where the same prefix maps to real schemas."""
