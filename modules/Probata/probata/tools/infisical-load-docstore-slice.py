"""Create the Propria project in Infisical and load the Docstore's secrets into it.

The first real slice, chosen because every value in it is already proven in production: the seven
the 0.9.0 Docstore runs on, plus the Coolify API token the nightly job uses. They are read from
the RUNNING container and from ~/.secrets rather than retyped, so nothing can drift in transit.

This does not change how anything currently gets its secrets. Infisical becomes a second, correct
copy first; rewiring consumers to read from it is a separate, reversible step. That ordering
matters -- a migration that cuts consumers over before the store is proven takes the service down
with it.

BIND_IP is deliberately excluded. It is configuration, not a secret, and a secrets manager that
fills up with non-secrets stops being useful.

Nothing secret is printed. The script reports names and lengths, and verifies by reading one
value back through the API and comparing hashes.

Usage:
    python tools/infisical-load-docstore-slice.py            # dry run: show what would load
    python tools/infisical-load-docstore-slice.py --write

Byline: Claude Code · Opus 5 · 2026-09-28
"""
from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import re
import subprocess
import sys
import urllib.error
import urllib.request

SECRETS = pathlib.Path.home() / ".secrets"
KEY = str(pathlib.Path.home() / ".ssh" / "ovh")
FILES_HOST = "root@100.91.190.107"
PROJECT = "propria"
ENVIRONMENT = "prod"

# Read from the running Docstore container: these are the values the service actually uses.
FROM_CONTAINER = ["SURREAL_DOCS_USER", "SURREAL_DOCS_PASS", "MEMORY_BASIC_AUTH",
                  "DOCSTORE_API_TOKEN", "DOCSTORE_CONTROL_TOKEN", "NVIDIA_API_KEY",
                  "DOCSTORE_LLM_API_KEY"]
# Read from ~/.secrets: {file: {env key: name to store under}}
FROM_FILES = {"coolify-ionos-api.env": {"COOLIFY_API_TOKEN": "COOLIFY_API_TOKEN"},
              "contextforge.env": {"CF_MCP_CLIENT_TOKEN": "CF_MCP_CLIENT_TOKEN"}}


def env_file(name: str) -> dict[str, str]:
    path = SECRETS / name
    if not path.is_file():
        return {}
    out = {}
    for line in path.read_text(encoding="utf-8", errors="ignore").splitlines():
        m = re.match(r"^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+?)\s*$", line)
        if m:
            out[m.group(1)] = m.group(2).strip().strip('"').strip("'")
    return out


class Infisical:
    def __init__(self) -> None:
        env = env_file("infisical.env")
        self.base = env["INFISICAL_URL"]
        self.token = env["INFISICAL_MACHINE_TOKEN"]
        self.org = env["INFISICAL_ORG_ID"]

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
            return exc.code, exc.read()[:400]


def digest(value: str) -> str:
    return hashlib.sha256(value.encode()).hexdigest()[:12]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write", action="store_true", help="create and load (default: dry run)")
    args = parser.parse_args()

    values: dict[str, str] = {}
    probe = subprocess.run(
        ["ssh", "-o", "BatchMode=yes", "-i", KEY, FILES_HOST,
         "C=$(docker ps --format '{{.Names}}' | grep '^docstore-' | head -1); "
         "docker exec $C printenv " + " ".join(FROM_CONTAINER)],
        capture_output=True, text=True, timeout=180)
    lines = [l for l in probe.stdout.splitlines()]
    if probe.returncode == 0 and len(lines) == len(FROM_CONTAINER):
        values.update(dict(zip(FROM_CONTAINER, (l.strip() for l in lines))))
    else:
        sys.stderr.write("  could not read the container environment; is the Docstore running?\n")
        return 1

    for filename, wanted in FROM_FILES.items():
        parsed = env_file(filename)
        for key, store_as in wanted.items():
            if parsed.get(key):
                values[store_as] = parsed[key]

    print(f"  {len(values)} secrets to load into {PROJECT}/{ENVIRONMENT}:")
    for name in sorted(values):
        print(f"    {name:26} {len(values[name]):>4} chars  sha256:{digest(values[name])}")
    if not args.write:
        print("\n  dry run; pass --write to create the project and load them")
        return 0

    api = Infisical()
    status, out = api.call("/api/v2/workspace", "POST",
                           {"projectName": PROJECT, "organizationId": api.org})
    if status in (200, 201):
        project = (out.get("project") or out.get("workspace") or out)
        project_id = project.get("id") or project.get("_id")
        print(f"  created project {PROJECT} id={project_id}")
    else:
        status, out = api.call(f"/api/v2/organizations/{api.org}/workspaces")
        existing = (out.get("workspaces") or []) if isinstance(out, dict) else out
        match = next((w for w in existing if w.get("name") == PROJECT), None)
        if not match:
            sys.stderr.write(f"  could not create or find the project: HTTP {status} {str(out)[:200]}\n")
            return 1
        project_id = match.get("id") or match.get("_id")
        print(f"  using existing project {PROJECT} id={project_id}")

    loaded, failed = 0, []
    for name, value in sorted(values.items()):
        status, out = api.call(f"/api/v3/secrets/raw/{name}", "POST", {
            "workspaceId": project_id, "environment": ENVIRONMENT,
            "secretPath": "/", "secretValue": value, "type": "shared"})
        if status in (200, 201):
            loaded += 1
        else:
            failed.append((name, status, str(out)[:120]))
    print(f"  loaded {loaded} of {len(values)}")
    for name, status, detail in failed:
        print(f"    FAILED {name}: HTTP {status} {detail}")

    # Prove it: read one back and compare hashes, without printing either value.
    check = "DOCSTORE_CONTROL_TOKEN"
    status, out = api.call(
        f"/api/v3/secrets/raw/{check}?workspaceId={project_id}"
        f"&environment={ENVIRONMENT}&secretPath=%2F")
    if status == 200:
        got = ((out.get("secret") or {}).get("secretValue")) if isinstance(out, dict) else None
        same = got is not None and digest(got) == digest(values[check])
        print(f"  read-back of {check}: {'matches the live value' if same else 'DOES NOT MATCH'}")
        return 0 if same else 1
    print(f"  read-back failed: HTTP {status} {str(out)[:200]}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
