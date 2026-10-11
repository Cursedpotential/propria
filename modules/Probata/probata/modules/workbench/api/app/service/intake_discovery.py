"""Pre-ingest index bridge. Byline: Codex · 2026-09-20."""
from __future__ import annotations

import os
import re
from urllib.parse import urlsplit

import httpx

from app.repo.intake_discovery import DiscoveryError, _query, configured


def index_url() -> str:
    value = os.getenv("INTAKE_DISCOVERY_INDEX_URL", "").rstrip("/")
    parsed = urlsplit(value)
    if not value:
        return ""
    if parsed.scheme not in {"http", "https"} or not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise DiscoveryError("Intake index URL configuration is invalid")
    return value


def capabilities() -> dict:
    catalog, index = configured(), bool(index_url())
    verified = False
    units_verified = False
    unit_types = []
    if catalog:
        try:
            _query("SELECT rel FROM raw_duck.intake_catalog_fs_20260917 LIMIT 1", ())
            _query("SELECT rel FROM raw_duck.intake_catalog_dirs_20260917 LIMIT 1", ())
            verified = True
        except DiscoveryError:
            pass
        try:
            _query("SELECT unit_id FROM raw_duck.atomic_units LIMIT 1", ())
            _query("SELECT unit_id FROM raw_duck.atomic_unit_members LIMIT 1", ())
            unit_types = [row["unit_type"] for row in _query(
                "SELECT DISTINCT unit_type FROM raw_duck.atomic_units WHERE unit_type IS NOT NULL ORDER BY unit_type LIMIT 100", ()
            )]
            units_verified = True
        except DiscoveryError:
            pass
    return {
        "backend": "casebible_preingest_discovery", "catalog_configured": catalog,
        "index_configured": index, "availability_verified": verified,
        "modes": {"filename_prefix": verified, "filename_substring": verified, "contents": index, "hybrid": index},
        "tree": verified, "graph": index, "coverage": "unknown",
        "filters": {"parent": catalog, "file_type": False, "atomic_unit": False},
        "zip_contents": False, "bulk_intake": False, "atomic_unit_catalog": units_verified,
        "unit_types": unit_types,
        "backend_status": {"catalog": "ready" if verified else "unavailable",
                           "atomic_units": "ready" if units_verified else "unavailable",
                           "index": "configured_unverified" if index else "not_configured"},
        "limitations": ["Filename prefix searches one catalog folder.",
                         "Names and paths search is case insensitive across the selected subtree, with a four-second database timeout.",
                         "Contents and hybrid search the configured Intake collection; folder and atomic filters are not supported by that upstream API.",
                         "ZIP member and atomic-unit coverage are not yet verified.",
                         "Selection preserves occurrences; intake requires a resolved source reference."],
    }


async def _upstream(path: str, payload: dict | None = None) -> dict:
    base = index_url()
    if not base:
        raise DiscoveryError("Pre-ingest content index is not configured")
    try:
        async with httpx.AsyncClient(timeout=25, follow_redirects=False) as client:
            response = await (client.post(base + path, json=payload) if payload is not None else client.get(base + path))
            response.raise_for_status()
            result = response.json()
            if not isinstance(result, dict):
                raise ValueError()
            return result
    except (httpx.HTTPError, ValueError):
        raise DiscoveryError("Pre-ingest index unavailable; no storage scan was substituted") from None


async def search_index(query: str, mode: str, limit: int) -> dict:
    """Return bounded Intake search hits with their original source locators.

    Inputs are query text, the requested search mode, and a result limit.
    Outputs retain collection/object, source/document/chunk, and vault identity;
    upstream locators are not verified ingestion or evidence references.
    Side effects are limited to a read-only HTTP search. Use this bridge for
    pre-ingest content search rather than catalog filename lookup.
    Byline: Codex · 2026-10-06.
    """
    body = await _upstream("/filesystem/search", {"query": query, "mode": "keyword" if mode == "contents" else "hybrid", "limit": limit})
    required = {"object_id", "source_path", "filename", "source_id", "document_id", "chunk_id", "text", "score"}
    if not isinstance(body.get("hits"), list) or any(not isinstance(h, dict) or not required.issubset(h) for h in body["hits"]):
        raise DiscoveryError("Pre-ingest index returned an incompatible result")
    return {"backend": "intake_weaviate", "items": [
        {"id": h["object_id"], "rel": h["source_path"], "parent": None,
         "name": h["filename"], "kind": "content_hit", "source_id": h["source_id"],
         "document_id": h["document_id"], "chunk_id": h["chunk_id"],
         "vault_key": h.get("vault_key"), "resolution": h.get("resolution"),
         "text": h["text"], "score": h["score"], "source_ref": None}
        for h in body["hits"][:limit]], "coverage": body.get("coverage", "unknown"),
        "collection": body.get("collection"), "complete": False, "has_more": False,
        "next_cursor": None, "query_scope": {"mode": mode, "query": query, "parent": None},
        "notice": "Ranked results are limited; successful retrieval does not prove complete indexing."}


async def neighbors(table: str, key: str, limit: int) -> dict:
    if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]{0,63}", table) or not re.fullmatch(r"[A-Za-z0-9_-]{1,255}", key):
        raise DiscoveryError("Invalid Intake graph reference", 422)
    return await _upstream(f"/filesystem/graph/neighbors/{table}/{key}?edge_limit={limit}")
