"""Project retained AI source turns and complete created works into a cited graph bundle.

Inputs are existing verified AI-content references and one explicit conversation;
outputs are an immutable private bundle for the Go Surreal projector. Effects are
read-only PostgreSQL/source reads and a retained derived file, never ingestion,
model extraction, schema changes or evidence promotion. Pick after prepare_content
and extract_work_products, independently of candidate extraction or embedding.
Byline: Codex · GPT-6 · 2026-10-07.
"""
from __future__ import annotations

import hashlib
import json
import os
from datetime import datetime
from itertools import pairwise
from pathlib import Path
from typing import Any
from uuid import uuid4

from server.analysis import ai_content as content

VERSION = "ai-context-graph-20261007"
MAX_NODES = 128
MAX_EDGES = 256
MAX_BODY = 32 * 1024


def _retain(path: Path, encoded: bytes) -> None:
    """Retain complete private derived bytes atomically and preserve interrupted pending files.

    Inputs are one analysis output path and exact bytes. Output is None after
    byte-equal readback; effects create private files and sync directory entries.
    Existing differing bytes fail closed. Pick for retry-safe graph bundles and
    manifests, never for original-source retention or custody hashing.
    """
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    if path.parent.is_symlink():
        raise content.ContentInvalid("analysis output directory cannot be a symlink")
    if path.exists():
        if path.is_symlink() or path.read_bytes() != encoded:
            raise content.ContentInvalid("different graph already occupies immutable generation")
        return
    pending = path.parent / (path.name + "." + uuid4().hex + ".pending")
    descriptor = os.open(pending, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o400)
    with os.fdopen(descriptor, "wb") as output:
        output.write(encoded)
        output.flush()
        os.fsync(output.fileno())
    try:
        os.link(pending, path)
    except FileExistsError:
        if path.is_symlink() or path.read_bytes() != encoded:
            raise content.ContentInvalid("concurrent graph generation differs") from None
    directory = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(directory)
    finally:
        os.close(directory)
    if path.read_bytes() != encoded:
        raise content.ContentInvalid("retained graph readback differs")


def _digest(raw: bytes) -> str:
    """Fingerprint derived bytes for repeatability; no original custody hash is computed."""
    return hashlib.sha256(raw).hexdigest()


def _id(kind: str, *coordinates: Any) -> str:
    """Derive a stable graph object ID from source coordinates without effects."""
    return _digest(content._json([VERSION, kind, *coordinates]))


def _body(text: str, ref: str) -> dict[str, str]:
    """Preserve full bounded text or its exact full-text reference, never a summary.

    Input is source-owned text and a reproducible retained locator. Output includes
    its derived UTF-8 hash and complete reference; no reads or writes occur.
    """
    result = {"body_hash": _digest(text.encode()), "body_ref": ref}
    if len(text.encode()) <= MAX_BODY:
        result["body"] = text
    return result


