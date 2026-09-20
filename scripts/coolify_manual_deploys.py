"""Make Coolify deploys deliberate: turn auto-deploy off and fix the stale repository name.

Byline: Claude Code · Fable 5.1 · 2026-09-20

Owner 2026-09-20 17:56: "that way we have to stage and purposely reload".
Clearing ``watch_paths`` does the opposite (an app with no watch paths deploys
on EVERY push), so the switch is ``is_auto_deploy_enabled=false``; a deploy is
then an explicit ``GET /deploy?uuid=`` or the Coolify button. Watch paths are
left as they are (inert while auto-deploy is off).

Also renames ``Cursedpotential/mcp-platform-agno-mvp`` -> ``Cursedpotential/probata``
(GitHub's rename redirect had been hiding 15 stale apps).

Dry run by default; ``--apply`` writes. The token is regex-parsed from
``~/.secrets`` and never printed. Reached on the tailnet only.

    python3 scripts/coolify_manual_deploys.py            # what would change
    python3 scripts/coolify_manual_deploys.py --apply
"""

from __future__ import annotations

import argparse
import json
import os
import re
import urllib.error
import urllib.request

API = "http://100.98.98.38:8000/api/v1"
OLD_REPO = "Cursedpotential/mcp-platform-agno-mvp"
NEW_REPO = "Cursedpotential/probata"


def _token() -> str:
    for name in ("coolify-ionos-api.env", "coolify.env", "probata.env"):
        path = os.path.expanduser("~/.secrets/" + name)
        if not os.path.exists(path):
            continue
        with open(path, encoding="utf-8", errors="replace") as handle:
            for line in handle:
                match = re.match(r"^\s*(COOLIFY[A-Z_]*)\s*=\s*(.+?)\s*$", line)
                if match:
                    return match.group(2).strip("'\"")
    raise SystemExit("no Coolify token found under ~/.secrets")


def _call(token: str, method: str, path: str, body: dict | None = None) -> tuple[int, str]:
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Authorization": "Bearer " + token, "Accept": "application/json"}
    if data:
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(API + path, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            return response.status, response.read().decode()
    except urllib.error.HTTPError as error:
        return error.code, error.read().decode()[:300]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.split("\n", 1)[0])
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    token = _token()
    status, text = _call(token, "GET", "/applications")
    if status != 200:
        raise SystemExit(f"list applications: HTTP {status}")
    for app in json.loads(text, strict=False):
        body: dict[str, object] = {"is_auto_deploy_enabled": False}
        if app.get("git_repository") == OLD_REPO:
            body["git_repository"] = NEW_REPO
        note = "repo name + auto-deploy off" if "git_repository" in body else "auto-deploy off"
        if not args.apply:
            print(f"would patch {app['name']:32} {app['uuid']}  {note}")
            continue
        status, text = _call(token, "PATCH", "/applications/" + app["uuid"], body)
        print(f"{app['name']:32} {app['uuid']}  {note}: HTTP {status} {'' if status == 200 else text}")


if __name__ == "__main__":
    main()
