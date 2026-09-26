"""Persistence package for Legal-Workspace.

> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_
"""

from legal_workspace.db.engine import default_sqlite_url, get_engine, is_sqlite
from legal_workspace.db.models import Base
from legal_workspace.db.store import WorkspaceStore, create_tables

__all__ = ["Base", "get_engine", "is_sqlite", "default_sqlite_url", "create_tables", "WorkspaceStore"]