def build_bundle(prepared: dict, works: dict, *, prepared_ref: str,
                 work_products_ref: str, source_id: str,
                 availability: dict[str, str | None], conversation_index: str,
                 access_policy_id: str, created_by_service: str,
                 record_ids: tuple[str, ...] | None = None) -> dict:
    """Convert one real retained conversation and its full works into cited nodes and relations.

    Inputs include server-read source identity/availability and source-owned role,
    native time and raw locators. Output is a strict bounded Go projector bundle.
    No effects occur. This is structural projection, not semantic extraction:
    assistant turns remain assistant-origin statements and never owner facts.
    """
    if prepared["pins"] != works["pins"] or prepared["source"] != works["source"]:
        raise content.ContentInvalid("graph input source bindings differ")
    if works.get("prepared_ref") != prepared_ref:
        raise content.ContentInvalid("work products depend on a different prepared bundle")
    pin, source = prepared["pins"], prepared["source"]
    records = [r for r in prepared["records"] if str(r["conversation_index"]) == conversation_index]
    if record_ids is not None:
        selected = set(record_ids)
        if len(selected) != len(record_ids) or not selected <= {r["record_id"] for r in records}:
            raise content.ContentInvalid("graph batch selection differs from its source conversation")
        records = [r for r in records if r["record_id"] in selected]
    products = [w for w in works["work_products"] if str(w["locator"]["conversation_index"]) == conversation_index]
    if record_ids is not None:
        products = [w for w in products if w["locator"]["record_id"] in selected]
    if not records or len({r["record_id"] for r in records}) != len(records):
        raise content.ContentInvalid("conversation is unavailable or has duplicate normalized records")
    if not source_id or not access_policy_id or not created_by_service:
        raise content.ContentInvalid("server-owned source and projection binding are required")
    scope = {"matter_id": pin["matter_id"], "case_id": pin["court_case_id"],
             "access_policy_id": access_policy_id, "created_by_service": created_by_service}
    generation = _id("generation", pin, conversation_index, [r["record_id"] for r in records],
                     prepared["bundle_fingerprint"], works["bundle_fingerprint"], scope)
    root_pin = {"source_id": source_id, "source_version_id": pin["source_version_id"],
                "source_hash": source["original_sha256"], "locator": source["original_uri"],
                "validation_ref": "normalized-verification:" + pin["verification_id"]}
    nodes: list[dict] = []
    edges: list[dict] = []

    def node(kind: str, key: str, citation: dict, **fields: Any) -> str:
        """Append one supported source-cited node; return its deterministic ID without I/O."""
        identifier = _id(kind, pin["source_version_id"], conversation_index, key)
        nodes.append({"node_id": identifier, "kind": kind, "source_pins": [citation], **fields})
        return identifier

    def edge(kind: str, start: str, end: str, citation: dict) -> None:
        """Append one actual supported relationship with root citation and stable identity."""
        edges.append({"edge_id": _id("edge", kind, start, end), "kind": kind,
                      "from_node_id": start, "to_node_id": end, "source_pins": [citation]})

    source_node = node("ctx_source", source_id, root_pin, derivative_kind="retained_original_reference")
    version_node = node("ctx_source_version", pin["source_version_id"], root_pin,
                        transformation_refs=["normalized-generation:" + pin["normalized_generation_id"]])
    edge("has_version", source_node, version_node, root_pin)
    speakers: dict[str, str] = {}
    turns: dict[str, tuple[str, dict]] = {}
    for record in records:
        role = record.get("role")
        if role not in {"user", "assistant", "system", "tool"} or not record.get("raw_occurrences"):
            raise content.ContentInvalid("source turn lacks supported native origin or raw lineage")
        locator = content._json({k: record.get(k) for k in (
            "record_id", "ordinal", "native_message_id", "mapping_key", "native_message_index",
            "conversation_id", "conversation_index", "raw_occurrences")}).decode()
        citation = {**root_pin, "locator": locator}
        if len(locator) > 2000:
            raise content.ContentInvalid("source turn locator exceeds graph bound")
        if role not in speakers:
            speakers[role] = node("ctx_entity", "speaker:" + role, citation,
                                  derivative_kind="speaker", source_origin=role)
        fields = _body(record["body"], prepared_ref + "#record_id=" + record["record_id"])
        fields.update(derivative_kind="ai_source_turn", source_origin=role,
                      transformation_refs=[prepared_ref, "normalized-record:" + record["record_id"]])
        if record.get("occurred_at"):
            # This is the source turn's clock, never the time of a narrated event.
            fields["occurred_at"] = record["occurred_at"]
        if availability.get(record["record_id"]):
            fields["source_available_from"] = availability[record["record_id"]]
        turn = node("ctx_content_unit", record["record_id"], citation, **fields)
        turns[record["record_id"]] = (turn, citation)
        edge("contains", version_node, turn, citation)
        edge("authored_by_asserted", turn, speakers[role], citation)
    # Source chronology follows actual native timestamps only. Missing times stay
    # unknown; normalized/import order never supplies an alleged event clock.
    timed = sorted((datetime.fromisoformat(r["occurred_at"]), r["record_id"])
                   for r in records if r.get("occurred_at"))
    for (previous_at, previous), (current_at, current) in pairwise(timed):
        if previous_at < current_at:
            edge("before", turns[previous][0], turns[current][0], turns[current][1])
    by_id = {r["record_id"]: r for r in records}
    for product in products:
        locator = product["locator"]
        record = by_id.get(locator.get("record_id"))
        if record is None or product["content"] != record["body"][product["body_start"]:product["body_end"]]:
            raise content.ContentInvalid("complete created work differs from its exact source turn span")
        turn, citation = turns[record["record_id"]]
        occurrence = product["occurrence_index"]
        key = [record["record_id"], occurrence, product["body_start"], product["body_end"]]
        work_citation = {**citation, "locator": content._json({"record_id": record["record_id"],
            "body_start": product["body_start"], "body_end": product["body_end"],
            "work_products_ref": work_products_ref, "occurrence_index": occurrence,
            "raw_occurrences": record["raw_occurrences"]}).decode()}
        work = node("ctx_content_unit", "work:" + _id("work", key), work_citation,
                    derivative_kind="created_work", source_origin=record["role"],
                    transformation_refs=[work_products_ref], input_node_ids=[turn],
                    **_body(product["content"], work_products_ref + "#record_id=" + record["record_id"] + "&occurrence_index=" + str(occurrence)))
        edge("derived_from", work, turn, work_citation)
        edge("depends_on", work, turn, work_citation)
        edge("contains", turn, work, work_citation)
    if len(nodes) > MAX_NODES or len(edges) > MAX_EDGES:
        raise content.ContentInvalid("conversation graph exceeds one projection batch")
    bundle = {"scope": scope, "generation_id": generation,
              "extraction_run_ref": prepared_ref, "nodes": nodes, "edges": edges}
    if len(content._json(bundle)) > 384 * 1024:
        raise content.ContentInvalid("conversation graph exceeds one bounded projector request")
    return bundle


