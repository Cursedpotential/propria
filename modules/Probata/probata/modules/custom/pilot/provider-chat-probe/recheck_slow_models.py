#!/usr/bin/env python3
"""Recheck the two slow non-NVIDIA chat candidates with a 60s first-event window."""
import asyncio, json
from datetime import datetime, timezone
from pathlib import Path
import httpx
from stream_provider_chat import PROVIDERS, ROOT, os, stream

TARGETS = [('ollama_cloud', 'nemotron-3-ultra'), ('mistral', 'ministral-14b-latest')]

async def main():
    timeout=httpx.Timeout(connect=15,read=None,write=30,pool=30)
    async with httpx.AsyncClient(timeout=timeout) as client:
        results=[]
        for provider,model in TARGETS:
            base,env,_=PROVIDERS[provider]
            results.append(await stream(client,base,os.environ[env],provider,model,60))
    out=ROOT/'slow-60s-recheck.json'
    out.write_text(json.dumps({'timestamp':datetime.now(timezone.utc).isoformat(timespec='seconds'),'results':results},indent=2),encoding='utf-8')
    print(out)
    for r in results: print(r['provider'],r['model'],r['classification'],r.get('first_data_s'),r.get('elapsed_s'))

asyncio.run(main())
