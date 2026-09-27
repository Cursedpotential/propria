"""Semantic code search over the probata repository via CocoIndex Code (`ccc`).

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import json
import os
import shutil
import subprocess

from crewai.tools import BaseTool
from pydantic import BaseModel, Field

REPO_ROOT = os.environ.get("PROBATA_REPO_ROOT", "E:/AI_Workspace/Projects/the-platform-workspace/probata")


def _ccc_exe() -> str:
    return shutil.which("ccc") or os.path.expanduser("~/.local/bin/ccc.exe")


class CccSearchInput(BaseModel):
    query: str = Field(..., description="Natural-language description of what you are looking for, e.g. 'where the proffer worker registers Temporal activities'")
    limit: int = Field(8, description="Maximum results (1-25)")
    path_glob: str | None = Field(None, description="Optional file path glob filter, e.g. 'modules/engine/**' or 'docs/**/*.md'")
    lang: str | None = Field(None, description="Optional language filter: go, python, sql, markdown, yaml, json, typescript")


class CccSearchTool(BaseTool):
    name: str = "Semantic code search (ccc)"
    description: str = (
        "Meaning-based search over the maintained CocoIndex index of the probata repository (48k chunks across markdown, "
        "python, sql, go, json, yaml, tsx). Returns the files and chunk excerpts whose MEANING matches the query, even when "
        "the words differ - use it to discover where a concept lives, then use ripgrep for exact identifiers and the file "
        "tool to read. Run several phrasings (synonyms, the D-number, the table name) for exhaustive coverage."
    )
    args_schema: type[BaseModel] = CccSearchInput

    def _run(self, query: str, limit: int = 8, path_glob: str | None = None, lang: str | None = None) -> str:
        limit = max(1, min(int(limit), 25))
        cmd = [_ccc_exe(), "search", query, "--limit", str(limit), "--json"]
        if path_glob:
            cmd += ["--path", path_glob]
        if lang:
            cmd += ["--lang", lang]
        try:
            out = subprocess.run(cmd, cwd=REPO_ROOT, capture_output=True, text=True, timeout=180, encoding="utf-8", errors="replace")
        except Exception as exc:  # noqa: BLE001
            return f"ccc failed: {exc!r}"
        raw = out.stdout.strip()
        if not raw:
            return f"ccc returned nothing (stderr: {out.stderr.strip()[:300]})"
        try:
            data = json.loads(raw)
        except json.JSONDecodeError:
            return raw[:4000]
        results = data.get("results") or []
        if not results:
            return f"No semantic matches for {query!r}" + (f" (stderr: {out.stderr.strip()[:200]})" if out.stderr.strip() else "")
        lines = [f"{len(results)} semantic matches for {query!r}:"]
        for r in results:
            loc = ""
            for key in ("start_line", "line_start", "start", "line"):
                if key in r:
                    loc = f":{r[key]}"
                    break
            score = r.get("score")
            score_s = f" score={score:.3f}" if isinstance(score, (int, float)) else ""
            content = " ".join(str(r.get("content", "")).split())[:320]
            lines.append(f"- {r.get('file_path')}{loc} ({r.get('language')}){score_s}\n    {content}")
        return "\n".join(lines)
