"""Independent shared-memory connection; never falls back to the docs database."""
import asyncio
import os
import sys
import json
from pathlib import Path
from surrealdb import AsyncSurreal
from context_pack import pack
from upgrade import rows
import recall
import sq


async def native_call(function,args):
    """Use the existing dedicated memory MCP; credentials never enter tool arguments."""
    import httpx
    from fastmcp import Client
    sys.path.insert(0,str(Path(__file__).resolve().parents[2]/'plugins/docstore/control'))
    from native_transport import DocstoreTransport
    def factory(**kwargs):
        return httpx.AsyncClient(**{**kwargs,'timeout':30,'follow_redirects':False,'trust_env':False})
    transport=DocstoreTransport(os.environ['MEMORY_MCP_URL'],headers={
        'Authorization':'Basic '+os.environ['MEMORY_BASIC_AUTH'],
        'surreal-ns':os.environ.get('MEMORY_SURREAL_NS','probata_memory'),
        'surreal-db':os.environ.get('MEMORY_SURREAL_DB','memory')},httpx_client_factory=factory)
    async with Client(transport,timeout=40) as client:
        result=await client.call_tool('run',{'function':'fn::'+function,'args':args})
        value=result.structured_content
        if value is None:
            text=''.join(getattr(c,'text','') for c in result.content)
            value=json.loads(text)
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
                raise ValueError('Invalid memory recall bounds')
            vector=await asyncio.to_thread(recall.embed,query)
            hits=rows(await db.query('RETURN fn::recall($query,$vector,$scope,$limit);',{'query':query,'vector':vector,'scope':payload.get('scope','probata'),'limit':limit}))
            results=[]
            for hit in hits:
                item=sq.norm(hit,True)
                item.update(source='remote-memory',source_record_id=str(hit.get('id','')),snippet=hit.get('claim',''),event_time=str(hit.get('observed_at','')) or None)
                results.append(item)
            return {**pack(results,limit=limit),'available':True,'queried':True}
        if action=='remember':
            required={'kind','claim','detail','evidence','agent'}
            if not required<=payload.keys() or len(str(payload))>20000:
                raise ValueError('Memory fields required or payload oversized')
            data={**payload,'scope':payload.get('scope','probata')}
            data['embedding']=await asyncio.to_thread(recall.embed,str(payload['claim']))
            result=await db.query('RETURN fn::remember($payload);',{'payload':data})
            return {'result':sq.norm(result,True),'available':True}
        raise ValueError('Unknown memory action')
    finally:
        await db.close()
