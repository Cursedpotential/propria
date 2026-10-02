#!/usr/bin/env python3
"""Register the Kasm Workspaces objects this repo declares (P-1 step 3), idempotently, through Kasm's own admin API.

Runs ON ovh-files as root (stdlib only), reading the admin credentials from /data/probata/secrets/kasm/kasm.env,
which install_kasm.sh wrote. Credentials are never printed. Workspace definitions are the tracked JSON files in
deploy/kasm/workspaces/, piped in on stdin as one JSON object {"<file name>": <file content>, ...} by the caller:

  python3 - <<< ''   # not used directly; see run_register.sh on the desktop side:
  ssh root@100.91.190.107 "python3 /data/probata/kasm-installer/register_kasm.py" < bundle.json

What it does, each step only if not already present (matched by friendly name / username / server name):
  0. the zone's proxy port follows the request port (0), for the 443 front doors.
  1. user `msalem` (owner) in the Administrators group, password generated once into kasm.env (KASM_OWNER_*).
  2. one Container workspace per workspaces/*.json with "workspace_type": "Container".
  3. the Guacamole RDP server + Server workspace from workspaces/*.json with "workspace_type": "Server".

Kasm's admin API is the one its own web UI calls (POST /api/authenticate, /api/admin/*). Route names were read
from the installed 1.19.0 API (admin_api); the public Developer API has no create_image.

Byline: Claude Code · Opus 5.5 · 2026-10-02
"""
from __future__ import annotations

import json
import re
import secrets
import ssl
import sys
import urllib.error
import urllib.request

ENV_FILE = "/data/probata/secrets/kasm/kasm.env"
API = "https://127.0.0.1:8443/api"
CTX = ssl._create_unverified_context()  # Kasm's own self-signed cert on loopback


def load_env() -> dict:
    env = {}
    for line in open(ENV_FILE, encoding="utf-8"):
        m = re.match(r"^\s*([A-Z_]+)\s*=\s*(.*?)\s*$", line)
        if m:
            env[m.group(1)] = m.group(2)
    return env


def secret_value(path: str | None, key: str | None) -> str:
    """Read one KEY=value from a host secrets file (never printed). Empty when not configured."""
    if not path or not key:
        return ""
    for line in open(path, encoding="utf-8"):
        m = re.match(rf"^\s*{re.escape(key)}\s*=\s*(.*?)\s*$", line)
        if m:
            return m.group(1)
    raise SystemExit(f"{key} not found in {path}")


def call(path: str, body: dict) -> dict:
    req = urllib.request.Request(API + path, data=json.dumps(body).encode(), method="POST",
                                 headers={"Content-Type": "application/json"})
    try:
        out = json.load(urllib.request.urlopen(req, context=CTX, timeout=120))
    except urllib.error.HTTPError as e:
        out = {"error_message": f"HTTP {e.code}: {e.read()[:300].decode(errors='replace')}"}
    if out.get("error_message"):
        raise SystemExit(f"FAILED {path}: {out['error_message']}")
    return out


def main() -> int:
    defs = json.load(sys.stdin)
    env = load_env()
    auth = call("/authenticate", {"username": env["KASM_ADMIN_USER"], "password": env["KASM_ADMIN_PASSWORD"],
                                  "logout_other_sessions": False})
    tok = {"token": auth["token"], "username": env["KASM_ADMIN_USER"]}

    # 1. owner user, admin
    users = call("/admin/get_users", dict(tok)).get("users", [])
    owner = next((u for u in users if u.get("username") == "msalem"), None)
    if owner is None:
        if "KASM_OWNER_PASSWORD" not in env:
            pw = secrets.token_urlsafe(21)
            with open(ENV_FILE, "a", encoding="utf-8") as f:
                f.write(f"KASM_OWNER_USER=msalem\nKASM_OWNER_PASSWORD={pw}\n")
            env["KASM_OWNER_PASSWORD"] = pw
        owner = call("/admin/create_user", {**tok, "target_user": {
            "username": "msalem", "password": env["KASM_OWNER_PASSWORD"], "first_name": "Matt", "last_name": "Salem",
            "locked": False, "disabled": False}})["user"]
        print("created user msalem")
    else:
        print("user msalem exists")
    groups = {g["name"]: g["group_id"] for g in call("/admin/get_groups", dict(tok))["groups"]}
    owner_groups = {g.get("name") for g in (owner.get("groups") or [])}
    if "Administrators" not in owner_groups:
        call("/admin/add_user_group", {**tok, "target_user": {"user_id": owner["user_id"]},
                                       "target_group": {"group_id": groups["Administrators"]}})
        print("msalem added to Administrators")

    # 0. zone: Kasm is reached through 443 front doors (svc:kasm, kasm.int behind Traefik), not its own 8443, so the
    # zone's proxy port must follow the request port (0); with 8443 the session websocket URLs point at a port the
    # front doors do not serve.
    for zone in call("/admin/get_zones", dict(tok)).get("zones", []):
        if zone.get("proxy_port") != 0:
            call("/admin/update_zone", {**tok, "target_zone": {**zone, "proxy_port": 0}})
            print(f"zone {zone.get('zone_name')!r}: proxy_port -> 0")

    # 2/3. workspaces
    images = {i.get("friendly_name"): i for i in call("/admin/get_images", dict(tok)).get("images", [])}
    for fname, d in sorted(defs.items()):
        name = d["friendly_name"]
        if name in images:
            print(f"workspace {name!r} exists (image_id {images[name]['image_id']})")
            continue
        target = {
            "friendly_name": name,
            "description": d.get("description", ""),
            "name": d.get("docker_image") or "",
            "cores": d.get("cores", 1),
            "memory": int(d.get("memory_mb", 2048)) * 1_000_000,
            "gpu_count": d.get("gpu_count", 0),
            "enabled": True,
            "image_src": "img/thumbnails/kasmweb_desktop.png",
            "docker_registry": d.get("docker_registry") or "",
            "persistent_profile_path": d.get("persistent_profile_path") or "",
            "run_config": json.dumps(d.get("docker_run_config_override") or {}),
            "volume_mappings": json.dumps(d.get("volume_mappings") or {}),
            "exec_config": json.dumps({}),
            "image_type": d["workspace_type"],
            "categories": "Propria",  # a multiline string in the admin API (one category per line)
        }
        if d["workspace_type"] == "Server":
            srv = d["server"]
            servers = {s.get("friendly_name"): s for s in call("/admin/get_servers", dict(tok)).get("servers", [])}
            s = servers.get(srv["friendly_name"])
            if s is None:
                s = call("/admin/create_server", {**tok, "target_server": {
                    "friendly_name": srv["friendly_name"], "hostname": srv["hostname"], "connection_type": srv["connection_type"],
                    "connection_port": srv["connection_port"], "connection_info": json.dumps(srv.get("connection_info", {})),
                    "max_simultaneous_sessions": 1, "enabled": True, "zone_id": None,
                    "connection_username": srv.get("connection_username", "{sso_username}"),
                    "connection_password": secret_value(srv.get("password_file"), srv.get("password_key")),
                    "use_user_private_key": False}})["server"]
                print(f"created server {srv['friendly_name']!r}")
            target.update(server_id=s["server_id"], name="")
        out = call("/admin/create_image", {**tok, "target_image": target})
        print(f"created workspace {name!r} (image_id {out.get('image', {}).get('image_id')})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
