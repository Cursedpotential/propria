"""Compose bounded whole-package review from existing source and native readers.

Inputs: actual category, handle, section and source-bound cursor. Outputs:
lossless artifact pages, actual edges and durable review capabilities. Effects:
reads only except explicit review, which delegates to SourceMetadataStore.
"""
import base64
import json
import re
from urllib.parse import quote, urlencode

from starlette.concurrency import run_in_threadpool

from app.repo import context_packages_pg as repo
from app.runtime.analysis import _actor_headers
from app.service import proffer
from app.service.matter_mode import configured_matter_id
from app.service.proffer_errors import ProfferError
from app.types.context_packages import PackageReview, PackageExtractionRetry

SECTIONS = ("allmetadata", "chunks", "events", "timeline", "entities", "locations", "strategy", "documents", "records", "proposals", "edges", "corrections", "embeddings")
PAGE_SIZE = 50
IMMUTABLE_FIELDS = ["body", "messages", "turns", "raw_body", "full_text", "message_body", "content", "text", "payload", "raw_reply", "reply", "replies"]


def _cursor(binding: dict, last: str | int) -> str:
    """Encode a continuation tied to its exact category/source/section/revision.

    Inputs: query coordinates and last key. Output: opaque cursor. Effects: none;
    choose over offsets that can accidentally cross packages or extraction attempts.
    """
    return base64.urlsafe_b64encode(json.dumps({"binding": binding, "last": last}, sort_keys=True).encode()).decode()


def _after(cursor: str | None, binding: dict, default=""):
    """Reject malformed and cross-query cursors before any reader call.

    Inputs: supplied cursor and current query. Output: last key. Effects: none;
    extraction/reference changes require restarting the page from its first row.
    """
    if not cursor:
        return default
    try:
        value = json.loads(base64.b64decode(cursor, altchars=b"-_", validate=True))
        if value["binding"] != binding or not isinstance(value["last"], (str, int)):
            raise ValueError()
        return value["last"]
    except (ValueError, KeyError, TypeError, UnicodeError):
        raise ProfferError("Cursor does not belong to this package/category revision", 409) from None


def _matter() -> str:
    """Resolve the existing personal matter for read-only package queries.

    Inputs: current configuration. Output: actual UUID; effects: none, no new gate.
    """
    return str(configured_matter_id("LIVE"))


def _placement(row: dict) -> dict:
    """Carry persisted placement and its basis without classifying by extension.

    Input: registered source row. Output: inherited placement descriptor. Effects:
    none; unresolved identity stays visible and never becomes a fabricated category.
    """
    return {"domain": row.get("domain"), "category": row.get("category"), "path": row.get("path"),
            "basis": row.get("classification_basis"), "unresolved": not bool(row.get("category")),
            "inherited_from": row.get("package_ref") or row["source_ref"]}


def _handle(row: dict) -> str | None:
    """Select a real registered native workflow or legacy preview handle.

    Input: source registration. Output: usable handle or explicit absence; effects:
    none, never creates normalized-record or opaque preview identities.
    """
    if row["source_type"] == "smsbackuprestore_xml":
        return "human-" + row["source_version_ref"]
    if (row.get("workflow_id") or "").startswith("context-source:"):
        return row["workflow_id"]
    return row.get("preview_handle")


async def categories(cursor: str | None) -> dict:
    """Return the category landing before any package/file is selected.

    Inputs: bounded cursor. Output: actual persisted categories and registered
    package counts. Effects: read-only; no physical directory or source movement.
    """
    binding = {"route": "categories", "matter": _matter()}
    rows = await run_in_threadpool(repo.categories, binding["matter"], _after(cursor, binding), PAGE_SIZE)
    more = len(rows) > PAGE_SIZE
    rows = rows[:PAGE_SIZE]
    return {"items": rows, "next_cursor": _cursor(binding, rows[-1]["cursor_key"]) if more else None,
            "coverage": "registered_sources_with_persisted_classification"}


