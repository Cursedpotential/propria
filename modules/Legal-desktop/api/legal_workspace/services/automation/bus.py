"""Tiny in-process publish list. No Redis.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Persists action names only: package_imported, deadline_changed, job_complete.
"""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from legal_workspace.services.automation import get_store_dir
from legal_workspace.services.persist import append_jsonl

BUS_ACTIONS = frozenset({"package_imported", "deadline_changed", "job_complete"})

_listeners: list[Callable[[dict[str, Any]], None]] = []


def reset_bus() -> None:
    _listeners.clear()


def subscribe(listener: Callable[[dict[str, Any]], None]) -> None:
    _listeners.append(listener)


def publish(action: str) -> dict[str, Any]:
    if action not in BUS_ACTIONS:
        raise ValueError(f"unknown automation action: {action}")
    event = {"action": action, "court_safe": False}
    append_jsonl(get_store_dir() / "automation_events.jsonl", event)
    for listener in list(_listeners):
        listener(event)
    return event
