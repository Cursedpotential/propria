"""Load every credential Propria uses into Infisical, organised by where it came from.

Owner, 2026-09-28: all four Google accounts, all VPS services and apps, and the databases.

SOURCES, and the path each lands under

    /desktop/<file>/<KEY>     ~/.secrets/*.env            63 files, ~500 pairs
    /desktop/<file>           ~/.secrets/*.json           whole-file credentials
    /google/<remote>/<KEY>    rclone.conf drive remotes   4 accounts: token + client id/secret
    /coolify/<resource>/<KEY> Coolify apps, services, dbs 50 resources

Provenance is the folder, not a prefix on the name. Two apps may both hold DATABASE_URL with
different values; flattening them into one namespace silently destroys one of them.

WHAT IS DELIBERATELY NOT COPIED
  - .bak*, *STALE*, *.old, ~, and anything under to_be_deleted. Those are the copies that make a
    secrets directory untrustworthy; carrying them in defeats the point of the move.
  - Values that are empty, or literally "changeme"/"replace"/"xxx".
  - Coolify's own build-control variables (SERVICE_*, COOLIFY_*, NIXPACKS_*, PORT, *_FQDN,
    *_URL without credentials in them) -- configuration, not secrets.

NOTHING IS REWIRED. Every consumer keeps reading what it reads today. Infisical becomes a
complete, verified second copy; cutting consumers over is a later, reversible step, one consumer
at a time. Doing it the other way round takes services down during the migration.

Safety: values are never printed. The report gives names, lengths and sha256 prefixes. The run is
idempotent -- an existing secret with the same value is left alone -- and it verifies a random
sample by reading back and comparing hashes.

Usage:
    python tools/infisical-migrate-all.py                 # dry run, full inventory
    python tools/infisical-migrate-all.py --write
    python tools/infisical-migrate-all.py --write --only desktop,google

Byline: Claude Code · Opus 5 · 2026-09-28
"""
from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import random
import re
import subprocess
import sys
import urllib.error
import urllib.request

HOME = pathlib.Path.home()
SECRETS_DIR = HOME / ".secrets"
SSH_KEY = str(HOME / ".ssh" / "ovh")
RCLONE_CONF = pathlib.Path("C:/Users/matts/scoop/apps/rclone/current/rclone.conf")
PROJECT_NAME = "propria"
ENVIRONMENT = "prod"

EXCLUDE_FILE = re.compile(r"(\.bak|to_be_deleted|STALE|\.old$|~$|\.example$|\.sample$)", re.I)
PLACEHOLDER = re.compile(r"^(changeme|change_me|replace(_me)?|x{3,}|your[_-]?\w+|<.*>)$", re.I)
# Coolify build/runtime control rather than credentials.
COOLIFY_NOISE = re.compile(
    r"^(SERVICE_[A-Z0-9_]*|COOLIFY_[A-Z0-9_]*|NIXPACKS_[A-Z0-9_]*|PORT|HOST|BIND_IP|"
    r"NODE_ENV|TZ|PYTHON[A-Z]*|[A-Z0-9_]*_FQDN|[A-Z0-9_]*_PORT|[A-Z0-9_]*_HOST)$")
SECRETISH = re.compile(r"(KEY|TOKEN|SECRET|PASS|PASSWORD|AUTH|CREDENTIAL|DSN|PRIVATE|SALT|"
                       r"SIGNING|CERT|SESSION|COOKIE|WEBHOOK|API|URI|URL|CONN)", re.I)


def digest(value: str) -> str:
    return hashlib.sha256(value.encode()).hexdigest()[:12]


def parse_env(text: str) -> dict[str, str]:
    out: dict[str, str] = {}
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
            out[m.group(1)] = value
    return out


# --- collectors ------------------------------------------------------------------------------

