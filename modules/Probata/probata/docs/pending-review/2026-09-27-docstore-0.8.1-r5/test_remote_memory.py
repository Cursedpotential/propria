"""0.8.1-r5 release tests: the memory write path reports why it failed.

Byline: Claude Code · Opus 5.5 · 2026-09-27. Installed as tests/test_remote_memory.py in the release tree
by apply_r5.py. The database side (fn::remember / fn::supersede_memory) is proven live; see README.md.
"""
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

import release_api
import remote_memory

GOOD = {'kind': 'constraint', 'claim': 'Owner rule: search with the owner tools, never grep.',
        'evidence': 'owner 2026-09-27 10:01 EDT', 'agent': 'test'}


@pytest.fixture
def api(monkeypatch):
    monkeypatch.setenv('MEMORY_MCP_URL', 'http://memory.invalid/mcp')
    monkeypatch.setattr(remote_memory.recall, 'embed', lambda text: [0.0] * 4)
    app = FastAPI()
    release_api.register(app, lambda authorization: None)
    return TestClient(app)


def use_database(monkeypatch, reply):
    calls = []
    async def native_call(function, args):
        calls.append((function, args))
        if isinstance(reply, Exception):
            raise reply
        return reply
    monkeypatch.setattr(remote_memory, 'native_call', native_call)
    return calls


def test_missing_and_unknown_fields_are_a_422_naming_each_problem(api, monkeypatch):
    calls = use_database(monkeypatch, {})
    response = api.post('/memory/remember', json={'content': 'x', 'tags': [], 'scope': 'probata', 'kind': 'note'})
    assert response.status_code == 422
    errors = ' '.join(response.json()['detail']['errors'])
    for expected in ("unknown field(s) ['content', 'tags']", "missing required field(s) ['claim', 'evidence', 'agent']",
                     'kind must be one of', 'scope must match ^propria'):
        assert expected in errors
    assert 'docstore_capabilities' in response.json()['detail']['schema']
    assert not calls


def test_write_defaults_scope_to_propria_and_reports_outcome(api, monkeypatch):
    calls = use_database(monkeypatch, {'written': 'memory:abc123', 'conflicts': []})
    response = api.post('/memory/remember', json=GOOD)
    assert response.status_code == 200
    assert response.json() == {'outcome': 'written', 'id': 'memory:abc123', 'superseded': None,
                               'scope': 'propria', 'available': True}
    assert calls[0][0] == 'remember' and calls[0][1][0]['scope'] == 'propria'


def test_near_duplicate_is_a_409_with_the_conflicting_ids(api, monkeypatch):
    use_database(monkeypatch, {'written': None, 'note': 'Similar active memory exists',
                               'conflicts': [{'id': 'memory:old1', 'claim': 'Owner rule: search with tools.',
                                              'dist': 0.09, 'confidence': 0.9, 'status': 'active'}]})
    response = api.post('/memory/remember', json=GOOD)
    assert response.status_code == 409
    detail = response.json()['detail']
    assert detail['conflicts'][0]['id'] == 'memory:old1' and 'supersede' in detail['next']


def test_supersession_reports_both_ids(api, monkeypatch):
    use_database(monkeypatch, {'written': 'memory:new1', 'superseded': 'memory:old1', 'conflicts': []})
    response = api.post('/memory/remember', json={**GOOD, 'supersede': 'memory:old1', 'reason': 'reworded'})
    assert response.json()['outcome'] == 'superseded' and response.json()['superseded'] == 'memory:old1'


@pytest.mark.parametrize('message,status', [
    ("Database index `memory_claim_uq` already contains ['propria', 'x']", 409),
    ('An error occurred: supersede target memory:gone does not exist', 422),
    ('internal storage failure', 502),
])
def test_database_errors_keep_their_reason(api, monkeypatch, message, status):
    use_database(monkeypatch, remote_memory.upstream_failure(message))
    response = api.post('/memory/remember', json=GOOD)
    assert response.status_code == status
    assert message.split(':')[0][:20] in response.json()['detail']['database']


def test_recall_defaults_to_propria_and_rejects_other_roots(api, monkeypatch):
    calls = use_database(monkeypatch, [])
    assert api.post('/memory/recall', json={'query': 'search tools'}).status_code == 200
    assert calls[0][1][2] == 'propria'
    response = api.post('/memory/recall', json={'query': 'search tools', 'scope': 'probata'})
    assert response.status_code == 422 and 'propria' in response.json()['detail']['reason']


def test_unknown_action_is_404(api):
    assert api.post('/memory/forget', json={}).status_code == 404
