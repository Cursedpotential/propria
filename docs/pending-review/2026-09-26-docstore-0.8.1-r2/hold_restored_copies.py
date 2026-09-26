"""Move today's restored/recovered doc copies out of the Probata checkout into the review hold area.

Byline: Claude Code · Opus 5.5 · 2026-09-26

Earlier today the Docstore fix restored 24 docs from the source mirror (untracked, at their original
paths) and wrote 34 recovered edits to docs/_recovered-2026-09-20-edits/. The parent session then
pushed Probata 51424e7, which carries all of them in git. As untracked files at the same paths, the
24 restores would block `git merge origin/main` in this checkout, so they move (never delete) to
Propria/.review_hold/2026-09-26-docstore-fix-restores/ with a hash check after each move.
Only untracked files are moved; a tracked file is never touched. Empty folders stay (no removal).

    python3 hold_restored_copies.py            # list what would move
    python3 hold_restored_copies.py --apply
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import subprocess
from pathlib import Path

REPO = Path(__file__).resolve().parents[3]
PLAN = REPO.parents[2] / ".reconciliation/probata-flatten-20260920/main-clean-plan.json"
HOLD = REPO.parents[2] / ".review_hold/2026-09-26-docstore-fix-restores"
RECOVERED = "docs/_recovered-2026-09-20-edits"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    plan = json.loads(PLAN.read_text(encoding="utf-8"))
    planned = [f["path"] for f in plan["files"] if f["path"].startswith("docs/") and f["path"].endswith(".md")]
    out = subprocess.run(["git", "-C", str(REPO), "ls-files", "-z", "--others", "--exclude-standard", "--", "docs"],
                         capture_output=True, check=True).stdout.decode("utf-8")
    untracked = set(filter(None, out.split("\0")))
    candidates = [p for p in planned if p in untracked]
    candidates += sorted(p for p in untracked if p.startswith(RECOVERED + "/"))
    moved = []
    for rel in candidates:
        source = REPO / rel
        if not source.is_file():
            continue
        digest = hashlib.sha256(source.read_bytes()).hexdigest()
        if not args.apply:
            print("would move", rel)
            continue
        target = HOLD / "modules/Probata/probata" / rel
        if target.exists():
            raise SystemExit(f"hold copy already exists, refusing to overwrite: {target}")
        target.parent.mkdir(parents=True, exist_ok=True)
        os.replace(source, target)
        if hashlib.sha256(target.read_bytes()).hexdigest() != digest:
            raise SystemExit(f"hash changed during move: {rel}")
        moved.append((rel, digest))
    if args.apply:
        lines = [f"- `{digest}`  {rel}" for rel, digest in moved]
        (HOLD / "README.md").write_text(
            "# Docstore-fix restores superseded by Probata 51424e7\n\n"
            "> Byline: Claude Code · Opus 5.5 · 2026-09-26\n\n"
            "Moved here, not deleted, so `git merge origin/main` in modules/Probata/probata is not blocked by\n"
            "untracked files. 51424e7 carries the same docs in git. Only the owner removes anything here.\n\n"
            + "\n".join(lines) + "\n", encoding="utf-8")
    print(json.dumps({"candidates": len(candidates), "moved": len(moved),
                      "restored": sum(1 for rel, _ in moved if not rel.startswith(RECOVERED)),
                      "recovered": sum(1 for rel, _ in moved if rel.startswith(RECOVERED))}))


if __name__ == "__main__":
    main()