async def packages(category: str, domain: str, cursor: str | None) -> dict:
    """Read one chosen category without flattening source-version package provenance.

    Inputs: category/domain and cursor. Output: bounded registered package entries
    carrying their existing package unit. Effects: SELECT only, no extension guesses.
    """
    binding = {"route": "packages", "matter": _matter(), "category": category, "domain": domain}
    rows = await run_in_threadpool(repo.packages, binding["matter"], category, domain, _after(cursor, binding), PAGE_SIZE)
    more = len(rows) > PAGE_SIZE
    rows = rows[:PAGE_SIZE]
    for row in rows:
        row.update(handle=_handle(row), package_ref=row.get("package_ref") or row["source_ref"], placement=_placement(row))
        row["label"] = row.get("label") or row["source_ref"].rsplit("/", 1)[-1]
        row["available"] = row["handle"] is not None
        if not row["available"]:
            row["reason"] = "Registered source has no native workflow or preview reader binding"
    return {"items": rows, "next_cursor": _cursor(binding, rows[-1]["source_version_ref"]) if more else None}


async def _engine(path: str, request, **kwargs) -> dict:
    """Read a trusted existing Go endpoint and preserve its entire response object.

    Inputs: server-selected path and actor. Output: lossless dictionary; effects:
    authenticated Go GET only, with no new group, hash or custody prerequisite.
    """
    response = await proffer._request("GET", path, headers=_actor_headers(request), **kwargs)
    value = proffer._json_payload(response, "package review reader")
    if not isinstance(value, dict):
        raise ProfferError("Package reader returned no object", 502)
    return value


def _edges(fields: dict) -> list[dict]:
    """Expose actual relationship arrays unchanged, including unknown relation kinds.

    Input: complete artifact. Output: stored edge objects only. Effects: none;
    no attachment, chronological or identity edge is fabricated for display.
    """
    rows = []
    for key, value in fields.items():
        if key in ("edges", "relations", "relationships") and isinstance(value, list):
            rows.extend(edge for edge in value if isinstance(edge, dict))
        elif isinstance(value, dict):
            rows.extend(_edges(value))
        elif isinstance(value, list):
            for child in value:
                if isinstance(child, dict):
                    rows.extend(_edges(child))
    return rows