def prepare_graph(params: dict[str, Any]) -> dict[str, Any]:
    """Prepare one actual verified conversation graph without re-ingesting or extracting.

    Inputs are common AI pins, retained prepared/work references, explicit native
    conversation index and server-owned service/policy binding. Output is a private
    graph file reference/hash and counts. Effects are read-only PostgreSQL queries
    and one immutable private derived write; pick before the Go projection Activity.
    """
    from sqlalchemy import text

    from server.context_chunks.db import read_only_connection

    pin = content.pins(params)
    prepared_ref, works_ref = str(params["prepared_ref"]), str(params["work_products_ref"])
    prepared = content._read(prepared_ref, pin, "prepared")
    works = content._read(works_ref, pin, "work_products")
    coordinate = str(params["conversation_index"])
    with read_only_connection() as conn:
        source = content.source_binding(conn, pin)
        if prepared["source"] != source:
            raise content.ContentInvalid("current retained source binding differs from graph input")
        source_id = conn.execute(text("SELECT source_id::text FROM context.source_version WHERE id=CAST(:source AS uuid)"),
                                 {"source": pin["source_version_id"]}).scalar_one()
        availability = conn.execute(text("SELECT id::text,working.source_available_from(id) AS available "
            "FROM context.normalized_record_identity WHERE normalized_generation_id=CAST(:generation AS uuid)"),
            {"generation": pin["normalized_generation_id"]}).all()
    bundle = build_bundle(prepared, works, prepared_ref=prepared_ref, work_products_ref=works_ref,
        source_id=source_id, availability={row[0]: row[1].isoformat() if row[1] else None for row in availability},
        conversation_index=coordinate, access_policy_id=str(params["access_policy_id"]),
        created_by_service=str(params["created_by_service"]),
        record_ids=tuple(params["record_ids"]) if "record_ids" in params else None)
    root = Path(os.environ.get("ANALYSIS_GRAPH_ROOT", "/data/proffer/derive-scratch/analysis-graphs"))
    if not root.is_absolute() or root.is_symlink():
        raise content.ContentInvalid("analysis graph root must be absolute and non-symlink")
    encoded = content._json(bundle)
    path = root / (bundle["generation_id"] + ".json")
    _retain(path, encoded)
    return {"generation_id": bundle["generation_id"], "bundle_ref": path.as_uri(),
            "bundle_hash": _digest(encoded), "nodes": len(bundle["nodes"]), "edges": len(bundle["edges"]),
            "source_version_id": pin["source_version_id"], "conversation_index": coordinate,
            "source_turns": sum(n.get("derivative_kind") == "ai_source_turn" for n in bundle["nodes"]),
            "created_works": sum(n.get("derivative_kind") == "created_work" for n in bundle["nodes"]),
            "semantic_candidates_included": False}


