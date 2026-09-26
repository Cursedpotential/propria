"""APScheduler automations are file-only. No live PG, PACER, or n8n.

> _Byline: Grok · grok-4.6 · 2026-08-18_
"""

from __future__ import annotations

from apscheduler.jobstores.memory import MemoryJobStore
from fastapi import FastAPI
from fastapi.testclient import TestClient

from legal_workspace.api.automation_routes import router
from legal_workspace.services.automation import set_store_dir
from legal_workspace.services.automation.bus import reset_bus
from legal_workspace.services.automation.runner import FOC_HEARING_PREP_STEPS, run_playbook
from legal_workspace.services.automation.scheduler import (
    CURRENCY_NUDGE_JOB_ID,
    create_scheduler,
    currency_flag_nudge,
    list_jobs,
    register_default_jobs,
    reset_scheduler,
    shutdown_scheduler,
)


def _cleanup() -> None:
    set_store_dir(None)
    reset_bus()
    reset_scheduler()


def test_scheduler_factory_does_not_require_postgres() -> None:
    sched = create_scheduler()
    assert sched.running is False
    store = sched._lookup_jobstore("default")
    assert isinstance(store, MemoryJobStore)


def test_default_jobs_include_currency_flag_nudge() -> None:
    sched = create_scheduler()
    register_default_jobs(sched)
    jobs = list_jobs(sched)
    assert any(row["id"] == CURRENCY_NUDGE_JOB_ID for row in jobs)
    assert any(row["id"] == "currency-flag-nudge" for row in jobs)
    for row in jobs:
        assert {"id", "next_run_time", "trigger"} <= set(row)


def test_shutdown_without_start_is_idempotent() -> None:
    reset_scheduler()
    shutdown_scheduler()
    shutdown_scheduler()


def test_runner_foc_hearing_prep_sequential_labels(tmp_path) -> None:
    set_store_dir(tmp_path)
    try:
        result = run_playbook("foc-hearing-prep")
        assert result["ok"] is True
        assert result["court_safe"] is False
        names = [row["step"] for row in result["steps"]]
        assert names == list(FOC_HEARING_PREP_STEPS)
        assert names == ["confirm-clerk-docket", "review-factor-j", "build-release"]
        assert all(row["passed"] and row["kind"] == "label" for row in result["steps"])
        assert "hearing_date" not in result
        log = (tmp_path / "playbook_runs.jsonl").read_text(encoding="utf-8")
        assert "foc-hearing-prep" in log
        assert "confirm-clerk-docket" in log
        assert "hearing_date" not in log
        snapshot = (tmp_path / "last_playbook_run.json").read_text(encoding="utf-8")
        assert result["run_id"] in snapshot
        events = (tmp_path / "automation_events.jsonl").read_text(encoding="utf-8")
        assert "job_complete" in events
    finally:
        _cleanup()


def test_currency_nudge_writes_court_unsafe_audit(tmp_path) -> None:
    set_store_dir(tmp_path)
    try:
        currency_flag_nudge()
        log = (tmp_path / "automation.jsonl").read_text(encoding="utf-8")
        assert "currency-flag-nudge" in log
        assert '"court_safe": false' in log
        assert "hearing_date" not in log
    finally:
        _cleanup()


def test_http_jobs_and_playbook_run(tmp_path) -> None:
    set_store_dir(tmp_path)
    reset_scheduler()
    try:
        app = FastAPI()
        app.include_router(router)
        client = TestClient(app)
        jobs = client.get("/v1/automations/jobs")
        assert jobs.status_code == 200
        assert any(row["id"] == "currency-flag-nudge" for row in jobs.json())
        books = client.get("/v1/automations/playbooks")
        assert books.status_code == 200
        assert any(row["playbook_id"] == "foc-hearing-prep" for row in books.json())
        posted = client.post("/v1/automations/playbooks/foc-hearing-prep:run")
        assert posted.status_code == 200, posted.text
        body = posted.json()
        assert body["ok"] is True
        assert body["court_safe"] is False
        assert [row["step"] for row in body["steps"]] == list(FOC_HEARING_PREP_STEPS)
        assert all(row["passed"] for row in body["steps"])
        assert (tmp_path / "playbook_runs.jsonl").is_file()
        assert (tmp_path / "automation_events.jsonl").is_file()
        missing = client.post("/v1/automations/playbooks/not-a-playbook:run")
        assert missing.status_code == 404
    finally:
        _cleanup()
