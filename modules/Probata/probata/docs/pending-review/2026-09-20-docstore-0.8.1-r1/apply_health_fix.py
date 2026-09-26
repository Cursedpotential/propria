"""Docstore 0.8.1-r1: health stays ok when the only degradation is pending provider enrichment.

Byline: Claude Code · Fable 5.1 · 2026-09-20

Runs on the release directory (default /data/propria/releases/docstore-0.8.1).
Edits scripts/docstore/api.py IN PLACE, byte-preserving every other line (the
file has mixed CRLF/LF endings), after asserting the anchor occurs exactly once.
Keeps api.py.bak-<utc>. Idempotent: a second run reports "already applied".

    python3 apply_health_fix.py [release_dir]
"""

from __future__ import annotations

import sys
import time
from pathlib import Path

ANCHOR = b'        "ok": store == "up" and sync["sync"] not in {"failed", "degraded", "invalid"},'
MARKER = b"0.8.1-r1"


def main() -> None:
    root = Path(sys.argv[1] if len(sys.argv) > 1 else "/data/propria/releases/docstore-0.8.1")
    target = root / "scripts" / "docstore" / "api.py"
    data = target.read_bytes()
    if MARKER in data:
        print("already applied")
        return
    if data.count(ANCHOR) != 1:
        raise SystemExit(f"anchor found {data.count(ANCHOR)} times; refusing to edit")
    start = data.index(ANCHOR)
    line_start = data.rfind(b"\n", 0, data.rfind(b"\n", 0, start)) + 1  # the `return _identified({` line
    newline = b"\r\n" if data[start + len(ANCHOR): start + len(ANCHOR) + 2] == b"\r\n" else b"\n"
    block = newline.join([
        b"    # 0.8.1-r1 (Claude Code / Fable 5.1, 2026-09-20): provider enrichment is an optional",
        b"    # layer over a verified index. When the ONLY degradation is pending enrichment",
        b'    # ("Source attribution verified; provider enrichment remains pending", worker_sync.py)',
        b"    # the service is healthy; the pending count stays visible.",
        b'    enrichment_pending = len((sync.get("enrichment") or {}).get("failed_documents") or [])',
        b"    enrichment_only = (",
        b'        sync["sync"] == "degraded" and enrichment_pending > 0 and not sync.get("error_type")',
        b'        and (sync.get("cdc_attribution") or {}).get("status") == "verified"',
        b"    )",
        b"",
    ])
    replacement = (
        b'        "ok": store == "up" and (enrichment_only or sync["sync"] not in {"failed", "degraded", "invalid"}),'
        + newline + b'        "enrichment_pending": enrichment_pending,'
    )
    patched = data[:line_start] + block + data[line_start:start] + replacement + data[start + len(ANCHOR):]
    compile(patched.decode("utf-8"), str(target), "exec")
    backup = target.with_name("api.py.bak-" + time.strftime("%Y%m%dT%H%M%SZ", time.gmtime()))
    backup.write_bytes(data)
    target.write_bytes(patched)
    print(f"patched {target} (+{len(patched) - len(data)} bytes); backup {backup.name}")


if __name__ == "__main__":
    main()
