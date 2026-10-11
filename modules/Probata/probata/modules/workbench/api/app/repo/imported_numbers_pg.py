"""Read-only Imported numbers queries over the platform projection.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

SQL, scope and result shapes are unchanged. Connection and error handling stay
in imported_pg; each query resolves that facade at call time for shared patches.
"""

from __future__ import annotations

from typing import Any

from app.repo import imported_pg as pg


def _number_forms(number: str) -> list[str]:
    """The ways a participant identifier spells a 10-digit number in the normalized records."""
    return [f'[{{"identifier": "{form}"}}]' for form in (f"+1{number}", number, f"1{number}", f"+{number}")]


def number_records(matter: str, number: str, *, ts: str | None, row_id: str | None, limit: int) -> list[dict[str, Any]]:
    """Every message and call that carries this number as a participant, newest first (keyset paged).

    Matches by JSON containment on the participants array (no per-row expansion), so it stays a single
    cheap scan of the matter's records.
    """
    return pg._query(
        f"""WITH {pg._SV}
        SELECT n.id::text AS id, n.occurred_at, n.record_type, n.source_version_id::text AS source_version_id,
               sv.source_key, sv.export_key, sv.conv,
               n.normalized_payload->'content'->>'body' AS body, n.normalized_payload->'content' AS content,
               n.normalized_payload->'participants' AS participants,
               n.normalized_payload->>'timestamp_certainty' AS certainty,
               r.projection_kind, m.has_attachments, m.attachment_count
        FROM context.normalized_record_identity n
        JOIN sv ON sv.id = n.source_version_id
        LEFT JOIN working.message_projection_route r ON r.normalized_record_id = n.id
        LEFT JOIN working.message m ON m.id = n.id
        WHERE n.normalized_payload->'participants' @> ANY (SELECT x::jsonb FROM unnest(%(forms)s::text[]) AS x)
          AND (%(ts)s::timestamptz IS NULL OR (n.occurred_at, n.id) < (%(ts)s::timestamptz, %(row)s::uuid))
        ORDER BY n.occurred_at DESC, n.id DESC
        LIMIT %(limit)s""",
        {"matter": matter, "forms": pg._number_forms(number), "ts": ts, "row": row_id, "limit": limit + 1},
    )


def number_record_counts(matter: str, number: str) -> dict[str, Any]:
    """Read the Imported number record counts projection.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    rows = pg._query(
        f"""WITH {pg._SV}
        SELECT count(*) FILTER (WHERE n.record_type = 'message') AS messages,
               count(*) FILTER (WHERE n.record_type = 'call') AS calls
        FROM context.normalized_record_identity n JOIN sv ON sv.id = n.source_version_id
        WHERE n.normalized_payload->'participants' @> ANY (SELECT x::jsonb FROM unnest(%(forms)s::text[]) AS x)""",
        {"matter": matter, "forms": pg._number_forms(number)},
    )
    return rows[0]
