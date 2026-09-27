"""Structural (AST-aware) grep by example via CocoIndex Code (`ccc grep`).

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import os
import shutil
import subprocess

from crewai.tools import BaseTool
from pydantic import BaseModel, Field

REPO_ROOT = os.environ.get("PROBATA_REPO_ROOT", "E:/AI_Workspace/Projects/the-platform-workspace/probata")


class CccGrepInput(BaseModel):
    pattern: str = Field(..., description=r"By-example structural pattern with \NAME metavariables, e.g. 'func \NAME(\(ARGS*\)) error' or 'RegisterActivity(\(ARGS*\))'")
    path: str = Field(".", description="File or directory to search, relative to the repository root (default: whole repo)")
    lang: str | None = Field(None, description="Only match files of this language, e.g. go, python, sql")
    max_lines: int = Field(60, description="Maximum output lines")


class CccGrepTool(BaseTool):
    name: str = "Structural code grep (ccc grep)"
    description: str = (
        "Syntax-aware grep by example over the probata repository: the pattern is real code with \\NAME metavariables "
        "and matches code SHAPES regardless of whitespace or variable names. Use it to find every call site or definition "
        "of a shape (e.g. every activity registration) when ripgrep on a string would miss variants."
    )
    args_schema: type[BaseModel] = CccGrepInput

    def _run(self, pattern: str, path: str = ".", lang: str | None = None, max_lines: int = 60) -> str:
        exe = shutil.which("ccc") or os.path.expanduser("~/.local/bin/ccc.exe")
        cmd = [exe, "grep", pattern, path, "--no-color"]
        if lang:
            cmd += ["--lang", lang]
        try:
            out = subprocess.run(cmd, cwd=REPO_ROOT, capture_output=True, text=True, timeout=180, encoding="utf-8", errors="replace")
        except Exception as exc:  # noqa: BLE001
            return f"ccc grep failed: {exc!r}"
        lines = [line for line in out.stdout.splitlines() if line.strip()]
        if not lines:
            return f"No structural matches for {pattern!r} under {path}" + (f" (stderr: {out.stderr.strip()[:300]})" if out.stderr.strip() else "")
        body = "\n".join(line[:300] for line in lines[:max_lines])
        if len(lines) > max_lines:
            body += f"\n... {len(lines) - max_lines} more lines not shown (narrow the path or pattern)"
        return body
