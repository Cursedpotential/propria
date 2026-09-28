"""Shared native-connector configuration contract. By Codex, 2026-09-20.

The native connector and portable client share mcp_servers.propria-docs in
Codex config.toml. Neither path needs or creates local state.
"""
from pathlib import Path
from dataclasses import dataclass, field
import json
import os
import re
from urllib.parse import urlsplit


class ConnectionConfigurationError(ValueError):
    pass


class ConnectionAuthenticationError(ValueError):
    pass


@dataclass(frozen=True)
class ConnectionSettings:
    url: str
    headers: dict = field(repr=False)


def load_connection(environ=None, manifest_path=None):
    environ = os.environ if environ is None else environ
    # Codex owns the native HTTP connection in config.toml. Read that same
    # section for fallback calls; never infer endpoint/state from cache ancestry.
    try:
        if manifest_path is not None:
            server = json.loads(Path(manifest_path).read_text(encoding='utf-8'))['mcpServers']['ctl']
        else:
            import tomllib
            codex_home = Path(environ.get('CODEX_HOME', str(Path.home() / '.codex')))
            configured = tomllib.loads((codex_home / 'config.toml').read_text(encoding='utf-8'))['mcp_servers']['propria-docs']
            if configured.get('enabled', True) is False:
                raise ConnectionConfigurationError('Codex propria-docs connection is disabled.')
            token_name = configured['bearer_token_env_var']
            if not re.fullmatch(r'[A-Z_][A-Z0-9_]*', token_name):
                raise ConnectionConfigurationError('Codex Docstore token variable is invalid.')
            server = {'type': 'http', 'url': configured['url'],
                      'headers': {'Authorization': 'Bearer ${' + token_name + '}'}}
    except (OSError, ValueError, KeyError, TypeError) as exc:
        if isinstance(exc, ConnectionConfigurationError):
            raise
        raise ConnectionConfigurationError('Codex mcp_servers.propria-docs configuration is missing or invalid.') from None

    def expand(value):
        def replace(match):
            name, default = match.group(1), match.group(2)
            result = environ.get(name, default)
            if result is None or not result.strip():
                if name == 'CF_MCP_CLIENT_TOKEN':
                    raise ConnectionAuthenticationError('CF_MCP_CLIENT_TOKEN is missing in this host process. Configure remote authentication, then reconnect.')
                raise ConnectionConfigurationError(f'{name} is empty or missing.')
            return result
        return re.sub(r'\$\{([A-Z_][A-Z0-9_]*)(?::-([^}]*))?\}', replace, value)

    try:
        if server['type'] != 'http':
            raise ConnectionConfigurationError('ctl must use the hosted HTTP MCP transport.')
        url = expand(server['url'])
        parsed = urlsplit(url)
        if (parsed.scheme not in {'https', 'http'} or not parsed.hostname or parsed.username
                or parsed.password or parsed.fragment or parsed.query
                or any(ord(c) <= 32 for c in url)):
            raise ConnectionConfigurationError('DOCSTORE_CONTROL_MCP_URL must be a credential-free hosted MCP URL.')
        if parsed.scheme == 'http':
            import ipaddress
            try:
                private_tailnet = ipaddress.ip_address(parsed.hostname) in ipaddress.ip_network('100.64.0.0/10')
            except ValueError:
                private_tailnet = False
            if parsed.hostname not in {'localhost', '127.0.0.1', '::1'} and not private_tailnet:
                raise ConnectionConfigurationError('Use HTTPS or an explicit private Tailnet control endpoint.')
        headers = {key: expand(value) for key, value in server.get('headers', {}).items()}
        if any('\r' in value or '\n' in value for value in headers.values()):
            raise ConnectionAuthenticationError('Remote authentication contains invalid header characters.')
    except (KeyError, TypeError):
        raise ConnectionConfigurationError('The plugin ctl connection manifest has invalid fields.') from None
    return ConnectionSettings(url, headers)


def error_report(exc):
    """Describe the failing layer without echoing secrets or server payloads."""
    import httpx
    seen = set()
    pending = [exc]
    while pending:
        current = pending.pop(0)
        if id(current) in seen:
            continue
        seen.add(id(current))
        if isinstance(current, ConnectionConfigurationError):
            return {'ok': False, 'category': 'local_configuration', 'detail': str(current)}
        if isinstance(current, ConnectionAuthenticationError):
            return {'ok': False, 'category': 'authentication', 'detail': str(current)}
        if isinstance(current, httpx.HTTPStatusError):
            status = current.response.status_code
            return {'ok': False, 'category': 'authentication' if status in {401, 403} else 'remote_service',
                    'http_status': status, 'detail': 'Hosted MCP returned an HTTP error.'}
        if isinstance(current, (httpx.RequestError, TimeoutError, ConnectionError)):
            return {'ok': False, 'category': 'network', 'detail': 'Could not complete the hosted MCP connection or request.'}
        pending.extend(getattr(current, 'exceptions', []))
        pending.extend(e for e in (current.__cause__, current.__context__) if e is not None)
    return {'ok': False, 'category': 'operation', 'error_type': type(exc).__name__,
            'detail': 'The requested operation failed; service availability has not been inferred from this error.'}
