"""The Activities, their names and the Go contract. Byline: Claude Code · Sonnet 5.5 · 2026-10-02"""

import re
from pathlib import Path

import pytest
from temporalio.exceptions import ApplicationError
from temporalio.testing import ActivityEnvironment

from casebible_index import ledger
from casebible_index import temporal_worker as tw

GO = (
    Path(__file__).resolve().parents[4]
    / "Probata"
    / "probata"
    / "modules"
    / "engine"
    / "superindex"
    / "workflow.go"
)


def test_activity_names_match_the_go_workflow_constants():
    if not GO.is_file():
        pytest.skip("Go engine not in this checkout")
    source = GO.read_text(encoding="utf-8")
    go_names = set(re.findall(r'\w+ActivityName\s*=\s*"(superindex_[a-z_]+)"', source))
    assert go_names == set(tw.ACTIVITY_NAMES)
    assert (
        re.search(r'TaskQueue\s*=\s*"superindex"', source) and tw.TASK_QUEUE_DEFAULT == "superindex"
    )


def test_every_activity_is_registered_under_its_name():
    registered = {fn.__temporal_activity_definition.name for fn in tw.ACTIVITIES}
    assert registered == set(tw.ACTIVITY_NAMES)


@pytest.mark.asyncio
async def test_commit_activity_runs_the_stage_and_advances_the_watermark(
    tmp_path: Path, monkeypatch
):
    monkeypatch.setenv("CASEBIBLE_OUTPUT_DIR", str(tmp_path / "out"))
    monkeypatch.setenv("CASEBIBLE_SOURCE_DIR", str(tmp_path / "src"))
    monkeypatch.setenv("INTAKE_SOURCE_MODE", "catalog")
    monkeypatch.setenv("INTAKE_VAULT_BUCKET", "salem-data")
    monkeypatch.setenv("INTAKE_LOCK_DIR", str(tmp_path / "locks"))
    watermark = [
        {"provider": "b2", "bucket": "salem-data", "listed_at": "t", "objects": 1, "bytes": 2}
    ]
    env = ActivityEnvironment()
    result = await env.run(
        tw.superindex_commit_activity,
        tw.StageRequest(cycle_id="20261003T000000Z", params={"watermark": watermark}),
    )
    assert result["committed"] is True
    assert ledger.last_commit(tmp_path / "out")["watermark"] == watermark


@pytest.mark.asyncio
async def test_a_request_without_a_cycle_id_is_not_retried():
    with pytest.raises(ApplicationError) as raised:
        await ActivityEnvironment().run(
            tw.superindex_commit_activity, tw.StageRequest(cycle_id=" ")
        )
    assert raised.value.non_retryable


@pytest.mark.asyncio
async def test_a_configuration_error_is_not_retried(monkeypatch, tmp_path: Path):
    monkeypatch.setenv("CASEBIBLE_OUTPUT_DIR", str(tmp_path / "out"))
    monkeypatch.setenv("CASEBIBLE_SOURCE_DIR", str(tmp_path / "src"))
    monkeypatch.setenv("INTAKE_SOURCE_MODE", "catalog")
    monkeypatch.setenv("INTAKE_VAULT_BUCKET", "salem-data")
    monkeypatch.setenv("INTAKE_LOCK_DIR", str(tmp_path / "locks"))
    monkeypatch.delenv("INTAKE_CATALOG_DSN", raising=False)
    with pytest.raises(ApplicationError) as raised:
        await ActivityEnvironment().run(
            tw.superindex_discover_activity, tw.StageRequest(cycle_id="20261003T000000Z")
        )
    assert raised.value.non_retryable and "INTAKE_CATALOG_DSN" in str(raised.value)
