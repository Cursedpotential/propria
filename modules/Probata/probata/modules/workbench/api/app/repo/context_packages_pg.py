"""Read registered package provenance and artifacts using existing read-only pools.

Inputs: actual case, source version and keyset bounds. Outputs: complete rows,
counts and persisted category paths. Effects: SELECT only; source bodies remain
in their existing stores and native bundles, never copied into PostgreSQL.
"""
from typing import Any

from app.repo import imported_pg as pg

# This is a persisted path boundary, not a file-extension classifier or taxonomy.
_CATEGORY = "nullif(split_part(split_part(s.source_key, '/SourceCorpus/', 2), '/', 1), '')"
_DOMAIN = f"coalesce(c.placement->>'domain',m.placement->>'domain', {_CATEGORY})"
_CLASSIFICATION = f"coalesce(c.placement->>'category',m.placement->>'category', {_CATEGORY})"
_BASE = f"""
 FROM context.source_version v JOIN context.source s ON s.id=v.source_id
 LEFT JOIN LATERAL (
   SELECT metadata->'placement' AS placement FROM context.source_metadata
   WHERE source_version_id=v.id AND metadata ? 'placement'
   ORDER BY generated_at DESC,id DESC LIMIT 1
 ) m ON true
 LEFT JOIN LATERAL (
   SELECT corrected_value->'fields' AS placement FROM context.source_metadata_correction
   WHERE source_version_id=v.id AND field_key='package:placement' AND action='correct'
   ORDER BY revision DESC LIMIT 1
 ) c ON true
 WHERE v.matter_id=%(matter)s::uuid
"""
_SELECT = f"""SELECT v.id::text AS source_version_ref,s.source_key AS source_ref,
 v.workflow_id,v.declared_format AS source_type,v.version_ordinal AS revision,
 v.original_filename AS label, v.acquired_at,v.status,
 {_DOMAIN} AS domain,{_CLASSIFICATION} AS category,
 coalesce(c.placement->>'path',m.placement->>'path',s.source_key) AS path,
 (SELECT nullif(metadata->>'package_ref','') FROM context.source_metadata WHERE source_version_id=v.id
   AND metadata ? 'package_ref' ORDER BY generated_at DESC LIMIT 1) AS package_ref,
 CASE WHEN c.placement IS NOT NULL THEN jsonb_build_object('kind','owner_correction','placement',c.placement)
 WHEN m.placement IS NOT NULL THEN jsonb_build_object('kind','persisted_source_metadata','placement',m.placement)
 WHEN {_CATEGORY} IS NOT NULL THEN jsonb_build_object('kind','persisted_SourceCorpus_path','source_ref',s.source_key)
 ELSE jsonb_build_object('kind','unresolved','reason','No persisted CaseVault classification on this source') END AS classification_basis,
 (SELECT preview_handle FROM context.proffer_preview_binding WHERE workflow_id=v.workflow_id ORDER BY preview_handle LIMIT 1) AS preview_handle
"""


def categories(matter: str, after: str, limit: int) -> list[dict[str, Any]]:
    """Read bounded category counts from persisted classifications and SourceCorpus paths.

    Inputs: configured matter and keyset. Output: one extra row for continuation.
    Effects: SELECT only; unknown taxonomy stays explicit rather than remapped.
    """
    return pg._query(f"""SELECT coalesce(category,'__unresolved__') AS id,
      coalesce(category,'Unresolved placement') AS label,domain,category,NULL::text AS path,
      coalesce(category,'__unresolved__')||chr(31)||coalesce(domain,'') AS cursor_key,
      count(*) AS count,'persisted_source_placement' AS classification_basis
      FROM ({_SELECT} {_BASE}) p
      WHERE coalesce(category,'__unresolved__')||chr(31)||coalesce(domain,'')>%(after)s
      GROUP BY domain,category ORDER BY cursor_key LIMIT %(limit)s""",
      {"matter": matter, "after": after, "limit": limit + 1})


