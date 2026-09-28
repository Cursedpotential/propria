"""Docstore 0.8.1-r6: the memory duplicate guard reports word overlap; the guard's SurrealQL ships with a test.

Byline: Claude Code · Opus 5.5 · 2026-09-28

Same mechanics as ../2026-09-27-docstore-0.8.1-r5/apply_r5.py: whole-file installs, each replaced file must
still have its expected r5 content (LF-normalized sha256) or the install is refused as drift; the live file's
line endings are kept; backups <name>.bak-<utc>-r6; idempotent; dry run unless --apply.

The guard itself is the memory-database migration scripts/docstore/schema/2026-09-28-memory-duplicate-guard-
lexical.surql, applied live through the memory MCP. It is copied into the release tree, with the 2026-09-27
migration it builds on, so tests/test_memory_duplicate_guard.py can run the real functions on embedded SurrealDB.

    python3 apply_r6.py <payload_dir> [release_dir] [--apply]
"""
from __future__ import annotations

import hashlib
import sys
import time
from pathlib import Path

# release-relative path -> LF-normalized sha256 of the content it may replace (None: new file)
TARGETS = {
    "scripts/docstore/remote_memory.py": "c3f59d981d5b80a392f7779d23f8258df34b76ed7f7f47e2791c6449fce9818c",
    "plugins/docstore/control/release_tools.py": "8280536490388dde0baf58c7d98933dabde4a0b71d23c2d00eea8e5e2fc3516b",
    "scripts/docstore/schema/2026-09-27-memory-remember-guard.surql": None,
    "scripts/docstore/schema/2026-09-28-memory-duplicate-guard-lexical.surql": None,
    "tests/test_memory_duplicate_guard.py": None,
}


def normalized(raw: bytes) -> bytes:
    return raw.replace(b"\r\n", b"\n")


def main() -> None:
    args = [a for a in sys.argv[1:] if a != "--apply"]
    apply = "--apply" in sys.argv[1:]
    payload = Path(args[0])
    root = Path(args[1] if len(args) > 1 else "/data/propria/releases/docstore-0.8.1")
    stamp = time.strftime("%Y%m%dT%H%M%SZ", time.gmtime())
    planned = []
    for relative, before in TARGETS.items():
        target, source = root / relative, payload / relative
        new = normalized(source.read_bytes())
        if relative.endswith(".py"):
            compile(new.decode("utf-8"), str(source), "exec")
        current = target.read_bytes() if target.exists() else None
        if current is not None and normalized(current) == new:
            print(f"already applied  {relative}")
            continue
        if before is None and current is not None:
            raise SystemExit(f"refusing: {relative} already exists with other content")
        if before is not None:
            if current is None:
                raise SystemExit(f"refusing: {relative} is missing")
            found = hashlib.sha256(normalized(current)).hexdigest()
            if found != before:
                raise SystemExit(f"refusing: {relative} drifted (sha256 {found[:16]})")
        crlf = current is not None and b"\r\n" in current
        planned.append((target, current, new.replace(b"\n", b"\r\n") if crlf else new))
        print(f"{'installing' if apply else 'would install'}  {relative} ({'CRLF' if crlf else 'LF'})")
    if not apply:
        return
    for target, current, data in planned:
        target.parent.mkdir(parents=True, exist_ok=True)
        if current is not None:
            target.with_name(f"{target.name}.bak-{stamp}-r6").write_bytes(current)
        target.write_bytes(data)
    print(f"applied {len(planned)} file(s); backups *.bak-{stamp}-r6")


if __name__ == "__main__":
    main()
