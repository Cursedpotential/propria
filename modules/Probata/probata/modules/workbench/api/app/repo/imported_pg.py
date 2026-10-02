"""Read-only reads of what has been imported: context.* and working.* on the platform database.

Byline: Claude Code · Sonnet · 2026-10-02

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


class ImportedError(Exception):
    def __init__(self, message: str, status: int = 503):
        self.message, self.status = message, status
        super().__init__(message)


# One source_version (a derived thread file, a calls export, a Facebook message file, ...) with its
# export (the original file it was split from) and its conversation key (the thread it belongs to).
_SV = r"""
sv AS (
  SELECT v.id, v.acquired_at, s.source_key,
         regexp_replace(s.source_key, '\.derived/.*$', '') AS export_key,
         COALESCE(substring(s.source_key from '\.derived/threads/(.+?)(?:\.[0-9]{4})?\.ndjson$'),
                  regexp_replace(s.source_key, '^.*/', '')) AS conv
  FROM context.source_version v
  JOIN context.source s ON s.id = v.source_id
  WHERE v.matter_id = %(matter)s::uuid
)
"""


def configured() -> bool:
    path = settings.imported_pg_password_file
    return bool(path and Path(path).is_file())


def _query(sql: str, params: dict[str, Any]) -> list[dict[str, Any]]:
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


def source_versions(matter: str) -> list[dict[str, Any]]:
    """Every source version of the matter with raw / normalized counts and its review state."""
    return _query(
        f"""WITH {_SV},
        rawc AS (SELECT r.source_version_id, count(*) AS n
                 FROM context.raw_record_identity r JOIN sv ON sv.id = r.source_version_id
                 WHERE r.record_status = 'parsed' GROUP BY 1),
        nr AS (SELECT n.source_version_id, count(*) AS n,
                      count(*) FILTER (WHERE n.record_type = 'message') AS msgs,
                      count(*) FILTER (WHERE n.record_type = 'call') AS calls,
                      min(n.occurred_at) AS first_at, max(n.occurred_at) AS last_at
               FROM context.normalized_record_identity n JOIN sv ON sv.id = n.source_version_id GROUP BY 1),
        appr AS (SELECT DISTINCT sn.source_version_id
                 FROM context.proffer_preview_snapshot sn
                 JOIN context.proffer_preview_decision d ON d.preview_handle = sn.preview_handle AND d.approved
                 JOIN sv ON sv.id = sn.source_version_id),
        rej AS (SELECT DISTINCT sn.source_version_id
                FROM context.proffer_preview_snapshot sn
                JOIN context.proffer_preview_decision d ON d.preview_handle = sn.preview_handle AND NOT d.approved
                JOIN sv ON sv.id = sn.source_version_id)
        SELECT sv.id::text AS id, sv.source_key, sv.export_key, sv.acquired_at,
               COALESCE(rawc.n, 0) AS raw_n, COALESCE(nr.n, 0) AS norm_n,
               COALESCE(nr.msgs, 0) AS msgs, COALESCE(nr.calls, 0) AS calls,
               nr.first_at, nr.last_at,
               (appr.source_version_id IS NOT NULL) AS approved,
               (rej.source_version_id IS NOT NULL) AS rejected
        FROM sv
        LEFT JOIN rawc ON rawc.source_version_id = sv.id
        LEFT JOIN nr ON nr.source_version_id = sv.id
        LEFT JOIN appr ON appr.source_version_id = sv.id
        LEFT JOIN rej ON rej.source_version_id = sv.id
        ORDER BY sv.acquired_at DESC, sv.id DESC""",
        {"matter": matter},
    )


def threads(matter: str, export_key: str, *, limit: int, offset: int) -> list[dict[str, Any]]:
    """The conversations inside one export, newest activity first."""
    return _query(
        f"""WITH {_SV}, sel AS (SELECT * FROM sv WHERE export_key = %(export)s)
        SELECT sel.conv, count(DISTINCT sel.id) AS files,
               count(n.id) AS records,
               count(n.id) FILTER (WHERE n.record_type = 'message') AS msgs,
               count(n.id) FILTER (WHERE n.record_type = 'call') AS calls,
               min(n.occurred_at) AS first_at, max(n.occurred_at) AS last_at,
               count(r.normalized_record_id) FILTER (WHERE r.projection_kind = 'first_party') AS first_party,
               count(r.normalized_record_id) FILTER (WHERE r.projection_kind <> 'first_party') AS third_party
        FROM sel
        LEFT JOIN context.normalized_record_identity n ON n.source_version_id = sel.id
        LEFT JOIN working.message_projection_route r ON r.normalized_record_id = n.id
        GROUP BY sel.conv
        HAVING count(n.id) > 0
        ORDER BY max(n.occurred_at) DESC NULLS LAST, sel.conv
        LIMIT %(limit)s OFFSET %(offset)s""",
        {"matter": matter, "export": export_key, "limit": limit + 1, "offset": offset},
    )


def thread_participants(matter: str, export_key: str, convs: list[str]) -> list[dict[str, Any]]:
    """Distinct participant identifiers per conversation (bounded to one page of threads)."""
    if not convs:
        return []
    return _query(
        f"""WITH {_SV}, sel AS (SELECT * FROM sv WHERE export_key = %(export)s AND conv = ANY(%(convs)s))
        SELECT sel.conv, p.value->>'identifier' AS identifier, count(*) AS n
        FROM sel
        JOIN context.normalized_record_identity n ON n.source_version_id = sel.id
        CROSS JOIN LATERAL jsonb_array_elements(n.normalized_payload->'participants') AS p(value)
        WHERE p.value->>'identifier' IS NOT NULL
        GROUP BY sel.conv, p.value->>'identifier'
        ORDER BY sel.conv, count(*) DESC""",
        {"matter": matter, "export": export_key, "convs": convs},
    )


def messages(
    matter: str,
    export_key: str,
    conv: str,
    *,
    ts: str | None,
    row_id: str | None,
    direction: str,
    limit: int,
) -> list[dict[str, Any]]:
    """One keyset page of a thread's messages. `before` is newest-first, `after` is oldest-first."""
    older = direction == "before"
    comparison = "<" if older else ">"
    order = "DESC" if older else "ASC"
    return _query(
        f"""WITH {_SV}, sel AS (SELECT id FROM sv WHERE export_key = %(export)s AND conv = %(conv)s)
        SELECT n.id::text AS id, n.occurred_at, n.source_version_id::text AS source_version_id,
               n.normalized_payload->'content'->>'body' AS body,
               n.normalized_payload->'participants' AS participants,
               n.normalized_payload->>'timestamp_certainty' AS certainty,
               r.projection_kind, m.has_attachments, m.attachment_count
        FROM context.normalized_record_identity n
        JOIN sel ON sel.id = n.source_version_id
        LEFT JOIN working.message_projection_route r ON r.normalized_record_id = n.id
        LEFT JOIN working.message m ON m.id = n.id
        WHERE n.record_type = 'message'
          AND (%(ts)s::timestamptz IS NULL OR (n.occurred_at, n.id) {comparison} (%(ts)s::timestamptz, %(row)s::uuid))
        ORDER BY n.occurred_at {order}, n.id {order}
        LIMIT %(limit)s""",
        {"matter": matter, "export": export_key, "conv": conv, "ts": ts, "row": row_id, "limit": limit + 1},
    )