def packages(matter: str, category: str, domain: str, after: str, limit: int) -> list[dict[str, Any]]:
    """Read only a selected persisted category with stable source-version pagination.

    Inputs: exact category/domain and last UUID. Output: registered sources,
    native workflow and preview references. Effects: bounded SELECT, no moves.
    """
    return pg._query(f"""SELECT * FROM ({_SELECT} {_BASE}) p
       WHERE coalesce(category,'__unresolved__')=%(category)s AND (%(domain)s='' OR domain=%(domain)s)
       AND source_version_ref>%(after)s ORDER BY source_version_ref LIMIT %(limit)s""",
       {"matter": matter, "category": category, "domain": domain, "after": after, "limit": limit + 1})


def source(matter: str, handle: str) -> dict[str, Any]:
    """Resolve a real native handle, existing workflow or preview without synthetic IDs.

    Inputs: matter and human UUID/workflow/preview handle. Output: exact registered
    source. Effects: bounded SELECT; a workflow with multiple versions is ambiguous.
    """
    version = handle.removeprefix("human-") if handle.startswith("human-") else ""
    rows = pg._query(f"""SELECT * FROM ({_SELECT} {_BASE}) p
       WHERE source_version_ref=%(version)s OR workflow_id=%(handle)s OR preview_handle=%(handle)s
       ORDER BY revision DESC LIMIT 2""", {"matter": matter, "version": version, "handle": handle})
    if not rows:
        raise pg.ImportedError("Package source is not registered", 404)
    if len(rows) > 1:
        raise pg.ImportedError("Package handle resolves to multiple source versions", 409)
    return rows[0]


def metadata(version: str, after: str, limit: int) -> list[dict[str, Any]]:
    """Read every metadata class with raw native fields and extraction provenance.

    Inputs: source version/keyset. Output: complete metadata rows plus continuation.
    Effects: SELECT only; no projection whitelist or silent 500-row truncation.
    """
    return pg._query("""SELECT to_jsonb(m) AS fields,m.id::text AS id
      FROM context.source_metadata m WHERE source_version_id=%(version)s::uuid
      AND id::text>%(after)s ORDER BY id LIMIT %(limit)s""",
      {"version": version, "after": after, "limit": limit + 1})


def corrections(version: str, after: str = "", limit: int = 50) -> list[dict[str, Any]]:
    """Read append-only correction heads without dropping their complete predecessor chain.

    Inputs: source version/keyset. Output: complete correction rows plus continuation.
    Effects: SELECT only; select the corrections section for the full history.
    """
    rows = pg._query("""SELECT to_jsonb(c)-'subject_sha256'-'content_digest' AS fields,
      encode(c.subject_sha256,'hex') AS subject_sha256,encode(c.content_digest,'hex') AS content_digest
      FROM context.source_metadata_correction c WHERE source_version_id=%(version)s::uuid
      AND c.correction_ref::text>%(after)s ORDER BY c.correction_ref LIMIT %(limit)s""",
      {"version": version, "after": after, "limit": limit + 1})
    return [{**r["fields"], "subject_sha256": r["subject_sha256"], "content_digest": r["content_digest"]} for r in rows]


def correction_heads(version: str, identities: list[str]) -> dict[str, dict]:
    """Read only latest correction heads for artifacts on the current bounded page.

    Inputs: real version and item IDs. Output: complete head rows indexed by item.
    Effects: SELECT only; history remains separately paginated without truncation.
    """
    rows = pg._query("""SELECT DISTINCT ON (field_key) to_jsonb(c)-'subject_sha256'-'content_digest' AS fields
      FROM context.source_metadata_correction c WHERE source_version_id=%(version)s::uuid
      AND subject_sha256 IS NULL AND field_key=ANY(%(keys)s)
      ORDER BY field_key,revision DESC""", {"version": version, "keys": ["package:" + key for key in identities]})
    return {r["fields"]["field_key"][8:]: r["fields"] for r in rows}


