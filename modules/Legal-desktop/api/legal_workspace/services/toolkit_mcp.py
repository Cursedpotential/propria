"""Server-side MCP client for the Family Law Toolkit's hosted console.

> _Byline: Claude Code · Opus 5.5 · 2026-10-02_
First slice of the MCP client in
docs/planning/2026-09-13-advocatio-reconciliation/MCP-CLIENT-AND-PORTS.md: one
connection (the toolkit's `family-court-console`, MCP Streamable HTTP with a bearer
token), a tool catalog with input schemas, and a manual invocation. Owner requirement
R45: every toolkit capability is usable from the desk.

The console owns the tools and the case store they read and write. The desk never
caches their results as its own records. Write-capable calls run only when the caller
confirms the exact call (`confirm_write`); the web page shows that call before the
owner clicks Run. Tool descriptions and annotations are data, not permission: the desk
also treats tools it knows can change the store or its files (case_import, case_export,
case_query) as writes whatever their annotations say.

The connection is configured, not registered: connection profiles, invocation history
and resources/prompts are later slices of the same design. Sibling pattern:
services/family_court_toolkit.py (read-through, empty URL = "not configured").
"""

from __future__ import annotations

import hashlib
import json
import uuid
from datetime import UTC, datetime
from typing import Any, Literal

import jsonschema
from fastmcp import Client
from fastmcp.client.transports import StreamableHttpTransport
from pydantic import BaseModel, Field

from legal_workspace.config import get_settings

CONNECTION_ID = "family-court-console"
CONNECTION_NAME = "Family Law Toolkit console"

# Annotated read-only by the console, but each can change the store or the files around it:
# case_import/case_export move data through files, and case_query refuses only DELETE, REMOVE
# and DEFINE without `write: true`, so CREATE, UPDATE, UPSERT and RELATE run through it
# (store.ts assertQueryAllowed, read 2026-10-02). The desk treats all three as writes.
_ALWAYS_WRITE = frozenset({"case_import", "case_export", "case_query"})

_TIMEOUT_SECONDS = 60
_TEXT_CAP = 200_000


class McpUnavailable(RuntimeError):
    """The console is not configured, unreachable, or answered with a protocol error."""


class UnknownTool(LookupError):
    pass


class WriteNotConfirmed(PermissionError):
    pass


class InvalidArguments(ValueError):
    pass


class McpConnection(BaseModel):
    id: str = CONNECTION_ID
    name: str = CONNECTION_NAME
    transport: str = "streamable-http"
    configured: bool
    reachable: bool = False
    tool_count: int = 0
    detail: str = ""


class ToolDefinition(BaseModel):
    connection_id: str = CONNECTION_ID
    name: str
    title: str = ""
    description: str = ""
    input_schema: dict[str, Any] = Field(default_factory=dict)
    annotations: dict[str, Any] = Field(default_factory=dict)
    schema_hash: str
    writes: bool


class ToolCatalog(BaseModel):
    connection_id: str = CONNECTION_ID
    items: list[ToolDefinition]


class InvocationRequest(BaseModel):
    connection_id: str = CONNECTION_ID
    tool: str
    arguments: dict[str, Any] = Field(default_factory=dict)
    confirm_write: bool = False


class InvocationResult(BaseModel):
    id: str
    connection_id: str = CONNECTION_ID
    tool: str
    schema_hash: str
    arguments_sha256: str
    writes: bool
    state: Literal["ok", "tool_error"]
    text: list[str]
    structured: Any = None
    content: list[dict[str, Any]]
    truncated: bool = False
    started_at: str
    completed_at: str


def _sha256(value: Any) -> str:
    return "sha256:" + hashlib.sha256(
        json.dumps(value, sort_keys=True, separators=(",", ":"), default=str).encode()
    ).hexdigest()


def _client() -> Client:
    settings = get_settings()
    if not settings.family_court_console_mcp_url:
        raise McpUnavailable("FAMILY_COURT_CONSOLE_MCP_URL is not set")
    transport = StreamableHttpTransport(
        settings.family_court_console_mcp_url,
        auth=settings.family_court_console_mcp_token or None,
    )
    return Client(transport, timeout=_TIMEOUT_SECONDS)


