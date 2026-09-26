"""APScheduler factory. MemoryJobStore only — no live Postgres.

> _Byline: Grok · grok-4.6 · 2026-08-18_
legal_core.automation_job is the later PG jobstore (Type 1 HOLD).
The currency-flag-nudge job invents no docket dates and does not call PACER.
"""

from __future__ import annotations

from typing import Any

from apscheduler.jobstores.memory import MemoryJobStore
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.schedulers.base import STATE_RUNNING
from apscheduler.triggers.interval import IntervalTrigger

from legal_workspace.services.automation import get_store_dir
from legal_workspace.services.persist import append_jsonl

CURRENCY_NUDGE_JOB_ID = "currency-flag-nudge"

_scheduler: AsyncIOScheduler | None = None


def create_scheduler() -> AsyncIOScheduler:
    """AsyncIOScheduler with an in-memory jobstore. Does not open Postgres."""
    return AsyncIOScheduler(
        jobstores={"default": MemoryJobStore()},
        timezone="UTC",
    )


def currency_flag_nudge() -> None:
    """No-op court-facing. Optional court_safe=false audit if the store exists."""
    root = get_store_dir()
    if not root.is_dir():
        return
    append_jsonl(
        root / "automation.jsonl",
        {
            "job_id": CURRENCY_NUDGE_JOB_ID,
            "court_safe": False,
            "note": "remind owner to re-check cited authority currency; no docket invented",
        },
    )


def register_default_jobs(scheduler: AsyncIOScheduler) -> None:
    scheduler.add_job(
        currency_flag_nudge,
        trigger=IntervalTrigger(hours=24),
        id=CURRENCY_NUDGE_JOB_ID,
        name="currency flag nudge",
        replace_existing=True,
        coalesce=True,
        max_instances=1,
    )


def _ensure_scheduler() -> AsyncIOScheduler:
    global _scheduler
    if _scheduler is None or getattr(_scheduler, "_legal_os_shutdown", False):
        _scheduler = create_scheduler()
        register_default_jobs(_scheduler)
    return _scheduler


def start_scheduler() -> AsyncIOScheduler:
    """Idempotent start. Recreates after shutdown (APScheduler cannot restart)."""
    sched = _ensure_scheduler()
    if sched.state != STATE_RUNNING:
        sched.start()
    return sched


def shutdown_scheduler() -> None:
    """Idempotent stop. Safe if never started."""
    global _scheduler
    if _scheduler is None:
        return
    if _scheduler.running:
        _scheduler.shutdown(wait=False)
    _scheduler._legal_os_shutdown = True


def reset_scheduler() -> None:
    """Drop the singleton after shutdown. Tests only."""
    global _scheduler
    shutdown_scheduler()
    _scheduler = None


def list_jobs(scheduler: AsyncIOScheduler | None = None) -> list[dict[str, Any]]:
    sched = scheduler if scheduler is not None else _ensure_scheduler()
    rows: list[dict[str, Any]] = []
    for job in sched.get_jobs():
        next_run = getattr(job, "next_run_time", None)
        rows.append(
            {
                "id": job.id,
                "next_run_time": next_run.isoformat() if next_run is not None else None,
                "trigger": str(job.trigger),
            }
        )
    return rows
