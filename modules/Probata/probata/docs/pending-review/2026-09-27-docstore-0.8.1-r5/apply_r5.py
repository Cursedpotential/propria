"""Docstore 0.8.1-r5: the memory write path works, reports its errors, and publishes its schema.

Byline: Claude Code · Opus 5.5 · 2026-09-27

Unlike r2-r4 (anchored edits), r5 installs whole files, because the seven files it touches were already
identical to this repository's copies (modules/Probata/probata/...) apart from line endings, checked
2026-09-27 by LF-normalized sha256. Each target is replaced only if its current content still has the
expected pre-r5 hash; anything else is refused as drift. The written file keeps the live file's line
endings (CRLF if it had any). Backups: <name>.bak-<utc>-r5. Idempotent. Dry run unless --apply.

    python3 apply_r5.py <payload_dir> [release_dir] [--apply]

<payload_dir> holds the new files at the same relative paths (modules/Probata/probata layout), plus
tests/test_remote_memory.py.
"""
from __future__ import annotations

import hashlib
import sys
import time
from pathlib import Path

# release-relative path -> LF-normalized sha256 of the content it may replace (None: new file). release_tools.py
# also accepts the first r5 build (duplicate cutoff 0.15 in its docstring; raised to 0.20 the same hour).
TARGETS = {
    "scripts/docstore/remote_memory.py": "890682ec05613da3f06f4218df94f0a797f9c3c6b6f05a0cbd7ea7653aa6f388",
    "scripts/docstore/release_api.py": "6b2c52f9cd9f571c95f79f35b56395c11a910c15169b4d3701c49ee71f593551",
    "plugins/docstore/control/server.py": "f7c332b6814dcd19673f6c7c63e6488b3cec4d67d8b0dd6f5234abde597d9197",
    "plugins/docstore/control/governance.py": "c1b6f6d4b27536be5ce6cc8a9caecf964daf3780aff8b8f9cb5df05422acf1fe",
    "plugins/docstore/control/release_tools.py": ("9567c24dcb61fc487f28c4e97c814308d49bd74a32c4c94dd4c4b630d9eaec24",
                                                  "06d41f5a543d161034d7116e4194c3e6b7cd20551781de06f8f0fa97cc4d6e4c"),
    "plugins/docstore/control/tests/test_server.py": "51f4108b3c256975fd6562dc4f81184a508c5009503b04695bbd15dd22118f34",
    "plugins/docstore/control/tests/test_governance.py": "d366b437d52c49e48a3bee02073875ca898121fdc451f2da087e4f5555d8fe29",
    "tests/test_remote_memory.py": None,
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
            if found not in ((before,) if isinstance(before, str) else before):
                raise SystemExit(f"refusing: {relative} drifted (sha256 {found[:16]})")
        crlf = current is not None and b"\r\n" in current
        planned.append((target, current, new.replace(b"\n", b"\r\n") if crlf else new))
        print(f"{'installing' if apply else 'would install'}  {relative} ({'CRLF' if crlf else 'LF'})")
    if not apply:
        return
    for target, current, data in planned:
        if current is not None:
            target.with_name(f"{target.name}.bak-{stamp}-r5").write_bytes(current)
        target.write_bytes(data)
    print(f"applied {len(planned)} file(s); backups *.bak-{stamp}-r5")


if __name__ == "__main__":
    main()
