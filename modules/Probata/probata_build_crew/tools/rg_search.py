"""Read-only ripgrep search tool for the planner crew.

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import subprocess

from crewai.tools import BaseTool
from pydantic import BaseModel, Field


class RgSearchInput(BaseModel):
    pattern: str = Field(..., description="Regex to search for (ripgrep syntax; case-insensitive when all lowercase)")
    path: str = Field(..., description="Absolute directory or file to search")
    glob: str | None = Field(None, description="Optional file glob filter, e.g. '*.go' or '*.sql'")
    max_results: int = Field(60, description="Maximum matching lines to return")


class RgSearchTool(BaseTool):
    name: str = "Search files with ripgrep"
    description: str = (
        "Read-only regex search over a directory or file. Returns 'path:line: text' lines. "
        "Use it to locate symbols, identifiers, table names, config keys or phrases BEFORE reading a file, "
        "so you read only the files that matter. Excludes .git, .venv, vendor, node_modules."
    )
    args_schema: type[BaseModel] = RgSearchInput

    def _run(self, pattern: str, path: str, glob: str | None = None, max_results: int = 60) -> str:
        cmd = [
            "rg", "-n", "--no-heading", "--color", "never", "-S", "--max-count", "20",
            "--glob", "!.git", "--glob", "!.venv", "--glob", "!vendor", "--glob", "!node_modules",
        ]
        if glob:
            cmd += ["--glob", glob]
        cmd += ["-e", pattern, path]
        try:
            out = subprocess.run(
                cmd, capture_output=True, text=True, timeout=60, encoding="utf-8", errors="replace"
            )
        except Exception as exc:  # noqa: BLE001
            return f"rg failed: {exc!r}"
        lines = out.stdout.splitlines()
        if not lines:
            err = out.stderr.strip()[:200]
            return f"No matches for {pattern!r} under {path}" + (f" (rg: {err})" if err else "")
        body = "\n".join(line[:300] for line in lines[:max_results])
        if len(lines) > max_results:
            body += f"\n... {len(lines) - max_results} more matching lines not shown (narrow the pattern or path)"
        return body
