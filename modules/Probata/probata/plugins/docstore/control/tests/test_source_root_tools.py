"""MCP source-root declaration transport contract; intercepted HTTP, no live store.

Byline: Codex / GPT-6.1 / 2026-10-07.
"""
import json

from fastmcp import Client
import pytest

from test_server import config, harness

ROOTS=['propria','probata','consignatio','consignatio-intake','advocatio',
       'vestigia','family-court-workbench']


@pytest.mark.parametrize('operation,extra,endpoint',[
    ('docstore_source_plan',{},'/sources/plan'),
    ('docstore_source_apply',{'plan_id':'exact-plan','retract':[]},'/sources/apply'),
])
async def test_explicit_roots_reach_worker_unchanged(config,operation,extra,endpoint):
    server,requests=harness(config)
    async with Client(server) as client:
        tools={tool.name:tool for tool in await client.list_tools()}
        assert 'roots' in tools[operation].inputSchema['properties']
        await client.call_tool(operation,{'files':[],'roots':ROOTS,**extra})
    assert len(requests)==1
    assert requests[0].url.path==endpoint
    assert json.loads(requests[0].content)=={'files':[],'roots':ROOTS,**extra}


async def test_legacy_plan_omits_optional_roots(config):
    server,requests=harness(config)
    async with Client(server) as client:
        await client.call_tool('docstore_source_plan',{'files':[]})
    assert json.loads(requests[0].content)=={'files':[]}
