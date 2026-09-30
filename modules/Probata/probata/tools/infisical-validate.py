"""Validate the credentials held in Infisical, and mark the ones their service rejects.

Owner, 2026-09-28: once the secrets live in the manager, the validator reads them from there --
so it is not tied to one machine and can run on a schedule.

That is the difference from secrets-validate.py, which walks ~/.secrets. This one asks Infisical,
which the tool-runtime container reaches on the tailnet, so it works from the desktop today and
from the VPS unchanged. It is also dynamic: a credential added to Infisical tomorrow is validated
on the next run without anyone editing a list.

Probe logic is NOT duplicated. Both callers import secret_probes, because two copies of a rule
about what counts as revoked will disagree eventually, and the disagreement will be silent.

Each Infisical folder is treated as one credential group -- /coolify/<app>, /desktop/<file>,
/google/<account> -- so a probe can find the partner value it needs: an R2 id beside its secret,
a Tailscale client id beside its OAuth secret.

WHAT IT DOES WITH A DEAD ONE

    --report        (default) prints the verdict table and writes JSON; changes nothing
    --annotate      sets a `validation` tag and a dated comment on the dead secrets in Infisical

It never deletes from Infisical and never quarantines: a value there may still be the only copy.
Marking it makes it visible; removing it is the owner's call, the same rule the filesystem
quarantine follows.

Usage:
    python tools/infisical-validate.py
    python tools/infisical-validate.py --annotate
    python tools/infisical-validate.py --folder /coolify

Byline: Claude Code · Opus 5 · 2026-09-28
"""
from __future__ import annotations

import argparse
import concurrent.futures
import hashlib
import json
import os
import pathlib
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import secret_probes  # noqa: E402

ENVIRONMENT = os.environ.get("INFISICAL_ENVIRONMENT", "prod")
PROJECT_NAME = os.environ.get("INFISICAL_PROJECT", "propria")


def digest(value: str) -> str:
    return hashlib.sha256(value.encode()).hexdigest()[:12]


def settings() -> dict[str, str]:
    """Configuration from the environment first, so the container needs no file."""
    found = {k: os.environ[k] for k in
             ("INFISICAL_URL", "INFISICAL_MACHINE_TOKEN", "INFISICAL_ORG_ID") if k in os.environ}
    if len(found) == 3:
        return found
    path = pathlib.Path.home() / ".secrets" / "infisical.env"
    if not path.is_file():
        raise SystemExit("  set INFISICAL_URL, INFISICAL_MACHINE_TOKEN and INFISICAL_ORG_ID, "
                         "or provide ~/.secrets/infisical.env")
    for line in path.read_text(encoding="utf-8", errors="ignore").splitlines():
        m = re.match(r"^\s*([A-Z_]+)\s*=\s*(.+?)\s*$", line)
        if m:
            found.setdefault(m.group(1), m.group(2).strip().strip('"').strip("'"))
    return found