def message_time(matter: str, row_id: str) -> dict[str, Any] | None:
    rows = _query(
        f"""WITH {_SV}
        SELECT n.occurred_at FROM context.normalized_record_identity n JOIN sv ON sv.id = n.source_version_id
        WHERE n.id = %(row)s::uuid""",
        {"matter": matter, "row": row_id},
    )
    return rows[0] if rows else None


def call_log_has_rows() -> bool:
    return bool(_query("SELECT EXISTS (SELECT 1 FROM working.call_log) AS has", {})[0]["has"])


def calls_normalized(matter: str, *, ts: str | None, row_id: str | None, limit: int) -> list[dict[str, Any]]:
    return _query(
        f"""WITH {_SV}
        SELECT n.id::text AS id, n.occurred_at, sv.export_key,
               n.normalized_payload->'content' AS content,
               n.normalized_payload->'participants' AS participants
        FROM context.normalized_record_identity n JOIN sv ON sv.id = n.source_version_id
        WHERE n.record_type = 'call'
          AND (%(ts)s::timestamptz IS NULL OR (n.occurred_at, n.id) < (%(ts)s::timestamptz, %(row)s::uuid))
        ORDER BY n.occurred_at DESC, n.id DESC
        LIMIT %(limit)s""",
        {"matter": matter, "ts": ts, "row": row_id, "limit": limit + 1},
    )


