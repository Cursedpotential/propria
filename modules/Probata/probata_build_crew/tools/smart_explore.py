"""Structural code exploration via the owner's smart-explore engine (tree-sitter, persistent DuckDB index).

Commands: search (ranked symbols + folded file views), outline (skeleton of one file), unfold (full source of one
symbol), refs (call sites), imports (a file's imports). Read-only.

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import os
import subprocess
import sys

from crewai.tools import BaseTool
from pydantic import BaseModel, Field

REPO_ROOT = os.environ.get("PROBATA_REPO_ROOT", "E:/AI_Workspace/Projects/the-platform-workspace/probata")
SKILL_DIR = os.environ.get("SMART_EXPLORE_DIR", os.path.expanduser("~/.agents/skills/smart-explore"))


class SmartExploreInput(BaseModel):
    command: str = Field(..., description="One of: search | outline | unfold | refs | imports")
    target: str = Field(..., description="search: the concept/symbol query · outline/imports: file path (relative to repo root) · unfold: file path · refs: symbol name")
    symbol: str | None = Field(None, description="unfold only: the symbol to expand from the file")
    path: str | None = Field(None, description="search/refs only: directory to scope to, relative to repo root (e.g. modules/engine). Default: modules/engine")
    max_results: int = Field(15, description="search only: maximum matching symbols")


class SmartExploreTool(BaseTool):
    name: str = "Structural code explorer (smart-explore)"
    description: str = (
        "Tree-sitter index of the code: 'search <concept>' returns ranked symbols (functions, types, methods) with "
        "file:line and a folded per-file view; 'outline <file>' gives a file's skeleton without its bodies; "
        "'unfold <file> <symbol>' returns one symbol's full source; 'refs <symbol>' lists call sites; "
        "'imports <file>' lists what a file imports. Prefer outline+unfold over reading whole files. "
        "The first call on a directory builds the index (up to a minute)."
    )
    args_schema: type[BaseModel] = SmartExploreInput

    def _run(self, command: str, target: str, symbol: str | None = None, path: str | None = None, max_results: int = 15) -> str:
        command = command.strip().lower()
        scope = path or "modules/engine"
        if command == "search":
            args = ["search", "--path", scope, "--max", str(max(1, min(int(max_results), 40))), target]
        elif command == "outline":
            args = ["outline", target]
        elif command == "unfold":
            if not symbol:
                return "unfold needs both target (file) and symbol"
            args = ["unfold", target, symbol]
        elif command == "refs":
            args = ["refs", target, "--path", scope]
        elif command == "imports":
            args = ["imports", "--file", target]
        else:
            return f"unknown command {command!r}; use search | outline | unfold | refs | imports"
        if sys.platform.startswith("win"):
            cmd = [os.path.join(SKILL_DIR, "se.cmd"), *args]
            shell = True
        else:
            cmd = ["bash", os.path.join(SKILL_DIR, "se"), *args]
            shell = False
        try:
            out = subprocess.run(cmd, cwd=REPO_ROOT, capture_output=True, text=True, timeout=600, shell=shell, encoding="utf-8", errors="replace")
        except Exception as exc:  # noqa: BLE001
            return f"smart-explore failed: {exc!r}"
        text = out.stdout.strip() or out.stderr.strip()
        if not text:
            return f"smart-explore {command} returned nothing for {target!r}"
        lines = text.splitlines()
        if len(lines) > 120:
            text = "\n".join(lines[:120]) + f"\n... {len(lines) - 120} more lines not shown"
        return text
