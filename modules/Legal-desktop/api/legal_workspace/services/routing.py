"""Owner-editable page catalog. Swap chat/agent backends without a rewrite.

> _Byline: Grok · grok-4.6 · 2026-08-18_
Load order: LEGAL_WORKSPACE_ROUTING_FILE, then data/workspace/routing.json,
then repo config/routing.json. Writes go to the overlay (workspace file)
unless an explicit path is passed. No tailnet IPs. No secrets.

Public contract is surfaces[] — path + English label + help + group.
There is no ticker / mnemonic / cmd field.
"""

from __future__ import annotations

import os
from pathlib import Path
from typing import Any

from pydantic import BaseModel, Field

from legal_workspace.services.persist import atomic_write_json, default_store_dir, read_json

_REPO_DEFAULT = Path(__file__).resolve().parents[3] / "config" / "routing.json"


class ChatRouting(BaseModel):
    backend: str = "legal-api-agent-runs"
    next_path: str = "/api/chat"
    run_path: str = "/v1/agent-runs"
    confidential_path: str = "/v1/gateway:invoke"
    default_confidential_model: str = "ollama-cloud"


class AgentBinding(BaseModel):
    backend: str = "legal-api"
    model: str = "unevaluated-manual"
    surface: str = "Home"


class SurfaceBinding(BaseModel):
    path: str
    label: str
    group: str
    help: str = ""
    advanced: bool = False


class RoutingTable(BaseModel):
    version: int = 1
    framework: str = "next-fastapi"
    notes: str = ""
    chat: ChatRouting = Field(default_factory=ChatRouting)
    agents: dict[str, AgentBinding] = Field(default_factory=dict)
    role_keywords: dict[str, list[str]] = Field(default_factory=dict)
    surfaces: list[SurfaceBinding] = Field(default_factory=list)


def default_routing_path() -> Path:
    env = os.environ.get("LEGAL_WORKSPACE_ROUTING_FILE", "").strip()
    if env:
        return Path(env)
    overlay = default_store_dir() / "routing.json"
    if overlay.is_file():
        return overlay
    return _REPO_DEFAULT


def overlay_path() -> Path:
    env = os.environ.get("LEGAL_WORKSPACE_ROUTING_FILE", "").strip()
    if env:
        return Path(env)
    return default_store_dir() / "routing.json"


def load_routing(path: Path | None = None) -> RoutingTable:
    target = path if path is not None else default_routing_path()
    raw = read_json(target)
    if raw is None and target != _REPO_DEFAULT:
        raw = read_json(_REPO_DEFAULT)
    if raw is None:
        return RoutingTable()
    return RoutingTable.model_validate(raw)


def save_routing(table: RoutingTable, path: Path | None = None) -> Path:
    target = path if path is not None else overlay_path()
    atomic_write_json(target, table.model_dump(mode="json"))
    return target


def model_for_role(role: str, requested: str, table: RoutingTable | None = None) -> str:
    if requested and requested != "unevaluated-manual":
        return requested
    snap = table if table is not None else load_routing()
    binding = snap.agents.get(role)
    return binding.model if binding is not None else requested


def surface_for_role(role: str, table: RoutingTable | None = None) -> str:
    snap = table if table is not None else load_routing()
    binding = snap.agents.get(role)
    return binding.surface if binding is not None else "Home"


def path_for_query(query: str, table: RoutingTable | None = None) -> str | None:
    raw = query.strip()
    if not raw:
        return None
    needle = raw.lower()
    snap = table if table is not None else load_routing()
    for item in snap.surfaces:
        if item.path.rstrip("/") == raw.rstrip("/"):
            return item.path
        label = item.label.strip().lower()
        if needle == label or label.startswith(needle):
            return item.path
    return None


def public_routing(table: RoutingTable | None = None) -> dict[str, Any]:
    snap = table if table is not None else load_routing()
    return snap.model_dump(mode="json")
