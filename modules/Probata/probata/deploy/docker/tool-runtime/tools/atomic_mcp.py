"""atomic_tools — one MCP entry point over the tool-runtime registry.

Byline: Claude Code · Opus 5.5 · 2026-09-28

Owner design, 2026-09-28 05:21: expose the atomic tools efficiently — ONE tool, whose
description carries a brief summary of what is available, with a directory listing of the
families and tools below it so an agent loads detail progressively instead of paying for
43 tool schemas up front.

An agent sees a single `atomic_tools` tool. It browses with `path`:

    path=""                      -> the families, with counts and capabilities
    path="messages"              -> that family's tools, one line each
    path="messages.sms-xml"      -> the full contract for one tool
    path="messages.sms-xml", run={...payload}  -> executes it

Everything shown is generated from the live registry manifest (`registry.contract_manifest()`),
the same declaration contract `GET /tools` serves, so the listing cannot drift from what the
runtime actually has. Execution goes through the same `tool.run(payload)` call as
`POST /tools/{id}/run`.

The browsing functions below are pure (manifest in, dict out) so they can be reasoned about and
tested without the MCP layer; `build_server` only wires them to the transport.
"""

from __future__ import annotations

from collections import defaultdict
from collections.abc import Callable
from typing import Any

# How many tool names to preview per family inside the master description.
_PREVIEW = 6


def families(manifest: list[dict[str, Any]]) -> dict[str, list[dict[str, Any]]]:
    """Group manifest entries by the family prefix of their id (`family.tool`)."""
    grouped: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for entry in manifest:
        grouped[str(entry["id"]).split(".", 1)[0]].append(entry)
    return {name: sorted(tools, key=lambda t: t["id"]) for name, tools in sorted(grouped.items())}


def _short(tool_id: str) -> str:
    return tool_id.split(".", 1)[1] if "." in tool_id else tool_id


def _capabilities(tools: list[dict[str, Any]]) -> list[str]:
    return sorted({str(t.get("capability") or "unspecified") for t in tools})


def master_description(manifest: list[dict[str, Any]]) -> str:
    """The tool description an agent sees before calling anything: a brief map of what exists."""
    grouped = families(manifest)
    lines = [
        f"Probata's atomic evidence tools: {len(manifest)} tools in {len(grouped)} families "
        "(parsers, extractors, repair and engine probes). Browse first, then run one.",
    ]
    for name, tools in grouped.items():
        preview = ", ".join(_short(t["id"]) for t in tools[:_PREVIEW])
        more = f", +{len(tools) - _PREVIEW} more" if len(tools) > _PREVIEW else ""
        lines.append(f"- {name} ({len(tools)}; {', '.join(_capabilities(tools))}): {preview}{more}")
    lines.append(
        "Use: path='' lists families; path='<family>' lists its tools; path='<family>.<tool>' shows "
        "the full contract; add run={...} to execute that tool. File paths in a payload must be "
        "visible to tool-runtime (custody blobs live under /r2/evidence/...)."
    )
    return "\n".join(lines)


def browse(manifest: list[dict[str, Any]], path: str) -> dict[str, Any]:
    """Resolve one level of the directory: root, a family, or a single tool."""
    path = (path or "").strip().strip("/").strip(".")
    grouped = families(manifest)

    if not path:
        return {
            "level": "root",
            "tool_count": len(manifest),
            "families": [
                {
                    "family": name,
                    "tools": len(tools),
                    "capabilities": _capabilities(tools),
                    "open": f"atomic_tools(path='{name}')",
                }
                for name, tools in grouped.items()
            ],
        }

    if path in grouped:
        return {
            "level": "family",
            "family": path,
            "tools": [
                {
                    "id": t["id"],
                    "description": t.get("description"),
                    "capability": t.get("capability"),
                    "side_effect": t.get("side_effect"),
                    "formats": t.get("formats") or [],
                }
                for t in grouped[path]
            ],
            "open": "atomic_tools(path='<id>') for the full contract",
        }

    by_id = {t["id"]: t for t in manifest}
    if path in by_id:
        return {
            "level": "tool",
            "tool": by_id[path],
            "run": f"atomic_tools(path='{path}', run={{...payload}})",
        }

    needle = path.lower()
    suggestions = sorted(i for i in by_id if needle in i.lower())[:10]
    return {
        "level": "not_found",
        "path": path,
        "suggestions": suggestions,
        "families": list(grouped),
    }


def execute(get_tool: Callable[[str], Any], tool_id: str, payload: dict[str, Any]) -> dict[str, Any]:
    """Execute one tool with the same semantics as POST /tools/{id}/run, as a result dict."""
    try:
        tool = get_tool(tool_id)
    except KeyError:
        return {"ok": False, "error": "unknown_tool", "detail": f"unknown tool {tool_id!r}"}
    try:
        return {"ok": True, "tool": tool_id, "result": tool.run(payload)}
    except FileNotFoundError as exc:
        return {"ok": False, "error": "file_not_found", "detail": str(exc)}
    except ValueError as exc:
        # Contract rejection (wrong format). The caller should try a sibling from the same family.
        return {"ok": False, "error": "contract_rejected", "detail": str(exc)}


def build_server(
    manifest: list[dict[str, Any]] | None,
    get_tool: Callable[[str], Any] | None,
    degraded_reason: str = "",
):
    """Wire the directory to one MCP tool. Returns the MCPServer (mcp 2.x)."""
    from mcp.server.mcpserver import MCPServer

    server = MCPServer(
        "probata-atomic-tools",
        instructions="One tool, `atomic_tools`. Browse with path, then run with run={...}.",
    )

    if manifest is None or get_tool is None:
        description = (
            "Probata's atomic evidence tools are UNAVAILABLE on this runtime: "
            f"{degraded_reason or 'registry failed to load'}. Calls return the same error."
        )
    else:
        description = master_description(manifest)

    @server.tool(name="atomic_tools", description=description)
    def atomic_tools(path: str = "", run: dict[str, Any] | None = None) -> dict[str, Any]:
        if manifest is None or get_tool is None:
            return {"ok": False, "error": "registry_unavailable", "detail": degraded_reason}
        if run is not None:
            return execute(get_tool, path.strip(), run)
        return browse(manifest, path)

    return server