def _item(row: dict, source: dict, section: str, index: int, correction: dict | None, handle: str) -> dict:
    """Wrap one original artifact without discarding native metadata or source pins.

    Inputs: complete row, exact source and current correction. Output: review item
    with all fields/edges and supported actions. Effects: none; observations remain
    separate from owner corrections and committed interpretations.
    """
    fields = row.get("fields", row)
    identity = str(row.get("id") or row.get("candidate_id") or row.get("record_id") or row.get("node_id") or row.get("chunk_ref") or row.get("edge_id") or row.get("work_product_ref") or (row.get("content_key") if section == "chunks" else None) or
                   (f"native-record:{source['source_version_ref']}:{row['ordinal']}" if "ordinal" in row else f"{section}:{index}"))
    native = fields.get("native", fields)
    candidate = fields.get("attrs", {}).get("candidate", {}) if isinstance(fields.get("attrs"), dict) else {}
    if not isinstance(candidate, dict):
        candidate = {}
    canonical = fields.get("source_raw_table") == "context.source_version" and bool(candidate.get("source_ref")) and bool(row.get("content_sha256"))
    working_candidate = identity.startswith(("candidate_entity:", "candidate_event:", "candidate_fact:"))
    if not isinstance(native, dict):
        native = fields
    locator = native.get("source_locator") or fields.get("source_locator_ref") or fields.get("locator_ref") or fields.get("source_span") or {"artifact_id": identity, "section": section}
    if canonical:
        locator = {"native_json_pointer": candidate.get("native_json_pointer"), "source_span": candidate.get("source_span"),
                   "span_unit": candidate.get("span_unit"), "prepared_ref": candidate.get("prepared_ref")}
    refs = row.get("source_refs") or fields.get("source_refs") or fields.get("source_pins") or [{"source_ref": source["source_ref"],
        "source_version_ref": source["source_version_ref"], "locator": locator}]
    root = {"source_ref": source["source_ref"], "source_version_ref": source["source_version_ref"], "locator": locator}
    if canonical:
        root.update(source_sha256=candidate.get("source_sha256"), provider_version_id=candidate.get("version_id"))
    if root not in refs:
        refs = [root, *refs]
    current = correction.get("corrected_value", {}) if correction else {}
    revision = correction["correction_ref"] if correction else ""
    state = current.get("review_state", row.get("review_state", "unreviewed"))
    if fields.get("review_state") == "superseded":
        state = "superseded"
    metadata_item = section == "allmetadata"
    operations = ["metadata"] if metadata_item else ["edit", "accept", "reject", "commit"]
    capabilities = []
    for operation in operations:
        allowed = operation != "commit" or state == "accepted"
        reason = "Accept the inspected proposal revision before interpretation commit"
        if working_candidate and not canonical:
            allowed = False
            reason = "This normalized or retained-source candidate uses its existing extraction review/commit route"
        if state in {"superseded", "committed"} or (canonical and state in {"approved", "rejected"}):
            allowed = False
            reason = "Historical candidate revision is preserved for inspection"
        # Native message/chunk text is immutable source data; corrections are
        # bounded assertions, never copies of those full source fields.
        capabilities.append({"operation": operation, "available": allowed,
          "immutable_fields": IMMUTABLE_FIELDS,
          "path": f"/api/context/packages/{quote(handle, safe='')}/review", "method": "POST",
          "body": {"operation": operation, "item_id": row.get("review_item_id", identity), "revision": revision, "source_refs": refs,
                   **({"candidate_content_sha256": row["content_sha256"]} if canonical else {})},
          "scope": "canonical_working_candidate" if canonical else "review_assertion",
          "editable_fields": ["name", "entity_type", "event_type", "predicate", "statement", "confidence", "occurred_at"] if canonical else None,
          **({"reason": reason} if not allowed else {})})
    return {"id": identity, "kind": row.get("kind") or section, "label": str(fields.get("label") or fields.get("name") or fields.get("title") or identity),
      "fields": fields, "source_refs": refs, "edges": [fields] if section == "edges" else _edges(fields), "review_state": state,
      "revision": revision, "capabilities": capabilities, "placement": _placement(source),
      "content_sha256": row.get("content_sha256"),
      "domain": source.get("domain"), "category": source.get("category"), "path": source.get("path"),
      "classification_basis": source.get("classification_basis"), "correction": current or None}


