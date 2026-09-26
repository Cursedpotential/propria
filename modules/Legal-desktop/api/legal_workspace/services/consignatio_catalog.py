"""Read-only view of the Consignatio catalog for the evidence desk.

> _Byline: Claude Code · Fable 5.1 · 2026-09-21_
The catalog is the authority; this module binds to its existing relations and
never writes. Bytes stay on B2. Bindings name the catalog's real relations and
columns (verified live 2026-09-20 against `casebible.catalog_reconcile`) and
are overridable with CONSIGNATIO_CATALOG_BINDINGS (a JSON file) so a catalog
change is a config edit. Lifecycle comes from the catalog's promotion record
only: folder names such as `vault/v1/Evidence` never make an object evidence.
The promotion relation is unbound until Consignatio publishes one.
"""

from __future__ import annotations

import json
import re
from functools import lru_cache
from pathlib import Path
from typing import Any
from urllib.parse import quote

from pydantic import BaseModel, Field
from sqlalchemy import create_engine, text
from sqlalchemy.engine import Engine

from legal_workspace.config import get_settings

_IDENT = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$")

DEFAULT_BINDINGS: dict[str, Any] = {
    "generation": {"relation": "catalog_reconcile.current_generation"},
    "objects": {
        "relation": "catalog_reconcile.object_versions",
        "id": "file_id",
        "bucket": "bucket",
        "key": "object_key",
        "size": "size",
        "hash": "sha1",
        "hash_algorithm": "sha1",
        "uploaded_at": "uploaded_at",
        "visible": "visible",
        "metadata": "source_metadata",
        "key_prefix": "consignatio/vault/",
    },
    "occurrences": {
        "relation": "catalog_reconcile.occurrences",
        "version_ids": "version_ids",
        "columns": [
            "occurrence_id",
            "source",
            "source_path",
            "provider_source_id",
            "availability",
            "match_basis",
            "size",
            "native_hash_kind",
            "native_hash",
            "recorded_md5",
            "recorded_modtime",
            "quality_flags",
            "source_metadata",
        ],
    },
    # Desk-side roles -> catalog columns. Null until the catalog has a promotion record.
    "promotions": {
        "relation": None,
        "promotion_id": None,
        "state": None,
        "pending_values": [],
        "completed_values": [],
        "promoted_from": None,
        "promoted_at": None,
        "source_hash": None,
        "evidence_bucket": None,
        "evidence_key": None,
    },
    # e.g. "https://<b2-access-host>/d/{bucket}/{key}"; empty = retrieval not configured.
    "artifact_url_template": "",
}


class CatalogUnavailable(RuntimeError):
    pass


class CatalogStatus(BaseModel):
    configured: bool
    reachable: bool = False
    detail: str = ""
    generation: dict[str, Any] | None = None
    promotion_binding: str = "unbound"
    retrieval_configured: bool = False


class CatalogObject(BaseModel):
    id: str
    bucket: str
    key: str
    size: int | None = None
    hash: str | None = None
    hash_algorithm: str = ""
    uploaded_at: str | None = None
    lifecycle: str = "context"
    promotion: dict[str, Any] | None = None
    artifact_url: str | None = None
    metadata: Any = None
    occurrences: list[dict[str, Any]] = Field(default_factory=list)


class CatalogPage(BaseModel):
    items: list[CatalogObject]
    next_after: str | None = None
    promotion_binding: str


def _ident(value: Any) -> str:
    if not isinstance(value, str) or not _IDENT.match(value):
        raise CatalogUnavailable(f"catalog binding is not a plain identifier: {value!r}")
    return value


def load_bindings() -> dict[str, Any]:
    bindings = json.loads(json.dumps(DEFAULT_BINDINGS))
    override = get_settings().consignatio_catalog_bindings_file
    if override:
        loaded = json.loads(Path(override).read_text(encoding="utf-8"))
        for section, value in loaded.items():
            if isinstance(value, dict) and isinstance(bindings.get(section), dict):
                bindings[section].update(value)
            else:
                bindings[section] = value
    return bindings


def _promotion_bound(bindings: dict[str, Any]) -> bool:
    promo = bindings["promotions"]
    return bool(promo.get("relation") and promo.get("state") and promo.get("promoted_from"))