def counts(version: str) -> dict[str, Any]:
    """Count actual metadata and retained package members for a registered source.

    Input: source version. Output: exact counts. Effects: read-only aggregates;
    a zero count asserts absence in these stores, not extraction completeness.
    """
    rows = pg._query("""SELECT
      (SELECT count(*) FROM context.source_metadata WHERE source_version_id=%(version)s::uuid) AS metadata,
      (SELECT count(*) FROM context.source_metadata_correction WHERE source_version_id=%(version)s::uuid) AS corrections,
      (SELECT correction_ref::text FROM context.source_metadata_correction WHERE source_version_id=%(version)s::uuid ORDER BY recorded_at DESC,correction_ref DESC LIMIT 1) AS review_revision,
      (SELECT count(*) FROM context.source_version_object WHERE source_version_id=%(version)s::uuid) AS documents""", {"version": version})
    return rows[0]


def members(version: str, after: str, limit: int) -> list[dict[str, Any]]:
    """Expose complete retained member rows together with original object references.

    Inputs: source version/keyset. Output: whole member and object metadata.
    Effects: bounded SELECT, preserving export membership and original locations.
    """
    return pg._query("""SELECT m.object_id::text AS id,jsonb_build_object('member',to_jsonb(m),'object',to_jsonb(o)) AS fields
      FROM context.source_version_object m JOIN context.retained_object o ON o.id=m.object_id
      WHERE m.source_version_id=%(version)s::uuid AND m.object_id::text>%(after)s
      ORDER BY m.object_id LIMIT %(limit)s""", {"version": version, "after": after, "limit": limit + 1})


def candidates(version: str, section: str, after: str, limit: int) -> tuple[list[dict[str, Any]], int]:
    """Read all existing source-pinned entity/event/fact candidates without an extraction cap.

    Inputs: source version, view and keyset. Output: original complete candidate
    rows/count. Effects: SELECT only; normalization generations are resolved via
    their actual preview snapshots, never substituted for the source version.
    """
    tables = {"entities": [("candidate_entity", "entity")], "events": [("candidate_event", "event")],
      "locations": [("candidate_entity", "entity")], "strategy": [("candidate_fact", "fact")],
      "proposals": [("candidate_entity", "entity"), ("candidate_event", "event"), ("candidate_fact", "fact")]}
    selects = []
    for table, kind in tables.get(section, []):
        extra = ""
        if section == "locations":
            extra = " AND (c.entity_type IN ('location','place','address') OR c.attrs->'candidate'->>'entity_type' IN ('location','place','address'))"
        if section == "strategy":
            extra = " AND coalesce(c.attrs->>'reported_kind',c.attrs->'candidate'->>'kind','') IN ('strategy','history')"
        selects.append(f"""SELECT '{table}:'||c.id::text AS id,'{kind}'::text AS kind,
          to_jsonb(c)-'content_sha256' AS fields,encode(c.content_sha256,'hex') AS content_sha256
          FROM working.{table} c WHERE
          (c.source_raw_table='context.source_version' AND c.source_raw_id=%(version)s
           OR EXISTS(SELECT 1 FROM context.proffer_preview_snapshot p
           WHERE p.source_version_id=%(version)s::uuid
           AND c.source_raw_id IN (p.raw_generation_id::text,p.normalized_generation_id::text))
           OR EXISTS(SELECT 1 FROM context.normalized_record_identity n
             WHERE n.source_version_id=%(version)s::uuid AND n.id::text=c.source_raw_id)) {extra}""")
    if not selects:
        return [], 0
    query = " UNION ALL ".join(selects)
    rows = pg._query(f"SELECT * FROM ({query}) candidates WHERE id>%(after)s ORDER BY id LIMIT %(limit)s",
       {"version": version, "after": after, "limit": limit + 1})
    totals = pg._query(f"SELECT count(*) AS total FROM ({query}) candidates", {"version": version})
    return rows, totals[0]["total"]