def calls_summary(matter: str) -> dict[str, Any]:
    rows = _query(
        f"""WITH {_SV}
        SELECT count(*) AS total,
               count(*) FILTER (WHERE (n.normalized_payload->'content'->>'missed')::boolean) AS missed,
               count(*) FILTER (WHERE n.normalized_payload->'content'->>'direction' = 'incoming') AS incoming,
               count(*) FILTER (WHERE n.normalized_payload->'content'->>'direction' = 'outgoing') AS outgoing,
               min(n.occurred_at) AS first_at, max(n.occurred_at) AS last_at
        FROM context.normalized_record_identity n JOIN sv ON sv.id = n.source_version_id
        WHERE n.record_type = 'call'""",
        {"matter": matter},
    )
    return rows[0]


def calls_from_log(*, ts: str | None, row_id: str | None, limit: int) -> list[dict[str, Any]]:
    return _query(
        """SELECT c.id::text AS id, c.started_at AS occurred_at, c.call_type, c.direction,
                  c.duration_s, c.from_e164, c.from_raw, c.to_e164, c.to_raw
           FROM working.call_log c
           WHERE (%(ts)s::timestamptz IS NULL OR (c.started_at, c.id) < (%(ts)s::timestamptz, %(row)s::uuid))
           ORDER BY c.started_at DESC NULLS LAST, c.id DESC
           LIMIT %(limit)s""",
        {"ts": ts, "row": row_id, "limit": limit + 1},
    )


def people() -> list[dict[str, Any]]:
    """Registry identifiers (the one identity store) so phone numbers show as names."""
    return _query(
        """SELECT person, display_name, role_in_case, identifier, kind
           FROM registry.vw_case_identifier WHERE status <> 'retired'""",
        {},
    )


def versions_to_threads(matter: str, version_ids: list[str]) -> list[dict[str, Any]]:
    """For search hits: which export and conversation each source version belongs to."""
    if not version_ids:
        return []
    return _query(
        f"""WITH {_SV}
        SELECT sv.id::text AS id, sv.export_key, sv.conv FROM sv WHERE sv.id = ANY(%(ids)s::uuid[])""",
        {"matter": matter, "ids": version_ids},
    )


def review_queue(matter: str) -> list[dict[str, Any]]:
    """Previews waiting for the owner's decision: newest snapshot is awaiting_decision, no decision yet."""
    return _query(
        f"""WITH {_SV},
        latest AS (SELECT DISTINCT ON (sn.preview_handle) sn.preview_handle, sn.phase, sn.source_version_id, sn.recorded_at
                   FROM context.proffer_preview_snapshot sn JOIN sv ON sv.id = sn.source_version_id
                   ORDER BY sn.preview_handle, sn.snapshot_seq DESC)
        SELECT l.preview_handle, l.recorded_at AS waiting_since, sv.id::text AS source_version_id,
               sv.source_key, sv.export_key, sv.conv,
               (SELECT count(*) FROM context.normalized_record_identity n WHERE n.source_version_id = sv.id) AS records
        FROM latest l JOIN sv ON sv.id = l.source_version_id
        WHERE l.phase = 'awaiting_decision'
          AND NOT EXISTS (SELECT 1 FROM context.proffer_preview_decision d WHERE d.preview_handle = l.preview_handle)
        ORDER BY l.recorded_at DESC""",
        {"matter": matter},
    )
