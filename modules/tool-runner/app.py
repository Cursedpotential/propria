"""Run gateway tools through a browser using the existing user's OAuth identity.

Inputs: environment configuration and authenticated HTTP requests. Outputs: live MCP
catalogs and results. Effects: only explicitly submitted tools execute. No service
account impersonation, token exposure to JavaScript, or automatic call retries.
"""
import asyncio
import base64
import hashlib
import json
import os
import secrets
import time
from pathlib import Path
from urllib.parse import urlencode, urlsplit

import httpx
from jsonschema import Draft202012Validator
from referencing import Registry
from starlette.applications import Starlette
from starlette.exceptions import HTTPException
from starlette.requests import Request
from starlette.responses import FileResponse, JSONResponse, RedirectResponse
from starlette.routing import Route, Mount
from starlette.staticfiles import StaticFiles
from starlette.middleware.base import BaseHTTPMiddleware

ROOT = Path(__file__).parent
PUBLIC_URL = os.getenv('PUBLIC_URL', 'http://127.0.0.1:8790/runner').rstrip('/')
PREFIX = urlsplit(PUBLIC_URL).path.rstrip('/')
ORIGIN = f'{urlsplit(PUBLIC_URL).scheme}://{urlsplit(PUBLIC_URL).netloc}'
ISSUER = os.getenv('OAUTH_ISSUER', 'https://auth.int.mitechconsult.com/application/o/propria-gateway/').rstrip('/')
AUTH_BASE = ISSUER.split('/application/o/')[0]
CLIENT_ID = os.getenv('OAUTH_CLIENT_ID', '')
CLIENT_SECRET = os.getenv('OAUTH_CLIENT_SECRET', '')
MCP_URL = os.getenv('MCP_URL', 'https://mcp.mitechconsult.com/servers/320572d2bdd440f3ac61c8b64418d7b3/mcp')
COOKIE = 'propria_runner'
SECURE = PUBLIC_URL.startswith('https://')
SESSIONS = {}
LOGIN_STATES = {}


def prune():
    """Expire in-memory login states and sessions; inputs none, no external effects."""
    now = time.time()
    for collection in (SESSIONS, LOGIN_STATES):
        for key in list(collection):
            if collection[key]['until'] < now:
                collection.pop(key, None)


def session(request):
    """Resolve the opaque cookie to a server-held identity; reject expired sessions."""
    prune()
    value = SESSIONS.get(request.cookies.get(COOKIE, ''))
    if not value:
        raise HTTPException(401, 'Sign in to Propria to continue.')
    return value


def check_csrf(request, value):
    """Require the same origin and per-session CSRF token for submitted actions."""
    if request.headers.get('origin') != ORIGIN or not secrets.compare_digest(
        request.headers.get('x-csrf-token', ''), value['csrf']
    ):
        raise HTTPException(403, 'This request did not come from your runner session.')


async def token_request(data):
    """Exchange an OAuth code or refresh token server-side; never return secrets to UI."""
    async with httpx.AsyncClient(timeout=30) as client:
        response = await client.post(AUTH_BASE + '/application/o/token/',
            data={**data, 'client_id': CLIENT_ID, 'client_secret': CLIENT_SECRET})
    if response.status_code != 200:
        raise HTTPException(401, 'Sign-in could not be completed. Please sign in again.')
    return response.json()


async def access_token(value):
    """Refresh expired user access under a lock; inputs session, output bearer token."""
    async with value['refresh_lock']:
        if value['expires'] < time.time() + 30:
            if not value.get('refresh_token'):
                raise HTTPException(401, 'Please sign in again.')
            tokens = await token_request({'grant_type': 'refresh_token', 'refresh_token': value['refresh_token']})
            value.update(access_token=tokens['access_token'],
                refresh_token=tokens.get('refresh_token', value['refresh_token']),
                expires=time.time() + tokens.get('expires_in', 3600))
        return value['access_token']


