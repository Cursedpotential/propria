"""Point the Coolify service propria-docstore-0-8-1 at a 0.8.1-rN image and publish its API on 8072.

Byline: Claude Code · Opus 5.5 · 2026-09-26

The service (uuid o8obobz576je1fbyygnywl83, ovh-files) keeps its compose only in Coolify, with
secret values inline, so this script edits exactly three things in the stored compose and never
prints a value:
  - image  propria-docstore:0.8.1 (r1)  ->  --image (default propria-docstore:0.8.1-r3; r2 was deployed first)
  - env    DOCSTORE_API_HOST=0.0.0.0     (service.py r2 binds the worker API beyond loopback)
  - ports  + 100.91.190.107:8072:8000    (canonical Docstore API port; svc:docstore-api points here)
Rollback: run with --rollback to restore image propria-docstore:0.8.1-r1 and drop the env and port.

Dry run by default (prints the changed keys only); --apply PATCHes the compose, --restart then
restarts the service through the Coolify API. The token is regex-parsed from ~/.secrets and never
printed. Coolify is reached on the tailnet only.

    python3 coolify_service_r2.py [--image TAG] [--apply] [--restart] [--rollback]
"""
from __future__ import annotations

import argparse
import base64
import json
import sys
from pathlib import Path

import yaml

sys.path.insert(0, str(Path(__file__).resolve().parents[3] / "scripts"))
from coolify_manual_deploys import _call, _token  # noqa: E402  (tracked helper: tailnet API URL + token parsing)

SERVICE = "o8obobz576je1fbyygnywl83"
SUB = "docstore"
IMAGE_DEFAULT, IMAGE_R1 = "propria-docstore:0.8.1-r3", "propria-docstore:0.8.1-r1"
API_PORT = "100.91.190.107:8072:8000"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--restart", action="store_true")
    parser.add_argument("--rollback", action="store_true")
    parser.add_argument("--image", default=IMAGE_DEFAULT)
    args = parser.parse_args()
    token = _token()
    status, body = _call(token, "GET", f"/services/{SERVICE}")
    if status != 200:
        raise SystemExit(f"GET service -> HTTP {status}")
    service = json.loads(body)
    compose = yaml.safe_load(service["docker_compose_raw"])
    spec = compose["services"][SUB]
    env = spec.setdefault("environment", {})
    if not isinstance(env, dict):
        raise SystemExit("environment is not a mapping; refusing to edit")
    ports = spec.setdefault("ports", [])
    before = {"image": spec.get("image"), "DOCSTORE_API_HOST": env.get("DOCSTORE_API_HOST"), "ports": list(ports)}
    if args.rollback:
        spec["image"] = IMAGE_R1
        env.pop("DOCSTORE_API_HOST", None)
        spec["ports"] = [p for p in ports if p != API_PORT]
    else:
        spec["image"] = args.image
        env["DOCSTORE_API_HOST"] = "0.0.0.0"
        if API_PORT not in ports:
            ports.append(API_PORT)
    after = {"image": spec.get("image"), "DOCSTORE_API_HOST": env.get("DOCSTORE_API_HOST"), "ports": list(spec["ports"])}
    print("service", service.get("name"), service.get("status"))
    print("before", json.dumps(before))
    print("after ", json.dumps(after))
    print("environment keys unchanged apart from DOCSTORE_API_HOST:", len(env), "keys")
    if not args.apply:
        return
    raw = yaml.safe_dump(compose, sort_keys=False, allow_unicode=True)
    status, body = _call(token, "PATCH", f"/services/{SERVICE}",
                         {"docker_compose_raw": base64.b64encode(raw.encode("utf-8")).decode("ascii")})
    print("PATCH service ->", status, "" if status < 300 else body[:300])
    if status >= 300:
        raise SystemExit(1)
    if args.restart:
        status, body = _call(token, "GET", f"/services/{SERVICE}/restart")
        if status == 405:
            status, body = _call(token, "POST", f"/services/{SERVICE}/restart")
        print("restart ->", status, body[:200])


if __name__ == "__main__":
    main()
