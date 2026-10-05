"""Read-only view of Family Law Toolkit records for the workdesk.

> _Byline: Claude Code · Opus 5.5 · 2026-09-27_
> _Updated by: OpenAI Codex · GPT-6-Luna · 2026-10-05 — expose bounded original links separately from exact record versions._
The toolkit's case store (SurrealDB `surreal-case`, namespace `fct`, database `case`)
owns these records: legal sources, cheat sheets and other reference rows, and case
documents. Advocatio never copies them; it reads each one through the shared
legal-record contract `propria.legal-record.v1` (SINGLE-WORKDESK-CONVERGENCE.md,
"Family Law Toolkit companion surface"), so a record opens here with the same id and
version the toolkit shows.

The store computes the version itself: RECORD_VERSION_SURQL is a byte-identical copy
of the toolkit's `RECORD_VERSION_SURQL` in
modules/Probata/probata/deploy/docker/family-court-console/src/mcp-app/src/store.ts
(its `case_record` tool). Change both together or the versions stop matching.

Sibling pattern: services/consignatio_catalog.py. Empty store URL = "not configured".
"""

from __future__ import annotations

import os
import re
from typing import Any
from urllib.parse import urlencode

import httpx
from pydantic import BaseModel, Field

from legal_workspace.config import get_settings

RECORD_CONTRACT = "propria.legal-record.v1"

RECORD_VERSION_SURQL = (
    "LET $r = (SELECT * OMIT embedding FROM ONLY type::record($tb, $id)); "
    "RETURN IF $r = NONE { NONE } ELSE { { tb: record::tb($r.id), id: <string> record::id($r.id), "
    "version: 'sha256:' + crypto::sha256(<string> $r), record: $r } };"
)

# No row cap: the old LIMIT 200 hid 123 of the 323 reference rows (Claude Code · Opus 5.5 · 2026-10-01).
# kind and category let the page group cheat sheets, checklists, templates and law notes.
_LIST_SURQL = (
    "SELECT VALUE { tb: record::tb(id), id: <string> record::id(id), "
    "title: title ?? citation ?? key ?? label ?? '', "
    "kind: kind ?? authority_class ?? '', category: category ?? '' } "
    "FROM type::table($tb);"
)

_ORIGINAL_LINKS_SURQL = (
    "SELECT <string> id AS binding_id, original_pointer FROM library_file "
    "WHERE record_id = $record_id AND provider = 'b2' AND account_scope = $scope "
    "AND bucket = 'salem-data' "
    "AND legal_root = 'consignatio/casevault/KnowledgeBase/legal/' LIMIT 8;"
)

# Keep this allowlist aligned with the toolkit's DATA_TABLES. It deliberately
# excludes library_validation, library_proposal, and library_revision, which
# are trusted workflow records rather than case/workdesk records.
# _Byline: Codex · GPT-6-Luna · 2026-10-04._
TOOLKIT_TABLES = (
    "person", "child", "order", "hearing", "deadline", "event", "message",
    "exhibit", "factor", "source", "note", "court", "court_event", "filing",
    "draft", "memo", "reference", "evidence_log", "eval", "case_status",
)

_REF = re.compile(r"^(?P<tb>[a-z_]+):(?P<id>.+)$")


class ToolkitUnavailable(RuntimeError):
    pass


class ToolkitStatus(BaseModel):
    configured: bool
    reachable: bool = False
    detail: str = ""
    contract: str = RECORD_CONTRACT


class ToolkitRecord(BaseModel):
    contract: str = RECORD_CONTRACT
    owner: str = "family-court-toolkit/surreal-case"
    id: str
    table: str
    version: str
    record: dict[str, Any]
    original_links: list[ToolkitOriginalLink] = Field(default_factory=list)


class ToolkitOriginalLink(BaseModel):
    """One exact B2 PDF binding exposed through the hosted original-file proxy.

    Inputs are validated library binding metadata; output preserves the exact
    version, raw hash, and byte count. It performs no I/O or record mutation.
    """

    binding_id: str
    version_id: str
    sha256: str
    bytes: int
    content_type: str = "application/pdf"
    href: str


class ToolkitListing(BaseModel):
    table: str
    items: list[dict[str, str]]


def _split_ref(ref: str) -> tuple[str, str]:
    match = _REF.match(ref)
    if not match:
        raise ValueError('expected a record reference "table:id"')
    table, record_id = match.group("tb"), match.group("id")
    if record_id.startswith("⟨") and record_id.endswith("⟩"):
        record_id = record_id[1:-1]
    if table not in TOOLKIT_TABLES:
        raise ValueError(f"table {table!r} is not shared with the workdesk")
    return table, record_id


