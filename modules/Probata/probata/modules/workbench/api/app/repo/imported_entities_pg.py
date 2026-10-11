"""Read-only Imported entities queries over the platform projection.

Byline: Codex · GPT-6.1-Sol · 2026-10-05.
Provenance: behavior-preserving split of the 2026-10-02 Imported read facade.

SQL, scope and result shapes are unchanged. Connection and error handling stay
in imported_pg; each query resolves that facade at call time for shared patches.
"""

from __future__ import annotations

from typing import Any

from app.repo import imported_pg as pg


def people() -> list[dict[str, Any]]:
    """Registry identifiers (the one identity store) so phone numbers show as names.

    Includes placeholder people (role_in_case 'unknown', verification_state 'proposed'); a merged
    person's identifiers have moved to the person it was merged into.
    """
    return pg._query(
        """SELECT e.id::text AS entity_id, e.display_name::text AS display_name,
                  coalesce(p.short_name, e.display_name::text) AS person, p.role_in_case, p.verification_state,
                  coalesce(a.normalized, registry.norm_identifier(a.alias_text::text)) AS identifier,
                  coalesce(a.alias_kind, 'other') AS kind, a.status AS alias_status,
                  a.alias_text::text AS alias_text_raw
           FROM registry.entity_alias a
           JOIN registry.entity e ON e.id = a.entity_id
           JOIN registry.person p ON p.id = e.id
           WHERE e.merged_into_id IS NULL AND a.status <> 'retired'""",
        {},
    )


def numbers_activity(matter: str) -> list[dict[str, Any]]:
    """Every phone-like participant of the imported records, with how often it appears."""
    return pg._query(
        f"""WITH {pg._SV},
        digits AS (
          SELECT n.id, n.record_type, n.occurred_at, regexp_replace(p.value->>'identifier', '[^0-9]', '', 'g') AS d
          FROM context.normalized_record_identity n
          JOIN sv ON sv.id = n.source_version_id
          CROSS JOIN LATERAL jsonb_array_elements(n.normalized_payload->'participants') AS p(value)
          WHERE p.value->>'identifier' IS NOT NULL AND p.value->>'identifier' <> 'self'
        )
        SELECT right(d, 10) AS number, record_type, count(DISTINCT id) AS n, max(occurred_at) AS last_at
        FROM digits
        WHERE length(d) = 10 OR (length(d) = 11 AND left(d, 1) = '1')
        GROUP BY 1, 2""",
        {"matter": matter},
    )


def entity_activity() -> list[dict[str, Any]]:
    """Calls and messages already linked to each registry person, from the working tables.

    Cheap on purpose (four small grouped scans, no JSON): the unnamed-numbers list ranks by this.
    """
    return pg._query(
        """SELECT entity_id::text AS entity_id, sum(calls)::bigint AS calls, sum(msgs)::bigint AS msgs, max(last_at) AS last_at
           FROM (
             SELECT from_entity_id AS entity_id, count(*) AS calls, 0 AS msgs, max(started_at) AS last_at
               FROM working.call_log WHERE from_entity_id IS NOT NULL GROUP BY 1
             UNION ALL
             SELECT to_entity_id, count(*), 0, max(started_at)
               FROM working.call_log WHERE to_entity_id IS NOT NULL GROUP BY 1
             UNION ALL
             SELECT entity_id, 0, count(DISTINCT message_id), NULL::timestamptz
               FROM working.message_participant WHERE entity_id IS NOT NULL GROUP BY 1
             UNION ALL
             SELECT entity_id, 0, count(DISTINCT message_id), NULL::timestamptz
               FROM working.third_party_message_participant WHERE entity_id IS NOT NULL GROUP BY 1
           ) u GROUP BY 1""",
        {},
    )


def working_unlinked_numbers() -> list[dict[str, Any]]:
    """Numbers on working.call_log / message participants whose entity column is still NULL."""
    return pg._query(
        """SELECT number, sum(n)::bigint AS n FROM (
             SELECT right(regexp_replace(coalesce(nullif(from_e164, ''), from_raw, ''), '[^0-9]', '', 'g'), 10) AS number, count(*) AS n
               FROM working.call_log WHERE from_entity_id IS NULL GROUP BY 1
             UNION ALL
             SELECT right(regexp_replace(coalesce(nullif(to_e164, ''), to_raw, ''), '[^0-9]', '', 'g'), 10), count(*)
               FROM working.call_log WHERE to_entity_id IS NULL GROUP BY 1
             UNION ALL
             SELECT right(regexp_replace(coalesce(nullif(participant_e164, ''), participant_raw, ''), '[^0-9]', '', 'g'), 10), count(*)
               FROM working.message_participant WHERE entity_id IS NULL GROUP BY 1
             UNION ALL
             SELECT right(regexp_replace(coalesce(nullif(participant_e164, ''), participant_raw, ''), '[^0-9]', '', 'g'), 10), count(*)
               FROM working.third_party_message_participant WHERE entity_id IS NULL GROUP BY 1
           ) u WHERE length(number) = 10 GROUP BY number""",
        {},
    )
