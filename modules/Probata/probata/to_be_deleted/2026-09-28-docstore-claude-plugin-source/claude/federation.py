"""Bounded local memory federation. No indexing, provider installation or hidden fallback."""
from __future__ import annotations
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

SOURCES=('claude','codex','read-memories','memsearch','cnf','.remember','remote-memory','docstore','ccc','smart-explore')


def text_content(value):
    if isinstance(value,str): return value
    if isinstance(value,list): return '\n'.join(text_content(v) for v in value)
    if isinstance(value,dict):
        return text_content(value.get('text') or value.get('content') or value.get('message') or value.get('payload') or '')
    return ''


def file_search(source,directories,query,limit=20,since=None,until=None):
    results=[]; examined=0; skipped=0
    files=[]
    for directory in directories:
        if directory.is_dir():
            for pattern in ('*.jsonl','*.md','*.json'):
                for file in directory.rglob(pattern):
                    if len(files)>=2000: break
                    if not file.is_symlink(): files.append(file)
    for file in sorted(set(files),key=lambda p:p.stat().st_mtime,reverse=True)[:200]:
        if file.stat().st_size>4*1024*1024:
            skipped+=1; continue
        examined+=1
        for line_number,line in enumerate(file.read_text(encoding='utf-8',errors='replace').splitlines(),1):
            record={}
            if file.suffix=='.jsonl':
                try: record=json.loads(line)
                except ValueError: continue
                text=text_content(record)
            else: text=line
            if query.casefold() not in text.casefold(): continue
            timestamp=record.get('timestamp') or record.get('created_at')
            if (since or until) and not timestamp: continue
            if since and str(timestamp)<since: continue
            if until and str(timestamp)>until: continue
            results.append({'source':source,'source_record_id':str(record.get('uuid') or record.get('id') or str(file)+':'+str(line_number)),
                            'path':str(file),'line':line_number,'snippet':text[:2000], 'score':1.0,
                            'message_timestamp':timestamp,'event_time':record.get('event_time'),
                            'created_at':record.get('created_at'),'updated_at':None,'ingested_at':None,
                            'file_modified_at':datetime.fromtimestamp(file.stat().st_mtime,timezone.utc).isoformat()})
            if len(results)>=limit: break
        if len(results)>=limit: break
    return results,{'available':bool(files),'queried':bool(examined),'files_examined':examined,'oversized_files_skipped':skipped,'bounded':True}


def recall(query,root,*,code=False,since=None,until=None,adapters=None):
    # A deployment may configure argv adapters for installed tools; no shell strings.
    adapters=dict(adapters or {})
    if os.environ.get('CF_MCP_CLIENT_TOKEN'):
        launcher=[sys.executable,str(Path(__file__).with_name('client.py')),'call']
        # 0.8.3 (Claude Code · Opus 5.5, 2026-09-27): the memory scope root is propria since 2026-09-19.
        adapters.setdefault('remote-memory',launcher+['docstore_memory_recall','--json',json.dumps({'query':'{query}','scope':'propria','limit':10})])
        adapters.setdefault('docstore',launcher+['coco_docstore_search','--json',json.dumps({'query':'{query}','domain':'docs','status':'all','presentation':'full','limit':10})])
    if code and shutil.which('ccc'):
        adapters.setdefault('ccc',['ccc','search','{query}','--limit','10','--json'])
    claude_home=Path(os.environ.get('CLAUDE_CONFIG_DIR',Path.home()/'.claude'))
    codex_home=Path(os.environ.get('CODEX_HOME',Path.home()/'.codex'))
    locations={'claude':[claude_home/'projects'],'codex':[codex_home/'sessions',codex_home/'archived_sessions'],
               'cnf':[Path(os.environ.get('CNF_MEMORY_DIR',root/'.cnf'))],'.remember':[Path(os.environ.get('REMEMBER_DIR',root/'.remember'))]}
    output=[]; states={}
    for source in SOURCES:
        if source in locations and source not in adapters:
            hits,state=file_search(source,locations[source],query,since=since,until=until); output.extend(hits); states[source]=state
        elif source in {'ccc','smart-explore'} and not code:
            states[source]={'available':bool(shutil.which('ccc' if source=='ccc' else 'se')),'queried':False,'reason':'code-only source'}
        elif source in adapters:
            argv=adapters[source]
            if not isinstance(argv,list) or not argv or any(not isinstance(a,str) for a in argv):
                raise ValueError('Adapter must be an argv list')
            try:
                def expand(argument):
                    if argument.startswith('{') and '"query"' in argument:
                        value=json.loads(argument);value['query']=query;return json.dumps(value)
                    return argument.replace('{query}',query).replace('{root}',str(root))
                result=subprocess.run([expand(a) for a in argv],cwd=root,capture_output=True,timeout=45,shell=False,
                                      env={**os.environ,'PYTHONUTF8':'1','PYTHONIOENCODING':'utf-8'})
                if result.returncode or len(result.stdout)>2*1024*1024: raise ValueError('Adapter failed')
                data=json.loads(result.stdout.decode('utf-8')); hits=data.get('results',[]) if isinstance(data,dict) else data
                if not isinstance(hits,list): raise ValueError('Adapter must return result rows')
                for row in hits[:30]:
                    output.append({**row,'source':source,'source_record_id':str(row.get('id') or row.get('source_record_id') or hashlib.sha256(json.dumps(row,sort_keys=True).encode()).hexdigest())})
                states[source]={'available':True,'queried':True,'count':min(len(hits),30)}
            except (OSError,ValueError,subprocess.TimeoutExpired):
                states[source]={'available':False,'queried':False,'reason':'adapter unavailable or invalid response'}
        else:
            states[source]={'available':False,'queried':False,'reason':'no configured adapter; direct upstream command remains available'}
    # Server packing is optional for local-private history; identical DuckDB implementation is bundled for local use.
    from context_pack import pack
    result=pack(output,limit=20,budget=16000)
    return {**result,'sources':states,'since':since,'until':until,'partial':any(not s.get('queried') for s in states.values())}


if __name__=='__main__':
    p=argparse.ArgumentParser(); p.add_argument('query'); p.add_argument('--root',default='.'); p.add_argument('--code',action='store_true'); p.add_argument('--since'); p.add_argument('--until'); p.add_argument('--adapters')
    a=p.parse_args(); adapters=json.loads(Path(a.adapters).read_text()) if a.adapters else {}
    if shutil.which('memsearch') and 'memsearch' not in adapters:
        adapters['memsearch']=['memsearch','search','{query}','--top-k','5','--json-output']
    print(json.dumps(recall(a.query,Path(a.root).resolve(),code=a.code,since=a.since,until=a.until,adapters=adapters),ensure_ascii=False))
