"""Non-recursive directory listing tool for the planner crew.

crewai_tools.DirectoryReadTool lists recursively, which on a Go module with a vendored tree
returns tens of thousands of paths and blows the model's context. This tool lists ONE level.

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import os

from crewai.tools import BaseTool
from pydantic import BaseModel, Field


class ListDirInput(BaseModel):
    path: str = Field(..., description="Absolute directory to list (one level only, never recursive)")
    max_entries: int = Field(200, description="Maximum entries to return")


class ListDirTool(BaseTool):
    name: str = "List one directory level"
    description: str = (
        "Lists the immediate children of ONE directory (no recursion): 'd <name>/' for directories and "
        "'f <size> <name>' for files. Hidden entries and .venv/vendor/node_modules/.git are marked but not expanded. "
        "Use it to orient, then use the ripgrep tool to find what to read."
    )
    args_schema: type[BaseModel] = ListDirInput

    def _run(self, path: str, max_entries: int = 200) -> str:
        if not os.path.isdir(path):
            return f"NOT A DIRECTORY: {path}"
        try:
            names = sorted(os.listdir(path), key=lambda n: (not os.path.isdir(os.path.join(path, n)), n.lower()))
        except OSError as exc:
            return f"list failed: {exc!r}"
        lines = []
        for name in names[:max_entries]:
            full = os.path.join(path, name)
            if os.path.isdir(full):
                note = "  (large tree, do not list recursively)" if name in {".venv", "vendor", "node_modules", ".git"} else ""
                lines.append(f"d {name}/{note}")
            else:
                try:
                    size = os.path.getsize(full)
                except OSError:
                    size = -1
                lines.append(f"f {size:>9} {name}")
        if len(names) > max_entries:
            lines.append(f"... {len(names) - max_entries} more entries not shown")
        return f"{path}\n" + "\n".join(lines)
