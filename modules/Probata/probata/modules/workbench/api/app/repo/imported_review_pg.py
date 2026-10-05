"""Read-only Imported review queries over the platform projection.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

SQL, scope and result shapes are unchanged. Connection and error handling stay
in imported_pg; each query resolves that facade at call time for shared patches.
"""

from __future__ import annotations

from typing import Any

from app.repo import imported_pg as pg


def review_queue(matter: str) -> list[dict[str, Any]]:
    """Previews waiting for the owner's decision: newest snapshot is awaiting_decision, no decision yet."""
    return pg._query(
        f"""WITH {pg._SV},
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