async def rpc(client, headers, method, params=None, notification=False, endpoint=None):
    """Send one MCP message, handling JSON/SSE responses without replaying operations.

    Inputs: authenticated HTTP client, method and parameters. Output: JSON-RPC result.
    Effects: one gateway request; transport errors leave execution status unknown.
    """
    rid = secrets.token_hex(12)
    message = {'jsonrpc': '2.0', 'method': method, 'params': params or {}}
    if not notification:
        message['id'] = rid
    async with client.stream('POST', endpoint or MCP_URL, headers=headers, json=message) as response:
        if response.status_code in (401, 403):
            raise HTTPException(response.status_code, 'Gateway denied access. Sign in again or check your account permissions.')
        if response.status_code >= 400:
            raise HTTPException(502, f'Gateway returned HTTP {response.status_code}. The call was not retried.')
        if response.headers.get('mcp-session-id'):
            headers['Mcp-Session-Id'] = response.headers['mcp-session-id']
        if notification:
            return {}
        result = None
        if 'text/event-stream' in response.headers.get('content-type', ''):
            data = []
            size = 0
            async for line in response.aiter_lines():
                size += len(line)
                if size > 16_000_000:
                    raise HTTPException(502, 'Result exceeds the 16 MB display limit. Execution may have completed.')
                if line.startswith('data:'):
                    data.append(line[5:].lstrip())
                elif line == '' and data:
                    event = json.loads('\n'.join(data))
                    data = []
                    if event.get('id') == rid:
                        result = event
                        break
        else:
            body = bytearray()
            async for chunk in response.aiter_bytes():
                body.extend(chunk)
                if len(body) > 16_000_000:
                    raise HTTPException(502, 'Result exceeds the 16 MB display limit. Execution may have completed.')
            result = json.loads(body)
        if not result or result.get('id') != rid:
            raise HTTPException(502, 'Gateway did not return a matching result. The call was not retried.')
        if 'error' in result:
            raise HTTPException(502, 'Gateway rejected the request: ' + str(result['error'].get('message', 'MCP error'))[:500])
        return result['result']


async def mcp(value, method, params=None, endpoint=None):
    """Initialize a fresh MCP session, perform one read/call, and return its result.

    Inputs: user's session, allowlisted method/arguments. Effects: gateway calls only;
    catalog pagination is bounded and tool execution is never automatically retried.
    """
    headers = {'Authorization': 'Bearer ' + await access_token(value),
               'Accept': 'application/json, text/event-stream'}
    async with httpx.AsyncClient(timeout=httpx.Timeout(180, connect=15)) as client:
        init = await rpc(client, headers, 'initialize', {'protocolVersion': '2025-03-26',
            'capabilities': {}, 'clientInfo': {'name': 'propria-tool-runner', 'version': '0.2.0'}}, endpoint=endpoint)
        headers['MCP-Protocol-Version'] = init['protocolVersion']
        await rpc(client, headers, 'notifications/initialized', notification=True, endpoint=endpoint)
        if method != 'tools/list':
            return await rpc(client, headers, method, params, endpoint=endpoint)
        result = []
        cursor = None
        seen = set()
        for _ in range(100):
            page = await rpc(client, headers, 'tools/list', {'cursor': cursor} if cursor else {}, endpoint=endpoint)
            result.extend(page.get('tools', []))
            cursor = page.get('nextCursor')
            if not cursor:
                return {'tools': result}
            if cursor in seen:
                break
            seen.add(cursor)
        raise HTTPException(502, 'Gateway catalog pagination did not finish.')


async def health(request):
    """Return process readiness only; does not claim gateway or identity-provider health."""
    return JSONResponse({'status': 'ok', 'configured': bool(CLIENT_ID and CLIENT_SECRET)})


async def index(request):
    """Serve the runner shell without user data; sign-in is required by every data API."""
    return FileResponse(ROOT / 'static' / 'index.html')


async def canonical_url(request):
    """Redirect the bare runner path to its slash form so relative asset URLs resolve."""
    return RedirectResponse(PUBLIC_URL + '/', status_code=307)


def open_runner():
    """Open Propria's searchable tool runner to select a tool, fill inputs and run directly.

    Inputs: none. Outputs: the hosted runner URL. Effects: none; opening the URL uses
    Authentik sign-in and the user's gateway permissions. Choose for manual execution
    instead of asking the assistant to choose and run a tool.
    """
    return {'content': [{'type': 'text', 'text': f'Open the Propria tool runner: {PUBLIC_URL}/'}],
            'structuredContent': {'url': PUBLIC_URL + '/'}, 'isError': False}


