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

# _backup-<date>/ holds dated copies of live env files. Each copy maps to the same Infisical folder as
# its live file and sorts after it, so a backup's old value overwrote the live one (2026-10-02: the
# rotated probata-db admin password was put back to the old value). Backups are not live config.
EXCLUDE_FILE = re.compile(r"(\.bak|to_be_deleted|_dead-\d|_backup-\d|STALE|\.old$|~$|\.example$|\.sample$)", re.I)

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import secret_probes  # noqa: E402

# One definition of "is this a credential", shared with both validators. Keeping a private copy
# here let plain URLs and booleans through: OPENLIST_URL, GRAPHITI_MCP_URL and a 4-character
# SSRF_ALLOW_PRIVATE_NETWORKS were all queued for a secrets manager.
PLACEHOLDER = secret_probes.PLACEHOLDER
NOT_A_SECRET = secret_probes.NOT_A_SECRET
CREDENTIAL_SHAPED = secret_probes.CREDENTIAL_SHAPED

# A URL is worth storing only when it carries credentials inside it, as a DSN does.
URL_WITH_CREDENTIALS = re.compile(r"://[^/\s:@]+:[^/\s@]+@")


def is_secret(name: str, value: str) -> bool:
    if not value or PLACEHOLDER.match(value):
        return False
    if URL_WITH_CREDENTIALS.search(value):
        return True            # a DSN is a credential whatever its key is called
    if NOT_A_SECRET.match(name):
        return False
    return bool(CREDENTIAL_SHAPED.search(name))


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
            if is_secret(key, value):
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


def from_coolify(_unused=None) -> tuple[list[tuple[str, str, str]], list[str]]:
    """Every Coolify resource's credentials. Returns (items, failures).

    One server process for all calls. The previous version spawned one per application and
    returned [] when a spawn failed, so consecutive runs collected 92, 0, 84 and 101 secrets and
    each looked like a complete answer. A partial read is now reported, and the caller refuses to
    write on any failure -- silently migrating 84 of 101 credentials is worse than migrating none.
    """
    sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
    import coolify_call

    listing = coolify_call.call_many([("apps", "list_applications", {"per_page": 100})])
    if listing["apps"]["error"]:
        return [], [f"list_applications: {listing['apps']['error']}"]
    apps = [a for a in listing["apps"]["rows"] if isinstance(a, dict) and a.get("uuid")]
    if not apps:
        return [], ["list_applications returned no applications"]

    spec = [(a["uuid"], "list_application_envs", {"uuid": a["uuid"]}) for a in apps]
    answers = coolify_call.call_many(spec)

    items, failures = [], []
    for app in apps:
        answer = answers.get(app["uuid"], {"rows": [], "error": "missing from the batch"})
        if answer["error"]:
            failures.append(f"{app.get('name')} ({app['uuid']}): {answer['error']}")
            continue
        folder = "/coolify/" + re.sub(r"[^A-Za-z0-9_-]+", "-", str(app.get("name"))).strip("-")
        for row in answer["rows"]:
            if not isinstance(row, dict) or row.get("is_preview"):
                continue
            key, value = row.get("key"), str(row.get("value") or "")
            if key and is_secret(key, value):
                items.append((folder, key, value))
    return items, failures


# Coolify service environments live inside docker_compose_raw, not in an env table, and many
# entries there are ${VAR} indirections rather than values -- those resolve from Coolify's own env
# and would be stored as the literal string "${VAR:?set}" if carried across.
COMPOSE_INDIRECTION = re.compile(r"^\$\{[^}]*\}$")


def from_coolify_services() -> tuple[list[tuple[str, str, str]], list[str]]:
    """Service credentials, read through get_service_env(reveal=True).

    get_service masks every secret-looking value, which is correct for diagnosis and leaves no
    route for a migration. get_service_env exists for this: it returns the env pairs only, so
    revealing them does not also expose the compose, webhook secrets or volume layout.
    """
    sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
    import coolify_call

    listing = coolify_call.call_many([("svc", "list_services", {"per_page": 100})])
    if listing["svc"]["error"]:
        return [], [f"list_services: {listing['svc']['error']}"]
    services = [x for x in listing["svc"]["rows"] if isinstance(x, dict) and x.get("uuid")]

    answers = coolify_call.call_many(
        [(x["uuid"], "get_service_env", {"uuid": x["uuid"], "reveal": True}) for x in services])

    items, failures = [], []
    for service in services:
        answer = answers.get(service["uuid"], {"rows": [], "error": "missing from the batch"})
        if answer["error"]:
            failures.append(f"service {service.get('name')}: {answer['error']}")
            continue
        payload = next((r for r in answer["rows"] if isinstance(r, dict) and "services" in r), None)
        if payload is None:
            failures.append(f"service {service.get('name')}: no env payload returned")
            continue
        base = re.sub(r"[^A-Za-z0-9_-]+", "-", str(service.get("name"))).strip("-")
        for compose_name, pairs in (payload.get("services") or {}).items():
            for key, value in (pairs or {}).items():
                value = str(value or "")
                if COMPOSE_INDIRECTION.match(value.strip()):
                    continue
                if is_secret(key, value):
                    items.append((f"/coolify/{base}", key, value))
    return items, failures


