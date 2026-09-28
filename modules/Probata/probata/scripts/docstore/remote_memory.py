"""Independent shared-memory connection; never falls back to the docs database."""
import asyncio
import os
import sys
import json
import re
from pathlib import Path
import httpx
from surrealdb import AsyncSurreal
from context_pack import pack
from upgrade import rows
import recall
import sq

# 0.8.1-r5 (Claude Code · Opus 5.5, 2026-09-27): every memory failure carries an HTTP status and a readable
# detail. A missing field used to be a bare ValueError, which release_api turned into "409 Conflict" and the
# ctl then reported as "Docstore unavailable"; no caller could tell what was wrong with its payload.
KINDS=('correction','preference','observation','handoff','fact','constraint','decision')
SCOPE=re.compile(r'^propria(/[a-z0-9_-]+)*$')
REQUIRED=('kind','claim','evidence','agent')
OPTIONAL=('scope','detail','confidence','observed_at','force','supersede','reason')
SUPERSEDE=re.compile(r'^memory:[A-Za-z0-9_]+$')
SCHEMA_HINT='docstore_capabilities(operation="docstore_memory_remember") lists every field'


class MemoryFailure(Exception):
    """A memory request that failed for a stated reason; release_api maps it to status + detail."""
    def __init__(self,status,detail):
        super().__init__(str(detail))
        self.status=status
        self.detail=detail


def rejected(reason,**extra):
    return MemoryFailure(422,{'reason':reason,**extra,'schema':SCHEMA_HINT})


def validate_remember(payload):
    """Return the payload fn::remember receives, or raise 422 naming every problem at once."""
    errors=[]
    unknown=sorted(set(payload)-set(REQUIRED)-set(OPTIONAL))
    if unknown:
        errors.append(f'unknown field(s) {unknown}; allowed: {list(REQUIRED+OPTIONAL)}')
    missing=[key for key in REQUIRED if payload.get(key) in (None,'')]
    if missing:
        errors.append(f'missing required field(s) {missing}')
    if payload.get('kind') is not None and payload['kind'] not in KINDS:
        errors.append(f'kind must be one of {list(KINDS)}')
    claim=payload.get('claim')
    if claim is not None and (not isinstance(claim,str) or not 10<len(claim)<600):
        errors.append('claim must be a string of 11-599 characters')
    for key in ('evidence','agent','detail','reason','observed_at'):
        if payload.get(key) is not None and not isinstance(payload[key],str):
            errors.append(f'{key} must be a string')
    scope=payload.get('scope','propria')
    if not isinstance(scope,str) or not SCOPE.fullmatch(scope):
        errors.append('scope must match ^propria(/[a-z0-9_-]+)*$ (e.g. "propria" or "propria/intake")')
    confidence=payload.get('confidence')
    if confidence is not None and (isinstance(confidence,bool) or not isinstance(confidence,(int,float)) or not 0<=confidence<=1):
        errors.append('confidence must be a number from 0 to 1')
    if payload.get('force') not in (None,True,False):
        errors.append('force must be true or false')
    supersede=payload.get('supersede')
    if supersede is not None and (not isinstance(supersede,str) or not SUPERSEDE.fullmatch(supersede)):
        errors.append('supersede must be a memory record id such as "memory:abc123"')
    if len(str(payload))>20000:
        errors.append('payload exceeds 20000 characters')
    if errors:
        raise rejected('invalid memory payload',errors=errors)
    return {**payload,'scope':scope}


def upstream_failure(message):
    """Classify an error the memory database raised into conflict, rejection or upstream failure."""
    text=' '.join(str(message).split())[:1500]
    if 'memory_claim_uq' in text or 'already contains' in text:
        return MemoryFailure(409,{'reason':'this exact claim already exists in this scope (active, superseded or '
                                  'retracted); reword the claim','database':text})
    if any(marker in text for marker in ('does not exist','not active','ASSERT','coerce','Found ','Expected ')):
        return MemoryFailure(422,{'reason':'the memory database rejected the write','database':text})
    return MemoryFailure(502,{'reason':'the memory database failed','database':text})


