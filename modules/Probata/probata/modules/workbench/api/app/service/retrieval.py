"""Forward Workbench search to the existing authenticated Platform retrieval service.

Inputs are validated browser choices; outputs retain the Platform's cited results
and per-source failures. Effects are read requests only. Use this for combined
content search instead of catalog-only discovery. Byline: Codex · 2026-10-06.
"""
from urllib.parse import urlencode, urlsplit, unquote
from uuid import UUID

from app.repo.spine_client import spine_json
from app.repo.object_store_client import validate_source_key
from app.types.source_roots import configured_source_roots
from app.service import imported
from app.service.intake_discovery import _upstream
from app.types.retrieval import GraphResolveRequest, SearchRequest


def search(payload: SearchRequest) -> dict:
    """Search approved source families while keeping provider and case authority server-owned.

    Input is SearchRequest; output is the full cited retrieval response including
    partial failures. Sends one read-only POST; creates no run or evidence item.
    """
    result = spine_json("POST", "/v1/context/retrieve", json={
        **payload.model_dump(mode="json"), "scope": {},
        "per_leg_limit": min(100, max(50, payload.limit)), "leg_timeout_seconds": 25.0,
    })
    return _attach_read_locations(_attach_file_locations(result))


def _attach_read_locations(result: dict) -> dict:
    """Resolve cited Proffer versions through the matter-scoped reader before offering navigation.

    Input is Platform's cited response; output adds original source and conversation
    links while preserving citations. Reads PostgreSQL only. Missing mappings remain
    missing; a reader outage marks navigation unavailable without hiding search hits.
    """
    candidates = [item for item in result.get("items", []) if "proffer" in item.get("legs", [])]
    if not candidates:
        return result
    ids = set()
    for item in candidates:
        for version in item["citation"].get("source_version_ids", []):
            try:
                ids.add(str(UUID(version)))
            except (ValueError, TypeError, AttributeError):
                continue
    try:
        rows = imported.pg.versions_to_threads(imported.live_matter(), sorted(ids)) if ids else []
    except imported.ImportedError:
        for item in candidates:
            item["navigation"] = {"status": "unavailable", "sources": []}
        return result
    by_version = {row["id"]: row for row in rows}
    for item in candidates:
        sources = []
        for version in item["citation"].get("source_version_ids", []):
            row = by_version.get(version)
            if not row:
                continue
            # Open the source's actual conversation; do not assume the chunk's
            # first message belongs to every version represented by that chunk.
            sources.append({"source_version_id": version, "source_uri": row["export_key"],
                            "name": row["export_key"].rsplit("/", 1)[-1],
                            "href": "/read?" + urlencode({"thread": imported.encode_id(row["export_key"], row["conv"])})})
        item["navigation"] = {"status": "resolved" if sources else "unresolved", "sources": sources}
    return result


def capabilities() -> dict:
    """Read supported modes and source availability from Platform without creating index work.

    Takes no input and returns the upstream capability response. Use before
    offering search controls; query-time failures remain independently visible.
    """
    return spine_json("GET", "/v1/context/retrieval-capabilities")


async def resolve_graph(payload: GraphResolveRequest) -> dict:
    """Resolve a search hit to verified completed Intake graph snapshots.

    Input is exact source identity; output retains all matches and ambiguity.
    Makes one read-only upstream POST. Use before following graph relationships;
    neither the browser nor this bridge selects a latest snapshot implicitly.
    """
    return await _upstream("/filesystem/graph/resolve", payload.model_dump(exclude_none=True))


def _attach_file_locations(result: dict) -> dict:
    """Offer a file-browser lookup for recorded Intake keys inside one configured source root.

    Input is a cited search response; output adds a location link while preserving
    original locators. Reads configuration only. This is navigation to inspect a
    recorded location, not a claim that current bytes match the indexed snapshot.
    Ambiguous buckets/roots and out-of-root keys get no guessed link.
    """
    roots = list(configured_source_roots().values())
    for item in result.get("items", []):
        if "intake" not in item.get("legs", []):
            continue
        locator = item.get("citation", {}).get("locator", {})
        value = locator.get("vault_key", "")
        if not value:
            continue
        try:
            parsed = urlsplit(value)
        except ValueError:
            continue
        # A versioned URL needs an exact-version reader; never drop its version.
        if parsed.query or parsed.fragment:
            continue
        key = unquote(parsed.path.lstrip("/")) if parsed.scheme else value
        try:
            if validate_source_key(key) != key:
                continue
        except ValueError:
            continue
        candidates = [root for root in roots
                      if key.startswith(root.key_prefix) and len(key) > len(root.key_prefix)
                      and (not parsed.scheme or (parsed.scheme == root.scheme and parsed.netloc == root.bucket))]
        if len(candidates) != 1:
            continue
        root = candidates[0]
        relative = key[len(root.key_prefix):]
        item["location"] = {"name": relative.rsplit("/", 1)[-1],
                            "href": "/sources?" + urlencode({"root": root.root_id, "file": relative})}
    return result
