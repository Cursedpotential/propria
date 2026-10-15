"""Verify runner authorization, OAuth binding, validation and single execution semantics.

Inputs: synthetic sessions and mocked gateway responses. Outputs: unittest results.
Effects: no network or production writes; use alongside live read-only acceptance.
"""
import asyncio
import time
import unittest
from unittest.mock import AsyncMock, patch

import httpx
import app as runner


class RunnerTests(unittest.IsolatedAsyncioTestCase):
    """Exercise HTTP routes with synthetic identities, without bypassing production auth."""
    async def asyncSetUp(self):
        runner.SESSIONS.clear()
        runner.LOGIN_STATES.clear()
        self.value = {'until': time.time()+3600, 'expires': time.time()+3600,
            'name': 'Test', 'csrf': 'csrf-test', 'access_token': 'test-access',
            'refresh_lock': asyncio.Lock(), 'run_lock': asyncio.Lock(), 'used_ids': set(),
            'tools': {'read': {'name': 'read', 'inputSchema': {'type': 'object',
                'properties': {'count': {'type': 'integer', 'minimum': 1}},
                'required': ['count'], 'additionalProperties': False},
                'annotations': {'readOnlyHint': True}}}}
        runner.SESSIONS['test-session'] = self.value
        self.client = httpx.AsyncClient(transport=httpx.ASGITransport(app=runner.app), base_url=runner.ORIGIN)
        self.client.cookies.set(runner.COOKIE, 'test-session')
        self.headers = {'Origin': runner.ORIGIN, 'X-CSRF-Token': 'csrf-test'}

    async def asyncTearDown(self):
        await self.client.aclose()

    async def call(self, **changes):
        data = {'name': 'read', 'arguments': {'count': 2}, 'request_id': 'unique-request-00001'}
        data.update(changes)
        return await self.client.post(runner.PREFIX+'/api/run', json=data, headers=self.headers)

    async def test_anonymous_cannot_read_or_execute(self):
        self.client.cookies.clear()
        self.assertEqual((await self.client.get(runner.PREFIX+'/api/tools')).status_code, 401)
        self.assertEqual((await self.call()).status_code, 401)

    async def test_csrf_wrong_origin_or_missing_token_rejected(self):
        for headers in ({'Origin': 'https://evil.example', 'X-CSRF-Token': 'csrf-test'}, {'Origin': runner.ORIGIN}):
            self.headers = headers
            self.assertEqual((await self.call()).status_code, 403)

    async def test_validation_and_unknown_tool_do_not_execute(self):
        with patch.object(runner, 'mcp', new_callable=AsyncMock) as mcp:
            self.assertEqual((await self.call(arguments={'count': '2'})).status_code, 422)
            self.assertEqual((await self.call(name='unknown')).status_code, 400)
            mcp.assert_not_called()

    async def test_external_schema_reference_does_not_fetch(self):
        self.value['tools']['read']['inputSchema'] = {'$ref': 'https://example.invalid/schema'}
        with patch.object(runner, 'mcp', new_callable=AsyncMock) as mcp:
            self.assertEqual((await self.call()).status_code, 422)
            mcp.assert_not_called()

    async def test_change_tool_requires_confirmation(self):
        self.value['tools']['read']['annotations'] = {}
        with patch.object(runner, 'mcp', new_callable=AsyncMock, return_value={'content': []}) as mcp:
            self.assertEqual((await self.call()).status_code, 409)
            self.assertEqual((await self.call(confirmed=True)).status_code, 200)
            self.assertEqual(mcp.await_count, 1)

    async def test_duplicate_and_timeout_never_replayed(self):
        with patch.object(runner, 'mcp', new_callable=AsyncMock, side_effect=httpx.ReadTimeout('timeout')) as mcp:
            self.assertEqual((await self.call()).status_code, 502)
            self.assertEqual((await self.call()).status_code, 409)
            self.assertEqual(mcp.await_count, 1)

    async def test_concurrent_submission_rejected(self):
        await self.value['run_lock'].acquire()
        self.assertEqual((await self.call()).status_code, 409)
        self.value['run_lock'].release()

    async def test_identity_hides_tokens_and_logout_ends_session(self):
        response = await self.client.get(runner.PREFIX+'/api/session')
        self.assertEqual(set(response.json()), {'name', 'csrf'})
        self.assertIn("frame-ancestors 'none'", response.headers['content-security-policy'])
        self.assertEqual((await self.client.post(runner.PREFIX+'/api/logout', headers=self.headers)).status_code, 200)
        self.assertEqual((await self.client.get(runner.PREFIX+'/api/session')).status_code, 401)

    async def test_login_uses_pkce_and_bound_cookie(self):
        with patch.object(runner, 'CLIENT_ID', 'client'), patch.object(runner, 'CLIENT_SECRET', 'secret'):
            response = await self.client.get(runner.PREFIX+'/auth/login')
        self.assertEqual(response.status_code, 302)
        self.assertIn('code_challenge_method=S256', response.headers['location'])
        self.assertIn('HttpOnly', response.headers['set-cookie'])
        self.assertEqual(len(runner.LOGIN_STATES), 1)
        state = next(iter(runner.LOGIN_STATES))
        self.client.cookies.clear()
        with patch.object(runner, 'token_request', new_callable=AsyncMock) as exchange:
            response = await self.client.get(runner.PREFIX+'/auth/callback', params={'state': state, 'code': 'code'})
            self.assertEqual(response.status_code, 400)
            exchange.assert_not_called()

    async def test_expired_state_rejected(self):
        runner.LOGIN_STATES['expired'] = {'binding': 'b', 'verifier': 'v', 'until': time.time()-1}
        response = await self.client.get(runner.PREFIX+'/auth/callback?state=expired&code=code')
        self.assertEqual(response.status_code, 400)


class ProtocolTests(unittest.IsolatedAsyncioTestCase):
    """Check actual wire parsing, request IDs, notifications and paginated discovery."""
    async def test_sse_and_session_id(self):
        def handler(request):
            data = __import__('json').loads(request.content)
            body = 'event: message\ndata: '+__import__('json').dumps({'jsonrpc':'2.0','id':data['id'],'result':{'ok':True}})+'\n\n'
            return httpx.Response(200, text=body, headers={'content-type':'text/event-stream','mcp-session-id':'session-123'})
        headers = {}
        async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
            self.assertEqual(await runner.rpc(client,headers,'tools/list'), {'ok':True})
        self.assertEqual(headers['Mcp-Session-Id'], 'session-123')

    async def test_catalog_pagination(self):
        responses = [{'protocolVersion':'2025-03-26'}, {}, {'tools':[{'name':'one'}], 'nextCursor':'next'}, {'tools':[{'name':'two'}]}]
        with patch.object(runner,'access_token',new_callable=AsyncMock,return_value='t'), patch.object(runner,'rpc',new_callable=AsyncMock,side_effect=responses) as rpc:
            result = await runner.mcp({},'tools/list')
            self.assertEqual([t['name'] for t in result['tools']], ['one','two'])
            self.assertEqual(rpc.call_args.args[3], {'cursor':'next'})


if __name__ == '__main__':
    unittest.main()