async def launcher_mcp(request):
    """Expose the public runner-link tool over stateless MCP; no account/data access.

    Inputs: initialize, tools/list, tools/call or initialized notification. Outputs:
    protocol metadata or public URL only. Effects: none; cannot execute gateway tools.
    """
    if request.headers.get('origin') not in (None, ORIGIN):
        raise HTTPException(403, 'Origin not allowed.')
    raw = bytearray()
    async for chunk in request.stream():
        raw.extend(chunk)
        if len(raw) > 64000:
            raise HTTPException(413, 'Request too large.')
    try:
        data = json.loads(raw)
    except ValueError:
        raise HTTPException(400, 'Invalid JSON.')
    if not isinstance(data, dict):
        raise HTTPException(400, 'Expected one MCP request.')
    method, rid = data.get('method'), data.get('id')
    if method == 'notifications/initialized':
        from starlette.responses import Response
        return Response(status_code=202)
    result = None
    if method == 'initialize':
        result = {'protocolVersion': '2025-03-26', 'capabilities': {'tools': {}},
                  'serverInfo': {'name': 'propria-runner', 'version': '1.0.0'}}
    elif method == 'ping':
        result = {}
    elif method == 'tools/list':
        result = {'tools': [{'name': 'open-runner', 'title': 'Open Propria tool runner',
            'description': open_runner.__doc__, 'inputSchema': {'type': 'object', 'properties': {}, 'additionalProperties': False},
            'annotations': {'readOnlyHint': True, 'destructiveHint': False, 'idempotentHint': True, 'openWorldHint': False}}]}
    elif method == 'tools/call' and data.get('params', {}).get('name') == 'open-runner':
        result = open_runner()
    if result is None:
        return JSONResponse({'jsonrpc': '2.0', 'id': rid, 'error': {'code': -32601, 'message': 'Unknown method or tool'}})
    return JSONResponse({'jsonrpc': '2.0', 'id': rid, 'result': result})


async def login(request):
    """Start Authentik authorization-code login with state, browser binding and PKCE."""
    prune()
    if not CLIENT_ID or not CLIENT_SECRET:
        raise HTTPException(503, 'Runner sign-in has not been configured.')
    if len(LOGIN_STATES) >= 1000:
        raise HTTPException(429, 'Too many pending sign-ins. Please try again shortly.')
    state, binding, verifier = (secrets.token_urlsafe(32) for _ in range(3))
    LOGIN_STATES[state] = {'binding': binding, 'verifier': verifier, 'until': time.time() + 600}
    challenge = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).decode().rstrip('=')
    url = AUTH_BASE + '/application/o/authorize/?' + urlencode({'client_id': CLIENT_ID,
        'redirect_uri': PUBLIC_URL + '/auth/callback', 'response_type': 'code',
        'scope': 'openid profile email offline_access', 'state': state,
        'code_challenge': challenge, 'code_challenge_method': 'S256'})
    response = RedirectResponse(url, status_code=302)
    response.set_cookie('runner_login', binding, max_age=600, secure=SECURE,
        httponly=True, samesite='lax', path=PREFIX + '/')
    return response


async def callback(request):
    """Validate browser-bound OAuth state, exchange the code and create an opaque session."""
    prune()
    state = LOGIN_STATES.pop(request.query_params.get('state', ''), None)
    if not state or not secrets.compare_digest(state['binding'], request.cookies.get('runner_login', '')):
        raise HTTPException(400, 'Sign-in expired or belongs to another browser. Please start again.')
    code = request.query_params.get('code')
    if not code:
        raise HTTPException(400, 'Sign-in was cancelled or denied.')
    tokens = await token_request({'grant_type': 'authorization_code', 'code': code,
        'redirect_uri': PUBLIC_URL + '/auth/callback', 'code_verifier': state['verifier']})
    async with httpx.AsyncClient(timeout=30) as client:
        user = await client.get(AUTH_BASE + '/application/o/userinfo/',
            headers={'Authorization': 'Bearer ' + tokens['access_token']})
    if user.status_code != 200:
        raise HTTPException(401, 'Could not verify your identity with Authentik.')
    identity = user.json()
    sid = secrets.token_urlsafe(48)
    SESSIONS.pop(request.cookies.get(COOKIE, ''), None)
    SESSIONS[sid] = {'access_token': tokens['access_token'], 'refresh_token': tokens.get('refresh_token'),
        'expires': time.time() + tokens.get('expires_in', 3600), 'until': time.time() + 43200,
        'name': identity.get('name') or identity.get('preferred_username', 'Signed in'),
        'csrf': secrets.token_urlsafe(32), 'refresh_lock': asyncio.Lock(), 'run_lock': asyncio.Lock(),
        'tools': {}, 'used_ids': set()}
    response = RedirectResponse(PUBLIC_URL + '/', status_code=303)
    response.set_cookie(COOKIE, sid, max_age=43200, secure=SECURE, httponly=True, samesite='lax', path=PREFIX + '/')
    response.delete_cookie('runner_login', path=PREFIX + '/')
    return response


async def whoami(request):
    """Return display identity and CSRF nonce, never access/refresh tokens."""
    value = session(request)
    return JSONResponse({'name': value['name'], 'csrf': value['csrf']})


async def logout(request):
    """End only the runner session after CSRF validation; preserve other app logins."""
    value = session(request)
    check_csrf(request, value)
    SESSIONS.pop(request.cookies[COOKIE], None)
    response = JSONResponse({'signed_out': True})
    response.delete_cookie(COOKIE, path=PREFIX + '/')
    return response