def _query(surql: str, params: dict[str, str]) -> Any:
    settings = get_settings()
    if not settings.family_court_toolkit_store_url:
        raise ToolkitUnavailable("FAMILY_COURT_TOOLKIT_STORE_URL is not set")
    try:
        response = httpx.post(
            f"{settings.family_court_toolkit_store_url.rstrip('/')}/sql",
            params=params,
            content=surql,
            headers={
                "Accept": "application/json",
                "surreal-ns": settings.family_court_toolkit_store_ns,
                "surreal-db": settings.family_court_toolkit_store_db,
                # The login is a database-level VIEWER user; SurrealDB authenticates it only when the
                # request names that level too (Claude Code · Opus 5.5 · 2026-09-27, verified live: 401 without).
                "surreal-auth-ns": settings.family_court_toolkit_store_ns,
                "surreal-auth-db": settings.family_court_toolkit_store_db,
            },
            auth=(settings.family_court_toolkit_store_user, settings.family_court_toolkit_store_pass),
            timeout=10.0,
        )
    except httpx.HTTPError as exc:  # keep URLs and credentials out of responses
        raise ToolkitUnavailable(f"toolkit store unreachable: {type(exc).__name__}") from exc
    if response.status_code != 200:
        raise ToolkitUnavailable(f"toolkit store answered HTTP {response.status_code}")
    statements = response.json()
    last = statements[-1] if isinstance(statements, list) and statements else None
    if not isinstance(last, dict) or last.get("status") != "OK":
        raise ToolkitUnavailable(f"toolkit store query failed: {last.get('result') if isinstance(last, dict) else 'no result'}")
    return last.get("result")


def toolkit_status() -> ToolkitStatus:
    settings = get_settings()
    if not settings.family_court_toolkit_store_url:
        return ToolkitStatus(configured=False, detail="FAMILY_COURT_TOOLKIT_STORE_URL is not set")
    status = ToolkitStatus(configured=True)
    try:
        _query("RETURN true;", {})
        status.reachable = True
    except ToolkitUnavailable as exc:
        status.detail = str(exc)
    return status


def _original_links(record_id: str) -> list[ToolkitOriginalLink]:
    """Read at most eight scoped original PDF pointers for one shared record.

    Input is the exact table:id; output is validated proxy links or an empty
    list when the feature is unconfigured or no mapping exists. Side effects
    are one parameterized, read-only metadata query; record versions and body
    fields are untouched. Use for the separate original_links envelope.
    """
    scope = os.getenv("TOOLKIT_LIBRARY_SYNC_ACCOUNT_SCOPE")
    if not scope:
        return []

    rows = _query(_ORIGINAL_LINKS_SURQL, {"record_id": record_id, "scope": scope}) or []
    if not isinstance(rows, list):
        return []

    links: list[ToolkitOriginalLink] = []
    for row in rows[:8]:
        if not isinstance(row, dict):
            continue
        binding_id = row.get("binding_id")
        pointer = row.get("original_pointer")
        if not isinstance(pointer, dict):
            continue

        version_id = pointer.get("version_id")
        sha256 = pointer.get("sha256")
        size = pointer.get("size")
        content_type = pointer.get("content_type")
        if (
            not isinstance(binding_id, str)
            or not re.fullmatch(r"library_file:[a-f0-9]{64}", binding_id)
            or not isinstance(version_id, str)
            or not version_id
            or version_id == "null"
            or any(char in version_id for char in "\r\n\0")
            or not isinstance(sha256, str)
            or not re.fullmatch(r"[a-f0-9]{64}", sha256)
            or type(size) is not int
            or size <= 0
            or size > 20 * 1024 * 1024
            or content_type != "application/pdf"
        ):
            continue
        try:
            if len(version_id.encode("utf-8")) > 2048:
                continue
        except UnicodeEncodeError:
            continue

        query = urlencode({"binding_id": binding_id, "version_id": version_id})
        links.append(
            ToolkitOriginalLink(
                binding_id=binding_id,
                version_id=version_id,
                sha256=sha256,
                bytes=size,
                content_type="application/pdf",
                href=f"https://family-court.tilapia-skilift.ts.net/api/library/original?{query}",
            )
        )
    return links


def get_record(ref: str) -> ToolkitRecord | None:
    """Read one exact shared record and its separate bounded original links.

    Input is a table:id reference; output is the unchanged body and store
    version plus scoped original-link metadata. Effects are read-only shared
    store queries. Use for Workdesk detail views that need the record and its
    original PDF entry point.
    """
    table, record_id = _split_ref(ref)
    result = _query(RECORD_VERSION_SURQL, {"tb": table, "id": record_id})
    if not isinstance(result, dict) or not result.get("version"):
        return None
    record = dict(result.get("record") or {})
    record.pop("id", None)
    return ToolkitRecord(
        id=f"{result['tb']}:{result['id']}",
        table=str(result["tb"]),
        version=str(result["version"]),
        record=record,
        original_links=_original_links(f"{result['tb']}:{result['id']}"),
    )


def list_records(table: str) -> ToolkitListing:
    if table not in TOOLKIT_TABLES:
        raise ValueError(f"table {table!r} is not shared with the workdesk")
    rows = _query(_LIST_SURQL, {"tb": table}) or []
    items = [
        {
            "id": f"{row['tb']}:{row['id']}",
            "title": str(row.get("title") or ""),
            "kind": str(row.get("kind") or ""),
            "category": str(row.get("category") or ""),
        }
        for row in rows
        if isinstance(row, dict)
    ]
    if table == "factor":  # statutory order, a through l
        items.sort(key=lambda item: item["id"])
    return ToolkitListing(table=table, items=items)