async def native_call(function,args):
    """Use the existing dedicated memory MCP; credentials never enter tool arguments."""
    from fastmcp import Client
    sys.path.insert(0,str(Path(__file__).resolve().parents[2]/'plugins/docstore/control'))
    from native_transport import DocstoreTransport
    def factory(**kwargs):
        return httpx.AsyncClient(**{**kwargs,'timeout':30,'follow_redirects':False,'trust_env':False})
    transport=DocstoreTransport(os.environ['MEMORY_MCP_URL'],headers={
        'Authorization':'Basic '+os.environ['MEMORY_BASIC_AUTH'],
        'surreal-ns':os.environ.get('MEMORY_SURREAL_NS','probata_memory'),
        'surreal-db':os.environ.get('MEMORY_SURREAL_DB','memory')},httpx_client_factory=factory)
    try:
        async with Client(transport,timeout=40) as client:
            result=await client.call_tool('run',{'function':'fn::'+function,'args':args},raise_on_error=False)
    except (httpx.HTTPError,OSError,TimeoutError) as exc:
        raise MemoryFailure(503,{'reason':'the memory service could not be reached','error':type(exc).__name__}) from None
    value=result.structured_content
    text=''.join(getattr(c,'text','') for c in result.content)
    if result.is_error:
        raise upstream_failure(text or value)
    if value is None:
        value=json.loads(text)
    if isinstance(value,dict) and value.get('status')=='error':
        raise upstream_failure(value.get('error') or value)
    if isinstance(value,dict) and value.get('status')=='ok' and 'value' in value:
        value=value['value']
    return value


class MemoryConnection:
    async def query(self,sql,params):
        if 'fn::recall' in sql:
            return await native_call('recall',[params['query'],params['vector'],params['scope'],params['limit']])
        return await native_call('remember',[params['payload']])
    async def close(self):
        pass


async def operation(action,payload):
    url=os.environ.get('MEMORY_SURREAL_URL')
    native=os.environ.get('MEMORY_MCP_URL')
    if not url and not native:
        return {'available':False,'queried':False,'reason':'Dedicated remote memory is not configured','results':[]}
    if not (native or url).startswith(('https://','http://','wss://','ws://')):
        raise ValueError('Remote memory URL required')
    db=MemoryConnection() if native else AsyncSurreal(url)
    if not native:
        await db.connect()
    try:
        if not native:
            await db.signin({'username':os.environ['MEMORY_SURREAL_USER'],'password':os.environ['MEMORY_SURREAL_PASS']})
            await db.use(os.environ.get('MEMORY_SURREAL_NS','probata_memory'),os.environ.get('MEMORY_SURREAL_DB','memory'))
        if action=='recall':
            query=payload.get('query',''); limit=int(payload.get('limit',10))
            if not 1<=len(query)<=2000 or not 1<=limit<=50:
                raise rejected('invalid memory recall bounds: query 1-2000 characters, limit 1-50')
            # 0.8.1-r5: the store's scope root is propria (2026-09-19 migration); a probata default matched nothing.
            if not SCOPE.fullmatch(str(payload.get('scope','propria'))):
                raise rejected('scope must match ^propria(/[a-z0-9_-]+)*$ (e.g. "propria" or "propria/intake")')
            vector=await asyncio.to_thread(recall.embed,query)
            hits=rows(await db.query('RETURN fn::recall($query,$vector,$scope,$limit);',{'query':query,'vector':vector,'scope':payload.get('scope','propria'),'limit':limit}))
            results=[]
            for hit in hits:
                item=sq.norm(hit,True)
                item.update(source='remote-memory',source_record_id=str(hit.get('id','')),snippet=hit.get('claim',''),event_time=str(hit.get('observed_at','')) or None)
                results.append(item)
            return {**pack(results,limit=limit),'available':True,'queried':True}
        if action=='remember':
            data=validate_remember(payload)
            data['embedding']=await asyncio.to_thread(recall.embed,str(payload['claim']))
            result=sq.norm(await db.query('RETURN fn::remember($payload);',{'payload':data}),True)
            if isinstance(result,list) and len(result)==1:
                result=result[0]
            if not isinstance(result,dict):
                raise MemoryFailure(502,{'reason':'fn::remember returned no result object','database':str(result)[:500]})
            if not result.get('written'):
                # A refused near-duplicate is a conflict, not a success: nothing was written.
                raise MemoryFailure(409,{'reason':'near-duplicate: similar active memory exists in this scope; nothing was written',
                                         'conflicts':[{k:c.get(k) for k in ('id','claim','dist','overlap','confidence')}
                                                      for c in result.get('conflicts') or []],
                                         'next':'pass supersede:"<id>" with a reworded claim to replace one, or force:true to keep both'})
            return {'outcome':'superseded' if result.get('superseded') else 'written','id':result['written'],
                    'superseded':result.get('superseded'),'scope':data['scope'],'available':True}
        raise MemoryFailure(404,{'reason':f'unknown memory action {action!r}; supported: recall, remember'})
    finally:
        await db.close()