async def snapshot(handle: str, section: str, cursor: str | None, request) -> dict:
    """Read the selected source-type package section with lossless paginated fields.

    Inputs: real handle, section and cursor. Output: dynamic sections, full artifact
    rows, exact source refs and stored edges. Effects: existing read-only stores;
    review decisions never gate source/chunk/search availability.
    """
    if section not in SECTIONS and not re.fullmatch(r"native:[A-Za-z0-9_]{1,64}", section):
        raise ProfferError("Unknown package section", 422)
    source = await run_in_threadpool(repo.source, _matter(), handle)
    counts, history = await run_in_threadpool(repo.counts, source["source_version_ref"]), await run_in_threadpool(repo.corrections, source["source_version_ref"])
    binding = {"handle": handle, "version": source["source_version_ref"], "section": section, "revision": source["revision"]}
    if section in {"entities", "events", "locations", "strategy", "proposals"}:
        binding["working_revision"] = counts.get("review_revision")
    native = handle.startswith("human-")
    ai = (source.get("workflow_id") or "").startswith("context-source:") and not native
    last = "" if (ai or native) and section != "corrections" else _after(cursor, binding)
    available = {"allmetadata": True, "documents": True, "corrections": True}
    if not ai:
        for candidate_section in ("entities", "events", "locations", "strategy", "proposals"):
            _, total = await run_in_threadpool(repo.candidates, source["source_version_ref"], candidate_section, "", 0)
            counts[candidate_section] = total
            available[candidate_section] = True
    reasons = {}
    raw_rows = []
    next_key = None
    upstream = {}
    if ai and section != "corrections":
        upstream = await _engine(f"/reference-import/context/workflows/{quote(source['workflow_id'], safe='')}/package", request,
            params={"section": section, "offset": 0})
        binding["reader_revision"] = upstream.get("revision")
        last = _after(cursor, binding)
        if section not in {"allmetadata", "proposals"} and last:
            page = await _engine(f"/reference-import/context/workflows/{quote(source['workflow_id'], safe='')}/package", request,
                params={"section": section, "offset": int(last)})
            if page.get("revision") != binding["reader_revision"]:
                raise ProfferError("Native extraction attempt changed while paging", 409)
            upstream = page
        if upstream.get("source_version_ref") != source["source_version_ref"]:
            raise ProfferError("Native package source version changed", 409)
        native_source = upstream.get("source") or {}
        if native_source and native_source.get("source_ref") != source["source_ref"]:
            raise ProfferError("Native package root source differs", 409)
        counts.update(upstream.get("counts", {}))
        available.update(upstream.get("available", {}))
    elif native and section != "corrections":
        upstream = await _engine(f"/reference-import/context/previews/{quote(handle, safe='')}", request)
        binding["reader_revision"] = upstream.get("manifest_ref")
        last = _after(cursor, binding)
        for stage in upstream.get("stages", []):
            counts["records"] = max(counts.get("records", 0), stage.get("records", 0))
            counts["timeline"] = counts["records"]
            counts["chunks"] = max(counts.get("chunks", 0), stage.get("chunks", 0))
        available.update(records=bool(upstream.get("manifest_ref")), timeline=bool(upstream.get("manifest_ref")), chunks=counts.get("chunks", 0) > 0)
    elif section not in {"allmetadata", "documents", "corrections", "entities", "events", "locations", "strategy", "proposals", "edges", "embeddings"}:
        await proffer._require_mode(handle, "LIVE")
        upstream = await _engine(f"/reference-import/previews/{quote(handle,safe='')}/content", request,
            params={"limit": PAGE_SIZE, "record_cursor": last if section != "chunks" else "", "chunk_cursor": last if section == "chunks" else ""})
        available.update(records=True, chunks=True, timeline=True)
    if section == "allmetadata":
        # Native stage metadata is additionally reachable as explicit artifacts;
        # source_metadata pages include all classes, raw records and new fields.
        rows = await run_in_threadpool(repo.metadata, source["source_version_ref"], str(last), PAGE_SIZE)
        raw_rows = rows[:PAGE_SIZE]
        counts["allmetadata"] = counts["metadata"] + 1
        if ai:
            counts["allmetadata"] += upstream.get("counts", {}).get("allmetadata", 0)
            if not cursor:
                raw_rows = [{"id": value["bundle_ref"] + "#metadata", "fields": value} for value in upstream.get("items", [])] + raw_rows
        next_key = raw_rows[-1]["id"] if len(rows) > PAGE_SIZE else None
    elif section == "corrections":
        rows = await run_in_threadpool(repo.corrections, source["source_version_ref"], str(last), PAGE_SIZE)
        raw_rows = [{"id": row["correction_ref"], "fields": row} for row in rows[:PAGE_SIZE]]
        next_key = raw_rows[-1]["id"] if len(rows) > PAGE_SIZE else None
    elif section == "documents" and not ai:
        rows = await run_in_threadpool(repo.members, source["source_version_ref"], str(last), PAGE_SIZE)
        raw_rows = rows[:PAGE_SIZE]
        next_key = raw_rows[-1]["id"] if len(rows) > PAGE_SIZE else None
    elif ai and section == "proposals":
        rows, total = await run_in_threadpool(repo.candidates, source["source_version_ref"], section, str(last), PAGE_SIZE)
        raw_rows = rows[:PAGE_SIZE]
        counts[section], available[section] = total, True
        next_key = raw_rows[-1]["id"] if len(rows) > PAGE_SIZE else None
    elif ai:
        raw_rows = upstream.get("items", [])
        next_key = upstream.get("next_offset")
    elif native and section in {"records", "timeline", "chunks"}:
        query = urlencode({"offset": int(last or 0), "limit": PAGE_SIZE, "section": "chunks" if section == "chunks" else "records"})
        page = await _engine(f"/reference-import/context/previews/{quote(handle,safe='')}/content?{query}", request)
        raw_rows = page.get("items", page.get("records", []))
        if section == "chunks":
            if not page.get("bundle_ref"):
                raise ProfferError("Native chunk page returned no actual bundle reference", 502)
            raw_rows = [{"id": page["bundle_ref"] + "#chunk/" + str(int(last or 0)+i), "fields": row,
                "source_refs": [{"source_ref": source["source_ref"], "source_version_ref": source["source_version_ref"],
                    "locator": {"bundle_ref": page["bundle_ref"], "ordinal": int(last or 0)+i}}]} for i, row in enumerate(raw_rows)]
        next_key = page.get("next_offset")
    elif section in {"entities", "events", "locations", "strategy", "proposals"}:
        rows, total = await run_in_threadpool(repo.candidates, source["source_version_ref"], section, str(last), PAGE_SIZE)
        raw_rows = rows[:PAGE_SIZE]
        counts[section] = total
        available[section] = True
        next_key = raw_rows[-1]["id"] if len(rows) > PAGE_SIZE else None
    elif not native and section in {"records", "timeline", "chunks"}:
        key = "chunks" if section == "chunks" else "records"
        value = upstream.get(key, {})
        raw_rows = value.get("items", []) if isinstance(value, dict) else value
        next_key = value.get("next_cursor") if isinstance(value, dict) else upstream.get("next_chunk_cursor" if key == "chunks" else "next_record_cursor")
    else:
        available[section] = False
        reasons[section] = "No retained reader/extraction artifact for this section on the current native source"
    page_ids = [row.get("review_item_id") or _item(row, source, section, int(last or 0)+index if str(last or 0).isdigit() else index, None, handle)["id"] for index, row in enumerate(raw_rows)]
    heads = await run_in_threadpool(repo.correction_heads, source["source_version_ref"], ["placement", *page_ids])
    items = []
    for index, row in enumerate(raw_rows):
        item = _item(row, source, section, int(last or 0)+index if str(last or 0).isdigit() else index, None, handle)
        review_key = row.get("review_item_id") or item["id"]
        if review_key in heads:
            item = _item(row, source, section, int(last or 0)+index if str(last or 0).isdigit() else index, heads[review_key], handle)
        items.append(item)
    # Placement is a source-wide metadata correction whose inherited value is
    # read by both category and package-list SQL, without relocating any object.
    if section == "allmetadata" and not cursor:
        item = _item({"id": "placement", "fields": _placement(source)}, source, section, 0, heads.get("placement"), handle)
        items.insert(0, item)
    section_keys = [*SECTIONS, *sorted(key for key in available if key not in SECTIONS)]
    sections = [{"id": key, "label": {"allmetadata": "All metadata", "strategy": "Strategy"}.get(key, key.title()),
                 "count": counts.get(key), "available": available.get(key, False),
                 **({"reason": reasons.get(key, "No retained artifact reader for this source section")} if not available.get(key, False) else {})} for key in section_keys]
    result = {"package_ref": source.get("package_ref") or source["source_ref"], "source_ref": source["source_ref"],
      "source_version_ref": source["source_version_ref"], "source_type": source["source_type"],
      "revision": "/".join(str(value or "") for value in (source["revision"], binding.get("reader_revision"), counts.get("review_revision"))),
      "domain": source.get("domain"), "category": source.get("category"), "path": source.get("path"),
      "classification_basis": source.get("classification_basis"), "placement": _placement(source),
      "metadata": {"source": source, "reader": {k:v for k,v in upstream.items() if k not in {"items","records","chunks"}}, "corrections": history[:PAGE_SIZE],
        "correction_history_next_cursor": _cursor({k:v for k,v in {**binding, "section": "corrections"}.items() if k != "reader_revision"}, history[PAGE_SIZE-1]["correction_ref"]) if len(history)>PAGE_SIZE else None}, "sections": sections,
      "items": items, "next_cursor": _cursor(binding, next_key) if next_key is not None and next_key != "" else None,
      "capabilities": [{"operation": "retry_extraction", "available": bool(ai and upstream.get("retry_extraction", {}).get("available")),
        "path": f"/api/context/packages/{quote(handle, safe='')}/retry", "method": "POST",
        "body": {"source_version_ref": source["source_version_ref"], "model_id": "configured"},
        "models": [{"id": "configured", "label": "Configured native extraction model"}],
        "reason": upstream.get("retry_extraction", {}).get("reason", "No configured native retry seam for this source")}]}
    if len(json.dumps(result, default=str).encode()) > 4 * 1024 * 1024:
        raise ProfferError("Package page exceeds explicit four MiB bound; no fields were silently omitted", 413)
    return result


