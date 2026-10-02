"""Case Bible catalog reads for the Case page: what the catalog holds per identifier.

Byline: Claude Code · Opus 5.5 · 2026-10-01

Read-only, over the same `metabase_ro` connection Sources uses
(`app.repo.intake_discovery._query`: read-only transaction, 4 s statement
timeout). Identities are never read or written here: they live in Probata
registry and reach this module as parameters. The counts table is built by
`Consignatio/casebible/tools/identity_event_counts_20261001.sql`; the event
lookups use its two expression indexes on `raw_duck.comm_events_20260918`.
"""

from __future__ import annotations

from typing import Any

from app.repo import intake_discovery

COUNTS = "raw_duck.identity_event_counts_20261001"
EVENTS = "raw_duck.comm_events_20260918"
STORE = {"store": "casebible", "table": COUNTS, "events": EVENTS, "snapshot": "2026-09-18"}
MATCH_ON = {"counterparty_phone", "sender"}

CatalogError = intake_discovery.DiscoveryError


def configured() -> bool:
    return intake_discovery.configured()


def counts(identifiers: list[str]) -> list[dict[str, Any]]:
    """Per-identifier counts for these normalized identifiers."""
    wanted = sorted({value for value in identifiers if value})[:2000]
    if not wanted:
        return []
    return intake_discovery._query(
        f"""SELECT identifier, match_on, event_kind, events, conversations, first_at, last_at, sources
        FROM {COUNTS} WHERE identifier = ANY(%s::text[]) ORDER BY identifier, match_on, event_kind""",
        (wanted,),
    )


def unknowns(known: list[str], *, kind: str = "phone", limit: int = 50) -> list[dict[str, Any]]:
    """Identifiers the catalog saw most often that no registry person carries.

    `known` is every normalized identifier the registry carries or the owner
    dismissed; it arrives from the engine's registry read, so the catalog
    never decides who someone is.
    """
    phone = kind == "phone"
    rows = intake_discovery._query(
        f"""SELECT identifier, sum(events)::bigint AS events, min(first_at) AS first_at, max(last_at) AS last_at,
               array_agg(DISTINCT match_on ORDER BY match_on) AS match_on
        FROM {COUNTS}
        WHERE NOT (identifier = ANY(%s::text[])) AND (identifier ~ '^[0-9]+$') = %s
        GROUP BY identifier ORDER BY sum(events) DESC, identifier LIMIT %s""",
        (sorted(set(known)), phone, max(1, min(limit, 200))),
    )
    for row in rows:
        row["kind"] = "phone" if phone else "name"
    return rows


def events(identifier: str, match_on: str, *, before: str | None = None, limit: int = 50) -> dict[str, Any]:
    """The catalog events behind one count, newest first, one page at a time."""
    if match_on not in MATCH_ON:
        raise CatalogError("match_on must be counterparty_phone or sender", 422)
    if not identifier or len(identifier) > 512 or "\x00" in identifier:
        raise CatalogError("Invalid identifier", 422)
    column = "counterparty_phone" if match_on == "counterparty_phone" else "sender"
    limit = max(1, min(limit, 200))
    rows = intake_discovery._query(
        f"""SELECT dedup_key, event_ts_utc, event_kind, source_format, direction, sender, recipients,
               contact_name, counterparty_phone, conversation_title, left(body, 600) AS body, n_sources
        FROM {EVENTS}
        WHERE raw_duck.norm_phone({column}) = %s AND (%s::timestamptz IS NULL OR event_ts_utc < %s::timestamptz)
        ORDER BY event_ts_utc DESC NULLS LAST LIMIT %s""",
        (identifier, before, before, limit + 1),
    )
    more = len(rows) > limit
    rows = rows[:limit]
    return {
        **STORE,
        "identifier": identifier,
        "match_on": match_on,
        "items": rows,
        "has_more": more,
        "next_before": rows[-1]["event_ts_utc"].isoformat() if more and rows[-1].get("event_ts_utc") else None,
    }
