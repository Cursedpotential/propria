"""Explain rejected MCP configuration without adding execution authority.

Byline: Codex · GPT-6.1 · 2026-10-07.
"""

from __future__ import annotations

import json
import re
from urllib.parse import urlsplit


_CONTEXTFORGE_HOSTS = {"contextforge.tilapia-skilift.ts.net", "mcp.mitechconsult.com"}
_VIRTUAL_SERVER_PATH = re.compile(r"/servers/[a-f0-9]{32}/mcp/?")


def admitted_publication(server: dict, bypass_allowed: bool) -> bool:
    """Admit a ContextForge virtual server, or an explicit diagnostic bypass.

    Inputs: configured server and bypass flag. Output: admission decision.
    Effects: none; choose this for both catalog visibility and call routing so
    a direct upstream cannot be made callable by merely changing its label.
    """
    if bypass_allowed:
        return True
    if server.get("gateway") != "contextforge":
        return False
    try:
        url = urlsplit(server.get("url", ""))
        return (
            url.scheme == "https"
            and url.hostname in _CONTEXTFORGE_HOSTS
            and url.username is None
            and url.password is None
            and _VIRTUAL_SERVER_PATH.fullmatch(url.path) is not None
            and not url.query
            and not url.fragment
        )
    except (TypeError, ValueError):
        return False


def configuration_errors(raw: str, bypass_allowed: bool) -> list[dict[str, str]]:
    """List malformed or excluded MCP doors for the operator catalog.

    Inputs: MCP_SERVERS JSON and diagnostic bypass flag. Output: safe labels
    and errors. Effects: none. Choose for display; the parsed allowlist alone
    governs calls and rejected URLs/tokens are never emitted.
    """
    try:
        servers = json.loads(raw)
    except (TypeError, ValueError):
        return [{"key": "mcp-config", "label": "Tool configuration", "error": "MCP_SERVERS is not valid JSON."}]
    if not isinstance(servers, list):
        return [{"key": "mcp-config", "label": "Tool configuration", "error": "MCP_SERVERS must be a list."}]
    errors: list[dict[str, str]] = []
    for index, server in enumerate(servers):
        if not isinstance(server, dict):
            errors.append({"key": f"mcp-config-{index}", "label": "Tool configuration", "error": "A configured MCP server is not an object."})
            continue
        if admitted_publication(server, bypass_allowed):
            continue
        key = str(server.get("key") or f"mcp-config-{index}")
        label = str(server.get("label") or key)
        errors.append({
            "key": key,
            "label": label,
            "error": "This MCP server is excluded: Workbench tools require a ContextForge virtual-server URL and client token. Direct upstream doors are disabled.",
        })
    return errors