async def retry_extraction(handle: str, body: PackageExtractionRetry, request, key: str) -> dict:
    """Start an additive configured-model retry pinned by the server to its original native workflow.

    Inputs: package handle, displayed version and actor click key. Output: real
    Temporal attempt/parent IDs. Effects: candidate extraction/staging only; no
    source reingestion, correction replacement or evidence promotion.
    """
    source = await run_in_threadpool(repo.source, _matter(), handle)
    workflow = source.get("workflow_id") or ""
    if not workflow.startswith("context-source:") or source["source_version_ref"] != body.source_version_ref:
        raise ProfferError("Retry requires this exact native source workflow and registered version", 409)
    response = await proffer._request("POST", f"/reference-import/context/workflows/{quote(workflow, safe='')}/retry",
        json=body.model_dump(mode="json"), headers=_actor_headers(request, key))
    value = proffer._json_payload(response, "extraction retry")
    if not isinstance(value, dict) or not value.get("workflow_id") or not value.get("run_id") or value.get("source_version_ref") != source["source_version_ref"]:
        raise ProfferError("Retry returned no source-pinned Temporal attempt", 502)
    ledger = PackageReview(operation="metadata", item_id="attempt:" + value["workflow_id"],
        fields={"extraction_attempt": value}, reason="Owner requested a configured-model extraction retry",
        source_refs=[{"source_ref": source["source_ref"], "source_version_ref": source["source_version_ref"],
                      "locator": {"workflow_id": value["workflow_id"], "run_id": value["run_id"]}}])
    value["attempt_receipt"] = await review(handle, ledger, request, "extraction-attempt:" + key)
    value["status_path"] = f"/api/context/packages/{quote(handle, safe='')}/attempts/{quote(value['workflow_id'], safe='')}"
    return value


