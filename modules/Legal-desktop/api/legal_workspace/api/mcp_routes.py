"""MCP client routes: the toolkit console's tools, browsable and runnable from the desk.

> _Byline: Claude Code · Opus 5.5 · 2026-10-02_
First slice of MCP-CLIENT-AND-PORTS.md (routes /v1/mcp/connections, /v1/mcp/tools,
/v1/mcp/invocations), pointed at one configured connection. Parent mounts `router`.
Errors: 503 console not configured/unreachable or protocol failure, 404 unknown tool,
422 arguments fail the tool's input schema, 409 a write call was not confirmed. A tool
that runs and reports an error returns 200 with state "tool_error".
Sibling: toolkit_routes.py.
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException, Query

from legal_workspace.services.toolkit_mcp import (
    InvalidArguments,
    InvocationRequest,
    InvocationResult,
    McpConnection,
    McpUnavailable,
    ToolCatalog,
    UnknownTool,
    WriteNotConfirmed,
    connection_status,
    invoke,
    list_tools,
)

router = APIRouter(prefix="/v1/mcp")


@router.get("/connections")
async def connections() -> list[McpConnection]:
    return [await connection_status()]


@router.get("/tools")
async def tools(q: str = Query(default="", max_length=200)) -> ToolCatalog:
    try:
        catalog = await list_tools()
    except McpUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    needle = q.strip().lower()
    if needle:
        catalog.items = [
            tool for tool in catalog.items if needle in f"{tool.name} {tool.title} {tool.description}".lower()
        ]
    return catalog


@router.post("/invocations")
async def invocations(request: InvocationRequest) -> InvocationResult:
    try:
        return await invoke(request)
    except UnknownTool as exc:
        raise HTTPException(status_code=404, detail=str(exc)) from exc
    except InvalidArguments as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc
    except WriteNotConfirmed as exc:
        raise HTTPException(status_code=409, detail=str(exc)) from exc
    except McpUnavailable as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