class Infisical:
    def __init__(self) -> None:
        conf = settings()
        self.base = conf["INFISICAL_URL"].rstrip("/")
        self.token = conf["INFISICAL_MACHINE_TOKEN"]
        self.org = conf["INFISICAL_ORG_ID"]
        self.project = self._project()

    def call(self, path, method="GET", body=None):
        data = json.dumps(body).encode() if body is not None else None
        request = urllib.request.Request(
            self.base + path, data=data, method=method,
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + self.token})
        try:
            with urllib.request.urlopen(request, timeout=60) as response:
                raw = response.read()
                return response.status, (json.loads(raw) if raw[:1] in (b"{", b"[") else raw[:200])
        except urllib.error.HTTPError as exc:
            return exc.code, exc.read()[:300]

    def _project(self) -> str:
        status, out = self.call(f"/api/v2/organizations/{self.org}/workspaces")
        rows = (out.get("workspaces") or []) if isinstance(out, dict) else (out or [])
        match = next((w for w in rows if w.get("name") == PROJECT_NAME), None)
        if not match:
            raise SystemExit(f"  project {PROJECT_NAME} not found in Infisical (HTTP {status})")
        return match.get("id") or match.get("_id")

    def folders(self, path="/") -> list[str]:
        """Every folder, walked breadth-first; Infisical lists one level at a time."""
        found, queue = [], [path]
        while queue:
            current = queue.pop(0)
            status, out = self.call(
                f"/api/v1/folders?workspaceId={self.project}&environment={ENVIRONMENT}"
                f"&path={urllib.parse.quote(current, safe='')}")
            if status != 200:
                continue
            for folder in (out.get("folders") or []) if isinstance(out, dict) else []:
                child = (current.rstrip("/") + "/" + folder["name"]) or "/"
                found.append(child)
                queue.append(child)
        return found

    def secrets(self, path: str) -> dict[str, str]:
        status, out = self.call(
            f"/api/v3/secrets/raw?workspaceId={self.project}&environment={ENVIRONMENT}"
            f"&secretPath={urllib.parse.quote(path, safe='')}")
        if status != 200 or not isinstance(out, dict):
            return {}
        return {s["secretKey"]: s.get("secretValue", "") for s in out.get("secrets", [])}

    def annotate(self, path: str, name: str, note: str):
        return self.call(f"/api/v3/secrets/raw/{name}", "PATCH", {
            "workspaceId": self.project, "environment": ENVIRONMENT,
            "secretPath": path, "secretComment": note, "tagSlugs": ["validation-dead"]})


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--annotate", action="store_true",
                        help="tag and comment the dead ones in Infisical (default: report only)")
    parser.add_argument("--folder", default="/", help="limit to one subtree")
    parser.add_argument("--out", default=str(pathlib.Path(__file__).parent / "infisical-validation.json"))
    args = parser.parse_args()

    api = Infisical()
    paths = [args.folder] + [p for p in api.folders(args.folder)]
    groups = {path: api.secrets(path) for path in paths}
    total = sum(len(v) for v in groups.values())
    print(f"  {total} secrets across {len([p for p in groups if groups[p]])} folders "
          f"in {PROJECT_NAME}/{ENVIRONMENT}")
    print(f"  {secret_probes.FAMILIES} probe families available")

    jobs = [(path, name) for path, group in groups.items() for name in group
            if secret_probes.pick(name) is not None]
    print(f"  {len(jobs)} testable")

    results: dict[tuple[str, str], str] = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
        futures = {pool.submit(secret_probes.check, groups[path], name): (path, name)
                   for path, name in jobs}
        for future in concurrent.futures.as_completed(futures):
            key = futures[future]
            try:
                results[key] = future.result()
            except Exception:  # noqa: BLE001
                results[key] = "unreachable"

    tally: dict[str, int] = {}
    report = []
    for (path, name), state in sorted(results.items()):
        tally[state] = tally.get(state, 0) + 1
        report.append({"path": path, "name": name, "verdict": state,
                       "sha256": digest(groups[path][name])})
    untested = total - len(jobs)

    print("\n  verdicts:")
    for state in ("live", "dead", "unreachable", "untestable"):
        if tally.get(state):
            print(f"    {state:12} {tally[state]}")
    print(f"    {'no probe':12} {untested}")

    dead = [r for r in report if r["verdict"] == "dead"]
    if dead:
        print(f"\n  DEAD in Infisical ({len(dead)}):")
        for row in dead[:30]:
            print(f"    {row['path']}/{row['name']}  sha256:{row['sha256']}")

    if args.annotate and dead:
        marked = 0
        for row in dead:
            status, _ = api.annotate(row["path"], row["name"],
                                     "validation: rejected by its service on this run")
            marked += status in (200, 201)
        print(f"  annotated {marked} of {len(dead)} in Infisical (nothing deleted)")

    pathlib.Path(args.out).write_text(json.dumps(report, indent=1), encoding="utf-8")
    print(f"\n  report written to {args.out} (paths, names and hashes only)")
    return 1 if dead else 0


if __name__ == "__main__":
    sys.exit(main())
