"""MCP source-root declaration transport contract; intercepted HTTP, no live store.

Byline: Codex / GPT-6.1 / 2026-10-07.
"""
import json
import importlib.util
from pathlib import Path

from fastmcp import Client
import pytest

# Exact-file loading supports both pytest's prepend and importlib modes without
# depending on which unrelated `tests` package happens to have been imported.
_spec=importlib.util.spec_from_file_location('docstore_source_roots_harness',Path(__file__).with_name('test_server.py'))
_helpers=importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_helpers)
config,harness=_helpers.config,_helpers.harness

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


async def test_bounded_adr_selector_uses_registered_payload_envelope(config):
    server,requests=harness(config)
    async with Client(server) as client:
        await client.call_tool('docstore_adr',{'action':'projections','payload':{'numbers':[100]}})
    assert len(requests)==1
    assert requests[0].url.path=='/adr/projections'
    assert json.loads(requests[0].content)=={'numbers':[100]}
