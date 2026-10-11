"""Read-only Imported calls queries over the platform projection.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

SQL, scope and result shapes are unchanged. Connection and error handling stay
in imported_pg; each query resolves that facade at call time for shared patches.
"""

from __future__ import annotations

from typing import Any

from app.repo import imported_pg as pg


def call_log_has_rows() -> bool:
    """Read the Imported call log has rows projection.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    return bool(pg._query("SELECT EXISTS (SELECT 1 FROM working.call_log) AS has", {})[0]["has"])


def calls_normalized(matter: str, *, ts: str | None, row_id: str | None, limit: int) -> list[dict[str, Any]]:
    """Read the Imported calls normalized projection.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    return pg._query(
        f"""WITH {pg._SV}
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
    """Read the Imported calls summary projection.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    rows = pg._query(
        f"""WITH {pg._SV}
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
    """Read the Imported calls from log projection.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    return pg._query(
        """SELECT c.id::text AS id, c.started_at AS occurred_at, c.call_type, c.direction,
                  c.duration_s, c.from_e164, c.from_raw, c.to_e164, c.to_raw
           FROM working.call_log c
           WHERE (%(ts)s::timestamptz IS NULL OR (c.started_at, c.id) < (%(ts)s::timestamptz, %(row)s::uuid))
           ORDER BY c.started_at DESC NULLS LAST, c.id DESC
           LIMIT %(limit)s""",
        {"ts": ts, "row": row_id, "limit": limit + 1},
    )
