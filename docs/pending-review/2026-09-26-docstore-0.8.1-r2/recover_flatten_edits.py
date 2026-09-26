"""Recover Probata doc edits lost in the 2026-09-20 git flatten from the Docstore source mirror.

Byline: Claude Code · Opus 5.5 · 2026-09-26

On 2026-09-20 (~20:50 UTC) a "git consolidation" moved 306 uncommitted files out of the Probata
checkout into to_be_deleted/git-flatten-20260920/dirty-main/ (plan with path, sha256 and
destination per file: Propria/.reconciliation/probata-flatten-20260920/main-clean-plan.json);
that quarantine was later removed. 62 of the files were docs/*.md: 24 vanished (restored earlier
today from the mirror) and 38 are tracked files whose uncommitted edits were reset to HEAD.

The Docstore source mirror on ovh-files was built on 2026-09-19 from the desktop, so it may still
hold those edited versions: either in place, or - for files the 2026-09-26 22:23 UTC sync replaced -
in the mirror's quarantine to_be_deleted/sync-<generation>/probata/. This script compares each
candidate's sha256 with the plan's sha256 (the edited version) and, with --apply, writes every exact
match to docs/_recovered-2026-09-20-edits/<same relative path>. Live files are never touched, and
nothing on the VPS is written. Read-only against the VPS (ssh + sha256sum/cat).

    python3 recover_flatten_edits.py            # report
    python3 recover_flatten_edits.py --apply    # also write the exact matches
"""
from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
from pathlib import Path

REPO = Path(__file__).resolve().parents[3]
PLAN = REPO.parents[2] / ".reconciliation/probata-flatten-20260920/main-clean-plan.json"
OUT = REPO / "docs/_recovered-2026-09-20-edits"
SSH = ["ssh", "-o", "ConnectTimeout=10", "-o", "BatchMode=yes", "-i", str(Path.home() / ".ssh/ovh"),
       "root@100.91.190.107"]
MIRROR = "/data/propria/releases/docstore-0.8.1/sources"
GENERATION = "2a00ad9b38534a90852120053ffd3aa3"


def remote_hashes(paths: list[str]) -> dict[str, str]:
    """sha256 of each existing remote path (absolute), read-only."""
    script = "while IFS= read -r p; do [ -f \"$p\" ] && sha256sum \"$p\"; done; true"
    # Bytes, not text: a Windows text-mode pipe would turn every "\n" into "\r\n" and no path would match.
    out = subprocess.run(SSH + [script], input=("\n".join(paths) + "\n").encode("utf-8"), capture_output=True,
                         check=True).stdout.decode("utf-8")
    return {line[66:]: line[:64] for line in out.splitlines() if len(line) > 66}


def remote_bytes(path: str) -> bytes:
    # The path travels on stdin as UTF-8 bytes: names such as "Codex Goal — Horizon Swift MVP.md" survive
    # neither JSON escaping nor Windows argument encoding.
    return subprocess.run(SSH + ['IFS= read -r p; cat -- "$p"'], input=(path + "\n").encode("utf-8"),
                          capture_output=True, check=True).stdout


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    plan = json.loads(PLAN.read_text(encoding="utf-8"))
    docs = [f for f in plan["files"] if f["path"].startswith("docs/") and f["path"].endswith(".md")]
    rows = []
    for item in docs:
        live = REPO / item["path"]
        live_hash = hashlib.sha256(live.read_bytes()).hexdigest() if live.is_file() else None
        rel = item["path"][len("docs/"):]
        rows.append({"path": item["path"], "rel": rel, "plan_sha256": item["sha256"], "live_sha256": live_hash,
                     "current": f"{MIRROR}/Probata/probata/docs/{rel}",
                     "quarantined": f"{MIRROR}/to_be_deleted/sync-{GENERATION}/probata/{rel}"})
    hashes = remote_hashes([r["current"] for r in rows] + [r["quarantined"] for r in rows])
    summary = {"docs_in_plan": len(rows), "restored_or_unchanged": 0, "edited_lost_locally": 0,
               "mirror_holds_edit": 0, "mirror_lacks_edit": 0, "written": 0}
    report = []
    for r in rows:
        if r["live_sha256"] == r["plan_sha256"]:
            summary["restored_or_unchanged"] += 1
            continue
        summary["edited_lost_locally"] += 1
        source = next((p for p in (r["quarantined"], r["current"]) if hashes.get(p) == r["plan_sha256"]), None)
        entry = {"path": r["path"], "plan_sha256": r["plan_sha256"], "live_sha256": r["live_sha256"],
                 "mirror_current_sha256": hashes.get(r["current"]), "mirror_quarantined_sha256": hashes.get(r["quarantined"]),
                 "recovered_from": source}
        if source is None:
            summary["mirror_lacks_edit"] += 1
        else:
            summary["mirror_holds_edit"] += 1
            if args.apply:
                data = remote_bytes(source)
                if hashlib.sha256(data).hexdigest() != r["plan_sha256"]:
                    raise SystemExit(f"hash changed in transfer: {r['path']}")
                target = OUT / r["rel"]
                if target.exists() and hashlib.sha256(target.read_bytes()).hexdigest() != r["plan_sha256"]:
                    raise SystemExit(f"refusing to overwrite a different recovered copy: {target}")
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(data)
                entry["written_to"] = target.relative_to(REPO).as_posix()
                summary["written"] += 1
        report.append(entry)
    print(json.dumps({"summary": summary, "edited": report}, indent=1))


if __name__ == "__main__":
    main()