def prepare_all_graphs(params: dict[str, Any]) -> dict[str, Any]:
    """Prepare every retained conversation in explicit bounded batches without dropping turns.

    Inputs are one verified prepared/work-product pair and server binding. Output
    is a private manifest with exact expected/observed turn and full-work counts.
    Effects are the same read-only queries and immutable derived files as
    prepare_graph; no model/extraction calls or Surreal writes occur. Pick for
    complete source projection after the first real conversation is verified.
    """
    pin = content.pins(params)
    prepared = content._read(str(params["prepared_ref"]), pin, "prepared")
    works = content._read(str(params["work_products_ref"]), pin, "work_products")
    groups: dict[str, list[str]] = {}
    for record in prepared["records"]:
        groups.setdefault(str(record["conversation_index"]), []).append(record["record_id"])
    batches = []
    observed = []
    for coordinate, ids in groups.items():
        cursor = 0
        while cursor < len(ids):
            size = min(64, len(ids) - cursor)
            while True:
                selection = ids[cursor:cursor + size]
                try:
                    receipt = prepare_graph({**params, "conversation_index": coordinate, "record_ids": selection})
                except content.ContentInvalid as error:
                    if "exceeds" not in str(error) or size == 1:
                        raise
                    size = max(1, size // 2)
                    continue
                break
            batches.append({**receipt, "record_ids": selection})
            observed.extend(selection)
            cursor += size
    expected = [r["record_id"] for r in prepared["records"]]
    if (sorted(observed) != sorted(expected) or len(set(observed)) != len(expected)
            or sum(b["source_turns"] for b in batches) != len(expected)
            or sum(b["created_works"] for b in batches) != len(works["work_products"])):
        raise content.ContentInvalid("graph batch manifest does not account for the complete retained source")
    manifest = {"byline": "Codex / GPT-6 / 2026-10-07", "version": VERSION, "pins": pin,
                "prepared_ref": params["prepared_ref"], "work_products_ref": params["work_products_ref"],
                "prepared_fingerprint": prepared["bundle_fingerprint"], "work_products_fingerprint": works["bundle_fingerprint"],
                "expected_source_turns": len(expected), "observed_source_turns": len(observed),
                "source_turn_manifest_hash": content._key(sorted(expected)),
                "expected_created_works": len(works["work_products"]),
                "observed_created_works": sum(b["created_works"] for b in batches),
                "conversations": len(groups), "batches": batches,
                "semantic_candidates_included": False,
                "ordering_scope": "strict native source timestamps within each bounded batch"}
    root = Path(os.environ.get("ANALYSIS_GRAPH_ROOT", "/data/proffer/derive-scratch/analysis-graphs"))
    path = root / (content._key(manifest) + ".manifest.json")
    encoded = content._json(manifest)
    _retain(path, encoded)
    return {"manifest_ref": path.as_uri(), "manifest_hash": _digest(encoded),
            "batches": len(batches), "source_turns": len(observed),
            "created_works": manifest["observed_created_works"], "conversations": len(groups),
            "projected_to_surreal": False}


def main() -> None:
    """Prepare real private graph bundles through the existing worker's mounted configuration.

    Inputs are a bounded private params JSON file and optional all-conversations
    flag. Output is refs/counts only; effects are prepare_graph's read-only queries
    and derived files. Pick for direct operations; Temporal callers use the same
    functions with reference-only history. Secrets and source bodies are unprinted.
    """
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--params", type=Path, required=True)
    parser.add_argument("--all", action="store_true", dest="all_conversations")
    args = parser.parse_args()
    try:
        if not args.params.is_absolute() or not args.params.is_file() or args.params.stat().st_size > 65536:
            raise content.ContentInvalid("absolute bounded private parameter file required")
        # docker exec starts a new process; worker CMD's exported secret-derived
        # URL is not inherited. Reuse that same existing mounted credential.
        if not os.environ.get("PLATFORM_DB_URL"):
            url = Path("/run/secrets/platform-database-url").read_text().strip()
            for scheme in ("postgresql://", "postgres://"):
                if url.startswith(scheme):
                    url = "postgresql+psycopg://" + url[len(scheme):]
                    break
            if not url.startswith("postgresql+psycopg://") or "\n" in url:
                raise content.ContentInvalid("existing mounted database URL is malformed")
            os.environ["PLATFORM_DB_URL"] = url
        params = json.loads(args.params.read_bytes())
        receipt = prepare_all_graphs(params) if args.all_conversations else prepare_graph(params)
        print(json.dumps(receipt, sort_keys=True))
    except Exception:  # noqa: BLE001 -- CLI must not print source bodies, DSNs or provider/driver errors
        raise SystemExit("analysis graph preparation failed; source and retained outputs preserved") from None


if __name__ == "__main__":
    main()
