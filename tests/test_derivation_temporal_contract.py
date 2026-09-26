"""Temporal-source contract for the PostgreSQL walk derivation.

Byline: Codex · GPT-5 · 2026-08-18
Byline amendment: Codex · gpt-6-astra · 2026-09-23 (walk schedule validation).
"""

from __future__ import annotations

import json
from datetime import datetime, timedelta, timezone

import pytest

from server.evidence.derivation import (
    _HINDSIGHT_SLICE_SQL,
    _SLICE_SQL,
    _canonical_slice,
    _compute_base_version,
    _validated_horizon_schedule,
    derive_walk,
)


class _Rows:
    def __init__(self, rows):
        self._rows = rows

    def fetchall(self):
        return self._rows


class _Conn:
    def __init__(self, rows):
        self.rows = rows
        self.sql = ""

    def execute(self, statement, params):
        self.sql = str(statement)
        assert params == {"cid": "primary"}
        return _Rows(self.rows)


def test_base_version_consumes_complete_version_input_view():
    conn = _Conn(
        [
            ("message_projection_route", "r1", {"projection_kind": "acquired_third_party"}),
            ("third_party_acquisition", "a1", {"acquired_at": "2025-01-01T00:00:00+00:00"}),
        ]
    )
    digest = _compute_base_version(conn, "primary")
    assert len(digest) == 64
    assert "vw_walk_base_version_input" in conn.sql
    assert "ORDER BY input_kind, input_key" in conn.sql


def test_slice_hash_keeps_source_and_realization_atoms_distinct():
    instant = datetime(2025, 1, 1, tzinfo=timezone.utc)
    payload = json.loads(
        _canonical_slice(
            [
                ("realization_event", "same-id", instant, instant, "discovered"),
                ("normalized_record", "same-id", instant, instant, "contemporaneous"),
            ],
            instant,
        )
    )
    assert [(row["atom_kind"], row["atom_id"]) for row in payload["slice"]] == [
        ("normalized_record", "same-id"),
        ("realization_event", "same-id"),
    ]


def test_as_lived_slice_fails_closed_and_hindsight_keeps_atoms_typed():
    assert "FROM working.vw_horizon_atom" in _SLICE_SQL
    assert "visible_from IS NOT NULL" in _SLICE_SQL
    assert "visible_from <= :horizon" in _SLICE_SQL
    assert "FROM working.vw_horizon_atom" in _HINDSIGHT_SLICE_SQL


def test_walk_schedule_preserves_strictly_advancing_utc_cutoffs():
    first = datetime(2025, 1, 1, tzinfo=timezone.utc)
    second = datetime(2024, 12, 31, 20, tzinfo=timezone(timedelta(hours=-5)))
    assert _validated_horizon_schedule("ignorant", [first, second], None) == [
        first,
        datetime(2025, 1, 1, 1, tzinfo=timezone.utc),
    ]


@pytest.mark.parametrize(
    ("policy", "schedule", "ceiling", "message"),
    [
        ("ignorant", [datetime(2025, 1, 1)], None, "timezone-aware"),
        (
            "ignorant",
            [datetime(2025, 1, 2, tzinfo=timezone.utc), datetime(2025, 1, 1, tzinfo=timezone.utc)],
            None,
            "advance strictly",
        ),
        (
            "ignorant",
            [datetime(2025, 1, 1, tzinfo=timezone.utc)] * 2,
            None,
            "advance strictly",
        ),
        (
            "custom",
            [datetime(2025, 1, 2, tzinfo=timezone.utc)],
            datetime(2025, 1, 1, tzinfo=timezone.utc),
            "exceeds its custom ceiling",
        ),
        (
            "custom",
            [datetime(2025, 1, 1, tzinfo=timezone.utc)],
            datetime(2025, 1, 2),
            "timezone-aware",
        ),
        ("hindsight", [datetime(2025, 1, 1, tzinfo=timezone.utc)], None, "cannot have"),
    ],
)
def test_invalid_walk_schedule_is_rejected_before_database_use(policy, schedule, ceiling, message):
    with pytest.raises(ValueError, match=message):
        derive_walk(
            agent_id="test-agent",
            horizon_policy=policy,
            horizon_schedule=schedule,
            custom_horizon_ceiling=ceiling,
            connection=object(),
        )
