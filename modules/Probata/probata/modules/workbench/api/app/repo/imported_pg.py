"""Read-only reads of what has been imported: context.* and working.* on the platform database.

Byline: Claude Code · Sonnet · 2026-10-02
Updated: Codex · GPT-6.1-Sol · 2026-10-05 (query-domain split).

The mobile Imported view (Workbench /m) is step 5 of the six steps, Preview: it shows what the
machine put into context so the owner can see it did it right. Nothing here writes. The login is
`workbench_reader` (sql/bootstrap/workbench_reader_20261002.sql): SELECT-only on the exact tables
below, read-only by default, 8 s statement timeout. The password is read from the mounted file on
every connection, so a rotation needs no restart.

Scope is one matter, the live case, passed in by the caller. Error text never carries a connection
string, a row or an upstream body.
"""

from __future__ import annotations

from pathlib import Path
from typing import Any

from app.config import settings


from app.repo.imported_sources_pg import (
    source_versions as source_versions,
    threads as threads,
    thread_participants as thread_participants,
    thread_last_messages as thread_last_messages,
    messages as messages,
    message_time as message_time,
    versions_to_threads as versions_to_threads,
)
from app.repo.imported_calls_pg import (
    call_log_has_rows as call_log_has_rows,
    calls_normalized as calls_normalized,
    calls_summary as calls_summary,
    calls_from_log as calls_from_log,
)
from app.repo.imported_entities_pg import (
    people as people,
    numbers_activity as numbers_activity,
    entity_activity as entity_activity,
    working_unlinked_numbers as working_unlinked_numbers,
)
from app.repo.imported_numbers_pg import (
    _number_forms as _number_forms,
    number_records as number_records,
    number_record_counts as number_record_counts,
)
from app.repo.imported_review_pg import (
    review_queue as review_queue,
)

class ImportedError(Exception):
    """Carry a sanitized Imported failure and its HTTP status across read layers."""

    def __init__(self, message: str, status: int = 503):
        """Retain only the caller-supplied safe message and status."""
        self.message, self.status = message, status
        super().__init__(message)


# One source_version (a derived thread file, a calls export, a Facebook message file, ...) with its
# export (the original file it was split from) and its conversation key (the thread it belongs to).
_SV = r"""
sv AS (
  SELECT v.id, v.acquired_at, s.source_key, v.version_ordinal,
         EXISTS (SELECT 1 FROM context.activity_execution ae
                 JOIN context.activity_receipt ar ON ar.activity_execution_id = ae.id
                 WHERE ae.source_version_id = v.id AND ae.activity_name = 'publish_generation_activity' AND ar.status = 'success') AS published,
         regexp_replace(s.source_key, '\.derived/.*$', '') AS export_key,
         COALESCE(substring(s.source_key from '\.derived/threads/(.+?)(?:\.[0-9]{4})?\.ndjson$'),
                  regexp_replace(s.source_key, '^.*/', '')) AS conv
  FROM context.source_version v
  JOIN context.source s ON s.id = v.source_id
  WHERE v.matter_id = %(matter)s::uuid
)
"""


def configured() -> bool:
    """Check whether the configured read-only password file exists."""
    path = settings.imported_pg_password_file
    return bool(path and Path(path).is_file())


def _query(sql: str, params: dict[str, Any]) -> list[dict[str, Any]]:
    """Execute one bounded read using the rotated password and reader login.

    Inputs: parameterized SQL and bound parameters. Output: dictionary rows.
    Effects: reads the mounted password and PostgreSQL, never writes canonical data.
    Pick for Imported query domains; all failures retain sanitized error text.
    """
    if not configured():
        raise ImportedError("The imported-data connection is not configured")
    try:
        import psycopg
        from psycopg.rows import dict_row

        password = Path(settings.imported_pg_password_file).read_text().strip()
        with psycopg.connect(
            host=settings.imported_pg_host,
            port=settings.imported_pg_port,
            dbname=settings.imported_pg_database,
            user=settings.imported_pg_user,
            password=password,
            connect_timeout=5,
            row_factory=dict_row,
            options="-c default_transaction_read_only=on -c statement_timeout=8000 -c lock_timeout=1000",
        ) as conn:
            return conn.execute(sql, params).fetchall()
    except ImportedError:
        raise
    except Exception:
        raise ImportedError("The imported-data query is unavailable or timed out") from None
