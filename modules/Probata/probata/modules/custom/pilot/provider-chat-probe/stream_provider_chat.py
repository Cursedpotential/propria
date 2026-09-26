#!/usr/bin/env python3
"""Streamed plain-chat liveness across configured non-NVIDIA providers.

Secrets are read only at runtime from the user's private env files and are
never written to the report.
"""
import asyncio, json, os, re, time
from datetime import datetime, timezone
from pathlib import Path
import httpx

ROOT = Path(__file__).resolve().parent
SECRETS = [Path.home()/'.secrets'/'Agno-MCP-Platform.env', Path.home()/'.secrets'/'mistral.env']
for f in SECRETS:
    if f.exists():
        for line in f.read_text(encoding='utf-8').splitlines():
            if '=' in line and not line.lstrip().startswith('#'):
                k,v=line.split('=',1); os.environ.setdefault(k.strip(), v.strip().strip('"'))

PROVIDERS = {
 'openrouter_free': ('https://openrouter.ai/api/v1', 'OPENROUTER_API_KEY', 5),
 'ollama_cloud': ('https://ollama.com/v1', 'OLLAMA_API_KEY', 3),
 'groq': ('https://api.groq.com/openai/v1', 'GROQ_API_KEY', 4),
 'mistral': ('https://api.mistral.ai/v1', 'MISTRAL_API_KEY', 2),
 'cerebras': ('https://api.cerebras.ai/v1', 'CEREBRAS_API_KEY', 2),
 'openai': ('https://api.openai.com/v1', 'OPENAI_API_KEY', 2),
}
SKIP = re.compile(r'(embed|rerank|moderation|whisper|tts|audio|image|dall-e|transcri|realtime|search-preview)', re.I)

def candidates(provider, rows):
    ids=[x.get('id') for x in rows if x.get('id') and not SKIP.search(x['id'])]
    if provider=='openrouter_free':
        ids=[x.get('id') for x in rows if x.get('id') and (x['id'].endswith(':free') or (x.get('pricing',{}).get('prompt')==0 and x.get('pricing',{}).get('completion')==0)) and not SKIP.search(x['id'])]
    return sorted(set(ids))

async def stream(client, base, key, provider, model, first_timeout=20):
    started=time.monotonic(); h={'Authorization':f'Bearer {key}','Content-Type':'application/json'}
    body={'model':model,'stream':True,'max_tokens':4096,'temperature':0,'messages':[{'role':'user','content':'Reply with exactly the word PONG.'}]}
    try:
        response=await asyncio.wait_for(client.send(client.build_request('POST',base+'/chat/completions',headers=h,json=body),stream=True),first_timeout)
    except asyncio.TimeoutError: return {'provider':provider,'model':model,'classification':'no_headers','elapsed_s':round(time.monotonic()-started,2)}
    except Exception as e: return {'provider':provider,'model':model,'classification':'connection_error','detail':repr(e)[:300],'elapsed_s':round(time.monotonic()-started,2)}
    if response.status_code!=200:
        raw=(await response.aread()).decode(errors='replace')[:500]; await response.aclose()
        return {'provider':provider,'model':model,'status':response.status_code,'classification':'http_error','detail':raw,'elapsed_s':round(time.monotonic()-started,2)}
    visible=[]; reasoning=[]; first=None; events=0; finish=None; it=response.aiter_lines()
    try:
        while True:
            try: line=await asyncio.wait_for(it.__anext__(),first_timeout)
            except StopAsyncIteration: break
            except asyncio.TimeoutError: return {'provider':provider,'model':model,'status':200,'classification':'no_first_data_event','elapsed_s':round(time.monotonic()-started,2)}
            if not line.startswith('data:'): continue
            data=line[5:].strip(); first=round(time.monotonic()-started,2)
            if data=='[DONE]': break
            events+=1
            try: c=json.loads(data).get('choices',[{}])[0]
            except Exception: continue
            d=c.get('delta') or {}; visible.append(d.get('content') or ''); reasoning.append(d.get('reasoning_content') or d.get('reasoning') or ''); finish=c.get('finish_reason') or finish; break
        async for line in it:
            if not line.startswith('data:'): continue
            data=line[5:].strip()
            if data=='[DONE]': break
            events+=1
            try: c=json.loads(data).get('choices',[{}])[0]
            except Exception: continue
            d=c.get('delta') or {}; visible.append(d.get('content') or ''); reasoning.append(d.get('reasoning_content') or d.get('reasoning') or ''); finish=c.get('finish_reason') or finish
    finally: await response.aclose()
    text=''.join(visible).strip(); think=''.join(reasoning).strip()
    return {'provider':provider,'model':model,'status':200,'classification':'visible_text' if text else 'reasoning_only' if think else 'events_without_text' if events else 'empty_stream','first_data_s':first,'elapsed_s':round(time.monotonic()-started,2),'exact_pong':text=='PONG','finish_reason':finish}

async def main():
    timeout=httpx.Timeout(connect=15,read=None,write=30,pool=30); results=[]; catalogs={}; missing=[]
    async with httpx.AsyncClient(timeout=timeout,limits=httpx.Limits(max_connections=12)) as client:
        work=[]
        for name,(base,env,workers) in PROVIDERS.items():
            key=os.getenv(env)
            if not key: missing.append(name); continue
            try:
                r=await client.get(base+'/models',headers={'Authorization':f'Bearer {key}'}); rows=r.json().get('data',[]) if r.status_code==200 else []
                catalogs[name]={'status':r.status_code,'listed':len(rows),'candidates':len(candidates(name,rows))}
            except Exception as e: catalogs[name]={'error':repr(e)}; continue
            gate=asyncio.Semaphore(workers)
            async def one(model, n=name, b=base, k=key, g=gate):
                async with g:
                    out=await stream(client,b,k,n,model); print(f"{n} {model} -> {out['classification']}",flush=True); return out
            work += [one(m) for m in candidates(name,rows)]
        results=await asyncio.gather(*work)
    run={'timestamp':datetime.now(timezone.utc).isoformat(timespec='seconds'),'method':'SSE; 20s to first event; no timeout after streaming begins','catalogs':catalogs,'missing_provider_keys':missing,'results':results}
    stamp=datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ'); p=ROOT/f'non-nvidia-streamed-{stamp}.json'; p.write_text(json.dumps(run,indent=2),encoding='utf-8'); (ROOT/'latest-non-nvidia-streamed.json').write_text(json.dumps(run,indent=2),encoding='utf-8'); print(p)
if __name__ == '__main__':
    asyncio.run(main())
