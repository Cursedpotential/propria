"""Read-only catalog queries for conversations with people: registry, bout sets, label passes, bouts, day messages.

Byline: Claude Code · Opus 5.5 · 2026-09-24. Tables are the Consignatio catalog's `raw_duck.msg_*_20260924` (messages
with people, renamed from chat_* on 2026-09-24) and `comm_events_20260918` for message text. Uses the Intake discovery
read-only connection (metabase_ro, read-only transaction, 4 s statement timeout). Every value is a bound parameter.
"""
from __future__ import annotations

from typing import Any

from app.repo.intake_discovery import DiscoveryError, _query

REGISTRY = "raw_duck.msg_conversation_registry_20260924"
BOUTS = "raw_duck.msg_bouts_20260924"
BOUT_MESSAGES = "raw_duck.msg_bout_messages_20260924"
LABELS = "raw_duck.msg_bout_labels_20260924"
FILES = "raw_duck.msg_files_20260924"
EVENTS = "raw_duck.comm_events_20260918"
# The bout builder cut days in America/Detroit (every rule set so far is "*-detroit"); message times use the same zone.
LOCAL_TZ = "America/Detroit"


def conversations() -> list[dict[str, Any]]:
    """Every registered conversation with its bout sets (newest build first) and its label passes."""
    return _query(f"""
        SELECT r.conv_key, r.description, r.custody_party, r.source_format,
          COALESCE((SELECT json_agg(json_build_object('rules', s.bout_rules, 'bouts', s.n, 'first_day', s.first_day,
                                                      'last_day', s.last_day) ORDER BY s.built_at DESC)
                    FROM (SELECT bout_rules, count(*) AS n, min(day) AS first_day, max(day) AS last_day,
                                 max(built_at) AS built_at
                          FROM {BOUTS} b WHERE b.source = r.conv_key GROUP BY bout_rules) s), '[]') AS bout_sets,
          COALESCE((SELECT json_agg(json_build_object('pass', p.pass, 'model', p.model, 'labelled', p.n,
                                                      'rules', p.bout_rules) ORDER BY p.labelled_at DESC)
                    FROM (SELECT l.pass, b.bout_rules, max(l.model) AS model, count(*) AS n,
                                 max(l.labelled_at) AS labelled_at
                          FROM {LABELS} l JOIN {BOUTS} b ON b.bout_id = l.bout_id
                          WHERE b.source = r.conv_key AND l.ok GROUP BY l.pass, b.bout_rules) p), '[]') AS label_passes
        FROM {REGISTRY} r ORDER BY r.conv_key""", ())


def conversation(conv_key: str) -> dict[str, Any]:
    for row in conversations():
        if row["conv_key"] == conv_key:
            return row
    raise DiscoveryError("Unknown conversation", 404)


def bouts(conv_key: str, rules: str, label_pass: str | None) -> list[dict[str, Any]]:
    """Every bout of one bout set, oldest first, with messages per sender and the chosen pass's raw label output."""
    return _query(f"""
        SELECT b.bout_id, b.day, to_char(b.start_local, 'HH24:MI') AS start, to_char(b.end_local, 'HH24:MI') AS "end",
               b.n_messages,
               (SELECT jsonb_object_agg(t.who, t.n)
                  FROM (SELECT m.who, count(*) AS n FROM {BOUT_MESSAGES} m WHERE m.bout_id = b.bout_id
                        GROUP BY m.who) t) AS senders,
               l.ok AS label_ok, l.output AS label
        FROM {BOUTS} b
        LEFT JOIN {LABELS} l ON l.bout_id = b.bout_id AND l.pass = %s AND l.ok
        WHERE b.source = %s AND b.bout_rules = %s
        ORDER BY b.start_local, b.bout_id""", (label_pass, conv_key, rules))


def day_messages(conv_key: str, rules: str, day: str) -> list[dict[str, Any]]:
    """The messages of every bout on one day: bout, position, local time, sender, text and source files."""
    return _query(f"""
        SELECT m.bout_id, m.ordinal, to_char(m.ts_utc AT TIME ZONE %s, 'HH24:MI') AS time, m.who,
               e.body, COALESCE(f.files, ARRAY[]::text[]) AS files
        FROM {BOUTS} b
        JOIN {BOUT_MESSAGES} m ON m.bout_id = b.bout_id
        LEFT JOIN {EVENTS} e ON e.dedup_key = m.dedup_key
        LEFT JOIN {FILES} f ON f.representative_key = m.dedup_key
        WHERE b.source = %s AND b.bout_rules = %s AND b.day = %s::date
        ORDER BY b.start_local, m.bout_id, m.ordinal""", (LOCAL_TZ, conv_key, rules, day))