async def catalog(request):
    """Fetch the current authorized tool catalog and cache its schemas in this session."""
    value = session(request)
    result = await mcp(value, 'tools/list')
    value['tools'] = {tool['name']: tool for tool in result['tools']}
    return JSONResponse(result)


async def run_tool(request):
    """Validate submitted inputs and execute exactly one explicitly chosen gateway tool.

    Inputs: name, arguments, request_id and confirmation for non-read-only tools.
    Output: untouched MCP result with timing. Effects: selected tool's documented
    effects; duplicate request IDs and concurrent runs are rejected, never replayed.
    """
    value = session(request)
    check_csrf(request, value)
    raw = bytearray()
    async for chunk in request.stream():
        raw.extend(chunk)
        if len(raw) > 2_000_000:
            raise HTTPException(413, 'Inputs exceed the 2 MB limit.')
    try:
        payload = json.loads(raw)
    except ValueError:
        raise HTTPException(400, 'Inputs must be valid JSON.')
    if not isinstance(payload, dict):
        raise HTTPException(400, 'Expected an input object.')
    name, arguments, rid = payload.get('name'), payload.get('arguments'), payload.get('request_id')
    if not isinstance(name, str) or name not in value['tools']:
        raise HTTPException(400, 'Choose a tool from the current catalog.')
    if not isinstance(rid, str) or not 16 <= len(rid) <= 100:
        raise HTTPException(400, 'A valid request ID is required.')
    tool = value['tools'][name]
    try:
        # An empty registry forbids network resolution of untrusted external schema references.
        validator = Draft202012Validator(tool['inputSchema'], registry=Registry())
        errors = list(validator.iter_errors(arguments))
    except Exception:
        raise HTTPException(422, 'This tool has an unsupported input schema; execution was not started.')
    if errors:
        raise HTTPException(422, 'Check input at ' + '/'.join(str(p) for p in errors[0].path) + ': ' + errors[0].message[:300])
    if tool.get('annotations', {}).get('readOnlyHint') is not True and payload.get('confirmed') is not True:
        raise HTTPException(409, 'Review the inputs and confirm this tool may make changes.')
    if value['run_lock'].locked():
        raise HTTPException(409, 'A tool is already running in this session.')
    async with value['run_lock']:
        if rid in value['used_ids']:
            raise HTTPException(409, 'This request was already submitted; it will not run twice.')
        if len(value['used_ids']) >= 10000:
            raise HTTPException(429, 'Session run limit reached. Sign in again.')
        value['used_ids'].add(rid)
        start = time.monotonic()
        if tool.get('_domain'):
            import domains
            result = await domains.invoke(value, tool['_domain'], name, arguments, confirmed=payload.get('confirmed', False))
        else:
            result = await mcp(value, 'tools/call', {'name': name, 'arguments': arguments})
        return JSONResponse({'tool': name, 'request_id': rid, 'elapsed_ms': round((time.monotonic()-start)*1000), 'result': result})


async def http_error(request, exc):
    """Return sanitized application errors as JSON for the runner UI."""
    return JSONResponse({'error': exc.detail}, status_code=exc.status_code)


async def transport_error(request, exc):
    """Report unknown execution outcome without logging credentials or retrying calls."""
    return JSONResponse({'error': 'Connection failed or timed out. Execution may have completed; check its result before running again.'}, status_code=502)


routes = [Route('/health', health), Route(PREFIX + '/', index),
    Route(PREFIX + '/mcp', launcher_mcp, methods=['POST']),
    Route(PREFIX + '/auth/login', login), Route(PREFIX + '/auth/callback', callback),
    Route(PREFIX + '/api/session', whoami), Route(PREFIX + '/api/logout', logout, methods=['POST']),
    Route(PREFIX + '/api/tools', catalog), Route(PREFIX + '/api/run', run_tool, methods=['POST']),
    Mount(PREFIX + '/static', app=StaticFiles(directory=ROOT / 'static'))]
if PREFIX:
    routes.insert(0, Route(PREFIX, canonical_url))
app = Starlette(routes=routes, exception_handlers={HTTPException: http_error, httpx.HTTPError: transport_error})


async def security_headers(request, call_next):
    """Apply no-store and browser isolation headers to all runner responses."""
    response = await call_next(request)
    response.headers['Cache-Control'] = 'no-store'
    response.headers['X-Content-Type-Options'] = 'nosniff'
    response.headers['Referrer-Policy'] = 'no-referrer'
    response.headers['Content-Security-Policy'] = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"
    return response


app.add_middleware(BaseHTTPMiddleware, dispatch=security_headers)

# Extend this same authenticated service; no duplicate application or identity.
import sys
import domains
domains.install(sys.modules[__name__])
