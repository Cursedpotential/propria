"""Profile a data file with DuckDB: schema, row count, first rows (mirrors duckdb-skills `read-file`, read-only).

Byline: Claude Code · Fable 5.1 · 2026-09-06
"""

import os

import duckdb
from crewai.tools import BaseTool
from pydantic import BaseModel, Field

_READERS = {
    ".csv": "read_csv({p})",
    ".tsv": "read_csv({p})",
    ".json": "read_json_auto({p})",
    ".jsonl": "read_json_auto({p})",
    ".ndjson": "read_json_auto({p})",
    ".parquet": "read_parquet({p})",
}


class DuckDbReadFileInput(BaseModel):
    path: str = Field(..., description="Absolute path to a CSV/TSV/JSON/JSONL/NDJSON/Parquet file")
    sample_rows: int = Field(5, description="How many leading rows to show (0-20)")


class DuckDbReadFileTool(BaseTool):
    name: str = "DuckDB profile a data file"
    description: str = (
        "Opens a CSV/TSV/JSON/JSONL/NDJSON/Parquet file with DuckDB and returns its inferred schema (DESCRIBE), row count "
        "and the first rows - the fastest way to know what a fixture, export or report actually contains. For XML or "
        "other formats say so and use the read-only SQL tool with the right reader instead."
    )
    args_schema: type[BaseModel] = DuckDbReadFileInput

    def _run(self, path: str, sample_rows: int = 5) -> str:
        if not os.path.isfile(path):
            return f"NOT A FILE: {path}"
        ext = os.path.splitext(path)[1].lower()
        reader = _READERS.get(ext)
        if not reader:
            return f"unsupported extension {ext!r}; supported: {sorted(_READERS)}"
        lit = "'" + path.replace("'", "''") + "'"
        src = reader.format(p=lit)
        sample_rows = max(0, min(int(sample_rows), 20))
        conn = duckdb.connect()
        try:
            schema = conn.execute(f"DESCRIBE SELECT * FROM {src}").fetchall()
            count = conn.execute(f"SELECT count(*) FROM {src}").fetchone()[0]
            head = conn.execute(f"SELECT * FROM {src} LIMIT {sample_rows}").fetchall() if sample_rows else []
        except Exception as exc:  # noqa: BLE001
            return f"read error: {str(exc)[:600]}"
        finally:
            conn.close()
        out = [f"{path}", f"size: {os.path.getsize(path)} bytes · rows: {count}", "columns:"]
        out += [f"  - {name}: {ctype}" for name, ctype, *_ in schema]
        if head:
            names = [s[0] for s in schema]
            out.append("first rows:")
            for r in head:
                out.append("  " + " | ".join(f"{n}={(' '.join(str(v).split()))[:60]}" for n, v in zip(names, r)))
        return "\n".join(out)