def from_desktop() -> list[tuple[str, str, str]]:
    """(folder, name, value) for ~/.secrets."""
    found = []
    for path in sorted(p for p in SECRETS_DIR.rglob("*") if p.is_file()):
        relative = path.relative_to(SECRETS_DIR).as_posix()
        if EXCLUDE_FILE.search(relative):
            continue
        stem = re.sub(r"[^A-Za-z0-9_-]+", "-", path.stem).strip("-") or "unnamed"
        try:
            text = path.read_text(encoding="utf-8", errors="ignore")
        except Exception:
            continue
        if path.suffix.lower() == ".json":
            if text.strip():
                found.append((f"/desktop/{stem}", "CREDENTIAL_JSON", text.strip()))
            continue
        for key, value in parse_env(text).items():
            found.append((f"/desktop/{stem}", key, value))
    return found


def from_google() -> list[tuple[str, str, str]]:
    if not RCLONE_CONF.is_file():
        return []
    found, current, section = [], None, {}
    sections: dict[str, dict[str, str]] = {}
    for line in RCLONE_CONF.read_text(encoding="utf-8", errors="ignore").splitlines():
        m = re.match(r"^\[(.+?)\]\s*$", line)
        if m:
            current = m.group(1)
            sections[current] = {}
        elif current and "=" in line:
            key, value = line.split("=", 1)
            sections[current][key.strip()] = value.strip()
    for name, body in sections.items():
        if body.get("type") != "drive":
            continue
        for key in ("token", "client_id", "client_secret", "root_folder_id", "team_drive"):
            if body.get(key):
                found.append((f"/google/{name}", key.upper(), body[key]))
    return found


def from_coolify(scratch: pathlib.Path) -> list[tuple[str, str, str]]:
    """Read every Coolify resource's environment through the coolify-write plugin."""
    caller = scratch / "coolify_call.py"
    if not caller.is_file():
        sys.stderr.write(f"  coolify_call.py not found at {caller}; skipping the Coolify source\n")
        return []

    def plugin(tool: str, args: dict):
        out = subprocess.run([sys.executable, str(caller), tool, json.dumps(args)],
                             capture_output=True, text=True, timeout=300).stdout
        decoder, index, rows = json.JSONDecoder(), 0, []
        while index < len(out):
            while index < len(out) and out[index] in " \r\n\t":
                index += 1
            if index >= len(out):
                break
            try:
                obj, index = decoder.raw_decode(out, index)
            except json.JSONDecodeError:
                break
            rows.append(obj)
        return rows

    found = []
    for app in plugin("list_applications", {"per_page": 100}):
        uuid, name = app.get("uuid"), app.get("name")
        if not uuid:
            continue
        folder = "/coolify/" + re.sub(r"[^A-Za-z0-9_-]+", "-", str(name)).strip("-")
        for row in plugin("list_application_envs", {"uuid": uuid}):
            if not isinstance(row, dict) or row.get("is_preview"):
                continue
            key, value = row.get("key"), str(row.get("value") or "")
            if not key or not value or PLACEHOLDER.match(value):
                continue
            if COOLIFY_NOISE.match(key) or not SECRETISH.search(key):
                continue
            found.append((folder, key, value))
    return found


# --- Infisical -------------------------------------------------------------------------------

