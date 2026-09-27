"""Read-only DuckDB SQL tool (bulk scans over files/docs, optional read-only attach of a .duckdb file).

Mirrors the owner's duckdb-skills `query` + `attach-db` skills in read-only form.

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import re
import threading

import duckdb
from crewai.tools import BaseTool
from pydantic import BaseModel, Field

_ALLOWED_FIRST = {"SELECT", "WITH", "FROM", "DESCRIBE", "SUMMARIZE", "SHOW", "EXPLAIN", "PIVOT", "UNPIVOT", "VALUES"}
_FORBIDDEN = re.compile(
    r"\b(COPY|EXPORT|IMPORT|INSTALL|LOAD|ATTACH|DETACH|CREATE|INSERT|UPDATE|DELETE|DROP|ALTER|TRUNCATE|CALL|SET|RESET|PRAGMA|BEGIN|COMMIT|VACUUM|CHECKPOINT)\b",
    re.IGNORECASE,
)


def _strip_comments(sql: str) -> str:
    sql = re.sub(r"/\*.*?\*/", " ", sql, flags=re.S)
    sql = re.sub(r"--[^\n]*", " ", sql)
    return sql.strip().rstrip(";").strip()


class DuckDbQueryInput(BaseModel):
    sql: str = Field(..., description="One read-only SQL statement (SELECT/WITH/FROM/DESCRIBE/SUMMARIZE/SHOW/EXPLAIN). Use absolute paths in read_text()/read_csv()/read_json()/read_parquet() globs.")
    db_path: str | None = Field(None, description="Optional .duckdb file to open READ-ONLY and query (its tables are visible unqualified)")
    max_rows: int = Field(50, description="Maximum rows to return (1-200)")


class DuckDbQueryTool(BaseTool):
    name: str = "DuckDB read-only SQL"
    description: str = (
        "Runs ONE read-only DuckDB statement (SELECT / WITH / FROM / DESCRIBE / SUMMARIZE / SHOW / EXPLAIN) and returns a "
        "markdown table. This is the bulk-scan engine: sweep many files at once with read_text('E:/.../docs/**/*.md') "
        "(columns filename, content), e.g. SELECT filename, regexp_extract_all(content, 'D-\\d{3}') AS decisions FROM "
        "read_text('<repo>/docs/**/*.md') WHERE content ILIKE '%read_xml%'; crunch CSV/JSON/Parquet with read_csv/"
        "read_json_auto/read_parquet; or pass db_path to query a .duckdb file read-only. Writes, COPY, INSTALL, ATTACH and "
        "PRAGMA are rejected. Long results are truncated - aggregate in SQL instead of paging."
    )
    args_schema: type[BaseModel] = DuckDbQueryInput

    def _run(self, sql: str, db_path: str | None = None, max_rows: int = 50) -> str:
        clean = _strip_comments(sql)
        if not clean:
            return "empty statement"
        if ";" in clean:
            return "one statement only (remove the ';' separated extras)"
        first = clean.split(None, 1)[0].upper().strip("(")
        if first not in _ALLOWED_FIRST:
            return f"rejected: statement must start with one of {sorted(_ALLOWED_FIRST)}; got {first!r}"
        bad = _FORBIDDEN.search(clean)
        if bad:
            return f"rejected: read-only tool, keyword {bad.group(0).upper()!r} is not allowed"
        max_rows = max(1, min(int(max_rows), 200))
        try:
            conn = duckdb.connect(db_path, read_only=True) if db_path else duckdb.connect()
        except Exception as exc:  # noqa: BLE001
            return f"connect failed: {exc!r}"
        timer = threading.Timer(120.0, conn.interrupt)
        timer.start()
        try:
            cur = conn.execute(clean)
            cols = [d[0] for d in cur.description] if cur.description else []
            rows = cur.fetchmany(max_rows + 1)
        except Exception as exc:  # noqa: BLE001
            return f"query error: {str(exc)[:600]}"
        finally:
            timer.cancel()
            conn.close()
        if not cols:
            return "statement ran but returned no columns"
        truncated = len(rows) > max_rows
        rows = rows[:max_rows]

        def cell(v: object) -> str:
            s = " ".join(str(v).split()) if v is not None else ""
            return (s[:200] + "…") if len(s) > 200 else s

        out = ["| " + " | ".join(cols) + " |", "|" + "---|" * len(cols)]
        out += ["| " + " | ".join(cell(v) for v in r) + " |" for r in rows]
        out.append(f"{len(rows)} row(s) shown" + (" (more available - aggregate or add LIMIT/WHERE)" if truncated else ""))
        return "\n".join(out)
