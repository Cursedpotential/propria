"""Prepare real source-cited AI graph batches as one independent Temporal Activity.

Inputs are existing source/generation/case pins and retained predecessor references;
outputs contain only batch/manifest references, hashes, counts and actual Temporal
identity. Effects are read-only source queries and private derived graph files.
Pick for the Go graph projection workflow; no Python workflow is introduced.
Byline: Codex · GPT-6 · 2026-10-07.
"""
from __future__ import annotations

import hashlib
import json
import os
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any
from urllib.parse import unquote, urlsplit

from temporalio import activity
from temporalio.exceptions import ApplicationError, CancelledError

from server.temporal.ai_content_activities import _heartbeats


@dataclass
class GraphPrepareParams:
    """Carry existing verified source pins and references without source payloads or human grants.

    Inputs are fixed source/case/service coordinates; output is a decodable Go/Python
    request value. No effects occur; pick for graph preparation independently of
    ingestion or model extraction. Backend configuration supplies the service scope.
    """
    request_id: str = ""
    source_version_id: str = ""
    normalized_generation_id: str = ""
    verification_id: str = ""
    operating_mode: str = ""
    matter_id: str = ""
    court_case_id: str = ""
    prepared_ref: str = ""
    work_products_ref: str = ""
    access_policy_id: str = ""
    created_by_service: str = ""
    source_pins: list[dict[str, str]] = field(default_factory=list)
    expected_source_turns: int = 0
    expected_created_works: int = 0
    expected_conversations: int = 0


@activity.defn(name="ai_prepare_context_graph_activity")
def ai_prepare_context_graph_activity(params: GraphPrepareParams) -> dict[str, Any]:
    """Prepare every real retained AI turn and complete work for downstream graph projection.

    Inputs are source/generation/verification pins and full predecessor references.
    Output is bounded ref-only batches plus actual Temporal workflow/run identity.
    Effects are existing read-only PostgreSQL reads and immutable private derived
    writes. Pick before Go projection/readback; missing source time remains unknown
    and never blocks neutral structural projection.
    """
    from server.analysis import ai_content, ai_context_graph

    try:
        bindings = {"matter_id": "ANALYSIS_GRAPH_MATTER_ID", "court_case_id": "ANALYSIS_GRAPH_CASE_ID",
                    "access_policy_id": "ANALYSIS_GRAPH_ACCESS_POLICY_ID",
                    "created_by_service": "ANALYSIS_GRAPH_CREATED_BY_SERVICE"}
        for field, name in bindings.items():
            configured = os.environ.get(name, "")
            if not configured or getattr(params, field) != configured:
                raise ai_content.ContentInvalid("graph request differs from managed server scope")
        with _heartbeats() as beat:
            receipt = ai_context_graph.prepare_all_graphs(asdict(params), beat=beat)
            parsed = urlsplit(receipt["manifest_ref"])
            root = Path(os.environ.get("ANALYSIS_GRAPH_ROOT", "/data/proffer/derive-scratch/analysis-graphs"))
            path = Path(unquote(parsed.path))
            if (parsed.scheme != "file" or parsed.netloc or parsed.query or parsed.fragment
                    or path.is_symlink() or not path.is_file() or not path.resolve().is_relative_to(root.resolve())
                    or path.stat().st_size > 2 * 1024 * 1024):
                raise ai_content.ContentInvalid("graph manifest is outside the private prepared root")
            encoded = path.read_bytes()
            if hashlib.sha256(encoded).hexdigest() != receipt["manifest_hash"]:
                raise ai_content.ContentInvalid("retained graph manifest readback hash differs")
            manifest = json.loads(encoded)
            for expected, observed in ((params.expected_source_turns, manifest["observed_source_turns"]),
                                       (params.expected_created_works, manifest["observed_created_works"]),
                                       (params.expected_conversations, manifest["conversations"])):
                if isinstance(expected, bool) or expected < 0 or expected != observed:
                    raise ai_content.ContentInvalid("prepared graph counts differ from exact requested source coverage")
            if not 0 < len(manifest["batches"]) <= 512:
                raise ai_content.ContentInvalid("graph batch reference count exceeds workflow bound")
            batches = [{"bundle_ref": item["bundle_ref"], "bundle_sha256": item["bundle_hash"],
                        "generation_id": item["generation_id"], "extraction_run_ref": params.prepared_ref,
                        **{key: item[key] for key in ("source_turns", "created_works", "nodes", "edges")}}
                       for item in manifest["batches"]]
            info = activity.info()
            if params.source_pins:
                first = Path(unquote(urlsplit(batches[0]["bundle_ref"]).path))
                if first.is_symlink() or not first.resolve().is_relative_to(root.resolve()) or first.stat().st_size > 384 * 1024:
                    raise ai_content.ContentInvalid("first graph batch lies outside private bounded output")
                data = json.loads(first.read_bytes())
                actual_pins = [pin for node in data["nodes"] if node["kind"] == "ctx_source" for pin in node["source_pins"]]
                if params.source_pins != actual_pins:
                    raise ai_content.ContentInvalid("requested root source pins differ from verified retained source")
            result = {**ai_content.pins(asdict(params)), **receipt, "batch_refs": batches,
                      "temporal_workflow_id": info.workflow_id, "temporal_run_id": info.workflow_run_id}
            if len(json.dumps(result).encode()) > 256 * 1024:
                raise ai_content.ContentInvalid("graph reference-only Activity receipt exceeds bound")
            beat("real graph batch preparation verified")
            return result
    except ai_content.ContentInvalid as error:
        raise ApplicationError(str(error), type="ContextGraphInvalid", non_retryable=True) from None
    except CancelledError:
        raise
    except Exception:  # noqa: BLE001 -- keep source bodies, driver details and credentials out of history
        raise ApplicationError("context graph preparation failed", type="ContextGraphUnavailable") from None


AI_CONTEXT_GRAPH_ACTIVITIES = [ai_prepare_context_graph_activity]
