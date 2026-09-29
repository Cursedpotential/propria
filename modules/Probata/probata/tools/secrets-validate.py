"""Probe every credential in ~/.secrets against its real service, before anything is migrated.

Owner, 2026-09-28: "You're gonna need to actually validate all of the ones in the dot secrets
directory... Don't add fucking stale shit."

Filtering by filename -- skipping .bak and STALE -- is a guess, not validation. A revoked key in a
normally named file looks identical to a live one, and the first pass proved it: the live
cloudflare.env holds a CLOUDFLARE_API_TOKEN the service rejects.

Each probe receives the WHOLE env dict of the file a key came from, because most credentials are
pairs -- R2 needs an id and a secret, Tailscale an OAuth client id and secret, Backblaze a key id
beside the application key. Probing values in isolation is why 678 entries had no probe on the
first pass.

VERDICTS
    live        the service accepted it
    dead        the service rejected it: 401, 403, or an explicit invalid-credential answer
    unreachable the service could not be contacted, so the credential is unproven either way
    untestable  no read-only probe exists for this kind, or it is not a credential at all

Only `live` is safe to migrate. `dead` is what makes a secrets directory untrustworthy.
`unreachable` is kept apart from `dead` on purpose: an unproven credential is not a revoked one,
and treating it as revoked would delete working keys.

Every probe is read-only -- list models, fetch the current user, read a version. Nothing is
created, modified or deleted, and no value is ever printed: the report carries names, lengths and
sha256 prefixes. Identical values are probed once however many files repeat them.

Usage:
    python tools/secrets-validate.py
    python tools/secrets-validate.py --kind nvidia
    python tools/secrets-validate.py --out report.json

Byline: Claude Code · Opus 5 · 2026-09-28
"""
from __future__ import annotations

import argparse
import base64
import concurrent.futures
import hashlib
import json
import pathlib
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

HOME = pathlib.Path.home()
SECRETS_DIR = HOME / ".secrets"
RCLONE = pathlib.Path("C:/Users/matts/scoop/shims/rclone.exe")
TIMEOUT = 25

PLACEHOLDER = re.compile(
    r"^(changeme|change_me|replace(_me)?|x{3,}|your[_-]?\w+|<.*>|true|false|\d{1,5})$", re.I)
QUARANTINE = re.compile(r"^(_dead-\d|to_be_deleted/)")
NOT_A_SECRET = re.compile(
    r"^([A-Z0-9_]*_)?(URL|URI|HOST|PORT|FQDN|ENDPOINT|REGION|BUCKET|PROJECT|PROJECT_ID|"
    r"ACCOUNT_ID|EMAIL|USER|USERNAME|DIR|PATH|MODEL|VERSION|ENABLED|TIMEOUT|NAMESPACE|DB|"
    r"DATABASE|COLLECTION|INDEX|TZ|LANG|ENV|NAME|PREFIX|MODE|LEVEL|FORMAT)$")


import sys as _sys
_sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import secret_probes  # noqa: E402

# The probe registry lives in secret_probes so this tool and infisical-validate.py cannot drift
# apart on what counts as revoked. Everything below is only about reading ~/.secrets.
digest = lambda value: __import__("hashlib").sha256(value.encode()).hexdigest()[:12]
PLACEHOLDER = secret_probes.PLACEHOLDER
NOT_A_SECRET = secret_probes.NOT_A_SECRET
pick_probe = secret_probes.pick


def collect() -> list[tuple[str, str, str, dict]]:
    """(file, name, value, the whole env of that file) for every credential-looking pair."""
    found = []
    for path in sorted(p for p in SECRETS_DIR.rglob("*") if p.is_file()):
        relative = path.relative_to(SECRETS_DIR).as_posix()
        # Skip the quarantine this tool's sibling creates. Re-validating the copies it holds
        # reports its own dead credentials back as findings, which reads as if nothing was fixed.
        if QUARANTINE.match(relative) or path.suffix.lower() == ".json":
            continue
        try:
            text = path.read_text(encoding="utf-8", errors="ignore")
        except Exception:
            continue
        env: dict[str, str] = {}
        for line in text.splitlines():
            if line.lstrip().startswith("#"):
                continue
            m = re.match(r"^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$", line)
            if not m:
                continue
            value = m.group(2).strip()
            if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
                value = value[1:-1]
            if value and not PLACEHOLDER.match(value):
                env[m.group(1)] = value
        for name, value in env.items():
            found.append((relative, name, value, env))
    return found


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--kind", default=None, help="only probe names matching this regex")
    parser.add_argument("--out",
                        default=str(pathlib.Path(__file__).parent / "secrets-validation.json"))
    args = parser.parse_args()

    entries = collect()
    print(f"  {len(entries)} key=value pairs across ~/.secrets")

    testable, skipped = [], []
    for relative, name, value, env in entries:
        if NOT_A_SECRET.match(name):
            skipped.append((relative, name, value, "not a credential"))
            continue
        probe = pick_probe(name)
        if probe is None:
            skipped.append((relative, name, value, "no read-only probe for this kind"))
            continue
        if args.kind and not re.search(args.kind, name, re.I):
            continue
        testable.append((relative, name, value, env, probe))

    by_value: dict[str, list] = {}
    for relative, name, value, env, probe in testable:
        by_value.setdefault(value, []).append((relative, name, env, probe))
    print(f"  {len(testable)} probeable entries, {len(by_value)} distinct values to call")

    results: dict[str, str] = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
        futures = {}
        for value, rows in by_value.items():
            _, name, env, probe = rows[0]
            futures[pool.submit(secret_probes.check, env, name)] = value
        for future in concurrent.futures.as_completed(futures):
            value = futures[future]
            try:
                results[value] = future.result()
            except Exception:  # noqa: BLE001
                results[value] = "unreachable"

    tally: dict[str, int] = {}
    report = []
    for value, rows in by_value.items():
        state = results.get(value, "unreachable")
        tally[state] = tally.get(state, 0) + 1
        for relative, name, _, _ in rows:
            report.append({"file": relative, "name": name, "len": len(value),
                           "sha256": digest(value), "verdict": state})

    print("\n  verdicts, by distinct value:")
    for state in ("live", "dead", "unreachable", "untestable"):
        if tally.get(state):
            print(f"    {state:12} {tally[state]}")
    print(f"    {'no probe':12} {len(skipped)}  (unprobeable kind, or not a credential)")

    dead_files = sorted({r["file"] for r in report if r["verdict"] == "dead"})
    if dead_files:
        print(f"\n  DEAD credentials appear in {len(dead_files)} files:")
        for name in dead_files[:30]:
            keys = sorted({r["name"] for r in report if r["file"] == name and r["verdict"] == "dead"})
            print(f"    {name:52} {', '.join(keys)[:60]}")
        if len(dead_files) > 30:
            print(f"    … and {len(dead_files) - 30} more")

    out = pathlib.Path(args.out)
    out.write_text(json.dumps(
        {"probed": report,
         "unprobed": [{"file": f, "name": n, "len": len(v), "why": w} for f, n, v, w in skipped]},
        indent=1), encoding="utf-8")
    print(f"\n  report written to {out} (names, lengths and hashes only)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
