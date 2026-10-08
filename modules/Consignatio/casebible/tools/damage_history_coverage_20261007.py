#!/usr/bin/env python3
"""Annotate the full damage register with bounded exact-key B2 history evidence.

Inputs: immutable run01 report bundle and completed run02 scope/validation receipts.
Outputs: a new seven-file run02 report bundle and SHA-256 readback manifest.
Effects: writes a new derived server report directory; never changes source or B2 objects.
Choose when exact-key history validation has completed and the old report must remain intact.
Byline: Codex / GPT-6.1 / 2026-10-07.
"""

import argparse
import csv
import hashlib
import json
import shutil
from datetime import datetime, timezone
from pathlib import Path


ARTIFACTS = (
    "damaged-files.csv", "damaged-files.jsonl", "damaged-b2-originals.csv",
    "damaged-b2-originals.jsonl", "summary.json", "evidence.json",
    "damaged-files-20261007.md",
)


def sha256(path: Path) -> str:
    """Return a file's SHA-256; input path, output digest, no effects; choose for receipt pins."""
    h = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def annotate(old: Path, history: Path, out: Path) -> dict:
    """Join validated exact-key history to all B2 ledger rows without altering old receipts.

    Inputs: complete old report directory, history receipt directory, unused output directory.
    Outputs: manifest of new artifact hashes and coverage counts.
    Effects: creates the new report directory only after all joins and proofs pass.
    Choose over per-file B2 reads when a completed version-bound history receipt exists.
    Byline: Codex / GPT-6.1 / 2026-10-07.
    """
    if out.exists():
        raise ValueError("output directory already exists")
    for name in ARTIFACTS:
        if not (old / name).is_file():
            raise ValueError(f"missing old report artifact: {name}")
    scope_path, validation_path = history / "scope.json", history / "validation.json"
    temporal_path, launch_path = history / "temporal_receipt.json", history / "launch-receipt.json"
    for path in (scope_path, validation_path, temporal_path, launch_path):
        if not path.is_file():
            raise ValueError(f"missing history proof: {path.name}")
    scope = json.loads(scope_path.read_text(encoding="utf-8"))
    validation = json.loads(validation_path.read_text(encoding="utf-8"))
    temporal = json.loads(temporal_path.read_text(encoding="utf-8"))
    old_rows = [json.loads(line) for line in (old / "damaged-b2-originals.jsonl").open(encoding="utf-8")]
    with (old / "damaged-b2-originals.csv").open(encoding="utf-8-sig", newline="") as source:
        old_csv = list(csv.DictReader(source))
    if len(old_rows) != 103 or len(old_csv) != 103:
        raise ValueError("expected the complete 103-row historical B2 ledger")
    old_keys = {row["src_key"] for row in old_rows}
    if len(old_keys) != 103 or {row["src_key"] for row in old_csv} != old_keys:
        raise ValueError("old CSV and JSONL B2 keys differ or repeat")
    history_keys = {item["damaged"]["src_key"] for item in validation["rows"]}
    if any(row["recovery_status"] != "unresolved_other_sources_not_exhausted" or row["replacement_evidence"] is not None for row in old_rows if row["src_key"] in history_keys):
        raise ValueError("history scope overlaps a previously repaired B2 row")
    if sha256(scope_path) != validation["scope_sha256"]:
        raise ValueError("history validation is not bound to the supplied scope")
    if len(validation["rows"]) != 60 or validation["unique_payload_versions"] != 60 or validation["bytes_budget_consumed"] != 29424988:
        raise ValueError("history receipt does not cover the expected bounded batch")
    if temporal.get("completed") is not True or temporal.get("error") is not None or temporal.get("workflow_id") != "restoration-b2-history-20261007-run02" or temporal.get("run_id") != "01a11655-abda-7475-9b99-24affb01dcd1":
        raise ValueError("history workflow lacks a completed terminal receipt")

    validation_sha256, scope_sha256 = sha256(validation_path), sha256(scope_path)
    coverage = {}
    version_ids = set()
    total_bytes = 0
    for item in validation["rows"]:
        damaged = item["damaged"]
        key = damaged["src_key"]
        if key in coverage or item["listing_status"] != "complete" or item["good_variants"] != 0:
            raise ValueError("duplicate, incomplete, or usable history row")
        payload = [check for check in item["checks"] if check["version"]["action"] == "upload"]
        nonpayload = [check for check in item["checks"] if check["version"]["action"] != "upload"]
        if len(payload) != 1 or len(nonpayload) != 1 or payload[0]["status"] != "all_zero" or nonpayload[0]["status"] != "non_payload_action":
            raise ValueError("history action/body classification differs from completed batch")
        checked = payload[0]
        version = checked["version"]
        if not checked.get("sha256") or not version.get("fileId") or version["fileId"] in version_ids:
            raise ValueError("history payload lacks a unique version and complete body hash")
        if damaged["file_id"] != version["fileId"] or int(damaged["size"]) != int(version["contentLength"]):
            raise ValueError("history version differs from the damaged ledger pin")
        version_ids.add(version["fileId"])
        total_bytes += int(version["contentLength"])
        coverage[key] = {
            "status": "exact_original_key_history_complete_all_payloads_zero",
            "key": key,
            "scope": "exact original B2 key only; aliases, archives, extracted units and other stores untested",
            "listing_status": "complete",
            "payload_version_count": 1,
            "nonpayload_action_count": 1,
            "zero_payload_bytes": int(version["contentLength"]),
            "payload_version_id": version["fileId"],
            "payload_sha256": checked["sha256"],
            "validation_sha256": validation_sha256,
            "scope_sha256": scope_sha256,
            "workflow_id": "restoration-b2-history-20261007-run02",
            "run_id": "01a11655-abda-7475-9b99-24affb01dcd1",
        }
    if len(coverage) != 60 or not set(coverage) <= old_keys or len(version_ids) != 60 or total_bytes != 29424988:
        raise ValueError("history and complete B2 ledger set/byte join failed")
    if len(scope.get("rows", [])) != 60:
        raise ValueError("scope is not the 60-key frozen batch")
    if {item["damaged"]["src_key"] for item in scope["rows"]} != set(coverage):
        raise ValueError("history scope and validated keys differ")

    at = datetime.now(timezone.utc).isoformat()
    updated = [{**row, "history_coverage": coverage.get(row["src_key"])} for row in old_rows]
    summary = json.loads((old / "summary.json").read_text(encoding="utf-8"))
    summary["history_exact_original_key_coverage"] = {
        "observed_at_utc": at, "workflow_id": "restoration-b2-history-20261007-run02",
        "run_id": "01a11655-abda-7475-9b99-24affb01dcd1", "complete_keys": 60,
        "payload_versions": 60, "nonpayload_actions": 60, "zero_payload_bytes": total_bytes,
        "usable_history_payloads": 0, "scope": "exact original B2 keys only; no general unrecoverability claim",
    }
    evidence = json.loads((old / "evidence.json").read_text(encoding="utf-8"))
    evidence["history_exact_original_key_coverage"] = {
        "scope_sha256": scope_sha256, "validation_sha256": validation_sha256,
        "temporal_receipt_sha256": sha256(temporal_path), "launch_receipt_sha256": sha256(launch_path),
        "previous_report_artifacts_sha256": {name: sha256(old / name) for name in ARTIFACTS},
    }
    report = (old / "damaged-files-20261007.md").read_text(encoding="utf-8")
    report += ("\n## Exact-key B2 history coverage added after the snapshot\n\n"
               "The completed run02 history check joined all 60 unresolved original B2 keys to the 103-row B2 ledger. "
               "It listed 120 version records: 60 upload bodies and 60 non-payload actions. All 60 unique payload "
               "versions were read in full (29,424,988 bytes) and were zero-filled. The row-level JSONL and CSV "
               "now pin that exact-key outcome. This does not establish unrecoverability through aliases, archives, "
               "extracted units, other buckets or local sources. Earlier repair evidence and the original report "
               "snapshot remain unchanged.\n\n"
               f"Run: `restoration-b2-history-20261007-run02` / `01a11655-abda-7475-9b99-24affb01dcd1`; "
               f"validation SHA-256 `{validation_sha256}`.\n")
    out.mkdir(parents=True)
    for name in ("damaged-files.csv", "damaged-files.jsonl"):
        shutil.copyfile(old / name, out / name)
    with (out / "damaged-b2-originals.jsonl").open("w", encoding="utf-8") as dest:
        for row in updated:
            dest.write(json.dumps(row, ensure_ascii=False) + "\n")
    fields = list(old_csv[0]) + ["history_exact_key_status", "history_payload_version_id", "history_zero_payload_bytes", "history_validation_sha256"]
    with (out / "damaged-b2-originals.csv").open("w", encoding="utf-8-sig", newline="") as dest:
        writer = csv.DictWriter(dest, fieldnames=fields, extrasaction="ignore")
        writer.writeheader()
        for row in old_csv:
            h = coverage.get(row["src_key"], {})
            writer.writerow({**row, "history_exact_key_status": h.get("status", "not_in_exact_key_history_scope"),
                             "history_payload_version_id": h.get("payload_version_id", ""),
                             "history_zero_payload_bytes": h.get("zero_payload_bytes", ""),
                             "history_validation_sha256": h.get("validation_sha256", "")})
    (out / "summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    (out / "evidence.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2), encoding="utf-8")
    (out / "damaged-files-20261007.md").write_text(report, encoding="utf-8")
    result = {"as_of_utc": at, "full_b2_rows": 103, "covered_exact_original_keys": 60,
              "remaining_b2_rows_outside_history_scope": 43, "zero_payload_bytes": total_bytes,
              "artifacts_sha256": {name: sha256(out / name) for name in ARTIFACTS}}
    (out / "history-coverage-readback.json").write_text(json.dumps(result, indent=2), encoding="utf-8")
    return result


def main() -> None:
    """Build a versioned report bundle from explicit paths; CLI inputs in, manifest out, derived files written."""
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--previous", type=Path, required=True)
    parser.add_argument("--history", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(annotate(args.previous, args.history, args.output), indent=2))


if __name__ == "__main__":
    main()
