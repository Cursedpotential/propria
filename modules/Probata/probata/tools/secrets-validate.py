"""Probe every credential in ~/.secrets against its real service, before anything is migrated.

Owner, 2026-09-28: "You're gonna need to actually validate all of the ones in the dot secrets
directory... Don't add fucking stale shit."

Filtering by filename -- skipping .bak and STALE -- is a guess, not validation. A revoked key in a
normally named file looks identical to a live one. This calls each service and records what it
actually says.

VERDICTS
    live        the service accepted it
    dead        the service rejected it: 401, 403, or an explicit invalid-credential answer
    unreachable the service could not be contacted, so the credential is unproven either way
    untestable  no read-only probe exists for this kind, or it is not a credential at all

Only `live` should be migrated. `dead` is what makes a secrets directory untrustworthy and is
reported so it can be removed at the source. `unreachable` and `untestable` are listed separately
rather than being quietly folded into either pile.

Every probe is read-only: list models, fetch the current user, read a version. Nothing is created,
modified or deleted, and no value is ever printed -- the report carries names, lengths and sha256
prefixes. Identical values across files are probed once.

Usage:
    python tools/secrets-validate.py                    # probe everything, write a report
    python tools/secrets-validate.py --kind nvidia      # one family
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
import urllib.request

HOME = pathlib.Path.home()
SECRETS_DIR = HOME / ".secrets"
TIMEOUT = 25

EXCLUDE_FILE = re.compile(r"(to_be_deleted/)", re.I)
PLACEHOLDER = re.compile(r"^(changeme|change_me|replace(_me)?|x{3,}|your[_-]?\w+|<.*>|true|false|\d{1,5})$", re.I)
NOT_A_SECRET = re.compile(
    r"^([A-Z0-9_]*_)?(URL|URI|HOST|PORT|FQDN|ENDPOINT|REGION|BUCKET|PROJECT|PROJECT_ID|"
    r"ACCOUNT_ID|EMAIL|USER|USERNAME|DIR|PATH|MODEL|VERSION|ENABLED|TIMEOUT|NAMESPACE|DB|"
    r"DATABASE|COLLECTION|INDEX|TZ|LANG|ENV)$")


def digest(value: str) -> str:
    return hashlib.sha256(value.encode()).hexdigest()[:12]


def get(url: str, headers: dict, method: str = "GET", data: bytes | None = None) -> tuple[int, str]:
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=TIMEOUT) as response:
            return response.status, response.read(400).decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read(300).decode("utf-8", "replace")
    except Exception as exc:  # noqa: BLE001 - network reachability, not a credential verdict
        return 0, type(exc).__name__


def verdict_from(status: int, body: str) -> str:
    if status == 0:
        return "unreachable"
    if status in (401, 403):
        return "dead"
    if 200 <= status < 300:
        return "live"
    if status == 404:
        return "live"          # authenticated but the path does not exist
    if status in (429,):
        return "live"          # rate limited means the credential was accepted
    if re.search(r"invalid.*(key|token|credential)|unauthor", body, re.I):
        return "dead"
    return "unreachable"


# --- probes ----------------------------------------------------------------------------------

def probe_nvidia(value: str) -> str:
    return verdict_from(*get("https://integrate.api.nvidia.com/v1/models",
                             {"Authorization": f"Bearer {value}"}))


def probe_openrouter(value: str) -> str:
    return verdict_from(*get("https://openrouter.ai/api/v1/key",
                             {"Authorization": f"Bearer {value}"}))


def probe_openai(value: str) -> str:
    return verdict_from(*get("https://api.openai.com/v1/models",
                             {"Authorization": f"Bearer {value}"}))


def probe_anthropic(value: str) -> str:
    return verdict_from(*get("https://api.anthropic.com/v1/models",
                             {"x-api-key": value, "anthropic-version": "2023-06-01"}))


def probe_github(value: str) -> str:
    return verdict_from(*get("https://api.github.com/user",
                             {"Authorization": f"Bearer {value}",
                              "Accept": "application/vnd.github+json"}))


def probe_voyage(value: str) -> str:
    return verdict_from(*get("https://api.voyageai.com/v1/embeddings",
                             {"Authorization": f"Bearer {value}", "Content-Type": "application/json"},
                             "POST", json.dumps({"input": ["ping"], "model": "voyage-3"}).encode()))


def probe_coolify(value: str) -> str:
    return verdict_from(*get("http://100.98.98.38:8000/api/v1/version",
                             {"Authorization": f"Bearer {value}"}))


def probe_tavily(value: str) -> str:
    return verdict_from(*get("https://api.tavily.com/search",
                             {"Content-Type": "application/json"}, "POST",
                             json.dumps({"api_key": value, "query": "ping", "max_results": 1}).encode()))


def probe_cloudflare(value: str) -> str:
    return verdict_from(*get("https://api.cloudflare.com/client/v4/user/tokens/verify",
                             {"Authorization": f"Bearer {value}"}))


PROBES = [
    (re.compile(r"NVIDIA|NIM_", re.I), probe_nvidia),
    (re.compile(r"OPENROUTER", re.I), probe_openrouter),
    (re.compile(r"^OPENAI_API_KEY$|OPENAI.*KEY", re.I), probe_openai),
    (re.compile(r"ANTHROPIC", re.I), probe_anthropic),
    (re.compile(r"GITHUB.*(TOKEN|PAT)|GH_TOKEN", re.I), probe_github),
    (re.compile(r"VOYAGE", re.I), probe_voyage),
    (re.compile(r"COOLIFY_API_TOKEN", re.I), probe_coolify),
    (re.compile(r"TAVILY", re.I), probe_tavily),
    (re.compile(r"CLOUDFLARE.*(TOKEN|KEY)|CF_API_TOKEN", re.I), probe_cloudflare),
]


def pick_probe(name: str):
    for pattern, probe in PROBES:
        if pattern.search(name):
            return probe
    return None


def collect() -> list[tuple[str, str, str]]:
    found = []
    for path in sorted(p for p in SECRETS_DIR.rglob("*") if p.is_file()):
        relative = path.relative_to(SECRETS_DIR).as_posix()
        if EXCLUDE_FILE.search(relative) or path.suffix.lower() == ".json":
            continue
        try:
            text = path.read_text(encoding="utf-8", errors="ignore")
        except Exception:
            continue
        for line in text.splitlines():
            if line.lstrip().startswith("#"):
                continue
            m = re.match(r"^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$", line)
            if not m:
                continue
            name, value = m.group(1), m.group(2).strip()
            if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
                value = value[1:-1]
            if not value or PLACEHOLDER.match(value):
                continue
            found.append((relative, name, value))
    return found


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--kind", default=None, help="only probe names matching this regex")
    parser.add_argument("--out", default=str(pathlib.Path(__file__).parent / "secrets-validation.json"))
    args = parser.parse_args()

    entries = collect()
    print(f"  {len(entries)} key=value pairs in ~/.secrets (excluding quarantine and JSON)")

    testable, skipped = [], []
    for relative, name, value in entries:
        if NOT_A_SECRET.match(name):
            skipped.append((relative, name, value, "not a credential"))
            continue
        probe = pick_probe(name)
        if probe is None:
            skipped.append((relative, name, value, "no read-only probe for this kind"))
            continue
        if args.kind and not re.search(args.kind, name, re.I):
            continue
        testable.append((relative, name, value, probe))

    # Identical values appear in many files; probe each distinct value once.
    by_value: dict[str, list] = {}
    for relative, name, value, probe in testable:
        by_value.setdefault(value, []).append((relative, name, probe))
    print(f"  {len(testable)} probeable entries, {len(by_value)} distinct values to call")

    results: dict[str, str] = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=6) as pool:
        futures = {pool.submit(rows[0][2], value): value for value, rows in by_value.items()}
        for future in concurrent.futures.as_completed(futures):
            value = futures[future]
            try:
                results[value] = future.result()
            except Exception as exc:  # noqa: BLE001
                results[value] = "unreachable"

    tally: dict[str, int] = {}
    report = []
    for value, rows in by_value.items():
        state = results.get(value, "unreachable")
        tally[state] = tally.get(state, 0) + 1
        for relative, name, _ in rows:
            report.append({"file": relative, "name": name, "len": len(value),
                           "sha256": digest(value), "verdict": state})

    print("\n  verdicts by distinct value:")
    for state in ("live", "dead", "unreachable"):
        if tally.get(state):
            print(f"    {state:12} {tally[state]}")
    print(f"    {'untestable':12} {len(skipped)}  (no probe, or not a credential)")

    dead = [r for r in report if r["verdict"] == "dead"]
    if dead:
        print("\n  DEAD — do not migrate, remove at the source:")
        for r in sorted(dead, key=lambda r: r["file"])[:40]:
            print(f"    {r['file']:38} {r['name']:30} sha256:{r['sha256']}")
        if len(dead) > 40:
            print(f"    … and {len(dead) - 40} more")

    out = pathlib.Path(args.out)
    out.write_text(json.dumps({"probed": report,
                               "untestable": [{"file": f, "name": n, "len": len(v), "why": w}
                                              for f, n, v, w in skipped]}, indent=1), encoding="utf-8")
    print(f"\n  report written to {out} (names, lengths and hashes only)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
