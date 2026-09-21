"""Read-only pre-ingest Case Bible catalog; never walks the storage corpus.

Byline: Codex · 2026-09-20.
"""
from __future__ import annotations

import base64
import binascii
import hashlib
import json
import os
from pathlib import Path
from typing import Any

CATALOG = "raw_duck.intake_catalog_fs_20260917"
DIRECTORIES = "raw_duck.intake_catalog_dirs_20260917"


class DiscoveryError(Exception):
    def __init__(self, message: str, status: int = 503):
        self.message, self.status = message, status


def configured() -> bool:
    path = os.getenv("INTAKE_DISCOVERY_PG_PASSWORD_FILE", "")
    return bool(path and Path(path).is_file())


def normalize_parent(value: str) -> str:
    if len(value) > 4096 or "\x00" in value or "\\" in value or value.startswith("/"):
        raise DiscoveryError("Invalid catalog directory reference", 422)
    if any(part in {".", ".."} for part in value.split("/")):
        raise DiscoveryError("Invalid catalog directory reference", 422)
    return value.rstrip("/")


def cursor_encode(last: str, binding: dict) -> str:
    data = {"last": last, "query": hashlib.sha256(json.dumps(binding, sort_keys=True).encode()).hexdigest()}
    return base64.urlsafe_b64encode(json.dumps(data).encode()).decode()


def cursor_decode(cursor: str | None, binding: dict) -> str:
    if not cursor:
        return ""
    try:
        data = json.loads(base64.urlsafe_b64decode(cursor))
        expected = hashlib.sha256(json.dumps(binding, sort_keys=True).encode()).hexdigest()
        if data["query"] != expected or not isinstance(data["last"], str):
            raise ValueError()
        return normalize_parent(data["last"])
    except (ValueError, KeyError, TypeError, UnicodeError, binascii.Error):
        raise DiscoveryError("Cursor does not belong to this query", 422) from None


def _query(sql: str, parameters: tuple) -> list[dict[str, Any]]:
    if not configured():
        raise DiscoveryError("Pre-ingest catalog connection is not configured")
    try:
        import psycopg
        from psycopg.rows import dict_row
        password = Path(os.environ["INTAKE_DISCOVERY_PG_PASSWORD_FILE"]).read_text().strip()
        with psycopg.connect(
            host=os.getenv("INTAKE_DISCOVERY_PG_HOST", "100.91.190.107"),
            port=int(os.getenv("INTAKE_DISCOVERY_PG_PORT", "5475")),
            dbname=os.getenv("INTAKE_DISCOVERY_PG_DATABASE", "casebible"),
            user=os.getenv("INTAKE_DISCOVERY_PG_USER", "metabase_ro"),
            password=password, connect_timeout=5, row_factory=dict_row,
            options="-c default_transaction_read_only=on -c statement_timeout=4000 -c lock_timeout=1000",
        ) as conn:
            return conn.execute(sql, parameters).fetchall()
    except Exception:
        # Connection strings, source text and upstream bodies must not reach error responses.
        raise DiscoveryError("Pre-ingest catalog query unavailable or timed out") from None


def catalog_page(*, parent: str = "", query: str = "", limit: int = 100,
                 cursor: str | None = None, directories: bool = True, mode: str = "filename_prefix") -> dict:
    parent = normalize_parent(parent)
    binding = {"parent": parent, "query": query, "directories": directories, "catalog": CATALOG, "mode": mode}
    last = cursor_decode(cursor, binding)
    # Prefix search stays inside one indexed parent. Wildcards are literal input.
    prefix = query.replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_") + "%"
    where = "parent = %s AND rel > %s AND name LIKE %s ESCAPE '\\'"
    params: tuple = (parent, last, prefix)
    if mode == "filename_substring":
        subtree = parent.replace("%", "\\%").replace("_", "\\_") + "/%"
        where = "(%s = '' OR rel LIKE %s ESCAPE '\\') AND rel > %s AND (name ILIKE %s ESCAPE '\\' OR rel ILIKE %s ESCAPE '\\')"
        params = (parent, subtree, last, "%" + prefix, "%" + prefix)
    files_sql = f"""SELECT rel, parent, name, 'file'::text AS kind, size, modtime AS modified_at,
        recorded_at, source, scope, vault_key FROM {CATALOG}
        WHERE {where}"""
    sql = files_sql
    if directories:
        sql += f""" UNION ALL SELECT rel, parent, name, 'directory'::text AS kind,
            NULL::bigint AS size, NULL AS modified_at, NULL AS recorded_at,
            NULL::text AS source, NULL::text AS scope, NULL::text AS vault_key
            FROM {DIRECTORIES} WHERE {where}"""
        params += params
    rows = _query(f"SELECT * FROM ({sql}) AS entries ORDER BY rel LIMIT %s", params + (limit + 1,))
    more = len(rows) > limit
    rows = rows[:limit]
    for row in rows:
        row["id"] = row["rel"]
        row["source_ref"] = None  # Never invent a raw-store intake reference from a catalog name.
    return {"backend": "casebible_preingest_catalog", "items": rows,
            "next_cursor": cursor_encode(rows[-1]["rel"], binding) if more else None,
            "has_more": more, "complete": not more, "coverage": "catalog_snapshot",
            "query_scope": {"parent": parent, "query": query, "mode": mode},
            "freshness": {"catalog_snapshot": "2026-09-17", "checked_at_is_source_update": False}}


def atomic_units(*, unit_type: str | None = None, after_id: int = -1, limit: int = 100) -> dict:
    rows = _query("""SELECT unit_id, unit_type, source, export_root, service, unit_root,
        member_count, total_bytes, members_without_sha1, parent_unit_id
        FROM raw_duck.atomic_units WHERE unit_id > %s AND (%s::text IS NULL OR unit_type = %s)
        ORDER BY unit_id LIMIT %s""", (after_id, unit_type, unit_type, limit + 1))
    return {"backend": "casebible_atomic_units", "items": rows[:limit],
            "next_after_id": rows[limit - 1]["unit_id"] if len(rows) > limit else None,
            "coverage": "historical_catalog", "source_links_verified": False,
            "notice": "Atomic-unit membership is recorded catalog metadata. Current vault object links require reconciliation before bulk intake."}


def atomic_members(unit_id: int, limit: int = 100, cursor: str | None = None) -> dict:
    binding = {"unit_id": unit_id, "table": "raw_duck.atomic_unit_members"}
    last = cursor_decode(cursor, binding)
    rows = _query("SELECT unit_id, key FROM raw_duck.atomic_unit_members WHERE unit_id = %s AND key > %s ORDER BY key LIMIT %s", (unit_id, last, limit + 1))
    return {"backend": "casebible_atomic_units", "items": rows[:limit],
            "has_more": len(rows) > limit, "source_links_verified": False,
            "next_cursor": cursor_encode(rows[limit - 1]["key"], binding) if len(rows) > limit else None,
            "notice": "Recorded keys are historical references, not verified intake source references."}
