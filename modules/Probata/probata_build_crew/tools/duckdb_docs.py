"""Full-text search over the cached DuckDB documentation index (mirrors duckdb-skills `duckdb-docs`).

Index: https://duckdb.org/data/docs-search.duckdb cached at ~/.duckdb-skills/cache/duckdb-docs.duckdb
(table docs_chunks; FTS index fts_main_docs_chunks). Read-only.

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import os
import urllib.request

import duckdb
from crewai.tools import BaseTool
from pydantic import BaseModel, Field

CACHE = os.path.expanduser("~/.duckdb-skills/cache/duckdb-docs.duckdb")
URL = "https://duckdb.org/data/docs-search.duckdb"


class DuckDbDocsInput(BaseModel):
    query: str = Field(..., description="Keywords or a question, e.g. 'read_xml xml extension options' or 'regexp_extract_all'")
    version: str = Field("lts", description="'lts' (default), 'current' (nightly docs) or 'blog'")
    limit: int = Field(5, description="Maximum chunks to return (1-10)")


class DuckDbDocsTool(BaseTool):
    name: str = "DuckDB documentation search"
    description: str = (
        "BM25 full-text search over the official DuckDB documentation and blog (offline cached index). Use it before "
        "designing any DuckDB SQL, extension usage (read_xml/webbed, httpfs, json), or pg_duckdb template so that "
        "function names, options and syntax are real, not guessed. Returns page, section, URL and the doc text."
    )
    args_schema: type[BaseModel] = DuckDbDocsInput

    def _run(self, query: str, version: str = "lts", limit: int = 5) -> str:
        if not os.path.isfile(CACHE):
            os.makedirs(os.path.dirname(CACHE), exist_ok=True)
            try:
                urllib.request.urlretrieve(URL, CACHE)
            except Exception as exc:  # noqa: BLE001
                return f"docs index missing and download failed: {exc!r}"
        limit = max(1, min(int(limit), 10))
        version = version if version in {"lts", "current", "blog"} else "lts"
        q = query.replace("'", "''")
        sql = (
            "SELECT page_title, section, url, text, score FROM ("
            "  SELECT *, fts_main_docs_chunks.match_bm25(chunk_id, ?) AS score FROM docs_chunks WHERE version = ?"
            ") WHERE score IS NOT NULL ORDER BY score DESC LIMIT ?"
        )
        try:
            conn = duckdb.connect(CACHE, read_only=True)
            rows = conn.execute(sql, [query, version, limit]).fetchall()
            conn.close()
        except Exception as exc:  # noqa: BLE001
            return f"docs search error: {str(exc)[:500]}"
        if not rows:
            return f"No documentation chunks matched {q!r} in version {version!r}"
        out = [f"{len(rows)} doc chunk(s) for {query!r} ({version}):"]
        for title, section, url, text, score in rows:
            body = " ".join(str(text).split())[:700]
            out.append(f"- {title}" + (f" › {section}" if section else "") + f"  (https://duckdb.org{url}, score {score:.2f})\n    {body}")
        return "\n".join(out)
