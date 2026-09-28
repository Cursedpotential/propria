"""0.8 API additions; all mutations use the existing server authentication."""
import asyncio
import os
from fastapi import Header, HTTPException, Request
from context_pack import pack
import adr
import knowledge
import upgrade
import source_sync
import remote_memory
from pathlib import Path
from run_support import worker_lock, WorkerBusy


def register(app,auth):
    async def invoke(call):
        try:
            return await call
        except remote_memory.MemoryFailure as exc:
            # 0.8.1-r5 (Claude Code · Opus 5.5, 2026-09-27): memory failures keep their own status and detail
            # (422 invalid payload, 409 near-duplicate with the conflicting ids, 502/503 memory service).
            raise HTTPException(exc.status,exc.detail) from None
        except ValueError as exc:
            raise HTTPException(409,str(exc)) from None

    @app.get('/release')
    async def release(authorization: str|None=Header(default=None)):
        auth(authorization)
        return {'version':upgrade.VERSION,'control':'hosted','raw_surreal_fallback':False,
                # Every root from scope.ROOTS, propria included. This used to hard-code
                # 'Propria/docs' and then exclude propria from the derived list, so /release
                # advertised one pre-2026-09-20 module path beside six junction paths.
                'source_roots':[p[0] for p in source_sync.ROOTS.values()],
                'server_processing':['CocoIndex','summary','classification','embedding','entity-resolution','rerank','DuckDB'],
                'semantic_mode':'remote-llm' if all(os.environ.get(key) for key in ('DOCSTORE_LLM_BASE_URL','DOCSTORE_LLM_MODEL','DOCSTORE_LLM_API_KEY')) else 'deterministic-extractive'}

    @app.post('/upgrade/{action}')
    async def upgrade_route(action: str,payload:dict,authorization: str|None=Header(default=None)):
        auth(authorization)
        return await invoke(upgrade.operation(action,payload.get('plan_id')))

    @app.post('/adr/{action}')
    async def adr_route(action:str,payload:dict,authorization: str|None=Header(default=None)):
        auth(authorization)
        if action in {'create','update','migration-apply'}:
            try:
                with worker_lock(Path(os.environ.get('DOCSTORE_SYNC_LOCK','/data/state/sync.lock'))):
                    return await invoke(adr.operation(action,payload))
            except WorkerBusy as exc:
                raise HTTPException(409,str(exc)) from None
        return await invoke(adr.operation(action,payload))

    @app.get('/sources/retraction-plan')
    async def retraction_plan(authorization: str|None=Header(default=None)):
        auth(authorization)
        from cdc_verify import snapshot_sources, retire_unexpected_projection
        expected,_=snapshot_sources()
        return await retire_unexpected_projection(expected,dry_run=True)

    @app.post('/knowledge/{action}')
    async def knowledge_route(action:str,payload:dict,authorization: str|None=Header(default=None)):
        auth(authorization)
        if action not in {'entity-upsert','relate','traverse','path'}:
            raise HTTPException(400,'Unsupported graph action')
        return await invoke(knowledge.graph(action,payload))

    @app.post('/sources/{action}')
    async def source_route(action:str,request:Request,authorization: str|None=Header(default=None)):
        auth(authorization)
        # Bound streaming request before JSON parsing, including chunked uploads.
        data=bytearray()
        async for chunk in request.stream():
            data.extend(chunk)
            if len(data)>40*1024*1024:
                raise HTTPException(413,'Source upload exceeds 40 MiB')
        import json
        try:
            return await asyncio.to_thread(source_sync.operation,action,json.loads(data))
        except ValueError as exc:
            raise HTTPException(409,str(exc)) from None

    @app.post('/context/pack')
    async def context_route(payload:dict,authorization: str|None=Header(default=None)):
        auth(authorization)
        try:
            return pack(payload.get('rows',[]),limit=payload.get('limit',20),budget=payload.get('budget',8000))
        except ValueError as exc:
            raise HTTPException(400,str(exc)) from None

    @app.post('/surrealql/read')
    async def surrealql_route(payload:dict,authorization: str|None=Header(default=None)):
        auth(authorization)
        return await invoke(knowledge.surrealql_read(payload.get('query','')))

    @app.post('/memory/{action}')
    async def memory_route(action:str,payload:dict,authorization: str|None=Header(default=None)):
        auth(authorization)
        return await invoke(remote_memory.operation(action,payload))
