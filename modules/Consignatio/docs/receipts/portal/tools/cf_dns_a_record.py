# Byline: Claude Code · Opus 5.5 · 2026-09-27
"""Idempotent Cloudflare DNS-only A record for a mitechconsult.com name (dry run unless --apply).

    python cf_dns_a_record.py authentik.mitechconsult.com 40.160.5.19 [--apply]

Public short names (<svc>.mitechconsult.com, tailnet redirect) and public *.int names are DNS-only
(proxied=false) A records at ovh-app's public address 40.160.5.19. The token is regex-parsed from
~/.secrets/cloudflare.env (never sourced, never printed). It is account-owned, so
/user/tokens/verify answers 401 although zone and DNS calls work.
"""
from __future__ import annotations

import json
import os
import re
import sys
import urllib.error
import urllib.request

ZONE_NAME = "mitechconsult.com"
API = "https://api.cloudflare.com/client/v4"


def _token() -> str:
    for line in open(os.path.expanduser("~/.secrets/cloudflare.env"), encoding="utf-8"):
        match = re.match(r"^\s*CLOUDFLARE_API_TOKEN\s*=\s*(.+?)\s*$", line)
        if match:
            return match.group(1).strip().strip("\"'")
    raise SystemExit("CLOUDFLARE_API_TOKEN not found")


def _call(method: str, path: str, body: dict | None = None) -> dict:
    request = urllib.request.Request(
        API + path,
        data=json.dumps(body).encode() if body is not None else None,
        method=method,
        headers={"Authorization": "Bearer " + _token(), "Content-Type": "application/json",
                 "User-Agent": "propria-edge/1.0"},
    )
    try:
        return json.load(urllib.request.urlopen(request, timeout=30))
    except urllib.error.HTTPError as err:
        return {"success": False, "http": err.code, "errors": err.read().decode()[:400]}


def main() -> int:
    name, address = sys.argv[1], sys.argv[2]
    apply = "--apply" in sys.argv[3:]
    if not name.endswith("." + ZONE_NAME):
        raise SystemExit(f"{name} is not in {ZONE_NAME}")
    zones = _call("GET", f"/zones?name={ZONE_NAME}")
    zone_id = zones["result"][0]["id"]
    existing = _call("GET", f"/zones/{zone_id}/dns_records?name={name}")["result"]
    wanted = {"type": "A", "name": name, "content": address, "proxied": False, "ttl": 1,
              "comment": "tailnet short name -> Traefik redirect to ts.net (Claude Code, 2026-09-27)"}
    print("existing:", [(r["type"], r["content"], r["proxied"]) for r in existing])
    if any(r["type"] == "A" and r["content"] == address and not r["proxied"] for r in existing):
        print("already correct; nothing to do")
        return 0
    if existing:
        print("a different record exists for this name; not touching it")
        return 1
    if not apply:
        print("dry run; would create:", wanted)
        return 0
    created = _call("POST", f"/zones/{zone_id}/dns_records", wanted)
    print("created:", created.get("success"), created.get("errors"))
    after = _call("GET", f"/zones/{zone_id}/dns_records?name={name}")["result"]
    print("read back:", [(r["type"], r["name"], r["content"], r["proxied"]) for r in after])
    return 0 if after else 1


if __name__ == "__main__":
    sys.exit(main())