@lru_cache(maxsize=2)
def _engine(url: str) -> Engine:
    return create_engine(
        url,
        pool_pre_ping=True,
        pool_size=2,
        max_overflow=2,
        connect_args={
            "options": "-c default_transaction_read_only=on -c statement_timeout=30000",
            "connect_timeout": 8,
        },
    )


def _connect():
    url = get_settings().consignatio_catalog_url
    if not url:
        raise CatalogUnavailable("CONSIGNATIO_CATALOG_URL is not set")
    try:
        return _engine(url).connect()
    except Exception as exc:  # driver errors carry the DSN; keep them out of responses
        raise CatalogUnavailable(f"catalog connection failed: {type(exc).__name__}") from exc


def _artifact_url(bindings: dict[str, Any], bucket: str, key: str) -> str | None:
    template = bindings.get("artifact_url_template") or ""
    if not template:
        return None
    return template.replace("{bucket}", quote(bucket, safe="")).replace(
        "{key}", quote(key, safe="/")
    )


def _b2_configured() -> bool:
    settings = get_settings()
    return bool(settings.b2_key_id and settings.b2_application_key and settings.b2_s3_endpoint)


def signed_artifact_link(object_id: str) -> dict[str, Any] | None:
    """Short-lived GET link for the exact catalogued object version. Read-only; signs locally."""
    settings = get_settings()
    if not _b2_configured():
        raise CatalogUnavailable("B2 access is not configured")
    bindings = load_bindings()
    obj = bindings["objects"]
    with _connect() as conn:
        row = conn.execute(
            text(
                f"SELECT {_ident(obj['bucket'])} AS bucket, {_ident(obj['key'])} AS key "
                f"FROM {_ident(obj['relation'])} WHERE {_ident(obj['id'])} = :id LIMIT 1"
            ),
            {"id": object_id},
        ).first()
    if row is None:
        return None
    import boto3
    from botocore.config import Config

    client = boto3.client(
        "s3",
        endpoint_url=settings.b2_s3_endpoint,
        region_name=settings.b2_region or None,
        aws_access_key_id=settings.b2_key_id,
        aws_secret_access_key=settings.b2_application_key,
        config=Config(signature_version="s3v4"),
    )
    url = client.generate_presigned_url(
        "get_object",
        # B2's S3 API uses the B2 file id as the version id: the link is pinned to that version.
        Params={"Bucket": row.bucket, "Key": row.key, "VersionId": object_id},
        ExpiresIn=settings.b2_link_ttl_seconds,
    )
    return {"url": url, "expires_in": settings.b2_link_ttl_seconds, "bucket": row.bucket, "key": row.key}


def _jsonable(value: Any) -> Any:
    if value is None or isinstance(value, (str, int, float, bool, list, dict)):
        return value
    return str(value)


def _object_select(obj: dict[str, Any]) -> str:
    return ", ".join(
        f"o.{_ident(obj[role])} AS {role}"
        for role in ("id", "bucket", "key", "size", "hash", "uploaded_at", "metadata")
    )


def _to_object(bindings: dict[str, Any], row: Any) -> CatalogObject:
    data = dict(row._mapping)
    return CatalogObject(
        id=str(data["id"]),
        bucket=data["bucket"],
        key=data["key"],
        size=data["size"],
        hash=data["hash"],
        hash_algorithm=bindings["objects"].get("hash_algorithm", ""),
        uploaded_at=_jsonable(data["uploaded_at"]),
        metadata=data["metadata"],
        artifact_url=_artifact_url(bindings, data["bucket"], data["key"]),
    )


def catalog_status() -> CatalogStatus:
    settings = get_settings()
    if not settings.consignatio_catalog_url:
        return CatalogStatus(configured=False, detail="CONSIGNATIO_CATALOG_URL is not set")
    bindings = load_bindings()
    status = CatalogStatus(
        configured=True,
        promotion_binding="bound" if _promotion_bound(bindings) else "unbound",
        retrieval_configured=bool(bindings.get("artifact_url_template")) or _b2_configured(),
    )
    try:
        with _connect() as conn:
            relation = _ident(bindings["generation"]["relation"])
            row = conn.execute(text(f"SELECT * FROM {relation} LIMIT 1")).first()
            status.reachable = True
            if row is not None:
                status.generation = {k: _jsonable(v) for k, v in row._mapping.items()}
    except CatalogUnavailable as exc:
        status.detail = str(exc)
    return status


