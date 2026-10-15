"""Explain rejected MCP configuration without adding execution authority.

Byline: Codex · GPT-6.1 · 2026-10-07.
"""

from __future__ import annotations

import json


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
        if server.get("gateway") == "portkey" or bypass_allowed:
            continue
        key = str(server.get("key") or f"mcp-config-{index}")
        label = str(server.get("label") or key)
        errors.append({
            "key": key,
            "label": label,
            "error": "This MCP server is excluded: normal Workbench tools must use a Portkey gateway publication. Configure the Portkey MCP URL and token environment; direct bypass is disabled.",
        })
    return errors
