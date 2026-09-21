"""Make ContextForge re-read one federated gateway's tools (names, schemas, annotations).

Byline: Claude Code · Fable 5.1 · 2026-09-20

Why: ContextForge stores each federated tool's input schema at registration.
The Docstore 0.8.1 `ctl08` gateway was registered at 15:11 UTC on 2026-09-20,
before the container was rebuilt at 16:09, so `docstore_capabilities` stayed
recorded with NO parameters although the live server advertises `group` and
`operation` — discovery, the heart of the five-tool design, was uncallable
through the federation.

    python3 scripts/contextforge_refresh_gateway.py ctl08            # show what would change
    python3 scripts/contextforge_refresh_gateway.py ctl08 --apply

Auth comes from ~/.secrets/contextforge.env (regex-parsed, never printed).
Tailnet only. Prints tool names and parameter names, nothing secret.
"""

from __future__ import annotations

import argparse
import base64
import json
import os
import re
import urllib.error
import urllib.request

BASE = "http://100.72.169.40:4444"


def _secrets() -> dict[str, str]:
    values: dict[str, str] = {}
    with open(os.path.expanduser("~/.secrets/contextforge.env"), encoding="utf-8", errors="replace") as handle:
        for line in handle:
            match = re.match(r"^\s*([A-Z0-9_]+)\s*=\s*(.+?)\s*$", line)
            if match:
                values[match.group(1)] = match.group(2).strip("'\"")
    return values


def _call(headers: dict[str, str], method: str, path: str, body: dict | None = None) -> tuple[int, object]:
    data = json.dumps(body).encode() if body is not None else None
    request = urllib.request.Request(BASE + path, data=data, method=method, headers={**headers, "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(request, timeout=90) as response:
            raw = response.read().decode()
            return response.status, (json.loads(raw) if raw else {})
    except urllib.error.HTTPError as error:
        return error.code, error.read().decode()[:300]


def _auth(secrets: dict[str, str]) -> dict[str, str]:
    candidates = []
    if secrets.get("CF_MCP_CLIENT_TOKEN"):
        candidates.append({"Authorization": "Bearer " + secrets["CF_MCP_CLIENT_TOKEN"]})
    if secrets.get("CF_BASIC_AUTH_PASSWORD"):
        basic = base64.b64encode(("admin:" + secrets["CF_BASIC_AUTH_PASSWORD"]).encode()).decode()
        candidates.append({"Authorization": "Basic " + basic})
    for headers in candidates:
        status, _ = _call(headers, "GET", "/gateways")
        if status == 200:
            return headers
    raise SystemExit("no ContextForge credential in ~/.secrets/contextforge.env was accepted for GET /gateways")


def _tools(headers: dict[str, str], gateway_id: str) -> dict[str, list[str]]:
    status, tools = _call(headers, "GET", "/tools?include_inactive=true&limit=0")
    if status != 200 or not isinstance(tools, (list, dict)):
        raise SystemExit(f"list tools: HTTP {status}")
    rows = tools if isinstance(tools, list) else tools.get("tools") or tools.get("items") or []
    found = {}
    for tool in rows:
        if (tool.get("gatewayId") or tool.get("gateway_id")) == gateway_id:
            schema = tool.get("inputSchema") or tool.get("input_schema") or {}
            found[tool.get("originalName") or tool.get("original_name") or tool["name"]] = sorted(schema.get("properties", {}))
    return found


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.split("\n", 1)[0])
    parser.add_argument("gateway")
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    headers = _auth(_secrets())
    status, gateways = _call(headers, "GET", "/gateways")
    rows = gateways if isinstance(gateways, list) else gateways.get("gateways", [])
    match = [g for g in rows if g.get("name") == args.gateway]
    if len(match) != 1:
        raise SystemExit(f"gateway {args.gateway!r} not found exactly once")
    gateway_id = match[0]["id"]
    before = _tools(headers, gateway_id)
    print("before:", json.dumps(before))
    if not args.apply:
        print("dry run; pass --apply to refresh")
        return
    for method, path in (("POST", f"/gateways/{gateway_id}/tools/refresh"), ("POST", f"/gateways/{gateway_id}/refresh")):
        status, body = _call(headers, method, path)
        print(method, path.replace(gateway_id, "<id>"), "->", status)
        if status in (200, 201, 202):
            break
    else:
        # Older builds re-read tools when a gateway is deactivated and reactivated.
        for activate in ("false", "true"):
            status, body = _call(headers, "POST", f"/gateways/{gateway_id}/toggle?activate={activate}")
            print("toggle activate=" + activate, "->", status)
    print("after: ", json.dumps(_tools(headers, gateway_id)))


if __name__ == "__main__":
    main()
