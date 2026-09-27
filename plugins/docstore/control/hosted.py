"""Authenticated stateless Streamable HTTP entry point for the hosted ctl service."""
import hashlib
import hmac
import os
import asyncio
from fastmcp.server.auth import TokenVerifier, AccessToken
from server import build_server, Config
from cli import mcp_runtime


class ControlTokenVerifier(TokenVerifier):
    def __init__(self,token):
        super().__init__()
        if len(token)<32:
            raise ValueError('DOCSTORE_CONTROL_TOKEN must contain at least 32 characters')
        self.digest=hashlib.sha256(token.encode()).digest()

    async def verify_token(self,token):
        if hmac.compare_digest(hashlib.sha256(token.encode()).digest(),self.digest):
            return AccessToken(token=token,client_id='docstore-client',scopes=['docstore'])
        return None


# 0.8.1-r2 (Claude Code · Opus 5.5, 2026-09-26): docstore_source_plan/apply carry the complete five-root
# manifest in one tool call (about 12 MB on 2026-09-26). The MCP SDK's Streamable HTTP transport refuses
# request bodies over 4 MiB with HTTP 413, so a client sync could never reach the source mirror. The hosted
# ctl accepts the worker API's own source-upload bound (40 MiB) instead.
def mcp_body_limit():
    limit=int(os.environ.get('DOCSTORE_MCP_MAX_BODY_BYTES',str(40*1024*1024)))
    if not 4*1024*1024<=limit<=64*1024*1024:
        raise ValueError('DOCSTORE_MCP_MAX_BODY_BYTES must be between 4 MiB and 64 MiB')
    return limit


def allow_source_manifests(limit):
    from mcp.server.streamable_http_manager import StreamableHTTPSessionManager
    original=StreamableHTTPSessionManager.__init__

    def __init__(self,*args,max_request_body_size=limit,**kwargs):
        original(self,*args,max_request_body_size=max_request_body_size,**kwargs)
    StreamableHTTPSessionManager.__init__=__init__


def main():
    if not os.environ.get('DOCSTORE_API_TOKEN'):
        raise ValueError('Authenticated worker API is required')
    os.environ['DOCSTORE_MCP_TRANSPORT']='http'
    from public_surface import build_public_server
    server=asyncio.run(build_public_server(build_server(Config.from_env())))
    server.auth=ControlTokenVerifier(os.environ.get('DOCSTORE_CONTROL_TOKEN',''))
    runtime=mcp_runtime({**os.environ,'DOCSTORE_MCP_TRANSPORT':'http'})
    allow_source_manifests(mcp_body_limit())
    server.run(**runtime)


if __name__=='__main__':
    main()