def _promotions_for(conn: Any, bindings: dict[str, Any], ids: list[str]) -> dict[str, dict[str, Any]]:
    if not ids or not _promotion_bound(bindings):
        return {}
    promo = bindings["promotions"]
    relation = _ident(promo["relation"])
    source = _ident(promo["promoted_from"])
    rows = conn.execute(
        text(f"SELECT * FROM {relation} WHERE {source} = ANY(:ids)"), {"ids": ids}
    ).all()
    return {str(r._mapping[source]): {k: _jsonable(v) for k, v in r._mapping.items()} for r in rows}


def _lifecycle(bindings: dict[str, Any], record: dict[str, Any] | None) -> str:
    if record is None:
        return "context"
    promo = bindings["promotions"]
    state = record.get(promo["state"])
    if state in promo.get("completed_values", []):
        return "promoted"
    if state in promo.get("pending_values", []):
        return "pending_promotion"
    return "context"


def list_objects(prefix: str = "", q: str = "", after: str = "", limit: int = 50) -> CatalogPage:
    bindings = load_bindings()
    obj = bindings["objects"]
    limit = max(1, min(limit, 200))
    key = _ident(obj["key"])
    sql = (
        f"SELECT {_object_select(obj)} FROM {_ident(obj['relation'])} o "
        f"WHERE o.{_ident(obj['visible'])} AND o.{key} LIKE :prefix "
        f"AND (:q = '' OR o.{key} ILIKE :pattern) AND o.{key} > :after "
        f"ORDER BY o.{key} LIMIT :limit"
    )
    like = (obj.get("key_prefix", "") + prefix).replace("%", r"\%").replace("_", r"\_") + "%"
    with _connect() as conn:
        rows = conn.execute(
            text(sql),
            {"prefix": like, "q": q, "pattern": f"%{q}%", "after": after, "limit": limit + 1},
        ).all()
        items = [_to_object(bindings, row) for row in rows[:limit]]
        promotions = _promotions_for(conn, bindings, [item.id for item in items])
    for item in items:
        item.promotion = promotions.get(item.id)
        item.lifecycle = _lifecycle(bindings, item.promotion)
    return CatalogPage(
        items=items,
        next_after=items[-1].key if len(rows) > limit else None,
        promotion_binding="bound" if _promotion_bound(bindings) else "unbound",
    )


def get_object(object_id: str) -> CatalogObject | None:
    bindings = load_bindings()
    obj, occ = bindings["objects"], bindings["occurrences"]
    with _connect() as conn:
        row = conn.execute(
            text(
                f"SELECT {_object_select(obj)} FROM {_ident(obj['relation'])} o "
                f"WHERE o.{_ident(obj['id'])} = :id LIMIT 1"
            ),
            {"id": object_id},
        ).first()
        if row is None:
            return None
        item = _to_object(bindings, row)
        columns = ", ".join(_ident(c) for c in occ["columns"])
        found = conn.execute(
            text(
                f"SELECT {columns} FROM {_ident(occ['relation'])} "
                f"WHERE {_ident(occ['version_ids'])} @> CAST(:ids AS jsonb) LIMIT 200"
            ),
            {"ids": json.dumps([object_id])},
        ).all()
        item.occurrences = [{k: _jsonable(v) for k, v in r._mapping.items()} for r in found]
        item.promotion = _promotions_for(conn, bindings, [item.id]).get(item.id)
    item.lifecycle = _lifecycle(bindings, item.promotion)
    return item


def list_promotions(state: str) -> list[dict[str, Any]]:
    """Promotion records straight from the catalog. Empty while the relation is unbound."""
    bindings = load_bindings()
    if not _promotion_bound(bindings):
        return []
    promo = bindings["promotions"]
    values = promo.get("completed_values" if state == "promoted" else "pending_values", [])
    if not values:
        return []
    with _connect() as conn:
        rows = conn.execute(
            text(
                f"SELECT * FROM {_ident(promo['relation'])} "
                f"WHERE {_ident(promo['state'])} = ANY(:values) LIMIT 500"
            ),
            {"values": values},
        ).all()
    out = []
    for row in rows:
        record = {k: _jsonable(v) for k, v in row._mapping.items()}
        bucket_col, key_col = promo.get("evidence_bucket"), promo.get("evidence_key")
        if bucket_col and key_col and record.get(bucket_col) and record.get(key_col):
            record["artifact_url"] = _artifact_url(bindings, record[bucket_col], record[key_col])
        out.append(record)
    return out
