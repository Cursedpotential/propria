"""build_timeline_generation_activity: the projection step of the Go extraction commit.

Byline: Claude Code · Opus 5.5 · 2026-09-25

No Temporal server, no database: the builder is replaced with a recorder, because the
contract under test is the seam — the Activity's registered name (the Go workflow schedules
it by that string), its payload shape (the Go ``flow.ProjectionRequest`` JSON) and its return
shape (``flow.ProjectionResult``). The builder itself is covered by
``tests/test_timeline_projection.py``.
"""

from __future__ import annotations

import subprocess
import sys
from contextlib import contextmanager

from temporalio.converter import DataConverter

from server.temporal import timeline_activities as ta
from server.timeline.models import GenerationResult


def test_activity_name_matches_the_go_workflow():
    from temporalio.activity import _Definition

    defn = _Definition.from_callable(ta.build_timeline_generation_activity)
    assert defn is not None
    assert defn.name == "build_timeline_generation_activity"


def test_go_projection_request_decodes_into_the_params_dataclass():
    converter = DataConverter.default
    # Exactly what flow.ProjectionRequest marshals to.
    payloads = converter.payload_converter.to_payloads(
        [{"collection_slug": "primary", "created_by": "extraction_commit:owner", "commit_id": "c-1"}]
    )
    (params,) = converter.payload_converter.from_payloads(payloads, [ta.BuildGenerationParams])
    assert params == ta.BuildGenerationParams("primary", "extraction_commit:owner", "c-1")


def test_activity_reuses_build_generation_and_returns_the_go_result_shape(monkeypatch):
    calls = []

    class _Engine:
        @contextmanager
        def begin(self):
            yield "conn"

    def fake_build(conn, *, collection_slug, created_by):
        calls.append((conn, collection_slug, created_by))
        return GenerationResult(generation_id="g-7", sequence=7, created=True, member_count=3)

    import server.timeline.db as db
    import server.timeline.generation as generation

    monkeypatch.setattr(db, "get_engine", lambda: _Engine())
    monkeypatch.setattr(generation, "build_generation", fake_build)
    result = ta.build_timeline_generation_activity(ta.BuildGenerationParams("", "  ", "c-9"))
    assert calls == [("conn", "primary", "timeline_projector")]
    assert result == {
        "generation_id": "g-7",
        "sequence": 7,
        "created": True,
        "member_count": 3,
        "skipped_unresolved_governed_members": [],
    }


def test_importing_the_module_reads_no_database_configuration():
    code = (
        "import sys; import server.temporal.timeline_activities; "
        "print('server.core.url' in sys.modules or 'server.timeline.db' in sys.modules)"
    )
    completed = subprocess.run([sys.executable, "-c", code], capture_output=True, text=True, check=True)
    assert completed.stdout.strip() == "False"