async def extraction_attempt(handle: str, workflow_id: str, request) -> dict:
    """Read an actual retry's business progress after checking its original registered version.

    Inputs: package and returned attempt ID. Output: compact source-pinned status.
    Effects: existing actor-bound Temporal read only; terminal orchestration never implies extraction success.
    """
    if not workflow_id.startswith("context-source:retry:") or len(workflow_id) > 160:
        raise ProfferError("Unknown extraction attempt", 404)
    source = await run_in_threadpool(repo.source, _matter(), handle)
    value = await _engine(f"/reference-import/context/workflows/{quote(workflow_id, safe='')}", request)
    if value.get("source_version_ref") != source["source_version_ref"]:
        raise ProfferError("Extraction attempt belongs to another source version", 409)
    return value


async def review(handle: str, body: PackageReview, request, key: str) -> dict:
    """Persist one explicit revision-bound action through the existing Go metadata writer.

    Inputs: actual package, edited assertions/source refs and actor click key.
    Output: append-only receipt. Effects: atomic Go correction/canonical candidate
    transaction; native candidate commit uses the existing analysis decision writer,
    while source bodies and evidence promotion remain outside this operation.
    """
    source = await run_in_threadpool(repo.source, _matter(), handle)
    response = await proffer._request("POST", f"/reference-import/context/packages/{source['source_version_ref']}/review",
        json=body.model_dump(mode="json"), headers=_actor_headers(request, key))
    value = proffer._json_payload(response, "package review receipt")
    if not isinstance(value, dict) or not value.get("correction_ref") or not value.get("receipt_ref"):
        raise ProfferError("Package review returned no durable receipt", 502)
    return value