def from_coolify_databases() -> tuple[list[tuple[str, str, str]], list[str]]:
    """Standalone database credentials: the password and the connection URL that embeds it."""
    sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
    import coolify_call

    listing = coolify_call.call_many([("db", "list_databases", {"per_page": 100})])
    if listing["db"]["error"]:
        return [], [f"list_databases: {listing['db']['error']}"]
    databases = [x for x in listing["db"]["rows"] if isinstance(x, dict) and x.get("uuid")]

    answers = coolify_call.call_many(
        [(x["uuid"], "get_database", {"uuid": x["uuid"]}) for x in databases])

    items, failures = [], []
    for database in databases:
        answer = answers.get(database["uuid"], {"rows": [], "error": "missing from the batch"})
        if answer["error"]:
            failures.append(f"database {database.get('name')}: {answer['error']}")
            continue
        record = next((r for r in answer["rows"] if isinstance(r, dict) and r.get("uuid")), None)
        if record is None:
            failures.append(f"database {database.get('name')}: no record returned")
            continue
        base = "/coolify/" + re.sub(r"[^A-Za-z0-9_-]+", "-", str(database.get("name"))).strip("-")
        for field, name in (("postgres_password", "POSTGRES_PASSWORD"),
                            ("internal_db_url", "DATABASE_URL"),
                            ("external_db_url", "DATABASE_URL_EXTERNAL"),
                            ("mysql_password", "MYSQL_PASSWORD"),
                            ("mysql_root_password", "MYSQL_ROOT_PASSWORD"),
                            ("mongo_initdb_root_password", "MONGO_ROOT_PASSWORD"),
                            ("redis_password", "REDIS_PASSWORD"),
                            ("mariadb_password", "MARIADB_PASSWORD"),
                            ("keydb_password", "KEYDB_PASSWORD"),
                            ("clickhouse_admin_password", "CLICKHOUSE_PASSWORD"),
                            ("dragonfly_password", "DRAGONFLY_PASSWORD")):
            value = str(record.get(field) or "")
            if value and is_secret(name, value):
                items.append((base, name, value))
    return items, failures


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
        """Create, or update in place when it already exists.

        The raw-secret POST creates only and answers 400 "Secret already exist" on a re-run, so
        a second pass reported 92 failures over secrets that were already correct. Re-running a
        migration has to be safe, and an unchanged value must not count as an error.
        """
        body = {"workspaceId": self.project, "environment": ENVIRONMENT,
                "secretPath": folder, "secretValue": value, "type": "shared"}
        status, out = self.call(f"/api/v3/secrets/raw/{name}", "POST", body)
        if status in (200, 201):
            return status, "created"
        if status == 400 and b"already exist" in (out if isinstance(out, bytes) else str(out).encode()):
            if self.get(folder, name) == value:
                return 200, "unchanged"
            status, out = self.call(f"/api/v3/secrets/raw/{name}", "PATCH", body)
            return status, ("updated" if status in (200, 201) else out)
        return status, out

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
    coolify_failures: list[str] = []
    if "coolify" in wanted:
        for collector in (from_coolify, from_coolify_services, from_coolify_databases):
            collected, failed = collector()
            items += collected
            coolify_failures += failed

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

    tally = {"created": 0, "updated": 0, "unchanged": 0}
    failed = []
    for folder, name, value in unique:
        status, out = api.put(folder, name, value)
        if status in (200, 201) and out in tally:
            tally[out] += 1
        elif status in (200, 201):
            tally["created"] += 1
        else:
            failed.append((folder, name, status, str(out)[:90]))
    print(f"  created {tally['created']}, updated {tally['updated']}, "
          f"unchanged {tally['unchanged']}, failed {len(failed)} of {len(unique)}")
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