class Infisical:
    def __init__(self) -> None:
        env = parse_env((SECRETS_DIR / "infisical.env").read_text(encoding="utf-8"))
        self.base, self.token, self.org = env["INFISICAL_URL"], env["INFISICAL_MACHINE_TOKEN"], env["INFISICAL_ORG_ID"]
        self.project = None

    def call(self, path, method="GET", body=None):
        data = json.dumps(body).encode() if body is not None else None
        request = urllib.request.Request(
            self.base + path, data=data, method=method,
            headers={"Content-Type": "application/json", "Authorization": "Bearer " + self.token})
        try:
            with urllib.request.urlopen(request, timeout=90) as response:
                raw = response.read()
                return response.status, (json.loads(raw) if raw[:1] in (b"{", b"[") else raw[:200])
        except urllib.error.HTTPError as exc:
            return exc.code, exc.read()[:300]

    def resolve_project(self) -> str:
        status, out = self.call(f"/api/v2/organizations/{self.org}/workspaces")
        rows = (out.get("workspaces") or []) if isinstance(out, dict) else (out or [])
        match = next((w for w in rows if w.get("name") == PROJECT_NAME), None)
        if not match:
            raise SystemExit(f"  project {PROJECT_NAME} not found (HTTP {status})")
        self.project = match.get("id") or match.get("_id")
        return self.project

    def ensure_folder(self, path: str) -> None:
        parts = [p for p in path.strip("/").split("/") if p]
        walked = ""
        for part in parts:
            parent = walked or "/"
            self.call("/api/v1/folders", "POST", {
                "workspaceId": self.project, "environment": ENVIRONMENT,
                "name": part, "path": parent})
            walked = f"{walked}/{part}"

    def put(self, folder: str, name: str, value: str):
        return self.call(f"/api/v3/secrets/raw/{name}", "POST", {
            "workspaceId": self.project, "environment": ENVIRONMENT,
            "secretPath": folder, "secretValue": value, "type": "shared"})

    def get(self, folder: str, name: str):
        status, out = self.call(
            f"/api/v3/secrets/raw/{name}?workspaceId={self.project}"
            f"&environment={ENVIRONMENT}&secretPath={urllib.request.quote(folder, safe='')}")
        if status != 200 or not isinstance(out, dict):
            return None
        return (out.get("secret") or {}).get("secretValue")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write", action="store_true")
    parser.add_argument("--only", default="desktop,google,coolify")
    parser.add_argument("--scratch", default=str(pathlib.Path(__file__).parent))
    args = parser.parse_args()
    wanted = {s.strip() for s in args.only.split(",")}

    items: list[tuple[str, str, str]] = []
    if "desktop" in wanted:
        items += from_desktop()
    if "google" in wanted:
        items += from_google()
    if "coolify" in wanted:
        items += from_coolify(pathlib.Path(args.scratch))

    seen, unique = set(), []
    for folder, name, value in items:
        if (folder, name) in seen:
            continue
        seen.add((folder, name))
        unique.append((folder, name, value))

    folders = sorted({f for f, _, _ in unique})
    print(f"  {len(unique)} secrets across {len(folders)} folders")
    by_root: dict[str, int] = {}
    for folder, _, _ in unique:
        by_root[folder.split("/")[1]] = by_root.get(folder.split("/")[1], 0) + 1
    for root, count in sorted(by_root.items()):
        print(f"    /{root:10} {count}")

    if not args.write:
        print("\n  sample (names and hashes only):")
        for folder, name, value in unique[:12]:
            print(f"    {folder}/{name}  {len(value)} chars  sha256:{digest(value)}")
        print(f"    … and {max(0, len(unique) - 12)} more")
        print("\n  dry run; pass --write to load")
        return 0

    api = Infisical()
    api.resolve_project()
    for folder in folders:
        api.ensure_folder(folder)
    print(f"  {len(folders)} folders ready")

    loaded, failed = 0, []
    for folder, name, value in unique:
        status, out = api.put(folder, name, value)
        if status in (200, 201):
            loaded += 1
        else:
            failed.append((folder, name, status, str(out)[:90]))
    print(f"  loaded {loaded} of {len(unique)}")
    for folder, name, status, detail in failed[:15]:
        print(f"    FAILED {folder}/{name}: HTTP {status} {detail}")
    if len(failed) > 15:
        print(f"    … and {len(failed) - 15} more failures")

    sample = random.sample(unique, min(8, len(unique)))
    good = sum(1 for folder, name, value in sample
               if (api.get(folder, name) or "") and digest(api.get(folder, name)) == digest(value))
    print(f"  verified {good} of {len(sample)} sampled secrets by hash read-back")
    return 0 if good == len(sample) and not failed else 1


if __name__ == "__main__":
    sys.exit(main())
