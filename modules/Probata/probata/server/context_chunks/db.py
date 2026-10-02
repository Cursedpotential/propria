"""One read-only Postgres connection for the chunk units.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02
"""

from __future__ import annotations

from contextlib import contextmanager

from sqlalchemy import text
from sqlalchemy.engine import Connection


@contextmanager
def read_only_connection():
    """A connection on the platform database (PLATFORM_DB_URL, as every worker), read-only for the session.

    Autocommit, so a long job never sits idle inside one open transaction; every statement is its own read-only one.
    """
    from server.timeline.db import get_engine

    with get_engine().connect().execution_options(isolation_level="AUTOCOMMIT") as conn:
        conn.execute(text("SET default_transaction_read_only = on"))
        yield conn


def check(conn: Connection) -> None:
    conn.execute(text("SELECT 1"))
