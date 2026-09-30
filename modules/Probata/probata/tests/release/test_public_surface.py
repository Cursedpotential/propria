"""Byline: Codex, 2026-09-20. Verify the actual MCP surface and write boundary."""
import pytest
from fastmcp import Client, FastMCP
from public_surface import build_public_server, PUBLIC_TOOLS


async def test_five_tools_discovery_read_and_explicit_write():
    backend = FastMCP('test-backend')
    writes = []
    for name in ('docstore_health', 'coco_docstore_search', 'docstore_get'):
        backend.tool(name=name, annotations={'readOnlyHint': True})(lambda: {'ok': True})

    @backend.tool(annotations={'readOnlyHint': False})
    def docstore_adr(action: str, payload: dict | None = None):
        if action == 'create': writes.append(payload)
        return {'action': action}

    server = await build_public_server(backend)
    async with Client(server) as client:
        assert {t.name for t in await client.list_tools()} == PUBLIC_TOOLS
        meta = (await client.call_tool('docstore_capabilities', {})).data
        assert meta['initial_tool_count'] == 5
        assert 'adr' in meta['groups']
        schema = (await client.call_tool('docstore_capabilities', {'operation': 'docstore_adr'})).data
        assert 'action' in schema['input_schema']['properties']
        read = await client.call_tool('docstore_query', {'operation': 'docstore_adr', 'arguments': {'action': 'list'}})
        assert read.data == {'action': 'list'}
        with pytest.raises(Exception, match='mode=write'):
            await client.call_tool('docstore_query', {'operation': 'docstore_adr', 'arguments': {'action': 'create'}})
        assert writes == []
        await client.call_tool('docstore_query', {'operation': 'docstore_adr', 'arguments': {'action': 'create', 'payload': {'title': 'test'}}, 'mode': 'write'})
        assert writes == [{'title': 'test'}]
        with pytest.raises(Exception, match='Unknown'):
            await client.call_tool('docstore_query', {'operation': 'raw_database_delete'})
        assert {t.name for t in await client.list_tools()} == PUBLIC_TOOLS
