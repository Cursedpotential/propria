"""Exercise source-route lock conflicts and authentication through real ASGI HTTP.

Inputs: synthetic requests and a source-operation double. Outputs: HTTP contract
assertions. Effects: temporary in-process app only, no source writes, live locks,
provider calls or database access. Choose for source-route exception mapping.
"""
import asyncio
import sys
from pathlib import Path
from unittest.mock import Mock

import httpx
import pytest
from fastapi import FastAPI, HTTPException

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import release_api
from run_support import WorkerBusy

AUTHORIZATION = 'Bearer synthetic-test-token'
PAYLOADS = {
    'read': {'paths': ['probata/example.md']},
    'plan': {'roots': list(release_api.source_sync.ROOTS), 'files': []},
    'apply': {'roots': list(release_api.source_sync.ROOTS), 'files': [], 'plan_id': 'synthetic-plan'},
}


@pytest.fixture
def api(monkeypatch):
    """Register production routes around explicit auth and operation test doubles.

    Input: monkeypatch fixture. Output: app, operation mock and event list.
    Effects: in-process registration only; the double never acquires a live lock.
    """
    events = []

    def authenticate(authorization):
        """Reject missing or incorrect synthetic credentials before source dispatch.

        Input: header. Output: None or HTTP 401. Effects: records test event only.
        """
        events.append('auth')
        if authorization != AUTHORIZATION:
            raise HTTPException(401, 'Unauthorized')

    operation = Mock()
    monkeypatch.setattr(release_api.source_sync, 'operation', operation)
    app = FastAPI()
    release_api.register(app, authenticate)
    return app, operation, events


async def post(app, action, *, authorized=True, **kwargs):
    """Send a source request through the ASGI app without opening network sockets.

    Inputs: app, action, auth flag and request options. Output: HTTPX response.
    Effects: in-process HTTP only; unexpected app errors retain HTTP 500 behavior.
    """
    transport = httpx.ASGITransport(app=app, raise_app_exceptions=False)
    async with httpx.AsyncClient(transport=transport, base_url='http://asgi.test') as client:
        return await client.post('/sources/' + action,
                                 headers={'Authorization': AUTHORIZATION} if authorized else {}, **kwargs)


@pytest.mark.parametrize('action', ['read', 'plan', 'apply'])
@pytest.mark.parametrize('reason', ['Another worker owns the OS lock',
                                  'Legacy or unknown lock: explicit reconciliation required'])
def test_source_lock_contention_returns_409_with_exact_safe_reason(api, action, reason):
    """Map both existing WorkerBusy reasons for read, plan and apply to HTTP 409.

    Inputs: route action and actual lock-reason text. Output: exact status/detail.
    Effects: operation double raises only; the underlying OS lock is untouched.
    """
    app, operation, events = api
    operation.side_effect = WorkerBusy(reason)
    response = asyncio.run(post(app, action, json=PAYLOADS[action]))
    assert response.status_code == 409
    assert response.json() == {'detail': reason}
    operation.assert_called_once_with(action, PAYLOADS[action])
    assert events == ['auth']


@pytest.mark.parametrize('action', ['read', 'plan', 'apply'])
def test_authentication_precedes_source_operation_and_body_read(api, action):
    """Reject unauthorized requests before consuming their stream or dispatching.

    Inputs: unauthorized streamed request. Output: HTTP 401 with no operation call.
    Effects: records only the auth event; malformed body bytes are never consumed.
    """
    app, operation, events = api

    async def body():
        """Record any attempted body read while yielding deliberately invalid JSON.

        Inputs: none. Output: synthetic bytes. Effects: fixture event only.
        """
        events.append('body-read')
        yield b'not JSON'

    response = asyncio.run(post(app, action, authorized=False, content=body()))
    assert response.status_code == 401
    assert response.json() == {'detail': 'Unauthorized'}
    assert events == ['auth']
    operation.assert_not_called()


@pytest.mark.parametrize(('action', 'receipt'), [
    ('read', {'files': [], 'missing': ['probata/example.md'], 'remaining': []}),
    ('plan', {'changed': [], 'retracted_sources': [], 'plan_id': 'synthetic-plan'}),
    ('apply', {'applied': True, 'verified': True, 'index_started': False}),
])
def test_successful_source_receipt_is_unchanged(api, action, receipt):
    """Preserve successful receipt bodies for each supported source action.

    Inputs: synthetic operation result. Output: unchanged HTTP 200 response.
    Effects: operation double only; no source validation or writes are bypassed live.
    """
    app, operation, _ = api
    operation.return_value = receipt
    response = asyncio.run(post(app, action, json=PAYLOADS[action]))
    assert response.status_code == 200 and response.json() == receipt
    operation.assert_called_once_with(action, PAYLOADS[action])


@pytest.mark.parametrize('action', ['read', 'plan', 'apply'])
def test_validation_errors_keep_existing_409_detail(api, action):
    """Retain the source route's established ValueError mapping after lock repair.

    Inputs: validation failure. Output: original HTTP 409 detail. Effects: mocks only.
    """
    app, operation, _ = api
    operation.side_effect = ValueError('Upload plan changed; re-plan before apply')
    response = asyncio.run(post(app, action, json=PAYLOADS[action]))
    assert response.status_code == 409
    assert response.json() == {'detail': 'Upload plan changed; re-plan before apply'}


def test_malformed_json_keeps_existing_409_without_source_dispatch(api):
    """Preserve bounded parsing failures before any source-operation invocation.

    Input: malformed JSON. Output: HTTP 409. Effects: in-process request only.
    """
    app, operation, _ = api
    response = asyncio.run(post(app, 'plan', content=b'{'))
    assert response.status_code == 409 and isinstance(response.json()['detail'], str)
    operation.assert_not_called()


def test_unrelated_runtime_failure_stays_500(api):
    """Avoid classifying unexpected runtime defects as ordinary lock contention.

    Input: unrelated RuntimeError. Output: HTTP 500. Effects: operation mock only.
    """
    app, operation, _ = api
    operation.side_effect = RuntimeError('synthetic unexpected failure')
    response = asyncio.run(post(app, 'read', json=PAYLOADS['read']))
    assert response.status_code == 500
