"""Move credentials the services rejected out of ~/.secrets into a dated quarantine folder.

Owner, 2026-09-28: "After [validation] move them into a stale folder so I can fucking remove them
from the directory too."

Reads the verdicts from secrets-validate.py and acts only on `dead` -- a credential the service
answered 401/403 to. `unreachable` is never touched: an unproven credential is not a revoked one,
and treating it as revoked would quarantine working keys.

WHAT IT DOES

  whole file dead     every credential in it was rejected -> the file moves to the quarantine
  file partly dead    the rejected lines are cut out and written to <quarantine>/<file>.dead,
                      and the live file keeps the rest, with a dated marker where they were

Nothing is deleted. Nothing is overwritten. Every edited file is copied to the quarantine first,
so the original is recoverable even if the rewrite is wrong. Only the owner empties the
quarantine -- that is the standing rule and this script does not break it.

Usage:
    python tools/secrets-quarantine-dead.py                 # dry run
    python tools/secrets-quarantine-dead.py --move

Byline: Claude Code · Opus 5 · 2026-09-28
"""
from __future__ import annotations

import argparse
import datetime
import json
import pathlib
import re
import shutil
import sys

HOME = pathlib.Path.home()
SECRETS_DIR = HOME / ".secrets"
REPORT = pathlib.Path(__file__).parent / "secrets-validation.json"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--move", action="store_true", help="perform the move (default: dry run)")
    parser.add_argument("--report", default=str(REPORT))
    args = parser.parse_args()

    report_path = pathlib.Path(args.report)
    if not report_path.is_file():
        sys.stderr.write(f"  no validation report at {report_path}; run secrets-validate.py first\n")
        return 1
    data = json.loads(report_path.read_text(encoding="utf-8"))

    dead_by_file: dict[str, set[str]] = {}
    live_by_file: dict[str, set[str]] = {}
    for row in data["probed"]:
        target = dead_by_file if row["verdict"] == "dead" else live_by_file
        target.setdefault(row["file"], set()).add(row["name"])
    # A key with no probe is not evidence of death; it counts as "keep".
    for row in data.get("unprobed", []):
        live_by_file.setdefault(row["file"], set()).add(row["name"])

    stamp = datetime.datetime.now().strftime("%Y%m%dT%H%M%S")
    quarantine = SECRETS_DIR / f"_dead-{stamp}"

    whole, partial = [], []
    for filename, dead_keys in sorted(dead_by_file.items()):
        path = SECRETS_DIR / filename
        if not path.is_file():
            continue
        survivors = live_by_file.get(filename, set()) - dead_keys
        (whole if not survivors else partial).append((filename, sorted(dead_keys), sorted(survivors)))

    print(f"  quarantine: {quarantine}")
    print(f"  {len(whole)} files where every probed credential is dead -> move the file")
    for filename, dead_keys, _ in whole:
        print(f"    {filename:54} {len(dead_keys)} dead")
    print(f"  {len(partial)} files with a mix -> cut the dead lines, keep the file")
    for filename, dead_keys, survivors in partial:
        print(f"    {filename:54} {len(dead_keys)} dead, {len(survivors)} kept")
        print(f"      dead: {', '.join(dead_keys)[:96]}")

    if not args.move:
        print("\n  dry run; pass --move to quarantine them")
        return 0

    quarantine.mkdir(parents=True, exist_ok=True)
    moved_files, cut_lines = 0, 0

    for filename, dead_keys, _ in whole:
        source = SECRETS_DIR / filename
        target = quarantine / filename.replace("/", "__")
        shutil.move(str(source), str(target))
        moved_files += 1
    print(f"  moved {moved_files} files whole")

    for filename, dead_keys, _ in partial:
        source = SECRETS_DIR / filename
        # Keep a full copy before rewriting, so a bad edit is always recoverable.
        shutil.copy2(source, quarantine / (filename.replace("/", "__") + ".original"))
        original = source.read_bytes()
        newline = b"\r\n" if b"\r\n" in original else b"\n"
        kept, removed = [], []
        for raw in original.split(newline):
            line = raw.decode("utf-8", "replace")
            m = re.match(r"^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=", line)
            if m and m.group(1) in dead_keys:
                removed.append(line)
                continue
            kept.append(line)
        if removed:
            marker = f"# {len(removed)} credential(s) rejected by their service were moved to {quarantine.name} on {stamp}"
            kept.insert(0, marker)
            source.write_bytes(newline.join(kept))
            (quarantine / (filename.replace("/", "__") + ".dead")).write_text(
                "\n".join(removed) + "\n", encoding="utf-8")
            cut_lines += len(removed)
    print(f"  cut {cut_lines} dead lines from {len(partial)} files (originals copied alongside)")
    print(f"\n  nothing was deleted. Review {quarantine} and remove it yourself when satisfied.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
