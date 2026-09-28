"""Create, check and deploy the Coolify application that runs the Propria Homepage portal.

Byline: Claude Code · Opus 5.5 · 2026-09-26

The portal (deploy/portal.yaml, deploy/portal/) is one Coolify application, `propria-portal`, built
from Cursedpotential/propria@main on ovh-app. This script is the declared record of that
application: SPEC and WATCH_PATHS below are what Coolify holds, and `create` / `sync` put them
there through the Coolify API (tailnet address; the public URL in the secrets file is dead).

    python coolify_app.py create     # create the app if absent (no deploy), then sync
    python coolify_app.py sync       # re-apply watch paths, base directory, compose location
    python coolify_app.py status     # app status + the last deployments
    python coolify_app.py deploy     # POST /deploy {"uuid": ...} for this app only, then wait

The API token is read from ~/.secrets/coolify-ionos-api.env (COOLIFY_API_TOKEN) by pattern
matching; it is never printed. Coolify 4.1.2 creates apps only through
POST /applications/private-github-app (plain POST /applications answers 404).
"""

import json
import os
import re
import sys
import time
import urllib.error
import urllib.request

API = os.environ.get("COOLIFY_API_URL", "http://100.98.98.38:8000/api/v1")
NAME = "propria-portal"
SPEC = {
    "name": NAME,
    "description": "Propria Homepage portal: tailnet (:3010) and public (:3012) instances, built from deploy/portal",
    "project_uuid": "nbg0ocqqrf91xag492yjqf5i",  # Propria
    "environment_name": "production",
    "server_uuid": "fmuao9enq3nxk8qw5hqjzzce",  # ovh-app
    "github_app_uuid": "r4mhpblr8cnxk3481r07xxz0",  # GitHub App "cursedpotential"
    "git_repository": "Cursedpotential/propria",
    "git_branch": "main",
    "build_pack": "dockercompose",
    "base_directory": "/modules/Probata/probata",
    "docker_compose_location": "/deploy/portal.yaml",
    "ports_exposes": "3000",
    "instant_deploy": False,
}
WATCH_PATHS = "\n".join(
    [
        "modules/Probata/probata/deploy/portal.yaml",
        "modules/Probata/probata/deploy/portal/Dockerfile",
        "modules/Probata/probata/deploy/portal/shared/**",
        "modules/Probata/probata/deploy/portal/tailnet/**",
        "modules/Probata/probata/deploy/portal/public/**",
    ]
)


def token() -> str:
    path = os.path.expanduser("~/.secrets/coolify-ionos-api.env")
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            match = re.match(r"^\s*(?:export\s+)?COOLIFY_API_TOKEN\s*=\s*['\"]?(.+?)['\"]?\s*$", line)
            if match:
                return match.group(1)
    raise SystemExit(f"COOLIFY_API_TOKEN not found in {path}")


def call(method: str, path: str, body: dict | None = None):
    data = json.dumps(body).encode() if body is not None else None
    request = urllib.request.Request(API + path, data=data, method=method)
    request.add_header("Authorization", "Bearer " + token())
    request.add_header("Accept", "application/json")
    if data is not None:
        request.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            raw = response.read().decode()
            return response.status, (json.loads(raw) if raw.strip() else None)
    except urllib.error.HTTPError as error:
        raw = error.read().decode()
        try:
            return error.code, json.loads(raw)
        except ValueError:
            return error.code, raw


def find_app() -> dict | None:
    status, apps = call("GET", "/applications")
    if status != 200:
        raise SystemExit(f"GET /applications -> {status}: {apps}")
    matches = [app for app in apps if app.get("name") == NAME]
    if len(matches) > 1:
        raise SystemExit(f"{len(matches)} applications are named {NAME}; resolve by hand")
    return matches[0] if matches else None


def summary(app: dict) -> dict:
    keys = ("uuid", "name", "status", "git_repository", "git_branch", "base_directory",
            "docker_compose_location", "watch_paths", "last_online_at")
    return {key: app.get(key) for key in keys}


def sync(uuid: str) -> None:
    body = {
        "description": SPEC["description"],
        "base_directory": SPEC["base_directory"],
        "docker_compose_location": SPEC["docker_compose_location"],
        "watch_paths": WATCH_PATHS,
    }
    status, out = call("PATCH", f"/applications/{uuid}", body)
    if status not in (200, 201):
        raise SystemExit(f"PATCH /applications/{uuid} -> {status}: {out}")
    _, app = call("GET", f"/applications/{uuid}")
    print(json.dumps(summary(app), indent=1))


def create() -> None:
    app = find_app()
    if app:
        print(f"{NAME} exists: {app['uuid']}")
    else:
        status, out = call("POST", "/applications/private-github-app", SPEC)
        if status not in (200, 201):
            raise SystemExit(f"POST /applications/private-github-app -> {status}: {out}")
        app = {"uuid": out["uuid"]}
        print(f"created {NAME}: {app['uuid']}")
    sync(app["uuid"])


def deployments(uuid: str, take: int = 5) -> list:
    _, out = call("GET", f"/deployments/applications/{uuid}?skip=0&take={take}")
    return (out.get("deployments", out) if isinstance(out, dict) else out) or []


def status() -> None:
    app = find_app()
    if not app:
        raise SystemExit(f"{NAME} does not exist")
    print(json.dumps(summary(app), indent=1))
    for dep in deployments(app["uuid"]):
        print(dep.get("created_at"), dep.get("status"), (dep.get("commit") or "")[:8],
              dep.get("deployment_uuid"))


def deploy() -> None:
    app = find_app()
    if not app:
        raise SystemExit(f"{NAME} does not exist")
    code, out = call("POST", "/deploy", {"uuid": app["uuid"], "force": False})
    print("POST /deploy ->", code, json.dumps(out))
    if code != 200:
        raise SystemExit(1)
    items = out.get("deployments", []) if isinstance(out, dict) else []
    dep_uuid = items[0].get("deployment_uuid") if items else None
    if not dep_uuid:
        return
    for _ in range(90):  # up to 15 minutes
        time.sleep(10)
        _, dep = call("GET", f"/deployments/{dep_uuid}")
        state = (dep or {}).get("status")
        print("deployment", dep_uuid, state, flush=True)
        if state in ("finished", "failed", "cancelled-by-user"):
            logs = (dep or {}).get("logs")
            if state != "finished" and logs:
                try:
                    lines = [entry.get("output", "") for entry in json.loads(logs)]
                except (ValueError, AttributeError):
                    lines = str(logs).splitlines()
                print("\n".join(line for line in lines[-40:] if line))
            if state != "finished":
                raise SystemExit(1)
            return
    raise SystemExit("deployment still running after 15 minutes")


if __name__ == "__main__":
    commands = {"create": create, "sync": lambda: sync(find_app()["uuid"]), "status": status, "deploy": deploy}
    if len(sys.argv) != 2 or sys.argv[1] not in commands:
        raise SystemExit("usage: coolify_app.py create|sync|status|deploy")
    commands[sys.argv[1]]()
