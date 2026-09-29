"""Restore the Docstore bearer on the ContextForge gateway. Runs INSIDE the ContextForge container.

The token arrives on stdin. That only works because this script is a FILE: the earlier attempt
piped the token to `python -`, which reads the script itself from stdin, so sys.stdin.read()
returned "" and every write stored an empty credential. ContextForge then sent the header
`Authorization: Bearer ` and answered "Illegal header value b'Bearer '".

Byline: Claude Code · Opus 5 · 2026-09-28
"""
import json
import os
import sys
import urllib.error
import urllib.request

BASE = "http://127.0.0.1:4444"


def call(path, method="GET", body=None, headers=None):
    data = json.dumps(body).encode() if body is not None else None
    head = {"Content-Type": "application/json"}
    head.update(headers or {})
    try:
        with urllib.request.urlopen(
                urllib.request.Request(BASE + path, data=data, headers=head, method=method),
                timeout=90) as response:
            raw = response.read()
            return response.status, (json.loads(raw) if raw[:1] in (b"{", b"[") else raw[:200])
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read()[:300]


token = sys.stdin.read().strip()
print(f"  token received inside the container: {len(token)} chars")
if len(token) < 20:
    raise SystemExit("  refusing to write an empty or truncated credential")

status, session = call("/auth/login", "POST", {
    "username": os.environ.get("PLATFORM_ADMIN_EMAIL") or os.environ.get("BASIC_AUTH_USER"),
    "password": os.environ.get("PLATFORM_ADMIN_PASSWORD") or os.environ.get("BASIC_AUTH_PASSWORD")})
if status != 200:
    raise SystemExit(f"  admin login failed: HTTP {status}")
auth = {"Authorization": "Bearer " + session["access_token"]}

status, rows = call("/gateways", headers=auth)
rows = rows if isinstance(rows, list) else rows.get("data", [])
gateway = next((g for g in rows if ":8175" in str(g.get("url", ""))), None)
if gateway is None:
    raise SystemExit("  no gateway points at :8175; nothing was changed")
gid = gateway["id"]
print(f"  gateway {gateway.get('name')} ({gid}) authType={gateway.get('authType')} "
      f"reachable={gateway.get('reachable')}")

# bearer is the shape this gateway had before the rename cleared its value.
status, out = call(f"/gateways/{gid}", "PUT", {"authType": "bearer", "authToken": token}, auth)
print(f"  PUT authType=bearer -> HTTP {status}")
if status not in (200, 201):
    print("   ", str(out)[:250])
    raise SystemExit(1)

print("  tools refresh ->", call(f"/gateways/{gid}/tools/refresh", "POST", {}, auth)[0])
print("  activate      ->", call(f"/gateways/{gid}/state", "POST", {"activate": True}, auth)[0])

status, rows = call("/gateways", headers=auth)
rows = rows if isinstance(rows, list) else rows.get("data", [])
gateway = next(g for g in rows if g["id"] == gid)
print(f"  final: authType={gateway.get('authType')} reachable={gateway.get('reachable')} "
      f"enabled={gateway.get('enabled')}")
