"""Docstore 0.8.1-r7: search attaches only the critical flags that bear on the query.

Byline: Claude Code · Opus 5.5 · 2026-09-28

Same mechanics as ../2026-09-28-docstore-0.8.1-r7/apply_r6.py: whole-file installs, each replaced file must
still have its expected content (LF-normalized sha256) or the install is refused as drift; the live file's
line endings are kept; backups <name>.bak-<utc>-r7; idempotent; dry run unless --apply.

Source is commit 0802774c on main. The release tree's server.py was checked equal to main just before that
commit (sha256 6f1961aa…), so the whole-file install carries exactly that one change.

    python3 apply_r7.py <payload_dir> [release_dir] [--apply]
"""
from __future__ import annotations

import hashlib
import sys
import time
from pathlib import Path

# release-relative path -> LF-normalized sha256 of the content it may replace (None: new file)
TARGETS = {
    "plugins/docstore/control/server.py": "6f1961aa826fef35b1d75fd69cc292893d77b0cd7cf7c3b3d14e8c78d7552c2a",
    "plugins/docstore/control/flag_relevance.py": None,
    "plugins/docstore/control/tests/test_flag_relevance.py": None,
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
            target.with_name(f"{target.name}.bak-{stamp}-r7").write_bytes(current)
        target.write_bytes(data)
    print(f"applied {len(planned)} file(s); backups *.bak-{stamp}-r7")


if __name__ == "__main__":
    main()
