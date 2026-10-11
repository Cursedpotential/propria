"""Read-only Imported sources queries over the platform projection.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

SQL, scope and result shapes are unchanged. Connection and error handling stay
in imported_pg; each query resolves that facade at call time for shared patches.
"""

from __future__ import annotations

from typing import Any

from app.repo import imported_pg as pg


def source_versions(matter: str) -> list[dict[str, Any]]:
    """Every source version of the matter with raw / normalized counts and its review state."""
    return pg._query(
        f"""WITH {pg._SV},
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
        SELECT sv.id::text AS id, sv.source_key, sv.export_key, sv.acquired_at, sv.version_ordinal, sv.published,
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
    return pg._query(
        f"""WITH {pg._SV}, sel AS (SELECT * FROM sv WHERE export_key = %(export)s)
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
    return pg._query(
        f"""WITH {pg._SV}, sel AS (SELECT * FROM sv WHERE export_key = %(export)s AND conv = ANY(%(convs)s))
        SELECT sel.conv, p.value->>'identifier' AS identifier, count(*) AS n
        FROM sel
        JOIN context.normalized_record_identity n ON n.source_version_id = sel.id
        CROSS JOIN LATERAL jsonb_array_elements(n.normalized_payload->'participants') AS p(value)
        WHERE p.value->>'identifier' IS NOT NULL
        GROUP BY sel.conv, p.value->>'identifier'
        ORDER BY sel.conv, count(*) DESC""",
        {"matter": matter, "export": export_key, "convs": convs},
    )


def thread_last_messages(matter: str, export_key: str, convs: list[str]) -> list[dict[str, Any]]:
    """The newest message text of each conversation (for the conversation list), one row per conversation."""
    if not convs:
        return []
    return pg._query(
        f"""WITH {pg._SV}, sel AS (SELECT * FROM sv WHERE export_key = %(export)s AND conv = ANY(%(convs)s))
        SELECT DISTINCT ON (sel.conv) sel.conv, left(n.normalized_payload->'content'->>'body', 160) AS body, n.occurred_at
        FROM sel JOIN context.normalized_record_identity n ON n.source_version_id = sel.id AND n.record_type = 'message'
        ORDER BY sel.conv, n.occurred_at DESC, n.id DESC""",
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
    return pg._query(
        f"""WITH {pg._SV}, sel AS (SELECT id FROM sv WHERE export_key = %(export)s AND conv = %(conv)s)
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
    """Read the Imported message time projection.
    Inputs: the read scope and pagination arguments in this signature.
    Output: the unchanged Imported projection; effects: read-only upstream calls.
    Pick this domain read for its corresponding Imported view.
    """
    rows = pg._query(
        f"""WITH {pg._SV}
        SELECT n.occurred_at FROM context.normalized_record_identity n JOIN sv ON sv.id = n.source_version_id
        WHERE n.id = %(row)s::uuid""",
        {"matter": matter, "row": row_id},
    )
    return rows[0] if rows else None


def versions_to_threads(matter: str, version_ids: list[str]) -> list[dict[str, Any]]:
    """For search hits: which export and conversation each source version belongs to."""
    if not version_ids:
        return []
    return pg._query(
        f"""WITH {pg._SV}
        SELECT sv.id::text AS id, sv.export_key, sv.conv FROM sv WHERE sv.id = ANY(%(ids)s::uuid[])""",
        {"matter": matter, "ids": version_ids},
    )
