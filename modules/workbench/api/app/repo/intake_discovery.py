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


def units_for_roots(roots: list[str]) -> dict:
    """Units whose unit_root or export_root is one of these exact prefixes.

    Byline: Claude Code · Opus 5 · 2026-09-22 — the folder rows on Sources ask
    "is this folder a unit?" for one page of folders at a time. Read-only:
    the catalog is never written from the Workbench.
    """
    wanted = [value.rstrip("/") for value in roots if value][:200]
    if not wanted:
        return {"backend": "casebible_atomic_units", "items": []}
    rows = _query("""SELECT unit_id, unit_type, source, export_root, service, unit_root,
        member_count, total_bytes, members_without_sha1, parent_unit_id
        FROM raw_duck.atomic_units
        WHERE rtrim(unit_root, '/') = ANY(%s) OR rtrim(export_root, '/') = ANY(%s)
        ORDER BY unit_id LIMIT 400""", (wanted, wanted))
    return {"backend": "casebible_atomic_units", "items": rows,
            "source_links_verified": False,
            "notice": "Atomic-unit membership is recorded catalog metadata, not a verified vault link."}


def units_for_member_keys(keys: list[str]) -> dict:
    """Which of these VAULT object keys are recorded members of a unit.

    Verified live 2026-09-22 against the catalog: `atomic_unit_members.key`
    holds the raw-dedupe B2 key, not the vault key the browser lists, and every
    one of the 128,834 member rows joins `intake_catalog_fs.b2_key_recorded`.
    The catalog row then carries `vault_key`, which IS what the Sources listing
    shows — so the member mark is that two-step join, not a direct compare.
    `vault_key` is indexed; one page of keys measured ~36 ms.
    """
    wanted = [value for value in keys if value][:400]
    if not wanted:
        return {"backend": "casebible_atomic_units", "items": []}
    rows = _query("""SELECT c.vault_key AS key, m.unit_id, u.unit_type, u.unit_root
        FROM raw_duck.atomic_unit_members m
        JOIN raw_duck.intake_catalog_fs_20260917 c ON c.b2_key_recorded = m.key
        JOIN raw_duck.atomic_units u ON u.unit_id = m.unit_id
        WHERE c.vault_key = ANY(%s) LIMIT 800""", (wanted,))
    return {"backend": "casebible_atomic_units", "items": rows, "source_links_verified": False}


def units_under_prefix(prefix: str) -> dict:
    """How many recorded units the files under ONE vault folder belong to.

    Byline: Claude Code · Opus 5 · 2026-09-22. `atomic_units.unit_root` is an
    ORIGINAL source path (`gdrive/salem85/Cube ACR`), so no vault folder ever
    equals it — verified live: 0 of 608 units are rooted under
    `consignatio/vault/`. A vault folder is therefore a unit only in the
    derived sense that everything under it belongs to one recorded unit, which
    is what this answers. One prefix per call: measured ~0.8 s, and the 4 s
    statement timeout bounds it.
    """
    root = normalize_parent(prefix)
    if not root:
        raise DiscoveryError("A folder is required for a unit lookup", 422)
    pattern = root.replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_") + "/%"
    rows = _query("""SELECT u.unit_id, u.unit_type, u.unit_root, u.member_count,
        u.total_bytes, u.members_without_sha1, u.parent_unit_id, u.source, u.export_root, u.service
        FROM raw_duck.intake_catalog_fs_20260917 c
        JOIN raw_duck.atomic_unit_members m ON m.key = c.b2_key_recorded
        JOIN raw_duck.atomic_units u ON u.unit_id = m.unit_id
        WHERE c.vault_key LIKE %s ESCAPE '\\'
        GROUP BY u.unit_id, u.unit_type, u.unit_root, u.member_count, u.total_bytes,
                 u.members_without_sha1, u.parent_unit_id, u.source, u.export_root, u.service
        ORDER BY u.unit_id LIMIT 6""", (pattern,))
    return {"backend": "casebible_atomic_units", "prefix": root, "units": rows,
            "single_unit": rows[0] if len(rows) == 1 else None,
            "units_truncated": len(rows) >= 6, "source_links_verified": False,
            "basis": "catalog membership of the files under this folder"}


def catalog_by_vault_key(vault_key: str) -> dict:
    """Catalog provenance for one vault object: where it came from, how often it occurs.

    Byline: Claude Code · Opus 5 · 2026-09-22 — the metadata panel's
    "catalog provenance" block. `vault_key` is matched exactly; nothing is
    pattern-expanded, so a caller cannot widen the scan.
    """
    if len(vault_key) > 4096 or "\x00" in vault_key:
        raise DiscoveryError("Invalid catalog vault key", 422)
    rows = _query(f"""SELECT rel, parent, name, size, modtime AS modified_at, recorded_at,
        source, scope FROM {CATALOG} WHERE vault_key = %s ORDER BY rel LIMIT 50""", (vault_key,))
    total = _query(f"SELECT count(*) AS occurrences FROM {CATALOG} WHERE vault_key = %s", (vault_key,))
    return {"backend": "casebible_preingest_catalog", "vault_key": vault_key,
            "occurrences": int(total[0]["occurrences"]) if total else 0,
            "items": rows, "items_truncated": len(rows) >= 50,
            "freshness": {"catalog_snapshot": "2026-09-17", "checked_at_is_source_update": False}}


def atomic_members(unit_id: int, limit: int = 100, cursor: str | None = None) -> dict:
    binding = {"unit_id": unit_id, "table": "raw_duck.atomic_unit_members"}
    last = cursor_decode(cursor, binding)
    rows = _query("SELECT unit_id, key FROM raw_duck.atomic_unit_members WHERE unit_id = %s AND key > %s ORDER BY key LIMIT %s", (unit_id, last, limit + 1))
    return {"backend": "casebible_atomic_units", "items": rows[:limit],
            "has_more": len(rows) > limit, "source_links_verified": False,
            "next_cursor": cursor_encode(rows[limit - 1]["key"], binding) if len(rows) > limit else None,
            "notice": "Recorded keys are historical references, not verified intake source references."}
