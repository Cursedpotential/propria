"""Read source-pinned conversation units without combining historical generations.

Inputs: server-resolved source keys and generation IDs. Outputs: PG readbacks.
Effects: SELECT only via the existing Imported reader. No schema or data writes.
Byline: ChatGPT, 2026-10-17.
"""
from __future__ import annotations

from typing import Any
from app.repo import imported_pg as pg


def snapshot(matter: str, exports: list[str], conv: str | None = None) -> list[dict[str, Any]]:
    """Resolve the latest retained version/generation for each exact source key.

    Input: server-resolved export keys and optional conversation. Output: version
    inventory, counts and selected generations. SELECT only; history is not deleted.
    """
    return pg._query(f"""WITH {pg._SV},
      picked AS (
        SELECT DISTINCT ON (sv.source_key) sv.*,
          count(*) OVER (PARTITION BY sv.source_key) AS retained_versions
        FROM sv WHERE sv.export_key = ANY(%(exports)s::text[])
          AND (%(conv)s::text IS NULL OR sv.conv = %(conv)s)
        ORDER BY sv.source_key, sv.version_ordinal DESC, sv.acquired_at DESC NULLS LAST, sv.id DESC
      )
      SELECT p.id::text AS source_version_id, p.source_key, p.export_key, p.conv,
        p.version_ordinal, p.retained_versions, p.acquired_at, p.published,
        g.id::text AS generation_id, g.generation_ordinal,
        (SELECT count(*) FROM context.raw_record_identity r
          WHERE r.source_version_id = p.id AND r.record_status = 'parsed') AS raw_records,
        (SELECT count(*) FROM context.normalized_record_identity n
          WHERE n.normalized_generation_id = g.id) AS normalized_records,
        (SELECT count(*) FROM context.normalized_record_identity n
          WHERE n.normalized_generation_id = g.id AND n.record_type = 'message') AS messages
      FROM picked p
      LEFT JOIN LATERAL (
        SELECT ng.id, ng.generation_ordinal FROM context.normalized_generation ng
        WHERE ng.source_version_id = p.id ORDER BY ng.generation_ordinal DESC, ng.id DESC LIMIT 1
      ) g ON true
      ORDER BY p.source_key""", {"matter": matter, "exports": exports, "conv": conv})


def message_page(matter: str, generations: list[str], *, q: str, after_time: str | None,
                 after_id: str | None, limit: int) -> list[dict[str, Any]]:
    """Read a keyset page after filtering the entire selected PG generation set.

    Input: exact server-selected generation IDs, literal text and cursor. Output:
    normalized records and their source keys, including complete normalized payloads.
    SELECT only; no body/time deduplication and no Weaviate or Super Index call.
    """
    return pg._query(f"""WITH {pg._SV}, filtered AS (
      SELECT n.id, n.occurred_at, n.source_version_id, n.normalized_generation_id,
        n.record_ordinal, n.normalized_payload, sv.source_key, sv.export_key, sv.conv
      FROM context.normalized_record_identity n JOIN sv ON sv.id = n.source_version_id
      WHERE n.normalized_generation_id = ANY(%(generations)s::uuid[])
        AND n.record_type = 'message'
        AND (%(q)s = '' OR strpos(lower(coalesce(n.normalized_payload->'content'->>'body', '')), lower(%(q)s)) > 0)
    )
    SELECT f.id::text AS id, f.occurred_at, f.source_version_id::text AS source_version_id,
      f.normalized_generation_id::text AS generation_id, f.record_ordinal,
      f.normalized_payload->'content'->>'body' AS body,
      f.normalized_payload->'participants' AS participants,
      f.normalized_payload->>'timestamp_certainty' AS certainty,
      f.source_key, f.export_key, f.conv,
      (SELECT sn.preview_handle FROM context.proffer_preview_snapshot sn
       WHERE sn.normalized_generation_id = f.normalized_generation_id
       ORDER BY sn.snapshot_seq DESC LIMIT 1) AS preview_handle,
      route.projection_kind, m.has_attachments, m.attachment_count
    FROM filtered f
    LEFT JOIN working.message m ON m.id = f.id
    LEFT JOIN LATERAL (
      SELECT CASE WHEN count(DISTINCT r.projection_kind) > 1 THEN 'mixed'
                  ELSE min(r.projection_kind::text) END AS projection_kind
      FROM working.message_projection_route r WHERE r.normalized_record_id = f.id
    ) route ON true
    WHERE (%(after_id)s::uuid IS NULL
      OR (%(after_time)s::timestamptz IS NOT NULL AND
          (f.occurred_at > %(after_time)s::timestamptz OR f.occurred_at IS NULL
           OR (f.occurred_at = %(after_time)s::timestamptz AND f.id > %(after_id)s::uuid)))
      OR (%(after_time)s::timestamptz IS NULL AND f.occurred_at IS NULL AND f.id > %(after_id)s::uuid))
    ORDER BY f.occurred_at ASC NULLS LAST, f.id ASC LIMIT %(limit)s""",
    {"matter": matter, "generations": generations, "q": q,
     "after_time": after_time, "after_id": after_id, "limit": limit + 1})


def record(matter: str, exports: list[str], record_id: str) -> dict[str, Any] | None:
    """Read one actual normalized row within the selected unit, including history.

    Inputs: server-derived unit keys and record UUID. Output: full normalized payload
    with exact stored coordinates or None. SELECT only; source membership is rechecked.
    """
    rows = pg._query(f"""WITH {pg._SV}
      SELECT n.id::text AS id, n.source_version_id::text AS source_version_id,
        n.normalized_generation_id::text AS generation_id, n.record_ordinal,
        n.record_type, n.occurred_at, n.normalized_payload, sv.source_key, sv.export_key,
        (SELECT sn.preview_handle FROM context.proffer_preview_snapshot sn
         WHERE sn.normalized_generation_id = n.normalized_generation_id
         ORDER BY sn.snapshot_seq DESC LIMIT 1) AS preview_handle
      FROM context.normalized_record_identity n JOIN sv ON sv.id = n.source_version_id
      WHERE sv.export_key = ANY(%(exports)s::text[]) AND n.id = %(record_id)s::uuid""",
      {"matter": matter, "exports": exports, "record_id": record_id})
    return rows[0] if rows else None
