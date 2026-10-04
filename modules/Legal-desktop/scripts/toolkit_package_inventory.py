"""Inventory toolkit archive members without selecting or changing source material.

Inputs: a server-side directory of ZIP packages and an output JSON path.
Outputs: member identity, streamed SHA-256, integrity errors, and audit candidates.
Side effects: creates a new receipt only; never extracts or modifies sources.
Use as the bounded package-inspection component of a tracked import Activity,
before citation comparison. Byte integrity does not establish citation correctness.
Byline: Codex, 2026-10-04.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import zipfile
from datetime import UTC, datetime
from pathlib import Path, PurePosixPath


def inspect_package(path: Path, max_bytes: int, max_members: int) -> dict:
    """Read one archive with explicit expansion limits and preserve member errors.

    Inputs: archive path, maximum expanded bytes and member count.
    Outputs: per-member fingerprints and explicit incomplete/error status.
    Side effects: none. Choose for comparison, never as legal validation.
    """
    result = {"package": path.name, "package_sha256": None, "members": [],
              "errors": [], "complete": False}
    try:
        if path.stat().st_size > max_bytes:
            raise ValueError("Compressed package exceeds inspection budget")
        package_digest = hashlib.sha256()
        with path.open("rb") as source:
            while block := source.read(1024 * 1024):
                package_digest.update(block)
        result["package_sha256"] = package_digest.hexdigest()
        with zipfile.ZipFile(path) as archive:
            entries = archive.infolist()
            if len(entries) > max_members or sum(e.file_size for e in entries) > max_bytes:
                raise ValueError("Archive exceeds configured inspection budget")
            seen = set()
            consumed = 0
            for entry in entries:
                name = entry.filename
                parts = PurePosixPath(name.replace(chr(92), "/"))
                if parts.is_absolute() or ".." in parts.parts or ":" in name:
                    result["errors"].append({"member": name, "error": "unsafe member path"})
                    continue
                if name in seen:
                    result["errors"].append({"member": name, "error": "duplicate member path"})
                seen.add(name)
                if entry.is_dir():
                    continue
                digest = hashlib.sha256()
                count = 0
                try:
                    with archive.open(entry) as stream:
                        while block := stream.read(1024 * 1024):
                            count += len(block)
                            consumed += len(block)
                            if consumed > max_bytes:
                                raise ValueError("Actual expansion exceeds inspection budget")
                            digest.update(block)
                    if count != entry.file_size:
                        raise ValueError("Member size differs from declared size")
                    result["members"].append({"path": name, "bytes": count,
                        "sha256": digest.hexdigest(), "empty": count == 0,
                        "audit_candidate": any(word in name.lower() for word in
                            ("audit", "ledger", "validation", "verification", "changelog"))})
                except (OSError, RuntimeError, ValueError, NotImplementedError, zipfile.BadZipFile) as exc:
                    result["errors"].append({"member": name, "error": str(exc)})
                    if consumed > max_bytes:
                        break
            result["complete"] = not result["errors"]
    except (OSError, ValueError, zipfile.BadZipFile) as exc:
        result["errors"].append({"error": str(exc)})
    return result


def main() -> int:
    """Write a new inventory receipt for server-side tracked package inspection.

    Inputs: command-line source/output and expansion budgets.
    Outputs: JSON receipt and nonzero exit for incomplete packages.
    Side effects: exclusive creation of output; existing receipts are preserved.
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--max-expanded-bytes", type=int, default=512 * 1024 * 1024)
    parser.add_argument("--max-members", type=int, default=20000)
    args = parser.parse_args()
    if args.max_expanded_bytes <= 0 or args.max_members <= 0:
        parser.error("Budgets must be positive")
    packages = sorted(args.source.glob("*.zip"))
    if not packages:
        parser.error("No ZIP packages found")
    results = [inspect_package(p, args.max_expanded_bytes, args.max_members) for p in packages]
    occurrences: dict[str, list[dict[str, str]]] = {}
    for package in results:
        for member in package["members"]:
            occurrences.setdefault(member["sha256"], []).append(
                {"package": package["package"], "member": member["path"]})
    receipt = {"schema": "toolkit-package-inventory/v1", "byline": "Codex, 2026-10-04",
        "created_at": datetime.now(UTC).isoformat(), "packages": results,
        "identical_member_candidates": [items for items in occurrences.values() if len(items) > 1],
        "legal_validation": "not performed; inspect substantive audit records separately"}
    with args.output.open("x", encoding="utf-8") as stream:
        json.dump(receipt, stream, indent=2, ensure_ascii=False)
        stream.write("\n")
    return 0 if all(p["complete"] for p in results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
