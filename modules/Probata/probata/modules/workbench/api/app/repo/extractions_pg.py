"""Read-only reads of what the extractors found for one conversation: working.extraction_run and working.candidate_*.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Same connection and rules as ``imported_pg`` (login ``workbench_reader``, SELECT-only, read-only session, 8 s statement
timeout; the connection helper and the ``sv`` CTE are imported from it, not copied). The extra grant is
``sql/bootstrap/workbench_extractions_20261002.sql``. Nothing here writes.

A conversation is its source versions; each source version's current normalized generation is what the extractors ran
over (the same resolution the engine uses, ``postgres/conversation_store.go``). Candidates of every extractor are returned,
each with the run it came from; a candidate the owner superseded is left out.
"""

from __future__ import annotations

from typing import Any

from app.repo.imported_pg import ImportedError, _query, _SV, configured  # noqa: F401 - ImportedError is re-exported to callers

# One page of each list per conversation; the response says when it was cut.
MAX_ROWS = 2000

_GENERATIONS = f"""{_SV},
gens AS (
  SELECT DISTINCT ON (sv.id) sv.id AS sv_id, g.id AS gen_id
  FROM sv JOIN context.normalized_generation g ON g.source_version_id = sv.id
  WHERE sv.export_key = %(export)s AND sv.conv = %(conv)s
  ORDER BY sv.id, g.generation_ordinal DESC
)"""


def runs(matter: str, export_key: str, conv: str) -> list[dict[str, Any]]:
    """Every extraction run of the conversation's current generations, oldest first."""
    return _query(
        f"""WITH {_GENERATIONS}
        SELECT r.id::text AS run_id, r.extractor, r.extractor_version, coalesce(r.model_id, '') AS model_id, r.status,
               coalesce(r.error, '') AS error, r.started_at, r.finished_at, r.stats, gens.gen_id::text AS generation_id
        FROM working.extraction_run r
        JOIN gens ON r.source_summary LIKE '%%generation:' || gens.gen_id::text || '%%'
        ORDER BY r.started_at, r.id""",
        {"matter": matter, "export": export_key, "conv": conv},
    )


def entities(matter: str, export_key: str, conv: str) -> list[dict[str, Any]]:
    """The conversation's non-superseded entity candidates of every extractor, by name."""
    return _query(
        f"""WITH {_GENERATIONS}
        SELECT e.id::text AS id, e.extraction_run_id::text AS run_id, e.name, e.entity_type,
               coalesce(e.confidence, 0)::float8 AS confidence, e.review_state,
               e.attrs->'aliases' AS aliases,
               coalesce(jsonb_array_length(e.attrs->'model_mentions'), 0) AS model_mentions,
               coalesce((e.attrs->>'mention_count')::int, 0) AS mention_count,
               (SELECT jsonb_agg(jsonb_build_object('record_id', m->>'record_id', 'surface', m->>'surface', 'snippet', m->>'snippet'))
                FROM (SELECT m FROM jsonb_array_elements(coalesce(e.attrs->'model_mentions', '[]'::jsonb)) AS m LIMIT 3) s) AS mentions
        FROM working.candidate_entity e
        JOIN gens ON e.source_raw_id = gens.gen_id::text
        WHERE e.source_raw_table = 'context.normalized_generation' AND e.review_state <> 'superseded'
        ORDER BY e.name, e.id
        LIMIT {MAX_ROWS + 1}""",
        {"matter": matter, "export": export_key, "conv": conv},
    )


def events(matter: str, export_key: str, conv: str) -> list[dict[str, Any]]:
    """The conversation's non-superseded event candidates of every extractor, by time."""
    return _query(
        f"""WITH {_GENERATIONS}
        SELECT v.id::text AS id, v.extraction_run_id::text AS run_id, v.summary AS title, v.event_type, v.occurred_at,
               v.attrs->>'temporal_precision' AS precision, coalesce(v.confidence, 0)::float8 AS confidence, v.review_state,
               (SELECT jsonb_agg(r->>'record_id') FROM jsonb_array_elements(coalesce(v.attrs->'source_records', '[]'::jsonb)) AS r) AS record_ids,
               v.attrs->>'description' AS description, v.attrs->>'when_stated' AS when_stated
        FROM working.candidate_event v
        JOIN gens ON v.source_raw_id = gens.gen_id::text
        WHERE v.source_raw_table = 'context.normalized_generation' AND v.review_state <> 'superseded'
        ORDER BY v.occurred_at NULLS LAST, v.id
        LIMIT {MAX_ROWS + 1}""",
        {"matter": matter, "export": export_key, "conv": conv},
    )
