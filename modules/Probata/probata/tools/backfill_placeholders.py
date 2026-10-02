#!/usr/bin/env python3
"""Back-fill a placeholder person for every number that STILL has no person, through the governed API.

Byline: Claude Code · Sonnet · 2026-10-02

Owner 2026-10-02 14:32: no party is ever NULL. Owner 15:34: run tools/import_contacts.py FIRST, because a
number a contact names becomes a person with that name, not a placeholder. This reads every phone number in
the imported calls and messages that no registry person carries (GET /api/imported/unknown-numbers?kind=no_person;
the contacts import has already given a person to every number it names), then asks the Workbench's
case-identity API (POST /api/case-identity/placeholders -> the engine) to create a placeholder person
for each, 200 numbers per request. The engine links every NULL call_log / message-participant entity
column for those numbers in the same transaction and appends registry.identity_change rows. A number
that something already carries is skipped, so the script is safe to re-run.

Dry run (default) does everything in the engine and rolls back; it only reports counts.

    python3 tools/backfill_placeholders.py                 # dry run, prints the counts
    python3 tools/backfill_placeholders.py --apply         # the real run

Standard library only. The base URL is the Workbench's tailnet door by default (no secrets involved).
"""

from __future__ import annotations

import argparse
import json
import sys
import time
import urllib.error
import urllib.request

DEFAULT_BASE = "https://workbench.tilapia-skilift.ts.net"
BATCH = 200
REASON = "back-fill: number appears in imported calls or messages and no person carried it (owner 2026-10-02)"


def call(base: str, method: str, path: str, body: dict | None = None, key: str | None = None) -> dict:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Idempotency-Key"] = key
    data = json.dumps(body).encode() if body is not None else None
    request = urllib.request.Request(base + path, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(request, timeout=300) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        detail = error.read().decode(errors="replace")[:400]
        raise SystemExit(f"{method} {path} -> HTTP {error.code}: {detail}") from None


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--apply", action="store_true", help="write for real (default is a dry run that rolls back)")
    parser.add_argument("--base", default=DEFAULT_BASE)
    parser.add_argument("--limit", type=int, default=0, help="stop after this many numbers (0 = all)")
    args = parser.parse_args()

    numbers: list[str] = []
    offset = 0
    while True:
        page = call(args.base, "GET", f"/api/imported/unknown-numbers?kind=no_person&limit=1000&offset={offset}")
        numbers += [item["number"] for item in page["items"]]
        if page.get("next_offset") is None:
            break
        offset = page["next_offset"]
    if args.limit:
        numbers = numbers[: args.limit]
    print(f"{len(numbers)} numbers still have no person ({'APPLY' if args.apply else 'dry run'})")

    totals = {"created": 0, "already_carried": 0, "not_a_phone_number": 0}
    linked: dict[str, int] = {}
    stamp = time.strftime("%Y%m%dT%H%M%S")
    for index in range(0, len(numbers), BATCH):
        chunk = numbers[index: index + BATCH]
        receipt = call(
            args.base, "POST", "/api/case-identity/placeholders",
            {"numbers": chunk, "change_reason": REASON, "dry_run": not args.apply},
            key=f"backfill-{stamp}-{index // BATCH:04d}",
        )
        detail = receipt.get("detail") or {}
        for name in totals:
            totals[name] += int(detail.get(name, 0))
        for name, count in (detail.get("linked") or {}).items():
            linked[name] = linked.get(name, 0) + int(count)
        print(f"  batch {index // BATCH + 1}: created {detail.get('created', 0)}, skipped {detail.get('already_carried', 0)}")
    print(json.dumps({"mode": "apply" if args.apply else "dry_run", **totals, "rows_linked": linked}, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