def _definition(tool: Any) -> ToolDefinition:
    annotations = tool.annotations.model_dump(by_alias=True, exclude_none=True) if tool.annotations else {}
    input_schema = dict(tool.input_schema or {})
    read_only = bool(annotations.get("readOnlyHint"))
    return ToolDefinition(
        name=tool.name,
        title=tool.title or (annotations.get("title") or ""),
        description=tool.description or "",
        input_schema=input_schema,
        annotations=annotations,
        schema_hash=_sha256(input_schema),
        writes=(not read_only) or tool.name in _ALWAYS_WRITE,
    )


async def _list(client: Client) -> list[ToolDefinition]:
    tools: list[ToolDefinition] = []
    cursor: str | None = None
    while True:
        page = await client.list_tools_mcp(cursor=cursor)
        tools.extend(_definition(tool) for tool in page.tools)
        cursor = page.next_cursor
        if not cursor:
            return tools


def _protocol_failure(exc: Exception) -> McpUnavailable:
    # Keep the URL and token out of the message.
    return McpUnavailable(f"toolkit console call failed: {type(exc).__name__}")


async def list_tools() -> ToolCatalog:
    try:
        async with _client() as client:
            return ToolCatalog(items=await _list(client))
    except McpUnavailable:
        raise
    except Exception as exc:  # transport, auth and JSON-RPC failures
        raise _protocol_failure(exc) from exc


async def connection_status() -> McpConnection:
    settings = get_settings()
    if not settings.family_court_console_mcp_url:
        return McpConnection(configured=False, detail="FAMILY_COURT_CONSOLE_MCP_URL is not set")
    try:
        catalog = await list_tools()
    except McpUnavailable as exc:
        return McpConnection(configured=True, detail=str(exc))
    return McpConnection(configured=True, reachable=True, tool_count=len(catalog.items))


def _block(block: Any) -> dict[str, Any]:
    return block.model_dump(mode="json", by_alias=True, exclude_none=True)


async def invoke(request: InvocationRequest) -> InvocationResult:
    if request.connection_id != CONNECTION_ID:
        raise UnknownTool(f"connection {request.connection_id!r} is not registered")
    started = datetime.now(UTC).isoformat()
    try:
        async with _client() as client:
            catalog = {tool.name: tool for tool in await _list(client)}
            definition = catalog.get(request.tool)
            if definition is None:
                raise UnknownTool(f"tool {request.tool!r} is not offered by {CONNECTION_ID}")
            try:
                jsonschema.validate(request.arguments, definition.input_schema)
            except jsonschema.ValidationError as exc:
                raise InvalidArguments(exc.message) from exc
            writes = definition.writes
            if writes and not request.confirm_write:
                raise WriteNotConfirmed(
                    f"{request.tool} changes the toolkit store; review the call and confirm it"
                )
            result = await client.call_tool_mcp(request.tool, request.arguments)
    except (McpUnavailable, UnknownTool, InvalidArguments, WriteNotConfirmed):
        raise
    except Exception as exc:
        raise _protocol_failure(exc) from exc

    content = [_block(block) for block in result.content]
    truncated = False
    text: list[str] = []
    for block in content:
        if block.get("type") == "text":
            value = str(block.get("text", ""))
            if len(value) > _TEXT_CAP:
                value, truncated = value[:_TEXT_CAP], True
                block["text"] = value
            text.append(value)
    return InvocationResult(
        id=f"inv_{uuid.uuid4().hex}",
        tool=request.tool,
        schema_hash=definition.schema_hash,
        arguments_sha256=_sha256(request.arguments),
        writes=writes,
        state="tool_error" if result.is_error else "ok",
        text=text,
        structured=result.structured_content,
        content=content,
        truncated=truncated,
        started_at=started,
        completed_at=datetime.now(UTC).isoformat(),
    )
